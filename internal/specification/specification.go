// Package specification implements the domain-only contracts for the
// canonical Requirement and Decision entities described in the approved
// AIDI v2.0 baseline (SPEC §4.1, §5.2-§5.3): the Specification -> Requirement
// hierarchy and the Known/Unknown/Assumed/Conflict/Risk/DecisionRequired/
// Derived classification model.
//
// This package is pure domain logic: no HTTP/API, no persistence, and no
// LLM/discovery integration. It mirrors the style of internal/canonical and
// internal/orchestration.
package specification

import (
	"errors"
	"fmt"
)

// RequirementID identifies a Requirement within a Specification.
type RequirementID string

// DecisionID identifies a Decision that resolves one or more Requirements.
type DecisionID string

// RequirementStatus is the classification model from SPEC §5.2:
// Known / Unknown / Assumed / Conflict / Risk / DecisionRequired / Derived.
type RequirementStatus string

const (
	StatusKnown            RequirementStatus = "KNOWN"
	StatusUnknown          RequirementStatus = "UNKNOWN"
	StatusAssumed          RequirementStatus = "ASSUMED"
	StatusConflict         RequirementStatus = "CONFLICT"
	StatusRisk             RequirementStatus = "RISK"
	StatusDecisionRequired RequirementStatus = "DECISION_REQUIRED"
	StatusDerived          RequirementStatus = "DERIVED"
)

// Severity classifies a Decision's impact, used to determine whether an
// unresolved Requirement blocks READY_FOR_APPROVAL (SPEC §5.3).
type Severity string

const (
	SeverityCritical Severity = "CRITICAL"
	SeverityNormal   Severity = "NORMAL"
)

var (
	// ErrInvalidRequirement is returned when a Requirement violates the
	// versionable/traceable/verifiable invariants required by SPEC §5.3.
	ErrInvalidRequirement = errors.New("invalid requirement")

	// ErrNonMonotonicVersion is returned when a Requirement update does not
	// strictly increase the version number.
	ErrNonMonotonicVersion = errors.New("requirement version must increase monotonically")

	// ErrInvalidDecision is returned when a Decision is missing required
	// linkage to a resolved Requirement or a valid Severity.
	ErrInvalidDecision = errors.New("invalid decision")

	// ErrInvalidStatusTransition is returned by ValidateStatusTransition when
	// a RequirementStatus transition is not permitted.
	ErrInvalidStatusTransition = errors.New("invalid requirement status transition")
)

// SourceRef traces a Requirement back to the chat/message/document that
// originated it (SPEC §5.3 — traceable to source).
type SourceRef struct {
	ChatID    string
	MessageID string
	Document  string
}

// IsEmpty reports whether the SourceRef carries no traceability information
// at all.
func (s SourceRef) IsEmpty() bool {
	return s.ChatID == "" && s.MessageID == "" && s.Document == ""
}

// Requirement is the canonical, versionable, traceable Requirement entity
// from SPEC §4.1 / §5.3.
type Requirement struct {
	ID                 RequirementID
	Version            uint64
	Status             RequirementStatus
	Source             SourceRef
	AcceptanceCriteria []string
}

// Validate checks the invariants required by SPEC §5.3: Requirements must be
// versionable (Version >= 1), traceable to an originating source, and linked
// to at least one acceptance criterion (verifiable).
func (r Requirement) Validate() error {
	if r.ID == "" {
		return fmt.Errorf("%w: empty id", ErrInvalidRequirement)
	}
	if r.Version == 0 {
		return fmt.Errorf("%w: version must be >= 1", ErrInvalidRequirement)
	}
	if !isValidStatus(r.Status) {
		return fmt.Errorf("%w: unknown status %q", ErrInvalidRequirement, r.Status)
	}
	if r.Source.IsEmpty() {
		return fmt.Errorf("%w: missing source traceability", ErrInvalidRequirement)
	}
	if len(r.AcceptanceCriteria) == 0 {
		return fmt.Errorf("%w: missing acceptance criteria linkage", ErrInvalidRequirement)
	}
	for _, ac := range r.AcceptanceCriteria {
		if ac == "" {
			return fmt.Errorf("%w: empty acceptance criteria entry", ErrInvalidRequirement)
		}
	}
	return nil
}

func isValidStatus(s RequirementStatus) bool {
	switch s {
	case StatusKnown, StatusUnknown, StatusAssumed, StatusConflict, StatusRisk, StatusDecisionRequired, StatusDerived:
		return true
	default:
		return false
	}
}

// ValidateUpdate checks that next is a valid successor of current: it must
// itself be a valid Requirement, must share the same ID, and its Version
// must strictly increase over current's Version.
func ValidateUpdate(current, next Requirement) error {
	if err := next.Validate(); err != nil {
		return err
	}
	if current.ID != "" && current.ID != next.ID {
		return fmt.Errorf("%w: id mismatch", ErrInvalidRequirement)
	}
	if next.Version <= current.Version {
		return fmt.Errorf("%w: current=%d next=%d", ErrNonMonotonicVersion, current.Version, next.Version)
	}
	return nil
}

// Decision links a resolved Requirement (or set of Requirements) to an
// outcome, carrying a Severity used by ApprovalReady (SPEC §4.1, §5.3).
type Decision struct {
	ID             DecisionID
	RequirementIDs []RequirementID
	Outcome        string
	Severity       Severity
}

// Validate checks that the Decision references at least one Requirement, has
// a non-empty outcome, and a recognized Severity.
func (d Decision) Validate() error {
	if d.ID == "" {
		return fmt.Errorf("%w: empty id", ErrInvalidDecision)
	}
	if len(d.RequirementIDs) == 0 {
		return fmt.Errorf("%w: missing requirement linkage", ErrInvalidDecision)
	}
	for _, id := range d.RequirementIDs {
		if id == "" {
			return fmt.Errorf("%w: empty requirement id", ErrInvalidDecision)
		}
	}
	if d.Outcome == "" {
		return fmt.Errorf("%w: missing outcome", ErrInvalidDecision)
	}
	switch d.Severity {
	case SeverityCritical, SeverityNormal:
	default:
		return fmt.Errorf("%w: unknown severity %q", ErrInvalidDecision, d.Severity)
	}
	return nil
}

// ApprovalReady implements SPEC §5.3 / AC-SPEC-004: a critical unresolved
// conflict blocks READY_FOR_APPROVAL. It returns false and the blocking
// RequirementIDs whenever any Requirement carries StatusConflict, and true
// with a nil slice otherwise.
func ApprovalReady(requirements []Requirement) (bool, []RequirementID) {
	var blocking []RequirementID
	for _, r := range requirements {
		if r.Status == StatusConflict {
			blocking = append(blocking, r.ID)
		}
	}
	if len(blocking) > 0 {
		return false, blocking
	}
	return true, nil
}

// ValidateStatusTransition rejects illegal RequirementStatus transitions,
// analogous to orchestration.ValidateCommandTransition.
//
// Rules (SPEC §5.2-§5.3):
//   - Same-state transitions are always allowed (idempotent).
//   - DERIVED cannot regress to UNKNOWN.
//   - Only CONFLICT, RISK, and DECISION_REQUIRED may resolve into KNOWN or
//     DERIVED.
//   - UNKNOWN may progress to ASSUMED, CONFLICT, RISK or DECISION_REQUIRED
//     as discovery narrows the classification, and to KNOWN once directly
//     confirmed.
//   - ASSUMED may progress to KNOWN (confirmed), or to CONFLICT, RISK or
//     DECISION_REQUIRED if discovery surfaces a problem.
//   - KNOWN may regress to CONFLICT, RISK or DECISION_REQUIRED if discovery
//     surfaces a problem with a previously known requirement.
func ValidateStatusTransition(current, next RequirementStatus) error {
	if !isValidStatus(current) {
		return fmt.Errorf("%w: unknown current status %q", ErrInvalidStatusTransition, current)
	}
	if !isValidStatus(next) {
		return fmt.Errorf("%w: unknown next status %q", ErrInvalidStatusTransition, next)
	}
	if current == next {
		return nil
	}

	switch current {
	case StatusUnknown:
		switch next {
		case StatusAssumed, StatusConflict, StatusRisk, StatusDecisionRequired, StatusKnown:
			return nil
		}
	case StatusAssumed:
		switch next {
		case StatusKnown, StatusConflict, StatusRisk, StatusDecisionRequired:
			return nil
		}
	case StatusKnown:
		switch next {
		case StatusConflict, StatusRisk, StatusDecisionRequired:
			return nil
		}
	case StatusConflict, StatusRisk, StatusDecisionRequired:
		switch next {
		case StatusKnown, StatusDerived, StatusConflict, StatusRisk, StatusDecisionRequired:
			return nil
		}
	case StatusDerived:
		// DERIVED cannot regress to UNKNOWN; no other transitions are
		// permitted out of DERIVED in this bounded slice.
	}

	return fmt.Errorf("%w: %s -> %s", ErrInvalidStatusTransition, current, next)
}
