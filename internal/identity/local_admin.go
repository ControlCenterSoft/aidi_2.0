package identity

import "errors"

// LocalAdminAccountState is the lifecycle state of the mandatory local
// administrative identity created on clean install (SPEC §3.1: "После clean
// install создаётся local admin/admin. Система остаётся UNINITIALIZED до
// обязательной смены пароля при первом входе.").
type LocalAdminAccountState string

const (
	// LocalAdminUninitialized is the initial state immediately after clean
	// install, before the mandatory first-login credential change
	// (AC-INST-002). Normal operation is not permitted in this state.
	LocalAdminUninitialized LocalAdminAccountState = "UNINITIALIZED"
	// LocalAdminActive is reached only after a proven credential-change
	// event and remains the system's break-glass identity thereafter
	// (AC-ID-001, AC-ID-004).
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
	// admin state transition is not the single permitted one-way move from
	// UNINITIALIZED to ACTIVE.
	ErrInvalidLocalAdminTransition = errors.New("identity: invalid local admin account state transition")
	// ErrLocalAdminCredentialNotChanged is returned when an ACTIVE local
	// admin account carries no recorded credential-change event, i.e. it is
	// still effectively in the default "admin/admin" state (SPEC §3.1).
	ErrLocalAdminCredentialNotChanged = errors.New("identity: active local admin account requires a recorded credential change")
	// ErrLocalAdminNotInitialized is returned by RequireInitialized when
	// normal operation is attempted while the local admin account is still
	// UNINITIALIZED (AC-INST-002).
	ErrLocalAdminNotInitialized = errors.New("identity: local admin account is not initialized; mandatory first-login credential change is required")
)

func (s LocalAdminAccountState) Validate() error {
	if _, ok := validLocalAdminAccountStates[s]; !ok {
		return ErrInvalidLocalAdminState
	}
	return nil
}

// LocalAdminAccount is the mandatory local administrative identity described
// in SPEC §3.1. It is a pure value type: no password hashing/crypto,
// persistence or HTTP/API surface lives here.
//
// CredentialRevision is a monotonic marker for "password changed at least
// once". It starts at 0 (never changed, i.e. still the clean-install
// default) and must be >= 1 once the account is ACTIVE.
type LocalAdminAccount struct {
	UserID             UserID
	State              LocalAdminAccountState
	CredentialRevision int
}

// NewLocalAdminAccount constructs a freshly installed local admin account:
// always UNINITIALIZED with no recorded credential change (SPEC §3.1).
func NewLocalAdminAccount(user UserID) LocalAdminAccount {
	return LocalAdminAccount{
		UserID:             user,
		State:              LocalAdminUninitialized,
		CredentialRevision: 0,
	}
}

// Validate enforces LocalAdminAccount invariants: a valid UserID, a
// recognized state, a non-negative CredentialRevision, and — critically —
// that an ACTIVE account is never still in the default "admin/admin"
// equivalent state (i.e. it must carry at least one recorded credential
// change).
func (a LocalAdminAccount) Validate() error {
	if err := a.UserID.Validate(); err != nil {
		return err
	}
	if err := a.State.Validate(); err != nil {
		return err
	}
	if a.CredentialRevision < 0 {
		return ErrLocalAdminCredentialNotChanged
	}
	if a.State == LocalAdminActive && a.CredentialRevision < 1 {
		return ErrLocalAdminCredentialNotChanged
	}
	return nil
}

// ValidateLocalAdminTransition rejects any LocalAdminAccountState transition
// other than the single legal, one-way move UNINITIALIZED -> ACTIVE, which
// must only be driven by a proven credential-change event. In particular
// ACTIVE -> UNINITIALIZED (reverting the mandatory first-login change) and
// any self-loop are rejected.
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

// RequireInitialized guards normal operation on behalf of a local admin
// account: it returns ErrLocalAdminNotInitialized while the account is
// UNINITIALIZED (AC-INST-002: "штатная эксплуатация до смены пароля
// невозможна"), and otherwise validates the account.
func RequireInitialized(account LocalAdminAccount) error {
	if account.State == LocalAdminUninitialized {
		return ErrLocalAdminNotInitialized
	}
	return account.Validate()
}

// BreakGlassEligible reports whether the given local admin account remains a
// valid recovery ("break-glass") identity. Per SPEC §3.1 ("Local
// administrative identity сохраняется как break-glass даже при
// использовании внешнего IdP") this depends only on the account itself being
// a valid, ACTIVE local admin account — externalIdPHealthy is intentionally
// never consulted, proving that Local Identity validity never depends on
// external IdP availability (AC-ID-001, AC-ID-004).
func BreakGlassEligible(account LocalAdminAccount, externalIdPHealthy bool) bool {
	_ = externalIdPHealthy
	if account.State != LocalAdminActive {
		return false
	}
	return account.Validate() == nil
}
