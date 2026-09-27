# GitHub-only autonomous execution

Status: **Release A development infrastructure**

This document defines the isolated GitHub execution path for the Release A Foundation development cycle.

## Isolation

The workflow in `.github/workflows/autonomous-core.yml` is allowed to use only:

- `ControlCenterSoft/aidi_2.0`;
- GitHub Actions;
- GitHub-hosted `ubuntu-latest` runners;
- the repository `GITHUB_TOKEN`;
- GitHub Copilot CLI authenticated through `GITHUB_TOKEN`.

It must not read from, write to, synchronize with, or execute against the current/local AIDI, Forgejo, local VMs, self-hosted runners, queues, databases, files, or runtime state.

## Scheduler ownership

The authoritative cross-cycle lease remains:

- branch: `automation-control`;
- file: `.automation/lock.json`;
- owner for this cycle: `core-:10`;
- lease duration: 45 minutes.

Every write-cycle reads the lock and blob SHA first. Acquisition and release use the GitHub Contents API with the expected SHA so a competing writer produces a conflict instead of overwriting another lease.

## Cycle

The hourly `:10` cycle performs one bounded Release A Foundation slice:

1. acquire the GitHub lease;
2. verify that no previous automation PR or queued/in-progress CI is active;
3. use the approved `docs/SPEC.md`, `docs/ROADMAP.md`, `docs/FOUNDATION.md` and current repository state to select one dependency-ready slice;
4. create the GitHub Issue;
5. create an isolated `automation/*` branch;
6. implement the slice with pinned GitHub Copilot CLI;
7. run deterministic checks locally on the GitHub-hosted runner;
8. create a draft PR;
9. explicitly dispatch the full `AIDI CI` workflow for the automation branch;
10. if CI fails, use the failed CI log for one bounded repair attempt and dispatch CI again;
11. mark the PR ready and merge only when CI is green and GitHub reports the PR cleanly mergeable;
12. explicitly validate fresh `main` with the full CI workflow;
13. fast-forward `development` to `main` without force and explicitly validate it with CI;
14. atomically release the lease and record completion evidence.

## Selector robustness

The Release A selector is allowed to emit brief explanatory text before its Markdown issue heading. The workflow normalizes selector output from the first top-level `# <issue title>` heading onward instead of assuming the H1 is the first byte of stdout.

A non-zero Copilot exit or output without a usable H1 is logged with bounded diagnostics and retried once. A second failure stops the cycle and releases the lease; it never creates an issue from ambiguous output.

Because `web/package-lock.json` is now pinned in the repository, autonomous pre-PR and repair checks use `npm ci` directly. They do not regenerate the lockfile.

## Why CI is explicitly dispatched

GitHub suppresses most new workflow runs caused by writes made with a repository `GITHUB_TOKEN`. The supported exception is `workflow_dispatch`. Therefore automation-created branch/PR writes do not rely on implicit push/PR events for qualification; the autonomous workflow explicitly dispatches `ci.yml` and waits for the result.

## Permissions

The autonomous workflow declares only the repository permissions it needs:

- `actions: write` — dispatch and inspect CI;
- `contents: write` — automation branches, merge/sync, and lease file;
- `issues: write` — create the bounded work item;
- `pull-requests: write` — create/transition/merge the PR;
- `copilot-requests: write` — use Copilot CLI through the built-in token.

The coding agent is not permitted to perform GitHub writes itself. GitHub writes are performed by deterministic workflow steps after validation.
