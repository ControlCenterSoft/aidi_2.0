package canonical

import (
	"errors"
	"testing"
)

func TestValidateTransitionAllowsDeclaredEdge(t *testing.T) {
	graph := NewTransitionGraph(map[State]StateSet{
		"PENDING": NewStateSet("READY"),
		"READY":   NewStateSet("RUNNING"),
		"RUNNING": NewStateSet("DONE", "FAILED"),
		"DONE":    NewStateSet(),
		"FAILED":  NewStateSet(),
	})

	if err := ValidateStateTransition("PENDING", "READY", graph); err != nil {
		t.Fatalf("expected allowed edge PENDING -> READY, got %v", err)
	}
}

func TestValidateTransitionRejectsDisallowedEdge(t *testing.T) {
	graph := NewTransitionGraph(map[State]StateSet{
		"PENDING": NewStateSet("READY"),
		"READY":   NewStateSet("RUNNING"),
	})

	err := ValidateStateTransition("PENDING", "RUNNING", graph)
	if err == nil {
		t.Fatal("expected error for disallowed edge PENDING -> RUNNING")
	}
	if !errors.Is(err, ErrInvalidStateTransition) {
		t.Fatalf("expected ErrInvalidStateTransition, got %v", err)
	}
}

func TestValidateTransitionRejectsAllOutgoingEdgesFromTerminalState(t *testing.T) {
	graph := NewTransitionGraph(map[State]StateSet{
		"RUNNING": NewStateSet("DONE"),
		"DONE":    NewStateSet(), // terminal: no outgoing edges at all.
	})

	// Same-state no-op out of a terminal state is rejected too.
	err := ValidateStateTransition("DONE", "DONE", graph)
	if err == nil {
		t.Fatal("expected error for same-state edge out of terminal state DONE")
	}
	if !errors.Is(err, ErrInvalidStateTransition) {
		t.Fatalf("expected ErrInvalidStateTransition, got %v", err)
	}

	// Any other outgoing edge out of DONE is rejected too.
	err = ValidateStateTransition("DONE", "RUNNING", graph)
	if err == nil {
		t.Fatal("expected error for DONE -> RUNNING")
	}
	if !errors.Is(err, ErrInvalidStateTransition) {
		t.Fatalf("expected ErrInvalidStateTransition, got %v", err)
	}
}

func TestValidateTransitionAllowsDeclaredSameStateNoOp(t *testing.T) {
	graph := NewTransitionGraph(map[State]StateSet{
		// A non-terminal state that explicitly declares a same-state
		// no-op edge is allowed to stay put.
		"BLOCKED": NewStateSet("BLOCKED", "READY"),
		"READY":   NewStateSet("RUNNING"),
	})

	if err := ValidateStateTransition("BLOCKED", "BLOCKED", graph); err != nil {
		t.Fatalf("expected declared same-state no-op to succeed, got %v", err)
	}
}

func TestValidateTransitionRejectsEdgeFromStateOutsideGraph(t *testing.T) {
	graph := NewTransitionGraph(map[State]StateSet{
		"PENDING": NewStateSet("READY"),
	})

	err := ValidateStateTransition("BOGUS", "READY", graph)
	if err == nil {
		t.Fatal("expected error for current state outside the declared graph")
	}
	if !errors.Is(err, ErrInvalidStateTransition) {
		t.Fatalf("expected ErrInvalidStateTransition, got %v", err)
	}
}

func TestValidateTransitionRejectsEdgeToStateOutsideGraph(t *testing.T) {
	graph := NewTransitionGraph(map[State]StateSet{
		"PENDING": NewStateSet("READY"),
	})

	err := ValidateStateTransition("PENDING", "BOGUS", graph)
	if err == nil {
		t.Fatal("expected error for next state outside the declared graph")
	}
	if !errors.Is(err, ErrInvalidStateTransition) {
		t.Fatalf("expected ErrInvalidStateTransition, got %v", err)
	}
}

func TestValidateTransitionComposesWithStateValidate(t *testing.T) {
	// The transition graph's declared states double as the allowed
	// StateSet for State.Validate, demonstrating the two primitives
	// compose rather than duplicate each other's closed-set logic.
	allowed := NewStateSet("PENDING", "READY", "RUNNING", "DONE", "FAILED")
	graph := NewTransitionGraph(map[State]StateSet{
		"PENDING": NewStateSet("READY"),
		"READY":   NewStateSet("RUNNING"),
		"RUNNING": NewStateSet("DONE", "FAILED"),
		"DONE":    NewStateSet(),
		"FAILED":  NewStateSet(),
	})

	current := State("READY")
	next := State("RUNNING")

	if err := current.Validate(allowed); err != nil {
		t.Fatalf("expected current state to be valid, got %v", err)
	}
	if err := next.Validate(allowed); err != nil {
		t.Fatalf("expected next state to be valid, got %v", err)
	}
	if err := ValidateStateTransition(current, next, graph); err != nil {
		t.Fatalf("expected allowed transition READY -> RUNNING, got %v", err)
	}
}
