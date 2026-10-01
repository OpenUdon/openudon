package browserauthor

import (
	"github.com/OpenUdon/browsertools/registrationauthorsession"
	"github.com/OpenUdon/uws/browserregistration"
	"testing"
)

func TestV4RequiresExplicitObservedVerificationReview(t *testing.T) {
	v := &browserregistration.HumanVerification{Provider: "hcaptcha", Activation: "approved_submit", WidgetBinding: "single_in_submit_form", SubmissionURL: "https://app.example.test/register", Dependencies: browserregistration.VerificationDependencies{Policy: "hcaptcha.v1", MaxRequests: 256, MaxResponseBytes: 32 << 20, TimeoutMS: 120000}}
	observation := registrationauthorsession.Observation{Generation: 1, Origin: "https://app.example.test", Path: "/register", Candidates: []registrationauthorsession.Candidate{{ID: "candidate-0123456789abcdef", Role: "button", Label: "Register", Matches: 1, Verification: &registrationauthorsession.VerificationObservation{Provider: v.Provider, Activation: v.Activation, SubmissionURL: v.SubmissionURL, WidgetBinding: v.WidgetBinding, Coverage: "standard_single_widget"}}}}
	state := registrationRunState{started: true, protocol: registrationauthorsession.ProtocolV4, phase: "observing", origins: []string{observation.Origin}, bounds: expectedRegistrationBounds(nil), observation: &observation}
	if !safeRegistrationObservation(observation, state) {
		t.Fatal("v4 observation denied")
	}
	legacy := state
	legacy.protocol = registrationauthorsession.ProtocolV3
	if safeRegistrationObservation(observation, legacy) {
		t.Fatal("new metadata crossed legacy boundary")
	}
	command := RegistrationCommand{Type: "approve_verification", Confirmed: true, Verification: v, VerificationCandidateID: observation.Candidates[0].ID}
	message, _, err := prepareRegistrationCommand(command, state)
	if err != nil {
		t.Fatal(err)
	}
	if message.Verification == nil || message.CandidateID != command.VerificationCandidateID {
		t.Fatal("authority dropped")
	}
	unconfirmed := command
	unconfirmed.Confirmed = false
	if _, _, err := prepareRegistrationCommand(unconfirmed, state); err == nil {
		t.Fatal("unconfirmed review accepted")
	}
	event, done, err := applyRegistrationResponse(&state, command, registrationauthorsession.ServerMessage{Protocol: state.protocol, Type: "state", Phase: "observing"}, nil)
	if err != nil || done || event.VerificationAuthority == nil {
		t.Fatal("review response lost")
	}
	if _, _, err := prepareRegistrationCommand(command, state); err == nil {
		t.Fatal("review authority renewed")
	}
}
