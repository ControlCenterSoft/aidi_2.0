# Release A — Foundation Policy domain model

Status: **IN DEVELOPMENT**

> This document covers three additive slices: `A5-001` (the `Policy`
> domain types and validation), `A5-002` (the `Evaluate` ALLOW/DENY
> decision function), and `A5-003` (the `PolicyRevision`
> revision/versioning domain contract). All three live in
> `internal/policy`.

SPEC §4.1 lists the canonical entity hierarchy as `Installation →
Identity/User → Workspace → Project → Specification → Requirement →
Release → Feature → Task → Workflow → Attempt →
ChangeSet/Verification/Evidence/Artifact`, plus the additional first-class
entities `Decision, Approval, Risk, ChangeRequest, Problem, RecoveryCase,
Policy, Resource, Event/Audit, OperationalKnowledge`. SPEC §12.1 further
requires the chain `Identity → Authentication → Role → Policy →
Authorization → Action → Audit`, where "RBAC дополняется policy engine;
explicit deny overrides allow". Before this slice there was no domain type
at all representing a `Policy` as a first-class canonical entity. This
slice (backlog `A5-001`) closes that gap with a pure Go domain contract in
`internal/policy`, mirroring the existing
`internal/authorization`/`internal/toolregistry` contract style: no HTTP
handler, no persistence, and no process/VM/queue/runner integration.

## Types

```go
type PolicyID string

type Effect string

const (
    EffectAllow Effect = "ALLOW"
    EffectDeny  Effect = "DENY"
)

type Statement struct {
    Subject  string
    Resource string
    Action   string
    Effect   Effect
}

type Policy struct {
    ID         PolicyID
    Statements []Statement
}

func (id PolicyID) Validate() error
func (e Effect) Validate() error
func (p Policy) Validate() error
func (p Policy) HasExplicitDeny(subject, resource, action string) bool

type Decision struct {
    Allowed bool
    Reason  string
}

func Evaluate(policy Policy, subject, resource, action string) (Decision, error)
```

- `PolicyID` — the canonical identifier of a Policy, aligned with the
  `internal/canonical` identifier conventions (non-empty, non-blank
  string).
- `Effect` — a closed set: `ALLOW`, `DENY`.
- `Statement` — a named `Subject`/`Resource`/`Action`/`Effect` tuple. No
  wildcard/glob expansion logic is included; that belongs to the later
  evaluation slice (`A5-002`).
- `Policy` — the first-class canonical aggregate: a `PolicyID` plus an
  ordered `Statements` list.

## Validation

`PolicyID.Validate()` rejects an empty/whitespace-only value with the
wrapped sentinel error `ErrInvalidPolicyID`.

`Effect.Validate()` rejects an empty value or any value outside the
closed `ALLOW`/`DENY` set with the wrapped sentinel error
`ErrInvalidEffect`. It never panics.

`Policy.Validate()` rejects, each via the single wrapped sentinel error
`ErrInvalidPolicy`:

- an invalid `PolicyID`;
- an empty `Statements` list;
- any `Statement` with a blank `Subject`, `Resource`, or `Action`;
- any `Statement` with an unknown `Effect`.

`Validate()` does **not** resolve conflicts between an `ALLOW` and a
`DENY` `Statement` sharing the identical `(Subject, Resource, Action)`
tuple: such a policy is structurally valid — it only rejects malformed
input.

`Policy.HasExplicitDeny(subject, resource, action)` returns `true` only
when a `DENY` `Statement` exists for the exact `(Subject, Resource,
Action)` tuple. It is deterministic, performs no I/O, and never panics on
a zero-value `Policy` (it returns `false`). `Evaluate` (below) uses it to
implement "explicit deny overrides allow" (SPEC §12.1) without
re-deriving tuple matching.

## Evaluation (`A5-002`)

`Evaluate(policy Policy, subject, resource, action string) (Decision, error)`
resolves the ALLOW/DENY `Decision` for a `(Subject, Resource, Action)`
tuple against `policy`'s ordered `Statements`, per SPEC §12.1 ("RBAC
дополняется policy engine; explicit deny overrides allow"):

1. **Explicit deny overrides allow**: if any `Statement` matching the
   exact tuple has `Effect == EffectDeny`, the result is
   `Decision{Allowed: false, Reason: "..."}`, regardless of any matching
   `ALLOW` statement.
2. Otherwise, if at least one `Statement` matching the exact tuple has
   `Effect == EffectAllow`, the result is `Decision{Allowed: true}`.
3. Otherwise (no `Statement` matches the tuple), the result is a
   **default-deny**: `Decision{Allowed: false, Reason: "..."}` explaining
   that no matching statement was found.

`Evaluate` rejects an invalid `policy` (via `Policy.Validate()`, wrapped
`ErrInvalidPolicy`) and an empty/whitespace-only `subject`, `resource`, or
`action` (wrapped `ErrInvalidTuple`), and never panics. It is a **pure
function**: no I/O, no clock, no global/singleton state — identical
inputs always produce an identical `Decision` (mirrors the existing
`authorization.Evaluate` / `apiratelimit.Evaluate` pattern already in this
codebase).

Matching beyond exact tuple equality (wildcards/globs) remains out of
scope for `Evaluate`.

## Revision/versioning (`A5-003`)

```go
type PolicyRevision struct {
    Policy   Policy
    Revision canonical.Revision
}

func (pr PolicyRevision) Validate() error

func NextPolicyRevision(current PolicyRevision, next Policy, expected canonical.Revision) (PolicyRevision, error)
```

`PolicyRevision` pairs an immutable `Policy` snapshot with a
`canonical.Revision` (the existing `internal/canonical` revision
primitive from `A1-003`), mirroring the optimistic-concurrency boundary
already used by `internal/repository` (`A2-003`). This is a **pure domain
contract only**: no persistence, no HTTP, no queue/VM/runner wiring — the
actual persistence/storage adapter, revision history retrieval/listing,
HTTP/API exposure, and policy diffing/audit trail are out of scope and
left to later, separate cards.

`PolicyRevision.Validate()` rejects, via the wrapped sentinel error
`ErrInvalidPolicyRevision`: an invalid underlying `Policy` (per
`Policy.Validate()`), and a zero `Revision`.

`NextPolicyRevision(current, next, expected)`:

1. Validates `next` via `Policy.Validate()`, returning the wrapped
   `ErrInvalidPolicy` on failure.
2. Requires `next.ID == current.Policy.ID`, returning the wrapped
   sentinel error `ErrPolicyIDMismatch` otherwise.
3. Enforces optimistic concurrency via
   `canonical.CheckExpectedRevision(expected, current.Revision)`: a
   stale/incorrect `expected` is rejected via the wrapped
   `canonical.ErrRevisionConflict` / `*canonical.RevisionConflictError`,
   without mutating `current`.
4. Advances the revision via `canonical.NextRevision(current.Revision)`
   and returns a new `PolicyRevision{Policy: next, Revision: ...}`.

The first revision of a given `PolicyID` starts at `expected = 0` and
advances to revision `1`; subsequent successful calls advance `N → N+1`,
matching the `internal/repository` convention. `NextPolicyRevision` is a
pure function: no I/O, no clock, no global/singleton state — identical
inputs always produce an identical result.

No file in `internal/policy` imports `net/http`, `database/sql`, or any
queue/workflow runtime package, and this slice introduces no new
`go.mod` dependency; `internal/policy`'s only internal dependency beyond
the Go standard library is `internal/canonical`.

## Authorization → Policy → Execution pipeline (`A5-005`)

`internal/authzpipeline.Authorize(registry *toolregistry.Registry, req authzpipeline.Request) (authzpipeline.Decision, error)`
composes this package's `Evaluate` with
`internal/authorization.Evaluate` and `internal/toolregistry.Registry.Get`
into a single deterministic pre-execution gate, with a fixed precedence
order:

1. **RBAC denial** (`internal/authorization.Evaluate`) short-circuits
   first: if the `Role` does not hold the `Capability`, the policy and
   tool are not evaluated at all.
2. **Explicit policy `DENY`**, then
3. **policy default-deny** (no matching statement) — both resolved by
   this package's `Evaluate`, unchanged.
4. **Non-ACTIVE or unregistered tool**: the `ToolID` must resolve to a
   registered `Tool` with `Status == toolregistry.StatusActive`.

The request is allowed only when RBAC allows **and** policy allows **and**
the tool is `StatusActive`. Every denial carries a non-empty,
distinguishable `Reason` identifying which stage denied the request (SPEC
§2.4). `Authorize` is a pure function (no I/O, clock, goroutines, or
global/singleton state); a malformed `Role`, `Capability`, `Policy`,
tuple, or `ToolID` returns a wrapped sentinel error, never a panic. See
`internal/authzpipeline`'s package doc for the full contract.

`internal/policy.Evaluate`/`Policy`/`Decision` are unchanged by A5-005:
`authzpipeline` only composes the existing contracts, it does not modify
this package's public API.

## Scope boundary

This slice deliberately excludes: policy revision/versioning
*persistence* (an actual storage adapter, revision history
retrieval/listing, and HTTP/API exposure — left to later, separate
cards; the `A5-003` pure domain contract itself is implemented, see
above), wildcard/glob matching beyond exact tuple equality, a
persisted audit trail, `SUPER_ADMIN` global-role handling, and any HTTP/
persistence/queue wiring. No file in `internal/policy` or
`internal/authzpipeline` imports `net/http`, `database/sql`, NATS,
Temporal, or any VM/runner/queue package, and neither package introduces
any new external module dependency.

Existing packages are unchanged: `A5-005`'s `internal/authzpipeline` is a
new, additive package only; it does not change `internal/policy`,
`internal/authorization`, or `internal/toolregistry` public APIs.

## Evidence

`internal/policy/policy_test.go` covers: valid/invalid `PolicyID`
validation; `Effect` validation (valid `ALLOW`/`DENY`, empty, and unknown
values); `Policy.Validate()` for a well-formed policy and every rejection
case above (invalid `PolicyID`, empty `Statements`, blank
`Subject`/`Resource`/`Action`, unknown `Effect`); `HasExplicitDeny`
true/false cases including an `ALLOW`-only policy, a `DENY`-only policy,
and a policy with both an `ALLOW` and a `DENY` statement for the same
tuple (which remains structurally valid); zero-value `Policy` safety; and
`Evaluate` coverage for explicit-deny-overrides-allow (matching ALLOW and
DENY on the same tuple), allow-only match, deny-only match, default-deny
(no matching statement), invalid `Policy` and invalid/empty tuple inputs
(table-driven, no panics), and determinism (identical inputs produce an
identical `Decision`).

`internal/policy/revision_test.go` covers (`A5-003`): valid/invalid
`PolicyRevision.Validate()` (invalid underlying `Policy`, zero
`Revision`); `NextPolicyRevision` success paths (`0 → 1` first revision,
`N → N+1` subsequent revision); rejection of mismatched `PolicyID`
between `current` and `next` (via `ErrPolicyIDMismatch`); rejection of a
stale `expected` revision (via the wrapped
`canonical.ErrRevisionConflict` / `*canonical.RevisionConflictError`)
with no mutation of `current`; rejection of an invalid `next` `Policy`
(via `ErrInvalidPolicy`); revision exhaustion (via
`canonical.ErrRevisionExhausted`); and determinism (identical inputs
produce an identical result, no I/O/clock/global state). `go test
./internal/policy/... -cover` reports 100% statement coverage.

`gofmt -l .`, `go vet ./...`, and `go test -race ./...` pass clean
repo-wide with no regression to existing packages.

`internal/authzpipeline/authzpipeline_test.go` covers (`A5-005`): every
precedence branch (RBAC allow+policy allow+tool active ⇒ allow; RBAC deny
short-circuiting before policy/tool evaluation; explicit policy deny
overriding a matching allow; policy default-deny; unregistered tool;
`DEPRECATED`/`DISABLED` non-active tool status), every error branch
(invalid `Role`, invalid/unknown `Capability`, invalid `Policy`, invalid
tuple, invalid/empty `ToolID`), a nil-`Registry` safety case, and
determinism (identical inputs across repeated calls produce an identical
`Decision`). `go test ./internal/authzpipeline/... -cover` reports 100%
statement coverage.
