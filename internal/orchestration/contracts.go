package orchestration

import (
	"context"
	"errors"
)

type CommandState string

const (
	CommandPending      CommandState = "PENDING"
	CommandDispatched   CommandState = "DISPATCHED"
	CommandAcknowledged CommandState = "ACKNOWLEDGED"
	CommandApplied      CommandState = "APPLIED"
	CommandRejected     CommandState = "REJECTED"
)

var ErrInvalidCommandTransition = errors.New("invalid command lifecycle transition")

type WorkflowCommand struct {
	CommandID     string
	CorrelationID string
	CausationID   string
	WorkflowID    WorkflowID
	Attempt       AttemptBinding
	FenceToken    FenceToken
	Name          string
	Payload       []byte
}

func (c WorkflowCommand) Validate() error {
	if c.CommandID == "" || c.WorkflowID == "" || c.FenceToken == 0 || c.Name == "" {
		return ErrOwnershipInvariant
	}
	return c.Attempt.Validate()
}

type WorkflowEvent struct {
	EventID       string
	CorrelationID string
	CausationID   string
	WorkflowID    WorkflowID
	AttemptID     AttemptID
	FenceToken    FenceToken
	Type          string
	Payload       []byte
}

func (e WorkflowEvent) Validate() error {
	if e.EventID == "" || e.WorkflowID == "" || e.AttemptID == "" || e.FenceToken == 0 || e.Type == "" {
		return ErrOwnershipInvariant
	}
	return nil
}

// DurableWorkflowEngine is the adapter boundary implemented by a durable
// workflow runtime such as Temporal. Canonical state remains outside the
// workflow transport and must be validated by authoritative controllers.
type DurableWorkflowEngine interface {
	Dispatch(context.Context, WorkflowCommand) error
	RequestReconciliation(context.Context, WorkflowID) error
}

// EventPublisher is the at-least-once event delivery boundary implemented by
// a bus adapter such as NATS JetStream. Consumers must deduplicate by EventID.
type EventPublisher interface {
	Publish(context.Context, WorkflowEvent) error
}

func ValidateCommandTransition(current, next CommandState) error {
	if current == next {
		return nil
	}

	switch current {
	case CommandPending:
		if next == CommandDispatched || next == CommandRejected {
			return nil
		}
	case CommandDispatched:
		if next == CommandAcknowledged || next == CommandRejected {
			return nil
		}
	case CommandAcknowledged:
		if next == CommandApplied || next == CommandRejected {
			return nil
		}
	case CommandApplied, CommandRejected:
		return ErrInvalidCommandTransition
	}

	return ErrInvalidCommandTransition
}
