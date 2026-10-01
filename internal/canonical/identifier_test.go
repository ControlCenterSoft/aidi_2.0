package canonical

import (
	"errors"
	"testing"
)

func TestKindValidateAcceptsAllSpec41Kinds(t *testing.T) {
	kinds := []Kind{
		KindInstallation,
		KindIdentityUser,
		KindWorkspace,
		KindProject,
		KindSpecification,
		KindRequirement,
		KindRelease,
		KindFeature,
		KindTask,
		KindWorkflow,
		KindAttempt,
		KindChangeSet,
		KindVerification,
		KindEvidence,
		KindArtifact,
		KindDecision,
		KindApproval,
		KindRisk,
		KindChangeRequest,
		KindProblem,
		KindRecoveryCase,
		KindPolicy,
		KindResource,
		KindEventAudit,
		KindOperationalKnowledge,
	}
	if len(kinds) != len(validKinds) {
		t.Fatalf("expected %d recognized kinds, test lists %d", len(validKinds), len(kinds))
	}
	for _, k := range kinds {
		if err := k.Validate(); err != nil {
			t.Fatalf("expected kind %q to be valid, got %v", k, err)
		}
	}
}

func TestKindValidateRejectsEmpty(t *testing.T) {
	var k Kind
	if err := k.Validate(); err == nil {
		t.Fatal("expected error for empty kind")
	} else if !errors.Is(err, ErrInvalidKind) {
		t.Fatalf("expected ErrInvalidKind, got %v", err)
	}
}

func TestKindValidateRejectsWhitespaceOnly(t *testing.T) {
	k := Kind("   ")
	if err := k.Validate(); err == nil {
		t.Fatal("expected error for whitespace-only kind")
	} else if !errors.Is(err, ErrInvalidKind) {
		t.Fatalf("expected ErrInvalidKind, got %v", err)
	}
}

func TestKindValidateRejectsUnknownKind(t *testing.T) {
	k := Kind("not_a_real_entity")
	if err := k.Validate(); err == nil {
		t.Fatal("expected error for unknown kind")
	} else if !errors.Is(err, ErrInvalidKind) {
		t.Fatalf("expected ErrInvalidKind, got %v", err)
	}
}

func TestIDValidateAcceptsNonEmpty(t *testing.T) {
	id := ID("task-1")
	if err := id.Validate(); err != nil {
		t.Fatalf("expected valid id, got %v", err)
	}
}

func TestIDValidateRejectsEmpty(t *testing.T) {
	var id ID
	if err := id.Validate(); err == nil {
		t.Fatal("expected error for empty id")
	} else if !errors.Is(err, ErrInvalidID) {
		t.Fatalf("expected ErrInvalidID, got %v", err)
	}
}

func TestIDValidateRejectsWhitespaceOnly(t *testing.T) {
	id := ID("   ")
	if err := id.Validate(); err == nil {
		t.Fatal("expected error for whitespace-only id")
	} else if !errors.Is(err, ErrInvalidID) {
		t.Fatalf("expected ErrInvalidID, got %v", err)
	}
}

func TestObjectRefValidateAcceptsKnownKindAndID(t *testing.T) {
	ref := ObjectRef{Kind: KindTask, ID: "task-1"}
	if err := ref.Validate(); err != nil {
		t.Fatalf("expected valid object ref, got %v", err)
	}
}

func TestObjectRefValidateRejectsEmptyKind(t *testing.T) {
	ref := ObjectRef{Kind: "", ID: "task-1"}
	if err := ref.Validate(); err == nil {
		t.Fatal("expected error for empty kind")
	} else if !errors.Is(err, ErrInvalidKind) {
		t.Fatalf("expected ErrInvalidKind, got %v", err)
	}
}

func TestObjectRefValidateRejectsUnknownKind(t *testing.T) {
	ref := ObjectRef{Kind: "bogus", ID: "task-1"}
	if err := ref.Validate(); err == nil {
		t.Fatal("expected error for unknown kind")
	} else if !errors.Is(err, ErrInvalidKind) {
		t.Fatalf("expected ErrInvalidKind, got %v", err)
	}
}

func TestObjectRefValidateRejectsEmptyID(t *testing.T) {
	ref := ObjectRef{Kind: KindTask, ID: ""}
	if err := ref.Validate(); err == nil {
		t.Fatal("expected error for empty id")
	} else if !errors.Is(err, ErrInvalidID) {
		t.Fatalf("expected ErrInvalidID, got %v", err)
	}
}

func TestEventValidationRejectsUnknownObjectKindViaSharedType(t *testing.T) {
	event := validEvent()
	event.Object.Kind = "not_a_real_entity"
	if _, err := NewEvent(event); err == nil {
		t.Fatal("expected error for unknown event object kind")
	} else if !errors.Is(err, ErrInvalidKind) {
		t.Fatalf("expected ErrInvalidKind, got %v", err)
	}
}

func TestChangeSetValidateRejectsUnknownTargetKindViaSharedType(t *testing.T) {
	cs := baseChangeSet()
	cs.Target = ObjectRef{Kind: "not_a_real_entity", ID: "task-1"}
	if err := cs.Validate(); err == nil {
		t.Fatal("expected error for unknown target kind")
	} else if !errors.Is(err, ErrInvalidKind) {
		t.Fatalf("expected ErrInvalidKind, got %v", err)
	} else if !errors.Is(err, ErrChangeSetInvariant) {
		t.Fatalf("expected ErrChangeSetInvariant, got %v", err)
	}
}
