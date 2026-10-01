# Release A — Tool Registry domain contract

Status: **IN DEVELOPMENT**

SPEC §8.4 (Software and Service Catalog) requires that AIDI be able to apply
third-party free tools at engineering stages once they pass
source/license/security/integrity/compatibility qualification, and that the
Catalog show purpose, version, license, status/health, resource needs,
dependencies, and a recommendation. SPEC §17.3 additionally requires that
every AIDI Orb Assistant action pass through a Tool Broker. Before this
slice there was no domain type at all representing a catalog Tool entry or
its provisioning policy. This slice (backlog `A5-004`, depends only on the
completed `A1-001` foundation slice) closes that gap with a pure Go domain
contract in `internal/toolregistry`, mirroring the existing
`internal/authorization`/`internal/canonical` contract style: no HTTP
handler, no persistence, and no process/VM/queue/runner integration.

## Types

```go
type ToolID string

type Status string

const (
    StatusActive     Status = "ACTIVE"
    StatusDeprecated Status = "DEPRECATED"
    StatusDisabled   Status = "DISABLED"
)

type ProvisionPolicy string

const (
    ProvisionPolicyAuto         ProvisionPolicy = "AUTO"
    ProvisionPolicyRequestAdmin ProvisionPolicy = "REQUEST_ADMIN"
    ProvisionPolicyDeny         ProvisionPolicy = "DENY"
)

type ResourceNeeds struct {
    CPUCores int
    MemoryMB int
    DiskMB   int
}

type Tool struct {
    ID              ToolID
    Name            string
    Version         string
    License         string
    Status          Status
    ProvisionPolicy ProvisionPolicy
    ResourceNeeds   ResourceNeeds
    Dependencies    []ToolID
}

type Registry struct { /* unexported */ }

func NewRegistry() *Registry
func (r *Registry) Register(tool Tool) error
func (r *Registry) Get(id ToolID) (Tool, bool)
func (r *Registry) List() []Tool
```

- `ToolID` — the canonical identifier of a Tool, aligned with the
  `internal/canonical` identifier conventions (non-empty, non-blank
  string).
- `Status` — a closed, SPEC §8.4 lifecycle set: `ACTIVE`, `DEPRECATED`,
  `DISABLED`.
- `ProvisionPolicy` — the closed SPEC §8.4 provision/update policy set:
  `AUTO`, `REQUEST_ADMIN`, `DENY`.
- `ResourceNeeds` — the advisory, coarse resource footprint shown by the
  Catalog (CPU/RAM/disk). This domain contract neither enforces units nor
  reserves/allocates resources; that belongs to the later Resource
  Management slice (SPEC §10.4).
- `Tool` — a Software and Service Catalog entry: name, version, license,
  `Status`, `ProvisionPolicy`, `ResourceNeeds`, and a `Dependencies` list of
  other `ToolID`s it depends on.
- `Registry` — an in-memory, pure-function domain aggregate of `Tool`
  entries keyed by `ToolID`. The zero value is not ready for use; construct
  one with `NewRegistry()`.

## Validation

`Tool.Validate()` rejects, each via the single wrapped sentinel error
`ErrInvalidTool`:

- an empty/whitespace-only `ToolID`, `Name`, `Version`, or `License`;
- an empty or unknown `Status`;
- an empty or unknown `ProvisionPolicy`.

`Registry.Register(tool)` first calls `tool.Validate()`, then additionally
rejects a `ToolID` that is already registered with the wrapped sentinel
error `ErrDuplicateTool`. A failed `Register` call never mutates the
`Registry`.

`Registry.Get(id)` returns the registered `Tool` and `true`, or a zero
`Tool` and `false`. `Registry.List()` returns every registered `Tool`
ordered deterministically by `ToolID` (an empty, non-nil slice for an empty
`Registry`).

`Registry` is otherwise a pure, deterministic, side-effect-free in-memory
structure: no global/singleton state, no clock, no I/O. Every exported
method is safe to call on a zero-value `*Registry`/`Registry` without
panicking (`Get` behaves as empty; `Register` lazily initializes storage).

## Scope boundary

This slice deliberately excludes: Tool Broker enforcement (actual
admission/authorization of a tool action, SPEC §17.3), any Catalog UI or
presentation, Catalog persistence, and runtime process/VM/queue/runner
execution of a tool. Those are later Release A/B slices layered on top of
this contract. No file in `internal/toolregistry` touches HTTP handlers,
PostgreSQL, NATS, Temporal, Forgejo, or any VM/runner/queue integration, and
the package introduces no new external module dependencies.

Existing packages are unchanged: this is a new, additive package only.

## Evidence

`internal/toolregistry/toolregistry_test.go` covers: valid registration and
`Get` round-trip, duplicate `ToolID` rejection (and that the registry stays
usable afterwards), each invalid-field case (`ErrInvalidTool` for empty/
unknown `ToolID`/name/version/license/`Status`/`ProvisionPolicy`), `Status`
and `ProvisionPolicy` validation in isolation, `List()` on an empty registry
and ordered by `ToolID` on a populated one, and that `Get`/`Register` never
panic on a zero-value `Registry`.

`gofmt -l .`, `go vet ./...`, and `go test -race ./...` pass clean
repo-wide with no regression to existing packages.
