# Release A — Public API rate-limit domain contract

Status: **IN DEVELOPMENT**

This slice implements the remaining part of the SPEC §13.1 requirement that
the Public API boundary support "idempotency command IDs, correlation,
optimistic concurrency, structured errors и rate limits." Release A already
delivers the first four (`internal/apicommand`, `internal/apiquery`,
`internal/apierror`, `canonical.Revision`), and `apierror.Code` already
reserves `RATE_LIMITED` — this slice adds the pure Go domain contract for
*what* gets rate-limited and *how* a limit decision is represented, in
`internal/apiratelimit`, mirroring the existing `internal/apicommand` /
`internal/apiquery` / `internal/apierror` contract style: no HTTP
framework, no persistence, and no queue/VM/runner dependency.

## Types

```go
type LimitKey struct {
    Subject   string
    Operation string
}

type Policy struct {
    Window time.Duration
    Limit  uint64
}

type Decision struct {
    Allowed    bool
    RetryAfter time.Duration
    Limit      uint64
    Remaining  uint64
}

func Evaluate(policy Policy, key LimitKey, requestsInWindow uint64, now time.Time) (Decision, error)
```

- `LimitKey` — the scope a limit applies to: a `Subject` (e.g. a
  service-identity/caller id) performing a named `Operation`. Both fields
  are mandatory (non-empty, non-whitespace), mirroring the
  `apicommand.Target` / `apiquery.Target` validation style.
- `Policy` — a declarative limit: at most `Limit` requests within `Window`.
  `Validate()` rejects a non-positive `Window` or a zero `Limit`. `Policy`
  carries no counters/storage of its own.
- `Decision` — the outcome of evaluating a `Policy` against
  `requestsInWindow` observed usage for a `LimitKey` at a given `now`.
  `RetryAfter` is zero when `Allowed` is `true`, and positive when
  `Allowed` is `false`. `Limit`/`Remaining` report the policy limit and the
  remaining budget in the current window. `Decision.Validate()` enforces
  this invariant (rejecting an `Allowed == true` decision with a non-zero
  `RetryAfter`, or any decision with a negative `RetryAfter`) and is
  invoked by `FromRateLimitDecision` so a manually constructed, malformed
  `Decision` can never be serialized into an API error.
- `Evaluate` — a pure function computing `Decision` deterministically from
  its inputs only: identical `(policy, key, requestsInWindow, now)` always
  produce an identical `Decision`. It reads no clock, global, or singleton
  state other than the `now` argument, performs no I/O, and spawns no
  goroutines.

## Allow/deny rule

- `requestsInWindow < policy.Limit` ⇒ `Allowed = true`, `RetryAfter = 0`,
  `Remaining = policy.Limit - requestsInWindow`.
- `requestsInWindow >= policy.Limit` ⇒ `Allowed = false`,
  `RetryAfter = policy.Window` (this pure contract has no visibility into
  when the current window started, so it reports the full window duration
  as the conservative retry hint), `Remaining = 0`.

Verified at the boundary values `limit-1` (allowed), `limit` (denied), and
`limit+1` (denied).

## Structured error mapping

`internal/apierror.FromRateLimitDecision(decision apiratelimit.Decision, correlationID string) (apierror.Error, error)`
maps a denied `Decision` onto the existing `apierror.Error` envelope,
following the `FromRevisionConflict` mapping pattern exactly:

- `Code = CodeRateLimited`.
- `CorrelationID` is preserved and mandatory (rejected by
  `apierror.Error.Validate()` otherwise).
- `Retryable = true` only when `RetryAfter > 0`.
- `RetryAfter` is surfaced via `Details["retry_after_ms"]` (milliseconds).
  `Details["limit"]` and `Details["remaining"]` are also included for
  machine-readable context.
- An `Allowed == true` decision is rejected: there is nothing to map into
  an error.
- A malformed `Decision` (e.g. `Allowed == true` with a non-zero
  `RetryAfter`, or a negative `RetryAfter`) is rejected via
  `Decision.Validate()` before any mapping is attempted.

## Scope boundary

This slice deliberately excludes: actual token-bucket/sliding-window
storage, HTTP middleware wiring, per-endpoint policy configuration/tuning,
distributed counters, and enforcement inside `apicommand`/`apiquery`
handlers. Those are later Release A/B slices layered on top of this
contract. No file in `internal/apiratelimit` touches HTTP handlers,
PostgreSQL, NATS, Temporal, Forgejo, or any VM/runner/queue integration,
and the package introduces no new external module dependencies.

Existing `internal/apierror`, `internal/apicommand`, `internal/apiquery`
behavior is unchanged: the only new public surface is the additive
`apierror.FromRateLimitDecision` function.

## Evidence

- `internal/apiratelimit/ratelimit_test.go` covers: valid/invalid `Policy`,
  valid/invalid `LimitKey`, `Evaluate` allow/deny at the `limit-1`/`limit`/
  `limit+1` boundary values, rejection of malformed inputs (zero `now`),
  and determinism (identical inputs ⇒ identical `Decision` across repeated
  calls).
- `internal/apierror/error_test.go` is extended with
  `FromRateLimitDecision` cases: mapping a denied `Decision` (with a
  positive `RetryAfter`, asserting `Retryable = true` and
  `Details["retry_after_ms"]`), a denied `Decision` with `RetryAfter == 0`
  (asserting `Retryable = false`), and rejection of an `Allowed == true`
  `Decision`.

`go test ./internal/apiratelimit/... ./internal/apierror/... -race`,
`go vet ./...`, and `gofmt -l .` all pass clean.
