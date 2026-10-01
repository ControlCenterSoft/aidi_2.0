package canonical

import (
	"errors"
	"testing"
)

// sampleEntity stands in for the hand-rolled Task/ChangeSet shape used to
// exercise RuleSet against a concrete value type without modifying
// task.go/changeset.go themselves.
type sampleEntity struct {
	state   State
	comment string
}

func TestRuleSetEvaluateEmptySetIsNoOp(t *testing.T) {
	var rs RuleSet[sampleEntity]

	if err := rs.Evaluate(sampleEntity{state: "ANYTHING"}); err != nil {
		t.Fatalf("expected empty RuleSet to be a no-op, got %v", err)
	}
}

func TestRuleSetEvaluateSinglePassingRule(t *testing.T) {
	rs := NewRuleSet(Rule[sampleEntity]{
		Name:  "always passes",
		Check: func(sampleEntity) error { return nil },
	})

	if err := rs.Evaluate(sampleEntity{}); err != nil {
		t.Fatalf("expected single passing rule to succeed, got %v", err)
	}
}

func TestRuleSetEvaluateSingleFailingRuleWrapsErrInvariantViolation(t *testing.T) {
	rs := NewRuleSet(Rule[sampleEntity]{
		Name:  "always fails",
		Check: func(sampleEntity) error { return errors.New("boom") },
	})

	err := rs.Evaluate(sampleEntity{})
	if err == nil {
		t.Fatal("expected single failing rule to return an error")
	}
	if !errors.Is(err, ErrInvariantViolation) {
		t.Fatalf("expected ErrInvariantViolation, got %v", err)
	}
}

func TestRuleSetEvaluateReportsFirstFailureInNonFirstPosition(t *testing.T) {
	errSecond := errors.New("second rule failure")
	var calls []string

	rs := NewRuleSet(
		Rule[sampleEntity]{
			Name: "first passes",
			Check: func(sampleEntity) error {
				calls = append(calls, "first")
				return nil
			},
		},
		Rule[sampleEntity]{
			Name: "second fails",
			Check: func(sampleEntity) error {
				calls = append(calls, "second")
				return errSecond
			},
		},
		Rule[sampleEntity]{
			Name: "third would fail too",
			Check: func(sampleEntity) error {
				calls = append(calls, "third")
				return errors.New("third rule failure")
			},
		},
	)

	err := rs.Evaluate(sampleEntity{})
	if err == nil {
		t.Fatal("expected an error from the second rule")
	}
	if !errors.Is(err, ErrInvariantViolation) {
		t.Fatalf("expected ErrInvariantViolation, got %v", err)
	}
	if !errors.Is(err, errSecond) {
		t.Fatalf("expected the wrapped error to be errSecond, got %v", err)
	}
	if len(calls) != 2 || calls[0] != "first" || calls[1] != "second" {
		t.Fatalf("expected evaluation to stop after the first failure, got calls=%v", calls)
	}
}

func TestKnownStateRuleRejectsUnknownState(t *testing.T) {
	// Mirrors the unknown-state rejection pattern duplicated today as
	// Task.validateKnownState and ChangeSet.validateKnownState, without
	// modifying either of those files.
	allowed := NewStateSet("PENDING", "READY", "RUNNING", "DONE", "BLOCKED", "FAILED")
	rule := KnownStateRule("known state", func(e sampleEntity) State { return e.state }, allowed)
	rs := NewRuleSet(rule)

	if err := rs.Evaluate(sampleEntity{state: "RUNNING"}); err != nil {
		t.Fatalf("expected known state RUNNING to pass, got %v", err)
	}

	err := rs.Evaluate(sampleEntity{state: "BOGUS"})
	if err == nil {
		t.Fatal("expected unknown state BOGUS to fail")
	}
	if !errors.Is(err, ErrInvariantViolation) {
		t.Fatalf("expected ErrInvariantViolation, got %v", err)
	}
}

func TestRequiredForStateRuleComposition(t *testing.T) {
	// Mirrors the "field required only for a specific state" pattern
	// duplicated today as Task's DONE-requires-evidence and
	// ChangeSet's analogous evidence check, generalized to an
	// arbitrary field (here: comment) rather than just Reason.
	requiresComment := NewStateSet("BLOCKED", "FAILED")
	rule := RequiredForStateRule(
		"comment required when blocked or failed",
		func(e sampleEntity) State { return e.state },
		requiresComment,
		func(e sampleEntity) bool { return e.comment != "" },
		"comment",
	)
	rs := NewRuleSet(rule)

	// Not in requiresComment: missing comment is fine.
	if err := rs.Evaluate(sampleEntity{state: "READY"}); err != nil {
		t.Fatalf("expected READY without comment to pass, got %v", err)
	}

	// In requiresComment with the field present: passes.
	if err := rs.Evaluate(sampleEntity{state: "BLOCKED", comment: "waiting on review"}); err != nil {
		t.Fatalf("expected BLOCKED with comment to pass, got %v", err)
	}

	// In requiresComment without the field: fails, wrapping
	// ErrInvariantViolation.
	err := rs.Evaluate(sampleEntity{state: "FAILED"})
	if err == nil {
		t.Fatal("expected FAILED without comment to fail")
	}
	if !errors.Is(err, ErrInvariantViolation) {
		t.Fatalf("expected ErrInvariantViolation, got %v", err)
	}
}

func TestRuleWithNilCheckAlwaysPasses(t *testing.T) {
	rs := NewRuleSet(Rule[sampleEntity]{Name: "no-op"})

	if err := rs.Evaluate(sampleEntity{}); err != nil {
		t.Fatalf("expected a rule with a nil Check to be treated as passing, got %v", err)
	}
}
