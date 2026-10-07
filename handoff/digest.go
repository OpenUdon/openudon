package handoff

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/OpenUdon/openudon/digest"
)

// DigestFile identifies an already-hashed artifact, without content or I/O.
type DigestFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

// DigestFiles preserves the review/package digest-v1 envelope and scope rules.
// It hashes identity only, and does not assess a workflow or authorize execution.
func DigestFiles(scope, version string, files []DigestFile) (string, error) {
	if len(files) > 1024 || len(scope) > 2048 || len(version) > 128 {
		return "", fmt.Errorf("digest inventory exceeds limits")
	}
	scope = strings.Trim(strings.TrimSpace(filepath.ToSlash(scope)), "/")
	version = strings.TrimSpace(version)
	if version == "" {
		version = "openudon.review-handoff-digest.v1"
	}
	var normalized []DigestFile
	seen := map[string]bool{}
	for _, file := range files {
		clean, ok := cleanReviewHandoffInputPath(file.Path)
		if !ok || seen[clean] || !digest.ValidSHA256(file.SHA256) || len(file.SHA256) != 64 || strings.ToLower(file.SHA256) != file.SHA256 {
			return "", fmt.Errorf("invalid digest inventory")
		}
		seen[clean] = true
		reportPath := clean
		if scope != "" {
			reportPath = filepath.ToSlash(filepath.Join(scope, clean))
		}
		normalized = append(normalized, DigestFile{Path: reportPath, SHA256: file.SHA256})
	}
	sort.Slice(normalized, func(i, j int) bool { return normalized[i].Path < normalized[j].Path })
	data, err := json.Marshal(struct {
		Version string       `json:"version"`
		Scope   string       `json:"scope"`
		Files   []DigestFile `json:"files"`
	}{version, scope, normalized})
	if err != nil {
		return "", err
	}
	return digest.SHA256(data), nil
}
