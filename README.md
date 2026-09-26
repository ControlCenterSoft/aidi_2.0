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

## Canonical project documents

- [Technical specification](docs/SPEC.md)
- [Implementation roadmap](docs/ROADMAP.md)
- [Acceptance Matrix / ПМИ](docs/ACCEPTANCE.md)
- [Controlled Cutover Plan](docs/CUTOVER.md)

The documents above are synchronized from the approved Google Drive baseline. Secrets and credentials must never be committed to this repository.
