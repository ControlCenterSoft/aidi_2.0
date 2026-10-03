package integrity

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ControlCenterSoft/aidi_2.0/internal/canonical"
	"github.com/ControlCenterSoft/aidi_2.0/internal/lease"
	"github.com/ControlCenterSoft/aidi_2.0/internal/repository"
)

func leaseTestRef() canonical.ObjectRef {
	return canonical.ObjectRef{Kind: canonical.KindWorkflow, ID: "workflow-lease-check"}
}

func TestLeaseDetectorDetectsStaleLease(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, time.October, 3, 5, 0, 0, 0, time.UTC)
	store := lease.NewRepositoryStore(repository.NewInMemory())
	ref := leaseTestRef()

	acquired, err := store.Acquire(ctx, ref, "owner-1", now, time.Second)
	if err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}

	probeCalled := false
	detector := NewLeaseDetector(store, func(context.Context, string) (bool, error) {
		probeCalled = true
		return true, nil
	})
	report, err := detector.Detect(ctx, now.Add(2*time.Second), ref)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if !report.Has(LeaseFindingStale) {
		t.Fatalf("Detect() findings = %+v, want stale lease", report.Findings)
	}
	if report.Has(LeaseFindingOrphan) {
		t.Fatalf("Detect() findings = %+v, stale lease must not also be classified orphan", report.Findings)
	}
	if probeCalled {
		t.Fatalf("owner liveness probe called for already-stale lease")
	}

	observed, err := store.Get(ctx, ref)
	if err != nil {
		t.Fatalf("Get() after Detect error = %v", err)
	}
	assertLeaseUnchanged(t, acquired, observed)
}

func TestLeaseDetectorDetectsOrphanActiveLease(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, time.October, 3, 5, 0, 0, 0, time.UTC)
	store := lease.NewRepositoryStore(repository.NewInMemory())
	ref := leaseTestRef()

	acquired, err := store.Acquire(ctx, ref, "owner-orphan", now, time.Minute)
	if err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}

	var probedOwner string
	detector := NewLeaseDetector(store, func(_ context.Context, owner string) (bool, error) {
		probedOwner = owner
		return false, nil
	})
	report, err := detector.Detect(ctx, now.Add(time.Second), ref)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if !report.Has(LeaseFindingOrphan) {
		t.Fatalf("Detect() findings = %+v, want orphan lease", report.Findings)
	}
	if report.Has(LeaseFindingStale) {
		t.Fatalf("Detect() findings = %+v, active orphan must not be classified stale", report.Findings)
	}
	if probedOwner != acquired.Owner {
		t.Fatalf("owner liveness probe owner = %q, want %q", probedOwner, acquired.Owner)
	}

	observed, err := store.Get(ctx, ref)
	if err != nil {
		t.Fatalf("Get() after Detect error = %v", err)
	}
	assertLeaseUnchanged(t, acquired, observed)
}

func TestLeaseDetectorHealthyForLiveOwnerAndNeverAcquiredLease(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, time.October, 3, 5, 0, 0, 0, time.UTC)
	store := lease.NewRepositoryStore(repository.NewInMemory())
	activeRef := leaseTestRef()
	neverRef := canonical.ObjectRef{Kind: canonical.KindWorkflow, ID: "workflow-never-acquired"}

	if _, err := store.Acquire(ctx, activeRef, "owner-live", now, time.Minute); err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}

	probes := 0
	detector := NewLeaseDetector(store, func(_ context.Context, owner string) (bool, error) {
		probes++
		if owner != "owner-live" {
			t.Fatalf("unexpected owner probe %q", owner)
		}
		return true, nil
	})
	report, err := detector.Detect(ctx, now.Add(time.Second), activeRef, neverRef)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if !report.Healthy() {
		t.Fatalf("Detect() findings = %+v, want healthy", report.Findings)
	}
	if probes != 1 {
		t.Fatalf("owner liveness probes = %d, want 1", probes)
	}
}

func TestLeaseDetectorPropagatesUnknownOwnerLiveness(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, time.October, 3, 5, 0, 0, 0, time.UTC)
	store := lease.NewRepositoryStore(repository.NewInMemory())
	ref := leaseTestRef()
	if _, err := store.Acquire(ctx, ref, "owner-unknown", now, time.Minute); err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}

	probeErr := errors.New("owner registry unavailable")
	detector := NewLeaseDetector(store, func(context.Context, string) (bool, error) {
		return false, probeErr
	})
	report, err := detector.Detect(ctx, now.Add(time.Second), ref)
	if !errors.Is(err, probeErr) {
		t.Fatalf("Detect() error = %v, want owner probe error", err)
	}
	if !report.Healthy() {
		t.Fatalf("unknown liveness misclassified as orphan: %+v", report.Findings)
	}
}

func TestLeaseDetectorFailsClosedWithoutDependencies(t *testing.T) {
	report, err := NewLeaseDetector(nil, nil).Detect(context.Background(), time.Now(), leaseTestRef())
	if !errors.Is(err, ErrLeaseDetectorUnavailable) {
		t.Fatalf("Detect() error = %v, want ErrLeaseDetectorUnavailable", err)
	}
	if !report.Healthy() {
		t.Fatalf("Detect() findings = %+v, want none", report.Findings)
	}
}

func assertLeaseUnchanged(t *testing.T, want, got lease.Lease) {
	t.Helper()
	if got.Ref != want.Ref || got.Owner != want.Owner || got.Generation != want.Generation || !got.ExpiresAt.Equal(want.ExpiresAt) {
		t.Fatalf("persisted lease changed by detection: got %+v, want %+v", got, want)
	}
}
