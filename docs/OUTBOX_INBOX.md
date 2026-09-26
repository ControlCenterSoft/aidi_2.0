# Transactional Outbox / Inbox / Idempotency foundation

Source: approved AIDI v2.0.0 SPEC §4.2 and §8.

## Outbox

A committed canonical transition produces an Outbox record containing the canonical Event and publication state. The future PostgreSQL implementation must persist canonical state, Event Journal and Outbox atomically in one database transaction.

Publishing is not the authority for whether the state transition happened; the committed canonical transaction is authoritative.

## Inbox / deduplication

A consumer identifies delivery by:

- consumer identity;
- external message ID;
- idempotency key.

A duplicate delivery or previously claimed idempotency key must not authorize a second application of the command or side effect.

## Side-effect record

External side effects use a stable idempotency key and a persistent side-effect record. Replay/retry must consult this record before executing the effect again.

## Time semantics

All persisted timestamps are normalized to UTC. Completion/publication/application timestamps cannot precede their corresponding creation/receipt timestamp.

## Next adapters

The current slice intentionally defines domain contracts only. Concrete PostgreSQL transaction boundaries, unique constraints, NATS JetStream delivery, retry/reconciliation and durable workflow adapters will be implemented in later Release A slices.

No component in this slice depends on current/legacy AIDI infrastructure.
