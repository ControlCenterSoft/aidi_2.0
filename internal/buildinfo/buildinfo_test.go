package buildinfo

import (
	"bytes"
	"testing"
)

func TestNewPreservesExplicitBuildMetadata(t *testing.T) {
	t.Parallel()

	got := New("2.0.0", "abc123", "2026-10-03T05:00:00Z", "github:example/repo@abc123")
	want := Info{
		Version:    "2.0.0",
		Commit:     "abc123",
		BuildTime:  "2026-10-03T05:00:00Z",
		Provenance: "github:example/repo@abc123",
	}
	if got != want {
		t.Fatalf("New() = %+v, want %+v", got, want)
	}
}

func TestNewUsesDeterministicDefaultsForMissingMetadata(t *testing.T) {
	t.Parallel()

	got := New("", "", "", "")
	want := Info{
		Version:    "dev",
		Commit:     "unknown",
		BuildTime:  "unknown",
		Provenance: "unknown",
	}
	if got != want {
		t.Fatalf("New() = %+v, want %+v", got, want)
	}
}

func TestVersionRequested(t *testing.T) {
	t.Parallel()

	for _, args := range [][]string{{"aidi", "--version"}, {"aidi", "version"}} {
		if !VersionRequested(args) {
			t.Fatalf("VersionRequested(%q) = false, want true", args)
		}
	}
	if VersionRequested([]string{"aidi"}) {
		t.Fatal("VersionRequested without an argument = true, want false")
	}
}

func TestWriteJSONIncludesProvenance(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	info := New("2.0.0", "abc123", "2026-10-03T05:00:00Z", "github:example/repo@abc123")
	if err := WriteJSON(&buf, info); err != nil {
		t.Fatalf("WriteJSON() error = %v", err)
	}
	want := "{\"version\":\"2.0.0\",\"commit\":\"abc123\",\"build_time\":\"2026-10-03T05:00:00Z\",\"provenance\":\"github:example/repo@abc123\"}\n"
	if got := buf.String(); got != want {
		t.Fatalf("WriteJSON() = %q, want %q", got, want)
	}
}
