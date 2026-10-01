# Canonical state foundation

Source: approved AIDI v2.0.0 SPEC §4.2.

## Unified identifier model

Source: approved AIDI v2.0.0 SPEC §4.1.

`internal/canonical/identifier.go` defines the single canonical identifier/entity-kind contract shared by every domain object:

- `Kind` — a closed, typed enumeration restricted exactly to the SPEC §4.1 entity hierarchy: Installation, Identity/User, Workspace, Project, Specification, Requirement, Release, Feature, Task, Workflow, Attempt, ChangeSet, Verification, Evidence, Artifact, Decision, Approval, Risk, ChangeRequest, Problem, RecoveryCase, Policy, Resource, Event/Audit, OperationalKnowledge. `Kind.Validate()` rejects both empty and any value outside this set — no invented entity kinds are recognized.
- `ID` — the canonical identifier type shared by all domain objects that reference an entity. `ID.Validate()` rejects empty/whitespace-only identifiers.
- `ObjectRef{Kind, ID}` — the single canonical Kind/ID reference used across the codebase (e.g. `Event.Object`, `ChangeSet.Target`), replacing the previously duplicated, free-form `Kind string` object-reference shapes.

`Event` and `ChangeSet` validation now delegates to this shared `Kind`/`ID`/`ObjectRef` contract while preserving their existing JSON field names and public behavior. This is a pure domain/Go contract: it introduces no PostgreSQL, NATS, Temporal, HTTP, or runner/VM/queue infrastructure.

## Closed-set state contract

`internal/canonical/state.go` defines the single reusable "value is one of a
closed, named set" contract:

- `State` — a `string`-based canonical state value.
- `StateSet` — a closed, named set of allowed `State` values, built with
  `NewStateSet(states ...State)`.
- `State.Validate(allowed StateSet) error` rejects an empty/whitespace-only
  state and any state outside the caller-supplied `allowed` set, returning
  the single exported sentinel `ErrInvalidState` in both cases (following the
  `ErrInvalidKind`/`ErrInvalidID` naming convention).

This extracts the pattern that `TaskState` (`task.go`), `ChangeSetState`
(`changeset.go`) and `specification.RequirementStatus` each hand-roll
independently today — a `string`-based state type with a private
`validateKnownState` and a bespoke `ErrXInvariant`/`ErrInvalidXTransition`
pair, duplicated three times. `State`/`StateSet` is purely additive: no
existing `TaskState`, `ChangeSetState`, or `RequirementStatus` caller is
rewired to use it in this slice. It is the foundation later cards build on
for state/reason separation, a generic transition-graph validator, and an
invariant framework.

## Object revision model

`internal/canonical/revisioned.go` defines the generic "object revision
model" built purely on the `Revision`/`CheckExpectedRevision`/
`NextRevision` primitives (`revision.go`, A1-001):

- `Revisioned[T any]{Revision, Value}` pairs a mutable canonical object's
  current payload (`Value`) with its current monotonic `Revision`.
  `NewRevisioned(value)` wraps a newly created object at the initial
  `Revision` (`1`), mirroring the `Version >= 1` contract already enforced
  for `specification.Requirement`.
- `Revisioned[T].Update(expected Revision, mutate func(T) (T, error))
  (Revisioned[T], error)` is the single mutation entry point: it first
  checks `expected` against the receiver's current `Revision` via
  `CheckExpectedRevision` — a stale `expected` is rejected with
  `*RevisionConflictError` (`ErrRevisionConflict`) and `mutate` is never
  invoked, so last-write-wins is not reachable through this method. On a
  matching `expected`, `mutate` runs against the current `Value`; if it
  returns an error, that error is returned unchanged and the `Revision` is
  not advanced. Only once `mutate` succeeds does the `Revision` advance by
  exactly one via `NextRevision` (surfacing `ErrRevisionExhausted` instead
  of wrapping) and the updated `Value` take effect.

This mirrors the `Rule[T]`/`RuleSet[T]` generic-extraction convention
(`invariant.go`, A1-006's dependency base): it is purely additive and does
not rewire any existing entity (`Task`, `ChangeSet`,
`specification.Requirement`) onto `Revisioned[T]` in this slice. It gives
later cards (e.g. canonical schema/persistence, optimistic concurrency
enforcement at the storage boundary) a single reusable shape for "a
mutable canonical object with monotonic revision metadata" instead of each
entity hand-rolling its own revision field and conflict check.

## State/reason separation

`internal/canonical/reason.go` defines a reusable "why a state holds"
contract that builds purely on the `State`/`StateSet` contract above:

- `Reason` — a `string`-based value distinct from `State`, recording *why* a
  canonical state holds (e.g. a blocked/failure/rejection reason).
  `Reason.IsEmpty()` treats an empty or whitespace-only value as "no reason
  supplied".
- `ErrInvalidReason` — the sentinel returned when a required `Reason` is
  missing, following the `ErrInvalidKind`/`ErrInvalidID`/`ErrInvalidState`
  naming convention.
- `ValidateStateReason(state State, reason Reason, allowed StateSet,
  reasonRequired StateSet) error` first validates `state` against `allowed`
  (returning `ErrInvalidState` as before), then enforces the reason
  requirement: a `Reason` is required (non-empty, non-whitespace-only) for
  any `state` in the caller-supplied `reasonRequired` `StateSet`, returning
  `ErrInvalidReason` (wrapped with `%w` for `errors.Is`) when missing.

Explicit rule for this slice: a `Reason` is always optional *extra* context.
Supplying one for a state outside `reasonRequired` is allowed, not rejected
as strict surplus context — only a missing/empty `Reason` for a state that
*is* in `reasonRequired` is an error.

This is purely additive: `TaskState` (`task.go`), `ChangeSetState`
(`changeset.go`) and `specification.RequirementStatus` are not rewired onto
`Reason`/`ValidateStateReason` in this slice, mirroring the additive-only
rule from the `State`/`StateSet` card (A1-003). Rewiring existing callers is
reserved for a later card.

## State transition validator

`internal/canonical/transition.go` defines a single reusable, closed-graph
"(current `State`) -> (next `State`)" contract, built purely on `State`/
`StateSet` (`state.go`, A1-003):

- `TransitionGraph` — a closed set of allowed `(current State) -> (next
  State)` edges, expressed as a `map[State]StateSet` built with
  `NewTransitionGraph(edges map[State]StateSet)`. A `State` mapped to an
  empty `StateSet` (`NewStateSet()` with no arguments) is terminal: no
  outgoing edge is allowed, not even a same-state no-op, unless the graph
  explicitly declares that same-state edge. A `State` absent from the graph
  entirely is outside the declared graph, and every edge into or out of it
  is rejected.
- `ValidateStateTransition(current, next State, graph TransitionGraph)
  error` rejects any edge not present in `graph`, returning the single
  exported sentinel `ErrInvalidStateTransition` (wrapped with `%w` for
  `errors.Is`), naming both `current` and `next`, following the
  `ErrInvalidKind`/`ErrInvalidID`/`ErrInvalidState`/`ErrInvalidReason`
  naming convention. Named `ValidateStateTransition` (mirroring
  `ValidateStateReason` from `reason.go`) rather than `ValidateTransition`
  to avoid colliding with the existing, unrelated
  `ValidateTransition(current, next ChangeSetState)` in `changeset.go`.

This extracts the shared shape that `TaskState.ValidateTransition`
(`task.go`), `ChangeSetState` (`changeset.go`),
`specification.RequirementStatus`, `identity.InvitationState`, and
`identity.LocalAdminAccountState` each hand-roll independently today as an
identical `switch current { case ... }` edge check plus a bespoke
`ErrInvalidXTransition` sentinel and terminal-state handling. This is purely
additive: no existing caller (`TaskState`, `ChangeSetState`,
`specification.RequirementStatus`, `identity.InvitationState`,
`identity.LocalAdminAccountState`, or `orchestration` command transitions)
is rewired onto `TransitionGraph`/`ValidateStateTransition` in this slice.
Rewiring existing callers, and the domain invariant framework that depends
on this primitive, are reserved for later cards.

## Event envelope

Every persisted significant transition must be representable by a versioned canonical Event containing:

- Event ID
- Event type
- Object kind and ID
- Object revision
- UTC timestamp
- Actor
- Correlation ID
- Optional causation ID for non-root chains
- Schema version

An invalid envelope is rejected before persistence or publication.

## Optimistic concurrency

Critical state uses exact expected-revision checks. Last-write-wins is not an acceptable conflict policy.

A stale caller receives a typed revision-conflict error containing both expected and actual revisions. The caller must reconcile against current canonical state rather than silently overwrite it.

`Revisioned[T].Update` (see "Object revision model" above) is the single
generic entry point enforcing this rule: a mismatched expected revision is
rejected before the mutation runs, and the revision only advances, by
exactly one, once the mutation itself succeeds.

## Boundary

This slice defines domain contracts only. PostgreSQL transactions, Transactional Outbox, Inbox/dedup, event publication, NATS JetStream, and durable workflow integration are separate Release A slices.

The GitHub development track remains isolated from the current/legacy AIDI runtime and state.
