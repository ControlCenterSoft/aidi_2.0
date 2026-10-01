# Release A — Foundation test tools

Status: **IN DEVELOPMENT**

Release A's Policy/Authorization/Tool-Broker stack
(`internal/authorization`, `internal/policy`, `internal/toolregistry`,
`internal/authzpipeline`) is already delivered and its own test files each
work correctly, but `authzpipeline_test.go` and sibling test files each
hand-roll near-identical `allowPolicy()`, `denyPolicy()`,
`noMatchPolicy()`, `baseRequest()`-style fixtures. This slice (backlog
`A5-006`, depends only on the completed `A5-005` `internal/authzpipeline`
slice) closes that duplication with a new, additive, pure Go test-support
package, `internal/foundationtest`, mirroring the existing
`internal/toolregistry`/`internal/authzpipeline` scope-boundary style.

## Types and functions

```go
const (
    Subject = "alice"
    Resource = "project-1"
    Action = "deploy"

    Role identity.Role = identity.RoleProjectOwner
    Capability authorization.Capability = authorization.CapabilityProjectMutate

    ActiveToolID   toolregistry.ToolID = "tool-1"
    InactiveToolID toolregistry.ToolID = "tool-2"
)

func AllowPolicy() policy.Policy
func DenyPolicy() policy.Policy
func NoMatchPolicy() policy.Policy

func ActiveTool() toolregistry.Tool
func InactiveTool() toolregistry.Tool

func NewActiveRegistry() *toolregistry.Registry
func NewInactiveRegistry() *toolregistry.Registry

func BaseRequest() authzpipeline.Request
```

- `AllowPolicy` — a minimal, independently valid `policy.Policy` with a
  single ALLOW `Statement` for `(Subject, Resource, Action)`.
- `DenyPolicy` — the same tuple with both a matching ALLOW `Statement` and
  an explicit DENY `Statement`, exercising SPEC §12.1's "explicit deny
  overrides allow".
- `NoMatchPolicy` — a `Policy` whose only `Statement` does not match
  `(Subject, Resource, Action)`, exercising `policy.Evaluate`'s
  default-deny path.
- `ActiveTool` / `InactiveTool` — a minimal, independently valid
  `toolregistry.Tool` in `toolregistry.StatusActive` and
  `toolregistry.StatusDeprecated` respectively, for positive/negative Tool
  Broker test cases.
- `NewActiveRegistry` / `NewInactiveRegistry` — a `*toolregistry.Registry`
  with `ActiveTool`/`InactiveTool` already registered.
- `BaseRequest` — a baseline, independently valid `authzpipeline.Request`:
  a valid `(Role, Capability)` pair, `AllowPolicy`, the
  `(Subject, Resource, Action)` tuple `AllowPolicy` ALLOWs, and
  `ActiveToolID`. Pairing `BaseRequest()` with `NewActiveRegistry()` yields
  an `Allowed: true` `authzpipeline.Authorize` outcome; callers exercising
  a denial path override individual fields (`Policy`, `ToolID`, `Role`) as
  needed.

## Validation guarantee

Every exported fixture constructor returns a value that independently
passes its own domain type's `Validate()`/registration path:
`AllowPolicy()`, `DenyPolicy()`, and `NoMatchPolicy()` each pass
`policy.Policy.Validate()`; `ActiveTool()` and `InactiveTool()` each pass
`toolregistry.Tool.Validate()` and register successfully via
`toolregistry.Registry.Register`; `BaseRequest()`'s embedded `Policy`,
`Role`, and `ToolID` each pass their own `Validate()`. This is covered by
`foundationtest_test.go`.

## Scope boundary

This slice deliberately excludes: wiring these fixtures into the existing
hand-rolled test files in `authzpipeline_test.go` and siblings (a
follow-up), any new runtime behavior, and `RA-GATE-10` itself (the gate
closes only after its own evidence pass). No file in `internal/foundationtest`
imports `net/http`, `database/sql`, NATS, Temporal, Forgejo, or any
VM/runner/queue package, and the package introduces no new external module
dependency: it imports only `internal/authorization`, `internal/identity`,
`internal/policy`, `internal/toolregistry`, and `internal/authzpipeline`
(each already part of Release A). Existing packages are unchanged: this is
a new, additive package only.

## Evidence

`internal/foundationtest/foundationtest_test.go` covers: each policy
fixture (`AllowPolicy`, `DenyPolicy`, `NoMatchPolicy`) independently passing
`Validate()`; each tool fixture (`ActiveTool`, `InactiveTool`) independently
passing `Validate()` and carrying the expected `Status`;
`NewActiveRegistry`/`NewInactiveRegistry` registering their respective
tool and it being retrievable via `Registry.Get`; `BaseRequest()`'s embedded
`Policy`/`Role`/`ToolID` independently passing `Validate()`; and one
end-to-end `authzpipeline.Authorize` composition test asserting the
expected ALLOW, explicit-DENY-overrides-ALLOW, and default-deny outcomes,
plus a non-ACTIVE-tool denial.

`gofmt -l .`, `go vet ./...`, and `go test -race ./...` pass clean
repo-wide with no regression to existing packages.
