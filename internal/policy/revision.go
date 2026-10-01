package policy

import (
	"errors"
	"fmt"

	"github.com/ControlCenterSoft/aidi_2.0/internal/canonical"
)

// PolicyRevision pairs an immutable Policy snapshot with a
// canonical.Revision, mirroring the optimistic-concurrency boundary
// already used by internal/repository (A2-003). It is a pure domain
// contract only: no persistence, no HTTP, no queue/VM/runner wiring.
type PolicyRevision struct {
	Policy   Policy             `json:"policy"`
	Revision canonical.Revision `json:"revision"`
}

// ErrInvalidPolicyRevision is returned (wrapped) when a PolicyRevision
// fails validation: an invalid underlying Policy, or a zero/invalid
// Revision.
var ErrInvalidPolicyRevision = errors.New("policy: invalid policy revision")

// Validate rejects an invalid underlying Policy and a zero Revision, each
// via the wrapped sentinel error ErrInvalidPolicyRevision. It never
// panics.
func (pr PolicyRevision) Validate() error {
	if err := pr.Policy.Validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidPolicyRevision, err)
	}
	if pr.Revision == 0 {
		return fmt.Errorf("%w: revision is required", ErrInvalidPolicyRevision)
	}
	return nil
}

// ErrPolicyIDMismatch is returned (wrapped) by NextPolicyRevision when
// next.ID does not match current.Policy.ID.
var ErrPolicyIDMismatch = errors.New("policy: policy id mismatch")

// NextPolicyRevision validates next, requires next.ID == current.Policy.ID,
// enforces optimistic concurrency via
// canonical.CheckExpectedRevision(expected, current.Revision), and advances
// the revision via canonical.NextRevision. A stale expected revision is
// rejected (via the wrapped canonical.ErrRevisionConflict /
// *canonical.RevisionConflictError) without mutating current. The first
// revision of a given PolicyID starts at expected = 0 and advances to
// revision 1; subsequent successful calls advance N -> N+1.
//
// NextPolicyRevision is a pure function: no I/O, no clock, no
// global/singleton state — identical inputs always produce an identical
// result.
func NextPolicyRevision(current PolicyRevision, next Policy, expected canonical.Revision) (PolicyRevision, error) {
	if err := next.Validate(); err != nil {
		return PolicyRevision{}, err
	}
	if next.ID != current.Policy.ID {
		return PolicyRevision{}, fmt.Errorf("%w: current %q, next %q", ErrPolicyIDMismatch, current.Policy.ID, next.ID)
	}
	if err := canonical.CheckExpectedRevision(expected, current.Revision); err != nil {
		return PolicyRevision{}, err
	}

	rev, err := canonical.NextRevision(current.Revision)
	if err != nil {
		return PolicyRevision{}, err
	}

	return PolicyRevision{Policy: next, Revision: rev}, nil
}
