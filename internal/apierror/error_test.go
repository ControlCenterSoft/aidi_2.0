package apierror

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/ControlCenterSoft/aidi_2.0/internal/canonical"
)

func TestNew_ValidError(t *testing.T) {
	e, err := New(CodeValidation, "field is required", "corr-1", map[string]string{"field": "name"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if e.Code != CodeValidation || e.Message != "field is required" || e.CorrelationID != "corr-1" {
		t.Fatalf("unexpected error value: %+v", e)
	}
	if e.Retryable {
		t.Fatalf("expected Retryable to default to false, got true")
	}
}

func TestValidate_RejectsMissingFields(t *testing.T) {
	cases := []struct {
		name string
		in   Error
	}{
		{
			name: "missing code",
			in:   Error{Code: "", Message: "boom", CorrelationID: "corr-1"},
		},
		{
			name: "unknown code",
			in:   Error{Code: Code("NOT_A_CODE"), Message: "boom", CorrelationID: "corr-1"},
		},
		{
			name: "missing message",
			in:   Error{Code: CodeInternal, Message: "", CorrelationID: "corr-1"},
		},
		{
			name: "whitespace-only message",
			in:   Error{Code: CodeInternal, Message: "   ", CorrelationID: "corr-1"},
		},
		{
			name: "missing correlation id",
			in:   Error{Code: CodeInternal, Message: "boom", CorrelationID: ""},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.in.Validate()
			if err == nil {
				t.Fatal("expected validation error")
			}
			if !errors.Is(err, ErrInvalidError) {
				t.Fatalf("expected ErrInvalidError, got %v", err)
			}
		})
	}
}

func TestNew_RejectsInvalidInput(t *testing.T) {
	if _, err := New("", "boom", "corr-1", nil); !errors.Is(err, ErrInvalidError) {
		t.Fatalf("expected ErrInvalidError for missing code, got %v", err)
	}
	if _, err := New(Code("UNKNOWN"), "boom", "corr-1", nil); !errors.Is(err, ErrInvalidError) {
		t.Fatalf("expected ErrInvalidError for unknown code, got %v", err)
	}
	if _, err := New(CodeInternal, "", "corr-1", nil); !errors.Is(err, ErrInvalidError) {
		t.Fatalf("expected ErrInvalidError for missing message, got %v", err)
	}
	if _, err := New(CodeInternal, "boom", "", nil); !errors.Is(err, ErrInvalidError) {
		t.Fatalf("expected ErrInvalidError for missing correlation id, got %v", err)
	}
}

func TestError_JSONRoundTrip(t *testing.T) {
	original, err := New(CodeNotFound, "project not found", "corr-42", map[string]string{"project_id": "p-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	original.Retryable = false

	raw, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("unmarshal into map failed: %v", err)
	}
	for _, field := range []string{"code", "message", "correlation_id", "retryable", "details"} {
		if _, ok := m[field]; !ok {
			t.Fatalf("expected field %q in JSON output, got %s", field, raw)
		}
	}

	var decoded Error
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if !reflect.DeepEqual(decoded, original) {
		t.Fatalf("round-trip mismatch: got %+v, want %+v", decoded, original)
	}
	if err := decoded.Validate(); err != nil {
		t.Fatalf("round-tripped error failed validation: %v", err)
	}
}

func TestError_JSONRoundTrip_NoDetails(t *testing.T) {
	original, err := New(CodeInternal, "unexpected failure", "corr-7", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	raw, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("unmarshal into map failed: %v", err)
	}
	if _, ok := m["details"]; ok {
		t.Fatalf("expected omitempty details to be absent, got %s", raw)
	}
}

func TestFromRevisionConflict(t *testing.T) {
	base := canonical.CheckExpectedRevision(canonical.Revision(7), canonical.Revision(9))
	var conflict *canonical.RevisionConflictError
	if !errors.As(base, &conflict) {
		t.Fatalf("expected RevisionConflictError, got %v", base)
	}

	mapped, err := FromRevisionConflict(conflict, "corr-99")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mapped.Code != CodeConflict {
		t.Fatalf("expected CodeConflict, got %v", mapped.Code)
	}
	if mapped.CorrelationID != "corr-99" {
		t.Fatalf("expected correlation id to be preserved, got %q", mapped.CorrelationID)
	}
	if mapped.Retryable {
		t.Fatalf("expected Retryable to be false for a revision conflict, got true")
	}
	if mapped.Details["expected_revision"] != "7" || mapped.Details["actual_revision"] != "9" {
		t.Fatalf("expected original revisions to be preserved unchanged, got %+v", mapped.Details)
	}
	if err := mapped.Validate(); err != nil {
		t.Fatalf("mapped error failed validation: %v", err)
	}
}

func TestFromRevisionConflict_RejectsNil(t *testing.T) {
	if _, err := FromRevisionConflict(nil, "corr-1"); !errors.Is(err, ErrInvalidError) {
		t.Fatalf("expected ErrInvalidError for nil conflict, got %v", err)
	}
}
