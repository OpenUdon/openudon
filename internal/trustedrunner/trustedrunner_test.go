package trustedrunner

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	asyncevidence "github.com/OpenUdon/evidence/async"
	evdigest "github.com/OpenUdon/evidence/digest"
	"github.com/OpenUdon/openudon/internal/authoring"
	"github.com/OpenUdon/openudon/internal/synthesize"
	"github.com/OpenUdon/openudon/internal/udonrunner"
	"github.com/OpenUdon/uws/validation"
)

func TestRunValidSandboxApprovalPassesDryRun(t *testing.T) {
	root, example := writeFixture(t, fixtureOptions{})
	now := fixedNow()
	approvalPath := writeApprovalTemplate(t, root, example, StateApprovedForSandbox, now)

	result, err := Run(context.Background(), Options{
		RepoRoot:     root,
		ExampleDir:   example,
		Tier:         TierSandbox,
		ApprovalPath: approvalPath,
		DryRun:       true,
		Now:          now,
		Assess:       passAssess,
		Invoke: func(context.Context, udonrunner.Invocation) error {
			t.Fatal("dry-run invoked runner")
			return nil
		},
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !result.DryRun || result.Scope != "examples/support-email" || result.PackageSHA256 == "" {
		t.Fatalf("unexpected result: %+v", result)
	}
	if result.StagePath == "" || result.RunEvidencePath == "" || result.AsyncEvidencePath == "" {
		t.Fatalf("dry-run did not stage package and write evidence: %+v", result)
	}
}

func TestRunDryRunStagesAndWritesEvidenceWithoutCredentialEnv(t *testing.T) {
	root, example := writeFixture(t, fixtureOptions{credentialBindings: []string{"support-api.token"}})
	now := fixedNow()
	approvalPath := writeApprovalTemplate(t, root, example, StateApprovedForSandbox, now)

	result, err := Run(context.Background(), Options{
		RepoRoot:     root,
		ExampleDir:   example,
		Tier:         TierSandbox,
		ApprovalPath: approvalPath,
		DryRun:       true,
		WorkDir:      "relative-workdir",
		Now:          now,
		Assess:       passAssess,
		Invoke: func(context.Context, udonrunner.Invocation) error {
			t.Fatal("dry-run invoked runner")
			return nil
		},
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !filepath.IsAbs(result.WorkDir) || !strings.HasPrefix(result.WorkDir, filepath.Join(root, "relative-workdir", "run-")) {
		t.Fatalf("workdir = %q, want unique absolute run path under repo root", result.WorkDir)
	}
	if _, err := os.Stat(filepath.Join(result.StagePath, "workflows", "workflow.uws.yaml")); err != nil {
		t.Fatalf("dry-run did not stage workflow: %v", err)
	}
	evidence := readRunEvidenceFile(t, result.RunEvidencePath)
	if evidence.Version != RunEvidenceVersion || !evidence.DryRun || evidence.Executor.Invoked || evidence.StageKind != "dry-run" {
		t.Fatalf("unexpected evidence: %#v", evidence)
	}
	if evidence.StagePath != result.StagePath || evidence.PackageSHA256 != result.PackageSHA256 {
		t.Fatalf("evidence does not match result: evidence=%#v result=%#v", evidence, result)
	}
	if !stringSliceContains(evidence.CredentialEnvNames, "UDON_CREDENTIAL_SUPPORT_API_TOKEN") {
		t.Fatalf("evidence missing credential env name: %#v", evidence.CredentialEnvNames)
	}
	ref, bundle := readReferencedAsyncEvidence(t, result.WorkDir, evidence)
	if ref.Path != "async-evidence.json" || ref.Records != 2 {
		t.Fatalf("unexpected async evidence ref: %#v", ref)
	}
	if result.AsyncEvidencePath != filepath.Join(result.WorkDir, ref.Path) {
		t.Fatalf("result async evidence path = %q, want workdir-relative ref %q", result.AsyncEvidencePath, ref.Path)
	}
	request := asyncExecutionRequest(t, bundle)
	response := asyncExecutionResponse(t, bundle)
	if request.Version != asyncevidence.ExecutionRequestVersion || response.Version != asyncevidence.ExecutionResponseVersion {
		t.Fatalf("unexpected async record versions: request=%q response=%q", request.Version, response.Version)
	}
	if request.Operation.SubjectKind != "openudon_package" || request.Operation.SubjectID != result.Scope || request.Operation.Action != "run" || request.Operation.SourceKind != "uws" {
		t.Fatalf("unexpected async operation ref: %#v", request.Operation)
	}
	if request.Operation.SourcePath != result.StagePath+"/workflows/workflow.uws.yaml" && request.Operation.SourcePath != filepath.Join(result.StagePath, "workflows", "workflow.uws.yaml") {
		t.Fatalf("async source path = %q, want staged workflow under %q", request.Operation.SourcePath, result.StagePath)
	}
	if request.Attempt.Source != "openudon.trustedrunner" || request.Attempt.Actor != "Ada" || request.RequestID != result.PackageSHA256 {
		t.Fatalf("unexpected async request attempt/request ID: %#v", request)
	}
	if request.Transport["runner_mode"] != "dry-run" || request.Transport["stage_kind"] != "dry-run" || request.Transport["tier"] != TierSandbox || request.Transport["dry_run"] != "true" {
		t.Fatalf("unexpected async request transport: %#v", request.Transport)
	}
	if response.Outcome != "accepted" || response.RequestEvidenceID != request.Attempt.EvidenceID {
		t.Fatalf("unexpected async response: %#v request=%#v", response, request)
	}
	data, err := os.ReadFile(result.RunEvidencePath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "super-secret") {
		t.Fatalf("run evidence leaked credential value:\n%s", data)
	}
}

func TestRunAsyncEvidenceReferenceSurvivesArchiveMove(t *testing.T) {
	root, example := writeFixture(t, fixtureOptions{})
	now := fixedNow()
	approvalPath := writeApprovalTemplate(t, root, example, StateApprovedForSandbox, now)

	result, err := Run(context.Background(), Options{
		RepoRoot:     root,
		ExampleDir:   example,
		Tier:         TierSandbox,
		ApprovalPath: approvalPath,
		DryRun:       true,
		WorkDir:      filepath.Join(root, "work"),
		Now:          now,
		Assess:       passAssess,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	original := readRunEvidenceFile(t, result.RunEvidencePath)
	ref := original.AsyncEvidenceFiles[0]
	archive := filepath.Join(root, "archive")
	if err := os.MkdirAll(archive, 0o755); err != nil {
		t.Fatal(err)
	}
	copyFile(t, result.RunEvidencePath, filepath.Join(archive, "run-evidence.json"))
	copyFile(t, filepath.Join(result.WorkDir, ref.Path), filepath.Join(archive, ref.Path))

	archived := readRunEvidenceFile(t, filepath.Join(archive, "run-evidence.json"))
	_, bundle := readReferencedAsyncEvidence(t, archive, archived)
	request := asyncExecutionRequest(t, bundle)
	if request.Operation.SubjectID != result.Scope || request.RequestID != result.PackageSHA256 {
		t.Fatalf("archived async request does not match run: %#v result=%#v", request, result)
	}
}

func TestVerifyRunEvidenceFileValidatesAsyncSidecar(t *testing.T) {
	result := writeVerifiableRunEvidence(t)
	verified, err := VerifyRunEvidenceFile(result.RunEvidencePath)
	if err != nil {
		t.Fatalf("VerifyRunEvidenceFile returned error: %v", err)
	}
	if verified.RunEvidencePath != result.RunEvidencePath || len(verified.AsyncEvidenceFiles) != 1 {
		t.Fatalf("unexpected verify result: %#v", verified)
	}
}

func TestRunEvidenceSignatureEmbeddedAndTrustedVerification(t *testing.T) {
	result := writeVerifiableRunEvidence(t)
	keys := t.TempDir()
	privatePath := filepath.Join(keys, "operator-private.pem")
	publicPath := filepath.Join(keys, "operator-public.pem")
	if err := GenerateSigningKey(privatePath, publicPath); err != nil {
		t.Fatal(err)
	}
	if _, err := SignRunEvidenceFile(result.RunEvidencePath, privatePath); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyRunEvidenceFileWithOptions(result.RunEvidencePath, VerifyRunEvidenceOptions{RequireSignature: true}); err != nil {
		t.Fatalf("embedded-key verification failed: %v", err)
	}
	if _, err := VerifyRunEvidenceFileWithOptions(result.RunEvidencePath, VerifyRunEvidenceOptions{TrustedPublicKey: publicPath}); err != nil {
		t.Fatalf("trusted-key verification failed: %v", err)
	}

	otherPrivate := filepath.Join(keys, "other-private.pem")
	otherPublic := filepath.Join(keys, "other-public.pem")
	if err := GenerateSigningKey(otherPrivate, otherPublic); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyRunEvidenceFileWithOptions(result.RunEvidencePath, VerifyRunEvidenceOptions{TrustedPublicKey: otherPublic}); err == nil || !strings.Contains(err.Error(), "trusted public key") {
		t.Fatalf("wrong trust key error = %v", err)
	}
}

func TestGenerateSigningKeyRejectsCollidingOutputPaths(t *testing.T) {
	root := t.TempDir()
	tests := []struct {
		name       string
		privateKey string
		publicKey  string
	}{
		{name: "identical", privateKey: filepath.Join(root, "same.pem"), publicKey: filepath.Join(root, "same.pem")},
		{name: "clean alias", privateKey: filepath.Join(root, "alias.pem"), publicKey: filepath.Join(root, ".", "alias.pem")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := GenerateSigningKey(test.privateKey, test.publicKey); err == nil || !strings.Contains(err.Error(), "distinct") {
				t.Fatalf("colliding key paths error = %v", err)
			}
			if _, err := os.Lstat(test.privateKey); !os.IsNotExist(err) {
				t.Fatalf("colliding key path was written: %v", err)
			}
		})
	}

	realDir := filepath.Join(root, "real")
	if err := os.Mkdir(realDir, 0o755); err != nil {
		t.Fatal(err)
	}
	aliasDir := filepath.Join(root, "linked")
	if err := os.Symlink(realDir, aliasDir); err != nil {
		t.Skipf("symlink aliases unavailable: %v", err)
	}
	privateKey := filepath.Join(realDir, "symlink.pem")
	publicKey := filepath.Join(aliasDir, "symlink.pem")
	if err := GenerateSigningKey(privateKey, publicKey); err == nil || !strings.Contains(err.Error(), "distinct") {
		t.Fatalf("symlink-colliding key paths error = %v", err)
	}
	if _, err := os.Lstat(privateKey); !os.IsNotExist(err) {
		t.Fatalf("symlink-colliding key path was written: %v", err)
	}
}

func TestRunEvidenceSignatureRejectsMissingAndChangedEvidence(t *testing.T) {
	unsigned := writeVerifiableRunEvidence(t)
	if _, err := VerifyRunEvidenceFileWithOptions(unsigned.RunEvidencePath, VerifyRunEvidenceOptions{RequireSignature: true}); err == nil || !strings.Contains(err.Error(), "required") {
		t.Fatalf("missing signature error = %v", err)
	}

	result := writeVerifiableRunEvidence(t)
	keys := t.TempDir()
	privatePath := filepath.Join(keys, "operator-private.pem")
	publicPath := filepath.Join(keys, "operator-public.pem")
	if err := GenerateSigningKey(privatePath, publicPath); err != nil {
		t.Fatal(err)
	}
	if _, err := SignRunEvidenceFile(result.RunEvidencePath, privatePath); err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(result.RunEvidencePath, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(" \n"); err != nil {
		f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyRunEvidenceFileWithOptions(result.RunEvidencePath, VerifyRunEvidenceOptions{RequireSignature: true}); err == nil || !strings.Contains(err.Error(), "digest mismatch") {
		t.Fatalf("changed evidence error = %v", err)
	}
}

func TestRunRejectsInvalidSigningKeyBeforeExecutorInvocation(t *testing.T) {
	root, example := writeFixture(t, fixtureOptions{})
	now := fixedNow()
	approvalPath := writeApprovalTemplate(t, root, example, StateApprovedForSandbox, now)
	keyPath := filepath.Join(root, "invalid-signing-key.pem")
	mustWriteFile(t, keyPath, []byte("not a private key\n"))
	invoked := false
	result, err := Run(context.Background(), Options{
		RepoRoot: root, ExampleDir: example, Tier: TierSandbox, ApprovalPath: approvalPath,
		Now: now, Assess: passAssess, SigningKey: keyPath,
		Invoke: func(context.Context, udonrunner.Invocation) error {
			invoked = true
			return nil
		},
	})
	if err == nil || !strings.Contains(err.Error(), "validate signing key before execution") {
		t.Fatalf("invalid signing key result=%#v error=%v", result, err)
	}
	if invoked {
		t.Fatal("executor was invoked before the signing key was validated")
	}
}

func TestAsyncEvidenceSidecarMatchesJSONSchema(t *testing.T) {
	result := writeVerifiableRunEvidence(t)
	schema := filepath.Join("..", "..", "docs", "schemas", "openudon.async-evidence-bundle.v1.schema.json")
	if err := validation.ValidateFile(schema, result.AsyncEvidencePath); err != nil {
		t.Fatalf("async evidence sidecar failed JSON schema validation: %v", err)
	}
}

func TestArchiveRunEvidenceCopiesAndVerifiesEvidence(t *testing.T) {
	result := writeVerifiableRunEvidence(t)
	evidence := readRunEvidenceFile(t, result.RunEvidencePath)
	reportPath := filepath.Join(result.WorkDir, "stage.fake", "executor-report.json")
	mustWriteFile(t, reportPath, []byte(`{"version":"udon.execution-report.v1","status":"success","started_at":"2026-04-29T12:00:00Z","finished_at":"2026-04-29T12:00:00Z","workflow_path":"workflow.uws.yaml","workflow_format":"uws-yaml","workdir":"."}`+"\n"))
	evidence.Executor.ReportPath = "stage.fake/executor-report.json"
	reportData, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatal(err)
	}
	reportDigest := sha256.Sum256(reportData)
	evidence.Executor.ReportSHA256 = fmt.Sprintf("%x", reportDigest[:])
	evidence.Executor.ReportSize = int64(len(reportData))
	writeRunEvidenceFileForTest(t, result.RunEvidencePath, evidence)

	archiveDir := filepath.Join(result.WorkDir, "archive")
	archived, err := ArchiveRunEvidence(ArchiveOptions{RunEvidencePath: result.RunEvidencePath, ArchiveDir: archiveDir})
	if err != nil {
		t.Fatalf("ArchiveRunEvidence returned error: %v", err)
	}
	for _, path := range []string{
		filepath.Join(archiveDir, "run-evidence.json"),
		filepath.Join(archiveDir, "async-evidence.json"),
		filepath.Join(archiveDir, "stage.fake", "executor-report.json"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("archive file missing %s: %v", path, err)
		}
	}
	if archived.VerifiedSidecars != 1 || archived.ExecutorReport == "" {
		t.Fatalf("unexpected archive result: %#v", archived)
	}
	if _, err := VerifyRunEvidenceFile(filepath.Join(archiveDir, "run-evidence.json")); err != nil {
		t.Fatalf("archived evidence did not verify: %v", err)
	}
}

func TestArchiveRunEvidenceRejectsCollision(t *testing.T) {
	result := writeVerifiableRunEvidence(t)
	archiveDir := filepath.Join(result.WorkDir, "occupied-archive")
	mustWriteFile(t, filepath.Join(archiveDir, "sentinel"), []byte("do not overwrite"))
	if _, err := ArchiveRunEvidence(ArchiveOptions{RunEvidencePath: result.RunEvidencePath, ArchiveDir: archiveDir}); err == nil || !strings.Contains(err.Error(), "not empty") {
		t.Fatalf("archive collision error = %v", err)
	}
}

func TestArchiveRunEvidenceRejectsSymlinkedArchiveRoot(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink permissions are not portable on Windows")
	}
	result := writeVerifiableRunEvidence(t)
	target := filepath.Join(result.WorkDir, "archive-target")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	archiveDir := filepath.Join(result.WorkDir, "archive-link")
	if err := os.Symlink(target, archiveDir); err != nil {
		t.Fatal(err)
	}
	if _, err := ArchiveRunEvidence(ArchiveOptions{RunEvidencePath: result.RunEvidencePath, ArchiveDir: archiveDir}); err == nil || !strings.Contains(err.Error(), "real directory") {
		t.Fatalf("symlinked archive root error = %v", err)
	}
	entries, err := os.ReadDir(target)
	if err != nil || len(entries) != 0 {
		t.Fatalf("symlink target was modified: entries=%v err=%v", entries, err)
	}
}

func TestVerifyRunEvidenceRejectsSymlinkedExecutorReport(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink permissions are not portable on Windows")
	}
	result := writeVerifiableRunEvidence(t)
	evidence := readRunEvidenceFile(t, result.RunEvidencePath)
	target := filepath.Join(t.TempDir(), "substitute-report.json")
	reportData := []byte("substituted report\n")
	mustWriteFile(t, target, reportData)
	reportRel := "stage.fake/executor-report.json"
	reportPath := filepath.Join(result.WorkDir, filepath.FromSlash(reportRel))
	if err := os.MkdirAll(filepath.Dir(reportPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, reportPath); err != nil {
		t.Fatal(err)
	}
	evidence.Executor.ReportPath = reportRel
	evidence.Executor.ReportSHA256 = fmt.Sprintf("%x", sha256.Sum256(reportData))
	evidence.Executor.ReportSize = int64(len(reportData))
	writeRunEvidenceFileForTest(t, result.RunEvidencePath, evidence)
	if _, err := VerifyRunEvidenceFile(result.RunEvidencePath); err == nil || !strings.Contains(err.Error(), "regular file") {
		t.Fatalf("symlinked executor report error = %v", err)
	}
}

func TestLegacyRunEvidenceIsInspectableButNotArchivable(t *testing.T) {
	result := writeVerifiableRunEvidence(t)
	evidence := readRunEvidenceFile(t, result.RunEvidencePath)
	evidence.Version = LegacyRunEvidenceVersion
	evidence.RunID = ""
	evidence.HandoffSHA256 = ""
	evidence.ApprovalSHA256 = ""
	evidence.RunConfigSHA256 = ""
	writeRunEvidenceFileForTest(t, result.RunEvidencePath, evidence)
	if _, err := VerifyRunEvidenceFile(result.RunEvidencePath); err != nil {
		t.Fatalf("legacy evidence inspection failed: %v", err)
	}
	if _, err := ArchiveRunEvidence(ArchiveOptions{RunEvidencePath: result.RunEvidencePath, ArchiveDir: filepath.Join(result.WorkDir, "legacy-archive")}); err == nil || !strings.Contains(err.Error(), "read-only") {
		t.Fatalf("legacy archive error = %v", err)
	}
}

func TestWriteReleaseNotesDraftIncludesEvidenceSummary(t *testing.T) {
	result := writeVerifiableRunEvidence(t)
	out := filepath.Join(result.WorkDir, "release-notes.md")
	written, err := WriteReleaseNotesDraft(context.Background(), ReleaseNotesOptions{
		RepoRoot:        result.WorkDir,
		RunEvidencePath: result.RunEvidencePath,
		OutPath:         out,
		Gates:           []string{"go test ./...=pass", "go vet ./...=pass"},
		Now:             fixedNow(),
		RunCommand: func(context.Context, string, ...string) ([]byte, error) {
			return []byte("0123456789abcdef0123456789abcdef01234567\n"), nil
		},
	})
	if err != nil {
		t.Fatalf("WriteReleaseNotesDraft returned error: %v", err)
	}
	if written.Path != out || written.Commit != "0123456789abcdef0123456789abcdef01234567" {
		t.Fatalf("unexpected release notes result: %#v", written)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, expected := range []string{
		"Commit: 0123456789abcdef0123456789abcdef01234567",
		"go test ./...=pass",
		"openudon run-evidence verify: pass",
		"async-evidence.json",
		result.PackageSHA256,
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("release notes missing %q:\n%s", expected, text)
		}
	}
}

func TestWriteReleaseNotesDraftAcceptsExplicitCommit(t *testing.T) {
	result := writeVerifiableRunEvidence(t)
	out := filepath.Join(result.WorkDir, "release-notes-explicit.md")
	written, err := WriteReleaseNotesDraft(context.Background(), ReleaseNotesOptions{
		RepoRoot:        result.WorkDir,
		RunEvidencePath: result.RunEvidencePath,
		OutPath:         out,
		Commit:          "0123456789abcdef0123456789abcdef01234567",
		Now:             fixedNow(),
		RunCommand: func(context.Context, string, ...string) ([]byte, error) {
			t.Fatal("Git command called for explicit commit")
			return nil, nil
		},
	})
	if err != nil {
		t.Fatalf("WriteReleaseNotesDraft returned error: %v", err)
	}
	if written.Commit != "0123456789abcdef0123456789abcdef01234567" {
		t.Fatalf("commit = %q, want explicit revision", written.Commit)
	}
}

func TestWriteReleaseNotesDraftRejectsInvalidExplicitCommit(t *testing.T) {
	result := writeVerifiableRunEvidence(t)
	_, err := WriteReleaseNotesDraft(context.Background(), ReleaseNotesOptions{
		RepoRoot:        result.WorkDir,
		RunEvidencePath: result.RunEvidencePath,
		OutPath:         filepath.Join(result.WorkDir, "release-notes-invalid.md"),
		Commit:          "not-a-revision",
		Now:             fixedNow(),
	})
	if err == nil || !strings.Contains(err.Error(), "hexadecimal") {
		t.Fatalf("error = %v, want hexadecimal revision failure", err)
	}
}

func TestVerifyRunEvidenceFileRejectsBadAsyncSidecars(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(t *testing.T, result *RunResult)
		want   string
	}{
		{
			name: "digest mismatch",
			mutate: func(t *testing.T, result *RunResult) {
				mustWriteFile(t, result.AsyncEvidencePath, []byte("{}\n"))
			},
			want: "digest mismatch",
		},
		{
			name: "unsafe path",
			mutate: func(t *testing.T, result *RunResult) {
				evidence := readRunEvidenceFile(t, result.RunEvidencePath)
				evidence.AsyncEvidenceFiles[0].Path = "../async-evidence.json"
				writeRunEvidenceFileForTest(t, result.RunEvidencePath, evidence)
			},
			want: "safe workdir-relative",
		},
		{
			name: "missing sidecar",
			mutate: func(t *testing.T, result *RunResult) {
				if err := os.Remove(result.AsyncEvidencePath); err != nil {
					t.Fatal(err)
				}
			},
			want: "read async evidence",
		},
		{
			name: "duplicate sidecar ref",
			mutate: func(t *testing.T, result *RunResult) {
				evidence := readRunEvidenceFile(t, result.RunEvidencePath)
				evidence.AsyncEvidenceFiles = append(evidence.AsyncEvidenceFiles, evidence.AsyncEvidenceFiles[0])
				writeRunEvidenceFileForTest(t, result.RunEvidencePath, evidence)
			},
			want: "duplicate async evidence path",
		},
		{
			name: "invalid purpose",
			mutate: func(t *testing.T, result *RunResult) {
				evidence := readRunEvidenceFile(t, result.RunEvidencePath)
				evidence.AsyncEvidenceFiles[0].Purpose = "other"
				writeRunEvidenceFileForTest(t, result.RunEvidencePath, evidence)
			},
			want: "purpose is invalid",
		},
		{
			name: "unknown kind",
			mutate: func(t *testing.T, result *RunResult) {
				evidence, bundle := readRunEvidenceAndBundle(t, result)
				bundle.Records[0].Kind = "mystery"
				writeBundleAndRefreshRef(t, result, evidence, bundle)
			},
			want: "unsupported async evidence record kind",
		},
		{
			name: "missing request payload",
			mutate: func(t *testing.T, result *RunResult) {
				evidence, bundle := readRunEvidenceAndBundle(t, result)
				bundle.Records[0].ExecutionRequest = nil
				writeBundleAndRefreshRef(t, result, evidence, bundle)
			},
			want: "missing execution_request",
		},
		{
			name: "mismatched status payload",
			mutate: func(t *testing.T, result *RunResult) {
				evidence, bundle := readRunEvidenceAndBundle(t, result)
				request := *bundle.Records[0].ExecutionRequest
				status := asyncevidence.NormalizeStatusObservation(asyncevidence.StatusObservation{
					Version:   asyncevidence.StatusObservationVersion,
					Attempt:   request.Attempt,
					Operation: request.Operation,
				})
				bundle.Records[0].StatusObservation = &status
				writeBundleAndRefreshRef(t, result, evidence, bundle)
			},
			want: "unexpected payload for execution_request",
		},
		{
			name: "bad status version",
			mutate: func(t *testing.T, result *RunResult) {
				evidence, bundle := readRunEvidenceAndBundle(t, result)
				request := *bundle.Records[0].ExecutionRequest
				status := asyncevidence.StatusObservation{
					Version:   "bad",
					Attempt:   request.Attempt,
					Operation: request.Operation,
				}
				bundle.Records = append(bundle.Records, AsyncEvidenceRecord{Kind: "status_observation", StatusObservation: &status})
				writeBundleAndRefreshRef(t, result, evidence, bundle)
			},
			want: "async.version_invalid",
		},
		{
			name: "unknown top-level field",
			mutate: func(t *testing.T, result *RunResult) {
				evidence := readRunEvidenceFile(t, result.RunEvidencePath)
				data, err := os.ReadFile(result.AsyncEvidencePath)
				if err != nil {
					t.Fatal(err)
				}
				var raw map[string]any
				if err := json.Unmarshal(data, &raw); err != nil {
					t.Fatal(err)
				}
				raw["extra"] = true
				writeRawBundleAndRefreshRef(t, result, evidence, raw)
			},
			want: "unknown field",
		},
		{
			name: "record count mismatch",
			mutate: func(t *testing.T, result *RunResult) {
				evidence := readRunEvidenceFile(t, result.RunEvidencePath)
				evidence.AsyncEvidenceFiles[0].Records++
				writeRunEvidenceFileForTest(t, result.RunEvidencePath, evidence)
			},
			want: "record count mismatch",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := writeVerifiableRunEvidence(t)
			tc.mutate(t, result)
			_, err := VerifyRunEvidenceFile(result.RunEvidencePath)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected %q error, got %v", tc.want, err)
			}
		})
	}
}

func TestRunValidProductionApprovalPassesDryRun(t *testing.T) {
	root, example := writeFixture(t, fixtureOptions{})
	now := fixedNow()
	approvalPath := writeApprovalTemplate(t, root, example, StateApprovedForProduction, now)

	if _, err := Run(context.Background(), Options{
		RepoRoot:     root,
		ExampleDir:   example,
		Tier:         TierProduction,
		ApprovalPath: approvalPath,
		DryRun:       true,
		Now:          now,
		Assess:       passAssess,
	}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
}

func TestRunProductionWithSandboxApprovalFails(t *testing.T) {
	root, example := writeFixture(t, fixtureOptions{})
	now := fixedNow()
	approvalPath := writeApprovalTemplate(t, root, example, StateApprovedForSandbox, now)

	_, err := Run(context.Background(), Options{
		RepoRoot:     root,
		ExampleDir:   example,
		Tier:         TierProduction,
		ApprovalPath: approvalPath,
		DryRun:       true,
		Now:          now,
		Assess:       passAssess,
	})
	if err == nil || !strings.Contains(err.Error(), "not valid for production") {
		t.Fatalf("expected tier/state failure, got %v", err)
	}
}

func TestRunMissingApprovalFails(t *testing.T) {
	root, example := writeFixture(t, fixtureOptions{})
	_, err := Run(context.Background(), Options{
		RepoRoot:     root,
		ExampleDir:   example,
		Tier:         TierSandbox,
		ApprovalPath: filepath.Join(root, "approvals", "missing.json"),
		DryRun:       true,
		Now:          fixedNow(),
		Assess:       passAssess,
	})
	if err == nil || !strings.Contains(err.Error(), "read approval") {
		t.Fatalf("expected missing approval failure, got %v", err)
	}
}

func TestRunExpiredApprovalFails(t *testing.T) {
	root, example := writeFixture(t, fixtureOptions{})
	now := fixedNow()
	approvalPath := writeApprovalTemplate(t, root, example, StateApprovedForSandbox, now)
	approval := readApprovalFile(t, approvalPath)
	approval.ExpiresAt = now().Add(-time.Hour).UTC().Format(time.RFC3339)
	writeApprovalFile(t, approvalPath, approval)

	_, err := Run(context.Background(), Options{
		RepoRoot:     root,
		ExampleDir:   example,
		Tier:         TierSandbox,
		ApprovalPath: approvalPath,
		DryRun:       true,
		Now:          now,
		Assess:       passAssess,
	})
	if err == nil || !strings.Contains(err.Error(), "expired") {
		t.Fatalf("expected expired approval failure, got %v", err)
	}
}

func TestRunScopeMismatchFails(t *testing.T) {
	root, example := writeFixture(t, fixtureOptions{})
	now := fixedNow()
	approvalPath := writeApprovalTemplate(t, root, example, StateApprovedForSandbox, now)
	approval := readApprovalFile(t, approvalPath)
	approval.Scope = "examples/other"
	writeApprovalFile(t, approvalPath, approval)

	_, err := Run(context.Background(), Options{
		RepoRoot:     root,
		ExampleDir:   example,
		Tier:         TierSandbox,
		ApprovalPath: approvalPath,
		DryRun:       true,
		Now:          now,
		Assess:       passAssess,
	})
	if err == nil || !strings.Contains(err.Error(), "scope") {
		t.Fatalf("expected scope mismatch failure, got %v", err)
	}
}

func TestRunApprovalTrailingJSONFails(t *testing.T) {
	root, example := writeFixture(t, fixtureOptions{})
	now := fixedNow()
	approvalPath := writeApprovalTemplate(t, root, example, StateApprovedForSandbox, now)
	data, err := os.ReadFile(approvalPath)
	if err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, approvalPath, append(data, []byte("{}")...))

	_, err = Run(context.Background(), Options{
		RepoRoot:     root,
		ExampleDir:   example,
		Tier:         TierSandbox,
		ApprovalPath: approvalPath,
		DryRun:       true,
		Now:          now,
		Assess:       passAssess,
	})
	if err == nil || !strings.Contains(err.Error(), "single JSON value") {
		t.Fatalf("expected trailing approval JSON failure, got %v", err)
	}
}

func TestRunDigestMismatchFails(t *testing.T) {
	root, example := writeFixture(t, fixtureOptions{})
	now := fixedNow()
	approvalPath := writeApprovalTemplate(t, root, example, StateApprovedForSandbox, now)
	mustWriteFile(t, filepath.Join(example, "project.md"), []byte("changed\n"))

	_, err := Run(context.Background(), Options{
		RepoRoot:     root,
		ExampleDir:   example,
		Tier:         TierSandbox,
		ApprovalPath: approvalPath,
		DryRun:       true,
		Now:          now,
		Assess:       passAssess,
	})
	if err == nil || !strings.Contains(err.Error(), "handoff input SHA-256 mismatch") {
		t.Fatalf("expected digest mismatch failure, got %v", err)
	}
}

func TestRunNonDryRunWritesRunEvidence(t *testing.T) {
	root, example := writeFixture(t, fixtureOptions{credentialBindings: []string{"support-api.token"}})
	now := fixedNow()
	approvalPath := writeApprovalTemplate(t, root, example, StateApprovedForSandbox, now)
	fakeExecutor := filepath.Join(root, "fake-udon")
	mustWriteFile(t, fakeExecutor, []byte("#!/usr/bin/env bash\nexit 0\n"))
	if err := os.Chmod(fakeExecutor, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("OPENUDON_EXECUTOR", fakeExecutor)
	t.Setenv("UDON_CREDENTIAL_SUPPORT_API_TOKEN", "super-secret")

	result, err := Run(context.Background(), Options{
		RepoRoot:     root,
		ExampleDir:   example,
		Tier:         TierSandbox,
		ApprovalPath: approvalPath,
		WorkDir:      filepath.Join(root, "work"),
		Now:          now,
		Assess:       passAssess,
		Invoke: func(_ context.Context, invocation udonrunner.Invocation) error {
			args := invocation.Argv[1:]
			reportPath := argValue(t, args, "--execution-report")
			writeUdonExecutionReport(t, reportPath, "success", now(), "sha256:"+strings.Repeat("a", 64))
			return nil
		},
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	evidence := readRunEvidenceFile(t, result.RunEvidencePath)
	if evidence.DryRun || !evidence.Executor.Invoked || evidence.Executor.Mode != "internal-runner" {
		t.Fatalf("unexpected executor evidence: %#v", evidence.Executor)
	}
	if evidence.StageKind != "executor" || runEvidenceGateStatus(evidence, "executor_invocation") != "pass" {
		t.Fatalf("unexpected handoff evidence: %#v", evidence)
	}
	if evidence.Executor.ReportPath == "" {
		t.Fatalf("executor report path missing: %#v", evidence.Executor)
	}
	if evidence.StagePath == "" || evidence.WorkflowPath == "" || len(evidence.PackagePaths) == 0 {
		t.Fatalf("evidence missing package staging fields: %#v", evidence)
	}
	if _, err := VerifyRunEvidenceFile(result.RunEvidencePath); err != nil {
		t.Fatalf("VerifyRunEvidenceFile rejected report-backed sidecar: %v", err)
	}
	_, bundle := readReferencedAsyncEvidence(t, result.WorkDir, evidence)
	request := asyncExecutionRequest(t, bundle)
	response := asyncExecutionResponse(t, bundle)
	if request.Transport["runner_mode"] != "internal-runner" || request.Transport["stage_kind"] != "executor" || request.Transport["dry_run"] != "false" {
		t.Fatalf("unexpected async request transport: %#v", request.Transport)
	}
	if response.Outcome != "accepted" || response.ErrorSummary != "" {
		t.Fatalf("unexpected async response: %#v", response)
	}
	status := asyncStatusObservation(t, bundle)
	if status.Status != "success" || status.TerminalityHint != "terminal" || len(status.PayloadDigests) != 1 {
		t.Fatalf("unexpected async status observation: %#v", status)
	}
	read := asyncConfirmationReadObservation(t, bundle)
	if read.Outcome != "confirmed" || len(read.ProjectedDigests) != 1 {
		t.Fatalf("unexpected async confirmation-read observation: %#v", read)
	}
	data, err := os.ReadFile(result.RunEvidencePath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "super-secret") {
		t.Fatalf("run evidence leaked credential value:\n%s", data)
	}
	withoutExecutorGate := evidence
	withoutExecutorGate.Gates = make([]RunEvidenceGate, 0, len(evidence.Gates)-1)
	for _, gate := range evidence.Gates {
		if gate.Name != "executor_invocation" {
			withoutExecutorGate.Gates = append(withoutExecutorGate.Gates, gate)
		}
	}
	writeRunEvidenceFileForTest(t, result.RunEvidencePath, withoutExecutorGate)
	if _, err := VerifyRunEvidenceFile(result.RunEvidencePath); err == nil || !strings.Contains(err.Error(), "requires an executor_invocation gate") {
		t.Fatalf("executor-gate-free evidence verification error = %v", err)
	}
	writeRunEvidenceFileForTest(t, result.RunEvidencePath, evidence)
	evidence.Executor.ReportPath = ""
	evidence.Executor.ReportSHA256 = ""
	evidence.Executor.ReportSize = 0
	writeRunEvidenceFileForTest(t, result.RunEvidencePath, evidence)
	if _, err := VerifyRunEvidenceFile(result.RunEvidencePath); err == nil || !strings.Contains(err.Error(), "requires an executor report") {
		t.Fatalf("report-free successful evidence verification error = %v", err)
	}
}

func TestRunNonDryRunRejectsMissingExecutorReport(t *testing.T) {
	root, example := writeFixture(t, fixtureOptions{})
	now := fixedNow()
	approvalPath := writeApprovalTemplate(t, root, example, StateApprovedForSandbox, now)
	fakeExecutor := filepath.Join(root, "fake-udon")
	mustWriteFile(t, fakeExecutor, []byte("#!/usr/bin/env bash\nexit 0\n"))
	if err := os.Chmod(fakeExecutor, 0o755); err != nil {
		t.Fatal(err)
	}

	result, err := Run(context.Background(), Options{
		RepoRoot: root, ExampleDir: example, Tier: TierSandbox, ApprovalPath: approvalPath,
		WorkDir: filepath.Join(root, "work"), Now: now, Assess: passAssess,
		Env: []string{"OPENUDON_EXECUTOR=" + fakeExecutor},
		Invoke: func(context.Context, udonrunner.Invocation) error {
			return nil
		},
	})
	if err == nil || !strings.Contains(err.Error(), "executor report") {
		t.Fatalf("missing executor report result=%#v error=%v", result, err)
	}
	if result == nil || result.WorkDir == "" {
		t.Fatalf("successful invocation failure must retain the partial run result: %#v", result)
	}
	if result.RunEvidencePath != "" {
		t.Fatalf("missing-report run recorded passing evidence: %#v", result)
	}
	if _, statErr := os.Lstat(filepath.Join(result.WorkDir, "run-evidence.json")); !os.IsNotExist(statErr) {
		t.Fatalf("missing-report run wrote evidence, stat error=%v", statErr)
	}
}

func TestRunNonDryRunWritesFailureEvidence(t *testing.T) {
	root, example := writeFixture(t, fixtureOptions{})
	now := fixedNow()
	approvalPath := writeApprovalTemplate(t, root, example, StateApprovedForSandbox, now)
	fakeExecutor := filepath.Join(root, "fake-udon")
	mustWriteFile(t, fakeExecutor, []byte("#!/usr/bin/env bash\nexit 0\n"))
	if err := os.Chmod(fakeExecutor, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("OPENUDON_EXECUTOR", fakeExecutor)
	invokeErr := errors.New("executor failed")

	result, err := Run(context.Background(), Options{
		RepoRoot:     root,
		ExampleDir:   example,
		Tier:         TierSandbox,
		ApprovalPath: approvalPath,
		WorkDir:      filepath.Join(root, "work"),
		Now:          now,
		Assess:       passAssess,
		Invoke: func(_ context.Context, invocation udonrunner.Invocation) error {
			args := invocation.Argv[1:]
			reportPath := argValue(t, args, "--execution-report")
			writeUdonExecutionReport(t, reportPath, "error", now(), "")
			return invokeErr
		},
	})
	if err == nil || !strings.Contains(err.Error(), "executor failed") {
		t.Fatalf("expected executor failure, got result=%#v err=%v", result, err)
	}
	if result == nil || result.RunEvidencePath == "" {
		t.Fatalf("expected partial result with run evidence, got %#v", result)
	}
	evidence := readRunEvidenceFile(t, result.RunEvidencePath)
	if evidence.StageKind != "executor" || !evidence.Executor.Invoked || runEvidenceGateStatus(evidence, "executor_invocation") != "fail" {
		t.Fatalf("unexpected failure evidence: %#v", evidence)
	}
	_, bundle := readReferencedAsyncEvidence(t, result.WorkDir, evidence)
	response := asyncExecutionResponse(t, bundle)
	if response.Outcome != "fatal_failure" || response.ErrorSummary == "" {
		t.Fatalf("unexpected async failure response: %#v", response)
	}
	status := asyncStatusObservation(t, bundle)
	if status.Status != "error" || len(status.PayloadDigests) != 0 {
		t.Fatalf("unexpected async failure status observation: %#v", status)
	}
	if hasConfirmationReadObservation(bundle) {
		t.Fatalf("failure bundle should not include confirmation-read observation: %#v", bundle)
	}
}

func TestRunFailedQualityReportFails(t *testing.T) {
	root, example := writeFixture(t, fixtureOptions{qualityStatus: "fail"})
	now := fixedNow()
	approvalPath := filepath.Join(root, "approval.json")
	mustWriteFile(t, approvalPath, []byte(`{"version":"openudon.approval.v1"}`))

	_, err := Run(context.Background(), Options{
		RepoRoot:     root,
		ExampleDir:   example,
		Tier:         TierSandbox,
		ApprovalPath: approvalPath,
		DryRun:       true,
		Now:          now,
		Assess:       passAssess,
	})
	if err == nil || !strings.Contains(err.Error(), "stored quality") {
		t.Fatalf("expected failed quality failure, got %v", err)
	}

	root, example = writeFixture(t, fixtureOptions{})
	approvalPath = writeApprovalTemplate(t, root, example, StateApprovedForSandbox, now)
	_, err = Run(context.Background(), Options{
		RepoRoot:     root,
		ExampleDir:   example,
		Tier:         TierSandbox,
		ApprovalPath: approvalPath,
		DryRun:       true,
		Now:          now,
		Assess: func(context.Context, synthesize.Options) (*synthesize.QualityReport, error) {
			return &synthesize.QualityReport{Status: "fail"}, nil
		},
	})
	if err == nil || !strings.Contains(err.Error(), "current quality") {
		t.Fatalf("expected current quality failure, got %v", err)
	}
}

func TestRunMalformedHandoffManifestFails(t *testing.T) {
	root, example := writeFixture(t, fixtureOptions{malformedHandoff: true})
	approvalPath := filepath.Join(root, "approval.json")
	mustWriteFile(t, approvalPath, []byte(`{"version":"openudon.approval.v1"}`))

	_, err := Run(context.Background(), Options{
		RepoRoot:     root,
		ExampleDir:   example,
		Tier:         TierSandbox,
		ApprovalPath: approvalPath,
		DryRun:       true,
		Now:          fixedNow(),
		Assess:       passAssess,
	})
	if err == nil || !strings.Contains(err.Error(), "valid JSON") {
		t.Fatalf("expected malformed handoff failure, got %v", err)
	}
}

func TestRunUnsafeManifestFails(t *testing.T) {
	for name, opts := range map[string]fixtureOptions{
		"credential values": {valuesAllowed: true},
		"direct production": {directProduction: true},
	} {
		t.Run(name, func(t *testing.T) {
			root, example := writeFixture(t, opts)
			now := fixedNow()
			approvalPath := writeApprovalTemplateWithoutPolicyCheck(t, root, example, StateApprovedForSandbox, now)
			_, err := Run(context.Background(), Options{
				RepoRoot:     root,
				ExampleDir:   example,
				Tier:         TierSandbox,
				ApprovalPath: approvalPath,
				DryRun:       true,
				Now:          now,
				Assess:       passAssess,
			})
			if err == nil || !strings.Contains(err.Error(), "manifest") {
				t.Fatalf("expected unsafe manifest failure, got %v", err)
			}
		})
	}
}

func TestRunNonDryRunInvokesRunner(t *testing.T) {
	root, example := writeFixture(t, fixtureOptions{})
	now := fixedNow()
	approvalPath := writeApprovalTemplate(t, root, example, StateApprovedForSandbox, now)
	runnerPath := filepath.Join(root, "fake-runner")
	mustWriteFile(t, runnerPath, []byte("#!/usr/bin/env bash\nexit 0\n"))
	if err := os.Chmod(runnerPath, 0o755); err != nil {
		t.Fatal(err)
	}
	var gotName string
	var gotArgs []string

	result, err := Run(context.Background(), Options{
		RepoRoot:     root,
		ExampleDir:   example,
		Tier:         TierSandbox,
		ApprovalPath: approvalPath,
		RunnerPath:   runnerPath,
		WorkDir:      filepath.Join(root, "work"),
		Now:          now,
		Assess:       passAssess,
		Invoke: func(_ context.Context, invocation udonrunner.Invocation) error {
			gotName = invocation.Argv[0]
			gotArgs = invocation.Argv[1:]
			writeExternalRunnerReport(t, invocation, now())
			return nil
		},
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if gotName != runnerPath {
		t.Fatalf("runner path = %q", gotName)
	}
	if len(gotArgs) != 6 || gotArgs[0] != "--config" || gotArgs[1] != result.RunConfigPath || gotArgs[2] != "--config-sha256" || gotArgs[4] != "--approval" || gotArgs[5] != approvalPath {
		t.Fatalf("runner args = %#v", gotArgs)
	}
	data, err := os.ReadFile(result.RunConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"version": "openudon.executor-run.v2"`) || !strings.Contains(string(data), `"workflow_path": "workflows/workflow.uws.yaml"`) {
		t.Fatalf("unexpected run config:\n%s", data)
	}
	evidence := readRunEvidenceFile(t, result.RunEvidencePath)
	if evidence.Executor.Mode != "external-runner" || evidence.StageKind != "preflight" || runEvidenceGateStatus(evidence, "executor_invocation") != "pass" {
		t.Fatalf("unexpected external runner evidence: %#v", evidence)
	}
	wantArgv := append([]string{runnerPath}, gotArgs...)
	if strings.Join(evidence.Executor.Argv, "\n") != strings.Join(wantArgv, "\n") {
		t.Fatalf("external runner evidence argv = %#v, want %#v", evidence.Executor.Argv, wantArgv)
	}
	_, bundle := readReferencedAsyncEvidence(t, result.WorkDir, evidence)
	request := asyncExecutionRequest(t, bundle)
	if request.Transport["runner_mode"] != "external-runner" || request.Transport["stage_kind"] != "preflight" {
		t.Fatalf("unexpected external async transport: %#v", request.Transport)
	}
	var asyncArgv []string
	if err := json.Unmarshal([]byte(request.Metadata["argv_json"]), &asyncArgv); err != nil {
		t.Fatalf("async argv metadata is not JSON array: %v %#v", err, request.Metadata)
	}
	if strings.Join(asyncArgv, "\n") != strings.Join(wantArgv, "\n") || request.Metadata["runner_path"] != runnerPath {
		t.Fatalf("unexpected async argv metadata: argv=%#v metadata=%#v", asyncArgv, request.Metadata)
	}
}

func TestRunExternalRunnerWritesFailureEvidence(t *testing.T) {
	root, example := writeFixture(t, fixtureOptions{})
	now := fixedNow()
	approvalPath := writeApprovalTemplate(t, root, example, StateApprovedForSandbox, now)
	runnerPath := filepath.Join(root, "fake-runner")
	mustWriteFile(t, runnerPath, []byte("#!/usr/bin/env bash\nexit 0\n"))
	if err := os.Chmod(runnerPath, 0o755); err != nil {
		t.Fatal(err)
	}
	invokeErr := errors.New("outer runner failed")

	result, err := Run(context.Background(), Options{
		RepoRoot:     root,
		ExampleDir:   example,
		Tier:         TierSandbox,
		ApprovalPath: approvalPath,
		RunnerPath:   runnerPath,
		WorkDir:      filepath.Join(root, "work"),
		Now:          now,
		Assess:       passAssess,
		Invoke: func(context.Context, udonrunner.Invocation) error {
			return invokeErr
		},
	})
	if err == nil || !strings.Contains(err.Error(), "outer runner failed") {
		t.Fatalf("expected external runner failure, got result=%#v err=%v", result, err)
	}
	if result == nil || result.RunEvidencePath == "" {
		t.Fatalf("expected partial result with run evidence, got %#v", result)
	}
	evidence := readRunEvidenceFile(t, result.RunEvidencePath)
	if evidence.Executor.Mode != "external-runner" || evidence.StageKind != "preflight" || runEvidenceGateStatus(evidence, "executor_invocation") != "fail" {
		t.Fatalf("unexpected external failure evidence: %#v", evidence)
	}
	_, bundle := readReferencedAsyncEvidence(t, result.WorkDir, evidence)
	request := asyncExecutionRequest(t, bundle)
	response := asyncExecutionResponse(t, bundle)
	if request.Transport["runner_mode"] != "external-runner" || request.Transport["stage_kind"] != "preflight" {
		t.Fatalf("unexpected external failure async transport: %#v", request.Transport)
	}
	if response.Outcome != "fatal_failure" || response.ErrorSummary == "" {
		t.Fatalf("unexpected external failure async response: %#v", response)
	}
}

func TestRunExternalRunnerInvocationEnvironmentIsAllowlisted(t *testing.T) {
	root, example := writeFixture(t, fixtureOptions{credentialBindings: []string{"support-api.token"}})
	now := fixedNow()
	approvalPath := writeApprovalTemplate(t, root, example, StateApprovedForSandbox, now)
	runnerPath := filepath.Join(root, "fake-runner")
	mustWriteFile(t, runnerPath, []byte("#!/usr/bin/env bash\nexit 0\n"))
	if err := os.Chmod(runnerPath, 0o755); err != nil {
		t.Fatal(err)
	}
	var invocation udonrunner.Invocation
	_, err := Run(context.Background(), Options{
		RepoRoot: root, ExampleDir: example, Tier: TierSandbox, ApprovalPath: approvalPath,
		RunnerPath: runnerPath, WorkDir: filepath.Join(root, "work"), Now: now, Assess: passAssess,
		Env: []string{
			"OPENUDON_EXECUTOR=/trusted/udon", "UDON_CREDENTIAL_SUPPORT_API_TOKEN=declared-secret",
			"PATH=/trusted/bin", "AWS_SECRET_ACCESS_KEY=must-not-pass", "HTTPS_PROXY=http://must-not-pass",
			"SSH_AUTH_SOCK=/must-not-pass", "UNRELATED_SENTINEL=must-not-pass",
		},
		Invoke: func(_ context.Context, got udonrunner.Invocation) error {
			invocation = got
			writeExternalRunnerReport(t, got, now())
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(invocation.Env, "\n")
	for _, required := range []string{"OPENUDON_EXECUTOR=/trusted/udon", "UDON_CREDENTIAL_SUPPORT_API_TOKEN=declared-secret", "PATH=/trusted/bin"} {
		if !strings.Contains(joined, required) {
			t.Fatalf("required %q missing from outer runner env: %#v", required, invocation.Env)
		}
	}
	for _, forbidden := range []string{"AWS_SECRET_ACCESS_KEY", "HTTPS_PROXY", "SSH_AUTH_SOCK", "UNRELATED_SENTINEL"} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("%s leaked into outer runner env: %#v", forbidden, invocation.Env)
		}
	}
}

func TestRunExternalRevalidatesPinnedConfigAndApproval(t *testing.T) {
	root, example := writeFixture(t, fixtureOptions{})
	now := fixedNow()
	approvalPath := writeApprovalTemplate(t, root, example, StateApprovedForSandbox, now)
	runnerPath := filepath.Join(root, "fake-runner")
	mustWriteFile(t, runnerPath, []byte("#!/usr/bin/env bash\nexit 0\n"))
	if err := os.Chmod(runnerPath, 0o755); err != nil {
		t.Fatal(err)
	}
	parent, err := Run(context.Background(), Options{
		RepoRoot: root, ExampleDir: example, Tier: TierSandbox, ApprovalPath: approvalPath,
		RunnerPath: runnerPath, WorkDir: filepath.Join(root, "work"), Now: now, Assess: passAssess,
		Invoke: func(_ context.Context, invocation udonrunner.Invocation) error {
			writeExternalRunnerReport(t, invocation, now())
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	configBytes, err := os.ReadFile(parent.RunConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	var parentConfig RunConfig
	if err := json.Unmarshal(configBytes, &parentConfig); err != nil {
		t.Fatal(err)
	}
	parentReportPath, err := externalExecutorReportPath(parentConfig)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(parentReportPath); err != nil {
		t.Fatal(err)
	}
	configSum := sha256.Sum256(configBytes)
	configDigest := fmt.Sprintf("%x", configSum[:])
	invoked := false
	base := ExternalOptions{
		ConfigPath: parent.RunConfigPath, ConfigSHA256: configDigest, ApprovalPath: approvalPath,
		Env: []string{"OPENUDON_EXECUTOR=/bin/true"}, Now: now, Assess: passAssess,
		Invoke: func(_ context.Context, invocation udonrunner.Invocation) error {
			invoked = true
			writeUdonExecutionReport(t, argValue(t, invocation.Argv, "--execution-report"), "success", now(), "")
			return nil
		},
	}
	if _, err := RunExternal(context.Background(), base); err != nil {
		t.Fatalf("validated external run failed: %v", err)
	}
	if !invoked {
		t.Fatal("validated external run did not invoke executor")
	}

	t.Run("config replacement", func(t *testing.T) {
		mustWriteFile(t, parent.RunConfigPath, append(append([]byte(nil), configBytes...), ' '))
		defer mustWriteFile(t, parent.RunConfigPath, configBytes)
		if _, err := RunExternal(context.Background(), base); err == nil || !strings.Contains(err.Error(), "SHA-256 mismatch") {
			t.Fatalf("config replacement error = %v", err)
		}
	})

	t.Run("approval replacement", func(t *testing.T) {
		approvalBytes, err := os.ReadFile(approvalPath)
		if err != nil {
			t.Fatal(err)
		}
		mustWriteFile(t, approvalPath, append(append([]byte(nil), approvalBytes...), ' '))
		defer mustWriteFile(t, approvalPath, approvalBytes)
		if _, err := RunExternal(context.Background(), base); err == nil || !strings.Contains(err.Error(), "approval SHA-256") {
			t.Fatalf("approval replacement error = %v", err)
		}
	})

	t.Run("forged canonical config", func(t *testing.T) {
		var forged RunConfig
		if err := json.Unmarshal(configBytes, &forged); err != nil {
			t.Fatal(err)
		}
		forged.WorkflowPath = "workflows/other.uws.yaml"
		forgedBytes, err := json.MarshalIndent(forged, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		forgedBytes = append(forgedBytes, '\n')
		mustWriteFile(t, parent.RunConfigPath, forgedBytes)
		defer mustWriteFile(t, parent.RunConfigPath, configBytes)
		sum := sha256.Sum256(forgedBytes)
		forgedOpts := base
		forgedOpts.ConfigSHA256 = fmt.Sprintf("%x", sum[:])
		if _, err := RunExternal(context.Background(), forgedOpts); err == nil || !strings.Contains(err.Error(), "canonical validated") {
			t.Fatalf("forged config error = %v", err)
		}
	})
}

func TestRunExternalRejectsLegacyV1Config(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy-config.json")
	data := []byte("{\"version\":\"openudon.executor-run.v1\"}\n")
	mustWriteFile(t, path, data)
	sum := sha256.Sum256(data)
	_, err := RunExternal(context.Background(), ExternalOptions{ConfigPath: path, ConfigSHA256: fmt.Sprintf("%x", sum[:]), ApprovalPath: filepath.Join(t.TempDir(), "unused")})
	if err == nil || !strings.Contains(err.Error(), "cannot execute") {
		t.Fatalf("legacy config error = %v", err)
	}
}

func TestRunRejectsUnsafeRunnerPathOverride(t *testing.T) {
	for _, tc := range []struct {
		name       string
		runnerPath string
		want       string
	}{
		{name: "relative", runnerPath: "fake-runner", want: "absolute path"},
		{name: "missing", runnerPath: filepath.Join(t.TempDir(), "missing-runner"), want: "executable file"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, example := writeFixture(t, fixtureOptions{})
			now := fixedNow()
			approvalPath := writeApprovalTemplate(t, root, example, StateApprovedForSandbox, now)
			_, err := Run(context.Background(), Options{
				RepoRoot:     root,
				ExampleDir:   example,
				Tier:         TierSandbox,
				ApprovalPath: approvalPath,
				RunnerPath:   tc.runnerPath,
				WorkDir:      filepath.Join(root, "work"),
				Now:          now,
				Assess:       passAssess,
				Invoke: func(context.Context, udonrunner.Invocation) error {
					t.Fatal("runner should not be invoked")
					return nil
				},
			})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected %q error, got %v", tc.want, err)
			}
		})
	}
}

func TestRunNonDryRunUsesDefaultGoRunner(t *testing.T) {
	root, example := writeFixture(t, fixtureOptions{})
	now := fixedNow()
	approvalPath := writeApprovalTemplate(t, root, example, StateApprovedForSandbox, now)
	fakeExecutor := filepath.Join(root, "fake-udon")
	mustWriteFile(t, fakeExecutor, []byte("#!/usr/bin/env bash\nexit 0\n"))
	if err := os.Chmod(fakeExecutor, 0o755); err != nil {
		t.Fatal(err)
	}
	var gotName string
	var gotArgs []string
	result, err := Run(context.Background(), Options{
		RepoRoot:     root,
		ExampleDir:   example,
		Tier:         TierSandbox,
		ApprovalPath: approvalPath,
		WorkDir:      filepath.Join(root, "work"),
		Now:          now,
		Assess:       passAssess,
		Env:          []string{"OPENUDON_EXECUTOR=" + fakeExecutor},
		Invoke: func(_ context.Context, invocation udonrunner.Invocation) error {
			gotName = invocation.Argv[0]
			gotArgs = append([]string(nil), invocation.Argv[1:]...)
			args := invocation.Argv[1:]
			writeUdonExecutionReport(t, argValue(t, args, "--execution-report"), "success", now(), "")
			return nil
		},
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if gotName != fakeExecutor {
		t.Fatalf("executor path = %q, want %q", gotName, fakeExecutor)
	}
	stagedWorkdir := argValue(t, gotArgs, "--workdir")
	if !strings.HasPrefix(stagedWorkdir, filepath.Join(root, "work", "run-")) || !strings.Contains(stagedWorkdir, string(os.PathSeparator)+"stage.") {
		t.Fatalf("executor workdir = %q, want fresh stage under work; args=%#v", stagedWorkdir, gotArgs)
	}
	if gotWorkflow := argValue(t, gotArgs, "--workflow"); gotWorkflow != filepath.Join(stagedWorkdir, "workflows", "workflow.uws.yaml") {
		t.Fatalf("executor workflow = %q, want staged workflow under %q; args=%#v", gotWorkflow, stagedWorkdir, gotArgs)
	}
	if result.RunConfigPath == "" {
		t.Fatalf("missing run config path in result: %+v", result)
	}
}

func TestRunConfigIncludesNestedOpenAPIPaths(t *testing.T) {
	root, example := writeFixture(t, fixtureOptions{extraRequiredInputs: []string{"openapi/nested/support.yaml"}})
	if err := os.MkdirAll(filepath.Join(example, "openapi", "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(example, "openapi", "nested", "support.yaml"), []byte("openapi: 3.0.0\ninfo: {title: Support, version: 1.0.0}\npaths: {}\n"))
	now := fixedNow()
	approvalPath := writeApprovalTemplate(t, root, example, StateApprovedForSandbox, now)
	result, err := Run(context.Background(), Options{
		RepoRoot:     root,
		ExampleDir:   example,
		Tier:         TierSandbox,
		ApprovalPath: approvalPath,
		DryRun:       true,
		WorkDir:      filepath.Join(root, "work"),
		Now:          now,
		Assess:       passAssess,
	})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(result.RunConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"openapi/nested/support.yaml"`) {
		t.Fatalf("run config missing nested OpenAPI path:\n%s", data)
	}
	if !strings.Contains(string(data), `"api_source_paths"`) {
		t.Fatalf("run config missing api_source_paths compatibility field:\n%s", data)
	}
	var config RunConfig
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"expected/plan.json",
		"expected/quality.json",
		"expected/refinement.json",
		"expected/review-handoff.json",
		"expected/review.md",
		"openapi/nested/support.yaml",
		"project.md",
		"workflows/intent.hcl",
		"workflows/workflow.hcl",
		"workflows/workflow.uws.yaml",
	}
	if strings.Join(config.PackagePaths, "\n") != strings.Join(want, "\n") {
		t.Fatalf("package paths = %#v, want %#v", config.PackagePaths, want)
	}
}

func TestRunConfigIncludesAdvisorySecuritySidecarPackagePath(t *testing.T) {
	root, example := writeFixture(t, fixtureOptions{extraRequiredInputs: []string{
		"google-discovery/gmail.json",
		"google-discovery/gmail.security.json",
	}})
	mustWriteFile(t, filepath.Join(example, "google-discovery", "gmail.json"), []byte(`{"discoveryVersion":"v1"}`))
	mustWriteFile(t, filepath.Join(example, "google-discovery", "gmail.security.json"), []byte(`{"security_schemes":[]}`))
	now := fixedNow()
	approvalPath := writeApprovalTemplate(t, root, example, StateApprovedForSandbox, now)
	result, err := Run(context.Background(), Options{
		RepoRoot:     root,
		ExampleDir:   example,
		Tier:         TierSandbox,
		ApprovalPath: approvalPath,
		DryRun:       true,
		WorkDir:      filepath.Join(root, "work"),
		Now:          now,
		Assess:       passAssess,
	})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(result.RunConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	var config RunConfig
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}
	if !stringSliceContains(config.PackagePaths, "google-discovery/gmail.security.json") {
		t.Fatalf("package paths missing advisory security sidecar: %#v", config.PackagePaths)
	}
	if !stringSliceContains(config.APISourcePaths, "google-discovery/gmail.json") || !stringSliceContains(config.OpenAPIPaths, "google-discovery/gmail.json") {
		t.Fatalf("API source paths missing source: api=%#v openapi=%#v", config.APISourcePaths, config.OpenAPIPaths)
	}
	if stringSliceContains(config.APISourcePaths, "google-discovery/gmail.security.json") || stringSliceContains(config.OpenAPIPaths, "google-discovery/gmail.security.json") {
		t.Fatalf("API source paths included advisory security sidecar: api=%#v openapi=%#v", config.APISourcePaths, config.OpenAPIPaths)
	}
}

func TestRunConfigIncludesRuntimeDataFile(t *testing.T) {
	root, example := writeFixture(t, fixtureOptions{extraRequiredInputs: []string{"expected/data.hcl"}})
	mustWriteFile(t, filepath.Join(example, "expected", "data.hcl"), []byte("inputs { recipient_email = \"me@example.com\" }\n"))
	now := fixedNow()
	approvalPath := writeApprovalTemplate(t, root, example, StateApprovedForSandbox, now)
	result, err := Run(context.Background(), Options{
		RepoRoot:     root,
		ExampleDir:   example,
		Tier:         TierSandbox,
		ApprovalPath: approvalPath,
		DryRun:       true,
		WorkDir:      filepath.Join(root, "work"),
		Now:          now,
		Assess:       passAssess,
	})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(result.RunConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	var config RunConfig
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}
	if !stringSliceContains(config.PackagePaths, "expected/data.hcl") {
		t.Fatalf("package paths missing runtime data file: %#v", config.PackagePaths)
	}
	if !stringSliceContains(config.DataFiles, "expected/data.hcl") {
		t.Fatalf("data files missing runtime data file: %#v", config.DataFiles)
	}
}

func TestRunRejectsOpenAPIFileMissingFromHandoffInputs(t *testing.T) {
	root, example := writeFixture(t, fixtureOptions{})
	if err := os.MkdirAll(filepath.Join(example, "openapi", "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(example, "openapi", "nested", "support.yaml"), []byte("openapi: 3.0.0\ninfo: {title: Support, version: 1.0.0}\npaths: {}\n"))

	_, err := Run(context.Background(), Options{
		RepoRoot:     root,
		ExampleDir:   example,
		Tier:         TierSandbox,
		ApprovalPath: filepath.Join(root, "missing-approval.json"),
		DryRun:       true,
		Now:          fixedNow(),
		Assess:       passAssess,
	})
	if err == nil || !strings.Contains(err.Error(), "openapi/nested/support.yaml") {
		t.Fatalf("expected missing OpenAPI handoff input error, got %v", err)
	}
}

func TestRunRejectsAdvisorySecuritySidecarMissingFromHandoffInputs(t *testing.T) {
	root, example := writeFixture(t, fixtureOptions{extraRequiredInputs: []string{"google-discovery/gmail.json"}})
	mustWriteFile(t, filepath.Join(example, "google-discovery", "gmail.json"), []byte(`{"discoveryVersion":"v1"}`))
	mustWriteFile(t, filepath.Join(example, "google-discovery", "gmail.security.json"), []byte(`{"security_schemes":[]}`))

	_, err := Run(context.Background(), Options{
		RepoRoot:     root,
		ExampleDir:   example,
		Tier:         TierSandbox,
		ApprovalPath: filepath.Join(root, "missing-approval.json"),
		DryRun:       true,
		Now:          fixedNow(),
		Assess:       passAssess,
	})
	if err == nil || !strings.Contains(err.Error(), "google-discovery/gmail.security.json") {
		t.Fatalf("expected missing advisory security sidecar handoff input error, got %v", err)
	}
}

func TestRunRejectsListedAdvisorySecuritySidecarMissingFromPackage(t *testing.T) {
	root, example := writeFixture(t, fixtureOptions{extraRequiredInputs: []string{
		"google-discovery/gmail.json",
		"google-discovery/gmail.security.json",
	}})
	mustWriteFile(t, filepath.Join(example, "google-discovery", "gmail.json"), []byte(`{"discoveryVersion":"v1"}`))

	_, err := Run(context.Background(), Options{
		RepoRoot:     root,
		ExampleDir:   example,
		Tier:         TierSandbox,
		ApprovalPath: filepath.Join(root, "missing-approval.json"),
		DryRun:       true,
		Now:          fixedNow(),
		Assess:       passAssess,
	})
	if err == nil || !strings.Contains(err.Error(), "google-discovery/gmail.security.json") {
		t.Fatalf("expected missing listed advisory security sidecar error, got %v", err)
	}
}

func TestRunRejectsOpenAPISymlink(t *testing.T) {
	root, example := writeFixture(t, fixtureOptions{extraRequiredInputs: []string{"openapi/support.yaml"}})
	if err := os.MkdirAll(filepath.Join(example, "openapi"), 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, "outside.yaml")
	mustWriteFile(t, target, []byte("openapi: 3.0.0\n"))
	if err := os.Symlink(target, filepath.Join(example, "openapi", "support.yaml")); err != nil {
		t.Fatal(err)
	}

	_, err := Run(context.Background(), Options{
		RepoRoot:     root,
		ExampleDir:   example,
		Tier:         TierSandbox,
		ApprovalPath: filepath.Join(root, "missing-approval.json"),
		DryRun:       true,
		Now:          fixedNow(),
		Assess:       passAssess,
	})
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected OpenAPI symlink error, got %v", err)
	}
}

func TestRunRejectsSymlinkedProjectBeforeApproval(t *testing.T) {
	root, example := writeFixture(t, fixtureOptions{})
	now := fixedNow()
	approvalPath := writeApprovalTemplate(t, root, example, StateApprovedForSandbox, now)
	projectPath := filepath.Join(example, "project.md")
	target := filepath.Join(root, "outside-project.md")
	mustWriteFile(t, target, []byte("# Outside\n"))
	if err := os.Remove(projectPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, projectPath); err != nil {
		t.Fatal(err)
	}

	_, err := Run(context.Background(), Options{
		RepoRoot:     root,
		ExampleDir:   example,
		Tier:         TierSandbox,
		ApprovalPath: approvalPath,
		DryRun:       true,
		Now:          now,
		Assess:       passAssess,
	})
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected project symlink rejection, got %v", err)
	}
}

func TestRunRejectsSymlinkedWorkflowBeforeExecutorInvocation(t *testing.T) {
	root, example := writeFixture(t, fixtureOptions{})
	now := fixedNow()
	approvalPath := writeApprovalTemplate(t, root, example, StateApprovedForSandbox, now)
	workflowPath := filepath.Join(example, "workflows", "workflow.uws.yaml")
	target := filepath.Join(root, "outside-workflow.yaml")
	mustWriteFile(t, target, []byte("version: outside\n"))
	if err := os.Remove(workflowPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, workflowPath); err != nil {
		t.Fatal(err)
	}

	_, err := Run(context.Background(), Options{
		RepoRoot:     root,
		ExampleDir:   example,
		Tier:         TierSandbox,
		ApprovalPath: approvalPath,
		RunnerPath:   filepath.Join(root, "fake-runner"),
		Now:          now,
		Assess:       passAssess,
		Invoke: func(context.Context, udonrunner.Invocation) error {
			t.Fatal("runner should not be invoked for symlinked workflow")
			return nil
		},
	})
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected workflow symlink rejection, got %v", err)
	}
}

func TestRunRejectsSymlinkedExampleDirBeforeApproval(t *testing.T) {
	root, realExample := writeFixture(t, fixtureOptions{})
	linkExample := filepath.Join(root, "examples", "support-email-link")
	if err := os.Symlink(realExample, linkExample); err != nil {
		t.Fatal(err)
	}

	_, err := Run(context.Background(), Options{
		RepoRoot:     root,
		ExampleDir:   linkExample,
		Tier:         TierSandbox,
		ApprovalPath: filepath.Join(root, "missing-approval.json"),
		DryRun:       true,
		Now:          fixedNow(),
		Assess:       passAssess,
	})
	if err == nil || !strings.Contains(err.Error(), "package root must not be a symlink") {
		t.Fatalf("expected package root symlink rejection, got %v", err)
	}
}

func TestRunPackageDigestChangesWhenOpenAPIChanges(t *testing.T) {
	root, example := writeFixture(t, fixtureOptions{extraRequiredInputs: []string{"openapi/support.yaml"}})
	if err := os.MkdirAll(filepath.Join(example, "openapi"), 0o755); err != nil {
		t.Fatal(err)
	}
	openAPIPath := filepath.Join(example, "openapi", "support.yaml")
	mustWriteFile(t, openAPIPath, []byte("openapi: 3.0.0\ninfo: {title: Support, version: 1.0.0}\npaths: {}\n"))
	now := fixedNow()
	approvalPath := writeApprovalTemplate(t, root, example, StateApprovedForSandbox, now)
	mustWriteFile(t, openAPIPath, []byte("openapi: 3.0.0\ninfo: {title: Changed, version: 1.0.0}\npaths: {}\n"))

	_, err := Run(context.Background(), Options{
		RepoRoot:     root,
		ExampleDir:   example,
		Tier:         TierSandbox,
		ApprovalPath: approvalPath,
		DryRun:       true,
		Now:          now,
		Assess:       passAssess,
	})
	if err == nil || !strings.Contains(err.Error(), "handoff input SHA-256 mismatch") {
		t.Fatalf("expected package digest mismatch, got %v", err)
	}
}

func TestUdonRunnerStagesPackageAndUsesConfiguredWorkdir(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	repoRoot := filepath.Clean(filepath.Join(wd, "..", ".."))
	tmp := t.TempDir()
	packageRoot := filepath.Join(tmp, "package")
	workdir := filepath.Join(tmp, "work")
	for _, dir := range []string{
		filepath.Join(packageRoot, "workflows"),
		filepath.Join(packageRoot, "openapi", "nested"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	mustWriteFile(t, filepath.Join(packageRoot, "workflows", "workflow.uws.yaml"), []byte("uws: 1.0.0\n"))
	mustWriteFile(t, filepath.Join(packageRoot, "openapi", "nested", "support.yaml"), []byte("openapi: 3.0.0\n"))
	configPath := filepath.Join(tmp, "run-config.json")
	config := RunConfig{
		Version:        RunConfigVersion,
		Scope:          "examples/test",
		PackageRoot:    packageRoot,
		WorkDir:        workdir,
		WorkflowPath:   "workflows/workflow.uws.yaml",
		WorkflowFormat: "uws-yaml",
		OpenAPIPaths:   []string{"openapi/nested/support.yaml"},
	}
	config = withRunnerPackageDigest(t, packageRoot, config)
	data, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, configPath, data)
	fakeExecutor := filepath.Join(tmp, "fake-udon")
	capture := filepath.Join(tmp, "args.txt")
	mustWriteFile(t, fakeExecutor, []byte(fmt.Sprintf("#!/bin/sh\nprintf '%%s\\n' \"$@\" > %q\n", capture)))
	if err := os.Chmod(fakeExecutor, 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := runnerCLICommand(t, repoRoot, configPath)
	cmd.Env = append(os.Environ(), "OPENUDON_EXECUTOR="+fakeExecutor)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("udon-runner failed: %v\n%s", err, out)
	}
	args, err := os.ReadFile(capture)
	if err != nil {
		t.Fatal(err)
	}
	stagedWorkdir := capturedArgValue(t, string(args), "--workdir")
	if stagedWorkdir == workdir || !strings.HasPrefix(stagedWorkdir, workdir+string(os.PathSeparator)+"stage.") {
		t.Fatalf("executor workdir = %q, want fresh stage under %q\nargs:\n%s", stagedWorkdir, workdir, args)
	}
	if gotWorkflow := capturedArgValue(t, string(args), "--workflow"); gotWorkflow != filepath.Join(stagedWorkdir, "workflows", "workflow.uws.yaml") {
		t.Fatalf("executor workflow = %q, want staged workflow under %q\nargs:\n%s", gotWorkflow, stagedWorkdir, args)
	}
	if strings.Contains(string(args), "--workdir\n"+packageRoot) {
		t.Fatalf("executor args did not use staged workdir:\n%s", args)
	}
	for _, path := range []string{
		filepath.Join(stagedWorkdir, "workflows", "workflow.uws.yaml"),
		filepath.Join(stagedWorkdir, "openapi", "nested", "support.yaml"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("staged path missing %s: %v", path, err)
		}
	}
	if _, err := os.Stat(filepath.Join(workdir, "workflows", "workflow.uws.yaml")); !os.IsNotExist(err) {
		t.Fatalf("workflow was staged in persistent workdir root, err=%v", err)
	}
}

func TestUdonRunnerRejectsSymlinkedPackageRoot(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	repoRoot := filepath.Clean(filepath.Join(wd, "..", ".."))
	tmp := t.TempDir()
	realPackageRoot := filepath.Join(tmp, "real-package")
	linkPackageRoot := filepath.Join(tmp, "package-link")
	workdir := filepath.Join(tmp, "work")
	mustWriteFile(t, filepath.Join(realPackageRoot, "workflows", "workflow.uws.yaml"), []byte("uws: 1.0.0\n"))
	if err := os.Symlink(realPackageRoot, linkPackageRoot); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(tmp, "run-config.json")
	data, err := json.Marshal(RunConfig{
		Version:        RunConfigVersion,
		PackageRoot:    linkPackageRoot,
		WorkDir:        workdir,
		WorkflowPath:   "workflows/workflow.uws.yaml",
		WorkflowFormat: "uws-yaml",
	})
	if err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, configPath, data)
	capture := filepath.Join(tmp, "args.txt")
	fakeExecutor := filepath.Join(tmp, "fake-udon")
	mustWriteFile(t, fakeExecutor, []byte(fmt.Sprintf("#!/bin/sh\nprintf '%%s\\n' \"$@\" > %q\n", capture)))
	if err := os.Chmod(fakeExecutor, 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := runnerCLICommand(t, repoRoot, configPath)
	cmd.Env = append(os.Environ(), "OPENUDON_EXECUTOR="+fakeExecutor)
	out, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(out), "package root must not be a symlink") {
		t.Fatalf("expected package root symlink rejection, err=%v out=%s", err, out)
	}
	if _, statErr := os.Stat(capture); !os.IsNotExist(statErr) {
		t.Fatalf("executor was invoked for symlinked package root, stat err=%v", statErr)
	}
}

func TestUdonRunnerRejectsSymlinkedWorkflow(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	repoRoot := filepath.Clean(filepath.Join(wd, "..", ".."))
	tmp := t.TempDir()
	packageRoot := filepath.Join(tmp, "package")
	workdir := filepath.Join(tmp, "work")
	if err := os.MkdirAll(filepath.Join(packageRoot, "workflows"), 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(tmp, "outside-workflow.yaml")
	mustWriteFile(t, target, []byte("uws: outside\n"))
	if err := os.Symlink(target, filepath.Join(packageRoot, "workflows", "workflow.uws.yaml")); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(tmp, "run-config.json")
	data, err := json.Marshal(RunConfig{
		Version:        RunConfigVersion,
		PackageRoot:    packageRoot,
		WorkDir:        workdir,
		WorkflowPath:   "workflows/workflow.uws.yaml",
		WorkflowFormat: "uws-yaml",
	})
	if err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, configPath, data)
	cmd := runnerCLICommand(t, repoRoot, configPath)
	cmd.Env = append(os.Environ(), "OPENUDON_EXECUTOR=/bin/true")
	out, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(out), "workflow file must not be a symlink") {
		t.Fatalf("expected workflow symlink rejection, err=%v out=%s", err, out)
	}
}

func TestUdonRunnerRejectsSymlinkedWorkflowParent(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	repoRoot := filepath.Clean(filepath.Join(wd, "..", ".."))
	tmp := t.TempDir()
	packageRoot := filepath.Join(tmp, "package")
	workdir := filepath.Join(tmp, "work")
	realWorkflows := filepath.Join(tmp, "real-workflows")
	if err := os.MkdirAll(realWorkflows, 0o755); err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(realWorkflows, "workflow.uws.yaml"), []byte("uws: outside\n"))
	if err := os.MkdirAll(packageRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(realWorkflows, filepath.Join(packageRoot, "workflows")); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(tmp, "run-config.json")
	data, err := json.Marshal(RunConfig{
		Version:        RunConfigVersion,
		PackageRoot:    packageRoot,
		WorkDir:        workdir,
		WorkflowPath:   "workflows/workflow.uws.yaml",
		WorkflowFormat: "uws-yaml",
	})
	if err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, configPath, data)
	cmd := runnerCLICommand(t, repoRoot, configPath)
	cmd.Env = append(os.Environ(), "OPENUDON_EXECUTOR=/bin/true")
	out, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(out), "workflow file must not be a symlink") {
		t.Fatalf("expected workflow parent symlink rejection, err=%v out=%s", err, out)
	}
}

func TestUdonRunnerRejectsUnsafeWorkflowTypes(t *testing.T) {
	for _, tc := range []struct {
		name       string
		setup      func(t *testing.T, path string)
		wantOutput string
	}{
		{
			name: "directory",
			setup: func(t *testing.T, path string) {
				t.Helper()
				if err := os.MkdirAll(path, 0o755); err != nil {
					t.Fatal(err)
				}
			},
			wantOutput: "workflow file must be a regular file",
		},
		{
			name: "non-regular",
			setup: func(t *testing.T, path string) {
				t.Helper()
				if runtime.GOOS == "windows" {
					t.Skip("fifo test requires Unix")
				}
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := syscall.Mkfifo(path, 0o600); err != nil {
					t.Skipf("fifo unavailable: %v", err)
				}
			},
			wantOutput: "workflow file must be a regular file",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			wd, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			repoRoot := filepath.Clean(filepath.Join(wd, "..", ".."))
			tmp := t.TempDir()
			packageRoot := filepath.Join(tmp, "package")
			workdir := filepath.Join(tmp, "work")
			workflowPath := filepath.Join(packageRoot, "workflows", "workflow.uws.yaml")
			tc.setup(t, workflowPath)
			configPath := filepath.Join(tmp, "run-config.json")
			data, err := json.Marshal(RunConfig{
				Version:        RunConfigVersion,
				PackageRoot:    packageRoot,
				WorkDir:        workdir,
				WorkflowPath:   "workflows/workflow.uws.yaml",
				WorkflowFormat: "uws-yaml",
			})
			if err != nil {
				t.Fatal(err)
			}
			mustWriteFile(t, configPath, data)
			cmd := runnerCLICommand(t, repoRoot, configPath)
			cmd.Env = append(os.Environ(), "OPENUDON_EXECUTOR=/bin/true")
			out, err := cmd.CombinedOutput()
			if err == nil || !strings.Contains(string(out), tc.wantOutput) {
				t.Fatalf("expected %q, err=%v out=%s", tc.wantOutput, err, out)
			}
		})
	}
}

func TestUdonRunnerRejectsUnsafeWorkflowPathFields(t *testing.T) {
	for _, tc := range []struct {
		name       string
		path       string
		wantOutput string
	}{
		{
			name:       "control-character",
			path:       "workflows/workflow.uws.yaml\n2",
			wantOutput: "control characters",
		},
		{
			name:       "absolute-outside-package",
			path:       "",
			wantOutput: "workflow_path escapes package_root",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			wd, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			repoRoot := filepath.Clean(filepath.Join(wd, "..", ".."))
			tmp := t.TempDir()
			packageRoot := filepath.Join(tmp, "package")
			workdir := filepath.Join(tmp, "work")
			mustWriteFile(t, filepath.Join(packageRoot, "workflows", "workflow.uws.yaml"), []byte("uws: 1.0.0\n"))
			workflowPath := tc.path
			if tc.name == "absolute-outside-package" {
				outside := filepath.Join(tmp, "outside-workflow.yaml")
				mustWriteFile(t, outside, []byte("uws: outside\n"))
				workflowPath = outside
			}
			configPath := filepath.Join(tmp, "run-config.json")
			data, err := json.Marshal(RunConfig{
				Version:        RunConfigVersion,
				PackageRoot:    packageRoot,
				WorkDir:        workdir,
				WorkflowPath:   workflowPath,
				WorkflowFormat: "uws-yaml",
			})
			if err != nil {
				t.Fatal(err)
			}
			mustWriteFile(t, configPath, data)
			cmd := runnerCLICommand(t, repoRoot, configPath)
			cmd.Env = append(os.Environ(), "OPENUDON_EXECUTOR=/bin/true")
			out, err := cmd.CombinedOutput()
			if err == nil || !strings.Contains(string(out), tc.wantOutput) {
				t.Fatalf("expected %q, err=%v out=%s", tc.wantOutput, err, out)
			}
		})
	}
}

func TestUdonRunnerRejectsOpenAPIUnsafePaths(t *testing.T) {
	for _, tc := range []struct {
		name       string
		path       string
		setup      func(t *testing.T, packageRoot string)
		wantOutput string
	}{
		{
			name: "control-character",
			path: "openapi/support.yaml\n2",
			setup: func(t *testing.T, packageRoot string) {
				t.Helper()
				mustWriteFile(t, filepath.Join(packageRoot, "openapi", "support.yaml"), []byte("openapi: 3.0.0\n"))
			},
			wantOutput: "control characters",
		},
		{
			name:       "absolute-outside-package",
			path:       "",
			setup:      func(t *testing.T, packageRoot string) {},
			wantOutput: "api source path escapes package_root",
		},
		{
			name: "symlink",
			path: "openapi/support.yaml",
			setup: func(t *testing.T, packageRoot string) {
				t.Helper()
				target := filepath.Join(filepath.Dir(packageRoot), "outside.yaml")
				mustWriteFile(t, target, []byte("openapi: 3.0.0\n"))
				if err := os.MkdirAll(filepath.Join(packageRoot, "openapi"), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target, filepath.Join(packageRoot, "openapi", "support.yaml")); err != nil {
					t.Fatal(err)
				}
			},
			wantOutput: "api source file must not be a symlink",
		},
		{
			name: "symlinked-parent",
			path: "openapi/support.yaml",
			setup: func(t *testing.T, packageRoot string) {
				t.Helper()
				realOpenAPI := filepath.Join(filepath.Dir(packageRoot), "real-openapi")
				if err := os.MkdirAll(realOpenAPI, 0o755); err != nil {
					t.Fatal(err)
				}
				mustWriteFile(t, filepath.Join(realOpenAPI, "support.yaml"), []byte("openapi: 3.0.0\n"))
				if err := os.Symlink(realOpenAPI, filepath.Join(packageRoot, "openapi")); err != nil {
					t.Fatal(err)
				}
			},
			wantOutput: "api source file must not be a symlink",
		},
		{
			name: "directory",
			path: "openapi/support.yaml",
			setup: func(t *testing.T, packageRoot string) {
				t.Helper()
				if err := os.MkdirAll(filepath.Join(packageRoot, "openapi", "support.yaml"), 0o755); err != nil {
					t.Fatal(err)
				}
			},
			wantOutput: "api source file must be a regular file",
		},
		{
			name: "non-regular",
			path: "openapi/support.yaml",
			setup: func(t *testing.T, packageRoot string) {
				t.Helper()
				if runtime.GOOS == "windows" {
					t.Skip("fifo test requires Unix")
				}
				path := filepath.Join(packageRoot, "openapi", "support.yaml")
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := syscall.Mkfifo(path, 0o600); err != nil {
					t.Skipf("fifo unavailable: %v", err)
				}
			},
			wantOutput: "api source file must be a regular file",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			wd, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			repoRoot := filepath.Clean(filepath.Join(wd, "..", ".."))
			tmp := t.TempDir()
			packageRoot := filepath.Join(tmp, "package")
			workdir := filepath.Join(tmp, "work")
			mustWriteFile(t, filepath.Join(packageRoot, "workflows", "workflow.uws.yaml"), []byte("uws: 1.0.0\n"))
			tc.setup(t, packageRoot)
			openAPIPath := tc.path
			if tc.name == "absolute-outside-package" {
				outside := filepath.Join(tmp, "outside.yaml")
				mustWriteFile(t, outside, []byte("openapi: 3.0.0\n"))
				openAPIPath = outside
			}
			configPath := filepath.Join(tmp, "run-config.json")
			data, err := json.Marshal(RunConfig{
				Version:        RunConfigVersion,
				PackageRoot:    packageRoot,
				WorkDir:        workdir,
				WorkflowPath:   "workflows/workflow.uws.yaml",
				WorkflowFormat: "uws-yaml",
				OpenAPIPaths:   []string{openAPIPath},
			})
			if err != nil {
				t.Fatal(err)
			}
			mustWriteFile(t, configPath, data)
			cmd := runnerCLICommand(t, repoRoot, configPath)
			cmd.Env = append(os.Environ(), "OPENUDON_EXECUTOR=/bin/true")
			out, err := cmd.CombinedOutput()
			if err == nil || !strings.Contains(string(out), tc.wantOutput) {
				t.Fatalf("expected %q, err=%v out=%s", tc.wantOutput, err, out)
			}
		})
	}
}

func TestUdonRunnerAcceptsAbsolutePathsInsidePackageRoot(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	repoRoot := filepath.Clean(filepath.Join(wd, "..", ".."))
	tmp := t.TempDir()
	packageRoot := filepath.Join(tmp, "package")
	workdir := filepath.Join(tmp, "work")
	workflowPath := filepath.Join(packageRoot, "workflows", "workflow.uws.yaml")
	openAPIPath := filepath.Join(packageRoot, "openapi", "nested", "support.yaml")
	mustWriteFile(t, workflowPath, []byte("uws: 1.0.0\n"))
	mustWriteFile(t, openAPIPath, []byte("openapi: 3.0.0\n"))
	configPath := filepath.Join(tmp, "run-config.json")
	config := RunConfig{
		Version:        RunConfigVersion,
		Scope:          "examples/test",
		PackageRoot:    packageRoot,
		WorkDir:        workdir,
		WorkflowPath:   workflowPath,
		WorkflowFormat: "uws-yaml",
		OpenAPIPaths:   []string{openAPIPath},
	}
	config = withRunnerPackageDigest(t, packageRoot, config)
	data, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, configPath, data)
	fakeExecutor := filepath.Join(tmp, "fake-udon")
	capture := filepath.Join(tmp, "args.txt")
	mustWriteFile(t, fakeExecutor, []byte(fmt.Sprintf("#!/bin/sh\nprintf '%%s\\n' \"$@\" > %q\n", capture)))
	if err := os.Chmod(fakeExecutor, 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := runnerCLICommand(t, repoRoot, configPath)
	cmd.Env = append(os.Environ(), "OPENUDON_EXECUTOR="+fakeExecutor)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("udon-runner failed: %v\n%s", err, out)
	}
	args, err := os.ReadFile(capture)
	if err != nil {
		t.Fatal(err)
	}
	stagedWorkdir := capturedArgValue(t, string(args), "--workdir")
	for _, path := range []string{
		filepath.Join(stagedWorkdir, "workflows", "workflow.uws.yaml"),
		filepath.Join(stagedWorkdir, "openapi", "nested", "support.yaml"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("staged path missing %s: %v", path, err)
		}
	}
}

func TestUdonRunnerFreshStageHidesPersistentStaleFiles(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	repoRoot := filepath.Clean(filepath.Join(wd, "..", ".."))
	tmp := t.TempDir()
	packageRoot := filepath.Join(tmp, "package")
	workdir := filepath.Join(tmp, "work")
	if err := os.MkdirAll(filepath.Join(packageRoot, "workflows"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(workdir, "openapi"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(packageRoot, "workflows", "workflow.uws.yaml"), []byte("uws: 1.0.0\n"))
	mustWriteFile(t, filepath.Join(workdir, "openapi", "stale.yaml"), []byte("openapi: 3.0.0\n"))
	configPath := filepath.Join(tmp, "run-config.json")
	config := RunConfig{
		Version:        RunConfigVersion,
		Scope:          "examples/test",
		PackageRoot:    packageRoot,
		WorkDir:        workdir,
		WorkflowPath:   "workflows/workflow.uws.yaml",
		WorkflowFormat: "uws-yaml",
	}
	config = withRunnerPackageDigest(t, packageRoot, config)
	data, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, configPath, data)
	fakeExecutor := filepath.Join(tmp, "fake-udon")
	capture := filepath.Join(tmp, "args.txt")
	mustWriteFile(t, fakeExecutor, []byte(fmt.Sprintf("#!/bin/sh\nprintf '%%s\\n' \"$@\" > %q\n", capture)))
	if err := os.Chmod(fakeExecutor, 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := runnerCLICommand(t, repoRoot, configPath)
	cmd.Env = append(os.Environ(), "OPENUDON_EXECUTOR="+fakeExecutor)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("udon-runner failed: %v\n%s", err, out)
	}
	args, err := os.ReadFile(capture)
	if err != nil {
		t.Fatal(err)
	}
	stagedWorkdir := capturedArgValue(t, string(args), "--workdir")
	if _, err := os.Stat(filepath.Join(stagedWorkdir, "openapi", "stale.yaml")); !os.IsNotExist(err) {
		t.Fatalf("stale OpenAPI file visible in staged workdir, err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(workdir, "openapi", "stale.yaml")); err != nil {
		t.Fatalf("persistent stale file should not be deleted: %v", err)
	}
}

func TestUdonRunnerCanInvokeDockerImage(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	repoRoot := filepath.Clean(filepath.Join(wd, "..", ".."))
	tmp := t.TempDir()
	packageRoot := filepath.Join(tmp, "package")
	workdir := filepath.Join(tmp, "work")
	if err := os.MkdirAll(filepath.Join(packageRoot, "workflows"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(packageRoot, "workflows", "workflow.uws.yaml"), []byte("uws: 1.0.0\n"))
	configPath := filepath.Join(tmp, "run-config.json")
	config := RunConfig{
		Version:            RunConfigVersion,
		Scope:              "examples/test",
		PackageRoot:        packageRoot,
		WorkDir:            workdir,
		WorkflowPath:       "workflows/workflow.uws.yaml",
		WorkflowFormat:     "uws-yaml",
		CredentialBindings: []string{"support-api.token"},
	}
	config = withRunnerPackageDigest(t, packageRoot, config)
	data, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, configPath, data)
	binDir := filepath.Join(tmp, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	capture := filepath.Join(tmp, "docker-args.txt")
	fakeDocker := filepath.Join(binDir, "docker")
	mustWriteFile(t, fakeDocker, []byte(fmt.Sprintf("#!/usr/bin/env bash\nprintf '%%s\\n' \"$@\" > %q\n", capture)))
	if err := os.Chmod(fakeDocker, 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := runnerCLICommand(t, repoRoot, configPath)
	cmd.Env = append(os.Environ(),
		"PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"),
		"OPENUDON_EXECUTOR=docker://udon:test",
		"UDON_CREDENTIAL_SUPPORT_API_TOKEN=super-secret",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("udon-runner docker failed: %v\n%s", err, out)
	}
	args, err := os.ReadFile(capture)
	if err != nil {
		t.Fatal(err)
	}
	mountPath := capturedArgValue(t, string(args), "-v")
	if !strings.HasSuffix(mountPath, ":/workspace") || !strings.HasPrefix(mountPath, workdir+string(os.PathSeparator)+"stage.") {
		t.Fatalf("docker mount = %q, want fresh stage under %q\nargs:\n%s", mountPath, workdir, args)
	}
	for _, want := range []string{"run\n", "--rm\n", "-e\nUDON_CREDENTIAL_SUPPORT_API_TOKEN\n", "udon:test\n", "--workdir\n/workspace\n", "--workflow\n/workspace/workflows/workflow.uws.yaml\n"} {
		if !strings.Contains(string(args), want) {
			t.Fatalf("docker args missing %q:\n%s", want, args)
		}
	}
	if strings.Contains(string(args), "super-secret") {
		t.Fatalf("docker args leaked credential value:\n%s", args)
	}
}

func TestUdonRunnerFailsWhenCredentialEnvMissing(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	repoRoot := filepath.Clean(filepath.Join(wd, "..", ".."))
	tmp := t.TempDir()
	packageRoot := filepath.Join(tmp, "package")
	workdir := filepath.Join(tmp, "work")
	if err := os.MkdirAll(filepath.Join(packageRoot, "workflows"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(packageRoot, "workflows", "workflow.uws.yaml"), []byte("uws: 1.0.0\n"))
	configPath := filepath.Join(tmp, "run-config.json")
	config := RunConfig{
		Version:            RunConfigVersion,
		Scope:              "examples/test",
		PackageRoot:        packageRoot,
		WorkDir:            workdir,
		WorkflowPath:       "workflows/workflow.uws.yaml",
		WorkflowFormat:     "uws-yaml",
		CredentialBindings: []string{"missing.plan.test"},
	}
	config = withRunnerPackageDigest(t, packageRoot, config)
	data, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, configPath, data)
	cmd := runnerCLICommand(t, repoRoot, configPath)
	cmd.Env = nil
	for _, item := range os.Environ() {
		if strings.HasPrefix(item, "UDON_CREDENTIAL_MISSING_PLAN_TEST=") {
			continue
		}
		cmd.Env = append(cmd.Env, item)
	}
	cmd.Env = append(cmd.Env, "OPENUDON_EXECUTOR=/bin/true")
	out, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(out), "UDON_CREDENTIAL_MISSING_PLAN_TEST") {
		t.Fatalf("expected missing credential env failure, err=%v out=%s", err, out)
	}
}

func TestUdonRunnerRejectsRelativeExecutorEnv(t *testing.T) {
	for _, tc := range []struct {
		name string
		env  string
	}{
		{name: "openudon-executor", env: "OPENUDON_EXECUTOR=relative-udon"},
		{name: "openudon-udon-bin", env: "OPENUDON_UDON_BIN=relative-udon"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			wd, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			repoRoot := filepath.Clean(filepath.Join(wd, "..", ".."))
			tmp := t.TempDir()
			packageRoot := filepath.Join(tmp, "package")
			workdir := filepath.Join(tmp, "work")
			mustWriteFile(t, filepath.Join(packageRoot, "workflows", "workflow.uws.yaml"), []byte("uws: 1.0.0\n"))
			configPath := filepath.Join(tmp, "run-config.json")
			config := RunConfig{
				Version:        RunConfigVersion,
				Scope:          "examples/test",
				PackageRoot:    packageRoot,
				WorkDir:        workdir,
				WorkflowPath:   "workflows/workflow.uws.yaml",
				WorkflowFormat: "uws-yaml",
			}
			config = withRunnerPackageDigest(t, packageRoot, config)
			data, err := json.Marshal(config)
			if err != nil {
				t.Fatal(err)
			}
			mustWriteFile(t, configPath, data)
			cmd := runnerCLICommand(t, repoRoot, configPath)
			cmd.Env = append(filteredExecutorEnv(os.Environ()), tc.env)
			out, err := cmd.CombinedOutput()
			if err == nil || !strings.Contains(string(out), "must be an absolute path") {
				t.Fatalf("expected absolute-path rejection, err=%v out=%s", err, out)
			}
		})
	}
}

func filteredExecutorEnv(env []string) []string {
	var out []string
	for _, item := range env {
		if strings.HasPrefix(item, "OPENUDON_EXECUTOR=") ||
			strings.HasPrefix(item, "OPENUDON_UDON_BIN=") ||
			strings.HasPrefix(item, "OPENUDON_UDON_IMAGE=") {
			continue
		}
		out = append(out, item)
	}
	return out
}

func TestUdonRunnerVerifiesStagedPackageDigestBeforeExecutor(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	repoRoot := filepath.Clean(filepath.Join(wd, "..", ".."))
	tmp := t.TempDir()
	packageRoot := filepath.Join(tmp, "package")
	workdir := filepath.Join(tmp, "work")
	workflowRel := "workflows/workflow.uws.yaml"
	workflowPath := filepath.Join(packageRoot, filepath.FromSlash(workflowRel))
	mustWriteFile(t, workflowPath, []byte("uws: 1.0.0\n"))
	digest, err := authoring.ComputeReviewHandoffDigest(authoring.ReviewHandoffDigestOptions{
		Root:    packageRoot,
		Version: "openudon.handoff-package-digest.v1",
		Inputs:  []authoring.ReviewHandoffInput{{Path: workflowRel, Required: true}},
	})
	if err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, workflowPath, []byte("uws: changed\n"))
	configPath := filepath.Join(tmp, "run-config.json")
	data, err := json.Marshal(RunConfig{
		Version:        RunConfigVersion,
		PackageRoot:    packageRoot,
		WorkDir:        workdir,
		WorkflowPath:   workflowRel,
		WorkflowFormat: "uws-yaml",
		PackagePaths:   []string{workflowRel},
		PackageSHA256:  digest,
	})
	if err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, configPath, data)
	capture := filepath.Join(tmp, "args.txt")
	fakeExecutor := filepath.Join(tmp, "fake-udon")
	mustWriteFile(t, fakeExecutor, []byte(fmt.Sprintf("#!/bin/sh\nprintf '%%s\\n' \"$@\" > %q\n", capture)))
	if err := os.Chmod(fakeExecutor, 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := runnerCLICommand(t, repoRoot, configPath)
	cmd.Env = append(os.Environ(), "OPENUDON_EXECUTOR="+fakeExecutor)
	out, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(out), "staged package_sha256") {
		t.Fatalf("expected staged digest mismatch, err=%v out=%s", err, out)
	}
	if _, statErr := os.Stat(capture); !os.IsNotExist(statErr) {
		t.Fatalf("executor was invoked despite staged digest mismatch, stat err=%v", statErr)
	}
}

func capturedArgValue(t *testing.T, args, flag string) string {
	t.Helper()
	lines := strings.Split(strings.TrimRight(args, "\n"), "\n")
	for i := 0; i < len(lines)-1; i++ {
		if lines[i] == flag {
			return lines[i+1]
		}
	}
	t.Fatalf("args missing %s:\n%s", flag, args)
	return ""
}

func withRunnerPackageDigest(t *testing.T, packageRoot string, config RunConfig) RunConfig {
	t.Helper()
	paths := []string{runnerPackageRel(t, packageRoot, config.WorkflowPath)}
	for _, path := range append(append([]string(nil), config.APISourcePaths...), config.OpenAPIPaths...) {
		paths = append(paths, runnerPackageRel(t, packageRoot, path))
	}
	config.PackagePaths = paths
	inputs := make([]authoring.ReviewHandoffInput, 0, len(paths))
	for _, path := range paths {
		inputs = append(inputs, authoring.ReviewHandoffInput{Path: path, Required: true})
	}
	digest, err := authoring.ComputeReviewHandoffDigest(authoring.ReviewHandoffDigestOptions{
		Root:    packageRoot,
		Scope:   config.Scope,
		Version: "openudon.handoff-package-digest.v1",
		Inputs:  inputs,
	})
	if err != nil {
		t.Fatal(err)
	}
	config.PackageSHA256 = digest
	return config
}

func runnerPackageRel(t *testing.T, packageRoot, path string) string {
	t.Helper()
	if filepath.IsAbs(path) {
		rel, err := filepath.Rel(packageRoot, path)
		if err != nil {
			t.Fatal(err)
		}
		return filepath.ToSlash(rel)
	}
	return filepath.ToSlash(filepath.Clean(path))
}

func argValue(t *testing.T, args []string, flag string) string {
	t.Helper()
	for i := 0; i < len(args)-1; i++ {
		if args[i] == flag {
			return args[i+1]
		}
	}
	t.Fatalf("args missing %s: %#v", flag, args)
	return ""
}

func runnerCLICommand(t *testing.T, repoRoot, configPath string) *exec.Cmd {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestRunnerHelperProcess$", "--", configPath)
	cmd.Dir = repoRoot
	return cmd
}

// TestRunnerHelperProcess keeps low-level staging/path tests below the external
// trusted boundary. cmd/udon-runner itself is covered separately with a fully
// validated package, config digest, and approval.
func TestRunnerHelperProcess(t *testing.T) {
	separator := -1
	for i, arg := range os.Args {
		if arg == "--" {
			separator = i
			break
		}
	}
	if separator < 0 || separator+1 >= len(os.Args) {
		return
	}
	config, err := udonrunner.LoadConfig(os.Args[separator+1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if config.RunID == "" {
		config.RunID = "0123456789abcdef0123456789abcdef"
	}
	if config.HandoffSHA256 == "" {
		config.HandoffSHA256 = strings.Repeat("a", 64)
	}
	if config.ApprovalSHA256 == "" {
		config.ApprovalSHA256 = strings.Repeat("b", 64)
	}
	wd, wdErr := os.Getwd()
	if wdErr != nil {
		fmt.Fprintln(os.Stderr, wdErr)
		os.Exit(1)
	}
	_, err = udonrunner.Run(context.Background(), config, udonrunner.Options{
		RepoRoot: wd, Env: os.Environ(), Stdout: os.Stdout, Stderr: os.Stderr,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

type fixtureOptions struct {
	qualityStatus       string
	malformedHandoff    bool
	valuesAllowed       bool
	directProduction    bool
	extraRequiredInputs []string
	credentialBindings  []string
}

func writeFixture(t *testing.T, opts fixtureOptions) (string, string) {
	t.Helper()
	root := t.TempDir()
	example := filepath.Join(root, "examples", "support-email")
	for _, dir := range []string{
		filepath.Join(example, "workflows"),
		filepath.Join(example, "expected"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	status := opts.qualityStatus
	if status == "" {
		status = "pass"
	}
	files := map[string][]byte{
		"project.md":                  []byte("# Project\n"),
		"workflows/intent.hcl":        []byte("intent {}\n"),
		"workflows/workflow.hcl":      []byte("workflow {}\n"),
		"workflows/workflow.uws.yaml": []byte("version: 1.0.0\n"),
		"expected/plan.json":          []byte("{}\n"),
		"expected/quality.json":       []byte(`{"status":"` + status + `"}` + "\n"),
		"expected/refinement.json":    []byte("{}\n"),
		"expected/review.md":          []byte("# Review\n"),
	}
	for rel, data := range files {
		mustWriteFile(t, filepath.Join(example, filepath.FromSlash(rel)), data)
	}
	if opts.malformedHandoff {
		mustWriteFile(t, filepath.Join(example, "expected", "review-handoff.json"), []byte("{"))
		return root, example
	}
	paths := []string{
		"project.md", "workflows/intent.hcl", "workflows/workflow.hcl", "workflows/workflow.uws.yaml",
		"expected/plan.json", "expected/quality.json", "expected/refinement.json", "expected/review.md", "expected/review-handoff.json",
	}
	for _, path := range opts.extraRequiredInputs {
		paths = append(paths, path)
	}
	inputs := make([]authoring.ReviewHandoffInput, 0, len(paths))
	for _, path := range paths {
		inputs = append(inputs, authoring.ReviewHandoffInput{Path: path, Purpose: "test package input", Required: true, SHA256: strings.Repeat("0", 64)})
	}
	manifest := authoring.NewReviewHandoff(authoring.ReviewHandoffOptions{
		Version: ReviewHandoffVersion, GeneratedState: string(authoring.ReviewStateGenerated), HandoffInputs: inputs,
		ApprovalStates: authoring.DefaultReviewStateMachine(),
		OwnerSplit: authoring.ReviewOwnerSplit{
			"openudon": {"artifact validation"}, "external_review_orchestration": {"approval routing"},
		},
		ExecutionPolicy: authoring.ReviewExecutionPolicy{DirectProductionExecution: opts.directProduction},
		CredentialBindings: authoring.ReviewCredentialBindings{
			Declared: append([]string(nil), opts.credentialBindings...), ExpectedFromPlan: append([]string(nil), opts.credentialBindings...),
			ValuesAllowedInArtifacts: opts.valuesAllowed,
		},
	})
	refreshFixtureHandoffDigests(t, example, &manifest)
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(example, "expected", "review-handoff.json"), append(data, '\n'))
	return root, example
}

func writeApprovalTemplate(t *testing.T, root, example, state string, now func() time.Time) string {
	t.Helper()
	refreshFixtureHandoffFile(t, example)
	approval, err := ApprovalTemplate(context.Background(), TemplateOptions{
		RepoRoot:   root,
		ExampleDir: example,
		State:      state,
		Reviewer:   "Ada",
		Now:        now,
		Assess:     passAssess,
	})
	if err != nil {
		t.Fatalf("ApprovalTemplate returned error: %v", err)
	}
	path := filepath.Join(root, "approval.json")
	writeApprovalFile(t, path, approval)
	return path
}

func writeApprovalTemplateWithoutPolicyCheck(t *testing.T, root, example, state string, now func() time.Time) string {
	t.Helper()
	refreshFixtureHandoffFile(t, example)
	data, err := os.ReadFile(filepath.Join(example, "expected", "review-handoff.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest handoffManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	p, err := resolvePaths(root, example)
	if err != nil {
		t.Fatal(err)
	}
	digest, err := computePackageDigest(p, manifest)
	if err != nil {
		t.Fatal(err)
	}
	approval := Approval{
		Version:       ApprovalVersion,
		Scope:         "examples/support-email",
		State:         state,
		Reviewer:      "Ada",
		ApprovedAt:    now().UTC().Format(time.RFC3339),
		PackageSHA256: digest,
	}
	path := filepath.Join(root, "approval.json")
	writeApprovalFile(t, path, approval)
	return path
}

func refreshFixtureHandoffFile(t *testing.T, example string) {
	t.Helper()
	path := filepath.Join(example, "expected", "review-handoff.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var manifest authoring.ReviewHandoff
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	refreshFixtureHandoffDigests(t, example, &manifest)
	data, err = json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, path, append(data, '\n'))
}

func refreshFixtureHandoffDigests(t *testing.T, example string, manifest *authoring.ReviewHandoff) {
	t.Helper()
	const self = "expected/review-handoff.json"
	for i := range manifest.HandoffInputs {
		input := &manifest.HandoffInputs[i]
		if input.Path == self {
			input.SHA256 = strings.Repeat("0", 64)
			continue
		}
		data, err := os.ReadFile(filepath.Join(example, filepath.FromSlash(input.Path)))
		if err != nil {
			input.SHA256 = strings.Repeat("0", 64)
			continue
		}
		digest := sha256.Sum256(data)
		input.SHA256 = fmt.Sprintf("%x", digest[:])
	}
	digest, err := authoring.ReviewHandoffSelfDigest(*manifest, self)
	if err != nil {
		t.Fatal(err)
	}
	for i := range manifest.HandoffInputs {
		if manifest.HandoffInputs[i].Path == self {
			manifest.HandoffInputs[i].SHA256 = digest
		}
	}
}

func passAssess(context.Context, synthesize.Options) (*synthesize.QualityReport, error) {
	return &synthesize.QualityReport{Status: "pass"}, nil
}

func fixedNow() func() time.Time {
	return func() time.Time {
		return time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)
	}
}

func readApprovalFile(t *testing.T, path string) Approval {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var approval Approval
	if err := json.Unmarshal(data, &approval); err != nil {
		t.Fatal(err)
	}
	return approval
}

func readRunEvidenceFile(t *testing.T, path string) RunEvidence {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var evidence RunEvidence
	if err := json.Unmarshal(data, &evidence); err != nil {
		t.Fatal(err)
	}
	return evidence
}

func writeVerifiableRunEvidence(t *testing.T) *RunResult {
	t.Helper()
	root, example := writeFixture(t, fixtureOptions{})
	now := fixedNow()
	approvalPath := writeApprovalTemplate(t, root, example, StateApprovedForSandbox, now)
	result, err := Run(context.Background(), Options{
		RepoRoot:     root,
		ExampleDir:   example,
		Tier:         TierSandbox,
		ApprovalPath: approvalPath,
		DryRun:       true,
		WorkDir:      filepath.Join(root, "work"),
		Now:          now,
		Assess:       passAssess,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	return result
}

func readRunEvidenceAndBundle(t *testing.T, result *RunResult) (RunEvidence, AsyncEvidenceBundle) {
	t.Helper()
	evidence := readRunEvidenceFile(t, result.RunEvidencePath)
	_, bundle := readReferencedAsyncEvidence(t, result.WorkDir, evidence)
	return evidence, bundle
}

func writeBundleAndRefreshRef(t *testing.T, result *RunResult, evidence RunEvidence, bundle AsyncEvidenceBundle) {
	t.Helper()
	data, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	evidence.AsyncEvidenceFiles[0].Records = len(bundle.Records)
	writeAsyncBytesAndRefreshRef(t, result, evidence, append(data, '\n'))
}

func writeRawBundleAndRefreshRef(t *testing.T, result *RunResult, evidence RunEvidence, raw map[string]any) {
	t.Helper()
	data, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	writeAsyncBytesAndRefreshRef(t, result, evidence, append(data, '\n'))
}

func writeAsyncBytesAndRefreshRef(t *testing.T, result *RunResult, evidence RunEvidence, data []byte) {
	t.Helper()
	mustWriteFile(t, result.AsyncEvidencePath, data)
	evidence.AsyncEvidenceFiles[0].Digest = evdigest.SHA256Bytes(data).String()
	writeRunEvidenceFileForTest(t, result.RunEvidencePath, evidence)
}

func writeRunEvidenceFileForTest(t *testing.T, path string, evidence RunEvidence) {
	t.Helper()
	data, err := json.MarshalIndent(evidence, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, path, append(data, '\n'))
}

func readReferencedAsyncEvidence(t *testing.T, workdir string, evidence RunEvidence) (RunEvidenceAsyncFile, AsyncEvidenceBundle) {
	t.Helper()
	if len(evidence.AsyncEvidenceFiles) != 1 {
		t.Fatalf("async evidence refs = %#v, want one", evidence.AsyncEvidenceFiles)
	}
	ref := evidence.AsyncEvidenceFiles[0]
	if ref.Purpose != "openudon_run_async_execution_forwarding" {
		t.Fatalf("unexpected async evidence purpose: %#v", ref)
	}
	data, err := os.ReadFile(filepath.Join(workdir, ref.Path))
	if err != nil {
		t.Fatal(err)
	}
	if got := evdigest.SHA256Bytes(data).String(); got != ref.Digest {
		t.Fatalf("async evidence digest = %q, want %q", ref.Digest, got)
	}
	var bundle AsyncEvidenceBundle
	if err := json.Unmarshal(data, &bundle); err != nil {
		t.Fatal(err)
	}
	if bundle.Version != AsyncEvidenceVersion {
		t.Fatalf("async evidence bundle version = %q", bundle.Version)
	}
	if ref.Records != len(bundle.Records) {
		t.Fatalf("async evidence ref records = %d, bundle has %d", ref.Records, len(bundle.Records))
	}
	return ref, bundle
}

func asyncExecutionRequest(t *testing.T, bundle AsyncEvidenceBundle) asyncevidence.ExecutionRequest {
	t.Helper()
	for _, record := range bundle.Records {
		if record.Kind == "execution_request" && record.ExecutionRequest != nil {
			return *record.ExecutionRequest
		}
	}
	t.Fatalf("missing execution_request record: %#v", bundle)
	return asyncevidence.ExecutionRequest{}
}

func asyncExecutionResponse(t *testing.T, bundle AsyncEvidenceBundle) asyncevidence.ExecutionResponse {
	t.Helper()
	for _, record := range bundle.Records {
		if record.Kind == "execution_response" && record.ExecutionResponse != nil {
			return *record.ExecutionResponse
		}
	}
	t.Fatalf("missing execution_response record: %#v", bundle)
	return asyncevidence.ExecutionResponse{}
}

func asyncStatusObservation(t *testing.T, bundle AsyncEvidenceBundle) asyncevidence.StatusObservation {
	t.Helper()
	for _, record := range bundle.Records {
		if record.Kind == "status_observation" && record.StatusObservation != nil {
			return *record.StatusObservation
		}
	}
	t.Fatalf("missing status_observation record: %#v", bundle)
	return asyncevidence.StatusObservation{}
}

func asyncConfirmationReadObservation(t *testing.T, bundle AsyncEvidenceBundle) asyncevidence.ConfirmationReadObservation {
	t.Helper()
	for _, record := range bundle.Records {
		if record.Kind == "confirmation_read_observation" && record.ConfirmationReadObservation != nil {
			return *record.ConfirmationReadObservation
		}
	}
	t.Fatalf("missing confirmation_read_observation record: %#v", bundle)
	return asyncevidence.ConfirmationReadObservation{}
}

func hasConfirmationReadObservation(bundle AsyncEvidenceBundle) bool {
	for _, record := range bundle.Records {
		if record.Kind == "confirmation_read_observation" && record.ConfirmationReadObservation != nil {
			return true
		}
	}
	return false
}

func writeUdonExecutionReport(t *testing.T, path, status string, now time.Time, outputDigest string) {
	t.Helper()
	if path == "" {
		t.Fatal("missing --execution-report argument")
	}
	report := UdonExecutionReport{
		Version:        UdonExecutionReportVersion,
		Status:         status,
		StartedAt:      now.UTC().Format(time.RFC3339),
		FinishedAt:     now.UTC().Format(time.RFC3339),
		WorkflowPath:   "workflows/workflow.uws.yaml",
		WorkflowFormat: "uws-yaml",
		WorkDir:        filepath.Dir(path),
		OutputPath:     filepath.Join(filepath.Dir(path), "output", "udon.hcl"),
		OutputDigest:   outputDigest,
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestDecodeUdonExecutionReportEnforcesPublishedContract(t *testing.T) {
	valid := UdonExecutionReport{
		Version: UdonExecutionReportVersion, Status: "success",
		StartedAt: "2026-04-29T12:00:00Z", FinishedAt: "2026-04-29T12:00:01Z",
		WorkflowPath: "workflow.uws.yaml", WorkflowFormat: "uws-yaml", WorkDir: ".",
	}
	for _, mutate := range []func(*UdonExecutionReport){
		func(report *UdonExecutionReport) { report.Status = "passed" },
		func(report *UdonExecutionReport) { report.StartedAt = "" },
		func(report *UdonExecutionReport) { report.FinishedAt = "2026-04-29T11:59:59Z" },
		func(report *UdonExecutionReport) { report.WorkflowPath = "" },
		func(report *UdonExecutionReport) { report.OutputDigest = "sha256:" + strings.Repeat("A", 64) },
	} {
		report := valid
		mutate(&report)
		data, err := json.Marshal(report)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := decodeUdonExecutionReport(data); err == nil {
			t.Fatalf("invalid executor report was accepted: %s", data)
		}
	}
}

func writeExternalRunnerReport(t *testing.T, invocation udonrunner.Invocation, now time.Time) {
	t.Helper()
	configPath := argValue(t, invocation.Argv, "--config")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	var config RunConfig
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}
	path, err := externalExecutorReportPath(config)
	if err != nil {
		t.Fatal(err)
	}
	writeUdonExecutionReport(t, path, "success", now, "")
}

func runEvidenceGateStatus(evidence RunEvidence, name string) string {
	for _, gate := range evidence.Gates {
		if gate.Name == name {
			return gate.Status
		}
	}
	return ""
}

func writeApprovalFile(t *testing.T, path string, approval Approval) {
	t.Helper()
	data, err := json.MarshalIndent(approval, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, path, append(data, '\n'))
}

func mustWriteFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func copyFile(t *testing.T, from, to string) {
	t.Helper()
	data, err := os.ReadFile(from)
	if err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, to, data)
}

func stringSliceContains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
