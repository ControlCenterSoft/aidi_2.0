// Package apiquery defines transport-independent Public API read-side
// Query contracts required by SPEC §13.1: "Commands отделены от Queries;
// поддерживаются idempotency command IDs, correlation, optimistic
// concurrency, structured errors и rate limits." Release A already
// implements the write side (internal/apicommand.MutationCommand) and the
// structured error side (internal/apierror); this package closes the
// read-side gap with the same style/conventions.
//
// A Query is read-only: it carries no ExpectedRevision/optimistic-
// concurrency input (that is a write-path concern owned by apicommand) and
// must never trigger a canonical state mutation or side effect.
package apiquery

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// ErrInvalidQuery is returned (wrapped) when a query does not carry the
// identity and scope metadata required by SPEC §13.1.
var ErrInvalidQuery = errors.New("invalid canonical query")

// Target identifies the canonical object, or collection scope, a query
// reads from.
type Target struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

// Validate checks that a target can be resolved without transport-specific
// assumptions. Both Kind and ID are required, mirroring
// apicommand.Target.Validate: a Query always names a concrete scope it
// reads from, even when that scope is a collection (ID then identifies the
// collection, e.g. a project id whose child list is being read).
func (t Target) Validate() error {
	if strings.TrimSpace(t.Kind) == "" {
		return fmt.Errorf("%w: target kind is required", ErrInvalidQuery)
	}
	if strings.TrimSpace(t.ID) == "" {
		return fmt.Errorf("%w: target id is required", ErrInvalidQuery)
	}
	return nil
}

// MaxPageLimit is the documented upper bound for Page.Limit. It exists so
// callers and read-model handlers share one authoritative cap without
// requiring HTTP framework or persistence-layer coordination.
const MaxPageLimit = 500

// Page is the caller-supplied pagination request for a list-shaped Query.
// Limit bounds the number of items a single page may return (must be
// non-zero and no greater than MaxPageLimit). Cursor is an opaque token
// identifying the position to resume from; an empty Cursor means "first
// page". This contract does not define how a Cursor is encoded/decoded —
// that remains a later Release A/B slice.
type Page struct {
	Limit  uint32 `json:"limit"`
	Cursor string `json:"cursor,omitempty"`
}

// Validate rejects a zero Limit and a Limit above MaxPageLimit. Cursor has
// no required shape: both empty (first page) and non-empty values are
// valid here.
func (p Page) Validate() error {
	if p.Limit == 0 {
		return fmt.Errorf("%w: page limit must be greater than zero", ErrInvalidQuery)
	}
	if p.Limit > MaxPageLimit {
		return fmt.Errorf("%w: page limit must not exceed %d", ErrInvalidQuery, MaxPageLimit)
	}
	return nil
}

// PageResult is the deterministic pagination outcome returned alongside a
// list-shaped Query's results. NextCursor is the opaque token to pass as
// the next Page.Cursor, empty when there is no further page. HasMore
// reports whether another page is available.
type PageResult struct {
	NextCursor string `json:"next_cursor,omitempty"`
	HasMore    bool   `json:"has_more"`
}

// Validate enforces the HasMore/NextCursor invariant: HasMore == true
// requires a non-empty NextCursor, and HasMore == false requires an empty
// NextCursor.
func (r PageResult) Validate() error {
	if r.HasMore && strings.TrimSpace(r.NextCursor) == "" {
		return fmt.Errorf("%w: next cursor is required when has more is true", ErrInvalidQuery)
	}
	if !r.HasMore && strings.TrimSpace(r.NextCursor) != "" {
		return fmt.Errorf("%w: next cursor must be empty when has more is false", ErrInvalidQuery)
	}
	return nil
}

// Query is the Public API read-side envelope for a bounded read against
// canonical state.
//
// QueryID is a stable identity used for tracing and dedup of read
// requests. CorrelationID is mandatory and ties the query to
// apierror.Error.CorrelationID so any resulting structured error can be
// traced back to the request that produced it. Target identifies the
// object or collection scope being read. Operation is the named read
// operation. Parameters is a deliberately opaque payload validated by the
// concrete read-model handler in a later slice. Page is an optional
// pagination request for list-shaped operations; it is nil for
// non-list/single-item operations and is validated when present.
type Query struct {
	QueryID       string          `json:"query_id"`
	CorrelationID string          `json:"correlation_id"`
	Target        Target          `json:"target"`
	Operation     string          `json:"operation"`
	Parameters    json.RawMessage `json:"parameters,omitempty"`
	Page          *Page           `json:"page,omitempty"`
}

// Validate enforces the metadata required before a query can reach a
// read-model handler. This slice deliberately does not execute the
// operation, resolve rate limits, or shape a result projection; those
// belong to later Release A/B slices layered on top of this contract.
func (q Query) Validate() error {
	if strings.TrimSpace(q.QueryID) == "" {
		return fmt.Errorf("%w: query id is required", ErrInvalidQuery)
	}
	if strings.TrimSpace(q.CorrelationID) == "" {
		return fmt.Errorf("%w: correlation id is required", ErrInvalidQuery)
	}
	if err := q.Target.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(q.Operation) == "" {
		return fmt.Errorf("%w: operation is required", ErrInvalidQuery)
	}
	if q.Page != nil {
		if err := q.Page.Validate(); err != nil {
			return err
		}
	}
	return nil
}
