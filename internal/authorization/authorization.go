// Package authorization defines the transport-independent Public API
// authorization domain contract required to represent an authorization
// denial on the Public API error boundary (docs/API_ERRORS.md,
// docs/API_RATE_LIMIT.md). Release A already delivers
// `internal/identity.Role`/`ProjectMembership` (SPEC §3.2) and the
// `internal/apierror` structured-error contract with a closed `Code` set;
// this package closes the remaining gap by defining *what* is being
// authorized (Capability) and *how* an authorization decision is
// represented (Decision) — as a pure Go domain contract with no HTTP
// framework, no persistence, and no queue/VM/runner dependency.
//
// Evaluate is a pure function: it computes a Decision deterministically
// from a hard-coded Role→Capability matrix only (no clock/global/singleton
// state, no goroutines, no storage), mirroring apiratelimit.Evaluate.
// HTTP/middleware wiring, per-endpoint capability configuration,
// dynamic/persisted policy storage, SUPER_ADMIN global-role handling, and
// enforcement inside apicommand/apiquery handlers are later Release A/B
// slices layered on top of this contract.
package authorization

import (
	"errors"
	"fmt"
	"strings"

	"github.com/ControlCenterSoft/aidi_2.0/internal/identity"
)

// Capability is a named permission a Role may or may not hold.
type Capability string

const (
	// CapabilityProjectRead grants read access to a Project.
	CapabilityProjectRead Capability = "project.read"
	// CapabilityProjectMutate grants the ability to change Project state
	// (e.g. Specification/Requirement/Release/Feature/Task content).
	CapabilityProjectMutate Capability = "project.mutate"
)

// ErrInvalidCapability is returned (wrapped) when a Capability is empty or
// not part of the closed set this package understands.
var ErrInvalidCapability = errors.New("authorization: invalid capability")

// ErrInvalidDecision is returned (wrapped) by Decision.Validate when a
// denied Decision does not carry a Reason.
var ErrInvalidDecision = errors.New("authorization: invalid decision")

// Decision is the outcome of evaluating whether a Role holds a Capability.
// Reason is mandatory when Allowed is false: a denial must always be
// explainable (SPEC §2.4).
type Decision struct {
	Allowed bool   `json:"allowed"`
	Reason  string `json:"reason,omitempty"`
}

// Validate rejects a denied Decision (Allowed == false) carrying an empty
// Reason. Callers that map a Decision onto another representation (e.g.
// apierror.FromAuthorizationDecision) must call Validate so a manually
// constructed, malformed Decision cannot be serialized downstream.
func (d Decision) Validate() error {
	if !d.Allowed && strings.TrimSpace(d.Reason) == "" {
		return fmt.Errorf("%w: reason is required when denied", ErrInvalidDecision)
	}
	return nil
}

// capabilityMatrix is the deterministic, hard-coded Role→Capability
// authorization matrix derived from SPEC §3.2 role semantics:
//
//   - Every project role can read a Project it is a member of.
//   - PROJECT_OWNER, PROJECT_MANAGER, PRODUCT_OWNER and CONTRIBUTOR can
//     mutate Project state (they own/drive the Requirements/Release/
//     Feature/Task lifecycle, SPEC §5.1).
//   - VIEWER and CLIENT are read-only: VIEWER is an internal observer role
//     and CLIENT is an external, contract-scoped account (SPEC §3.2); the
//     mutate capability is explicitly not granted to either.
//
// SUPER_ADMIN is a global system role (SPEC §3.2), not a project Role, and
// is intentionally out of scope for this pure per-project contract.
var capabilityMatrix = map[identity.Role]map[Capability]bool{
	identity.RoleProjectOwner: {
		CapabilityProjectRead:   true,
		CapabilityProjectMutate: true,
	},
	identity.RoleProjectManager: {
		CapabilityProjectRead:   true,
		CapabilityProjectMutate: true,
	},
	identity.RoleProductOwner: {
		CapabilityProjectRead:   true,
		CapabilityProjectMutate: true,
	},
	identity.RoleContributor: {
		CapabilityProjectRead:   true,
		CapabilityProjectMutate: true,
	},
	identity.RoleViewer: {
		CapabilityProjectRead:   true,
		CapabilityProjectMutate: false,
	},
	identity.RoleClient: {
		CapabilityProjectRead:   true,
		CapabilityProjectMutate: false,
	},
}

// knownCapabilities is the closed set of Capability values this package
// understands, derived from capabilityMatrix so it can never drift out of
// sync with the roles it is defined for.
var knownCapabilities = func() map[Capability]struct{} {
	set := map[Capability]struct{}{}
	for _, caps := range capabilityMatrix {
		for capability := range caps {
			set[capability] = struct{}{}
		}
	}
	return set
}()

// Evaluate computes a Decision deterministically for a given role and
// capability: identical (role, capability) inputs always produce an
// identical Decision. It is a pure function: it reads no clock, global, or
// singleton state, and performs no I/O.
//
// An unknown/invalid Role or an empty/unknown Capability returns a
// wrapped sentinel error (identity.ErrInvalidRole or
// ErrInvalidCapability), never a panic.
func Evaluate(role identity.Role, capability Capability) (Decision, error) {
	if err := role.Validate(); err != nil {
		return Decision{}, err
	}
	if strings.TrimSpace(string(capability)) == "" {
		return Decision{}, fmt.Errorf("%w: capability is required", ErrInvalidCapability)
	}
	if _, ok := knownCapabilities[capability]; !ok {
		return Decision{}, fmt.Errorf("%w: unknown capability %q", ErrInvalidCapability, capability)
	}

	if capabilityMatrix[role][capability] {
		return Decision{Allowed: true}, nil
	}
	return Decision{
		Allowed: false,
		Reason:  fmt.Sprintf("role %q does not have capability %q", role, capability),
	}, nil
}
