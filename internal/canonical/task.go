package canonical

import (
	"errors"
	"fmt"
	"strings"

	"github.com/ControlCenterSoft/aidi_2.0/internal/orchestration"
)

// TaskState models the SPEC §2.2/§8.1 rule that Task and Attempt are
// separate entities: a Task's lifecycle is tracked independently of any
// particular Attempt that is currently executing it (ACCEPTANCE.md
// AC-EXEC-001).
type TaskState string

const (
	TaskPending TaskState = "PENDING"
	TaskReady   TaskState = "READY"
	TaskRunning TaskState = "RUNNING"
	TaskDone    TaskState = "DONE"
	TaskBlocked TaskState = "BLOCKED"
	TaskFailed  TaskState = "FAILED"
)

var (
	ErrInvalidTaskTransition = errors.New("invalid task lifecycle transition")
	ErrTaskInvariant         = errors.New("task invariant violation")
)

// Task is the SPEC §2.2/§8.1 canonical unit of work. It is a distinct
// entity from any orchestration.AttemptBinding executing it: RUNNING is only
// reachable while a real executor/attempt lease is bound
// (ACCEPTANCE.md AC-EXEC-001), and a Task may outlive many Attempts as
// retries are bound and replaced without changing the Task's identity or
// transition rules.
//
// DONE additionally requires at least one EvidenceRef: an LLM/agent "done"
// claim alone must never satisfy completion (ACCEPTANCE.md AC-EXEC-004).
type Task struct {
	TaskID   string                        `json:"task_id"`
	State    TaskState                     `json:"state"`
	Attempt  *orchestration.AttemptBinding `json:"attempt,omitempty"`
	Evidence []EvidenceRef                 `json:"evidence,omitempty"`
}

// Validate checks the structural invariants of the Task: a required
// TaskID, a known lifecycle state, a valid non-empty AttemptBinding while
// RUNNING, and at least one evidence reference once DONE.
func (t Task) Validate() error {
	if strings.TrimSpace(t.TaskID) == "" {
		return fmt.Errorf("%w: task id is required", ErrTaskInvariant)
	}
	if err := t.validateKnownState(); err != nil {
		return err
	}
	if t.State == TaskRunning {
		if t.Attempt == nil {
			return fmt.Errorf("%w: %s requires a bound attempt", ErrTaskInvariant, TaskRunning)
		}
		if err := t.Attempt.Validate(); err != nil {
			return fmt.Errorf("%w: %s", ErrTaskInvariant, err)
		}
	}
	if t.State == TaskDone {
		if len(t.Evidence) == 0 {
			return fmt.Errorf("%w: %s requires at least one evidence reference", ErrTaskInvariant, TaskDone)
		}
	}
	for _, evidence := range t.Evidence {
		if err := evidence.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (t Task) validateKnownState() error {
	switch t.State {
	case TaskPending, TaskReady, TaskRunning, TaskDone, TaskBlocked, TaskFailed:
		return nil
	default:
		return fmt.Errorf("%w: unknown state %q", ErrTaskInvariant, t.State)
	}
}

// ValidateTransition allows only the documented lifecycle edges:
//
//	PENDING -> READY
//	READY   -> RUNNING
//	RUNNING -> DONE | BLOCKED | FAILED
//	BLOCKED -> READY
//
// DONE and FAILED are terminal: no transition out of them, including to the
// same state, is allowed.
func ValidateTaskTransition(current, next TaskState) error {
	switch current {
	case TaskPending:
		if next == TaskReady {
			return nil
		}
	case TaskReady:
		if next == TaskRunning {
			return nil
		}
	case TaskRunning:
		if next == TaskDone || next == TaskBlocked || next == TaskFailed {
			return nil
		}
	case TaskBlocked:
		if next == TaskReady {
			return nil
		}
	case TaskDone, TaskFailed:
		// terminal: no outgoing transitions, not even a same-state no-op.
	}
	return fmt.Errorf("%w: %s -> %s", ErrInvalidTaskTransition, current, next)
}

// Transition moves the Task to next if the edge is allowed and the
// resulting Task satisfies Validate (e.g. an attempt binding is required to
// reach RUNNING, evidence is required to reach DONE).
func (t Task) Transition(next TaskState) (Task, error) {
	if err := ValidateTaskTransition(t.State, next); err != nil {
		return Task{}, err
	}
	updated := t
	updated.State = next
	if err := updated.Validate(); err != nil {
		return Task{}, err
	}
	return updated, nil
}
