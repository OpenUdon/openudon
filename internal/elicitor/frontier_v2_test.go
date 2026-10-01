package elicitor

import (
	"reflect"
	"testing"

	publicinterview "github.com/OpenUdon/authoring/interview"
	"github.com/OpenUdon/openudon/internal/authoring"
	"github.com/OpenUdon/openudon/internal/projectdoc"
	"github.com/OpenUdon/openudon/internal/projectwizard"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

func TestV2DraftMergeReachesDeterministicFrontier(t *testing.T) {
	example := t.TempDir()
	writeOpenAPI(t, example)
	docs, err := DiscoverLocalAPIs(example, "Fetch a support ticket.")
	if err != nil {
		t.Fatal(err)
	}
	base := Session{}
	applyProgressiveAnswer(&base, QuestionPlan{Slots: []string{"workflow.goal"}}, "Fetch a support ticket.", docs)
	base.Normalize()
	deterministicPrefill(&base, docs)
	draft := supportTicketDraft(false)
	merged := mergeProgressiveSessions(base, draft, docs)
	initialIssues := CheckReadiness(merged, docs)
	initialFrontier, err := PlanFrontier(&merged, docs, initialIssues)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("initial frontier=%#v issues=%#v", initialFrontier, initialIssues)
	request := BuildRequestMappingRequest("Fetch a support ticket.", merged, docs, initialIssues, QuestionPlan{})
	applyRequestMappingResponse(&merged, request, RequestMappingResponse{Steps: []RequestMappingStepResponse{{Name: "get_ticket", With: map[string]string{"ticketId": "inputs.ticketId"}}}})
	deterministicPrefill(&merged, docs)
	issues := CheckReadiness(merged, docs)
	frontier, err := PlanFrontier(&merged, docs, issues)
	if err != nil {
		t.Fatal(err)
	}
	if _, renderErr := RenderArtifacts(merged); renderErr != nil {
		t.Logf("render error=%v frontier=%#v issues=%#v", renderErr, frontier, issues)
		if len(frontier) == 0 {
			t.Fatalf("render error %v with no frontier; issues=%#v session=%#v", renderErr, issues, merged)
		}
	}
}

func TestCloneSessionDeepCopiesDraftState(t *testing.T) {
	original := Session{
		DraftOperations: []OperationDetailRef{{DocumentPath: "openapi/items.yaml", OperationID: "getItem"}},
		DraftEvents: []TranscriptEvent{{Kind: "draft", Data: map[string]any{
			"operations": []any{"getItem"}, "detail": map[string]any{"status": "ready"},
		}}},
	}
	clone, err := cloneSession(original)
	if err != nil {
		t.Fatal(err)
	}
	clone.DraftOperations[0].OperationID = "changed"
	cloneData := clone.DraftEvents[0].Data.(map[string]any)
	cloneData["operations"].([]any)[0] = "changed"
	cloneData["detail"].(map[string]any)["status"] = "changed"
	if original.DraftOperations[0].OperationID != "getItem" {
		t.Fatal("DraftOperations shared storage with clone")
	}
	want := map[string]any{"operations": []any{"getItem"}, "detail": map[string]any{"status": "ready"}}
	if !reflect.DeepEqual(original.DraftEvents[0].Data, want) {
		t.Fatalf("DraftEvents shared nested storage with clone: %#v", original.DraftEvents)
	}
}

func TestMergeProgressiveSessionsDeepCopiesDraftState(t *testing.T) {
	base := Session{
		DraftOperations: []OperationDetailRef{{DocumentPath: "openapi/base.yaml", OperationID: "base"}},
		DraftEvents:     []TranscriptEvent{{Kind: "base", Data: map[string]any{"nested": map[string]any{"status": "base"}}}},
	}
	overlay := Session{
		DraftOperations: []OperationDetailRef{{DocumentPath: "openapi/overlay.yaml", OperationID: "overlay"}},
		DraftEvents:     []TranscriptEvent{{Kind: "overlay", Data: map[string]any{"nested": map[string]any{"status": "overlay"}}}},
	}
	merged := mergeProgressiveSessions(base, overlay, nil)
	merged.DraftOperations[0].OperationID = "changed"
	merged.DraftEvents[0].Data.(map[string]any)["nested"].(map[string]any)["status"] = "changed"
	merged.DraftEvents[1].Data.(map[string]any)["nested"].(map[string]any)["status"] = "changed"
	if base.DraftOperations[0].OperationID != "base" || base.DraftEvents[0].Data.(map[string]any)["nested"].(map[string]any)["status"] != "base" {
		t.Fatalf("base draft state shared storage with merge: %#v", base)
	}
	if overlay.DraftEvents[0].Data.(map[string]any)["nested"].(map[string]any)["status"] != "overlay" {
		t.Fatalf("overlay draft events shared storage with merge: %#v", overlay.DraftEvents)
	}
}

func TestPlanFrontierIncludesWholeReadyBoundaryRound(t *testing.T) {
	session := supportTicketDraft(true)
	session.Boundary.Actor = ""
	session.Boundary.Trigger = ""
	session.Boundary.SuccessEvidence = nil
	session.Boundary.Confirmed = false
	frontier, err := PlanFrontier(&session, nil, CheckReadiness(session, nil))
	if err != nil {
		t.Fatal(err)
	}
	if !hasQuestionID(frontier, nodeActorTrigger) || !hasQuestionID(frontier, nodeSuccessEvidence) {
		t.Fatalf("boundary frontier = %#v", frontier)
	}
}

func TestPlanFrontierForcesActiveWorkflowSelection(t *testing.T) {
	session := Session{CandidateWorkflows: []CandidateWorkflow{{Title: "Notify", Outcome: "Notify the operator", DeferralReason: "reporting is active", PromotionTrigger: "reporting is complete"}}}
	session.Boundary.Outcome = "Build the report"
	frontier, err := PlanFrontier(&session, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, question := range frontier {
		if question.ID == nodeActiveWorkflow {
			if !question.Forced {
				t.Fatal("active workflow selection was not forced")
			}
			return
		}
	}
	t.Fatalf("active workflow question missing: %#v", frontier)
}

func TestApplyFrontierRoundRecordsStructuredTechnicalDeferral(t *testing.T) {
	session := Session{}
	session.Boundary = WorkflowBoundary{Outcome: "Fetch a report", Actor: "operator", Trigger: "on demand", SuccessEvidence: []string{"report is returned"}}
	session.Intent.Workflow = &rollout.WorkflowMeta{Name: "fetch_report", Description: "Fetch a report"}
	session.Intent.OpenAPI = "openapi/report.yaml"
	session.Fallback, session.FallbackSet = "stop cleanly and report the failure", true
	session.SideEffectScope = projectwizard.SideEffectReadOnly
	issues := []ReadinessIssue{{Code: "missing_api_doc", Slot: "intent.source", Severity: readinessBlocking, Message: "source required"}}
	frontier, err := PlanFrontier(&session, nil, issues)
	if err != nil {
		t.Fatal(err)
	}
	var source QuestionPlan
	for _, question := range frontier {
		if question.ID == nodeSourceSelection {
			source = question
		}
	}
	if source.ID == "" {
		t.Fatalf("source question missing: %#v", frontier)
	}
	err = ApplyFrontierRound(&session, []authoring.RoundAnswer{{QuestionID: source.ID, Slots: source.Slots, Value: "defer: API owner | implementation blocked | provider publishes a spec | add --api-source", Source: "user"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(session.Interview.Deferrals) != 1 || session.Interview.Deferrals[0].NodeID != nodeSourceSelection {
		t.Fatalf("deferrals = %#v", session.Interview.Deferrals)
	}
	for _, node := range session.Interview.Nodes {
		if node.ID == nodeSourceSelection && node.Status != publicinterview.StatusDeferred {
			t.Fatalf("source status = %q", node.Status)
		}
	}
}

func TestCandidateWorkflowProjectRecordHasNoTechnicalLeaves(t *testing.T) {
	candidate := projectdoc.CandidateWorkflow{Title: "Notify", Outcome: "Notify the operator", DeferralReason: "reporting is active", PromotionTrigger: "reporting is complete"}
	if candidate.Title == "" || candidate.Outcome == "" || candidate.DeferralReason == "" || candidate.PromotionTrigger == "" {
		t.Fatal("candidate boundary is incomplete")
	}
}

func hasQuestionID(questions []QuestionPlan, id string) bool {
	for _, question := range questions {
		if question.ID == id {
			return true
		}
	}
	return false
}
