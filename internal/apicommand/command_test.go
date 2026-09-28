package apicommand

import (
	"errors"
	"testing"

	"github.com/ControlCenterSoft/aidi_2.0/internal/canonical"
)

func validCommand() MutationCommand {
	return MutationCommand{
		CommandID: "cmd-123",
		CorrelationID: "corr-456",
		Target: Target{Kind: "project", ID: "project-789"},
		Operation: "rename",
		ExpectedRevision: canonical.Revision(7),
	}
}

func TestMutationCommandValidate(t *testing.T) {
	if err := validCommand().Validate(); err != nil {
		t.Fatalf("valid command rejected: %v", err)
	}
}

func TestMutationCommandValidateRejectsMissingInvariants(t *testing.T) {
	tests := []struct{
		name string
		mutate func(*MutationCommand)
	}{
		{"command id", func(c *MutationCommand){ c.CommandID = " " }},
		{"correlation id", func(c *MutationCommand){ c.CorrelationID = "" }},
		{"target kind", func(c *MutationCommand){ c.Target.Kind = " " }},
		{"target id", func(c *MutationCommand){ c.Target.ID = "" }},
		{"operation", func(c *MutationCommand){ c.Operation = " " }},
		{"expected revision", func(c *MutationCommand){ c.ExpectedRevision = 0 }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			command := validCommand()
			tc.mutate(&command)
			err := command.Validate()
			if err == nil { t.Fatal("expected validation error") }
			if !errors.Is(err, ErrInvalidMutationCommand) {
				t.Fatalf("expected ErrInvalidMutationCommand, got %v", err)
			}
		})
	}
}
