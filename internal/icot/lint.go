package icot

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/OpenUdon/openudon/internal/icot/elicitor"
	"github.com/OpenUdon/openudon/internal/synthesize"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

func runLint(args []string, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("icot lint", flag.ContinueOnError)
	fs.SetOutput(out)
	example := fs.String("example", "", "Example directory containing project.md")
	file := fs.String("file", "", "Path to a project.md file")
	jsonOutput := fs.Bool("json", false, "Write a structured JSON report to stdout")
	reportPath := fs.String("report", "", "Write a structured JSON report to this path")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: icot lint --example examples/<name>\n")
		fmt.Fprintf(fs.Output(), "       icot lint --file path/to/project.md\n\n")
		fmt.Fprintf(fs.Output(), "Runs deterministic project.md authoring checks without LLM or udon execution.\n\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	path := strings.TrimSpace(*file)
	if path == "" && strings.TrimSpace(*example) != "" {
		path = filepath.Join(strings.TrimSpace(*example), "project.md")
	}
	if path == "" {
		fmt.Fprintln(errOut, "--example or --file is required")
		return 2
	}
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	report := lintReport{
		Version:       lintReportVersion,
		Status:        statusPass,
		Example:       strings.TrimSpace(*example),
		File:          path,
		ProjectChecks: synthesize.LintProjectMarkdown(string(data)),
	}
	failed := false
	for _, check := range report.ProjectChecks {
		if check.Status == "fail" {
			failed = true
			if report.FailureFamily == "" {
				report.FailureFamily = failureFamilyForQualityCode(check.Code)
			}
		}
	}
	if !*jsonOutput {
		printLint(out, report.ProjectChecks)
	}
	intentExampleDir := strings.TrimSpace(*example)
	if intentExampleDir == "" && filepath.Base(path) == "project.md" {
		intentExampleDir = filepath.Dir(path)
	}
	if intentExampleDir != "" {
		intentPath := filepath.Join(intentExampleDir, "workflows", "intent.hcl")
		if _, statErr := os.Stat(intentPath); statErr == nil {
			intent, check, err := lintIntentReport(intentPath)
			report.IntentCheck = &check
			if !*jsonOutput {
				printIntentLint(out, check)
			}
			if err != nil {
				failed = true
				if report.FailureFamily == "" {
					report.FailureFamily = failureFamilyForQualityCode(check.Code)
				}
			} else {
				report.DriftChecks = elicitor.CompareProjectIntentDrift(string(data), intent)
				if !*jsonOutput {
					printDrift(out, report.DriftChecks)
				}
			}
		}
	}
	if failed {
		report.Status = statusFail
		if report.FailureFamily == "" {
			report.FailureFamily = failureUnknown
		}
	} else {
		report.Status = statusPass
	}
	if strings.TrimSpace(*reportPath) != "" {
		if err := writeJSONFile(*reportPath, report); err != nil {
			fmt.Fprintln(errOut, err)
			return 1
		}
	}
	if *jsonOutput {
		if err := json.NewEncoder(out).Encode(report); err != nil {
			fmt.Fprintln(errOut, err)
			return 1
		}
	}
	if failed {
		return 1
	}
	return 0
}

func printLint(out io.Writer, checks []synthesize.QualityCheck) bool {
	failed := false
	fmt.Fprintln(out, "icot: lint")
	for _, check := range checks {
		fmt.Fprintf(out, "  %s: %s - %s\n", check.Code, check.Status, check.Message)
		if check.Detail != "" {
			fmt.Fprintf(out, "    detail: %s\n", check.Detail)
		}
		if check.Status == "fail" {
			failed = true
		}
	}
	return failed
}

func lintIntent(out io.Writer, path string) (*rollout.Intent, error) {
	intent, check, err := lintIntentReport(path)
	printIntentLint(out, check)
	return intent, err
}

func lintIntentReport(path string) (*rollout.Intent, synthesize.QualityCheck, error) {
	intent, err := rollout.ParseIntentFile(path)
	if err != nil {
		return nil, synthesize.QualityCheck{Code: "intent.parse", Status: "fail", Message: "workflows/intent.hcl failed to parse", Detail: err.Error()}, err
	}
	missing := intent.MissingSlots()
	if len(missing) > 0 {
		err := fmt.Errorf("missing %s", strings.Join(missing, ", "))
		return nil, synthesize.QualityCheck{Code: "intent.slots", Status: "fail", Message: "workflows/intent.hcl is incomplete", Detail: err.Error()}, err
	}
	return intent, synthesize.QualityCheck{Code: "intent.parse", Status: "pass", Message: "workflows/intent.hcl parses"}, nil
}

func printIntentLint(out io.Writer, check synthesize.QualityCheck) {
	fmt.Fprintf(out, "  %s: %s - %s\n", check.Code, check.Status, check.Message)
	if check.Detail != "" {
		fmt.Fprintf(out, "    detail: %s\n", check.Detail)
	}
}

func printDrift(out io.Writer, checks []elicitor.DriftCheck) {
	if len(checks) == 0 {
		fmt.Fprintln(out, "  icot.drift: pass - project.md matches workflows/intent.hcl")
		return
	}
	for _, check := range checks {
		fmt.Fprintf(out, "  %s: warn - %s\n", check.Code, check.Message)
		if check.Detail != "" {
			fmt.Fprintf(out, "    detail: %s\n", check.Detail)
		}
	}
}
