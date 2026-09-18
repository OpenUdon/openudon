package ui

import (
	"encoding/json"
	"errors"
	"os"
	"regexp"

	"github.com/OpenUdon/browsertools/authordiagnostic"
	"github.com/OpenUdon/openudon/internal/evidencefile"
	"github.com/OpenUdon/openudon/internal/icot/browserauthor"
)

// CaptureDiagnosticConfig is an opt-in private file, never an application frame.
// One application instance may consume at most one diagnostic capture.
type CaptureDiagnosticConfig struct{ Path, Attempt, Binding string }

type captureDiagnostic struct {
	Version       string                       `json:"version"`
	Attempt       string                       `json:"attempt_id"`
	Binding       string                       `json:"packet_sha256"`
	State         string                       `json:"state"`
	Code          string                       `json:"code"`
	Failure       browserauthor.FailureDetails `json:"failure"`
	BackendStatus string                       `json:"backend_status"`
	Backend       authordiagnostic.Class       `json:"backend"`
	EventsClosed  bool                         `json:"events_closed"`
}

type captureDiagnosticSink struct {
	config CaptureDiagnosticConfig
	file   *os.File
	used   bool
}

func newCaptureDiagnostic(config *CaptureDiagnosticConfig) (*captureDiagnosticSink, error) {
	if config == nil {
		return nil, nil
	}
	if !regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,79}$`).MatchString(config.Attempt) || !evidencefile.ValidSHA256(config.Binding) {
		return nil, errors.New("invalid capture diagnostic binding")
	}
	f, err := authordiagnostic.Reserve(config.Path)
	if err != nil {
		return nil, err
	}
	return &captureDiagnosticSink{config: *config, file: f}, nil
}

func captureDiagnosticCode(code string) string {
	switch code {
	case "":
		return "none"
	case "worker_start", "worker_teardown", "absolute_timeout", "operator_idle_timeout", "worker_protocol", "worker_exit", "protocol_negotiation", "attestation", "worker_write", "protocol_mismatch", "protocol_state", "malformed_state", "malformed_observation", "invalid_response", "malformed_approval", "malformed_checkpoint", "malformed_result", "malformed_diagnostic", "worker_failed", "unexpected_message", "missing_terminal":
		return code
	}
	return "unknown"
}

// Called only with the server lock, after joined event delivery (or failure to
// create a session). No raw error, event, observation or config is serialized.
func (s *Server) finishCaptureDiagnostic(session CaptureSession, state, code string, detail *browserauthor.FailureDetails) {
	sink := s.captureDiagnostic
	if sink == nil || sink.file == nil {
		return
	}
	failure := browserauthor.NoFailureDetails()
	if detail != nil && detail.Valid() {
		failure = *detail
	}
	status, backend := "disabled", authordiagnostic.Class{Stage: "none", Reason: "none"}
	if session != nil {
		status, backend = "missing", authordiagnostic.Class{Stage: "unknown", Reason: "unknown"}
		if source, ok := session.(interface {
			BackendDiagnostic() (string, authordiagnostic.Class)
		}); ok {
			status, backend = source.BackendDiagnostic()
			if !backend.Valid() || (status != "available" && status != "missing" && status != "invalid" && status != "disabled") {
				status, backend = "invalid", authordiagnostic.Class{Stage: "unknown", Reason: "unknown"}
			}
		}
	}
	switch state {
	case "failed", "canceled", "stage_review":
	default:
		state = "failed"
	}
	record := captureDiagnostic{"openudon.capture-diagnostic.v1", sink.config.Attempt, sink.config.Binding, state, captureDiagnosticCode(code), failure, status, backend, session != nil}
	err := json.NewEncoder(sink.file).Encode(record)
	closeErr := sink.file.Close()
	sink.file = nil
	if err != nil || closeErr != nil {
		// Incomplete evidence cannot be mistaken for valid diagnostics. Block a
		// later capture; the caller must independently check the private file.
		s.captureContainmentFailed = true

		s.captureResult = nil
		s.captureAttestation = nil
		if s.capture != nil {
			s.capture.State = "failed"
			s.capture.ResultReady = false
			s.capture.ContainmentFailed = true
			s.capture.Message = "Private capture diagnostics could not be retained; restart before another capture."
		}
		_ = s.updateRevisionLocked()
	}
}
