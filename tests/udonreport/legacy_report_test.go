package udonreport_test

import (
	"encoding/json"
	"testing"

	"github.com/OpenUdon/openudon/udonreport"
)

func TestDecodeRequiresTypedV3FailureAndNoSuccessCode(t *testing.T) {
	base := udonreport.Report{Version: udonreport.Version, Status: "success", StartedAt: "2026-08-21T00:00:00Z", FinishedAt: "2026-08-21T00:00:01Z", WorkflowPath: "workflow.json", WorkflowFormat: "uws-json", WorkDir: "."}
	if data, _ := json.Marshal(base); mustDecode(t, data) == nil {
		t.Fatal("valid success was rejected")
	}
	for _, code := range []string{"approval_required", "invalid_parameters", "mfa_timeout", "mfa_denied", "credentials_invalid", "session_expired", "driver_error", "unsupported_challenge", "captcha_required", "origin_rejected", "ambiguous_locator", "invalid_context", "invalid_response", "secret_output", udonreport.CodeUnclassified} {
		failure := base
		failure.Status, failure.ErrorCode, failure.ErrorSummary = "error", code, "redacted failure"
		data, _ := json.Marshal(failure)
		if got := udonreport.FailureCode(data); got != code {
			t.Fatalf("udonreport.FailureCode(%s) = %q", code, got)
		}
	}
	for _, code := range []string{"registration_indeterminate", "registration_retry_forbidden", "registration_checkpoint_timeout", "registration_checkpoint_denied"} {
		failure := base
		failure.Status, failure.ErrorCode, failure.ErrorSummary = "error", code, "redacted registration failure"
		data, _ := json.Marshal(failure)
		if got := udonreport.FailureCode(data); got != code {
			t.Fatalf("udonreport.FailureCode(%s) = %q", code, got)
		}
		failure.Version = udonreport.VersionV2
		legacyData, _ := json.Marshal(failure)
		if _, err := udonreport.Decode(legacyData); err == nil {
			t.Fatalf("legacy v2 accepted registration code %s", code)
		}
	}
	legacy := base
	legacy.Version = udonreport.VersionV2
	legacyData, _ := json.Marshal(legacy)
	if _, err := udonreport.Decode(legacyData); err != nil {
		t.Fatalf("legacy v2 success rejected: %v", err)
	}
	for _, mutate := range []func(*udonreport.Report){
		func(report *udonreport.Report) { report.Version = "udon.execution-report.v1" },
		func(report *udonreport.Report) { report.Status, report.ErrorSummary = "error", "redacted" },
		func(report *udonreport.Report) {
			report.Status, report.ErrorCode, report.ErrorSummary = "error", "invented", "redacted"
		},
		func(report *udonreport.Report) { report.ErrorCode = "driver_error" },
	} {
		candidate := base
		mutate(&candidate)
		data, _ := json.Marshal(candidate)
		if _, err := udonreport.Decode(data); err == nil || udonreport.FailureCode(data) != udonreport.CodeUnclassified {
			t.Fatalf("invalid report was classified: %s", data)
		}
	}
}

func mustDecode(t *testing.T, data []byte) *udonreport.Report {
	t.Helper()
	report, err := udonreport.Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	return report
}
