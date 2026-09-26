# PostgreSQL canonical state transaction contract

Source: approved AIDI v2.0.0 SPEC §4.2.

## Mandatory command transaction

A state-changing authoritative controller must execute the following steps in **one PostgreSQL transaction**:

1. Read/lock the current canonical object revision.
2. Compare the persisted revision with the command's exact expected revision.
3. Reject stale expected revisions; last-write-wins is forbidden.
4. Validate the proposed state transition.
5. Advance the canonical object revision exactly once.
6. Insert the corresponding immutable Event Journal row.
7. Insert the matching Outbox row referencing that Event.
8. Commit all changes atomically.

If any step fails, canonical state, Event Journal and Outbox must all roll back.

## Event Journal

The Event Journal is append-only. Database trigger protection rejects UPDATE and DELETE independently of application logic.

Event identity and the tuple `(object_kind, object_id, revision)` are unique.

## Outbox

Outbox publication is delivery state only. A message-bus outage does not undo an already committed canonical transition. Unpublished rows remain queryable through the partial pending index.

## Inbox

The primary delivery identity is scoped by `(consumer, message_id)`. Idempotency is independently constrained by `(consumer, idempotency_key)`, preventing a redelivery from applying the same command twice without causing unrelated consumers to collide.

## Side effects

A persistent side-effect record must be established before or atomically with an externally visible effect according to the adapter's strategy. Retry/replay checks the idempotency key before repeating the effect.

## Isolation

These migrations are source artifacts only. They are not applied to any current/legacy AIDI database or VM.
