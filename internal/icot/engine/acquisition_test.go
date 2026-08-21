package engine

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/OpenUdon/browsertools/authorresult"
	"github.com/OpenUdon/openudon/internal/icot/artifactwriter"
)

const uploadedOpenAPI = `{
  "openapi": "3.0.3",
  "info": {"title": "Member API", "version": "1.0.0"},
  "paths": {"/members": {"get": {"operationId": "listMembers", "responses": {"200": {"description": "ok"}}}}}
}`

func TestJourneyAndUploadedSourceLifecycle(t *testing.T) {
	root := t.TempDir()
	example := filepath.Join(root, "example")
	privateRoot := filepath.Join(root, "private")
	if err := os.Mkdir(example, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(privateRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	eng, _, err := Open(context.Background(), Config{ExampleDir: example, PrivateRoot: privateRoot, NetworkPolicy: "never"})
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := eng.SelectJourney(context.Background(), "api", "List the members visible to an operator")
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Journey.Starter != "api" || snapshot.Journey.Goal == "" {
		t.Fatalf("journey = %#v", snapshot.Journey)
	}
	uploaded, snapshot, err := eng.UploadSource(context.Background(), "member-api.json", strings.NewReader(uploadedOpenAPI))
	if err != nil {
		t.Fatal(err)
	}
	if uploaded.Kind != "openapi" || uploaded.CanonicalTarget != "openapi/member-api.json" || len(snapshot.UploadedSources) != 1 {
		t.Fatalf("uploaded = %#v, snapshot = %#v", uploaded, snapshot.UploadedSources)
	}
	snapshot, err = eng.StageUploadedSource(context.Background(), uploaded.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.UploadedSources) != 0 || len(snapshot.StagedSources) != 1 || len(snapshot.SourceCandidates.Local.Candidates) != 1 {
		t.Fatalf("staged snapshot = uploads %#v staged %#v candidates %#v", snapshot.UploadedSources, snapshot.StagedSources, snapshot.SourceCandidates.Local.Candidates)
	}
	if data, err := os.ReadFile(filepath.Join(example, "openapi", "member-api.json")); err != nil || !strings.Contains(string(data), "listMembers") {
		t.Fatalf("staged bytes = %q, %v", data, err)
	}
	snapshot, err = eng.RemoveStagedSource(context.Background(), uploaded.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.StagedSources) != 0 {
		t.Fatalf("staged sources after removal = %#v", snapshot.StagedSources)
	}
	if _, err := os.Stat(filepath.Join(example, "openapi", "member-api.json")); !os.IsNotExist(err) {
		t.Fatalf("removed source stat = %v", err)
	}
}

func TestUploadRejectsSecretsAndRequiresPrivateRoot(t *testing.T) {
	example := t.TempDir()
	eng, _, err := Open(context.Background(), Config{ExampleDir: example, NetworkPolicy: "never"})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := eng.UploadSource(context.Background(), "api.json", strings.NewReader(uploadedOpenAPI)); err == nil || !strings.Contains(err.Error(), "--private-root") {
		t.Fatalf("missing private root error = %v", err)
	}
	privateRoot := filepath.Join(t.TempDir(), "private")
	if err := os.Mkdir(privateRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	eng, _, err = Open(context.Background(), Config{ExampleDir: example, PrivateRoot: privateRoot, NetworkPolicy: "never"})
	if err != nil {
		t.Fatal(err)
	}
	secret := strings.Replace(uploadedOpenAPI, `"paths"`, `"api_key":"sk-proj-012345678901234567890123456789","paths"`, 1)
	if _, _, err := eng.UploadSource(context.Background(), "api.json", strings.NewReader(secret)); err == nil || !strings.Contains(err.Error(), "secret-like") {
		t.Fatalf("secret upload error = %v", err)
	}
}

func TestPrivateRootMustBeExactModeAndDisjoint(t *testing.T) {
	example := t.TempDir()
	inside := filepath.Join(example, "private")
	if err := os.Mkdir(inside, 0o700); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Open(context.Background(), Config{ExampleDir: example, PrivateRoot: inside}); err == nil || !strings.Contains(err.Error(), "disjoint") {
		t.Fatalf("inside private root error = %v", err)
	}
	outside := filepath.Join(t.TempDir(), "private")
	if err := os.Mkdir(outside, 0o750); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Open(context.Background(), Config{ExampleDir: example, PrivateRoot: outside}); err == nil || !strings.Contains(err.Error(), "0700") {
		t.Fatalf("mode private root error = %v", err)
	}
}

func TestCaptureStagingAndRemovalRejectDriftAtEveryCommitBoundary(t *testing.T) {
	for _, phase := range []string{"before_observation", "after_observation", "before_replacement"} {
		t.Run("stage_"+phase, func(t *testing.T) {
			eng, example, _ := acquisitionTestEngine(t)
			stage := validBrowserCaptureStage(t, "member")
			target := filepath.Join(example, "browser-profiles", "member.json")
			assertMutationRejected(t, phase, target, []byte("external profile\n"), func() error {
				_, err := eng.StageBrowserCapture(context.Background(), stage)
				return err
			})
		})
		t.Run("remove_"+phase, func(t *testing.T) {
			eng, example, uploaded := stagedAcquisitionTestEngine(t)
			target := filepath.Join(example, "openapi", "member-api.json")
			assertMutationRejected(t, phase, target, []byte("external source\n"), func() error {
				_, err := eng.RemoveStagedSource(context.Background(), uploaded.ID)
				return err
			})
		})
	}
}

func TestBrowserCaptureStagingRequiresConfiguredPrivateRoot(t *testing.T) {
	example := t.TempDir()
	eng, _, err := Open(context.Background(), Config{ExampleDir: example, NetworkPolicy: "never"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := eng.StageBrowserCapture(context.Background(), validBrowserCaptureStage(t, "member")); err == nil || !strings.Contains(err.Error(), "engine-configured private root") {
		t.Fatalf("missing private root error = %v", err)
	}
}

func assertMutationRejected(t *testing.T, phase, target string, external []byte, mutate func() error) {
	t.Helper()
	writeExternal := func() error {
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, external, 0o600)
	}
	originalCommit := commitPrepared
	defer func() { commitPrepared = originalCommit }()
	if phase == "before_observation" {
		if err := writeExternal(); err != nil {
			t.Fatal(err)
		}
	} else {
		commitPrepared = func(prepared artifactwriter.Prepared, force bool, beforeReplace func() error) (artifactwriter.Result, error) {
			return originalCommit(prepared, force, func() error {
				if phase == "after_observation" {
					if err := writeExternal(); err != nil {
						return err
					}
				}
				if err := beforeReplace(); err != nil {
					return err
				}
				if phase == "before_replacement" {
					return writeExternal()
				}
				return nil
			})
		}
	}
	mutationErr := mutate()
	if mutationErr == nil {
		t.Fatalf("%s drift was accepted", phase)
	}
	data, err := os.ReadFile(target)
	if err != nil || string(data) != string(external) {
		t.Fatalf("external bytes after %s = %q, %v; mutation error: %v", phase, data, err, mutationErr)
	}
}

func acquisitionTestEngine(t *testing.T) (*Engine, string, string) {
	t.Helper()
	root := t.TempDir()
	example := filepath.Join(root, "example")
	privateRoot := filepath.Join(root, "private")
	if err := os.Mkdir(example, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(privateRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	eng, _, err := Open(context.Background(), Config{ExampleDir: example, PrivateRoot: privateRoot, NetworkPolicy: "never"})
	if err != nil {
		t.Fatal(err)
	}
	return eng, example, privateRoot
}

func stagedAcquisitionTestEngine(t *testing.T) (*Engine, string, UploadedSource) {
	t.Helper()
	eng, example, _ := acquisitionTestEngine(t)
	uploaded, _, err := eng.UploadSource(context.Background(), "member-api.json", strings.NewReader(uploadedOpenAPI))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := eng.StageUploadedSource(context.Background(), uploaded.ID); err != nil {
		t.Fatal(err)
	}
	return eng, example, uploaded
}

func validBrowserCaptureStage(t *testing.T, profileID string) BrowserCaptureStage {
	t.Helper()
	capability, err := os.ReadFile(filepath.Join(repoRoot(t), "examples", "eval", "browser-status-read", "browser-profiles", "status.json"))
	if err != nil {
		t.Fatal(err)
	}
	authentication := []byte(`profile: uws.browser-authentication.1.1
info:
  title: Member popup and frame login
  applicationOrigins: [https://members.example.test]
  authenticationOrigins: [https://members.example.test]
observationKind: accessibility_snapshot
evidence: {learnedAt: "2026-08-20T00:00:00Z", source: reviewed}
confidence: high
expiresAfter: P30D
verification: {lastVerifiedAt: "2026-08-20T00:00:00Z", successfulRuns: 1, uiStabilityScore: 1}
credentialSlots:
  username: {kind: identifier}
  password: {kind: password}
flows:
  member_login:
    sequence:
      - navigate: https://members.example.test/login
      - type_credential:
          locator: {role: textbox, name: Username}
          slot: username
      - type_credential:
          locator: {role: textbox, name: Password}
          slot: password
      - wait_for:
          locator: {role: heading, name: Dashboard}
    effects: [establishes_session]
    success:
      origin: https://members.example.test
      path: /dashboard
      locator: {role: heading, name: Dashboard}
`)
	observedAt := "2026-08-20T00:00:00Z"
	digest := "sha256:" + strings.Repeat("a", 64)
	review := browserCaptureReviewCollection{Version: browserCaptureReviewVersion, Captures: []browserCaptureSafeReview{{
		Version: browserCaptureReviewVersion, ProfileID: profileID,
		AuthenticationTarget: "browser-authentication/" + profileID + "-auth.json", CapabilityTarget: "browser-profiles/" + profileID + ".json",
		EnvelopeSHA256: digest, ObservedAt: observedAt, Goal: "Reach dashboard",
		GoalPredicate: authorresult.GoalPredicate{Origin: "https://members.example.test", Path: "/dashboard", Context: "main", Role: "heading", Label: "Dashboard"},
		Origins:       []string{"https://members.example.test"}, Contexts: map[string]authorresult.Context{},
		Bounds:               authorresult.Bounds{NavigationTimeoutMS: 30000, TotalTimeoutMS: 600000, MaxRequests: 100, MaxResponseBytes: 1 << 20, MaxObservations: 100, MaxCandidates: 100, MaxOutputs: 16},
		TraceSteps:           1,
		AuthenticationReview: authorresult.Review{Schema: "browsertools.authenticated-profile-review.v1", Kind: "authentication", ProfileDigest: digest, AssessedAt: observedAt, Decisions: []string{"reviewed"}},
		CapabilityReview:     authorresult.Review{Schema: "browsertools.authenticated-profile-review.v1", Kind: "capability", ProfileDigest: digest, AssessedAt: observedAt, Decisions: []string{"reviewed"}},
		PrivateEnvelopeKept:  true,
	}}}
	safeReview, err := json.Marshal(review)
	if err != nil {
		t.Fatal(err)
	}
	return BrowserCaptureStage{ProfileID: profileID, Authentication: authentication, Capability: capability, SafeReview: safeReview}
}
