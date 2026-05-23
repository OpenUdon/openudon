package icot

import (
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
	failed := printLint(out, synthesize.LintProjectMarkdown(string(data)))
	intentExampleDir := strings.TrimSpace(*example)
	if intentExampleDir == "" && filepath.Base(path) == "project.md" {
		intentExampleDir = filepath.Dir(path)
	}
	if intentExampleDir != "" {
		intentPath := filepath.Join(intentExampleDir, "workflows", "intent.hcl")
		if _, statErr := os.Stat(intentPath); statErr == nil {
			intent, err := lintIntent(out, intentPath)
			if err != nil {
				failed = true
			} else {
				printDrift(out, elicitor.CompareProjectIntentDrift(string(data), intent))
			}
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
	fmt.Fprintf(out, "  intent.parse: ")
	intent, err := rollout.ParseIntentFile(path)
	if err != nil {
		fmt.Fprintf(out, "fail - %v\n", err)
		return nil, err
	}
	missing := intent.MissingSlots()
	if len(missing) > 0 {
		fmt.Fprintf(out, "fail - missing %s\n", strings.Join(missing, ", "))
		return nil, fmt.Errorf("missing %s", strings.Join(missing, ", "))
	}
	fmt.Fprintln(out, "pass - workflows/intent.hcl parses")
	return intent, nil
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
