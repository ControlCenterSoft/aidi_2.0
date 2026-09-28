package canonical

import (
	"errors"
	"testing"

	"github.com/ControlCenterSoft/aidi_2.0/internal/orchestration"
)

func testTaskAttempt(id string) orchestration.AttemptBinding {
	return orchestration.AttemptBinding{
		AttemptID:   orchestration.AttemptID(id),
		SourceSHA:   "0123456789abcdef",
		WorkspaceID: "workspace-" + id,
		ExecutorID:  "runner-" + id,
	}
}

func baseTask() Task {
	return Task{
		TaskID: "task-1",
		State:  TaskPending,
	}
}

func TestTaskValidateRejectsMissingTaskID(t *testing.T) {
	task := baseTask()
	task.TaskID = ""

	if err := task.Validate(); err == nil {
		t.Fatal("expected error for missing task id")
	}
}

func TestTaskValidateRejectsUnknownState(t *testing.T) {
	task := baseTask()
	task.State = TaskState("BOGUS")

	if err := task.Validate(); err == nil {
		t.Fatal("expected error for unknown state")
	}
}

func TestTaskValidateRejectsRunningWithoutAttemptBinding(t *testing.T) {
	task := baseTask()
	task.State = TaskRunning
	task.Attempt = nil

	if err := task.Validate(); err == nil {
		t.Fatal("expected error for RUNNING without attempt binding")
	}
}

func TestTaskValidateRejectsRunningWithInvalidAttemptBinding(t *testing.T) {
	task := baseTask()
	task.State = TaskRunning
	invalid := orchestration.AttemptBinding{}
	task.Attempt = &invalid

	if err := task.Validate(); err == nil {
		t.Fatal("expected error for RUNNING with zero-value attempt binding")
	}
}

func TestTaskValidateAcceptsRunningWithValidAttemptBinding(t *testing.T) {
	task := baseTask()
	task.State = TaskRunning
	attempt := testTaskAttempt("attempt-1")
	task.Attempt = &attempt

	if err := task.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTaskValidateRejectsDoneWithoutEvidence(t *testing.T) {
	task := baseTask()
	task.State = TaskDone
	task.Evidence = nil

	if err := task.Validate(); err == nil {
		t.Fatal("expected error for DONE without evidence")
	}
}

func TestTaskValidateRejectsDoneWithEmptyEvidenceSlice(t *testing.T) {
	// A model/agent asserting "done" without an actual evidence reference
	// must never satisfy the invariant (ACCEPTANCE.md AC-EXEC-004).
	task := baseTask()
	task.State = TaskDone
	task.Evidence = []EvidenceRef{}

	if err := task.Validate(); err == nil {
		t.Fatal("expected error for DONE with empty evidence slice")
	}
}

func TestTaskValidateAcceptsDoneWithEvidence(t *testing.T) {
	task := baseTask()
	task.State = TaskDone
	task.Evidence = []EvidenceRef{{Kind: "unit-test", ID: "run-1"}}

	if err := task.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTaskValidateRejectsInvalidEvidenceRef(t *testing.T) {
	cases := []EvidenceRef{
		{Kind: "", ID: "run-1"},
		{Kind: "unit-test", ID: ""},
	}
	for _, evidence := range cases {
		task := baseTask()
		task.State = TaskDone
		task.Evidence = []EvidenceRef{evidence}

		if err := task.Validate(); err == nil {
			t.Fatalf("expected error for invalid evidence ref %+v", evidence)
		}
	}
}

func TestValidateTaskTransitionAllowsDocumentedEdges(t *testing.T) {
	allowed := []struct {
		current TaskState
		next    TaskState
	}{
		{TaskPending, TaskReady},
		{TaskReady, TaskRunning},
		{TaskRunning, TaskDone},
		{TaskRunning, TaskBlocked},
		{TaskRunning, TaskFailed},
		{TaskBlocked, TaskReady},
	}
	for _, edge := range allowed {
		if err := ValidateTaskTransition(edge.current, edge.next); err != nil {
			t.Fatalf("expected %s -> %s to be allowed, got %v", edge.current, edge.next, err)
		}
	}
}

func TestValidateTaskTransitionRejectsUndocumentedEdges(t *testing.T) {
	rejected := []struct {
		current TaskState
		next    TaskState
	}{
		{TaskPending, TaskPending},
		{TaskPending, TaskRunning},
		{TaskPending, TaskDone},
		{TaskPending, TaskBlocked},
		{TaskPending, TaskFailed},
		{TaskReady, TaskReady},
		{TaskReady, TaskPending},
		{TaskReady, TaskDone},
		{TaskReady, TaskBlocked},
		{TaskReady, TaskFailed},
		{TaskRunning, TaskRunning},
		{TaskRunning, TaskPending},
		{TaskRunning, TaskReady},
		{TaskBlocked, TaskBlocked},
		{TaskBlocked, TaskPending},
		{TaskBlocked, TaskRunning},
		{TaskBlocked, TaskDone},
		{TaskBlocked, TaskFailed},
		{TaskDone, TaskDone},
		{TaskDone, TaskPending},
		{TaskDone, TaskReady},
		{TaskDone, TaskRunning},
		{TaskDone, TaskBlocked},
		{TaskDone, TaskFailed},
		{TaskFailed, TaskFailed},
		{TaskFailed, TaskPending},
		{TaskFailed, TaskReady},
		{TaskFailed, TaskRunning},
		{TaskFailed, TaskBlocked},
		{TaskFailed, TaskDone},
	}
	for _, edge := range rejected {
		if err := ValidateTaskTransition(edge.current, edge.next); err == nil {
			t.Fatalf("expected %s -> %s to be rejected", edge.current, edge.next)
		} else if !errors.Is(err, ErrInvalidTaskTransition) {
			t.Fatalf("expected ErrInvalidTaskTransition for %s -> %s, got %v", edge.current, edge.next, err)
		}
	}
}

func TestTaskFullLifecycleValidPath(t *testing.T) {
	task := baseTask()

	ready, err := task.Transition(TaskReady)
	if err != nil {
		t.Fatalf("ready: %v", err)
	}

	attempt := testTaskAttempt("attempt-1")
	ready.Attempt = &attempt
	running, err := ready.Transition(TaskRunning)
	if err != nil {
		t.Fatalf("running: %v", err)
	}

	running.Evidence = []EvidenceRef{{Kind: "unit-test", ID: "run-1"}}
	done, err := running.Transition(TaskDone)
	if err != nil {
		t.Fatalf("done: %v", err)
	}
	if done.State != TaskDone {
		t.Fatalf("expected done state, got %s", done.State)
	}
}

func TestTaskLifecycleRetryBindsNewAttemptWithoutChangingTaskID(t *testing.T) {
	// A Task can outlive multiple Attempts: replacing the bound Attempt on
	// retry (e.g. BLOCKED -> READY -> RUNNING again) does not change the
	// TaskID or the transition rules.
	task := baseTask()

	ready, err := task.Transition(TaskReady)
	if err != nil {
		t.Fatalf("ready: %v", err)
	}

	firstAttempt := testTaskAttempt("attempt-1")
	ready.Attempt = &firstAttempt
	running, err := ready.Transition(TaskRunning)
	if err != nil {
		t.Fatalf("running: %v", err)
	}

	blocked, err := running.Transition(TaskBlocked)
	if err != nil {
		t.Fatalf("blocked: %v", err)
	}
	if blocked.TaskID != task.TaskID {
		t.Fatalf("expected task id to remain %s, got %s", task.TaskID, blocked.TaskID)
	}

	readyAgain, err := blocked.Transition(TaskReady)
	if err != nil {
		t.Fatalf("ready again: %v", err)
	}

	secondAttempt := testTaskAttempt("attempt-2")
	readyAgain.Attempt = &secondAttempt
	runningAgain, err := readyAgain.Transition(TaskRunning)
	if err != nil {
		t.Fatalf("running again: %v", err)
	}
	if runningAgain.TaskID != task.TaskID {
		t.Fatalf("expected task id to remain %s, got %s", task.TaskID, runningAgain.TaskID)
	}
	if runningAgain.Attempt.AttemptID != secondAttempt.AttemptID {
		t.Fatalf("expected second attempt to be bound, got %+v", runningAgain.Attempt)
	}
}

func TestTaskValidateRejectsRunningWithoutEvidenceRequirement(t *testing.T) {
	// RUNNING must not require evidence: evidence is only mandatory at DONE.
	task := baseTask()
	task.State = TaskRunning
	attempt := testTaskAttempt("attempt-1")
	task.Attempt = &attempt
	task.Evidence = nil

	if err := task.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTaskTransitionRejectsMissingEvidenceReachingDone(t *testing.T) {
	task := baseTask()
	task.State = TaskRunning
	attempt := testTaskAttempt("attempt-1")
	task.Attempt = &attempt

	if _, err := task.Transition(TaskDone); err == nil {
		t.Fatal("expected error transitioning to DONE without evidence")
	} else if !errors.Is(err, ErrTaskInvariant) {
		t.Fatalf("expected ErrTaskInvariant, got %v", err)
	}
}

func TestTaskTransitionRejectsRunningWithoutAttemptBinding(t *testing.T) {
	task := baseTask()
	task.State = TaskReady

	if _, err := task.Transition(TaskRunning); err == nil {
		t.Fatal("expected error transitioning to RUNNING without attempt binding")
	} else if !errors.Is(err, ErrTaskInvariant) {
		t.Fatalf("expected ErrTaskInvariant, got %v", err)
	}
}

func TestValidateTaskTransitionRejectsOutOfTerminalStates(t *testing.T) {
	for _, terminal := range []TaskState{TaskDone, TaskFailed} {
		for _, next := range []TaskState{TaskPending, TaskReady, TaskRunning, TaskBlocked, TaskDone, TaskFailed} {
			if err := ValidateTaskTransition(terminal, next); err == nil {
				t.Fatalf("expected %s -> %s to be rejected", terminal, next)
			}
		}
	}
}
