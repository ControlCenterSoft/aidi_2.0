package canonical

import (
	"fmt"
	"strings"
	"time"
)

type DeliveryKey struct {
	Consumer  string `json:"consumer"`
	MessageID string `json:"message_id"`
}

func (k DeliveryKey) Validate() error {
	switch {
	case strings.TrimSpace(k.Consumer) == "":
		return fmt.Errorf("delivery consumer is required")
	case strings.TrimSpace(k.MessageID) == "":
		return fmt.Errorf("delivery message id is required")
	default:
		return nil
	}
}

func (k DeliveryKey) Equal(other DeliveryKey) bool {
	return k.Consumer == other.Consumer && k.MessageID == other.MessageID
}

type OutboxRecord struct {
	ID              string     `json:"id"`
	Event           Event      `json:"event"`
	CreatedAt       time.Time  `json:"created_at"`
	PublishedAt     *time.Time `json:"published_at,omitempty"`
	PublishAttempts uint32     `json:"publish_attempts"`
}

func NewOutboxRecord(record OutboxRecord) (OutboxRecord, error) {
	record.CreatedAt = record.CreatedAt.UTC()
	record.PublishedAt = utcPointer(record.PublishedAt)
	if err := record.Validate(); err != nil {
		return OutboxRecord{}, err
	}
	return record, nil
}

func (r OutboxRecord) Validate() error {
	switch {
	case strings.TrimSpace(r.ID) == "":
		return fmt.Errorf("outbox id is required")
	case r.CreatedAt.IsZero():
		return fmt.Errorf("outbox created_at is required")
	}
	if err := r.Event.Validate(); err != nil {
		return fmt.Errorf("outbox event: %w", err)
	}
	if r.PublishedAt != nil && r.PublishedAt.Before(r.CreatedAt) {
		return fmt.Errorf("outbox published_at cannot precede created_at")
	}
	return nil
}

type InboxRecord struct {
	Delivery       DeliveryKey `json:"delivery"`
	IdempotencyKey string      `json:"idempotency_key"`
	ReceivedAt     time.Time   `json:"received_at"`
	AppliedAt      *time.Time  `json:"applied_at,omitempty"`
}

func NewInboxRecord(record InboxRecord) (InboxRecord, error) {
	record.ReceivedAt = record.ReceivedAt.UTC()
	record.AppliedAt = utcPointer(record.AppliedAt)
	if err := record.Validate(); err != nil {
		return InboxRecord{}, err
	}
	return record, nil
}

func (r InboxRecord) Validate() error {
	if err := r.Delivery.Validate(); err != nil {
		return err
	}
	switch {
	case strings.TrimSpace(r.IdempotencyKey) == "":
		return fmt.Errorf("inbox idempotency key is required")
	case r.ReceivedAt.IsZero():
		return fmt.Errorf("inbox received_at is required")
	case r.AppliedAt != nil && r.AppliedAt.Before(r.ReceivedAt):
		return fmt.Errorf("inbox applied_at cannot precede received_at")
	default:
		return nil
	}
}

type SideEffectRecord struct {
	IdempotencyKey string     `json:"idempotency_key"`
	Kind           string     `json:"kind"`
	Target         string     `json:"target"`
	RecordedAt     time.Time  `json:"recorded_at"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
}

func NewSideEffectRecord(record SideEffectRecord) (SideEffectRecord, error) {
	record.RecordedAt = record.RecordedAt.UTC()
	record.CompletedAt = utcPointer(record.CompletedAt)
	if err := record.Validate(); err != nil {
		return SideEffectRecord{}, err
	}
	return record, nil
}

func (r SideEffectRecord) Validate() error {
	switch {
	case strings.TrimSpace(r.IdempotencyKey) == "":
		return fmt.Errorf("side effect idempotency key is required")
	case strings.TrimSpace(r.Kind) == "":
		return fmt.Errorf("side effect kind is required")
	case strings.TrimSpace(r.Target) == "":
		return fmt.Errorf("side effect target is required")
	case r.RecordedAt.IsZero():
		return fmt.Errorf("side effect recorded_at is required")
	case r.CompletedAt != nil && r.CompletedAt.Before(r.RecordedAt):
		return fmt.Errorf("side effect completed_at cannot precede recorded_at")
	default:
		return nil
	}
}

func DuplicateDelivery(existing *InboxRecord, incoming InboxRecord) (bool, error) {
	if err := incoming.Validate(); err != nil {
		return false, err
	}
	if existing == nil {
		return false, nil
	}
	if err := existing.Validate(); err != nil {
		return false, fmt.Errorf("existing inbox record: %w", err)
	}
	sameScopedIdempotency := existing.Delivery.Consumer == incoming.Delivery.Consumer &&
		existing.IdempotencyKey == incoming.IdempotencyKey
	return existing.Delivery.Equal(incoming.Delivery) || sameScopedIdempotency, nil
}

func DuplicateSideEffect(existing *SideEffectRecord, incoming SideEffectRecord) (bool, error) {
	if err := incoming.Validate(); err != nil {
		return false, err
	}
	if existing == nil {
		return false, nil
	}
	if err := existing.Validate(); err != nil {
		return false, fmt.Errorf("existing side effect record: %w", err)
	}
	return existing.IdempotencyKey == incoming.IdempotencyKey, nil
}

func utcPointer(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	normalized := value.UTC()
	return &normalized
}
