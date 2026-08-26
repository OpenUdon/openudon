package browsercandidate

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/OpenUdon/browsertools/registrationauthorresult"
	"github.com/OpenUdon/browsertools/registrationauthorsession"
	"github.com/OpenUdon/browsertools/registrationprofile"
	"github.com/OpenUdon/openudon/internal/browsertransaction"
)

var registrationFixture = []byte(`profile: uws.browser-registration.1.0
info:
  title: Synthetic dedicated test registration
  applicationOrigins: [https://app.example.test]
  registrationOrigins: [https://app.example.test]
observationKind: accessibility_snapshot
evidence: {learnedAt: "2026-08-25T00:00:00Z", source: synthetic_fixture}
confidence: high
expiresAfter: P30D
verification: {lastVerifiedAt: "2026-08-25T00:00:00Z"}
credentialSlots:
  identifier: {kind: identifier}
  password: {kind: password}
flows:
  create_dedicated_test_user:
    sequence:
      - navigate: https://app.example.test/register
      - type_credential: {locator: {role: textbox}, slot: identifier}
      - type_credential: {locator: {role: textbox, name: Password}, slot: password}
      - submit: {locator: {role: button, name: Register}}
      - human_checkpoint: {kind: email_verification}
      - wait_for: {locator: {role: status}}
    effects: [creates_account, sends_verification, requires_human_verification]
    confirmationPolicy: {required: true}
    success: {origin: https://app.example.test, path: /registration-complete, locator: {role: status}}
`)

func TestPrivateInboxAdoptsExactRegistrationCandidate(t *testing.T) {
	root := privateRegistrationRoot(t)
	inbox, err := OpenPrivateInbox(root)
	if err != nil {
		t.Fatal(err)
	}
	defer inbox.Close()
	envelope, written := writeRegistrationResult(t, root)
	candidate, err := inbox.AdoptNewRegistration(adoptionRequest(envelope))
	if err != nil {
		t.Fatal(err)
	}
	transaction := candidate.Transaction()
	if transaction.Kind != browsertransaction.KindRegistration || transaction.State != browsertransaction.StateCandidate ||
		transaction.Session != "" || transaction.Provenance.ResultSHA256 != written.Digest ||
		transaction.Provenance.ResultVersion != browsertransaction.ResultRegistrationAuthoringV1 ||
		len(transaction.Candidates) != 1 || transaction.Candidates[0].SourceSHA256 != envelope.Candidate.SourceDigest {
		t.Fatalf("adopted transaction = %#v", transaction)
	}
	if candidate.ProfileID() != envelope.Candidate.ProfileID || candidate.Flow() != envelope.Flow.Name || candidate.CleanupDisposition() != "delete_separately" ||
		!bytes.Equal(candidate.Source(), envelope.Candidate.Source) {
		t.Fatalf("adopted candidate identity changed")
	}
	reviewed, err := candidate.ReviewedTransaction()
	if err != nil {
		t.Fatal(err)
	}
	if reviewed.State != browsertransaction.StateReviewed || candidate.Transaction().State != browsertransaction.StateCandidate {
		t.Fatalf("review transition mutated candidate or produced wrong state: candidate=%s reviewed=%s", candidate.Transaction().State, reviewed.State)
	}
	firstSource := candidate.Source()
	firstSource[0] ^= 0xff
	firstTransaction := candidate.Transaction()
	firstTransaction.Provenance.Origins[0] = "https://changed.example.test"
	if bytes.Equal(firstSource, candidate.Source()) || candidate.Transaction().Provenance.Origins[0] != envelope.Origins[0] {
		t.Fatal("candidate accessors did not return defensive copies")
	}
	serialized, err := json.Marshal(candidate)
	if err != nil || string(serialized) != "{}" || bytes.Contains(serialized, []byte(root)) || strings.Contains(string(candidate.Source()), root) || strings.Contains(string(candidate.Review()), root) {
		t.Fatalf("private path escaped adopted candidate: %q, %v", serialized, err)
	}
}

func TestPrivateInboxRejectsReviewBindingAndLifecycleDrift(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*AdoptRegistrationRequest)
	}{
		{name: "not confirmed", mutate: func(value *AdoptRegistrationRequest) { value.Review.Confirmed = false }},
		{name: "source digest", mutate: func(value *AdoptRegistrationRequest) { value.Review.SourceSHA256 = browserDigest("changed") }},
		{name: "profile identity", mutate: func(value *AdoptRegistrationRequest) { value.Review.ProfileID = "other" }},
		{name: "flow", mutate: func(value *AdoptRegistrationRequest) { value.Review.Flow = "other" }},
		{name: "candidate id", mutate: func(value *AdoptRegistrationRequest) {
			value.Review.ReviewedCandidates[0].ID = "candidate-ffffffffffffffff"
		}},
		{name: "candidate generation", mutate: func(value *AdoptRegistrationRequest) { value.Review.ReviewedCandidates[0].Generation++ }},
		{name: "bounds", mutate: func(value *AdoptRegistrationRequest) { value.Review.Bounds.MaxRequests++ }},
		{name: "observations", mutate: func(value *AdoptRegistrationRequest) { value.Review.Observations++ }},
		{name: "network lower bound", mutate: func(value *AdoptRegistrationRequest) { value.Review.MinimumRequests++ }},
		{name: "origin", mutate: func(value *AdoptRegistrationRequest) { value.Review.Origins[0] = "https://other.example.test" }},
		{name: "cleanup", mutate: func(value *AdoptRegistrationRequest) {
			value.Review.CleanupDisposition = "retain_dedicated_test_identity"
		}},
		{name: "missing binding", mutate: func(value *AdoptRegistrationRequest) { value.CredentialBindings = value.CredentialBindings[:1] }},
		{name: "duplicate binding", mutate: func(value *AdoptRegistrationRequest) { value.CredentialBindings[1].Slot = "identifier" }},
		{name: "stale", mutate: func(value *AdoptRegistrationRequest) { value.AssessedAt = time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC) }},
		{name: "invalid transaction", mutate: func(value *AdoptRegistrationRequest) { value.TransactionID = "../private" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := privateRegistrationRoot(t)
			inbox, err := OpenPrivateInbox(root)
			if err != nil {
				t.Fatal(err)
			}
			defer inbox.Close()
			envelope, _ := writeRegistrationResult(t, root)
			request := adoptionRequest(envelope)
			test.mutate(&request)
			if _, err := inbox.AdoptNewRegistration(request); err == nil || strings.Contains(err.Error(), root) {
				t.Fatalf("drift error = %v", err)
			}
		})
	}
}

func TestPrivateInboxRejectsUnsafeAndChangedResults(t *testing.T) {
	t.Run("symlink", func(t *testing.T) {
		root := privateRegistrationRoot(t)
		inbox, err := OpenPrivateInbox(root)
		if err != nil {
			t.Fatal(err)
		}
		defer inbox.Close()
		target := filepath.Join(t.TempDir(), "target")
		if err := os.WriteFile(target, []byte("{}\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(target, filepath.Join(root, "registration-authoring-0000000000000000.json")); err != nil {
			t.Fatal(err)
		}
		if _, err := inbox.AdoptNewRegistration(AdoptRegistrationRequest{}); err == nil {
			t.Fatal("symlink result was accepted")
		}
	})
	t.Run("public mode", func(t *testing.T) {
		root := privateRegistrationRoot(t)
		inbox, err := OpenPrivateInbox(root)
		if err != nil {
			t.Fatal(err)
		}
		defer inbox.Close()
		if err := os.WriteFile(filepath.Join(root, "registration-authoring-0000000000000000.json"), []byte("{}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := inbox.AdoptNewRegistration(AdoptRegistrationRequest{}); err == nil {
			t.Fatal("public result was accepted")
		}
	})
	t.Run("replacement during read", func(t *testing.T) {
		root := privateRegistrationRoot(t)
		inbox, err := OpenPrivateInbox(root)
		if err != nil {
			t.Fatal(err)
		}
		defer inbox.Close()
		envelope, written := writeRegistrationResult(t, root)
		replacement := filepath.Join(root, ".replacement")
		data, err := os.ReadFile(written.Path)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(replacement, data, 0o600); err != nil {
			t.Fatal(err)
		}
		original := privateResultReadHook
		defer func() { privateResultReadHook = original }()
		privateResultReadHook = func() {
			_ = os.Remove(written.Path)
			_ = os.Rename(replacement, written.Path)
		}
		if _, err := inbox.AdoptNewRegistration(adoptionRequest(envelope)); err == nil {
			t.Fatal("result replacement was accepted")
		}
	})
	t.Run("root replacement", func(t *testing.T) {
		root := privateRegistrationRoot(t)
		inbox, err := OpenPrivateInbox(root)
		if err != nil {
			t.Fatal(err)
		}
		defer inbox.Close()
		old := root + "-old"
		if err := os.Rename(root, old); err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(root, 0o700); err != nil {
			t.Fatal(err)
		}
		if _, err := inbox.AdoptNewRegistration(AdoptRegistrationRequest{}); err == nil {
			t.Fatal("private root replacement was accepted")
		}
	})
}

func TestPrivateInboxRejectsNonCanonicalResultJSON(t *testing.T) {
	tests := map[string][]byte{
		"duplicate": []byte(`{"schema":"browsertools.registration-authoring.v1","schema":"browsertools.registration-authoring.v1"}`),
		"unknown":   []byte(`{"unknown":true}`),
		"trailing":  []byte(`{} {}`),
		"utf8":      {0xff, '\n'},
	}
	for name, data := range tests {
		t.Run(name, func(t *testing.T) {
			root := privateRegistrationRoot(t)
			inbox, err := OpenPrivateInbox(root)
			if err != nil {
				t.Fatal(err)
			}
			defer inbox.Close()
			name := resultName(browserDigestBytes(data))
			if err := os.WriteFile(filepath.Join(root, name), data, 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := inbox.AdoptNewRegistration(AdoptRegistrationRequest{Review: RegistrationReview{Confirmed: true}, AssessedAt: time.Date(2026, 8, 25, 1, 0, 0, 0, time.UTC)}); err == nil {
				t.Fatal("non-canonical result was accepted")
			}
		})
	}
}

func TestPrivateInboxRejectsPartialOversizedAndAmbiguousWrites(t *testing.T) {
	t.Run("oversized", func(t *testing.T) {
		root := privateRegistrationRoot(t)
		inbox, err := OpenPrivateInbox(root)
		if err != nil {
			t.Fatal(err)
		}
		defer inbox.Close()
		data := bytes.Repeat([]byte{' '}, maxPrivateResultBytes+1)
		if err := os.WriteFile(filepath.Join(root, resultName(browserDigestBytes(data))), data, 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := inbox.AdoptNewRegistration(AdoptRegistrationRequest{}); err == nil {
			t.Fatal("oversized partial result was accepted")
		}
	})
	t.Run("multiple new results", func(t *testing.T) {
		root := privateRegistrationRoot(t)
		inbox, err := OpenPrivateInbox(root)
		if err != nil {
			t.Fatal(err)
		}
		defer inbox.Close()
		for _, data := range [][]byte{[]byte("{}\n"), []byte("{ }\n")} {
			if err := os.WriteFile(filepath.Join(root, resultName(browserDigestBytes(data))), data, 0o600); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := inbox.AdoptNewRegistration(AdoptRegistrationRequest{}); err == nil {
			t.Fatal("ambiguous worker output was accepted")
		}
	})
	t.Run("digest-name tamper", func(t *testing.T) {
		root := privateRegistrationRoot(t)
		inbox, err := OpenPrivateInbox(root)
		if err != nil {
			t.Fatal(err)
		}
		defer inbox.Close()
		envelope, written := writeRegistrationResult(t, root)
		data, err := os.ReadFile(written.Path)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(written.Path, append(data, ' '), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := inbox.AdoptNewRegistration(adoptionRequest(envelope)); err == nil {
			t.Fatal("digest-name mismatch was accepted")
		}
	})
}

func writeRegistrationResult(t *testing.T, root string) (*registrationauthorresult.Envelope, *registrationauthorresult.Written) {
	t.Helper()
	profile, err := registrationprofile.Parse(registrationFixture)
	if err != nil {
		t.Fatal(err)
	}
	profileBytes, err := registrationprofile.MarshalJSON(profile)
	if err != nil {
		t.Fatal(err)
	}
	observedAt := time.Date(2026, 8, 25, 0, 30, 0, 0, time.UTC)
	envelope, err := registrationauthorresult.Build(registrationauthorresult.BuildRequest{
		CreatedAt: observedAt.Add(30 * time.Minute),
		Completion: &registrationauthorsession.Completion{
			Protocol: registrationauthorsession.Protocol, ProfileID: "synthetic_registration",
			Profile: *profile, ProfileBytes: profileBytes,
			ReviewedCandidates: []registrationauthorsession.ReviewedCandidate{{
				ID: "candidate-0123456789abcdef", Generation: 1, Role: "button", Label: "Register", Matches: 1,
			}},
			Flow: "create_dedicated_test_user", CleanupDisposition: "delete_separately",
			Origins: []string{"https://app.example.test"}, ObservedAt: observedAt,
			Bounds: registrationauthorsession.Bounds{
				NavigationTimeoutMS: 20_000, TotalTimeoutMS: 300_000, MaxRequests: 256,
				MaxResponseBytes: 32 << 20, MaxObservations: 64, MaxCandidates: 128,
			},
			Observations: 1, Network: registrationauthorsession.NetworkSummary{Requests: 1, GETRequests: 1},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	written, err := registrationauthorresult.WritePrivateExclusive(root, envelope)
	if err != nil {
		t.Fatal(err)
	}
	return envelope, written
}

func adoptionRequest(envelope *registrationauthorresult.Envelope) AdoptRegistrationRequest {
	return AdoptRegistrationRequest{
		TransactionID: "registration-transaction",
		CredentialBindings: []browsertransaction.CredentialBinding{
			{Slot: "identifier", Binding: "registration_identifier"},
			{Slot: "password", Binding: "registration_password"},
		},
		Review: RegistrationReview{
			Confirmed: true, ProfileID: envelope.Candidate.ProfileID, Flow: envelope.Flow.Name,
			SourceSHA256:       envelope.Candidate.SourceDigest,
			ReviewedCandidates: append([]registrationauthorsession.ReviewedCandidate(nil), envelope.ReviewedCandidates...),
			CleanupDisposition: envelope.CallPolicy.CleanupDisposition,
			Origins:            append([]string(nil), envelope.Origins...),
			Bounds:             envelope.Bounds,
			Observations:       envelope.Observations,
			MinimumRequests:    1,
		},
		AssessedAt: time.Date(2026, 8, 25, 1, 0, 1, 0, time.UTC),
	}
}

func privateRegistrationRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	return root
}

func browserDigest(value string) string { return browserDigestBytes([]byte(value)) }

func browserDigestBytes(data []byte) string {
	return digest(data)
}
