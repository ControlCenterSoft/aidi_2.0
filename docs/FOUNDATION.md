# Release A — Foundation operational baseline

Status: **IN DEVELOPMENT**

This document defines the first executable slice of the approved AIDI v2.0.0 baseline.

## Runtime contract

The Go core exposes:

- `GET /health` — process liveness; independent of downstream readiness.
- `GET /ready` — readiness for receiving workload; returns HTTP 503 while the process is not ready.
- Both endpoints return `Cache-Control: no-store`.
- Build metadata includes version, commit and build time.

## Configuration

Runtime configuration is environment-based and fail-fast:

| Variable | Default | Contract |
| --- | --- | --- |
| `AIDI_HTTP_ADDR` | `:8080` | HTTP listen address |
| `AIDI_LOG_LEVEL` | `info` | `debug`, `info`, `warn`, or `error` |
| `AIDI_SHUTDOWN_TIMEOUT` | `10s` | positive Go duration |

Secrets are intentionally absent from the baseline configuration.

## Lifecycle

1. Validate configuration.
2. Initialize structured JSON logging.
3. Initialize build metadata.
4. Start HTTP server.
5. Mark readiness.
6. On SIGINT/SIGTERM, remove readiness.
7. Perform bounded graceful shutdown.

## Worker and recovery contracts

The Python worker package now defines infrastructure-agnostic Release A primitives:

- worker request/result contracts with evidence required for successful results;
- capability-based model deployments and deterministic selection;
- local-first routing: local qualified deployments are preferred, and a local-required request can never fall through to cloud;
- normalized failure classes and failure kinds;
- retry guards that prevent unchanged deterministic failures from blind retry;
- lease/ownership uncertainty that forces reconciliation before any retry/side effect;
- bounded transient retry budget and escalation when exhausted;
- recovery evidence rules that prevent workflow resume before recovery is proven.

These contracts contain no model runtime, GPU, VM, Forgejo, local queue, or current-AIDI dependency. Concrete adapters remain later Release A/D/G work.

## Qualification

The GitHub-only CI gate validates:

- Go formatting.
- `go vet`.
- race-enabled Go unit tests.
- Go binary build with injected build metadata.
- Python source compilation and worker smoke test.
- Python worker/recovery contract unit tests.
- React/TypeScript/Vite production build.

This GitHub development track remains isolated from all current/legacy AIDI execution infrastructure.
