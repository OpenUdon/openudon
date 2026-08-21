package elicitor

import (
	"fmt"
	"strings"
)

const (
	JourneyAPI                        = "api"
	JourneyExistingAccountSignIn      = "existing_account_sign_in"
	JourneyAuthenticatedAction        = "authenticated_action"
	JourneyExistingReviewedCapability = "existing_reviewed_capability"
	JourneyFreeformMixed              = "freeform_mixed"
	journeyStarterMetadataKey         = "journey_starter"
	journeyGoalMetadataKey            = "journey_goal"
)

var journeyStarters = map[string]bool{
	JourneyAPI:                        true,
	JourneyExistingAccountSignIn:      true,
	JourneyAuthenticatedAction:        true,
	JourneyExistingReviewedCapability: true,
	JourneyFreeformMixed:              true,
}

// JourneySelection is human decision evidence that routes the acquisition
// shell without changing the v2 authoring-session wire contract.
type JourneySelection struct {
	Starter string `json:"starter,omitempty"`
	Goal    string `json:"goal,omitempty"`
}

// Journey returns the persisted selection, if any. Older sessions without a
// starter remain valid and return the zero value.
func (s Session) Journey() JourneySelection {
	return JourneySelection{
		Starter: strings.TrimSpace(s.Interview.Metadata[journeyStarterMetadataKey]),
		Goal:    strings.TrimSpace(s.Interview.Metadata[journeyGoalMetadataKey]),
	}
}

// SelectJourney records an explicit human starter and goal.
func SelectJourney(session *Session, starter, goal string) error {
	if session == nil {
		return fmt.Errorf("authoring session is required")
	}
	starter = strings.ToLower(strings.TrimSpace(starter))
	goal = strings.TrimSpace(goal)
	if !journeyStarters[starter] {
		return fmt.Errorf("journey starter is invalid")
	}
	if goal == "" || len(goal) > 1024 || strings.ContainsAny(goal, "\r\n\x00") {
		return fmt.Errorf("journey goal must contain 1 through 1024 safe characters")
	}
	if session.Interview.Metadata == nil {
		session.Interview.Metadata = map[string]string{}
	}
	session.Interview.Metadata[journeyStarterMetadataKey] = starter
	session.Interview.Metadata[journeyGoalMetadataKey] = goal
	addDecisionEvidence(session, DecisionEvidence{
		Stage: "journey_selection", Slot: "journey.starter", Value: starter,
		Source: mappingSourceUser, Confidence: "high", Reason: "selected by the operator in the iCoT authoring shell",
	})
	addDecisionEvidence(session, DecisionEvidence{
		Stage: "journey_selection", Slot: "journey.goal", Value: goal,
		Source: mappingSourceUser, Confidence: "high", Reason: "entered by the operator in the iCoT authoring shell",
	})
	return nil
}
