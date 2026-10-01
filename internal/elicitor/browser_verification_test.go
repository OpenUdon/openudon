package elicitor

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/OpenUdon/browsertools/profile"
	"github.com/OpenUdon/openudon/internal/browserverify"
	"github.com/OpenUdon/openudon/internal/projectwizard"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

func TestAttachBrowserVerificationsBindsDeduplicatesAndRevalidates(t *testing.T) {
	profileData, prof := browserVerificationFixture(t)
	source := browserVerificationSource(profileData)
	report := browserVerificationLiveReport(t, prof, true)
	first := writeBrowserVerificationFile(t, report)
	second := writeBrowserVerificationFile(t, report)

	sources, err := AttachBrowserVerifications([]SourceMaterialization{source}, []string{first, first, second}, browserVerificationAt())
	if err != nil {
		t.Fatal(err)
	}
	if len(sources) != 1 || len(sources[0].BrowserVerifications) != 1 {
		t.Fatalf("sources = %#v", sources)
	}
	attachment := sources[0].BrowserVerifications[0]
	if attachment.Summary.ReportVersion != browserverify.LiveCheckVersion || !attachment.Summary.OK || attachment.SourcePath == "" {
		t.Fatalf("attachment = %#v", attachment)
	}
	if _, err := RevalidateBrowserVerifications(sources, browserVerificationAt()); err != nil {
		t.Fatal(err)
	}

	changed := browserVerificationLiveReport(t, prof, true)
	changed.CheckedAt = "2026-08-16T12:01:00Z"
	data, err := json.Marshal(changed)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(attachment.SourcePath, append(data, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := RevalidateBrowserVerifications(sources, browserVerificationAt()); err == nil || !strings.Contains(err.Error(), "changed after review") {
		t.Fatalf("changed report error = %v", err)
	}
}

func TestAttachBrowserVerificationsRetainedSummaryWinsOverRepeatedCLIPath(t *testing.T) {
	profileData, prof := browserVerificationFixture(t)
	reportPath := writeBrowserVerificationFile(t, browserVerificationLiveReport(t, prof, true))
	sources, err := AttachBrowserVerifications(
		[]SourceMaterialization{browserVerificationSource(profileData)},
		[]string{reportPath}, browserVerificationAt(),
	)
	if err != nil {
		t.Fatal(err)
	}
	changed := browserVerificationLiveReport(t, prof, true)
	changed.CheckedAt = "2026-08-16T12:01:00Z"
	data, err := json.Marshal(changed)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(reportPath, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := AttachBrowserVerifications(sources, []string{reportPath}, browserVerificationAt()); err == nil || !strings.Contains(err.Error(), "changed after review") {
		t.Fatalf("repeated changed report error = %v", err)
	}
}

func TestAttachBrowserVerificationsCanonicalizesBeforeDuplicateOrdering(t *testing.T) {
	profileData, prof := browserVerificationFixture(t)
	reportDir := t.TempDir()
	reportPath := filepath.Join(reportDir, "verification.json")
	data, err := json.Marshal(browserVerificationLiveReport(t, prof, true))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(reportPath, data, 0o600); err != nil {
		t.Fatal(err)
	}
	sources, err := AttachBrowserVerifications([]SourceMaterialization{browserVerificationSource(profileData)}, []string{reportPath}, browserVerificationAt())
	if err != nil {
		t.Fatal(err)
	}
	changed := browserVerificationLiveReport(t, prof, true)
	changed.CheckedAt = "2026-08-16T12:01:00Z"
	data, _ = json.Marshal(changed)
	if err := os.WriteFile(reportPath, data, 0o600); err != nil {
		t.Fatal(err)
	}
	child := filepath.Join(reportDir, "child")
	if err := os.Mkdir(child, 0o700); err != nil {
		t.Fatal(err)
	}
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(child); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(previous)
	if _, err := AttachBrowserVerifications(sources, []string{"../verification.json"}, browserVerificationAt()); err == nil || !strings.Contains(err.Error(), "changed after review") {
		t.Fatalf("relative duplicate bypassed retained summary: %v", err)
	}
}

func TestAttachBrowserVerificationsRejectsMismatchPrivateAndConflict(t *testing.T) {
	profileData, prof := browserVerificationFixture(t)
	source := browserVerificationSource(profileData)

	mismatch := browserVerificationLiveReport(t, prof, true)
	mismatch.ProfileDigest = "sha256:" + strings.Repeat("0", 64)
	if _, err := AttachBrowserVerifications([]SourceMaterialization{source}, []string{writeBrowserVerificationFile(t, mismatch)}, browserVerificationAt()); err == nil || !strings.Contains(err.Error(), "does not match a selected") {
		t.Fatalf("mismatch error = %v", err)
	}

	profileDigest, err := browserverify.ProfileDigest(prof)
	if err != nil {
		t.Fatal(err)
	}
	private := map[string]any{"version": "browsertools.private-rich-evidence.v1", "profileDigest": profileDigest}
	if _, err := AttachBrowserVerifications([]SourceMaterialization{source}, []string{writeBrowserVerificationFile(t, private)}, browserVerificationAt()); err == nil || !strings.Contains(err.Error(), "not a value-free") {
		t.Fatalf("private error = %v", err)
	}

	passed := browserVerificationLiveReport(t, prof, true)
	failed := browserVerificationLiveReport(t, prof, false)
	if _, err := AttachBrowserVerifications([]SourceMaterialization{source}, []string{writeBrowserVerificationFile(t, passed), writeBrowserVerificationFile(t, failed)}, browserVerificationAt()); err == nil || !strings.Contains(err.Error(), "conflict") {
		t.Fatalf("conflict error = %v", err)
	}
}

func TestAttachBrowserVerificationsRejectsMalformedRetainedAttachment(t *testing.T) {
	profileData, _ := browserVerificationFixture(t)
	source := browserVerificationSource(profileData)
	source.BrowserVerifications = []browserverify.Attachment{{}}
	if _, err := AttachBrowserVerifications([]SourceMaterialization{source}, nil, browserVerificationAt()); err == nil || !strings.Contains(err.Error(), "path") {
		t.Fatalf("malformed retained attachment error = %v", err)
	}
}

func TestAttachBrowserVerificationsRejectsAmbiguousIdenticalProfiles(t *testing.T) {
	profileData, prof := browserVerificationFixture(t)
	first := browserVerificationSource(profileData)
	second := first
	second.ID = "copy"
	second.TargetPath = "browser-profiles/copy.json"
	report := writeBrowserVerificationFile(t, browserVerificationLiveReport(t, prof, true))
	if _, err := AttachBrowserVerifications([]SourceMaterialization{first, second}, []string{report}, browserVerificationAt()); err == nil || !strings.Contains(err.Error(), "multiple selected browser profiles") {
		t.Fatalf("ambiguous error = %v", err)
	}
}

func TestValidateBrowserVerificationCoverageRejectsFailedAndUnrelatedReports(t *testing.T) {
	profileData, prof := browserVerificationFixture(t)
	source := browserVerificationSource(profileData)
	failedPath := writeBrowserVerificationFile(t, browserVerificationLiveReport(t, prof, false))
	failed, err := AttachBrowserVerifications([]SourceMaterialization{source}, []string{failedPath}, browserVerificationAt())
	if err != nil {
		t.Fatal(err)
	}
	session := Session{SourcePlan: failed, Intent: rollout.Intent{
		Source: source.TargetPath,
		Steps:  []*rollout.Step{{Name: "read", Type: "browser", Source: source.TargetPath, Operation: "read_status"}},
	}}
	if err := ValidateBrowserVerificationCoverage(session); err == nil || !strings.Contains(err.Error(), "failed check") {
		t.Fatalf("failed coverage error = %v", err)
	}

	passedPath := writeBrowserVerificationFile(t, browserVerificationLiveReport(t, prof, true))
	passed, err := AttachBrowserVerifications([]SourceMaterialization{source}, []string{passedPath}, browserVerificationAt())
	if err != nil {
		t.Fatal(err)
	}
	session.SourcePlan = passed
	if err := ValidateBrowserVerificationCoverage(session); err != nil {
		t.Fatal(err)
	}
	session.Intent.Steps[0].Source = "./" + source.TargetPath
	if err := ValidateBrowserVerificationCoverage(session); err != nil {
		t.Fatalf("normalized source reference rejected: %v", err)
	}
	session.Intent.Steps[0].Operation = "another_action"
	if err := ValidateBrowserVerificationCoverage(session); err == nil || !strings.Contains(err.Error(), "does not cover") {
		t.Fatalf("unrelated coverage error = %v", err)
	}

	session.SourcePlan[0].BrowserVerifications = nil
	if err := ValidateBrowserVerificationCoverage(session); err != nil {
		t.Fatalf("optional verification rejected: %v", err)
	}
}

func TestBrowserVerificationAttachmentSurvivesDraftRoundTrip(t *testing.T) {
	profileData, prof := browserVerificationFixture(t)
	source := browserVerificationSource(profileData)
	reportPath := writeBrowserVerificationFile(t, browserVerificationLiveReport(t, prof, true))
	sources, err := AttachBrowserVerifications([]SourceMaterialization{source}, []string{reportPath}, browserVerificationAt())
	if err != nil {
		t.Fatal(err)
	}
	session := Session{
		Version:    SessionVersion,
		Project:    projectwizard.Answers{ProjectName: "Status", Goal: "Read status"},
		SourcePlan: sources,
	}
	path := filepath.Join(t.TempDir(), "session.yaml")
	if err := SaveDraft(path, session); err != nil {
		t.Fatal(err)
	}
	loaded, ok, err := LoadDraft(path)
	if err != nil || !ok {
		t.Fatalf("LoadDraft ok=%t err=%v", ok, err)
	}
	if len(loaded.SourcePlan) != 1 || len(loaded.SourcePlan[0].BrowserVerifications) != 1 {
		t.Fatalf("loaded source plan = %#v", loaded.SourcePlan)
	}
	loaded.SourcePlan[0].MaterializedContent = profileData
	if _, err := RevalidateBrowserVerifications(loaded.SourcePlan, browserVerificationAt()); err != nil {
		t.Fatal(err)
	}
}

func browserVerificationFixture(t *testing.T) ([]byte, *profile.Profile) {
	t.Helper()
	data := []byte(`{"profile":"uws.browser.1.5","info":{"title":"Status","origin":"https://example.test"},"observationKind":"accessibility_snapshot","evidence":{"learnedAt":"2026-08-15T00:00:00Z","source":"synthetic_fixture"},"confidence":"high","expiresAfter":"P30D","verification":{"lastVerifiedAt":"2026-08-15T00:00:00Z","successfulRuns":1},"actions":{"read_status":{"sequence":[{"navigate":"/member"},{"wait_for":{"role":"status","name":"Ready"}}],"outputs":{"status":{"type":"string","source":"a11y","locator":{"role":"status","name":"Ready"}}},"sideEffects":["read_only"],"confirmationPolicy":{"required":false}}}}`)
	prof, err := profile.ParseJSON(data)
	if err != nil {
		t.Fatal(err)
	}
	return data, prof
}

func browserVerificationSource(data []byte) SourceMaterialization {
	digest := sha256.Sum256(data)
	return SourceMaterialization{
		Kind: browserSourceFamily, SourceKind: "profile", ID: "status", SourcePath: "/operator/status.json",
		TargetPath: "browser-profiles/status.json", SHA256: hex.EncodeToString(digest[:]), Title: "Status",
		OperationCount: 1, Actions: []string{"read_status"}, Origins: []string{"https://example.test"},
		Lifecycle: "active", ExpiresAt: "2026-09-14T00:00:00Z", Provenance: "synthetic_fixture",
		MaterializedContent: append([]byte(nil), data...),
	}
}

type browserLiveReportWire struct {
	Version       string                `json:"version"`
	ProfileDigest string                `json:"profileDigest"`
	CheckedAt     string                `json:"checkedAt"`
	Origin        string                `json:"origin"`
	Actions       []string              `json:"actions"`
	OK            bool                  `json:"ok"`
	Checks        []browserverify.Check `json:"checks"`
}

func browserVerificationLiveReport(t *testing.T, prof *profile.Profile, ok bool) browserLiveReportWire {
	t.Helper()
	digest, err := browserverify.ProfileDigest(prof)
	if err != nil {
		t.Fatal(err)
	}
	matches := 1
	observed := profile.OutputString
	outputMessage := "declared output source and JSON type matched"
	locatorMessage := "declared accessibility locator resolved exactly once"
	if !ok {
		matches = 0
		observed = ""
		outputMessage = "declared output source or JSON type did not match"
		locatorMessage = "declared accessibility locator did not resolve exactly once"
	}
	return browserLiveReportWire{
		Version: browserverify.LiveCheckVersion, ProfileDigest: digest, CheckedAt: "2026-08-16T12:00:00Z",
		Origin: "https://example.test", Actions: []string{"read_status"}, OK: ok,
		Checks: []browserverify.Check{
			{Kind: "output", Path: "actions.read_status.outputs.status", OK: ok, Matches: matches, ExpectedType: profile.OutputString, ObservedType: observed, Message: outputMessage},
			{Kind: "locator", Path: "actions.read_status.sequence[1].wait_for", OK: ok, Matches: matches, Message: locatorMessage},
		},
	}
}

func writeBrowserVerificationFile(t *testing.T, value any) string {
	t.Helper()
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "verification.json")
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func browserVerificationAt() time.Time {
	return time.Date(2026, 8, 16, 13, 0, 0, 0, time.UTC)
}
