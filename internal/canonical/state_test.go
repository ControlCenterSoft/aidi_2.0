package canonical

import (
	"errors"
	"testing"
)

func TestStateValidateAcceptsMemberOfAllowedSet(t *testing.T) {
	allowed := NewStateSet("PENDING", "READY", "DONE")
	if err := State("READY").Validate(allowed); err != nil {
		t.Fatalf("expected valid state, got %v", err)
	}
}

func TestStateValidateRejectsEmpty(t *testing.T) {
	allowed := NewStateSet("PENDING", "READY", "DONE")
	var s State
	if err := s.Validate(allowed); err == nil {
		t.Fatal("expected error for empty state")
	} else if !errors.Is(err, ErrInvalidState) {
		t.Fatalf("expected ErrInvalidState, got %v", err)
	}
}

func TestStateValidateRejectsWhitespaceOnly(t *testing.T) {
	allowed := NewStateSet("PENDING", "READY", "DONE")
	s := State("   ")
	if err := s.Validate(allowed); err == nil {
		t.Fatal("expected error for whitespace-only state")
	} else if !errors.Is(err, ErrInvalidState) {
		t.Fatalf("expected ErrInvalidState, got %v", err)
	}
}

func TestStateValidateRejectsValueOutsideAllowedSet(t *testing.T) {
	allowed := NewStateSet("PENDING", "READY", "DONE")
	s := State("BOGUS")
	if err := s.Validate(allowed); err == nil {
		t.Fatal("expected error for state outside allowed set")
	} else if !errors.Is(err, ErrInvalidState) {
		t.Fatalf("expected ErrInvalidState, got %v", err)
	}
}

func TestStateSetContains(t *testing.T) {
	allowed := NewStateSet("PENDING", "READY")
	if !allowed.Contains("PENDING") {
		t.Fatal("expected set to contain PENDING")
	}
	if allowed.Contains("DONE") {
		t.Fatal("expected set to not contain DONE")
	}
}

func TestNewStateSetEmpty(t *testing.T) {
	empty := NewStateSet()
	if empty.Contains("ANYTHING") {
		t.Fatal("expected empty set to contain nothing")
	}
	if err := State("ANYTHING").Validate(empty); err == nil {
		t.Fatal("expected error validating against empty allowed set")
	} else if !errors.Is(err, ErrInvalidState) {
		t.Fatalf("expected ErrInvalidState, got %v", err)
	}
}
