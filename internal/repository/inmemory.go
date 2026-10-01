package repository

import (
	"context"
	"sync"

	"github.com/ControlCenterSoft/aidi_2.0/internal/canonical"
)

// InMemory is a minimal, non-persistent reference implementation of
// Repository used only to exercise/validate conformance to the interface
// in unit tests. It is not a persistence adapter and must not be used to
// represent any durable or production storage.
type InMemory struct {
	mu      sync.Mutex
	objects map[canonical.ObjectRef]StoredObject
}

// NewInMemory constructs an empty InMemory repository.
func NewInMemory() *InMemory {
	return &InMemory{objects: make(map[canonical.ObjectRef]StoredObject)}
}

var _ Repository = (*InMemory)(nil)

// Get implements Repository.
func (m *InMemory) Get(_ context.Context, ref canonical.ObjectRef) (StoredObject, error) {
	if err := ref.Validate(); err != nil {
		return StoredObject{}, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	obj, ok := m.objects[ref]
	if !ok {
		return StoredObject{}, ErrNotFound
	}
	return obj, nil
}

// Save implements Repository.
func (m *InMemory) Save(_ context.Context, ref canonical.ObjectRef, expectedRevision canonical.Revision, payload []byte) (canonical.Revision, error) {
	if err := ref.Validate(); err != nil {
		return 0, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	var current canonical.Revision
	if existing, ok := m.objects[ref]; ok {
		current = existing.Revision
	}

	if err := canonical.CheckExpectedRevision(expectedRevision, current); err != nil {
		return 0, err
	}

	next, err := canonical.NextRevision(current)
	if err != nil {
		return 0, err
	}

	m.objects[ref] = StoredObject{Ref: ref, Revision: next, Payload: payload}
	return next, nil
}
