package lease

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ControlCenterSoft/aidi_2.0/internal/canonical"
	"github.com/ControlCenterSoft/aidi_2.0/internal/repository"
)

func testRef() canonical.ObjectRef {
	return canonical.ObjectRef{Kind: canonical.KindWorkflow, ID: "workflow-1"}
}

func TestLeaseValidateZeroValue(t *testing.T) {
	l := Lease{Ref: testRef()}
	if err := l.Validate(); err != nil {
		t.Fatalf("Validate() error = %v, want nil for never-acquired sentinel", err)
	}
}

func TestLeaseValidateRejectsInconsistentState(t *testing.T) {
	cases := []Lease{
		{Ref: testRef(), Owner: "owner-1"},                                             // Generation 0 but Owner set
		{Ref: testRef(), ExpiresAt: time.Now()},                                        // Generation 0 but ExpiresAt set
		{Ref: testRef(), Generation: 1},                                                // Generation>0 but no Owner/ExpiresAt
		{Ref: testRef(), Generation: 1, Owner: "owner-1"},                              // Generation>0, Owner set, ExpiresAt zero
		{Ref: testRef(), Generation: 1, ExpiresAt: time.Now()},                         // Generation>0, ExpiresAt set, Owner empty
		{Ref: canonical.ObjectRef{}, Generation: 1, Owner: "o", ExpiresAt: time.Now()}, // invalid ref
	}
	for i, l := range cases {
		if err := l.Validate(); err == nil {
			t.Fatalf("case %d: Validate() error = nil, want error for %+v", i, l)
		}
	}
}

func TestLeaseHeld(t *testing.T) {
	now := time.Now()
	notHeld := Lease{}
	if notHeld.Held(now) {
		t.Fatalf("zero Lease.Held() = true, want false")
	}

	expired := Lease{Generation: 1, Owner: "o", ExpiresAt: now.Add(-time.Second)}
	if expired.Held(now) {
		t.Fatalf("expired Lease.Held() = true, want false")
	}

	active := Lease{Generation: 1, Owner: "o", ExpiresAt: now.Add(time.Second)}
	if !active.Held(now) {
		t.Fatalf("active Lease.Held() = false, want true")
	}
}

func TestRepositoryStoreGetInvalidRef(t *testing.T) {
	store := NewRepositoryStore(repository.NewInMemory())
	if _, err := store.Get(context.Background(), canonical.ObjectRef{}); err == nil {
		t.Fatalf("Get() error = nil, want error for invalid ref")
	}
}

func TestRepositoryStoreGetNeverAcquiredReturnsZeroLease(t *testing.T) {
	store := NewRepositoryStore(repository.NewInMemory())
	ref := testRef()

	l, err := store.Get(context.Background(), ref)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if l.Generation != 0 || l.Owner != "" {
		t.Fatalf("Get() = %+v, want zero Lease sentinel", l)
	}
}

func TestRepositoryStoreAcquireRejectsInvalidInput(t *testing.T) {
	store := NewRepositoryStore(repository.NewInMemory())
	ref := testRef()
	now := time.Now()

	if _, err := store.Acquire(context.Background(), canonical.ObjectRef{}, "owner-1", now, time.Minute); err == nil {
		t.Fatalf("Acquire() error = nil, want error for invalid ref")
	}
	if _, err := store.Acquire(context.Background(), ref, "   ", now, time.Minute); !errors.Is(err, ErrInvalidOwner) {
		t.Fatalf("Acquire() error = %v, want ErrInvalidOwner", err)
	}
	if _, err := store.Acquire(context.Background(), ref, "owner-1", now, 0); !errors.Is(err, ErrInvalidTTL) {
		t.Fatalf("Acquire() error = %v, want ErrInvalidTTL", err)
	}
	if _, err := store.Acquire(context.Background(), ref, "owner-1", now, -time.Second); !errors.Is(err, ErrInvalidTTL) {
		t.Fatalf("Acquire() error = %v, want ErrInvalidTTL", err)
	}
}

func TestRepositoryStoreAcquireFirstTimeSucceeds(t *testing.T) {
	store := NewRepositoryStore(repository.NewInMemory())
	ref := testRef()
	now := time.Now()

	l, err := store.Acquire(context.Background(), ref, "owner-1", now, time.Minute)
	if err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}
	if l.Owner != "owner-1" || l.Generation != 1 || !l.ExpiresAt.Equal(now.Add(time.Minute).UTC()) {
		t.Fatalf("Acquire() = %+v, want Owner=owner-1 Generation=1 ExpiresAt=%v", l, now.Add(time.Minute).UTC())
	}

	stored, err := store.Get(context.Background(), ref)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if stored.Owner != l.Owner || stored.Generation != l.Generation || !stored.ExpiresAt.Equal(l.ExpiresAt) {
		t.Fatalf("Get() = %+v, want %+v", stored, l)
	}
}

func TestRepositoryStoreAcquireRejectsWhileHeldByAnotherOwner(t *testing.T) {
	store := NewRepositoryStore(repository.NewInMemory())
	ref := testRef()
	now := time.Now()

	if _, err := store.Acquire(context.Background(), ref, "owner-1", now, time.Minute); err != nil {
		t.Fatalf("Acquire() first error = %v", err)
	}

	if _, err := store.Acquire(context.Background(), ref, "owner-2", now, time.Minute); !errors.Is(err, ErrLeaseHeld) {
		t.Fatalf("Acquire() by second owner error = %v, want ErrLeaseHeld", err)
	}
}

func TestRepositoryStoreAcquireSucceedsAfterExpiry(t *testing.T) {
	store := NewRepositoryStore(repository.NewInMemory())
	ref := testRef()
	now := time.Now()

	first, err := store.Acquire(context.Background(), ref, "owner-1", now, time.Second)
	if err != nil {
		t.Fatalf("Acquire() first error = %v", err)
	}

	later := now.Add(2 * time.Second)
	second, err := store.Acquire(context.Background(), ref, "owner-2", later, time.Minute)
	if err != nil {
		t.Fatalf("Acquire() after expiry error = %v, want nil", err)
	}
	if second.Owner != "owner-2" || second.Generation != first.Generation+1 {
		t.Fatalf("Acquire() after expiry = %+v, want Owner=owner-2 Generation=%d", second, first.Generation+1)
	}
}

func TestRepositoryStoreAcquireSameOwnerAdvancesGeneration(t *testing.T) {
	store := NewRepositoryStore(repository.NewInMemory())
	ref := testRef()
	now := time.Now()

	first, err := store.Acquire(context.Background(), ref, "owner-1", now, time.Minute)
	if err != nil {
		t.Fatalf("Acquire() first error = %v", err)
	}

	second, err := store.Acquire(context.Background(), ref, "owner-1", now, time.Minute)
	if err != nil {
		t.Fatalf("Acquire() re-acquire by same owner error = %v", err)
	}
	if second.Generation != first.Generation+1 {
		t.Fatalf("Acquire() re-acquire Generation = %d, want %d", second.Generation, first.Generation+1)
	}
}

func TestRepositoryStoreRenewExtendsExpiryWithoutChangingGeneration(t *testing.T) {
	store := NewRepositoryStore(repository.NewInMemory())
	ref := testRef()
	now := time.Now()

	acquired, err := store.Acquire(context.Background(), ref, "owner-1", now, time.Minute)
	if err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}

	later := now.Add(30 * time.Second)
	renewed, err := store.Renew(context.Background(), ref, "owner-1", acquired.Generation, later, time.Minute)
	if err != nil {
		t.Fatalf("Renew() error = %v", err)
	}
	if renewed.Generation != acquired.Generation {
		t.Fatalf("Renew() Generation = %d, want unchanged %d", renewed.Generation, acquired.Generation)
	}
	if !renewed.ExpiresAt.Equal(later.Add(time.Minute).UTC()) {
		t.Fatalf("Renew() ExpiresAt = %v, want %v", renewed.ExpiresAt, later.Add(time.Minute).UTC())
	}
}

func TestRepositoryStoreRenewRejectsWrongOwnerOrGeneration(t *testing.T) {
	store := NewRepositoryStore(repository.NewInMemory())
	ref := testRef()
	now := time.Now()

	acquired, err := store.Acquire(context.Background(), ref, "owner-1", now, time.Minute)
	if err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}

	if _, err := store.Renew(context.Background(), ref, "owner-2", acquired.Generation, now, time.Minute); !errors.Is(err, ErrStaleLease) {
		t.Fatalf("Renew() wrong owner error = %v, want ErrStaleLease", err)
	}
	if _, err := store.Renew(context.Background(), ref, "owner-1", acquired.Generation+1, now, time.Minute); !errors.Is(err, ErrStaleLease) {
		t.Fatalf("Renew() wrong generation error = %v, want ErrStaleLease", err)
	}
}

func TestRepositoryStoreRenewRejectsWhenNeverAcquired(t *testing.T) {
	store := NewRepositoryStore(repository.NewInMemory())
	ref := testRef()
	now := time.Now()

	if _, err := store.Renew(context.Background(), ref, "owner-1", 1, now, time.Minute); !errors.Is(err, ErrNotHeld) {
		t.Fatalf("Renew() error = %v, want ErrNotHeld", err)
	}
}

func TestRepositoryStoreReleaseAllowsImmediateReacquisitionByAnotherOwner(t *testing.T) {
	store := NewRepositoryStore(repository.NewInMemory())
	ref := testRef()
	now := time.Now()

	acquired, err := store.Acquire(context.Background(), ref, "owner-1", now, time.Minute)
	if err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}

	released, err := store.Release(context.Background(), ref, "owner-1", acquired.Generation, now)
	if err != nil {
		t.Fatalf("Release() error = %v", err)
	}
	if released.Held(now) {
		t.Fatalf("released Lease.Held() = true, want false")
	}
	// The last holder remains visible for audit.
	if released.Owner != "owner-1" || released.Generation != acquired.Generation {
		t.Fatalf("Release() = %+v, want Owner/Generation preserved from %+v", released, acquired)
	}

	next, err := store.Acquire(context.Background(), ref, "owner-2", now, time.Minute)
	if err != nil {
		t.Fatalf("Acquire() after Release() error = %v", err)
	}
	if next.Owner != "owner-2" || next.Generation != acquired.Generation+1 {
		t.Fatalf("Acquire() after Release() = %+v, want Owner=owner-2 Generation=%d", next, acquired.Generation+1)
	}
}

func TestRepositoryStoreReleaseRejectsStaleCaller(t *testing.T) {
	store := NewRepositoryStore(repository.NewInMemory())
	ref := testRef()
	now := time.Now()

	acquired, err := store.Acquire(context.Background(), ref, "owner-1", now, time.Minute)
	if err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}

	if _, err := store.Release(context.Background(), ref, "owner-2", acquired.Generation, now); !errors.Is(err, ErrStaleLease) {
		t.Fatalf("Release() wrong owner error = %v, want ErrStaleLease", err)
	}
}

// TestLeaseSurvivesControlProcessRestart is the direct evidence for the
// A7-003 acceptance criterion "Lease survives control-process restart": a
// second RepositoryStore, constructed independently from the first and
// sharing only the same backing repository.Repository, observes the
// identical persisted Lease — exactly what a freshly started control
// process would observe after loading its backing store on restart.
func TestLeaseSurvivesControlProcessRestart(t *testing.T) {
	backingStore := repository.NewInMemory()
	ref := testRef()
	now := time.Now()

	beforeRestart := NewRepositoryStore(backingStore)
	acquired, err := beforeRestart.Acquire(context.Background(), ref, "owner-1", now, time.Minute)
	if err != nil {
		t.Fatalf("Acquire() before restart error = %v", err)
	}

	// Simulate a control-process restart: a brand new RepositoryStore,
	// with no shared in-process state, backed by the same durable
	// repository.Repository.
	afterRestart := NewRepositoryStore(backingStore)
	observed, err := afterRestart.Get(context.Background(), ref)
	if err != nil {
		t.Fatalf("Get() after restart error = %v", err)
	}
	if observed.Owner != acquired.Owner || observed.Generation != acquired.Generation || !observed.ExpiresAt.Equal(acquired.ExpiresAt) {
		t.Fatalf("Get() after restart = %+v, want identical to pre-restart Lease %+v", observed, acquired)
	}

	// The restarted process can also Renew the lease it did not itself
	// acquire in-process, proving ownership state (not just a cached
	// read) survived the restart.
	renewed, err := afterRestart.Renew(context.Background(), ref, "owner-1", observed.Generation, now.Add(time.Second), time.Minute)
	if err != nil {
		t.Fatalf("Renew() after restart error = %v", err)
	}
	if renewed.Generation != acquired.Generation {
		t.Fatalf("Renew() after restart Generation = %d, want unchanged %d", renewed.Generation, acquired.Generation)
	}
}

func TestRepositoryStoreImplementsStore(t *testing.T) {
	var _ Store = NewRepositoryStore(repository.NewInMemory())
}
