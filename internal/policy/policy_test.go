package policy

import (
	"errors"
	"testing"
)

func TestPolicyID_Validate(t *testing.T) {
	cases := []struct {
		name    string
		id      PolicyID
		wantErr bool
	}{
		{name: "valid", id: "policy-1", wantErr: false},
		{name: "empty", id: "", wantErr: true},
		{name: "whitespace only", id: "   ", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.id.Validate()
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if !errors.Is(err, ErrInvalidPolicyID) {
					t.Fatalf("expected ErrInvalidPolicyID, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestEffect_Validate(t *testing.T) {
	cases := []struct {
		name    string
		effect  Effect
		wantErr bool
	}{
		{name: "allow", effect: EffectAllow, wantErr: false},
		{name: "deny", effect: EffectDeny, wantErr: false},
		{name: "empty", effect: "", wantErr: true},
		{name: "unknown", effect: "MAYBE", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.effect.Validate()
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if !errors.Is(err, ErrInvalidEffect) {
					t.Fatalf("expected ErrInvalidEffect, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func validStatement() Statement {
	return Statement{
		Subject:  "user:alice",
		Resource: "project:1",
		Action:   "read",
		Effect:   EffectAllow,
	}
}

func TestPolicy_Validate_WellFormed(t *testing.T) {
	p := Policy{
		ID:         "policy-1",
		Statements: []Statement{validStatement()},
	}
	if err := p.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPolicy_Validate_InvalidPolicyID(t *testing.T) {
	p := Policy{
		ID:         "",
		Statements: []Statement{validStatement()},
	}
	err := p.Validate()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, ErrInvalidPolicy) {
		t.Fatalf("expected ErrInvalidPolicy, got %v", err)
	}
}

func TestPolicy_Validate_EmptyStatements(t *testing.T) {
	p := Policy{ID: "policy-1", Statements: nil}
	err := p.Validate()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, ErrInvalidPolicy) {
		t.Fatalf("expected ErrInvalidPolicy, got %v", err)
	}
}

func TestPolicy_Validate_BlankStatementFields(t *testing.T) {
	cases := []struct {
		name string
		stmt Statement
	}{
		{name: "blank subject", stmt: Statement{Subject: "", Resource: "r", Action: "a", Effect: EffectAllow}},
		{name: "whitespace subject", stmt: Statement{Subject: "   ", Resource: "r", Action: "a", Effect: EffectAllow}},
		{name: "blank resource", stmt: Statement{Subject: "s", Resource: "", Action: "a", Effect: EffectAllow}},
		{name: "blank action", stmt: Statement{Subject: "s", Resource: "r", Action: "", Effect: EffectAllow}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := Policy{ID: "policy-1", Statements: []Statement{tc.stmt}}
			err := p.Validate()
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !errors.Is(err, ErrInvalidPolicy) {
				t.Fatalf("expected ErrInvalidPolicy, got %v", err)
			}
		})
	}
}

func TestPolicy_Validate_UnknownEffect(t *testing.T) {
	p := Policy{
		ID: "policy-1",
		Statements: []Statement{
			{Subject: "s", Resource: "r", Action: "a", Effect: "MAYBE"},
		},
	}
	err := p.Validate()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, ErrInvalidPolicy) {
		t.Fatalf("expected ErrInvalidPolicy, got %v", err)
	}
}

func TestPolicy_HasExplicitDeny_AllowOnly(t *testing.T) {
	p := Policy{
		ID: "policy-1",
		Statements: []Statement{
			{Subject: "s", Resource: "r", Action: "a", Effect: EffectAllow},
		},
	}
	if p.HasExplicitDeny("s", "r", "a") {
		t.Fatalf("expected false for ALLOW-only policy")
	}
}

func TestPolicy_HasExplicitDeny_DenyOnly(t *testing.T) {
	p := Policy{
		ID: "policy-1",
		Statements: []Statement{
			{Subject: "s", Resource: "r", Action: "a", Effect: EffectDeny},
		},
	}
	if !p.HasExplicitDeny("s", "r", "a") {
		t.Fatalf("expected true for DENY policy on matching tuple")
	}
	if p.HasExplicitDeny("other", "r", "a") {
		t.Fatalf("expected false for non-matching subject")
	}
}

func TestPolicy_HasExplicitDeny_AllowAndDenySameTuple(t *testing.T) {
	p := Policy{
		ID: "policy-1",
		Statements: []Statement{
			{Subject: "s", Resource: "r", Action: "a", Effect: EffectAllow},
			{Subject: "s", Resource: "r", Action: "a", Effect: EffectDeny},
		},
	}
	// Validate does not resolve conflicts; both-present is structurally valid.
	if err := p.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !p.HasExplicitDeny("s", "r", "a") {
		t.Fatalf("expected true: explicit deny must be reported even alongside an allow")
	}
}

func TestPolicy_HasExplicitDeny_ZeroValueSafety(t *testing.T) {
	var p Policy
	if p.HasExplicitDeny("s", "r", "a") {
		t.Fatalf("expected false for zero-value Policy")
	}
}
