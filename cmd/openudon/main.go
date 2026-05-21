package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/OpenUdon/openudon/internal/config"
	evalpkg "github.com/OpenUdon/openudon/internal/eval"
	"github.com/OpenUdon/openudon/internal/localcheck"
	"github.com/OpenUdon/openudon/internal/n8nbridge"
	"github.com/OpenUdon/openudon/internal/readiness"
	"github.com/OpenUdon/openudon/internal/synthesize"
	"github.com/OpenUdon/openudon/internal/tfconvert"
	"github.com/OpenUdon/openudon/internal/trustedrunner"
	uwsprofile "github.com/OpenUdon/openudon/internal/uwsexec"
	"github.com/OpenUdon/openudon/internal/uwsschema"
	"github.com/OpenUdon/openudon/internal/uwsvalidate"
)

const version = "0.1.0"

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: openudon <command>\n\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Commands:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  check     verify required sibling repositories are present\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  assess    assess existing example artifacts and write quality reports\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  approval-template print approval JSON for a validated handoff package\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  build     regenerate workflow/UWS from an existing intent.hcl\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  catalog   inspect first-class provider catalog metadata\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  check-apitools-boundary verify OpenUdon repository boundaries\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  check-doc-memory verify local memory-bank and evolution harness files\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  convert   generate draft review scaffolding from supported source formats\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  eval      run synthesis eval briefs and write pass/fail reports\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  n8n-bridge validate review-first n8n pattern summary evidence\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  promote   export/validate UWS from an existing workflow.hcl\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  readiness write local private-checkout and deterministic-gate readiness report\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  run       validate approval gates and invoke a trusted executor handoff\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  synthesize generate intent, workflow, UWS, and review artifacts for an example\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  validate  validate one UWS JSON/YAML file or a directory of UWS artifacts\n")
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
		fmt.Println("openudon: required sibling repositories found")
	case "check-apitools-boundary":
		if err := localcheck.CheckAPIToolsBoundary("."); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("openudon: repository boundary check passed")
	case "check-doc-memory":
		if err := runCheckDocMemory(".", os.Stdout, os.Stderr); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "convert":
		runConvertCommand(flag.Args()[1:])
	case "catalog":
		runCatalogCommand(flag.Args()[1:])
	case "validate":
		if flag.NArg() < 2 {
			fmt.Fprintln(os.Stderr, "usage: openudon validate [--allow-empty] <uws-file-or-dir>")
			os.Exit(2)
		}
		runValidateCommand(flag.Args()[1:])
	case "synthesize", "build", "promote", "assess":
		runArtifactCommand(command, flag.Args()[1:])
	case "run":
		runTrustedCommand(flag.Args()[1:])
	case "approval-template":
		runApprovalTemplateCommand(flag.Args()[1:])
	case "eval":
		runEvalCommand(flag.Args()[1:])
	case "n8n-bridge":
		runN8nBridgeCommand(flag.Args()[1:])
	case "readiness":
		runReadinessCommand(flag.Args()[1:])
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

func runCheckDocMemory(root string, out, errOut io.Writer) error {
	result, err := localcheck.CheckDocMemory(root)
	if err != nil {
		return err
	}
	fmt.Fprintln(out, "openudon: doc memory check passed")
	for _, file := range result.CheckedFiles {
		fmt.Fprintf(out, "openudon: checked %s\n", file)
	}
	for _, warning := range result.Warnings {
		fmt.Fprintf(errOut, "openudon: warning: %s\n", warning)
	}
	return nil
}

type repeatedStringFlag []string

func (f *repeatedStringFlag) String() string {
	return strings.Join(*f, ",")
}

func (f *repeatedStringFlag) Set(value string) error {
	*f = append(*f, value)
	return nil
}

func runConvertCommand(args []string) {
	if len(args) == 0 || args[0] != "tf" {
		fmt.Fprintln(os.Stderr, "usage: openudon convert tf [--config-dir DIR] --openapi ID=PATH [--action create|update|delete|replace] [--target ADDRESS] [--out DIR] [--strict]")
		os.Exit(2)
	}
	runConvertTFCommand(args[1:])
}

func runConvertTFCommand(args []string) {
	fs := flag.NewFlagSet("convert tf", flag.ExitOnError)
	configDir := fs.String("config-dir", ".", "Terraform/OpenTofu configuration directory")
	action := fs.String("action", "", "Managed resource action: create, update, delete, or replace")
	outDir := fs.String("out", "./.openudon/convert", "Output directory for draft review artifacts")
	strict := fs.Bool("strict", false, "Fail when strict-failure diagnostics remain")
	var openAPIs repeatedStringFlag
	var targets repeatedStringFlag
	fs.Var(&openAPIs, "openapi", "Repeatable OpenAPI input as ID=PATH")
	fs.Var(&targets, "target", "Repeatable Terraform address target")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: openudon convert tf [--config-dir DIR] --openapi ID=PATH [--action create|update|delete|replace] [--target ADDRESS] [--out DIR] [--strict]\n")
		fmt.Fprintf(fs.Output(), "\nGenerates draft OpenUdon review scaffolding from static Terraform/OpenTofu configuration and local OpenAPI documents. It does not execute Terraform, providers, OpenAPI operations, or UWS workflows.\n\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	inputs, err := parseOpenAPIFlags(openAPIs)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	result, err := tfconvert.Convert(ctx, tfconvert.Options{
		ConfigDir: *configDir,
		OpenAPIs:  inputs,
		Action:    *action,
		Targets:   []string(targets),
		OutDir:    *outDir,
		Strict:    *strict,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		if result != nil {
			fmt.Fprintf(os.Stderr, "diagnostics: %s\n", result.DiagnosticsJSON)
		}
		if tfconvert.IsStrictFailure(err) {
			os.Exit(1)
		}
		os.Exit(1)
	}
	fmt.Printf("openudon: convert tf wrote %s\n", result.OutDir)
	fmt.Printf("  project:     %s\n", result.ProjectPath)
	fmt.Printf("  intent:      %s\n", result.IntentPath)
	fmt.Printf("  workflow:    %s\n", result.WorkflowPath)
	fmt.Printf("  uws:         %s\n", result.UWSPath)
	fmt.Printf("  plan:        %s\n", result.PlanJSONPath)
	fmt.Printf("  diagnostics: %s\n", result.DiagnosticsJSON)
	fmt.Printf("  review:      %s\n", result.ReviewPath)
	fmt.Printf("  quality:     %s\n", result.QualityJSONPath)
}

func runN8nBridgeCommand(args []string) {
	if len(args) == 0 || args[0] != "validate" {
		fmt.Fprintln(os.Stderr, "usage: openudon n8n-bridge validate [--root examples/eval] [--file examples/eval/<name>/reference/n8n-bridge.json]")
		os.Exit(2)
	}
	runN8nBridgeValidateCommand(args[1:])
}

func runN8nBridgeValidateCommand(args []string) {
	fs := flag.NewFlagSet("n8n-bridge validate", flag.ExitOnError)
	root := fs.String("root", "examples/eval", "Eval fixture root to scan for reference/n8n-bridge.json summaries")
	file := fs.String("file", "", "Validate one n8n bridge summary file instead of scanning --root")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: openudon n8n-bridge validate [--root examples/eval] [--file examples/eval/<name>/reference/n8n-bridge.json]\n")
		fmt.Fprintf(fs.Output(), "\nValidates %s evidence. The bridge is authoring assistance only: it does not import, execute, or emulate n8n workflows.\n\n", n8nbridge.SummaryVersion)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	var (
		results []n8nbridge.ValidationResult
		err     error
	)
	if strings.TrimSpace(*file) != "" {
		var result n8nbridge.ValidationResult
		result, err = n8nbridge.ValidateFile(*file)
		results = []n8nbridge.ValidationResult{result}
	} else {
		results, err = n8nbridge.ValidateRoot(*root)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if len(results) == 0 {
		fmt.Fprintf(os.Stderr, "no n8n bridge summaries found under %s\n", *root)
		os.Exit(1)
	}
	fmt.Printf("openudon: n8n bridge validated %d summary file(s)\n", len(results))
	for _, result := range results {
		fmt.Printf("openudon: checked %s (%s, %s)\n", result.Path, result.Summary.Fixture, result.Summary.Validation.Status)
	}
}

func parseOpenAPIFlags(values []string) ([]tfconvert.OpenAPIInput, error) {
	inputs := make([]tfconvert.OpenAPIInput, 0, len(values))
	for _, value := range values {
		id, path, ok := strings.Cut(value, "=")
		if !ok || strings.TrimSpace(id) == "" || strings.TrimSpace(path) == "" {
			return nil, fmt.Errorf("--openapi must be ID=PATH, got %q", value)
		}
		inputs = append(inputs, tfconvert.OpenAPIInput{ID: strings.TrimSpace(id), Path: strings.TrimSpace(path)})
	}
	return inputs, nil
}

func runValidateCommand(args []string) {
	fs := flag.NewFlagSet("validate", flag.ExitOnError)
	allowEmpty := fs.Bool("allow-empty", false, "Allow directory validation to pass when no UWS artifacts are found")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: openudon validate [--allow-empty] <uws-file-or-dir>\n")
		fmt.Fprintf(fs.Output(), "\nValidates one UWS JSON/YAML file or every *.uws.json/*.uws.yaml/*.uws.yml artifact under a directory.\n\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if fs.NArg() != 1 {
		fs.Usage()
		os.Exit(2)
	}
	if err := validateUWSPath(fs.Arg(0), os.Stdout, *allowEmpty); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func defaultUWSSchemaForFile(path string) string {
	version := "1.0.0"
	if doc, err := uwsprofile.LoadDocumentFile(path, uwsprofile.DocumentFormatAuto); err == nil && doc != nil && strings.TrimSpace(doc.UWS) != "" {
		version = strings.TrimSpace(doc.UWS)
	}
	return uwsschema.PathForVersion(".", version)
}

func validateUWSPath(target string, out io.Writer, allowEmpty bool) error {
	return validateUWSPathWithSchema(target, out, defaultUWSSchemaForFile, allowEmpty)
}

func validateUWSPathWithSchema(target string, out io.Writer, schemaForFile func(string) string, allowEmpty bool) error {
	info, err := os.Stat(target)
	if err != nil {
		return fmt.Errorf("target does not exist: %s", target)
	}
	if !info.IsDir() {
		return validateUWSFile(target, out, schemaForFile)
	}

	files, err := collectUWSArtifactFiles(target)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		if allowEmpty {
			fmt.Fprintf(out, "no UWS artifacts found under %s\n", target)
			return nil
		}
		return fmt.Errorf("no UWS artifacts found under %s; pass --allow-empty to allow this", target)
	}
	fmt.Fprintf(out, "found %d UWS artifact(s); schema selected from document version\n", len(files))
	for _, file := range files {
		if err := validateUWSFile(file, out, schemaForFile); err != nil {
			return err
		}
	}
	return nil
}

func validateUWSFile(path string, out io.Writer, schemaForFile func(string) string) error {
	if err := uwsvalidate.ValidateFile(schemaForFile(path), path); err != nil {
		return err
	}
	fmt.Fprintf(out, "openudon: %s is valid UWS\n", path)
	return nil
}

func collectUWSArtifactFiles(root string) ([]string, error) {
	var files []string
	if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if isUWSArtifactFile(path) {
			files = append(files, path)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

func isUWSArtifactFile(path string) bool {
	lower := strings.ToLower(path)
	return strings.HasSuffix(lower, ".uws.json") || strings.HasSuffix(lower, ".uws.yaml") || strings.HasSuffix(lower, ".uws.yml")
}

func runReadinessCommand(args []string) {
	fs := flag.NewFlagSet("readiness", flag.ExitOnError)
	out := fs.String("out", "", "Write readiness JSON to this path instead of stdout")
	runGates := fs.Bool("run-gates", false, "Run deterministic gates: go test ./..., go vet ./..., make check, and git diff --check")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: openudon readiness [--out eval/readiness/<name>.json] [--run-gates]\n")
		fmt.Fprintf(fs.Output(), "\nWrites %s JSON for XRD-007 local optional-sibling checkout readiness without printing secret values.\n", readiness.ReportVersion)
		fmt.Fprintf(fs.Output(), "By default, deterministic gates are marked skipped; pass --run-gates for release-readiness evidence.\n\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	report, err := readiness.Build(ctx, readiness.Options{
		RepoRoot: ".",
		RunGates: *runGates,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if strings.TrimSpace(*out) != "" {
		if err := readiness.WriteFile(*out, report); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Printf("openudon: readiness %s wrote %s\n", report.Status, *out)
	} else if err := readiness.Write(os.Stdout, report); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if report.Status == "fail" {
		os.Exit(1)
	}
}

func runTrustedCommand(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	example := fs.String("example", "", "Example directory containing generated OpenUdon artifacts")
	tier := fs.String("tier", "", "Execution tier: sandbox or production")
	approval := fs.String("approval", "", "Approval JSON file")
	workdir := fs.String("workdir", "", "executor work directory; defaults to .openudon-run/<example>")
	dryRun := fs.Bool("dry-run", false, "Validate gates and write run config without invoking the executor")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: openudon run --example examples/<name> --tier sandbox|production --approval approvals/<name>.json [--workdir .openudon-run/<name>] [--dry-run]\n")
		fmt.Fprintf(fs.Output(), "\nValidates the OpenUdon handoff package, current quality gates, approval scope, approval digest, and tier/state compatibility before writing %s run config and invoking the trusted executor runner.\n", trustedrunner.RunConfigVersion)
		fmt.Fprintf(fs.Output(), "\nTier rules:\n")
		fmt.Fprintf(fs.Output(), "  sandbox accepts approved_for_sandbox or approved_for_production\n")
		fmt.Fprintf(fs.Output(), "  production accepts approved_for_production only\n\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	result, err := trustedrunner.Run(ctx, trustedrunner.Options{
		RepoRoot:     ".",
		ExampleDir:   *example,
		Tier:         *tier,
		ApprovalPath: *approval,
		WorkDir:      *workdir,
		DryRun:       *dryRun,
		RunnerPath:   os.Getenv("OPENUDON_UDON_RUNNER"),
		Stdout:       os.Stdout,
		Stderr:       os.Stderr,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if result.DryRun {
		fmt.Printf("openudon: run dry-run passed for %s (%s)\n", result.Scope, result.Tier)
	} else {
		fmt.Printf("openudon: run completed for %s (%s)\n", result.Scope, result.Tier)
	}
	fmt.Printf("  workflow: %s\n", result.WorkflowPath)
	fmt.Printf("  config:   %s\n", result.RunConfigPath)
	fmt.Printf("  workdir:  %s\n", result.WorkDir)
	fmt.Printf("  digest:   %s\n", result.PackageSHA256)
}

func runApprovalTemplateCommand(args []string) {
	fs := flag.NewFlagSet("approval-template", flag.ExitOnError)
	example := fs.String("example", "", "Example directory containing generated OpenUdon artifacts")
	state := fs.String("state", "", "Approval state: approved_for_sandbox or approved_for_production")
	reviewer := fs.String("reviewer", "", "Reviewer name recorded in the approval JSON")
	notes := fs.String("notes", "", "Optional approval notes")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: openudon approval-template --example examples/<name> --state approved_for_sandbox|approved_for_production --reviewer <name> [--notes <text>]\n")
		fmt.Fprintf(fs.Output(), "\nPrints %s JSON to stdout with the current handoff package SHA-256 digest.\n", trustedrunner.ApprovalVersion)
		fmt.Fprintf(fs.Output(), "Schema fields: version, scope, state, reviewer, approved_at, expires_at, package_sha256, notes.\n\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	approval, err := trustedrunner.ApprovalTemplate(ctx, trustedrunner.TemplateOptions{
		RepoRoot:   ".",
		ExampleDir: *example,
		State:      *state,
		Reviewer:   *reviewer,
		Notes:      *notes,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := trustedrunner.WriteApproval(os.Stdout, approval); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runEvalCommand(args []string) {
	fs := flag.NewFlagSet("eval", flag.ExitOnError)
	root := fs.String("root", "examples/eval", "Directory containing eval example subdirectories")
	name := fs.String("name", "", "Run a single eval brief by directory name")
	provider := fs.String("provider", "", "LLM provider: copilot-api, openai, anthropic, or gemini")
	model := fs.String("model", "", "LLM model")
	timeout := fs.Duration("timeout", 2*time.Minute, "LLM generation timeout")
	maxAttempts := fs.Int("max-attempts", 5, "Maximum refinement attempts")
	temperature := fs.Float64("temperature", 0.2, "Intent generation temperature")
	concurrency := fs.Int("concurrency", 2, "Maximum concurrent eval runs")
	releaseGate := fs.Bool("release-gate", false, "Fail unless eval results meet local release criteria")
	minBriefs := fs.Int("min-briefs", 0, "Minimum eval brief count required by --release-gate")
	compare := fs.String("compare", "", "Compare this eval report against a specific previous JSON report")
	noCompare := fs.Bool("no-compare", false, "Disable previous-run comparison")
	archiveDir := fs.String("archive-dir", "", "Copy generated eval workspaces under this directory for manual inspection")
	out := fs.String("out", evalpkg.DefaultOutputPath(time.Now()), "JSON report output path")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: openudon eval [--root examples/eval] [--name support-email] [--out eval/runs/<ts>.json] [--release-gate] [--min-briefs N] [--compare eval/runs/<previous>.json] [--no-compare] [--archive-dir eval/artifacts]\n")
		fmt.Fprintf(fs.Output(), "\nRuns synthesis against temporary copies of eval briefs and writes JSON/Markdown reports with optional run comparison.\n")
		fmt.Fprintf(fs.Output(), "Normal evals print comparison regressions for review but exit successfully when synthesis completes.\n")
		fmt.Fprintf(fs.Output(), "With --release-gate, absolute release criteria and comparison regressions fail the command.\n")
		fmt.Fprintf(fs.Output(), "\nExamples:\n")
		fmt.Fprintf(fs.Output(), "  openudon eval --root examples/eval --provider copilot-api --model gpt-5.4-mini\n")
		fmt.Fprintf(fs.Output(), "  openudon eval --root examples/eval --name support-email --provider copilot-api --model gpt-5.4-mini\n")
		fmt.Fprintf(fs.Output(), "  openudon eval --root examples/eval --provider copilot-api --model gpt-5.4-mini --release-gate\n\n")
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
		MinBriefs:   *minBriefs,
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
	fmt.Printf("openudon: eval wrote %s\n", *out)
	fmt.Print(evalpkg.MarkdownReport(report))
	if *releaseGate {
		criteria := evalpkg.DefaultReleaseCriteria()
		criteria.MinBriefs = *minBriefs
		if err := evalpkg.ReleaseCriteriaError(results, criteria); err != nil {
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
	provider := fs.String("provider", "", "LLM provider: copilot-api, openai, anthropic, or gemini; defaults to OpenUdon provider env behavior")
	model := fs.String("model", "", "LLM model")
	timeout := fs.Duration("timeout", 2*time.Minute, "LLM generation timeout")
	maxAttempts := fs.Int("max-attempts", 5, "Maximum refinement attempts for synthesize/build")
	temperature := fs.Float64("temperature", 0.2, "Intent generation temperature for synthesize")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: openudon %s --example examples/<name> [--provider copilot-api --model gpt-5.4-mini]\n", command)
		fmt.Fprintf(fs.Output(), "\n%s\n", artifactCommandDescription(command))
		fmt.Fprintf(fs.Output(), "\nExamples:\n")
		switch command {
		case "synthesize":
			fmt.Fprintf(fs.Output(), "  openudon synthesize --example examples/support-email --provider copilot-api --model gpt-5.4-mini --max-attempts 5\n")
		case "build":
			fmt.Fprintf(fs.Output(), "  openudon build --example examples/support-email --provider copilot-api --model gpt-5.4-mini\n")
		case "promote":
			fmt.Fprintf(fs.Output(), "  openudon promote --example examples/support-email\n")
		case "assess":
			fmt.Fprintf(fs.Output(), "  openudon assess --example examples/support-email\n")
		}
		fmt.Fprintf(fs.Output(), "\nArtifacts:\n")
		fmt.Fprintf(fs.Output(), "  workflows/intent.hcl        structured intent generated from project.md\n")
		fmt.Fprintf(fs.Output(), "  workflows/workflow.hcl      public UWS HCL artifact\n")
		fmt.Fprintf(fs.Output(), "  workflows/workflow.uws.yaml public UWS YAML artifact\n")
		fmt.Fprintf(fs.Output(), "  expected/plan.json          expected operations, bindings, credentials, and control flow\n")
		fmt.Fprintf(fs.Output(), "  expected/review.md          trusted execution review evidence and handoff notes\n")
		fmt.Fprintf(fs.Output(), "  expected/symphony-handoff.json machine-readable Symphony approval handoff\n")
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
	fmt.Printf("openudon: %s %s\n", command, result.ExampleDir)
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
		return "Run a OpenUdon artifact command."
	}
}

func printQuality(report *synthesize.QualityReport) {
	if report == nil {
		return
	}
	fmt.Printf("openudon: quality %s\n", report.Status)
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
		return "Add or fix OpenAPI documents under openapi/ with operation IDs, request fields, response schemas, and security schemes; or declare OpenAPI: none required when no API is needed."
	case code == "plan.gaps":
		return "Resolve missing operations, required parameters, or credential bindings in project.md or intent.hcl."
	case code == "intent.data_flow.required_params":
		return "Map every required OpenAPI path, query, header, or body field to an input, safe literal, prior-step bind, or credential binding name; document SaaS request mappings in Data Flow."
	case code == "intent.data_flow.response_paths":
		return "Use response fields present in the OpenAPI schema or update Outputs and Data Flow; avoid guessing SaaS response paths."
	case code == "intent.data_flow.explicit":
		return "Add Data Flow guidance with request field sources, prior-step bindings, credential binding names, and final output sources."
	case code == "intent.openapi_operations":
		return "Select only operationId values listed in local OpenAPI documents and document unresolved SaaS capability gaps instead of inventing provider operations."
	case strings.HasPrefix(code, "intent."):
		return "Inspect workflows/intent.hcl and project.md; rerun synthesize when the brief needs regeneration."
	case code == "credentials.security_schemes":
		return "Declare symbolic credential binding names for required OpenAPI security schemes in project.md, then rerun synthesize or build."
	case code == "credentials.bindings", code == "workflow.credentials_bound":
		return "Name runtime credential bindings in project.md and ensure workflow request fields reference binding names, never secret values."
	case strings.HasPrefix(code, "workflow."):
		return "Inspect workflows/workflow.hcl against expected/plan.md, then rerun build or synthesize."
	case strings.HasPrefix(code, "uws."):
		return "Inspect workflows/workflow.uws.yaml, then rerun promote or build after fixing workflow.hcl."
	case code == "side_effects.environment":
		return "Use sandbox/test endpoints or add explicit production handoff approval language to Safety and Approval Boundary."
	case code == "side_effects.policy":
		return "Add approval, trusted-runtime, and sandbox proof-run policy to Safety and Approval Boundary."
	case code == "review.credential_bindings":
		return "Update Credentials and Secrets with binding names only, then regenerate review evidence with build/synthesize."
	case code == "review.approval_states":
		return "State generated, review_required, approved_for_sandbox, and approved_for_production approval requirements in review evidence."
	case code == "review.sandbox_handoff":
		return "Scope trusted-runner handoff to approved sandbox or proof runs before production handoff."
	case code == "review.trusted_runner":
		return "Regenerate review evidence so expected/review.md includes the trusted-runner handoff command."
	case code == "review.trusted_runner_dry_run":
		return "Regenerate review evidence so expected/review.md includes the trusted-runner dry-run command and run-config boundary."
	case code == "review.production_boundary":
		return "Regenerate review evidence so it states OpenUdon synthesis does not directly execute production workflows."
	case code == "review.approval_artifact":
		return "Regenerate review evidence so it describes approval JSON fields, tier state, expiry, and package_sha256 requirements."
	case code == "review.credential_scope":
		return "Regenerate review evidence so it includes the credential scope matrix for declared and expected bindings."
	case code == "review.side_effect_risk":
		return "Regenerate review evidence so it lists side-effect risk and approved sandbox/production handoff states."
	case strings.HasPrefix(code, "review."):
		return "Update Safety and Approval Boundary or regenerate review evidence with build/synthesize."
	case strings.HasPrefix(code, "symphony_handoff."):
		return "Regenerate expected/symphony-handoff.json with build/synthesize so Symphony can consume the approval handoff contract."
	case code == "artifacts.no_secrets":
		return "Remove literal secret-like values from artifacts; keep only credential binding names."
	default:
		return "Inspect expected/quality.md for details, fix the referenced artifact, and rerun assess."
	}
}
