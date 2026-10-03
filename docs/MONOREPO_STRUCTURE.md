# AIDI 2.0 monorepo structure

Status: **Release A canonical bootstrap layout**

This document records the repository layout used by the approved AIDI 2.0
bootstrap baseline. It does not expand Release A scope; it makes the existing
Go, Python worker, Web, documentation, and automation boundaries explicit and
machine-verifiable for `A0-002`.

## Required layout

| Path | Responsibility |
| --- | --- |
| `cmd/aidi/` | Go entry point for the AIDI core service |
| `internal/` | Internal Go domain, application, and infrastructure packages |
| `workers/src/` | Python AI/ML worker package |
| `workers/tests/` | Python worker tests |
| `web/src/` | React/TypeScript Web UI source |
| `docs/` | Canonical product, architecture, and operational documentation |
| `.github/workflows/` | GitHub-only CI and autonomous-development workflows |
| `.github/scripts/` | Deterministic CI/automation validation helpers |
| `.automation/` | Release tracker metadata and isolated automation state |

The root remains the integration boundary for shared build/development files,
including `go.mod`, `Makefile`, `README.md`, and `DEVELOPMENT.md`.

Additional directories may be introduced by later approved cards. The
`A0-002` verifier asserts only the required baseline paths so future approved
expansion does not require weakening this contract.

## Verification

`.github/scripts/test_monorepo_structure.py` validates the required layout and
the key scaffold files for the Go core, Python workers, and Web UI. AIDI CI runs
this verifier in the Python job before language-specific qualification.

This GitHub monorepo remains isolated from current/legacy AIDI infrastructure
as required by `DEVELOPMENT.md`.
