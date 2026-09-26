package verification

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// Evidence is the minimum integrity envelope used to tie a verification result
// to an exact requirement, source revision, artifact and validator.
type Evidence struct {
	RequirementID string    `json:"requirement_id"`
	SourceSHA     string    `json:"source_sha"`
	ArtifactPath  string    `json:"artifact_path"`
	SHA256        string    `json:"sha256"`
	Validator     string    `json:"validator"`
	ProducedAt    time.Time `json:"produced_at"`
}

// Validate rejects evidence that cannot be traced to an exact source/artifact
// pair or that uses an ambiguous artifact path.
func (e Evidence) Validate() error {
	switch {
	case strings.TrimSpace(e.RequirementID) == "":
		return fmt.Errorf("requirement id is required")
	case !isHexDigest(e.SourceSHA, 40):
		return fmt.Errorf("source sha must be a 40-character hexadecimal git commit")
	case strings.TrimSpace(e.ArtifactPath) == "":
		return fmt.Errorf("artifact path is required")
	case filepath.IsAbs(e.ArtifactPath):
		return fmt.Errorf("artifact path must be repository-relative")
	case hasTraversal(e.ArtifactPath):
		return fmt.Errorf("artifact path must not contain parent traversal")
	case !isHexDigest(e.SHA256, 64):
		return fmt.Errorf("sha256 must be a 64-character hexadecimal digest")
	case strings.TrimSpace(e.Validator) == "":
		return fmt.Errorf("validator is required")
	case e.ProducedAt.IsZero():
		return fmt.Errorf("produced at timestamp is required")
	default:
		return nil
	}
}

// VerifySHA256 deterministically checks artifact bytes against the digest
// recorded in the evidence envelope.
func VerifySHA256(data []byte, expected string) error {
	if !isHexDigest(expected, 64) {
		return fmt.Errorf("expected sha256 must be a 64-character hexadecimal digest")
	}

	actual := sha256.Sum256(data)
	if !strings.EqualFold(hex.EncodeToString(actual[:]), expected) {
		return fmt.Errorf("artifact sha256 mismatch")
	}
	return nil
}

func isHexDigest(value string, length int) bool {
	if len(value) != length {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func hasTraversal(path string) bool {
	cleaned := filepath.ToSlash(filepath.Clean(path))
	return cleaned == ".." || strings.HasPrefix(cleaned, "../")
}
