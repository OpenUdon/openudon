package presentation

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/OpenUdon/openudon/internal/browsertransaction"
	"github.com/OpenUdon/openudon/internal/browsertransaction/engine"
)

func TestReviewCompositionIsKindSpecificValueFreeAndRuntimeFree(t *testing.T) {
	registration := presentationTransaction(browsertransaction.KindRegistration)
	registration.Session = ""
	registration.Candidates = []browsertransaction.Candidate{{Kind: browsertransaction.CandidateRegistration, Schema: "uws.browser-registration.1.0", SourceSHA256: presentationDigest("1"), ReviewSHA256: presentationDigest("2")}}
	registration.Provenance.ResultVersion = browsertransaction.ResultRegistrationAuthoringV1
	resource := New(engine.Snapshot{Version: engine.Version, Revision: presentationDigest("8"), Transaction: &registration, AllowedOperations: []engine.Operation{engine.OperationReview}})
	if resource.Review == nil || resource.Review.Composition != "BRP" || resource.Review.RegistrationAuthoring == nil {
		t.Fatalf("BRP review = %#v", resource.Review)
	}
	registrationReview := resource.Review.RegistrationAuthoring
	if registrationReview.SubmitSupported || registrationReview.AccountAttemptSupported || registrationReview.SessionEstablishment || registrationReview.RuntimeSupported || registrationReview.MutationRequestsAllowed || registrationReview.ApprovalSymbolIsAuthority {
		t.Fatalf("BRP disclosure granted authority = %#v", registrationReview)
	}
	if strings.Join(registrationReview.NetworkMethods, "/") != "GET/HEAD" || !strings.Contains(registrationReview.AccessibilityLabels, "heuristic") || !strings.Contains(registrationReview.AccessibilityLabels, "not data loss prevention") {
		t.Fatalf("BRP disclosure = %#v", registrationReview)
	}

	authentication := presentationTransaction(browsertransaction.KindAuthenticationCapability)
	authentication.Candidates = []browsertransaction.Candidate{
		{Kind: browsertransaction.CandidateAuthentication, Schema: "uws.browser-authentication.1.1", SourceSHA256: presentationDigest("1"), ReviewSHA256: presentationDigest("2")},
		{Kind: browsertransaction.CandidateCapability, Schema: "uws.browser.1.7", SourceSHA256: presentationDigest("3"), ReviewSHA256: presentationDigest("4")},
	}
	authentication.Provenance.ResultVersion = browsertransaction.ResultAuthenticatedAuthoringV2
	authentication.Session = "account_session"
	authentication.CredentialBindings = []browsertransaction.CredentialBinding{}
	resource = New(engine.Snapshot{Version: engine.Version, Revision: presentationDigest("9"), Transaction: &authentication, AllowedOperations: []engine.Operation{engine.OperationReview}})
	if resource.Review == nil || resource.Review.Composition != "BAP+BCP" || resource.Review.Session != "account_session" || resource.Review.RegistrationAuthoring != nil || resource.Review.CredentialBindings == nil {
		t.Fatalf("BAP+BCP review = %#v", resource.Review)
	}
	data, err := json.Marshal(resource)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"private_root", "result_path", "worker_output", "page_content", `"runtime_execution_supported":true`} {
		if strings.Contains(string(data), forbidden) {
			t.Fatalf("presentation contains %q: %s", forbidden, data)
		}
	}
}

func presentationTransaction(kind browsertransaction.Kind) browsertransaction.Transaction {
	return browsertransaction.Transaction{
		Version: browsertransaction.Version, ID: "presentation", Kind: kind, State: browsertransaction.StateCandidate,
		Provenance:         browsertransaction.Provenance{Producer: "browsertools", ResultSHA256: presentationDigest("7"), ObservedAt: "2026-08-26T12:00:00Z", ExpiresAt: "2026-08-26T13:00:00Z", Origins: []string{"https://example.test"}},
		CredentialBindings: []browsertransaction.CredentialBinding{{Slot: "email", Binding: "account_email"}},
	}
}

func presentationDigest(character string) string { return "sha256:" + strings.Repeat(character, 64) }
