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
	"strings"
	"time"

	"github.com/OpenUdon/openudon/internal/authoring"
	evalpkg "github.com/OpenUdon/openudon/internal/eval"
	"github.com/OpenUdon/openudon/internal/icot/elicitor"
	"github.com/OpenUdon/openudon/internal/projectwizard"
	"github.com/OpenUdon/openudon/internal/synthesize"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
	runner "github.com/OpenUdon/openudon/internal/workflowintent"
)

const authoringEvalReportVersion = "openudon.icot-authoring-eval.v1"

type authoringEvalReport struct {
	Version         string                `json:"version"`
	Status          string                `json:"status"`
	Root            string                `json:"root"`
	OutDir          string                `json:"out_dir"`
	Provider        string                `json:"provider,omitempty"`
	Model           string                `json:"model,omitempty"`
	PromptVersion   string                `json:"prompt_version,omitempty"`
	IncludeVariants bool                  `json:"include_variants,omitempty"`
	Summary         authoringEvalSummary  `json:"summary"`
	Results         []authoringEvalResult `json:"results"`
}

type authoringEvalSummary struct {
	Total             int            `json:"total"`
	Passed            int            `json:"passed"`
	Failed            int            `json:"failed"`
	ByObservedOutcome map[string]int `json:"by_observed_outcome,omitempty"`
	ByFailureFamily   map[string]int `json:"by_failure_family,omitempty"`
}

type authoringEvalResult struct {
	Name                  string                 `json:"name"`
	Fixture               string                 `json:"fixture"`
	VariantID             string                 `json:"variant_id,omitempty"`
	Brief                 string                 `json:"brief"`
	Class                 string                 `json:"class,omitempty"`
	ExpectedOutcome       string                 `json:"expected_outcome,omitempty"`
	ExpectedFailureFamily string                 `json:"expected_failure_family,omitempty"`
	ObservedOutcome       string                 `json:"observed_outcome"`
	Passed                bool                   `json:"passed"`
	Provider              string                 `json:"provider,omitempty"`
	Model                 string                 `json:"model,omitempty"`
	PromptVersion         string                 `json:"prompt_version,omitempty"`
	LLMCallCount          int                    `json:"llm_call_count"`
	GeneratedProject      string                 `json:"generated_project,omitempty"`
	GeneratedIntent       string                 `json:"generated_intent,omitempty"`
	TranscriptPath        string                 `json:"transcript_path,omitempty"`
	QualityReport         string                 `json:"quality_report,omitempty"`
	FailureFamily         string                 `json:"failure_family,omitempty"`
	FailureCodes          []string               `json:"failure_codes,omitempty"`
	CredentialScanStatus  string                 `json:"credential_scan_status,omitempty"`
	CredentialDiagnostics []authoring.Diagnostic `json:"credential_scan_diagnostics,omitempty"`
	Blocking              int                    `json:"blocking"`
	Warning               int                    `json:"warning"`
	Advisory              int                    `json:"advisory"`
	ReferenceIssues       []evalpkg.CompareIssue `json:"reference_issues,omitempty"`
	Error                 string                 `json:"error,omitempty"`
}

type authoringEvalItem struct {
	Fixture               string
	FixtureName           string
	VariantID             string
	Brief                 string
	Class                 string
	ExpectedOutcome       string
	ExpectedFailureFamily string
}

type authoringEvalExtractorFactory func(provider, model string, temperature float64, calls *[]replayLLMCall) (elicitor.Extractor, string, string, error)

var newAuthoringEvalExtractor authoringEvalExtractorFactory = defaultAuthoringEvalExtractor

func runAuthoringEval(args []string, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("icot authoring-eval", flag.ContinueOnError)
	fs.SetOutput(out)
	root := fs.String("root", "examples/eval", "Directory containing eval example subdirectories")
	name := fs.String("name", "", "Run one eval fixture by directory name")
	includeVariants := fs.Bool("include-variants", false, "Run variants declared by reference/authoring-variants.json instead of only the fixture brief")
	provider := fs.String("provider", providerFromEnv(), "LLM provider for real iCoT extraction")
	model := fs.String("model", "", "LLM model for real iCoT extraction")
	temperature := fs.Float64("temperature", 0.2, "LLM extraction temperature")
	timeout := fs.Duration("timeout", 2*time.Minute, "Timeout per authoring attempt")
	outDir := fs.String("out", filepath.Join("eval", "runs", "icot-authoring-eval-"+time.Now().UTC().Format("20060102T150405Z")), "Output directory for authoring-eval artifacts")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: icot authoring-eval [--root examples/eval] [--include-variants] [--provider copilot-api --model gpt-5.4-mini] [--out eval/runs/icot-authoring-eval-<ts>]\n\n")
		fmt.Fprintf(fs.Output(), "Runs optional real-LLM natural-language iCoT authoring evidence. This is not part of provider-free release gates.\n\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	fixtures := discoverReplayFixtures(*root, *name)
	if len(fixtures) == 0 {
		fmt.Fprintf(errOut, "no eval fixtures found under %s\n", *root)
		return 1
	}
	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	report := authoringEvalReport{
		Version:         authoringEvalReportVersion,
		Status:          statusPass,
		Root:            *root,
		OutDir:          *outDir,
		Provider:        strings.TrimSpace(*provider),
		Model:           strings.TrimSpace(*model),
		PromptVersion:   elicitor.PromptVersion,
		IncludeVariants: *includeVariants,
		Summary: authoringEvalSummary{
			ByObservedOutcome: map[string]int{},
			ByFailureFamily:   map[string]int{},
		},
	}
	for _, fixture := range fixtures {
		items := authoringEvalItems(fixture, *includeVariants)
		if len(items) == 0 {
			continue
		}
		for _, item := range items {
			result := runAuthoringEvalItem(item, *provider, *model, *temperature, *timeout, *outDir)
			appendAuthoringEvalResult(&report, result, out)
		}
	}
	reportPath := filepath.Join(*outDir, "authoring-eval.json")
	redacted, err := writeAuthoringEvalReportFile(reportPath, report)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	fmt.Fprintf(out, "icot authoring-eval: wrote %s\n", reportPath)
	if redacted || report.Status != statusPass {
		return 1
	}
	return 0
}

func authoringEvalItems(fixture string, includeVariants bool) []authoringEvalItem {
	fixtureName := filepath.Base(filepath.Clean(fixture))
	if includeVariants {
		variants, err := evalpkg.ReadAuthoringVariants(filepath.Join(fixture, "reference", "authoring-variants.json"))
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return []authoringEvalItem{{
				Fixture:         fixture,
				FixtureName:     fixtureName,
				VariantID:       "authoring-variants",
				Brief:           err.Error(),
				ExpectedOutcome: "policy_error",
			}}
		}
		out := make([]authoringEvalItem, 0, len(variants.Variants))
		for _, variant := range variants.Variants {
			out = append(out, authoringEvalItem{
				Fixture:               fixture,
				FixtureName:           fixtureName,
				VariantID:             variant.ID,
				Brief:                 variant.Brief,
				Class:                 variant.Class,
				ExpectedOutcome:       variant.ExpectedOutcome,
				ExpectedFailureFamily: variant.ExpectedFailureFamily,
			})
		}
		return out
	}
	brief := ""
	if data, err := os.ReadFile(filepath.Join(fixture, "project.md")); err == nil {
		if project, loadErr := projectwizard.LoadAnswersFromMarkdown(string(data)); loadErr == nil {
			brief = project.Goal
		}
	}
	if strings.TrimSpace(brief) == "" {
		if intent, err := rollout.ParseIntentFile(filepath.Join(fixture, "reference", "intent.hcl")); err == nil && intent.Workflow != nil {
			brief = intent.Workflow.Description
		}
	}
	if strings.TrimSpace(brief) == "" {
		return nil
	}
	return []authoringEvalItem{{
		Fixture:         fixture,
		FixtureName:     fixtureName,
		Brief:           brief,
		Class:           "fixture-brief",
		ExpectedOutcome: statusPass,
	}}
}

func runAuthoringEvalItem(item authoringEvalItem, provider, model string, temperature float64, timeout time.Duration, outDir string) authoringEvalResult {
	name := item.FixtureName
	if item.VariantID != "" {
		name += "#" + item.VariantID
	}
	result := authoringEvalResult{
		Name:                  name,
		Fixture:               item.FixtureName,
		VariantID:             item.VariantID,
		Brief:                 item.Brief,
		Class:                 item.Class,
		ExpectedOutcome:       item.ExpectedOutcome,
		ExpectedFailureFamily: item.ExpectedFailureFamily,
		PromptVersion:         elicitor.PromptVersion,
	}
	if item.ExpectedOutcome == "policy_error" {
		result.ObservedOutcome = "policy_error"
		result.Error = item.Brief
		result.FailureFamily = failureUnknown
		return result
	}
	workspace := filepath.Join(outDir, "workspaces", safeScorecardName(strings.ReplaceAll(name, "#", "__")))
	_ = os.RemoveAll(workspace)
	if err := copySeedSourceArtifacts(item.Fixture, workspace, true); err != nil {
		result.ObservedOutcome = "icot_fail"
		result.Error = err.Error()
		result.FailureFamily = failureUnknown
		return result
	}
	var calls []replayLLMCall
	extractor, actualProvider, actualModel, err := newAuthoringEvalExtractor(provider, model, temperature, &calls)
	result.Provider = actualProvider
	result.Model = actualModel
	if err != nil {
		result.ObservedOutcome = "icot_fail"
		result.Error = err.Error()
		result.FailureFamily = failureUnknown
		result.Passed = authoringEvalOutcomeMatches(result)
		return result
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	seed := elicitor.Session{}
	preDraft, draftErr := extractor.Draft(ctx, elicitor.DraftRequest{
		Opening: item.Brief,
		Brief:   item.Brief,
	})
	if draftErr != nil {
		result.LLMCallCount = len(calls)
		result.ObservedOutcome = "icot_fail"
		result.Error = draftErr.Error()
		result.FailureFamily = failureFamilyForDetail(draftErr.Error())
		result.Passed = authoringEvalOutcomeMatches(result)
		return result
	}
	if elicitor.LooksLikeSession(preDraft) {
		seed = preDraft
		seed.Normalize()
	}
	var stdout bytes.Buffer
	transcriptPath := filepath.Join(outDir, "transcripts", safeScorecardName(strings.ReplaceAll(name, "#", "__"))+".json")
	artifacts, err := elicitor.Run(ctx, strings.NewReader(item.Brief+"\n"), &stdout, seed, elicitor.Options{
		ExampleDir:     workspace,
		NoLLM:          false,
		Extractor:      extractor,
		DefaultMode:    authoring.PromptDefaultsSilent,
		TranscriptPath: transcriptPath,
	})
	result.LLMCallCount = len(calls)
	result.TranscriptPath = transcriptPath
	if err != nil {
		result.ObservedOutcome = observedAuthoringEvalErrorOutcome(err)
		result.Error = err.Error()
		result.FailureFamily = failureFamilyForDetail(err.Error())
		result.Passed = authoringEvalOutcomeMatches(result)
		return result
	}
	projectPath := filepath.Join(workspace, "project.md")
	intentPath := filepath.Join(workspace, "workflows", "intent.hcl")
	if err := writeArtifacts(projectPath, intentPath, artifacts, true, true, strings.NewReader(""), io.Discard); err != nil {
		result.ObservedOutcome = "icot_fail"
		result.Error = err.Error()
		result.FailureFamily = failureIntentParse
		result.Passed = authoringEvalOutcomeMatches(result)
		return result
	}
	result.GeneratedProject = projectPath
	result.GeneratedIntent = intentPath
	if scanned := scanAuthoringEvalResultCredentials(result); len(scanned) > 0 {
		result.CredentialScanStatus = statusFail
		result.CredentialDiagnostics = scanned
		result.ObservedOutcome = "icot_fail"
		result.Error = "authoring-eval output appears to contain a literal credential value"
		result.FailureFamily = failureCredentialBindingGap
		result.FailureCodes = []string{"authoring_eval.literal_credential"}
		result.Passed = false
		return result
	}
	result.CredentialScanStatus = statusPass
	lintCode, lintFamily, lintCodes, lintErr := runAuthoringEvalLint(workspace)
	if lintCode != 0 {
		result.ObservedOutcome = "icot_fail"
		result.Error = lintErr
		result.FailureFamily = firstNonEmpty(lintFamily, failureUnknown)
		result.FailureCodes = lintCodes
		result.Passed = authoringEvalOutcomeMatches(result)
		return result
	}
	_, quality, err := synthesize.PackageFromIntent(context.Background(), synthesize.Options{ExampleDir: workspace})
	result.QualityReport = filepath.Join(workspace, "expected", "quality.json")
	if err != nil {
		result.ObservedOutcome = "build_fail"
		result.Error = err.Error()
		result.FailureCodes = []string{"build:error"}
		result.FailureFamily = failureFamilyForDetail(err.Error())
		result.Passed = authoringEvalOutcomeMatches(result)
		return result
	}
	if quality == nil || !quality.Passed() {
		result.ObservedOutcome = "build_fail"
		result.FailureCodes = scorecardFailedQualityCodes(quality)
		result.Error = scorecardQualityFailureDetails(quality)
		result.FailureFamily = failureFamilyForQualityCode(firstFailedQualityCode(quality))
		result.Passed = authoringEvalOutcomeMatches(result)
		return result
	}
	policy, _ := evalpkg.ReadReferencePolicy(filepath.Join(item.Fixture, "reference", "policy.json"))
	issues, compareErr := evalpkg.CompareIntentFiles(intentPath, filepath.Join(item.Fixture, "reference", "intent.hcl"), policy)
	if compareErr != nil {
		result.ObservedOutcome = "build_fail"
		result.Error = compareErr.Error()
		result.FailureFamily = failureIntentParse
		result.Passed = authoringEvalOutcomeMatches(result)
		return result
	}
	result.ReferenceIssues = issues
	for _, issue := range issues {
		switch issue.Severity {
		case "blocking":
			result.Blocking++
		case "warning":
			result.Warning++
		case "advisory":
			result.Advisory++
		}
	}
	if !authoringEvalDriftPasses(result, policy) {
		result.ObservedOutcome = "build_fail"
		result.FailureFamily = failureBadRequestMapping
	} else {
		result.ObservedOutcome = statusPass
	}
	result.Passed = authoringEvalOutcomeMatches(result)
	return result
}

func scanAuthoringEvalResultCredentials(result authoringEvalResult) []authoring.Diagnostic {
	var artifacts []authoring.Artifact
	for _, path := range []string{result.GeneratedProject, result.GeneratedIntent, result.TranscriptPath} {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		artifacts = append(artifacts, authoring.Artifact{
			Path:      path,
			MediaType: "text/plain",
			Content:   data,
		})
	}
	return authoring.ScanCredentialValues(artifacts)
}

func writeAuthoringEvalReportFile(path string, report authoringEvalReport) (bool, error) {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return false, err
	}
	data = append(data, '\n')
	redacted := false
	if authoring.ContainsLikelyCredentialValue(data) {
		redacted = true
		safeReport := authoringEvalReport{
			Version: report.Version,
			Status:  statusFail,
			Root:    report.Root,
			OutDir:  report.OutDir,
			Summary: authoringEvalSummary{
				Total:             report.Summary.Total,
				Passed:            0,
				Failed:            report.Summary.Total,
				ByObservedOutcome: map[string]int{"icot_fail": report.Summary.Total},
				ByFailureFamily:   map[string]int{failureCredentialBindingGap: report.Summary.Total},
			},
			Results: []authoringEvalResult{{
				Name:                 "authoring-eval-redacted",
				ObservedOutcome:      "icot_fail",
				FailureFamily:        failureCredentialBindingGap,
				FailureCodes:         []string{"authoring_eval.report_literal_credential"},
				CredentialScanStatus: statusFail,
				Error:                "authoring-eval report was not written because it appears to contain a literal credential value",
			}},
		}
		data, err = json.MarshalIndent(safeReport, "", "  ")
		if err != nil {
			return false, err
		}
		data = append(data, '\n')
	}
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return false, err
		}
	}
	return redacted, os.WriteFile(path, data, 0o644)
}

func defaultAuthoringEvalExtractor(provider, model string, temperature float64, calls *[]replayLLMCall) (elicitor.Extractor, string, string, error) {
	llm, actualProvider, actualModel, err := runner.NewLLMClientFromEnvWithOptions(provider, model, runner.LLMOptions{
		Temperature: &temperature,
	})
	if err != nil {
		return nil, "", "", err
	}
	chat, ok := llm.(rollout.ChatClient)
	if !ok {
		return nil, actualProvider, actualModel, fmt.Errorf("provider %s does not support chat", actualProvider)
	}
	return elicitor.NewChatExtractor(&recordingChatClient{base: chat, calls: calls}, &temperature), actualProvider, actualModel, nil
}

func runAuthoringEvalLint(example string) (int, string, []string, string) {
	var stdout, stderr bytes.Buffer
	code := runLint([]string{"--example", example, "--json"}, &stdout, &stderr)
	if code == 0 {
		return 0, "", nil, ""
	}
	var report lintReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err == nil {
		var codes []string
		for _, check := range report.ProjectChecks {
			if check.Status == "fail" {
				codes = append(codes, check.Code)
			}
		}
		if report.IntentCheck != nil && report.IntentCheck.Status == "fail" {
			codes = append(codes, report.IntentCheck.Code)
		}
		return code, report.FailureFamily, codes, strings.TrimSpace(firstNonEmpty(report.FailureFamily, stderr.String()))
	}
	return code, failureFamilyForDetail(stderr.String()), nil, strings.TrimSpace(stderr.String())
}

func observedAuthoringEvalErrorOutcome(err error) string {
	text := strings.ToLower(err.Error())
	if strings.Contains(text, "unexpected eof") || strings.Contains(text, "needs input") {
		return statusNeedsInput
	}
	return "icot_fail"
}

func authoringEvalDriftPasses(result authoringEvalResult, policy evalpkg.ReferencePolicy) bool {
	if policy.MaxBlocking != nil {
		if result.Blocking > *policy.MaxBlocking {
			return false
		}
	} else if result.Blocking > 0 {
		return false
	}
	if policy.MaxWarning != nil && result.Warning > *policy.MaxWarning {
		return false
	}
	if policy.MaxAdvisory != nil && result.Advisory > *policy.MaxAdvisory {
		return false
	}
	return true
}

func authoringEvalOutcomeMatches(result authoringEvalResult) bool {
	expected := result.ExpectedOutcome
	if expected == "" {
		expected = statusPass
	}
	if result.ObservedOutcome != expected {
		return false
	}
	if expected != statusPass && result.ExpectedFailureFamily != "" && result.FailureFamily != result.ExpectedFailureFamily {
		return false
	}
	return true
}

func appendAuthoringEvalResult(report *authoringEvalReport, result authoringEvalResult, out io.Writer) {
	report.Results = append(report.Results, result)
	report.Summary.Total++
	report.Summary.ByObservedOutcome[result.ObservedOutcome]++
	if result.FailureFamily != "" {
		report.Summary.ByFailureFamily[result.FailureFamily]++
	}
	if result.Passed {
		report.Summary.Passed++
		fmt.Fprintf(out, "icot authoring-eval: pass %s\n", result.Name)
		return
	}
	report.Summary.Failed++
	report.Status = statusFail
	fmt.Fprintf(out, "icot authoring-eval: fail %s - expected %s, observed %s\n", result.Name, result.ExpectedOutcome, result.ObservedOutcome)
}
