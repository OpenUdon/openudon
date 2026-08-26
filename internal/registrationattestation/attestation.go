// Package registrationattestation owns the private, value-free approval
// artifact required before OpenUdon may construct a browser-registration
// executor handoff. The artifact is consumed locally and is never staged.
package registrationattestation

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/OpenUdon/openudon/internal/evidencefile"
)

const (
	Version  = "openudon.browser-registration-attestation.v1"
	MaxBytes = 16 << 10
)

var (
	digestPattern   = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
	symbolPattern   = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]{0,127}$`)
	reviewerPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9 ._-]{0,127}$`)
)

type Artifact struct {
	Version            string `json:"version"`
	PackageSHA256      string `json:"package_sha256"`
	ProfileSHA256      string `json:"profile_sha256"`
	Operation          string `json:"operation"`
	Flow               string `json:"flow"`
	PriorAttempts      int    `json:"prior_attempts"`
	DedicatedTest      bool   `json:"dedicated_test"`
	CleanupDisposition string `json:"cleanup_disposition"`
	Reviewer           string `json:"reviewer"`
	ExpiresAt          string `json:"expires_at"`
}

type Expected struct {
	PackageSHA256      string
	ProfileSHA256      string
	Operation          string
	Flow               string
	CleanupDisposition string
}

// ReadOutsideRepo reads and validates one exact owner-only, non-symlink
// artifact outside repoRoot. Requiring that placement makes the artifact
// untrackable by the OpenUdon repository containing the package.
func ReadOutsideRepo(path, repoRoot string, expected Expected, now time.Time) (Artifact, string, error) {
	cleanPath, err := validatePrivatePath(path, repoRoot)
	if err != nil {
		return Artifact{}, "", err
	}
	before, err := os.Lstat(cleanPath)
	if err != nil {
		return Artifact{}, "", errors.New("read browser registration attestation")
	}
	if !before.Mode().IsRegular() || before.Mode()&os.ModeSymlink != 0 || before.Mode().Perm()&0o077 != 0 || before.Mode().Perm()&0o400 == 0 || before.Size() <= 0 || before.Size() > MaxBytes {
		return Artifact{}, "", errors.New("browser registration attestation must be an owner-readable, owner-only regular file")
	}
	file, err := os.Open(cleanPath)
	if err != nil {
		return Artifact{}, "", errors.New("read browser registration attestation")
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !os.SameFile(before, opened) {
		return Artifact{}, "", errors.New("browser registration attestation changed during open")
	}
	data, err := io.ReadAll(io.LimitReader(file, MaxBytes+1))
	if err != nil || len(data) == 0 || len(data) > MaxBytes {
		return Artifact{}, "", errors.New("read browser registration attestation")
	}
	after, err := file.Stat()
	pathAfter, pathErr := os.Lstat(cleanPath)
	if err != nil || pathErr != nil || !sameFile(opened, after) || !sameFile(after, pathAfter) || after.Size() != int64(len(data)) {
		return Artifact{}, "", errors.New("browser registration attestation changed during read")
	}
	artifact, err := Decode(data, now)
	if err != nil {
		return Artifact{}, "", err
	}
	if artifact.PackageSHA256 != expected.PackageSHA256 || artifact.ProfileSHA256 != expected.ProfileSHA256 ||
		artifact.Operation != expected.Operation || artifact.Flow != expected.Flow || artifact.CleanupDisposition != expected.CleanupDisposition {
		return Artifact{}, "", errors.New("browser registration attestation does not match the exact package operation")
	}
	sum := sha256.Sum256(data)
	return artifact, "sha256:" + hex.EncodeToString(sum[:]), nil
}

// Decode validates the closed value-free artifact without reading a path.
func Decode(data []byte, now time.Time) (Artifact, error) {
	if len(data) == 0 || len(data) > MaxBytes || !utf8.Valid(data) {
		return Artifact{}, errors.New("browser registration attestation is invalid")
	}
	var artifact Artifact
	if err := evidencefile.DecodeStrict(data, &artifact); err != nil {
		return Artifact{}, errors.New("browser registration attestation is invalid")
	}
	if artifact.Version != Version || !digestPattern.MatchString(artifact.PackageSHA256) || !digestPattern.MatchString(artifact.ProfileSHA256) ||
		!symbolPattern.MatchString(artifact.Operation) || !symbolPattern.MatchString(artifact.Flow) || artifact.PriorAttempts != 0 || !artifact.DedicatedTest ||
		artifact.CleanupDisposition != "delete_separately" && artifact.CleanupDisposition != "retain_dedicated_test_identity" ||
		!reviewerPattern.MatchString(artifact.Reviewer) {
		return Artifact{}, errors.New("browser registration attestation is invalid")
	}
	if now.IsZero() {
		return Artifact{}, errors.New("browser registration attestation assessment time is required")
	}
	now = now.UTC()
	expires, err := time.Parse(time.RFC3339, artifact.ExpiresAt)
	if err != nil || expires.Location() != time.UTC || expires.Nanosecond() != 0 || artifact.ExpiresAt != expires.Format(time.RFC3339) || !expires.After(now) || expires.After(now.Add(24*time.Hour)) {
		return Artifact{}, errors.New("browser registration attestation expiry is invalid")
	}
	return artifact, nil
}

func validatePrivatePath(path, repoRoot string) (string, error) {
	clean := filepath.Clean(strings.TrimSpace(path))
	if clean == "." || !filepath.IsAbs(clean) || clean != strings.TrimSpace(path) {
		return "", errors.New("browser registration attestation path must be absolute and canonical")
	}
	resolved, err := filepath.EvalSymlinks(clean)
	if err != nil || resolved != clean {
		return "", errors.New("browser registration attestation path must not contain symlinks")
	}
	repo, err := filepath.Abs(strings.TrimSpace(repoRoot))
	if err != nil {
		return "", errors.New("resolve repository root")
	}
	repo, err = filepath.EvalSymlinks(filepath.Clean(repo))
	if err != nil {
		return "", errors.New("resolve repository root")
	}
	relative, err := filepath.Rel(repo, clean)
	if err != nil {
		return "", errors.New("compare browser registration attestation path")
	}
	if relative == "." || relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("browser registration attestation must be outside the repository")
	}
	return clean, nil
}

func sameFile(left, right os.FileInfo) bool {
	return left != nil && right != nil && os.SameFile(left, right) && left.Mode() == right.Mode() && left.Size() == right.Size() && left.ModTime().Equal(right.ModTime())
}
