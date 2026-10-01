# Persisted Lease foundation

Source: approved AIDI v2.0.0 SPEC §4.3 and §8 ("Workflow/Attempt state persisted, а не RAM-only. Lease и fencing token являются first-class; просроченный owner не может записать результат после потери ownership.").

## Persisted Lease (`A7-003`, implemented)

`internal/lease` implements the Persisted Lease domain contract: a `Lease` over a `canonical.ObjectRef` that exposes `Owner`, `Generation` and `ExpiresAt`, and is durable across a control-process restart. Its sole hard dependency is the existing data-access/domain repository boundary package (`internal/repository`, `A2-003`); persistence is delegated entirely to a `repository.Repository`, so a `Lease` observed before a restart is the same `Lease` observed by a freshly constructed `Store` after restart, as long as both are backed by the same durable `repository.Repository`.

- `Lease{Ref, Owner, Generation, ExpiresAt}` is the opaque, storage-neutral representation required by the acceptance criteria. The zero value (`Generation 0`) is the "never acquired" sentinel; once acquired, `Generation` is strictly positive and only ever increases, advancing by one on every successful `Acquire` (including a later re-acquisition after expiry or release).
- `Store` is the storage-neutral contract:
  - `Get(ctx, ref) (Lease, error)` returns the currently persisted `Lease`, or the zero `Lease` when `ref` has never been acquired.
  - `Acquire(ctx, ref, owner, now, ttl) (Lease, error)` persists a new `Lease` for `ref`, advancing `Generation` by one. It fails with `ErrLeaseHeld` only when `ref` is currently held (`Lease.Held(now)`) by a *different* owner; it succeeds when `ref` has never been acquired, when the current holder's lease has expired, or when the same owner re-acquires it.
  - `Renew(ctx, ref, owner, generation, now, ttl) (Lease, error)` extends `ExpiresAt` without changing `Generation`/`Owner`, and requires an exact `owner`/`generation` match against what is currently persisted (`ErrStaleLease` otherwise, `ErrNotHeld` when never acquired).
  - `Release(ctx, ref, owner, generation, now) (Lease, error)` sets `ExpiresAt` to `now` (immediately expired) without clearing `Owner`/`Generation`, so the last holder stays visible for audit while `ref` becomes immediately available to the next `Acquire` by any owner. It requires the same exact `owner`/`generation` match as `Renew`.
- `RepositoryStore` is the only implementation: a thin adapter over `repository.Repository` that encodes `Lease` as its `StoredObject.Payload` and uses `repository.Repository.Save`'s optimistic concurrency (A2-003) as the compare-and-swap for every `Acquire`/`Renew`/`Release`. `RepositoryStore` holds no state of its own — all `Lease` data lives in the wrapped `repository.Repository` — which is what makes restart-survival a direct consequence of A2-003's persistence rather than something `internal/lease` has to reimplement.

## Fencing generation (`A7-004`, implemented)

`Lease.FencingToken() FencingToken` is a thin, named alias over `Lease.Generation` (`FencingToken(l.Generation)`): it is the monotonic fencing token a Lease holder presents on external command/event writes so that a downstream consumer can reject a write carrying a token older than the authoritative `Lease`'s current `Generation`.

Because `Acquire` always advances `Generation` by one from whatever was most recently persisted for `ref` — including when the new acquisition is by a different, replacement owner after the previous holder's lease expired or was released, and including across a control-process restart (`A7-003`'s durability) — a replacement owner's `FencingToken()` is always strictly greater than every `FencingToken()` issued to a previous holder of the same `ref`. `FencingToken(0)` is never issued by `Acquire` and is reserved as the "no token held" sentinel, mirroring `Generation 0`.

## Scope boundary

This slice implements the persisted Lease primitive — `Owner`, `Generation`, `ExpiresAt`, and the minimal owner/generation compare-and-swap needed for `Acquire`/`Renew`/`Release` to be meaningful — together with the `FencingToken` derived from `Generation` (`A7-004`). It intentionally does not implement:

- expiry-driven reconciliation state machines, i.e. a `RECONCILING` state that must clear before re-acquisition (`A7-005`, "Lease expiry reconciliation");
- validating a presented `FencingToken` against the authoritative `Lease` at the point of use so that a stale owner's write is rejected (`A7-006`, "Stale-owner rejection").

Those remain separate, later Release A slices layered on top of this persisted Lease and its `FencingToken`. `internal/orchestration`'s existing RAM-only `Ownership` model independently covers adjacent domain logic (acquire/renew/release, fencing tokens, reconciliation) for workflow ownership; it is untouched by this slice and remains a separate card lineage.

## Evidence

The package tests (`internal/lease/lease_test.go`) cover:

- `Lease.Validate` accepting the never-acquired sentinel and rejecting every inconsistent Owner/Generation/ExpiresAt combination;
- `Lease.Held` for not-yet-acquired, expired and currently-held leases;
- `Acquire` succeeding on first acquisition, after expiry, and on same-owner re-acquisition (each advancing `Generation`), and rejecting acquisition while held by a different owner (`ErrLeaseHeld`);
- `Renew` extending `ExpiresAt` without changing `Generation`, and rejecting a wrong owner/generation (`ErrStaleLease`) or a never-acquired `ref` (`ErrNotHeld`);
- `Release` making `ref` immediately available to a new owner while preserving the last holder for audit, and rejecting a stale caller;
- `TestLeaseSurvivesControlProcessRestart` — the direct evidence for the acceptance criterion "Lease survives control-process restart": a second `RepositoryStore`, constructed independently and sharing only the same backing `repository.Repository`, observes the identical persisted `Lease` and can `Renew` it, exactly as a freshly started control process would after reloading its backing store.
- `TestLeaseFencingTokenMatchesGeneration` — `Lease.FencingToken()` is exactly `FencingToken(Generation)` for every `Generation` a valid `Lease` can hold.
- `TestReplacementOwnerReceivesMonotonicallyNewerFencingTokenAfterExpiry`, `TestReplacementOwnerReceivesMonotonicallyNewerFencingTokenAfterRelease` and `TestReplacementOwnerFencingTokenMonotonicAcrossControlProcessRestart` — the direct evidence for the `A7-004` acceptance criterion "Replacement owner receives monotonically newer generation": a different owner acquiring `ref` after the previous holder's lease expired, after it was explicitly released, or through a freshly constructed `RepositoryStore` sharing only the same backing `repository.Repository` (simulating a control-process restart), always receives a `FencingToken()` strictly greater than the one issued to the previous holder.

This package is isolated from the current/local AIDI runtime: no component in this slice depends on any live/current AIDI database, Forgejo, local VM/runner infrastructure, or message queue.
