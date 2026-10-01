package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/ControlCenterSoft/aidi_2.0/internal/canonical"
)

func TestStoredObjectValidate(t *testing.T) {
	tests := []struct {
		name    string
		obj     StoredObject
		wantErr error
	}{
		{
			name:    "valid",
			obj:     StoredObject{Ref: canonical.ObjectRef{Kind: canonical.KindTask, ID: "task-1"}},
			wantErr: nil,
		},
		{
			name:    "invalid kind",
			obj:     StoredObject{Ref: canonical.ObjectRef{Kind: "bogus", ID: "task-1"}},
			wantErr: canonical.ErrInvalidKind,
		},
		{
			name:    "invalid id",
			obj:     StoredObject{Ref: canonical.ObjectRef{Kind: canonical.KindTask, ID: ""}},
			wantErr: canonical.ErrInvalidID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.obj.Validate()
			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("Validate() = %v, want nil", err)
				}
				return
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Validate() = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestInMemoryGetNotFound(t *testing.T) {
	repo := NewInMemory()
	ref := canonical.ObjectRef{Kind: canonical.KindTask, ID: "task-1"}

	_, err := repo.Get(context.Background(), ref)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get() error = %v, want ErrNotFound", err)
	}
}

func TestInMemoryGetInvalidRef(t *testing.T) {
	repo := NewInMemory()

	_, err := repo.Get(context.Background(), canonical.ObjectRef{Kind: "bogus", ID: "task-1"})
	if !errors.Is(err, canonical.ErrInvalidKind) {
		t.Fatalf("Get() error = %v, want ErrInvalidKind", err)
	}

	_, err = repo.Get(context.Background(), canonical.ObjectRef{Kind: canonical.KindTask, ID: ""})
	if !errors.Is(err, canonical.ErrInvalidID) {
		t.Fatalf("Get() error = %v, want ErrInvalidID", err)
	}
}

func TestInMemorySaveInvalidRef(t *testing.T) {
	repo := NewInMemory()

	_, err := repo.Save(context.Background(), canonical.ObjectRef{Kind: "bogus", ID: "task-1"}, 0, []byte("payload"))
	if !errors.Is(err, canonical.ErrInvalidKind) {
		t.Fatalf("Save() error = %v, want ErrInvalidKind", err)
	}

	// Verify no state was touched: a subsequent valid Get on a different
	// ref with the same empty store still reports not found, and the
	// invalid ref itself was never persisted.
	if _, ok := repo.objects[canonical.ObjectRef{Kind: "bogus", ID: "task-1"}]; ok {
		t.Fatalf("Save() with invalid ref must not mutate state")
	}
}

func TestInMemoryFirstWrite(t *testing.T) {
	repo := NewInMemory()
	ref := canonical.ObjectRef{Kind: canonical.KindTask, ID: "task-1"}

	rev, err := repo.Save(context.Background(), ref, 0, []byte("v1"))
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if rev != 1 {
		t.Fatalf("Save() revision = %d, want 1", rev)
	}

	obj, err := repo.Get(context.Background(), ref)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if obj.Revision != 1 || string(obj.Payload) != "v1" {
		t.Fatalf("Get() = %+v, want revision=1 payload=v1", obj)
	}
}

func TestInMemoryStaleWriteRejected(t *testing.T) {
	repo := NewInMemory()
	ref := canonical.ObjectRef{Kind: canonical.KindTask, ID: "task-1"}

	if _, err := repo.Save(context.Background(), ref, 0, []byte("v1")); err != nil {
		t.Fatalf("Save() first write error = %v", err)
	}

	// Stale expected revision (still 0, but current is now 1) must be
	// rejected as a *canonical.RevisionConflictError, and must not
	// advance the stored revision or payload (no last-write-wins).
	_, err := repo.Save(context.Background(), ref, 0, []byte("v2-stale"))
	var conflict *canonical.RevisionConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("Save() error = %v, want *canonical.RevisionConflictError", err)
	}
	if conflict.Expected != 0 || conflict.Actual != 1 {
		t.Fatalf("conflict = %+v, want expected=0 actual=1", conflict)
	}

	obj, err := repo.Get(context.Background(), ref)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if obj.Revision != 1 || string(obj.Payload) != "v1" {
		t.Fatalf("Get() after stale write = %+v, want unchanged revision=1 payload=v1", obj)
	}
}

func TestInMemorySuccessfulSubsequentWrite(t *testing.T) {
	repo := NewInMemory()
	ref := canonical.ObjectRef{Kind: canonical.KindTask, ID: "task-1"}

	first, err := repo.Save(context.Background(), ref, 0, []byte("v1"))
	if err != nil {
		t.Fatalf("Save() first write error = %v", err)
	}

	second, err := repo.Save(context.Background(), ref, first, []byte("v2"))
	if err != nil {
		t.Fatalf("Save() second write error = %v", err)
	}
	if second != 2 {
		t.Fatalf("Save() second revision = %d, want 2", second)
	}

	obj, err := repo.Get(context.Background(), ref)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if obj.Revision != 2 || string(obj.Payload) != "v2" {
		t.Fatalf("Get() after second write = %+v, want revision=2 payload=v2", obj)
	}
}

func TestInMemoryConcurrentStaleWriteRejected(t *testing.T) {
	repo := NewInMemory()
	ref := canonical.ObjectRef{Kind: canonical.KindTask, ID: "task-1"}

	first, err := repo.Save(context.Background(), ref, 0, []byte("v1"))
	if err != nil {
		t.Fatalf("Save() first write error = %v", err)
	}

	const writers = 8
	type result struct {
		rev canonical.Revision
		err error
	}
	results := make(chan result, writers)
	for i := 0; i < writers; i++ {
		go func() {
			rev, err := repo.Save(context.Background(), ref, first, []byte("concurrent"))
			results <- result{rev: rev, err: err}
		}()
	}

	var successes, conflicts int
	for i := 0; i < writers; i++ {
		r := <-results
		switch {
		case r.err == nil:
			successes++
		case errors.Is(r.err, canonical.ErrRevisionConflict):
			conflicts++
		default:
			t.Fatalf("Save() unexpected error = %v", r.err)
		}
	}

	if successes != 1 {
		t.Fatalf("successes = %d, want exactly 1", successes)
	}
	if conflicts != writers-1 {
		t.Fatalf("conflicts = %d, want %d", conflicts, writers-1)
	}
}
