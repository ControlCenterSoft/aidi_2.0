package canonical

import (
	"fmt"
	"strings"
	"time"
)

type ObjectRef struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

type Event struct {
	ID            string    `json:"id"`
	Type          string    `json:"type"`
	Object        ObjectRef `json:"object"`
	Revision      Revision  `json:"revision"`
	Timestamp     time.Time `json:"timestamp"`
	Actor         string    `json:"actor"`
	CorrelationID string    `json:"correlation_id"`
	CausationID   string    `json:"causation_id,omitempty"`
	SchemaVersion uint32    `json:"schema_version"`
}

func NewEvent(event Event) (Event, error) {
	if !event.Timestamp.IsZero() {
		event.Timestamp = event.Timestamp.UTC()
	}
	if err := event.Validate(); err != nil {
		return Event{}, err
	}
	return event, nil
}

func (e Event) Validate() error {
	switch {
	case strings.TrimSpace(e.ID) == "":
		return fmt.Errorf("event id is required")
	case strings.TrimSpace(e.Type) == "":
		return fmt.Errorf("event type is required")
	case strings.TrimSpace(e.Object.Kind) == "":
		return fmt.Errorf("event object kind is required")
	case strings.TrimSpace(e.Object.ID) == "":
		return fmt.Errorf("event object id is required")
	case e.Revision == 0:
		return fmt.Errorf("event revision must be greater than zero")
	case e.Timestamp.IsZero():
		return fmt.Errorf("event timestamp is required")
	case strings.TrimSpace(e.Actor) == "":
		return fmt.Errorf("event actor is required")
	case strings.TrimSpace(e.CorrelationID) == "":
		return fmt.Errorf("event correlation id is required")
	case e.SchemaVersion == 0:
		return fmt.Errorf("event schema version must be greater than zero")
	default:
		return nil
	}
}
