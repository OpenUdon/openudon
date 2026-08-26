package udonreport

import (
	"encoding/json"
	"testing"
)

func TestDecodeRequiresTypedV3FailureAndNoSuccessCode(t *testing.T) {
	base := Report{Version: Version, Status: "success", StartedAt: "2026-08-21T00:00:00Z", FinishedAt: "2026-08-21T00:00:01Z", WorkflowPath: "workflow.json", WorkflowFormat: "uws-json", WorkDir: "."}
	if data, _ := json.Marshal(base); mustDecode(t, data) == nil {
		t.Fatal("valid success was rejected")
	}
	for _, code := range []string{"approval_required", "invalid_parameters", "mfa_timeout", "mfa_denied", "credentials_invalid", "session_expired", "driver_error", "unsupported_challenge", "captcha_required", "origin_rejected", "ambiguous_locator", "invalid_context", "invalid_response", "secret_output", CodeUnclassified} {
		failure := base
		failure.Status, failure.ErrorCode, failure.ErrorSummary = "error", code, "redacted failure"
		data, _ := json.Marshal(failure)
		if got := FailureCode(data); got != code {
			t.Fatalf("FailureCode(%s) = %q", code, got)
		}
	}
	for _, code := range []string{"registration_indeterminate", "registration_retry_forbidden", "registration_checkpoint_timeout", "registration_checkpoint_denied"} {
		failure := base
		failure.Status, failure.ErrorCode, failure.ErrorSummary = "error", code, "redacted registration failure"
		data, _ := json.Marshal(failure)
		if got := FailureCode(data); got != code {
			t.Fatalf("FailureCode(%s) = %q", code, got)
		}
		failure.Version = VersionV2
		legacyData, _ := json.Marshal(failure)
		if _, err := Decode(legacyData); err == nil {
			t.Fatalf("legacy v2 accepted registration code %s", code)
		}
	}
	legacy := base
	legacy.Version = VersionV2
	legacyData, _ := json.Marshal(legacy)
	if _, err := Decode(legacyData); err != nil {
		t.Fatalf("legacy v2 success rejected: %v", err)
	}
	for _, mutate := range []func(*Report){
		func(report *Report) { report.Version = "udon.execution-report.v1" },
		func(report *Report) { report.Status, report.ErrorSummary = "error", "redacted" },
		func(report *Report) {
			report.Status, report.ErrorCode, report.ErrorSummary = "error", "invented", "redacted"
		},
		func(report *Report) { report.ErrorCode = "driver_error" },
	} {
		candidate := base
		mutate(&candidate)
		data, _ := json.Marshal(candidate)
		if _, err := Decode(data); err == nil || FailureCode(data) != CodeUnclassified {
			t.Fatalf("invalid report was classified: %s", data)
		}
	}
}

func mustDecode(t *testing.T, data []byte) *Report {
	t.Helper()
	report, err := Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	return report
}
