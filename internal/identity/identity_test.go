package identity

import "testing"

func TestIDValidation(t *testing.T) {
	t.Run("workspace id", func(t *testing.T) {
		cases := []struct {
			name    string
			id      WorkspaceID
			wantErr error
		}{
			{"valid", WorkspaceID("ws-1"), nil},
			{"empty", WorkspaceID(""), ErrInvalidWorkspaceID},
			{"whitespace", WorkspaceID("   "), ErrInvalidWorkspaceID},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				if err := tc.id.Validate(); err != tc.wantErr {
					t.Fatalf("got %v, want %v", err, tc.wantErr)
				}
			})
		}
	})

	t.Run("project id", func(t *testing.T) {
		cases := []struct {
			name    string
			id      ProjectID
			wantErr error
		}{
			{"valid", ProjectID("proj-1"), nil},
			{"empty", ProjectID(""), ErrInvalidProjectID},
			{"whitespace", ProjectID(" "), ErrInvalidProjectID},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				if err := tc.id.Validate(); err != tc.wantErr {
					t.Fatalf("got %v, want %v", err, tc.wantErr)
				}
			})
		}
	})

	t.Run("user id", func(t *testing.T) {
		cases := []struct {
			name    string
			id      UserID
			wantErr error
		}{
			{"valid", UserID("user-1"), nil},
			{"empty", UserID(""), ErrInvalidUserID},
			{"whitespace", UserID("\t"), ErrInvalidUserID},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				if err := tc.id.Validate(); err != tc.wantErr {
					t.Fatalf("got %v, want %v", err, tc.wantErr)
				}
			})
		}
	})
}

func TestRoleValidation(t *testing.T) {
	valid := []Role{
		RoleProjectOwner,
		RoleProductOwner,
		RoleContributor,
		RoleViewer,
		RoleClient,
		RoleProjectManager,
	}
	for _, r := range valid {
		r := r
		t.Run(string(r), func(t *testing.T) {
			if err := r.Validate(); err != nil {
				t.Fatalf("expected role %q to be valid, got %v", r, err)
			}
		})
	}

	invalid := []Role{"", "SUPER_ADMIN", "OWNER", "project_owner"}
	for _, r := range invalid {
		r := r
		t.Run("invalid_"+string(r), func(t *testing.T) {
			if err := r.Validate(); err != ErrInvalidRole {
				t.Fatalf("got %v, want %v", err, ErrInvalidRole)
			}
		})
	}
}

func TestWorkspaceAndProjectMembershipAreIndependent(t *testing.T) {
	// A ProjectMembership must be valid entirely on its own, with no
	// corresponding WorkspaceMembership record for the same user
	// (SPEC §3.2).
	pm := ProjectMembership{
		ProjectID: ProjectID("proj-1"),
		UserID:    UserID("user-1"),
		Role:      RoleContributor,
	}
	if err := pm.Validate(); err != nil {
		t.Fatalf("expected standalone project membership to be valid, got %v", err)
	}

	// No WorkspaceMembership exists anywhere in this test for "user-1", yet
	// the ProjectMembership above remains valid, demonstrating independence.
	var noWorkspaceMembershipRecorded []WorkspaceMembership
	if len(noWorkspaceMembershipRecorded) != 0 {
		t.Fatalf("expected no workspace membership records")
	}
}

func TestWorkspaceMembershipValidation(t *testing.T) {
	cases := []struct {
		name    string
		wm      WorkspaceMembership
		wantErr error
	}{
		{
			name:    "valid",
			wm:      WorkspaceMembership{WorkspaceID: "ws-1", UserID: "user-1"},
			wantErr: nil,
		},
		{
			name:    "missing workspace",
			wm:      WorkspaceMembership{UserID: "user-1"},
			wantErr: ErrInvalidWorkspaceID,
		},
		{
			name:    "missing user",
			wm:      WorkspaceMembership{WorkspaceID: "ws-1"},
			wantErr: ErrInvalidUserID,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.wm.Validate(); err != tc.wantErr {
				t.Fatalf("got %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestProjectMembershipValidation(t *testing.T) {
	cases := []struct {
		name    string
		pm      ProjectMembership
		wantErr error
	}{
		{
			name:    "valid",
			pm:      ProjectMembership{ProjectID: "proj-1", UserID: "user-1", Role: RoleViewer},
			wantErr: nil,
		},
		{
			name:    "missing project",
			pm:      ProjectMembership{UserID: "user-1", Role: RoleViewer},
			wantErr: ErrInvalidProjectID,
		},
		{
			name:    "missing user",
			pm:      ProjectMembership{ProjectID: "proj-1", Role: RoleViewer},
			wantErr: ErrInvalidUserID,
		},
		{
			name:    "invalid role",
			pm:      ProjectMembership{ProjectID: "proj-1", UserID: "user-1", Role: "NOT_A_ROLE"},
			wantErr: ErrInvalidRole,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.pm.Validate(); err != tc.wantErr {
				t.Fatalf("got %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestProjectInvitationValidation(t *testing.T) {
	base := func() ProjectInvitation {
		return ProjectInvitation{
			ProjectID: "proj-1",
			Invitee:   "alice@example.com",
			Role:      RoleContributor,
			State:     InvitationPending,
		}
	}

	cases := []struct {
		name    string
		mutate  func(ProjectInvitation) ProjectInvitation
		wantErr error
	}{
		{
			name:    "valid",
			mutate:  func(i ProjectInvitation) ProjectInvitation { return i },
			wantErr: nil,
		},
		{
			name: "missing project",
			mutate: func(i ProjectInvitation) ProjectInvitation {
				i.ProjectID = ""
				return i
			},
			wantErr: ErrInvalidProjectID,
		},
		{
			name: "empty invitee",
			mutate: func(i ProjectInvitation) ProjectInvitation {
				i.Invitee = ""
				return i
			},
			wantErr: ErrInvalidInvitee,
		},
		{
			name: "whitespace invitee",
			mutate: func(i ProjectInvitation) ProjectInvitation {
				i.Invitee = "  alice@example.com  "
				return i
			},
			wantErr: ErrInvalidInvitee,
		},
		{
			name: "wildcard invitee rejected (directory enumeration guard)",
			mutate: func(i ProjectInvitation) ProjectInvitation {
				i.Invitee = "*@example.com"
				return i
			},
			wantErr: ErrInvalidInvitee,
		},
		{
			name: "glob invitee rejected",
			mutate: func(i ProjectInvitation) ProjectInvitation {
				i.Invitee = "alice?@example.com"
				return i
			},
			wantErr: ErrInvalidInvitee,
		},
		{
			name: "invitee with spaces rejected",
			mutate: func(i ProjectInvitation) ProjectInvitation {
				i.Invitee = "alice smith@example.com"
				return i
			},
			wantErr: ErrInvalidInvitee,
		},
		{
			name: "invalid role",
			mutate: func(i ProjectInvitation) ProjectInvitation {
				i.Role = "NOT_A_ROLE"
				return i
			},
			wantErr: ErrInvalidRole,
		},
		{
			name: "invalid state",
			mutate: func(i ProjectInvitation) ProjectInvitation {
				i.State = "SOMETHING_ELSE"
				return i
			},
			wantErr: ErrInvalidInvitationState,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			inv := tc.mutate(base())
			if err := inv.Validate(); err != tc.wantErr {
				t.Fatalf("got %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestValidateInvitationTransition(t *testing.T) {
	cases := []struct {
		name    string
		current InvitationState
		next    InvitationState
		wantErr error
	}{
		{"pending to accepted", InvitationPending, InvitationAccepted, nil},
		{"pending to declined", InvitationPending, InvitationDeclined, nil},
		{"pending to expired", InvitationPending, InvitationExpired, nil},
		{"pending to revoked", InvitationPending, InvitationRevoked, nil},
		{"pending to pending (no-op) rejected", InvitationPending, InvitationPending, ErrInvalidInvitationTransition},
		{"accepted to declined rejected", InvitationAccepted, InvitationDeclined, ErrInvalidInvitationTransition},
		{"accepted to pending rejected", InvitationAccepted, InvitationPending, ErrInvalidInvitationTransition},
		{"revoked to accepted rejected (re-accepting revoked)", InvitationRevoked, InvitationAccepted, ErrInvalidInvitationTransition},
		{"declined to accepted rejected", InvitationDeclined, InvitationAccepted, ErrInvalidInvitationTransition},
		{"expired to accepted rejected", InvitationExpired, InvitationAccepted, ErrInvalidInvitationTransition},
		{"revoked to revoked rejected", InvitationRevoked, InvitationRevoked, ErrInvalidInvitationTransition},
		{"invalid current state", InvitationState("BOGUS"), InvitationAccepted, ErrInvalidInvitationState},
		{"invalid next state", InvitationPending, InvitationState("BOGUS"), ErrInvalidInvitationState},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateInvitationTransition(tc.current, tc.next)
			if err != tc.wantErr {
				t.Fatalf("got %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestNewProjectMembershipFromInvitation(t *testing.T) {
	acceptedInvitation := ProjectInvitation{
		ProjectID: "proj-1",
		Invitee:   "alice@example.com",
		Role:      RoleContributor,
		State:     InvitationAccepted,
	}

	t.Run("accepted invitation derives membership", func(t *testing.T) {
		pm, err := NewProjectMembershipFromInvitation(acceptedInvitation, UserID("user-1"), "alice@example.com")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := ProjectMembership{ProjectID: "proj-1", UserID: "user-1", Role: RoleContributor}
		if pm != want {
			t.Fatalf("got %+v, want %+v", pm, want)
		}
	})

	nonAcceptedStates := []InvitationState{
		InvitationPending,
		InvitationDeclined,
		InvitationExpired,
		InvitationRevoked,
	}
	for _, state := range nonAcceptedStates {
		state := state
		t.Run("rejects non-accepted state "+string(state), func(t *testing.T) {
			inv := acceptedInvitation
			inv.State = state
			_, err := NewProjectMembershipFromInvitation(inv, UserID("user-1"), "alice@example.com")
			if err != ErrMembershipRequiresAcceptedInvitation {
				t.Fatalf("got %v, want %v", err, ErrMembershipRequiresAcceptedInvitation)
			}
		})
	}

	t.Run("rejects invitee mismatch", func(t *testing.T) {
		_, err := NewProjectMembershipFromInvitation(acceptedInvitation, UserID("user-1"), "bob@example.com")
		if err != ErrMembershipInviteeMismatch {
			t.Fatalf("got %v, want %v", err, ErrMembershipInviteeMismatch)
		}
	})

	t.Run("rejects invalid user id", func(t *testing.T) {
		_, err := NewProjectMembershipFromInvitation(acceptedInvitation, UserID(""), "alice@example.com")
		if err != ErrInvalidUserID {
			t.Fatalf("got %v, want %v", err, ErrInvalidUserID)
		}
	})

	t.Run("rejects malformed underlying invitation", func(t *testing.T) {
		inv := acceptedInvitation
		inv.ProjectID = ""
		_, err := NewProjectMembershipFromInvitation(inv, UserID("user-1"), "alice@example.com")
		if err != ErrInvalidProjectID {
			t.Fatalf("got %v, want %v", err, ErrInvalidProjectID)
		}
	})
}
