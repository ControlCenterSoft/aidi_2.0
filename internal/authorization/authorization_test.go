package authorization

import (
	"errors"
	"testing"

	"github.com/ControlCenterSoft/aidi_2.0/internal/identity"
)

func TestEvaluate_AllowDenyPerRole(t *testing.T) {
	cases := []struct {
		name       string
		role       identity.Role
		capability Capability
		wantAllow  bool
	}{
		{name: "owner read", role: identity.RoleProjectOwner, capability: CapabilityProjectRead, wantAllow: true},
		{name: "owner mutate", role: identity.RoleProjectOwner, capability: CapabilityProjectMutate, wantAllow: true},
		{name: "manager read", role: identity.RoleProjectManager, capability: CapabilityProjectRead, wantAllow: true},
		{name: "manager mutate", role: identity.RoleProjectManager, capability: CapabilityProjectMutate, wantAllow: true},
		{name: "product owner read", role: identity.RoleProductOwner, capability: CapabilityProjectRead, wantAllow: true},
		{name: "product owner mutate", role: identity.RoleProductOwner, capability: CapabilityProjectMutate, wantAllow: true},
		{name: "contributor read", role: identity.RoleContributor, capability: CapabilityProjectRead, wantAllow: true},
		{name: "contributor mutate", role: identity.RoleContributor, capability: CapabilityProjectMutate, wantAllow: true},
		{name: "viewer read", role: identity.RoleViewer, capability: CapabilityProjectRead, wantAllow: true},
		{name: "viewer mutate", role: identity.RoleViewer, capability: CapabilityProjectMutate, wantAllow: false},
		{name: "client read", role: identity.RoleClient, capability: CapabilityProjectRead, wantAllow: true},
		{name: "client mutate", role: identity.RoleClient, capability: CapabilityProjectMutate, wantAllow: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			decision, err := Evaluate(tc.role, tc.capability)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if decision.Allowed != tc.wantAllow {
				t.Fatalf("expected Allowed=%v, got %v (%+v)", tc.wantAllow, decision.Allowed, decision)
			}
			if !tc.wantAllow {
				if decision.Reason == "" {
					t.Fatalf("expected non-empty Reason for denied decision")
				}
			} else if decision.Reason != "" {
				t.Fatalf("expected empty Reason for allowed decision, got %q", decision.Reason)
			}
			if err := decision.Validate(); err != nil {
				t.Fatalf("decision failed validation: %v", err)
			}
		})
	}
}

func TestEvaluate_IsDeterministic(t *testing.T) {
	first, err := Evaluate(identity.RoleViewer, CapabilityProjectMutate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for i := 0; i < 10; i++ {
		again, err := Evaluate(identity.RoleViewer, CapabilityProjectMutate)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if again != first {
			t.Fatalf("expected identical Decision on repeated calls, got %+v vs %+v", again, first)
		}
	}
}

func TestEvaluate_RejectsInvalidRole(t *testing.T) {
	if _, err := Evaluate(identity.Role("NOT_A_ROLE"), CapabilityProjectRead); !errors.Is(err, identity.ErrInvalidRole) {
		t.Fatalf("expected ErrInvalidRole, got %v", err)
	}
	if _, err := Evaluate(identity.Role(""), CapabilityProjectRead); !errors.Is(err, identity.ErrInvalidRole) {
		t.Fatalf("expected ErrInvalidRole for empty role, got %v", err)
	}
}

func TestEvaluate_RejectsInvalidCapability(t *testing.T) {
	if _, err := Evaluate(identity.RoleProjectOwner, Capability("")); !errors.Is(err, ErrInvalidCapability) {
		t.Fatalf("expected ErrInvalidCapability for empty capability, got %v", err)
	}
	if _, err := Evaluate(identity.RoleProjectOwner, Capability("   ")); !errors.Is(err, ErrInvalidCapability) {
		t.Fatalf("expected ErrInvalidCapability for whitespace capability, got %v", err)
	}
	if _, err := Evaluate(identity.RoleProjectOwner, Capability("unknown.capability")); !errors.Is(err, ErrInvalidCapability) {
		t.Fatalf("expected ErrInvalidCapability for unknown capability, got %v", err)
	}
}

func TestEvaluate_NeverPanics(t *testing.T) {
	inputs := []struct {
		role       identity.Role
		capability Capability
	}{
		{role: "", capability: ""},
		{role: "GARBAGE", capability: "GARBAGE"},
		{role: identity.RoleProjectOwner, capability: ""},
		{role: "", capability: CapabilityProjectRead},
	}
	for _, in := range inputs {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("Evaluate(%q, %q) panicked: %v", in.role, in.capability, r)
				}
			}()
			_, _ = Evaluate(in.role, in.capability)
		}()
	}
}

func TestDecision_Validate(t *testing.T) {
	cases := []struct {
		name     string
		decision Decision
		wantErr  bool
	}{
		{name: "allowed no reason", decision: Decision{Allowed: true, Reason: ""}, wantErr: false},
		{name: "allowed with reason", decision: Decision{Allowed: true, Reason: "explicit grant"}, wantErr: false},
		{name: "denied with reason", decision: Decision{Allowed: false, Reason: "missing capability"}, wantErr: false},
		{name: "denied without reason", decision: Decision{Allowed: false, Reason: ""}, wantErr: true},
		{name: "denied with whitespace reason", decision: Decision{Allowed: false, Reason: "   "}, wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.decision.Validate()
			if tc.wantErr {
				if !errors.Is(err, ErrInvalidDecision) {
					t.Fatalf("expected ErrInvalidDecision, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
