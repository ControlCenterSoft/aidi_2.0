package inbox

import (
	"bytes"
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
)

// equalRecords compares two Records by value, since Record embeds a []byte
// and is therefore not comparable with ==.
func equalRecords(a, b Record) bool {
	return a.CommandID == b.CommandID && bytes.Equal(a.Result, b.Result)
}

func TestValidateCommandIDRejectsEmpty(t *testing.T) {
	for _, commandID := range []string{"", "   ", "\t\n"} {
		if err := ValidateCommandID(commandID); !errors.Is(err, ErrInvalidCommandID) {
			t.Fatalf("ValidateCommandID(%q) error = %v, want ErrInvalidCommandID", commandID, err)
		}
	}
}

func TestValidateCommandIDAcceptsNonEmpty(t *testing.T) {
	if err := ValidateCommandID("cmd-1"); err != nil {
		t.Fatalf("ValidateCommandID() error = %v, want nil", err)
	}
}

func TestInMemoryGetRejectsInvalidCommandID(t *testing.T) {
	store := NewInMemory()
	if _, err := store.Get(context.Background(), ""); !errors.Is(err, ErrInvalidCommandID) {
		t.Fatalf("Get() error = %v, want ErrInvalidCommandID", err)
	}
}

func TestInMemoryGetNotFound(t *testing.T) {
	store := NewInMemory()
	if _, err := store.Get(context.Background(), "cmd-1"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get() error = %v, want ErrNotFound", err)
	}
}

func TestInMemoryOnceRejectsInvalidCommandID(t *testing.T) {
	store := NewInMemory()
	called := false
	_, produced, err := store.Once(context.Background(), "", func(context.Context) ([]byte, error) {
		called = true
		return []byte("result"), nil
	})
	if !errors.Is(err, ErrInvalidCommandID) {
		t.Fatalf("Once() error = %v, want ErrInvalidCommandID", err)
	}
	if produced {
		t.Fatalf("Once() produced = true, want false for invalid commandID")
	}
	if called {
		t.Fatalf("Once() invoked produce for an invalid commandID")
	}
}

func TestInMemoryOnceFirstCallProduces(t *testing.T) {
	store := NewInMemory()

	record, produced, err := store.Once(context.Background(), "cmd-1", func(context.Context) ([]byte, error) {
		return []byte("result-1"), nil
	})
	if err != nil {
		t.Fatalf("Once() error = %v", err)
	}
	if !produced {
		t.Fatalf("Once() produced = false, want true for first call")
	}
	if record.CommandID != "cmd-1" || string(record.Result) != "result-1" {
		t.Fatalf("Once() record = %+v, want CommandID=cmd-1 Result=result-1", record)
	}

	stored, err := store.Get(context.Background(), "cmd-1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !equalRecords(stored, record) {
		t.Fatalf("Get() = %+v, want %+v", stored, record)
	}
}

func TestInMemoryOnceDuplicateDoesNotReproduce(t *testing.T) {
	store := NewInMemory()
	var calls int

	first, produced, err := store.Once(context.Background(), "cmd-1", func(context.Context) ([]byte, error) {
		calls++
		return []byte("result-1"), nil
	})
	if err != nil || !produced {
		t.Fatalf("Once() first call = (%+v, %v, %v), want produced=true, err=nil", first, produced, err)
	}

	second, produced, err := store.Once(context.Background(), "cmd-1", func(context.Context) ([]byte, error) {
		calls++
		return []byte("result-2"), nil
	})
	if err != nil {
		t.Fatalf("Once() duplicate call error = %v", err)
	}
	if produced {
		t.Fatalf("Once() duplicate call produced = true, want false")
	}
	if !equalRecords(second, first) {
		t.Fatalf("Once() duplicate call = %+v, want identical to first call %+v", second, first)
	}
	if calls != 1 {
		t.Fatalf("produce invoked %d times, want exactly 1 (duplicate Command ID must produce one logical result)", calls)
	}
}

func TestInMemoryOnceProduceErrorNotRecorded(t *testing.T) {
	store := NewInMemory()
	wantErr := errors.New("side effect failed")

	_, produced, err := store.Once(context.Background(), "cmd-1", func(context.Context) ([]byte, error) {
		return nil, wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Once() error = %v, want %v", err, wantErr)
	}
	if produced {
		t.Fatalf("Once() produced = true, want false when produce errors")
	}

	if _, err := store.Get(context.Background(), "cmd-1"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get() after failed produce error = %v, want ErrNotFound (nothing should be recorded)", err)
	}

	// A later call with the same commandID must still be able to run
	// produce: a failed attempt is not a logical state-changing result.
	record, produced, err := store.Once(context.Background(), "cmd-1", func(context.Context) ([]byte, error) {
		return []byte("result-1"), nil
	})
	if err != nil || !produced {
		t.Fatalf("Once() retry after failed produce = (%+v, %v, %v), want produced=true, err=nil", record, produced, err)
	}
}

func TestExecuteReturnsSingleResultForDuplicateCommandID(t *testing.T) {
	store := NewInMemory()
	var calls int32

	produce := func(context.Context) ([]byte, error) {
		atomic.AddInt32(&calls, 1)
		return []byte("side-effect-applied"), nil
	}

	first, err := Execute(context.Background(), store, "cmd-1", produce)
	if err != nil {
		t.Fatalf("Execute() first call error = %v", err)
	}

	second, err := Execute(context.Background(), store, "cmd-1", produce)
	if err != nil {
		t.Fatalf("Execute() duplicate call error = %v", err)
	}

	if !equalRecords(first, second) {
		t.Fatalf("Execute() duplicate Command ID = %+v, want identical to first result %+v", second, first)
	}
	if calls != 1 {
		t.Fatalf("produce invoked %d times via Execute, want exactly 1", calls)
	}
}

// TestInMemoryOnceConcurrentDuplicatesProduceExactlyOnce exercises the
// A3-004 acceptance criteria under real concurrency (run with -race): many
// goroutines submit the same Command ID simultaneously, and exactly one of
// them must cause produce to run, with every goroutine observing the
// identical logical Record.
func TestInMemoryOnceConcurrentDuplicatesProduceExactlyOnce(t *testing.T) {
	store := NewInMemory()

	const goroutines = 50
	var calls int32

	produce := func(context.Context) ([]byte, error) {
		atomic.AddInt32(&calls, 1)
		return []byte("result-1"), nil
	}

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		results = make([]Record, 0, goroutines)
	)
	start := make(chan struct{})

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			record, _, err := store.Once(context.Background(), "cmd-concurrent", produce)
			if err != nil {
				t.Errorf("Once() error = %v", err)
				return
			}
			mu.Lock()
			results = append(results, record)
			mu.Unlock()
		}()
	}

	close(start)
	wg.Wait()

	if calls != 1 {
		t.Fatalf("produce invoked %d times across %d concurrent duplicate Command IDs, want exactly 1", calls, goroutines)
	}
	if len(results) != goroutines {
		t.Fatalf("got %d results, want %d", len(results), goroutines)
	}
	want := results[0]
	for i, got := range results {
		if !equalRecords(got, want) {
			t.Fatalf("result[%d] = %+v, want identical to result[0] %+v", i, got, want)
		}
	}
}
