package synthesize

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const defaultMaxAttempts = 5
const intentPromptVersion = "intent.v3"

type RefinementReport struct {
	Status         string              `json:"status"`
	Example        string              `json:"example"`
	MaxAttempts    int                 `json:"max_attempts"`
	PromptVersion  string              `json:"prompt_version"`
	PromptSnapshot string              `json:"prompt_snapshot,omitempty"`
	Attempts       []RefinementAttempt `json:"attempts"`
	StopReason     string              `json:"stop_reason,omitempty"`
}

type RefinementAttempt struct {
	Number        int      `json:"number"`
	Action        string   `json:"action"`
	Mode          string   `json:"mode,omitempty"`
	Status        string   `json:"status"`
	FailureClass  string   `json:"failure_class,omitempty"`
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
		Status:        "running",
		Example:       result.ExampleDir,
		MaxAttempts:   max,
		PromptVersion: intentPromptVersion,
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
	attempt.FailureClass = classifyFailureClass(report, err)
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

func (r *RefinementReport) setLastAttemptMode(mode string) {
	mode = strings.TrimSpace(mode)
	if r == nil || mode == "" || len(r.Attempts) == 0 {
		return
	}
	r.Attempts[len(r.Attempts)-1].Mode = mode
}

func classifyFailureClass(report *QualityReport, err error) string {
	if report != nil && !report.Passed() {
		return "validation"
	}
	if err == nil {
		return ""
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return "model"
	}
	lower := strings.ToLower(err.Error())
	switch {
	case strings.Contains(lower, "api returned status"),
		strings.Contains(lower, "extract intent json"),
		strings.Contains(lower, "decode intent json"),
		strings.Contains(lower, "render intent hcl"),
		strings.Contains(lower, "generate intent"),
		strings.Contains(lower, "generate workflow hcl"):
		return "model"
	case strings.Contains(lower, "generated intent referenced"),
		strings.Contains(lower, "project declares openapi"),
		strings.Contains(lower, "intent uses runtime not allowed"),
		strings.Contains(lower, "intent references unavailable"),
		strings.Contains(lower, "validate exported uws"):
		return "validation"
	case strings.Contains(lower, "read "),
		strings.Contains(lower, "write "),
		strings.Contains(lower, "mkdir"),
		strings.Contains(lower, "discover "):
		return "infra"
	default:
		return "infra"
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

func qualityFailureSignature(report *QualityReport) string {
	failures := failingChecks(report)
	if len(failures) == 0 {
		return ""
	}
	best := checkRepairMetadata{family: "unknown", priority: -1}
	for _, check := range failures {
		meta := qualityCheckRepairMetadata(check.Code)
		if meta.terminal {
			return meta.family
		}
		if meta.priority > best.priority || (meta.priority == best.priority && meta.family < best.family) {
			best = meta
		}
	}
	return best.family
}

func classifyRefinementAction(report *QualityReport) (string, bool) {
	failures := failingChecks(report)
	if len(failures) == 0 {
		return "complete", true
	}
	best := checkRepairMetadata{action: "stop", terminal: true, priority: -1}
	for _, check := range failures {
		meta := qualityCheckRepairMetadata(check.Code)
		if meta.terminal {
			return "stop", true
		}
		if meta.priority > best.priority {
			best = meta
		}
	}
	return best.action, best.terminal
}

type checkRepairMetadata struct {
	action   string
	terminal bool
	priority int
	family   string
}

func failingChecks(report *QualityReport) []QualityCheck {
	if report == nil {
		return nil
	}
	var out []QualityCheck
	for _, check := range report.Checks {
		if check.Status == "fail" {
			out = append(out, check)
		}
	}
	return out
}

func qualityCheckRepairMetadata(code string) checkRepairMetadata {
	family := qualityCheckFamily(code)
	switch {
	case code == "project.present", code == "artifacts.no_secrets", code == "credentials.bindings", code == "intent.runtime_policy":
		return checkRepairMetadata{action: "stop", terminal: true, priority: 100, family: family}
	case strings.HasPrefix(code, "openapi."), code == "plan.gaps":
		return checkRepairMetadata{action: "discover_openapi", priority: 30, family: family}
	case strings.HasPrefix(code, "intent."):
		return checkRepairMetadata{action: "regenerate_intent", priority: 20, family: family}
	case strings.HasPrefix(code, "workflow."), strings.HasPrefix(code, "uws."), strings.HasPrefix(code, "review."):
		return checkRepairMetadata{action: "regenerate_workflow", priority: 10, family: family}
	default:
		return checkRepairMetadata{action: "stop", terminal: true, priority: 100, family: family}
	}
}

func qualityCheckFamily(code string) string {
	code = strings.TrimSpace(code)
	if code == "" {
		return "unknown"
	}
	if code == "plan.gaps" {
		return "plan"
	}
	if prefix, _, ok := strings.Cut(code, "."); ok {
		return prefix
	}
	return code
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
	b.WriteString("# OpenUdon Refinement Report\n\n")
	fmt.Fprintf(&b, "Status: `%s`\n\n", report.Status)
	fmt.Fprintf(&b, "Max attempts: `%d`\n\n", report.MaxAttempts)
	if report.PromptVersion != "" {
		fmt.Fprintf(&b, "Prompt version: `%s`\n\n", report.PromptVersion)
	}
	if report.PromptSnapshot != "" {
		b.WriteString("Prompt snapshot captured in JSON report.\n\n")
	}
	for _, attempt := range report.Attempts {
		fmt.Fprintf(&b, "- Attempt `%d` `%s` %s\n", attempt.Number, attempt.Action, attempt.Status)
		if attempt.Mode != "" {
			fmt.Fprintf(&b, "  Mode: `%s`\n", attempt.Mode)
		}
		if attempt.FailureClass != "" {
			fmt.Fprintf(&b, "  Failure class: `%s`\n", attempt.FailureClass)
		}
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
