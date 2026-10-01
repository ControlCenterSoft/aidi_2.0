package canonical

import (
	"errors"
	"testing"
)

func TestValidateStateReasonAcceptsReasonPresentWhenRequired(t *testing.T) {
	allowed := NewStateSet("PENDING", "BLOCKED", "DONE")
	required := NewStateSet("BLOCKED")
	if err := ValidateStateReason(State("BLOCKED"), Reason("waiting on dependency"), allowed, required); err != nil {
		t.Fatalf("expected valid state/reason pair, got %v", err)
	}
}

func TestValidateStateReasonRejectsMissingReasonWhenRequired(t *testing.T) {
	allowed := NewStateSet("PENDING", "BLOCKED", "DONE")
	required := NewStateSet("BLOCKED")
	err := ValidateStateReason(State("BLOCKED"), Reason(""), allowed, required)
	if err == nil {
		t.Fatal("expected error for missing reason on required state")
	}
	if !errors.Is(err, ErrInvalidReason) {
		t.Fatalf("expected ErrInvalidReason, got %v", err)
	}
}

func TestValidateStateReasonRejectsWhitespaceOnlyReasonWhenRequired(t *testing.T) {
	allowed := NewStateSet("PENDING", "BLOCKED", "DONE")
	required := NewStateSet("BLOCKED")
	err := ValidateStateReason(State("BLOCKED"), Reason("   "), allowed, required)
	if err == nil {
		t.Fatal("expected error for whitespace-only reason on required state")
	}
	if !errors.Is(err, ErrInvalidReason) {
		t.Fatalf("expected ErrInvalidReason, got %v", err)
	}
}

func TestValidateStateReasonAllowsReasonOmittedWhenNotRequired(t *testing.T) {
	allowed := NewStateSet("PENDING", "BLOCKED", "DONE")
	required := NewStateSet("BLOCKED")
	if err := ValidateStateReason(State("PENDING"), Reason(""), allowed, required); err != nil {
		t.Fatalf("expected valid state/reason pair, got %v", err)
	}
}

func TestValidateStateReasonAllowsReasonSuppliedWhenNotRequired(t *testing.T) {
	allowed := NewStateSet("PENDING", "BLOCKED", "DONE")
	required := NewStateSet("BLOCKED")
	if err := ValidateStateReason(State("PENDING"), Reason("extra context"), allowed, required); err != nil {
		t.Fatalf("expected optional extra context to be allowed, got %v", err)
	}
}

func TestValidateStateReasonRejectsInvalidState(t *testing.T) {
	allowed := NewStateSet("PENDING", "BLOCKED", "DONE")
	required := NewStateSet("BLOCKED")
	err := ValidateStateReason(State("BOGUS"), Reason("anything"), allowed, required)
	if err == nil {
		t.Fatal("expected error for state outside allowed set")
	}
	if !errors.Is(err, ErrInvalidState) {
		t.Fatalf("expected ErrInvalidState, got %v", err)
	}
}

func TestReasonIsEmpty(t *testing.T) {
	cases := []struct {
		name string
		r    Reason
		want bool
	}{
		{"empty string", Reason(""), true},
		{"whitespace only", Reason("   "), true},
		{"non-empty", Reason("some reason"), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.r.IsEmpty(); got != tc.want {
				t.Fatalf("IsEmpty() = %v, want %v", got, tc.want)
			}
		})
	}
}
