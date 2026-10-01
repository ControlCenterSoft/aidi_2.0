package canonical

import (
	"errors"
	"fmt"
	"strings"

	"github.com/ControlCenterSoft/aidi_2.0/internal/orchestration"
)

// ChangeSetState models the SPEC §8.1/§8.2 contract that generated code does
// not modify the canonical branch directly: a ChangeSet is submitted,
// validated and only then merged.
type ChangeSetState string

const (
	ChangeSetDraft      ChangeSetState = "DRAFT"
	ChangeSetSubmitted  ChangeSetState = "SUBMITTED"
	ChangeSetValidating ChangeSetState = "VALIDATING"
	ChangeSetValidated  ChangeSetState = "VALIDATED"
	ChangeSetRejected   ChangeSetState = "REJECTED"
	ChangeSetMerged     ChangeSetState = "MERGED"
)

var (
	ErrInvalidChangeSetTransition = errors.New("invalid changeset lifecycle transition")
	ErrChangeSetInvariant         = errors.New("changeset invariant violation")
)

// EvidenceRef identifies a piece of verification/integration evidence (test
// run, deterministic check, review) attached to a ChangeSet.
type EvidenceRef struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

func (e EvidenceRef) Validate() error {
	switch {
	case strings.TrimSpace(e.Kind) == "":
		return fmt.Errorf("%w: evidence kind is required", ErrChangeSetInvariant)
	case strings.TrimSpace(e.ID) == "":
		return fmt.Errorf("%w: evidence id is required", ErrChangeSetInvariant)
	default:
		return nil
	}
}

// ChangeSet is the SPEC §8.1/§8.2 unit of integration: generated code from an
// Attempt is bound to a target canonical object and a base Revision, and must
// carry verification evidence before it may reach VALIDATED or MERGED
// (ACCEPTANCE.md AC-EXEC-003, AC-EXEC-006).
type ChangeSet struct {
	ChangeSetID  string                       `json:"changeset_id"`
	Attempt      orchestration.AttemptBinding `json:"attempt"`
	Target       ObjectRef                    `json:"target"`
	BaseRevision Revision                     `json:"base_revision"`
	State        ChangeSetState               `json:"state"`
	Evidence     []EvidenceRef                `json:"evidence,omitempty"`
}

// Validate checks the structural invariants of the ChangeSet: required
// AttemptBinding coordinates, a target object reference, a base revision, a
// known lifecycle state and, once VALIDATED or MERGED, at least one evidence
// reference.
func (c ChangeSet) Validate() error {
	if strings.TrimSpace(c.ChangeSetID) == "" {
		return fmt.Errorf("%w: changeset id is required", ErrChangeSetInvariant)
	}
	if err := c.Attempt.Validate(); err != nil {
		return fmt.Errorf("%w: %s", ErrChangeSetInvariant, err)
	}
	if err := c.Target.Validate(); err != nil {
		return fmt.Errorf("%w: target object ref is invalid: %w", ErrChangeSetInvariant, err)
	}
	if c.BaseRevision == 0 {
		return fmt.Errorf("%w: base revision is required", ErrChangeSetInvariant)
	}
	if err := c.validateKnownState(); err != nil {
		return err
	}
	if c.State == ChangeSetValidated || c.State == ChangeSetMerged {
		if len(c.Evidence) == 0 {
			return fmt.Errorf("%w: %s requires at least one evidence reference", ErrChangeSetInvariant, c.State)
		}
	}
	for _, evidence := range c.Evidence {
		if err := evidence.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (c ChangeSet) validateKnownState() error {
	switch c.State {
	case ChangeSetDraft, ChangeSetSubmitted, ChangeSetValidating, ChangeSetValidated, ChangeSetRejected, ChangeSetMerged:
		return nil
	default:
		return fmt.Errorf("%w: unknown state %q", ErrChangeSetInvariant, c.State)
	}
}

// ValidateTransition allows only the documented lifecycle edges:
//
//	DRAFT -> DRAFT (explicit no-op, editing before submission)
//	DRAFT -> SUBMITTED
//	SUBMITTED -> VALIDATING
//	VALIDATING -> VALIDATED | REJECTED
//	VALIDATED -> MERGED
//
// REJECTED and MERGED are terminal: no transition out of them, including to
// the same state, is allowed.
func ValidateTransition(current, next ChangeSetState) error {
	switch current {
	case ChangeSetDraft:
		if next == ChangeSetDraft || next == ChangeSetSubmitted {
			return nil
		}
	case ChangeSetSubmitted:
		if next == ChangeSetValidating {
			return nil
		}
	case ChangeSetValidating:
		if next == ChangeSetValidated || next == ChangeSetRejected {
			return nil
		}
	case ChangeSetValidated:
		if next == ChangeSetMerged {
			return nil
		}
	case ChangeSetRejected, ChangeSetMerged:
		// terminal: no outgoing transitions, not even a same-state no-op.
	}
	return fmt.Errorf("%w: %s -> %s", ErrInvalidChangeSetTransition, current, next)
}

// Transition moves the ChangeSet to next if the edge is allowed and the
// resulting ChangeSet satisfies Validate (e.g. evidence required to reach
// VALIDATED).
func (c ChangeSet) Transition(next ChangeSetState) (ChangeSet, error) {
	if err := ValidateTransition(c.State, next); err != nil {
		return ChangeSet{}, err
	}
	updated := c
	updated.State = next
	if err := updated.Validate(); err != nil {
		return ChangeSet{}, err
	}
	return updated, nil
}

// Merge transitions a VALIDATED ChangeSet to MERGED. It is rejected unless
// currentRevision still matches the ChangeSet's BaseRevision (via
// CheckExpectedRevision), so a stale ChangeSet must reconcile rather than
// silently merge over intervening canonical changes.
func (c ChangeSet) Merge(currentRevision Revision) (ChangeSet, error) {
	updated, err := c.Transition(ChangeSetMerged)
	if err != nil {
		return ChangeSet{}, err
	}
	if err := CheckExpectedRevision(c.BaseRevision, currentRevision); err != nil {
		return ChangeSet{}, err
	}
	return updated, nil
}
