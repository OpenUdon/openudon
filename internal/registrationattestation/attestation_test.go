package registrationattestation

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestReadOutsideRepoAcceptsExactOwnerOnlyArtifact(t *testing.T) {
	now := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	repo := filepath.Join(t.TempDir(), "repo")
	if err := os.Mkdir(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	artifact := validArtifact(now)
	path := writeArtifact(t, t.TempDir(), artifact, 0o600)
	got, digest, err := ReadOutsideRepo(path, repo, expected(artifact), now)
	if err != nil {
		t.Fatal(err)
	}
	if got != artifact {
		t.Fatalf("artifact = %#v", got)
	}
	if !digestPattern.MatchString(digest) {
		t.Fatalf("artifact digest = %q", digest)
	}
}

func TestReadOutsideRepoRejectsUnsafePathModeAndDrift(t *testing.T) {
	now := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	repo := filepath.Join(t.TempDir(), "repo")
	if err := os.Mkdir(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	artifact := validArtifact(now)
	inside := writeArtifact(t, repo, artifact, 0o600)
	if _, _, err := ReadOutsideRepo(inside, repo, expected(artifact), now); err == nil || !strings.Contains(err.Error(), "outside") {
		t.Fatalf("inside-repository artifact error = %v", err)
	}
	public := writeArtifact(t, t.TempDir(), artifact, 0o644)
	if _, _, err := ReadOutsideRepo(public, repo, expected(artifact), now); err == nil || !strings.Contains(err.Error(), "owner-only") {
		t.Fatalf("public artifact error = %v", err)
	}
	private := writeArtifact(t, t.TempDir(), artifact, 0o400)
	drifted := expected(artifact)
	drifted.Flow = "different"
	if _, _, err := ReadOutsideRepo(private, repo, drifted, now); err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("drifted artifact error = %v", err)
	}
}

func TestDecodeRejectsValuesUnknownFieldsAndLifecycleDrift(t *testing.T) {
	now := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	base := validArtifact(now)
	mutations := []func(*Artifact){
		func(value *Artifact) { value.Version = "future" },
		func(value *Artifact) { value.PackageSHA256 = "sha256:" + strings.Repeat("A", 64) },
		func(value *Artifact) { value.ProfileSHA256 = "account@example.test" },
		func(value *Artifact) { value.Operation = "register account@example.test" },
		func(value *Artifact) { value.Flow = "" },
		func(value *Artifact) { value.PriorAttempts = 1 },
		func(value *Artifact) { value.DedicatedTest = false },
		func(value *Artifact) { value.CleanupDisposition = "keep_forever" },
		func(value *Artifact) { value.Reviewer = "reviewer@example.test" },
		func(value *Artifact) { value.ExpiresAt = now.Format(time.RFC3339) },
		func(value *Artifact) { value.ExpiresAt = now.Add(25 * time.Hour).Format(time.RFC3339) },
	}
	for index, mutate := range mutations {
		value := base
		mutate(&value)
		data, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Decode(data, now); err == nil {
			t.Fatalf("mutation %d accepted", index)
		}
	}
	data, err := json.Marshal(base)
	if err != nil {
		t.Fatal(err)
	}
	unknown := strings.TrimSuffix(string(data), "}") + `,"account":"forbidden"}`
	if _, err := Decode([]byte(unknown), now); err == nil {
		t.Fatal("unknown account field accepted")
	}
	if _, err := Decode(append(data, data...), now); err == nil {
		t.Fatal("multiple JSON documents accepted")
	}
	duplicate := strings.Replace(string(data), `"version":"`+Version+`"`, `"version":"`+Version+`","version":"`+Version+`"`, 1)
	if _, err := Decode([]byte(duplicate), now); err == nil {
		t.Fatal("duplicate JSON field accepted")
	}
}

func validArtifact(now time.Time) Artifact {
	return Artifact{
		Version: Version, PackageSHA256: "sha256:" + strings.Repeat("a", 64), ProfileSHA256: "sha256:" + strings.Repeat("b", 64),
		Operation: "register_test_user", Flow: "create_dedicated_test_user", PriorAttempts: 0, DedicatedTest: true,
		CleanupDisposition: "delete_separately", Reviewer: "Synthetic Reviewer", ExpiresAt: now.Add(time.Hour).Format(time.RFC3339),
	}
}

func expected(value Artifact) Expected {
	return Expected{PackageSHA256: value.PackageSHA256, ProfileSHA256: value.ProfileSHA256, Operation: value.Operation, Flow: value.Flow, CleanupDisposition: value.CleanupDisposition}
}

func writeArtifact(t *testing.T, root string, value Artifact, mode os.FileMode) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "registration-attestation.json")
	if err := os.WriteFile(path, append(data, '\n'), mode); err != nil {
		t.Fatal(err)
	}
	return path
}
