package icot

import (
	"strings"

	"github.com/OpenUdon/openudon/internal/icot/elicitor"
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

	failureMissingAPISource     = "missing_api_source"
	failureMissingOperation     = "missing_operation"
	failureBadRequestMapping    = "bad_request_mapping"
	failureBadResponsePath      = "bad_response_path"
	failureCredentialBindingGap = "credential_binding_gap"
	failureSideEffectPolicyGap  = "side_effect_policy_gap"
	failureAmbiguousUserIntent  = "ambiguous_user_intent"
	failureRuntimeProfileGap    = "runtime_profile_gap"
	failureIntentParse          = "intent_parse"
	failureBuildError           = "build_error"
	failureUnknown              = "unknown"
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
	Total             int            `json:"total"`
	Passed            int            `json:"passed"`
	Failed            int            `json:"failed"`
	ByClass           map[string]int `json:"by_class,omitempty"`
	ByFailureFamily   map[string]int `json:"by_failure_family,omitempty"`
	ByObservedOutcome map[string]int `json:"by_observed_outcome,omitempty"`
}

type scorecardResult struct {
	Name             string   `json:"name"`
	Class            string   `json:"class,omitempty"`
	ExpectedOutcome  string   `json:"expected_outcome,omitempty"`
	ObservedOutcome  string   `json:"observed_outcome"`
	Passed           bool     `json:"passed"`
	FailureFamily    string   `json:"failure_family,omitempty"`
	FailureCodes     []string `json:"failure_codes,omitempty"`
	Detail           string   `json:"detail,omitempty"`
	ExampleDir       string   `json:"example_dir,omitempty"`
	GeneratedProject string   `json:"generated_project,omitempty"`
	GeneratedIntent  string   `json:"generated_intent,omitempty"`
	QualityReport    string   `json:"quality_report,omitempty"`
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
	switch strings.TrimSpace(code) {
	case "missing_api_doc":
		return failureMissingAPISource
	case "missing_operation":
		return failureMissingOperation
	case "missing_required_request_values", "conflicting_mapping", "low_confidence_mapping":
		return failureBadRequestMapping
	case "missing_credential_bindings":
		return failureCredentialBindingGap
	case "missing_side_effect_policy", "unconfirmed_side_effect_commitment":
		return failureSideEffectPolicyGap
	case "missing_goal", "missing_outputs", "conflicting_decision_evidence", "low_confidence_decision":
		return failureAmbiguousUserIntent
	case "intent_render_invalid", "missing_runtime_inputs":
		return failureIntentParse
	default:
		return failureUnknown
	}
}

func failureFamilyForQualityCode(code string) string {
	code = strings.TrimSpace(code)
	switch {
	case code == "":
		return ""
	case strings.Contains(code, "openapi_refs"), strings.Contains(code, "openapi.local"):
		return failureMissingAPISource
	case strings.Contains(code, "openapi_operations"):
		return failureMissingOperation
	case strings.Contains(code, "required_params"), strings.Contains(code, "binding_sources"), strings.Contains(code, "explicit"):
		return failureBadRequestMapping
	case strings.Contains(code, "response_paths"), strings.Contains(code, "sources"):
		return failureBadResponsePath
	case strings.Contains(code, "credential"):
		return failureCredentialBindingGap
	case strings.Contains(code, "side_effect"), strings.Contains(code, "runtime_policy"):
		return failureSideEffectPolicyGap
	case strings.Contains(code, "runtime"):
		return failureRuntimeProfileGap
	case strings.Contains(code, "intent.parse"), strings.Contains(code, "intent.slots"):
		return failureIntentParse
	default:
		return failureUnknown
	}
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
