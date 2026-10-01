package repository

import (
	"context"
	"errors"

	"github.com/ControlCenterSoft/aidi_2.0/internal/canonical"
)

// Optimistic concurrency enforcement at the storage boundary (A2-004).
//
// This slice builds purely on the already-existing domain repository
// contract (Repository, A2-003) and the generic object revision model
// (canonical.Revisioned[T], A1-002), per docs/CANONICAL_STATE.md "Object
// revision model": it is "later cards (e.g. canonical schema/persistence,
// optimistic concurrency enforcement at the storage boundary)" building on
// that primitive. It is purely additive: no existing entity (Task,
// ChangeSet, specification.Requirement) is rewired onto Update/Codec in
// this slice, and no new storage adapter, encoding format, database/sql,
// net, or queue/workflow runtime dependency is introduced.

// Codec encodes/decodes a domain value of type T to/from the opaque
// payload bytes that Repository stores. The repository boundary itself
// does not interpret payloads (see StoredObject); Codec is supplied by the
// caller of Update to give Update type-safe access to the current value.
type Codec[T any] struct {
	Encode func(T) ([]byte, error)
	Decode func([]byte) (T, error)
}

// Update performs a single optimistic-concurrency-safe load/mutate/save
// cycle for a typed domain object stored behind Repository at ref.
//
// The caller supplies expectedRevision: the Revision it believes is
// currently stored (0 for an object it believes does not exist yet). Update:
//
//  1. Loads the StoredObject currently at ref. ErrNotFound is treated as
//     "does not exist yet" (the zero value of T at Revision 0); any other
//     Get error (including an invalid ref) is returned unchanged and
//     nothing further is attempted.
//  2. Checks expectedRevision against the just-loaded current Revision via
//     canonical.CheckExpectedRevision. A mismatch returns a
//     *canonical.RevisionConflictError immediately: mutate is never
//     invoked and Save is never called, so a stale caller can never
//     overwrite newer canonical state (no last-write-wins), mirroring the
//     "mutate never runs on a stale expected revision" contract already
//     enforced by canonical.Revisioned[T].Update.
//  3. Only once that check passes does it decode the current payload (via
//     codec.Decode), invoke mutate on the decoded value, encode the result
//     (via codec.Encode) and call repo.Save with expectedRevision.
//
// repo.Save independently re-checks expectedRevision against whatever is
// actually stored at Save time (see Repository.Save), so a second writer
// that commits between this call's Get and Save is still rejected with the
// same *canonical.RevisionConflictError and leaves the winning writer's
// state untouched: this closes the storage-boundary race a naive
// Get-then-Save caller could otherwise hit, without Update needing any
// lock of its own.
func Update[T any](ctx context.Context, repo Repository, ref canonical.ObjectRef, codec Codec[T], expectedRevision canonical.Revision, mutate func(T) (T, error)) (canonical.Revisioned[T], error) {
	stored, err := repo.Get(ctx, ref)

	var (
		current         T
		currentRevision canonical.Revision
	)
	switch {
	case err == nil:
		currentRevision = stored.Revision
		if current, err = codec.Decode(stored.Payload); err != nil {
			return canonical.Revisioned[T]{}, err
		}
	case errors.Is(err, ErrNotFound):
		// currentRevision stays 0; current stays the zero value of T.
	default:
		return canonical.Revisioned[T]{}, err
	}

	if err := canonical.CheckExpectedRevision(expectedRevision, currentRevision); err != nil {
		return canonical.Revisioned[T]{}, err
	}

	updated, err := mutate(current)
	if err != nil {
		return canonical.Revisioned[T]{}, err
	}

	payload, err := codec.Encode(updated)
	if err != nil {
		return canonical.Revisioned[T]{}, err
	}

	newRevision, err := repo.Save(ctx, ref, expectedRevision, payload)
	if err != nil {
		return canonical.Revisioned[T]{}, err
	}

	return canonical.Revisioned[T]{Revision: newRevision, Value: updated}, nil
}
