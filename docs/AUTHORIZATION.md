# Release A — RBAC authorization-decision domain contract

Status: **IN DEVELOPMENT**

Release A already defines `internal/identity.Role`/`ProjectMembership`
(SPEC §3.2) and the Public API structured-error contract
`internal/apierror.Error` with a closed `Code` set plus deterministic
mapping helpers `FromRevisionConflict` and `FromRateLimitDecision`. There
was previously no domain contract that decides whether a given `Role` may
perform a given operation, and no `FORBIDDEN`/authorization error code — an
authorization denial could not be represented on the Public API boundary at
all. This slice closes that gap with a pure Go domain contract in
`internal/authorization`, mirroring the existing
`internal/apiratelimit`/`internal/apierror` contract style: no HTTP
framework, no persistence, and no queue/VM/runner dependency.

## Types

```go
type Capability string

const (
    CapabilityProjectRead   Capability = "project.read"
    CapabilityProjectMutate Capability = "project.mutate"
)

type Decision struct {
    Allowed bool
    Reason  string
}

func Evaluate(role identity.Role, capability Capability) (Decision, error)
```

- `Capability` — a named permission (e.g. `project.read`,
  `project.mutate`).
- `Decision` — the outcome of evaluating whether a `Role` holds a
  `Capability`. `Reason` is mandatory when `Allowed` is `false`: a denial
  must always be explainable (SPEC §2.4). `Decision.Validate()` enforces
  this invariant, mirroring `apiratelimit.Decision.Validate()`.
- `Evaluate` — a pure function computing `Decision` deterministically from
  a hard-coded Role→Capability matrix only: identical `(role, capability)`
  inputs always produce an identical `Decision`. It reads no clock, global,
  or singleton state, performs no I/O, and spawns no goroutines. An
  unknown/invalid `Role` or an empty/unknown `Capability` returns a wrapped
  sentinel error (`identity.ErrInvalidRole` or
  `authorization.ErrInvalidCapability`), never a panic.

## Role → Capability matrix

Derived from SPEC §3.2 project role semantics:

| Role | `project.read` | `project.mutate` |
| --- | --- | --- |
| `PROJECT_OWNER` | allow | allow |
| `PROJECT_MANAGER` | allow | allow |
| `PRODUCT_OWNER` | allow | allow |
| `CONTRIBUTOR` | allow | allow |
| `VIEWER` | allow | deny |
| `CLIENT` | allow | deny |

`SUPER_ADMIN` is a global system role (SPEC §3.2), not a project `Role`,
and is intentionally out of scope for this pure per-project contract.

## Structured error mapping

`internal/apierror.FromAuthorizationDecision(decision authorization.Decision, correlationID string) (apierror.Error, error)`
maps a denied `Decision` onto the existing `apierror.Error` envelope,
following the `FromRateLimitDecision` mapping pattern exactly:

- `Code = CodeForbidden` (`"FORBIDDEN"`, added to the closed `apierror.Code`
  set).
- `CorrelationID` is preserved and mandatory (rejected by
  `apierror.Error.Validate()` otherwise).
- `Retryable = false` always: an authorization denial cannot be resolved by
  blindly retrying the same request.
- `Reason` is surfaced via `Details["reason"]`.
- An `Allowed == true` decision is rejected: there is nothing to map into an
  error.
- A malformed `Decision` (denied without a `Reason`) is rejected via
  `Decision.Validate()` before any mapping is attempted.

## Scope boundary

This slice deliberately excludes: HTTP/middleware wiring, per-endpoint
capability configuration, dynamic/persisted policy storage, `SUPER_ADMIN`
global-role handling, and enforcement inside `apicommand`/`apiquery`
handlers. Those are later Release A/B slices layered on top of this
contract. No file in `internal/authorization` touches HTTP handlers,
PostgreSQL, NATS, Temporal, Forgejo, or any VM/runner/queue integration,
and the package introduces no new external module dependencies.

Existing `internal/identity`, `internal/apierror`, `internal/apicommand`,
`internal/apiquery` behavior is unchanged beyond the additive `Code`
value and mapping function: no existing test is modified to pass.

## Evidence

- `internal/authorization/authorization_test.go` covers: allow/deny for
  every `identity.Role` × `Capability` pair in the matrix above, rejection
  of an invalid/empty `Role` and an empty/unknown `Capability` (never a
  panic), determinism (identical inputs ⇒ identical `Decision` across
  repeated calls), and `Decision.Validate()` allow/deny cases.
- `internal/apierror/error_test.go` is extended with `CodeForbidden` and
  `FromAuthorizationDecision` cases: the new code passing `Validate()` with
  all mandatory fields present, mapping a denied `Decision` (asserting
  `Code = FORBIDDEN`, `Retryable = false`, `Details["reason"]`), and
  rejection of an `Allowed == true` `Decision`.

`go vet ./...` and
`go test -race ./internal/authorization/... ./internal/apierror/...` pass
clean.
