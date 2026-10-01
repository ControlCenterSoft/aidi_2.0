package canonical

import (
	"errors"
	"fmt"
	"strings"
)

// Kind enumerates the SPEC §4.1 canonical entity hierarchy:
//
//	Installation → Identity/User → Workspace → Project → Specification →
//	Requirement → Release → Feature → Task → Workflow → Attempt →
//	ChangeSet/Verification/Evidence/Artifact, plus the additional
//	first-class entities Decision, Approval, Risk, ChangeRequest, Problem,
//	RecoveryCase, Policy, Resource, Event/Audit, OperationalKnowledge.
//
// It is a closed set: no entity kind outside SPEC §4.1 is recognized.
type Kind string

const (
	KindInstallation         Kind = "installation"
	KindIdentityUser         Kind = "identity_user"
	KindWorkspace            Kind = "workspace"
	KindProject              Kind = "project"
	KindSpecification        Kind = "specification"
	KindRequirement          Kind = "requirement"
	KindRelease              Kind = "release"
	KindFeature              Kind = "feature"
	KindTask                 Kind = "task"
	KindWorkflow             Kind = "workflow"
	KindAttempt              Kind = "attempt"
	KindChangeSet            Kind = "change_set"
	KindVerification         Kind = "verification"
	KindEvidence             Kind = "evidence"
	KindArtifact             Kind = "artifact"
	KindDecision             Kind = "decision"
	KindApproval             Kind = "approval"
	KindRisk                 Kind = "risk"
	KindChangeRequest        Kind = "change_request"
	KindProblem              Kind = "problem"
	KindRecoveryCase         Kind = "recovery_case"
	KindPolicy               Kind = "policy"
	KindResource             Kind = "resource"
	KindEventAudit           Kind = "event_audit"
	KindOperationalKnowledge Kind = "operational_knowledge"
)

// ErrInvalidKind is returned when a Kind is empty or outside the SPEC §4.1
// entity-kind set.
var ErrInvalidKind = errors.New("invalid canonical entity kind")

// ErrInvalidID is returned when an ID is empty or blank.
var ErrInvalidID = errors.New("invalid canonical identifier")

// validKinds is the closed SPEC §4.1 entity-kind set recognized by Validate.
var validKinds = map[Kind]struct{}{
	KindInstallation:         {},
	KindIdentityUser:         {},
	KindWorkspace:            {},
	KindProject:              {},
	KindSpecification:        {},
	KindRequirement:          {},
	KindRelease:              {},
	KindFeature:              {},
	KindTask:                 {},
	KindWorkflow:             {},
	KindAttempt:              {},
	KindChangeSet:            {},
	KindVerification:         {},
	KindEvidence:             {},
	KindArtifact:             {},
	KindDecision:             {},
	KindApproval:             {},
	KindRisk:                 {},
	KindChangeRequest:        {},
	KindProblem:              {},
	KindRecoveryCase:         {},
	KindPolicy:               {},
	KindResource:             {},
	KindEventAudit:           {},
	KindOperationalKnowledge: {},
}

// Validate rejects an empty Kind and any value outside the SPEC §4.1
// entity-kind set.
func (k Kind) Validate() error {
	if strings.TrimSpace(string(k)) == "" {
		return fmt.Errorf("%w: kind is required", ErrInvalidKind)
	}
	if _, ok := validKinds[k]; !ok {
		return fmt.Errorf("%w: unknown kind %q", ErrInvalidKind, string(k))
	}
	return nil
}

// ID is the canonical identifier type shared by every domain object that
// references an entity.
type ID string

// Validate rejects an empty or whitespace-only ID.
func (id ID) Validate() error {
	if strings.TrimSpace(string(id)) == "" {
		return fmt.Errorf("%w: id is required", ErrInvalidID)
	}
	return nil
}

// ObjectRef is the single canonical Kind/ID reference used by every
// canonical object that points at an entity (e.g. Event.Object,
// ChangeSet.Target). It replaces the previously duplicated, free-form
// Kind-string ObjectRef shapes.
type ObjectRef struct {
	Kind Kind `json:"kind"`
	ID   ID   `json:"id"`
}

// Validate checks that the referenced Kind is recognized and the ID is
// non-empty.
func (r ObjectRef) Validate() error {
	if err := r.Kind.Validate(); err != nil {
		return err
	}
	return r.ID.Validate()
}
