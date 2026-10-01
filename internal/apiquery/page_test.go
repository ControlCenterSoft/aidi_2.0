package apiquery

import (
	"errors"
	"testing"
)

func TestPageValidate(t *testing.T) {
	tests := []struct {
		name    string
		page    Page
		wantErr bool
	}{
		{"valid without cursor", Page{Limit: 10}, false},
		{"valid with cursor", Page{Limit: 10, Cursor: "cursor-abc"}, false},
		{"valid at max limit", Page{Limit: MaxPageLimit}, false},
		{"zero limit", Page{Limit: 0}, true},
		{"over max limit", Page{Limit: MaxPageLimit + 1}, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.page.Validate()
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected validation error")
				}
				if !errors.Is(err, ErrInvalidQuery) {
					t.Fatalf("expected ErrInvalidQuery, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("valid page rejected: %v", err)
			}
		})
	}
}

func TestPageResultValidate(t *testing.T) {
	tests := []struct {
		name    string
		result  PageResult
		wantErr bool
	}{
		{"no more pages", PageResult{HasMore: false, NextCursor: ""}, false},
		{"has more with cursor", PageResult{HasMore: true, NextCursor: "cursor-next"}, false},
		{"has more without cursor", PageResult{HasMore: true, NextCursor: ""}, true},
		{"no more with cursor", PageResult{HasMore: false, NextCursor: "cursor-next"}, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.result.Validate()
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected validation error")
				}
				if !errors.Is(err, ErrInvalidQuery) {
					t.Fatalf("expected ErrInvalidQuery, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("valid page result rejected: %v", err)
			}
		})
	}
}

func TestQueryValidateWithPage(t *testing.T) {
	t.Run("nil page is unaffected", func(t *testing.T) {
		query := validQuery()
		query.Page = nil

		if err := query.Validate(); err != nil {
			t.Fatalf("query without page rejected: %v", err)
		}
	})

	t.Run("valid page", func(t *testing.T) {
		query := validQuery()
		query.Page = &Page{Limit: 25}

		if err := query.Validate(); err != nil {
			t.Fatalf("query with valid page rejected: %v", err)
		}
	})

	t.Run("invalid page propagates wrapped error", func(t *testing.T) {
		query := validQuery()
		query.Page = &Page{Limit: 0}

		err := query.Validate()
		if err == nil {
			t.Fatal("expected validation error")
		}
		if !errors.Is(err, ErrInvalidQuery) {
			t.Fatalf("expected ErrInvalidQuery, got %v", err)
		}
	})
}
