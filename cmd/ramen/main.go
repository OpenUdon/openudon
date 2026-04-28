package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/genelet/ramen/internal/config"
	evalpkg "github.com/genelet/ramen/internal/eval"
	"github.com/genelet/ramen/internal/synthesize"
	"github.com/genelet/ramen/internal/uwsvalidate"
)

const version = "0.1.0"

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: ramen <command>\n\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Commands:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  check     verify required sibling repositories are present\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  assess    assess existing example artifacts and write quality reports\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  build     regenerate workflow/UWS from an existing intent.hcl\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  eval      run synthesis eval briefs and write pass/fail reports\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  promote   export/validate UWS from an existing workflow.hcl\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  synthesize generate intent, workflow, UWS, and review artifacts for an example\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  validate  validate one UWS JSON/YAML file against the sibling UWS schema\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  version   print version\n")
	}
	flag.Parse()

	command := "check"
	if flag.NArg() > 0 {
		command = flag.Arg(0)
	}

	switch command {
	case "check":
		if err := config.CheckSiblings("."); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("ramen: required sibling repositories found")
	case "validate":
		if flag.NArg() < 2 {
			fmt.Fprintln(os.Stderr, "usage: ramen validate <uws-file>")
			os.Exit(2)
		}
		if err := uwsvalidate.ValidateFile("../uws/versions/1.0.0.json", flag.Arg(1)); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Printf("ramen: %s is valid UWS\n", flag.Arg(1))
	case "synthesize", "build", "promote", "assess":
		runArtifactCommand(command, flag.Args()[1:])
	case "eval":
		runEvalCommand(flag.Args()[1:])
	case "version":
		fmt.Println(version)
	case "-h", "--help", "help":
		flag.Usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", command)
		flag.Usage()
		os.Exit(2)
	}
}

func runEvalCommand(args []string) {
	fs := flag.NewFlagSet("eval", flag.ExitOnError)
	root := fs.String("root", "examples/eval", "Directory containing eval example subdirectories")
	name := fs.String("name", "", "Run a single eval brief by directory name")
	provider := fs.String("provider", "", "LLM provider: openai, anthropic, or gemini; defaults to udon runner env behavior")
	model := fs.String("model", "", "LLM model")
	timeout := fs.Duration("timeout", 2*time.Minute, "LLM generation timeout")
	maxAttempts := fs.Int("max-attempts", 5, "Maximum refinement attempts")
	temperature := fs.Float64("temperature", 0.2, "Intent generation temperature")
	concurrency := fs.Int("concurrency", 2, "Maximum concurrent eval runs")
	releaseGate := fs.Bool("release-gate", false, "Fail unless eval results meet local release criteria")
	compare := fs.String("compare", "", "Compare this eval report against a specific previous JSON report")
	noCompare := fs.Bool("no-compare", false, "Disable previous-run comparison")
	archiveDir := fs.String("archive-dir", "", "Copy generated eval workspaces under this directory for manual inspection")
	out := fs.String("out", evalpkg.DefaultOutputPath(time.Now()), "JSON report output path")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: ramen eval [--root examples/eval] [--name support-email] [--out eval/runs/<ts>.json] [--release-gate] [--compare eval/runs/<previous>.json] [--no-compare] [--archive-dir eval/artifacts]\n")
		fmt.Fprintf(fs.Output(), "\nRuns synthesis against temporary copies of eval briefs and writes JSON/Markdown reports with optional run comparison.\n")
		fmt.Fprintf(fs.Output(), "\nExamples:\n")
		fmt.Fprintf(fs.Output(), "  ramen eval --root examples/eval --provider gemini --model gemini-2.5-flash\n")
		fmt.Fprintf(fs.Output(), "  ramen eval --root examples/eval --name support-email --provider gemini --model gemini-2.5-flash\n")
		fmt.Fprintf(fs.Output(), "  ramen eval --root examples/eval --provider gemini --model gemini-2.5-flash --release-gate\n\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	opts := synthesize.Options{
		Provider:          *provider,
		Model:             *model,
		Timeout:           *timeout,
		MaxAttempts:       *maxAttempts,
		IntentTemperature: temperature,
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	var results []evalpkg.EvalResult
	if strings.TrimSpace(*name) != "" {
		results = []evalpkg.EvalResult{evalpkg.RunOne(ctx, filepath.Join(*root, strings.TrimSpace(*name)), opts)}
	} else {
		results = evalpkg.RunAll(ctx, *root, opts, *concurrency)
	}
	if len(results) == 0 {
		fmt.Fprintf(os.Stderr, "no eval briefs found under %s\n", *root)
		os.Exit(1)
	}
	runID := runIDFromOutput(*out)
	if strings.TrimSpace(*archiveDir) != "" {
		archived, err := evalpkg.ArchiveGeneratedDirs(results, *archiveDir, runID)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		results = archived
	}
	commit, dirty := gitMetadata()
	metadata := evalpkg.RunMetadata{
		RunID:       runID,
		Commit:      commit,
		Dirty:       dirty,
		EvalRoot:    *root,
		OutputPath:  *out,
		Provider:    strings.TrimSpace(*provider),
		Model:       strings.TrimSpace(*model),
		ReleaseGate: *releaseGate,
		ArchiveDir:  strings.TrimSpace(*archiveDir),
	}
	var comparison *evalpkg.RunComparison
	if !*noCompare {
		previousPath := strings.TrimSpace(*compare)
		if previousPath == "" {
			var err error
			previousPath, err = evalpkg.FindPreviousRun(*out)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		}
		if previousPath != "" {
			previous, err := evalpkg.ReadResults(previousPath)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			cmp := evalpkg.CompareRuns(results, previous, previousPath)
			comparison = &cmp
			metadata.ComparePath = previousPath
		}
	}
	report := evalpkg.BuildRunReport(results, evalpkg.ReportOptions{
		Metadata:   metadata,
		Comparison: comparison,
	})
	if err := evalpkg.WriteReport(*out, report); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("ramen: eval wrote %s\n", *out)
	fmt.Print(evalpkg.MarkdownReport(report))
	if *releaseGate {
		if err := evalpkg.ReleaseCriteriaError(results, evalpkg.DefaultReleaseCriteria()); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := evalpkg.ComparisonRegressionError(comparison); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}

func runIDFromOutput(outPath string) string {
	base := strings.TrimSuffix(filepath.Base(outPath), filepath.Ext(outPath))
	if strings.TrimSpace(base) == "" {
		return time.Now().UTC().Format("20060102T150405Z")
	}
	return base
}

func gitMetadata() (string, bool) {
	commitBytes, err := exec.Command("git", "rev-parse", "--short=12", "HEAD").Output()
	if err != nil {
		return "", false
	}
	statusBytes, err := exec.Command("git", "status", "--porcelain").Output()
	if err != nil {
		return strings.TrimSpace(string(commitBytes)), false
	}
	return strings.TrimSpace(string(commitBytes)), strings.TrimSpace(string(statusBytes)) != ""
}

func runArtifactCommand(command string, args []string) {
	fs := flag.NewFlagSet(command, flag.ExitOnError)
	example := fs.String("example", "", "Example directory containing project.md and artifact subdirectories")
	provider := fs.String("provider", "", "LLM provider: openai, anthropic, or gemini; defaults to udon runner env behavior")
	model := fs.String("model", "", "LLM model")
	timeout := fs.Duration("timeout", 2*time.Minute, "LLM generation timeout")
	maxAttempts := fs.Int("max-attempts", 5, "Maximum refinement attempts for synthesize/build")
	temperature := fs.Float64("temperature", 0.2, "Intent generation temperature for synthesize")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: ramen %s --example examples/<name> [--provider gemini --model gemini-2.5-flash]\n", command)
		fmt.Fprintf(fs.Output(), "\n%s\n", artifactCommandDescription(command))
		fmt.Fprintf(fs.Output(), "\nExamples:\n")
		switch command {
		case "synthesize":
			fmt.Fprintf(fs.Output(), "  ramen synthesize --example examples/support-email --provider gemini --model gemini-2.5-flash --max-attempts 5\n")
		case "build":
			fmt.Fprintf(fs.Output(), "  ramen build --example examples/support-email --provider gemini --model gemini-2.5-flash\n")
		case "promote":
			fmt.Fprintf(fs.Output(), "  ramen promote --example examples/support-email\n")
		case "assess":
			fmt.Fprintf(fs.Output(), "  ramen assess --example examples/support-email\n")
		}
		fmt.Fprintf(fs.Output(), "\nArtifacts:\n")
		fmt.Fprintf(fs.Output(), "  workflows/intent.hcl        structured intent generated from project.md\n")
		fmt.Fprintf(fs.Output(), "  workflows/workflow.hcl      udon workflow artifact\n")
		fmt.Fprintf(fs.Output(), "  workflows/workflow.uws.yaml exported UWS artifact\n")
		fmt.Fprintf(fs.Output(), "  expected/plan.json          expected operations, bindings, credentials, and control flow\n")
		fmt.Fprintf(fs.Output(), "  expected/review.md          trusted execution review evidence and handoff notes\n")
		fmt.Fprintf(fs.Output(), "  expected/quality.json       deterministic quality gate results\n\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	opts := synthesize.Options{
		ExampleDir:        *example,
		Provider:          *provider,
		Model:             *model,
		Timeout:           *timeout,
		MaxAttempts:       *maxAttempts,
		IntentTemperature: temperature,
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	var result *synthesize.Result
	var report *synthesize.QualityReport
	var err error
	switch command {
	case "synthesize":
		result, err = synthesize.Synthesize(ctx, opts)
	case "build":
		result, err = synthesize.Build(ctx, opts)
	case "promote":
		result, err = synthesize.Promote(ctx, opts)
	case "assess":
		report, err = synthesize.AssessContext(ctx, opts)
		if err == nil {
			printQuality(report)
			if !report.Passed() {
				os.Exit(1)
			}
			return
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		if report, qerr := synthesize.Assess(opts); qerr == nil {
			fmt.Fprintf(os.Stderr, "quality report: %s\n", report.Artifacts.QualityJSONPath)
		}
		os.Exit(1)
	}
	printResult(command, result)
}

func printResult(command string, result *synthesize.Result) {
	if result == nil {
		return
	}
	fmt.Printf("ramen: %s %s\n", command, result.ExampleDir)
	if result.PrimaryOpenAPI != "" {
		fmt.Printf("  openapi:  %s\n", result.PrimaryOpenAPI)
	}
	fmt.Printf("  intent:   %s\n", result.IntentPath)
	fmt.Printf("  workflow: %s\n", result.WorkflowPath)
	fmt.Printf("  uws:      %s\n", result.UWSPath)
	fmt.Printf("  plan:     %s\n", result.PlanJSONPath)
	fmt.Printf("  refine:   %s\n", result.RefinementJSONPath)
	fmt.Printf("  review:   %s\n", result.ReviewPath)
	fmt.Printf("  quality:  %s\n", result.QualityJSONPath)
}

func artifactCommandDescription(command string) string {
	switch command {
	case "synthesize":
		return "Generate intent, workflow, UWS, plan, review evidence, refinement report, and quality report from project.md."
	case "build":
		return "Regenerate workflow, UWS, review evidence, and quality reports from an existing workflows/intent.hcl."
	case "promote":
		return "Export and validate workflows/workflow.uws.yaml from an existing workflows/workflow.hcl."
	case "assess":
		return "Run deterministic quality gates against existing artifacts and rewrite expected/quality.{json,md}."
	default:
		return "Run a Ramen artifact command."
	}
}

func printQuality(report *synthesize.QualityReport) {
	if report == nil {
		return
	}
	fmt.Printf("ramen: quality %s\n", report.Status)
	fmt.Printf("  report: %s\n", report.Artifacts.QualityJSONPath)
	for _, check := range report.Checks {
		fmt.Printf("  %s: %s\n", check.Code, check.Status)
		if check.Status == "fail" {
			if check.Detail != "" {
				fmt.Printf("    detail: %s\n", check.Detail)
			}
			if next := nextActionForQualityCheck(check.Code); next != "" {
				fmt.Printf("    next: %s\n", next)
			}
		}
	}
}

func nextActionForQualityCheck(code string) string {
	switch {
	case code == "project.present":
		return "Create project.md from templates/project.md, then rerun synthesize or assess."
	case strings.HasPrefix(code, "project.authoring."):
		return "Fill the missing project.md section so synthesis decisions are auditable."
	case strings.HasPrefix(code, "openapi."):
		return "Add or fix OpenAPI documents under openapi/, or declare OpenAPI: none required when no API is needed."
	case code == "plan.gaps":
		return "Resolve missing operations, required parameters, or credential bindings in project.md or intent.hcl."
	case strings.HasPrefix(code, "intent."):
		return "Inspect workflows/intent.hcl and project.md; rerun synthesize when the brief needs regeneration."
	case code == "credentials.bindings", code == "workflow.credentials_bound":
		return "Name runtime credential bindings in project.md and ensure workflow request fields reference binding names, never secret values."
	case strings.HasPrefix(code, "workflow."):
		return "Inspect workflows/workflow.hcl against expected/plan.md, then rerun build or synthesize."
	case strings.HasPrefix(code, "uws."):
		return "Inspect workflows/workflow.uws.yaml, then rerun promote or build after fixing workflow.hcl."
	case code == "review.credential_bindings":
		return "Update Credentials and Secrets with binding names only, then regenerate review evidence with build/synthesize."
	case code == "review.approval_states", code == "review.sandbox_handoff", strings.HasPrefix(code, "review."), code == "side_effects.policy":
		return "Update Safety and Approval Boundary or regenerate review evidence with build/synthesize."
	case code == "artifacts.no_secrets":
		return "Remove literal secret-like values from artifacts; keep only credential binding names."
	default:
		return "Inspect expected/quality.md for details, fix the referenced artifact, and rerun assess."
	}
}
