package elicitor

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/OpenUdon/apitools"
	"github.com/OpenUdon/openudon/internal/projectwizard"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

func TestRunProgressiveRejectsIncompleteLocalDiscovery(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "service.json"), []byte(`{"openapi":"3.0.3","info":{"title":"Service","version":"1"},"paths":{"/items":{"get":{"operationId":"listItems"}}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "ambiguous.json"), []byte(`{"name":"unknown","methods":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := runProgressive(context.Background(), nil, io.Discard, Session{}, Options{
		NoLLM: true, SourceRoots: []string{root}, NetworkPolicy: "ask",
	})
	if err == nil || !strings.Contains(err.Error(), "local source discovery is incomplete") {
		t.Fatalf("runProgressive error = %v", err)
	}
}

func TestLocalSourceDiscoveryBlockerRejectsTruncation(t *testing.T) {
	report := apitools.LocalSourceDiscoveryReport{Truncated: true, Candidates: []apitools.LocalSourceCandidate{{ID: "usable"}}}
	if err := localSourceDiscoveryBlocker(report); err == nil || !strings.Contains(err.Error(), "truncated=true") {
		t.Fatalf("blocker error = %v", err)
	}
}

func TestProgressiveSessionPreservesNetworkPolicyAfterDiscovery(t *testing.T) {
	session := progressiveSessionAfterDiscovery(Session{}, nil, nil, "ask")
	if got := session.Interview.Metadata["network_policy"]; got != "ask" {
		t.Fatalf("network policy = %q", got)
	}
	session.Boundary = WorkflowBoundary{Outcome: "Fetch a report", Actor: "operator", Trigger: "on demand", SuccessEvidence: []string{"report returned"}}
	session.Intent = rollout.Intent{Workflow: &rollout.WorkflowMeta{Name: "fetch_report", Description: "Fetch a report"}, OpenAPI: "openapi/report.yaml"}
	session.Fallback, session.FallbackSet = "stop cleanly", true
	session.SideEffectScope = projectwizard.SideEffectReadOnly
	frontier, err := PlanFrontier(&session, nil, []ReadinessIssue{{Code: "missing_api_doc", Severity: readinessBlocking, Slot: "source.selection", Message: "source required"}})
	if err != nil {
		t.Fatal(err)
	}
	if !hasQuestionID(frontier, nodeRemoteLookup) {
		t.Fatalf("remote lookup approval missing from frontier: %#v", frontier)
	}
}

func TestSessionResumeRestoresSafetyEvidenceFromUnifiedLedger(t *testing.T) {
	session := Session{
		Project: projectwizard.Answers{Goal: "send a report"},
		Intent: rollout.Intent{
			Workflow: &rollout.WorkflowMeta{Name: "send_report", Description: "send a report"},
			Source:   "openapi/mail.json",
			Steps:    []*rollout.Step{{Name: "send", Type: "http", Provider: "mail", OpenAPI: "openapi/mail.json", Operation: "sendMessage", Do: "Send the report."}},
		},
	}
	addMappingClassification(&session, MappingClassification{
		Slot: "steps.send.operation", Value: "sendMessage", Source: mappingSourceLLM,
		Confidence: mappingConfidenceReview, Reason: "inferred from the request", RequiresConfirmation: true,
	})
	path := filepath.Join(t.TempDir(), "session.yaml")
	if err := SaveDraft(path, session); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "classifications:") || !strings.Contains(string(data), "requires_confirmation") || !strings.Contains(string(data), "openudon.record") {
		t.Fatalf("session did not use the unified durable evidence ledger:\n%s", data)
	}
	loaded, ok, err := LoadDraft(path)
	if err != nil || !ok {
		t.Fatalf("LoadDraft ok=%t error=%v", ok, err)
	}
	if len(loaded.Classifications) == 0 || loaded.Classifications[0].Confidence != mappingConfidenceReview || !loaded.Classifications[0].RequiresConfirmation {
		t.Fatalf("restored classifications = %#v", loaded.Classifications)
	}
	if len(loaded.DecisionEvidence) == 0 || !loaded.DecisionEvidence[0].RequiresConfirmation {
		t.Fatalf("restored decision evidence = %#v", loaded.DecisionEvidence)
	}
	docs := []APIDocument{{RelativePath: "openapi/mail.json", Title: "Mail", Operations: []apitools.OperationSummary{{OperationID: "sendMessage", Method: "POST", Summary: "Send a message."}}}}
	if !hasReadinessCode(CheckReadiness(loaded, docs), readinessUnconfirmedSideEffectCommitment) {
		t.Fatalf("resumed session lost side-effect confirmation blocker: %#v", CheckReadiness(loaded, docs))
	}
}

func TestSourcePlanReusesIdenticalTargetsAndRejectsConflicts(t *testing.T) {
	digestA := strings.Repeat("a", 64)
	digestB := strings.Repeat("b", 64)
	identical := Session{SourcePlan: []SourceMaterialization{
		{Kind: "openapi", ID: "one", SourcePath: "/tmp/one.json", TargetPath: "openapi/service.json", SHA256: digestA, Provenance: "local:one"},
		{Kind: "openapi", ID: "two", SourcePath: "/tmp/two.json", TargetPath: "openapi/service.json", SHA256: digestA, Provenance: "local:two"},
	}}
	identical.Normalize()
	if len(identical.SourcePlan) != 1 {
		t.Fatalf("identical target plan = %#v", identical.SourcePlan)
	}
	conflicting := Session{SourcePlan: []SourceMaterialization{
		{Kind: "openapi", ID: "one", SourcePath: "/tmp/one.json", TargetPath: "openapi/service.json", SHA256: digestA, Provenance: "local:one"},
		{Kind: "openapi", ID: "two", SourcePath: "/tmp/two.json", TargetPath: "openapi/service.json", SHA256: digestB, Provenance: "local:two"},
	}}
	conflicting.Normalize()
	if err := validateV2State(conflicting); err == nil || !strings.Contains(err.Error(), "conflicting SHA-256 digests") {
		t.Fatalf("validateV2State error = %v", err)
	}
}
