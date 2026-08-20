package icotreport

import "testing"

func TestFailureFamilyClassifiers(t *testing.T) {
	readiness := map[string]string{
		"missing_api_doc": FailureMissingAPISource, "missing_operation": FailureMissingOperation,
		"missing_required_request_values": FailureBadRequestMapping, "conflicting_mapping": FailureBadRequestMapping,
		"missing_credential_bindings": FailureCredentialBindingGap, "inline_secret_value": FailureCredentialBindingGap,
		"missing_side_effect_policy": FailureSideEffectPolicyGap, "unsafe_review_bypass": FailureSideEffectPolicyGap,
		"missing_goal": FailureAmbiguousUserIntent, "conflicting_decision_evidence": FailureAmbiguousUserIntent,
		"intent_render_invalid": FailureIntentParse, "unrecognized": FailureUnknown,
	}
	for input, want := range readiness {
		if got := FailureFamilyForReadiness(input); got != want {
			t.Errorf("FailureFamilyForReadiness(%q) = %q, want %q", input, got, want)
		}
	}

	quality := map[string]string{
		"openapi_refs.required": FailureMissingAPISource, "openapi_operations.missing": FailureMissingOperation,
		"required_params.missing": FailureBadRequestMapping, "response_paths.invalid": FailureBadResponsePath,
		"credential.binding": FailureCredentialBindingGap, "side_effect.review": FailureSideEffectPolicyGap,
		"runtime.profile": FailureRuntimeProfileGap, "intent.parse": FailureIntentParse,
		"unrecognized": FailureUnknown, "": "",
	}
	for input, want := range quality {
		if got := FailureFamilyForQualityCode(input); got != want {
			t.Errorf("FailureFamilyForQualityCode(%q) = %q, want %q", input, got, want)
		}
	}

	details := map[string]string{
		"OpenAPI source unavailable": FailureMissingAPISource, "operation missing": FailureMissingOperation,
		"credential needed": FailureCredentialBindingGap, "runtime profile missing": FailureRuntimeProfileGap,
		"intent cannot parse": FailureIntentParse, "compiler stopped": FailureBuildError,
	}
	for input, want := range details {
		if got := FailureFamilyForDetail(input); got != want {
			t.Errorf("FailureFamilyForDetail(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestAllFailureFamiliesAreValid(t *testing.T) {
	for _, family := range []string{
		FailureMissingAPISource, FailureMissingOperation, FailureBadRequestMapping,
		FailureBadResponsePath, FailureCredentialBindingGap, FailureSideEffectPolicyGap,
		FailureAmbiguousUserIntent, FailureRuntimeProfileGap, FailureIntentParse,
		FailureBuildError, FailureUnknown,
	} {
		if !IsValidFailureFamily(family) {
			t.Errorf("failure family %q is not valid", family)
		}
	}
	if IsValidFailureFamily("") || IsValidFailureFamily("future_family") {
		t.Fatal("unknown failure family was accepted")
	}
}
