// Package foundationtest provides shared, deterministic test fixture
// builders for the Release A Policy/Authorization/Tool-Broker stack
// (`internal/authorization`, `internal/policy`, `internal/toolregistry`,
// `internal/authzpipeline`), closing backlog card `A5-006`.
//
// Before this slice, `authzpipeline_test.go` and sibling test files each
// hand-rolled near-identical `allowPolicy()`, `denyPolicy()`,
// `noMatchPolicy()`, `baseRequest()`-style fixtures. This package collects
// that duplicated shape into a single, additive, test-support package so
// both `_test.go` files and any non-test caller that needs a canned,
// already-valid fixture can reuse it instead of re-deriving it.
//
// It exports constructors for: a minimal valid `policy.Policy` in each of
// the three shapes SPEC §12.1 requires reasoning about (explicit ALLOW,
// explicit DENY that overrides a matching ALLOW, and default-deny/no
// matching statement); a registered `toolregistry.Tool` in
// `toolregistry.StatusActive` and one in a non-active state
// (`toolregistry.StatusDeprecated`) for negative cases; and a baseline
// `authzpipeline.Request` composing the above with a valid
// `identity.Role`/`authorization.Capability` pair.
//
// This package deliberately excludes any new runtime behavior: every
// constructor returns a value already accepted by its own domain type's
// `Validate()`/registration path (see `foundationtest_test.go`). It
// introduces no HTTP, PostgreSQL, NATS, Temporal, Forgejo, VM, runner, or
// queue dependency, and no new external module dependency: it imports only
// `internal/authorization`, `internal/identity`, `internal/policy`,
// `internal/toolregistry`, and `internal/authzpipeline` (a subset of the
// permitted `internal/canonical` + those five — this slice has no fixture
// that needs `internal/canonical` directly), mirroring the existing
// `internal/toolregistry`/`internal/authzpipeline` scope-boundary style.
//
// Wiring these fixtures into the existing hand-rolled test files is
// explicitly out of scope for this slice (a follow-up), as is
// `RA-GATE-10` itself (the gate closes only after its own evidence pass).
package foundationtest

import (
	"github.com/ControlCenterSoft/aidi_2.0/internal/authorization"
	"github.com/ControlCenterSoft/aidi_2.0/internal/authzpipeline"
	"github.com/ControlCenterSoft/aidi_2.0/internal/identity"
	"github.com/ControlCenterSoft/aidi_2.0/internal/policy"
	"github.com/ControlCenterSoft/aidi_2.0/internal/toolregistry"
)

// The following constants are the single (subject, resource, action) tuple
// and (role, capability) pair shared by every fixture in this package, so
// that AllowPolicy/DenyPolicy/NoMatchPolicy/BaseRequest compose without the
// caller having to keep them in sync by hand.
const (
	// Subject is the canonical subject used by AllowPolicy, DenyPolicy and
	// BaseRequest.
	Subject = "alice"
	// Resource is the canonical resource used by AllowPolicy, DenyPolicy
	// and BaseRequest.
	Resource = "project-1"
	// Action is the canonical action used by AllowPolicy, DenyPolicy and
	// BaseRequest.
	Action = "deploy"

	// Role is a valid identity.Role (SPEC §3.2) that, paired with
	// Capability, is granted by the capabilityMatrix (SPEC §3.2 role
	// semantics documented in internal/authorization).
	Role = identity.RoleProjectOwner
	// Capability is a valid authorization.Capability that Role holds.
	Capability = authorization.CapabilityProjectMutate

	// ActiveToolID is the ToolID registered by NewActiveRegistry and used
	// by BaseRequest.
	ActiveToolID = toolregistry.ToolID("tool-1")
	// InactiveToolID is the ToolID registered by NewInactiveRegistry,
	// carrying a non-active Status for negative test cases.
	InactiveToolID = toolregistry.ToolID("tool-2")
)

// AllowPolicy returns a minimal, independently valid policy.Policy with a
// single ALLOW Statement for (Subject, Resource, Action).
func AllowPolicy() policy.Policy {
	return policy.Policy{
		ID: "foundationtest-allow-policy",
		Statements: []policy.Statement{
			{Subject: Subject, Resource: Resource, Action: Action, Effect: policy.EffectAllow},
		},
	}
}

// DenyPolicy returns a minimal, independently valid policy.Policy with a
// matching ALLOW Statement and an explicit DENY Statement for the same
// (Subject, Resource, Action) tuple, exercising SPEC §12.1's "explicit deny
// overrides allow".
func DenyPolicy() policy.Policy {
	return policy.Policy{
		ID: "foundationtest-deny-policy",
		Statements: []policy.Statement{
			{Subject: Subject, Resource: Resource, Action: Action, Effect: policy.EffectAllow},
			{Subject: Subject, Resource: Resource, Action: Action, Effect: policy.EffectDeny},
		},
	}
}

// NoMatchPolicy returns a minimal, independently valid policy.Policy whose
// only Statement does not match (Subject, Resource, Action), exercising
// policy.Evaluate's default-deny path.
func NoMatchPolicy() policy.Policy {
	return policy.Policy{
		ID: "foundationtest-nomatch-policy",
		Statements: []policy.Statement{
			{Subject: "bob", Resource: "project-2", Action: "read", Effect: policy.EffectAllow},
		},
	}
}

// ActiveTool returns a minimal, independently valid toolregistry.Tool in
// toolregistry.StatusActive, identified by ActiveToolID.
func ActiveTool() toolregistry.Tool {
	return toolregistry.Tool{
		ID:              ActiveToolID,
		Name:            "Foundation Test Tool",
		Version:         "1.0.0",
		License:         "MIT",
		Status:          toolregistry.StatusActive,
		ProvisionPolicy: toolregistry.ProvisionPolicyAuto,
	}
}

// InactiveTool returns a minimal, independently valid toolregistry.Tool in
// a non-active Status (toolregistry.StatusDeprecated), identified by
// InactiveToolID, for negative test cases.
func InactiveTool() toolregistry.Tool {
	return toolregistry.Tool{
		ID:              InactiveToolID,
		Name:            "Foundation Test Deprecated Tool",
		Version:         "1.0.0",
		License:         "MIT",
		Status:          toolregistry.StatusDeprecated,
		ProvisionPolicy: toolregistry.ProvisionPolicyAuto,
	}
}

// NewActiveRegistry returns a *toolregistry.Registry with ActiveTool
// already registered. It panics if registration fails, which would only
// happen if this package's own fixtures became malformed (a programming
// error, never a caller input).
func NewActiveRegistry() *toolregistry.Registry {
	registry := toolregistry.NewRegistry()
	if err := registry.Register(ActiveTool()); err != nil {
		panic(err)
	}
	return registry
}

// NewInactiveRegistry returns a *toolregistry.Registry with InactiveTool
// already registered. It panics if registration fails, which would only
// happen if this package's own fixtures became malformed (a programming
// error, never a caller input).
func NewInactiveRegistry() *toolregistry.Registry {
	registry := toolregistry.NewRegistry()
	if err := registry.Register(InactiveTool()); err != nil {
		panic(err)
	}
	return registry
}

// BaseRequest returns a baseline, independently valid
// authzpipeline.Request: a valid (Role, Capability) pair, AllowPolicy, the
// (Subject, Resource, Action) tuple AllowPolicy ALLOWs, and ActiveToolID.
// Pairing BaseRequest with NewActiveRegistry() yields an Allowed: true
// authzpipeline.Authorize outcome; callers exercising a denial path
// override individual fields (Policy, ToolID, Role) as needed.
func BaseRequest() authzpipeline.Request {
	return authzpipeline.Request{
		Role:       Role,
		Capability: Capability,
		Policy:     AllowPolicy(),
		Subject:    Subject,
		Resource:   Resource,
		Action:     Action,
		ToolID:     ActiveToolID,
	}
}
