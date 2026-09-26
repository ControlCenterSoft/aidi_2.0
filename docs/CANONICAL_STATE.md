# Canonical state foundation

Source: approved AIDI v2.0.0 SPEC §4.2.

## Event envelope

Every persisted significant transition must be representable by a versioned canonical Event containing:

- Event ID
- Event type
- Object kind and ID
- Object revision
- UTC timestamp
- Actor
- Correlation ID
- Optional causation ID for non-root chains
- Schema version

An invalid envelope is rejected before persistence or publication.

## Optimistic concurrency

Critical state uses exact expected-revision checks. Last-write-wins is not an acceptable conflict policy.

A stale caller receives a typed revision-conflict error containing both expected and actual revisions. The caller must reconcile against current canonical state rather than silently overwrite it.

## Boundary

This slice defines domain contracts only. PostgreSQL transactions, Transactional Outbox, Inbox/dedup, event publication, NATS JetStream, and durable workflow integration are separate Release A slices.

The GitHub development track remains isolated from the current/legacy AIDI runtime and state.
