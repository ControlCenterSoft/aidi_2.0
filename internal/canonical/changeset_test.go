package canonical

import (
	"errors"
	"testing"

	"github.com/ControlCenterSoft/aidi_2.0/internal/orchestration"
)

func testChangeSetAttempt(id string) orchestration.AttemptBinding {
	return orchestration.AttemptBinding{
		AttemptID:   orchestration.AttemptID(id),
		SourceSHA:   "0123456789abcdef",
		WorkspaceID: "workspace-" + id,
		ExecutorID:  "runner-" + id,
	}
}

func baseChangeSet() ChangeSet {
	return ChangeSet{
		ChangeSetID:  "cs-1",
		Attempt:      testChangeSetAttempt("attempt-1"),
		Target:       ObjectRef{Kind: "task", ID: "task-1"},
		BaseRevision: 1,
		State:        ChangeSetDraft,
	}
}

func TestChangeSetValidateRejectsMissingAttemptBinding(t *testing.T) {
	cs := baseChangeSet()
	cs.Attempt = orchestration.AttemptBinding{}

	if err := cs.Validate(); err == nil {
		t.Fatal("expected error for missing attempt binding")
	}
}

func TestChangeSetValidateRejectsMissingTargetObjectRef(t *testing.T) {
	cases := []ObjectRef{
		{Kind: "", ID: "task-1"},
		{Kind: "task", ID: ""},
		{},
	}
	for _, target := range cases {
		cs := baseChangeSet()
		cs.Target = target
		if err := cs.Validate(); err == nil {
			t.Fatalf("expected error for target %+v", target)
		}
	}
}

func TestChangeSetValidateRejectsMissingBaseRevision(t *testing.T) {
	cs := baseChangeSet()
	cs.BaseRevision = 0

	if err := cs.Validate(); err == nil {
		t.Fatal("expected error for missing base revision")
	}
}

func TestChangeSetValidateRejectsMissingChangeSetID(t *testing.T) {
	cs := baseChangeSet()
	cs.ChangeSetID = ""

	if err := cs.Validate(); err == nil {
		t.Fatal("expected error for missing changeset id")
	}
}

func TestChangeSetValidateRejectsUnknownState(t *testing.T) {
	cs := baseChangeSet()
	cs.State = ChangeSetState("BOGUS")

	if err := cs.Validate(); err == nil {
		t.Fatal("expected error for unknown state")
	}
}

func TestChangeSetValidateRejectsMissingEvidenceForValidatedOrMerged(t *testing.T) {
	for _, state := range []ChangeSetState{ChangeSetValidated, ChangeSetMerged} {
		cs := baseChangeSet()
		cs.State = state
		cs.Evidence = nil

		if err := cs.Validate(); err == nil {
			t.Fatalf("expected error for state %s without evidence", state)
		}
	}
}

func TestChangeSetValidateAcceptsEvidenceForValidatedAndMerged(t *testing.T) {
	for _, state := range []ChangeSetState{ChangeSetValidated, ChangeSetMerged} {
		cs := baseChangeSet()
		cs.State = state
		cs.Evidence = []EvidenceRef{{Kind: "unit-test", ID: "run-1"}}

		if err := cs.Validate(); err != nil {
			t.Fatalf("unexpected error for state %s: %v", state, err)
		}
	}
}

func TestChangeSetValidateRejectsInvalidEvidenceRef(t *testing.T) {
	cases := []EvidenceRef{
		{Kind: "", ID: "run-1"},
		{Kind: "unit-test", ID: ""},
	}
	for _, evidence := range cases {
		cs := baseChangeSet()
		cs.State = ChangeSetValidated
		cs.Evidence = []EvidenceRef{evidence}

		if err := cs.Validate(); err == nil {
			t.Fatalf("expected error for invalid evidence ref %+v", evidence)
		}
	}
}

func TestValidateTransitionAllowsDocumentedEdges(t *testing.T) {
	allowed := []struct {
		current ChangeSetState
		next    ChangeSetState
	}{
		{ChangeSetDraft, ChangeSetDraft},
		{ChangeSetDraft, ChangeSetSubmitted},
		{ChangeSetSubmitted, ChangeSetValidating},
		{ChangeSetValidating, ChangeSetValidated},
		{ChangeSetValidating, ChangeSetRejected},
		{ChangeSetValidated, ChangeSetMerged},
	}
	for _, edge := range allowed {
		if err := ValidateTransition(edge.current, edge.next); err != nil {
			t.Fatalf("expected %s -> %s to be allowed, got %v", edge.current, edge.next, err)
		}
	}
}

func TestValidateTransitionRejectsUndocumentedEdges(t *testing.T) {
	rejected := []struct {
		current ChangeSetState
		next    ChangeSetState
	}{
		{ChangeSetSubmitted, ChangeSetSubmitted},
		{ChangeSetSubmitted, ChangeSetDraft},
		{ChangeSetSubmitted, ChangeSetValidated},
		{ChangeSetValidating, ChangeSetValidating},
		{ChangeSetValidating, ChangeSetSubmitted},
		{ChangeSetValidated, ChangeSetValidated},
		{ChangeSetValidated, ChangeSetValidating},
		{ChangeSetRejected, ChangeSetRejected},
		{ChangeSetRejected, ChangeSetDraft},
		{ChangeSetRejected, ChangeSetSubmitted},
		{ChangeSetMerged, ChangeSetMerged},
		{ChangeSetMerged, ChangeSetValidated},
		{ChangeSetDraft, ChangeSetValidating},
		{ChangeSetDraft, ChangeSetValidated},
		{ChangeSetDraft, ChangeSetMerged},
		{ChangeSetDraft, ChangeSetRejected},
	}
	for _, edge := range rejected {
		if err := ValidateTransition(edge.current, edge.next); err == nil {
			t.Fatalf("expected %s -> %s to be rejected", edge.current, edge.next)
		} else if !errors.Is(err, ErrInvalidChangeSetTransition) {
			t.Fatalf("expected ErrInvalidChangeSetTransition for %s -> %s, got %v", edge.current, edge.next, err)
		}
	}
}

func TestChangeSetFullLifecycleValidPath(t *testing.T) {
	cs := baseChangeSet()

	submitted, err := cs.Transition(ChangeSetSubmitted)
	if err != nil {
		t.Fatalf("submit: %v", err)
	}

	validating, err := submitted.Transition(ChangeSetValidating)
	if err != nil {
		t.Fatalf("validating: %v", err)
	}

	validating.Evidence = []EvidenceRef{{Kind: "unit-test", ID: "run-1"}}
	validated, err := validating.Transition(ChangeSetValidated)
	if err != nil {
		t.Fatalf("validated: %v", err)
	}

	merged, err := validated.Merge(1)
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	if merged.State != ChangeSetMerged {
		t.Fatalf("expected merged state, got %s", merged.State)
	}
}

func TestChangeSetTransitionRejectsMissingEvidenceReachingValidated(t *testing.T) {
	cs := baseChangeSet()
	cs.State = ChangeSetValidating

	if _, err := cs.Transition(ChangeSetValidated); err == nil {
		t.Fatal("expected error transitioning to VALIDATED without evidence")
	} else if !errors.Is(err, ErrChangeSetInvariant) {
		t.Fatalf("expected ErrChangeSetInvariant, got %v", err)
	}
}

func TestChangeSetMergeRejectsInvalidTransition(t *testing.T) {
	cs := baseChangeSet()
	cs.State = ChangeSetSubmitted

	if _, err := cs.Merge(1); err == nil {
		t.Fatal("expected error merging from SUBMITTED")
	} else if !errors.Is(err, ErrInvalidChangeSetTransition) {
		t.Fatalf("expected ErrInvalidChangeSetTransition, got %v", err)
	}
}

func TestChangeSetMergeRejectsStaleRevision(t *testing.T) {
	cs := baseChangeSet()
	cs.State = ChangeSetValidated
	cs.Evidence = []EvidenceRef{{Kind: "unit-test", ID: "run-1"}}
	cs.BaseRevision = 1

	if _, err := cs.Merge(2); err == nil {
		t.Fatal("expected error for stale base revision")
	} else if !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("expected ErrRevisionConflict, got %v", err)
	}
}

func TestChangeSetMergeAcceptsMatchingRevision(t *testing.T) {
	cs := baseChangeSet()
	cs.State = ChangeSetValidated
	cs.Evidence = []EvidenceRef{{Kind: "unit-test", ID: "run-1"}}
	cs.BaseRevision = 5

	merged, err := cs.Merge(5)
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	if merged.State != ChangeSetMerged {
		t.Fatalf("expected merged state, got %s", merged.State)
	}
}
