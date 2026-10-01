# Release A — data-access/domain repository boundary

Status: **IN DEVELOPMENT**

This slice implements the transport/storage-neutral **domain repository contract** (`A2-003`) for canonical objects, expressed purely in terms of `internal/canonical` primitives. It follows the same pure-Go interfaces/types style already used by `internal/orchestration`.

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

## InMemory reference implementation

`InMemory` is a minimal, non-persistent implementation of `Repository` used only to exercise/validate conformance to the interface in unit tests. It is not a persistence adapter and must not be treated as durable or production storage.

## Scope and isolation

This package depends solely on `internal/canonical` and the Go standard library. It introduces zero new external module dependencies and no file in the package imports `database/sql`, `net`, `net/http`, or any queue/workflow runtime package.

Out of scope for this slice (left to later, separate cards): any `database/sql`/`pgx`/Postgres driver, actual schema binding, migration execution, schema compatibility metadata, and the broader optimistic-concurrency enforcement card that will build atop this boundary.

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
