package artifactwriter_test

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
	"github.com/OpenUdon/openudon/internal/icot/artifactwriter"
	"github.com/OpenUdon/openudon/internal/icot/elicitor"
	"github.com/OpenUdon/openudon/internal/packageartifacts"
	"github.com/OpenUdon/openudon/internal/synthesize"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

func TestReviewedRegistrationPreparationBuildsSessionFreeUWSPackage(t *testing.T) {
	at := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	profileValue, err := registrationprofile.Parse([]byte(`profile: uws.browser-registration.1.0
info:
  title: Reviewed registration
  applicationOrigins: [https://app.example.test]
  registrationOrigins: [https://app.example.test]
observationKind: accessibility_snapshot
evidence: {learnedAt: "2026-08-25T00:00:00Z", source: synthetic_fixture}
confidence: high
expiresAfter: P30D
verification: {lastVerifiedAt: "2026-08-25T00:00:00Z"}
credentialSlots:
  identifier: {kind: identifier}
flows:
  create_account:
    sequence:
      - navigate: https://app.example.test/register
      - type_credential: {locator: {role: textbox, name: Account identifier}, slot: identifier}
      - submit: {locator: {role: button, name: Create account}}
      - wait_for: {locator: {role: status, name: Complete}}
    effects: [creates_account]
    confirmationPolicy: {required: true}
    success: {origin: https://app.example.test, locator: {role: status, name: Complete}}
`))
	if err != nil {
		t.Fatal(err)
	}
	profileBytes, err := registrationprofile.MarshalJSON(profileValue)
	if err != nil {
		t.Fatal(err)
	}
	profileBytes = append(profileBytes, '\n')
	bundle, err := registrationreview.Build(profileValue, at.Add(-24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	reviewBytes, err := json.Marshal(bundle)
	if err != nil {
		t.Fatal(err)
	}
	reviewBytes = append(reviewBytes, '\n')
	profileDigest, reviewDigest := sha256.Sum256(profileBytes), sha256.Sum256(reviewBytes)
	expiresAt, err := registrationprofile.ExpiresAt(profileValue)
	if err != nil {
		t.Fatal(err)
	}
	profilePath := "browser-registration/reviewed.json"
	reviewPath := packageartifacts.BrowserRegistrationBundlePath(profilePath)
	timeout := 300.0
	intent := rollout.Intent{
		Workflow: &rollout.WorkflowMeta{Name: "register_account", Description: "Prepare one reviewed account registration."},
		Steps: []*rollout.Step{{
			Name: "register_account", Type: "browser_registration", Do: "Create one account only after exact approval.",
			Source: profilePath, RegistrationFlow: "create_account", RegistrationApproval: "register_account",
			DuplicatePrevention: "operator_attestation", OnDuplicate: "fail", AmbiguousOutcome: "stop_without_retry", CleanupDisposition: "delete_separately",
			CredentialBindings: map[string]string{"identifier": "registration_identifier"}, Timeout: &timeout,
		}},
	}
	intentHCL, err := rollout.RenderIntentHCL(&intent)
	if err != nil {
		t.Fatal(err)
	}
	session := elicitor.Session{
		Intent: intent, BrowserRoute: "browser", BrowserSession: "none", Credentials: []string{"registration_identifier"}, CredentialsSet: true,
		SourcePlan: []elicitor.SourceMaterialization{{
			Kind: "browser-registration", ID: "reviewed", SourceKind: "browsertools_transaction", SourcePath: "virtual-browser://reviewed/registration/source",
			TargetPath: profilePath, SHA256: hex.EncodeToString(profileDigest[:]), ReviewPath: reviewPath, ReviewSHA256: hex.EncodeToString(reviewDigest[:]),
			Title: profileValue.Info.Title, OperationCount: 1, Flows: []string{"create_account"}, FlowCredentialSlots: map[string][]string{"create_account": {"identifier"}},
			Origins: registrationprofile.Origins(profileValue), Lifecycle: "active", ExpiresAt: expiresAt.Format(time.RFC3339), Provenance: "browsertools-transaction:fixture",
			MaterializedContent: profileBytes, MaterializedReview: reviewBytes,
		}},
	}
	example := filepath.Join(t.TempDir(), "reviewed-registration")
	prepared, err := artifactwriter.Prepare(example, elicitor.Artifacts{
		ProjectMD: `# Reviewed Registration

## Goal

Build a deterministic package from reviewed registration intent.

## Inputs

- Inputs are declared in intent.hcl.

## Outputs

- Outputs are declared in intent.hcl.

## External Systems and OpenAPI

OpenAPI: none required

- Use only the reviewed local browser registration profile.

## Data Flow

- Follow explicit intent.hcl step request mappings and dependencies.

## Runtime Policy

- browser_registration is allowed only for an explicitly approved sandbox proof run through a trusted runtime.
- Account creation requires exact approval; package and dry-run qualification do not execute it.

## Function Contracts

- No function runtime is required.

## Credentials and Secrets

- Use the symbolic runtime binding registration_identifier; never store its value.

## Safety and Approval Boundary

- Generate and validate artifacts only.
- Any side-effectful workflow requires approved trusted runner execution.
- Use sandbox endpoints for proof runs before production handoff.

## Fallback Behavior

- Stop if required package artifacts cannot be generated or validated.
`,
		IntentHCL: intentHCL,
		Session:   session,
	}, false, at)
	if err != nil {
		t.Fatal(err)
	}
	for _, relative := range []string{profilePath, reviewPath, packageartifacts.BrowserRegistrationReviewPath} {
		want := filepath.Join(example, filepath.FromSlash(relative))
		found := false
		for _, file := range prepared.Files {
			if file.Path == want && !file.Remove {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("prepared transaction omitted %s", relative)
		}
	}
	if _, err := artifactwriter.Commit(prepared, false); err != nil {
		t.Fatal(err)
	}
	metadata, err := os.ReadFile(filepath.Join(example, filepath.FromSlash(packageartifacts.BrowserRegistrationReviewPath)))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"approval": "register_account"`, `"cleanup_disposition": "delete_separately"`, `"review_path": "browser-registration/reviewed.review.json"`} {
		if !strings.Contains(string(metadata), want) {
			t.Fatalf("registration metadata omitted %s:\n%s", want, metadata)
		}
	}
	if strings.Contains(string(metadata), "browser_session") {
		t.Fatalf("registration metadata carries a browser session: %s", metadata)
	}
	result, report, err := synthesize.PackageFromIntent(context.Background(), synthesize.Options{ExampleDir: example})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Passed() {
		t.Fatalf("registration package quality failed: %#v", report.Checks)
	}
	uws, err := os.ReadFile(result.UWSPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(uws), "uws.browser-registration-call.1.0") || strings.Contains(string(uws), "browserSession") || strings.Contains(string(uws), "browser_session") {
		t.Fatalf("registration UWS lowering is not session-free: %s", uws)
	}

	tampered := session
	tampered.SourcePlan = append([]elicitor.SourceMaterialization(nil), session.SourcePlan...)
	tampered.SourcePlan[0].MaterializedReview = append([]byte(nil), session.SourcePlan[0].MaterializedReview...)
	tampered.SourcePlan[0].MaterializedReview[0] ^= 0xff
	tamperedDigest := sha256.Sum256(tampered.SourcePlan[0].MaterializedReview)
	tampered.SourcePlan[0].ReviewSHA256 = hex.EncodeToString(tamperedDigest[:])
	if _, _, err := artifactwriter.BrowserRegistrationMetadataJSON(tampered, at); err == nil || !strings.Contains(err.Error(), "review is invalid") {
		t.Fatalf("tampered registration review error = %v", err)
	}
}
