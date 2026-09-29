package identity

import (
	"errors"
	"strings"
	"time"
)

// ServiceIdentityID identifies a service/workload identity (SPEC §3.3). It
// is a distinct type from UserID: a service identity can never be
// constructed as, or satisfy validation for, a user or ProjectMembership.
type ServiceIdentityID string

// ServiceKind classifies a ServiceIdentity's principal type. SPEC §3.3
// requires Node Agent, CRM, providers and automation to use dedicated
// service/workload identities rather than user accounts.
type ServiceKind string

const (
	ServiceKindNodeAgent  ServiceKind = "NODE_AGENT"
	ServiceKindCRM        ServiceKind = "CRM"
	ServiceKindProvider   ServiceKind = "PROVIDER"
	ServiceKindAutomation ServiceKind = "AUTOMATION"
)

var validServiceKinds = map[ServiceKind]struct{}{
	ServiceKindNodeAgent:  {},
	ServiceKindCRM:        {},
	ServiceKindProvider:   {},
	ServiceKindAutomation: {},
}

// Scope is an explicit, enumerable least-privilege permission grant for a
// ServiceIdentity (SPEC §3.3, "least privilege обязателен"). It never
// carries wildcard/glob semantics, mirroring the ProjectInvitation exact-
// invitee anti-wildcard pattern: a Scope always names one concrete
// Permission, never a broadcast/blanket grant.
type Scope struct {
	Permission Permission
}

// Permission is an exact, enumerable capability string (for example
// "project:read", "task:write"). Empty, whitespace-padded or wildcard/
// glob-like values are rejected.
type Permission string

var (
	// ErrInvalidServiceIdentityID is returned when a ServiceIdentityID is
	// empty/blank.
	ErrInvalidServiceIdentityID = errors.New("identity: invalid service identity id")
	// ErrInvalidServiceKind is returned when a ServiceKind is not one of
	// the SPEC §3.3 service/workload kinds.
	ErrInvalidServiceKind = errors.New("identity: invalid service kind")
	// ErrInvalidScope is returned when a Scope's Permission is empty,
	// whitespace-padded, or wildcard/glob-like.
	ErrInvalidScope = errors.New("identity: scope must be an exact, non-wildcard permission")
	// ErrNoScopes is returned when a ServiceIdentity carries zero scopes.
	ErrNoScopes = errors.New("identity: service identity requires at least one explicit scope")
	// ErrDuplicateScope is returned when a ServiceIdentity lists the same
	// Scope more than once.
	ErrDuplicateScope = errors.New("identity: duplicate scope")
	// ErrInvalidCredentialLifetime is returned when a ServiceCredential's
	// IssuedAt/ExpiresAt are zero or do not describe a strictly positive,
	// bounded lifetime (ExpiresAt must be after IssuedAt).
	ErrInvalidCredentialLifetime = errors.New("identity: service credential must have a strictly bounded lifetime (issued before expiry)")
)

func (id ServiceIdentityID) Validate() error {
	if strings.TrimSpace(string(id)) == "" {
		return ErrInvalidServiceIdentityID
	}
	return nil
}

func (k ServiceKind) Validate() error {
	if _, ok := validServiceKinds[k]; !ok {
		return ErrInvalidServiceKind
	}
	return nil
}

// Validate rejects empty, whitespace-padded and wildcard/glob-like
// permissions so a Scope can never express a blanket grant.
func (s Scope) Validate() error {
	trimmed := strings.TrimSpace(string(s.Permission))
	if trimmed == "" || trimmed != string(s.Permission) {
		return ErrInvalidScope
	}
	if strings.ContainsAny(trimmed, "*?%") {
		return ErrInvalidScope
	}
	if strings.Contains(trimmed, " ") {
		return ErrInvalidScope
	}
	return nil
}

// ServiceCredential carries the validity window of a ServiceIdentity's
// credential. SPEC §3.3 requires workload credentials to be short-lived
// where possible; Validate encodes the checkable minimum invariant: a
// strictly bounded, non-zero lifetime (IssuedAt < ExpiresAt).
type ServiceCredential struct {
	IssuedAt  time.Time
	ExpiresAt time.Time
}

func (c ServiceCredential) Validate() error {
	if c.IssuedAt.IsZero() || c.ExpiresAt.IsZero() {
		return ErrInvalidCredentialLifetime
	}
	if !c.ExpiresAt.After(c.IssuedAt) {
		return ErrInvalidCredentialLifetime
	}
	return nil
}

// ServiceIdentity is a non-user, workload/automation principal (SPEC §3.3):
// Node Agent, CRM, providers and automation use these instead of user
// accounts. It is structurally separate from UserID/Role/ProjectMembership
// so a ServiceIdentity can never be constructed as, or satisfy validation
// for, a user identity or project membership.
type ServiceIdentity struct {
	ID         ServiceIdentityID
	Kind       ServiceKind
	Scopes     []Scope
	Credential ServiceCredential
}

// Validate enforces: a valid ID, a valid/known ServiceKind, at least one
// explicit scope, no duplicate scopes, and a valid, strictly bounded,
// non-expired-at-issuance credential.
func (s ServiceIdentity) Validate() error {
	if err := s.ID.Validate(); err != nil {
		return err
	}
	if err := s.Kind.Validate(); err != nil {
		return err
	}
	if len(s.Scopes) == 0 {
		return ErrNoScopes
	}

	seen := make(map[Permission]struct{}, len(s.Scopes))
	for _, scope := range s.Scopes {
		if err := scope.Validate(); err != nil {
			return err
		}
		if _, dup := seen[scope.Permission]; dup {
			return ErrDuplicateScope
		}
		seen[scope.Permission] = struct{}{}
	}

	return s.Credential.Validate()
}
