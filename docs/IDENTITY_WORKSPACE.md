# Release A — identity, workspace and project membership contracts

Status: **IN DEVELOPMENT**

This slice implements the first domain-only Identity/Workspace/Project membership and invitation lifecycle contracts required by the approved AIDI v2.0 baseline (SPEC §3.2, §4.1).

## Scope

`internal/identity` contains pure types, invariants and state-transition validation only:

- no HTTP/API;
- no persistence;
- no external identity provider (AD/LDAP/OIDC) integration.

Concrete Identity provider adapters, Public API surfaces and persistence remain later Release A/B work.

## Roles

`Role` enumerates the project roles from SPEC §3.2: `PROJECT_OWNER`, `PRODUCT_OWNER`, `CONTRIBUTOR`, `VIEWER`, `CLIENT`, `PROJECT_MANAGER`. Roles are project-scoped; this package defines no workspace-level role.

## Workspace/Project membership independence

SPEC §3.2 states: "Workspace — самостоятельная сущность. Workspace membership и Project membership независимы: участие в Workspace не означает автоматического доступа ко всем Projects."

`WorkspaceMembership` and `ProjectMembership` are independent records:

- a `WorkspaceMembership` carries no `ProjectID` and grants no project access;
- a `ProjectMembership` carries no `WorkspaceID` and does not require, assume or imply any `WorkspaceMembership` for the same user;
- validating one record never depends on the presence of the other.

## Project invitation lifecycle

`ProjectInvitation` is keyed by an exact login/email identifier (`Invitee`). Empty, whitespace-only and wildcard/enumeration-style targets (`*`, `?`) are rejected — the global user directory/autocomplete is never exposed through invitation targeting (AC-INV-001, AC-INV-002).

Lifecycle: `PENDING → ACCEPTED | DECLINED | EXPIRED | REVOKED`. All outcomes except `PENDING` are terminal.

`ValidateInvitationTransition` rejects every transition out of a terminal state, including re-accepting a `REVOKED` invitation, re-declining an `ACCEPTED` invitation, or reverting any terminal state back to `PENDING`. Only `PENDING → {ACCEPTED, DECLINED, EXPIRED, REVOKED}` and same-state no-ops are legal.

## Membership derivation

`DeriveProjectMembership` derives a `ProjectMembership` from a `ProjectInvitation` and rejects derivation unless the invitation is exactly `ACCEPTED` (AC-INV-003). A `PENDING`, `DECLINED`, `EXPIRED` or `REVOKED` invitation can never produce a `ProjectMembership`.

## Acceptance cross-reference

- SPEC §3.2 — Roles and membership; Workspace/Project membership independence; exact login/email invitation; ACCEPT/DECLINE/EXPIRE/REVOKE lifecycle.
- SPEC §4.1 — canonical hierarchy placement of Workspace/Project/Identity/User.
- AC-ID-001..006 (`docs/ACCEPTANCE.md`) — Identity & access acceptance criteria.
- AC-INV-001..004 (`docs/ACCEPTANCE.md`) — Project invitation acceptance criteria: exact login/email targeting, no directory enumeration, no `ProjectMembership` before `ACCEPT`, lifecycle availability.

## Evidence

The package tests cover:

- valid/invalid `Role` and ID (`WorkspaceID`/`ProjectID`/`UserID`) validation;
- `WorkspaceMembership`/`ProjectMembership` independence;
- every legal and illegal `ProjectInvitation` state transition;
- rejection of invitations not addressed to an exact login/email identifier;
- rejection of deriving `ProjectMembership` from a non-`ACCEPTED` invitation.

This package has no dependency on `net/http`, `database/sql`, NATS, Temporal or any identity provider SDK, and is isolated from the current/local AIDI runtime, Forgejo, local VM/runner infrastructure, local queues or local databases.
