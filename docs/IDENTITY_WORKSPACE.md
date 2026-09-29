# Release A — identity/workspace/project membership contracts

Status: **IN DEVELOPMENT**

This slice implements the first transport-neutral domain contract for
Workspace/Project membership and the Project Invitation lifecycle required
by the approved AIDI v2.0 baseline (SPEC §3, §4.1).

## Scope

`internal/identity` contains pure domain types, invariants and
state-transition validation only:

- `WorkspaceID`, `ProjectID`, `UserID`, `Role` value types with validation.
- `WorkspaceMembership` and `ProjectMembership` as independent records.
- `ProjectInvitation` keyed by an exact login/email identifier with a
  deterministic lifecycle.
- `ValidateInvitationTransition`, analogous to
  `orchestration.ValidateCommandTransition`, rejecting invalid invitation
  state transitions.
- `NewProjectMembershipFromInvitation`, which fails closed unless the source
  invitation is `ACCEPTED`.
- `ServiceIdentity` and its `ServiceIdentityID`/`ServiceKind`/`Scope`/
  `ServiceCredential` building blocks (SPEC §3.3) — see "Service Identities"
  below.

There is no HTTP/API surface, no persistence and no external identity
provider integration in this package. Concrete adapters (Local Identity, AD,
Samba AD, LDAP/LDAPS, FreeIPA, OIDC, and canonical DB persistence) remain
later Release A/B work.

## Roles

Project roles (SPEC §3.2): `PROJECT_OWNER`, `PRODUCT_OWNER`, `CONTRIBUTOR`,
`VIEWER`, `CLIENT`, `PROJECT_MANAGER`. The global `SUPER_ADMIN` role is a
system-level role, not a project role, and is intentionally out of scope for
this package.

## Workspace/Project membership independence

Per SPEC §3.2 ("Workspace membership и Project membership независимы"),
`WorkspaceMembership` and `ProjectMembership` are independent records:

- A `WorkspaceMembership` never implies access to any Project.
- A `ProjectMembership` is valid entirely on its own and does not require, or
  assume the existence of, a corresponding `WorkspaceMembership` for the same
  user.

## Project Invitation lifecycle

`ProjectInvitation` is addressed to an exact login/email `Invitee`. Wildcard,
glob-like or whitespace-padded identifiers are rejected — invitations cannot
be used to enumerate or broadcast to the global user directory
(AC-INV-001, AC-INV-002).

Lifecycle:

```
PENDING → ACCEPTED | DECLINED | EXPIRED | REVOKED
```

`PENDING` is the only non-terminal state. Every other state is terminal: an
`ACCEPTED`, `DECLINED`, `EXPIRED` or `REVOKED` invitation cannot transition
again (for example, a `REVOKED` invitation cannot later be re-accepted).
`ValidateInvitationTransition` enforces this and rejects any transition that
is not `PENDING → {ACCEPTED, DECLINED, EXPIRED, REVOKED}`.

`ProjectMembership` cannot exist or be derived from an invitation that is
not `ACCEPTED` (AC-INV-003). `NewProjectMembershipFromInvitation` enforces
this by construction: it returns `ErrMembershipRequiresAcceptedInvitation`
for any non-`ACCEPTED` invitation state, and `ErrMembershipInviteeMismatch`
if the derived user does not match the invitation's exact invitee.

## Service Identities

Per SPEC §3.3 ("Node Agent, CRM, providers и automation используют отдельные
service/workload identities, а не пользовательские аккаунты"), `internal/
identity` also defines a `ServiceIdentity` contract for non-user,
workload/automation principals — structurally separate from
`UserID`/`Role`/`ProjectMembership` so a service identity can never be
constructed as, or satisfy validation for, a user account or project
membership:

- `ServiceIdentityID` — a distinct identifier type from `UserID`, validated
  the same way (non-empty, non-blank).
- `ServiceKind` — one of `NODE_AGENT`, `CRM`, `PROVIDER`, `AUTOMATION`.
- `Scope`/`Permission` — an explicit, enumerable least-privilege grant.
  Mirroring the `ProjectInvitation` exact-invitee anti-wildcard pattern, a
  `Scope`'s `Permission` must be a single, non-empty, non-whitespace-padded,
  non-wildcard/glob-like string — there is no blanket/wildcard scope.
- `ServiceCredential` — carries `IssuedAt`/`ExpiresAt`. SPEC §3.3 requires
  workload credentials to be short-lived where possible; `Validate()`
  encodes the checkable minimum invariant: both timestamps must be set and
  `ExpiresAt` must be strictly after `IssuedAt` (a zero, unbounded, or
  already-expired-at-issuance credential is rejected).
- `ServiceIdentity` — the aggregate (`ID`, `Kind`, `Scopes`, `Credential`).
  `Validate()` enforces: a valid `ID`, a known `Kind`, at least one explicit
  scope, no duplicate scopes, and a valid, strictly bounded credential.

Concrete credential issuance/rotation, persistence, Node Agent/CRM adapter
wiring and any transport binding are out of scope for this slice and remain
later Release A/B work.

## Local admin bootstrap and break-glass identity

Per SPEC §3.1 ("После clean install создаётся local admin/admin. Система
остаётся UNINITIALIZED до обязательной смены пароля при первом входе" /
"Local administrative identity сохраняется как break-glass даже при
использовании внешнего IdP"), `internal/identity` also defines the mandatory
local administrative identity contract, matching AC-INST-002, AC-ID-001 and
AC-ID-004:

- `LocalAdminAccountState` — `UNINITIALIZED` (the post-install bootstrap
  "admin/admin" state) and `ACTIVE` (reached only after a proven
  credential-change event). `ValidateLocalAdminTransition` permits only the
  one-way transition `UNINITIALIZED → ACTIVE`; `ACTIVE → UNINITIALIZED` and
  any self-loop are rejected — the transition is non-reversible through this
  contract.
- `LocalAdminAccount` — `UserID`, `State`, `CredentialRevision` (a monotonic
  marker for "password changed"). A freshly constructed (zero-value)
  account is `UNINITIALIZED` via `EffectiveState`. `Validate()` rejects an
  `ACTIVE` account with `CredentialRevision < 1`, i.e. an account that
  claims to be `ACTIVE` while still carrying no recorded credential change
  (still "admin/admin"-equivalent).
- `RequireInitialized` returns `ErrLocalAdminNotInitialized` when normal
  operation is attempted while `State == UNINITIALIZED`, encoding "штатная
  эксплуатация до смены password невозможна" (AC-INST-002).
- `BreakGlassEligible(account, externalIdPHealthy)` returns true for a
  valid, `ACTIVE` local admin account regardless of `externalIdPHealthy` —
  Local Identity validity never depends on external IdP state (AC-ID-001,
  AC-ID-004).

No password hashing/crypto, persistence, HTTP/API surface or external IdP
wiring is introduced by this contract; issuance of the actual bootstrap
credential, its storage and the first-login password-change flow remain
later Release A/B work.

## Evidence

The package tests cover:

- valid/invalid `WorkspaceID`/`ProjectID`/`UserID`/`Role` validation;
- workspace/project membership independence (a project membership is valid
  standalone, without any workspace membership record);
- every legal and illegal `ProjectInvitation` state transition;
- rejection of invitations not addressed to an exact login/email
  (empty, whitespace-padded, wildcard/glob, or multi-token identifiers);
- rejection of deriving `ProjectMembership` from a non-`ACCEPTED`
  invitation;
- valid/invalid `ServiceIdentityID`/`ServiceKind`/`Scope` validation,
  including rejection of wildcard/glob-like and whitespace-padded scopes;
- `ServiceCredential` lifetime validation, including zero timestamps,
  equal `IssuedAt`/`ExpiresAt`, and `ExpiresAt` before `IssuedAt`;
- `ServiceIdentity.Validate()` for a well-formed identity and every
  rejection case (invalid id, unknown kind, zero scopes, duplicate scopes,
  wildcard scope, invalid credential lifetime).
- every legal/illegal `LocalAdminAccountState` transition, in particular
  that only `UNINITIALIZED → ACTIVE` is legal and that `ACTIVE →
  UNINITIALIZED` and both self-loops are rejected (AC-INST-002);
- `LocalAdminAccount.Validate()` for a freshly constructed (zero-value,
  `UNINITIALIZED`) account, a valid `ACTIVE` account, and rejection of an
  `ACTIVE` account with no recorded credential change or a negative
  `CredentialRevision`;
- `RequireInitialized` rejecting an `UNINITIALIZED` account and accepting
  a properly initialized `ACTIVE` account (AC-INST-002);
- `BreakGlassEligible` returning true for a valid `ACTIVE` local admin
  account for both `externalIdPHealthy = true` and `false`, and returning
  false for an `UNINITIALIZED` or otherwise invalid account (AC-ID-001,
  AC-ID-004).

This package has no dependency on `net/http`, `database/sql`, NATS,
Temporal, or any external identity provider — pure domain logic only,
consistent with `docs/FOUNDATION.md`'s isolation requirement.
