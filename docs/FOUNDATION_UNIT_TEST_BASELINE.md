# Release A foundation unit-test baseline (A12-001)

This document records the bounded Release A unit-test baseline for the Go
foundation delivered by A0-A11.

## Baseline

Every mandatory foundation package below must retain at least one package-local
`*_test.go` file with a top-level `Test*` function. The repository-wide unit
suite remains the authoritative execution check and is already run by the
GitHub-hosted AIDI CI as:

```sh
go test -race -coverprofile=coverage.out ./...
```

The A12-001 baseline guard lives in
`internal/foundationbaseline/baseline_test.go`; it fails when a mandatory
package loses its package-local unit tests.

Mandatory packages:

- `internal/apierror`
- `internal/apicommand`
- `internal/apiquery`
- `internal/apiratelimit`
- `internal/authorization`
- `internal/authzpipeline`
- `internal/buildinfo`
- `internal/canonical`
- `internal/commandendpoint`
- `internal/config`
- `internal/foundationtest`
- `internal/health`
- `internal/identity`
- `internal/inbox`
- `internal/integrity`
- `internal/lease`
- `internal/orchestration`
- `internal/persistence/postgres`
- `internal/policy`
- `internal/repository`
- `internal/specification`
- `internal/toolregistry`

`cmd/aidi` is the composition entry point rather than a foundation library
package; its buildability remains covered by the CI build step and repository
unit run.

## Gap closed by A12-001

At the selected-card handoff, `internal/buildinfo` was the only mandatory
foundation package without a package-local unit test. A12-001 adds direct tests
for explicit provenance values and deterministic fallback values
(`dev`/`unknown`/`unknown`), then locks the package-level baseline with the
guard above.

No runtime behavior, workflow, automation-control logic, or Release A canonical
manifest is changed by this slice.
