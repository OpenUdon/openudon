package browserscenario

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/OpenUdon/browsertools/authprofile"
	"github.com/OpenUdon/browsertools/profile"
)

func TestEmbeddedScenarioCorpusIsCompleteAndStrict(t *testing.T) {
	now := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	manifests, err := LoadManifests(now)
	if err != nil {
		t.Fatal(err)
	}
	loopback, err := SelectManifests(manifests, SuiteLoopback, nil)
	if err != nil {
		t.Fatal(err)
	}
	public, err := SelectManifests(manifests, SuitePublic, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(loopback) != 21 || len(public) != 4 {
		t.Fatalf("scenario counts = loopback %d public %d", len(loopback), len(public))
	}
	wantChallenges := map[string]bool{"totp": false, "sms_otp": false, "email_otp": false, "voice_otp": false, "push": false, "push_number_match": false, "passkey": false, "security_key": false}
	for _, manifest := range loopback {
		if manifest.Authentication.ChallengeKind != "" {
			wantChallenges[manifest.Authentication.ChallengeKind] = true
		}
	}
	for kind, found := range wantChallenges {
		if !found {
			t.Fatalf("scenario corpus does not cover MFA kind %q", kind)
		}
	}
	wantPublic := []string{"books-to-scrape", "hacker-news", "quotes-to-scrape-js", "wikipedia-playwright"}
	for index, manifest := range public {
		if manifest.ID != wantPublic[index] {
			t.Fatalf("public scenario[%d] = %q", index, manifest.ID)
		}
	}
}

func TestScenarioSelectionAndManifestValidationFailClosed(t *testing.T) {
	now := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	manifests, err := LoadManifests(now)
	if err != nil {
		t.Fatal(err)
	}
	selected, err := SelectManifests(manifests, SuiteLoopback, []string{"password-main", "mfa-push"})
	if err != nil || len(selected) != 2 || selected[0].ID != "mfa-push" || selected[1].ID != "password-main" {
		t.Fatalf("selected = %#v, %v", selected, err)
	}
	for _, ids := range [][]string{{"missing"}, {"password-main", "password-main"}, {"../escape"}} {
		if _, err := SelectManifests(manifests, SuiteLoopback, ids); err == nil {
			t.Fatalf("selection %#v succeeded", ids)
		}
	}
	base := selected[1]
	tests := []struct {
		name   string
		mutate func(*Manifest)
	}{
		{"arbitrary target", func(value *Manifest) {
			value.Target = &PublicTarget{URL: "https://example.com", Origins: []string{"https://example.com"}}
		}},
		{"secret key", func(value *Manifest) { value.Outputs[0].Key = "access_token" }},
		{"unknown kind", func(value *Manifest) { value.Authentication.ChallengeKind = "guessed" }},
		{"unbounded outputs", func(value *Manifest) { value.Outputs = append(make([]Output, 18), value.Outputs...) }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			candidate := cloneManifest(t, base)
			test.mutate(&candidate)
			if err := ValidateManifest(candidate, now); err == nil {
				t.Fatal("invalid scenario succeeded")
			}
		})
	}
}

func TestPublicQuarantineIsBoundedAndExpires(t *testing.T) {
	now := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	manifests, err := LoadManifests(now)
	if err != nil {
		t.Fatal(err)
	}
	public, err := SelectManifests(manifests, SuitePublic, []string{"books-to-scrape"})
	if err != nil {
		t.Fatal(err)
	}
	manifest := public[0]
	manifest.Quarantine = &Quarantine{Since: "2026-08-16", Until: "2026-08-29", Reason: "upstream_markup_drift"}
	if err := ValidateManifest(manifest, now); err != nil {
		t.Fatal(err)
	}
	if err := ValidateManifest(manifest, time.Date(2026, 8, 30, 0, 0, 0, 0, time.UTC)); err == nil || !strings.Contains(err.Error(), "expired") {
		t.Fatalf("expired quarantine = %v", err)
	}
	manifest.Quarantine.Until = "2026-08-31"
	if err := ValidateManifest(manifest, now); err == nil {
		t.Fatal("overlong quarantine succeeded")
	}
}

func TestPublicProfilesAreCredentialFreePresenceOnlyContracts(t *testing.T) {
	now := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	manifests, err := LoadManifests(now)
	if err != nil {
		t.Fatal(err)
	}
	public, err := SelectManifests(manifests, SuitePublic, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, manifest := range public {
		t.Run(manifest.ID, func(t *testing.T) {
			prof, data, err := publicScenarioProfile(manifest, now)
			if err != nil {
				t.Fatal(err)
			}
			if prof.Schema != manifest.Expected.BrowserProfile || len(prof.Actions) != 1 || len(prof.Actions[publicScenarioAction].Outputs) != len(manifest.Probes) {
				t.Fatalf("capability profile = %#v", prof)
			}
			for key, output := range prof.Actions[publicScenarioAction].Outputs {
				if output.Type != profile.OutputBoolean || output.Source != profile.OutputA11y || output.Presence == nil || !*output.Presence || output.Locator == nil {
					t.Fatalf("output %s = %#v", key, output)
				}
			}
			if strings.Contains(string(data), "cookie") || strings.Contains(string(data), "storageState") {
				t.Fatalf("capability profile widened session state: %s", data)
			}

			authenticationData, err := publicScenarioAuthenticationProfile(manifest, now)
			if err != nil {
				t.Fatal(err)
			}
			authentication, err := authprofile.Parse(authenticationData)
			if err != nil {
				t.Fatal(err)
			}
			flow := authentication.Flows[publicScenarioFlow]
			if authentication.Profile != "uws.browser-authentication.1.0" || len(authentication.CredentialSlots) != 0 || len(authentication.Flows) != 1 || len(flow.Sequence) != 2 || len(flow.Effects) != 1 || flow.Effects[0] != "establishes_session" {
				t.Fatalf("authentication bootstrap = %#v", authentication)
			}
		})
	}
}

func TestPublicManifestRejectsUnboundedOrUnsafeProbeContracts(t *testing.T) {
	now := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	manifests, err := LoadManifests(now)
	if err != nil {
		t.Fatal(err)
	}
	public, err := SelectManifests(manifests, SuitePublic, []string{"books-to-scrape"})
	if err != nil {
		t.Fatal(err)
	}
	tests := []func(*Manifest){
		func(value *Manifest) { value.Probes[0].MaxMatches = 2 },
		func(value *Manifest) { value.Probes[0].Role = "document" },
		func(value *Manifest) { value.Probes[0].Name = "sk-proj-abcdefghijklmnopqrstuvwxyz1234567890" },
		func(value *Manifest) { value.Target.Origins = []string{"https://z.example", "https://a.example"} },
	}
	for index, mutate := range tests {
		candidate := cloneManifest(t, public[0])
		mutate(&candidate)
		if err := ValidateManifest(candidate, now); err == nil {
			t.Fatalf("invalid public contract %d succeeded", index)
		}
	}
}

func TestScenarioEnvironmentExcludesCredentialsAndRetainsNetworkProxy(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "must-not-cross")
	t.Setenv("SCENARIO_PASSWORD", "must-not-cross")
	t.Setenv("HTTPS_PROXY", "http://proxy.example:8080")
	values := scenarioEnvironment(nil)
	joined := strings.Join(values, "\n")
	if strings.Contains(joined, "OPENAI_API_KEY") || strings.Contains(joined, "SCENARIO_PASSWORD") || !strings.Contains(joined, "HTTPS_PROXY=http://proxy.example:8080") {
		t.Fatalf("scenario environment = %q", joined)
	}
}

func TestCompatibilityLockMatchesPublishedTypedBrowserRevisions(t *testing.T) {
	lock, err := LoadCompatibilityLock()
	if err != nil {
		t.Fatal(err)
	}
	if lock.Playwright != "1.62.1" || lock.Components[0].Name != "browserdriver" || lock.Components[3].Name != "uws" {
		t.Fatalf("lock = %#v", lock)
	}
}

func TestReportRoundTripTamperAndWireStrictness(t *testing.T) {
	report := sampleReport(t)
	root := t.TempDir()
	filename := filepath.Join(root, "report.json")
	if err := WriteReport(filename, report); err != nil {
		t.Fatal(err)
	}
	verified, err := VerifyReportFile(filename, true)
	if err != nil || verified.Status != StatusPass || verified.Summary.Passed != 1 {
		t.Fatalf("verified = %#v, %v", verified, err)
	}
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filename, append(data, ' '), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyReportFile(filename, true); err == nil || !strings.Contains(err.Error(), "digest") {
		t.Fatalf("tamper verification = %v", err)
	}
	if err := WriteReport(filename, report); err != nil {
		t.Fatal(err)
	}
	data, _ = os.ReadFile(filename)
	data = []byte(strings.Replace(string(data), `"engine": "chromium"`, `"engine": "chromium", "unknown": true`, 1))
	if err := os.WriteFile(filename, data, 0o644); err != nil {
		t.Fatal(err)
	}
	sum := sha256Line(data, filepath.Base(filename))
	if err := os.WriteFile(filename+".sha256", []byte(sum), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyReportFile(filename, false); err == nil {
		t.Fatal("unknown report field succeeded")
	}
}

func TestReportRejectsUnsafeClaimsAndUnprovenPass(t *testing.T) {
	report := sampleReport(t)
	report.ContainsPageContent = true
	if err := ValidateReport(report); err == nil {
		t.Fatal("page-content report succeeded")
	}
	report = sampleReport(t)
	report.Scenarios[0].Assertions = nil
	if err := ValidateReport(report); err == nil {
		t.Fatal("passing report without assertions succeeded")
	}
}

func sampleReport(t *testing.T) *Report {
	t.Helper()
	lock, err := LoadCompatibilityLock()
	if err != nil {
		t.Fatal(err)
	}
	components := map[string]LockedRevision{}
	for _, component := range lock.Components {
		components[component.Name] = component
	}
	return NewReport(SuiteLoopback, time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC), []RepositoryRevision{
		{Name: "openudon", Commit: "6a08b81317a852b9b8581c502eadbbdf591508e1"},
		{Name: "browsertools", Commit: components["browsertools"].Commit},
		{Name: "udon", Commit: components["udon"].Commit},
		{Name: "browserdriver", Commit: components["browserdriver"].Commit},
	}, []DependencyRevision{
		{Module: components["browsertools"].Module, Version: components["browsertools"].Version},
		{Module: components["uws"].Module, Version: components["uws"].Version},
	}, []ScenarioResult{{
		ID: "password-main", Status: StatusPass, Attempts: 1, Detail: "ok",
		Phases:     []PhaseResult{{ID: "fixture_ready", Status: StatusPass, Detail: "ok"}, {ID: "teardown", Status: StatusPass, Detail: "ok"}},
		Assertions: canonicalAssertions([]string{"private_material_absent", "author_session_v2"}),
	}})
}

func cloneManifest(t *testing.T, source Manifest) Manifest {
	t.Helper()
	data, err := json.Marshal(source)
	if err != nil {
		t.Fatal(err)
	}
	var result Manifest
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func sha256Line(data []byte, name string) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]) + "  " + name + "\n"
}
