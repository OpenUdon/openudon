package synthesize

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/OpenUdon/browsertools/profile"
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
	invented := intent.Clone()
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
	data := synthesizeBrowserProfileFixture(true, true, "note")
	mustWriteSynthesizeTestFile(t, filepath.Join(example, filepath.FromSlash(path)), data)
	digest := sha256.Sum256(data)
	reviewData := `{"version":"openudon.browser-source-review.v1","route":"browser","session_posture":"opaque-runtime-binding-required","mutation_approvals":["update"],"sources":[{"id":"editor","target_path":"browser-profiles/editor.json","sha256":"` + hex.EncodeToString(digest[:]) + `","actions":["update_record"],"origins":["https://example.test"],"lifecycle":"active","expires_at":"2026-09-14T00:00:00Z","login_state_required":true,"provenance":"local:test"}]}`
	mustWriteSynthesizeTestFile(t, filepath.Join(example, filepath.FromSlash(packageartifacts.BrowserSourceReviewPath)), []byte(reviewData))
	result := resultPaths(example)
	markdown := reviewMarkdown(result, "", "")
	for _, want := range []string{"## Browser Sources", hex.EncodeToString(digest[:]), "`update_record`", "`https://example.test`", "opaque operator-owned runtime binding required", "`update`"} {
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

func boolJSON(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
