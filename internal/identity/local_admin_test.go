package identity

import "testing"

func TestNewLocalAdminAccount(t *testing.T) {
	acc := NewLocalAdminAccount(UserID("admin"))
	if acc.State != LocalAdminUninitialized {
		t.Fatalf("got state %v, want %v", acc.State, LocalAdminUninitialized)
	}
	if acc.CredentialRevision != 0 {
		t.Fatalf("got credential revision %d, want 0", acc.CredentialRevision)
	}
	if err := RequireInitialized(acc); err != ErrLocalAdminNotInitialized {
		t.Fatalf("got %v, want %v", err, ErrLocalAdminNotInitialized)
	}
}

func TestLocalAdminAccountValidation(t *testing.T) {
	cases := []struct {
		name    string
		acc     LocalAdminAccount
		wantErr error
	}{
		{
			name:    "uninitialized valid",
			acc:     LocalAdminAccount{UserID: "admin", State: LocalAdminUninitialized, CredentialRevision: 0},
			wantErr: nil,
		},
		{
			name:    "active with recorded credential change valid",
			acc:     LocalAdminAccount{UserID: "admin", State: LocalAdminActive, CredentialRevision: 1},
			wantErr: nil,
		},
		{
			name:    "active with higher credential revision valid",
			acc:     LocalAdminAccount{UserID: "admin", State: LocalAdminActive, CredentialRevision: 42},
			wantErr: nil,
		},
		{
			name:    "missing user id",
			acc:     LocalAdminAccount{State: LocalAdminUninitialized, CredentialRevision: 0},
			wantErr: ErrInvalidUserID,
		},
		{
			name:    "invalid state",
			acc:     LocalAdminAccount{UserID: "admin", State: "BOGUS", CredentialRevision: 0},
			wantErr: ErrInvalidLocalAdminState,
		},
		{
			name:    "negative credential revision",
			acc:     LocalAdminAccount{UserID: "admin", State: LocalAdminUninitialized, CredentialRevision: -1},
			wantErr: ErrLocalAdminCredentialNotChanged,
		},
		{
			name:    "active without recorded credential change (still admin/admin-equivalent)",
			acc:     LocalAdminAccount{UserID: "admin", State: LocalAdminActive, CredentialRevision: 0},
			wantErr: ErrLocalAdminCredentialNotChanged,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.acc.Validate(); err != tc.wantErr {
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
		{"uninitialized to uninitialized (self-loop) rejected", LocalAdminUninitialized, LocalAdminUninitialized, ErrInvalidLocalAdminTransition},
		{"active to uninitialized rejected", LocalAdminActive, LocalAdminUninitialized, ErrInvalidLocalAdminTransition},
		{"active to active (self-loop) rejected", LocalAdminActive, LocalAdminActive, ErrInvalidLocalAdminTransition},
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
	t.Run("rejects uninitialized account", func(t *testing.T) {
		acc := NewLocalAdminAccount(UserID("admin"))
		if err := RequireInitialized(acc); err != ErrLocalAdminNotInitialized {
			t.Fatalf("got %v, want %v", err, ErrLocalAdminNotInitialized)
		}
	})

	t.Run("accepts valid active account", func(t *testing.T) {
		acc := LocalAdminAccount{UserID: "admin", State: LocalAdminActive, CredentialRevision: 1}
		if err := RequireInitialized(acc); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("still rejects malformed active account", func(t *testing.T) {
		acc := LocalAdminAccount{UserID: "admin", State: LocalAdminActive, CredentialRevision: 0}
		if err := RequireInitialized(acc); err != ErrLocalAdminCredentialNotChanged {
			t.Fatalf("got %v, want %v", err, ErrLocalAdminCredentialNotChanged)
		}
	})
}

func TestBreakGlassEligible(t *testing.T) {
	activeAccount := LocalAdminAccount{UserID: "admin", State: LocalAdminActive, CredentialRevision: 1}

	t.Run("eligible with healthy external IdP", func(t *testing.T) {
		if !BreakGlassEligible(activeAccount, true) {
			t.Fatalf("expected active local admin account to be break-glass eligible when IdP is healthy")
		}
	})

	t.Run("eligible with unhealthy/down external IdP", func(t *testing.T) {
		if !BreakGlassEligible(activeAccount, false) {
			t.Fatalf("expected active local admin account to be break-glass eligible when IdP is unhealthy (AC-ID-001, AC-ID-004)")
		}
	})

	t.Run("not eligible while uninitialized", func(t *testing.T) {
		acc := NewLocalAdminAccount(UserID("admin"))
		if BreakGlassEligible(acc, true) {
			t.Fatalf("expected uninitialized local admin account to not be break-glass eligible")
		}
		if BreakGlassEligible(acc, false) {
			t.Fatalf("expected uninitialized local admin account to not be break-glass eligible")
		}
	})

	t.Run("not eligible when malformed (no recorded credential change)", func(t *testing.T) {
		acc := LocalAdminAccount{UserID: "admin", State: LocalAdminActive, CredentialRevision: 0}
		if BreakGlassEligible(acc, true) {
			t.Fatalf("expected malformed active account to not be break-glass eligible")
		}
	})

	t.Run("not eligible with invalid user id", func(t *testing.T) {
		acc := LocalAdminAccount{UserID: "", State: LocalAdminActive, CredentialRevision: 1}
		if BreakGlassEligible(acc, true) {
			t.Fatalf("expected account with invalid user id to not be break-glass eligible")
		}
	})
}
