package verification

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
	"time"
)

func validEvidence() Evidence {
	sum := sha256.Sum256([]byte("release-a-evidence"))
	return Evidence{
		RequirementID: "AC-20.5-EVIDENCE",
		SourceSHA:     "16f7f4a35a7ad04f1dcc92fa5a5fd12750d149f2",
		ArtifactPath:  "dist/aidi",
		SHA256:        hex.EncodeToString(sum[:]),
		Validator:     "go-test",
		ProducedAt:    time.Date(2026, 9, 26, 20, 55, 0, 0, time.UTC),
	}
}

func TestEvidenceValidate(t *testing.T) {
	if err := validEvidence().Validate(); err != nil {
		t.Fatalf("valid evidence rejected: %v", err)
	}
}

func TestEvidenceValidationRejectsInvalidFields(t *testing.T) {
	tests := []struct {
		name string
		edit func(*Evidence)
		want string
	}{
		{"missing requirement", func(e *Evidence) { e.RequirementID = " " }, "requirement id"},
		{"short source sha", func(e *Evidence) { e.SourceSHA = "abc123" }, "source sha"},
		{"nonhex source sha", func(e *Evidence) { e.SourceSHA = strings.Repeat("z", 40) }, "source sha"},
		{"missing artifact", func(e *Evidence) { e.ArtifactPath = "" }, "artifact path"},
		{"absolute artifact", func(e *Evidence) { e.ArtifactPath = "/tmp/aidi" }, "repository-relative"},
		{"artifact traversal", func(e *Evidence) { e.ArtifactPath = "../artifact" }, "parent traversal"},
		{"short digest", func(e *Evidence) { e.SHA256 = "deadbeef" }, "sha256"},
		{"nonhex digest", func(e *Evidence) { e.SHA256 = strings.Repeat("x", 64) }, "sha256"},
		{"missing validator", func(e *Evidence) { e.Validator = "" }, "validator"},
		{"missing timestamp", func(e *Evidence) { e.ProducedAt = time.Time{} }, "timestamp"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evidence := validEvidence()
			tt.edit(&evidence)
			err := evidence.Validate()
			if err == nil {
				t.Fatal("expected validation error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q in error, got %q", tt.want, err)
			}
		})
	}
}

func TestVerifySHA256(t *testing.T) {
	data := []byte("release-a-evidence")
	sum := sha256.Sum256(data)
	if err := VerifySHA256(data, hex.EncodeToString(sum[:])); err != nil {
		t.Fatalf("matching digest rejected: %v", err)
	}
}

func TestVerifySHA256RejectsMismatch(t *testing.T) {
	sum := sha256.Sum256([]byte("other-evidence"))
	err := VerifySHA256([]byte("release-a-evidence"), hex.EncodeToString(sum[:]))
	if err == nil || !strings.Contains(err.Error(), "mismatch") {
		t.Fatalf("expected mismatch error, got %v", err)
	}
}

func TestVerifySHA256RejectsMalformedExpectedDigest(t *testing.T) {
	err := VerifySHA256([]byte("release-a-evidence"), "deadbeef")
	if err == nil || !strings.Contains(err.Error(), "64-character") {
		t.Fatalf("expected malformed digest error, got %v", err)
	}
}
