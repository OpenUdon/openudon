package smokematrix

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestDefaultScenariosCoverBroadSmokeMatrix(t *testing.T) {
	scenarios := DefaultScenarios()
	if len(scenarios) < 8 {
		t.Fatalf("default scenarios = %d, want at least 8", len(scenarios))
	}
	var hasRequiredSlack bool
	for _, scenario := range scenarios {
		if scenario.ID == "slack-post" && scenario.RequiredLive {
			hasRequiredSlack = true
		}
		if strings.TrimSpace(scenario.Sentence) == "" {
			t.Fatalf("scenario %q missing natural-language sentence", scenario.ID)
		}
	}
	if !hasRequiredSlack {
		t.Fatalf("default scenarios must include required Slack live smoke")
	}
}

func TestRunDryRunWritesRedactedSummary(t *testing.T) {
	repoRoot := repoRootForTest(t)
	runRoot := filepath.Join(repoRoot, ".openudon-run", "test-smoke-dry-run-"+strconv.Itoa(os.Getpid()))
	out := filepath.Join(runRoot, "summary.json")
	t.Cleanup(func() {
		_ = os.RemoveAll(runRoot)
	})
	secret := "slack-secret-value-for-redaction"
	t.Setenv("UDON_CREDENTIAL_SLACK_BOT_TOKEN", secret)
	report, err := Run(context.Background(), Options{
		RepoRoot: repoRoot,
		WorkDir:  filepath.Join(runRoot, "work"),
		OutPath:  out,
		Mode:     ModeDryRun,
		Now:      fixedNow,
		Scenarios: []Scenario{{
			ID:         "runtime-only-render-test",
			Fixture:    "runtime-only-render",
			Sentence:   "Render a local audit note without calling an external API.",
			LiveKind:   "dry-run-only",
			Inputs:     map[string]any{"summary": "test summary"},
			Overlay:    "",
			DummyCreds: false,
			SensitiveKeys: []string{
				"UDON_CREDENTIAL_SLACK_BOT_TOKEN",
			},
		}},
	})
	if err != nil {
		t.Fatalf("Run dry-run returned error: %v; report=%#v", err, report)
	}
	if report.Status != StatusPass {
		t.Fatalf("status = %s, want pass", report.Status)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read summary: %v", err)
	}
	if strings.Contains(string(data), secret) {
		t.Fatalf("summary leaked secret value:\n%s", data)
	}
	var decoded Report
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("summary JSON invalid: %v", err)
	}
	if decoded.Version != ReportVersion || len(decoded.Scenarios) != 1 {
		t.Fatalf("unexpected decoded report: %#v", decoded)
	}
}

func TestRunLocalUdonSmokeUsesBuiltExecutorAndExpandsAsyncEvidence(t *testing.T) {
	repoRoot := repoRootForTest(t)
	runRoot := filepath.Join(repoRoot, ".openudon-run", "test-local-udon-smoke-"+strconv.Itoa(os.Getpid()))
	t.Cleanup(func() {
		_ = os.RemoveAll(runRoot)
	})
	report, err := RunLocalUdonSmoke(context.Background(), LocalUdonSmokeOptions{
		RepoRoot: repoRoot,
		UdonRepo: filepath.Join(runRoot, "fake-udon-repo"),
		WorkDir:  runRoot,
		OutPath:  filepath.Join(runRoot, "summary.json"),
		Now:      fixedNow,
		BuildCommand: func(_ context.Context, _ string, out string) error {
			if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(out, []byte("#!/usr/bin/env bash\nexit 0\n"), 0o755); err != nil {
				return err
			}
			return nil
		},
		RunCommand: func(_ context.Context, _ string, args ...string) error {
			reportPath := smokeArgValue(args, "--execution-report")
			data := `{"version":"udon.execution-report.v1","status":"success","started_at":"2026-04-29T12:00:00Z","finished_at":"2026-04-29T12:00:00Z","workflow_path":"workflow.uws.yaml","workflow_format":"uws-yaml","workdir":".","output_path":"output.hcl","output_digest":"sha256:` + strings.Repeat("a", 64) + `"}` + "\n"
			if err := os.MkdirAll(filepath.Dir(reportPath), 0o755); err != nil {
				return err
			}
			return os.WriteFile(reportPath, []byte(data), 0o600)
		},
	})
	if err != nil {
		t.Fatalf("RunLocalUdonSmoke returned error: %v; report=%#v", err, report)
	}
	if report.Status != StatusPass || len(report.Scenarios) != 1 {
		t.Fatalf("unexpected report: %#v", report)
	}
	evidencePath := filepath.FromSlash(report.Scenarios[0].RunEvidencePath)
	data, err := os.ReadFile(evidencePath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"mode": "internal-runner"`) || !strings.Contains(string(data), `executor-report.json`) {
		t.Fatalf("run evidence missing executor proof:\n%s", data)
	}
	asyncPath := filepath.Join(filepath.Dir(evidencePath), "async-evidence.json")
	asyncData, err := os.ReadFile(asyncPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(asyncData), "status_observation") || !strings.Contains(string(asyncData), "confirmation_read_observation") {
		t.Fatalf("async evidence missing expanded observations:\n%s", asyncData)
	}
}

func TestLiveRequiredMissingEnvFailsBeforeExecution(t *testing.T) {
	repoRoot := repoRootForTest(t)
	t.Cleanup(func() {
		_ = os.RemoveAll(filepath.Join(repoRoot, ".openudon-run", "test-smoke-missing-required"))
	})
	report, err := Run(context.Background(), Options{
		RepoRoot: repoRoot,
		WorkDir:  filepath.Join(repoRoot, ".openudon-run", "test-smoke-missing-required"),
		Mode:     ModeLive,
		Now:      fixedNow,
		Scenarios: []Scenario{{
			ID:           "required-live",
			Fixture:      "runtime-only-render",
			Sentence:     "Required live scenario.",
			LiveKind:     "external-provider",
			RequiredLive: true,
			RequiredEnv:  []string{"OPENUDON_TEST_REQUIRED_ENV"},
		}},
	})
	if err == nil {
		t.Fatalf("expected required missing env failure")
	}
	if report == nil || report.Status != StatusFail {
		t.Fatalf("report status = %#v, want fail", report)
	}
	if got := report.Scenarios[0].MissingEnv; len(got) != 1 || got[0] != "OPENUDON_TEST_REQUIRED_ENV" {
		t.Fatalf("missing env = %#v", got)
	}
}

func smokeArgValue(args []string, name string) string {
	for i, arg := range args {
		if arg == name && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

func TestLiveOptionalMissingEnvSkipsWithoutFailure(t *testing.T) {
	repoRoot := repoRootForTest(t)
	t.Cleanup(func() {
		_ = os.RemoveAll(filepath.Join(repoRoot, ".openudon-run", "test-smoke-missing-optional"))
	})
	report, err := Run(context.Background(), Options{
		RepoRoot: repoRoot,
		WorkDir:  filepath.Join(repoRoot, ".openudon-run", "test-smoke-missing-optional"),
		Mode:     ModeLive,
		Now:      fixedNow,
		Scenarios: []Scenario{{
			ID:          "optional-live",
			Fixture:     "runtime-only-render",
			Sentence:    "Optional live scenario.",
			LiveKind:    "external-provider",
			RequiredEnv: []string{"OPENUDON_TEST_OPTIONAL_ENV"},
		}},
	})
	if err != nil {
		t.Fatalf("optional missing env should not fail: %v", err)
	}
	if report.Status != StatusPass || report.Scenarios[0].Status != StatusSkippedMissingEnv {
		t.Fatalf("unexpected report: %#v", report)
	}
}

func TestScenarioFilterSelectsRequestedIDs(t *testing.T) {
	scenarios := filterScenarios(DefaultScenarios(), []string{"weather-read", "inventory-api-key"})
	if len(scenarios) != 2 {
		t.Fatalf("filtered scenarios = %d, want 2", len(scenarios))
	}
	if scenarios[0].ID != "weather-read" || scenarios[1].ID != "inventory-api-key" {
		t.Fatalf("unexpected filter order: %#v", scenarios)
	}
}

func TestSlackAliasesCoverDeclaredAndExecutorBindings(t *testing.T) {
	var slack Scenario
	for _, scenario := range DefaultScenarios() {
		if scenario.ID == "slack-post" {
			slack = scenario
			break
		}
	}
	for _, name := range []string{"UDON_CREDENTIAL_SLACK", "UDON_CREDENTIAL_SLACKBEARER"} {
		if slack.EnvAliases[name] != "UDON_CREDENTIAL_SLACK_BOT_TOKEN" {
			t.Fatalf("Slack alias %s = %q", name, slack.EnvAliases[name])
		}
	}
}

func TestSlackLiveOverlayUsesExplicitAPIPostMessagePath(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "openapi"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "workflows"), 0o755); err != nil {
		t.Fatal(err)
	}
	projectPath := filepath.Join(dir, "project.md")
	project := "## Safety and Approval Boundary\n\n- No credentials are required for this sandbox fixture.\n"
	if err := os.WriteFile(projectPath, []byte(project), 0o644); err != nil {
		t.Fatal(err)
	}
	err := applyOverlay(dir, ModeLive, Scenario{Overlay: "slack-live"}, "")
	if err != nil {
		t.Fatalf("apply Slack overlay: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "openapi", "slack.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, "url: https://slack.com\n") {
		t.Fatalf("Slack overlay should keep server host-only:\n%s", text)
	}
	if !strings.Contains(text, "/api/chat.postMessage:") {
		t.Fatalf("Slack overlay should use explicit API path:\n%s", text)
	}
	intent, err := os.ReadFile(filepath.Join(dir, "workflows", "intent.hcl"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(intent), "render_audit_log") {
		t.Fatalf("Slack live overlay should not require a runtime render helper:\n%s", intent)
	}
	if !strings.Contains(string(intent), `from = "post_message.received_body"`) {
		t.Fatalf("Slack live overlay should return provider response metadata:\n%s", intent)
	}
}

func TestWeatherUsesOpenWeatherCredentialName(t *testing.T) {
	var weather Scenario
	for _, scenario := range DefaultScenarios() {
		if scenario.ID == "weather-read" {
			weather = scenario
			break
		}
	}
	if len(weather.RequiredEnv) != 1 || weather.RequiredEnv[0] != "UDON_CREDENTIAL_OPENWEATHERAPIKEY" {
		t.Fatalf("weather required env = %#v", weather.RequiredEnv)
	}
	if weather.EnvAliases["UDON_CREDENTIAL_INPUTS_APPID"] != "UDON_CREDENTIAL_OPENWEATHERAPIKEY" {
		t.Fatalf("weather compatibility alias = %#v", weather.EnvAliases)
	}
}

func TestSanitizeDetailRedactsAndLimitsOutput(t *testing.T) {
	t.Setenv("OPENUDON_TEST_SECRET", "secret-value")
	detail := "prefix secret-value " + strings.Repeat("x", maxScenarioDetailLength*2)
	got := sanitizeDetail(detail, []string{"OPENUDON_TEST_SECRET"})
	if strings.Contains(got, "secret-value") {
		t.Fatalf("sanitizeDetail leaked secret: %q", got)
	}
	if !strings.Contains(got, "[redacted]") {
		t.Fatalf("sanitizeDetail did not include redaction marker: %q", got)
	}
	if len(got) > maxScenarioDetailLength {
		t.Fatalf("sanitizeDetail length = %d, want <= %d", len(got), maxScenarioDetailLength)
	}
	if !strings.HasSuffix(got, truncatedDetailSuffix) {
		t.Fatalf("sanitizeDetail suffix = %q, want truncation suffix", got)
	}
}

func repoRootForTest(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func fixedNow() time.Time {
	return time.Date(2026, 5, 26, 0, 0, 0, 0, time.UTC)
}
