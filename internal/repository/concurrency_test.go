package repository

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/ControlCenterSoft/aidi_2.0/internal/canonical"
)

// stringCodec is a minimal Codec[string] used only to exercise Update in
// tests: the payload bytes are the string itself.
var stringCodec = Codec[string]{
	Encode: func(s string) ([]byte, error) { return []byte(s), nil },
	Decode: func(b []byte) (string, error) { return string(b), nil },
}

func TestUpdateFirstWriteCreatesObject(t *testing.T) {
	repo := NewInMemory()
	ref := canonical.ObjectRef{Kind: canonical.KindTask, ID: "task-1"}

	var sawCurrent string
	result, err := Update(context.Background(), repo, ref, stringCodec, 0, func(current string) (string, error) {
		sawCurrent = current
		return "v1", nil
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if sawCurrent != "" {
		t.Fatalf("mutate saw current = %q, want empty (object did not exist)", sawCurrent)
	}
	if result.Revision != 1 || result.Value != "v1" {
		t.Fatalf("Update() = %+v, want revision=1 value=v1", result)
	}

	obj, err := repo.Get(context.Background(), ref)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if obj.Revision != 1 || string(obj.Payload) != "v1" {
		t.Fatalf("Get() = %+v, want revision=1 payload=v1", obj)
	}
}

func TestUpdateSuccessfulSubsequentWrite(t *testing.T) {
	repo := NewInMemory()
	ref := canonical.ObjectRef{Kind: canonical.KindTask, ID: "task-1"}

	first, err := Update(context.Background(), repo, ref, stringCodec, 0, func(string) (string, error) {
		return "v1", nil
	})
	if err != nil {
		t.Fatalf("Update() first write error = %v", err)
	}

	second, err := Update(context.Background(), repo, ref, stringCodec, first.Revision, func(current string) (string, error) {
		return current + "+v2", nil
	})
	if err != nil {
		t.Fatalf("Update() second write error = %v", err)
	}
	if second.Revision != 2 || second.Value != "v1+v2" {
		t.Fatalf("Update() second = %+v, want revision=2 value=v1+v2", second)
	}
}

func TestUpdateStaleExpectedRevisionRejectedBeforeMutate(t *testing.T) {
	repo := NewInMemory()
	ref := canonical.ObjectRef{Kind: canonical.KindTask, ID: "task-1"}

	if _, err := Update(context.Background(), repo, ref, stringCodec, 0, func(string) (string, error) {
		return "v1", nil
	}); err != nil {
		t.Fatalf("Update() first write error = %v", err)
	}

	mutateCalled := false
	_, err := Update(context.Background(), repo, ref, stringCodec, 0, func(current string) (string, error) {
		mutateCalled = true
		return "v2-stale", nil
	})

	var conflict *canonical.RevisionConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("Update() error = %v, want *canonical.RevisionConflictError", err)
	}
	if conflict.Expected != 0 || conflict.Actual != 1 {
		t.Fatalf("conflict = %+v, want expected=0 actual=1", conflict)
	}
	if mutateCalled {
		t.Fatalf("Update() invoked mutate on a stale expected revision")
	}

	obj, err := repo.Get(context.Background(), ref)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if obj.Revision != 1 || string(obj.Payload) != "v1" {
		t.Fatalf("Get() after stale Update = %+v, want unchanged revision=1 payload=v1", obj)
	}
}

func TestUpdateMutateErrorLeavesStateUnchanged(t *testing.T) {
	repo := NewInMemory()
	ref := canonical.ObjectRef{Kind: canonical.KindTask, ID: "task-1"}

	if _, err := Update(context.Background(), repo, ref, stringCodec, 0, func(string) (string, error) {
		return "v1", nil
	}); err != nil {
		t.Fatalf("Update() first write error = %v", err)
	}

	wantErr := errors.New("mutate boom")
	_, err := Update(context.Background(), repo, ref, stringCodec, 1, func(string) (string, error) {
		return "", wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Update() error = %v, want %v", err, wantErr)
	}

	obj, err := repo.Get(context.Background(), ref)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if obj.Revision != 1 || string(obj.Payload) != "v1" {
		t.Fatalf("Get() after failed mutate = %+v, want unchanged revision=1 payload=v1", obj)
	}
}

func TestUpdateDecodeErrorSurfacedWithoutSave(t *testing.T) {
	repo := NewInMemory()
	ref := canonical.ObjectRef{Kind: canonical.KindTask, ID: "task-1"}

	if _, err := repo.Save(context.Background(), ref, 0, []byte("not-valid")); err != nil {
		t.Fatalf("Save() seed error = %v", err)
	}

	wantErr := errors.New("decode boom")
	codec := Codec[string]{
		Encode: stringCodec.Encode,
		Decode: func([]byte) (string, error) { return "", wantErr },
	}

	_, err := Update(context.Background(), repo, ref, codec, 1, func(current string) (string, error) {
		t.Fatalf("mutate must not run when decode fails")
		return current, nil
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Update() error = %v, want %v", err, wantErr)
	}
}

func TestUpdateInvalidRefSurfacedWithoutMutate(t *testing.T) {
	repo := NewInMemory()
	ref := canonical.ObjectRef{Kind: "bogus", ID: "task-1"}

	_, err := Update(context.Background(), repo, ref, stringCodec, 0, func(string) (string, error) {
		t.Fatalf("mutate must not run for an invalid ref")
		return "", nil
	})
	if !errors.Is(err, canonical.ErrInvalidKind) {
		t.Fatalf("Update() error = %v, want ErrInvalidKind", err)
	}
}

func TestUpdateConcurrentStaleWriteRejected(t *testing.T) {
	repo := NewInMemory()
	ref := canonical.ObjectRef{Kind: canonical.KindTask, ID: "task-1"}

	first, err := Update(context.Background(), repo, ref, stringCodec, 0, func(string) (string, error) {
		return "v1", nil
	})
	if err != nil {
		t.Fatalf("Update() first write error = %v", err)
	}

	const writers = 8
	var wg sync.WaitGroup
	var mu sync.Mutex
	successes, conflicts := 0, 0

	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := Update(context.Background(), repo, ref, stringCodec, first.Revision, func(current string) (string, error) {
				return current + "+concurrent", nil
			})
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				successes++
			case errors.Is(err, canonical.ErrRevisionConflict):
				conflicts++
			default:
				t.Errorf("Update() unexpected error = %v", err)
			}
		}()
	}
	wg.Wait()

	if successes != 1 {
		t.Fatalf("successes = %d, want exactly 1", successes)
	}
	if conflicts != writers-1 {
		t.Fatalf("conflicts = %d, want %d", conflicts, writers-1)
	}

	obj, err := repo.Get(context.Background(), ref)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if obj.Revision != 2 || !strings.HasPrefix(string(obj.Payload), "v1+concurrent") {
		t.Fatalf("Get() after concurrent Update = %+v, want revision=2 payload=v1+concurrent", obj)
	}
}
