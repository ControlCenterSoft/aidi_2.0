package specification

import (
	"errors"
	"fmt"
)

// ApprovalID identifies an Approval decision on a Specification.
type ApprovalID string

// ApprovalDecision is the outcome of an approval review (SPEC §5.5).
type ApprovalDecision string

const (
	DecisionApproved ApprovalDecision = "APPROVED"
	DecisionRejected ApprovalDecision = "REJECTED"
)

// ErrInvalidApproval is returned when an Approval violates the invariants
// required by SPEC §5.5/§9.5: required identifiers, a positive exact
// Specification version, a known decision value, and a non-empty approver.
var ErrInvalidApproval = errors.New("invalid approval")

// ErrApprovalNotReady is returned when an Approval is constructed while its
// target Specification still has outstanding Requirements that block
// READY_FOR_APPROVAL (see ApprovalReady, AC-SPEC-004).
var ErrApprovalNotReady = errors.New("approval blocked: specification not ready for approval")

// ErrApprovalVersionMismatch is returned by ApprovalCurrentForVersion when an
// Approval's bound SpecificationVersion does not match the Specification's
// current version (SPEC §5.5, AC-SPEC-009: "Approval всегда относится к
// exact Specification version").
var ErrApprovalVersionMismatch = errors.New("approval does not authorize this specification version")

// Approval is the canonical, first-class Approval entity from SPEC §4.1:
// it always attaches to an exact SpecificationVersion, mirroring the
// Requirement.Version pattern. Once approved, further changes are only
// permitted through a Change Request — not a silent edit of the same
// Approval or Specification version.
type Approval struct {
	ID                   ApprovalID
	SpecificationID      string
	SpecificationVersion uint64
	Approver             string
	Decision             ApprovalDecision
}

// Validate checks the invariants required by SPEC §5.5/§9.5: a non-empty
// ApprovalID, a non-empty target SpecificationID, a positive
// SpecificationVersion (the exact version being approved/rejected), a known
// ApprovalDecision, and a non-empty Approver identity.
func (a Approval) Validate() error {
	if a.ID == "" {
		return fmt.Errorf("%w: empty id", ErrInvalidApproval)
	}
	if a.SpecificationID == "" {
		return fmt.Errorf("%w: empty specification id", ErrInvalidApproval)
	}
	if a.SpecificationVersion == 0 {
		return fmt.Errorf("%w: specification version must be >= 1", ErrInvalidApproval)
	}
	switch a.Decision {
	case DecisionApproved, DecisionRejected:
	default:
		return fmt.Errorf("%w: unknown decision %q", ErrInvalidApproval, a.Decision)
	}
	if a.Approver == "" {
		return fmt.Errorf("%w: empty approver", ErrInvalidApproval)
	}
	return nil
}

// NewApproval constructs and validates an Approval, composing with
// ApprovalReady (SPEC §5.3/AC-SPEC-004) as a precondition: an Approval
// cannot be constructed while any Requirement of the target Specification
// carries a CONFLICT status. It returns ErrApprovalNotReady (wrapping the
// blocking RequirementIDs in the error message) when that precondition
// fails, or the result of Validate otherwise.
func NewApproval(a Approval, requirements []Requirement) (Approval, error) {
	if ready, blocking := ApprovalReady(requirements); !ready {
		return Approval{}, fmt.Errorf("%w: blocking=%v", ErrApprovalNotReady, blocking)
	}
	if err := a.Validate(); err != nil {
		return Approval{}, err
	}
	return a, nil
}

// ApprovalCurrentForVersion reports whether approval still authorizes the
// given current Specification version. An Approval bound to version N does
// not silently cover version N+1 (SPEC §5.5, AC-SPEC-009): once the
// Specification advances beyond the approved version, further changes must
// go through a Change Request rather than being treated as still approved.
//
// It first validates approval itself; a malformed Approval is never
// considered current.
func ApprovalCurrentForVersion(approval Approval, currentSpecificationVersion uint64) error {
	if err := approval.Validate(); err != nil {
		return err
	}
	if approval.SpecificationVersion != currentSpecificationVersion {
		return fmt.Errorf("%w: approved=%d current=%d", ErrApprovalVersionMismatch, approval.SpecificationVersion, currentSpecificationVersion)
	}
	return nil
}
