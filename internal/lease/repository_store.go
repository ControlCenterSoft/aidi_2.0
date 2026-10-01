package lease

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/ControlCenterSoft/aidi_2.0/internal/canonical"
	"github.com/ControlCenterSoft/aidi_2.0/internal/repository"
)

// RepositoryStore is a Store implementation that persists every Lease via
// a repository.Repository (A2-003), using repository.Repository's
// optimistic-concurrency Save to make Acquire/Renew/Release atomic
// compare-and-swap operations. RepositoryStore holds no state of its own:
// all Lease data lives in the wrapped repository.Repository, so
// constructing a new RepositoryStore around the same durable
// repository.Repository after a control-process restart observes the
// identical Lease for every ref (A7-003 acceptance criterion).
type RepositoryStore struct {
	repo repository.Repository
}

// NewRepositoryStore constructs a RepositoryStore backed by repo.
func NewRepositoryStore(repo repository.Repository) *RepositoryStore {
	return &RepositoryStore{repo: repo}
}

var _ Store = (*RepositoryStore)(nil)

// leaseDocument is the JSON-encoded payload persisted via
// repository.StoredObject.Payload. Ref is not duplicated in the payload:
// it is already the repository key.
type leaseDocument struct {
	Owner      string    `json:"owner"`
	Generation uint64    `json:"generation"`
	ExpiresAt  time.Time `json:"expires_at"`
}

// Get implements Store.
func (s *RepositoryStore) Get(ctx context.Context, ref canonical.ObjectRef) (Lease, error) {
	if err := ref.Validate(); err != nil {
		return Lease{}, err
	}
	l, _, err := s.load(ctx, ref)
	return l, err
}

// Acquire implements Store.
func (s *RepositoryStore) Acquire(ctx context.Context, ref canonical.ObjectRef, owner string, now time.Time, ttl time.Duration) (Lease, error) {
	if err := ref.Validate(); err != nil {
		return Lease{}, err
	}
	if err := validateOwner(owner); err != nil {
		return Lease{}, err
	}
	if err := validateTTL(ttl); err != nil {
		return Lease{}, err
	}

	current, rev, err := s.load(ctx, ref)
	if err != nil {
		return Lease{}, err
	}
	if current.Held(now) && current.Owner != owner {
		return Lease{}, ErrLeaseHeld
	}

	next := Lease{
		Ref:        ref,
		Owner:      owner,
		Generation: current.Generation + 1,
		ExpiresAt:  now.Add(ttl),
	}
	if err := s.save(ctx, ref, rev, next); err != nil {
		return Lease{}, err
	}
	return next, nil
}

// Renew implements Store.
func (s *RepositoryStore) Renew(ctx context.Context, ref canonical.ObjectRef, owner string, generation uint64, now time.Time, ttl time.Duration) (Lease, error) {
	if err := ref.Validate(); err != nil {
		return Lease{}, err
	}
	if err := validateOwner(owner); err != nil {
		return Lease{}, err
	}
	if err := validateTTL(ttl); err != nil {
		return Lease{}, err
	}

	current, rev, err := s.load(ctx, ref)
	if err != nil {
		return Lease{}, err
	}
	if err := checkHolder(current, owner, generation); err != nil {
		return Lease{}, err
	}

	next := current
	next.ExpiresAt = now.Add(ttl)
	if err := s.save(ctx, ref, rev, next); err != nil {
		return Lease{}, err
	}
	return next, nil
}

// Release implements Store.
func (s *RepositoryStore) Release(ctx context.Context, ref canonical.ObjectRef, owner string, generation uint64, now time.Time) (Lease, error) {
	if err := ref.Validate(); err != nil {
		return Lease{}, err
	}
	if err := validateOwner(owner); err != nil {
		return Lease{}, err
	}

	current, rev, err := s.load(ctx, ref)
	if err != nil {
		return Lease{}, err
	}
	if err := checkHolder(current, owner, generation); err != nil {
		return Lease{}, err
	}

	next := current
	next.ExpiresAt = now
	if err := s.save(ctx, ref, rev, next); err != nil {
		return Lease{}, err
	}
	return next, nil
}

// load returns the Lease currently persisted for ref (the zero Lease when
// ref has never been acquired) together with the repository revision
// Acquire/Renew/Release must pass back to Save as expectedRevision.
func (s *RepositoryStore) load(ctx context.Context, ref canonical.ObjectRef) (Lease, canonical.Revision, error) {
	obj, err := s.repo.Get(ctx, ref)
	if errors.Is(err, repository.ErrNotFound) {
		return Lease{Ref: ref}, 0, nil
	}
	if err != nil {
		return Lease{}, 0, err
	}

	var doc leaseDocument
	if err := json.Unmarshal(obj.Payload, &doc); err != nil {
		return Lease{}, 0, err
	}
	l := Lease{
		Ref:        ref,
		Owner:      doc.Owner,
		Generation: doc.Generation,
		ExpiresAt:  doc.ExpiresAt.UTC(),
	}
	if err := l.Validate(); err != nil {
		return Lease{}, 0, err
	}
	return l, obj.Revision, nil
}

// save persists next for ref, enforcing that expectedRevision still
// matches the repository's current revision (A2-003 optimistic
// concurrency).
func (s *RepositoryStore) save(ctx context.Context, ref canonical.ObjectRef, expectedRevision canonical.Revision, next Lease) error {
	payload, err := json.Marshal(leaseDocument{
		Owner:      next.Owner,
		Generation: next.Generation,
		ExpiresAt:  next.ExpiresAt.UTC(),
	})
	if err != nil {
		return err
	}
	_, err = s.repo.Save(ctx, ref, expectedRevision, payload)
	return err
}
