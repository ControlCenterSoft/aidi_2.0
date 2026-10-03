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

Acquisition creates `.automation/lease.active` without a prior blob SHA. GitHub permits only one creator for the path, so a competing writer loses safely. Release verifies the owner and deletes the exact current blob. An expired lease is removed when its owner is no longer represented by queued/in-progress GitHub work. An open automation PR is durable recoverable state and does not keep an expired process lease alive. Completed-cycle evidence is append-only under `.automation/history/`.

## Cycle

The workflow has watchdog start opportunities at `:08/:38`. Healthy write cycles may explicitly hand off to the next cycle through `workflow_dispatch`; watchdog starts are only a fallback. The existence-based lease and previous-work checks ensure that overlapping starts do not create parallel conflicting write cycles.

The coding provider is controlled by `.automation/coding-provider.json`. The supported external provider is `chatgpt-connector`: Autonomous Core remains the canonical selector, lease/recovery authority and promotion controller, while it publishes the exact selected card to `automation-control:.automation/selected-card.json` for bounded implementation through the isolated ChatGPT GitHub connector. When `enabled=false`, product-write execution is intentionally paused. A published handoff is active only while its canonical Issue remains open; Core removes a handoff that points to a completed Issue before publishing another card or declaring the GitHub-only queue idle.

1. acquire the GitHub lease;
2. verify that no previous automation PR or queued/in-progress CI is active;
3. use the approved `docs/SPEC.md`, `docs/ROADMAP.md`, `docs/FOUNDATION.md` and current repository state to select one dependency-ready slice;
4. read the coding-provider configuration;
5. if the provider is disabled, release the lease without product writes;
6. for `chatgpt-connector`, publish the selected canonical card as a durable handoff and let the isolated connector implement exactly that bounded slice on a `chatgpt/*` branch; native `automation/*` execution remains reserved for an installed in-workflow adapter; if the connector leaves a branch without a PR, GitHub Doctor reconstructs the missing draft PR from the newest unlinked retry branch for that single canonical issue;
7. run deterministic checks locally on the GitHub-hosted runner;
8. create a draft PR;
9. explicitly dispatch the full `AIDI CI` workflow for the automation branch;
10. if CI fails while no coding provider is configured, stop and preserve the PR/CI evidence for later repair;
11. mark the PR ready and merge only when CI is green and GitHub reports the PR cleanly mergeable;
12. explicitly validate fresh `main` with the full CI workflow;
13. fast-forward `development` to `main` without force and explicitly validate it with CI;
14. atomically release the lease and record completion evidence.

## Provider and selector robustness

Release A selection remains deterministic and independent of the coding provider. A disabled provider is a normal paused state, not a recovery failure and not a reason to open the Doctor circuit.

Cards listed in `.automation/backlog/release-a/github-only-blocked.txt` are deferred external prerequisites. The isolated GitHub executor never selects or closes those cards, and they do not count as canonical completion. For GitHub-only dependency readiness, however, they are treated as execution-satisfied so an unavailable external prerequisite cannot deadlock downstream work that is fully implementable and testable inside this repository. The selector reports any such deferred dependency explicitly in `github_only_deferred_dependencies`.

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

The external `chatgpt-connector` may create or update only the bounded `chatgpt/*` implementation branch for the card published by Autonomous Core. Lease ownership, recovery decisions, PR promotion, exact-head CI dispatch, issue closure, merge, and `main`/`development` updates remain deterministic Core/Doctor responsibilities.


## Scheduled ChatGPT task role

Autonomous Core is the authoritative canonical selector, lease/recovery authority and promotion controller. The isolated ChatGPT GitHub connector is the configured external coding executor: it may implement only the card published by Core in `.automation/selected-card.json`, must use `chatgpt/*` branches, and must never bypass exact-head CI or the repository lease. A single hourly connector task performs bounded implementation/recovery; the hourly user report remains read-only.
