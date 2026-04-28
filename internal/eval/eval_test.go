package eval

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/genelet/ramen/internal/synthesize"
	"github.com/genelet/udon/pkg/rollout"
)

type fakeRuntimeClient struct{}

func (fakeRuntimeClient) Chat(context.Context, []rollout.ChatMessage) (string, error) {
	return `{
  "workflow": {"name": "runtime_only_render", "description": "Render a local summary report."},
  "inputs": [{"name": "summary", "type": "string", "required": true}],
  "steps": [
    {"name": "render_report", "type": "fnct", "do": "Render the summary report.", "with": {"summary": "inputs.summary"}}
  ],
  "outputs": [{"name": "report", "from": "render_report.received_body"}]
}`, nil
}

func (fakeRuntimeClient) Generate(context.Context, string) (string, error) {
	return "```json\n" + `{
  "steps": [
    {
      "type": "fnct",
      "name": "render_report",
      "body": {"attributes": {"function": "identity"}}
    }
  ]
}` + "\n```", nil
}

func TestCompareIntentsReportsStructuralIssues(t *testing.T) {
	generated := &rollout.Intent{
		Steps: []*rollout.Step{{
			Name:      "fetch_ticket",
			Type:      "http",
			Operation: "wrongOperation",
		}},
	}
	reference := &rollout.Intent{
		Steps: []*rollout.Step{{
			Name:      "fetch_ticket",
			Type:      "http",
			Operation: "getTicket",
		}},
	}
	issues := CompareIntents(generated, reference)
	if len(issues) != 1 || issues[0].Code != "intent.step_operation" {
		t.Fatalf("unexpected issues: %#v", issues)
	}
	if issues[0].Severity != "blocking" {
		t.Fatalf("severity = %q, want blocking", issues[0].Severity)
	}
}

func TestReferenceIssueSeverityClassifiesAdvisoryDrift(t *testing.T) {
	generated := &rollout.Intent{
		Outputs: []*rollout.Output{{Name: "status_text", From: "render_report.received_body"}},
		Steps: []*rollout.Step{{
			Name: "render_report",
			Type: "fnct",
			With: map[string]string{"summary": "inputs.summary"},
		}},
	}
	reference := &rollout.Intent{
		Outputs: []*rollout.Output{{Name: "status", From: "render_report.received_body"}},
		Steps: []*rollout.Step{{
			Name: "render_report",
			Type: "fnct",
			Binds: []*rollout.StepBind{{
				From:   "source",
				Fields: map[string]string{"summary": "received_body.summary"},
			}},
		}},
	}
	issues := CompareIntents(generated, reference)
	summary := summarizeReferenceIssues(issues)
	if summary.Advisory == 0 || summary.Blocking != 0 {
		t.Fatalf("summary = %#v issues = %#v, want advisory-only drift", summary, issues)
	}
}

func TestReferencePolicyDowngradesIllustrativeReferenceDrift(t *testing.T) {
	issues := applyReferencePolicy([]CompareIssue{{
		Code:     "intent.step_operation",
		Detail:   `fetch expected "getTicket" got "listTickets"`,
		Severity: "blocking",
	}}, ReferencePolicy{
		Mode: "advisory",
		IssueNotes: map[string]string{
			"*": "illustrative reference",
		},
	})
	if len(issues) != 1 {
		t.Fatalf("unexpected issues: %#v", issues)
	}
	if issues[0].Severity != "advisory" || issues[0].Note != "illustrative reference" {
		t.Fatalf("issue = %#v, want advisory with note", issues[0])
	}
}

func TestReferencePolicyStrictPreservesDefaultSeverity(t *testing.T) {
	issues := applyReferencePolicy([]CompareIssue{{
		Code:     "intent.step_operation",
		Detail:   `fetch expected "getTicket" got "listTickets"`,
		Severity: "blocking",
	}}, ReferencePolicy{Mode: "strict"})
	if len(issues) != 1 {
		t.Fatalf("unexpected issues: %#v", issues)
	}
	if issues[0].Severity != "blocking" {
		t.Fatalf("severity = %q, want blocking", issues[0].Severity)
	}
}

func TestReadReferencePolicyNormalizesModeAndSeverity(t *testing.T) {
	path := filepath.Join(t.TempDir(), "policy.json")
	if err := os.WriteFile(path, []byte(`{
  "mode": "ADVISORY",
  "severity_overrides": {"intent.step_operation": "BLOCKING"},
  "max_blocking": 1
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	policy, err := ReadReferencePolicy(path)
	if err != nil {
		t.Fatal(err)
	}
	if policy.Mode != "advisory" {
		t.Fatalf("mode = %q, want advisory", policy.Mode)
	}
	if policy.SeverityOverrides["intent.step_operation"] != "blocking" {
		t.Fatalf("severity overrides = %#v", policy.SeverityOverrides)
	}
	if policy.MaxBlocking == nil || *policy.MaxBlocking != 1 {
		t.Fatalf("max blocking = %#v, want 1", policy.MaxBlocking)
	}
}

func TestRunOneReportsMalformedReferencePolicyWarning(t *testing.T) {
	example := filepath.Join(t.TempDir(), "runtime-only")
	if err := os.MkdirAll(filepath.Join(example, "reference"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(example, "project.md"), []byte(`# Runtime Only

## Goal

Render a local summary report.

## Inputs

- No inputs are required.

## Outputs

- Rendered report.

## External Systems and OpenAPI

OpenAPI: none required

## Runtime Policy

- fnct allowed.

## Function Contracts

- render_report
  - Inputs: summary.
  - Outputs: rendered report.
  - Side effects: none.

## Credentials and Secrets

- No credentials are required.

## Safety and Approval Boundary

- Generate only.

## Fallback Behavior

- Stop on failure.
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(example, "reference", "intent.hcl"), []byte(`workflow {
  name = "runtime_only_render"
}

step "render_report" {
  type = "fnct"
  do   = "Render the summary report."
  with = {
    summary = "inputs.summary"
  }
}

output "report" {
  from = "render_report.received_body"
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(example, "reference", "policy.json"), []byte(`{`), 0o644); err != nil {
		t.Fatal(err)
	}
	schemaPath, err := filepath.Abs(filepath.Join("..", "..", "..", "uws", "versions", "1.0.0.json"))
	if err != nil {
		t.Fatal(err)
	}
	result := RunOne(context.Background(), example, synthesize.Options{
		LLMClient:  fakeRuntimeClient{},
		ChatClient: fakeRuntimeClient{},
		SchemaPath: schemaPath,
	})
	var found bool
	for _, issue := range result.ReferenceIssues {
		if issue.Code == "reference.policy" && issue.Severity == "warning" {
			found = true
		}
	}
	if !found {
		t.Fatalf("reference issues = %#v, want reference.policy warning", result.ReferenceIssues)
	}
}

func TestRunOneUsesTempCopyAndReadsRefinement(t *testing.T) {
	example := filepath.Join(t.TempDir(), "runtime-only")
	if err := os.MkdirAll(filepath.Join(example, "reference"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(example, "project.md"), []byte(`# Runtime Only

## Goal

Render a local summary report.

## Inputs

- summary: string.

## Outputs

- Rendered report.

## External Systems and OpenAPI

OpenAPI: none required

## Runtime Policy

- fnct allowed.

## Function Contracts

- render_report
  - Inputs: summary.
  - Outputs: rendered report.
  - Side effects: none.

## Credentials and Secrets

- No credentials are required.

## Safety and Approval Boundary

- Generate only.

## Fallback Behavior

- Stop on failure.
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(example, "reference", "intent.hcl"), []byte(`workflow {
  name = "runtime_only_render"
}

input "summary" {
  type     = "string"
  required = true
}

step "render_report" {
  type = "fnct"
  do   = "Render the summary report."
  with = {
    summary = "inputs.summary"
  }
}

output "report" {
  from = "render_report.received_body"
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	schemaPath, err := filepath.Abs(filepath.Join("..", "..", "..", "uws", "versions", "1.0.0.json"))
	if err != nil {
		t.Fatal(err)
	}
	result := RunOne(context.Background(), example, synthesize.Options{
		LLMClient:  fakeRuntimeClient{},
		ChatClient: fakeRuntimeClient{},
		SchemaPath: schemaPath,
	})
	if !result.Passed {
		t.Fatalf("expected eval to pass: %#v", result)
	}
	if result.PromptVersion == "" || result.AttemptsToPass != 1 {
		t.Fatalf("result missing refinement evidence: %#v", result)
	}
	if result.AttemptCount != 1 || result.RepeatedRepairLoop {
		t.Fatalf("result repair-loop evidence = attempts %d repeated %v, want one non-repeated attempt", result.AttemptCount, result.RepeatedRepairLoop)
	}
	if result.ReferenceSummary != (ReferenceSummary{}) || len(result.ReferenceIssues) != 0 {
		t.Fatalf("reference summary = %#v issues = %#v, want no drift", result.ReferenceSummary, result.ReferenceIssues)
	}
	if result.GeneratedDir == "" || result.GeneratedDir == example {
		t.Fatalf("expected generated temp dir, got %q", result.GeneratedDir)
	}
	if _, err := os.Stat(filepath.Join(example, "workflows")); !os.IsNotExist(err) {
		t.Fatalf("eval dirtied source example workflows dir: %v", err)
	}
}

func TestRegressionErrorDetectsFailedPreviouslyPassingBrief(t *testing.T) {
	err := RegressionError(
		[]EvalResult{{Name: "a", Passed: false}},
		[]EvalResult{{Name: "a", Passed: true}},
	)
	if err == nil || !strings.Contains(err.Error(), "pass rate regressed") {
		t.Fatalf("expected regression error, got %v", err)
	}
}

func TestRegressionErrorDetectsLegacyFallbackIncrease(t *testing.T) {
	err := RegressionError(
		[]EvalResult{{Name: "a", Passed: true, Mode: "legacy", UsedLegacyExtract: true}},
		[]EvalResult{{Name: "a", Passed: true, Mode: "structured"}},
	)
	if err == nil || !strings.Contains(err.Error(), "legacy extractJSON fallback count regressed") {
		t.Fatalf("expected legacy fallback regression, got %v", err)
	}
}

func TestRegressionErrorDetectsBlockingReferenceRegression(t *testing.T) {
	err := RegressionError(
		[]EvalResult{{
			Name:   "a",
			Passed: true,
			ReferenceIssues: []CompareIssue{{
				Code:     "intent.step_operation",
				Detail:   `fetch expected "getTicket" got "listTickets"`,
				Severity: "blocking",
			}},
		}},
		[]EvalResult{{Name: "a", Passed: true}},
	)
	if err == nil || !strings.Contains(err.Error(), "blocking reference issue count regressed") {
		t.Fatalf("expected blocking reference regression, got %v", err)
	}
}

func TestCompareRunsReportsAnalyticsDeltas(t *testing.T) {
	previous := []EvalResult{
		{Name: "a", Passed: true, Mode: "structured", AttemptCount: 1, PromptTokensApprox: 100, DurationMs: 10},
		{Name: "b", Passed: false, FailingChecks: []string{"old.check"}, AttemptCount: 3, PromptTokensApprox: 200, DurationMs: 20},
	}
	current := []EvalResult{
		{
			Name:               "a",
			Passed:             false,
			Mode:               "legacy",
			UsedLegacyExtract:  true,
			AttemptCount:       2,
			FailingChecks:      []string{"new.check"},
			PromptTokensApprox: 150,
			DurationMs:         15,
			ReferenceIssues: []CompareIssue{{
				Code:     "intent.step_operation",
				Severity: "blocking",
			}},
		},
		{Name: "b", Passed: true, AttemptCount: 2, PromptTokensApprox: 250, DurationMs: 35},
	}
	comparison := CompareRuns(current, previous, "eval/runs/previous.json")
	if !comparison.HasRegression {
		t.Fatalf("expected regression: %#v", comparison)
	}
	if comparison.PreviousPath != "eval/runs/previous.json" || comparison.PassRateDelta != 0 {
		t.Fatalf("unexpected pass comparison: %#v", comparison)
	}
	if comparison.LegacyFallbackDelta != 1 || comparison.BlockingReferenceDelta != 1 {
		t.Fatalf("unexpected regression deltas: %#v", comparison)
	}
	if strings.Join(comparison.NewlyFailingBriefs, ",") != "a" || strings.Join(comparison.FixedBriefs, ",") != "b" {
		t.Fatalf("unexpected brief deltas: %#v", comparison)
	}
	if len(comparison.AttemptRegressions) != 1 || comparison.AttemptRegressions[0].Name != "a" {
		t.Fatalf("unexpected attempt regressions: %#v", comparison.AttemptRegressions)
	}
	if strings.Join(comparison.NewFailingChecks, ",") != "new.check" || strings.Join(comparison.ResolvedFailingChecks, ",") != "old.check" {
		t.Fatalf("unexpected check deltas: %#v", comparison)
	}
	if comparison.PromptTokensApproxDelta != 100 || comparison.DurationMsDelta != 20 {
		t.Fatalf("unexpected resource deltas: %#v", comparison)
	}
}

func TestReleaseCriteriaErrorCatchesReleaseGateFailures(t *testing.T) {
	err := ReleaseCriteriaError([]EvalResult{
		{
			Name:              "legacy",
			Passed:            true,
			Mode:              "legacy",
			UsedLegacyExtract: true,
			AttemptCount:      3,
			ReferenceIssues: []CompareIssue{{
				Code:     "intent.step_operation",
				Detail:   "wrong operation",
				Severity: "blocking",
			}},
		},
		{
			Name:          "secret",
			Passed:        false,
			FailingChecks: []string{"artifacts.no_secrets"},
		},
	}, DefaultReleaseCriteria())
	for _, expected := range []string{
		"pass rate",
		"legacy fallback count",
		"attempt count exceeds",
		"blocking reference issues",
		"secret-scan failures",
	} {
		if err == nil || !strings.Contains(err.Error(), expected) {
			t.Fatalf("release criteria error missing %q: %v", expected, err)
		}
	}
}

func TestReleaseCriteriaErrorUsesPerFixtureReferenceThreshold(t *testing.T) {
	allowed := 1
	err := ReleaseCriteriaError([]EvalResult{{
		Name:         "known-reference-drift",
		Passed:       true,
		Mode:         "structured",
		AttemptCount: 1,
		ReferencePolicy: &ReferencePolicy{
			MaxBlocking: &allowed,
		},
		ReferenceIssues: []CompareIssue{{
			Code:     "intent.step_operation",
			Detail:   "known drift",
			Severity: "blocking",
		}},
	}}, DefaultReleaseCriteria())
	if err != nil {
		t.Fatalf("unexpected release criteria error: %v", err)
	}
}

func TestReleaseCriteriaErrorPassesCleanRun(t *testing.T) {
	err := ReleaseCriteriaError([]EvalResult{{
		Name:          "ok",
		Passed:        true,
		Mode:          "structured",
		AttemptCount:  1,
		FailingChecks: nil,
	}}, DefaultReleaseCriteria())
	if err != nil {
		t.Fatalf("unexpected release criteria error: %v", err)
	}
}

func TestMarkdownIncludesReferenceSeveritySummary(t *testing.T) {
	policy := ReferencePolicy{Mode: "advisory"}
	md := Markdown([]EvalResult{{
		Name:               "a",
		Provider:           "gemini",
		Model:              "gemini-2.5-flash",
		PromptVersion:      "ramen.prompt.v1",
		Mode:               "structured",
		Passed:             true,
		AttemptCount:       2,
		RepeatedRepairLoop: true,
		PromptTokensApprox: 123,
		DurationMs:         456,
		ReferenceSummary: ReferenceSummary{
			Advisory: 2,
			Warning:  1,
			Blocking: 0,
		},
		ReferencePolicy: &policy,
		ReferenceIssues: []CompareIssue{{
			Code:     "intent.outputs",
			Detail:   "extra output",
			Severity: "advisory",
			Note:     "illustrative reference",
		}},
	}})
	for _, expected := range []string{
		"Reference issues (A/W/B)",
		"Reference policies: `advisory`=1",
		"Repeated repair loops: `1`",
		"Prompt tokens approx total: `123`",
		"Modes: `structured`=1",
		"Providers: `gemini`=1",
		"Models: `gemini-2.5-flash`=1",
		"Prompt versions: `ramen.prompt.v1`=1",
		"| `a` | pass | gemini | gemini-2.5-flash | ramen.prompt.v1 | structured | 2 |  |  | 2/1/0 | advisory | 123 | 456ms |",
		"## Reference Issue Details",
		"- advisory `intent.outputs`: extra output (illustrative reference)",
	} {
		if !strings.Contains(md, expected) {
			t.Fatalf("markdown missing %q:\n%s", expected, md)
		}
	}
}

func TestBuildRunSummaryAggregatesAnalytics(t *testing.T) {
	summary := BuildRunSummary([]EvalResult{
		{
			Name:               "pass",
			Provider:           "gemini",
			Model:              "gemini-2.5-flash",
			PromptVersion:      "prompt-a",
			Mode:               "structured",
			Passed:             true,
			AttemptCount:       1,
			PromptTokensApprox: 100,
			DurationMs:         10,
			ReferencePolicy:    &ReferencePolicy{Mode: "strict"},
		},
		{
			Name:               "repair",
			Provider:           "gemini",
			Model:              "gemini-2.5-flash",
			PromptVersion:      "prompt-a",
			Mode:               "legacy",
			UsedLegacyExtract:  true,
			Passed:             false,
			AttemptCount:       3,
			RepeatedRepairLoop: true,
			FailureClass:       "workflow",
			FailingChecks:      []string{"workflow.plan_match", "workflow.plan_match", "uws.validate"},
			PromptTokensApprox: 300,
			DurationMs:         30,
		},
	})
	if summary.Briefs != 2 || summary.Passed != 1 || summary.Failed != 1 || summary.PassRate != 0.5 {
		t.Fatalf("unexpected pass summary: %#v", summary)
	}
	if summary.LegacyFallbacks != 1 || summary.RepeatedRepairLoops != 1 || strings.Join(summary.RepeatedRepairBriefs, ",") != "repair" {
		t.Fatalf("unexpected repair summary: %#v", summary)
	}
	if summary.PromptTokensApproxTotal != 400 || summary.DurationMsTotal != 40 || summary.DurationMsAvg != 20 || summary.DurationMsMax != 30 {
		t.Fatalf("unexpected token/duration summary: %#v", summary)
	}
	if len(summary.TopFailingChecks) == 0 || summary.TopFailingChecks[0].Name != "workflow.plan_match" || summary.TopFailingChecks[0].Count != 2 {
		t.Fatalf("unexpected top failing checks: %#v", summary.TopFailingChecks)
	}
	if len(summary.FailureClasses) != 1 || summary.FailureClasses[0].Name != "workflow" {
		t.Fatalf("unexpected failure classes: %#v", summary.FailureClasses)
	}
	if len(summary.Providers) != 1 || summary.Providers[0].Name != "gemini" || summary.Providers[0].Count != 2 {
		t.Fatalf("unexpected providers: %#v", summary.Providers)
	}
	if len(summary.ReferencePolicies) != 1 || summary.ReferencePolicies[0].Name != "strict" {
		t.Fatalf("unexpected reference policies: %#v", summary.ReferencePolicies)
	}
}

func TestWriteReportsIncludesSummary(t *testing.T) {
	out := filepath.Join(t.TempDir(), "run.json")
	results := []EvalResult{{
		Name:               "a",
		Provider:           "gemini",
		Passed:             true,
		PromptTokensApprox: 20,
		DurationMs:         5,
	}}
	if err := WriteReports(out, results); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"summary"`) || !strings.Contains(string(data), `"prompt_tokens_approx_total": 20`) {
		t.Fatalf("json report missing summary:\n%s", data)
	}
	if read, err := ReadResults(out); err != nil || len(read) != 1 || read[0].Name != "a" {
		t.Fatalf("ReadResults() = %#v, %v", read, err)
	}
	md, err := os.ReadFile(strings.TrimSuffix(out, filepath.Ext(out)) + ".md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(md), "Prompt tokens approx total: `20`") {
		t.Fatalf("markdown report missing summary:\n%s", md)
	}
}

func TestWriteReportIncludesMetadataComparisonAndUsage(t *testing.T) {
	out := filepath.Join(t.TempDir(), "run.json")
	results := []EvalResult{{
		Name:               "a",
		Passed:             true,
		PromptTokensApprox: 20,
		TokenUsage: &TokenUsage{
			PromptReported:  18,
			Completion:      7,
			TotalReported:   25,
			ReportedCostUSD: 0.001,
		},
	}}
	comparison := CompareRuns(results, []EvalResult{{Name: "a", Passed: true, PromptTokensApprox: 10}}, "previous.json")
	report := BuildRunReport(results, ReportOptions{
		Metadata: RunMetadata{
			RunID:      "run",
			Commit:     "abcdef123456",
			Dirty:      true,
			EvalRoot:   "examples/eval",
			OutputPath: out,
		},
		Comparison: &comparison,
	})
	if err := WriteReport(out, report); err != nil {
		t.Fatal(err)
	}
	read, err := ReadReport(out)
	if err != nil {
		t.Fatal(err)
	}
	if read.Metadata.RunID != "run" || read.Metadata.Commit != "abcdef123456" || !read.Metadata.Dirty {
		t.Fatalf("metadata not preserved: %#v", read.Metadata)
	}
	if read.Summary.TotalTokensReported != 25 || read.Summary.ReportedCostUSD != 0.001 {
		t.Fatalf("usage summary not preserved: %#v", read.Summary)
	}
	if read.Comparison == nil || read.Comparison.PreviousPath != "previous.json" {
		t.Fatalf("comparison not preserved: %#v", read.Comparison)
	}
	md, err := os.ReadFile(strings.TrimSuffix(out, filepath.Ext(out)) + ".md")
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		"Run ID: `run`",
		"Commit: `abcdef123456` (dirty)",
		"## Run Comparison",
		"Provider-reported usage",
	} {
		if !strings.Contains(string(md), expected) {
			t.Fatalf("markdown missing %q:\n%s", expected, md)
		}
	}
}

func TestReadReportSupportsLegacyResultArray(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.json")
	data, err := json.Marshal([]EvalResult{{Name: "legacy", Passed: true}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := ReadReport(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Results) != 1 || report.Results[0].Name != "legacy" || report.Summary.Briefs != 1 {
		t.Fatalf("unexpected legacy report: %#v", report)
	}
}

func TestArchiveGeneratedDirsCopiesWorkspace(t *testing.T) {
	root := t.TempDir()
	generated := filepath.Join(root, "generated", "brief")
	if err := os.MkdirAll(filepath.Join(generated, "workflows"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(generated, "workflows", "intent.hcl"), []byte("workflow {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	archiveRoot := filepath.Join(root, "archive")
	results, err := ArchiveGeneratedDirs([]EvalResult{{
		Name:         "brief",
		GeneratedDir: generated,
	}}, archiveRoot, "run:1")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(archiveRoot, "run-1", "brief", "workflows", "intent.hcl")
	if _, err := os.Stat(want); err != nil {
		t.Fatalf("archived file missing: %v", err)
	}
	if results[0].GeneratedDir != filepath.Join(archiveRoot, "run-1", "brief") {
		t.Fatalf("generated dir = %q", results[0].GeneratedDir)
	}
}
