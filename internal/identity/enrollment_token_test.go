package identity

import (
	"testing"
	"time"
)

func TestEnrollmentToken(t *testing.T) {
	issuedAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	expiresAt := issuedAt.Add(time.Hour)

	validPending := EnrollmentToken{
		ID:        EnrollmentTokenID("tok-1"),
		IssuedAt:  issuedAt,
		ExpiresAt: expiresAt,
		State:     EnrollmentTokenPending,
	}

	t.Run("enrollment token id validation", func(t *testing.T) {
		cases := []struct {
			name    string
			id      EnrollmentTokenID
			wantErr error
		}{
			{"valid", EnrollmentTokenID("tok-1"), nil},
			{"empty", EnrollmentTokenID(""), ErrInvalidEnrollmentTokenID},
			{"whitespace", EnrollmentTokenID("   "), ErrInvalidEnrollmentTokenID},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				if err := tc.id.Validate(); err != tc.wantErr {
					t.Fatalf("got %v, want %v", err, tc.wantErr)
				}
			})
		}
	})

	t.Run("enrollment token state validation", func(t *testing.T) {
		cases := []struct {
			name    string
			state   EnrollmentTokenState
			wantErr error
		}{
			{"pending", EnrollmentTokenPending, nil},
			{"redeemed", EnrollmentTokenRedeemed, nil},
			{"expired", EnrollmentTokenExpired, nil},
			{"unknown", EnrollmentTokenState("BOGUS"), ErrInvalidEnrollmentTokenState},
			{"empty", EnrollmentTokenState(""), ErrInvalidEnrollmentTokenState},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				if err := tc.state.Validate(); err != tc.wantErr {
					t.Fatalf("got %v, want %v", err, tc.wantErr)
				}
			})
		}
	})

	t.Run("validate", func(t *testing.T) {
		cases := []struct {
			name    string
			token   EnrollmentToken
			wantErr error
		}{
			{"valid pending", validPending, nil},
			{
				name: "empty id",
				token: EnrollmentToken{
					ID: EnrollmentTokenID(""), IssuedAt: issuedAt, ExpiresAt: expiresAt, State: EnrollmentTokenPending,
				},
				wantErr: ErrInvalidEnrollmentTokenID,
			},
			{
				name: "unknown state",
				token: EnrollmentToken{
					ID: EnrollmentTokenID("tok-1"), IssuedAt: issuedAt, ExpiresAt: expiresAt, State: EnrollmentTokenState("BOGUS"),
				},
				wantErr: ErrInvalidEnrollmentTokenState,
			},
			{
				name: "zero issued at",
				token: EnrollmentToken{
					ID: EnrollmentTokenID("tok-1"), ExpiresAt: expiresAt, State: EnrollmentTokenPending,
				},
				wantErr: ErrInvalidEnrollmentTokenLifetime,
			},
			{
				name: "zero expires at",
				token: EnrollmentToken{
					ID: EnrollmentTokenID("tok-1"), IssuedAt: issuedAt, State: EnrollmentTokenPending,
				},
				wantErr: ErrInvalidEnrollmentTokenLifetime,
			},
			{
				name: "expires equal issued",
				token: EnrollmentToken{
					ID: EnrollmentTokenID("tok-1"), IssuedAt: issuedAt, ExpiresAt: issuedAt, State: EnrollmentTokenPending,
				},
				wantErr: ErrInvalidEnrollmentTokenLifetime,
			},
			{
				name: "expires before issued",
				token: EnrollmentToken{
					ID: EnrollmentTokenID("tok-1"), IssuedAt: expiresAt, ExpiresAt: issuedAt, State: EnrollmentTokenPending,
				},
				wantErr: ErrInvalidEnrollmentTokenLifetime,
			},
			{
				name: "redeemed without redeemed-at",
				token: EnrollmentToken{
					ID: EnrollmentTokenID("tok-1"), IssuedAt: issuedAt, ExpiresAt: expiresAt, State: EnrollmentTokenRedeemed,
				},
				wantErr: ErrEnrollmentTokenMissingRedeemedAt,
			},
			{
				name: "redeemed with redeemed-at before issued-at",
				token: EnrollmentToken{
					ID: EnrollmentTokenID("tok-1"), IssuedAt: issuedAt, ExpiresAt: expiresAt,
					State: EnrollmentTokenRedeemed, RedeemedAt: issuedAt.Add(-time.Minute),
				},
				wantErr: ErrEnrollmentTokenMissingRedeemedAt,
			},
			{
				name: "redeemed with valid redeemed-at",
				token: EnrollmentToken{
					ID: EnrollmentTokenID("tok-1"), IssuedAt: issuedAt, ExpiresAt: expiresAt,
					State: EnrollmentTokenRedeemed, RedeemedAt: issuedAt.Add(time.Minute),
				},
				wantErr: nil,
			},
			{
				name: "expired state",
				token: EnrollmentToken{
					ID: EnrollmentTokenID("tok-1"), IssuedAt: issuedAt, ExpiresAt: expiresAt, State: EnrollmentTokenExpired,
				},
				wantErr: nil,
			},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				if err := tc.token.Validate(); err != tc.wantErr {
					t.Fatalf("got %v, want %v", err, tc.wantErr)
				}
			})
		}
	})

	t.Run("is expired", func(t *testing.T) {
		cases := []struct {
			name  string
			token EnrollmentToken
			now   time.Time
			want  bool
		}{
			{"pending before expiry", validPending, issuedAt.Add(30 * time.Minute), false},
			{"pending exactly at expiry", validPending, expiresAt, true},
			{"pending after expiry", validPending, expiresAt.Add(time.Minute), true},
			{"explicitly expired state", EnrollmentToken{State: EnrollmentTokenExpired}, issuedAt, true},
			{
				"redeemed is never expired even past window",
				EnrollmentToken{
					ID: EnrollmentTokenID("tok-1"), IssuedAt: issuedAt, ExpiresAt: expiresAt,
					State: EnrollmentTokenRedeemed, RedeemedAt: issuedAt.Add(time.Minute),
				},
				expiresAt.Add(24 * time.Hour),
				false,
			},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				if got := tc.token.IsExpired(tc.now); got != tc.want {
					t.Fatalf("got %v, want %v", got, tc.want)
				}
			})
		}
	})

	t.Run("transition validation", func(t *testing.T) {
		cases := []struct {
			name    string
			current EnrollmentTokenState
			next    EnrollmentTokenState
			wantErr error
		}{
			{"pending to redeemed", EnrollmentTokenPending, EnrollmentTokenRedeemed, nil},
			{"pending to expired", EnrollmentTokenPending, EnrollmentTokenExpired, nil},
			{"pending to pending", EnrollmentTokenPending, EnrollmentTokenPending, ErrInvalidEnrollmentTokenTransition},
			{"redeemed to expired", EnrollmentTokenRedeemed, EnrollmentTokenExpired, ErrInvalidEnrollmentTokenTransition},
			{"redeemed to pending", EnrollmentTokenRedeemed, EnrollmentTokenPending, ErrInvalidEnrollmentTokenTransition},
			{"expired to redeemed", EnrollmentTokenExpired, EnrollmentTokenRedeemed, ErrInvalidEnrollmentTokenTransition},
			{"expired to pending", EnrollmentTokenExpired, EnrollmentTokenPending, ErrInvalidEnrollmentTokenTransition},
			{"unknown current", EnrollmentTokenState("BOGUS"), EnrollmentTokenRedeemed, ErrInvalidEnrollmentTokenState},
			{"unknown next", EnrollmentTokenPending, EnrollmentTokenState("BOGUS"), ErrInvalidEnrollmentTokenState},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				if err := ValidateEnrollmentTokenTransition(tc.current, tc.next); err != tc.wantErr {
					t.Fatalf("got %v, want %v", err, tc.wantErr)
				}
			})
		}
	})

	t.Run("redeem succeeds exactly once (single-use, AC: unusable after enrollment)", func(t *testing.T) {
		redeemAt := issuedAt.Add(10 * time.Minute)
		redeemed, err := Redeem(validPending, redeemAt)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if redeemed.State != EnrollmentTokenRedeemed {
			t.Fatalf("expected state REDEEMED, got %v", redeemed.State)
		}
		if !redeemed.RedeemedAt.Equal(redeemAt) {
			t.Fatalf("expected RedeemedAt %v, got %v", redeemAt, redeemed.RedeemedAt)
		}

		// A second enrollment attempt with the now-redeemed token must
		// never succeed (single-use; "unusable after successful
		// enrollment").
		_, err = Redeem(redeemed, redeemAt.Add(time.Minute))
		if err != ErrEnrollmentTokenAlreadyRedeemed {
			t.Fatalf("got %v, want %v", err, ErrEnrollmentTokenAlreadyRedeemed)
		}
	})

	t.Run("redeem fails once expired (AC: enrollment token expires)", func(t *testing.T) {
		_, err := Redeem(validPending, expiresAt)
		if err != ErrEnrollmentTokenExpired {
			t.Fatalf("got %v, want %v", err, ErrEnrollmentTokenExpired)
		}

		_, err = Redeem(validPending, expiresAt.Add(time.Hour))
		if err != ErrEnrollmentTokenExpired {
			t.Fatalf("got %v, want %v", err, ErrEnrollmentTokenExpired)
		}

		alreadyExpired := EnrollmentToken{
			ID: EnrollmentTokenID("tok-1"), IssuedAt: issuedAt, ExpiresAt: expiresAt, State: EnrollmentTokenExpired,
		}
		_, err = Redeem(alreadyExpired, issuedAt.Add(time.Minute))
		if err != ErrEnrollmentTokenExpired {
			t.Fatalf("got %v, want %v", err, ErrEnrollmentTokenExpired)
		}
	})

	t.Run("redeem rejects an invalid token", func(t *testing.T) {
		invalid := EnrollmentToken{ID: EnrollmentTokenID("")}
		_, err := Redeem(invalid, issuedAt)
		if err != ErrInvalidEnrollmentTokenID {
			t.Fatalf("got %v, want %v", err, ErrInvalidEnrollmentTokenID)
		}
	})

	t.Run("expire transitions a lapsed pending token", func(t *testing.T) {
		expired, err := Expire(validPending, expiresAt)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if expired.State != EnrollmentTokenExpired {
			t.Fatalf("expected state EXPIRED, got %v", expired.State)
		}
	})

	t.Run("expire is idempotent once already expired", func(t *testing.T) {
		alreadyExpired := EnrollmentToken{
			ID: EnrollmentTokenID("tok-1"), IssuedAt: issuedAt, ExpiresAt: expiresAt, State: EnrollmentTokenExpired,
		}
		got, err := Expire(alreadyExpired, expiresAt.Add(time.Hour))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.State != EnrollmentTokenExpired {
			t.Fatalf("expected state EXPIRED, got %v", got.State)
		}
	})

	t.Run("expire rejects a redeemed token", func(t *testing.T) {
		redeemed := EnrollmentToken{
			ID: EnrollmentTokenID("tok-1"), IssuedAt: issuedAt, ExpiresAt: expiresAt,
			State: EnrollmentTokenRedeemed, RedeemedAt: issuedAt.Add(time.Minute),
		}
		_, err := Expire(redeemed, expiresAt.Add(time.Hour))
		if err != ErrEnrollmentTokenAlreadyRedeemed {
			t.Fatalf("got %v, want %v", err, ErrEnrollmentTokenAlreadyRedeemed)
		}
	})

	t.Run("expire rejects a still-valid pending token", func(t *testing.T) {
		_, err := Expire(validPending, issuedAt.Add(time.Minute))
		if err != ErrInvalidEnrollmentTokenTransition {
			t.Fatalf("got %v, want %v", err, ErrInvalidEnrollmentTokenTransition)
		}
	})
}
