package canonical

import (
	"strings"
	"testing"
	"time"
)

func validEvent() Event {
	return Event{
		ID:            "evt-001",
		Type:          "project.created",
		Object:        ObjectRef{Kind: "project", ID: "project-001"},
		Revision:      1,
		Timestamp:     time.Date(2026, 9, 26, 18, 45, 0, 0, time.FixedZone("test", 3*60*60)),
		Actor:         "user:pavel",
		CorrelationID: "corr-001",
		SchemaVersion: 1,
	}
}

func TestNewEventNormalizesTimestampToUTC(t *testing.T) {
	event, err := NewEvent(validEvent())
	if err != nil {
		t.Fatal(err)
	}
	if event.Timestamp.Location() != time.UTC {
		t.Fatalf("expected UTC timestamp, got %s", event.Timestamp.Location())
	}
}

func TestEventValidation(t *testing.T) {
	tests := []struct {
		name string
		edit func(*Event)
		want string
	}{
		{"missing id", func(e *Event) { e.ID = "" }, "event id"},
		{"missing type", func(e *Event) { e.Type = "" }, "event type"},
		{"missing object kind", func(e *Event) { e.Object.Kind = "" }, "object kind"},
		{"missing object id", func(e *Event) { e.Object.ID = "" }, "object id"},
		{"zero revision", func(e *Event) { e.Revision = 0 }, "revision"},
		{"zero timestamp", func(e *Event) { e.Timestamp = time.Time{} }, "timestamp"},
		{"missing actor", func(e *Event) { e.Actor = "" }, "actor"},
		{"missing correlation", func(e *Event) { e.CorrelationID = "" }, "correlation"},
		{"zero schema", func(e *Event) { e.SchemaVersion = 0 }, "schema version"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := validEvent()
			tt.edit(&event)
			_, err := NewEvent(event)
			if err == nil {
				t.Fatal("expected validation error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q in error, got %q", tt.want, err)
			}
		})
	}
}

func TestRootEventMayOmitCausationID(t *testing.T) {
	event := validEvent()
	event.CausationID = ""
	if _, err := NewEvent(event); err != nil {
		t.Fatalf("root event should allow empty causation id: %v", err)
	}
}
