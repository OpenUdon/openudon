package eval

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type RunReport struct {
	GeneratedAt string       `json:"generated_at"`
	Results     []EvalResult `json:"results"`
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
	var b strings.Builder
	fmt.Fprintf(&b, "# Ramen Eval Report\n\n")
	fmt.Fprintf(&b, "- Briefs: `%d`\n", len(results))
	fmt.Fprintf(&b, "- Pass rate: `%.1f%%`\n\n", passRate(results)*100)
	fmt.Fprintf(&b, "- Legacy extractJSON fallback: `%d`\n\n", legacyExtractCount(results))
	b.WriteString("| Brief | Status | Mode | Attempts | Failure class | Failing checks | Reference issues (A/W/B) | Duration |\n")
	b.WriteString("| --- | --- | --- | ---: | --- | --- | --- | ---: |\n")
	for _, result := range results {
		status := "fail"
		if result.Passed {
			status = "pass"
		}
		failing := strings.Join(result.FailingChecks, ", ")
		if failing == "" && result.Error != "" {
			failing = result.Error
		}
		fmt.Fprintf(&b, "| `%s` | %s | %s | %d | %s | %s | %s | %dms |\n",
			result.Name,
			status,
			escapeTable(result.Mode),
			result.AttemptsToPass,
			escapeTable(result.FailureClass),
			escapeTable(failing),
			referenceSummaryText(result.ReferenceSummary, result.ReferenceIssues),
			result.DurationMs,
		)
	}
	return b.String()
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
