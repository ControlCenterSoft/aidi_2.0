package toolregistry

import (
	"errors"
	"reflect"
	"testing"
)

func validTool(id ToolID) Tool {
	return Tool{
		ID:              id,
		Name:            "golangci-lint",
		Version:         "1.60.0",
		License:         "GPL-3.0",
		Status:          StatusActive,
		ProvisionPolicy: ProvisionPolicyAuto,
		ResourceNeeds:   ResourceNeeds{CPUCores: 1, MemoryMB: 256, DiskMB: 100},
		Dependencies:    []ToolID{"go"},
	}
}

func TestRegistry_RegisterAndGet(t *testing.T) {
	r := NewRegistry()
	tool := validTool("golangci-lint")

	if err := r.Register(tool); err != nil {
		t.Fatalf("unexpected error registering valid tool: %v", err)
	}

	got, ok := r.Get(tool.ID)
	if !ok {
		t.Fatalf("expected tool %q to be found after registration", tool.ID)
	}
	if !reflect.DeepEqual(got, tool) {
		t.Fatalf("expected Get to return the registered tool, got %+v vs %+v", got, tool)
	}
}

func TestRegistry_Get_NotFound(t *testing.T) {
	r := NewRegistry()
	if _, ok := r.Get("missing"); ok {
		t.Fatalf("expected Get to return false for an unregistered id")
	}
}

func TestRegistry_Register_RejectsDuplicate(t *testing.T) {
	r := NewRegistry()
	tool := validTool("golangci-lint")

	if err := r.Register(tool); err != nil {
		t.Fatalf("unexpected error on first registration: %v", err)
	}
	if err := r.Register(tool); !errors.Is(err, ErrDuplicateTool) {
		t.Fatalf("expected ErrDuplicateTool on duplicate registration, got %v", err)
	}

	// Registering a different id must still succeed after a duplicate
	// rejection: the failed Register call must not have left the registry
	// in a bad state.
	other := validTool("other-tool")
	if err := r.Register(other); err != nil {
		t.Fatalf("unexpected error registering a distinct tool id: %v", err)
	}
}

func TestRegistry_Register_RejectsInvalidFields(t *testing.T) {
	cases := []struct {
		name string
		tool Tool
	}{
		{name: "empty id", tool: func() Tool { tl := validTool(""); return tl }()},
		{name: "whitespace id", tool: func() Tool { tl := validTool("   "); return tl }()},
		{name: "empty name", tool: func() Tool { tl := validTool("t1"); tl.Name = ""; return tl }()},
		{name: "whitespace name", tool: func() Tool { tl := validTool("t2"); tl.Name = "   "; return tl }()},
		{name: "empty version", tool: func() Tool { tl := validTool("t3"); tl.Version = ""; return tl }()},
		{name: "empty license", tool: func() Tool { tl := validTool("t4"); tl.License = ""; return tl }()},
		{name: "unknown status", tool: func() Tool { tl := validTool("t5"); tl.Status = Status("UNKNOWN"); return tl }()},
		{name: "empty status", tool: func() Tool { tl := validTool("t6"); tl.Status = Status(""); return tl }()},
		{name: "unknown provision policy", tool: func() Tool { tl := validTool("t7"); tl.ProvisionPolicy = ProvisionPolicy("MAYBE"); return tl }()},
		{name: "empty provision policy", tool: func() Tool { tl := validTool("t8"); tl.ProvisionPolicy = ProvisionPolicy(""); return tl }()},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := NewRegistry()
			if err := r.Register(tc.tool); !errors.Is(err, ErrInvalidTool) {
				t.Fatalf("expected ErrInvalidTool, got %v", err)
			}
			if _, ok := r.Get(tc.tool.ID); ok {
				t.Fatalf("expected invalid tool not to be registered")
			}
		})
	}
}

func TestTool_Validate(t *testing.T) {
	valid := validTool("golangci-lint")
	if err := valid.Validate(); err != nil {
		t.Fatalf("unexpected error validating a well-formed tool: %v", err)
	}
}

func TestStatus_Validate(t *testing.T) {
	cases := []struct {
		status  Status
		wantErr bool
	}{
		{status: StatusActive, wantErr: false},
		{status: StatusDeprecated, wantErr: false},
		{status: StatusDisabled, wantErr: false},
		{status: Status(""), wantErr: true},
		{status: Status("UNKNOWN"), wantErr: true},
	}
	for _, tc := range cases {
		err := tc.status.Validate()
		if tc.wantErr && !errors.Is(err, ErrInvalidTool) {
			t.Fatalf("status %q: expected ErrInvalidTool, got %v", tc.status, err)
		}
		if !tc.wantErr && err != nil {
			t.Fatalf("status %q: unexpected error: %v", tc.status, err)
		}
	}
}

func TestProvisionPolicy_Validate(t *testing.T) {
	cases := []struct {
		policy  ProvisionPolicy
		wantErr bool
	}{
		{policy: ProvisionPolicyAuto, wantErr: false},
		{policy: ProvisionPolicyRequestAdmin, wantErr: false},
		{policy: ProvisionPolicyDeny, wantErr: false},
		{policy: ProvisionPolicy(""), wantErr: true},
		{policy: ProvisionPolicy("UNKNOWN"), wantErr: true},
	}
	for _, tc := range cases {
		err := tc.policy.Validate()
		if tc.wantErr && !errors.Is(err, ErrInvalidTool) {
			t.Fatalf("policy %q: expected ErrInvalidTool, got %v", tc.policy, err)
		}
		if !tc.wantErr && err != nil {
			t.Fatalf("policy %q: unexpected error: %v", tc.policy, err)
		}
	}
}

func TestRegistry_List_Empty(t *testing.T) {
	r := NewRegistry()
	list := r.List()
	if list == nil {
		t.Fatalf("expected non-nil empty slice from List on empty registry")
	}
	if len(list) != 0 {
		t.Fatalf("expected empty list, got %+v", list)
	}
}

func TestRegistry_List_Populated_SortedByID(t *testing.T) {
	r := NewRegistry()
	toolC := validTool("c-tool")
	toolA := validTool("a-tool")
	toolB := validTool("b-tool")

	for _, tl := range []Tool{toolC, toolA, toolB} {
		if err := r.Register(tl); err != nil {
			t.Fatalf("unexpected error registering %q: %v", tl.ID, err)
		}
	}

	list := r.List()
	if len(list) != 3 {
		t.Fatalf("expected 3 tools, got %d", len(list))
	}
	wantOrder := []ToolID{"a-tool", "b-tool", "c-tool"}
	for i, want := range wantOrder {
		if list[i].ID != want {
			t.Fatalf("expected List()[%d].ID = %q, got %q", i, want, list[i].ID)
		}
	}
}

func TestRegistry_ZeroValueGet_NeverPanics(t *testing.T) {
	var r Registry
	defer func() {
		if rec := recover(); rec != nil {
			t.Fatalf("Get on zero-value Registry panicked: %v", rec)
		}
	}()
	if _, ok := r.Get("anything"); ok {
		t.Fatalf("expected Get on zero-value Registry to return false")
	}
}

func TestRegistry_ZeroValueRegister_NeverPanics(t *testing.T) {
	var r Registry
	defer func() {
		if rec := recover(); rec != nil {
			t.Fatalf("Register on zero-value Registry panicked: %v", rec)
		}
	}()
	if err := r.Register(validTool("golangci-lint")); err != nil {
		t.Fatalf("unexpected error registering on zero-value Registry: %v", err)
	}
}
