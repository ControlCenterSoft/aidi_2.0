// Package identity implements domain-only contracts for the canonical
// Workspace/Project/Membership/ProjectInvitation entities described in
// SPEC §3 and §4.1. It contains pure types, invariants and state-transition
// validation — no HTTP/API surface, no persistence and no external identity
// provider integration.
package identity

import (
	"errors"
	"strings"
)

// WorkspaceID identifies a Workspace, an entity independent from Project
// membership (SPEC §3.2).
type WorkspaceID string

// ProjectID identifies a Project.
type ProjectID string

// UserID identifies a canonical Identity/User.
type UserID string

// Role is a project role assignable to a ProjectMembership.
//
// SPEC §3.2 defines the project role set: PROJECT_OWNER, PRODUCT_OWNER,
// CONTRIBUTOR, VIEWER, CLIENT, PROJECT_MANAGER. SUPER_ADMIN is a global
// system role, not a project role, and is intentionally excluded here.
type Role string

const (
	RoleProjectOwner   Role = "PROJECT_OWNER"
	RoleProductOwner   Role = "PRODUCT_OWNER"
	RoleContributor    Role = "CONTRIBUTOR"
	RoleViewer         Role = "VIEWER"
	RoleClient         Role = "CLIENT"
	RoleProjectManager Role = "PROJECT_MANAGER"
)

var validRoles = map[Role]struct{}{
	RoleProjectOwner:   {},
	RoleProductOwner:   {},
	RoleContributor:    {},
	RoleViewer:         {},
	RoleClient:         {},
	RoleProjectManager: {},
}

var (
	// ErrInvalidWorkspaceID is returned when a WorkspaceID is empty/blank.
	ErrInvalidWorkspaceID = errors.New("identity: invalid workspace id")
	// ErrInvalidProjectID is returned when a ProjectID is empty/blank.
	ErrInvalidProjectID = errors.New("identity: invalid project id")
	// ErrInvalidUserID is returned when a UserID is empty/blank.
	ErrInvalidUserID = errors.New("identity: invalid user id")
	// ErrInvalidRole is returned when a Role is not one of the SPEC §3.2
	// project roles.
	ErrInvalidRole = errors.New("identity: invalid role")
	// ErrInvalidInvitee is returned when a ProjectInvitation identifier is
	// not an exact login/email (AC-INV-001, AC-INV-002).
	ErrInvalidInvitee = errors.New("identity: invitation invitee must be an exact login/email")
	// ErrInvalidInvitationState is returned when a ProjectInvitation carries
	// an unrecognized lifecycle state.
	ErrInvalidInvitationState = errors.New("identity: invalid invitation state")
	// ErrInvalidInvitationTransition is returned when a requested invitation
	// state transition is not permitted by the deterministic lifecycle.
	ErrInvalidInvitationTransition = errors.New("identity: invalid invitation state transition")
	// ErrMembershipRequiresAcceptedInvitation is returned when a
	// ProjectMembership is derived from an invitation that is not ACCEPTED
	// (AC-INV-003).
	ErrMembershipRequiresAcceptedInvitation = errors.New("identity: project membership requires an accepted invitation")
	// ErrMembershipInviteeMismatch is returned when the membership user does
	// not match the accepted invitation's invitee.
	ErrMembershipInviteeMismatch = errors.New("identity: membership user does not match invitation invitee")
)

func (id WorkspaceID) Validate() error {
	if strings.TrimSpace(string(id)) == "" {
		return ErrInvalidWorkspaceID
	}
	return nil
}

func (id ProjectID) Validate() error {
	if strings.TrimSpace(string(id)) == "" {
		return ErrInvalidProjectID
	}
	return nil
}

func (id UserID) Validate() error {
	if strings.TrimSpace(string(id)) == "" {
		return ErrInvalidUserID
	}
	return nil
}

func (r Role) Validate() error {
	if _, ok := validRoles[r]; !ok {
		return ErrInvalidRole
	}
	return nil
}

// WorkspaceMembership is a standalone record of a user's participation in a
// Workspace. It intentionally carries no reference to any Project: per
// SPEC §3.2, workspace membership does not imply project access.
type WorkspaceMembership struct {
	WorkspaceID WorkspaceID
	UserID      UserID
}

func (m WorkspaceMembership) Validate() error {
	if err := m.WorkspaceID.Validate(); err != nil {
		return err
	}
	return m.UserID.Validate()
}

// ProjectMembership is a standalone record of a user's role in a Project. It
// must not assume or require a corresponding WorkspaceMembership for the
// same user/workspace: workspace and project membership are independent
// (SPEC §3.2, "Workspace membership и Project membership независимы").
type ProjectMembership struct {
	ProjectID ProjectID
	UserID    UserID
	Role      Role
}

func (m ProjectMembership) Validate() error {
	if err := m.ProjectID.Validate(); err != nil {
		return err
	}
	if err := m.UserID.Validate(); err != nil {
		return err
	}
	return m.Role.Validate()
}

// InvitationState is the lifecycle state of a ProjectInvitation.
type InvitationState string

const (
	InvitationPending  InvitationState = "PENDING"
	InvitationAccepted InvitationState = "ACCEPTED"
	InvitationDeclined InvitationState = "DECLINED"
	InvitationExpired  InvitationState = "EXPIRED"
	InvitationRevoked  InvitationState = "REVOKED"
)

var validInvitationStates = map[InvitationState]struct{}{
	InvitationPending:  {},
	InvitationAccepted: {},
	InvitationDeclined: {},
	InvitationExpired:  {},
	InvitationRevoked:  {},
}

// terminalInvitationStates cannot transition to any other state.
var terminalInvitationStates = map[InvitationState]struct{}{
	InvitationAccepted: {},
	InvitationDeclined: {},
	InvitationExpired:  {},
	InvitationRevoked:  {},
}

func (s InvitationState) Validate() error {
	if _, ok := validInvitationStates[s]; !ok {
		return ErrInvalidInvitationState
	}
	return nil
}

// ProjectInvitation is keyed by an exact login/email identifier. It never
// exposes/derives from directory enumeration or wildcard matching
// (AC-INV-001, AC-INV-002).
type ProjectInvitation struct {
	ProjectID ProjectID
	Invitee   string // exact login or email address.
	Role      Role
	State     InvitationState
}

// Validate checks structural invariants of a ProjectInvitation. It does not
// perform directory lookups; "exact" only means the identifier is a single,
// non-wildcard, non-empty login/email string.
func (i ProjectInvitation) Validate() error {
	if err := i.ProjectID.Validate(); err != nil {
		return err
	}
	if err := validateExactInvitee(i.Invitee); err != nil {
		return err
	}
	if err := i.Role.Validate(); err != nil {
		return err
	}
	return i.State.Validate()
}

// validateExactInvitee rejects empty, whitespace-only and wildcard/glob-like
// identifiers so that invitations cannot be used as a directory enumeration
// or broadcast mechanism (AC-INV-001, AC-INV-002).
func validateExactInvitee(invitee string) error {
	trimmed := strings.TrimSpace(invitee)
	if trimmed == "" || trimmed != invitee {
		return ErrInvalidInvitee
	}
	if strings.ContainsAny(trimmed, "*?%") {
		return ErrInvalidInvitee
	}
	if strings.Contains(trimmed, " ") {
		return ErrInvalidInvitee
	}
	return nil
}

// ValidateInvitationTransition rejects invalid ProjectInvitation lifecycle
// transitions. The only non-terminal state is PENDING; every other state is
// terminal (e.g. an ACCEPTED or REVOKED invitation cannot transition again).
//
// Legal transitions:
//
//	PENDING -> ACCEPTED
//	PENDING -> DECLINED
//	PENDING -> EXPIRED
//	PENDING -> REVOKED
func ValidateInvitationTransition(current, next InvitationState) error {
	if err := current.Validate(); err != nil {
		return err
	}
	if err := next.Validate(); err != nil {
		return err
	}

	if current == next {
		return ErrInvalidInvitationTransition
	}

	if _, terminal := terminalInvitationStates[current]; terminal {
		return ErrInvalidInvitationTransition
	}

	// current == InvitationPending here: any other valid state is a legal
	// target because every non-PENDING state is terminal.
	if current != InvitationPending {
		return ErrInvalidInvitationTransition
	}

	return nil
}

// NewProjectMembershipFromInvitation derives a ProjectMembership from a
// ProjectInvitation. It fails closed unless the invitation is ACCEPTED
// (AC-INV-003) and addressed to the given user's exact login/email.
func NewProjectMembershipFromInvitation(invitation ProjectInvitation, user UserID, invitee string) (ProjectMembership, error) {
	if err := invitation.Validate(); err != nil {
		return ProjectMembership{}, err
	}
	if err := user.Validate(); err != nil {
		return ProjectMembership{}, err
	}
	if invitation.State != InvitationAccepted {
		return ProjectMembership{}, ErrMembershipRequiresAcceptedInvitation
	}
	if err := validateExactInvitee(invitee); err != nil {
		return ProjectMembership{}, err
	}
	if invitation.Invitee != invitee {
		return ProjectMembership{}, ErrMembershipInviteeMismatch
	}

	membership := ProjectMembership{
		ProjectID: invitation.ProjectID,
		UserID:    user,
		Role:      invitation.Role,
	}
	if err := membership.Validate(); err != nil {
		return ProjectMembership{}, err
	}
	return membership, nil
}
