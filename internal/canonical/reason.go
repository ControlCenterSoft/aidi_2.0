package canonical

import (
	"errors"
	"fmt"
	"strings"
)

// Reason is the reusable, string-based canonical state/reason-separation
// primitive: it records *why* a State holds (e.g. a blocked/failure/
// rejection reason), distinct from the State value itself. It builds
// purely on the existing State/StateSet contract (state.go, card A1-003)
// without modifying any current caller (TaskState, ChangeSetState,
// specification.RequirementStatus). Later cards may rewire those callers
// onto this primitive; this slice only introduces it.
type Reason string

// ErrInvalidReason is returned when a Reason is missing/empty for a State
// that requires one, following the ErrInvalidKind/ErrInvalidID/
// ErrInvalidState naming convention.
var ErrInvalidReason = errors.New("invalid canonical state reason")

// IsEmpty reports whether r is empty or whitespace-only. An empty or
// whitespace-only Reason is treated as "no reason supplied".
func (r Reason) IsEmpty() bool {
	return strings.TrimSpace(string(r)) == ""
}

// ValidateStateReason validates that reason is consistent with state given
// reasonRequired, the closed StateSet of states that require a non-empty
// Reason.
//
// The explicit rule chosen for this slice: a Reason is optional extra
// context for any state. Supplying one for a state that is not in
// reasonRequired is always allowed (not rejected as strict surplus
// context) — only a missing/empty Reason for a state that *is* in
// reasonRequired is an error. Whitespace-only Reason values are treated as
// missing.
//
// state is first validated against allowed (the ErrInvalidState case from
// state.go); only once state itself is valid is the reason requirement
// checked, returning ErrInvalidReason (wrapped via %w for errors.Is
// compatibility) when a required reason is missing.
func ValidateStateReason(state State, reason Reason, allowed StateSet, reasonRequired StateSet) error {
	if err := state.Validate(allowed); err != nil {
		return err
	}
	if reasonRequired.Contains(state) && reason.IsEmpty() {
		return fmt.Errorf("%w: reason is required for state %q", ErrInvalidReason, string(state))
	}
	return nil
}
