package canonical

import (
	"errors"
	"math"
	"testing"
)

func TestCheckExpectedRevision(t *testing.T) {
	if err := CheckExpectedRevision(7, 7); err != nil {
		t.Fatalf("matching revision rejected: %v", err)
	}

	err := CheckExpectedRevision(7, 8)
	if err == nil {
		t.Fatal("expected revision conflict")
	}
	if !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("expected ErrRevisionConflict, got %v", err)
	}

	var conflict *RevisionConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("expected RevisionConflictError, got %T", err)
	}
	if conflict.Expected != 7 || conflict.Actual != 8 {
		t.Fatalf("unexpected conflict: %+v", conflict)
	}
}

func TestNextRevision(t *testing.T) {
	next, err := NextRevision(41)
	if err != nil {
		t.Fatal(err)
	}
	if next != 42 {
		t.Fatalf("expected 42, got %d", next)
	}

	if _, err := NextRevision(Revision(math.MaxUint64)); !errors.Is(err, ErrRevisionExhausted) {
		t.Fatalf("expected ErrRevisionExhausted, got %v", err)
	}
}
