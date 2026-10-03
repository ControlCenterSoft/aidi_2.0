package releasebuild_test

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestReleaseBuildIsReproducible(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("release build contract targets the Debian/Linux release environment")
	}

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate repository root")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))

	cmd := exec.Command("make", "verify-release-reproducible", "RELEASE_VERSION=2.0.0-test")
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("reproducible release verification failed: %v\n%s", err, output)
	}
}
