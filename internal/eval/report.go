package eval

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type RunReport struct {
	GeneratedAt        string              `json:"generated_at"`
	Metadata           RunMetadata         `json:"metadata,omitempty"`
	Summary            RunSummary          `json:"summary"`
	ProviderDriftWatch *ProviderDriftWatch `json:"provider_drift_watch,omitempty"`
	Comparison         *RunComparison      `json:"comparison,omitempty"`
	Results            []EvalResult        `json:"results"`
}

type RunMetadata struct {
	RunID       string `json:"run_id,omitempty"`
	Commit      string `json:"commit,omitempty"`
	Dirty       bool   `json:"dirty,omitempty"`
	EvalRoot    string `json:"eval_root,omitempty"`
	OutputPath  string `json:"output_path,omitempty"`
	Provider    string `json:"provider,omitempty"`
	Model       string `json:"model,omitempty"`
	ReleaseGate bool   `json:"release_gate,omitempty"`
	MinBriefs   int    `json:"min_briefs,omitempty"`
	ComparePath string `json:"compare_path,omitempty"`
	ArchiveDir  string `json:"archive_dir,omitempty"`
}

type ReportOptions struct {
	GeneratedAt time.Time
	Metadata    RunMetadata
	Comparison  *RunComparison
}

type RunSummary struct {
	Briefs                  int          `json:"briefs"`
	Passed                  int          `json:"passed"`
	Failed                  int          `json:"failed"`
	PassRate                float64      `json:"pass_rate"`
	LegacyFallbacks         int          `json:"legacy_fallbacks"`
	RepeatedRepairLoops     int          `json:"repeated_repair_loops"`
	PromptTokensApproxTotal int          `json:"prompt_tokens_approx_total"`
	PromptTokensReported    int          `json:"prompt_tokens_reported,omitempty"`
	CompletionTokens        int          `json:"completion_tokens,omitempty"`
	TotalTokensReported     int          `json:"total_tokens_reported,omitempty"`
	ReportedCostUSD         float64      `json:"reported_cost_usd,omitempty"`
	DurationMsTotal         int64        `json:"duration_ms_total"`
	DurationMsAvg           int64        `json:"duration_ms_avg"`
	DurationMsMax           int64        `json:"duration_ms_max"`
	Providers               []SummaryRow `json:"providers,omitempty"`
	Models                  []SummaryRow `json:"models,omitempty"`
	Modes                   []SummaryRow `json:"modes,omitempty"`
	PromptVersions          []SummaryRow `json:"prompt_versions,omitempty"`
	FailureClasses          []SummaryRow `json:"failure_classes,omitempty"`
	TopFailingChecks        []SummaryRow `json:"top_failing_checks,omitempty"`
	ReferencePolicies       []SummaryRow `json:"reference_policies,omitempty"`
	RepeatedRepairBriefs    []string     `json:"repeated_repair_briefs,omitempty"`
}

type SummaryRow struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type ProviderDriftWatch struct {
	Status                  string                 `json:"status"`
	StructuredFallbacks     int                    `json:"structured_fallbacks"`
	StructuredFallbackDelta int                    `json:"structured_fallback_delta,omitempty"`
	ProviderFailures        []ProviderDriftFailure `json:"provider_failures,omitempty"`
	ModelAvailability       string                 `json:"model_availability"`
	ModelAvailabilityDetail string                 `json:"model_availability_detail,omitempty"`
	MaxAttemptsToPass       int                    `json:"max_attempts_to_pass"`
	RepeatedRepairLoops     int                    `json:"repeated_repair_loops"`
	AttemptRegressions      []BriefIntDelta        `json:"attempt_regressions,omitempty"`
	ReleaseGateEvaluated    bool                   `json:"release_gate_evaluated"`
	ReleaseGateFailures     []string               `json:"release_gate_failures,omitempty"`
}

type ProviderDriftFailure struct {
	Brief  string `json:"brief"`
	Signal string `json:"signal"`
	Detail string `json:"detail"`
}

func DefaultOutputPath(now time.Time) string {
	return filepath.Join("eval", "runs", now.UTC().Format("20060102T150405Z")+".json")
}

func BuildRunReport(results []EvalResult, opts ReportOptions) RunReport {
	generatedAt := opts.GeneratedAt
	if generatedAt.IsZero() {
		generatedAt = time.Now().UTC()
	}
	summary := BuildRunSummary(results)
	report := RunReport{
		GeneratedAt: generatedAt.UTC().Format(time.RFC3339),
		Metadata:    opts.Metadata,
		Summary:     summary,
		Comparison:  opts.Comparison,
		Results:     results,
	}
	watch := BuildProviderDriftWatch(report, summary)
	report.ProviderDriftWatch = &watch
	return report
}

func WriteReports(outPath string, results []EvalResult) error {
	return WriteReport(outPath, BuildRunReport(results, ReportOptions{}))
}

func WriteReport(outPath string, report RunReport) error {
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	if report.GeneratedAt == "" {
		report.GeneratedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if report.Summary.Briefs == 0 && len(report.Results) > 0 {
		report.Summary = BuildRunSummary(report.Results)
	}
	if report.ProviderDriftWatch == nil {
		watch := BuildProviderDriftWatch(report, report.Summary)
		report.ProviderDriftWatch = &watch
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(outPath, append(data, '\n'), 0o644); err != nil {
		return err
	}
	mdPath := strings.TrimSuffix(outPath, filepath.Ext(outPath)) + ".md"
	return os.WriteFile(mdPath, []byte(MarkdownReport(report)), 0o644)
}

func ReadReport(path string) (RunReport, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return RunReport{}, err
	}
	var report RunReport
	if err := json.Unmarshal(data, &report); err == nil && report.Results != nil {
		if report.Summary.Briefs == 0 && len(report.Results) > 0 {
			report.Summary = BuildRunSummary(report.Results)
		}
		return report, nil
	}
	var results []EvalResult
	if err := json.Unmarshal(data, &results); err != nil {
		return RunReport{}, err
	}
	return RunReport{
		Summary: BuildRunSummary(results),
		Results: results,
	}, nil
}

func ReadResults(path string) ([]EvalResult, error) {
	report, err := ReadReport(path)
	if err != nil {
		return nil, err
	}
	return report.Results, nil
}

func Markdown(results []EvalResult) string {
	return MarkdownReport(BuildRunReport(results, ReportOptions{}))
}

func MarkdownReport(report RunReport) string {
	summary := report.Summary
	if summary.Briefs == 0 && len(report.Results) > 0 {
		summary = BuildRunSummary(report.Results)
	}
	watch := report.ProviderDriftWatch
	if watch == nil {
		built := BuildProviderDriftWatch(report, summary)
		watch = &built
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# OpenUdon Eval Report\n\n")
	if report.Metadata.RunID != "" {
		fmt.Fprintf(&b, "- Run ID: `%s`\n", report.Metadata.RunID)
	}
	if report.GeneratedAt != "" {
		fmt.Fprintf(&b, "- Generated at: `%s`\n", report.GeneratedAt)
	}
	if report.Metadata.Commit != "" {
		fmt.Fprintf(&b, "- Commit: `%s`", report.Metadata.Commit)
		if report.Metadata.Dirty {
			b.WriteString(" (dirty)")
		}
		b.WriteString("\n")
	}
	if report.Metadata.EvalRoot != "" {
		fmt.Fprintf(&b, "- Eval root: `%s`\n", report.Metadata.EvalRoot)
	}
	fmt.Fprintf(&b, "- Briefs: `%d`\n", summary.Briefs)
	fmt.Fprintf(&b, "- Pass rate: `%.1f%%` (`%d` pass / `%d` fail)\n", summary.PassRate*100, summary.Passed, summary.Failed)
	fmt.Fprintf(&b, "- Legacy extractJSON fallback: `%d`\n", summary.LegacyFallbacks)
	fmt.Fprintf(&b, "- Repeated repair loops: `%d`\n", summary.RepeatedRepairLoops)
	fmt.Fprintf(&b, "- Prompt tokens approx total: `%d`\n", summary.PromptTokensApproxTotal)
	if summary.TotalTokensReported > 0 || summary.ReportedCostUSD > 0 {
		fmt.Fprintf(&b, "- Provider-reported usage: prompt `%d`, completion `%d`, total `%d`, cost `$%.6f`\n",
			summary.PromptTokensReported,
			summary.CompletionTokens,
			summary.TotalTokensReported,
			summary.ReportedCostUSD,
		)
	}
	fmt.Fprintf(&b, "- Duration: avg `%dms`, max `%dms`, total `%dms`\n", summary.DurationMsAvg, summary.DurationMsMax, summary.DurationMsTotal)
	if len(summary.Modes) > 0 {
		fmt.Fprintf(&b, "- Modes: %s\n", summaryRowsText(summary.Modes))
	}
	if len(summary.Providers) > 0 {
		fmt.Fprintf(&b, "- Providers: %s\n", summaryRowsText(summary.Providers))
	}
	if len(summary.Models) > 0 {
		fmt.Fprintf(&b, "- Models: %s\n", summaryRowsText(summary.Models))
	}
	if len(summary.PromptVersions) > 0 {
		fmt.Fprintf(&b, "- Prompt versions: %s\n", summaryRowsText(summary.PromptVersions))
	}
	if len(summary.FailureClasses) > 0 {
		fmt.Fprintf(&b, "- Failure classes: %s\n", summaryRowsText(summary.FailureClasses))
	}
	if len(summary.TopFailingChecks) > 0 {
		fmt.Fprintf(&b, "- Top failing checks: %s\n", summaryRowsText(summary.TopFailingChecks))
	}
	if len(summary.ReferencePolicies) > 0 {
		fmt.Fprintf(&b, "- Reference policies: %s\n", summaryRowsText(summary.ReferencePolicies))
	}
	if len(summary.RepeatedRepairBriefs) > 0 {
		fmt.Fprintf(&b, "- Repeated repair briefs: `%s`\n", strings.Join(summary.RepeatedRepairBriefs, "`, `"))
	}
	b.WriteString("\n## Provider Drift Watch\n\n")
	b.WriteString(providerDriftWatchMarkdown(report, summary, *watch))
	if report.Comparison != nil {
		b.WriteString("\n## Run Comparison\n\n")
		b.WriteString(comparisonMarkdown(*report.Comparison))
	}
	b.WriteString("\n| Brief | Status | Provider | Model | Prompt | Mode | Attempts | Failure class | Failing checks | Reference issues (A/W/B) | Reference policy | Tokens | Duration | Generated dir |\n")
	b.WriteString("| --- | --- | --- | --- | --- | --- | ---: | --- | --- | --- | --- | ---: | ---: | --- |\n")
	for _, result := range report.Results {
		status := "fail"
		if result.Passed {
			status = "pass"
		}
		failing := strings.Join(result.FailingChecks, ", ")
		if failing == "" && result.Error != "" {
			failing = result.Error
		}
		attempts := result.AttemptCount
		if attempts == 0 {
			attempts = result.AttemptsToPass
		}
		fmt.Fprintf(&b, "| `%s` | %s | %s | %s | %s | %s | %d | %s | %s | %s | %s | %d | %dms | %s |\n",
			result.Name,
			status,
			escapeTable(result.Provider),
			escapeTable(result.Model),
			escapeTable(result.PromptVersion),
			escapeTable(result.Mode),
			attempts,
			escapeTable(result.FailureClass),
			escapeTable(failing),
			referenceSummaryText(result.ReferenceSummary, result.ReferenceIssues),
			escapeTable(referencePolicyText(result.ReferencePolicy)),
			result.PromptTokensApprox,
			result.DurationMs,
			escapeTable(result.GeneratedDir),
		)
	}
	if details := referenceIssueDetailsMarkdown(report.Results); details != "" {
		b.WriteString("\n## Reference Issue Details\n\n")
		b.WriteString(details)
	}
	return b.String()
}

func providerDriftWatchMarkdown(report RunReport, summary RunSummary, watch ProviderDriftWatch) string {
	var b strings.Builder
	fmt.Fprintf(&b, "- Status: `%s`\n", watch.Status)
	fmt.Fprintf(&b, "- Structured fallback count: `%d`\n", watch.StructuredFallbacks)
	if watch.StructuredFallbackDelta != 0 {
		fmt.Fprintf(&b, "- Structured fallback delta: `%+d`\n", watch.StructuredFallbackDelta)
	}
	fmt.Fprintf(&b, "- Rate/transient/model failures: %s\n", providerFailureWatchText(watch.ProviderFailures))
	fmt.Fprintf(&b, "- Model availability: %s\n", modelAvailabilityWatchText(report, summary, watch))
	fmt.Fprintf(&b, "- Attempts-to-pass: max `%d`, repeated repair loops `%d`\n", watch.MaxAttemptsToPass, watch.RepeatedRepairLoops)
	if len(watch.AttemptRegressions) > 0 {
		fmt.Fprintf(&b, "- Attempt regressions: %s\n", briefIntDeltasMarkdown(watch.AttemptRegressions))
	}
	if watch.ReleaseGateEvaluated {
		if len(watch.ReleaseGateFailures) > 0 {
			fmt.Fprintf(&b, "- Release-gate failures: `%s`\n", strings.Join(watch.ReleaseGateFailures, "`; `"))
		} else {
			b.WriteString("- Release-gate failures: none\n")
		}
	} else {
		b.WriteString("- Release-gate failures: not evaluated (`release_gate=false`)\n")
	}
	return b.String()
}

func BuildProviderDriftWatch(report RunReport, summary RunSummary) ProviderDriftWatch {
	if summary.Briefs == 0 && len(report.Results) > 0 {
		summary = BuildRunSummary(report.Results)
	}
	watch := ProviderDriftWatch{
		Status:               "clear",
		StructuredFallbacks:  summary.LegacyFallbacks,
		ProviderFailures:     providerDriftFailures(report.Results),
		ModelAvailability:    "recorded",
		MaxAttemptsToPass:    maxAttemptCount(report.Results),
		RepeatedRepairLoops:  summary.RepeatedRepairLoops,
		ReleaseGateEvaluated: report.Metadata.ReleaseGate,
	}
	if report.Comparison != nil {
		watch.StructuredFallbackDelta = report.Comparison.LegacyFallbackDelta
		watch.AttemptRegressions = append(watch.AttemptRegressions, report.Comparison.AttemptRegressions...)
	}
	if modelUnavailableDetected(report.Results) {
		watch.ModelAvailability = "error"
		watch.ModelAvailabilityDetail = "model availability error detected"
	} else if report.Metadata.Provider == "" && report.Metadata.Model == "" && len(summary.Providers) == 0 && len(summary.Models) == 0 {
		watch.ModelAvailability = "not_recorded"
		watch.ModelAvailabilityDetail = "provider/model not recorded"
	} else {
		watch.ModelAvailabilityDetail = modelAvailabilityDetail(report, summary)
	}
	if report.Metadata.ReleaseGate {
		watch.ReleaseGateFailures = releaseGateWatchFailures(report)
	}
	if watch.StructuredFallbacks > 0 ||
		watch.StructuredFallbackDelta > 0 ||
		len(watch.ProviderFailures) > 0 ||
		watch.ModelAvailability == "error" ||
		watch.MaxAttemptsToPass > DefaultReleaseCriteria().MaxAttemptsPerBrief ||
		len(watch.AttemptRegressions) > 0 ||
		len(watch.ReleaseGateFailures) > 0 {
		watch.Status = "drift_detected"
	}
	return watch
}

func releaseGateWatchFailures(report RunReport) []string {
	var failures []string
	criteria := DefaultReleaseCriteria()
	criteria.MinBriefs = report.Metadata.MinBriefs
	if err := ReleaseCriteriaError(report.Results, criteria); err != nil {
		failures = append(failures, err.Error())
	}
	if err := ComparisonRegressionError(report.Comparison); err != nil {
		failures = append(failures, err.Error())
	}
	return failures
}

func providerDriftFailures(results []EvalResult) []ProviderDriftFailure {
	var failures []ProviderDriftFailure
	for _, result := range results {
		if result.Passed {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(result.FailureClass), "model") || providerErrorLooksTransient(result.Error) {
			name := strings.TrimSpace(result.Name)
			if name == "" {
				name = "<unnamed>"
			}
			detail := strings.TrimSpace(result.Error)
			if detail == "" {
				detail = strings.TrimSpace(result.FailureClass)
			}
			if detail == "" {
				detail = "provider/model failure"
			}
			failures = append(failures, ProviderDriftFailure{
				Brief:  name,
				Signal: providerDriftSignal(result),
				Detail: detail,
			})
		}
	}
	sort.Slice(failures, func(i, j int) bool {
		if failures[i].Brief != failures[j].Brief {
			return failures[i].Brief < failures[j].Brief
		}
		return failures[i].Detail < failures[j].Detail
	})
	return failures
}

func providerFailureWatchText(failures []ProviderDriftFailure) string {
	if len(failures) == 0 {
		return "none detected"
	}
	parts := make([]string, 0, len(failures))
	for _, failure := range failures {
		parts = append(parts, fmt.Sprintf("%s: %s", failure.Brief, failure.Detail))
	}
	return "`" + strings.Join(parts, "`; `") + "`"
}

func providerDriftSignal(result EvalResult) string {
	if modelUnavailableDetected([]EvalResult{result}) {
		return "model_availability"
	}
	if providerErrorLooksTransient(result.Error) {
		return "rate_or_transient"
	}
	if strings.EqualFold(strings.TrimSpace(result.FailureClass), "model") {
		return "model"
	}
	return "provider"
}

func modelAvailabilityWatchText(report RunReport, summary RunSummary, watch ProviderDriftWatch) string {
	if watch.ModelAvailabilityDetail != "" {
		return watch.ModelAvailabilityDetail
	}
	return modelAvailabilityDetail(report, summary)
}

func modelAvailabilityDetail(report RunReport, summary RunSummary) string {
	var parts []string
	if report.Metadata.Provider != "" {
		parts = append(parts, "provider `"+report.Metadata.Provider+"`")
	}
	if report.Metadata.Model != "" {
		parts = append(parts, "model `"+report.Metadata.Model+"`")
	}
	if len(parts) == 0 && len(summary.Providers) > 0 {
		parts = append(parts, "providers "+summaryRowsText(summary.Providers))
	}
	if len(summary.Models) > 0 {
		parts = append(parts, "models "+summaryRowsText(summary.Models))
	}
	if len(parts) == 0 {
		return "provider/model not recorded"
	}
	return strings.Join(parts, ", ")
}

func modelUnavailableDetected(results []EvalResult) bool {
	for _, result := range results {
		lower := strings.ToLower(result.Error)
		if strings.Contains(lower, "model") && (strings.Contains(lower, "unavailable") ||
			strings.Contains(lower, "not found") ||
			strings.Contains(lower, "not supported") ||
			strings.Contains(lower, "permission")) {
			return true
		}
	}
	return false
}

func providerErrorLooksTransient(text string) bool {
	lower := strings.ToLower(text)
	for _, needle := range []string{
		"429",
		"rate limit",
		"quota",
		"timeout",
		"deadline exceeded",
		"temporarily unavailable",
		"unavailable",
		"transient",
		"503",
		"502",
		"500",
	} {
		if strings.Contains(lower, needle) {
			return true
		}
	}
	return false
}

func maxAttemptCount(results []EvalResult) int {
	maxAttempts := 0
	for _, result := range results {
		attempts := result.AttemptCount
		if attempts == 0 {
			attempts = result.AttemptsToPass
		}
		if attempts > maxAttempts {
			maxAttempts = attempts
		}
	}
	return maxAttempts
}

func BuildRunSummary(results []EvalResult) RunSummary {
	summary := RunSummary{
		Briefs:          len(results),
		PassRate:        passRate(results),
		LegacyFallbacks: legacyExtractCount(results),
	}
	providers := map[string]int{}
	models := map[string]int{}
	modes := map[string]int{}
	promptVersions := map[string]int{}
	failureClasses := map[string]int{}
	failingChecks := map[string]int{}
	referencePolicies := map[string]int{}
	for _, result := range results {
		if result.Passed {
			summary.Passed++
		} else {
			summary.Failed++
		}
		if result.RepeatedRepairLoop || result.AttemptCount > 1 || result.AttemptsToPass > 1 {
			summary.RepeatedRepairLoops++
			if strings.TrimSpace(result.Name) != "" {
				summary.RepeatedRepairBriefs = append(summary.RepeatedRepairBriefs, result.Name)
			}
		}
		summary.PromptTokensApproxTotal += result.PromptTokensApprox
		if result.TokenUsage != nil {
			summary.PromptTokensReported += result.TokenUsage.PromptReported
			summary.CompletionTokens += result.TokenUsage.Completion
			summary.TotalTokensReported += result.TokenUsage.TotalReported
			summary.ReportedCostUSD += result.TokenUsage.ReportedCostUSD
		}
		summary.DurationMsTotal += result.DurationMs
		if result.DurationMs > summary.DurationMsMax {
			summary.DurationMsMax = result.DurationMs
		}
		incrementSummary(providers, result.Provider)
		incrementSummary(models, result.Model)
		incrementSummary(modes, result.Mode)
		incrementSummary(promptVersions, result.PromptVersion)
		incrementSummary(failureClasses, result.FailureClass)
		incrementSummary(referencePolicies, referencePolicyText(result.ReferencePolicy))
		for _, check := range result.FailingChecks {
			incrementSummary(failingChecks, check)
		}
	}
	if summary.Briefs > 0 {
		summary.DurationMsAvg = summary.DurationMsTotal / int64(summary.Briefs)
	}
	sort.Strings(summary.RepeatedRepairBriefs)
	summary.Providers = summaryRows(providers, 0)
	summary.Models = summaryRows(models, 0)
	summary.Modes = summaryRows(modes, 0)
	summary.PromptVersions = summaryRows(promptVersions, 0)
	summary.FailureClasses = summaryRows(failureClasses, 0)
	summary.TopFailingChecks = summaryRows(failingChecks, 5)
	summary.ReferencePolicies = summaryRows(referencePolicies, 0)
	return summary
}

func incrementSummary(counts map[string]int, name string) {
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	counts[name]++
}

func summaryRows(counts map[string]int, limit int) []SummaryRow {
	rows := make([]SummaryRow, 0, len(counts))
	for name, count := range counts {
		rows = append(rows, SummaryRow{Name: name, Count: count})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Count != rows[j].Count {
			return rows[i].Count > rows[j].Count
		}
		return rows[i].Name < rows[j].Name
	})
	if limit > 0 && len(rows) > limit {
		return rows[:limit]
	}
	return rows
}

func summaryRowsText(rows []SummaryRow) string {
	parts := make([]string, 0, len(rows))
	for _, row := range rows {
		parts = append(parts, fmt.Sprintf("`%s`=%d", row.Name, row.Count))
	}
	return strings.Join(parts, ", ")
}

func referenceSummaryText(summary ReferenceSummary, issues []CompareIssue) string {
	if summary == (ReferenceSummary{}) {
		summary = summarizeReferenceIssues(issues)
	}
	if summary == (ReferenceSummary{}) {
		return "0/0/0"
	}
	return fmt.Sprintf("%d/%d/%d", summary.Advisory, summary.Warning, summary.Blocking)
}

func referencePolicyText(policy *ReferencePolicy) string {
	if policy == nil || policy.IsZero() {
		return ""
	}
	if strings.TrimSpace(policy.Mode) != "" {
		return strings.TrimSpace(policy.Mode)
	}
	return "custom"
}

func referenceIssueDetailsMarkdown(results []EvalResult) string {
	var b strings.Builder
	for _, result := range results {
		if len(result.ReferenceIssues) == 0 && (result.ReferencePolicy == nil || len(result.ReferencePolicy.Notes) == 0) {
			continue
		}
		fmt.Fprintf(&b, "- `%s`", result.Name)
		if policy := referencePolicyText(result.ReferencePolicy); policy != "" {
			fmt.Fprintf(&b, " policy `%s`", policy)
		}
		b.WriteString("\n")
		if result.ReferencePolicy != nil {
			for _, note := range result.ReferencePolicy.Notes {
				note = strings.TrimSpace(note)
				if note != "" {
					fmt.Fprintf(&b, "  - policy: %s\n", escapeTable(note))
				}
			}
		}
		for _, issue := range result.ReferenceIssues {
			severity := normalizedReferenceSeverity(issue)
			fmt.Fprintf(&b, "  - %s `%s`: %s", severity, issue.Code, escapeTable(issue.Detail))
			if note := strings.TrimSpace(issue.Note); note != "" {
				fmt.Fprintf(&b, " (%s)", escapeTable(note))
			}
			b.WriteString("\n")
		}
	}
	return b.String()
}

func comparisonMarkdown(comparison RunComparison) string {
	var b strings.Builder
	if comparison.PreviousPath != "" {
		fmt.Fprintf(&b, "- Previous run: `%s`\n", comparison.PreviousPath)
	}
	fmt.Fprintf(&b, "- Pass rate delta: `%+.1f%%` (`%.1f%%` -> `%.1f%%`)\n",
		comparison.PassRateDelta*100,
		comparison.PreviousPassRate*100,
		comparison.CurrentPassRate*100,
	)
	fmt.Fprintf(&b, "- Legacy fallback delta: `%+d`\n", comparison.LegacyFallbackDelta)
	fmt.Fprintf(&b, "- Blocking reference delta: `%+d`\n", comparison.BlockingReferenceDelta)
	fmt.Fprintf(&b, "- Prompt token delta: `%+d`\n", comparison.PromptTokensApproxDelta)
	fmt.Fprintf(&b, "- Duration delta: `%+dms`\n", comparison.DurationMsDelta)
	if len(comparison.NewlyFailingBriefs) > 0 {
		fmt.Fprintf(&b, "- Newly failing briefs: `%s`\n", strings.Join(comparison.NewlyFailingBriefs, "`, `"))
	}
	if len(comparison.FixedBriefs) > 0 {
		fmt.Fprintf(&b, "- Fixed briefs: `%s`\n", strings.Join(comparison.FixedBriefs, "`, `"))
	}
	if len(comparison.AttemptRegressions) > 0 {
		fmt.Fprintf(&b, "- Attempt regressions: %s\n", briefIntDeltasMarkdown(comparison.AttemptRegressions))
	}
	if len(comparison.AttemptImprovements) > 0 {
		fmt.Fprintf(&b, "- Attempt improvements: %s\n", briefIntDeltasMarkdown(comparison.AttemptImprovements))
	}
	if len(comparison.NewFailingChecks) > 0 {
		fmt.Fprintf(&b, "- New failing checks: `%s`\n", strings.Join(comparison.NewFailingChecks, "`, `"))
	}
	if len(comparison.ResolvedFailingChecks) > 0 {
		fmt.Fprintf(&b, "- Resolved failing checks: `%s`\n", strings.Join(comparison.ResolvedFailingChecks, "`, `"))
	}
	if comparison.HasRegression {
		b.WriteString("- Regression: `true`\n")
	} else {
		b.WriteString("- Regression: `false`\n")
	}
	return b.String()
}

func briefIntDeltasMarkdown(deltas []BriefIntDelta) string {
	parts := make([]string, 0, len(deltas))
	for _, delta := range deltas {
		parts = append(parts, fmt.Sprintf("`%s` %d -> %d", delta.Name, delta.Previous, delta.Current))
	}
	return strings.Join(parts, ", ")
}

func legacyExtractCount(results []EvalResult) int {
	var count int
	for _, result := range results {
		if result.UsedLegacyExtract || result.Mode == "legacy" {
			count++
		}
	}
	return count
}

func escapeTable(value string) string {
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "|", "\\|")
	return value
}
