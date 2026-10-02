package commandendpoint

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/ControlCenterSoft/aidi_2.0/internal/apicommand"
	"github.com/ControlCenterSoft/aidi_2.0/internal/authorization"
	"github.com/ControlCenterSoft/aidi_2.0/internal/authzpipeline"
	"github.com/ControlCenterSoft/aidi_2.0/internal/canonical"
	"github.com/ControlCenterSoft/aidi_2.0/internal/identity"
	"github.com/ControlCenterSoft/aidi_2.0/internal/inbox"
	"github.com/ControlCenterSoft/aidi_2.0/internal/policy"
	"github.com/ControlCenterSoft/aidi_2.0/internal/toolregistry"
)

const commandToolID toolregistry.ToolID = "foundation-command"

func validCommand() apicommand.MutationCommand {
	return apicommand.MutationCommand{
		CommandID:        "cmd-115",
		CorrelationID:    "corr-115",
		Target:           apicommand.Target{Kind: "project", ID: "p1"},
		Operation:        "rename",
		ExpectedRevision: canonical.Revision(7),
		Payload:          json.RawMessage(`{"name":"renamed"}`),
	}
}

func testService(t *testing.T, role identity.Role, effect policy.Effect, calls *atomic.Int32) (Service, *inbox.InMemory) {
	t.Helper()

	registry := toolregistry.NewRegistry()
	if err := registry.Register(toolregistry.Tool{
		ID:              commandToolID,
		Name:            "Foundation Command",
		Version:         "1.0.0",
		License:         "Apache-2.0",
		Status:          toolregistry.StatusActive,
		ProvisionPolicy: toolregistry.ProvisionPolicyAuto,
	}); err != nil {
		t.Fatalf("register command tool: %v", err)
	}

	store := inbox.NewInMemory()
	pol := policy.Policy{
		ID: "foundation-command-policy",
		Statements: []policy.Statement{{
			Subject:  "user-1",
			Resource: "project:p1",
			Action:   "rename",
			Effect:   effect,
		}},
	}

	service := Service{
		Inbox:    store,
		Registry: registry,
		BuildAuthorization: func(context.Context, apicommand.MutationCommand) (authzpipeline.Request, error) {
			return authzpipeline.Request{
				Role:       role,
				Capability: authorization.CapabilityProjectMutate,
				Policy:     pol,
				Subject:    "user-1",
				Resource:   "project:p1",
				Action:     "rename",
				ToolID:     commandToolID,
			}, nil
		},
		Execute: func(context.Context, apicommand.MutationCommand) (json.RawMessage, error) {
			calls.Add(1)
			return json.RawMessage(`{"revision":8}`), nil
		},
	}
	return service, store
}

func TestExecuteCommandEnforcesIdempotency(t *testing.T) {
	var calls atomic.Int32
	service, _ := testService(t, identity.RoleContributor, policy.EffectAllow, &calls)
	ctx := context.Background()

	first, err := service.ExecuteCommand(ctx, validCommand())
	if err != nil {
		t.Fatalf("first execution failed: %v", err)
	}
	second, err := service.ExecuteCommand(ctx, validCommand())
	if err != nil {
		t.Fatalf("duplicate execution failed: %v", err)
	}

	if got := calls.Load(); got != 1 {
		t.Fatalf("executor called %d times, want exactly once", got)
	}
	if first.Duplicate {
		t.Fatal("first execution marked duplicate")
	}
	if !second.Duplicate {
		t.Fatal("second execution was not marked duplicate")
	}
	if !bytes.Equal(first.Result, second.Result) {
		t.Fatalf("duplicate returned different result: %s vs %s", first.Result, second.Result)
	}
}

func TestExecuteCommandEnforcesRBACBeforeInbox(t *testing.T) {
	var calls atomic.Int32
	service, store := testService(t, identity.RoleViewer, policy.EffectAllow, &calls)
	command := validCommand()

	_, err := service.ExecuteCommand(context.Background(), command)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
	if got := calls.Load(); got != 0 {
		t.Fatalf("executor called %d times after RBAC denial", got)
	}
	if _, err := store.Get(context.Background(), command.CommandID); !errors.Is(err, inbox.ErrNotFound) {
		t.Fatalf("denied command must not enter inbox, got %v", err)
	}
}

func TestExecuteCommandEnforcesPolicyBeforeInbox(t *testing.T) {
	var calls atomic.Int32
	service, store := testService(t, identity.RoleContributor, policy.EffectDeny, &calls)
	command := validCommand()

	_, err := service.ExecuteCommand(context.Background(), command)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
	if got := calls.Load(); got != 0 {
		t.Fatalf("executor called %d times after policy denial", got)
	}
	if _, err := store.Get(context.Background(), command.CommandID); !errors.Is(err, inbox.ErrNotFound) {
		t.Fatalf("denied command must not enter inbox, got %v", err)
	}
}

func TestDuplicateIsReauthorized(t *testing.T) {
	var calls atomic.Int32
	allowService, store := testService(t, identity.RoleContributor, policy.EffectAllow, &calls)
	command := validCommand()

	if _, err := allowService.ExecuteCommand(context.Background(), command); err != nil {
		t.Fatalf("initial execution failed: %v", err)
	}

	denyService, _ := testService(t, identity.RoleViewer, policy.EffectAllow, &calls)
	denyService.Inbox = store
	_, err := denyService.ExecuteCommand(context.Background(), command)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected duplicate to be reauthorized and denied, got %v", err)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("duplicate denial must not execute again; calls=%d", got)
	}
}

func TestHandlerReturnsStableDuplicateResult(t *testing.T) {
	var calls atomic.Int32
	service, _ := testService(t, identity.RoleContributor, policy.EffectAllow, &calls)
	handler := Handler(service)
	body, err := json.Marshal(validCommand())
	if err != nil {
		t.Fatal(err)
	}

	call := func() Result {
		req := httptest.NewRequest(http.MethodPost, "/api/foundation/commands", bytes.NewReader(body))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		var result Result
		if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		return result
	}

	first := call()
	second := call()
	if first.Duplicate || !second.Duplicate {
		t.Fatalf("unexpected duplicate flags: first=%v second=%v", first.Duplicate, second.Duplicate)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("executor called %d times, want 1", got)
	}
}

func TestHandlerRejectsDeniedCommand(t *testing.T) {
	var calls atomic.Int32
	service, _ := testService(t, identity.RoleViewer, policy.EffectAllow, &calls)
	body, err := json.Marshal(validCommand())
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/foundation/commands", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	Handler(service).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if got := calls.Load(); got != 0 {
		t.Fatalf("executor called %d times for denied command", got)
	}
}

func TestHandlerRejectsNonPost(t *testing.T) {
	var calls atomic.Int32
	service, _ := testService(t, identity.RoleContributor, policy.EffectAllow, &calls)
	req := httptest.NewRequest(http.MethodGet, "/api/foundation/commands", nil)
	rec := httptest.NewRecorder()

	Handler(service).ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
	if got := rec.Header().Get("Allow"); got != http.MethodPost {
		t.Fatalf("Allow=%q, want POST", got)
	}
}
