// Package identity provides domain-only contracts for the canonical
// Workspace/Project/Membership/ProjectInvitation entities defined by SPEC
// §3.1-3.2 and §4.1. It contains pure types, invariants and
// state-transition validation only: no HTTP/API, no persistence and no
// external identity provider (AD/LDAP/OIDC) integration.
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

var (
	ErrInvalidWorkspaceID          = errors.New("invalid workspace id")
	ErrInvalidProjectID            = errors.New("invalid project id")
	ErrInvalidUserID               = errors.New("invalid user id")
	ErrInvalidRole                 = errors.New("invalid role")
	ErrInvalidInvitationTarget     = errors.New("invitation must address an exact login/email identifier")
	ErrInvalidInvitationState      = errors.New("invalid invitation state")
	ErrInvalidInvitationTransition = errors.New("invalid invitation lifecycle transition")
	ErrMembershipRequiresAccept    = errors.New("project membership cannot be derived from an invitation that is not ACCEPTED")
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

// Role enumerates the project roles defined by SPEC §3.2. Roles are
// project-scoped; there is no implied workspace-level role in this
// package.
type Role string

const (
	RoleProjectOwner   Role = "PROJECT_OWNER"
	RoleProductOwner   Role = "PRODUCT_OWNER"
	RoleContributor    Role = "CONTRIBUTOR"
	RoleViewer         Role = "VIEWER"
	RoleClient         Role = "CLIENT"
	RoleProjectManager Role = "PROJECT_MANAGER"
)

func (r Role) Validate() error {
	switch r {
	case RoleProjectOwner, RoleProductOwner, RoleContributor, RoleViewer, RoleClient, RoleProjectManager:
		return nil
	default:
		return ErrInvalidRole
	}
}

// WorkspaceMembership records that a user belongs to a Workspace. It is an
// independent record from ProjectMembership: a valid WorkspaceMembership
// must not be assumed from, or imply, a ProjectMembership for the same
// user (SPEC §3.2 — "Workspace membership и Project membership
// независимы").
type WorkspaceMembership struct {
	WorkspaceID WorkspaceID
	UserID      UserID
}

func (m WorkspaceMembership) Validate() error {
	if err := m.WorkspaceID.Validate(); err != nil {
		return err
	}
	if err := m.UserID.Validate(); err != nil {
		return err
	}
	return nil
}

// ProjectMembership records that a user holds a Role on a Project. It does
// not assume or imply a corresponding WorkspaceMembership for the same
// user (SPEC §3.2).
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
	if err := m.Role.Validate(); err != nil {
		return err
	}
	return nil
}

// InvitationState enumerates the ProjectInvitation lifecycle from SPEC
// §3.2: PENDING → ACCEPTED | DECLINED | EXPIRED | REVOKED. All states
// except PENDING are terminal.
type InvitationState string

const (
	InvitationPending  InvitationState = "PENDING"
	InvitationAccepted InvitationState = "ACCEPTED"
	InvitationDeclined InvitationState = "DECLINED"
	InvitationExpired  InvitationState = "EXPIRED"
	InvitationRevoked  InvitationState = "REVOKED"
)

func (s InvitationState) Validate() error {
	switch s {
	case InvitationPending, InvitationAccepted, InvitationDeclined, InvitationExpired, InvitationRevoked:
		return nil
	default:
		return ErrInvalidInvitationState
	}
}

// Terminal reports whether the state is a terminal outcome of the
// invitation lifecycle. Only PENDING is non-terminal.
func (s InvitationState) Terminal() bool {
	return s != InvitationPending
}

// ProjectInvitation is keyed by an exact login/email identifier. It never
// exposes a wildcard or directory-enumeration target: Invitee must be an
// exact, non-empty login/email string (AC-INV-001, AC-INV-002).
type ProjectInvitation struct {
	ProjectID ProjectID
	Invitee   string
	Role      Role
	State     InvitationState
}

// isExactIdentifier rejects empty, whitespace-only and wildcard/enumeration
// patterns ("*", "?") so that a ProjectInvitation can only ever address a
// single exact login/email identifier.
func isExactIdentifier(value string) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return false
	}
	if strings.ContainsAny(trimmed, "*?") {
		return false
	}
	return true
}

func (i ProjectInvitation) Validate() error {
	if err := i.ProjectID.Validate(); err != nil {
		return err
	}
	if !isExactIdentifier(i.Invitee) {
		return ErrInvalidInvitationTarget
	}
	if err := i.Role.Validate(); err != nil {
		return err
	}
	if err := i.State.Validate(); err != nil {
		return err
	}
	return nil
}

// ValidateInvitationTransition rejects illegal ProjectInvitation state
// transitions. Only PENDING may move to a terminal state; every terminal
// state (ACCEPTED, DECLINED, EXPIRED, REVOKED) is final and cannot
// transition again, including re-accepting a REVOKED invitation.
func ValidateInvitationTransition(current, next InvitationState) error {
	if err := current.Validate(); err != nil {
		return err
	}
	if err := next.Validate(); err != nil {
		return err
	}

	if current == next {
		return nil
	}

	if current == InvitationPending {
		switch next {
		case InvitationAccepted, InvitationDeclined, InvitationExpired, InvitationRevoked:
			return nil
		}
	}

	return ErrInvalidInvitationTransition
}

// DeriveProjectMembership derives a ProjectMembership from an accepted
// ProjectInvitation. A ProjectMembership cannot be derived from an
// invitation that is not ACCEPTED (AC-INV-003).
func DeriveProjectMembership(invitation ProjectInvitation, user UserID) (ProjectMembership, error) {
	if err := invitation.Validate(); err != nil {
		return ProjectMembership{}, err
	}
	if err := user.Validate(); err != nil {
		return ProjectMembership{}, err
	}
	if invitation.State != InvitationAccepted {
		return ProjectMembership{}, ErrMembershipRequiresAccept
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
