package canonical

import (
	"strings"
	"testing"
	"time"
)

func testOutboxEvent() Event {
	event := validEvent()
	event.ID = "evt-outbox-001"
	event.Timestamp = time.Date(2026, 9, 26, 19, 0, 0, 0, time.UTC)
	return event
}

func TestOutboxRecordValidation(t *testing.T) {
	record, err := NewOutboxRecord(OutboxRecord{
		ID:        "outbox-001",
		Event:     testOutboxEvent(),
		CreatedAt: time.Date(2026, 9, 26, 19, 0, 1, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	if record.ID != "outbox-001" {
		t.Fatalf("unexpected record: %+v", record)
	}

	published := time.Date(2026, 9, 26, 18, 59, 0, 0, time.UTC)
	_, err = NewOutboxRecord(OutboxRecord{
		ID:          "outbox-002",
		Event:       testOutboxEvent(),
		CreatedAt:   time.Date(2026, 9, 26, 19, 0, 0, 0, time.UTC),
		PublishedAt: &published,
	})
	if err == nil || !strings.Contains(err.Error(), "cannot precede") {
		t.Fatalf("expected timestamp ordering error, got %v", err)
	}
}

func TestDuplicateDeliveryDoesNotAuthorizeReapplication(t *testing.T) {
	received := time.Date(2026, 9, 26, 19, 1, 0, 0, time.UTC)
	existing, err := NewInboxRecord(InboxRecord{
		Delivery:       DeliveryKey{Consumer: "planner", MessageID: "msg-001"},
		IdempotencyKey: "cmd-001",
		ReceivedAt:     received,
	})
	if err != nil {
		t.Fatal(err)
	}

	duplicate := existing
	duplicate.ReceivedAt = received.Add(time.Second)

	got, err := DuplicateDelivery(&existing, duplicate)
	if err != nil {
		t.Fatal(err)
	}
	if !got {
		t.Fatal("duplicate delivery must not authorize a second application")
	}
}

func TestSameIdempotencyKeyIsDuplicateAcrossRedeliveryID(t *testing.T) {
	received := time.Date(2026, 9, 26, 19, 2, 0, 0, time.UTC)
	existing, _ := NewInboxRecord(InboxRecord{
		Delivery:       DeliveryKey{Consumer: "planner", MessageID: "msg-001"},
		IdempotencyKey: "cmd-001",
		ReceivedAt:     received,
	})
	incoming, _ := NewInboxRecord(InboxRecord{
		Delivery:       DeliveryKey{Consumer: "planner", MessageID: "msg-002"},
		IdempotencyKey: "cmd-001",
		ReceivedAt:     received.Add(time.Second),
	})

	got, err := DuplicateDelivery(&existing, incoming)
	if err != nil {
		t.Fatal(err)
	}
	if !got {
		t.Fatal("same idempotency key must be treated as duplicate")
	}
}

func TestDistinctDeliveryMayProceed(t *testing.T) {
	received := time.Date(2026, 9, 26, 19, 3, 0, 0, time.UTC)
	existing, _ := NewInboxRecord(InboxRecord{
		Delivery:       DeliveryKey{Consumer: "planner", MessageID: "msg-001"},
		IdempotencyKey: "cmd-001",
		ReceivedAt:     received,
	})
	incoming, _ := NewInboxRecord(InboxRecord{
		Delivery:       DeliveryKey{Consumer: "planner", MessageID: "msg-002"},
		IdempotencyKey: "cmd-002",
		ReceivedAt:     received.Add(time.Second),
	})

	got, err := DuplicateDelivery(&existing, incoming)
	if err != nil {
		t.Fatal(err)
	}
	if got {
		t.Fatal("distinct delivery incorrectly classified as duplicate")
	}
}

func TestDuplicateSideEffect(t *testing.T) {
	now := time.Date(2026, 9, 26, 19, 4, 0, 0, time.UTC)
	existing, err := NewSideEffectRecord(SideEffectRecord{
		IdempotencyKey: "effect-001",
		Kind:           "repository.create",
		Target:         "project-001",
		RecordedAt:     now,
	})
	if err != nil {
		t.Fatal(err)
	}
	incoming := existing
	incoming.RecordedAt = now.Add(time.Second)

	duplicate, err := DuplicateSideEffect(&existing, incoming)
	if err != nil {
		t.Fatal(err)
	}
	if !duplicate {
		t.Fatal("recorded side effect must block duplicate execution")
	}
}
