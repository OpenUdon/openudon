package icot

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/OpenUdon/apitools"
	evalpkg "github.com/OpenUdon/openudon/internal/eval"
	"github.com/OpenUdon/openudon/internal/icot/elicitor"
	"github.com/OpenUdon/openudon/internal/projectwizard"
	"github.com/OpenUdon/openudon/internal/synthesize"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
	runner "github.com/OpenUdon/openudon/internal/workflowintent"
)

func TestMainPreviewEOFCancelsWithoutWriting(t *testing.T) {
	example := filepath.Join(t.TempDir(), "guided")
	var stdout, stderr bytes.Buffer
	code := Main([]string{"--example", example, "--no-llm"}, strings.NewReader(testProjectInput(false)), &stdout, &stderr)
	if code == 0 {
		t.Fatalf("Main succeeded with EOF at preview\nstdout:\n%s\nstderr:\n%s", stdout.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join(example, "project.md")); !os.IsNotExist(err) {
		t.Fatalf("EOF wrote project.md or unexpected stat error: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join(example, "workflows", "intent.hcl")); !os.IsNotExist(err) {
		t.Fatalf("EOF wrote intent.hcl or unexpected stat error: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join(example, ".icot", "session.yaml")); err != nil {
		t.Fatalf("EOF should preserve draft session: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "unexpected EOF") {
		t.Fatalf("stderr missing unexpected EOF:\n%s", stderr.String())
	}
}

func TestBackupProjectCreatesDistinctBackups(t *testing.T) {
	dir := t.TempDir()
	projectPath := filepath.Join(dir, "project.md")
	if err := os.WriteFile(projectPath, []byte("old\n"), 0o644); err != nil {
		t.Fatalf("write old project: %v", err)
	}
	if err := backupProject(projectPath); err != nil {
		t.Fatalf("first backup failed: %v", err)
	}
	if err := os.WriteFile(projectPath, []byte("new\n"), 0o644); err != nil {
		t.Fatalf("write new project: %v", err)
	}
	if err := backupProject(projectPath); err != nil {
		t.Fatalf("second backup failed: %v", err)
	}
	backups, err := filepath.Glob(filepath.Join(dir, "project.md.bak.*"))
	if err != nil {
		t.Fatalf("glob backups: %v", err)
	}
	if len(backups) != 2 {
		t.Fatalf("backup count = %d, want 2: %v", len(backups), backups)
	}
	contents := map[string]bool{}
	for _, path := range backups {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read backup %s: %v", path, err)
		}
		contents[string(data)] = true
	}
	if !contents["old\n"] || !contents["new\n"] {
		t.Fatalf("backup contents = %#v, want old and new", contents)
	}
}

func TestLoadSeedSessionUsesReferenceIntent(t *testing.T) {
	seedDir, err := filepath.Abs(filepath.Join("..", "..", "examples", "eval", "slack-message-audit-log"))
	if err != nil {
		t.Fatalf("resolve seed dir: %v", err)
	}
	intent, err := parseSeedIntent(seedDir)
	if err != nil {
		t.Fatalf("parseSeedIntent failed: %v", err)
	}
	if len(intent.Steps) == 0 || intent.Steps[0].Operation != "postMessage" {
		t.Fatalf("parsed seed intent missing postMessage: %#v", intent.Steps)
	}
	session, source, err := authorSession("", seedDir, filepath.Join(t.TempDir(), "seeded"), false, false)
	if err != nil {
		t.Fatalf("authorSession failed: %v", err)
	}
	if source != seedSourceSeed {
		t.Fatalf("source = %q, want %q", source, seedSourceSeed)
	}
	if !completeSession(session) {
		_, err := elicitor.RenderArtifacts(session)
		t.Fatalf("seed session is incomplete: %v", err)
	}
	if got := session.Intent.OpenAPI; got != "openapi/slack.yaml" {
		t.Fatalf("intent openapi = %q, want openapi/slack.yaml", got)
	}
	if len(session.Intent.Steps) == 0 || session.Intent.Steps[0].Operation != "postMessage" {
		t.Fatalf("seed steps missing postMessage: %#v", session.Intent.Steps)
	}
}

func TestAgentJSONNeedsInputWithoutWriting(t *testing.T) {
	example := filepath.Join(t.TempDir(), "agent")
	var stdout, stderr bytes.Buffer
	code := Main([]string{"--example", example, "--agent", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("agent returned code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	var report authorReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal report: %v\n%s", err, stdout.String())
	}
	if report.Status != statusNeedsInput || report.FailureFamily != failureAmbiguousUserIntent {
		t.Fatalf("report = %#v", report)
	}
	if _, err := os.Stat(filepath.Join(example, "project.md")); !os.IsNotExist(err) {
		t.Fatalf("agent wrote project.md unexpectedly: %v", err)
	}
}

func TestAgentJSONCompleteSessionWritesArtifacts(t *testing.T) {
	dir := t.TempDir()
	example := filepath.Join(dir, "agent-complete")
	sessionPath := writeCompleteRuntimeSession(t, dir)
	var stdout, stderr bytes.Buffer
	code := Main([]string{"--example", example, "--answers", sessionPath, "--agent", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("agent complete returned code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	var report authorReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal report: %v\n%s", err, stdout.String())
	}
	if report.Status != statusPass || report.GeneratedIntent == "" {
		t.Fatalf("report = %#v", report)
	}
	if _, err := os.Stat(filepath.Join(example, "workflows", "intent.hcl")); err != nil {
		t.Fatalf("intent not written: %v", err)
	}
}

func TestAgentJSONBlocksRenderableLowDecisionEvidence(t *testing.T) {
	dir := t.TempDir()
	example := filepath.Join(dir, "agent-blocked")
	project := projectwizard.Answers{
		ProjectName:     "Blocked Agent",
		Goal:            "Render a report",
		SideEffectScope: projectwizard.SideEffectSandboxOnly,
		Safety:          "Sandbox proof runs only",
		Fallback:        "Stop if rendering fails",
	}
	session := elicitor.SessionFromIntent(testIntent("blocked_agent", "Render a report", "render_report"), project)
	session.DecisionEvidence = []elicitor.DecisionEvidence{{
		Stage:      "output_selection",
		Slot:       "intent.outputs.ticket",
		Value:      "ticket=render_report.received_body",
		Source:     "llm",
		Confidence: "low",
	}}
	sessionPath := writeSessionJSON(t, dir, session)
	var stdout, stderr bytes.Buffer
	code := Main([]string{"--example", example, "--answers", sessionPath, "--agent", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("agent returned code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	var report authorReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal report: %v\n%s", err, stdout.String())
	}
	if report.Status != statusNeedsInput || report.TopIssue == nil || report.TopIssue.Code != "low_confidence_decision" {
		t.Fatalf("report = %#v", report)
	}
	if _, err := os.Stat(filepath.Join(example, "workflows", "intent.hcl")); !os.IsNotExist(err) {
		t.Fatalf("agent wrote intent despite readiness blocker: %v", err)
	}
}

func TestAgentJSONLoadsCompleteDraft(t *testing.T) {
	example := filepath.Join(t.TempDir(), "agent-draft")
	draftPath := writeCompleteDraft(t, example)
	var stdout, stderr bytes.Buffer
	code := Main([]string{"--example", example, "--agent", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("agent draft returned code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	var report authorReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal report: %v\n%s", err, stdout.String())
	}
	if report.Status != statusPass || report.GeneratedIntent == "" {
		t.Fatalf("report = %#v", report)
	}
	if _, err := os.Stat(filepath.Join(example, "workflows", "intent.hcl")); err != nil {
		t.Fatalf("intent not written from draft: %v", err)
	}
	if _, err := os.Stat(draftPath); !os.IsNotExist(err) {
		t.Fatalf("draft not deleted after agent save: %v", err)
	}
}

func TestAgentReportWriteFailureReturnsError(t *testing.T) {
	dir := t.TempDir()
	example := filepath.Join(dir, "agent-report-fail")
	sessionPath := writeCompleteRuntimeSession(t, dir)
	notDir := filepath.Join(dir, "not-dir")
	if err := os.WriteFile(notDir, []byte("file\n"), 0o644); err != nil {
		t.Fatalf("write not-dir file: %v", err)
	}
	var stdout, stderr bytes.Buffer
	code := Main([]string{"--example", example, "--answers", sessionPath, "--agent", "--json", "--report", filepath.Join(notDir, "report.json")}, strings.NewReader(""), &stdout, &stderr)
	if code != 1 {
		t.Fatalf("agent code = %d, want 1\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	if strings.TrimSpace(stderr.String()) == "" {
		t.Fatalf("stderr missing report write error")
	}
}

func TestEvalReferenceSeedBuildMatrix(t *testing.T) {
	fixtureRoot, err := filepath.Abs(filepath.Join("..", "..", "examples", "eval"))
	if err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(fixtureRoot)
	if err != nil {
		t.Fatal(err)
	}
	outRoot := t.TempDir()
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		t.Run(name, func(t *testing.T) {
			seedDir := filepath.Join(fixtureRoot, name)
			policy, err := evalpkg.ReadReferencePolicy(filepath.Join(seedDir, "reference", "policy.json"))
			if err != nil {
				t.Fatalf("read reference policy: %v", err)
			}
			if policy.SeedBuild == nil || policy.SeedBuild.Expected == "" || policy.SeedBuild.Class == "" {
				t.Fatalf("reference policy must declare seed_build.expected and seed_build.class")
			}
			exampleDir := filepath.Join(outRoot, name)
			var stdout, stderr bytes.Buffer
			code := Main([]string{"--example", exampleDir, "--from-example", seedDir, "--no-llm", "--no-transcript"}, strings.NewReader(""), &stdout, &stderr)
			if code != 0 {
				assertSeedBuildOutcome(t, policy, "icot_fail", nil, strings.TrimSpace(stderr.String()))
				return
			}
			_, report, err := synthesize.PackageFromIntent(context.Background(), synthesize.Options{ExampleDir: exampleDir})
			if err != nil {
				assertSeedBuildOutcome(t, policy, "build_fail", []string{"build:error"}, err.Error())
				return
			}
			if report == nil || !report.Passed() {
				assertSeedBuildOutcome(t, policy, "build_fail", failedQualityCodes(report), qualityFailureDetails(report))
				return
			}
			assertSeedBuildOutcome(t, policy, "pass", nil, "")
		})
	}
}

func TestLintJSONReport(t *testing.T) {
	example, err := filepath.Abs(filepath.Join("..", "..", "examples", "eval", "runtime-only-render"))
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := Main([]string{"lint", "--example", example, "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("lint json returned code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	var report lintReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal lint report: %v\n%s", err, stdout.String())
	}
	if report.Version != lintReportVersion || report.Status != statusPass || len(report.ProjectChecks) == 0 {
		t.Fatalf("lint report = %#v", report)
	}
}

func TestScorecardSingleFixture(t *testing.T) {
	outDir := filepath.Join(t.TempDir(), "scorecard")
	var stdout, stderr bytes.Buffer
	code := Main([]string{"scorecard", "--root", filepath.Join("..", "..", "examples", "eval"), "--name", "runtime-only-render", "--out", outDir}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("scorecard returned code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	data, err := os.ReadFile(filepath.Join(outDir, "scorecard.json"))
	if err != nil {
		t.Fatalf("read scorecard: %v", err)
	}
	var report scorecardReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatalf("unmarshal scorecard: %v\n%s", err, data)
	}
	if report.Status != statusPass || report.Summary.Total != 1 || report.Results[0].ObservedOutcome != "pass" {
		t.Fatalf("scorecard report = %#v", report)
	}
	if report.RunID == "" || report.GeneratedAt == "" || report.PromptVersion == "" || report.ReadinessClassifierVersion == "" || report.ScorecardCommand == "" {
		t.Fatalf("scorecard provenance missing: %#v", report)
	}
	if _, err := os.Stat(filepath.Join(outDir, "scorecard.json.sha256")); err != nil {
		t.Fatalf("scorecard digest sidecar missing: %v", err)
	}
}

func TestScorecardIncludesAuthoringVariants(t *testing.T) {
	outDir := filepath.Join(t.TempDir(), "scorecard-variants")
	var stdout, stderr bytes.Buffer
	code := Main([]string{"scorecard", "--root", filepath.Join("..", "..", "examples", "eval"), "--name", "slack-message-audit-log", "--include-variants", "--out", outDir}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("scorecard variants returned code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	data, err := os.ReadFile(filepath.Join(outDir, "scorecard.json"))
	if err != nil {
		t.Fatalf("read scorecard: %v", err)
	}
	var report scorecardReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatalf("unmarshal scorecard: %v\n%s", err, data)
	}
	if report.Summary.ByProviderFamily["slack"] == 0 || report.Summary.ByVariantClass["unsafe-negative"] == 0 {
		t.Fatalf("summary missing provider/variant grouping: %#v", report.Summary)
	}
	var sawInlineSecret, sawMissingChannel bool
	for _, result := range report.Results {
		if result.VariantID == "missing-channel" {
			sawMissingChannel = true
			if result.ObservedOutcome != statusNeedsInput || result.FailureFamily != failureBadRequestMapping || result.ExpectedTopIssueCode != "missing_required_request_values" || result.TopIssueCode != result.ExpectedTopIssueCode || result.TopIssueSlot != result.ExpectedTopIssueSlot || result.SuggestedAnswer == "" {
				t.Fatalf("missing-channel result = %#v", result)
			}
		}
		if result.VariantID == "inline-token" {
			sawInlineSecret = true
			if result.ObservedOutcome != statusNeedsInput || result.FailureFamily != failureCredentialBindingGap || result.GeneratedIntent != "" {
				t.Fatalf("inline-token result = %#v", result)
			}
		}
	}
	if !sawInlineSecret {
		t.Fatalf("inline-token variant missing from results: %#v", report.Results)
	}
	if !sawMissingChannel {
		t.Fatalf("missing-channel variant missing from results: %#v", report.Results)
	}
	if report.Summary.MissingDetailFalsePass != 0 || report.Summary.UnsafeFalsePass != 0 || report.Summary.NeedsInputDiagnosticGap != 0 {
		t.Fatalf("false pass counters should be zero for passing scorecard: %#v", report.Summary)
	}
}

func TestReportVerifyScorecard(t *testing.T) {
	outDir := filepath.Join(t.TempDir(), "scorecard")
	var scoreOut, scoreErr bytes.Buffer
	code := Main([]string{"scorecard", "--root", filepath.Join("..", "..", "examples", "eval"), "--name", "runtime-only-render", "--out", outDir}, strings.NewReader(""), &scoreOut, &scoreErr)
	if code != 0 {
		t.Fatalf("scorecard returned code %d\nstdout:\n%s\nstderr:\n%s", code, scoreOut.String(), scoreErr.String())
	}
	var stdout, stderr bytes.Buffer
	code = Main([]string{"report", "verify", "--file", filepath.Join(outDir, "scorecard.json")}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 || !strings.Contains(stdout.String(), scorecardReportVersion) {
		t.Fatalf("report verify code=%d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
}

func TestReportVerifyRejectsDigestMismatch(t *testing.T) {
	outDir := filepath.Join(t.TempDir(), "scorecard")
	var scoreOut, scoreErr bytes.Buffer
	code := Main([]string{"scorecard", "--root", filepath.Join("..", "..", "examples", "eval"), "--name", "runtime-only-render", "--out", outDir}, strings.NewReader(""), &scoreOut, &scoreErr)
	if code != 0 {
		t.Fatalf("scorecard returned code %d\nstdout:\n%s\nstderr:\n%s", code, scoreOut.String(), scoreErr.String())
	}
	path := filepath.Join(outDir, "scorecard.json")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatalf("open scorecard: %v", err)
	}
	if _, err := f.WriteString("\n"); err != nil {
		_ = f.Close()
		t.Fatalf("tamper scorecard: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close scorecard: %v", err)
	}
	var stdout, stderr bytes.Buffer
	code = Main([]string{"report", "verify", "--file", path}, strings.NewReader(""), &stdout, &stderr)
	if code != 1 || !strings.Contains(stderr.String(), "digest mismatch") {
		t.Fatalf("report verify code=%d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
}

func TestReportVerifyRejectsVariantTopIssueMismatch(t *testing.T) {
	outDir := filepath.Join(t.TempDir(), "scorecard")
	var scoreOut, scoreErr bytes.Buffer
	code := Main([]string{"scorecard", "--root", filepath.Join("..", "..", "examples", "eval"), "--name", "slack-message-audit-log", "--include-variants", "--out", outDir}, strings.NewReader(""), &scoreOut, &scoreErr)
	if code != 0 {
		t.Fatalf("scorecard returned code %d\nstdout:\n%s\nstderr:\n%s", code, scoreOut.String(), scoreErr.String())
	}
	path := filepath.Join(outDir, "scorecard.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read scorecard: %v", err)
	}
	var report scorecardReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatalf("unmarshal scorecard: %v", err)
	}
	for i := range report.Results {
		if report.Results[i].VariantID == "missing-channel" {
			report.Results[i].ExpectedTopIssueCode = "missing_runtime_inputs"
			break
		}
	}
	if err := writeJSONReportWithDigest(path, report); err != nil {
		t.Fatalf("rewrite tampered scorecard: %v", err)
	}
	var stdout, stderr bytes.Buffer
	code = Main([]string{"report", "verify", "--file", path}, strings.NewReader(""), &stdout, &stderr)
	if code != 1 || !strings.Contains(stderr.String(), "does not match expected/observed outcome") {
		t.Fatalf("report verify code=%d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
}

func TestScorecardCountsVariantFalsePasses(t *testing.T) {
	report := scorecardReport{
		Status: statusPass,
		Summary: scorecardSummary{
			ByClass:                 map[string]int{},
			ByFailureFamily:         map[string]int{},
			ByObservedOutcome:       map[string]int{},
			ByProviderFamily:        map[string]int{},
			ByProviderFailureFamily: map[string]map[string]int{},
			ByVariantClass:          map[string]int{},
		},
	}
	appendScorecardResult(&report, scorecardResult{
		Name:            "fixture#missing",
		Kind:            "authoring_variant",
		Class:           "missing-detail",
		ExpectedOutcome: statusNeedsInput,
		ObservedOutcome: statusPass,
	}, io.Discard)
	appendScorecardResult(&report, scorecardResult{
		Name:            "fixture#unsafe",
		Kind:            "authoring_variant",
		Class:           "unsafe-negative",
		ExpectedOutcome: statusNeedsInput,
		ObservedOutcome: statusPass,
	}, io.Discard)
	if report.Summary.MissingDetailFalsePass != 1 || report.Summary.UnsafeFalsePass != 1 || report.Status != statusFail {
		t.Fatalf("false pass counters = %#v status=%s", report.Summary, report.Status)
	}
}

func TestScorecardCountsNeedsInputDiagnosticGaps(t *testing.T) {
	report := scorecardReport{
		Status: statusPass,
		Summary: scorecardSummary{
			ByClass:                 map[string]int{},
			ByFailureFamily:         map[string]int{},
			ByObservedOutcome:       map[string]int{},
			ByProviderFamily:        map[string]int{},
			ByProviderFailureFamily: map[string]map[string]int{},
			ByVariantClass:          map[string]int{},
		},
	}
	appendScorecardResult(&report, scorecardResult{
		Name:                  "fixture#missing",
		Kind:                  "authoring_variant",
		Class:                 "missing-detail",
		ExpectedOutcome:       statusNeedsInput,
		ExpectedFailureFamily: failureBadRequestMapping,
		ObservedOutcome:       statusNeedsInput,
		FailureFamily:         failureBadRequestMapping,
		Passed:                true,
	}, io.Discard)
	if report.Summary.NeedsInputDiagnosticGap != 1 || report.Summary.Passed != 0 || report.Summary.Failed != 1 || report.Status != statusFail {
		t.Fatalf("diagnostic gap summary = %#v status=%s", report.Summary, report.Status)
	}
}

func TestMissingDetailVariantTopIssues(t *testing.T) {
	root := filepath.Join("..", "..", "examples", "eval")
	cases := []struct {
		fixture string
		variant string
		family  string
		code    string
		slot    string
	}{
		{
			fixture: "slack-message-audit-log",
			variant: "missing-channel",
			family:  failureBadRequestMapping,
			code:    "missing_required_request_values",
			slot:    "steps.post_message.with",
		},
		{
			fixture: "gmail-send-audit-receipt",
			variant: "missing-recipient",
			family:  failureBadRequestMapping,
			code:    "missing_required_request_values",
			slot:    "steps.send_message.with",
		},
		{
			fixture: "weather-toronto",
			variant: "missing-location",
			family:  failureBadRequestMapping,
			code:    "missing_required_request_values",
			slot:    "steps.get_coordinates.with",
		},
		{
			fixture: "weather-toronto-gmail",
			variant: "missing-recipient",
			family:  failureAmbiguousUserIntent,
			code:    "missing_runtime_inputs",
			slot:    "intent.inputs",
		},
	}
	outRoot := t.TempDir()
	for _, tc := range cases {
		t.Run(tc.fixture+"/"+tc.variant, func(t *testing.T) {
			fixture := filepath.Join(root, tc.fixture)
			variants, err := evalpkg.ReadAuthoringVariants(filepath.Join(fixture, "reference", "authoring-variants.json"))
			if err != nil {
				t.Fatalf("read variants: %v", err)
			}
			var variant evalpkg.AuthoringVariant
			for _, candidate := range variants.Variants {
				if candidate.ID == tc.variant {
					variant = candidate
					break
				}
			}
			if variant.ID == "" {
				t.Fatalf("variant %q not found", tc.variant)
			}
			if variant.ExpectedTopIssueCode != tc.code || variant.ExpectedTopIssueSlot != tc.slot {
				t.Fatalf("variant expectations = %s/%s, want %s/%s", variant.ExpectedTopIssueCode, variant.ExpectedTopIssueSlot, tc.code, tc.slot)
			}
			result := runNeedsInputVariant(fixture, tc.fixture, variant, filepath.Join(outRoot, safeScorecardName(tc.fixture+"_"+tc.variant)))
			if result.ObservedOutcome != statusNeedsInput || result.FailureFamily != tc.family || result.TopIssueCode != tc.code || result.TopIssueSlot != tc.slot || result.SuggestedAnswer == "" {
				t.Fatalf("missing-detail result = %#v", result)
			}
		})
	}
}

func TestVariantsValidateCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Main([]string{"variants", "validate", "--root", filepath.Join("..", "..", "examples", "eval"), "--name", "slack-message-audit-log", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("variants validate returned code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	var report variantsValidationReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal variants report: %v\n%s", err, stdout.String())
	}
	if report.Version != variantsValidationReportVersion || report.Status != statusPass || len(report.Results) != 1 || report.Results[0].VariantCount == 0 {
		t.Fatalf("variants report = %#v", report)
	}
}

func TestVariantsCoverageCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Main([]string{"variants", "coverage", "--root", filepath.Join("..", "..", "examples", "eval"), "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("variants coverage returned code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	var report variantsCoverageReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal coverage report: %v\n%s", err, stdout.String())
	}
	if report.Version != variantsCoverageReportVersion || report.Status != statusPass || len(report.Results) == 0 {
		t.Fatalf("coverage report = %#v", report)
	}
	for _, result := range report.Results {
		if result.ProviderFamily == "slack" {
			if result.Positive == 0 || result.MissingDetail == 0 || result.UnsafeNegative == 0 {
				t.Fatalf("slack coverage = %#v", result)
			}
			return
		}
	}
	t.Fatalf("slack coverage missing: %#v", report.Results)
}

func TestVariantsCoverageRejectsMissingClass(t *testing.T) {
	root := t.TempDir()
	fixture := filepath.Join(root, "partial")
	if err := copyScorecardTree(filepath.Join("..", "..", "examples", "eval", "slack-message-audit-log"), fixture); err != nil {
		t.Fatalf("copy fixture: %v", err)
	}
	path := filepath.Join(fixture, "reference", "authoring-variants.json")
	data := `{
  "version": "openudon.icot-authoring-variants.v1",
  "provider_families": ["slack"],
  "variants": [
    {
      "id": "direct-post",
      "brief": "Post to Slack.",
      "class": "positive",
      "expected_outcome": "pass"
    },
    {
      "id": "missing-channel",
      "brief": "Post to the team in Slack.",
      "class": "missing-detail",
      "expected_outcome": "needs_input",
      "expected_failure_family": "bad_request_mapping",
      "expected_top_issue_code": "missing_required_request_values",
      "expected_top_issue_slot": "steps.post_message.with",
      "seed_from_reference": true,
      "clear_slots": ["steps.post_message.with.channel"]
    }
  ]
}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write variants: %v", err)
	}
	var stdout, stderr bytes.Buffer
	code := Main([]string{"variants", "coverage", "--root", root, "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != 1 {
		t.Fatalf("variants coverage code = %d, want 1\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	var report variantsCoverageReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal coverage report: %v\n%s", err, stdout.String())
	}
	if report.Status != statusFail || len(report.Results) != 1 || !strings.Contains(strings.Join(report.Results[0].Errors, "\n"), "missing unsafe-negative variant") {
		t.Fatalf("coverage report = %#v", report)
	}
}

func TestVariantsValidateRejectsUnknownClearSlot(t *testing.T) {
	root := t.TempDir()
	fixture := filepath.Join(root, "bad")
	if err := copyScorecardTree(filepath.Join("..", "..", "examples", "eval", "slack-message-audit-log"), fixture); err != nil {
		t.Fatalf("copy fixture: %v", err)
	}
	path := filepath.Join(fixture, "reference", "authoring-variants.json")
	data := `{
  "version": "openudon.icot-authoring-variants.v1",
  "variants": [
    {
      "id": "bad-clear",
      "brief": "Send to Slack.",
      "class": "missing-detail",
      "expected_outcome": "needs_input",
      "expected_failure_family": "bad_request_mapping",
      "expected_top_issue_code": "missing_required_request_values",
      "expected_top_issue_slot": "steps.post_message.with",
      "seed_from_reference": true,
      "clear_slots": ["steps.post_message.with.not_a_field"]
    }
  ]
}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write variants: %v", err)
	}
	var stdout, stderr bytes.Buffer
	code := Main([]string{"variants", "validate", "--root", root, "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != 1 {
		t.Fatalf("variants validate code = %d, want 1\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	var report variantsValidationReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal variants report: %v\n%s", err, stdout.String())
	}
	if report.Status != statusFail || len(report.Results) != 1 || !strings.Contains(strings.Join(report.Results[0].Errors, "\n"), "unknown request field") {
		t.Fatalf("variants report = %#v", report)
	}
}

func TestVariantsValidateRejectsMissingExpectedTopIssue(t *testing.T) {
	root := t.TempDir()
	fixture := filepath.Join(root, "bad")
	if err := copyScorecardTree(filepath.Join("..", "..", "examples", "eval", "slack-message-audit-log"), fixture); err != nil {
		t.Fatalf("copy fixture: %v", err)
	}
	path := filepath.Join(fixture, "reference", "authoring-variants.json")
	data := `{
  "version": "openudon.icot-authoring-variants.v1",
  "variants": [
    {
      "id": "missing-diagnostics",
      "brief": "Send to Slack.",
      "class": "missing-detail",
      "expected_outcome": "needs_input",
      "expected_failure_family": "bad_request_mapping",
      "seed_from_reference": true,
      "clear_slots": ["steps.post_message.with.channel"]
    }
  ]
}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write variants: %v", err)
	}
	var stdout, stderr bytes.Buffer
	code := Main([]string{"variants", "validate", "--root", root, "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != 1 {
		t.Fatalf("variants validate code = %d, want 1\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	var report variantsValidationReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal variants report: %v\n%s", err, stdout.String())
	}
	errors := strings.Join(report.Results[0].Errors, "\n")
	if report.Status != statusFail || !strings.Contains(errors, "expected_top_issue_code") || !strings.Contains(errors, "expected_top_issue_slot") {
		t.Fatalf("variants report = %#v", report)
	}
}

func TestScorecardVariantOutcomeRequiresExpectedTopIssue(t *testing.T) {
	result := scorecardResult{
		ExpectedOutcome:       statusNeedsInput,
		ExpectedFailureFamily: failureBadRequestMapping,
		ExpectedTopIssueCode:  "missing_required_request_values",
		ExpectedTopIssueSlot:  "steps.post_message.with",
		ObservedOutcome:       statusNeedsInput,
		FailureFamily:         failureBadRequestMapping,
		TopIssueCode:          "missing_runtime_inputs",
		TopIssueSlot:          "intent.inputs",
	}
	if scorecardVariantOutcomeMatches(result) {
		t.Fatalf("mismatched top issue passed: %#v", result)
	}
	result.TopIssueCode = result.ExpectedTopIssueCode
	result.TopIssueSlot = result.ExpectedTopIssueSlot
	if !scorecardVariantOutcomeMatches(result) {
		t.Fatalf("matching top issue failed: %#v", result)
	}
}

func TestFailureFamilyClassifiesReadinessCodes(t *testing.T) {
	cases := map[string]string{
		"invented_request_field":          failureBadRequestMapping,
		"invalid_request_body_path":       failureBadRequestMapping,
		"incompatible_request_value_type": failureBadRequestMapping,
		"undeclared_credential_reference": failureCredentialBindingGap,
		"missing_required_request_values": failureBadRequestMapping,
		"missing_runtime_inputs":          failureAmbiguousUserIntent,
		"missing_api_doc":                 failureMissingAPISource,
		"missing_operation":               failureMissingOperation,
		"missing_credential_bindings":     failureCredentialBindingGap,
		"missing_side_effect_policy":      failureSideEffectPolicyGap,
	}
	for code, want := range cases {
		if got := failureFamilyForReadiness(code); got != want {
			t.Fatalf("failureFamilyForReadiness(%q) = %q, want %q", code, got, want)
		}
	}
}

func TestAuthoringEvalWithFakeExtractor(t *testing.T) {
	previous := newAuthoringEvalExtractor
	newAuthoringEvalExtractor = func(provider, model string, temperature float64, calls *[]replayLLMCall) (elicitor.Extractor, string, string, error) {
		return fakeAuthoringEvalExtractor{calls: calls}, "fake", "fake-model", nil
	}
	defer func() { newAuthoringEvalExtractor = previous }()

	outDir := filepath.Join(t.TempDir(), "authoring-eval")
	var stdout, stderr bytes.Buffer
	code := Main([]string{"authoring-eval", "--root", filepath.Join("..", "..", "examples", "eval"), "--name", "runtime-only-render", "--provider", "fake", "--model", "fake-model", "--out", outDir}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("authoring-eval returned code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	data, err := os.ReadFile(filepath.Join(outDir, "authoring-eval.json"))
	if err != nil {
		t.Fatalf("read authoring eval report: %v", err)
	}
	var report authoringEvalReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatalf("unmarshal authoring eval report: %v\n%s", err, data)
	}
	if report.Version != authoringEvalReportVersion || report.Status != statusPass || report.Summary.Total != 1 {
		t.Fatalf("authoring eval report = %#v", report)
	}
	if report.RunID == "" || report.GeneratedAt == "" || report.PromptVersion == "" || report.ReadinessClassifierVersion == "" || report.AuthoringEvalCommand == "" {
		t.Fatalf("authoring eval provenance missing: %#v", report)
	}
	if _, err := os.Stat(filepath.Join(outDir, "authoring-eval.json.sha256")); err != nil {
		t.Fatalf("authoring eval digest sidecar missing: %v", err)
	}
	result := report.Results[0]
	if result.Provider != "fake" || result.Model != "fake-model" || result.LLMCallCount == 0 || result.GeneratedIntent == "" {
		t.Fatalf("authoring eval result = %#v", result)
	}
	var verifyOut, verifyErr bytes.Buffer
	code = Main([]string{"report", "verify", "--file", filepath.Join(outDir, "authoring-eval.json")}, strings.NewReader(""), &verifyOut, &verifyErr)
	if code != 0 || !strings.Contains(verifyOut.String(), authoringEvalReportVersion) {
		t.Fatalf("authoring eval report verify code=%d\nstdout:\n%s\nstderr:\n%s", code, verifyOut.String(), verifyErr.String())
	}
}

func TestAuthoringEvalClassifiesProviderTimeout(t *testing.T) {
	previous := newAuthoringEvalExtractor
	newAuthoringEvalExtractor = func(provider, model string, temperature float64, calls *[]replayLLMCall) (elicitor.Extractor, string, string, error) {
		return nil, "fake", "fake-model", context.DeadlineExceeded
	}
	defer func() { newAuthoringEvalExtractor = previous }()

	outDir := filepath.Join(t.TempDir(), "authoring-eval")
	var stdout, stderr bytes.Buffer
	code := Main([]string{"authoring-eval", "--root", filepath.Join("..", "..", "examples", "eval"), "--name", "runtime-only-render", "--provider", "fake", "--model", "fake-model", "--out", outDir}, strings.NewReader(""), &stdout, &stderr)
	if code != 1 {
		t.Fatalf("authoring-eval code = %d, want 1\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	data, err := os.ReadFile(filepath.Join(outDir, "authoring-eval.json"))
	if err != nil {
		t.Fatalf("read authoring eval report: %v", err)
	}
	var report authoringEvalReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatalf("unmarshal authoring eval report: %v\n%s", err, data)
	}
	if report.Summary.ByFailureCategory[authoringEvalProviderTimeout] != 1 || len(report.Results) != 1 || report.Results[0].FailureCategory != authoringEvalProviderTimeout {
		t.Fatalf("authoring eval report = %#v", report)
	}
}

func TestAuthoringEvalScansGeneratedArtifactsForCredentials(t *testing.T) {
	previous := newAuthoringEvalExtractor
	newAuthoringEvalExtractor = func(provider, model string, temperature float64, calls *[]replayLLMCall) (elicitor.Extractor, string, string, error) {
		return fakeSecretAuthoringEvalExtractor{calls: calls}, "fake", "fake-model", nil
	}
	defer func() { newAuthoringEvalExtractor = previous }()

	outDir := filepath.Join(t.TempDir(), "authoring-eval")
	var stdout, stderr bytes.Buffer
	code := Main([]string{"authoring-eval", "--root", filepath.Join("..", "..", "examples", "eval"), "--name", "runtime-only-render", "--provider", "fake", "--model", "fake-model", "--out", outDir}, strings.NewReader(""), &stdout, &stderr)
	if code != 1 {
		t.Fatalf("authoring-eval code = %d, want 1\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	data, err := os.ReadFile(filepath.Join(outDir, "authoring-eval.json"))
	if err != nil {
		t.Fatalf("read authoring eval report: %v", err)
	}
	var report authoringEvalReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatalf("unmarshal authoring eval report: %v\n%s", err, data)
	}
	if report.Status != statusFail || len(report.Results) != 1 {
		t.Fatalf("authoring eval report = %#v", report)
	}
	result := report.Results[0]
	if result.CredentialScanStatus != statusFail || result.FailureFamily != failureCredentialBindingGap || result.FailureCategory != authoringEvalCredentialScanFail || !containsString(result.FailureCodes, "authoring_eval.literal_credential") || len(result.CredentialDiagnostics) == 0 {
		t.Fatalf("authoring eval result = %#v", result)
	}
}

func TestAuthoringEvalReportRedactsCredentialLikeJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "authoring-eval.json")
	redacted, err := writeAuthoringEvalReportFile(path, authoringEvalReport{
		Version:                    authoringEvalReportVersion,
		Status:                     statusPass,
		Root:                       "examples/eval",
		OutDir:                     filepath.Dir(path),
		RunID:                      "test-run",
		GeneratedAt:                "2026-05-27T00:00:00Z",
		PromptVersion:              elicitor.PromptVersion,
		ReadinessClassifierVersion: readinessClassifierVersion,
		AuthoringEvalCommand:       "icot authoring-eval --provider fake",
		Summary:                    authoringEvalSummary{Total: 1, Passed: 1},
		Results: []authoringEvalResult{{
			Name:            "bad-report",
			ObservedOutcome: statusPass,
			Brief:           `operator pasted api_key = "sk-proj-abcdefghijklmnopqrstuvwxyz1234567890"`,
			Passed:          true,
		}},
	})
	if err != nil {
		t.Fatalf("write report: %v", err)
	}
	if !redacted {
		t.Fatalf("report was not redacted")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	if strings.Contains(string(data), "sk-proj-abcdefghijklmnopqrstuvwxyz1234567890") {
		t.Fatalf("redacted report leaked credential-like value:\n%s", data)
	}
	var report authoringEvalReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatalf("unmarshal report: %v\n%s", err, data)
	}
	if report.Status != statusFail || len(report.Results) != 1 || report.Results[0].FailureFamily != failureCredentialBindingGap || report.Results[0].FailureCategory != authoringEvalCredentialScanFail {
		t.Fatalf("redacted report = %#v", report)
	}
	if _, err := os.Stat(path + ".sha256"); err != nil {
		t.Fatalf("redacted authoring eval digest sidecar missing: %v", err)
	}
}

func TestRepairDryRunJSON(t *testing.T) {
	example := filepath.Join(t.TempDir(), "repair")
	var setupOut, setupErr bytes.Buffer
	code := Main([]string{"--example", example, "--from-example", filepath.Join("..", "..", "examples", "eval", "runtime-only-render"), "--no-llm", "--no-transcript"}, strings.NewReader(""), &setupOut, &setupErr)
	if code != 0 {
		t.Fatalf("setup failed code %d\nstdout:\n%s\nstderr:\n%s", code, setupOut.String(), setupErr.String())
	}
	var stdout, stderr bytes.Buffer
	code = Main([]string{"repair", "--example", example, "--dry-run", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("repair dry-run returned code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	var report repairReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal repair report: %v\n%s", err, stdout.String())
	}
	if report.Status != statusDryRun || !report.DryRun {
		t.Fatalf("repair report = %#v", report)
	}
}

func TestRepairAddsDependsOnFromStepReference(t *testing.T) {
	example := filepath.Join(t.TempDir(), "repair-deps")
	writeRepairDependencyExample(t, example)
	before, err := rollout.ParseIntentFile(filepath.Join(example, "workflows", "intent.hcl"))
	if err != nil {
		t.Fatalf("parse initial intent: %v", err)
	}
	initialSend := findTestStep(before.Steps, "send_report")
	if initialSend == nil || initialSend.With["body"] != "render_report.received_body" || len(initialSend.DependsOn) != 0 {
		t.Fatalf("initial send_report = %#v", initialSend)
	}
	var stdout, stderr bytes.Buffer
	code := Main([]string{"repair", "--example", example, "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("repair returned code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	var report repairReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal repair report: %v\n%s", err, stdout.String())
	}
	if report.Status != statusPass || !repairReportHasSlot(report, "steps.send_report.depends_on") {
		t.Fatalf("repair report = %#v", report)
	}
	intent, err := rollout.ParseIntentFile(filepath.Join(example, "workflows", "intent.hcl"))
	if err != nil {
		t.Fatalf("parse repaired intent: %v", err)
	}
	send := findTestStep(intent.Steps, "send_report")
	if send == nil || len(send.DependsOn) != 1 || send.DependsOn[0] != "render_report" {
		t.Fatalf("send_report depends_on = %#v", send)
	}
}

type fakeAuthoringEvalExtractor struct {
	calls *[]replayLLMCall
}

type fakeSecretAuthoringEvalExtractor struct {
	calls *[]replayLLMCall
}

func (f fakeAuthoringEvalExtractor) Kickoff(context.Context, string) (elicitor.Session, error) {
	if f.calls != nil {
		*f.calls = append(*f.calls, replayLLMCall{Kind: "fake_kickoff", Response: "runtime_only_render"})
	}
	return runtimeOnlyRenderSession(), nil
}

func (fakeAuthoringEvalExtractor) CatalogPlan(context.Context, elicitor.CatalogPlanRequest) (elicitor.CatalogPlanResponse, error) {
	return elicitor.CatalogPlanResponse{}, nil
}

func (fakeAuthoringEvalExtractor) RequestMappings(context.Context, elicitor.RequestMappingRequest) (elicitor.RequestMappingResponse, error) {
	return elicitor.RequestMappingResponse{}, nil
}

func (fakeAuthoringEvalExtractor) ReviewDraft(context.Context, elicitor.DraftReviewRequest) (elicitor.DraftReviewResponse, error) {
	return elicitor.DraftReviewResponse{}, nil
}

func (f fakeAuthoringEvalExtractor) Draft(context.Context, elicitor.DraftRequest) (elicitor.Session, error) {
	if f.calls != nil {
		*f.calls = append(*f.calls, replayLLMCall{Kind: "fake_draft", Response: "runtime_only_render"})
	}
	return runtimeOnlyRenderSession(), nil
}

func (fakeAuthoringEvalExtractor) Refine(_ context.Context, session elicitor.Session) (elicitor.Session, error) {
	if !elicitor.LooksLikeSession(session) {
		return runtimeOnlyRenderSession(), nil
	}
	return session, nil
}

func (fakeAuthoringEvalExtractor) Disambiguate(context.Context, string, []elicitor.APIDocument) ([]string, error) {
	return nil, nil
}

func (f fakeSecretAuthoringEvalExtractor) Kickoff(context.Context, string) (elicitor.Session, error) {
	if f.calls != nil {
		*f.calls = append(*f.calls, replayLLMCall{Kind: "fake_kickoff", Response: "runtime_only_render_secret"})
	}
	return runtimeOnlyRenderSecretSession(), nil
}

func (fakeSecretAuthoringEvalExtractor) CatalogPlan(context.Context, elicitor.CatalogPlanRequest) (elicitor.CatalogPlanResponse, error) {
	return elicitor.CatalogPlanResponse{}, nil
}

func (fakeSecretAuthoringEvalExtractor) RequestMappings(context.Context, elicitor.RequestMappingRequest) (elicitor.RequestMappingResponse, error) {
	return elicitor.RequestMappingResponse{}, nil
}

func (fakeSecretAuthoringEvalExtractor) ReviewDraft(context.Context, elicitor.DraftReviewRequest) (elicitor.DraftReviewResponse, error) {
	return elicitor.DraftReviewResponse{}, nil
}

func (f fakeSecretAuthoringEvalExtractor) Draft(context.Context, elicitor.DraftRequest) (elicitor.Session, error) {
	if f.calls != nil {
		*f.calls = append(*f.calls, replayLLMCall{Kind: "fake_draft", Response: "runtime_only_render_secret"})
	}
	return runtimeOnlyRenderSecretSession(), nil
}

func (fakeSecretAuthoringEvalExtractor) Refine(_ context.Context, session elicitor.Session) (elicitor.Session, error) {
	if !elicitor.LooksLikeSession(session) {
		return runtimeOnlyRenderSecretSession(), nil
	}
	return session, nil
}

func (fakeSecretAuthoringEvalExtractor) Disambiguate(context.Context, string, []elicitor.APIDocument) ([]string, error) {
	return nil, nil
}

func runtimeOnlyRenderSession() elicitor.Session {
	project := projectwizard.Answers{
		ProjectName:     "Runtime Only Render",
		Goal:            "Render a local summary report.",
		Inputs:          "summary: required string",
		Outputs:         "Rendered report body.",
		SideEffectScope: projectwizard.SideEffectReadOnly,
		Safety:          "Generate and validate artifacts only.",
		Fallback:        "Stop if no approved function runtime exists.",
	}
	return elicitor.SessionFromIntent(&rollout.Intent{
		Workflow: &rollout.WorkflowMeta{
			Name:        "runtime_only_render",
			Description: "Render a local summary report.",
		},
		Inputs: []*rollout.Input{{
			Name:     "summary",
			Type:     "string",
			Required: true,
		}},
		Steps: []*rollout.Step{{
			Name: "render_report",
			Type: "fnct",
			Do:   "Render the summary report.",
			With: map[string]string{"summary": "inputs.summary"},
		}},
		Outputs: []*rollout.Output{{
			Name: "report",
			From: "render_report.received_body",
		}},
	}, project)
}

func runtimeOnlyRenderSecretSession() elicitor.Session {
	session := runtimeOnlyRenderSession()
	session.Project.Fallback = `Stop if the model suggests api_key = "sk-proj-abcdefghijklmnopqrstuvwxyz1234567890".`
	session.Fallback = session.Project.Fallback
	session.FallbackSet = true
	return session
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func TestRepairRequestMappingRejectsInventedField(t *testing.T) {
	session := elicitor.Session{Intent: rollout.Intent{
		OpenAPI: "openapi/support.yaml",
		Workflow: &rollout.WorkflowMeta{
			Name:        "support_lookup",
			Description: "Fetch a support ticket",
		},
		Steps: []*rollout.Step{{
			Name:      "get_ticket",
			Type:      "http",
			OpenAPI:   "openapi/support.yaml",
			Operation: "getTicket",
			With:      map[string]string{},
		}},
		Outputs: []*rollout.Output{{Name: "ticket", From: "get_ticket.received_body"}},
	}}
	docs := []elicitor.APIDocument{{
		RelativePath: "openapi/support.yaml",
		Operations: []apitools.OperationSummary{{
			OperationID: "getTicket",
			Parameters: []apitools.ParameterSummary{{
				Name:     "ticketId",
				In:       "path",
				Required: true,
			}},
		}},
	}}
	ok, reject := repairRequestMappingFromSuggestion(&session, docs, elicitor.ReadinessIssue{
		Code:            "missing_required_request_values",
		Slot:            "steps.get_ticket.with",
		SuggestedAnswer: "ticketId=inputs.ticketId, invented=inputs.other",
	})
	if ok || !strings.Contains(reject, "unknown request field invented") {
		t.Fatalf("repair result ok=%v reject=%q", ok, reject)
	}
	if len(session.Intent.Steps[0].With) != 0 {
		t.Fatalf("invented repair mutated step: %#v", session.Intent.Steps[0].With)
	}
}

func assertSeedBuildOutcome(t *testing.T, policy evalpkg.ReferencePolicy, got string, codes []string, detail string) {
	t.Helper()
	expected := policy.SeedBuild.Expected
	if got != expected {
		t.Fatalf("seed/build outcome = %s, want %s; codes=%v detail=%s", got, expected, codes, detail)
	}
	if expected == "pass" || len(policy.SeedBuild.AllowedFailureCodes) == 0 {
		return
	}
	allowed := map[string]bool{}
	for _, code := range policy.SeedBuild.AllowedFailureCodes {
		allowed[code] = true
	}
	for _, code := range codes {
		if allowed[code] {
			return
		}
	}
	t.Fatalf("seed/build failure codes = %v, want one of %v; detail=%s", codes, policy.SeedBuild.AllowedFailureCodes, detail)
}

func failedQualityCodes(report *synthesize.QualityReport) []string {
	if report == nil {
		return nil
	}
	var out []string
	for _, check := range report.Checks {
		if check.Status == "fail" {
			out = append(out, check.Code)
		}
	}
	return out
}

func qualityFailureDetails(report *synthesize.QualityReport) string {
	if report == nil {
		return ""
	}
	var out []string
	for _, check := range report.Checks {
		if check.Status == "fail" {
			out = append(out, check.Code+": "+check.Detail)
		}
	}
	return strings.Join(out, "; ")
}

func TestAutosaveResumesAndDeletesAfterSave(t *testing.T) {
	example := filepath.Join(t.TempDir(), "guided")
	var stdout, stderr bytes.Buffer
	code := Main([]string{"--example", example, "--no-llm"}, strings.NewReader(testProjectInput(false)), &stdout, &stderr)
	if code == 0 {
		t.Fatalf("first Main unexpectedly succeeded\nstdout:\n%s\nstderr:\n%s", stdout.String(), stderr.String())
	}
	draftPath := filepath.Join(example, ".icot", "session.yaml")
	if _, err := os.Stat(draftPath); err != nil {
		t.Fatalf("draft missing after EOF: %v", err)
	}
	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"--example", example, "--no-llm"}, strings.NewReader("save\n"), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("resume failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join(example, "project.md")); err != nil {
		t.Fatalf("project.md missing after resume: %v", err)
	}
	if _, err := os.Stat(filepath.Join(example, "workflows", "intent.hcl")); err != nil {
		t.Fatalf("intent.hcl missing after resume: %v", err)
	}
	if _, err := os.Stat(draftPath); !os.IsNotExist(err) {
		t.Fatalf("draft not deleted after save: %v", err)
	}
}

func TestCompleteDraftResumeCancelDeletesDraftWithoutWriting(t *testing.T) {
	example := filepath.Join(t.TempDir(), "guided")
	draftPath := writeCompleteDraft(t, example)
	var stdout, stderr bytes.Buffer
	code := Main([]string{"--example", example, "--no-llm"}, strings.NewReader("cancel\n"), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("cancel failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join(example, "project.md")); !os.IsNotExist(err) {
		t.Fatalf("cancel wrote project.md or unexpected stat error: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join(example, "workflows", "intent.hcl")); !os.IsNotExist(err) {
		t.Fatalf("cancel wrote intent.hcl or unexpected stat error: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(draftPath); !os.IsNotExist(err) {
		t.Fatalf("draft not deleted after cancel: %v", err)
	}
}

func TestCompleteDraftResumeSaveWritesAndDeletesDraft(t *testing.T) {
	example := filepath.Join(t.TempDir(), "guided")
	draftPath := writeCompleteDraft(t, example)
	var stdout, stderr bytes.Buffer
	code := Main([]string{"--example", example, "--no-llm"}, strings.NewReader("save\n"), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("save failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	for _, rel := range []string{"project.md", "workflows/intent.hcl"} {
		if _, err := os.Stat(filepath.Join(example, rel)); err != nil {
			t.Fatalf("%s missing after save: %v\nstdout:\n%s\nstderr:\n%s", rel, err, stdout.String(), stderr.String())
		}
	}
	if _, err := os.Stat(draftPath); !os.IsNotExist(err) {
		t.Fatalf("draft not deleted after save: %v", err)
	}
}

func TestPromptModeFastAcceptsCompleteDraftSaveDefaultSilently(t *testing.T) {
	example := filepath.Join(t.TempDir(), "guided")
	draftPath := writeCompleteDraft(t, example)
	var stdout, stderr bytes.Buffer
	code := Main([]string{"--example", example, "--no-llm", "--prompt-mode", "fast"}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("fast save failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	for _, rel := range []string{"project.md", "workflows/intent.hcl"} {
		if _, err := os.Stat(filepath.Join(example, rel)); err != nil {
			t.Fatalf("%s missing after fast save: %v\nstdout:\n%s\nstderr:\n%s", rel, err, stdout.String(), stderr.String())
		}
	}
	if _, err := os.Stat(draftPath); !os.IsNotExist(err) {
		t.Fatalf("draft not deleted after fast save: %v", err)
	}
	if strings.Contains(stdout.String(), "Type save, edit <slot>, explain <assumption-id>, regenerate, or cancel") {
		t.Fatalf("stdout printed auto-accepted save prompt:\n%s", stdout.String())
	}
}

func TestPromptModeNormalAcceptsCompleteDraftSaveDefaultVisibly(t *testing.T) {
	example := filepath.Join(t.TempDir(), "guided")
	draftPath := writeCompleteDraft(t, example)
	var stdout, stderr bytes.Buffer
	code := Main([]string{"--example", example, "--no-llm", "--prompt-mode", "normal"}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("normal save failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	for _, rel := range []string{"project.md", "workflows/intent.hcl"} {
		if _, err := os.Stat(filepath.Join(example, rel)); err != nil {
			t.Fatalf("%s missing after normal save: %v\nstdout:\n%s\nstderr:\n%s", rel, err, stdout.String(), stderr.String())
		}
	}
	if _, err := os.Stat(draftPath); !os.IsNotExist(err) {
		t.Fatalf("draft not deleted after normal save: %v", err)
	}
	if !strings.Contains(stdout.String(), "Type save, edit <slot>, explain <assumption-id>, regenerate, or cancel [save]: save") {
		t.Fatalf("stdout missing visibly auto-accepted save prompt:\n%s", stdout.String())
	}
}

func TestPromptModeFastWritesManualDraftFromOpeningOnly(t *testing.T) {
	example := filepath.Join(t.TempDir(), "guided")
	var stdout, stderr bytes.Buffer
	input := "Render a local summary report from a runtime input\n"
	code := Main([]string{"--example", example, "--no-llm", "--prompt-mode", "fast"}, strings.NewReader(input), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("fast manual draft failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	for _, rel := range []string{"project.md", "workflows/intent.hcl"} {
		if _, err := os.Stat(filepath.Join(example, rel)); err != nil {
			t.Fatalf("%s missing after fast manual draft: %v\nstdout:\n%s\nstderr:\n%s", rel, err, stdout.String(), stderr.String())
		}
	}
	for _, unexpected := range []string{
		"icot: running without LLM extraction",
		"Workflow timeout seconds (blank for none):",
		"Workflow name [",
		"Use OpenAPI/API steps?",
		"Type save, edit <slot>, explain <assumption-id>, regenerate, or cancel",
	} {
		if strings.Contains(stdout.String(), unexpected) {
			t.Fatalf("stdout printed auto-accepted prompt %q:\n%s", unexpected, stdout.String())
		}
	}
	if !strings.Contains(stdout.String(), "Workflow brief:") {
		t.Fatalf("stdout missing required no-default prompt:\n%s", stdout.String())
	}
}

func TestPromptModeRejectsUnknownValue(t *testing.T) {
	example := filepath.Join(t.TempDir(), "guided")
	var stdout, stderr bytes.Buffer
	code := Main([]string{"--example", example, "--prompt-mode", "turbo"}, strings.NewReader(""), &stdout, &stderr)
	if code != 2 {
		t.Fatalf("Main exit code = %d, want 2\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "--prompt-mode must be full, normal, or fast") {
		t.Fatalf("stderr missing prompt-mode error:\n%s", stderr.String())
	}
}

func TestReplayTranscriptMetricsUsesActualTranscriptTurns(t *testing.T) {
	path := filepath.Join(t.TempDir(), "transcript.json")
	data := `{
  "turns": [
    {"label": "Workflow goal", "answer": "fetch weather"},
    {"label": "Type save, edit <slot>, explain <assumption-id>, or cancel", "answer": "save"}
  ],
  "events": [
    {"kind": "draft_repair_attempt", "data": {"changed": true}},
    {"kind": "draft_repair_rejected", "data": ["intent.outputs.result"]},
    {"kind": "draft_flow_review_result", "data": {"issues": [{"code": "llm_flow_review_output"}]}}
  ]
}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write transcript: %v", err)
	}
	metrics := replayTranscriptMetrics(path, "Workflow goal: fetch weather\n")
	if metrics == nil {
		t.Fatalf("metrics nil")
	}
	if metrics.RepairAttempts != 1 || metrics.RepairRejected != 1 || metrics.UnresolvedReview != 1 {
		t.Fatalf("metrics = %#v", metrics)
	}
	if len(metrics.Turns) != 2 {
		t.Fatalf("turns = %#v", metrics.Turns)
	}
	if metrics.AutoAccepted != 1 {
		t.Fatalf("auto accepted = %d, want 1", metrics.AutoAccepted)
	}
}

func TestReplayPassesPolicyHonorsWarningAndReviewLimits(t *testing.T) {
	zero := 0
	result := replayEvalResult{
		Warning:          1,
		UnresolvedReview: 1,
	}
	policy := evalpkg.ReferencePolicy{
		MaxBlocking:         &zero,
		MaxWarning:          &zero,
		MaxUnresolvedReview: &zero,
	}
	if replayPassesPolicy(result, policy) {
		t.Fatalf("policy passed result with warnings: %#v", result)
	}
	result.Warning = 0
	result.UnresolvedReview = 0
	if !replayPassesPolicy(result, policy) {
		t.Fatalf("policy rejected clean result: %#v", result)
	}
}

func TestCompleteDraftPrintWritesNoFilesAndPreservesDraft(t *testing.T) {
	example := filepath.Join(t.TempDir(), "guided")
	draftPath := writeCompleteDraft(t, example)
	var stdout, stderr bytes.Buffer
	code := Main([]string{"--example", example, "--print", "--no-llm"}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("print failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join(example, "project.md")); !os.IsNotExist(err) {
		t.Fatalf("--print wrote project.md or unexpected stat error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(example, "workflows", "intent.hcl")); !os.IsNotExist(err) {
		t.Fatalf("--print wrote intent.hcl or unexpected stat error: %v", err)
	}
	if _, err := os.Stat(draftPath); err != nil {
		t.Fatalf("--print should preserve draft: %v", err)
	}
	if !strings.Contains(stdout.String(), "----- project.md -----") || !strings.Contains(stdout.String(), "----- workflows/intent.hcl -----") {
		t.Fatalf("print output missing artifacts:\n%s", stdout.String())
	}
}

func TestNoTranscriptSkipsLocalTranscript(t *testing.T) {
	example := filepath.Join(t.TempDir(), "guided")
	var stdout, stderr bytes.Buffer
	code := Main([]string{"--example", example, "--no-llm", "--no-transcript"}, strings.NewReader(testProjectInput(false)+"save\n"), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Main failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join(example, ".icot", "transcript.json")); !os.IsNotExist(err) {
		t.Fatalf("--no-transcript wrote transcript or unexpected stat error: %v", err)
	}
}

func TestProviderFromEnvDefaultsToCopilotAPI(t *testing.T) {
	t.Setenv("OPENUDON_LLM_PROVIDER", "")
	t.Setenv("COPILOT_API_BASE_URL", "")
	t.Setenv("COPILOT_API_KEY", "")
	t.Setenv("GEMINI_API_KEY", "gemini-key")
	t.Setenv("OPENAI_API_KEY", "openai-key")
	t.Setenv("ANTHROPIC_API_KEY", "anthropic-key")

	if got := providerFromEnv(); got != "copilot-api" {
		t.Fatalf("providerFromEnv() = %q, want copilot-api", got)
	}
}

func TestProviderFromEnvHonorsOpenUdonProviderOverride(t *testing.T) {
	t.Setenv("OPENUDON_LLM_PROVIDER", "gemini")

	if got := providerFromEnv(); got != "gemini" {
		t.Fatalf("providerFromEnv() = %q, want gemini", got)
	}
}

func TestCompleteDraftEditsReplaceSeededPolicyFields(t *testing.T) {
	example := filepath.Join(t.TempDir(), "guided")
	writeCompleteDraftWithPolicy(t, example, []string{"old_token"}, "Old safety note", "Old fallback note")
	input := strings.Join([]string{
		"edit credentials",
		"new_token",
		"edit safety",
		"New safety note",
		"edit fallback",
		"New fallback note",
		"save",
	}, "\n") + "\n"
	var stdout, stderr bytes.Buffer
	code := Main([]string{"--example", example, "--no-llm"}, strings.NewReader(input), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("policy edit failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	project, err := os.ReadFile(filepath.Join(example, "project.md"))
	if err != nil {
		t.Fatalf("read project: %v", err)
	}
	text := string(project)
	for _, unexpected := range []string{"old_token", "Old safety note", "Old fallback note"} {
		if strings.Contains(text, unexpected) {
			t.Fatalf("project retained stale policy value %q:\n%s", unexpected, text)
		}
	}
	for _, expected := range []string{"new_token", "New safety note", "New fallback note"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("project missing edited policy value %q:\n%s", expected, text)
		}
	}
}

func TestCompleteDraftCredentialsNoneClearsSeededBindings(t *testing.T) {
	example := filepath.Join(t.TempDir(), "guided")
	writeCompleteDraftWithPolicy(t, example, []string{"old_token"}, "Sandbox proof runs only", "Stop on errors")
	var stdout, stderr bytes.Buffer
	code := Main([]string{"--example", example, "--no-llm"}, strings.NewReader("edit credentials\nnone\nsave\n"), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("credential clear failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	project, err := os.ReadFile(filepath.Join(example, "project.md"))
	if err != nil {
		t.Fatalf("read project: %v", err)
	}
	text := string(project)
	if strings.Contains(text, "old_token") || !strings.Contains(text, "Credential bindings: none declared") {
		t.Fatalf("credentials were not cleared:\n%s", text)
	}
}

func TestReconcileRegeneratesProjectOnlyAndPreservesPolicy(t *testing.T) {
	example := t.TempDir()
	intent := testIntent("support_ticket", "Fetch support ticket", "get_ticket")
	intent.OpenAPI = "openapi/support.yaml"
	intent.Steps[0].Type = "http"
	intent.Steps[0].Operation = "getTicket"
	intent.Steps[0].With = map[string]string{"api_key": "credentials.support_api_token"}
	intentHCL, err := runner.RenderIntentHCL(intent)
	if err != nil {
		t.Fatalf("render intent: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(example, "workflows"), 0o755); err != nil {
		t.Fatalf("mkdir workflows: %v", err)
	}
	intentPath := filepath.Join(example, "workflows", "intent.hcl")
	if err := os.WriteFile(intentPath, []byte(intentHCL), 0o644); err != nil {
		t.Fatalf("write intent: %v", err)
	}
	oldProject := projectwizard.Render(projectwizard.Answers{
		ProjectName:     "Old Project",
		Goal:            "Old goal",
		Credentials:     []string{"support_api_token"},
		SideEffectScope: projectwizard.SideEffectSandboxOnly,
		Safety:          "Use sandbox endpoints only",
		Fallback:        "Stop if support API is unavailable",
	})
	if err := os.WriteFile(filepath.Join(example, "project.md"), []byte(oldProject), 0o644); err != nil {
		t.Fatalf("write project: %v", err)
	}
	var stdout, stderr bytes.Buffer
	code := Main([]string{"reconcile", "--example", example, "--yes"}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("reconcile failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	project, err := os.ReadFile(filepath.Join(example, "project.md"))
	if err != nil {
		t.Fatalf("read project: %v", err)
	}
	for _, expected := range []string{"Fetch support ticket", "openapi/support.yaml", "support_api_token", "Use sandbox endpoints only", "Stop if support API is unavailable"} {
		if !strings.Contains(string(project), expected) {
			t.Fatalf("reconciled project missing %q:\n%s", expected, project)
		}
	}
	backups, err := filepath.Glob(filepath.Join(example, "project.md.bak.*"))
	if err != nil || len(backups) != 1 {
		t.Fatalf("expected project backup, got %v err %v", backups, err)
	}
	intentBackups, err := filepath.Glob(filepath.Join(example, "workflows", "intent.hcl.bak.*"))
	if err != nil || len(intentBackups) != 0 {
		t.Fatalf("reconcile should not back up intent, got %v err %v", intentBackups, err)
	}
}

func TestReconcileProjectIncludesNestedIntentDetails(t *testing.T) {
	example := t.TempDir()
	intent := nestedIntent()
	intentHCL, err := runner.RenderIntentHCL(intent)
	if err != nil {
		t.Fatalf("render intent: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(example, "workflows"), 0o755); err != nil {
		t.Fatalf("mkdir workflows: %v", err)
	}
	if err := os.WriteFile(filepath.Join(example, "workflows", "intent.hcl"), []byte(intentHCL), 0o644); err != nil {
		t.Fatalf("write intent: %v", err)
	}
	project := projectwizard.Render(projectwizard.Answers{
		ProjectName:     "Nested",
		Credentials:     []string{"support_api_token"},
		SideEffectScope: projectwizard.SideEffectSandboxOnly,
		Safety:          "Sandbox proof runs only",
		Fallback:        "Stop on errors",
	})
	if err := os.WriteFile(filepath.Join(example, "project.md"), []byte(project), 0o644); err != nil {
		t.Fatalf("write project: %v", err)
	}
	var stdout, stderr bytes.Buffer
	code := Main([]string{"reconcile", "--example", example, "--print"}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("reconcile print failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	text := stdout.String()
	for _, expected := range []string{
		"openapi/nested.yaml",
		"`nested_lookup.ticketId` comes from `get_ticket.received_body.id`",
		"- `prepare_default`\n  - Purpose: Prepare the default result.",
		"support_api_token",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("reconciled project missing nested detail %q:\n%s", expected, text)
		}
	}
}

func TestLintDriftWarnsForNestedIntentFieldsAndRuntimeApprovals(t *testing.T) {
	example := t.TempDir()
	project := projectwizard.Render(projectwizard.Answers{
		ProjectName:     "Nested Drift",
		Goal:            "Handle nested workflow",
		UsesOpenAPI:     false,
		SideEffectScope: projectwizard.SideEffectReadOnly,
		Fallback:        "Stop on errors",
	})
	if err := os.WriteFile(filepath.Join(example, "project.md"), []byte(project), 0o644); err != nil {
		t.Fatalf("write project: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(example, "workflows"), 0o755); err != nil {
		t.Fatalf("mkdir workflows: %v", err)
	}
	intentHCL, err := runner.RenderIntentHCL(nestedIntent())
	if err != nil {
		t.Fatalf("render intent: %v", err)
	}
	if err := os.WriteFile(filepath.Join(example, "workflows", "intent.hcl"), []byte(intentHCL), 0o644); err != nil {
		t.Fatalf("write intent: %v", err)
	}
	var stdout, stderr bytes.Buffer
	code := Main([]string{"lint", "--example", example}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("lint drift should exit zero, got %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	text := stdout.String()
	for _, expected := range []string{
		"icot.drift.openapi: warn",
		"icot.drift.data_flow: warn",
		"icot.drift.functions: warn",
		"icot.drift.credentials: warn",
		"icot.drift.runtime.cmd: warn",
		"icot.drift.runtime.ssh: warn",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("lint output missing nested drift warning %q:\n%s", expected, text)
		}
	}
}

func TestLintDriftWarnsButExitsZero(t *testing.T) {
	example := t.TempDir()
	project := projectwizard.Render(projectwizard.Answers{
		ProjectName:     "Drift",
		Goal:            "Render something else",
		UsesOpenAPI:     false,
		SideEffectScope: projectwizard.SideEffectReadOnly,
		Fallback:        "Stop on errors",
	})
	if err := os.WriteFile(filepath.Join(example, "project.md"), []byte(project), 0o644); err != nil {
		t.Fatalf("write project: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(example, "workflows"), 0o755); err != nil {
		t.Fatalf("mkdir workflows: %v", err)
	}
	intentHCL, err := runner.RenderIntentHCL(testIntent("drift", "Fetch a ticket", "get_ticket"))
	if err != nil {
		t.Fatalf("render intent: %v", err)
	}
	if err := os.WriteFile(filepath.Join(example, "workflows", "intent.hcl"), []byte(intentHCL), 0o644); err != nil {
		t.Fatalf("write intent: %v", err)
	}
	var stdout, stderr bytes.Buffer
	code := Main([]string{"lint", "--example", example}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("lint drift should exit zero, got %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "icot.drift.goal: warn") || !strings.Contains(stdout.String(), "icot.drift.inputs: warn") {
		t.Fatalf("lint output missing drift warnings:\n%s", stdout.String())
	}
}

func TestAtomicWriterDoesNotModifyFinalFilesWhenValidationFails(t *testing.T) {
	dir := t.TempDir()
	projectPath := filepath.Join(dir, "project.md")
	intentPath := filepath.Join(dir, "workflows", "intent.hcl")
	if err := os.MkdirAll(filepath.Dir(intentPath), 0o755); err != nil {
		t.Fatalf("mkdir workflows: %v", err)
	}
	if err := os.WriteFile(projectPath, []byte("old project\n"), 0o644); err != nil {
		t.Fatalf("write project: %v", err)
	}
	if err := os.WriteFile(intentPath, []byte("workflow {}\n"), 0o644); err != nil {
		t.Fatalf("write intent: %v", err)
	}
	err := writeGeneratedFilesAtomic([]generatedFile{
		{Path: projectPath, Content: "new project\n"},
		{Path: intentPath, Content: "not valid hcl"},
	}, true)
	if err == nil {
		t.Fatalf("atomic write unexpectedly succeeded")
	}
	project, _ := os.ReadFile(projectPath)
	intent, _ := os.ReadFile(intentPath)
	if string(project) != "old project\n" || string(intent) != "workflow {}\n" {
		t.Fatalf("files changed after failed atomic write:\nproject=%q\nintent=%q", project, intent)
	}
}

func testIntent(name, goal, stepName string) *rollout.Intent {
	return &rollout.Intent{
		Workflow: &rollout.WorkflowMeta{Name: name, Description: goal},
		Inputs:   []*rollout.Input{{Name: "ticket_id", Type: "string", Required: true}},
		Steps: []*rollout.Step{{
			Name: stepName,
			Type: "fnct",
			Do:   goal,
			With: map[string]string{"ticket_id": "inputs.ticket_id"},
		}},
		Outputs: []*rollout.Output{{Name: "ticket", From: stepName + ".received_body"}},
	}
}

func nestedIntent() *rollout.Intent {
	return &rollout.Intent{
		OpenAPI:  "openapi/nested.yaml",
		Workflow: &rollout.WorkflowMeta{Name: "nested_workflow", Description: "Handle nested workflow"},
		Inputs:   []*rollout.Input{{Name: "ticketId", Type: "string", Required: true}},
		Steps: []*rollout.Step{
			{
				Name:      "get_ticket",
				Type:      "http",
				Do:        "Fetch the ticket",
				Operation: "getTicket",
				With:      map[string]string{"ticketId": "inputs.ticketId", "api_key": "credentials.support_api_token"},
			},
			{
				Name:      "route_ticket",
				Type:      "switch",
				DependsOn: []string{"get_ticket"},
				Cases: []*rollout.StepCase{{
					Name: "urgent",
					When: "get_ticket.received_body.priority == \"urgent\"",
					Steps: []*rollout.Step{
						{
							Name:      "nested_lookup",
							Type:      "http",
							Do:        "Look up nested context",
							OpenAPI:   "openapi/nested.yaml",
							Operation: "getNested",
							Binds: []*rollout.StepBind{{
								From:   "get_ticket",
								Fields: map[string]string{"ticketId": "received_body.id"},
							}},
						},
						{Name: "run_local_command", Type: "cmd", Do: "Run approved local command"},
					},
				}},
				Default: &rollout.StepDefault{Steps: []*rollout.Step{
					{
						Name: "prepare_default",
						Type: "fnct",
						Do:   "Prepare the default result",
						Binds: []*rollout.StepBind{{
							From:   "get_ticket",
							Fields: map[string]string{"ticket": "received_body"},
						}},
					},
					{Name: "check_remote_host", Type: "ssh", Do: "Check remote host"},
				}},
			},
		},
		Outputs: []*rollout.Output{{Name: "result", From: "prepare_default.received_body"}},
	}
}

func writeCompleteDraft(t *testing.T, example string) string {
	t.Helper()
	return writeCompleteDraftWithPolicy(t, example, nil, "Sandbox proof runs only", "Stop if inputs are missing")
}

func writeCompleteDraftWithPolicy(t *testing.T, example string, credentials []string, safety, fallback string) string {
	t.Helper()
	project := projectwizard.Answers{
		ProjectName:     "Draft Project",
		Goal:            "Render a draft report",
		Credentials:     credentials,
		SideEffectScope: projectwizard.SideEffectSandboxOnly,
		Safety:          safety,
		Fallback:        fallback,
	}
	session := elicitor.SessionFromIntent(testIntent("draft_project", "Render a draft report", "render_report"), project)
	draftPath := elicitor.DraftPath(example)
	if err := elicitor.SaveDraft(draftPath, session); err != nil {
		t.Fatalf("save draft: %v", err)
	}
	return draftPath
}

func testProjectInput(withOpenAPI bool) string {
	answers := []string{
		"Render a local summary report from a runtime input",
		"guided_project",
		"Render a local summary report",
		"",
		"",
	}
	if withOpenAPI {
		answers = append(answers, "yes", "openapi/support.yaml")
	} else {
		answers = append(answers, "no")
	}
	answers = append(answers,
		"summary:string",
		"render_report",
		"fnct",
		"Render the summary report",
		"",
		"summary",
		"",
		"",
		"report",
		"render_report.received_body",
		"sandbox-only",
		"",
		"Sandbox proof runs only",
		"Stop if required services are unavailable",
	)
	return strings.Join(answers, "\n") + "\n"
}

func writeSessionJSON(t *testing.T, dir string, session elicitor.Session) string {
	t.Helper()
	path := filepath.Join(dir, "session.json")
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		t.Fatalf("marshal session: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write session: %v", err)
	}
	return path
}

func writeRepairDependencyExample(t *testing.T, example string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(example, "workflows"), 0o755); err != nil {
		t.Fatalf("mkdir example: %v", err)
	}
	project := projectwizard.Answers{
		ProjectName:     "Repair Dependencies",
		Goal:            "Render and send a report",
		SideEffectScope: projectwizard.SideEffectSandboxOnly,
		Safety:          "Sandbox proof runs only",
		Fallback:        "Stop if rendering fails",
	}
	if err := os.WriteFile(filepath.Join(example, "project.md"), []byte(projectwizard.Render(project)), 0o644); err != nil {
		t.Fatalf("write project: %v", err)
	}
	rendered := `workflow {
  name        = "repair_dependencies"
  description = "Render and send a report"
}

step "render_report" {
  type = "fnct"
  do   = "Render the report"
}

step "send_report" {
  type = "fnct"
  do   = "Send the report"
  with = {
    body = "render_report.received_body"
  }
}

output "result" {
  from = "send_report.received_body"
}
`
	if _, err := runner.ParseIntent([]byte(rendered), "intent.hcl"); err != nil {
		t.Fatalf("parse fixture intent: %v", err)
	}
	if err := os.WriteFile(filepath.Join(example, "workflows", "intent.hcl"), []byte(rendered), 0o644); err != nil {
		t.Fatalf("write intent: %v", err)
	}
}

func repairReportHasSlot(report repairReport, slot string) bool {
	for _, change := range report.Applied {
		if change.Slot == slot {
			return true
		}
	}
	return false
}

func findTestStep(steps []*rollout.Step, name string) *rollout.Step {
	for _, step := range steps {
		if step == nil {
			continue
		}
		if step.Name == name {
			return step
		}
		if found := findTestStep(step.Steps, name); found != nil {
			return found
		}
	}
	return nil
}

func writeCompleteRuntimeSession(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "session.yaml")
	data := "project:\n" +
		"  project_name: Runtime Report\n" +
		"  goal: Render a local runtime report\n" +
		"  inputs: '`summary`: required string'\n" +
		"  outputs: '`report`: rendered report body'\n" +
		"  uses_openapi: false\n" +
		"  safety: Sandbox proof runs only\n" +
		"  fallback: Stop if rendering fails\n" +
		"safety: Sandbox proof runs only\n" +
		"fallback: Stop if rendering fails\n" +
		"side_effect_scope: sandbox-only\n" +
		"intent:\n" +
		"  workflow:\n" +
		"    name: runtime_report\n" +
		"    description: Render a local runtime report\n" +
		"  inputs:\n" +
		"    - name: summary\n" +
		"      type: string\n" +
		"      required: true\n" +
		"  steps:\n" +
		"    - name: render_report\n" +
		"      type: fnct\n" +
		"      do: Render the report body\n" +
		"      with:\n" +
		"        summary: inputs.summary\n" +
		"  outputs:\n" +
		"    - name: report\n" +
		"      from: render_report.received_body\n"
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write complete session: %v", err)
	}
	return path
}
