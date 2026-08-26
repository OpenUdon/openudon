package ui

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/OpenUdon/browsertools/registrationauthorsession"
	"github.com/OpenUdon/browsertools/registrationprofile"
)

func TestBuildRegistrationDraftProducesCanonicalV2ProfileAndDisclosure(t *testing.T) {
	request := validRegistrationDraftRequest()
	canonical, candidates, bindings, disclosure, err := buildRegistrationDraft(
		request,
		registrationAuthoringStartRequest{Origins: []string{"https://app.example.test"}},
		registrationDraftObservation(),
		time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatal(err)
	}
	profile, err := registrationprofile.Parse(canonical)
	if err != nil {
		t.Fatal(err)
	}
	reencoded, err := registrationprofile.MarshalJSON(profile)
	if err != nil || !bytes.Equal(canonical, reencoded) || registrationprofile.ValidateRetainedNavigationV2(profile) != nil {
		t.Fatalf("canonical profile mismatch: %v", err)
	}
	if len(candidates) != 4 || len(bindings) != 2 || bindings[0].Slot != "identifier" || bindings[1].Slot != "password" {
		t.Fatalf("review authority = candidates=%v bindings=%v", candidates, bindings)
	}
	if disclosure.ProfileSHA256 == "" || !bytes.Equal(disclosure.Canonical, canonical) || len(disclosure.RetainedQueries) != 1 ||
		disclosure.RetainedQueries[0].Parameters[0].Key != "action" || disclosure.RetainedQueries[0].Parameters[0].Value != "startnew" ||
		disclosure.CallControls.AmbiguousOutcome != "stop_without_retry" || !strings.Contains(disclosure.AccessibilityLabels, "Accessibility") {
		t.Fatalf("draft disclosure = %#v", disclosure)
	}
	if strings.Contains(string(canonical), "dedicated_test_identifier") || strings.Contains(string(canonical), "dedicated_test_password") {
		t.Fatalf("symbolic environment bindings crossed into profile: %s", canonical)
	}
}

func TestBuildRegistrationDraftRejectsUnsafeQueriesAndIncompleteAuthority(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*registrationDraftRequest)
	}{
		{name: "sensitive key", mutate: func(v *registrationDraftRequest) {
			v.Flow.Steps[0].Navigate = "https://app.example.test/register?token=structural"
		}},
		{name: "secret shaped value", mutate: func(v *registrationDraftRequest) {
			v.Flow.Steps[0].Navigate = "https://app.example.test/register?action=" + "sk_" + "live_" + strings.Repeat("1", 20)
		}},
		{name: "repeated key", mutate: func(v *registrationDraftRequest) {
			v.Flow.Steps[0].Navigate = "https://app.example.test/register?action=startnew&action=again"
		}},
		{name: "fragment", mutate: func(v *registrationDraftRequest) {
			v.Flow.Steps[0].Navigate = "https://app.example.test/register?action=startnew#private"
		}},
		{name: "origin escape", mutate: func(v *registrationDraftRequest) {
			v.Flow.Steps[0].Navigate = "https://other.example.test/register?action=startnew"
		}},
		{name: "no submit", mutate: func(v *registrationDraftRequest) { v.Flow.Steps[3].Type = "click" }},
		{name: "unfixed controls", mutate: func(v *registrationDraftRequest) { v.CallControls.OnDuplicate = "retry" }},
		{name: "binding collision", mutate: func(v *registrationDraftRequest) { v.CredentialSlots[1].Binding = v.CredentialSlots[0].Binding }},
		{name: "secret shaped binding", mutate: func(v *registrationDraftRequest) { v.CredentialSlots[1].Binding = "m8z_pq4_r2x7_n1cv9bk3sd6fh0jl5wt2" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := validRegistrationDraftRequest()
			test.mutate(&request)
			if _, _, _, _, err := buildRegistrationDraft(request, registrationAuthoringStartRequest{Origins: []string{"https://app.example.test"}}, registrationDraftObservation(), time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)); err == nil {
				t.Fatal("unsafe registration draft was accepted")
			}
		})
	}
}

func validRegistrationDraftRequest() registrationDraftRequest {
	return registrationDraftRequest{
		Title: "Synthetic dedicated test registration", Provider: "Synthetic loopback", Confidence: "high", ExpiresAfter: "P30D",
		CredentialSlots: []registrationDraftSlot{
			{Slot: "identifier", Kind: "identifier", Binding: "dedicated_test_identifier"},
			{Slot: "password", Kind: "password", Binding: "dedicated_test_password"},
		},
		Flow: registrationDraftFlow{
			Name: "create_dedicated_test_user", Description: "Create one dedicated test identity.",
			Steps: []registrationDraftStep{
				{Type: "navigate", Navigate: "https://app.example.test/register?action=startnew"},
				{Type: "type_credential", CandidateID: "candidate-0000000000000001", Slot: "identifier"},
				{Type: "type_credential", CandidateID: "candidate-0000000000000002", Slot: "password"},
				{Type: "submit", CandidateID: "candidate-0000000000000003"},
				{Type: "human_checkpoint", CheckpointKind: "email_verification"},
				{Type: "wait_for", CandidateID: "candidate-0000000000000004"},
			},
			Effects:            []string{"creates_account", "sends_verification", "requires_human_verification"},
			ConfirmationPrompt: "Approve creation of one dedicated test identity.",
			Success:            registrationDraftSuccess{Origin: "https://app.example.test", Path: "/registration-complete", CandidateID: "candidate-0000000000000004"},
		},
		CallControls: registrationDraftCallControls{
			Approval: "browser_registration_submit", DuplicatePrevention: "operator_attestation", OnDuplicate: "fail",
			AmbiguousOutcome: "stop_without_retry", CleanupDisposition: "delete_separately",
		},
	}
}

func registrationDraftObservation() registrationauthorsession.Observation {
	return registrationauthorsession.Observation{
		Generation: 1, Origin: "https://app.example.test", Path: "/register",
		Candidates: []registrationauthorsession.Candidate{
			{ID: "candidate-0000000000000001", Role: "textbox", Label: "Email", Matches: 1},
			{ID: "candidate-0000000000000002", Role: "textbox", Label: "Password", Matches: 1},
			{ID: "candidate-0000000000000003", Role: "button", Label: "Register", Matches: 1},
			{ID: "candidate-0000000000000004", Role: "status", Label: "Registration complete", Matches: 1},
		}, Diagnostics: []string{},
	}
}
