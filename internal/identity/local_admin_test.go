package identity

import "testing"

func TestLocalAdminAccountStateValidation(t *testing.T) {
	cases := []struct {
		name    string
		state   LocalAdminAccountState
		wantErr error
	}{
		{"uninitialized", LocalAdminUninitialized, nil},
		{"active", LocalAdminActive, nil},
		{"empty", LocalAdminAccountState(""), ErrInvalidLocalAdminState},
		{"bogus", LocalAdminAccountState("BOGUS"), ErrInvalidLocalAdminState},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.state.Validate(); err != tc.wantErr {
				t.Fatalf("got %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestLocalAdminAccountValidate(t *testing.T) {
	cases := []struct {
		name    string
		account LocalAdminAccount
		wantErr error
	}{
		{
			name:    "freshly constructed account is UNINITIALIZED and valid",
			account: LocalAdminAccount{UserID: "admin"},
			wantErr: nil,
		},
		{
			name: "active account with credential change is valid",
			account: LocalAdminAccount{
				UserID:             "admin",
				State:              LocalAdminActive,
				CredentialRevision: 1,
			},
			wantErr: nil,
		},
		{
			name: "active account with higher revision is valid",
			account: LocalAdminAccount{
				UserID:             "admin",
				State:              LocalAdminActive,
				CredentialRevision: 7,
			},
			wantErr: nil,
		},
		{
			name: "active account with no recorded credential change rejected",
			account: LocalAdminAccount{
				UserID:             "admin",
				State:              LocalAdminActive,
				CredentialRevision: 0,
			},
			wantErr: ErrLocalAdminMissingCredentialChange,
		},
		{
			name: "negative credential revision rejected",
			account: LocalAdminAccount{
				UserID:             "admin",
				State:              LocalAdminUninitialized,
				CredentialRevision: -1,
			},
			wantErr: ErrLocalAdminMissingCredentialChange,
		},
		{
			name:    "missing user id rejected",
			account: LocalAdminAccount{State: LocalAdminUninitialized},
			wantErr: ErrInvalidUserID,
		},
		{
			name: "invalid state rejected",
			account: LocalAdminAccount{
				UserID: "admin",
				State:  "BOGUS",
			},
			wantErr: ErrInvalidLocalAdminState,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.account.Validate(); err != tc.wantErr {
				t.Fatalf("got %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestValidateLocalAdminTransition(t *testing.T) {
	cases := []struct {
		name    string
		current LocalAdminAccountState
		next    LocalAdminAccountState
		wantErr error
	}{
		{"uninitialized to active legal", LocalAdminUninitialized, LocalAdminActive, nil},
		{"active to uninitialized rejected (non-reversible)", LocalAdminActive, LocalAdminUninitialized, ErrInvalidLocalAdminTransition},
		{"uninitialized to uninitialized rejected (self-loop)", LocalAdminUninitialized, LocalAdminUninitialized, ErrInvalidLocalAdminTransition},
		{"active to active rejected (self-loop)", LocalAdminActive, LocalAdminActive, ErrInvalidLocalAdminTransition},
		{"invalid current state", LocalAdminAccountState("BOGUS"), LocalAdminActive, ErrInvalidLocalAdminState},
		{"invalid next state", LocalAdminUninitialized, LocalAdminAccountState("BOGUS"), ErrInvalidLocalAdminState},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateLocalAdminTransition(tc.current, tc.next)
			if err != tc.wantErr {
				t.Fatalf("got %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestRequireInitialized(t *testing.T) {
	t.Run("freshly constructed account is rejected", func(t *testing.T) {
		account := LocalAdminAccount{UserID: "admin"}
		if err := RequireInitialized(account); err != ErrLocalAdminNotInitialized {
			t.Fatalf("got %v, want %v", err, ErrLocalAdminNotInitialized)
		}
	})

	t.Run("active account with credential change is accepted", func(t *testing.T) {
		account := LocalAdminAccount{
			UserID:             "admin",
			State:              LocalAdminActive,
			CredentialRevision: 1,
		}
		if err := RequireInitialized(account); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("active account without credential change still rejected", func(t *testing.T) {
		account := LocalAdminAccount{
			UserID: "admin",
			State:  LocalAdminActive,
		}
		if err := RequireInitialized(account); err != ErrLocalAdminMissingCredentialChange {
			t.Fatalf("got %v, want %v", err, ErrLocalAdminMissingCredentialChange)
		}
	})

	t.Run("invalid state surfaces state error", func(t *testing.T) {
		account := LocalAdminAccount{UserID: "admin", State: "BOGUS"}
		if err := RequireInitialized(account); err != ErrInvalidLocalAdminState {
			t.Fatalf("got %v, want %v", err, ErrInvalidLocalAdminState)
		}
	})
}

func TestBreakGlassEligible(t *testing.T) {
	activeAccount := LocalAdminAccount{
		UserID:             "admin",
		State:              LocalAdminActive,
		CredentialRevision: 1,
	}

	for _, idpHealthy := range []bool{true, false} {
		idpHealthy := idpHealthy
		t.Run("active account eligible regardless of IdP health", func(t *testing.T) {
			if !BreakGlassEligible(activeAccount, idpHealthy) {
				t.Fatalf("expected active local admin account to be break-glass eligible when externalIdPHealthy=%v", idpHealthy)
			}
		})
	}

	t.Run("uninitialized account not eligible", func(t *testing.T) {
		account := LocalAdminAccount{UserID: "admin"}
		if BreakGlassEligible(account, false) {
			t.Fatalf("expected UNINITIALIZED account to not be break-glass eligible")
		}
		if BreakGlassEligible(account, true) {
			t.Fatalf("expected UNINITIALIZED account to not be break-glass eligible")
		}
	})

	t.Run("active but invalid account (no credential change) not eligible", func(t *testing.T) {
		account := LocalAdminAccount{
			UserID: "admin",
			State:  LocalAdminActive,
		}
		if BreakGlassEligible(account, true) {
			t.Fatalf("expected invalid ACTIVE account to not be break-glass eligible")
		}
	})
}
