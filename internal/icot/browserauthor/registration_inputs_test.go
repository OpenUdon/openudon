package browserauthor

import (
	"testing"

	"github.com/OpenUdon/browsertools/registrationauthorsession"
)

func TestV3ObservationsCopyNestedDefinitionsAndPermitObservationOnlyCandidates(t *testing.T) {
	yes := true
	observation := registrationauthorsession.Observation{Generation: 1, Origin: "https://app.example.test", Path: "/register", Candidates: []registrationauthorsession.Candidate{{ID: "candidate-0123456789abcdef", Role: "combobox", Label: "Account kind", Matches: 1, Control: &registrationauthorsession.ControlMetadata{Kind: "select", Required: &yes, Options: []registrationauthorsession.PublicOption{{Value: "member", Label: "Member"}}}}}}
	state := registrationRunState{started: true, protocol: registrationauthorsession.ProtocolV3, origins: []string{observation.Origin}, bounds: expectedRegistrationBounds(nil)}
	if !safeRegistrationObservation(observation, state) {
		t.Fatal("valid definition refused")
	}
	copy := cloneRegistrationObservation(observation)
	observation.Candidates[0].Control.Options[0].Value = "changed"
	*observation.Candidates[0].Control.Required = false
	if copy.Candidates[0].Control.Options[0].Value != "member" || !*copy.Candidates[0].Control.Required {
		t.Fatal("retained definition aliases the response")
	}
	copy.Candidates[0].Control = nil
	if !safeRegistrationObservation(copy, state) {
		t.Fatal("observation-only candidate refused")
	}
	state.protocol = registrationauthorsession.ProtocolV2
	if safeRegistrationObservation(observation, state) {
		t.Fatal("new metadata crossed legacy protocol")
	}
	choice := "member"
	preview := registrationauthorsession.PreviewRecord{Request: registrationauthorsession.PreviewRequest{Option: &choice}, NextGeneration: 2}
	previews := cloneRegistrationPreviews([]registrationauthorsession.PreviewRecord{preview})
	choice = "changed"
	if *previews[0].Request.Option != "member" {
		t.Fatal("reviewed preview aliases caller memory")
	}
}
