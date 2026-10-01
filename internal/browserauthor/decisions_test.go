package browserauthor

import (
	"testing"

	"github.com/OpenUdon/browsertools/authorresult"
	"github.com/OpenUdon/browsertools/authorsession"
)

func TestSupervisingDecisionsUseOnlyCurrentOfferedFields(t *testing.T) {
	const candidate = "candidate-0123456789abcdef"
	observation := &authorsession.Observation{Context: "main", Candidates: []authorsession.Candidate{{ID: candidate, Role: "button", Matches: 1}}, Contexts: map[string]authorresult.Context{}}
	config := Config{DashboardURL: "https://members.example.test/dashboard"}
	tests := []struct {
		name     string
		event    Event
		response Response
		valid    bool
	}{
		{"current click", Event{State: "exploration", Observation: observation}, Response{Kind: "click", CandidateID: candidate, POSTBudget: 1}, true},
		{"invented candidate", Event{State: "exploration", Observation: observation}, Response{Kind: "click", CandidateID: "candidate-ffffffffffffffff"}, false},
		{"POST ceiling", Event{State: "exploration", Observation: observation}, Response{Kind: "click", CandidateID: candidate, POSTBudget: 33}, false},
		{"extraneous navigation", Event{State: "exploration", Observation: observation}, Response{Kind: "click", CandidateID: candidate, URL: "https://members.example.test/private"}, false},
		{"unknown context", Event{State: "exploration", Observation: observation}, Response{Kind: "observe", Context: "unknown"}, false},
		{"current observe", Event{State: "exploration", Observation: observation}, Response{Kind: "observe"}, true},
		{"new origin remains worker-gated", Event{State: "exploration", Observation: observation}, Response{Kind: "navigate_get", URL: "https://other.example.test/"}, true},
		{"credential URL", Event{State: "exploration", Observation: observation}, Response{Kind: "navigate_get", URL: "https://members.example.test/?token=PRIVATE_CANARY"}, false},
		{"current approval", Event{State: "action_approval", Approval: &authorsession.Approval{ID: "approval-0001"}}, Response{Kind: "approve", ApprovalID: "approval-0001"}, true},
		{"wrong approval", Event{State: "action_approval", Approval: &authorsession.Approval{ID: "approval-0001"}}, Response{Kind: "approve", ApprovalID: "approval-0002"}, false},
		{"approval mixed fields", Event{State: "action_approval", Approval: &authorsession.Approval{ID: "approval-0001"}}, Response{Kind: "approve", ApprovalID: "approval-0001", Confirmed: true}, false},
		{"TOTP", Event{State: "human_input", Checkpoint: &authorsession.Checkpoint{Kind: "mfa", CandidateID: candidate, ChallengeKinds: []string{"totp"}}}, Response{Kind: "continue", CandidateID: candidate, ChallengeKind: "totp"}, true},
		{"unoffered MFA", Event{State: "human_input", Checkpoint: &authorsession.Checkpoint{Kind: "mfa", CandidateID: candidate, ChallengeKinds: []string{"totp"}}}, Response{Kind: "continue", CandidateID: candidate, ChallengeKind: "sms_otp"}, false},
		{"credential acknowledgement", Event{State: "human_input", Checkpoint: &authorsession.Checkpoint{Kind: "credential", CandidateID: candidate}}, Response{Kind: "continue", CandidateID: candidate}, true},
		{"output current ID", Event{State: "completion_review", Observation: observation, Checkpoint: &authorsession.Checkpoint{Kind: "completion"}}, Response{Kind: "confirm", Confirmed: true, Outputs: []authorsession.OutputRequest{{CandidateID: candidate}}}, true},
		{"output invented ID", Event{State: "completion_review", Observation: observation, Checkpoint: &authorsession.Checkpoint{Kind: "completion"}}, Response{Kind: "confirm", Confirmed: true, Outputs: []authorsession.OutputRequest{{CandidateID: "candidate-ffffffffffffffff"}}}, false},
		{"no pending decision", Event{State: "launching"}, Response{Kind: "observe"}, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateResponse(config, test.event, test.response)
			if (err == nil) != test.valid {
				t.Fatalf("offered=%v, error=%v", test.valid, err)
			}
		})
	}
}
