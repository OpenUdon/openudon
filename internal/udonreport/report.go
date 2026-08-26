// Package udonreport owns strict decoding of the public Udon execution report
// consumed by OpenUdon evidence and browser scenario evaluators.
package udonreport

import (
	"fmt"
	"strings"
	"time"

	"github.com/OpenUdon/openudon/internal/evidencefile"
)

const (
	VersionV2        = "udon.execution-report.v2"
	VersionV3        = "udon.execution-report.v3"
	Version          = VersionV3
	CodeUnclassified = "unclassified"
)

var failureCodes = map[string]bool{
	"approval_required": true, "invalid_parameters": true,
	"mfa_timeout": true, "mfa_denied": true, "credentials_invalid": true, "session_expired": true,
	"driver_error": true, "unsupported_challenge": true, "captcha_required": true,
	"origin_rejected": true, "ambiguous_locator": true, "invalid_context": true, "invalid_response": true,
	"secret_output": true, CodeUnclassified: true,
}

var registrationFailureCodes = map[string]bool{
	"registration_indeterminate": true, "registration_retry_forbidden": true,
	"registration_checkpoint_timeout": true, "registration_checkpoint_denied": true,
}

type Report struct {
	Version        string `json:"version"`
	Status         string `json:"status"`
	StartedAt      string `json:"started_at"`
	FinishedAt     string `json:"finished_at"`
	WorkflowPath   string `json:"workflow_path"`
	WorkflowFormat string `json:"workflow_format"`
	WorkDir        string `json:"workdir"`
	OutputPath     string `json:"output_path,omitempty"`
	OutputDigest   string `json:"output_digest,omitempty"`
	ErrorCode      string `json:"error_code,omitempty"`
	ErrorSummary   string `json:"error_summary,omitempty"`
}

func ValidFailureCode(code string) bool {
	code = strings.TrimSpace(code)
	return failureCodes[code] || registrationFailureCodes[code]
}

func Decode(data []byte) (*Report, error) {
	var report Report
	if err := evidencefile.DecodeStrict(data, &report); err != nil {
		return nil, fmt.Errorf("must be valid JSON: %w", err)
	}
	if report.Version != VersionV2 && report.Version != VersionV3 {
		return nil, fmt.Errorf("version must be %s or read-only legacy %s", VersionV3, VersionV2)
	}
	if report.Status != "success" && report.Status != "error" {
		return nil, fmt.Errorf("status must be success or error")
	}
	started, err := time.Parse(time.RFC3339, strings.TrimSpace(report.StartedAt))
	if err != nil {
		return nil, fmt.Errorf("started_at must be RFC3339: %w", err)
	}
	finished, err := time.Parse(time.RFC3339, strings.TrimSpace(report.FinishedAt))
	if err != nil {
		return nil, fmt.Errorf("finished_at must be RFC3339: %w", err)
	}
	if finished.Before(started) {
		return nil, fmt.Errorf("finished_at precedes started_at")
	}
	for _, field := range []struct{ name, value string }{
		{name: "workflow_path", value: report.WorkflowPath}, {name: "workflow_format", value: report.WorkflowFormat}, {name: "workdir", value: report.WorkDir},
	} {
		if strings.TrimSpace(field.value) == "" {
			return nil, fmt.Errorf("%s is required", field.name)
		}
	}
	if report.OutputDigest != "" {
		algorithm, value, ok := strings.Cut(report.OutputDigest, ":")
		if !ok || algorithm != "sha256" || value != strings.ToLower(value) || !evidencefile.ValidSHA256(value) {
			return nil, fmt.Errorf("output_digest must be sha256 followed by 64 lowercase hexadecimal characters")
		}
	}
	if report.Status == "success" {
		if report.ErrorCode != "" || report.ErrorSummary != "" {
			return nil, fmt.Errorf("successful report must not contain error_code or error_summary")
		}
	} else {
		if !failureCodes[report.ErrorCode] && (report.Version != VersionV3 || !registrationFailureCodes[report.ErrorCode]) {
			return nil, fmt.Errorf("failed report requires a closed error_code")
		}
		if strings.TrimSpace(report.ErrorSummary) == "" {
			return nil, fmt.Errorf("failed report requires error_summary")
		}
	}
	return &report, nil
}

// FailureCode returns a typed failed-report code. Missing, malformed,
// successful, unknown, or unrelated reports are deliberately unclassified.
func FailureCode(data []byte) string {
	report, err := Decode(data)
	if err != nil || report.Status != "error" || !ValidFailureCode(report.ErrorCode) {
		return CodeUnclassified
	}
	return report.ErrorCode
}
