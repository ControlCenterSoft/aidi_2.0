package orchestration

import (
	"errors"
	"testing"
	"time"
)

func testAttempt(id string) AttemptBinding {
	return AttemptBinding{
		AttemptID:   AttemptID(id),
		SourceSHA:   "0123456789abcdef",
		WorkspaceID: "workspace-" + id,
		ExecutorID:  "runner-" + id,
	}
}

func TestAcquireRenewAndRelease(t *testing.T) {
	now := time.Date(2026, 9, 26, 21, 0, 0, 0, time.UTC)
	current := Ownership{State: OwnershipUnowned}

	owned, err := Acquire(current, "wf-1", "owner-a", testAttempt("attempt-1"), now, 30*time.Second)
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	if owned.Token != 1 || owned.State != OwnershipOwned {
		t.Fatalf("unexpected ownership: %+v", owned)
	}

	renewed, err := Renew(owned, "owner-a", owned.Token, now.Add(10*time.Second), 30*time.Second)
	if err != nil {
		t.Fatalf("renew: %v", err)
	}
	wantLease := now.Add(40 * time.Second)
	if !renewed.LeaseUntil.Equal(wantLease) {
		t.Fatalf("lease until=%s want=%s", renewed.LeaseUntil, wantLease)
	}

	released, err := Release(renewed, "owner-a", renewed.Token, now.Add(20*time.Second))
	if err != nil {
		t.Fatalf("release: %v", err)
	}
	if released.State != OwnershipUnowned || released.OwnerID != "" || released.Token != 1 {
		t.Fatalf("unexpected released state: %+v", released)
	}
}

func TestStaleFenceCannotWrite(t *testing.T) {
	now := time.Date(2026, 9, 26, 21, 0, 0, 0, time.UTC)
	owned, err := Acquire(Ownership{State: OwnershipUnowned}, "wf-1", "owner-a", testAttempt("attempt-1"), now, time.Minute)
	if err != nil {
		t.Fatal(err)
	}

	if err := ValidateWrite(owned, "owner-a", owned.Token+1, now.Add(time.Second)); !errors.Is(err, ErrStaleFence) {
		t.Fatalf("stale token error=%v", err)
	}
	if err := ValidateWrite(owned, "owner-b", owned.Token, now.Add(time.Second)); !errors.Is(err, ErrStaleFence) {
		t.Fatalf("stale owner error=%v", err)
	}
}

func TestLeaseExpiryRequiresReconciliationBeforeReacquire(t *testing.T) {
	now := time.Date(2026, 9, 26, 21, 0, 0, 0, time.UTC)
	owned, err := Acquire(Ownership{State: OwnershipUnowned}, "wf-1", "owner-a", testAttempt("attempt-1"), now, 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	expiredAt := now.Add(10 * time.Second)
	if _, err := Acquire(owned, "wf-1", "owner-b", testAttempt("attempt-2"), expiredAt, time.Minute); !errors.Is(err, ErrReconciliationRequired) {
		t.Fatalf("acquire after expiry error=%v", err)
	}
	if err := ValidateWrite(owned, "owner-a", owned.Token, expiredAt); !errors.Is(err, ErrLeaseExpired) {
		t.Fatalf("expired write error=%v", err)
	}

	reconciling, err := EnterReconciliation(owned, expiredAt)
	if err != nil {
		t.Fatalf("enter reconciliation: %v", err)
	}
	if reconciling.State != OwnershipReconciling {
		t.Fatalf("state=%s", reconciling.State)
	}

	cleared, err := CompleteReconciliation(reconciling)
	if err != nil {
		t.Fatalf("complete reconciliation: %v", err)
	}
	reacquired, err := Acquire(cleared, "wf-1", "owner-b", testAttempt("attempt-2"), expiredAt.Add(time.Second), time.Minute)
	if err != nil {
		t.Fatalf("reacquire: %v", err)
	}
	if reacquired.Token != owned.Token+1 {
		t.Fatalf("token=%d want=%d", reacquired.Token, owned.Token+1)
	}
	if reacquired.Attempt.AttemptID != "attempt-2" {
		t.Fatalf("attempt=%s", reacquired.Attempt.AttemptID)
	}
}

func TestAttemptBindingRequiresIsolationCoordinates(t *testing.T) {
	cases := []AttemptBinding{
		{SourceSHA: "sha", WorkspaceID: "ws", ExecutorID: "runner"},
		{AttemptID: "a", WorkspaceID: "ws", ExecutorID: "runner"},
		{AttemptID: "a", SourceSHA: "sha", ExecutorID: "runner"},
		{AttemptID: "a", SourceSHA: "sha", WorkspaceID: "ws"},
	}
	for i, tc := range cases {
		if err := tc.Validate(); !errors.Is(err, ErrInvalidAttempt) {
			t.Fatalf("case %d error=%v", i, err)
		}
	}
}

func TestCommandLifecycle(t *testing.T) {
	valid := [][2]CommandState{
		{CommandPending, CommandDispatched},
		{CommandDispatched, CommandAcknowledged},
		{CommandAcknowledged, CommandApplied},
		{CommandPending, CommandRejected},
		{CommandDispatched, CommandRejected},
		{CommandAcknowledged, CommandRejected},
	}
	for _, pair := range valid {
		if err := ValidateCommandTransition(pair[0], pair[1]); err != nil {
			t.Fatalf("%s -> %s: %v", pair[0], pair[1], err)
		}
	}

	invalid := [][2]CommandState{
		{CommandPending, CommandApplied},
		{CommandApplied, CommandPending},
		{CommandRejected, CommandDispatched},
	}
	for _, pair := range invalid {
		if err := ValidateCommandTransition(pair[0], pair[1]); !errors.Is(err, ErrInvalidCommandTransition) {
			t.Fatalf("%s -> %s error=%v", pair[0], pair[1], err)
		}
	}
}

func TestTransportContractsRequireFenceAndAttempt(t *testing.T) {
	cmd := WorkflowCommand{
		CommandID:  "cmd-1",
		WorkflowID: "wf-1",
		Attempt:    testAttempt("attempt-1"),
		FenceToken: 2,
		Name:       "execute",
	}
	if err := cmd.Validate(); err != nil {
		t.Fatalf("command validate: %v", err)
	}

	event := WorkflowEvent{
		EventID:    "evt-1",
		WorkflowID: "wf-1",
		AttemptID:  "attempt-1",
		FenceToken: 2,
		Type:       "attempt.started",
	}
	if err := event.Validate(); err != nil {
		t.Fatalf("event validate: %v", err)
	}
}
