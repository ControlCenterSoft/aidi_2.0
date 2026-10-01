// Package policy defines the transport-independent Foundation Policy
// domain contract required by SPEC §4.1 ("Installation → Identity/User →
// Workspace → Project → ... → Policy → ...") and SPEC §12.1 ("Identity →
// Authentication → Role → Policy → Authorization → Action → Audit"; "RBAC
// дополняется policy engine; explicit deny overrides allow"). Before this
// slice there was no domain type at all representing a Policy as a
// first-class canonical entity.
//
// This package defines pure domain types and invariants only: a PolicyID
// identifier (aligned with internal/canonical identifier conventions), a
// closed-set Effect (ALLOW/DENY), a Subject/Resource/Action/Effect
// Statement tuple, and a Policy aggregate (an ordered list of Statements)
// with a Validate method. Validate rejects malformed input; it does not
// resolve ALLOW/DENY conflicts between Statements. Policy additionally
// exposes HasExplicitDeny so a later evaluator can implement "explicit
// deny overrides allow" (SPEC §12.1) without re-deriving tuple matching.
//
// This package deliberately excludes policy evaluation/decisioning
// (backlog A5-002), revision/versioning persistence (A5-003), the
// Authorization→Policy→Execution pipeline (A5-005), and any HTTP/
// persistence/queue wiring. No file in this package imports net/http,
// database/sql, NATS, Temporal, or any VM/runner/queue package, and the
// package introduces no new external module dependency, mirroring
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
// Callers (a later A5-002 evaluator) can use it to implement "explicit
// deny overrides allow" (SPEC §12.1) without re-deriving tuple matching.
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
