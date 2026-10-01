// Package repository defines the transport/storage-neutral domain
// repository boundary for canonical objects (A2-003). It is expressed
// solely in terms of internal/canonical primitives and introduces no
// database/sql, net, net/http, or queue/workflow runtime dependency.
//
// This package is a source-only contract. It is not wired to any
// live/current AIDI database, Forgejo instance, VM, self-hosted runner, or
// message queue, and the in-memory reference implementation exists only to
// exercise conformance to the Repository interface in unit tests.
package repository

import (
	"context"
	"errors"

	"github.com/ControlCenterSoft/aidi_2.0/internal/canonical"
)

// ErrNotFound is returned by Get when no StoredObject exists for the
// requested canonical.ObjectRef.
var ErrNotFound = errors.New("repository: object not found")

// StoredObject is the opaque, storage-neutral representation of a
// persisted canonical object: a reference, its current revision, and an
// opaque payload. The payload encoding is owned by the caller; the
// repository boundary does not interpret it.
type StoredObject struct {
	Ref      canonical.ObjectRef
	Revision canonical.Revision
	Payload  []byte
}

// Validate rejects a StoredObject whose ObjectRef is invalid per
// canonical.ObjectRef.Validate.
func (s StoredObject) Validate() error {
	return s.Ref.Validate()
}

// Repository is the transport/storage-neutral domain repository contract
// for canonical objects. Implementations must enforce optimistic
// concurrency on Save using canonical.CheckExpectedRevision /
// canonical.NextRevision; they must not implement last-write-wins
// semantics.
type Repository interface {
	// Get returns the StoredObject currently associated with ref. It
	// returns ErrNotFound when no object exists for ref, and returns the
	// error from ref.Validate when ref is invalid, before touching any
	// state.
	Get(ctx context.Context, ref canonical.ObjectRef) (StoredObject, error)

	// Save writes payload for ref, enforcing that the caller's
	// expectedRevision matches the currently stored revision (0 for an
	// object that does not yet exist). On success it returns the new
	// revision advanced via canonical.NextRevision. On a stale
	// expectedRevision it returns a *canonical.RevisionConflictError and
	// leaves existing state unchanged. It returns the error from
	// ref.Validate when ref is invalid, before touching any state.
	Save(ctx context.Context, ref canonical.ObjectRef, expectedRevision canonical.Revision, payload []byte) (canonical.Revision, error)
}
