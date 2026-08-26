package engine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/OpenUdon/browsertools/registrationprofile"
	"github.com/OpenUdon/browsertools/registrationreview"
	"github.com/OpenUdon/openudon/internal/browsertransaction"
	"github.com/OpenUdon/openudon/internal/icot/elicitor"
)

var engineVirtualTime = time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC)

func TestEngineVirtualBrowserSelectionIsGenerationBoundAndWriteFree(t *testing.T) {
	example := filepath.Join(t.TempDir(), "example")
	input := engineVirtualRegistrationInput(t, "signup")
	eng, opened, err := Open(context.Background(), Config{
		ExampleDir: example, NetworkPolicy: "never", Now: func() time.Time { return engineVirtualTime },
		VirtualBrowserTransactions: []elicitor.VirtualBrowserTransactionInput{input},
	})
	if err != nil {
		t.Fatal(err)
	}
	virtual := opened.SourceCandidates.VirtualBrowser
	if virtual.Generation != 1 || len(virtual.Candidates) != 1 || virtual.Candidates[0].Selected {
		t.Fatalf("opened virtual catalog = %#v", virtual)
	}
	encoded, err := json.Marshal(opened)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), string(input.Sources[0].Source)) || strings.Contains(string(encoded), string(input.Sources[0].Review)) {
		t.Fatalf("snapshot exposed source or review bytes: %s", encoded)
	}
	input.Sources[0].Source[0] ^= 0xff
	input.Sources[0].Review[0] ^= 0xff
	target := filepath.Join(example, "browser-registration", "signup.json")
	if _, err := os.Lstat(target); !os.IsNotExist(err) {
		t.Fatalf("virtual discovery wrote target before selection: %v", err)
	}
	selected, err := eng.SelectVirtualBrowserSources(context.Background(), virtual.Generation, []string{virtual.Candidates[0].ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(selected.SelectedSources) != 1 || len(selected.SelectedSources[0].MaterializedContent) != 0 || !selected.SourceCandidates.VirtualBrowser.Candidates[0].Selected {
		t.Fatalf("selected snapshot = %#v", selected)
	}
	if _, err := os.Lstat(target); !os.IsNotExist(err) {
		t.Fatalf("virtual selection copied target before approval: %v", err)
	}
	if _, err := eng.SelectVirtualBrowserSources(context.Background(), 0, nil); err == nil {
		t.Fatal("stale virtual generation was accepted")
	} else if class, code := FailureDetails(err); class != FailureConflict || code != "virtual_sources_stale" {
		t.Fatalf("stale generation = %s %s %v", class, code, err)
	}
}

func TestEngineVirtualBrowserReplacementRevalidatesSelectedIdentity(t *testing.T) {
	example := filepath.Join(t.TempDir(), "example")
	input := engineVirtualRegistrationInput(t, "signup")
	eng, opened, err := Open(context.Background(), Config{
		ExampleDir: example, NetworkPolicy: "never", Now: func() time.Time { return engineVirtualTime },
		VirtualBrowserTransactions: []elicitor.VirtualBrowserTransactionInput{input},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := eng.SelectVirtualBrowserSources(context.Background(), opened.SourceCandidates.VirtualBrowser.Generation, []string{"signup/registration"}); err != nil {
		t.Fatal(err)
	}
	if _, err := eng.ReplaceVirtualBrowserSources(context.Background(), 1, nil); err == nil || !strings.Contains(err.Error(), "stale or unavailable") {
		t.Fatalf("selected-candidate removal error = %v", err)
	} else if class, code := FailureDetails(err); class != FailureConflict || code != "virtual_sources_selected" {
		t.Fatalf("selected-candidate removal classification = %s %s", class, code)
	}
	replaced, err := eng.ReplaceVirtualBrowserSources(context.Background(), 1, []elicitor.VirtualBrowserTransactionInput{input})
	if err != nil {
		t.Fatal(err)
	}
	if replaced.SourceCandidates.VirtualBrowser.Generation != 2 || !replaced.SourceCandidates.VirtualBrowser.Candidates[0].Selected {
		t.Fatalf("replacement snapshot = %#v", replaced.SourceCandidates.VirtualBrowser)
	}
	if _, err := eng.ReplaceVirtualBrowserSources(context.Background(), 1, []elicitor.VirtualBrowserTransactionInput{input}); err == nil {
		t.Fatal("stale replacement generation was accepted")
	}
}

func engineVirtualRegistrationInput(t *testing.T, id string) elicitor.VirtualBrowserTransactionInput {
	t.Helper()
	value, err := registrationprofile.Parse([]byte(`profile: uws.browser-registration.1.0
info:
  title: Engine registration
  applicationOrigins: [https://app.example.test]
  registrationOrigins: [https://app.example.test]
observationKind: accessibility_snapshot
evidence: {learnedAt: "2026-08-25T00:00:00Z", source: synthetic_fixture}
confidence: high
expiresAfter: P30D
verification: {lastVerifiedAt: "2026-08-25T00:00:00Z"}
credentialSlots:
  identifier: {kind: identifier}
flows:
  create_account:
    sequence:
      - navigate: https://app.example.test/register
      - type_credential: {locator: {role: textbox}, slot: identifier}
      - submit: {locator: {role: button, name: Register}}
      - wait_for: {locator: {role: status}}
    effects: [creates_account]
    confirmationPolicy: {required: true}
    success: {origin: https://app.example.test, locator: {role: status}}
`))
	if err != nil {
		t.Fatal(err)
	}
	source, err := registrationprofile.MarshalJSON(value)
	if err != nil {
		t.Fatal(err)
	}
	review, err := registrationreview.Build(value, engineVirtualTime)
	if err != nil {
		t.Fatal(err)
	}
	reviewBytes, err := json.Marshal(review)
	if err != nil {
		t.Fatal(err)
	}
	transaction := browsertransaction.Transaction{
		Version: browsertransaction.Version, ID: id, Kind: browsertransaction.KindRegistration, State: browsertransaction.StateCandidate,
		Candidates: []browsertransaction.Candidate{{Kind: browsertransaction.CandidateRegistration, Schema: value.Profile, SourceSHA256: engineVirtualDigest(source), ReviewSHA256: engineVirtualDigest(reviewBytes)}},
		Provenance: browsertransaction.Provenance{
			Producer: "browsertools", ResultVersion: browsertransaction.ResultRegistrationAuthoringV1,
			ResultSHA256: engineVirtualDigest([]byte("private result")), ObservedAt: engineVirtualTime.Add(-time.Hour).Format(time.RFC3339Nano),
			ExpiresAt: engineVirtualTime.Add(24 * time.Hour).Format(time.RFC3339Nano), Origins: []string{"https://app.example.test"},
		},
		CredentialBindings: []browsertransaction.CredentialBinding{{Slot: "identifier", Binding: "registration_identifier"}},
	}
	return elicitor.VirtualBrowserTransactionInput{Transaction: transaction, Sources: []elicitor.VirtualBrowserSourceInput{{Kind: browsertransaction.CandidateRegistration, Flow: "create_account", Source: source, Review: reviewBytes}}}
}

func engineVirtualDigest(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}
