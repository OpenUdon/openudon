package icot

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	publicreadiness "github.com/OpenUdon/authoring/readiness"
	publicreport "github.com/OpenUdon/authoring/report"
	"github.com/OpenUdon/evidence/digest"
	"github.com/OpenUdon/openudon/internal/icot/elicitor"
	"github.com/OpenUdon/openudon/internal/icotreport"
	"github.com/OpenUdon/openudon/internal/synthesize"
)

const (
	authorReportVersion    = "openudon.icot-author-report.v1"
	lintReportVersion      = "openudon.icot-lint-report.v1"
	scorecardReportVersion = "openudon.icot-scorecard.v1"
	repairReportVersion    = "openudon.icot-repair-report.v1"

	statusPass       = "pass"
	statusFail       = "fail"
	statusNeedsInput = "needs_input"
	statusDryRun     = "dry_run"

	failureMissingAPISource     = icotreport.FailureMissingAPISource
	failureMissingOperation     = icotreport.FailureMissingOperation
	failureBadRequestMapping    = icotreport.FailureBadRequestMapping
	failureBadResponsePath      = icotreport.FailureBadResponsePath
	failureCredentialBindingGap = icotreport.FailureCredentialBindingGap
	failureSideEffectPolicyGap  = icotreport.FailureSideEffectPolicyGap
	failureAmbiguousUserIntent  = icotreport.FailureAmbiguousUserIntent
	failureRuntimeProfileGap    = icotreport.FailureRuntimeProfileGap
	failureIntentParse          = icotreport.FailureIntentParse
	failureBuildError           = icotreport.FailureBuildError
	failureUnknown              = icotreport.FailureUnknown

	readinessClassifierVersion = "icot-readiness.v1"

	retentionReleaseEvidence = "release_evidence"
	retentionLocalEphemeral  = "local_ephemeral"
)

type authorReport struct {
	Version          string                    `json:"version"`
	Status           string                    `json:"status"`
	Example          string                    `json:"example"`
	ProjectPath      string                    `json:"project_path,omitempty"`
	IntentPath       string                    `json:"intent_path,omitempty"`
	TranscriptPath   string                    `json:"transcript_path,omitempty"`
	FailureFamily    string                    `json:"failure_family,omitempty"`
	TopIssue         *elicitor.ReadinessIssue  `json:"top_issue,omitempty"`
	ReadinessIssues  []elicitor.ReadinessIssue `json:"readiness_issues,omitempty"`
	SuggestedAnswer  string                    `json:"suggested_answer,omitempty"`
	GeneratedProject string                    `json:"generated_project,omitempty"`
	GeneratedIntent  string                    `json:"generated_intent,omitempty"`
	Error            string                    `json:"error,omitempty"`
}

type lintReport struct {
	Version       string                    `json:"version"`
	Status        string                    `json:"status"`
	Example       string                    `json:"example,omitempty"`
	File          string                    `json:"file"`
	ProjectChecks []synthesize.QualityCheck `json:"project_checks"`
	IntentCheck   *synthesize.QualityCheck  `json:"intent_check,omitempty"`
	DriftChecks   []elicitor.DriftCheck     `json:"drift_checks,omitempty"`
	FailureFamily string                    `json:"failure_family,omitempty"`
}

type scorecardReport struct {
	Version                      string                  `json:"version"`
	Status                       string                  `json:"status"`
	Root                         string                  `json:"root"`
	OutDir                       string                  `json:"out_dir"`
	RunID                        string                  `json:"run_id,omitempty"`
	GeneratedAt                  string                  `json:"generated_at,omitempty"`
	Commit                       string                  `json:"commit,omitempty"`
	PromptVersion                string                  `json:"prompt_version,omitempty"`
	ReadinessClassifierVersion   string                  `json:"readiness_classifier_version,omitempty"`
	ScorecardCommand             string                  `json:"scorecard_command,omitempty"`
	RetentionClass               string                  `json:"retention_class,omitempty"`
	ContainsProviderOutput       bool                    `json:"contains_provider_output"`
	SafeToArchive                bool                    `json:"safe_to_archive"`
	RedactionRequiredBeforeShare bool                    `json:"redaction_required_before_share"`
	Summary                      scorecardSummary        `json:"summary"`
	Results                      []scorecardResult       `json:"results"`
	AuthoringScorecard           *publicreport.Scorecard `json:"authoring_scorecard,omitempty"`
}

type scorecardSummary struct {
	Total                   int                       `json:"total"`
	Passed                  int                       `json:"passed"`
	Failed                  int                       `json:"failed"`
	ByClass                 map[string]int            `json:"by_class,omitempty"`
	ByFailureFamily         map[string]int            `json:"by_failure_family,omitempty"`
	ByObservedOutcome       map[string]int            `json:"by_observed_outcome,omitempty"`
	ByProviderFamily        map[string]int            `json:"by_provider_family,omitempty"`
	ByProviderFailureFamily map[string]map[string]int `json:"by_provider_failure_family,omitempty"`
	ByVariantClass          map[string]int            `json:"by_variant_class,omitempty"`
	MissingDetailFalsePass  int                       `json:"missing_detail_false_pass"`
	UnsafeFalsePass         int                       `json:"unsafe_false_pass"`
	NeedsInputDiagnosticGap int                       `json:"needs_input_diagnostic_gap"`
}

type scorecardResult struct {
	Name                  string   `json:"name"`
	Kind                  string   `json:"kind,omitempty"`
	Fixture               string   `json:"fixture,omitempty"`
	VariantID             string   `json:"variant_id,omitempty"`
	Brief                 string   `json:"brief,omitempty"`
	Class                 string   `json:"class,omitempty"`
	ExpectedOutcome       string   `json:"expected_outcome,omitempty"`
	ExpectedFailureFamily string   `json:"expected_failure_family,omitempty"`
	ExpectedTopIssueCode  string   `json:"expected_top_issue_code,omitempty"`
	ExpectedTopIssueSlot  string   `json:"expected_top_issue_slot,omitempty"`
	ObservedOutcome       string   `json:"observed_outcome"`
	Passed                bool     `json:"passed"`
	FailureFamily         string   `json:"failure_family,omitempty"`
	FailureCodes          []string `json:"failure_codes,omitempty"`
	TopIssueCode          string   `json:"top_issue_code,omitempty"`
	TopIssueSlot          string   `json:"top_issue_slot,omitempty"`
	TopIssueMessage       string   `json:"top_issue_message,omitempty"`
	SuggestedAnswer       string   `json:"suggested_answer,omitempty"`
	Detail                string   `json:"detail,omitempty"`
	ProviderFamilies      []string `json:"provider_families,omitempty"`
	Tags                  []string `json:"tags,omitempty"`
	ExampleDir            string   `json:"example_dir,omitempty"`
	GeneratedProject      string   `json:"generated_project,omitempty"`
	GeneratedIntent       string   `json:"generated_intent,omitempty"`
	QualityReport         string   `json:"quality_report,omitempty"`
}

type repairReport struct {
	Version          string         `json:"version"`
	Status           string         `json:"status"`
	Example          string         `json:"example"`
	DryRun           bool           `json:"dry_run,omitempty"`
	MaxAttempts      int            `json:"max_attempts"`
	Attempts         int            `json:"attempts"`
	Applied          []repairChange `json:"applied,omitempty"`
	Rejected         []string       `json:"rejected,omitempty"`
	FailureFamily    string         `json:"failure_family,omitempty"`
	FailureCodes     []string       `json:"failure_codes,omitempty"`
	QualityReport    string         `json:"quality_report,omitempty"`
	GeneratedProject string         `json:"generated_project,omitempty"`
	GeneratedIntent  string         `json:"generated_intent,omitempty"`
	Error            string         `json:"error,omitempty"`
}

type repairChange struct {
	Slot   string `json:"slot"`
	Before string `json:"before,omitempty"`
	After  string `json:"after,omitempty"`
	Reason string `json:"reason,omitempty"`
}

func failureFamilyForReadiness(code string) string {
	return icotreport.FailureFamilyForReadiness(code)
}

func failureFamilyForQualityCode(code string) string {
	return icotreport.FailureFamilyForQualityCode(code)
}

func firstFailedQualityCode(report *synthesize.QualityReport) string {
	if report == nil {
		return ""
	}
	for _, check := range report.Checks {
		if check.Status == "fail" {
			return check.Code
		}
	}
	return ""
}

func reportRunID(kind, generatedAt, commit string) string {
	kind = safeScorecardName(kind)
	generatedAt = strings.NewReplacer(":", "", "-", "", "T", "", "Z", "").Replace(strings.TrimSpace(generatedAt))
	commit = safeScorecardName(commit)
	if generatedAt == "" {
		generatedAt = "unknown-time"
	}
	if commit == "" {
		commit = "unknown-commit"
	}
	return kind + "-" + generatedAt + "-" + commit
}

func writeJSONReportWithDigest(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return err
	}
	digestLine := digest.SHA256Bytes(data).Value + "  " + filepath.Base(path) + "\n"
	return os.WriteFile(path+".sha256", []byte(digestLine), 0o644)
}

func verifyJSONReportDigest(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	digestData, err := os.ReadFile(path + ".sha256")
	if err != nil {
		return fmt.Errorf("read digest sidecar %s: %w", path+".sha256", err)
	}
	fields := strings.Fields(string(digestData))
	if len(fields) == 0 {
		return fmt.Errorf("digest sidecar %s is empty", path+".sha256")
	}
	want := strings.ToLower(strings.TrimSpace(fields[0]))
	got := digest.SHA256Bytes(data).Value
	if want != got {
		return fmt.Errorf("digest mismatch for %s: sidecar %s, computed %s", path, want, got)
	}
	if len(fields) > 1 && fields[1] != filepath.Base(path) {
		return fmt.Errorf("digest sidecar names %s, want %s", fields[1], filepath.Base(path))
	}
	return nil
}

func writeScorecardReportFile(path string, report scorecardReport) error {
	scorecard := authoringScorecardContract(report)
	report.AuthoringScorecard = &scorecard
	if err := validateScorecardReport(report); err != nil {
		return err
	}
	return writeJSONReportWithDigest(path, report)
}

func validateAuthorReportContract(report authorReport) error {
	result := authoringAgentResultContract(report)
	if result.Version != publicreport.Version {
		return fmt.Errorf("authoring result version = %q, want %q", result.Version, publicreport.Version)
	}
	if result.Status == "" {
		return fmt.Errorf("authoring result status is empty")
	}
	if report.Status == statusNeedsInput && result.TopIssue == nil {
		return fmt.Errorf("authoring needs_input result missing top issue")
	}
	return nil
}

func authoringAgentResultContract(report authorReport) publicreport.Result {
	readiness := publicreadiness.Evaluate(authoringReadinessIssues(report.ReadinessIssues))
	result := publicreport.Result{
		Status:    authoringStatus(report.Status),
		Summary:   firstNonEmpty(report.Error, report.FailureFamily),
		Readiness: &readiness,
		Metadata: map[string]string{
			"openudon_version": authorReportVersion,
			"example":          report.Example,
		},
	}
	if len(report.ReadinessIssues) == 0 {
		result.Readiness = nil
	}
	if report.TopIssue != nil {
		issue := authoringReadinessIssue(*report.TopIssue)
		result.TopIssue = &issue
	}
	return publicreport.Normalize(result)
}

func authoringStatus(status string) string {
	switch status {
	case statusPass:
		return publicreport.StatusComplete
	case statusNeedsInput:
		return publicreport.StatusNeedsInput
	case statusFail:
		return publicreport.StatusFailed
	default:
		return publicreport.StatusForError(fmt.Errorf("openudon status %s", status))
	}
}

func authoringReadinessIssues(issues []elicitor.ReadinessIssue) []publicreadiness.Issue {
	out := make([]publicreadiness.Issue, 0, len(issues))
	for _, issue := range issues {
		out = append(out, authoringReadinessIssue(issue))
	}
	return out
}

func authoringReadinessIssue(issue elicitor.ReadinessIssue) publicreadiness.Issue {
	return publicreadiness.Issue{
		Code:            issue.Code,
		Severity:        issue.Severity,
		Slot:            issue.Slot,
		Message:         issue.Message,
		SuggestedAnswer: issue.SuggestedAnswer,
	}
}

func validateScorecardReport(report scorecardReport) error {
	if report.Version != scorecardReportVersion {
		return fmt.Errorf("scorecard report version = %q, want %q", report.Version, scorecardReportVersion)
	}
	if diagnostics := publicreport.ValidateScorecard(authoringScorecardContract(report)); len(diagnostics) > 0 {
		return fmt.Errorf("authoring scorecard contract: %s", diagnostics[0].Message)
	}
	if report.AuthoringScorecard != nil {
		if diagnostics := publicreport.ValidateScorecard(*report.AuthoringScorecard); len(diagnostics) > 0 {
			return fmt.Errorf("embedded authoring scorecard: %s", diagnostics[0].Message)
		}
		if report.AuthoringScorecard.Summary.Total != report.Summary.Total {
			return fmt.Errorf("embedded authoring scorecard total = %d, want %d", report.AuthoringScorecard.Summary.Total, report.Summary.Total)
		}
	}
	for name, value := range map[string]string{
		"status":                       report.Status,
		"root":                         report.Root,
		"out_dir":                      report.OutDir,
		"run_id":                       report.RunID,
		"generated_at":                 report.GeneratedAt,
		"prompt_version":               report.PromptVersion,
		"readiness_classifier_version": report.ReadinessClassifierVersion,
		"scorecard_command":            report.ScorecardCommand,
		"retention_class":              report.RetentionClass,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("scorecard report missing %s", name)
		}
	}
	if err := validateReportRetention(report.RetentionClass, report.ContainsProviderOutput, report.SafeToArchive, report.RedactionRequiredBeforeShare, retentionReleaseEvidence); err != nil {
		return fmt.Errorf("scorecard report retention metadata: %w", err)
	}
	if report.Summary.Total != len(report.Results) {
		return fmt.Errorf("scorecard summary total = %d, results = %d", report.Summary.Total, len(report.Results))
	}
	passed, failed, diagnosticGaps := 0, 0, 0
	for _, result := range report.Results {
		if result.Passed {
			passed++
		} else {
			failed++
		}
		if result.Kind == "authoring_variant" && result.Passed != scorecardVariantOutcomeMatches(result) {
			return fmt.Errorf("scorecard result %s passed=%v does not match expected/observed outcome", result.Name, result.Passed)
		}
		if result.Kind == "authoring_variant" && result.ObservedOutcome == statusNeedsInput {
			if strings.TrimSpace(result.TopIssueCode) == "" || strings.TrimSpace(result.TopIssueMessage) == "" || strings.TrimSpace(result.SuggestedAnswer) == "" {
				diagnosticGaps++
			}
		}
	}
	if report.Summary.Passed != passed || report.Summary.Failed != failed {
		return fmt.Errorf("scorecard summary pass/fail = %d/%d, results = %d/%d", report.Summary.Passed, report.Summary.Failed, passed, failed)
	}
	if report.Summary.NeedsInputDiagnosticGap != diagnosticGaps {
		return fmt.Errorf("scorecard needs_input diagnostic gap = %d, results = %d", report.Summary.NeedsInputDiagnosticGap, diagnosticGaps)
	}
	if report.Status == statusPass && failed != 0 {
		return fmt.Errorf("scorecard status pass with %d failed result(s)", failed)
	}
	if report.Status == statusFail && failed == 0 {
		return fmt.Errorf("scorecard status fail with no failed results")
	}
	return nil
}

func authoringScorecardContract(report scorecardReport) publicreport.Scorecard {
	var variants []publicreport.VariantResult
	for _, result := range report.Results {
		variants = append(variants, publicreport.VariantResult{
			FixtureID:       firstNonEmpty(result.Fixture, result.Name),
			VariantID:       result.VariantID,
			Group:           firstNonEmpty(result.Kind, result.Class),
			ExpectedOutcome: result.ExpectedOutcome,
			ObservedOutcome: result.ObservedOutcome,
			FailureFamily:   result.FailureFamily,
			Message:         result.Detail,
			Metadata: map[string]string{
				"openudon_result": result.Name,
				"openudon_class":  result.Class,
			},
		})
	}
	return publicreport.NormalizeScorecard(publicreport.Scorecard{
		Name:     "openudon-icot",
		Variants: variants,
		Report: authoringReportMetadata(
			report.RunID,
			report.ScorecardCommand,
			report.Commit,
			report.GeneratedAt,
			report.RetentionClass,
			report.ContainsProviderOutput,
			report.SafeToArchive,
			report.RedactionRequiredBeforeShare,
		),
	})
}

func validateAuthoringEvalReport(report authoringEvalReport) error {
	if report.Version != authoringEvalReportVersion {
		return fmt.Errorf("authoring-eval report version = %q, want %q", report.Version, authoringEvalReportVersion)
	}
	for name, value := range map[string]string{
		"status":                       report.Status,
		"root":                         report.Root,
		"out_dir":                      report.OutDir,
		"run_id":                       report.RunID,
		"generated_at":                 report.GeneratedAt,
		"prompt_version":               report.PromptVersion,
		"readiness_classifier_version": report.ReadinessClassifierVersion,
		"authoring_eval_command":       report.AuthoringEvalCommand,
		"retention_class":              report.RetentionClass,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("authoring-eval report missing %s", name)
		}
	}
	if err := validateReportRetention(report.RetentionClass, report.ContainsProviderOutput, report.SafeToArchive, report.RedactionRequiredBeforeShare, retentionLocalEphemeral); err != nil {
		return fmt.Errorf("authoring-eval report retention metadata: %w", err)
	}
	if report.Summary.Total != len(report.Results) {
		return fmt.Errorf("authoring-eval summary total = %d, results = %d", report.Summary.Total, len(report.Results))
	}
	passed, failed := 0, 0
	for _, result := range report.Results {
		if result.Passed {
			passed++
		} else {
			failed++
		}
		if result.Passed != authoringEvalOutcomeMatches(result) {
			return fmt.Errorf("authoring-eval result %s passed=%v does not match expected/observed outcome", result.Name, result.Passed)
		}
		if result.FailureCategory != "" && !isValidAuthoringEvalFailureCategory(result.FailureCategory) {
			return fmt.Errorf("authoring-eval result %s has unsupported failure_category %q", result.Name, result.FailureCategory)
		}
	}
	if report.Summary.Passed != passed || report.Summary.Failed != failed {
		return fmt.Errorf("authoring-eval summary pass/fail = %d/%d, results = %d/%d", report.Summary.Passed, report.Summary.Failed, passed, failed)
	}
	if report.Status == statusPass && failed != 0 {
		return fmt.Errorf("authoring-eval status pass with %d failed result(s)", failed)
	}
	if report.Status == statusFail && failed == 0 {
		return fmt.Errorf("authoring-eval status fail with no failed results")
	}
	return nil
}

func validateReportRetention(retentionClass string, containsProviderOutput, safeToArchive, redactionRequiredBeforeShare bool, expectedClass string) error {
	metadata := authoringReportMetadata("", "", "", "", expectedClass, containsProviderOutput, safeToArchive, redactionRequiredBeforeShare)
	if metadata == nil {
		return fmt.Errorf("authoring report metadata is empty")
	}
	if retentionClass != expectedClass {
		return fmt.Errorf("retention_class = %q, want %q", retentionClass, expectedClass)
	}
	switch expectedClass {
	case retentionReleaseEvidence:
		if containsProviderOutput {
			return fmt.Errorf("contains_provider_output = true, want false")
		}
		if !safeToArchive {
			return fmt.Errorf("safe_to_archive = false, want true")
		}
		if redactionRequiredBeforeShare {
			return fmt.Errorf("redaction_required_before_share = true, want false")
		}
	case retentionLocalEphemeral:
		if !containsProviderOutput {
			return fmt.Errorf("contains_provider_output = false, want true")
		}
		if safeToArchive {
			return fmt.Errorf("safe_to_archive = true, want false")
		}
		if !redactionRequiredBeforeShare {
			return fmt.Errorf("redaction_required_before_share = false, want true")
		}
	default:
		return fmt.Errorf("unsupported retention_class %q", expectedClass)
	}
	return nil
}

func authoringReportMetadata(runID, command, commit, generatedAt, retentionClass string, containsProviderOutput, safeToArchive, redactionRequiredBeforeShare bool) *publicreport.ReportMetadata {
	return publicreport.NormalizeReportMetadata(&publicreport.ReportMetadata{
		RunID:             runID,
		Command:           command,
		Commit:            commit,
		GeneratedUTC:      generatedAt,
		RetentionClass:    authoringRetentionClass(retentionClass),
		ProviderOutput:    containsProviderOutput,
		ArchiveSafe:       safeToArchive,
		RedactionRequired: redactionRequiredBeforeShare,
		Metadata: map[string]string{
			"openudon_retention_class": retentionClass,
		},
	})
}

func authoringRetentionClass(retentionClass string) string {
	switch retentionClass {
	case retentionReleaseEvidence:
		return publicreport.RetentionArchive
	case retentionLocalEphemeral:
		return publicreport.RetentionEphemeral
	default:
		return publicreport.RetentionRun
	}
}

func isValidAuthoringEvalFailureCategory(category string) bool {
	switch category {
	case authoringEvalProviderUnavailable,
		authoringEvalProviderTimeout,
		authoringEvalMalformedModelJSON,
		authoringEvalStructuredOutputUnsupported,
		authoringEvalModelRefusal,
		authoringEvalIncompleteDraft,
		authoringEvalLintFail,
		authoringEvalCredentialScanFail,
		authoringEvalBuildFail,
		authoringEvalReferenceDrift,
		authoringEvalPolicyError,
		authoringEvalUnknown:
		return true
	default:
		return false
	}
}
