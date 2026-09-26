# Release A — durable orchestration contracts

Status: **IN DEVELOPMENT**

This slice implements the first transport-neutral durable ownership contract required by the approved AIDI v2.0 baseline.

## Ownership model

A workflow has an explicit persisted ownership state:

- `UNOWNED` — no executor owns the workflow.
- `OWNED` — exactly one owner/attempt holds a time-bounded lease and fencing token.
- `RECONCILING` — the previous lease expired and authoritative reconciliation is required before ownership can be granted again.

Lease expiry is **not** equivalent to failure and does not trigger blind retry. The expired ownership must first enter reconciliation. Only after reconciliation clears the previous attempt may another attempt acquire ownership.

Fencing tokens are monotonic for the workflow. Every write-capable command/event carries the active fencing token. A stale owner or stale token cannot commit a result after ownership changes.

## Attempt isolation

Each attempt binding records:

- immutable Attempt ID;
- exact source SHA/revision;
- isolated workspace ID;
- executor/runner identity.

Task and Attempt remain distinct entities. A replacement attempt receives a new Attempt ID and a later fencing token.

## Command lifecycle

The initial command lifecycle is deliberately small and deterministic:

`PENDING → DISPATCHED → ACKNOWLEDGED → APPLIED`

A command may move to `REJECTED` before `APPLIED`. `APPLIED` and `REJECTED` are terminal. Replaying an already applied command must be handled by the future durable side-effect/idempotency store rather than by repeating the external effect.

## Temporal integration boundary

`DurableWorkflowEngine` is the adapter contract for a durable workflow runtime such as Temporal.

Temporal may schedule, resume and request reconciliation, but it is **not** the canonical state store. Authoritative controllers must validate the current canonical ownership revision and fencing token before accepting a result or side effect.

## NATS integration boundary

`EventPublisher` is the adapter contract for an at-least-once event bus such as NATS JetStream.

Events carry Event ID, workflow/attempt identity and fencing token. Consumers must deduplicate by Event ID; message delivery does not grant workflow ownership and does not replace canonical state transitions.

## Evidence

The package tests cover:

- acquire/renew/release;
- stale owner/token rejection;
- lease expiry requiring reconciliation;
- monotonic fencing across replacement attempts;
- required attempt-isolation coordinates;
- command lifecycle transitions;
- Temporal/NATS-facing command/event envelope invariants.

This package is isolated from the current/local AIDI runtime and has no dependency on Forgejo, local VM/runner infrastructure, local queues or local databases.
