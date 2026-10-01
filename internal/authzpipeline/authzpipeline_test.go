package authzpipeline

import (
	"errors"
	"strings"
	"testing"

	"github.com/ControlCenterSoft/aidi_2.0/internal/authorization"
	"github.com/ControlCenterSoft/aidi_2.0/internal/identity"
	"github.com/ControlCenterSoft/aidi_2.0/internal/policy"
	"github.com/ControlCenterSoft/aidi_2.0/internal/toolregistry"
)

// allowPolicy is a minimal well-formed Policy that ALLOWs the
// (subject, resource, action) tuple used across tests.
func allowPolicy() policy.Policy {
	return policy.Policy{
		ID: "policy-1",
		Statements: []policy.Statement{
			{Subject: "alice", Resource: "project-1", Action: "deploy", Effect: policy.EffectAllow},
		},
	}
}

// denyPolicy is a well-formed Policy with an explicit DENY statement for
// the same tuple allowPolicy ALLOWs (plus a matching ALLOW statement, to
// prove explicit-deny-overrides-allow).
func denyPolicy() policy.Policy {
	return policy.Policy{
		ID: "policy-2",
		Statements: []policy.Statement{
			{Subject: "alice", Resource: "project-1", Action: "deploy", Effect: policy.EffectAllow},
			{Subject: "alice", Resource: "project-1", Action: "deploy", Effect: policy.EffectDeny},
		},
	}
}

// noMatchPolicy is a well-formed Policy with no Statement matching the
// tuple used across tests, exercising default-deny.
func noMatchPolicy() policy.Policy {
	return policy.Policy{
		ID: "policy-3",
		Statements: []policy.Statement{
			{Subject: "bob", Resource: "project-2", Action: "read", Effect: policy.EffectAllow},
		},
	}
}

func baseRequest() Request {
	return Request{
		Role:       identity.RoleProjectOwner,
		Capability: authorization.CapabilityProjectMutate,
		Policy:     allowPolicy(),
		Subject:    "alice",
		Resource:   "project-1",
		Action:     "deploy",
		ToolID:     "tool-1",
	}
}

func registryWithTool(status toolregistry.Status) *toolregistry.Registry {
	registry := toolregistry.NewRegistry()
	err := registry.Register(toolregistry.Tool{
		ID:              "tool-1",
		Name:            "Tool One",
		Version:         "1.0.0",
		License:         "MIT",
		Status:          status,
		ProvisionPolicy: toolregistry.ProvisionPolicyAuto,
	})
	if err != nil {
		panic(err)
	}
	return registry
}

func TestAuthorize_Precedence(t *testing.T) {
	cases := []struct {
		name        string
		registry    *toolregistry.Registry
		req         Request
		wantAllowed bool
		wantReason  string // substring match; empty means "non-empty, unchecked"
	}{
		{
			name:        "allow: rbac allow, policy allow, tool active",
			registry:    registryWithTool(toolregistry.StatusActive),
			req:         baseRequest(),
			wantAllowed: true,
		},
		{
			name:     "deny: rbac denies (viewer cannot mutate) short-circuits policy and tool",
			registry: registryWithTool(toolregistry.StatusActive),
			req: func() Request {
				r := baseRequest()
				r.Role = identity.RoleViewer
				// Policy would also deny (bad tuple), proving RBAC short-circuits
				// before policy is even evaluated: if policy were evaluated this
				// would return an error instead of a clean denial.
				r.Subject = ""
				return r
			}(),
			wantAllowed: false,
			wantReason:  "rbac denied",
		},
		{
			name:     "deny: explicit policy deny overrides matching allow",
			registry: registryWithTool(toolregistry.StatusActive),
			req: func() Request {
				r := baseRequest()
				r.Policy = denyPolicy()
				return r
			}(),
			wantAllowed: false,
			wantReason:  "policy denied",
		},
		{
			name:     "deny: policy default-deny (no matching statement)",
			registry: registryWithTool(toolregistry.StatusActive),
			req: func() Request {
				r := baseRequest()
				r.Policy = noMatchPolicy()
				return r
			}(),
			wantAllowed: false,
			wantReason:  "policy denied",
		},
		{
			name:        "deny: tool not registered",
			registry:    toolregistry.NewRegistry(),
			req:         baseRequest(),
			wantAllowed: false,
			wantReason:  "not registered",
		},
		{
			name:        "deny: tool deprecated (non-active status)",
			registry:    registryWithTool(toolregistry.StatusDeprecated),
			req:         baseRequest(),
			wantAllowed: false,
			wantReason:  "not active",
		},
		{
			name:        "deny: tool disabled (non-active status)",
			registry:    registryWithTool(toolregistry.StatusDisabled),
			req:         baseRequest(),
			wantAllowed: false,
			wantReason:  "not active",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Authorize(tc.registry, tc.req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Allowed != tc.wantAllowed {
				t.Fatalf("Allowed = %v, want %v (reason: %q)", got.Allowed, tc.wantAllowed, got.Reason)
			}
			if !tc.wantAllowed {
				if got.Reason == "" {
					t.Fatalf("expected non-empty Reason for denial")
				}
				if tc.wantReason != "" && !strings.Contains(got.Reason, tc.wantReason) {
					t.Fatalf("Reason = %q, want substring %q", got.Reason, tc.wantReason)
				}
			} else if got.Reason != "" {
				t.Fatalf("expected empty Reason for allow, got %q", got.Reason)
			}
		})
	}
}

func TestAuthorize_Determinism(t *testing.T) {
	registry := registryWithTool(toolregistry.StatusActive)
	req := baseRequest()

	first, err := Authorize(registry, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for i := 0; i < 5; i++ {
		got, err := Authorize(registry, req)
		if err != nil {
			t.Fatalf("unexpected error on call %d: %v", i, err)
		}
		if got != first {
			t.Fatalf("Authorize is not deterministic: call %d = %+v, want %+v", i, got, first)
		}
	}
}

func TestAuthorize_ErrorBranches(t *testing.T) {
	registry := registryWithTool(toolregistry.StatusActive)

	cases := []struct {
		name    string
		req     Request
		wantErr error
	}{
		{
			name: "invalid role",
			req: func() Request {
				r := baseRequest()
				r.Role = "NOT_A_ROLE"
				return r
			}(),
			wantErr: identity.ErrInvalidRole,
		},
		{
			name: "invalid capability",
			req: func() Request {
				r := baseRequest()
				r.Capability = ""
				return r
			}(),
			wantErr: authorization.ErrInvalidCapability,
		},
		{
			name: "unknown capability",
			req: func() Request {
				r := baseRequest()
				r.Capability = "not.a.capability"
				return r
			}(),
			wantErr: authorization.ErrInvalidCapability,
		},
		{
			name: "invalid policy",
			req: func() Request {
				r := baseRequest()
				r.Policy = policy.Policy{}
				return r
			}(),
			wantErr: policy.ErrInvalidPolicy,
		},
		{
			name: "invalid tuple (empty subject)",
			req: func() Request {
				r := baseRequest()
				r.Subject = ""
				r.Policy = policy.Policy{
					ID: "policy-x",
					Statements: []policy.Statement{
						{Subject: "alice", Resource: "project-1", Action: "deploy", Effect: policy.EffectAllow},
					},
				}
				return r
			}(),
			wantErr: policy.ErrInvalidTuple,
		},
		{
			name: "invalid (empty) tool id",
			req: func() Request {
				r := baseRequest()
				r.ToolID = ""
				return r
			}(),
			wantErr: toolregistry.ErrInvalidTool,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Authorize(registry, tc.req)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected error wrapping %v, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestAuthorize_NilRegistry(t *testing.T) {
	// A nil *toolregistry.Registry must not panic: Registry.Get is
	// nil-safe and reports "not found", which Authorize treats as a
	// denial (bucketed with the non-ACTIVE tool case).
	got, err := Authorize(nil, baseRequest())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Allowed {
		t.Fatalf("expected denial for nil registry, got Allowed = true")
	}
	if got.Reason == "" {
		t.Fatalf("expected non-empty Reason")
	}
}
