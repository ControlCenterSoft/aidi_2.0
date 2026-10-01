# Canonical state foundation

Source: approved AIDI v2.0.0 SPEC §4.2.

## Unified identifier model

Source: approved AIDI v2.0.0 SPEC §4.1.

`internal/canonical/identifier.go` defines the single canonical identifier/entity-kind contract shared by every domain object:

- `Kind` — a closed, typed enumeration restricted exactly to the SPEC §4.1 entity hierarchy: Installation, Identity/User, Workspace, Project, Specification, Requirement, Release, Feature, Task, Workflow, Attempt, ChangeSet, Verification, Evidence, Artifact, Decision, Approval, Risk, ChangeRequest, Problem, RecoveryCase, Policy, Resource, Event/Audit, OperationalKnowledge. `Kind.Validate()` rejects both empty and any value outside this set — no invented entity kinds are recognized.
- `ID` — the canonical identifier type shared by all domain objects that reference an entity. `ID.Validate()` rejects empty/whitespace-only identifiers.
- `ObjectRef{Kind, ID}` — the single canonical Kind/ID reference used across the codebase (e.g. `Event.Object`, `ChangeSet.Target`), replacing the previously duplicated, free-form `Kind string` object-reference shapes.

`Event` and `ChangeSet` validation now delegates to this shared `Kind`/`ID`/`ObjectRef` contract while preserving their existing JSON field names and public behavior. This is a pure domain/Go contract: it introduces no PostgreSQL, NATS, Temporal, HTTP, or runner/VM/queue infrastructure.

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

## Boundary

This slice defines domain contracts only. PostgreSQL transactions, Transactional Outbox, Inbox/dedup, event publication, NATS JetStream, and durable workflow integration are separate Release A slices.

The GitHub development track remains isolated from the current/legacy AIDI runtime and state.
