package icotreport

import "strings"

const (
	FailureMissingAPISource     = "missing_api_source"
	FailureMissingOperation     = "missing_operation"
	FailureBadRequestMapping    = "bad_request_mapping"
	FailureBadResponsePath      = "bad_response_path"
	FailureCredentialBindingGap = "credential_binding_gap"
	FailureSideEffectPolicyGap  = "side_effect_policy_gap"
	FailureAmbiguousUserIntent  = "ambiguous_user_intent"
	FailureRuntimeProfileGap    = "runtime_profile_gap"
	FailureIntentParse          = "intent_parse"
	FailureBuildError           = "build_error"
	FailureUnknown              = "unknown"
)

var validFailureFamilies = map[string]bool{
	FailureMissingAPISource:     true,
	FailureMissingOperation:     true,
	FailureBadRequestMapping:    true,
	FailureBadResponsePath:      true,
	FailureCredentialBindingGap: true,
	FailureSideEffectPolicyGap:  true,
	FailureAmbiguousUserIntent:  true,
	FailureRuntimeProfileGap:    true,
	FailureIntentParse:          true,
	FailureBuildError:           true,
	FailureUnknown:              true,
}

func IsValidFailureFamily(value string) bool {
	return validFailureFamilies[strings.TrimSpace(value)]
}

func FailureFamilyForReadiness(code string) string {
	switch strings.TrimSpace(code) {
	case "missing_api_doc":
		return FailureMissingAPISource
	case "missing_operation":
		return FailureMissingOperation
	case "missing_required_request_values",
		"conflicting_mapping",
		"low_confidence_mapping",
		"invented_request_field",
		"invalid_request_body_path",
		"incompatible_request_value_type":
		return FailureBadRequestMapping
	case "missing_credential_bindings", "inline_secret_value", "undeclared_credential_reference":
		return FailureCredentialBindingGap
	case "missing_side_effect_policy", "unconfirmed_side_effect_commitment", "unsafe_review_bypass":
		return FailureSideEffectPolicyGap
	case "missing_goal", "missing_outputs", "missing_runtime_inputs", "conflicting_decision_evidence", "low_confidence_decision":
		return FailureAmbiguousUserIntent
	case "intent_render_invalid":
		return FailureIntentParse
	default:
		return FailureUnknown
	}
}

func FailureFamilyForQualityCode(code string) string {
	code = strings.TrimSpace(code)
	switch {
	case code == "":
		return ""
	case strings.Contains(code, "openapi_refs"), strings.Contains(code, "openapi.local"):
		return FailureMissingAPISource
	case strings.Contains(code, "openapi_operations"):
		return FailureMissingOperation
	case strings.Contains(code, "required_params"), strings.Contains(code, "binding_sources"), strings.Contains(code, "explicit"):
		return FailureBadRequestMapping
	case strings.Contains(code, "response_paths"), strings.Contains(code, "sources"):
		return FailureBadResponsePath
	case strings.Contains(code, "credential"):
		return FailureCredentialBindingGap
	case strings.Contains(code, "side_effect"), strings.Contains(code, "runtime_policy"):
		return FailureSideEffectPolicyGap
	case strings.Contains(code, "runtime"):
		return FailureRuntimeProfileGap
	case strings.Contains(code, "intent.parse"), strings.Contains(code, "intent.slots"):
		return FailureIntentParse
	default:
		return FailureUnknown
	}
}

func FailureFamilyForDetail(detail string) string {
	lower := strings.ToLower(detail)
	switch {
	case strings.Contains(lower, "openapi"), strings.Contains(lower, "api document"), strings.Contains(lower, "source"):
		return FailureMissingAPISource
	case strings.Contains(lower, "operation"):
		return FailureMissingOperation
	case strings.Contains(lower, "credential"):
		return FailureCredentialBindingGap
	case strings.Contains(lower, "runtime"):
		return FailureRuntimeProfileGap
	case strings.Contains(lower, "intent"):
		return FailureIntentParse
	default:
		return FailureBuildError
	}
}
