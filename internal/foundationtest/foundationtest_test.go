package foundationtest

import (
	"testing"

	"github.com/ControlCenterSoft/aidi_2.0/internal/authzpipeline"
	"github.com/ControlCenterSoft/aidi_2.0/internal/policy"
	"github.com/ControlCenterSoft/aidi_2.0/internal/toolregistry"
)

// TestAllowPolicy_Valid asserts AllowPolicy independently passes
// policy.Policy.Validate().
func TestAllowPolicy_Valid(t *testing.T) {
	if err := AllowPolicy().Validate(); err != nil {
		t.Fatalf("AllowPolicy() failed Validate(): %v", err)
	}
}

// TestDenyPolicy_Valid asserts DenyPolicy independently passes
// policy.Policy.Validate().
func TestDenyPolicy_Valid(t *testing.T) {
	if err := DenyPolicy().Validate(); err != nil {
		t.Fatalf("DenyPolicy() failed Validate(): %v", err)
	}
}

// TestNoMatchPolicy_Valid asserts NoMatchPolicy independently passes
// policy.Policy.Validate().
func TestNoMatchPolicy_Valid(t *testing.T) {
	if err := NoMatchPolicy().Validate(); err != nil {
		t.Fatalf("NoMatchPolicy() failed Validate(): %v", err)
	}
}

// TestActiveTool_ValidAndActive asserts ActiveTool independently passes
// toolregistry.Tool.Validate() and carries toolregistry.StatusActive.
func TestActiveTool_ValidAndActive(t *testing.T) {
	tool := ActiveTool()
	if err := tool.Validate(); err != nil {
		t.Fatalf("ActiveTool() failed Validate(): %v", err)
	}
	if tool.Status != toolregistry.StatusActive {
		t.Fatalf("ActiveTool().Status = %q, want %q", tool.Status, toolregistry.StatusActive)
	}
}

// TestInactiveTool_ValidAndNonActive asserts InactiveTool independently
// passes toolregistry.Tool.Validate() and carries a non-active Status.
func TestInactiveTool_ValidAndNonActive(t *testing.T) {
	tool := InactiveTool()
	if err := tool.Validate(); err != nil {
		t.Fatalf("InactiveTool() failed Validate(): %v", err)
	}
	if tool.Status == toolregistry.StatusActive {
		t.Fatalf("InactiveTool().Status = %q, want a non-active status", tool.Status)
	}
}

// TestNewActiveRegistry_RegistersActiveTool asserts NewActiveRegistry
// registers ActiveTool and it is retrievable via Registry.Get.
func TestNewActiveRegistry_RegistersActiveTool(t *testing.T) {
	registry := NewActiveRegistry()
	tool, ok := registry.Get(ActiveToolID)
	if !ok {
		t.Fatalf("NewActiveRegistry(): ActiveToolID %q not registered", ActiveToolID)
	}
	if tool.Status != toolregistry.StatusActive {
		t.Fatalf("registered tool Status = %q, want %q", tool.Status, toolregistry.StatusActive)
	}
}

// TestNewInactiveRegistry_RegistersInactiveTool asserts NewInactiveRegistry
// registers InactiveTool and it is retrievable via Registry.Get.
func TestNewInactiveRegistry_RegistersInactiveTool(t *testing.T) {
	registry := NewInactiveRegistry()
	tool, ok := registry.Get(InactiveToolID)
	if !ok {
		t.Fatalf("NewInactiveRegistry(): InactiveToolID %q not registered", InactiveToolID)
	}
	if tool.Status == toolregistry.StatusActive {
		t.Fatalf("registered tool Status = %q, want a non-active status", tool.Status)
	}
}

// TestBaseRequest_Valid asserts BaseRequest's embedded Policy independently
// passes Validate() and the Role/Capability pair is well-formed.
func TestBaseRequest_Valid(t *testing.T) {
	req := BaseRequest()
	if err := req.Policy.Validate(); err != nil {
		t.Fatalf("BaseRequest().Policy failed Validate(): %v", err)
	}
	if err := req.Role.Validate(); err != nil {
		t.Fatalf("BaseRequest().Role failed Validate(): %v", err)
	}
	if err := req.ToolID.Validate(); err != nil {
		t.Fatalf("BaseRequest().ToolID failed Validate(): %v", err)
	}
}

// TestAuthorize_AllowDenyNoMatch composes the fixtures end-to-end through
// authzpipeline.Authorize, covering one ALLOW, one explicit-DENY-overrides-
// ALLOW, and one default-deny outcome — the acceptance criterion this
// package must satisfy.
func TestAuthorize_AllowDenyNoMatch(t *testing.T) {
	cases := []struct {
		name        string
		policy      policy.Policy
		wantAllowed bool
	}{
		{name: "allow policy grants access", policy: AllowPolicy(), wantAllowed: true},
		{name: "explicit deny overrides matching allow", policy: DenyPolicy(), wantAllowed: false},
		{name: "default-deny when no statement matches", policy: NoMatchPolicy(), wantAllowed: false},
	}

	registry := NewActiveRegistry()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := BaseRequest()
			req.Policy = tc.policy

			got, err := authzpipeline.Authorize(registry, req)
			if err != nil {
				t.Fatalf("Authorize() unexpected error: %v", err)
			}
			if got.Allowed != tc.wantAllowed {
				t.Fatalf("Authorize().Allowed = %v, want %v (reason: %q)", got.Allowed, tc.wantAllowed, got.Reason)
			}
			if !tc.wantAllowed && got.Reason == "" {
				t.Fatalf("expected non-empty Reason for denial")
			}
		})
	}
}

// TestAuthorize_InactiveToolDenies composes InactiveTool/NewInactiveRegistry
// through authzpipeline.Authorize, asserting a non-ACTIVE tool denies
// execution even when RBAC and policy both allow.
func TestAuthorize_InactiveToolDenies(t *testing.T) {
	registry := NewInactiveRegistry()
	req := BaseRequest()
	req.ToolID = InactiveToolID

	got, err := authzpipeline.Authorize(registry, req)
	if err != nil {
		t.Fatalf("Authorize() unexpected error: %v", err)
	}
	if got.Allowed {
		t.Fatalf("Authorize().Allowed = true, want false for a non-active tool")
	}
	if got.Reason == "" {
		t.Fatalf("expected non-empty Reason for denial")
	}
}
