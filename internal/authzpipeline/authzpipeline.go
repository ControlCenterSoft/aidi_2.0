// Package authzpipeline defines the transport-independent
// Authorization → Policy → Execution pre-execution gate (backlog
// A5-005, SPEC §12.1: "Identity → Authentication → Role → Policy →
// Authorization → Action → Audit"; SPEC §2.4: significant decisions must
// be explainable). Release A already delivers the three contracts this
// slice composes:
//
//   - internal/authorization.Evaluate(role, capability) — the RBAC
//     capability check (A5-002).
//   - internal/policy.Evaluate(policy, subject, resource, action) — the
//     ALLOW/DENY policy check with explicit-deny-overrides-allow and
//     default-deny (A5-002).
//   - internal/toolregistry.Registry.Get(toolID) — tool lookup, admitting
//     execution only when Tool.Status == toolregistry.StatusActive
//     (A5-004).
//
// Authorize is a pure function composing these three contracts into a
// single combined Decision with a fixed, deterministic precedence order:
//
//  1. RBAC denial (authorization.Evaluate) short-circuits first: if the
//     Role does not hold the Capability, Authorize returns a denial
//     immediately without evaluating the policy or the tool.
//  2. Explicit policy DENY, then
//  3. default-deny-if-no-matching-statement (both resolved by a single
//     policy.Evaluate call, which already implements SPEC §12.1's
//     "explicit deny overrides allow").
//  4. Non-ACTIVE or unregistered tool: the ToolID must resolve to a
//     registered Tool whose Status is toolregistry.StatusActive.
//
// Authorize allows the request only when every one of the above checks
// passes: RBAC allow AND policy allow AND toolregistry.StatusActive.
// Every denial path returns Allowed: false with a non-empty, distinct
// Reason explaining which stage denied the request (SPEC §2.4). It is a
// pure function: no I/O, no clock, no goroutines, no global/singleton
// state — identical inputs always yield an identical Decision.
//
// A malformed Role, Capability, Policy, (subject, resource, action)
// tuple, or ToolID returns a wrapped sentinel error from the underlying
// package (identity.ErrInvalidRole, authorization.ErrInvalidCapability,
// policy.ErrInvalidPolicy, policy.ErrInvalidTuple,
// toolregistry.ErrInvalidTool), never a panic. A well-formed ToolID that
// is simply not registered in the Registry is not a programming error:
// it is treated as a denial (bucketed with the non-ACTIVE tool case)
// with a distinguishing Reason.
//
// This package deliberately excludes a persisted audit trail,
// wildcard/glob matching (both already out of scope for
// internal/policy), and SUPER_ADMIN global-role handling — those remain
// later slices. No file in this package imports net/http, database/sql,
// NATS, Temporal, or any VM/runner/queue package, and it introduces no
// new external module dependency: it imports only internal/authorization,
// internal/policy, internal/toolregistry, and internal/identity.
package authzpipeline

import (
	"fmt"

	"github.com/ControlCenterSoft/aidi_2.0/internal/authorization"
	"github.com/ControlCenterSoft/aidi_2.0/internal/identity"
	"github.com/ControlCenterSoft/aidi_2.0/internal/policy"
	"github.com/ControlCenterSoft/aidi_2.0/internal/toolregistry"
)

// Request bundles every input Authorize needs: the RBAC (Role,
// Capability) pair, the Policy and the (Subject, Resource, Action) tuple
// it is evaluated against, and the ToolID whose registered Status gates
// execution.
type Request struct {
	Role       identity.Role
	Capability authorization.Capability
	Policy     policy.Policy
	Subject    string
	Resource   string
	Action     string
	ToolID     toolregistry.ToolID
}

// Decision is the combined outcome of the Authorization → Policy →
// Execution pipeline. Reason is always non-empty when Allowed is false,
// mirroring the Decision shape already used by
// internal/authorization.Decision and internal/policy.Decision.
type Decision struct {
	Allowed bool   `json:"allowed"`
	Reason  string `json:"reason,omitempty"`
}

// Authorize evaluates req against registry and returns a combined
// Decision with the fixed precedence order documented at package level:
// RBAC denial, then explicit policy DENY, then policy default-deny, then
// non-ACTIVE/unregistered tool. The only Allowed: true outcome requires
// RBAC allow, policy allow, and a registered Tool with
// toolregistry.StatusActive.
//
// Authorize is a pure function: it performs no I/O, reads no clock, and
// holds no global/singleton state (registry is read-only via
// Registry.Get, which performs no mutation). Identical (registry, req)
// inputs always yield an identical Decision. A malformed Role,
// Capability, Policy, tuple, or ToolID returns a wrapped sentinel error,
// never a panic.
func Authorize(registry *toolregistry.Registry, req Request) (Decision, error) {
	rbac, err := authorization.Evaluate(req.Role, req.Capability)
	if err != nil {
		return Decision{}, err
	}
	if !rbac.Allowed {
		return Decision{
			Allowed: false,
			Reason:  fmt.Sprintf("rbac denied: %s", rbac.Reason),
		}, nil
	}

	pol, err := policy.Evaluate(req.Policy, req.Subject, req.Resource, req.Action)
	if err != nil {
		return Decision{}, err
	}
	if !pol.Allowed {
		return Decision{
			Allowed: false,
			Reason:  fmt.Sprintf("policy denied: %s", pol.Reason),
		}, nil
	}

	if err := req.ToolID.Validate(); err != nil {
		return Decision{}, err
	}

	tool, ok := registry.Get(req.ToolID)
	if !ok {
		return Decision{
			Allowed: false,
			Reason:  fmt.Sprintf("tool %q is not registered", req.ToolID),
		}, nil
	}
	if tool.Status != toolregistry.StatusActive {
		return Decision{
			Allowed: false,
			Reason:  fmt.Sprintf("tool %q status %q is not active", req.ToolID, tool.Status),
		}, nil
	}

	return Decision{Allowed: true}, nil
}
