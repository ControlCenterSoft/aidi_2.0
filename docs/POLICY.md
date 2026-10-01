# Release A — Foundation Policy domain model

Status: **IN DEVELOPMENT**

> This document covers two additive slices: `A5-001` (the `Policy`
> domain types and validation) and `A5-002` (the `Evaluate` ALLOW/DENY
> decision function). Both live in `internal/policy`.

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

Matching beyond exact tuple equality (wildcards/globs), policy
revision/versioning persistence (`A5-003`), and the
Authorization→Policy→Execution pipeline (`A5-005`) remain out of scope
for `Evaluate`.

## Scope boundary

This slice deliberately excludes: policy revision/versioning persistence
(`A5-003`), the Authorization→Policy→Execution pipeline (`A5-005`),
wildcard/glob matching beyond exact tuple equality, and any HTTP/
persistence/queue wiring. No file in `internal/policy` imports
`net/http`, `database/sql`, NATS, Temporal, or any VM/runner/queue
package, and the package introduces no new external module dependency.

Existing packages are unchanged: `A5-002` is additive to `internal/policy`
only.

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

`gofmt -l .`, `go vet ./...`, and `go test -race ./...` pass clean
repo-wide with no regression to existing packages.
