// Package apiratelimit defines the transport-independent Public API rate
// limit domain contract required by SPEC §13.1: "Commands отделены от
// Queries; поддерживаются idempotency command IDs, correlation, optimistic
// concurrency, structured errors и rate limits." Release A already
// delivers the first four (internal/apicommand, internal/apiquery,
// internal/apierror, canonical.Revision); this package closes the
// remaining gap by defining *what* gets rate-limited (LimitKey), *how* a
// limit is declared (Policy), and *how* a limit decision is represented
// (Decision) — as a pure Go domain contract with no HTTP framework, no
// persistence, and no queue/VM/runner dependency.
//
// Evaluate is a pure function: it computes a Decision deterministically
// from its inputs only (no clock/global/singleton state, no goroutines, no
// storage). Actual token-bucket/sliding-window storage, HTTP middleware
// wiring, per-endpoint policy configuration/tuning, distributed counters,
// and enforcement inside apicommand/apiquery handlers are later Release
// A/B slices layered on top of this contract.
package apiratelimit

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// ErrInvalidPolicy is returned (wrapped) by Policy.Validate when a policy
// does not describe a usable rate limit.
var ErrInvalidPolicy = errors.New("invalid rate limit policy")

// ErrInvalidLimitKey is returned (wrapped) by LimitKey.Validate when a key
// does not identify a concrete rate-limit scope.
var ErrInvalidLimitKey = errors.New("invalid rate limit key")

// ErrInvalidDecision is returned (wrapped) by Evaluate/mapping helpers when
// inputs cannot produce a well-formed Decision.
var ErrInvalidDecision = errors.New("invalid rate limit decision")

// LimitKey identifies the scope a rate limit Policy applies to: a subject
// (e.g. a service-identity or caller id) performing a named Operation,
// mirroring the apicommand.Target/apiquery.Target validation style.
type LimitKey struct {
	Subject   string `json:"subject"`
	Operation string `json:"operation"`
}

// Validate rejects an empty/whitespace Subject or Operation.
func (k LimitKey) Validate() error {
	if strings.TrimSpace(k.Subject) == "" {
		return fmt.Errorf("%w: subject is required", ErrInvalidLimitKey)
	}
	if strings.TrimSpace(k.Operation) == "" {
		return fmt.Errorf("%w: operation is required", ErrInvalidLimitKey)
	}
	return nil
}

// Policy is a declarative rate limit: at most Limit requests may occur
// within Window. Policy carries no storage/counters of its own; observed
// usage is supplied to Evaluate by the caller.
type Policy struct {
	Window time.Duration `json:"window"`
	Limit  uint64        `json:"limit"`
}

// Validate rejects a non-positive Window or a zero Limit.
func (p Policy) Validate() error {
	if p.Window <= 0 {
		return fmt.Errorf("%w: window must be greater than zero", ErrInvalidPolicy)
	}
	if p.Limit == 0 {
		return fmt.Errorf("%w: limit must be greater than zero", ErrInvalidPolicy)
	}
	return nil
}

// Decision is the outcome of evaluating a Policy against observed usage for
// a LimitKey. RetryAfter is zero when Allowed is true, and is non-negative
// when Allowed is false. Limit/Remaining report the policy limit and the
// remaining budget in the current window (Remaining is zero once the
// policy has been met or exceeded).
type Decision struct {
	Allowed    bool          `json:"allowed"`
	RetryAfter time.Duration `json:"retry_after"`
	Limit      uint64        `json:"limit"`
	Remaining  uint64        `json:"remaining"`
}

// Validate enforces the Decision invariant documented above: RetryAfter
// must be zero when Allowed is true, and must be non-negative in all
// cases. Callers that map a Decision onto another representation (e.g.
// apierror.FromRateLimitDecision) must call Validate so a manually
// constructed, malformed Decision cannot be serialized downstream.
func (d Decision) Validate() error {
	if d.Allowed && d.RetryAfter != 0 {
		return fmt.Errorf("%w: retry after must be zero when allowed", ErrInvalidDecision)
	}
	if d.RetryAfter < 0 {
		return fmt.Errorf("%w: retry after must be non-negative", ErrInvalidDecision)
	}
	return nil
}

// Evaluate computes a Decision deterministically from policy, key,
// requestsInWindow (usage already observed in the current window) and now
// (the caller-supplied evaluation instant). It is a pure function: it
// reads no clock, global, or singleton state other than the arguments
// given, and performs no I/O.
//
// requestsInWindow < policy.Limit is allowed with RetryAfter == 0 and
// Remaining == policy.Limit - requestsInWindow. requestsInWindow >=
// policy.Limit is denied with RetryAfter == policy.Window (the caller must
// wait out the remainder of the current window; this pure contract has no
// visibility into window start time, so it reports the full window as the
// conservative retry hint) and Remaining == 0.
func Evaluate(policy Policy, key LimitKey, requestsInWindow uint64, now time.Time) (Decision, error) {
	if err := policy.Validate(); err != nil {
		return Decision{}, err
	}
	if err := key.Validate(); err != nil {
		return Decision{}, err
	}
	if now.IsZero() {
		return Decision{}, fmt.Errorf("%w: now is required", ErrInvalidDecision)
	}

	if requestsInWindow < policy.Limit {
		return Decision{
			Allowed:    true,
			RetryAfter: 0,
			Limit:      policy.Limit,
			Remaining:  policy.Limit - requestsInWindow,
		}, nil
	}

	return Decision{
		Allowed:    false,
		RetryAfter: policy.Window,
		Limit:      policy.Limit,
		Remaining:  0,
	}, nil
}
