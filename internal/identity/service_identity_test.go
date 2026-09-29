package identity

import (
	"testing"
	"time"
)

func TestServiceIdentity(t *testing.T) {
	baseIssued := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	baseExpires := baseIssued.Add(time.Hour)

	validCredential := ServiceCredential{IssuedAt: baseIssued, ExpiresAt: baseExpires}

	t.Run("service identity id validation", func(t *testing.T) {
		cases := []struct {
			name    string
			id      ServiceIdentityID
			wantErr error
		}{
			{"valid", ServiceIdentityID("svc-1"), nil},
			{"empty", ServiceIdentityID(""), ErrInvalidServiceIdentityID},
			{"whitespace", ServiceIdentityID("   "), ErrInvalidServiceIdentityID},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				if err := tc.id.Validate(); err != tc.wantErr {
					t.Fatalf("got %v, want %v", err, tc.wantErr)
				}
			})
		}
	})

	t.Run("service kind validation", func(t *testing.T) {
		cases := []struct {
			name    string
			kind    ServiceKind
			wantErr error
		}{
			{"node agent", ServiceKindNodeAgent, nil},
			{"crm", ServiceKindCRM, nil},
			{"provider", ServiceKindProvider, nil},
			{"automation", ServiceKindAutomation, nil},
			{"unknown", ServiceKind("BOGUS"), ErrInvalidServiceKind},
			{"empty", ServiceKind(""), ErrInvalidServiceKind},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				if err := tc.kind.Validate(); err != tc.wantErr {
					t.Fatalf("got %v, want %v", err, tc.wantErr)
				}
			})
		}
	})

	t.Run("scope validation", func(t *testing.T) {
		cases := []struct {
			name    string
			scope   Scope
			wantErr error
		}{
			{"valid", Scope{Permission: "project:read"}, nil},
			{"empty", Scope{Permission: ""}, ErrInvalidScope},
			{"whitespace padded", Scope{Permission: " project:read "}, ErrInvalidScope},
			{"wildcard star", Scope{Permission: "project:*"}, ErrInvalidScope},
			{"wildcard question", Scope{Permission: "project:read?"}, ErrInvalidScope},
			{"wildcard percent", Scope{Permission: "project:%"}, ErrInvalidScope},
			{"contains space", Scope{Permission: "project read"}, ErrInvalidScope},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				if err := tc.scope.Validate(); err != tc.wantErr {
					t.Fatalf("got %v, want %v", err, tc.wantErr)
				}
			})
		}
	})

	t.Run("credential lifetime validation", func(t *testing.T) {
		cases := []struct {
			name       string
			credential ServiceCredential
			wantErr    error
		}{
			{"valid bounded lifetime", validCredential, nil},
			{"zero issued at", ServiceCredential{ExpiresAt: baseExpires}, ErrInvalidCredentialLifetime},
			{"zero expires at", ServiceCredential{IssuedAt: baseIssued}, ErrInvalidCredentialLifetime},
			{"both zero", ServiceCredential{}, ErrInvalidCredentialLifetime},
			{"expires equal issued", ServiceCredential{IssuedAt: baseIssued, ExpiresAt: baseIssued}, ErrInvalidCredentialLifetime},
			{"expires before issued", ServiceCredential{IssuedAt: baseExpires, ExpiresAt: baseIssued}, ErrInvalidCredentialLifetime},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				if err := tc.credential.Validate(); err != tc.wantErr {
					t.Fatalf("got %v, want %v", err, tc.wantErr)
				}
			})
		}
	})

	t.Run("service identity validation", func(t *testing.T) {
		validIdentity := ServiceIdentity{
			ID:         ServiceIdentityID("svc-1"),
			Kind:       ServiceKindNodeAgent,
			Scopes:     []Scope{{Permission: "project:read"}, {Permission: "task:write"}},
			Credential: validCredential,
		}

		cases := []struct {
			name     string
			identity ServiceIdentity
			wantErr  error
		}{
			{
				name:     "valid, at least one scope, bounded credential",
				identity: validIdentity,
				wantErr:  nil,
			},
			{
				name: "empty id",
				identity: ServiceIdentity{
					ID:         ServiceIdentityID(""),
					Kind:       ServiceKindNodeAgent,
					Scopes:     []Scope{{Permission: "project:read"}},
					Credential: validCredential,
				},
				wantErr: ErrInvalidServiceIdentityID,
			},
			{
				name: "unknown kind",
				identity: ServiceIdentity{
					ID:         ServiceIdentityID("svc-1"),
					Kind:       ServiceKind("BOGUS"),
					Scopes:     []Scope{{Permission: "project:read"}},
					Credential: validCredential,
				},
				wantErr: ErrInvalidServiceKind,
			},
			{
				name: "zero scopes",
				identity: ServiceIdentity{
					ID:         ServiceIdentityID("svc-1"),
					Kind:       ServiceKindNodeAgent,
					Scopes:     nil,
					Credential: validCredential,
				},
				wantErr: ErrNoScopes,
			},
			{
				name: "duplicate scopes",
				identity: ServiceIdentity{
					ID:         ServiceIdentityID("svc-1"),
					Kind:       ServiceKindNodeAgent,
					Scopes:     []Scope{{Permission: "project:read"}, {Permission: "project:read"}},
					Credential: validCredential,
				},
				wantErr: ErrDuplicateScope,
			},
			{
				name: "wildcard scope",
				identity: ServiceIdentity{
					ID:         ServiceIdentityID("svc-1"),
					Kind:       ServiceKindNodeAgent,
					Scopes:     []Scope{{Permission: "project:*"}},
					Credential: validCredential,
				},
				wantErr: ErrInvalidScope,
			},
			{
				name: "expired-at-issuance credential",
				identity: ServiceIdentity{
					ID:         ServiceIdentityID("svc-1"),
					Kind:       ServiceKindNodeAgent,
					Scopes:     []Scope{{Permission: "project:read"}},
					Credential: ServiceCredential{IssuedAt: baseIssued, ExpiresAt: baseIssued},
				},
				wantErr: ErrInvalidCredentialLifetime,
			},
			{
				name: "zero-value credential",
				identity: ServiceIdentity{
					ID:         ServiceIdentityID("svc-1"),
					Kind:       ServiceKindNodeAgent,
					Scopes:     []Scope{{Permission: "project:read"}},
					Credential: ServiceCredential{},
				},
				wantErr: ErrInvalidCredentialLifetime,
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				if err := tc.identity.Validate(); err != tc.wantErr {
					t.Fatalf("got %v, want %v", err, tc.wantErr)
				}
			})
		}
	})

	t.Run("type separation from user identity", func(t *testing.T) {
		// ServiceIdentityID/ServiceIdentity are distinct named types from
		// UserID/ProjectMembership/Role: this is enforced by the Go
		// compiler, not just by naming convention. A ServiceIdentityID
		// cannot be assigned to a field typed UserID (or vice versa)
		// without an explicit, visible conversion — there is no implicit
		// interchangeability. Uncommenting either line below would fail to
		// compile, demonstrating the separation:
		//
		//	var _ UserID = ServiceIdentityID("svc-1")
		//	var _ ProjectMembership = ServiceIdentity{}
		svc := ServiceIdentity{
			ID:         ServiceIdentityID("svc-1"),
			Kind:       ServiceKindAutomation,
			Scopes:     []Scope{{Permission: "workflow:trigger"}},
			Credential: validCredential,
		}
		if err := svc.Validate(); err != nil {
			t.Fatalf("expected valid service identity, got %v", err)
		}
	})
}
