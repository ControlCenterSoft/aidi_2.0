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
- `LocalAdminAccount` and its `LocalAdminAccountState`/
  `ValidateLocalAdminTransition`/`RequireInitialized`/`BreakGlassEligible`
  contract for the mandatory local admin bootstrap and break-glass identity
  (SPEC §3.1) — see "Local Admin Bootstrap and Break-Glass" below.
- `EnrollmentToken` and its `EnrollmentTokenID`/`EnrollmentTokenState`/
  `ValidateEnrollmentTokenTransition`/`Redeem`/`Expire`/`IsExpired` contract
  for the one-time Node Agent enrollment token (SPEC §10.2) — see
  "Enrollment Token" below.

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

## Local Admin Bootstrap and Break-Glass

Per SPEC §3.1 ("После clean install создаётся local admin/admin. Система
остаётся UNINITIALIZED до обязательной смены пароля при первом входе." /
"Local administrative identity сохраняется как break-glass даже при
использовании внешнего IdP"), `LocalAdminAccount` models the mandatory local
administrative identity as a pure value type — no password hashing/crypto,
persistence or HTTP/API surface:

- `LocalAdminAccountState` — `UNINITIALIZED` (the clean-install default) or
  `ACTIVE`.
- `ValidateLocalAdminTransition(current, next, credentialRevision)` allows
  only the single, one-way `UNINITIALIZED → ACTIVE` transition, and only
  when `credentialRevision >= 1` proves an actual credential-change event
  occurred; a state-only request with no such proof is rejected
  (`ErrLocalAdminTransitionUnproven`). `ACTIVE → UNINITIALIZED` and any
  self-loop are rejected regardless of `credentialRevision`.
- `LocalAdminAccount.Validate()` enforces that an `ACTIVE` account always
  carries `CredentialRevision >= 1` — an `ACTIVE` account with no recorded
  credential change (still "admin/admin"-equivalent) is rejected
  (`ErrLocalAdminCredentialNotChanged`).
- `RequireInitialized` is the guard normal operation must call: it returns
  `ErrLocalAdminNotInitialized` while `State == UNINITIALIZED`, encoding that
  "штатная эксплуатация до смены пароля невозможна" (AC-INST-002).
- `BreakGlassEligible(account, externalIdPHealthy)` reports whether the
  account is a valid break-glass recovery identity. It never inspects
  `externalIdPHealthy` beyond accepting it as a parameter — eligibility
  depends solely on the account being a valid, `ACTIVE` local admin account,
  proving Local Identity validity never depends on external IdP
  availability (AC-ID-001, AC-ID-004).

Password hashing/verification, first-login enforcement wiring, and any HTTP/
persistence integration remain later Release A/B work.

## Enrollment Token

Per SPEC §10.2 ("Enrollment: one-time token → cryptographic/mTLS machine
identity"), `EnrollmentToken` models the one-time Node Agent enrollment
token as a pure value type — no cryptographic generation/hashing,
persistence or transport binding happens here, only the checkable lifecycle
invariants every concrete adapter must uphold:

- `EnrollmentTokenID` — an opaque reference to the issued token (for example
  a stored hash or lookup key), mirroring the §15.3 Local Password Reset
  Token pattern ("raw token not stored"): it is never the raw,
  bearer-usable secret handed to the enrolling Node Agent.
- `EnrollmentTokenState` — `PENDING` (freshly issued), `REDEEMED` (consumed
  by a successful enrollment) or `EXPIRED`. Lifecycle:

  ```
  PENDING → REDEEMED | EXPIRED
  ```

  `PENDING` is the only non-terminal state; both `REDEEMED` and `EXPIRED`
  are terminal — a redeemed token can never later be reported as expired,
  and an expired token can never later be redeemed.
  `ValidateEnrollmentTokenTransition` enforces this.
- `EnrollmentToken.Validate()` enforces a valid `ID`, a known `State`, a
  strictly bounded `IssuedAt`/`ExpiresAt` window (AC: "Enrollment token
  expires"), and — when `State == REDEEMED` — a recorded `RedeemedAt` that
  is not before `IssuedAt`, proving the redemption actually occurred rather
  than being a state-only assertion.
- `EnrollmentToken.IsExpired(now)` reports whether the validity window has
  elapsed as of `now`, or the token is already `EXPIRED`. A `REDEEMED` token
  is never reported as expired: redemption and expiry are mutually
  exclusive terminal outcomes of the same `PENDING` token.
- `Redeem(token, now)` consumes a token exactly once as part of a successful
  enrollment (AC: "Enrollment token is single-use", "Token is unusable after
  successful enrollment"). It fails closed with
  `ErrEnrollmentTokenAlreadyRedeemed` if the token was already redeemed —
  so a second enrollment attempt with the same token can never succeed —
  and with `ErrEnrollmentTokenExpired` if the token is already `EXPIRED` or
  `now` is at/after `ExpiresAt`, even if never previously redeemed.
  Otherwise it returns a new `EnrollmentToken` in the `REDEEMED` state with
  `RedeemedAt` set to `now`; it never mutates the input token.
- `Expire(token, now)` transitions a lapsed `PENDING` token to `EXPIRED`
  (idempotent once already `EXPIRED`). It rejects an already-`REDEEMED`
  token with `ErrEnrollmentTokenAlreadyRedeemed` (redemption is a prior,
  successful, terminal outcome) and rejects expiring a still-valid
  `PENDING` token early with `ErrInvalidEnrollmentTokenTransition`.

Concrete token minting/hashing, delivery, persistence, and the resulting
`ServiceIdentity`/machine-identity issuance on successful enrollment
(SPEC §10.2's "cryptographic/mTLS machine identity", tracked separately as
Machine Identity) remain later Release A/B work.

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
- a freshly constructed `LocalAdminAccount` is `UNINITIALIZED` with
  `CredentialRevision == 0` and `RequireInitialized` rejects it
  (AC-INST-002);
- every legal and illegal `LocalAdminAccountState` transition — only
  `UNINITIALIZED → ACTIVE` is legal, and only when accompanied by a proven
  credential-change event (`credentialRevision >= 1`); a state-only request
  without that proof is rejected (`ErrLocalAdminTransitionUnproven`), and
  `ACTIVE → UNINITIALIZED` and every self-loop is rejected regardless of
  `credentialRevision`;
- `LocalAdminAccount.Validate()` rejecting an `ACTIVE` account with
  `CredentialRevision < 1` (still admin/admin-equivalent);
- `BreakGlassEligible` returning `true` for a valid `ACTIVE` local admin
  account for both `externalIdPHealthy == true` and `== false`, and `false`
  for an uninitialized or otherwise invalid account (AC-ID-001, AC-ID-004).
- valid/invalid `EnrollmentTokenID`/`EnrollmentTokenState` validation;
- `EnrollmentToken.Validate()` for a well-formed `PENDING`/`EXPIRED` token
  and every rejection case (invalid id, unknown state, zero/unbounded
  lifetime, a `REDEEMED` token missing or predating its `RedeemedAt`);
- `IsExpired` across a not-yet-expired `PENDING` token, a token exactly at
  and past its `ExpiresAt`, an explicitly `EXPIRED` token, and a `REDEEMED`
  token that remains never-expired even long past its original window;
- every legal and illegal `EnrollmentTokenState` transition — only
  `PENDING → REDEEMED` and `PENDING → EXPIRED` are legal; both terminal
  states reject any further transition, including self-loops;
- `Redeem` succeeding exactly once and then rejecting a second attempt on
  the same (now `REDEEMED`) token with `ErrEnrollmentTokenAlreadyRedeemed`
  (AC: single-use, "unusable after successful enrollment");
- `Redeem` rejecting an already-lapsed or already-`EXPIRED` token with
  `ErrEnrollmentTokenExpired` (AC: "Enrollment token expires"), and
  rejecting a structurally invalid token outright;
- `Expire` transitioning a lapsed `PENDING` token, idempotently no-op'ing on
  an already-`EXPIRED` token, rejecting a `REDEEMED` token with
  `ErrEnrollmentTokenAlreadyRedeemed`, and rejecting an early expiry attempt
  on a still-valid `PENDING` token with
  `ErrInvalidEnrollmentTokenTransition`.

This package has no dependency on `net/http`, `database/sql`, NATS,
Temporal, or any external identity provider — pure domain logic only,
consistent with `docs/FOUNDATION.md`'s isolation requirement.
