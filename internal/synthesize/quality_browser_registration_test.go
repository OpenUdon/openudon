package synthesize

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/OpenUdon/browsertools/registrationprofile"
	"github.com/OpenUdon/browsertools/registrationreview"
	"github.com/OpenUdon/openudon/internal/packageartifacts"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
	"github.com/OpenUdon/uws/browserregistration"
	"github.com/OpenUdon/uws/uws1"
)

func TestRegistrationPlanInventoryMustMatchIntent(t *testing.T) {
	intent := &rollout.Intent{Steps: []*rollout.Step{{Name: "register_test_user", Type: "browser_registration"}}}
	plan := &WorkflowPlan{Version: workflowPlanVersion, Steps: []PlanStep{}}
	mismatches := planIntentInventoryMismatches(intent, plan)
	if len(mismatches) != 1 || !strings.Contains(mismatches[0], "missing intent step") {
		t.Fatalf("inventory mismatches = %#v", mismatches)
	}
}

func TestCompiledRegistrationValidationRejectsUnknownRawFields(t *testing.T) {
	doc := &uws1.Document{Operations: []*uws1.Operation{{
		OperationID: "register_test_user",
		Extensions: map[string]any{
			uws1.ExtensionOperationProfile: browserregistration.CallProfileName,
			browserregistration.ExtensionRegistration: map[string]any{
				"profile": "browser-registration/dedicated.yaml", "flow": "create_dedicated_test_user",
				"credentialBindings": map[string]any{"identifier": "test_identifier", "password": "test_password"},
				"approval":           "approve_account_creation", "duplicatePrevention": "operator_attestation", "onDuplicate": "fail",
				"ambiguousOutcome": "stop_without_retry", "cleanupDisposition": "delete_separately",
				"accountIdentifier": "must-not-be-accepted",
			},
		},
	}}}
	if err := validateCompiledRegistrationOperations(doc); err == nil || !strings.Contains(err.Error(), "additional properties") {
		t.Fatalf("unknown registration field error = %v", err)
	}
}

func TestPackageFromIntentBuildsBrowserRegistrationWorkflow(t *testing.T) {
	example := t.TempDir()
	profilePath := "browser-registration/dedicated.yaml"
	reviewPath := "browser-registration/dedicated.review.json"
	profileData := synthesizeBrowserRegistrationFixture()
	mustWriteSynthesizeTestFile(t, filepath.Join(example, filepath.FromSlash(profilePath)), profileData)
	profile, err := registrationprofile.Parse(profileData)
	if err != nil {
		t.Fatal(err)
	}
	assessedAt := time.Date(2026, 8, 25, 0, 0, 0, 0, time.UTC)
	bundle, err := registrationreview.Build(profile, assessedAt)
	if err != nil {
		t.Fatal(err)
	}
	bundleData, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	bundleData = append(bundleData, '\n')
	mustWriteSynthesizeTestFile(t, filepath.Join(example, filepath.FromSlash(reviewPath)), bundleData)

	profileRawDigest := sha256.Sum256(profileData)
	bundleRawDigest := sha256.Sum256(bundleData)
	timeout := 300.0
	intent := &rollout.Intent{
		Workflow: &rollout.WorkflowMeta{Name: "register_dedicated_test_user", Description: "Prepare one explicitly approved dedicated test-user registration."},
		Steps: []*rollout.Step{{
			Name: "register_dedicated_test_user", Type: "browser_registration", Do: "Create one dedicated test identity after exact approval.",
			Source: profilePath, RegistrationFlow: "create_dedicated_test_user", RegistrationApproval: "register_dedicated_test_user",
			DuplicatePrevention: "operator_attestation", OnDuplicate: "fail", AmbiguousOutcome: "stop_without_retry",
			CleanupDisposition: "delete_separately", CredentialBindings: map[string]string{
				"identifier": "dedicated_test_identifier", "password": "dedicated_test_password",
			}, Timeout: &timeout,
		}},
	}
	intentHCL, err := rollout.RenderIntentHCL(intent)
	if err != nil {
		t.Fatal(err)
	}
	mustWriteSynthesizeTestFile(t, filepath.Join(example, rollout.IntentPath), []byte(intentHCL))
	project := buildMatrixProject(
		"Dedicated Test Registration",
		"OpenAPI: none required\n\n- Use only the reviewed local browser registration profile.",
		"- browser_registration is allowed only for an explicitly approved sandbox proof run through a trusted runtime.\n- Account creation requires exact approval; package and dry-run qualification do not execute it.",
		"- No function runtime is required.",
		"- Use symbolic runtime bindings dedicated_test_identifier and dedicated_test_password; never store values.",
	)
	mustWriteSynthesizeTestFile(t, filepath.Join(example, "project.md"), []byte(project))
	openUdonReview := browserRegistrationReview{
		Version: browserRegistrationReviewVersion,
		Calls: []browserRegistrationReviewedCall{{
			Step: "register_dedicated_test_user", Source: profilePath, Flow: "create_dedicated_test_user",
			CredentialBindings: map[string]string{"identifier": "dedicated_test_identifier", "password": "dedicated_test_password"},
			Approval:           "register_dedicated_test_user", DuplicatePrevention: "operator_attestation", OnDuplicate: "fail",
			AmbiguousOutcome: "stop_without_retry", CleanupDisposition: "delete_separately", Timeout: timeout,
		}},
		Sources: []browserRegistrationReviewedSource{{
			ID: "dedicated_registration", TargetPath: profilePath, SHA256: hex.EncodeToString(profileRawDigest[:]),
			ReviewPath: reviewPath, ReviewSHA256: hex.EncodeToString(bundleRawDigest[:]), ProfileDigest: bundle.ProfileDigest,
			Title: "Dedicated test registration", Flows: []string{"create_dedicated_test_user"},
			FlowCredentialSlots: map[string][]string{"create_dedicated_test_user": {"identifier", "password"}},
			Origins:             []string{"https://example.test"}, Lifecycle: "active", ExpiresAt: bundle.ExpiresAt, Provenance: "synthetic_fixture",
		}},
	}
	reviewData, err := json.Marshal(openUdonReview)
	if err != nil {
		t.Fatal(err)
	}
	mustWriteSynthesizeTestFile(t, filepath.Join(example, filepath.FromSlash(packageartifacts.BrowserRegistrationReviewPath)), reviewData)

	result, report, err := PackageFromIntent(context.Background(), Options{ExampleDir: example})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Passed() {
		t.Fatalf("quality report failed: %#v", report.Checks)
	}
	assertPackageFileContains(t, example, "workflows/workflow.uws.yaml", "uws: 1.9.0", browserregistration.CallProfileName, "create_dedicated_test_user", "operator_attestation", "stop_without_retry", "delete_separately")
	workflowData, err := os.ReadFile(result.UWSPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"@example", "cookie", "verification_token"} {
		if strings.Contains(strings.ToLower(string(workflowData)), forbidden) {
			t.Fatalf("workflow contains forbidden value marker %q", forbidden)
		}
	}
	paths, err := packageartifacts.RequiredPackagePaths(result.ExampleDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{profilePath, reviewPath, packageartifacts.BrowserRegistrationReviewPath} {
		if !containsString(paths, want) {
			t.Fatalf("required package paths missing %s: %#v", want, paths)
		}
	}
}

func TestValidateBrowserRegistrationReviewRejectsTamper(t *testing.T) {
	example := t.TempDir()
	profilePath := "browser-registration/dedicated.yaml"
	profileData := synthesizeBrowserRegistrationFixture()
	mustWriteSynthesizeTestFile(t, filepath.Join(example, filepath.FromSlash(profilePath)), profileData)
	profile, err := registrationprofile.Parse(profileData)
	if err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 8, 25, 0, 0, 0, 0, time.UTC)
	bundle, err := registrationreview.Build(profile, at)
	if err != nil {
		t.Fatal(err)
	}
	bundleData, _ := json.Marshal(bundle)
	reviewPath := packageartifacts.BrowserRegistrationBundlePath(profilePath)
	mustWriteSynthesizeTestFile(t, filepath.Join(example, filepath.FromSlash(reviewPath)), bundleData)
	profileRawDigest := sha256.Sum256(profileData)
	bundleRawDigest := sha256.Sum256(bundleData)
	timeout := 300.0
	intent := &rollout.Intent{Steps: []*rollout.Step{{
		Name: "register", Type: "browser_registration", Source: profilePath, RegistrationFlow: "create_dedicated_test_user",
		RegistrationApproval: "register", DuplicatePrevention: "operator_attestation", OnDuplicate: "fail",
		AmbiguousOutcome: "stop_without_retry", CleanupDisposition: "retain_dedicated_test_identity",
		CredentialBindings: map[string]string{"identifier": "test_identifier", "password": "test_password"}, Timeout: &timeout,
	}}}
	review := browserRegistrationReview{
		Version: browserRegistrationReviewVersion,
		Calls: []browserRegistrationReviewedCall{{
			Step: "register", Source: profilePath, Flow: "create_dedicated_test_user", Approval: "register",
			CredentialBindings:  map[string]string{"identifier": "test_identifier", "password": "test_password"},
			DuplicatePrevention: "operator_attestation", OnDuplicate: "fail", AmbiguousOutcome: "stop_without_retry",
			CleanupDisposition: "retain_dedicated_test_identity", Timeout: timeout,
		}},
		Sources: []browserRegistrationReviewedSource{{
			ID: "dedicated", TargetPath: profilePath, SHA256: hex.EncodeToString(profileRawDigest[:]), ReviewPath: reviewPath,
			ReviewSHA256: hex.EncodeToString(bundleRawDigest[:]), ProfileDigest: bundle.ProfileDigest, Title: profile.Info.Title,
			Flows: []string{"create_dedicated_test_user"}, FlowCredentialSlots: map[string][]string{"create_dedicated_test_user": {"identifier", "password"}},
			Origins: []string{"https://example.test"}, Lifecycle: "active", ExpiresAt: bundle.ExpiresAt, Provenance: "synthetic_fixture",
		}},
	}
	if err := validateBrowserRegistrationReview(example, []string{profilePath}, intent, review, at); err != nil {
		t.Fatal(err)
	}
	review.Calls[0].CleanupDisposition = "delete_separately"
	if err := validateBrowserRegistrationReview(example, []string{profilePath}, intent, review, at); err == nil || !strings.Contains(err.Error(), "does not exactly match") {
		t.Fatalf("cleanup tamper error = %v", err)
	}
	review.Calls[0].CleanupDisposition = "retain_dedicated_test_identity"
	review.Sources[0].Provenance = "different_fixture"
	if err := validateBrowserRegistrationReview(example, []string{profilePath}, intent, review, at); err == nil || !strings.Contains(err.Error(), "provenance") {
		t.Fatalf("provenance tamper error = %v", err)
	}
}

func TestBrowserRegistrationProfileRejectsPIIAndSecretShapedValues(t *testing.T) {
	for _, test := range []struct {
		name        string
		replacement string
		want        string
	}{
		{"account identifier", "person" + string(rune(64)) + "example.test", "PII-shaped value"},
		{"credential literal", "Bearer " + strings.Repeat("a", 32), "secret-shaped value"},
	} {
		t.Run(test.name, func(t *testing.T) {
			example := t.TempDir()
			path := "browser-registration/dedicated.yaml"
			data := strings.Replace(string(synthesizeBrowserRegistrationFixture()), "synthetic_fixture", test.replacement, 1)
			mustWriteSynthesizeTestFile(t, filepath.Join(example, filepath.FromSlash(path)), []byte(data))
			_, err := browserRegistrationProfile(example, path)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
}

func synthesizeBrowserRegistrationFixture() []byte {
	return []byte(`profile: uws.browser-registration.1.0
info:
  title: Dedicated test registration
  applicationOrigins: [https://example.test]
  registrationOrigins: [https://example.test]
observationKind: accessibility_snapshot
evidence: {learnedAt: "2026-08-25T00:00:00Z", source: synthetic_fixture}
confidence: high
expiresAfter: P30D
verification: {lastVerifiedAt: "2026-08-25T00:00:00Z", uiStabilityScore: 0.95}
credentialSlots:
  identifier: {kind: identifier}
  password: {kind: password}
flows:
  create_dedicated_test_user:
    sequence:
      - navigate: https://example.test/register
      - type_credential: {locator: {role: textbox, name: Test identifier}, slot: identifier}
      - type_credential: {locator: {role: textbox, name: Test password}, slot: password}
      - submit: {locator: {role: button, name: Create test account}}
      - wait_for: {locator: {role: heading, name: Registration complete}}
    effects: [creates_account]
    confirmationPolicy: {required: true, prompt: Approve creation of one dedicated test account.}
    success: {origin: https://example.test, path: /registered, locator: {role: heading, name: Registration complete}}
`)
}
