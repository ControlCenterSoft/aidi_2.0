// Package policy defines the transport-independent Foundation Policy
// domain contract required by SPEC §4.1 ("Installation → Identity/User →
// Workspace → Project → ... → Policy → ...") and SPEC §12.1 ("Identity →
// Authentication → Role → Policy → Authorization → Action → Audit"; "RBAC
// дополняется policy engine; explicit deny overrides allow"). Before this
// slice there was no domain type at all representing a Policy as a
// first-class canonical entity.
//
// This package defines pure domain types and invariants: a PolicyID
// identifier (aligned with internal/canonical identifier conventions), a
// closed-set Effect (ALLOW/DENY), a Subject/Resource/Action/Effect
// Statement tuple, a Policy aggregate (an ordered list of Statements)
// with a Validate method, and a pure Evaluate function (backlog A5-002)
// that resolves the ALLOW/DENY Decision for a (Subject, Resource, Action)
// tuple against a Policy's ordered Statements, implementing "explicit
// deny overrides allow" (SPEC §12.1) and default-deny when no Statement
// matches. Policy.Validate rejects malformed input; it does not resolve
// ALLOW/DENY conflicts between Statements — that is Evaluate's job.
// Policy also exposes HasExplicitDeny, which Evaluate uses internally so
// callers needing just the deny check need not re-derive tuple matching.
//
// This package deliberately excludes policy revision/versioning
// persistence (backlog A5-003), the Authorization→Policy→Execution
// pipeline (A5-005), wildcard/glob matching beyond exact tuple equality,
// and any HTTP/persistence/queue wiring. No file in this package imports
// net/http, database/sql, NATS, Temporal, or any VM/runner/queue package,
// and the package introduces no new external module dependency, mirroring
// internal/authorization and internal/toolregistry.
package policy

import (
	"errors"
	"fmt"
	"strings"
)

// PolicyID is the canonical identifier of a Policy, aligned with the
// internal/canonical identifier conventions (a non-empty, non-blank
// string).
type PolicyID string

// ErrInvalidPolicyID is returned (wrapped) when a PolicyID is empty or
// whitespace-only.
var ErrInvalidPolicyID = errors.New("policy: invalid policy id")

// Validate rejects an empty or whitespace-only PolicyID.
func (id PolicyID) Validate() error {
	if strings.TrimSpace(string(id)) == "" {
		return fmt.Errorf("%w: policy id is required", ErrInvalidPolicyID)
	}
	return nil
}

// Effect is the closed set of outcomes a Statement may declare for a
// (Subject, Resource, Action) tuple (SPEC §12.1).
type Effect string

const (
	// EffectAllow permits the (Subject, Resource, Action) tuple.
	EffectAllow Effect = "ALLOW"
	// EffectDeny forbids the (Subject, Resource, Action) tuple. SPEC §12.1
	// requires that an explicit DENY overrides an ALLOW for the identical
	// tuple; this package only exposes HasExplicitDeny to support that
	// resolution, it does not perform the resolution itself.
	EffectDeny Effect = "DENY"
)

// ErrInvalidEffect is returned (wrapped) when an Effect is empty or outside
// the closed ALLOW/DENY set.
var ErrInvalidEffect = errors.New("policy: invalid effect")

// validEffects is the closed Effect set recognized by Validate.
var validEffects = map[Effect]struct{}{
	EffectAllow: {},
	EffectDeny:  {},
}

// Validate rejects an empty Effect and any value outside the closed
// ALLOW/DENY set. It never panics.
func (e Effect) Validate() error {
	if strings.TrimSpace(string(e)) == "" {
		return fmt.Errorf("%w: effect is required", ErrInvalidEffect)
	}
	if _, ok := validEffects[e]; !ok {
		return fmt.Errorf("%w: unknown effect %q", ErrInvalidEffect, string(e))
	}
	return nil
}

// Statement is a single Subject/Resource/Action/Effect tuple. It
// deliberately has named fields only: no wildcard/glob expansion logic,
// which belongs to later evaluation work (backlog A5-002).
type Statement struct {
	Subject  string `json:"subject"`
	Resource string `json:"resource"`
	Action   string `json:"action"`
	Effect   Effect `json:"effect"`
}

// Policy is the first-class canonical Policy aggregate (SPEC §4.1,
// §12.1): an ID and an ordered list of Statements.
type Policy struct {
	ID         PolicyID    `json:"id"`
	Statements []Statement `json:"statements"`
}

// ErrInvalidPolicy is returned (wrapped) when a Policy fails validation: an
// invalid PolicyID, an empty Statements list, a Statement with a blank
// Subject/Resource/Action, or a Statement with an unknown Effect.
var ErrInvalidPolicy = errors.New("policy: invalid policy")

// Validate rejects an invalid PolicyID, an empty Statements list, any
// Statement with a blank Subject/Resource/Action, and any Statement with
// an unknown Effect — each via the wrapped sentinel error ErrInvalidPolicy.
// Validate does not resolve conflicts between an ALLOW and a DENY
// Statement sharing the identical (Subject, Resource, Action) tuple: such
// a policy is structurally valid. Validate never panics.
func (p Policy) Validate() error {
	if err := p.ID.Validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidPolicy, err)
	}
	if len(p.Statements) == 0 {
		return fmt.Errorf("%w: statements is required", ErrInvalidPolicy)
	}
	for i, stmt := range p.Statements {
		if strings.TrimSpace(stmt.Subject) == "" {
			return fmt.Errorf("%w: statement %d: subject is required", ErrInvalidPolicy, i)
		}
		if strings.TrimSpace(stmt.Resource) == "" {
			return fmt.Errorf("%w: statement %d: resource is required", ErrInvalidPolicy, i)
		}
		if strings.TrimSpace(stmt.Action) == "" {
			return fmt.Errorf("%w: statement %d: action is required", ErrInvalidPolicy, i)
		}
		if err := stmt.Effect.Validate(); err != nil {
			return fmt.Errorf("%w: statement %d: %v", ErrInvalidPolicy, i, err)
		}
	}
	return nil
}

// HasExplicitDeny reports whether p contains a DENY Statement for the
// exact (subject, resource, action) tuple. It is deterministic, performs
// no I/O, and never panics on a zero-value Policy (it returns false).
// Callers (the A5-002 Evaluate function below) can use it to implement
// "explicit deny overrides allow" (SPEC §12.1) without re-deriving tuple
// matching.
func (p Policy) HasExplicitDeny(subject, resource, action string) bool {
	for _, stmt := range p.Statements {
		if stmt.Effect == EffectDeny &&
			stmt.Subject == subject &&
			stmt.Resource == resource &&
			stmt.Action == action {
			return true
		}
	}
	return false
}

// Decision is the deterministic outcome of evaluating a Policy against a
// (Subject, Resource, Action) tuple (SPEC §12.1, backlog A5-002). Reason
// is non-empty whenever Allowed is false, explaining why the tuple was
// denied (explicit DENY statement, or no matching statement at all).
type Decision struct {
	Allowed bool   `json:"allowed"`
	Reason  string `json:"reason,omitempty"`
}

// ErrInvalidTuple is returned (wrapped) by Evaluate when subject, resource,
// or action is empty/whitespace-only.
var ErrInvalidTuple = errors.New("policy: invalid tuple")

// Evaluate resolves the ALLOW/DENY Decision for the (subject, resource,
// action) tuple against policy's ordered Statements, per SPEC §12.1 ("RBAC
// дополняется policy engine; explicit deny overrides allow"):
//
//   - If any Statement matching the exact tuple has Effect == EffectDeny,
//     the result is Allowed: false with a non-empty Reason, regardless of
//     any matching ALLOW statement (explicit deny overrides allow).
//   - Else if at least one Statement matching the exact tuple has
//     Effect == EffectAllow, the result is Allowed: true.
//   - Else (no Statement matches the tuple), the result is a default-deny:
//     Allowed: false with a Reason explaining no matching statement was
//     found.
//
// Evaluate rejects an invalid policy (via Policy.Validate()) and an empty
// subject/resource/action with a wrapped sentinel error
// (ErrInvalidPolicy or ErrInvalidTuple) and never panics. It is a pure
// function: no I/O, no clock, no global/singleton state — identical
// inputs always produce an identical Decision.
func Evaluate(policy Policy, subject, resource, action string) (Decision, error) {
	if err := policy.Validate(); err != nil {
		return Decision{}, err
	}
	if strings.TrimSpace(subject) == "" {
		return Decision{}, fmt.Errorf("%w: subject is required", ErrInvalidTuple)
	}
	if strings.TrimSpace(resource) == "" {
		return Decision{}, fmt.Errorf("%w: resource is required", ErrInvalidTuple)
	}
	if strings.TrimSpace(action) == "" {
		return Decision{}, fmt.Errorf("%w: action is required", ErrInvalidTuple)
	}

	if policy.HasExplicitDeny(subject, resource, action) {
		return Decision{
			Allowed: false,
			Reason:  fmt.Sprintf("explicit deny for subject %q, resource %q, action %q", subject, resource, action),
		}, nil
	}

	for _, stmt := range policy.Statements {
		if stmt.Effect == EffectAllow &&
			stmt.Subject == subject &&
			stmt.Resource == resource &&
			stmt.Action == action {
			return Decision{Allowed: true}, nil
		}
	}

	return Decision{
		Allowed: false,
		Reason:  fmt.Sprintf("no matching statement for subject %q, resource %q, action %q", subject, resource, action),
	}, nil
}
