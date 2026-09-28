# GitHub-only automation lease protocol

This branch uses an existence-based lease file as the single writer mutex for scheduled AIDI development tasks.

## Canonical mutex

- Active lease path: `.automation/lease.active`
- Legacy `.automation/lock.json` is status/history only and MUST NOT be used for mutual exclusion.

## Acquire

1. Inspect open PRs, queued/in-progress CI, dependencies, and area conflicts.
2. Read `.automation/lease.active`.
3. If it exists and is not stale, do not write and exit.
4. If it does not exist, acquire the lease by creating it with GitHub Contents API `create_file`.
5. The lease JSON must include at least:
   - `owner`
   - `started_at`
   - `lease_until`
   - `task_role`
   - optional `issue` / `slice`
6. If `create_file` fails because the path already exists or another writer won the race, treat that as normal contention and exit without development changes.

## Release

1. Re-read `.automation/lease.active`.
2. Verify the owner matches the current cycle.
3. Delete it using `delete_file` with the current blob SHA.
4. Never delete a lease owned by another active cycle.

## Stale recovery

Only Recovery Doctor may remove an expired lease owned by another cycle, and only after verifying there is no matching open PR, queued/in-progress CI, or other evidence that the work is still active.

## Prohibited

- Do not use `update_file` or `update_ref` as the mutex mechanism.
- Never use force updates.
- Never start product-source writes before lease acquisition succeeds.
- Never introduce dependencies on local AIDI, Forgejo, local VM/runner/queue/DB/state.

Completed-cycle evidence should be append-only where practical (for example `.automation/history/*.json`) rather than mutating the mutex file.
