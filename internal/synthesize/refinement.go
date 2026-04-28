package synthesize

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const defaultMaxAttempts = 5

type RefinementReport struct {
	Status      string              `json:"status"`
	Example     string              `json:"example"`
	MaxAttempts int                 `json:"max_attempts"`
	Attempts    []RefinementAttempt `json:"attempts"`
	StopReason  string              `json:"stop_reason,omitempty"`
}

type RefinementAttempt struct {
	Number        int      `json:"number"`
	Action        string   `json:"action"`
	Status        string   `json:"status"`
	FailingChecks []string `json:"failing_checks,omitempty"`
	Detail        string   `json:"detail,omitempty"`
	StopReason    string   `json:"stop_reason,omitempty"`
}

func maxAttempts(value int) int {
	if value <= 0 {
		return defaultMaxAttempts
	}
	return value
}

func newRefinementReport(result Result, max int) *RefinementReport {
	return &RefinementReport{
		Status:      "running",
		Example:     result.ExampleDir,
		MaxAttempts: max,
	}
}

func (r *RefinementReport) addAttempt(number int, action string, report *QualityReport, err error, stopReason string) {
	attempt := RefinementAttempt{
		Number:     number,
		Action:     action,
		StopReason: stopReason,
	}
	if report != nil {
		attempt.Status = report.Status
		attempt.FailingChecks = failingCheckCodes(report)
		attempt.Detail = failingCheckDetails(report)
	}
	if err != nil {
		attempt.Status = "fail"
		attempt.Detail = err.Error()
	}
	if attempt.Status == "" {
		attempt.Status = "pass"
	}
	r.Attempts = append(r.Attempts, attempt)
	if stopReason != "" {
		r.StopReason = stopReason
	}
	if attempt.Status == "pass" {
		r.Status = "pass"
	} else {
		r.Status = "fail"
	}
}

func failingCheckCodes(report *QualityReport) []string {
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

func failingCheckDetails(report *QualityReport) string {
	if report == nil {
		return ""
	}
	var details []string
	for _, check := range report.Checks {
		if check.Status != "fail" {
			continue
		}
		if check.Detail != "" {
			details = append(details, fmt.Sprintf("%s: %s", check.Code, check.Detail))
		} else {
			details = append(details, check.Code)
		}
	}
	return strings.Join(details, "; ")
}

func firstFailingCheck(report *QualityReport) QualityCheck {
	if report == nil {
		return QualityCheck{}
	}
	for _, check := range report.Checks {
		if check.Status == "fail" {
			return check
		}
	}
	return QualityCheck{}
}

func qualityFailureSignature(report *QualityReport) string {
	return strings.Join(failingCheckCodes(report), "|")
}

func classifyRefinementAction(report *QualityReport) (string, bool) {
	check := firstFailingCheck(report)
	code := check.Code
	switch {
	case code == "":
		return "complete", true
	case code == "project.present", code == "artifacts.no_secrets", code == "credentials.bindings", code == "intent.runtime_policy":
		return "stop", true
	case strings.HasPrefix(code, "openapi."), code == "plan.gaps":
		return "discover_openapi", false
	case strings.HasPrefix(code, "intent."):
		return "regenerate_intent", false
	case strings.HasPrefix(code, "workflow."), strings.HasPrefix(code, "uws."), strings.HasPrefix(code, "review."):
		return "regenerate_workflow", false
	default:
		return "stop", true
	}
}

func writeRefinementReport(result Result, report *RefinementReport) error {
	if report == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(result.RefinementJSONPath), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(result.RefinementJSONPath, append(data, '\n'), 0o644); err != nil {
		return err
	}
	return os.WriteFile(result.RefinementMDPath, []byte(refinementMarkdown(report)), 0o644)
}

func refinementMarkdown(report *RefinementReport) string {
	var b strings.Builder
	b.WriteString("# Ramen Refinement Report\n\n")
	fmt.Fprintf(&b, "Status: `%s`\n\n", report.Status)
	fmt.Fprintf(&b, "Max attempts: `%d`\n\n", report.MaxAttempts)
	for _, attempt := range report.Attempts {
		fmt.Fprintf(&b, "- Attempt `%d` `%s` %s\n", attempt.Number, attempt.Action, attempt.Status)
		if len(attempt.FailingChecks) > 0 {
			fmt.Fprintf(&b, "  Failing checks: `%s`\n", strings.Join(attempt.FailingChecks, "`, `"))
		}
		if attempt.Detail != "" {
			fmt.Fprintf(&b, "  Detail: %s\n", attempt.Detail)
		}
		if attempt.StopReason != "" {
			fmt.Fprintf(&b, "  Stop reason: %s\n", attempt.StopReason)
		}
	}
	if report.StopReason != "" {
		fmt.Fprintf(&b, "\nStop reason: %s\n", report.StopReason)
	}
	return b.String()
}
