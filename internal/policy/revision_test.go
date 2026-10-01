package policy_test

import (
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/ControlCenterSoft/aidi_2.0/internal/canonical"
	"github.com/ControlCenterSoft/aidi_2.0/internal/policy"
)

func validPolicy(id policy.PolicyID) policy.Policy {
	return policy.Policy{
		ID: id,
		Statements: []policy.Statement{
			{Subject: "user:alice", Resource: "res:1", Action: "read", Effect: policy.EffectAllow},
		},
	}
}

func TestPolicyRevisionValidate(t *testing.T) {
	t.Parallel()

	t.Run("valid", func(t *testing.T) {
		t.Parallel()
		pr := policy.PolicyRevision{Policy: validPolicy("p1"), Revision: 1}
		if err := pr.Validate(); err != nil {
			t.Fatalf("expected valid PolicyRevision, got error: %v", err)
		}
	})

	t.Run("invalid policy", func(t *testing.T) {
		t.Parallel()
		pr := policy.PolicyRevision{Policy: policy.Policy{}, Revision: 1}
		err := pr.Validate()
		if err == nil {
			t.Fatal("expected error for invalid policy")
		}
		if !errors.Is(err, policy.ErrInvalidPolicyRevision) {
			t.Fatalf("expected ErrInvalidPolicyRevision, got %v", err)
		}
	})

	t.Run("zero revision", func(t *testing.T) {
		t.Parallel()
		pr := policy.PolicyRevision{Policy: validPolicy("p1"), Revision: 0}
		err := pr.Validate()
		if err == nil {
			t.Fatal("expected error for zero revision")
		}
		if !errors.Is(err, policy.ErrInvalidPolicyRevision) {
			t.Fatalf("expected ErrInvalidPolicyRevision, got %v", err)
		}
	})
}

func TestNextPolicyRevision_FirstRevision(t *testing.T) {
	t.Parallel()

	p := validPolicy("p1")
	current := policy.PolicyRevision{Policy: p} // zero-value: Revision 0

	got, err := policy.NextPolicyRevision(current, p, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Revision != 1 {
		t.Fatalf("expected revision 1, got %d", got.Revision)
	}
	if got.Policy.ID != p.ID {
		t.Fatalf("expected policy id %q, got %q", p.ID, got.Policy.ID)
	}
}

func TestNextPolicyRevision_SubsequentRevision(t *testing.T) {
	t.Parallel()

	p := validPolicy("p1")
	current := policy.PolicyRevision{Policy: p, Revision: 5}

	next := p
	next.Statements = append(next.Statements, policy.Statement{
		Subject: "user:bob", Resource: "res:2", Action: "write", Effect: policy.EffectDeny,
	})

	got, err := policy.NextPolicyRevision(current, next, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Revision != 6 {
		t.Fatalf("expected revision 6, got %d", got.Revision)
	}
	if len(got.Policy.Statements) != 2 {
		t.Fatalf("expected 2 statements, got %d", len(got.Policy.Statements))
	}
}

func TestNextPolicyRevision_MismatchedPolicyID(t *testing.T) {
	t.Parallel()

	current := policy.PolicyRevision{Policy: validPolicy("p1"), Revision: 1}
	next := validPolicy("p2")

	_, err := policy.NextPolicyRevision(current, next, 1)
	if err == nil {
		t.Fatal("expected error for mismatched policy id")
	}
	if !errors.Is(err, policy.ErrPolicyIDMismatch) {
		t.Fatalf("expected ErrPolicyIDMismatch, got %v", err)
	}
}

func TestNextPolicyRevision_StaleExpectedRevision_NoMutation(t *testing.T) {
	t.Parallel()

	p := validPolicy("p1")
	current := policy.PolicyRevision{Policy: p, Revision: 5}
	currentSnapshot := policy.PolicyRevision{
		Policy:   validPolicy("p1"),
		Revision: current.Revision,
	}

	next := policy.Policy{
		ID: "p1",
		Statements: []policy.Statement{
			{Subject: "user:alice", Resource: "res:1", Action: "write", Effect: policy.EffectAllow},
		},
	}

	_, err := policy.NextPolicyRevision(current, next, 4) // stale: expected 5
	if err == nil {
		t.Fatal("expected error for stale expected revision")
	}

	var conflictErr *canonical.RevisionConflictError
	if !errors.As(err, &conflictErr) {
		t.Fatalf("expected *canonical.RevisionConflictError, got %v (%T)", err, err)
	}
	if !errors.Is(err, canonical.ErrRevisionConflict) {
		t.Fatalf("expected wrapped canonical.ErrRevisionConflict, got %v", err)
	}
	if conflictErr.Expected != 4 || conflictErr.Actual != 5 {
		t.Fatalf("expected Expected=4 Actual=5, got Expected=%d Actual=%d", conflictErr.Expected, conflictErr.Actual)
	}

	if !reflect.DeepEqual(current, currentSnapshot) {
		t.Fatalf("expected current to be unmutated, got %+v want %+v", current, currentSnapshot)
	}
}

func TestNextPolicyRevision_InvalidNextPolicy(t *testing.T) {
	t.Parallel()

	current := policy.PolicyRevision{Policy: validPolicy("p1"), Revision: 1}

	_, err := policy.NextPolicyRevision(current, policy.Policy{ID: "p1"}, 1)
	if err == nil {
		t.Fatal("expected error for invalid next policy")
	}
	if !errors.Is(err, policy.ErrInvalidPolicy) {
		t.Fatalf("expected ErrInvalidPolicy, got %v", err)
	}
}

func TestNextPolicyRevision_RevisionExhausted(t *testing.T) {
	t.Parallel()

	p := validPolicy("p1")
	current := policy.PolicyRevision{Policy: p, Revision: canonical.Revision(math.MaxUint64)}

	_, err := policy.NextPolicyRevision(current, p, current.Revision)
	if !errors.Is(err, canonical.ErrRevisionExhausted) {
		t.Fatalf("expected ErrRevisionExhausted, got %v", err)
	}
}

func TestEvaluateRevision_ReferencesExactRevision(t *testing.T) {
	t.Parallel()

	p := validPolicy("p1")
	revision := policy.PolicyRevision{Policy: p, Revision: 1}

	got, err := policy.EvaluateRevision(revision, "user:alice", "res:1", "read")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.Decision.Allowed {
		t.Fatalf("expected Allowed=true, got false (reason=%q)", got.Decision.Reason)
	}
	if got.PolicyID != p.ID {
		t.Fatalf("expected PolicyID %q, got %q", p.ID, got.PolicyID)
	}
	if got.Revision != revision.Revision {
		t.Fatalf("expected Revision %d, got %d", revision.Revision, got.Revision)
	}
}

func TestEvaluateRevision_SurvivesLaterRevisionChange(t *testing.T) {
	t.Parallel()

	p := validPolicy("p1")
	original := policy.PolicyRevision{Policy: p, Revision: 1}

	// A historical decision made against revision 1...
	historical, err := policy.EvaluateRevision(original, "user:alice", "res:1", "read")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// ...must still reference revision 1 even after the policy advances to
	// revision 2 with a contradictory statement.
	next := policy.Policy{
		ID: "p1",
		Statements: []policy.Statement{
			{Subject: "user:alice", Resource: "res:1", Action: "read", Effect: policy.EffectDeny},
		},
	}
	if _, err := policy.NextPolicyRevision(original, next, 1); err != nil {
		t.Fatalf("unexpected error advancing revision: %v", err)
	}

	if historical.Revision != 1 {
		t.Fatalf("expected historical decision to still reference revision 1, got %d", historical.Revision)
	}
	if !historical.Decision.Allowed {
		t.Fatalf("expected historical decision to remain Allowed=true despite later revision change")
	}
}

func TestEvaluateRevision_DenyDecision(t *testing.T) {
	t.Parallel()

	p := policy.Policy{
		ID: "p1",
		Statements: []policy.Statement{
			{Subject: "user:alice", Resource: "res:1", Action: "read", Effect: policy.EffectDeny},
		},
	}
	revision := policy.PolicyRevision{Policy: p, Revision: 3}

	got, err := policy.EvaluateRevision(revision, "user:alice", "res:1", "read")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Decision.Allowed {
		t.Fatalf("expected Allowed=false, got true")
	}
	if strings.TrimSpace(got.Decision.Reason) == "" {
		t.Fatalf("expected non-empty Reason for deny decision")
	}
	if got.Revision != 3 {
		t.Fatalf("expected Revision 3, got %d", got.Revision)
	}
}

func TestEvaluateRevision_InvalidRevision(t *testing.T) {
	t.Parallel()

	revision := policy.PolicyRevision{Policy: validPolicy("p1"), Revision: 0}
	_, err := policy.EvaluateRevision(revision, "user:alice", "res:1", "read")
	if err == nil {
		t.Fatal("expected error for zero revision")
	}
	if !errors.Is(err, policy.ErrInvalidPolicyRevision) {
		t.Fatalf("expected ErrInvalidPolicyRevision, got %v", err)
	}
}

func TestEvaluateRevision_InvalidTuple(t *testing.T) {
	t.Parallel()

	revision := policy.PolicyRevision{Policy: validPolicy("p1"), Revision: 1}
	_, err := policy.EvaluateRevision(revision, "", "res:1", "read")
	if err == nil {
		t.Fatal("expected error for empty subject")
	}
	if !errors.Is(err, policy.ErrInvalidTuple) {
		t.Fatalf("expected ErrInvalidTuple, got %v", err)
	}
}

func TestEvaluateRevision_Determinism(t *testing.T) {
	t.Parallel()

	revision := policy.PolicyRevision{Policy: validPolicy("p1"), Revision: 1}

	got1, err1 := policy.EvaluateRevision(revision, "user:alice", "res:1", "read")
	got2, err2 := policy.EvaluateRevision(revision, "user:alice", "res:1", "read")
	if err1 != nil || err2 != nil {
		t.Fatalf("unexpected errors: %v, %v", err1, err2)
	}
	if !reflect.DeepEqual(got1, got2) {
		t.Fatalf("expected identical results, got %+v vs %+v", got1, got2)
	}
}

func TestNextPolicyRevision_Determinism(t *testing.T) {
	t.Parallel()

	p := validPolicy("p1")
	current := policy.PolicyRevision{Policy: p, Revision: 2}
	next := validPolicy("p1")
	next.Statements[0].Action = "write"

	got1, err1 := policy.NextPolicyRevision(current, next, 2)
	got2, err2 := policy.NextPolicyRevision(current, next, 2)
	if err1 != nil || err2 != nil {
		t.Fatalf("unexpected errors: %v, %v", err1, err2)
	}
	if !reflect.DeepEqual(got1, got2) {
		t.Fatalf("expected identical results, got %+v vs %+v", got1, got2)
	}
}
