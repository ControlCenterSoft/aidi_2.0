package buildinfo

import "testing"

func TestNewPreservesExplicitBuildMetadata(t *testing.T) {
	t.Parallel()

	got := New("2.0.0", "abc123", "2026-10-03T05:00:00Z")
	want := Info{
		Version:   "2.0.0",
		Commit:    "abc123",
		BuildTime: "2026-10-03T05:00:00Z",
	}
	if got != want {
		t.Fatalf("New() = %+v, want %+v", got, want)
	}
}

func TestNewUsesDeterministicDefaultsForMissingMetadata(t *testing.T) {
	t.Parallel()

	got := New("", "", "")
	want := Info{
		Version:   "dev",
		Commit:    "unknown",
		BuildTime: "unknown",
	}
	if got != want {
		t.Fatalf("New() = %+v, want %+v", got, want)
	}
}
