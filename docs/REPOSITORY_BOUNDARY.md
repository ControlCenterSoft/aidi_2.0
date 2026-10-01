# Release A — data-access/domain repository boundary and optimistic concurrency enforcement

Status: **IN DEVELOPMENT**

This slice implements the transport/storage-neutral **domain repository contract** (`A2-003`) for canonical objects, expressed purely in terms of `internal/canonical` primitives, and the **optimistic concurrency enforcement** card that builds atop it (`A2-004`). It follows the same pure-Go interfaces/types style already used by `internal/orchestration`.

## Repository contract

`Repository` is the boundary interface implemented by any future storage adapter:

- `Get(ctx, canonical.ObjectRef) (StoredObject, error)` returns the currently stored object for a reference.
- `Save(ctx, canonical.ObjectRef, expectedRevision canonical.Revision, payload []byte) (canonical.Revision, error)` writes a new payload for a reference, enforcing optimistic concurrency.

Both methods validate the `canonical.ObjectRef` (per `ObjectRef.Validate`) before touching any state, rejecting an invalid Kind or empty ID up front.

## StoredObject

`StoredObject` is the opaque, storage-neutral representation of a persisted canonical object: an `ObjectRef`, its current `Revision`, and an opaque payload (`[]byte`). The repository boundary does not interpret the payload; encoding is owned by the caller. `StoredObject.Validate` rejects an invalid `Ref`.

## Errors

- `ErrNotFound` is returned by `Get` when no `StoredObject` exists for the requested `ObjectRef`.
- `*canonical.RevisionConflictError` (wrapping `canonical.ErrRevisionConflict`) is returned by `Save` when the caller's `expectedRevision` does not match the currently stored revision. Last-write-wins is forbidden at the domain-contract level, independent of any SQL engine.

## Optimistic concurrency

`Save` delegates to `canonical.CheckExpectedRevision(expectedRevision, current)` before advancing the revision via `canonical.NextRevision(current)`. A stale `expectedRevision` — including one submitted concurrently with another in-flight write — is rejected without mutating existing state. The first write for a reference uses `expectedRevision = 0` and advances to revision `1`.

## Optimistic concurrency enforcement at the storage boundary (A2-004)

`internal/repository/concurrency.go` builds a generic, type-safe load/mutate/save
helper purely on top of the `Repository` contract above and the generic object
revision model (`canonical.Revisioned[T]`, A1-002):

- `Codec[T]{Encode, Decode}` — caller-supplied encode/decode functions between
  a domain value `T` and the opaque payload bytes `Repository` stores.
- `Update[T any](ctx, repo Repository, ref canonical.ObjectRef, codec Codec[T], expectedRevision canonical.Revision, mutate func(T) (T, error)) (canonical.Revisioned[T], error)`
  performs a single optimistic-concurrency-safe cycle:
  1. `Get`s the current `StoredObject` at `ref` (`ErrNotFound` is treated as
     "does not exist yet": the zero value of `T` at revision `0`).
  2. Checks `expectedRevision` against the just-loaded current revision via
     `canonical.CheckExpectedRevision`. A stale `expectedRevision` returns a
     `*canonical.RevisionConflictError` immediately — `mutate` is never
     invoked and `Save` is never called, so a stale caller can never
     overwrite newer canonical state.
  3. Only once that check passes does it decode the current payload, run
     `mutate`, encode the result, and call `repo.Save` with
     `expectedRevision`.

`repo.Save` independently re-checks `expectedRevision` against whatever is
actually stored at save time, so a second writer that commits between this
call's `Get` and `Save` is still rejected with the same
`*canonical.RevisionConflictError`, and the winning writer's state is left
untouched. This closes the storage-boundary race a naive Get-then-Save
caller could otherwise hit, satisfying the acceptance criteria that a stale
revision returns an explicit conflict and a stale write never overwrites
newer state.

This is purely additive: no existing entity (`Task`, `ChangeSet`,
`specification.Requirement`) is rewired onto `Update`/`Codec` in this
slice.

## InMemory reference implementation

`InMemory` is a minimal, non-persistent implementation of `Repository` used only to exercise/validate conformance to the interface in unit tests. It is not a persistence adapter and must not be treated as durable or production storage.

## Scope and isolation

This package depends solely on `internal/canonical` and the Go standard library. It introduces zero new external module dependencies and no file in the package imports `database/sql`, `net`, `net/http`, or any queue/workflow runtime package.

Out of scope for this slice (left to later, separate cards): any `database/sql`/`pgx`/Postgres driver, actual schema binding, and migration execution/schema compatibility metadata.

This package is isolated from the current/local AIDI runtime and has no dependency on any live/current AIDI database, Forgejo, local VM/runner infrastructure, or local queues. The `InMemory` reference implementation exists only to exercise the contract's conformance in unit tests, not to connect to any such system.

## Evidence

The package tests cover:

- `StoredObject.Validate` rejecting an invalid `ObjectRef`;
- `Get` on an absent `ObjectRef` returning `ErrNotFound`;
- `Get`/`Save` rejecting an invalid `ObjectRef` before touching any state;
- first write for a reference (revision `0` → `1`);
- stale expected-revision rejection (no last-write-wins);
- successful subsequent write (revision `N` → `N+1`);
- concurrent/stale write rejection under `-race`, where exactly one of several concurrent `Save` calls with the same stale `expectedRevision` succeeds.

`internal/repository/concurrency_test.go` covers `Update`:

- first write creates the object (`mutate` observes the zero value, revision `0` → `1`);
- successful subsequent write (revision `N` → `N+1`) with the prior payload decoded and passed to `mutate`;
- a stale `expectedRevision` is rejected with `*canonical.RevisionConflictError` before `mutate` runs and before any state changes;
- a `mutate` error is surfaced unchanged and leaves stored state untouched;
- a `codec.Decode` error is surfaced unchanged without calling `Save`;
- an invalid `ObjectRef` is surfaced without calling `mutate`;
- concurrent/stale write rejection under `-race`, where exactly one of several concurrent `Update` calls with the same stale `expectedRevision` succeeds and the stored state reflects only that winner.
