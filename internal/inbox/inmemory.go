package inbox

import (
	"context"
	"sync"
)

// InMemory is a minimal, non-persistent reference implementation of Store
// used only to exercise/validate conformance to the interface in unit
// tests. It is not a persistence adapter and must not be used to represent
// any durable or production storage.
//
// A single mutex guards the whole Record map, and Once holds it for the
// full duration of its call — including the produce invocation — so that,
// for a given commandID, produce runs at most once even under concurrent
// Once calls from multiple goroutines, and every caller observes the same
// Record. This mirrors internal/repository.InMemory's single-mutex
// simplicity; a production storage adapter would instead rely on a unique
// constraint / conditional insert on commandID scoped to that key alone,
// not a process-wide critical section.
type InMemory struct {
	mu      sync.Mutex
	records map[string]Record
}

// NewInMemory constructs an empty InMemory Store.
func NewInMemory() *InMemory {
	return &InMemory{records: make(map[string]Record)}
}

var _ Store = (*InMemory)(nil)

// Get implements Store.
func (m *InMemory) Get(_ context.Context, commandID string) (Record, error) {
	if err := ValidateCommandID(commandID); err != nil {
		return Record{}, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	record, ok := m.records[commandID]
	if !ok {
		return Record{}, ErrNotFound
	}
	return record, nil
}

// Once implements Store.
func (m *InMemory) Once(ctx context.Context, commandID string, produce func(ctx context.Context) ([]byte, error)) (Record, bool, error) {
	if err := ValidateCommandID(commandID); err != nil {
		return Record{}, false, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if existing, ok := m.records[commandID]; ok {
		return existing, false, nil
	}

	result, err := produce(ctx)
	if err != nil {
		return Record{}, false, err
	}

	record := Record{CommandID: commandID, Result: result}
	m.records[commandID] = record
	return record, true, nil
}
