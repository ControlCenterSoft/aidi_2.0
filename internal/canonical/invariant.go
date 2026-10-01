package canonical

import (
	"errors"
	"fmt"
)

// Rule is a single named invariant check against a value of type T: a
// Check function plus a stable Name/label used for diagnostics when the
// rule fails. It is the generic extraction of the shape every existing
// entity (Task in task.go, ChangeSet in changeset.go) hand-rolls today as
// a bespoke ErrXInvariant sentinel plus ad-hoc per-field checks mixed
// together inside Validate().
type Rule[T any] struct {
	// Name identifies the rule for diagnostics. It is included verbatim
	// in the error returned by RuleSet.Evaluate when Check fails.
	Name string
	// Check reports whether value satisfies the rule. A nil error means
	// the rule is satisfied.
	Check func(value T) error
}

// RuleSet is a closed, ordered collection of Rules evaluated against a
// single value of type T, mirroring the StateSet/TransitionGraph
// constructor convention (state.go, transition.go).
type RuleSet[T any] []Rule[T]

// NewRuleSet builds a RuleSet from the supplied rules, preserving the
// given order. Evaluation order is significant: see RuleSet.Evaluate.
func NewRuleSet[T any](rules ...Rule[T]) RuleSet[T] {
	set := make(RuleSet[T], len(rules))
	copy(set, rules)
	return set
}

// ErrInvariantViolation is the single framework-level sentinel every
// RuleSet.Evaluate failure wraps (via %w, so errors.Is(err,
// ErrInvariantViolation) always succeeds), regardless of which Rule
// failed or whether an entity also keeps its own, more specific sentinel
// (e.g. ErrTaskInvariant, ErrChangeSetInvariant) wrapped alongside it.
var ErrInvariantViolation = errors.New("invariant violation")

// Evaluate runs the Rules in rs, in order, against value.
//
// Contract: rules are evaluated strictly in slice order; evaluation stops
// at, and Evaluate returns, the error of the first Rule whose Check
// returns a non-nil error. Subsequent rules are not run. The returned
// error wraps ErrInvariantViolation (via %w, so errors.Is succeeds) and
// names the failing Rule, and also wraps the Rule's own Check error (via
// %w, so errors.Is against that error, or any sentinel it itself wraps,
// still succeeds).
//
// An empty RuleSet, or a RuleSet all of whose Rules pass, is a no-op:
// Evaluate returns nil. A Rule with a nil Check is treated as always
// passing.
func (rs RuleSet[T]) Evaluate(value T) error {
	for _, rule := range rs {
		if rule.Check == nil {
			continue
		}
		if err := rule.Check(value); err != nil {
			name := rule.Name
			if name == "" {
				name = "<unnamed rule>"
			}
			return fmt.Errorf("%w: rule %q: %w", ErrInvariantViolation, name, err)
		}
	}
	return nil
}

// KnownStateRule builds a Rule[T] expressing the "unknown state
// rejection" pattern duplicated today as Task.validateKnownState and
// ChangeSet.validateKnownState: stateOf extracts the candidate State from
// value, which must belong to allowed.
func KnownStateRule[T any](name string, stateOf func(T) State, allowed StateSet) Rule[T] {
	return Rule[T]{
		Name: name,
		Check: func(value T) error {
			state := stateOf(value)
			if !allowed.Contains(state) {
				return fmt.Errorf("unknown state %q", string(state))
			}
			return nil
		},
	}
}

// RequiredForStateRule builds a Rule[T] expressing the "field required
// only for a specific state" pattern duplicated today as Task's
// RUNNING-requires-Attempt/DONE-requires-Evidence checks and ChangeSet's
// analogous evidence check — generalized to an arbitrary field via
// present, and generalized beyond Reason (unlike ValidateStateReason in
// reason.go, which is Reason-specific).
//
// stateOf extracts the candidate State from value. present reports
// whether the field under test is populated on value. Whenever stateOf(
// value) belongs to requiredFor, present(value) must be true; otherwise
// the Rule fails, naming fieldLabel and the offending state.
func RequiredForStateRule[T any](name string, stateOf func(T) State, requiredFor StateSet, present func(T) bool, fieldLabel string) Rule[T] {
	return Rule[T]{
		Name: name,
		Check: func(value T) error {
			state := stateOf(value)
			if requiredFor.Contains(state) && !present(value) {
				return fmt.Errorf("%s is required for state %q", fieldLabel, string(state))
			}
			return nil
		},
	}
}
