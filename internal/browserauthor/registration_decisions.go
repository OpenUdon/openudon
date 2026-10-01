package browserauthor

import (
	"errors"
	"reflect"
	"sort"

	"github.com/OpenUdon/browsertools/registrationauthorsession"
)

// rememberDecisionState retains only an immutable snapshot of controller-owned
// state. History/preview records are append-only; prior elements are never
// mutated, so a value copy keeps its original length without retransmission.
func (s *RegistrationSession) rememberDecisionState(state registrationRunState, event RegistrationEvent) {
	copy := state
	copy.origins = append([]string(nil), state.origins...)
	if state.observation != nil {
		observation := cloneRegistrationObservation(*state.observation)
		copy.observation = &observation
	}
	if state.verification != nil {
		verification := *state.verification
		copy.verification = &verification
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.decisionState, s.decisionEventState, s.decisionGeneration = &copy, event.State, 0
	if event.Observation != nil {
		s.decisionGeneration = event.Observation.Generation
	}
}

// ValidateDecision checks a supervising proposal against the same current
// controller state used on actual dispatch. It never sends a worker command,
// changes a review, or grants registration/verification traffic authority.
func (s *RegistrationSession) ValidateDecision(expectedState string, expectedGeneration int, command RegistrationCommand) error {
	if s == nil {
		return errors.New("registration decision unavailable")
	}
	s.mu.Lock()
	if s.closed || s.decisionState == nil || s.decisionEventState != expectedState || s.decisionGeneration != expectedGeneration {
		s.mu.Unlock()
		return errors.New("registration decision state changed")
	}
	state := *s.decisionState
	s.mu.Unlock()
	if !registrationDecisionShape(command) {
		return errors.New("registration decision shape invalid")
	}
	if command.Type == "navigate" {
		_, origin, _, err := registrationauthorsession.ValidateNavigationURL(state.protocol, command.URL)
		if err != nil || (command.Method != "GET" && command.Method != "HEAD") || !containsRegistrationString(state.origins, origin) {
			return errors.New("registration navigation outside approved authority")
		}
	}
	_, _, err := prepareRegistrationCommand(command, state)
	if err != nil {
		return errors.New("registration decision not offered")
	}
	return nil
}

// NormalizeRegistrationStart fixes the reviewed no-submit initial authority.
// Native URL/bounds validators remain authoritative; unsupported credentials
// and fields are rejected before starting a worker.
func NormalizeRegistrationStart(protocol string, command RegistrationCommand) (RegistrationCommand, error) {
	if command.Type != "start" || !registrationDecisionShape(command) || !profileIDPattern.MatchString(command.ProfileID) || len(command.Origins) == 0 || len(command.Origins) > 32 {
		return RegistrationCommand{}, errors.New("registration start invalid")
	}
	url, initialOrigin, _, err := registrationauthorsession.ValidateNavigationURL(protocol, command.URL)
	if err != nil {
		return RegistrationCommand{}, errors.New("registration initial navigation invalid")
	}
	command = cloneRegistrationCommand(command)
	command.URL = url
	for _, origin := range command.Origins {
		canonical, err := cleanOrigin(origin)
		if err != nil || canonical != origin {
			return RegistrationCommand{}, errors.New("registration origins invalid")
		}
	}
	sort.Strings(command.Origins)
	for i, origin := range command.Origins {
		if i > 0 && origin == command.Origins[i-1] {
			return RegistrationCommand{}, errors.New("registration origins duplicated")
		}
	}
	if !containsRegistrationString(command.Origins, initialOrigin) {
		return RegistrationCommand{}, errors.New("registration initial origin not approved")
	}
	if _, _, err := prepareRegistrationCommand(command, registrationRunState{protocol: protocol}); err != nil {
		return RegistrationCommand{}, errors.New("registration initial bounds invalid")
	}
	return command, nil
}

func registrationDecisionShape(command RegistrationCommand) bool {
	allowed := RegistrationCommand{Type: command.Type}
	switch command.Type {
	case "start":
		allowed.ProfileID, allowed.URL, allowed.Origins, allowed.Bounds = command.ProfileID, command.URL, command.Origins, command.Bounds
	case "approve_verification":
		allowed.Confirmed, allowed.Verification, allowed.VerificationCandidateID = command.Confirmed, command.Verification, command.VerificationCandidateID
	case "preview":
		allowed.Confirmed, allowed.Preview = command.Confirmed, command.Preview
	case "navigate":
		allowed.Method, allowed.URL = command.Method, command.URL
	case "review":
		allowed.Confirmed, allowed.Profile, allowed.CandidateIDs, allowed.StepCandidates = command.Confirmed, command.Profile, command.CandidateIDs, command.StepCandidates
		allowed.Flow, allowed.CleanupDisposition, allowed.CredentialBindings = command.Flow, command.CleanupDisposition, command.CredentialBindings
	case "finish":
		allowed.Confirmed = command.Confirmed
	case "observe", "close":
	default:
		return false
	}
	return reflect.DeepEqual(command, allowed)
}
