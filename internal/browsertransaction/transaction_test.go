package browsertransaction

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

func TestCanonicalRoundTripAndDigest(t *testing.T) {
	transaction := validAuthenticationCapability()
	transaction.Candidates[0], transaction.Candidates[1] = transaction.Candidates[1], transaction.Candidates[0]
	transaction.CredentialBindings[0], transaction.CredentialBindings[1] = transaction.CredentialBindings[1], transaction.CredentialBindings[0]
	transaction.Provenance.Origins[0], transaction.Provenance.Origins[1] = transaction.Provenance.Origins[1], transaction.Provenance.Origins[0]

	data, err := CanonicalBytes(transaction)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Candidates[0].Kind != CandidateAuthentication || decoded.CredentialBindings[0].Slot != "password" || decoded.Provenance.Origins[0] != "https://app.example.test" {
		t.Fatalf("canonical order was not applied: %#v", decoded)
	}
	digest, err := Digest(decoded)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeVerified(data, digest); err != nil {
		t.Fatalf("verified decode: %v", err)
	}

	var changed Transaction
	if err := json.Unmarshal(data, &changed); err != nil {
		t.Fatal(err)
	}
	changed.ID = "changed"
	changedData, err := CanonicalBytes(changed)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeVerified(changedData, digest); err == nil || !strings.Contains(err.Error(), "digest mismatch") {
		t.Fatalf("changed transaction was not rejected: %v", err)
	}
}

func TestCompositionAndValueFreeBoundary(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Transaction)
	}{
		{"registration session", func(value *Transaction) {
			registration := validRegistration()
			*value = registration
			value.Session = "forbidden"
		}},
		{"authentication missing session", func(value *Transaction) { value.Session = "" }},
		{"authentication missing capability", func(value *Transaction) { value.Candidates = value.Candidates[:1] }},
		{"candidate schema family mismatch", func(value *Transaction) { value.Candidates[0].Schema = "uws.browser.1.7" }},
		{"duplicate binding", func(value *Transaction) { value.CredentialBindings[1].Binding = value.CredentialBindings[0].Binding }},
		{"noncanonical origin", func(value *Transaction) { value.Provenance.Origins[0] = "https://LOGIN.example.test" }},
		{"expired at observation", func(value *Transaction) { value.Provenance.ExpiresAt = value.Provenance.ObservedAt }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value := validAuthenticationCapability()
			test.mutate(&value)
			if err := value.Validate(); err == nil {
				t.Fatal("invalid transaction accepted")
			}
		})
	}
	credentialless := validAuthenticationCapability()
	credentialless.CredentialBindings = []CredentialBinding{}
	if err := credentialless.Validate(); err != nil {
		t.Fatalf("credentialless authentication-capability transaction: %v", err)
	}
	nullBindings, err := CanonicalBytes(validRegistration())
	if err != nil {
		t.Fatal(err)
	}
	nullBindings = []byte(strings.Replace(string(nullBindings), `"credential_bindings":[{"slot":"identifier","binding":"registration_identifier"},{"slot":"password","binding":"registration_password"}]`, `"credential_bindings":null`, 1))
	if _, err := Decode(nullBindings); err == nil {
		t.Fatal("null credential bindings accepted")
	}
	registrationWithoutBindings := validRegistration()
	registrationWithoutBindings.CredentialBindings = []CredentialBinding{}
	if err := registrationWithoutBindings.Validate(); err == nil {
		t.Fatal("credentialless registration transaction accepted")
	}
	missingBindings := validAuthenticationCapability()
	missingBindings.CredentialBindings = nil
	if _, err := CanonicalBytes(missingBindings); err == nil {
		t.Fatal("canonical encoding accepted missing credential bindings")
	}

	data, err := CanonicalBytes(validRegistration())
	if err != nil {
		t.Fatal(err)
	}
	forbidden := strings.TrimSuffix(string(data), "}") + `,"private_result_path":"/private/result.json"}`
	if _, err := Decode([]byte(forbidden)); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("private path field was not rejected: %v", err)
	}
	for _, field := range []string{"credential_value", "account_identifier", "page_content", "raw_worker_output", "session_state"} {
		candidate := strings.TrimSuffix(string(data), "}") + `,"` + field + `":"forbidden"}`
		if _, err := Decode([]byte(candidate)); err == nil {
			t.Fatalf("forbidden field %s was accepted", field)
		}
	}
}

func TestPublicSchemaCompilesAndAcceptsCanonicalTransactions(t *testing.T) {
	schemaBytes, err := os.ReadFile(filepath.Join("..", "..", "docs", "schemas", "openudon.browser-profile-transaction.v1.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	document, err := jsonschema.UnmarshalJSON(bytes.NewReader(schemaBytes))
	if err != nil {
		t.Fatal(err)
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("transaction.schema.json", document); err != nil {
		t.Fatal(err)
	}
	schema, err := compiler.Compile("transaction.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, transaction := range []Transaction{validAuthenticationCapability(), validRegistration()} {
		data, err := CanonicalBytes(transaction)
		if err != nil {
			t.Fatal(err)
		}
		instance, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
		if err != nil {
			t.Fatal(err)
		}
		if err := schema.Validate(instance); err != nil {
			t.Fatalf("schema rejected canonical %s transaction: %v", transaction.Kind, err)
		}
	}

	invalid := validRegistration()
	invalid.Session = "forbidden"
	data, err := json.Marshal(invalid)
	if err != nil {
		t.Fatal(err)
	}
	instance, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if err := schema.Validate(instance); err == nil {
		t.Fatal("schema accepted registration session")
	}

	invalid = validAuthenticationCapability()
	invalid.Candidates[0], invalid.Candidates[1] = invalid.Candidates[1], invalid.Candidates[0]
	data, err = json.Marshal(invalid)
	if err != nil {
		t.Fatal(err)
	}
	instance, err = jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if err := schema.Validate(instance); err == nil {
		t.Fatal("schema accepted noncanonical candidate composition")
	}
}

func TestPublishedExamplesValidate(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join("..", "..", "docs", "examples", "browser-profile-transaction-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 2 {
		t.Fatalf("found %d published transaction examples, want 2", len(paths))
	}
	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := Decode(data); err != nil {
				t.Fatalf("published example is invalid: %v", err)
			}
		})
	}
}

func TestLifecycleAndImmutableTransitions(t *testing.T) {
	candidate := validAuthenticationCapability()
	reviewed := candidate
	reviewed.State = StateReviewed
	if err := ValidateTransition(candidate, reviewed); err != nil {
		t.Fatal(err)
	}
	prepared := reviewed
	prepared.State = StatePrepared
	prepared.Preparation = validPreparation()
	if err := ValidateTransition(reviewed, prepared); err != nil {
		t.Fatal(err)
	}
	promoted := prepared
	promoted.State = StatePromoted
	promoted.Promotion = &Promotion{GenerationSHA256: digest("e")}
	if err := ValidateTransition(prepared, promoted); err != nil {
		t.Fatal(err)
	}
	if err := ValidateTransition(promoted, reviewed); err == nil {
		t.Fatal("terminal promoted transaction advanced")
	}

	changed := reviewed
	changed.Candidates = append([]Candidate(nil), reviewed.Candidates...)
	changed.Candidates[0].ReviewSHA256 = digest("f")
	changed.State = StatePrepared
	changed.Preparation = validPreparation()
	if err := ValidateTransition(reviewed, changed); err == nil || !strings.Contains(err.Error(), "immutable") {
		t.Fatalf("candidate digest drift was not rejected: %v", err)
	}

	indeterminate := prepared
	indeterminate.State = StateIndeterminate
	indeterminate.Failure = &Failure{Class: FailureIndeterminate, Code: FailurePromotionIndeterminate}
	if err := ValidateTransition(prepared, indeterminate); err != nil {
		t.Fatal(err)
	}
	reconciled := indeterminate
	reconciled.State = StatePromoted
	reconciled.Failure = nil
	reconciled.Promotion = &Promotion{GenerationSHA256: digest("e")}
	if err := ValidateTransition(indeterminate, reconciled); err != nil {
		t.Fatal(err)
	}
}

func TestLifecycleFieldsAndFailureCodes(t *testing.T) {
	invalidPrepared := validRegistration()
	invalidPrepared.State = StatePrepared
	if err := invalidPrepared.Validate(); err == nil {
		t.Fatal("prepared state without preparation accepted")
	}
	invalidFailure := validRegistration()
	invalidFailure.State = StateFailed
	invalidFailure.Failure = &Failure{Class: FailureConflict, Code: FailurePromotionFailed}
	if err := invalidFailure.Validate(); err == nil {
		t.Fatal("mismatched failure class/code accepted")
	}
}

func TestClosedTransitionMatrix(t *testing.T) {
	states := []State{
		StateCandidate,
		StateReviewed,
		StatePrepared,
		StatePromoted,
		StateCancelled,
		StateFailed,
		StateIndeterminate,
	}
	want := map[State]map[State]bool{
		StateCandidate:     {StateReviewed: true, StateCancelled: true, StateFailed: true},
		StateReviewed:      {StatePrepared: true, StateCancelled: true, StateFailed: true},
		StatePrepared:      {StatePromoted: true, StateCancelled: true, StateFailed: true, StateIndeterminate: true},
		StateIndeterminate: {StatePrepared: true, StatePromoted: true, StateCancelled: true, StateFailed: true},
	}
	for _, previous := range states {
		for _, next := range states {
			if got := allowedTransition(previous, next); got != want[previous][next] {
				t.Errorf("allowedTransition(%q, %q) = %t, want %t", previous, next, got, want[previous][next])
			}
		}
	}
}

func TestStrictDecodeRejectsUnknownAndMultipleDocuments(t *testing.T) {
	data, err := CanonicalBytes(validRegistration())
	if err != nil {
		t.Fatal(err)
	}
	unknown := strings.Replace(string(data), `"state":"candidate"`, `"state":"candidate","message":"raw error"`, 1)
	if _, err := Decode([]byte(unknown)); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("unknown field accepted: %v", err)
	}
	if _, err := Decode(append(append([]byte(nil), data...), data...)); err == nil || !strings.Contains(err.Error(), "multiple JSON values") {
		t.Fatalf("multiple documents accepted: %v", err)
	}
}

func validAuthenticationCapability() Transaction {
	return Transaction{
		Version: Version, ID: "browser-login-dashboard", Kind: KindAuthenticationCapability, State: StateCandidate,
		Candidates: []Candidate{
			{Kind: CandidateAuthentication, Schema: "uws.browser-authentication.1.1", SourceSHA256: digest("a"), ReviewSHA256: digest("b")},
			{Kind: CandidateCapability, Schema: "uws.browser.1.7", SourceSHA256: digest("c"), ReviewSHA256: digest("d")},
		},
		Provenance: Provenance{
			Producer: "browsertools", ResultVersion: "browsertools.authenticated-authoring.v2", ResultSHA256: digest("9"),
			ObservedAt: "2026-08-25T12:00:00Z", ExpiresAt: "2026-08-26T12:00:00Z",
			Origins: []string{"https://app.example.test", "https://login.example.test"},
		},
		CredentialBindings: []CredentialBinding{{Slot: "password", Binding: "member_password"}, {Slot: "username", Binding: "member_username"}},
		Session:            "dashboard_session",
	}
}

func validRegistration() Transaction {
	return Transaction{
		Version: Version, ID: "browser-registration", Kind: KindRegistration, State: StateCandidate,
		Candidates: []Candidate{{Kind: CandidateRegistration, Schema: "uws.browser-registration.1.0", SourceSHA256: digest("1"), ReviewSHA256: digest("2")}},
		Provenance: Provenance{
			Producer: "browsertools", ResultVersion: "browsertools.registration-authoring.v1", ResultSHA256: digest("3"),
			ObservedAt: "2026-08-25T12:00:00Z", ExpiresAt: "2026-08-26T12:00:00Z", Origins: []string{"https://register.example.test"},
		},
		CredentialBindings: []CredentialBinding{{Slot: "identifier", Binding: "registration_identifier"}, {Slot: "password", Binding: "registration_password"}},
	}
}

func validPreparation() *Preparation {
	return &Preparation{PackageSHA256: digest("6"), QualificationSHA256: digest("7")}
}

func digest(character string) string { return "sha256:" + strings.Repeat(character, 64) }
