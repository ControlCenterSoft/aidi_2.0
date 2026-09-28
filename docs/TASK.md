# Release A — canonical Task lifecycle contract

Status: **IN DEVELOPMENT**

This slice implements the SPEC §2.2/§8.1 rule that **Task and Attempt are
separate entities**: a Task tracks the lifecycle of a unit of work
independently of any particular Attempt currently executing it
(ACCEPTANCE.md AC-EXEC-001), and `RUNNING` is only reachable while a real
executor/attempt lease is bound.

## Lifecycle

```
PENDING ──▶ READY ──▶ RUNNING ──▶ DONE
              ▲           │
              │           ├──▶ BLOCKED ──▶ (back to READY)
              │           │
              └───────────┴──▶ FAILED
```

- `PENDING` — not yet ready to execute (e.g. waiting on dependencies).
- `READY` — eligible for an Attempt to be bound and started.
- `RUNNING` — bound to a real `orchestration.AttemptBinding` (exact source
  SHA, isolated workspace, executor identity); cannot be entered without one.
- `DONE` — terminal; requires at least one `EvidenceRef`.
- `BLOCKED` — needs intervention/reconciliation; can return to `READY` to
  retry with a new Attempt.
- `FAILED` — terminal.

`DONE` and `FAILED` are terminal: no transition out of them is permitted,
including to the same state. Any transition not listed above is rejected by
`ValidateTaskTransition`.

## Task ≠ Attempt

A `Task` carries an optional *current* `orchestration.AttemptBinding`
reference — it does not embed or own Attempt lifecycle state. Replacing the
bound Attempt (e.g. binding a fresh Attempt after `BLOCKED -> READY`) does
not change the Task's `TaskID` or its transition rules: a Task can outlive
many Attempts (SPEC §8.1, ACCEPTANCE.md AC-EXEC-001).

## Evidence-before-DONE rule

Reaching `DONE` without at least one `EvidenceRef` (e.g. a test run or
deterministic check result) is rejected by `Task.Validate`. An LLM/agent
"done" claim alone never satisfies this invariant — there is no lifecycle
path that reaches `DONE` while `Evidence` is empty (ACCEPTANCE.md
AC-EXEC-004).

## Evidence

The package tests (`internal/canonical/task_test.go`) cover:

- rejection of a `Task` missing `TaskID`, with an unknown state, `RUNNING`
  without (or with an invalid) attempt binding, or `DONE` without evidence;
- the full valid lifecycle `PENDING -> READY -> RUNNING -> DONE`;
- retry: `RUNNING -> BLOCKED -> READY -> RUNNING` binding a new Attempt
  without changing `TaskID`;
- `ValidateTaskTransition` allowing only the documented edges and rejecting
  all others, including same-state churn and any transition out of
  `DONE`/`FAILED`;
- rejection of invalid `EvidenceRef` entries.

## Scope

This package is a pure domain contract with no persistence, workflow engine,
message bus or HTTP wiring. It has no dependency on Forgejo, local VM/runner
infrastructure, local queues or local databases. Persistence and
workflow-engine wiring for `Task` are explicitly out of scope for this slice
and are left to a later Release A slice.
