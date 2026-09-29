package apiratelimit

import (
	"errors"
	"testing"
	"time"
)

func TestPolicy_Validate(t *testing.T) {
	cases := []struct {
		name    string
		policy  Policy
		wantErr bool
	}{
		{name: "valid", policy: Policy{Window: time.Second, Limit: 10}, wantErr: false},
		{name: "zero window", policy: Policy{Window: 0, Limit: 10}, wantErr: true},
		{name: "negative window", policy: Policy{Window: -time.Second, Limit: 10}, wantErr: true},
		{name: "zero limit", policy: Policy{Window: time.Second, Limit: 0}, wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.policy.Validate()
			if tc.wantErr {
				if !errors.Is(err, ErrInvalidPolicy) {
					t.Fatalf("expected ErrInvalidPolicy, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestLimitKey_Validate(t *testing.T) {
	cases := []struct {
		name    string
		key     LimitKey
		wantErr bool
	}{
		{name: "valid", key: LimitKey{Subject: "svc-1", Operation: "create_project"}, wantErr: false},
		{name: "empty subject", key: LimitKey{Subject: "", Operation: "create_project"}, wantErr: true},
		{name: "whitespace subject", key: LimitKey{Subject: "   ", Operation: "create_project"}, wantErr: true},
		{name: "empty operation", key: LimitKey{Subject: "svc-1", Operation: ""}, wantErr: true},
		{name: "whitespace operation", key: LimitKey{Subject: "svc-1", Operation: "   "}, wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.key.Validate()
			if tc.wantErr {
				if !errors.Is(err, ErrInvalidLimitKey) {
					t.Fatalf("expected ErrInvalidLimitKey, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestEvaluate_RejectsInvalidInputs(t *testing.T) {
	validPolicy := Policy{Window: time.Second, Limit: 5}
	validKey := LimitKey{Subject: "svc-1", Operation: "create_project"}
	now := time.Unix(1_700_000_000, 0)

	if _, err := Evaluate(Policy{}, validKey, 0, now); !errors.Is(err, ErrInvalidPolicy) {
		t.Fatalf("expected ErrInvalidPolicy, got %v", err)
	}
	if _, err := Evaluate(validPolicy, LimitKey{}, 0, now); !errors.Is(err, ErrInvalidLimitKey) {
		t.Fatalf("expected ErrInvalidLimitKey, got %v", err)
	}
	if _, err := Evaluate(validPolicy, validKey, 0, time.Time{}); !errors.Is(err, ErrInvalidDecision) {
		t.Fatalf("expected ErrInvalidDecision for zero now, got %v", err)
	}
}

func TestEvaluate_BoundaryValues(t *testing.T) {
	policy := Policy{Window: 10 * time.Second, Limit: 5}
	key := LimitKey{Subject: "svc-1", Operation: "create_project"}
	now := time.Unix(1_700_000_000, 0)

	cases := []struct {
		name             string
		requestsInWindow uint64
		wantAllowed      bool
		wantRemaining    uint64
	}{
		{name: "limit-1", requestsInWindow: 4, wantAllowed: true, wantRemaining: 1},
		{name: "limit", requestsInWindow: 5, wantAllowed: false, wantRemaining: 0},
		{name: "limit+1", requestsInWindow: 6, wantAllowed: false, wantRemaining: 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			decision, err := Evaluate(policy, key, tc.requestsInWindow, now)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if decision.Allowed != tc.wantAllowed {
				t.Fatalf("expected Allowed=%v, got %v", tc.wantAllowed, decision.Allowed)
			}
			if decision.Remaining != tc.wantRemaining {
				t.Fatalf("expected Remaining=%d, got %d", tc.wantRemaining, decision.Remaining)
			}
			if decision.Limit != policy.Limit {
				t.Fatalf("expected Limit=%d, got %d", policy.Limit, decision.Limit)
			}
			if tc.wantAllowed {
				if decision.RetryAfter != 0 {
					t.Fatalf("expected RetryAfter=0 when allowed, got %v", decision.RetryAfter)
				}
			} else {
				if decision.RetryAfter <= 0 {
					t.Fatalf("expected positive RetryAfter when denied, got %v", decision.RetryAfter)
				}
			}
		})
	}
}

func TestDecision_Validate(t *testing.T) {
	cases := []struct {
		name     string
		decision Decision
		wantErr  bool
	}{
		{name: "valid allowed", decision: Decision{Allowed: true, RetryAfter: 0, Limit: 5, Remaining: 1}, wantErr: false},
		{name: "valid denied", decision: Decision{Allowed: false, RetryAfter: time.Second, Limit: 5, Remaining: 0}, wantErr: false},
		{name: "valid denied zero retry", decision: Decision{Allowed: false, RetryAfter: 0, Limit: 5, Remaining: 0}, wantErr: false},
		{name: "allowed with non-zero retry", decision: Decision{Allowed: true, RetryAfter: time.Second, Limit: 5, Remaining: 1}, wantErr: true},
		{name: "denied with negative retry", decision: Decision{Allowed: false, RetryAfter: -time.Second, Limit: 5, Remaining: 0}, wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.decision.Validate()
			if tc.wantErr {
				if !errors.Is(err, ErrInvalidDecision) {
					t.Fatalf("expected ErrInvalidDecision, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestEvaluate_IsDeterministic(t *testing.T) {
	policy := Policy{Window: 30 * time.Second, Limit: 3}
	key := LimitKey{Subject: "svc-1", Operation: "create_project"}
	now := time.Unix(1_700_000_000, 0)

	first, err := Evaluate(policy, key, 3, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for i := 0; i < 10; i++ {
		again, err := Evaluate(policy, key, 3, now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if again != first {
			t.Fatalf("expected identical Decision on repeated calls, got %+v vs %+v", again, first)
		}
	}
}
