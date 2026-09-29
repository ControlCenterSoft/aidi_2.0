package apiquery

import (
	"errors"
	"testing"
)

func validQuery() Query {
	return Query{
		QueryID:       "qry-123",
		CorrelationID: "corr-456",
		Target:        Target{Kind: "project", ID: "project-789"},
		Operation:     "get",
	}
}

func TestQueryValidate(t *testing.T) {
	if err := validQuery().Validate(); err != nil {
		t.Fatalf("valid query rejected: %v", err)
	}
}

func TestQueryValidateRejectsMissingInvariants(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Query)
	}{
		{"query id", func(q *Query) { q.QueryID = " " }},
		{"correlation id", func(q *Query) { q.CorrelationID = "" }},
		{"target kind", func(q *Query) { q.Target.Kind = " " }},
		{"target id", func(q *Query) { q.Target.ID = "" }},
		{"operation", func(q *Query) { q.Operation = " " }},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			query := validQuery()
			tc.mutate(&query)

			err := query.Validate()
			if err == nil {
				t.Fatal("expected validation error")
			}
			if !errors.Is(err, ErrInvalidQuery) {
				t.Fatalf("expected ErrInvalidQuery, got %v", err)
			}
		})
	}
}

func TestQueryParametersMayBeAbsent(t *testing.T) {
	query := validQuery()
	query.Parameters = nil

	if err := query.Validate(); err != nil {
		t.Fatalf("parameter-free query rejected: %v", err)
	}
}
