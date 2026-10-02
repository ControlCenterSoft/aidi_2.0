# GitHub-only autonomous execution

Status: **Release A development infrastructure**

This document defines the isolated GitHub execution path for the Release A Foundation development cycle.

## Isolation

The workflow in `.github/workflows/autonomous-core.yml` is allowed to use only:

- `ControlCenterSoft/aidi_2.0`;
- GitHub Actions;
- GitHub-hosted `ubuntu-latest` runners;
- the repository `GITHUB_TOKEN`.


It must not read from, write to, synchronize with, or execute against the current/local AIDI, Forgejo, local VMs, self-hosted runners, queues, databases, files, or runtime state.

## Scheduler ownership

The authoritative cross-cycle mutex is now existence-based:

- branch: `automation-control`;
- active lease file: `.automation/lease.active`;
- lease duration: 45 minutes;
- legacy `.automation/lock.json` is status/history only and is not used for mutual exclusion.

Acquisition creates `.automation/lease.active` without a prior blob SHA. GitHub permits only one creator for the path, so a competing writer loses safely. Release verifies the owner and deletes the exact current blob. An expired lease is removed only when no matching open automation PR or queued/in-progress GitHub run remains. Completed-cycle evidence is append-only under `.automation/history/`.

## Cycle

The workflow has watchdog start opportunities at `:08/:38`. Healthy write cycles may explicitly hand off to the next cycle through `workflow_dispatch`; watchdog starts are only a fallback. The existence-based lease and previous-work checks ensure that overlapping starts do not create parallel conflicting write cycles.

The coding provider is controlled by `.automation/coding-provider.json`. When `enabled=false`, product-write execution is intentionally paused: the Core may validate/select the next canonical card, but it must not create an implementation branch, edit source, or open a product PR.

1. acquire the GitHub lease;
2. verify that no previous automation PR or queued/in-progress CI is active;
3. use the approved `docs/SPEC.md`, `docs/ROADMAP.md`, `docs/FOUNDATION.md` and current repository state to select one dependency-ready slice;
4. read the coding-provider configuration;
5. if the provider is disabled, release the lease without product writes;
6. when an approved executor adapter is configured, create an isolated `automation/*` branch and implement exactly one bounded slice;
7. run deterministic checks locally on the GitHub-hosted runner;
8. create a draft PR;
9. explicitly dispatch the full `AIDI CI` workflow for the automation branch;
10. if CI fails, use the failed CI log for one bounded repair attempt and dispatch CI again;
11. mark the PR ready and merge only when CI is green and GitHub reports the PR cleanly mergeable;
12. explicitly validate fresh `main` with the full CI workflow;
13. fast-forward `development` to `main` without force and explicitly validate it with CI;
14. atomically release the lease and record completion evidence.

## Provider and selector robustness

Release A selection remains deterministic and independent of the coding provider. A disabled provider is a normal paused state, not a recovery failure and not a reason to open the Doctor circuit.

No source-repair fallback is performed when the provider is disabled. Failed exact-head CI or blocking review findings remain durable recovery state until an approved executor is configured.

Because `web/package-lock.json` is pinned in the repository, autonomous pre-PR checks use `npm ci` directly. They do not regenerate the lockfile.

## Why CI is explicitly dispatched

GitHub suppresses most new workflow runs caused by writes made with a repository `GITHUB_TOKEN`. The supported exception is `workflow_dispatch`. Therefore automation-created branch/PR writes do not rely on implicit push/PR events for qualification; the autonomous workflow explicitly dispatches `ci.yml` and waits for the result.

## Permissions

The autonomous workflow declares only the repository permissions it needs:

- `actions: write` — dispatch and inspect CI;
- `contents: write` — automation branches, merge/sync, and lease file;
- `issues: write` — create the bounded work item;
- `pull-requests: write` — create/transition/merge the PR;

Any future coding executor is not permitted to perform GitHub writes itself. GitHub writes are performed by deterministic workflow steps after validation.


## Scheduled ChatGPT task role

ChatGPT scheduled tasks are the supervisory/control plane for this GitHub-only contour: planning, observation, review, recovery analysis, architecture/evidence audits, and the hourly user report. The authoritative product-write execution path is the GitHub-hosted `AIDI Autonomous Core` workflow. This avoids making connector-side write approvals a dependency of autonomous development.
