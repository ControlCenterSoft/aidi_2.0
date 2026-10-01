// Package inbox implements the transport/storage-neutral Inbox /
// idempotency domain contract (A3-004) required by approved SPEC §4.2 /
// §8 ("Transactional Outbox предотвращает потерю committed events при
// недоступности message bus; Inbox/dedup предотвращает повторные side
// effects") and documented narratively in docs/OUTBOX_INBOX.md ("A
// duplicate delivery or previously claimed idempotency key must not
// authorize a second application of the command or side effect").
//
// Its sole hard dependency is the already-existing data-access/domain
// repository boundary package (internal/repository, A2-003); this slice
// follows the same pure-Go interfaces/types style already used there and
// in internal/orchestration. It intentionally does not depend on, and is
// not wired to, any of: internal/apicommand (Public API command
// envelopes), the Event Journal (A3-001), the atomic state+Event write
// (A3-002), the Transactional Outbox (A3-003), NATS JetStream (A3-005), or
// publisher/consumer contracts (A3-006) — those remain separate, later
// cards. A Command ID here is an opaque caller-supplied string (matching
// apicommand.MutationCommand.CommandID's shape), not a canonical.ObjectRef:
// idempotency keys are not members of the closed canonical.Kind entity
// hierarchy (SPEC §4.1), so this package defines its own minimal,
// self-contained storage contract rather than reusing
// internal/repository.Repository.
//
// This package is isolated from the current/local AIDI runtime: it has no
// dependency on any live/current AIDI database, Forgejo, local VM/runner
// infrastructure, message queue, or local state. The InMemory reference
// implementation exists only to exercise the contract's conformance in
// unit tests, not to connect to any such system.
package inbox

import (
	"context"
	"errors"
	"strings"
)

// ErrInvalidCommandID is returned when a Command ID is empty or
// whitespace-only.
var ErrInvalidCommandID = errors.New("inbox: command id is required")

// ErrNotFound is returned by Store.Get when no Record has been produced yet
// for the requested Command ID.
var ErrNotFound = errors.New("inbox: command id not recorded")

// ValidateCommandID rejects an empty or whitespace-only commandID. It is
// exported so callers (and Store implementations) can reuse the exact
// validation this package applies before touching any state.
func ValidateCommandID(commandID string) error {
	if strings.TrimSpace(commandID) == "" {
		return ErrInvalidCommandID
	}
	return nil
}

// Record is the opaque, storage-neutral representation of the single
// logical state-changing result produced for a given Command ID. Result
// encoding is owned by the caller; the Inbox boundary does not interpret
// it.
type Record struct {
	CommandID string
	Result    []byte
}

// Store is the transport/storage-neutral domain contract for the Inbox.
//
// Implementations must guarantee that, for any given commandID, the
// produce callback passed to Once is invoked at most once across all calls
// to Once for that commandID — including calls racing concurrently with
// one another — and that every caller for that commandID (the one whose
// produce call actually ran, every concurrent duplicate, and every later
// duplicate) observes the identical Record. This is what makes a duplicate
// Command ID produce exactly one logical state-changing result (A3-004
// acceptance criteria), rather than merely detecting the duplicate after
// the fact.
type Store interface {
	// Get returns the Record previously produced for commandID. It
	// returns ErrNotFound when no Record has been produced yet, and
	// returns the error from ValidateCommandID when commandID is invalid,
	// before touching any state.
	Get(ctx context.Context, commandID string) (Record, error)

	// Once ensures produce is invoked at most one time for commandID.
	//
	//   - If no Record exists yet for commandID, Once invokes produce,
	//     stores its result as the Record for commandID, and returns it
	//     with produced=true. If produce returns an error, Once returns
	//     that error unchanged, produced=false, and stores nothing: a
	//     failed attempt is not a logical state-changing result, so a
	//     later call for the same commandID may still run produce.
	//   - If a Record already exists for commandID (including one
	//     established by a concurrent caller that won the race to
	//     produce it first), Once does not invoke produce again; it
	//     returns the existing Record unchanged with produced=false.
	//
	// It returns the error from ValidateCommandID when commandID is
	// invalid, before touching any state or invoking produce.
	Once(ctx context.Context, commandID string, produce func(ctx context.Context) ([]byte, error)) (record Record, produced bool, err error)
}

// Execute is a convenience wrapper over Store.Once for callers that only
// need the single logical Record a Command ID produces and do not need to
// distinguish "this call produced it" from "this call observed a duplicate
// Command ID". It performs no decoding/interpretation of the produced or
// returned Result: that remains the caller's responsibility, exactly as
// with Store.Once.
func Execute(ctx context.Context, store Store, commandID string, produce func(ctx context.Context) ([]byte, error)) (Record, error) {
	record, _, err := store.Once(ctx, commandID, produce)
	return record, err
}
