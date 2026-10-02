// Package commandendpoint implements the Release A Foundation Command
// endpoint boundary (A10-004). It composes the existing Public API command
// envelope, Authorization -> Policy -> Execution gate, and Inbox idempotency
// contract without introducing persistence or infrastructure dependencies.
package commandendpoint

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/ControlCenterSoft/aidi_2.0/internal/apicommand"
	"github.com/ControlCenterSoft/aidi_2.0/internal/authzpipeline"
	"github.com/ControlCenterSoft/aidi_2.0/internal/inbox"
	"github.com/ControlCenterSoft/aidi_2.0/internal/toolregistry"
)

var (
	// ErrForbidden is returned when the Authorization -> Policy -> Execution
	// pipeline denies a command. Authorization is evaluated before the Inbox
	// so a duplicate command never bypasses a newly-effective denial.
	ErrForbidden = errors.New("foundation command endpoint: forbidden")

	// ErrInvalidService is returned when required endpoint dependencies were
	// not configured.
	ErrInvalidService = errors.New("foundation command endpoint: invalid service")

	// ErrInvalidExecutionResult is returned when an executor returns bytes
	// that cannot be represented as a JSON response.
	ErrInvalidExecutionResult = errors.New("foundation command endpoint: invalid execution result")
)

// AuthorizationRequestBuilder resolves the effective server-side
// authorization/policy inputs for a validated command. Clients do not provide
// the effective policy directly; the caller wiring this endpoint owns that
// lookup.
type AuthorizationRequestBuilder func(context.Context, apicommand.MutationCommand) (authzpipeline.Request, error)

// Executor applies one authorized command. GitHub/HTTP transport concerns are
// intentionally outside this callback. The returned payload must be valid JSON
// because it is persisted by the Inbox as the logical result for CommandID.
type Executor func(context.Context, apicommand.MutationCommand) (json.RawMessage, error)

// Service is the transport-neutral Foundation Command endpoint service.
//
// The order is security-critical:
//  1. validate command metadata;
//  2. resolve and enforce Authorization -> Policy -> Execution;
//  3. enter Inbox.Once keyed by CommandID;
//  4. invoke Executor only for the first logical execution.
//
// Therefore duplicate Command IDs cannot repeat a state-changing operation,
// and idempotency never bypasses an authorization/policy decision.
type Service struct {
	Inbox              inbox.Store
	Registry           *toolregistry.Registry
	BuildAuthorization AuthorizationRequestBuilder
	Execute            Executor
}

// Result is returned for both a first execution and a duplicate replay.
// Duplicate reports whether this call observed an already-recorded result.
type Result struct {
	CommandID string          `json:"command_id"`
	Duplicate bool            `json:"duplicate"`
	Result    json.RawMessage `json:"result"`
}

// ExecuteCommand validates, authorizes and idempotently executes command.
func (s Service) ExecuteCommand(ctx context.Context, command apicommand.MutationCommand) (Result, error) {
	if err := command.Validate(); err != nil {
		return Result{}, err
	}
	if s.Inbox == nil {
		return Result{}, fmt.Errorf("%w: inbox is required", ErrInvalidService)
	}
	if s.Registry == nil {
		return Result{}, fmt.Errorf("%w: tool registry is required", ErrInvalidService)
	}
	if s.BuildAuthorization == nil {
		return Result{}, fmt.Errorf("%w: authorization request builder is required", ErrInvalidService)
	}
	if s.Execute == nil {
		return Result{}, fmt.Errorf("%w: executor is required", ErrInvalidService)
	}

	authRequest, err := s.BuildAuthorization(ctx, command)
	if err != nil {
		return Result{}, err
	}
	decision, err := authzpipeline.Authorize(s.Registry, authRequest)
	if err != nil {
		return Result{}, err
	}
	if !decision.Allowed {
		return Result{}, fmt.Errorf("%w: %s", ErrForbidden, decision.Reason)
	}

	record, produced, err := s.Inbox.Once(ctx, command.CommandID, func(ctx context.Context) ([]byte, error) {
		payload, err := s.Execute(ctx, command)
		if err != nil {
			return nil, err
		}
		if len(payload) == 0 {
			payload = json.RawMessage("null")
		}
		if !json.Valid(payload) {
			return nil, ErrInvalidExecutionResult
		}
		return append([]byte(nil), payload...), nil
	})
	if err != nil {
		return Result{}, err
	}

	return Result{
		CommandID: record.CommandID,
		Duplicate: !produced,
		Result:    append(json.RawMessage(nil), record.Result...),
	}, nil
}

// Handler exposes Service as a minimal JSON HTTP endpoint. Runtime wiring owns
// authentication and the BuildAuthorization implementation; this handler never
// accepts an effective policy from the request body.
func Handler(service Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "POST is required", "")
			return
		}

		var command apicommand.MutationCommand
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&command); err != nil {
			writeError(w, http.StatusBadRequest, "VALIDATION", "invalid command JSON", "")
			return
		}
		var trailing any
		if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
			writeError(w, http.StatusBadRequest, "VALIDATION", "request body must contain exactly one JSON value", command.CorrelationID)
			return
		}

		result, err := service.ExecuteCommand(r.Context(), command)
		if err != nil {
			switch {
			case errors.Is(err, apicommand.ErrInvalidMutationCommand):
				writeError(w, http.StatusBadRequest, "VALIDATION", err.Error(), command.CorrelationID)
			case errors.Is(err, ErrForbidden):
				writeError(w, http.StatusForbidden, "FORBIDDEN", err.Error(), command.CorrelationID)
			default:
				writeError(w, http.StatusInternalServerError, "INTERNAL", "command execution failed", command.CorrelationID)
			}
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(result)
	})
}

type errorResponse struct {
	Code          string `json:"code"`
	Message       string `json:"message"`
	CorrelationID string `json:"correlation_id,omitempty"`
}

func writeError(w http.ResponseWriter, status int, code, message, correlationID string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorResponse{
		Code:          code,
		Message:       message,
		CorrelationID: correlationID,
	})
}
