package browserauthor

import (
	"errors"
	"io"
)

// FailureDetails is bounded private evidence, separate from the stable event
// error code. Event deliberately excludes it from its public JSON projection.
type FailureDetails struct {
	WorkerDiagnostic string `json:"worker_diagnostic"`
	StreamPhase      string `json:"stream_phase"`
	StreamFailure    string `json:"stream_failure"`
}

func NoFailureDetails() FailureDetails {
	return FailureDetails{WorkerDiagnostic: "none", StreamPhase: "none", StreamFailure: "none"}
}

func (detail FailureDetails) Valid() bool {
	if reduceWorkerDiagnostic(detail.WorkerDiagnostic) != detail.WorkerDiagnostic {
		return false
	}
	switch detail.StreamPhase {
	case "none":
		return detail.StreamFailure == "none"
	case "receive":
		return detail.StreamFailure == "eof" || detail.StreamFailure == "decode" || detail.StreamFailure == "size" || detail.StreamFailure == "read"
	case "drain":
		return detail.StreamFailure == "decode" || detail.StreamFailure == "size" || detail.StreamFailure == "read" || detail.StreamFailure == "trailing_message"
	}
	return false
}

// Only producer-owned fixed classes cross this boundary. Even a syntactically
// valid code can contain private values, so a regular expression is insufficient.
func reduceWorkerDiagnostic(code string) string {
	switch code {
	case "none", "unknown", "malformed_message", "protocol_limit", "unexpected_eof",
		"protocol_mismatch", "approval_pending", "unknown_message", "invalid_state",
		"invalid_start", "clock_unavailable", "invalid_origin", "browser_failure",
		"unknown_context", "observation_limit", "invalid_observation", "invalid_context",
		"invalid_candidate", "human_input_mismatch", "challenge_kind_invalid", "invalid_action",
		"ambiguous_target", "approval_limit", "approval_mismatch", "approval_denied",
		"post_budget", "completion_denied", "output_selection_invalid", "teardown_failure",
		"result_invalid", "artifact_write", "canceled":
		return code
	}
	return "unknown"
}

var (
	errAuthorDecode  = errors.New("author stream decode failure")
	errAuthorSize    = errors.New("author stream size limit")
	errAuthorRead    = errors.New("author stream read failure")
	errAuthorTrailer = errors.New("author stream trailing message")
)

func authorStreamFailure(err error) string {
	switch {
	case errors.Is(err, io.EOF):
		return "eof"
	case errors.Is(err, errAuthorDecode):
		return "decode"
	case errors.Is(err, errAuthorSize):
		return "size"
	case errors.Is(err, errAuthorTrailer):
		return "trailing_message"
	default:
		return "read"
	}
}
