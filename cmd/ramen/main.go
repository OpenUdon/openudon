package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/genelet/ramen/internal/config"
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

func runArtifactCommand(command string, args []string) {
	fs := flag.NewFlagSet(command, flag.ExitOnError)
	example := fs.String("example", "", "Example directory containing project.md and artifact subdirectories")
	provider := fs.String("provider", "", "LLM provider: openai, anthropic, or gemini; defaults to udon runner env behavior")
	model := fs.String("model", "", "LLM model")
	timeout := fs.Duration("timeout", 2*time.Minute, "LLM generation timeout")
	maxAttempts := fs.Int("max-attempts", 5, "Maximum refinement attempts for synthesize/build")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: ramen %s --example examples/<name> [--provider gemini --model gemini-2.5-pro]\n", command)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	opts := synthesize.Options{
		ExampleDir:  *example,
		Provider:    *provider,
		Model:       *model,
		Timeout:     *timeout,
		MaxAttempts: *maxAttempts,
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

func printQuality(report *synthesize.QualityReport) {
	if report == nil {
		return
	}
	fmt.Printf("ramen: quality %s\n", report.Status)
	fmt.Printf("  report: %s\n", report.Artifacts.QualityJSONPath)
	for _, check := range report.Checks {
		fmt.Printf("  %s: %s\n", check.Code, check.Status)
	}
}
