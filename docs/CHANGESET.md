# Release A — canonical ChangeSet contract

Status: **IN DEVELOPMENT**

This slice implements the SPEC §8.1/§8.2 rule that generated code does not
modify the canonical branch directly: results are packaged as a `ChangeSet`
and must pass a validated lifecycle before merge (ACCEPTANCE.md AC-EXEC-003,
AC-EXEC-006).

## Lifecycle

```
DRAFT ──▶ SUBMITTED ──▶ VALIDATING ──▶ VALIDATED ──▶ MERGED
  │                                       │
  └── (no-op, editing before submission)  └──▶ REJECTED
```

- `DRAFT` — being assembled from an Attempt's output; may stay `DRAFT` while
  edited (`DRAFT -> DRAFT` is the only explicit same-state no-op).
- `SUBMITTED` — handed to the verification/integration queue.
- `VALIDATING` — deterministic checks/tests/review in progress.
- `VALIDATED` — passed verification; requires at least one evidence
  reference.
- `REJECTED` — terminal; verification failed.
- `MERGED` — terminal; integrated into the canonical branch. Requires
  evidence and that the ChangeSet's base revision still matches the current
  canonical revision.

`REJECTED` and `MERGED` are terminal: no transition out of them is permitted,
including to the same state. Any transition not listed above is rejected by
`ValidateTransition`.

## Binding to Attempt isolation

Each `ChangeSet` carries the `orchestration.AttemptBinding` (Attempt ID,
exact source SHA, isolated workspace, executor identity) that produced it,
plus a target canonical object reference and the base `canonical.Revision`
the change was computed against.

## Evidence-before-merge rule

Reaching `VALIDATED` or `MERGED` without at least one `EvidenceRef` (e.g. a
test run or deterministic check result) is rejected by `ChangeSet.Validate`.
This rule is non-waivable in this domain contract: there is no lifecycle path
that reaches `VALIDATED`/`MERGED` while `Evidence` is empty.

## Stale ChangeSet handling

`ChangeSet.Merge(currentRevision)` reuses `canonical.CheckExpectedRevision` to
compare the ChangeSet's `BaseRevision` against the current canonical
revision. A mismatch returns `ErrRevisionConflict` instead of silently
merging; the stale ChangeSet must be reconciled (rebase/regenerate) before it
can merge again, consistent with SPEC §8.1's stale-source handling.

## Evidence

The package tests (`internal/canonical/changeset_test.go`) cover:

- rejection of a `ChangeSet` missing AttemptBinding fields, target object
  ref, changeset id, base revision, or with an unknown state;
- the full valid lifecycle `DRAFT -> SUBMITTED -> VALIDATING -> VALIDATED ->
  MERGED`;
- `ValidateTransition` allowing only the documented edges and rejecting all
  others, including same-state churn outside the `DRAFT -> DRAFT` no-op and
  any transition out of `REJECTED`/`MERGED`;
- rejection of `VALIDATED`/`MERGED` without evidence;
- rejection of `MERGED` when the base revision no longer matches the current
  canonical revision.

This package is a pure domain contract with no persistence, workflow engine,
message bus or HTTP wiring, and has no dependency on Forgejo, local VM/runner
infrastructure, local queues or local databases.
