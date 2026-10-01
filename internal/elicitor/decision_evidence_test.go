package elicitor

import (
	"encoding/json"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestDecisionEvidenceNormalizesAndDedupes(t *testing.T) {
	session := Session{}
	addDecisionEvidence(&session, DecisionEvidence{
		Stage:      "Catalog Plan",
		Slot:       "intent.source",
		Value:      "google-discovery/gmail.json",
		Source:     mappingSourceLLM,
		Confidence: mappingConfidenceReview,
		Reason:     "selected from validated catalog metadata",
		Evidence:   "gmail discovery",
	})
	addDecisionEvidence(&session, DecisionEvidence{
		Stage:      "catalog_plan",
		Slot:       "intent.source",
		Value:      "google-discovery/gmail.json",
		Source:     mappingSourceLLM,
		Confidence: mappingConfidenceReview,
	})
	if len(session.DecisionEvidence) != 1 {
		t.Fatalf("decision evidence count = %d, want 1: %#v", len(session.DecisionEvidence), session.DecisionEvidence)
	}
	if got := session.DecisionEvidence[0].Stage; got != "catalog_plan" {
		t.Fatalf("stage = %q", got)
	}
}

func TestDecisionEvidenceReadinessSkipsClassificationDuplicate(t *testing.T) {
	session := Session{}
	addMappingClassification(&session, MappingClassification{
		Slot:       "steps.send.with.raw",
		Value:      "gmail.received_body",
		Source:     mappingSourceLLM,
		Confidence: mappingConfidenceLow,
	})
	issues := decisionEvidenceIssues(session)
	if len(issues) != 0 {
		t.Fatalf("decision evidence duplicated mapping issue: %#v", issues)
	}
}

func TestLowDecisionEvidenceForcesQuestion(t *testing.T) {
	session := Session{}
	addDecisionEvidence(&session, DecisionEvidence{
		Stage:      decisionStageCatalogPlan,
		Slot:       "sources.crm",
		Value:      "openapi/crm.yaml",
		Source:     mappingSourceLLM,
		Confidence: mappingConfidenceLow,
	})
	issues := decisionEvidenceIssues(session)
	issue := readinessIssue(issues, "low_confidence_decision")
	if issue.Code == "" {
		t.Fatalf("missing low confidence decision issue: %#v", issues)
	}
	plan := PlanNextQuestion(session, nil, issues)
	if !plan.ForceAsk {
		t.Fatalf("low confidence decision did not force ask: %#v", plan)
	}
	applyProgressiveAnswer(&session, plan, plan.SuggestedAnswer, nil)
	if issues := decisionEvidenceIssues(session); len(issues) != 0 {
		t.Fatalf("user confirmation did not resolve decision evidence issue: %#v", issues)
	}
}

func TestDecisionEvidenceUsesSharedConflictBehaviorAndPreservesReasonWireField(t *testing.T) {
	session := Session{}
	addDecisionEvidence(&session, DecisionEvidence{
		Stage:      decisionStageOperation,
		Slot:       "steps.gmail.operation",
		Value:      "gmail_users_messages_send",
		Source:     mappingSourceLLM,
		Confidence: mappingConfidenceHigh,
		Reason:     "ranked from local operation metadata",
	})
	addDecisionEvidence(&session, DecisionEvidence{
		Stage:      decisionStageOperation,
		Slot:       "steps.gmail.operation",
		Value:      "gmail_users_drafts_create",
		Source:     mappingSourceDeterministic,
		Confidence: mappingConfidenceReview,
		Reason:     "fallback candidate",
	})
	if len(session.DecisionEvidence) != 2 {
		t.Fatalf("decision evidence = %#v, want two conflict records", session.DecisionEvidence)
	}
	for _, evidence := range session.DecisionEvidence {
		if evidence.Confidence != mappingConfidenceConflict || !evidence.RequiresConfirmation || len(evidence.Alternatives) == 0 {
			t.Fatalf("evidence = %#v, want shared conflict behavior", evidence)
		}
	}
	data, err := json.Marshal(session.DecisionEvidence[0])
	if err != nil {
		t.Fatalf("json marshal: %v", err)
	}
	if !strings.Contains(string(data), `"reason"`) || strings.Contains(string(data), `"rationale"`) {
		t.Fatalf("json = %s, want OpenUdon reason compatibility", data)
	}
	yamlData, err := yaml.Marshal(session.DecisionEvidence[0])
	if err != nil {
		t.Fatalf("yaml marshal: %v", err)
	}
	if !strings.Contains(string(yamlData), "reason:") || strings.Contains(string(yamlData), "rationale:") {
		t.Fatalf("yaml = %s, want OpenUdon reason compatibility", yamlData)
	}
}
