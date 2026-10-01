package canonical

import (
	"errors"
	"fmt"
)

// TransitionGraph is a closed, generic "(current State) -> (next State)"
// edge contract, built purely on State/StateSet (state.go, card A1-003). It
// extracts the shared shape that TaskState.ValidateTransition (task.go),
// ChangeSetState (changeset.go), specification.RequirementStatus,
// identity.InvitationState, and identity.LocalAdminAccountState each
// hand-roll independently today as an identical `switch current { case ...
// }` edge check plus a bespoke ErrInvalidXTransition sentinel.
//
// TransitionGraph maps each current State to the closed StateSet of next
// States it may transition to. A State present in the graph with an empty
// allowed-next StateSet (e.g. via NewStateSet() with no arguments) is a
// terminal state: no outgoing edge is allowed, not even a same-state
// no-op, unless the graph explicitly declares that same-state edge. A
// State absent from the graph entirely (on either side of the edge) is
// outside the declared graph and every edge involving it is rejected.
type TransitionGraph map[State]StateSet

// NewTransitionGraph builds a TransitionGraph from the supplied edges,
// mirroring the NewStateSet constructor convention. edges maps each
// current State to the StateSet of next States it may reach; a State
// mapped to an empty StateSet (NewStateSet() with no arguments) declares
// that State terminal.
func NewTransitionGraph(edges map[State]StateSet) TransitionGraph {
	graph := make(TransitionGraph, len(edges))
	for current, next := range edges {
		graph[current] = next
	}
	return graph
}

// ErrInvalidStateTransition is returned when an edge from current to next
// is not an allowed edge of the supplied TransitionGraph — including edges
// out of a declared terminal state and edges into/out of a State outside
// the declared graph — following the
// ErrInvalidKind/ErrInvalidID/ErrInvalidState/ErrInvalidReason naming
// convention.
var ErrInvalidStateTransition = errors.New("invalid canonical state transition")

// ValidateStateTransition rejects any (current, next) edge that is not
// present in graph. current must be a key of graph whose allowed-next
// StateSet contains next; otherwise ErrInvalidStateTransition is returned
// (wrapped with %w so errors.Is succeeds), naming both current and next.
//
// Named ValidateStateTransition (rather than ValidateTransition) to avoid
// colliding with the existing, unrelated ChangeSetState-specific
// ValidateTransition(current, next ChangeSetState) in changeset.go, while
// still following the ValidateStateReason naming convention from
// reason.go.
func ValidateStateTransition(current, next State, graph TransitionGraph) error {
	allowedNext, ok := graph[current]
	if !ok || !allowedNext.Contains(next) {
		return fmt.Errorf("%w: %s -> %s", ErrInvalidStateTransition, current, next)
	}
	return nil
}
