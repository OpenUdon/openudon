package icot

import (
	"context"
	"errors"

	"github.com/OpenUdon/openudon/internal/icot/browserauthor"
)

// BrowserScenarioAuthorDiagnostic contains a closed operation phase and failure
// code. It never includes an error string, path, observed label, or input value.
type BrowserScenarioAuthorDiagnostic struct {
	Phase string `json:"phase"`
	Code  string `json:"code"`
}

func (diagnostic BrowserScenarioAuthorDiagnostic) Valid() bool {
	return validScenarioAuthorPhase(diagnostic.Phase) && validScenarioAuthorCode(diagnostic.Code)
}

type browserScenarioAuthorFailure struct {
	diagnostic BrowserScenarioAuthorDiagnostic
	cause      error
}

func (failure *browserScenarioAuthorFailure) Error() string {
	return "browser scenario author failed (" + failure.diagnostic.Phase + ": " + failure.diagnostic.Code + ")"
}
func (failure *browserScenarioAuthorFailure) Unwrap() error { return failure.cause }

// BrowserScenarioFailureDiagnostic extracts only validated closed metadata.
func BrowserScenarioFailureDiagnostic(err error) (BrowserScenarioAuthorDiagnostic, bool) {
	var failure *browserScenarioAuthorFailure
	if !errors.As(err, &failure) || failure == nil || !failure.diagnostic.Valid() {
		return BrowserScenarioAuthorDiagnostic{}, false
	}
	return failure.diagnostic, true
}

func scenarioAuthorStageError(phase string, err error) error {
	if err == nil {
		return nil
	}
	if _, ok := BrowserScenarioFailureDiagnostic(err); ok {
		return err
	}
	code := "operation_failed"
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		code = "absolute_timeout"
	case errors.Is(err, context.Canceled):
		code = "operation_canceled"
	case err.Error() == "required reduced scenario candidate is missing or ambiguous":
		code = "candidate_binding"
	case err.Error() == "output selection bound exceeded":
		code = "output_bound"
	}
	if !validScenarioAuthorPhase(phase) {
		phase = "unknown"
	}
	return &browserScenarioAuthorFailure{BrowserScenarioAuthorDiagnostic{phase, code}, err}
}

func scenarioControllerFailure(phase string, event browserauthor.Event, cause error) error {
	code := event.ErrorCode
	if code == "" {
		switch event.State {
		case "canceled":
			code = "operation_canceled"
		case "closed":
			code = "worker_closed"
		default:
			code = "worker_failed"
		}
	}
	if !validScenarioAuthorCode(code) {
		code = "unclassified_worker_failure"
	}
	if !validScenarioAuthorPhase(phase) {
		phase = "unknown"
	}
	return &browserScenarioAuthorFailure{BrowserScenarioAuthorDiagnostic{phase, code}, cause}
}

func validScenarioAuthorPhase(phase string) bool {
	switch phase {
	case "request", "configuration", "worker_start", "controller", "output_selection", "result_tamper", "assessment", "result_import", "candidate_observer", "package_stage", "result_read", "result_decode", "profile_summary", "worker_cleanup", "unknown":
		return true
	}
	return false
}

func validScenarioAuthorCode(code string) bool {
	switch code {
	case "operation_failed", "operation_canceled", "candidate_binding", "output_bound", "unclassified_worker_failure", "worker_closed",
		"absolute_timeout", "attestation", "invalid_response", "malformed_approval", "malformed_checkpoint", "malformed_diagnostic", "malformed_observation", "malformed_result", "malformed_state", "protocol_mismatch", "protocol_negotiation", "protocol_state", "unexpected_message", "worker_failed", "worker_protocol", "worker_write", "worker_exit", "worker_teardown", "operator_idle_timeout":
		return true
	}
	return false
}
