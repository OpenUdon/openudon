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
	GeneratedAt string       `json:"generated_at"`
	Summary     RunSummary   `json:"summary"`
	Results     []EvalResult `json:"results"`
}

type RunSummary struct {
	Briefs                  int          `json:"briefs"`
	Passed                  int          `json:"passed"`
	Failed                  int          `json:"failed"`
	PassRate                float64      `json:"pass_rate"`
	LegacyFallbacks         int          `json:"legacy_fallbacks"`
	RepeatedRepairLoops     int          `json:"repeated_repair_loops"`
	PromptTokensApproxTotal int          `json:"prompt_tokens_approx_total"`
	DurationMsTotal         int64        `json:"duration_ms_total"`
	DurationMsAvg           int64        `json:"duration_ms_avg"`
	DurationMsMax           int64        `json:"duration_ms_max"`
	Providers               []SummaryRow `json:"providers,omitempty"`
	Models                  []SummaryRow `json:"models,omitempty"`
	Modes                   []SummaryRow `json:"modes,omitempty"`
	PromptVersions          []SummaryRow `json:"prompt_versions,omitempty"`
	FailureClasses          []SummaryRow `json:"failure_classes,omitempty"`
	TopFailingChecks        []SummaryRow `json:"top_failing_checks,omitempty"`
	RepeatedRepairBriefs    []string     `json:"repeated_repair_briefs,omitempty"`
}

type SummaryRow struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

func DefaultOutputPath(now time.Time) string {
	return filepath.Join("eval", "runs", now.UTC().Format("20060102T150405Z")+".json")
}

func WriteReports(outPath string, results []EvalResult) error {
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	report := RunReport{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Summary:     BuildRunSummary(results),
		Results:     results,
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(outPath, append(data, '\n'), 0o644); err != nil {
		return err
	}
	mdPath := strings.TrimSuffix(outPath, filepath.Ext(outPath)) + ".md"
	return os.WriteFile(mdPath, []byte(Markdown(results)), 0o644)
}

func ReadResults(path string) ([]EvalResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var report RunReport
	if err := json.Unmarshal(data, &report); err == nil && report.Results != nil {
		return report.Results, nil
	}
	var results []EvalResult
	if err := json.Unmarshal(data, &results); err != nil {
		return nil, err
	}
	return results, nil
}

func Markdown(results []EvalResult) string {
	summary := BuildRunSummary(results)
	var b strings.Builder
	fmt.Fprintf(&b, "# Ramen Eval Report\n\n")
	fmt.Fprintf(&b, "- Briefs: `%d`\n", summary.Briefs)
	fmt.Fprintf(&b, "- Pass rate: `%.1f%%` (`%d` pass / `%d` fail)\n", summary.PassRate*100, summary.Passed, summary.Failed)
	fmt.Fprintf(&b, "- Legacy extractJSON fallback: `%d`\n", summary.LegacyFallbacks)
	fmt.Fprintf(&b, "- Repeated repair loops: `%d`\n", summary.RepeatedRepairLoops)
	fmt.Fprintf(&b, "- Prompt tokens approx total: `%d`\n", summary.PromptTokensApproxTotal)
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
	if len(summary.RepeatedRepairBriefs) > 0 {
		fmt.Fprintf(&b, "- Repeated repair briefs: `%s`\n", strings.Join(summary.RepeatedRepairBriefs, "`, `"))
	}
	b.WriteString("\n| Brief | Status | Provider | Model | Prompt | Mode | Attempts | Failure class | Failing checks | Reference issues (A/W/B) | Tokens | Duration | Generated dir |\n")
	b.WriteString("| --- | --- | --- | --- | --- | --- | ---: | --- | --- | --- | ---: | ---: | --- |\n")
	for _, result := range results {
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
		fmt.Fprintf(&b, "| `%s` | %s | %s | %s | %s | %s | %d | %s | %s | %s | %d | %dms | %s |\n",
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
			result.PromptTokensApprox,
			result.DurationMs,
			escapeTable(result.GeneratedDir),
		)
	}
	return b.String()
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
		summary.DurationMsTotal += result.DurationMs
		if result.DurationMs > summary.DurationMsMax {
			summary.DurationMsMax = result.DurationMs
		}
		incrementSummary(providers, result.Provider)
		incrementSummary(models, result.Model)
		incrementSummary(modes, result.Mode)
		incrementSummary(promptVersions, result.PromptVersion)
		incrementSummary(failureClasses, result.FailureClass)
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
