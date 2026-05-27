package icot

import (
	"bytes"
	"context"
	"encoding/json"
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
	"github.com/OpenUdon/openudon/internal/icot/elicitor"
	"github.com/OpenUdon/openudon/internal/projectwizard"
	"github.com/OpenUdon/openudon/internal/synthesize"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

func runScorecard(args []string, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("icot scorecard", flag.ContinueOnError)
	fs.SetOutput(out)
	root := fs.String("root", "examples/eval", "Directory containing eval example subdirectories")
	name := fs.String("name", "", "Run one eval fixture by directory name")
	outDir := fs.String("out", filepath.Join("eval", "runs", "icot-scorecard-"+time.Now().UTC().Format("20060102T150405Z")), "Output directory for scorecard artifacts")
	includeVariants := fs.Bool("include-variants", false, "Also run provider-free natural-language authoring variants declared by fixtures")
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
			ByClass:                 map[string]int{},
			ByFailureFamily:         map[string]int{},
			ByObservedOutcome:       map[string]int{},
			ByProviderFamily:        map[string]int{},
			ByProviderFailureFamily: map[string]map[string]int{},
			ByVariantClass:          map[string]int{},
		},
	}
	for _, fixture := range fixtures {
		result := runScorecardFixture(fixture, *outDir)
		appendScorecardResult(&report, result, out)
		if *includeVariants {
			for _, variantResult := range runScorecardVariants(fixture, *outDir) {
				appendScorecardResult(&report, variantResult, out)
			}
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
	result := scorecardResult{Name: name, Kind: "seed_build", Fixture: name}
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

func appendScorecardResult(report *scorecardReport, result scorecardResult, out io.Writer) {
	report.Results = append(report.Results, result)
	report.Summary.Total++
	if result.Class != "" {
		report.Summary.ByClass[result.Class]++
	}
	if result.Kind == "authoring_variant" && result.Class != "" {
		report.Summary.ByVariantClass[result.Class]++
	}
	report.Summary.ByObservedOutcome[result.ObservedOutcome]++
	if result.FailureFamily != "" {
		report.Summary.ByFailureFamily[result.FailureFamily]++
	}
	for _, provider := range result.ProviderFamilies {
		report.Summary.ByProviderFamily[provider]++
		if result.FailureFamily != "" {
			if report.Summary.ByProviderFailureFamily[provider] == nil {
				report.Summary.ByProviderFailureFamily[provider] = map[string]int{}
			}
			report.Summary.ByProviderFailureFamily[provider][result.FailureFamily]++
		}
	}
	if result.Passed {
		report.Summary.Passed++
		fmt.Fprintf(out, "icot scorecard: pass %s\n", result.Name)
		return
	}
	report.Summary.Failed++
	report.Status = statusFail
	fmt.Fprintf(out, "icot scorecard: fail %s - expected %s, observed %s\n", result.Name, result.ExpectedOutcome, result.ObservedOutcome)
}

func runScorecardVariants(fixture, outDir string) []scorecardResult {
	path := filepath.Join(fixture, "reference", "authoring-variants.json")
	variants, err := evalpkg.ReadAuthoringVariants(path)
	if os.IsNotExist(err) {
		return nil
	}
	fixtureName := filepath.Base(filepath.Clean(fixture))
	if err != nil {
		return []scorecardResult{{
			Name:            fixtureName + "#authoring-variants",
			Kind:            "authoring_variant",
			Fixture:         fixtureName,
			ExpectedOutcome: "pass",
			ObservedOutcome: "policy_error",
			FailureFamily:   failureUnknown,
			Detail:          err.Error(),
		}}
	}
	var results []scorecardResult
	for _, variant := range variants.Variants {
		results = append(results, runScorecardVariant(fixture, variants.ProviderFamilies, variant, outDir))
	}
	return results
}

func runScorecardVariant(fixture string, providers []string, variant evalpkg.AuthoringVariant, outDir string) scorecardResult {
	fixtureName := filepath.Base(filepath.Clean(fixture))
	result := scorecardResult{
		Name:                  fixtureName + "#" + variant.ID,
		Kind:                  "authoring_variant",
		Fixture:               fixtureName,
		VariantID:             variant.ID,
		Brief:                 variant.Brief,
		Class:                 variant.Class,
		ExpectedOutcome:       variant.ExpectedOutcome,
		ExpectedFailureFamily: variant.ExpectedFailureFamily,
		ProviderFamilies:      append([]string(nil), providers...),
		Tags:                  append([]string(nil), variant.Tags...),
	}
	if variant.ExpectedOutcome == "pass" {
		seedDir, err := prepareScorecardVariantSeed(fixture, variant, outDir)
		if err != nil {
			result.ObservedOutcome = "icot_fail"
			result.Detail = err.Error()
			result.FailureFamily = failureUnknown
			return result
		}
		observed := runScorecardFixture(seedDir, outDir)
		result.ObservedOutcome = observed.ObservedOutcome
		result.FailureFamily = observed.FailureFamily
		result.FailureCodes = observed.FailureCodes
		result.Detail = observed.Detail
		result.ExampleDir = observed.ExampleDir
		result.GeneratedProject = observed.GeneratedProject
		result.GeneratedIntent = observed.GeneratedIntent
		result.QualityReport = observed.QualityReport
		result.Passed = scorecardVariantOutcomeMatches(result)
		return result
	}
	observed := runNeedsInputVariant(fixture, fixtureName, variant, outDir)
	result.ObservedOutcome = observed.ObservedOutcome
	result.FailureFamily = observed.FailureFamily
	result.FailureCodes = observed.FailureCodes
	result.Detail = observed.Detail
	result.ExampleDir = observed.ExampleDir
	result.GeneratedProject = observed.GeneratedProject
	result.GeneratedIntent = observed.GeneratedIntent
	result.Passed = scorecardVariantOutcomeMatches(result)
	return result
}

func prepareScorecardVariantSeed(fixture string, variant evalpkg.AuthoringVariant, outDir string) (string, error) {
	seedDir := filepath.Join(outDir, "variant-seeds", safeScorecardName(filepath.Base(filepath.Clean(fixture))+"__"+variant.ID))
	_ = os.RemoveAll(seedDir)
	if err := copyScorecardTree(fixture, seedDir); err != nil {
		return "", err
	}
	projectPath := filepath.Join(seedDir, "project.md")
	data, err := os.ReadFile(projectPath)
	if err != nil {
		return "", err
	}
	project, err := projectwizard.LoadAnswersFromMarkdown(string(data))
	if err != nil {
		return "", err
	}
	project.Goal = variant.Brief
	if err := os.WriteFile(projectPath, []byte(projectwizard.Render(project)), 0o644); err != nil {
		return "", err
	}
	return seedDir, nil
}

func runNeedsInputVariant(fixture, fixtureName string, variant evalpkg.AuthoringVariant, outDir string) scorecardResult {
	workspace := filepath.Join(outDir, "variant-workspaces", safeScorecardName(fixtureName), safeScorecardName(variant.ID))
	_ = os.RemoveAll(workspace)
	sessionPath := filepath.Join(outDir, "variant-sessions", safeScorecardName(fixtureName), safeScorecardName(variant.ID)+".json")
	project := projectwizard.Answers{
		ProjectName:     fixtureName + " " + variant.ID,
		Goal:            variant.Brief,
		SideEffectScope: projectwizard.SideEffectAfterApproval,
		Safety:          "Generate and validate artifacts only; side-effectful execution requires approval.",
		Fallback:        "Stop until missing authoring details are provided.",
	}
	if data, err := os.ReadFile(filepath.Join(fixture, "project.md")); err == nil {
		if loaded, loadErr := projectwizard.LoadAnswersFromMarkdown(string(data)); loadErr == nil {
			project.UsesOpenAPI = loaded.UsesOpenAPI
			project.OpenAPI = loaded.OpenAPI
			project.Credentials = loaded.Credentials
		}
	}
	session := elicitor.Session{
		Project: project,
		Intent: rollout.Intent{
			Workflow: &rollout.WorkflowMeta{
				Name:        safeScorecardName(fixtureName + "_" + variant.ID),
				Description: variant.Brief,
			},
		},
	}
	result := scorecardResult{ExampleDir: workspace}
	if err := copySeedSourceArtifacts(fixture, workspace, true); err != nil {
		result.ObservedOutcome = "icot_fail"
		result.FailureFamily = failureUnknown
		result.Detail = err.Error()
		return result
	}
	if err := writeJSONFile(sessionPath, session); err != nil {
		result.ObservedOutcome = "icot_fail"
		result.FailureFamily = failureUnknown
		result.Detail = err.Error()
		return result
	}
	var stdout, stderr bytes.Buffer
	code := Main([]string{"--example", workspace, "--answers", sessionPath, "--agent", "--json", "--no-transcript"}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		result.ObservedOutcome = "icot_fail"
		result.FailureFamily = failureFamilyForDetail(stderr.String())
		result.Detail = strings.TrimSpace(stderr.String())
		return result
	}
	var report authorReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		result.ObservedOutcome = "icot_fail"
		result.FailureFamily = failureUnknown
		result.Detail = err.Error()
		return result
	}
	result.ObservedOutcome = report.Status
	result.FailureFamily = report.FailureFamily
	result.Detail = report.Error
	result.GeneratedProject = report.GeneratedProject
	result.GeneratedIntent = report.GeneratedIntent
	return result
}

func scorecardVariantOutcomeMatches(result scorecardResult) bool {
	if result.ObservedOutcome != result.ExpectedOutcome {
		return false
	}
	if result.ExpectedFailureFamily != "" && result.FailureFamily != result.ExpectedFailureFamily {
		return false
	}
	return true
}

func copyScorecardTree(src, dst string) error {
	return filepath.WalkDir(src, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if entry.IsDir() {
			if entry.Name() == ".icot" || entry.Name() == "output" || entry.Name() == "output-debug" {
				return filepath.SkipDir
			}
			return os.MkdirAll(target, 0o755)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode().Perm())
	})
}

func safeScorecardName(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	out := strings.Trim(b.String(), "_")
	if out == "" {
		return "variant"
	}
	return out
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
