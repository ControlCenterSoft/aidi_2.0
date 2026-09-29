# Release A — specification engine domain contracts

Status: **IN DEVELOPMENT**

This slice implements the first domain-only contracts for the canonical
Requirement and Decision entities required by the approved AIDI v2.0
baseline (SPEC §4.1, §5.2–§5.3).

## Canonical hierarchy

The `internal/specification` package models the `Specification → Requirement`
slice of the canonical hierarchy (SPEC §4.1:
`Installation → Identity/User → Workspace → Project → Specification →
Requirement → Release → ...`). `Decision` is a first-class entity linked to
one or more `Requirement`s (SPEC §4.1).

## Requirement classification model

`RequirementStatus` implements the structured model from SPEC §5.2:

- `KNOWN` — confirmed and not in dispute.
- `UNKNOWN` — not yet determined.
- `ASSUMED` — a working assumption pending confirmation.
- `CONFLICT` — contradictory information; must be resolved before approval.
- `RISK` — a known risk requiring tracking/mitigation.
- `DECISION_REQUIRED` — an explicit decision is needed to proceed.
- `DERIVED` — an engineering requirement derived from other requirements.

## Requirement quality invariants (SPEC §5.3)

A `Requirement` must be:

- **versionable** — `Version` starts at 1 and every update via
  `ValidateUpdate` must strictly increase the version; non-monotonic updates
  are rejected with `ErrNonMonotonicVersion`.
- **traceable** — `Source` (chat/message/document reference) must not be
  empty.
- **verifiable** — at least one non-empty `AcceptanceCriteria` entry must be
  linked.

Violations of any of the above are reported via `ErrInvalidRequirement`.

## Decision

A `Decision` links one or more resolved `Requirement`s (`RequirementIDs`) to
an `Outcome`, and carries a `Severity` of `CRITICAL` or `NORMAL`. `Validate`
rejects a `Decision` missing requirement linkage, outcome, or a recognized
severity (`ErrInvalidDecision`).

## Approval readiness (SPEC §5.3, AC-SPEC-004)

`ApprovalReady(requirements []Requirement) (bool, []RequirementID)`
implements: *"Critical unresolved conflict блокирует READY_FOR_APPROVAL"*.
It returns `false` together with the blocking `RequirementID`s whenever any
`Requirement` carries `CONFLICT` status, and `true` with a `nil` slice
otherwise.

This directly supports **AC-SPEC-003** (significant decisions become
structured Requirements/Decisions) and **AC-SPEC-004** (a critical conflict
blocks Approval), as listed in `docs/ACCEPTANCE.md`.

## Approval bound to an exact Specification version (SPEC §5.5, AC-SPEC-004/AC-SPEC-009)

`Approval` is the first-class entity from SPEC §4.1 recording the decision
that closes out a Specification review. It always attaches to an exact
`SpecificationVersion` (`uint64`, mirroring the `Requirement.Version`
pattern) rather than to the Specification in general, and carries a
`DecidedAt` (`time.Time`) timestamp recording when the decision was made:

- `Approval.Validate()` enforces SPEC §5.5's required fields: a non-empty
  `ID`, a non-empty target `SpecificationID`, a positive
  `SpecificationVersion`, a known `Decision` (`APPROVED`/`REJECTED`), and a
  non-empty `Approver`. Violations are reported via `ErrInvalidApproval`.
  `DecidedAt` records the decision timestamp.
- `NewApproval(a Approval, requirements []Requirement) (Approval, error)`
  composes with the existing `ApprovalReady` gate (SPEC §5.3, AC-SPEC-004)
  as a precondition instead of duplicating the conflict check: it returns
  `ErrApprovalNotReady` when any `Requirement` for the target Specification
  still carries `CONFLICT`, and otherwise delegates to `Validate()`.
- `ApprovalCurrentForVersion(approval Approval, currentSpecificationVersion uint64) error`
  implements SPEC §5.5 / AC-SPEC-009 — *"Approval всегда относится к exact
  Specification version"*: an `Approval` bound to version `N` does **not**
  silently authorize version `N+1`. It returns `ErrApprovalVersionMismatch`
  whenever `approval.SpecificationVersion` differs from the Specification's
  current version, so any further change beyond the approved version must
  go through a Change Request rather than being treated as still approved.

This directly covers **AC-SPEC-004** (critical conflict blocks Approval,
reused rather than reimplemented) and **AC-SPEC-009** (Approval bound to an
exact Specification version), as listed in `docs/ACCEPTANCE.md`. Change
Request lifecycle, persistence, HTTP/API, and workflow-engine wiring remain
out of scope for this bounded slice.

## Status transitions

`ValidateStatusTransition(current, next RequirementStatus) error` is the
Requirement-status analogue of `orchestration.ValidateCommandTransition`:

- Identity transitions (`current == next`) are always legal.
- `UNKNOWN` may progress to `ASSUMED`, `CONFLICT`, `RISK`,
  `DECISION_REQUIRED`, or directly to `KNOWN`.
- `ASSUMED` may resolve to `KNOWN`, or surface a problem as `CONFLICT`,
  `RISK`, or `DECISION_REQUIRED`.
- `KNOWN` may regress only into a problem classification (`CONFLICT`,
  `RISK`, `DECISION_REQUIRED`) if discovery reopens the requirement.
- `CONFLICT`, `RISK`, and `DECISION_REQUIRED` may resolve into `KNOWN` or
  `DERIVED`, or move between one another, but cannot regress to `UNKNOWN` or
  `ASSUMED`.
- `DERIVED` cannot regress to `UNKNOWN` (and, in this bounded slice, has no
  other outbound transition).

Illegal transitions return `ErrInvalidStatusTransition`.

## Evidence

The package tests (`internal/specification/specification_test.go`) cover:

- valid/invalid `Requirement` (id, version, status, source traceability,
  acceptance-criteria linkage);
- monotonic version enforcement via `ValidateUpdate`, including id-mismatch
  and version-regression rejection;
- valid/invalid `Decision` (requirement linkage, outcome, severity);
- `ApprovalReady` returning `false` with correct blocking IDs when any
  unresolved `CONFLICT` exists, and `true` otherwise (including empty input);
- every legal and illegal `RequirementStatus` transition.

`internal/specification/approval_test.go` covers:

- valid/invalid `Approval` (id, specification id, version, decision,
  approver);
- `NewApproval` refusing construction while a `CONFLICT` requirement is
  outstanding (`ErrApprovalNotReady`), and permitting it once resolved or
  when there are no requirements at all;
- `ApprovalCurrentForVersion` accepting a matching version and rejecting a
  mismatched one (`ErrApprovalVersionMismatch`), including the explicit
  N vs. N+1 case from AC-SPEC-009.

This package is pure domain logic: no HTTP/API, no persistence
(`database/sql`), no durable-workflow (Temporal) or event-bus (NATS)
dependency, and no LLM/discovery SDK dependency. It has no dependency on
Forgejo, local VM/runner infrastructure, local queues, or local databases.
