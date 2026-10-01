package canonical

import (
	"errors"
	"fmt"
	"strings"
)

// State is the reusable, string-based canonical state primitive: a single
// value drawn from a closed, named set. It is the shared contract that
// TaskState (task.go), ChangeSetState (changeset.go) and
// specification.RequirementStatus each hand-roll independently today —
// duplicating an identical string-state/validate-known-state/
// ErrXInvariant+ErrInvalidXTransition pattern three times. This primitive is
// purely additive: it extracts the common shape without rewiring any
// existing caller, in the same spirit as Kind/ID/ObjectRef in identifier.go.
// Later cards (state/reason separation, a generic transition validator, an
// invariant framework) build on top of it.
type State string

// ErrInvalidState is returned when a State is empty/whitespace-only or does
// not belong to the caller-supplied allowed StateSet, following the
// ErrInvalidKind/ErrInvalidID naming convention.
var ErrInvalidState = errors.New("invalid canonical state")

// StateSet is a closed, named set of allowed State values supplied by the
// caller (e.g. the set of states a particular entity's lifecycle
// recognizes).
type StateSet map[State]struct{}

// NewStateSet builds a StateSet from the given states, suitable for passing
// to State.Validate.
func NewStateSet(states ...State) StateSet {
	set := make(StateSet, len(states))
	for _, s := range states {
		set[s] = struct{}{}
	}
	return set
}

// Contains reports whether state belongs to the set.
func (set StateSet) Contains(state State) bool {
	_, ok := set[state]
	return ok
}

// Validate rejects an empty/whitespace-only State and any State outside the
// supplied allowed StateSet.
func (s State) Validate(allowed StateSet) error {
	if strings.TrimSpace(string(s)) == "" {
		return fmt.Errorf("%w: state is required", ErrInvalidState)
	}
	if !allowed.Contains(s) {
		return fmt.Errorf("%w: unknown state %q", ErrInvalidState, string(s))
	}
	return nil
}
