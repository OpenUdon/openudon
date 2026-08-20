package synthesize

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/OpenUdon/browsertools/profile"
	"github.com/OpenUdon/openudon/internal/browserverify"
	"github.com/OpenUdon/openudon/internal/packageartifacts"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

func TestValidateBrowserSourceReviewBindsProfileActionAndSafety(t *testing.T) {
	example := t.TempDir()
	path := "browser-profiles/editor.json"
	data := synthesizeBrowserProfileFixture(true, true, "note")
	mustWriteSynthesizeTestFile(t, filepath.Join(example, filepath.FromSlash(path)), data)
	value, err := profile.ParseJSON(data)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(data)
	review := browserSourceReview{
		Version: browserSourceReviewVersion, Route: "browser", SessionPosture: "opaque-runtime-binding-required",
		MutationApprovals: []string{"update"},
		Sources: []browserReviewedSource{{
			ID: "editor", TargetPath: path, SHA256: hex.EncodeToString(digest[:]), Actions: value.SortedActionNames(),
			Origins: []string(value.Info.Origin), Lifecycle: "active", ExpiresAt: "2026-09-14T00:00:00Z",
			LoginStateRequired: true, Provenance: "local:editor.json",
		}},
	}
	intent := &rollout.Intent{
		Source: path,
		Steps:  []*rollout.Step{{Name: "update", Type: "browser", Source: path, Operation: "update_record", With: map[string]string{"note": "inputs.note"}}},
	}
	at := time.Date(2026, 8, 16, 0, 0, 0, 0, time.UTC)
	if err := validateBrowserSourceReview(example, []string{path}, intent, review, at); err != nil {
		t.Fatal(err)
	}

	withoutApproval := review
	withoutApproval.MutationApprovals = nil
	if err := validateBrowserSourceReview(example, []string{path}, intent, withoutApproval, at); err == nil || !strings.Contains(err.Error(), "without operation-specific authoring approval") {
		t.Fatalf("expected mutation approval rejection, got %v", err)
	}
	invented, err := intent.Clone()
	if err != nil {
		t.Fatal(err)
	}
	invented.Steps[0].Operation = "invented_action"
	if err := validateBrowserSourceReview(example, []string{path}, invented, review, at); err == nil || !strings.Contains(err.Error(), "invents browser action") {
		t.Fatalf("expected invented action rejection, got %v", err)
	}
	if err := validateBrowserSourceReview(example, []string{path}, intent, review, time.Date(2026, 9, 14, 0, 0, 0, 0, time.UTC)); err == nil || !strings.Contains(err.Error(), "expired") {
		t.Fatalf("expected stale profile rejection, got %v", err)
	}
	tampered := review
	tampered.Sources = append([]browserReviewedSource(nil), review.Sources...)
	tampered.Sources[0].SHA256 = strings.Repeat("0", 64)
	if err := validateBrowserSourceReview(example, []string{path}, intent, tampered, at); err == nil || !strings.Contains(err.Error(), "digest mismatch") {
		t.Fatalf("expected digest rejection, got %v", err)
	}
}

func TestValidateBrowserSourceReviewAcceptsOnlyBoundSuccessfulVerification(t *testing.T) {
	example := t.TempDir()
	path := "browser-profiles/status.json"
	data := synthesizeLongLivedBrowserProfileFixture(false, false, "item")
	mustWriteSynthesizeTestFile(t, filepath.Join(example, filepath.FromSlash(path)), data)
	value, err := profile.ParseJSON(data)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(data)
	summary := synthesizeLiveVerification(t, value)
	review := browserSourceReview{
		Version: browserSourceReviewVersion, Route: "browser", SessionPosture: "none",
		Sources: []browserReviewedSource{{
			ID: "status", TargetPath: path, SHA256: hex.EncodeToString(digest[:]), Actions: value.SortedActionNames(),
			Origins: []string(value.Info.Origin), Lifecycle: "active", ExpiresAt: "2126-08-15T00:00:00Z",
			Provenance: "local:status.json", Verifications: []browserverify.Summary{summary},
		}},
	}
	intent := &rollout.Intent{Source: path, Steps: []*rollout.Step{{Name: "read", Type: "browser", Source: path, Operation: "read_status"}}}
	at := time.Date(2026, 8, 16, 13, 0, 0, 0, time.UTC)
	if err := validateBrowserSourceReview(example, []string{path}, intent, review, at); err != nil {
		t.Fatal(err)
	}

	failed := review
	failed.Sources = append([]browserReviewedSource(nil), review.Sources...)
	failed.Sources[0].Verifications = append([]browserverify.Summary(nil), summary)
	failed.Sources[0].Verifications[0].OK = false
	if err := validateBrowserSourceReview(example, []string{path}, intent, failed, at); err == nil || !strings.Contains(err.Error(), "ok does not match") {
		t.Fatalf("failed summary error = %v", err)
	}

	duplicate := review
	duplicate.Sources = append([]browserReviewedSource(nil), review.Sources...)
	duplicate.Sources[0].Verifications = []browserverify.Summary{summary, summary}
	if err := validateBrowserSourceReview(example, []string{path}, intent, duplicate, at); err == nil || !strings.Contains(err.Error(), "duplicated") {
		t.Fatalf("duplicate summary error = %v", err)
	}
}

func TestBrowserSourceReviewStrictDecodeRejectsUnknownAndTrailingFields(t *testing.T) {
	for _, data := range [][]byte{
		[]byte(`{"version":"openudon.browser-source-review.v1","route":"browser","session_posture":"none","sources":[],"raw_capture":"secret"}`),
		[]byte(`{"version":"openudon.browser-source-review.v1","route":"browser","session_posture":"none","sources":[]} {}`),
		[]byte(`{"version":"openudon.browser-source-review.v1","route":"browser","route":"browser","session_posture":"none","sources":[]}`),
	} {
		var review browserSourceReview
		if err := decodeBrowserSourceReview(data, &review); err == nil {
			t.Fatalf("strict decoder accepted %s", data)
		}
	}
}

func TestReviewMarkdownSurfacesValueFreeBrowserVerification(t *testing.T) {
	example := t.TempDir()
	path := "browser-profiles/status.json"
	data := synthesizeLongLivedBrowserProfileFixture(false, false, "item")
	mustWriteSynthesizeTestFile(t, filepath.Join(example, filepath.FromSlash(path)), data)
	value, err := profile.ParseJSON(data)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(data)
	review := browserSourceReview{
		Version: browserSourceReviewVersion, Route: "browser", SessionPosture: "none",
		Sources: []browserReviewedSource{{
			ID: "status", TargetPath: path, SHA256: hex.EncodeToString(digest[:]), Actions: value.SortedActionNames(),
			Origins: []string(value.Info.Origin), Lifecycle: "active", ExpiresAt: "2126-08-15T00:00:00Z",
			Provenance: "local:test", Verifications: []browserverify.Summary{synthesizeLiveVerification(t, value)},
		}},
	}
	reviewData, err := json.Marshal(review)
	if err != nil {
		t.Fatal(err)
	}
	mustWriteSynthesizeTestFile(t, filepath.Join(example, filepath.FromSlash(packageartifacts.BrowserSourceReviewPath)), reviewData)
	markdown := reviewMarkdown(resultPaths(example), "", "")
	for _, want := range []string{"Current-page verification", "Chromium", "value-free check", "report SHA-256"} {
		if !strings.Contains(markdown, want) {
			t.Fatalf("review evidence missing %q:\n%s", want, markdown)
		}
	}
}

func TestReviewMarkdownDoesNotTrustTamperedBrowserVerification(t *testing.T) {
	example := t.TempDir()
	path := "browser-profiles/status.json"
	data := synthesizeLongLivedBrowserProfileFixture(false, false, "item")
	mustWriteSynthesizeTestFile(t, filepath.Join(example, filepath.FromSlash(path)), data)
	value, err := profile.ParseJSON(data)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(data)
	summary := synthesizeLiveVerification(t, value)
	summary.Checks[0].Path = "actions.read_status.outputs.invented"
	review := browserSourceReview{
		Version: browserSourceReviewVersion, Route: "browser", SessionPosture: "none",
		Sources: []browserReviewedSource{{
			ID: "status", TargetPath: path, SHA256: hex.EncodeToString(digest[:]), Actions: value.SortedActionNames(),
			Origins: []string(value.Info.Origin), Lifecycle: "active", ExpiresAt: "2126-08-15T00:00:00Z",
			Provenance: "local:status.json", Verifications: []browserverify.Summary{summary},
		}},
	}
	reviewData, err := json.Marshal(review)
	if err != nil {
		t.Fatal(err)
	}
	mustWriteSynthesizeTestFile(t, filepath.Join(example, filepath.FromSlash(packageartifacts.BrowserSourceReviewPath)), reviewData)
	markdown := reviewMarkdown(resultPaths(example), "", "")
	if !strings.Contains(markdown, "Browser source review evidence is invalid") || strings.Contains(markdown, "Current-page verification: `passed`") {
		t.Fatalf("tampered verification was trusted:\n%s", markdown)
	}
}

func TestValidatePackagedBrowserProfileRejectsRawOrSecretShapedFields(t *testing.T) {
	value, err := profile.ParseJSON(synthesizeBrowserProfileFixture(false, false, "session_cookie"))
	if err != nil {
		t.Fatal(err)
	}
	if err := validatePackagedBrowserProfile(value); err == nil || !strings.Contains(err.Error(), "credential, session, or raw-capture shaped") {
		t.Fatalf("expected sensitive browser field rejection, got %v", err)
	}
}

func TestReviewHandoffIncludesBrowserProfileAndReviewEvidence(t *testing.T) {
	example := t.TempDir()
	mustWriteSynthesizeTestFile(t, filepath.Join(example, "browser-profiles", "status.json"), synthesizeBrowserProfileFixture(false, false, "item"))
	mustWriteSynthesizeTestFile(t, filepath.Join(example, filepath.FromSlash(packageartifacts.BrowserSourceReviewPath)), []byte(`{"version":"openudon.browser-source-review.v1"}`))
	inputs, err := reviewHandoffInputs(resultPaths(example))
	if err != nil {
		t.Fatal(err)
	}
	if !handoffInputContains(inputs, "browser-profiles/status.json") || !handoffInputContains(inputs, packageartifacts.BrowserSourceReviewPath) {
		t.Fatalf("browser handoff inputs = %#v", inputs)
	}
	paths, err := packageartifacts.RequiredPackagePaths(example)
	if err != nil {
		t.Fatal(err)
	}
	if !containsString(paths, "browser-profiles/status.json") || !containsString(paths, packageartifacts.BrowserSourceReviewPath) {
		t.Fatalf("browser package inventory = %#v", paths)
	}
}

func TestBrowserSideEffectsComeFromProfileMetadata(t *testing.T) {
	example := t.TempDir()
	path := "browser-profiles/editor.json"
	mustWriteSynthesizeTestFile(t, filepath.Join(example, filepath.FromSlash(path)), synthesizeBrowserProfileFixture(true, false, "note"))
	intent := &rollout.Intent{Source: path, Steps: []*rollout.Step{{Name: "apply", Type: "browser", Source: path, Operation: "update_record"}}}
	profile := sideEffectProfileForSources(projectPolicy{}, intent, nil, "", example)
	if !profile.SideEffectful || len(profile.Effects) == 0 || profile.Effects[0].Kind != "browser" {
		t.Fatalf("browser side-effect profile = %#v", profile)
	}
}

func TestReviewMarkdownSurfacesBrowserDigestActionAndSafetyEvidence(t *testing.T) {
	example := t.TempDir()
	path := "browser-profiles/editor.json"
	data := synthesizeLongLivedBrowserProfileFixture(true, true, "note")
	mustWriteSynthesizeTestFile(t, filepath.Join(example, filepath.FromSlash(path)), data)
	digest := sha256.Sum256(data)
	reviewData := `{"version":"openudon.browser-source-review.v1","route":"browser","session_posture":"opaque-runtime-binding-required","sources":[{"id":"editor","target_path":"browser-profiles/editor.json","sha256":"` + hex.EncodeToString(digest[:]) + `","actions":["update_record"],"origins":["https://example.test"],"lifecycle":"active","expires_at":"2126-08-15T00:00:00Z","login_state_required":true,"provenance":"local:test"}]}`
	mustWriteSynthesizeTestFile(t, filepath.Join(example, filepath.FromSlash(packageartifacts.BrowserSourceReviewPath)), []byte(reviewData))
	result := resultPaths(example)
	markdown := reviewMarkdown(result, "", "")
	for _, want := range []string{"## Browser Sources", hex.EncodeToString(digest[:]), "`update_record`", "`https://example.test`", "opaque operator-owned runtime binding required"} {
		if !strings.Contains(markdown, want) {
			t.Fatalf("review evidence missing %q:\n%s", want, markdown)
		}
	}
}

func synthesizeBrowserProfileFixture(mutating, login bool, parameter string) []byte {
	effect := "read_only"
	actionName := "read_status"
	confirmation := `{"required":false}`
	if mutating {
		effect = "updates_record"
		actionName = "update_record"
		confirmation = `{"required":true,"prompt":"Approve updating the selected record."}`
	}
	return []byte(`{"profile":"uws.browser.1.5","info":{"title":"Example Browser","origin":"https://example.test","loginStateRequired":` + boolJSON(login) + `},"observationKind":"accessibility_snapshot","evidence":{"learnedAt":"2026-08-15T00:00:00Z","source":"synthetic_fixture"},"confidence":"high","expiresAfter":"P30D","verification":{"lastVerifiedAt":"2026-08-15T00:00:00Z","successfulRuns":2,"uiStabilityScore":0.95},"actions":{"` + actionName + `":{"description":"Reviewed browser action.","parameters":{"type":"object","properties":{"` + parameter + `":{"type":"string"}},"required":["` + parameter + `"]},"sequence":[{"navigate":"/status"},{"wait_for":{"role":"status","name":"Ready"}}],"outputs":{"status":{"type":"string","source":"a11y","locator":{"role":"status","name":"Ready"}}},"sideEffects":["` + effect + `"],"confirmationPolicy":` + confirmation + `}}}`)
}

func synthesizeLongLivedBrowserProfileFixture(mutating, login bool, parameter string) []byte {
	return []byte(strings.Replace(string(synthesizeBrowserProfileFixture(mutating, login, parameter)), `"expiresAfter":"P30D"`, `"expiresAfter":"P100Y"`, 1))
}

func synthesizeLiveVerification(t *testing.T, value *profile.Profile) browserverify.Summary {
	t.Helper()
	digest, err := browserverify.ProfileDigest(value)
	if err != nil {
		t.Fatal(err)
	}
	action := value.SortedActionNames()[0]
	return browserverify.Summary{
		ReportVersion: browserverify.LiveCheckVersion, SourceSHA256: "sha256:" + strings.Repeat("a", 64),
		ProfileDigest: digest, CheckedAt: "2026-08-16T00:00:00Z", Origin: "https://example.test",
		Actions: []string{action}, OK: true, Engine: "chromium",
		Checks: []browserverify.Check{
			{Kind: "output", Path: "actions." + action + ".outputs.status", OK: true, Matches: 1, ExpectedType: profile.OutputString, ObservedType: profile.OutputString, Message: "declared output source and JSON type matched"},
			{Kind: "locator", Path: "actions." + action + ".sequence[1].wait_for", OK: true, Matches: 1, Message: "declared accessibility locator resolved exactly once"},
		},
	}
}

func boolJSON(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
