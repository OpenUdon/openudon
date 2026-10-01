// Package browsercapture defines the supervising-product protocol above the
// existing Browsertools-backed controllers. It does not execute browser actions.
package browsercapture

import (
	"encoding/json"
	"errors"
	"regexp"
	"time"

	"github.com/OpenUdon/browsertools/authorsession"
	"github.com/OpenUdon/browsertools/registrationauthorsession"
	"github.com/OpenUdon/openudon/internal/browserauthor"
	"github.com/OpenUdon/openudon/internal/browsertransaction"
	"github.com/OpenUdon/openudon/internal/evidencefile"
	"github.com/OpenUdon/uws/browserregistration"
)

const (
	Version         = "openudon.browser-capture.v1"
	MaxMessageBytes = 256 << 10
	MaxEvents       = 4096
	Authenticated   = "authenticated"
	Registration    = "registration"
)

var sessionPattern = regexp.MustCompile(`^capture-[0-9a-f]{32}$`)

// Binding scopes every decision to an issued event in one live session.
type Binding struct {
	Version   string `json:"version"`
	SessionID string `json:"session_id"`
	Mode      string `json:"mode"`
	Revision  uint64 `json:"revision"`
	EventID   string `json:"event_id"`
}

// AuthenticatedView contains only reduced controller records. Worker results,
// attestations, private locators and diagnostic stderr are deliberately absent.
type AuthenticatedView struct {
	Observation *authorsession.Observation `json:"observation,omitempty"`
	Approval    *authorsession.Approval    `json:"approval,omitempty"`
	Checkpoint  *authorsession.Checkpoint  `json:"checkpoint,omitempty"`
}

// RegistrationView sends the current observation and latest preview only.
// The controller owns full history; consumers can accumulate these deltas in
// transient memory without repeatedly transmitting all prior observations.
type RegistrationView struct {
	Observation           *registrationauthorsession.Observation   `json:"observation,omitempty"`
	Bounds                *registrationauthorsession.Bounds        `json:"bounds,omitempty"`
	Preview               *registrationauthorsession.PreviewRecord `json:"preview,omitempty"`
	VerificationAuthority *browserregistration.HumanVerification   `json:"verification_authority,omitempty"`
}

type Result struct {
	ProfileID         string `json:"profile_id"`
	TransactionSHA256 string `json:"transaction_sha256"`
	Effect            string `json:"effect"`
}

type View struct {
	State          string             `json:"state"`
	Phase          string             `json:"phase,omitempty"`
	Authentication *AuthenticatedView `json:"authentication,omitempty"`
	Registration   *RegistrationView  `json:"registration,omitempty"`
	Diagnostic     string             `json:"diagnostic,omitempty"`
	Result         *Result            `json:"result,omitempty"`
}

// RegistrationCommand carries reviewed structural artifacts and symbolic
// bindings, never credential values. Profile is canonical public profile JSON;
// the existing registration controller remains its semantic validator.
type RegistrationCommand struct {
	Type               string                                    `json:"type"`
	Method             string                                    `json:"method,omitempty"`
	URL                string                                    `json:"url,omitempty"`
	Confirmed          bool                                      `json:"confirmed,omitempty"`
	Verification       *browserregistration.HumanVerification    `json:"verification,omitempty"`
	CandidateID        string                                    `json:"candidate_id,omitempty"`
	Preview            *registrationauthorsession.PreviewRequest `json:"preview,omitempty"`
	Profile            string                                    `json:"profile,omitempty"`
	CandidateIDs       []string                                  `json:"candidate_ids,omitempty"`
	StepCandidates     []string                                  `json:"step_candidates,omitempty"`
	Flow               string                                    `json:"flow,omitempty"`
	CleanupDisposition string                                    `json:"cleanup_disposition,omitempty"`
	CredentialBindings []browsertransaction.CredentialBinding    `json:"credential_bindings,omitempty"`
}

// Command is a closed union. No arbitrary data, credential, cookie, token,
// human input value, browser handle, or worker command can be submitted.
type Command struct {
	Authentication      *browserauthor.Response `json:"authentication,omitempty"`
	Registration        *RegistrationCommand    `json:"registration,omitempty"`
	DiscloseObservation *bool                   `json:"disclose_observation,omitempty"`
}

type Proposal struct {
	Binding
	Type    string  `json:"type"`
	Command Command `json:"command"`
}

// Decision approves or refuses the exact server-held proposal. It cannot
// replace arguments after the action has been rendered.
type Decision struct {
	Binding
	Type          string `json:"type"`
	ActionID      string `json:"action_id"`
	CommandSHA256 string `json:"command_sha256"`
	Approved      bool   `json:"approved"`
}

type Cancellation struct {
	Binding
	Type string `json:"type"`
}

type Action struct {
	ID            string  `json:"id"`
	CommandSHA256 string  `json:"command_sha256"`
	Command       Command `json:"command"`
}

type Event struct {
	Binding
	Type     string  `json:"type"`
	Deadline string  `json:"deadline"`
	View     View    `json:"view"`
	Action   *Action `json:"action,omitempty"`
}

// DecodeEvent validates observations as data, never as new execution authority.
func DecodeEvent(data []byte) (Event, error) {
	var value Event
	err := decode(data, &value)
	if err == nil && !validBinding(value.Binding) {
		err = errors.New("invalid capture event")
	}
	if err == nil {
		if _, parseErr := time.Parse(time.RFC3339Nano, value.Deadline); parseErr != nil {
			err = errors.New("invalid capture deadline")
		}
	}
	return value, err
}

func DecodeProposal(data []byte) (Proposal, error) {
	var value Proposal
	err := decode(data, &value)
	if err == nil && (value.Type != "propose" || !validBinding(value.Binding)) {
		err = errors.New("invalid capture proposal")
	}
	return value, err
}

func DecodeDecision(data []byte) (Decision, error) {
	var value Decision
	err := decode(data, &value)
	if err == nil && (value.Type != "decide" || !validBinding(value.Binding) || !evidencefile.ValidSHA256(value.CommandSHA256)) {
		err = errors.New("invalid capture decision")
	}
	return value, err
}

func DecodeCancellation(data []byte) (Cancellation, error) {
	var value Cancellation
	err := decode(data, &value)
	if err == nil && (value.Type != "cancel" || !validBinding(value.Binding)) {
		err = errors.New("invalid capture cancellation")
	}
	return value, err
}

func decode(data []byte, target any) error {
	if len(data) == 0 || len(data) > MaxMessageBytes {
		return errors.New("capture message exceeds bounds")
	}
	if err := evidencefile.DecodeStrict(data, target); err != nil {
		return errors.New("invalid capture JSON")
	}
	return validateSchema(data, target)
}

func validBinding(value Binding) bool {
	return value.Version == Version && sessionPattern.MatchString(value.SessionID) &&
		(value.Mode == Authenticated || value.Mode == Registration) && value.Revision > 0 &&
		value.Revision <= MaxEvents && referencePattern.MatchString(value.EventID)
}

func boundedJSON(value any) ([]byte, error) {
	data, err := json.Marshal(value)
	if err != nil || len(data) > MaxMessageBytes {
		return nil, errors.New("capture record exceeds bounds")
	}
	return data, nil
}

func validDeadline(deadline time.Time, now time.Time) bool {
	return !deadline.IsZero() && now.Before(deadline)
}
