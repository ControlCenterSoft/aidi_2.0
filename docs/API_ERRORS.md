# Release A — Public API structured error contract

Status: **IN DEVELOPMENT**

This slice implements the SPEC §13.1 requirement that the Public API returns
**structured errors** correlated to the command/query that produced them, as
a pure Go domain contract in `internal/apierror` (no HTTP framework, no
persistence, no queue/VM dependency), mirroring the existing
`internal/canonical`/`internal/orchestration` contract style.

## Envelope fields

```go
type Error struct {
    Code          Code
    Message       string
    CorrelationID string
    Retryable     bool
    Details       map[string]string // optional
}
```

- `Code` — one of the closed set `VALIDATION`, `CONFLICT`, `NOT_FOUND`,
  `RATE_LIMITED`, `INTERNAL`. Any other value is rejected.
- `Message` — human-readable description of the failure.
- `CorrelationID` — **mandatory**. Ties the error back to the originating
  command/query (SPEC §13.1 "correlation" requirement). An `Error` without a
  `CorrelationID` is invalid and cannot be constructed via `New`, nor pass
  `Validate`.
- `Retryable` — whether the caller may safely retry the same request as-is.
  `New`/mapping helpers never set this `true` for a conflict: the caller
  must reconcile (re-fetch and retry with the correct expectation), not
  blindly resend.
- `Details` — optional machine-readable context (e.g. conflicting
  revisions). Omitted from JSON when empty/nil.

JSON field names (`code`, `message`, `correlation_id`, `retryable`,
`details`) are stable and are the single source of truth for Public API
transport wiring layered on top of this contract in later Release A/B
slices.

## Non-waivable rules

- **Correlation is mandatory.** `Error.Validate()` (and the `New`
  constructor) reject an empty `Code`, empty `Message`, empty
  `CorrelationID`, or an unknown `Code` value. There is no supported way to
  construct or validate a structured error without a correlation ID.
- **Revision conflicts map deterministically and are never retryable.**
  `FromRevisionConflict` converts a
  `canonical.RevisionConflictError` into an `apierror.Error` with
  `Code = CONFLICT`, preserves the original `Expected`/`Actual` revisions
  unchanged in `Details` (`expected_revision`/`actual_revision`), and always
  sets `Retryable = false` — consistent with `docs/CANONICAL_STATE.md`'s
  reconcile-not-retry rule for stale writers.

## Scope boundary

This slice deliberately excludes HTTP transport wiring, rate-limit
enforcement, and Public API routing — those are later Release A/B slices
layered on top of this contract. No file in `internal/apierror` touches HTTP
handlers, PostgreSQL, NATS, Temporal, Forgejo, or any VM/runner/queue
integration, and the package introduces no new external module
dependencies.

## Evidence

The package tests (`internal/apierror/error_test.go`) cover:

- construction and validation of a valid `Error` with all required fields;
- rejection of a missing `Code`, an unknown `Code`, a missing/whitespace-only
  `Message`, and a missing `CorrelationID`;
- stable-field-name JSON round-trip, including the `details` field being
  omitted when absent;
- the `RevisionConflictError → apierror.Error` mapping: `Code = CONFLICT`,
  unchanged `Expected`/`Actual` revisions in `Details`, `Retryable = false`,
  and rejection of a `nil` conflict.

`go test ./internal/apierror/... -race`, `gofmt -l internal/apierror`, and
`go vet ./internal/apierror/...` all pass clean.
