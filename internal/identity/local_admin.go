package identity

import "errors"

// LocalAdminAccountState is the lifecycle state of the mandatory local
// administrative identity created by a clean install (SPEC §3.1,
// AC-INST-002). It intentionally has only two states: the bootstrap
// "admin/admin" state and the post-credential-change operational state.
type LocalAdminAccountState string

const (
	// LocalAdminUninitialized is the state immediately after clean install:
	// the well-known "admin/admin" bootstrap credential is in effect and
	// normal operation is not permitted (SPEC §3.1, AC-INST-002).
	LocalAdminUninitialized LocalAdminAccountState = "UNINITIALIZED"
	// LocalAdminActive is reached only after a proven credential-change
	// event (the mandatory first-login password change). It is the sole
	// state in which normal operation is permitted.
	LocalAdminActive LocalAdminAccountState = "ACTIVE"
)

var validLocalAdminAccountStates = map[LocalAdminAccountState]struct{}{
	LocalAdminUninitialized: {},
	LocalAdminActive:        {},
}

var (
	// ErrInvalidLocalAdminState is returned when a LocalAdminAccountState is
	// not one of the two recognized states.
	ErrInvalidLocalAdminState = errors.New("identity: invalid local admin account state")
	// ErrInvalidLocalAdminTransition is returned when a requested local
	// admin state transition is not the one legal, one-way transition
	// UNINITIALIZED -> ACTIVE.
	ErrInvalidLocalAdminTransition = errors.New("identity: invalid local admin account state transition")
	// ErrLocalAdminNotInitialized is returned by RequireInitialized when
	// normal operation is attempted while the local admin account is still
	// UNINITIALIZED (SPEC §3.1, AC-INST-002: "штатная эксплуатация до смены
	// password невозможна").
	ErrLocalAdminNotInitialized = errors.New("identity: local admin account requires mandatory password change before use")
	// ErrLocalAdminMissingCredentialChange is returned when an ACTIVE local
	// admin account carries no recorded credential change, i.e. it is still
	// in the "admin/admin"-equivalent state despite claiming to be ACTIVE.
	ErrLocalAdminMissingCredentialChange = errors.New("identity: active local admin account has no recorded credential change")
)

func (s LocalAdminAccountState) Validate() error {
	if _, ok := validLocalAdminAccountStates[s]; !ok {
		return ErrInvalidLocalAdminState
	}
	return nil
}

// LocalAdminAccount is the mandatory local administrative identity created
// by a clean install (SPEC §3.1). It is a pure value type: no password
// hashing/crypto, no persistence and no HTTP/API surface.
//
// A freshly constructed LocalAdminAccount (Go zero value, State == "") is
// UNINITIALIZED: this mirrors the clean-install bootstrap "admin/admin"
// state without requiring callers to explicitly set State. Use
// EffectiveState (or Validate/RequireInitialized, which use it internally)
// rather than comparing State directly.
//
// CredentialRevision is a monotonic marker for "password changed": zero
// means the bootstrap "admin/admin" credential is still in effect; any
// value >= 1 records that the mandatory first-login password change (or a
// subsequent change) has occurred.
type LocalAdminAccount struct {
	UserID             UserID
	State              LocalAdminAccountState
	CredentialRevision int
}

// EffectiveState returns the account's State, treating the Go zero value
// ("") as LocalAdminUninitialized. A freshly constructed LocalAdminAccount
// (zero value) is therefore UNINITIALIZED by default, matching a clean
// install's bootstrap "admin/admin" state (SPEC §3.1) without requiring
// callers to explicitly set State.
func (a LocalAdminAccount) EffectiveState() LocalAdminAccountState {
	if a.State == "" {
		return LocalAdminUninitialized
	}
	return a.State
}

// Validate enforces the state/revision invariants of a LocalAdminAccount:
// a valid UserID, a recognized state, a non-negative CredentialRevision,
// and — critically — an ACTIVE account must have CredentialRevision >= 1
// (AC-INST-002: an ACTIVE account with no recorded credential change would
// still be the "admin/admin"-equivalent state).
func (a LocalAdminAccount) Validate() error {
	if err := a.UserID.Validate(); err != nil {
		return err
	}
	state := a.EffectiveState()
	if err := state.Validate(); err != nil {
		return err
	}
	if a.CredentialRevision < 0 {
		return ErrLocalAdminMissingCredentialChange
	}
	if state == LocalAdminActive && a.CredentialRevision < 1 {
		return ErrLocalAdminMissingCredentialChange
	}
	return nil
}

// ValidateLocalAdminTransition rejects any LocalAdminAccountState
// transition other than the single legal, one-way transition
// UNINITIALIZED -> ACTIVE (SPEC §3.1). In particular:
//
//   - ACTIVE -> UNINITIALIZED is rejected: the bootstrap state cannot be
//     re-entered through this contract (the transition is non-reversible).
//   - Any self-loop (UNINITIALIZED -> UNINITIALIZED, ACTIVE -> ACTIVE) is
//     rejected.
func ValidateLocalAdminTransition(current, next LocalAdminAccountState) error {
	if err := current.Validate(); err != nil {
		return err
	}
	if err := next.Validate(); err != nil {
		return err
	}
	if current == LocalAdminUninitialized && next == LocalAdminActive {
		return nil
	}
	return ErrInvalidLocalAdminTransition
}

// RequireInitialized returns ErrLocalAdminNotInitialized when normal
// operation is attempted while the local admin account is still
// UNINITIALIZED, and otherwise validates the account (AC-INST-002).
func RequireInitialized(account LocalAdminAccount) error {
	state := account.EffectiveState()
	if err := state.Validate(); err != nil {
		return err
	}
	if state == LocalAdminUninitialized {
		return ErrLocalAdminNotInitialized
	}
	return account.Validate()
}

// BreakGlassEligible reports whether the given local admin account remains
// a valid administrative recovery path, independent of external IdP
// availability (AC-ID-001, AC-ID-004). Local Identity validity must never
// depend on IdP state, so externalIdPHealthy does not affect the result:
// eligibility is determined solely by the account being a valid, ACTIVE
// local admin account.
func BreakGlassEligible(account LocalAdminAccount, externalIdPHealthy bool) bool {
	_ = externalIdPHealthy // IdP health is intentionally irrelevant (AC-ID-001, AC-ID-004).
	if account.EffectiveState() != LocalAdminActive {
		return false
	}
	return account.Validate() == nil
}
