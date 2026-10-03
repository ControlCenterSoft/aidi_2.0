# AIDI 2.0

AIDI v2.0.0 — autonomous AI development infrastructure.

## Baseline status

- Specification: **APPROVED / 100%**
- Roadmap: **APPROVED**
- Acceptance Matrix / ПМИ: **APPROVED**
- Controlled Cutover Plan: **APPROVED**
- Active development branch: `development`
- Release sequence: **A → B → C → D → E → F → G → H → I → J → K → RC**

## Technology baseline

- OS: Debian 13
- Core: Go
- AI/ML workers: Python
- Web UI: React + TypeScript + Vite
- PostgreSQL 16 + Patroni + etcd
- Temporal
- NATS JetStream
- Gitea integration
- SeaweedFS
- Qdrant
- OpenBao
- containerd + BuildKit
- Caddy
- vLLM primary, llama.cpp fallback, optional Ollama adapter
- OpenTelemetry Collector + Prometheus + Jaeger

## Go bootstrap

The Release A Go codebase uses the repository-root module `github.com/ControlCenterSoft/aidi_2.0` with Go 1.24.

A0-003 establishes compile-only bootstrap entry points for the four Foundation binaries:

- `cmd/aidi-control`
- `cmd/aidi-node-agent`
- `cmd/aidi-installer`
- `cmd/aidi-admin`

GitHub-hosted CI verifies module integrity and compiles each bootstrap entry point from a fresh checkout. Runtime behavior and reproducible release packaging are introduced by their later canonical Release A cards; these bootstrap entry points intentionally have no product behavior yet.

## Canonical project documents

- [Technical specification](docs/SPEC.md)
- [Implementation roadmap](docs/ROADMAP.md)
- [Acceptance Matrix / ПМИ](docs/ACCEPTANCE.md)
- [Controlled Cutover Plan](docs/CUTOVER.md)

The documents above are synchronized from the approved Google Drive baseline. Secrets and credentials must never be committed to this repository.

## Development isolation

GitHub development is a standalone AIDI 2.0 track.

- Do not consume tasks, state, artifacts, databases, queues, Forgejo repositories, VMs, runners, or work-in-progress from the current/legacy AIDI development environment.
- Do not push GitHub-development changes back into the current/legacy AIDI environment automatically.
- CI must use GitHub-hosted runners unless a separate GitHub-only runner pool is explicitly approved later.
- The canonical source for this track is this GitHub repository.
- Cross-contamination between the current environment and GitHub development is treated as a blocking defect.
