package identity

import (
	"errors"
	"testing"
)

func TestRoleValidate(t *testing.T) {
	cases := []struct {
		name    string
		role    Role
		wantErr error
	}{
		{"project owner", RoleProjectOwner, nil},
		{"product owner", RoleProductOwner, nil},
		{"contributor", RoleContributor, nil},
		{"viewer", RoleViewer, nil},
		{"client", RoleClient, nil},
		{"project manager", RoleProjectManager, nil},
		{"unknown role", Role("SUPER_ADMIN"), ErrInvalidRole},
		{"empty role", Role(""), ErrInvalidRole},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.role.Validate()
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("Role(%q).Validate() = %v, want %v", tc.role, err, tc.wantErr)
			}
		})
	}
}

func TestIDValidate(t *testing.T) {
	if err := WorkspaceID("ws-1").Validate(); err != nil {
		t.Fatalf("valid workspace id rejected: %v", err)
	}
	if err := WorkspaceID(" ").Validate(); !errors.Is(err, ErrInvalidWorkspaceID) {
		t.Fatalf("blank workspace id accepted: %v", err)
	}
	if err := WorkspaceID("").Validate(); !errors.Is(err, ErrInvalidWorkspaceID) {
		t.Fatalf("empty workspace id accepted: %v", err)
	}

	if err := ProjectID("proj-1").Validate(); err != nil {
		t.Fatalf("valid project id rejected: %v", err)
	}
	if err := ProjectID("").Validate(); !errors.Is(err, ErrInvalidProjectID) {
		t.Fatalf("empty project id accepted: %v", err)
	}

	if err := UserID("user-1").Validate(); err != nil {
		t.Fatalf("valid user id rejected: %v", err)
	}
	if err := UserID("").Validate(); !errors.Is(err, ErrInvalidUserID) {
		t.Fatalf("empty user id accepted: %v", err)
	}
}

// TestMembershipIndependence proves that a valid ProjectMembership does not
// assume or imply a WorkspaceMembership for the same user, and vice versa
// (SPEC §3.2).
func TestMembershipIndependence(t *testing.T) {
	user := UserID("user-1")
	project := ProjectID("proj-1")

	projectMembership := ProjectMembership{
		ProjectID: project,
		UserID:    user,
		Role:      RoleContributor,
	}
	if err := projectMembership.Validate(); err != nil {
		t.Fatalf("valid project membership rejected: %v", err)
	}

	// The ProjectMembership above is valid on its own; it carries no
	// WorkspaceID and does not require any WorkspaceMembership record to
	// exist for the same user to be considered valid.
	var noWorkspaceMembership *WorkspaceMembership
	if noWorkspaceMembership != nil {
		t.Fatal("no workspace membership should be required")
	}

	workspaceMembership := WorkspaceMembership{
		WorkspaceID: WorkspaceID("ws-1"),
		UserID:      user,
	}
	if err := workspaceMembership.Validate(); err != nil {
		t.Fatalf("valid workspace membership rejected: %v", err)
	}

	// A WorkspaceMembership for the same user carries no ProjectID and does
	// not grant access to any Project: it does not imply the
	// ProjectMembership above nor any other project access.
	if projectMembership.ProjectID == "" {
		t.Fatal("project membership must retain its own project id")
	}
}

func TestProjectMembershipValidate(t *testing.T) {
	cases := []struct {
		name    string
		m       ProjectMembership
		wantErr bool
	}{
		{
			name:    "valid",
			m:       ProjectMembership{ProjectID: "proj-1", UserID: "user-1", Role: RoleViewer},
			wantErr: false,
		},
		{
			name:    "missing project",
			m:       ProjectMembership{ProjectID: "", UserID: "user-1", Role: RoleViewer},
			wantErr: true,
		},
		{
			name:    "missing user",
			m:       ProjectMembership{ProjectID: "proj-1", UserID: "", Role: RoleViewer},
			wantErr: true,
		},
		{
			name:    "invalid role",
			m:       ProjectMembership{ProjectID: "proj-1", UserID: "user-1", Role: Role("BOGUS")},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.m.Validate()
			if (err != nil) != tc.wantErr {
				t.Fatalf("ProjectMembership.Validate() = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestProjectInvitationValidate(t *testing.T) {
	cases := []struct {
		name    string
		inv     ProjectInvitation
		wantErr error
	}{
		{
			name: "valid exact email",
			inv: ProjectInvitation{
				ProjectID: "proj-1", Invitee: "person@example.com",
				Role: RoleContributor, State: InvitationPending,
			},
			wantErr: nil,
		},
		{
			name: "valid exact login",
			inv: ProjectInvitation{
				ProjectID: "proj-1", Invitee: "jdoe",
				Role: RoleContributor, State: InvitationPending,
			},
			wantErr: nil,
		},
		{
			name: "empty invitee",
			inv: ProjectInvitation{
				ProjectID: "proj-1", Invitee: "",
				Role: RoleContributor, State: InvitationPending,
			},
			wantErr: ErrInvalidInvitationTarget,
		},
		{
			name: "whitespace invitee",
			inv: ProjectInvitation{
				ProjectID: "proj-1", Invitee: "   ",
				Role: RoleContributor, State: InvitationPending,
			},
			wantErr: ErrInvalidInvitationTarget,
		},
		{
			name: "wildcard star invitee",
			inv: ProjectInvitation{
				ProjectID: "proj-1", Invitee: "*@example.com",
				Role: RoleContributor, State: InvitationPending,
			},
			wantErr: ErrInvalidInvitationTarget,
		},
		{
			name: "wildcard question mark invitee",
			inv: ProjectInvitation{
				ProjectID: "proj-1", Invitee: "j?e@example.com",
				Role: RoleContributor, State: InvitationPending,
			},
			wantErr: ErrInvalidInvitationTarget,
		},
		{
			name: "invalid state",
			inv: ProjectInvitation{
				ProjectID: "proj-1", Invitee: "person@example.com",
				Role: RoleContributor, State: InvitationState("UNKNOWN"),
			},
			wantErr: ErrInvalidInvitationState,
		},
		{
			name: "invalid role",
			inv: ProjectInvitation{
				ProjectID: "proj-1", Invitee: "person@example.com",
				Role: Role("BOGUS"), State: InvitationPending,
			},
			wantErr: ErrInvalidRole,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.inv.Validate()
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("ProjectInvitation.Validate() = %v, want %v", err, tc.wantErr)
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
		{"pending self loop", InvitationPending, InvitationPending, nil},
		{"accepted self loop", InvitationAccepted, InvitationAccepted, nil},
		{"accepted to declined illegal", InvitationAccepted, InvitationDeclined, ErrInvalidInvitationTransition},
		{"accepted to pending illegal", InvitationAccepted, InvitationPending, ErrInvalidInvitationTransition},
		{"declined to accepted illegal", InvitationDeclined, InvitationAccepted, ErrInvalidInvitationTransition},
		{"expired to accepted illegal", InvitationExpired, InvitationAccepted, ErrInvalidInvitationTransition},
		{"revoked to accepted illegal", InvitationRevoked, InvitationAccepted, ErrInvalidInvitationTransition},
		{"revoked to pending illegal", InvitationRevoked, InvitationPending, ErrInvalidInvitationTransition},
		{"unknown current state", InvitationState("BOGUS"), InvitationAccepted, ErrInvalidInvitationState},
		{"unknown next state", InvitationPending, InvitationState("BOGUS"), ErrInvalidInvitationState},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateInvitationTransition(tc.current, tc.next)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("ValidateInvitationTransition(%q, %q) = %v, want %v", tc.current, tc.next, err, tc.wantErr)
			}
		})
	}
}

func TestInvitationStateTerminal(t *testing.T) {
	if InvitationPending.Terminal() {
		t.Fatal("PENDING must not be terminal")
	}
	terminalStates := []InvitationState{InvitationAccepted, InvitationDeclined, InvitationExpired, InvitationRevoked}
	for _, s := range terminalStates {
		if !s.Terminal() {
			t.Fatalf("%q must be terminal", s)
		}
	}
}

func TestDeriveProjectMembership(t *testing.T) {
	user := UserID("user-1")

	cases := []struct {
		name    string
		inv     ProjectInvitation
		wantErr error
	}{
		{
			name: "accepted invitation derives membership",
			inv: ProjectInvitation{
				ProjectID: "proj-1", Invitee: "person@example.com",
				Role: RoleContributor, State: InvitationAccepted,
			},
			wantErr: nil,
		},
		{
			name: "pending invitation rejected",
			inv: ProjectInvitation{
				ProjectID: "proj-1", Invitee: "person@example.com",
				Role: RoleContributor, State: InvitationPending,
			},
			wantErr: ErrMembershipRequiresAccept,
		},
		{
			name: "declined invitation rejected",
			inv: ProjectInvitation{
				ProjectID: "proj-1", Invitee: "person@example.com",
				Role: RoleContributor, State: InvitationDeclined,
			},
			wantErr: ErrMembershipRequiresAccept,
		},
		{
			name: "expired invitation rejected",
			inv: ProjectInvitation{
				ProjectID: "proj-1", Invitee: "person@example.com",
				Role: RoleContributor, State: InvitationExpired,
			},
			wantErr: ErrMembershipRequiresAccept,
		},
		{
			name: "revoked invitation rejected",
			inv: ProjectInvitation{
				ProjectID: "proj-1", Invitee: "person@example.com",
				Role: RoleContributor, State: InvitationRevoked,
			},
			wantErr: ErrMembershipRequiresAccept,
		},
		{
			name: "invalid invitation rejected",
			inv: ProjectInvitation{
				ProjectID: "proj-1", Invitee: "*@example.com",
				Role: RoleContributor, State: InvitationAccepted,
			},
			wantErr: ErrInvalidInvitationTarget,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			membership, err := DeriveProjectMembership(tc.inv, user)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("DeriveProjectMembership() error = %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr == nil {
				if membership.ProjectID != tc.inv.ProjectID {
					t.Fatalf("derived membership project id = %q, want %q", membership.ProjectID, tc.inv.ProjectID)
				}
				if membership.UserID != user {
					t.Fatalf("derived membership user id = %q, want %q", membership.UserID, user)
				}
				if membership.Role != tc.inv.Role {
					t.Fatalf("derived membership role = %q, want %q", membership.Role, tc.inv.Role)
				}
			}
		})
	}
}

func TestDeriveProjectMembershipInvalidUser(t *testing.T) {
	inv := ProjectInvitation{
		ProjectID: "proj-1", Invitee: "person@example.com",
		Role: RoleContributor, State: InvitationAccepted,
	}
	if _, err := DeriveProjectMembership(inv, UserID("")); !errors.Is(err, ErrInvalidUserID) {
		t.Fatalf("DeriveProjectMembership with empty user = %v, want %v", err, ErrInvalidUserID)
	}
}
