package elicitor

import (
	"path/filepath"
	"testing"

	"github.com/OpenUdon/apitools"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

func TestSecurityAlternativeSelectionPersistsAndDoesNotUnionCredentials(t *testing.T) {
	op := &apitools.OperationSummary{
		OperationID: "listItems", Method: "GET", Path: "/items/{id}",
		Parameters: []apitools.ParameterSummary{{Name: "id", In: "path", Required: true, Type: "string"}},
		SecurityRequirementSets: []apitools.SecurityRequirementSetSummary{
			{Requirements: []apitools.SecuritySummary{{Name: "api_key", Type: "apiKey", In: "header", ParameterName: "X-API-Key"}, {Name: "client_certificate", Type: "mutualTLS"}}},
			{Requirements: []apitools.SecuritySummary{{Name: "bearer", Type: "http", Scheme: "bearer"}}},
			{},
		},
	}
	step := &rollout.Step{
		Name: "list", Type: "http", Operation: "listItems",
		With: map[string]string{"id": "inputs.id", "api_key": "credentials.api_key", "client_certificate": "credentials.client_certificate", "Authorization": "credentials.bearer"},
	}
	session := Session{Intent: rollout.Intent{Steps: []*rollout.Step{step}}, Credentials: []string{"api_key", "client_certificate", "bearer"}, CredentialsSet: true}
	if !selectSecurityAlternative(&session, step, op, "api_key + client_certificate") {
		t.Fatal("failed to select the conjunctive API-key and client-certificate alternative")
	}
	selected, ok := selectedSecurityAlternative(session, step, op)
	if !ok || len(selected.Requirements) != 2 {
		t.Fatalf("selected alternative = %#v, ok=%t", selected, ok)
	}
	if missing := missingRequiredFields(session, step, op); len(missing) != 0 {
		t.Fatalf("selected alternative fields reported missing: %#v", missing)
	}
	issues := validateOpenAPIRequestMappings(session, step, op, "steps.list")
	if !hasReadinessCode(issues, "invented_request_field") {
		t.Fatalf("unselected bearer field was accepted: %#v", issues)
	}

	path := filepath.Join(t.TempDir(), "session.yaml")
	if err := SaveDraft(path, session); err != nil {
		t.Fatal(err)
	}
	resumed, ok, err := LoadDraft(path)
	if err != nil || !ok {
		t.Fatalf("LoadDraft ok=%t error=%v", ok, err)
	}
	selected, ok = selectedSecurityAlternative(resumed, resumed.Intent.Steps[0], op)
	if !ok || len(selected.Requirements) != 2 || len(resumed.DecisionEvidence) == 0 {
		t.Fatalf("resumed selection/evidence = %#v / %#v", selected, resumed.DecisionEvidence)
	}
}

func TestAnonymousSecurityAlternativeRequiresNoCredentialField(t *testing.T) {
	op := &apitools.OperationSummary{SecurityRequirementSets: []apitools.SecurityRequirementSetSummary{
		{Requirements: []apitools.SecuritySummary{{Name: "bearer", Type: "http", Scheme: "bearer"}}},
		{},
	}}
	step := &rollout.Step{Name: "read"}
	session := Session{}
	if !selectSecurityAlternative(&session, step, op, "anonymous") {
		t.Fatal("failed to select anonymous alternative")
	}
	fields, ok := selectedSecurityCredentialFields(session, step, op)
	if !ok || len(fields) != 0 {
		t.Fatalf("anonymous credential fields = %#v, ok=%t", fields, ok)
	}
	if operationNeedsCredentialForStep(session, step, op) {
		t.Fatal("anonymous alternative still requires a credential binding")
	}
}

func TestSecurityAlternativeSelectionUsesFingerprintAcrossReordering(t *testing.T) {
	op := &apitools.OperationSummary{SecurityRequirementSets: []apitools.SecurityRequirementSetSummary{
		{Requirements: []apitools.SecuritySummary{{Name: "oauth", Type: "oauth2", Scopes: []string{"read"}}}},
		{Requirements: []apitools.SecuritySummary{{Name: "oauth", Type: "oauth2", Scopes: []string{"write"}}}},
	}}
	step := &rollout.Step{Name: "sync"}
	session := Session{}
	if !selectSecurityAlternative(&session, step, op, "2") {
		t.Fatal("failed to select the second OAuth scope alternative")
	}
	selected, ok := selectedSecurityAlternative(session, step, op)
	if !ok || len(selected.Requirements) != 1 || len(selected.Requirements[0].Scopes) != 1 || selected.Requirements[0].Scopes[0] != "write" {
		t.Fatalf("selected alternative = %#v, ok=%t", selected, ok)
	}
	reordered := &apitools.OperationSummary{SecurityRequirementSets: []apitools.SecurityRequirementSetSummary{
		op.SecurityRequirementSets[1], op.SecurityRequirementSets[0],
	}}
	selected, ok = selectedSecurityAlternative(session, step, reordered)
	if !ok || selected.Requirements[0].Scopes[0] != "write" {
		t.Fatalf("reordered selection = %#v, ok=%t", selected, ok)
	}
	if selectSecurityAlternative(&session, step, &apitools.OperationSummary{SecurityRequirementSets: []apitools.SecurityRequirementSetSummary{
		{Requirements: []apitools.SecuritySummary{{Name: "same"}}},
		{Requirements: []apitools.SecuritySummary{{Name: "same"}}},
	}}, "same") {
		t.Fatal("ambiguous repeated security label was accepted")
	}
}

func TestLegacySecurityAlternativeIndexRequiresUniqueDecisionEvidence(t *testing.T) {
	op := &apitools.OperationSummary{SecurityRequirementSets: []apitools.SecurityRequirementSetSummary{
		{Requirements: []apitools.SecuritySummary{{Name: "api_key", Type: "apiKey"}}},
		{Requirements: []apitools.SecuritySummary{{Name: "bearer", Type: "http"}}},
	}}
	step := &rollout.Step{Name: "read"}
	session := Session{}
	session.Interview.Metadata = map[string]string{securityAlternativeMetadataKey(step): "2"}
	if _, ok := selectedSecurityAlternative(session, step, op); ok {
		t.Fatal("legacy index without decision evidence was accepted")
	}
	session.DecisionEvidence = []DecisionEvidence{{
		Slot: securityAlternativeSlot(step), Value: "bearer", Source: mappingSourceUser, Confidence: mappingConfidenceHigh,
	}}
	selected, ok := selectedSecurityAlternative(session, step, op)
	if !ok || selected.Requirements[0].Name != "bearer" {
		t.Fatalf("uniquely confirmed legacy selection = %#v, ok=%t", selected, ok)
	}
	session.DecisionEvidence = append(session.DecisionEvidence, DecisionEvidence{
		Slot: securityAlternativeSlot(step), Value: "bearer", Source: mappingSourceUser, Confidence: mappingConfidenceHigh,
	})
	if _, ok := selectedSecurityAlternative(session, step, op); ok {
		t.Fatal("multiply confirmed legacy index was accepted")
	}
}

func TestLegacySecurityAlternativeRejectsNonHumanOrUnconfirmedEvidence(t *testing.T) {
	op := &apitools.OperationSummary{SecurityRequirementSets: []apitools.SecurityRequirementSetSummary{
		{Requirements: []apitools.SecuritySummary{{Name: "api_key", Type: "apiKey"}}},
		{Requirements: []apitools.SecuritySummary{{Name: "bearer", Type: "http"}}},
	}}
	step := &rollout.Step{Name: "read"}
	for _, evidence := range []DecisionEvidence{
		{Slot: securityAlternativeSlot(step), Value: "bearer", Source: mappingSourceLLM, Confidence: mappingConfidenceHigh},
		{Slot: securityAlternativeSlot(step), Value: "bearer", Source: mappingSourceUser, Confidence: mappingConfidenceLow},
		{Slot: securityAlternativeSlot(step), Value: "bearer", Source: mappingSourceUser, Confidence: mappingConfidenceHigh, RequiresConfirmation: true},
	} {
		session := Session{DecisionEvidence: []DecisionEvidence{evidence}}
		session.Interview.Metadata = map[string]string{securityAlternativeMetadataKey(step): "2"}
		if _, ok := selectedSecurityAlternative(session, step, op); ok {
			t.Fatalf("legacy selection accepted unconfirmed evidence: %#v", evidence)
		}
	}
}
