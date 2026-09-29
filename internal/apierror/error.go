// Package apierror implements the SPEC §13.1 requirement that the Public
// API returns structured errors correlated to the command/query that
// produced them. This is a pure Go domain contract: no HTTP framework, no
// persistence, and no queue/VM/runner dependency. HTTP transport wiring,
// rate-limit enforcement and Public API routing are later Release A/B
// slices layered on top of this contract.
package apierror

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/ControlCenterSoft/aidi_2.0/internal/apiratelimit"
	"github.com/ControlCenterSoft/aidi_2.0/internal/canonical"
)

// Code is a closed set of canonical Public API error codes. Any value not
// listed in this file is rejected by Validate.
type Code string

const (
	// CodeValidation indicates the request failed input/business
	// validation before any state change was attempted.
	CodeValidation Code = "VALIDATION"
	// CodeConflict indicates an optimistic-concurrency or state conflict:
	// the caller must reconcile (re-fetch and retry with the correct
	// expectation), not blindly retry the same request.
	CodeConflict Code = "CONFLICT"
	// CodeNotFound indicates the referenced resource does not exist.
	CodeNotFound Code = "NOT_FOUND"
	// CodeRateLimited indicates the caller exceeded an enforced rate
	// limit.
	CodeRateLimited Code = "RATE_LIMITED"
	// CodeInternal indicates an unexpected server-side failure.
	CodeInternal Code = "INTERNAL"
)

// knownCodes is the closed set of Code values accepted by Validate.
var knownCodes = map[Code]struct{}{
	CodeValidation:  {},
	CodeConflict:    {},
	CodeNotFound:    {},
	CodeRateLimited: {},
	CodeInternal:    {},
}

// ErrInvalidError is returned (wrapped) by Validate when an Error value does
// not satisfy the structured-error contract.
var ErrInvalidError = errors.New("invalid structured api error")

// Error is the Public API structured error envelope required by SPEC
// §13.1: every error returned to a caller carries a canonical Code, a
// human-readable Message, a mandatory CorrelationID (so the error can be
// tied back to the originating command/query), whether the caller may
// safely Retry the same request, and optional Details for machine-readable
// context (e.g. conflicting revisions).
type Error struct {
	Code          Code              `json:"code"`
	Message       string            `json:"message"`
	CorrelationID string            `json:"correlation_id"`
	Retryable     bool              `json:"retryable"`
	Details       map[string]string `json:"details,omitempty"`
}

// Error implements the error interface so apierror.Error can be used/wrapped
// like any other Go error.
func (e Error) Error() string {
	return fmt.Sprintf("%s: %s (correlation_id=%s)", e.Code, e.Message, e.CorrelationID)
}

// New constructs an Error and validates it, returning the zero Error and a
// wrapped ErrInvalidError if any required field is missing or the code is
// unknown.
func New(code Code, message, correlationID string, details map[string]string) (Error, error) {
	e := Error{
		Code:          code,
		Message:       message,
		CorrelationID: correlationID,
		Details:       details,
	}
	if err := e.Validate(); err != nil {
		return Error{}, err
	}
	return e, nil
}

// Validate rejects an Error missing Code, Message or CorrelationID, or
// carrying an unknown Code. Correlation is mandatory per SPEC §13.1's
// "correlation" requirement: every structured error must be traceable back
// to the command/query that produced it.
func (e Error) Validate() error {
	if strings.TrimSpace(string(e.Code)) == "" {
		return fmt.Errorf("%w: code is required", ErrInvalidError)
	}
	if _, ok := knownCodes[e.Code]; !ok {
		return fmt.Errorf("%w: unknown code %q", ErrInvalidError, e.Code)
	}
	if strings.TrimSpace(e.Message) == "" {
		return fmt.Errorf("%w: message is required", ErrInvalidError)
	}
	if strings.TrimSpace(e.CorrelationID) == "" {
		return fmt.Errorf("%w: correlation id is required", ErrInvalidError)
	}
	return nil
}

// MarshalJSON round-trips the envelope with stable field names, delegating
// to the default struct encoding (kept explicit so the field order/name
// contract in this file is the single source of truth for callers).
func (e Error) MarshalJSON() ([]byte, error) {
	type alias Error
	return json.Marshal(alias(e))
}

// FromRevisionConflict maps a canonical.RevisionConflictError onto the
// Public API structured error envelope: Code = CONFLICT, the original
// Expected/Actual revisions are preserved unchanged in Details, and the
// error is never marked Retryable. A conflicted ChangeSet/canonical write
// must be reconciled (rebase/regenerate against the current revision), not
// blindly retried — consistent with docs/CANONICAL_STATE.md.
func FromRevisionConflict(conflict *canonical.RevisionConflictError, correlationID string) (Error, error) {
	if conflict == nil {
		return Error{}, fmt.Errorf("%w: revision conflict is required", ErrInvalidError)
	}
	return New(
		CodeConflict,
		conflict.Error(),
		correlationID,
		map[string]string{
			"expected_revision": fmt.Sprintf("%d", conflict.Expected),
			"actual_revision":   fmt.Sprintf("%d", conflict.Actual),
		},
	)
}

// FromRateLimitDecision maps a denied apiratelimit.Decision onto the Public
// API structured error envelope: Code = CodeRateLimited, Retryable is true
// only when RetryAfter > 0, and RetryAfter is surfaced via
// Details["retry_after_ms"] — following the FromRevisionConflict mapping
// pattern exactly. An Allowed decision has nothing to map into an error and
// is rejected.
func FromRateLimitDecision(decision apiratelimit.Decision, correlationID string) (Error, error) {
	if decision.Allowed {
		return Error{}, fmt.Errorf("%w: allowed decision cannot be mapped to an error", ErrInvalidError)
	}
	if err := decision.Validate(); err != nil {
		return Error{}, fmt.Errorf("%w: %v", ErrInvalidError, err)
	}
	e, err := New(
		CodeRateLimited,
		"rate limit exceeded",
		correlationID,
		map[string]string{
			"retry_after_ms": fmt.Sprintf("%d", decision.RetryAfter.Milliseconds()),
			"limit":          fmt.Sprintf("%d", decision.Limit),
			"remaining":      fmt.Sprintf("%d", decision.Remaining),
		},
	)
	if err != nil {
		return Error{}, err
	}
	e.Retryable = decision.RetryAfter > 0
	return e, nil
}
