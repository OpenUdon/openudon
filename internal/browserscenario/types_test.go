package browserscenario

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/OpenUdon/browsertools/authprofile"
	"github.com/OpenUdon/browsertools/profile"
	"github.com/OpenUdon/openudon/internal/synthesize"
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
	journey, err := SelectManifests(manifests, SuiteJourney, nil)
	if err != nil {
		t.Fatal(err)
	}
	public, err := SelectManifests(manifests, SuitePublic, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(loopback) != 23 || len(journey) != 8 || len(public) != 4 {
		t.Fatalf("scenario counts = loopback %d journey %d public %d", len(loopback), len(journey), len(public))
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
	wantJourney := []string{"catalog-pagination", "catalog-search-filter", "order-structured-read", "parameter-contract-rejected", "record-update-ambiguous", "record-update-approved", "record-update-unapproved", "session-lifecycle"}
	for index, manifest := range journey {
		if manifest.ID != wantJourney[index] {
			t.Fatalf("journey scenario[%d] = %q", index, manifest.ID)
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

func TestJourneyManifestOutcomesAreFixedByCase(t *testing.T) {
	now := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	manifests, err := LoadManifests(now)
	if err != nil {
		t.Fatal(err)
	}
	journeys, err := SelectManifests(manifests, SuiteJourney, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, manifest := range journeys {
		candidate := cloneManifest(t, manifest)
		if candidate.Expected.Replay == "pass" {
			candidate.Expected.Replay = "rejected"
			candidate.Expected.FailureCode = "invalid_parameters"
		} else {
			candidate.Expected.Replay = "pass"
			candidate.Expected.FailureCode = ""
		}
		if err := ValidateManifest(candidate, now); err == nil {
			t.Fatalf("journey %s accepted a different expected outcome", manifest.ID)
		}
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
	t.Setenv("GOWORK", "private.work")
	values := scenarioEnvironment(nil)
	joined := strings.Join(values, "\n")
	if strings.Contains(joined, "OPENAI_API_KEY") || strings.Contains(joined, "SCENARIO_PASSWORD") || strings.Contains(joined, "HTTPS_PROXY") || strings.Contains(joined, "GOWORK") {
		t.Fatalf("scenario environment = %q", joined)
	}
	qualifiedValues := scenarioEnvironment(qualificationGoBuildEnvironment())
	valuesByName := map[string]string{}
	for _, value := range qualifiedValues {
		name, item, _ := strings.Cut(value, "=")
		valuesByName[name] = item
	}
	for name, want := range map[string]string{
		"GOENV": "off", "GOPROXY": "off", "GOTOOLCHAIN": "go1.26.6", "GOWORK": "off",
	} {
		if valuesByName[name] != want {
			t.Fatalf("scenario environment %s = %q, want %q", name, valuesByName[name], want)
		}
	}
}

func TestCompatibilityLockMatchesExactTypedBrowserRevisions(t *testing.T) {
	lock, err := LoadCompatibilityLock()
	if err != nil {
		t.Fatal(err)
	}
	if lock.Playwright != "1.62.1" || lock.Chromium != "151.0.7922.34" || lock.Components[0].Name != "browserdriver" || lock.Components[3].Name != "uws" {
		t.Fatalf("lock = %#v", lock)
	}
	components := map[string]LockedRevision{}
	for _, component := range lock.Components {
		components[component.Name] = component
	}
	if components["browserdriver"].Commit != "a97b1aed6ea69a30591815da8ca07ac9e7c87623" ||
		components["udon"].Commit != "e0e6559e839bed788201cf0c55a3eb296d375987" ||
		components["browsertools"].Commit != "75fd5c3ab81f904243f8c2650c61ba1cd8c00540" ||
		components["uws"].Commit != "9e676eaa469e9168225a7dcee75eb309e3499637" {
		t.Fatalf("qualification component pins = %#v", components)
	}
	buildLock, err := LoadQualificationBuildInputLock(lock)
	if err != nil {
		t.Fatal(err)
	}
	buildComponents := map[string]QualificationBuildInput{}
	for _, component := range buildLock.Components {
		buildComponents[component.Name] = component
	}
	if buildComponents["browsertools"].Commit != "d26f2982db352619d7a7f6563add802b56e10824" ||
		buildComponents["uws"].Commit != "895aa4546067e25f9dd525b1356abf1945d223b4" {
		t.Fatalf("Udon qualification module pins = %#v", buildComponents)
	}
}

func TestQualificationBuildInputLockRejectsUnboundAndDirtyReplacements(t *testing.T) {
	parent := t.TempDir()
	browsertoolsCommit := initializeQualificationRepository(t, filepath.Join(parent, "browsertools"))
	uwsCommit := initializeQualificationRepository(t, filepath.Join(parent, "uws"))
	udonRoot := filepath.Join(parent, "udon")
	if err := os.Mkdir(udonRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	goMod := "module example.test/udon\n\ngo 1.26.6\n\n" +
		"replace github.com/OpenUdon/browsertools => ../browsertools\n" +
		"replace github.com/OpenUdon/uws => ../uws\n"
	if err := os.WriteFile(filepath.Join(udonRoot, "go.mod"), []byte(goMod), 0o600); err != nil {
		t.Fatal(err)
	}
	lock := QualificationBuildInputLock{
		Version: qualificationBuildInputLockVersion,
		Components: []QualificationBuildInput{
			{Name: "browsertools", Module: "github.com/OpenUdon/browsertools", Replacement: "../browsertools", Commit: browsertoolsCommit},
			{Name: "uws", Module: "github.com/OpenUdon/uws", Replacement: "../uws", Commit: uwsCommit},
		},
	}
	compatibility, err := LoadCompatibilityLock()
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateQualificationBuildInputLock(lock, compatibility); err != nil {
		t.Fatal(err)
	}
	if err := validateQualificationBuildInputs(context.Background(), udonRoot, lock); err != nil {
		t.Fatal(err)
	}

	drifted := lock
	drifted.Components = append([]QualificationBuildInput(nil), lock.Components...)
	drifted.Components[0].Commit = strings.Repeat("a", 40)
	if err := validateQualificationBuildInputs(context.Background(), udonRoot, drifted); err == nil || !strings.Contains(err.Error(), "revision") {
		t.Fatalf("drifted build input error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(parent, "browsertools", "untracked"), []byte("drift"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := validateQualificationBuildInputs(context.Background(), udonRoot, lock); err == nil || !strings.Contains(err.Error(), "dirty") {
		t.Fatalf("dirty build input error = %v", err)
	}
	if err := os.Remove(filepath.Join(parent, "browsertools", "untracked")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(parent, "browsertools", "ignored"), []byte("unbound"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := validateQualificationBuildInputs(context.Background(), udonRoot, lock); err == nil || !strings.Contains(err.Error(), "dirty") {
		t.Fatalf("ignored build input error = %v", err)
	}
	if err := os.Remove(filepath.Join(parent, "browsertools", "ignored")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(udonRoot, "go.mod"), []byte(goMod+"replace example.test/unbound => ../unbound\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := validateQualificationBuildInputs(context.Background(), udonRoot, lock); err == nil || !strings.Contains(err.Error(), "do not match") {
		t.Fatalf("unbound replacement error = %v", err)
	}
}

func TestQualificationBaselineIsCommittedAndBuildable(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join("..", ".."))
	destination := filepath.Join(t.TempDir(), "baseline")
	if err := copyQualificationBaseline(destination, repoRoot); err != nil {
		t.Fatal(err)
	}
	if _, err := synthesize.Build(context.Background(), synthesize.Options{ExampleDir: destination}); err != nil {
		t.Fatalf("build qualification baseline: %v", err)
	}
	probe := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	probe.Dir = repoRoot
	if err := probe.Run(); err != nil {
		return
	}
	for _, relative := range qualificationBaselineRequiredPaths {
		path := filepath.ToSlash(filepath.Join(qualificationBaselineScope, relative))
		command := exec.Command("git", "ls-files", "--error-unmatch", "--", path)
		command.Dir = repoRoot
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("qualification baseline path %s is not committed: %v: %s", path, err, strings.TrimSpace(string(output)))
		}
	}
}

func initializeQualificationRepository(t *testing.T, root string) string {
	t.Helper()
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("qualification fixture\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte("ignored\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	commands := [][]string{
		{"init", "--quiet"},
		{"add", "README.md", ".gitignore"},
		{"-c", "user.name=OpenUdon Qualification", "-c", "user.email=qualification@example.invalid", "commit", "--quiet", "-m", "fixture"},
	}
	for _, args := range commands {
		command := exec.Command("git", args...)
		command.Dir = root
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, strings.TrimSpace(string(output)))
		}
	}
	command := exec.Command("git", "rev-parse", "HEAD")
	command.Dir = root
	output, err := command.Output()
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(output))
}

func TestCompatibilityLockRejectsDirtyOrDriftedSibling(t *testing.T) {
	lock, err := LoadCompatibilityLock()
	if err != nil {
		t.Fatal(err)
	}
	states := map[string]RepositoryState{}
	for _, component := range lock.Components {
		states[component.Name] = RepositoryState{Commit: component.Commit}
	}
	if err := ValidateRepositoryStates(lock, states); err != nil {
		t.Fatal(err)
	}
	dirty := states["browserdriver"]
	dirty.Dirty = true
	states["browserdriver"] = dirty
	if err := ValidateRepositoryStates(lock, states); err == nil || !strings.Contains(err.Error(), "dirty") {
		t.Fatalf("dirty sibling error = %v", err)
	}
	dirty.Dirty = false
	dirty.Commit = strings.Repeat("a", 40)
	states["browserdriver"] = dirty
	if err := ValidateRepositoryStates(lock, states); err == nil || !strings.Contains(err.Error(), "compatibility lock") {
		t.Fatalf("drifted sibling error = %v", err)
	}
}

func TestCompatibilityLockValidatesOpenUdonAndBrowsertoolsModulePins(t *testing.T) {
	lock, err := LoadCompatibilityLock()
	if err != nil {
		t.Fatal(err)
	}
	locked := map[string]LockedRevision{}
	for _, component := range lock.Components {
		locked[component.Name] = component
	}
	openudonRoot := t.TempDir()
	browsertoolsRoot := t.TempDir()
	write := func(root, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	openudonMod := fmt.Sprintf("module example.test/openudon\n\nrequire (\n\t%s %s\n\t%s %s\n)\n",
		locked["browsertools"].Module, locked["browsertools"].Version,
		locked["uws"].Module, locked["uws"].Version)
	browsertoolsMod := fmt.Sprintf("module example.test/browsertools\n\nrequire %s %s\n", locked["uws"].Module, locked["uws"].Version)
	write(openudonRoot, openudonMod)
	write(browsertoolsRoot, browsertoolsMod)
	if err := ValidateGoModulePins(openudonRoot, browsertoolsRoot, lock); err != nil {
		t.Fatal(err)
	}
	write(browsertoolsRoot, strings.Replace(browsertoolsMod, locked["uws"].Version, "v0.0.0-20000101000000-aaaaaaaaaaaa", 1))
	if err := ValidateGoModulePins(openudonRoot, browsertoolsRoot, lock); err == nil || !strings.Contains(err.Error(), "browsertools") {
		t.Fatalf("Browsertools UWS pin error = %v", err)
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
	if err := WriteReport(filename, report); err != nil {
		t.Fatal(err)
	}
	data, _ = os.ReadFile(filename)
	data = []byte(strings.Replace(string(data), `"engine": "chromium"`, `"engine": "chromium", "engine": "chromium"`, 1))
	if err := os.WriteFile(filename, data, 0o644); err != nil {
		t.Fatal(err)
	}
	sum = sha256Line(data, filepath.Base(filename))
	if err := os.WriteFile(filename+".sha256", []byte(sum), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyReportFile(filename, false); err == nil || !strings.Contains(err.Error(), "duplicate JSON object key") {
		t.Fatalf("duplicate report field verification = %v", err)
	}
}

func TestAllSkippedSuiteIsNotRunAndCannotPassReleaseVerification(t *testing.T) {
	report := sampleReport(t)
	report.Scenarios[0] = ScenarioResult{
		ID: "password-main", Status: StatusSkipped, Attempts: 1, Detail: "dependency_unavailable",
		Phases: []PhaseResult{{ID: "fixture_ready", Status: StatusSkipped, Detail: "dependency_unavailable"}},
	}
	report = NewReport(report.Suite, time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC), report.Repositories, report.Dependencies, report.Scenarios)
	if report.Status != StatusNotRun || report.Summary.Passed != 0 || report.Summary.Skipped != 1 {
		t.Fatalf("all-skipped report = %#v", report)
	}
	path := filepath.Join(t.TempDir(), "not-run.json")
	if err := WriteReport(path, report); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyReportFile(path, false); err != nil {
		t.Fatalf("structural not_run verification failed: %v", err)
	}
	if _, err := VerifyReportFile(path, true); err == nil || !strings.Contains(err.Error(), StatusNotRun) {
		t.Fatalf("release verification error = %v", err)
	}
}

func TestScenarioReportRejectsDirtyOpenUdonRoot(t *testing.T) {
	report := sampleReport(t)
	report.Repositories[0].Dirty = true
	if err := ValidateReport(report); err == nil || !strings.Contains(err.Error(), "openudon is dirty") {
		t.Fatalf("dirty OpenUdon report error = %v", err)
	}
}

func TestStrictJSONRejectsDuplicateKeysAtEveryObjectDepth(t *testing.T) {
	for _, data := range []string{
		`{"field": 1, "field": 2}`,
		`{"outer": {"field": 1, "\u0066ield": 2}}`,
		`{"items": [{"field": 1, "field": 2}]}`,
	} {
		var target any
		if err := decodeStrict([]byte(data), &target); err == nil || err.Error() != "duplicate JSON object key" {
			t.Fatalf("duplicate input %s = %v", data, err)
		}
	}
	var target any
	if err := decodeStrict([]byte(`{"left": {"field": 1}, "right": {"field": 2}}`), &target); err != nil {
		t.Fatalf("independent object keys rejected: %v", err)
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

func TestJourneyReportUsesDedicatedWireVersion(t *testing.T) {
	report := sampleReport(t)
	report.Suite = SuiteJourney
	report.Version = JourneyReportVersion
	report.HeadedAuthoring = false
	report.Scenarios[0] = ScenarioResult{
		ID: "catalog-search-filter", Status: StatusPass, Attempts: 1, Detail: "ok",
		Phases:     []PhaseResult{{ID: "fixture_ready", Status: StatusPass, Detail: "ok"}, {ID: "teardown", Status: StatusPass, Detail: "ok"}},
		Assertions: []string{"guided_authoring_v1"},
	}
	if err := ValidateReport(report); err != nil {
		t.Fatal(err)
	}
	report.Version = ReportVersion
	if err := ValidateReport(report); err == nil {
		t.Fatal("journey report accepted the legacy wire version")
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
		{Name: "uws", Commit: components["uws"].Commit},
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
