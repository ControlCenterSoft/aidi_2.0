package identity

import (
	"errors"
	"strings"
	"time"
)

// EnrollmentTokenID identifies an EnrollmentToken record. Per SPEC §10.2
// ("Enrollment: one-time token → cryptographic/mTLS machine identity") and
// the §15.3 Local Password Reset Token pattern it mirrors ("raw token not
// stored"), this is an opaque reference to the issued token (for example a
// stored hash or lookup key) — it is never the raw, bearer-usable secret
// handed to the enrolling Node Agent. Minting/hashing the raw secret and any
// out-of-band delivery remain outside this pure domain package.
type EnrollmentTokenID string

// EnrollmentTokenState is the lifecycle state of an EnrollmentToken.
type EnrollmentTokenState string

const (
	// EnrollmentTokenPending is the state of a freshly issued token: it has
	// not yet been redeemed or expired.
	EnrollmentTokenPending EnrollmentTokenState = "PENDING"
	// EnrollmentTokenRedeemed is the terminal state reached after a
	// successful enrollment consumes the token exactly once (AC: "Token is
	// unusable after successful enrollment").
	EnrollmentTokenRedeemed EnrollmentTokenState = "REDEEMED"
	// EnrollmentTokenExpired is the terminal state reached once the
	// token's validity window has elapsed without redemption (AC:
	// "Enrollment token expires").
	EnrollmentTokenExpired EnrollmentTokenState = "EXPIRED"
)

var validEnrollmentTokenStates = map[EnrollmentTokenState]struct{}{
	EnrollmentTokenPending:  {},
	EnrollmentTokenRedeemed: {},
	EnrollmentTokenExpired:  {},
}

// terminalEnrollmentTokenStates cannot transition to any other state. Both
// REDEEMED and EXPIRED are terminal: a redeemed token cannot later be
// reported as expired, and an expired token cannot later be redeemed.
var terminalEnrollmentTokenStates = map[EnrollmentTokenState]struct{}{
	EnrollmentTokenRedeemed: {},
	EnrollmentTokenExpired:  {},
}

var (
	// ErrInvalidEnrollmentTokenID is returned when an EnrollmentTokenID is
	// empty/blank.
	ErrInvalidEnrollmentTokenID = errors.New("identity: invalid enrollment token id")
	// ErrInvalidEnrollmentTokenState is returned when an
	// EnrollmentTokenState is not one of the known lifecycle states.
	ErrInvalidEnrollmentTokenState = errors.New("identity: invalid enrollment token state")
	// ErrInvalidEnrollmentTokenLifetime is returned when an
	// EnrollmentToken's IssuedAt/ExpiresAt are zero or do not describe a
	// strictly positive, bounded lifetime (ExpiresAt must be after
	// IssuedAt).
	ErrInvalidEnrollmentTokenLifetime = errors.New("identity: enrollment token must have a strictly bounded lifetime (issued before expiry)")
	// ErrInvalidEnrollmentTokenTransition is returned when a requested
	// EnrollmentToken state transition is not permitted by the
	// deterministic lifecycle.
	ErrInvalidEnrollmentTokenTransition = errors.New("identity: invalid enrollment token state transition")
	// ErrEnrollmentTokenAlreadyRedeemed is returned by Redeem when the
	// token has already been consumed by a prior successful enrollment
	// (single-use enforcement).
	ErrEnrollmentTokenAlreadyRedeemed = errors.New("identity: enrollment token already redeemed")
	// ErrEnrollmentTokenExpired is returned by Redeem when the token's
	// validity window has elapsed (or it is already marked EXPIRED).
	ErrEnrollmentTokenExpired = errors.New("identity: enrollment token expired")
	// ErrEnrollmentTokenMissingRedeemedAt is returned when a REDEEMED
	// token does not carry a RedeemedAt timestamp proving the redemption
	// actually occurred.
	ErrEnrollmentTokenMissingRedeemedAt = errors.New("identity: redeemed enrollment token must record a redemption time")
)

func (id EnrollmentTokenID) Validate() error {
	if strings.TrimSpace(string(id)) == "" {
		return ErrInvalidEnrollmentTokenID
	}
	return nil
}

func (s EnrollmentTokenState) Validate() error {
	if _, ok := validEnrollmentTokenStates[s]; !ok {
		return ErrInvalidEnrollmentTokenState
	}
	return nil
}

// EnrollmentToken is the one-time Node Agent enrollment token described by
// SPEC §10.2. It is a pure value type: no cryptographic generation/hashing,
// persistence or transport binding happens here — only the checkable
// lifecycle invariants every concrete adapter must uphold.
type EnrollmentToken struct {
	ID         EnrollmentTokenID
	IssuedAt   time.Time
	ExpiresAt  time.Time
	State      EnrollmentTokenState
	RedeemedAt time.Time // zero unless State == EnrollmentTokenRedeemed
}

// Validate enforces: a valid ID, a known State, a strictly bounded
// IssuedAt/ExpiresAt window, and — when State is REDEEMED — a recorded
// RedeemedAt that is not before IssuedAt, proving the redemption actually
// occurred rather than being a state-only assertion.
func (t EnrollmentToken) Validate() error {
	if err := t.ID.Validate(); err != nil {
		return err
	}
	if err := t.State.Validate(); err != nil {
		return err
	}
	if t.IssuedAt.IsZero() || t.ExpiresAt.IsZero() {
		return ErrInvalidEnrollmentTokenLifetime
	}
	if !t.ExpiresAt.After(t.IssuedAt) {
		return ErrInvalidEnrollmentTokenLifetime
	}
	if t.State == EnrollmentTokenRedeemed {
		if t.RedeemedAt.IsZero() || t.RedeemedAt.Before(t.IssuedAt) {
			return ErrEnrollmentTokenMissingRedeemedAt
		}
	}
	return nil
}

// IsExpired reports whether the token's validity window has elapsed as of
// now, or it is already marked EXPIRED. A REDEEMED token is never reported
// as expired: redemption and expiry are mutually exclusive terminal
// outcomes of the same PENDING token.
func (t EnrollmentToken) IsExpired(now time.Time) bool {
	if t.State == EnrollmentTokenExpired {
		return true
	}
	if t.State == EnrollmentTokenRedeemed {
		return false
	}
	return !now.Before(t.ExpiresAt)
}

// ValidateEnrollmentTokenTransition rejects invalid EnrollmentToken
// lifecycle transitions. The only non-terminal state is PENDING; both
// REDEEMED and EXPIRED are terminal.
//
// Legal transitions:
//
//	PENDING -> REDEEMED
//	PENDING -> EXPIRED
func ValidateEnrollmentTokenTransition(current, next EnrollmentTokenState) error {
	if err := current.Validate(); err != nil {
		return err
	}
	if err := next.Validate(); err != nil {
		return err
	}
	if current == next {
		return ErrInvalidEnrollmentTokenTransition
	}
	if _, terminal := terminalEnrollmentTokenStates[current]; terminal {
		return ErrInvalidEnrollmentTokenTransition
	}
	// current == EnrollmentTokenPending here: REDEEMED and EXPIRED are the
	// only other valid states, and both are legal targets from PENDING.
	return nil
}

// Redeem consumes an EnrollmentToken exactly once as part of a successful
// Node Agent enrollment. It fails closed:
//
//   - ErrEnrollmentTokenAlreadyRedeemed if the token was already redeemed
//     (single-use enforcement; also covers "unusable after successful
//     enrollment" — a second enrollment attempt with the same token can
//     never succeed);
//   - ErrEnrollmentTokenExpired if the token is already EXPIRED or now is
//     at/after ExpiresAt (expiry enforcement), even if never previously
//     redeemed;
//   - otherwise it returns a new EnrollmentToken in the REDEEMED state with
//     RedeemedAt set to now.
//
// Redeem never mutates the input token; it returns a new value, consistent
// with the rest of this package's state-transition functions.
func Redeem(token EnrollmentToken, now time.Time) (EnrollmentToken, error) {
	if err := token.Validate(); err != nil {
		return EnrollmentToken{}, err
	}
	if token.State == EnrollmentTokenRedeemed {
		return EnrollmentToken{}, ErrEnrollmentTokenAlreadyRedeemed
	}
	if token.IsExpired(now) {
		return EnrollmentToken{}, ErrEnrollmentTokenExpired
	}
	if err := ValidateEnrollmentTokenTransition(token.State, EnrollmentTokenRedeemed); err != nil {
		return EnrollmentToken{}, err
	}

	redeemed := token
	redeemed.State = EnrollmentTokenRedeemed
	redeemed.RedeemedAt = now
	if err := redeemed.Validate(); err != nil {
		return EnrollmentToken{}, err
	}
	return redeemed, nil
}

// Expire transitions a PENDING EnrollmentToken to EXPIRED once its validity
// window has elapsed. It fails closed if the token is not currently
// eligible: an already-REDEEMED token cannot be retroactively expired
// (ErrEnrollmentTokenAlreadyRedeemed — redemption is a prior, successful,
// terminal outcome), and a still-valid PENDING token cannot be expired
// early (ErrInvalidEnrollmentTokenTransition).
func Expire(token EnrollmentToken, now time.Time) (EnrollmentToken, error) {
	if err := token.Validate(); err != nil {
		return EnrollmentToken{}, err
	}
	if token.State == EnrollmentTokenRedeemed {
		return EnrollmentToken{}, ErrEnrollmentTokenAlreadyRedeemed
	}
	if token.State == EnrollmentTokenExpired {
		return token, nil
	}
	if !now.Before(token.ExpiresAt) {
		if err := ValidateEnrollmentTokenTransition(token.State, EnrollmentTokenExpired); err != nil {
			return EnrollmentToken{}, err
		}
		expired := token
		expired.State = EnrollmentTokenExpired
		return expired, nil
	}
	return EnrollmentToken{}, ErrInvalidEnrollmentTokenTransition
}
