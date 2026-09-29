package specification

import (
	"errors"
	"testing"
)

func validApproval() Approval {
	return Approval{
		ID:                   "appr-1",
		SpecificationID:      "spec-1",
		SpecificationVersion: 1,
		Approver:             "user-1",
		Decision:             DecisionApproved,
	}
}

func TestApprovalValidate(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(Approval) Approval
		wantErr error
	}{
		{
			name:    "valid approved",
			mutate:  func(a Approval) Approval { return a },
			wantErr: nil,
		},
		{
			name:    "valid rejected",
			mutate:  func(a Approval) Approval { a.Decision = DecisionRejected; return a },
			wantErr: nil,
		},
		{
			name:    "empty id",
			mutate:  func(a Approval) Approval { a.ID = ""; return a },
			wantErr: ErrInvalidApproval,
		},
		{
			name:    "empty specification id",
			mutate:  func(a Approval) Approval { a.SpecificationID = ""; return a },
			wantErr: ErrInvalidApproval,
		},
		{
			name:    "zero specification version",
			mutate:  func(a Approval) Approval { a.SpecificationVersion = 0; return a },
			wantErr: ErrInvalidApproval,
		},
		{
			name:    "unknown decision",
			mutate:  func(a Approval) Approval { a.Decision = "MAYBE"; return a },
			wantErr: ErrInvalidApproval,
		},
		{
			name:    "empty approver",
			mutate:  func(a Approval) Approval { a.Approver = ""; return a },
			wantErr: ErrInvalidApproval,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.mutate(validApproval()).Validate()
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

func TestNewApprovalComposesWithApprovalReady(t *testing.T) {
	t.Run("rejected while a critical conflict is outstanding", func(t *testing.T) {
		reqs := []Requirement{
			{ID: "req-1", Status: StatusKnown},
			{ID: "req-2", Status: StatusConflict},
		}
		_, err := NewApproval(validApproval(), reqs)
		if !errors.Is(err, ErrApprovalNotReady) {
			t.Fatalf("error=%v want=%v", err, ErrApprovalNotReady)
		}
	})

	t.Run("permitted once the conflict is resolved", func(t *testing.T) {
		reqs := []Requirement{
			{ID: "req-1", Status: StatusKnown},
			{ID: "req-2", Status: StatusDerived},
		}
		got, err := NewApproval(validApproval(), reqs)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != validApproval() {
			t.Fatalf("got=%v want=%v", got, validApproval())
		}
	})

	t.Run("permitted with no requirements at all", func(t *testing.T) {
		_, err := NewApproval(validApproval(), nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("invalid approval still rejected once ready", func(t *testing.T) {
		invalid := validApproval()
		invalid.Approver = ""
		_, err := NewApproval(invalid, nil)
		if !errors.Is(err, ErrInvalidApproval) {
			t.Fatalf("error=%v want=%v", err, ErrInvalidApproval)
		}
	})
}

func TestApprovalCurrentForVersion(t *testing.T) {
	t.Run("matching version is current", func(t *testing.T) {
		if err := ApprovalCurrentForVersion(validApproval(), 1); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("mismatched version is not current", func(t *testing.T) {
		err := ApprovalCurrentForVersion(validApproval(), 2)
		if !errors.Is(err, ErrApprovalVersionMismatch) {
			t.Fatalf("error=%v want=%v", err, ErrApprovalVersionMismatch)
		}
	})

	t.Run("approval bound to N does not cover N+1", func(t *testing.T) {
		approval := validApproval()
		approval.SpecificationVersion = 3
		if err := ApprovalCurrentForVersion(approval, 4); !errors.Is(err, ErrApprovalVersionMismatch) {
			t.Fatalf("error=%v want=%v", err, ErrApprovalVersionMismatch)
		}
		if err := ApprovalCurrentForVersion(approval, 3); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("invalid approval is never current", func(t *testing.T) {
		invalid := validApproval()
		invalid.SpecificationID = ""
		err := ApprovalCurrentForVersion(invalid, 1)
		if !errors.Is(err, ErrInvalidApproval) {
			t.Fatalf("error=%v want=%v", err, ErrInvalidApproval)
		}
	})
}
