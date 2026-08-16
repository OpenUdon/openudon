package browserverify

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/OpenUdon/browsertools/profile"
)

func TestInspectAndValidateLiveSummary(t *testing.T) {
	prof := verificationProfile(t)
	report := validLiveReport(t, prof)
	path := writeVerificationReport(t, report)
	summary, err := Inspect(path, prof, verificationNow())
	if err != nil {
		t.Fatal(err)
	}
	if summary.ReportVersion != LiveCheckVersion || summary.Engine != "chromium" || !summary.OK || len(summary.Checks) != 3 {
		t.Fatalf("summary = %#v", summary)
	}
	if err := ValidateSummary(prof, summary, verificationNow()); err != nil {
		t.Fatal(err)
	}
	wantSource := sha256.Sum256(mustRead(t, path))
	if summary.SourceSHA256 != "sha256:"+hex.EncodeToString(wantSource[:]) {
		t.Fatalf("source digest = %q", summary.SourceSHA256)
	}
}

func TestInspectRejectsUnknownPrivateStaleAndMismatchedFacts(t *testing.T) {
	prof := verificationProfile(t)
	tests := []struct {
		name   string
		mutate func(map[string]any)
		want   string
	}{
		{name: "unknown", mutate: func(value map[string]any) { value["pageValue"] = "secret" }, want: "unknown field"},
		{name: "private rich version", mutate: func(value map[string]any) { value["version"] = "browsertools.private-rich-evidence.v1" }, want: "not a value-free"},
		{name: "guided authoring version", mutate: func(value map[string]any) { value["version"] = "browsertools.guided-authoring.v1" }, want: "not a value-free"},
		{name: "assisted authentication version", mutate: func(value map[string]any) { value["version"] = "browsertools.assisted-authentication.v1" }, want: "not a value-free"},
		{name: "doctor version", mutate: func(value map[string]any) { value["version"] = "browsertools.playwright-doctor.v1" }, want: "not a value-free"},
		{name: "stale", mutate: func(value map[string]any) { value["checkedAt"] = "2026-08-14T00:00:00Z" }, want: "stale"},
		{name: "origin", mutate: func(value map[string]any) { value["origin"] = "https://other.test" }, want: "exact canonical profile origin"},
		{name: "path", mutate: func(value map[string]any) {
			checks := value["checks"].([]any)
			checks[0].(map[string]any)["path"] = "actions.read_status.outputs.invented"
		}, want: "kind/path"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data, err := json.Marshal(validLiveReport(t, prof))
			if err != nil {
				t.Fatal(err)
			}
			var value map[string]any
			if err := json.Unmarshal(data, &value); err != nil {
				t.Fatal(err)
			}
			test.mutate(value)
			path := writeVerificationReport(t, value)
			if _, err := Inspect(path, prof, verificationNow()); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Inspect error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestInspectRejectsDuplicateAndMissingRequiredWireFields(t *testing.T) {
	prof := verificationProfile(t)
	report := validLiveReport(t, prof)
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	duplicate := bytes.Replace(data, []byte(`"version":"browsertools.live-check.v1"`), []byte(`"version":"browsertools.live-check.v1","version":"browsertools.live-check.v1"`), 1)
	path := filepath.Join(t.TempDir(), "duplicate.json")
	if err := os.WriteFile(path, duplicate, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Inspect(path, prof, verificationNow()); err == nil || !strings.Contains(err.Error(), "duplicate JSON field") {
		t.Fatalf("duplicate error = %v", err)
	}

	var object map[string]any
	if err := json.Unmarshal(data, &object); err != nil {
		t.Fatal(err)
	}
	delete(object, "ok")
	if _, err := Inspect(writeVerificationReport(t, object), prof, verificationNow()); err == nil || !strings.Contains(err.Error(), `required field "ok"`) {
		t.Fatalf("missing ok error = %v", err)
	}
	object["ok"] = true
	object["checks"] = nil
	if _, err := Inspect(writeVerificationReport(t, object), prof, verificationNow()); err == nil || !strings.Contains(err.Error(), `required field "checks"`) {
		t.Fatalf("null checks error = %v", err)
	}
}

func TestInspectPortabilityValidatesExactEnginesAndBaseline(t *testing.T) {
	prof := verificationProfile(t)
	live := validLiveReport(t, prof)
	report := portabilityReport{
		Version: PortabilityVersion, ProfileDigest: live.ProfileDigest, CheckedAt: live.CheckedAt,
		Origin: live.Origin, Actions: live.Actions, OK: true,
		Engines: []EngineResult{
			{Engine: "chromium", Status: "passed", Checks: cloneChecks(live.Checks)},
			{Engine: "firefox", Status: "passed", Checks: cloneChecks(live.Checks)},
		},
		ContractPressure: contractPressure(),
	}
	summary, err := Inspect(writeVerificationReport(t, report), prof, verificationNow())
	if err != nil {
		t.Fatal(err)
	}
	if summary.ReportVersion != PortabilityVersion || !summary.OK || len(summary.Engines) != 2 {
		t.Fatalf("summary = %#v", summary)
	}
	if err := ValidateSummary(prof, summary, verificationNow()); err != nil {
		t.Fatal(err)
	}

	invented := report
	invented.Engines = cloneEngines(report.Engines)
	// Navigation-wait match counts are value-free but are not success
	// predicates. Changing one still proves that the engine shapes differ.
	for index := range invented.Engines[1].Checks {
		if invented.Engines[1].Checks[index].Kind == probeNavigationWait {
			invented.Engines[1].Checks[index].Matches = 1
		}
	}
	if _, err := Inspect(writeVerificationReport(t, invented), prof, verificationNow()); err == nil || !strings.Contains(err.Error(), "invents portability success") {
		t.Fatalf("invented portability error = %v", err)
	}

	pressure := report
	pressure.ContractPressure = append([]contextPressure(nil), report.ContractPressure...)
	pressure.ContractPressure[0].Disposition = "public"
	if _, err := Inspect(writeVerificationReport(t, pressure), prof, verificationNow()); err == nil || !strings.Contains(err.Error(), "contractPressure") {
		t.Fatalf("pressure error = %v", err)
	}
}

func TestInspectPortabilityAcceptsExplicitUnavailableBaselineButRejectsMissingShape(t *testing.T) {
	prof := verificationProfile(t)
	live := validLiveReport(t, prof)
	report := portabilityReport{
		Version: PortabilityVersion, ProfileDigest: live.ProfileDigest, CheckedAt: live.CheckedAt,
		Origin: live.Origin, Actions: live.Actions, OK: false,
		Engines: []EngineResult{
			{Engine: "chromium", Status: "unavailable", Diagnostic: "engine_unavailable", Checks: []Check{}},
			{Engine: "firefox", Status: "failed", Diagnostic: "chromium_baseline_unavailable", Checks: cloneChecks(live.Checks)},
		},
		ContractPressure: contractPressure(),
	}
	if _, err := Inspect(writeVerificationReport(t, report), prof, verificationNow()); err != nil {
		t.Fatal(err)
	}

	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	var object map[string]any
	if err := json.Unmarshal(data, &object); err != nil {
		t.Fatal(err)
	}
	engines := object["engines"].([]any)
	delete(engines[0].(map[string]any), "checks")
	if _, err := Inspect(writeVerificationReport(t, object), prof, verificationNow()); err == nil || !strings.Contains(err.Error(), `required field "checks"`) {
		t.Fatalf("missing engine checks error = %v", err)
	}

	missingAlternate := report
	missingAlternate.Engines = missingAlternate.Engines[:1]
	if _, err := Inspect(writeVerificationReport(t, missingAlternate), prof, verificationNow()); err == nil || !strings.Contains(err.Error(), "one or two alternate") {
		t.Fatalf("missing alternate error = %v", err)
	}
}

func TestInspectAcceptsBrowsertoolsGenericOutputFailureShape(t *testing.T) {
	prof := verificationProfile(t)
	report := validLiveReport(t, prof)
	report.OK = false
	report.Checks[0] = Check{
		Kind: probeOutput, Path: "actions.read_status.outputs.status",
		Message: "read-only browser observation failed closed",
	}
	if _, err := Inspect(writeVerificationReport(t, report), prof, verificationNow()); err != nil {
		t.Fatal(err)
	}
}

func TestInspectRejectsSymlinkAndOversizedReports(t *testing.T) {
	prof := verificationProfile(t)
	realPath := writeVerificationReport(t, validLiveReport(t, prof))
	symlink := filepath.Join(t.TempDir(), "report.json")
	if err := os.Symlink(realPath, symlink); err != nil {
		t.Fatal(err)
	}
	if _, err := Inspect(symlink, prof, verificationNow()); err == nil || !strings.Contains(err.Error(), "non-symlink") {
		t.Fatalf("symlink error = %v", err)
	}
	oversized := filepath.Join(t.TempDir(), "large.json")
	if err := os.WriteFile(oversized, []byte(strings.Repeat("x", MaxReportBytes+1)), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Inspect(oversized, prof, verificationNow()); err == nil || !strings.Contains(err.Error(), "size") {
		t.Fatalf("oversized error = %v", err)
	}
}

func validLiveReport(t *testing.T, prof *profile.Profile) liveReport {
	t.Helper()
	digest, err := ProfileDigest(prof)
	if err != nil {
		t.Fatal(err)
	}
	return liveReport{
		Version: LiveCheckVersion, ProfileDigest: digest, CheckedAt: "2026-08-16T12:00:00Z",
		Origin: "https://example.test", Actions: []string{"read_status"}, OK: true,
		Checks: []Check{
			{Kind: probeOutput, Path: "actions.read_status.outputs.status", OK: true, Matches: 1, ExpectedType: profile.OutputString, ObservedType: profile.OutputString, Message: "declared output source and JSON type matched"},
			{Kind: probeLocator, Path: "actions.read_status.sequence[1].wait_for", OK: true, Matches: 1, Message: "declared accessibility locator resolved exactly once"},
			{Kind: probeNavigationWait, Path: "actions.read_status.sequence[2].wait_for.navigation", OK: true, Message: "declared navigation wait was reached without executing an action macro"},
		},
	}
}

func verificationProfile(t *testing.T) *profile.Profile {
	t.Helper()
	data := []byte(`{"profile":"uws.browser.1.5","info":{"title":"Status","origin":"https://example.test"},"observationKind":"accessibility_snapshot","evidence":{"learnedAt":"2026-08-15T00:00:00Z","source":"synthetic_fixture"},"confidence":"high","expiresAfter":"P30D","verification":{"lastVerifiedAt":"2026-08-15T00:00:00Z","successfulRuns":1},"actions":{"read_status":{"sequence":[{"navigate":"/member"},{"wait_for":{"role":"status","name":"Ready"}},{"wait_for":{"navigation":"load"}}],"outputs":{"status":{"type":"string","source":"a11y","locator":{"role":"status","name":"Ready"}}},"sideEffects":["read_only"],"confirmationPolicy":{"required":false}}}}`)
	prof, err := profile.ParseJSON(data)
	if err != nil {
		t.Fatal(err)
	}
	return prof
}

func verificationNow() time.Time {
	return time.Date(2026, 8, 16, 13, 0, 0, 0, time.UTC)
}

func writeVerificationReport(t *testing.T, value any) string {
	t.Helper()
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "report.json")
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
