package icot

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	evalpkg "github.com/OpenUdon/openudon/internal/eval"
	"github.com/OpenUdon/openudon/internal/synthesize"
)

func runScorecard(args []string, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("icot scorecard", flag.ContinueOnError)
	fs.SetOutput(out)
	root := fs.String("root", "examples/eval", "Directory containing eval example subdirectories")
	name := fs.String("name", "", "Run one eval fixture by directory name")
	outDir := fs.String("out", filepath.Join("eval", "runs", "icot-scorecard-"+time.Now().UTC().Format("20060102T150405Z")), "Output directory for scorecard artifacts")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: icot scorecard [--root examples/eval] [--out eval/runs/icot-scorecard-<ts>]\n\n")
		fmt.Fprintf(fs.Output(), "Runs the provider-free iCoT seed/build reliability scorecard.\n\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	fixtures := discoverScorecardFixtures(*root, *name)
	if len(fixtures) == 0 {
		fmt.Fprintf(errOut, "no eval fixtures found under %s\n", *root)
		return 1
	}
	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	report := scorecardReport{
		Version: scorecardReportVersion,
		Status:  statusPass,
		Root:    *root,
		OutDir:  *outDir,
		Summary: scorecardSummary{
			ByClass:           map[string]int{},
			ByFailureFamily:   map[string]int{},
			ByObservedOutcome: map[string]int{},
		},
	}
	for _, fixture := range fixtures {
		result := runScorecardFixture(fixture, *outDir)
		report.Results = append(report.Results, result)
		report.Summary.Total++
		report.Summary.ByClass[result.Class]++
		report.Summary.ByObservedOutcome[result.ObservedOutcome]++
		if result.FailureFamily != "" {
			report.Summary.ByFailureFamily[result.FailureFamily]++
		}
		if result.Passed {
			report.Summary.Passed++
			fmt.Fprintf(out, "icot scorecard: pass %s\n", result.Name)
		} else {
			report.Summary.Failed++
			report.Status = statusFail
			fmt.Fprintf(out, "icot scorecard: fail %s - expected %s, observed %s\n", result.Name, result.ExpectedOutcome, result.ObservedOutcome)
		}
	}
	reportPath := filepath.Join(*outDir, "scorecard.json")
	if err := writeJSONFile(reportPath, report); err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	fmt.Fprintf(out, "icot scorecard: wrote %s\n", reportPath)
	if report.Status != statusPass {
		return 1
	}
	return 0
}

func discoverScorecardFixtures(root, name string) []string {
	if strings.TrimSpace(name) != "" {
		path := filepath.Join(root, strings.TrimSpace(name))
		if _, err := os.Stat(filepath.Join(path, "reference", "policy.json")); err == nil {
			return []string{path}
		}
		return nil
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	var out []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(root, entry.Name())
		if _, err := os.Stat(filepath.Join(path, "reference", "policy.json")); err == nil {
			out = append(out, path)
		}
	}
	sort.Strings(out)
	return out
}

func runScorecardFixture(seedDir, outDir string) scorecardResult {
	name := filepath.Base(filepath.Clean(seedDir))
	result := scorecardResult{Name: name}
	policy, err := evalpkg.ReadReferencePolicy(filepath.Join(seedDir, "reference", "policy.json"))
	if err != nil {
		result.ObservedOutcome = "policy_error"
		result.Detail = err.Error()
		result.FailureFamily = failureUnknown
		return result
	}
	if policy.SeedBuild != nil {
		result.ExpectedOutcome = policy.SeedBuild.Expected
		result.Class = policy.SeedBuild.Class
	}
	if result.ExpectedOutcome == "" {
		result.ExpectedOutcome = "pass"
	}
	workspace := filepath.Join(outDir, "workspaces", name)
	_ = os.RemoveAll(workspace)
	result.ExampleDir = workspace
	var stdout, stderr bytes.Buffer
	code := Main([]string{"--example", workspace, "--from-example", seedDir, "--no-llm", "--no-transcript"}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		result.ObservedOutcome = "icot_fail"
		result.Detail = strings.TrimSpace(stderr.String())
		result.FailureFamily = failureFamilyForDetail(result.Detail)
		result.Passed = scorecardOutcomeMatches(result, policy)
		return result
	}
	result.GeneratedProject = filepath.Join(workspace, "project.md")
	result.GeneratedIntent = filepath.Join(workspace, "workflows", "intent.hcl")
	_, quality, err := synthesize.PackageFromIntent(context.Background(), synthesize.Options{ExampleDir: workspace})
	result.QualityReport = filepath.Join(workspace, "expected", "quality.json")
	if err != nil {
		result.ObservedOutcome = "build_fail"
		result.FailureCodes = []string{"build:error"}
		result.Detail = err.Error()
		result.FailureFamily = failureFamilyForDetail(err.Error())
		result.Passed = scorecardOutcomeMatches(result, policy)
		return result
	}
	if quality == nil || !quality.Passed() {
		result.ObservedOutcome = "build_fail"
		result.FailureCodes = scorecardFailedQualityCodes(quality)
		result.Detail = scorecardQualityFailureDetails(quality)
		result.FailureFamily = failureFamilyForQualityCode(firstFailedQualityCode(quality))
		result.Passed = scorecardOutcomeMatches(result, policy)
		return result
	}
	result.ObservedOutcome = "pass"
	result.Passed = scorecardOutcomeMatches(result, policy)
	return result
}

func scorecardFailedQualityCodes(report *synthesize.QualityReport) []string {
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

func scorecardQualityFailureDetails(report *synthesize.QualityReport) string {
	if report == nil {
		return ""
	}
	var out []string
	for _, check := range report.Checks {
		if check.Status == "fail" {
			out = append(out, check.Code+": "+check.Detail)
		}
	}
	return strings.Join(out, "; ")
}

func scorecardOutcomeMatches(result scorecardResult, policy evalpkg.ReferencePolicy) bool {
	expected := result.ExpectedOutcome
	if expected == "" {
		expected = "pass"
	}
	if result.ObservedOutcome != expected {
		return false
	}
	if expected == "pass" || policy.SeedBuild == nil || len(policy.SeedBuild.AllowedFailureCodes) == 0 {
		return true
	}
	allowed := map[string]bool{}
	for _, code := range policy.SeedBuild.AllowedFailureCodes {
		allowed[code] = true
	}
	for _, code := range result.FailureCodes {
		if allowed[code] {
			return true
		}
	}
	return false
}

func failureFamilyForDetail(detail string) string {
	lower := strings.ToLower(detail)
	switch {
	case strings.Contains(lower, "openapi"), strings.Contains(lower, "api document"), strings.Contains(lower, "source"):
		return failureMissingAPISource
	case strings.Contains(lower, "operation"):
		return failureMissingOperation
	case strings.Contains(lower, "credential"):
		return failureCredentialBindingGap
	case strings.Contains(lower, "runtime"):
		return failureRuntimeProfileGap
	case strings.Contains(lower, "intent"):
		return failureIntentParse
	default:
		return failureBuildError
	}
}
