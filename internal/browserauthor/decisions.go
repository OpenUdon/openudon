package browserauthor

import (
	"errors"
	"reflect"
)

// NormalizeConfig reuses the controller's authority and deadline validation
// before a supervising transport allocates its session or starts a worker.
func NormalizeConfig(config Config) (Config, error) { return normalizeConfig(config) }

// ValidateResponse is a pure check of a proposed human decision against one
// current controller event. Dispatch retains the controller and worker checks;
// validation neither responds to a session nor changes its attestation.
func ValidateResponse(config Config, event Event, response Response) error {
	invalid := errors.New("decision is not offered by current browser state")
	allowed := Response{Kind: response.Kind}
	switch {
	case event.Checkpoint != nil:
		if event.State != "human_input" && event.State != "completion_review" {
			return invalid
		}
		if _, err := checkpointResponse(response, *event.Checkpoint); err != nil {
			return invalid
		}
		switch event.Checkpoint.Kind {
		case "credential":
			allowed.CandidateID = response.CandidateID
		case "mfa":
			allowed.CandidateID, allowed.ChallengeKind = response.CandidateID, response.ChallengeKind
		case "completion":
			allowed.Confirmed, allowed.Outputs = response.Confirmed, response.Outputs
			// Candidate identity is controller-owned. Profile/output semantics
			// remain in the existing worker and independent import validator.
			for _, output := range response.Outputs {
				known := false
				if event.Observation != nil {
					for _, candidate := range event.Observation.Candidates {
						known = known || candidate.ID == output.CandidateID
					}
				}
				if !known {
					return invalid
				}
			}
		}
	case event.Approval != nil:
		if event.State != "action_approval" || (response.Kind != "approve" && response.Kind != "deny") || response.ApprovalID != event.Approval.ID {
			return invalid
		}
		allowed.ApprovalID = response.ApprovalID
	case event.Observation != nil:
		if event.State != "exploration" {
			return invalid
		}
		if _, err := observationResponse(response, *event.Observation, config); err != nil {
			return invalid
		}
		switch response.Kind {
		case "focus_human_input":
			allowed.CandidateID = response.CandidateID
		case "click":
			allowed.CandidateID, allowed.POSTBudget = response.CandidateID, response.POSTBudget
			if response.POSTBudget > 32 { // Existing Browsertools author-session ceiling.
				return invalid
			}
		case "navigate_get":
			allowed.URL, allowed.Context = response.URL, response.Context
		case "observe":
			allowed.Context = response.Context
		}
		if response.Context != "" && response.Context != "main" {
			if _, ok := event.Observation.Contexts[response.Context]; !ok {
				return invalid
			}
		}
	default:
		return invalid
	}
	if !reflect.DeepEqual(response, allowed) {
		return invalid
	}
	return nil
}
