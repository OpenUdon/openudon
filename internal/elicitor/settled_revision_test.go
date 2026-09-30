package elicitor

import (
	"reflect"
	"testing"

	publicinterview "github.com/OpenUdon/authoring/interview"
	"github.com/OpenUdon/openudon/internal/authoring"
	"github.com/OpenUdon/openudon/internal/projectwizard"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

func TestReopenSettledDecisionPersistsPendingActorUntilReplacement(t *testing.T) {
	const outcome = "Render the reviewed capability report"
	const original = "operator | on demand"
	session := Session{
		Boundary: WorkflowBoundary{Outcome: outcome, Actor: "operator", Trigger: "on demand", SuccessEvidence: []string{"report exists"}, Confirmed: true},
		Project:  projectwizard.Answers{ProjectName: "report", Goal: outcome},
		Intent:   rollout.Intent{Workflow: &rollout.WorkflowMeta{Name: "report", Description: outcome}},
		Interview: publicinterview.State{
			Nodes:   []publicinterview.Node{{ID: nodeActorTrigger, Prompt: "Who starts it?", Status: publicinterview.StatusSettled}},
			Answers: []publicinterview.Answer{{ID: "answer.001." + nodeActorTrigger, NodeID: nodeActorTrigger, Value: original, Source: "user"}},
			Round:   1,
		},
	}
	session.Normalize()
	decisions := BuildRevisableDecisions(session)
	if len(decisions) != 1 || decisions[0].QuestionID != nodeActorTrigger || decisions[0].Value != original {
		t.Fatalf("revisable decisions = %#v", decisions)
	}

	if err := ReopenSettledDecision(&session, nodeActorTrigger, nil); err != nil {
		t.Fatal(err)
	}
	if !HasPendingRevision(session) || session.Boundary.Actor != "" || session.Boundary.Trigger != "" || session.Boundary.Outcome != outcome {
		t.Fatalf("reopened state was repopulated: %#v / %#v / %#v", session.Boundary, session.Project, session.Intent.Workflow)
	}
	if len(BuildRevisableDecisions(session)) != 0 {
		t.Fatalf("open decision remained revisable: %#v", BuildRevisableDecisions(session))
	}
	data, persists, err := DraftBytes(session)
	if err != nil || !persists {
		t.Fatalf("draft bytes = persists %t, error %v", persists, err)
	}
	resumed, err := DecodeSession(data, ".yaml")
	if err != nil {
		t.Fatal(err)
	}
	if !HasPendingRevision(resumed) || resumed.Boundary.Actor != "" || resumed.Boundary.Trigger != "" || resumed.Boundary.Outcome != outcome {
		t.Fatalf("resumed pending revision = %#v", resumed)
	}
	frontier, err := PlanFrontier(&resumed, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	foundOutcome := false
	for _, question := range frontier {
		foundOutcome = foundOutcome || question.ID == nodeActorTrigger
	}
	if !foundOutcome {
		t.Fatalf("replacement frontier = %#v", frontier)
	}
	const replacement = "reviewer | after approval"
	answers := make([]authoring.RoundAnswer, 0, len(frontier))
	for _, question := range frontier {
		value := question.Recommendation
		if question.ID == nodeActorTrigger {
			value = replacement
		}
		if value == "" {
			value = "reviewed answer"
		}
		answers = append(answers, authoring.RoundAnswer{QuestionID: question.ID, Slots: question.Slots, Value: value, Source: "user"})
	}
	if err := ApplyFrontierRound(&resumed, answers, nil); err != nil {
		t.Fatal(err)
	}
	if HasPendingRevision(resumed) || resumed.Boundary.Actor != "reviewer" || resumed.Boundary.Trigger != "after approval" || resumed.Boundary.Outcome != outcome {
		t.Fatalf("replacement did not settle revision: %#v", resumed)
	}
}

func TestReopenSettledDecisionRejectsIneligibleAndNonHumanState(t *testing.T) {
	session := Session{Interview: publicinterview.State{
		Nodes: []publicinterview.Node{
			{ID: nodeActorTrigger, Status: publicinterview.StatusSettled},
			{ID: nodeBoundaryOutcome, Status: publicinterview.StatusSettled},
			{ID: "source.selection", Status: publicinterview.StatusSettled},
		},
		Answers: []publicinterview.Answer{
			{NodeID: nodeActorTrigger, Value: "operator | on demand", Source: "user"},
			{NodeID: nodeActorTrigger, Value: "model replacement", Source: "model"},
			{NodeID: nodeBoundaryOutcome, Value: "new outcome", Source: "user"},
			{NodeID: "source.selection", Value: "api", Source: "user"},
		},
	}}
	before := session
	before.Interview.Nodes = append([]publicinterview.Node(nil), session.Interview.Nodes...)
	before.Interview.Answers = append([]publicinterview.Answer(nil), session.Interview.Answers...)
	if decisions := BuildRevisableDecisions(session); len(decisions) != 0 {
		// source.selection is deliberately excluded even when human-authored;
		// the later model answer also excludes actor/trigger.
		t.Fatalf("unexpected revisable decisions = %#v", decisions)
	}
	if err := ReopenSettledDecision(&session, nodeActorTrigger, nil); err == nil {
		t.Fatal("non-human latest answer was accepted")
	}
	if err := ReopenSettledDecision(&session, "source.selection", nil); err == nil {
		t.Fatal("ineligible source selection was accepted")
	}
	if err := ReopenSettledDecision(&session, nodeBoundaryOutcome, nil); err == nil {
		t.Fatal("outcome without a safe dependent-state cascade was accepted")
	}
	if !reflect.DeepEqual(session, before) {
		t.Fatalf("rejected reopen mutated session: before %#v after %#v", before, session)
	}
}

func TestSelectedBrowserRegistrySourceMakesPolicyNonRevisable(t *testing.T) {
	session := Session{
		SourcePlan: []SourceMaterialization{{Kind: "browser-profile", ID: "status", Registry: "https://profiles.example.test"}},
		Interview: publicinterview.State{
			Nodes:   []publicinterview.Node{{ID: nodeBrowserRegistry, Status: publicinterview.StatusSettled}},
			Answers: []publicinterview.Answer{{ID: "answer.001." + nodeBrowserRegistry, NodeID: nodeBrowserRegistry, Value: "allow", Source: "user"}},
		},
	}
	if decisions := BuildRevisableDecisions(session); len(decisions) != 0 {
		t.Fatalf("selected registry policy was advertised as revisable: %#v", decisions)
	}
	if err := ReopenSettledDecision(&session, nodeBrowserRegistry, nil); err == nil {
		t.Fatal("selected registry policy was reopened without a revocation cascade")
	}
}

func TestReopenSideEffectPostureSurvivesNormalization(t *testing.T) {
	session := Session{
		Boundary: WorkflowBoundary{Outcome: "Render a report", Actor: "operator", Trigger: "on demand", SuccessEvidence: []string{"report exists"}, Confirmed: true},
		Project: projectwizard.Answers{
			ProjectName: "report", Goal: "Render a report", Safety: "Production execution requires approval.", SideEffectScope: projectwizard.SideEffectAfterApproval,
		},
		Intent:          rollout.Intent{Workflow: &rollout.WorkflowMeta{Name: "report", Description: "Render a report"}},
		Safety:          "Production execution requires approval.",
		SafetySet:       true,
		SideEffectScope: projectwizard.SideEffectAfterApproval,
		Interview: publicinterview.State{
			Nodes:   []publicinterview.Node{{ID: nodeSideEffectPosture, Prompt: "Choose side-effect posture", Status: publicinterview.StatusSettled}},
			Answers: []publicinterview.Answer{{ID: "answer.1000." + nodeSideEffectPosture, NodeID: nodeSideEffectPosture, Value: projectwizard.SideEffectAfterApproval, Source: "user"}},
			Round:   1000,
		},
		DecisionEvidence: []DecisionEvidence{{Stage: decisionStageSideEffect, Slot: "side_effect_scope", Value: projectwizard.SideEffectAfterApproval, Source: "user"}},
	}
	session.Normalize()
	if err := ReopenSettledDecision(&session, nodeSideEffectPosture, nil); err != nil {
		t.Fatal(err)
	}
	if session.SideEffectScope != "" || session.Project.SideEffectScope != "" || session.Safety != "" || !HasPendingRevision(session) || len(session.DecisionEvidence) != 0 {
		t.Fatalf("side-effect posture was restored after reopen: %#v", session)
	}
	frontier, err := PlanFrontier(&session, nil, CheckReadiness(session, nil))
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, question := range frontier {
		found = found || question.ID == nodeSideEffectPosture
	}
	if !found {
		t.Fatalf("side-effect replacement question missing: %#v", frontier)
	}
}

func TestLatestHumanAnswerUsesNumericRoundOrder(t *testing.T) {
	answers := []publicinterview.Answer{
		{ID: "answer.1000." + nodeActorTrigger, NodeID: nodeActorTrigger, Value: "reviewer | after approval", Source: "user"},
		{ID: "answer.999." + nodeActorTrigger, NodeID: nodeActorTrigger, Value: "model | earlier", Source: "model"},
	}
	latest := latestAnswersByNode(answers)[nodeActorTrigger]
	if latest.Value != "reviewer | after approval" || !hasHumanAnswer(answers, nodeActorTrigger) {
		t.Fatalf("latest answer = %#v", latest)
	}
}

func TestReopenedDeferrableDecisionAcceptsExplicitDeferral(t *testing.T) {
	session := Session{
		Boundary: WorkflowBoundary{Outcome: "Render a report", Actor: "operator", Trigger: "on demand", SuccessEvidence: []string{"report exists"}, Confirmed: true},
		Project:  projectwizard.Answers{ProjectName: "report", Goal: "Render a report", Fallback: "stop cleanly"},
		Intent:   rollout.Intent{Workflow: &rollout.WorkflowMeta{Name: "report", Description: "Render a report"}},
		Fallback: "stop cleanly", FallbackSet: true,
		Interview: publicinterview.State{
			Nodes:   []publicinterview.Node{{ID: nodeFallback, Prompt: "What happens on failure?", Status: publicinterview.StatusSettled, Deferrable: true}},
			Answers: []publicinterview.Answer{{ID: "answer.001." + nodeFallback, NodeID: nodeFallback, Value: "stop cleanly", Source: "user"}},
			Round:   1,
		},
	}
	session.Normalize()
	if err := ReopenSettledDecision(&session, nodeFallback, nil); err != nil {
		t.Fatal(err)
	}
	deferred := false
	for round := 0; round < 5 && HasPendingRevision(session); round++ {
		frontier, err := PlanFrontier(&session, nil, CheckReadiness(session, nil))
		if err != nil {
			t.Fatal(err)
		}
		var answers []authoring.RoundAnswer
		for _, question := range frontier {
			value := question.Recommendation
			if question.ID == nodeFallback {
				value = "defer:operator | draft remains incomplete | failure policy is reviewed | replace the fallback"
				deferred = true
			} else if value == "" {
				value = "reviewed answer"
			}
			answers = append(answers, authoring.RoundAnswer{QuestionID: question.ID, Slots: question.Slots, Value: value, Source: "user"})
		}
		if err := ApplyFrontierRound(&session, answers, nil); err != nil {
			t.Fatal(err)
		}
	}
	if !deferred || HasPendingRevision(session) || len(session.Interview.Deferrals) != 1 || session.Interview.Deferrals[0].NodeID != nodeFallback {
		t.Fatalf("deferral did not replace reopened answer: %#v", session.Interview)
	}
}
