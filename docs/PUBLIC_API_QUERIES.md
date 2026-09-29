# Release A Public API read-side Query contract

Status: IN DEVELOPMENT

This slice defines a transport-independent Query for reads against canonical
state, mirroring `docs/PUBLIC_API_COMMANDS.md`'s style for the write side.

Required metadata:
- query_id: stable identity for tracing/dedup of read requests
- correlation_id: trace/error correlation (ties to `apierror.Error.CorrelationID`)
- target kind and id
- operation
- parameters: opaque payload, optional

Read/write boundary versus `apicommand.MutationCommand`:
- A Query is read-only and carries no `expected_revision` /
  optimistic-concurrency input. Optimistic concurrency is a write-path
  concern owned by `apicommand.MutationCommand`, not the read side.
- A Query must never trigger a canonical state mutation or side effect.

This slice supports SPEC §13.1's requirement that "Commands отделены от
Queries; поддерживаются idempotency command IDs, correlation, optimistic
concurrency, structured errors и rate limits" — closing the Query side of
that contract now that the Command side (`internal/apicommand`) and the
structured error side (`internal/apierror`) already exist.

Out of scope for this slice (left to later Release A/B slices): HTTP/REST
routing, OpenAPI schema, persistence/read-model execution, pagination/rate-limit
enforcement, and query-result projection shape.

Tests cover the valid query and every required invariant (query id,
correlation id, target kind, target id, operation), plus that `parameters`
may be absent.
