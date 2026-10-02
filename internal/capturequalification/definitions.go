package capturequalification

import (
	"github.com/OpenUdon/browsertools/registrationauthorsession"
	"github.com/OpenUdon/openudon/internal/registrationdraft"
	"github.com/OpenUdon/uws/browserregistration"
)

func candidateID(o registrationauthorsession.Observation, label string) string {
	id := ""
	for _, c := range o.Candidates {
		if c.Label == label && c.Matches == 1 {
			if id != "" {
				return ""
			}
			id = c.ID
		}
	}
	return id
}
func typedDefinition(initialURL, origin string, first, second, third registrationauthorsession.Observation) registrationdraft.Request {
	yes, no := true, false
	zero, ten, one := float64(0), float64(10), float64(1)
	fields := map[string]browserregistration.InputSlot{
		"contact_name": {Type: "string", Label: "Contact name", Required: &yes},
		"account_kind": {Type: "string", Label: "Account kind", Required: &yes, Enum: []any{"individual", "business"}},
		"company":      {Type: "string", Label: "Company name", RequiredWhen: &browserregistration.InputCondition{Slot: "account_kind", Equals: "business"}},
		"phone":        {Type: "string", Label: "Phone", Required: &no}, "updates": {Type: "boolean", Label: "Product updates", Required: &no},
		"quantity": {Type: "integer", Label: "Quantity", Required: &yes, Minimum: &zero, Maximum: &ten}, "ratio": {Type: "number", Label: "Ratio", Required: &yes, Minimum: &zero, Maximum: &one},
	}
	draft := registrationdraft.Request{Title: "Synthetic typed registration", Provider: "Synthetic loopback", Confidence: "high", ExpiresAfter: "P30D", InputsReviewed: true, InputSlots: fields,
		CredentialSlots: []registrationdraft.Slot{{Slot: "identifier", Kind: "identifier", Binding: "registration_identifier"}, {Slot: "password", Kind: "password", Binding: "reg_password"}},
		Flow: registrationdraft.Flow{Name: "create_dedicated_test_user", Description: "Create one synthetic member through reviewed typed checkpoints.", ConfirmationPrompt: "Approve one synthetic registration.", Effects: []string{"creates_account", "requires_human_verification", "sends_verification"},
			Steps: []registrationdraft.Step{
				{Type: "input_checkpoint", CheckpointID: "identity", Slots: []string{"identifier", "password", "contact_name", "account_kind", "company"}},
				{Type: "navigate", Navigate: initialURL},
				{Type: "type_credential", Slot: "identifier", CandidateID: candidateID(first, "Email")}, {Type: "type_credential", Slot: "password", CandidateID: candidateID(first, "Password")},
				{Type: "fill_input", Slot: "contact_name", Control: "fill", CandidateID: candidateID(first, "Contact name")},
				{Type: "fill_input", Slot: "account_kind", Control: "select", CandidateID: candidateID(first, "Account kind")},
				{Type: "fill_input", Slot: "company", Control: "fill", CandidateID: candidateID(second, "Company name")},
				{Type: "click", CandidateID: candidateID(second, "Next")},
				{Type: "input_checkpoint", CheckpointID: "contact", Slots: []string{"phone", "updates", "quantity", "ratio"}},
				{Type: "fill_input", Slot: "phone", Control: "fill", CandidateID: candidateID(third, "Phone")},
				{Type: "fill_input", Slot: "updates", Control: "check", CandidateID: candidateID(third, "Product updates")},
				{Type: "fill_input", Slot: "quantity", Control: "fill", CandidateID: candidateID(third, "Quantity")},
				{Type: "fill_input", Slot: "ratio", Control: "fill", CandidateID: candidateID(third, "Ratio")},
				{Type: "submit", CandidateID: candidateID(third, "Register")},
				{Type: "human_checkpoint", CheckpointKind: "email_verification"},
			},
			Success: registrationdraft.Success{Origin: origin, Path: "/registration-complete", Proof: registrationdraft.SuccessProofOperatorReviewedDeferred, OperatorReviewed: true, Locator: registrationdraft.SuccessLocator{Role: "status", Name: "Registration complete"}},
		}, CallControls: registrationdraft.CallControls{Approval: "browser_registration_submit", DuplicatePrevention: "operator_attestation", OnDuplicate: "fail", AmbiguousOutcome: "stop_without_retry", CleanupDisposition: "delete_separately"},
	}
	return draft
}
