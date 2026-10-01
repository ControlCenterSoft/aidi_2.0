// Package lease implements the transport/storage-neutral Persisted Lease
// domain contract (A7-003) required by approved SPEC §4.3/§8 ("Workflow/
// Attempt state persisted, а не RAM-only. Lease и fencing token являются
// first-class; просроченный owner не может записать результат после
// потери ownership.").
//
// Its sole hard dependency is the already-existing data-access/domain
// repository boundary package (internal/repository, A2-003): persistence
// of a Lease is delegated entirely to a repository.Repository, so a Lease
// observed before a control-process restart is the same Lease observed by
// a freshly constructed Store after restart, as long as both are backed by
// the same durable repository.Repository. The RepositoryStore below holds
// no in-process state of its own beyond the repository.Repository
// reference it wraps.
//
// This package deliberately implements only the persisted Lease primitive
// itself — Owner, Generation and ExpiresAt — and the minimal CAS
// (compare-and-swap by owner+generation) needed for Acquire/Renew/Release
// to be meaningful. It intentionally does not implement: monotonic
// fencing-token enforcement across external command/event writes
// (`A7-004`), expiry-driven reconciliation state machines (`A7-005`), or
// stale-owner write rejection at the point of use (`A7-006`). Those remain
// separate, later cards layered on top of this persisted Lease.
//
// This package is isolated from the current/local AIDI runtime: it has no
// dependency on any live/current AIDI database, Forgejo, local VM/runner
// infrastructure, message queue, or local state. The reference
// RepositoryStore implementation exists only to exercise the contract's
// conformance in unit tests, not to connect to any such system.
package lease

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/ControlCenterSoft/aidi_2.0/internal/canonical"
)

var (
	// ErrInvalidOwner is returned when an owner id is empty or
	// whitespace-only.
	ErrInvalidOwner = errors.New("lease: owner id is required")

	// ErrInvalidTTL is returned when a requested lease TTL is not
	// positive.
	ErrInvalidTTL = errors.New("lease: ttl must be positive")

	// ErrInvalidLease is returned by Lease.Validate when Owner,
	// Generation and ExpiresAt are not in a consistent state.
	ErrInvalidLease = errors.New("lease: invalid lease state")

	// ErrLeaseHeld is returned by Acquire when ref is currently held by a
	// different owner whose lease has not yet expired.
	ErrLeaseHeld = errors.New("lease: already held by another owner")

	// ErrNotHeld is returned by Renew/Release when ref has never had a
	// lease acquired (Generation 0).
	ErrNotHeld = errors.New("lease: no lease currently held")

	// ErrStaleLease is returned by Renew/Release when the supplied
	// owner/generation does not match the currently persisted Lease.
	ErrStaleLease = errors.New("lease: owner/generation does not match current lease")
)

// Lease is the opaque, storage-neutral representation of a persisted
// lease over a single canonical.ObjectRef: Owner, Generation and
// ExpiresAt, exactly as required by the A7-003 acceptance criteria.
//
// The zero value (Generation 0, Owner "", ExpiresAt zero) is the sentinel
// for "no lease has ever been acquired for this Ref". Once acquired,
// Generation is strictly positive and never decreases: it increments by
// one on every successful Acquire, including a later Acquire that follows
// a Release or an expiry, which is what lets a caller observe that
// ownership changed even if the same Owner re-acquires it. Release does
// not reset Generation or Owner to the zero sentinel; it sets ExpiresAt to
// the release time, so the last holder remains visible for audit while
// the lease is immediately available for the next Acquire.
type Lease struct {
	Ref        canonical.ObjectRef
	Owner      string
	Generation uint64
	ExpiresAt  time.Time
}

// Validate rejects a Lease whose fields are not in a consistent state. A
// Generation of 0 requires an empty Owner and a zero ExpiresAt (the
// never-acquired sentinel); a Generation greater than 0 requires a
// non-empty Owner and a non-zero ExpiresAt (ExpiresAt may be in the past,
// representing an expired or released lease).
func (l Lease) Validate() error {
	if err := l.Ref.Validate(); err != nil {
		return err
	}
	if l.Generation == 0 {
		if l.Owner != "" || !l.ExpiresAt.IsZero() {
			return ErrInvalidLease
		}
		return nil
	}
	if strings.TrimSpace(l.Owner) == "" {
		return ErrInvalidLease
	}
	if l.ExpiresAt.IsZero() {
		return ErrInvalidLease
	}
	return nil
}

// Held reports whether the Lease is currently held by its Owner as of
// now, i.e. Generation > 0 and now is strictly before ExpiresAt.
func (l Lease) Held(now time.Time) bool {
	return l.Generation > 0 && now.Before(l.ExpiresAt)
}

// Store is the transport/storage-neutral domain contract for a persisted
// Lease keyed by canonical.ObjectRef.
//
// Implementations must persist every successful Acquire/Renew/Release so
// that a subsequently constructed Store instance backed by the same
// durable storage observes the identical Lease (A7-003: "Lease survives
// control-process restart").
type Store interface {
	// Get returns the Lease currently persisted for ref, or the zero
	// Lease (Generation 0) when no lease has ever been acquired for ref.
	// It returns the error from ref.Validate when ref is invalid, before
	// touching any state.
	Get(ctx context.Context, ref canonical.ObjectRef) (Lease, error)

	// Acquire persists a new Lease for ref owned by owner, expiring at
	// now.Add(ttl), and returns it. Generation advances by one from the
	// Generation most recently persisted for ref.
	//
	// Acquire fails with ErrLeaseHeld when ref is currently held
	// (Lease.Held(now)) by a different owner. It succeeds when ref has
	// never been acquired, when the current holder's lease has expired
	// (regardless of owner), or when owner already holds it (a
	// re-acquisition, which still advances Generation).
	//
	// It returns the error from ref.Validate when ref is invalid,
	// ErrInvalidOwner when owner is empty/whitespace-only, and
	// ErrInvalidTTL when ttl is not positive — all before touching any
	// state.
	Acquire(ctx context.Context, ref canonical.ObjectRef, owner string, now time.Time, ttl time.Duration) (Lease, error)

	// Renew extends ExpiresAt to now.Add(ttl) for the Lease currently
	// persisted for ref, without changing Generation or Owner.
	//
	// It returns ErrNotHeld when ref has never been acquired, and
	// ErrStaleLease when owner/generation does not exactly match the
	// Lease currently persisted for ref (including an expired lease:
	// Renew does not implicitly re-acquire). It returns the error from
	// ref.Validate when ref is invalid, ErrInvalidOwner when owner is
	// empty/whitespace-only, and ErrInvalidTTL when ttl is not positive —
	// all before touching any state.
	Renew(ctx context.Context, ref canonical.ObjectRef, owner string, generation uint64, now time.Time, ttl time.Duration) (Lease, error)

	// Release sets ExpiresAt to now for the Lease currently persisted for
	// ref, without changing Generation or Owner, making ref immediately
	// available to a subsequent Acquire by any owner.
	//
	// It returns ErrNotHeld when ref has never been acquired, and
	// ErrStaleLease when owner/generation does not exactly match the
	// Lease currently persisted for ref. It returns the error from
	// ref.Validate when ref is invalid, and ErrInvalidOwner when owner is
	// empty/whitespace-only — all before touching any state.
	Release(ctx context.Context, ref canonical.ObjectRef, owner string, generation uint64, now time.Time) (Lease, error)
}

func validateOwner(owner string) error {
	if strings.TrimSpace(owner) == "" {
		return ErrInvalidOwner
	}
	return nil
}

func validateTTL(ttl time.Duration) error {
	if ttl <= 0 {
		return ErrInvalidTTL
	}
	return nil
}

func checkHolder(current Lease, owner string, generation uint64) error {
	if current.Generation == 0 {
		return ErrNotHeld
	}
	if current.Owner != owner || current.Generation != generation {
		return ErrStaleLease
	}
	return nil
}
