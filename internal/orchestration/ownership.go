package orchestration

import (
	"errors"
	"fmt"
	"math"
	"time"
)

type WorkflowID string
type AttemptID string
type OwnerID string
type FenceToken uint64

type OwnershipState string

const (
	OwnershipUnowned     OwnershipState = "UNOWNED"
	OwnershipOwned       OwnershipState = "OWNED"
	OwnershipReconciling OwnershipState = "RECONCILING"
)

var (
	ErrInvalidLeaseTTL         = errors.New("invalid lease ttl")
	ErrInvalidAttempt          = errors.New("invalid attempt binding")
	ErrLeaseHeld               = errors.New("workflow lease already held")
	ErrLeaseExpired            = errors.New("workflow lease expired")
	ErrStaleFence              = errors.New("stale fencing token")
	ErrNotOwned                = errors.New("workflow is not owned")
	ErrReconciliationRequired  = errors.New("workflow reconciliation required")
	ErrFenceExhausted          = errors.New("fencing token exhausted")
	ErrOwnershipInvariant      = errors.New("ownership invariant violation")
)

type AttemptBinding struct {
	AttemptID   AttemptID
	SourceSHA   string
	WorkspaceID string
	ExecutorID  string
}

func (a AttemptBinding) Validate() error {
	if a.AttemptID == "" || a.SourceSHA == "" || a.WorkspaceID == "" || a.ExecutorID == "" {
		return ErrInvalidAttempt
	}
	return nil
}

type Ownership struct {
	WorkflowID WorkflowID
	State      OwnershipState
	OwnerID    OwnerID
	Token      FenceToken
	LeaseUntil time.Time
	Attempt    AttemptBinding
	Revision   uint64
}

func (o Ownership) Validate() error {
	switch o.State {
	case OwnershipUnowned:
		if o.OwnerID != "" || !o.LeaseUntil.IsZero() || o.Attempt.AttemptID != "" {
			return ErrOwnershipInvariant
		}
	case OwnershipOwned:
		if o.WorkflowID == "" || o.OwnerID == "" || o.Token == 0 || o.LeaseUntil.IsZero() {
			return ErrOwnershipInvariant
		}
		if err := o.Attempt.Validate(); err != nil {
			return err
		}
	case OwnershipReconciling:
		if o.WorkflowID == "" || o.Token == 0 {
			return ErrOwnershipInvariant
		}
	default:
		return ErrOwnershipInvariant
	}
	return nil
}

func Acquire(current Ownership, workflowID WorkflowID, ownerID OwnerID, attempt AttemptBinding, now time.Time, ttl time.Duration) (Ownership, error) {
	if workflowID == "" || ownerID == "" {
		return Ownership{}, ErrOwnershipInvariant
	}
	if ttl <= 0 {
		return Ownership{}, ErrInvalidLeaseTTL
	}
	if err := attempt.Validate(); err != nil {
		return Ownership{}, err
	}
	if current.WorkflowID != "" && current.WorkflowID != workflowID {
		return Ownership{}, ErrOwnershipInvariant
	}

	switch current.State {
	case "":
		current.State = OwnershipUnowned
	case OwnershipOwned:
		if now.Before(current.LeaseUntil) {
			return Ownership{}, ErrLeaseHeld
		}
		return Ownership{}, ErrReconciliationRequired
	case OwnershipReconciling:
		return Ownership{}, ErrReconciliationRequired
	case OwnershipUnowned:
	default:
		return Ownership{}, ErrOwnershipInvariant
	}

	if current.Token == FenceToken(math.MaxUint64) {
		return Ownership{}, ErrFenceExhausted
	}

	next := Ownership{
		WorkflowID: workflowID,
		State:      OwnershipOwned,
		OwnerID:    ownerID,
		Token:      current.Token + 1,
		LeaseUntil: now.Add(ttl),
		Attempt:    attempt,
		Revision:   current.Revision + 1,
	}
	if err := next.Validate(); err != nil {
		return Ownership{}, err
	}
	return next, nil
}

func Renew(current Ownership, ownerID OwnerID, token FenceToken, now time.Time, ttl time.Duration) (Ownership, error) {
	if ttl <= 0 {
		return Ownership{}, ErrInvalidLeaseTTL
	}
	if err := validateFence(current, ownerID, token, now); err != nil {
		return Ownership{}, err
	}

	next := current
	next.LeaseUntil = now.Add(ttl)
	next.Revision++
	return next, nil
}

func ValidateWrite(current Ownership, ownerID OwnerID, token FenceToken, now time.Time) error {
	return validateFence(current, ownerID, token, now)
}

func EnterReconciliation(current Ownership, now time.Time) (Ownership, error) {
	if current.State != OwnershipOwned {
		return Ownership{}, ErrNotOwned
	}
	if now.Before(current.LeaseUntil) {
		return Ownership{}, fmt.Errorf("%w: lease active until %s", ErrLeaseHeld, current.LeaseUntil.UTC().Format(time.RFC3339Nano))
	}

	next := current
	next.State = OwnershipReconciling
	next.Revision++
	return next, nil
}

func CompleteReconciliation(current Ownership) (Ownership, error) {
	if current.State != OwnershipReconciling {
		return Ownership{}, ErrReconciliationRequired
	}

	next := current
	next.State = OwnershipUnowned
	next.OwnerID = ""
	next.LeaseUntil = time.Time{}
	next.Attempt = AttemptBinding{}
	next.Revision++
	return next, nil
}

func Release(current Ownership, ownerID OwnerID, token FenceToken, now time.Time) (Ownership, error) {
	if err := validateFence(current, ownerID, token, now); err != nil {
		return Ownership{}, err
	}

	next := current
	next.State = OwnershipUnowned
	next.OwnerID = ""
	next.LeaseUntil = time.Time{}
	next.Attempt = AttemptBinding{}
	next.Revision++
	return next, nil
}

func validateFence(current Ownership, ownerID OwnerID, token FenceToken, now time.Time) error {
	if current.State != OwnershipOwned {
		return ErrNotOwned
	}
	if current.OwnerID != ownerID || current.Token != token {
		return ErrStaleFence
	}
	if !now.Before(current.LeaseUntil) {
		return ErrLeaseExpired
	}
	return nil
}
