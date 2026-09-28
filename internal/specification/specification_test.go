package specification

import (
	"errors"
	"testing"
)

func validRequirement() Requirement {
	return Requirement{
		ID:                 "req-1",
		Version:            1,
		Status:             StatusKnown,
		Source:             SourceRef{ChatID: "chat-1", MessageID: "msg-1"},
		AcceptanceCriteria: []string{"ac-1"},
	}
}

func TestRequirementValidate(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(Requirement) Requirement
		wantErr error
	}{
		{
			name:    "valid",
			mutate:  func(r Requirement) Requirement { return r },
			wantErr: nil,
		},
		{
			name:    "empty id",
			mutate:  func(r Requirement) Requirement { r.ID = ""; return r },
			wantErr: ErrInvalidRequirement,
		},
		{
			name:    "zero version",
			mutate:  func(r Requirement) Requirement { r.Version = 0; return r },
			wantErr: ErrInvalidRequirement,
		},
		{
			name:    "unknown status",
			mutate:  func(r Requirement) Requirement { r.Status = "BOGUS"; return r },
			wantErr: ErrInvalidRequirement,
		},
		{
			name:    "missing source traceability",
			mutate:  func(r Requirement) Requirement { r.Source = SourceRef{}; return r },
			wantErr: ErrInvalidRequirement,
		},
		{
			name:    "missing acceptance criteria",
			mutate:  func(r Requirement) Requirement { r.AcceptanceCriteria = nil; return r },
			wantErr: ErrInvalidRequirement,
		},
		{
			name: "empty acceptance criteria entry",
			mutate: func(r Requirement) Requirement {
				r.AcceptanceCriteria = []string{""}
				return r
			},
			wantErr: ErrInvalidRequirement,
		},
		{
			name:    "document-only source is valid",
			mutate:  func(r Requirement) Requirement { r.Source = SourceRef{Document: "spec.md#5.3"}; return r },
			wantErr: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.mutate(validRequirement()).Validate()
			if tc.wantErr == nil {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("error=%v want=%v", err, tc.wantErr)
			}
		})
	}
}

func TestValidateUpdateMonotonicVersion(t *testing.T) {
	current := validRequirement()

	t.Run("strictly increasing version accepted", func(t *testing.T) {
		next := current
		next.Version = 2
		if err := ValidateUpdate(current, next); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("equal version rejected", func(t *testing.T) {
		next := current
		if err := ValidateUpdate(current, next); !errors.Is(err, ErrNonMonotonicVersion) {
			t.Fatalf("error=%v want=%v", err, ErrNonMonotonicVersion)
		}
	})

	t.Run("regressing version rejected", func(t *testing.T) {
		next := current
		next.Version = 3
		regressed := next
		regressed.Version = 2
		if err := ValidateUpdate(next, regressed); !errors.Is(err, ErrNonMonotonicVersion) {
			t.Fatalf("error=%v want=%v", err, ErrNonMonotonicVersion)
		}
	})

	t.Run("mismatched id rejected", func(t *testing.T) {
		next := current
		next.ID = "req-2"
		next.Version = 2
		if err := ValidateUpdate(current, next); !errors.Is(err, ErrInvalidRequirement) {
			t.Fatalf("error=%v want=%v", err, ErrInvalidRequirement)
		}
	})

	t.Run("invalid next requirement rejected", func(t *testing.T) {
		next := current
		next.Version = 2
		next.Source = SourceRef{}
		if err := ValidateUpdate(current, next); !errors.Is(err, ErrInvalidRequirement) {
			t.Fatalf("error=%v want=%v", err, ErrInvalidRequirement)
		}
	})
}

func TestDecisionValidate(t *testing.T) {
	base := Decision{
		ID:             "dec-1",
		RequirementIDs: []RequirementID{"req-1"},
		Outcome:        "accepted as KNOWN",
		Severity:       SeverityNormal,
	}

	cases := []struct {
		name    string
		mutate  func(Decision) Decision
		wantErr error
	}{
		{name: "valid", mutate: func(d Decision) Decision { return d }, wantErr: nil},
		{name: "empty id", mutate: func(d Decision) Decision { d.ID = ""; return d }, wantErr: ErrInvalidDecision},
		{name: "missing requirement linkage", mutate: func(d Decision) Decision { d.RequirementIDs = nil; return d }, wantErr: ErrInvalidDecision},
		{
			name: "empty requirement id entry",
			mutate: func(d Decision) Decision {
				d.RequirementIDs = []RequirementID{""}
				return d
			},
			wantErr: ErrInvalidDecision,
		},
		{name: "missing outcome", mutate: func(d Decision) Decision { d.Outcome = ""; return d }, wantErr: ErrInvalidDecision},
		{name: "invalid severity", mutate: func(d Decision) Decision { d.Severity = "WHATEVER"; return d }, wantErr: ErrInvalidDecision},
		{name: "critical severity valid", mutate: func(d Decision) Decision { d.Severity = SeverityCritical; return d }, wantErr: nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.mutate(base).Validate()
			if tc.wantErr == nil {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("error=%v want=%v", err, tc.wantErr)
			}
		})
	}
}

func TestApprovalReady(t *testing.T) {
	t.Run("true when no conflicts", func(t *testing.T) {
		reqs := []Requirement{
			{ID: "req-1", Status: StatusKnown},
			{ID: "req-2", Status: StatusDerived},
			{ID: "req-3", Status: StatusRisk},
		}
		ready, blocking := ApprovalReady(reqs)
		if !ready {
			t.Fatalf("expected ready=true")
		}
		if blocking != nil {
			t.Fatalf("expected nil blocking, got %v", blocking)
		}
	})

	t.Run("false with blocking ids when any conflict exists", func(t *testing.T) {
		reqs := []Requirement{
			{ID: "req-1", Status: StatusKnown},
			{ID: "req-2", Status: StatusConflict},
			{ID: "req-3", Status: StatusConflict},
		}
		ready, blocking := ApprovalReady(reqs)
		if ready {
			t.Fatalf("expected ready=false")
		}
		want := []RequirementID{"req-2", "req-3"}
		if len(blocking) != len(want) {
			t.Fatalf("blocking=%v want=%v", blocking, want)
		}
		for i := range want {
			if blocking[i] != want[i] {
				t.Fatalf("blocking=%v want=%v", blocking, want)
			}
		}
	})

	t.Run("empty input is ready", func(t *testing.T) {
		ready, blocking := ApprovalReady(nil)
		if !ready || blocking != nil {
			t.Fatalf("expected ready=true, nil blocking; got ready=%v blocking=%v", ready, blocking)
		}
	})
}

func TestValidateStatusTransition(t *testing.T) {
	cases := []struct {
		current RequirementStatus
		next    RequirementStatus
		wantOK  bool
	}{
		// Identity transitions are always legal.
		{StatusKnown, StatusKnown, true},
		{StatusConflict, StatusConflict, true},

		// UNKNOWN may progress into any classification, including direct
		// confirmation to KNOWN.
		{StatusUnknown, StatusAssumed, true},
		{StatusUnknown, StatusConflict, true},
		{StatusUnknown, StatusRisk, true},
		{StatusUnknown, StatusDecisionRequired, true},
		{StatusUnknown, StatusKnown, true},
		{StatusUnknown, StatusDerived, false},

		// ASSUMED may resolve to KNOWN or surface a problem.
		{StatusAssumed, StatusKnown, true},
		{StatusAssumed, StatusConflict, true},
		{StatusAssumed, StatusRisk, true},
		{StatusAssumed, StatusDecisionRequired, true},
		{StatusAssumed, StatusDerived, false},
		{StatusAssumed, StatusUnknown, false},

		// KNOWN may regress only into a problem classification.
		{StatusKnown, StatusConflict, true},
		{StatusKnown, StatusRisk, true},
		{StatusKnown, StatusDecisionRequired, true},
		{StatusKnown, StatusUnknown, false},
		{StatusKnown, StatusAssumed, false},
		{StatusKnown, StatusDerived, false},

		// CONFLICT/RISK/DECISION_REQUIRED may resolve into KNOWN or DERIVED,
		// or move between each other, but not regress to UNKNOWN/ASSUMED.
		{StatusConflict, StatusKnown, true},
		{StatusConflict, StatusDerived, true},
		{StatusConflict, StatusRisk, true},
		{StatusConflict, StatusDecisionRequired, true},
		{StatusConflict, StatusUnknown, false},
		{StatusConflict, StatusAssumed, false},
		{StatusRisk, StatusKnown, true},
		{StatusRisk, StatusDerived, true},
		{StatusRisk, StatusUnknown, false},
		{StatusDecisionRequired, StatusKnown, true},
		{StatusDecisionRequired, StatusDerived, true},
		{StatusDecisionRequired, StatusUnknown, false},

		// DERIVED cannot regress to UNKNOWN, and no other transitions are
		// permitted out of DERIVED in this bounded slice.
		{StatusDerived, StatusUnknown, false},
		{StatusDerived, StatusKnown, false},
		{StatusDerived, StatusAssumed, false},
		{StatusDerived, StatusConflict, false},

		// Unknown enum values are always rejected.
		{"BOGUS", StatusKnown, false},
		{StatusKnown, "BOGUS", false},
	}

	for _, tc := range cases {
		t.Run(string(tc.current)+"->"+string(tc.next), func(t *testing.T) {
			err := ValidateStatusTransition(tc.current, tc.next)
			if tc.wantOK && err != nil {
				t.Fatalf("expected transition to be legal, got error: %v", err)
			}
			if !tc.wantOK {
				if err == nil {
					t.Fatalf("expected transition to be rejected")
				}
				if !errors.Is(err, ErrInvalidStatusTransition) {
					t.Fatalf("error=%v want wrapped %v", err, ErrInvalidStatusTransition)
				}
			}
		})
	}
}
