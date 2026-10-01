package canonical

import (
	"errors"
	"math"
	"testing"
)

func TestNewRevisionedStartsAtRevisionOne(t *testing.T) {
	r := NewRevisioned("payload")

	if r.Revision != 1 {
		t.Fatalf("expected initial revision 1, got %d", r.Revision)
	}
	if r.Value != "payload" {
		t.Fatalf("expected value %q, got %q", "payload", r.Value)
	}
}

func TestRevisionedUpdateAdvancesRevisionMonotonically(t *testing.T) {
	r := NewRevisioned(1)

	r, err := r.Update(1, func(v int) (int, error) { return v + 1, nil })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Revision != 2 {
		t.Fatalf("expected revision 2, got %d", r.Revision)
	}
	if r.Value != 2 {
		t.Fatalf("expected value 2, got %d", r.Value)
	}

	r, err = r.Update(2, func(v int) (int, error) { return v + 1, nil })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Revision != 3 {
		t.Fatalf("expected revision 3, got %d", r.Revision)
	}
	if r.Value != 3 {
		t.Fatalf("expected value 3, got %d", r.Value)
	}
}

func TestRevisionedUpdateRejectsStaleExpectedRevision(t *testing.T) {
	r := NewRevisioned("v1")

	mutateCalled := false
	_, err := r.Update(0, func(v string) (string, error) {
		mutateCalled = true
		return v, nil
	})
	if err == nil {
		t.Fatal("expected revision conflict, got nil")
	}
	if !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("expected ErrRevisionConflict, got %v", err)
	}
	var conflict *RevisionConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("expected RevisionConflictError, got %T", err)
	}
	if conflict.Expected != 0 || conflict.Actual != 1 {
		t.Fatalf("unexpected conflict: %+v", conflict)
	}
	if mutateCalled {
		t.Fatal("mutate must not run on a stale expected revision")
	}
	if r.Revision != 1 || r.Value != "v1" {
		t.Fatalf("receiver must be left unchanged, got %+v", r)
	}
}

func TestRevisionedUpdatePropagatesMutateErrorWithoutAdvancingRevision(t *testing.T) {
	r := NewRevisioned("v1")
	boom := errors.New("mutate failed")

	_, err := r.Update(1, func(string) (string, error) {
		return "", boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("expected mutate error to propagate, got %v", err)
	}
	if r.Revision != 1 || r.Value != "v1" {
		t.Fatalf("receiver must be left unchanged, got %+v", r)
	}
}

func TestRevisionedUpdateSurfacesRevisionExhaustion(t *testing.T) {
	r := Revisioned[string]{Revision: Revision(math.MaxUint64), Value: "v1"}

	_, err := r.Update(Revision(math.MaxUint64), func(v string) (string, error) { return v, nil })
	if !errors.Is(err, ErrRevisionExhausted) {
		t.Fatalf("expected ErrRevisionExhausted, got %v", err)
	}
}
