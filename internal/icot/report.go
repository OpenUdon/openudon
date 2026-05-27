package icot

import (
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
	Version string            `json:"version"`
	Status  string            `json:"status"`
	Root    string            `json:"root"`
	OutDir  string            `json:"out_dir"`
	Summary scorecardSummary  `json:"summary"`
	Results []scorecardResult `json:"results"`
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
