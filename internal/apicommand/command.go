// Package apicommand defines transport-independent Public API mutation
// command contracts. It intentionally does not depend on orchestration
// transport types: Public API commands describe caller intent against
// canonical state, while orchestration.WorkflowCommand describes durable
// workflow transport.
package apicommand

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/ControlCenterSoft/aidi_2.0/internal/canonical"
)

// ErrInvalidMutationCommand is returned (wrapped) when a mutation command
// does not carry the identity and concurrency metadata required by SPEC §13.1.
var ErrInvalidMutationCommand = errors.New("invalid canonical mutation command")

// Target identifies the canonical object a mutation is expected to change.
type Target struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

// Validate checks that a target can be resolved without transport-specific
// assumptions.
func (t Target) Validate() error {
	if strings.TrimSpace(t.Kind) == "" {
		return fmt.Errorf("%w: target kind is required", ErrInvalidMutationCommand)
	}
	if strings.TrimSpace(t.ID) == "" {
		return fmt.Errorf("%w: target id is required", ErrInvalidMutationCommand)
	}
	return nil
}

// MutationCommand is the Public API command envelope for a bounded mutation
// of existing canonical state.
//
// CommandID is the stable idempotency key required by SPEC §13.1.
// CorrelationID ties the mutation to traces/errors.
// ExpectedRevision makes optimistic concurrency explicit at the API/domain
// boundary; callers must supply the authoritative revision they observed.
// Payload is deliberately opaque here and is validated by the concrete
// operation handler in a later slice.
type MutationCommand struct {
	CommandID        string             `json:"command_id"`
	CorrelationID    string             `json:"correlation_id"`
	Target           Target             `json:"target"`
	Operation        string             `json:"operation"`
	ExpectedRevision canonical.Revision `json:"expected_revision"`
	Payload          json.RawMessage    `json:"payload,omitempty"`
}

// Validate enforces the metadata required before a command can reach a
// canonical mutation handler. This slice deliberately does not execute the
// operation or compare ExpectedRevision with storage; those belong to the
// handler/persistence boundary.
func (c MutationCommand) Validate() error {
	if strings.TrimSpace(c.CommandID) == "" {
		return fmt.Errorf("%w: command id is required", ErrInvalidMutationCommand)
	}
	if strings.TrimSpace(c.CorrelationID) == "" {
		return fmt.Errorf("%w: correlation id is required", ErrInvalidMutationCommand)
	}
	if err := c.Target.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(c.Operation) == "" {
		return fmt.Errorf("%w: operation is required", ErrInvalidMutationCommand)
	}
	if c.ExpectedRevision == 0 {
		return fmt.Errorf("%w: expected revision must be greater than zero", ErrInvalidMutationCommand)
	}
	return nil
}
