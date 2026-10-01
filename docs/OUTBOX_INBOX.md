# Transactional Outbox / Inbox / Idempotency foundation

Source: approved AIDI v2.0.0 SPEC §4.2 and §8.

## Outbox

A committed canonical transition produces an Outbox record containing the canonical Event and publication state. The future PostgreSQL implementation must persist canonical state, Event Journal and Outbox atomically in one database transaction.

Publishing is not the authority for whether the state transition happened; the committed canonical transaction is authoritative.

Outbox itself (`A3-003`) remains a separate, later Release A slice: this document still defines it only as a domain contract, with no adapter implementation yet.

## Inbox / deduplication (`A3-004`, implemented)

`internal/inbox` implements the Inbox/idempotency domain contract: a duplicate Command ID must produce exactly one logical state-changing result, never a second application of the command or its side effect. Its sole hard dependency is the existing data-access/domain repository boundary package (`internal/repository`, `A2-003`); it does not depend on the Event Journal, atomic state+Event write, Outbox, NATS JetStream, or publisher/consumer contracts (`A3-001`/`A3-002`/`A3-003`/`A3-005`/`A3-006`), which remain separate, later cards.

A Command ID is an opaque caller-supplied string (matching `apicommand.MutationCommand.CommandID`'s shape), not a `canonical.ObjectRef`: idempotency keys are not members of the closed `canonical.Kind` entity hierarchy, so `internal/inbox` defines its own minimal storage contract rather than reusing `internal/repository.Repository`.

- `Record{CommandID, Result}` is the opaque, storage-neutral representation of the single logical result produced for a Command ID. `Result` encoding is owned by the caller.
- `Store` is the storage-neutral contract:
  - `Get(ctx, commandID) (Record, error)` returns the previously produced `Record`, or `ErrNotFound` when none exists yet.
  - `Once(ctx, commandID, produce) (Record, produced bool, error)` ensures `produce` runs at most once per `commandID`: the first call for a `commandID` invokes `produce` and stores its result (`produced=true`); every later or concurrently-racing call for the same `commandID` returns the already-stored `Record` without invoking `produce` again (`produced=false`). A `produce` error is surfaced unchanged and nothing is recorded, so a failed attempt can still be retried.
- `Execute(ctx, store, commandID, produce) (Record, error)` is a convenience wrapper over `Store.Once` for callers that only need the single logical `Record`, not whether their call happened to be the one that produced it.
- `ErrInvalidCommandID` / `ValidateCommandID` reject an empty or whitespace-only Command ID before any state is touched.
- `InMemory` is a non-persistent reference implementation used only to exercise `Store` conformance in unit tests; it is not a persistence adapter. It serializes `Once` (including the `produce` call) behind a single mutex, guaranteeing at-most-once execution per Command ID under concurrency; a production adapter would instead rely on a unique constraint / conditional insert scoped to that Command ID alone.

A consumer identifies delivery by consumer identity, external message ID, and idempotency key; `internal/inbox` models the idempotency-key dimension only. A duplicate delivery or previously claimed idempotency key must not authorize a second application of the command or side effect — `Store.Once`'s at-most-once guarantee is what enforces this.

## Side-effect record

External side effects use a stable idempotency key and a persistent side-effect record. Replay/retry must consult this record before executing the effect again. `internal/inbox.Record` is this side-effect record's storage-neutral shape; `Store.Once` is the consult-before-executing primitive.

## Time semantics

All persisted timestamps are normalized to UTC. Completion/publication/application timestamps cannot precede their corresponding creation/receipt timestamp.

## Next adapters

The Inbox/idempotency domain contract (`A3-004`) is implemented in `internal/inbox` (above); Outbox (`A3-003`) and the Event Journal (`A3-001`/`A3-002`) remain domain contracts only in this document. Concrete PostgreSQL transaction boundaries, unique constraints, NATS JetStream delivery, retry/reconciliation and durable workflow adapters will be implemented in later Release A slices (`A3-003`, `A3-005`, `A3-006`, `A12-003`).

No component in this slice depends on current/legacy AIDI infrastructure.
