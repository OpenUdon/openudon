package elicitor

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	publicinterview "github.com/OpenUdon/authoring/interview"
	"github.com/OpenUdon/openudon/internal/authoring"
)

// RevisableDecision is one operator-provided settled answer that can be
// explicitly reopened without guessing how imported or model-derived state was
// produced.
type RevisableDecision struct {
	QuestionID string   `json:"question_id"`
	Prompt     string   `json:"prompt"`
	Slots      []string `json:"slots,omitempty"`
	Value      string   `json:"value"`
	Impact     string   `json:"impact"`
}

const revisionPendingMetadataPrefix = "revision_pending."

// HasPendingRevision reports whether an explicit reopen still requires a
// replacement complete-frontier round.
func HasPendingRevision(session Session) bool {
	for key, value := range session.Interview.Metadata {
		if strings.HasPrefix(key, revisionPendingMetadataPrefix) && value == "true" {
			return true
		}
	}
	return false
}

// BuildRevisableDecisions returns only settled nodes with a durable human
// answer and a product-owned clearing rule.
func BuildRevisableDecisions(session Session) []RevisableDecision {
	latest := latestAnswersByNode(session.Interview.Answers)
	decisions := make([]RevisableDecision, 0, len(latest))
	for _, node := range session.Interview.Nodes {
		answer, ok := latest[node.ID]
		if !ok || strings.TrimSpace(answer.Source) != "user" || node.Status != publicinterview.StatusSettled || !reopenableDecision(node.ID) {
			continue
		}
		decisions = append(decisions, RevisableDecision{
			QuestionID: node.ID,
			Prompt:     firstNonEmpty(node.Prompt, node.Title, node.ID),
			Slots:      settledDecisionSlots(node.ID),
			Value:      answer.Value,
			Impact:     settledDecisionImpact(node.ID),
		})
	}
	sort.SliceStable(decisions, func(i, j int) bool { return decisions[i].QuestionID < decisions[j].QuestionID })
	return decisions
}

// ReopenSettledDecision clears one eligible answer's authoritative state. A
// later ordinary complete frontier round records the replacement value.
func ReopenSettledDecision(session *Session, questionID string, docs []APIDocument) error {
	questionID = strings.TrimSpace(questionID)
	if session == nil {
		return authoring.WithQuestionID(questionID, errors.New("authoring session is required"))
	}
	if !reopenableDecision(questionID) {
		return authoring.WithQuestionID(questionID, fmt.Errorf("decision %q is not eligible for browser revision", questionID))
	}
	nodeIndex := -1
	for index, node := range session.Interview.Nodes {
		if node.ID == questionID {
			nodeIndex = index
			if node.Status != publicinterview.StatusSettled {
				return authoring.WithQuestionID(questionID, fmt.Errorf("decision %q is not currently settled", questionID))
			}
			break
		}
	}
	if nodeIndex < 0 || !hasHumanAnswer(session.Interview.Answers, questionID) {
		return authoring.WithQuestionID(questionID, fmt.Errorf("decision %q has no revisable human answer", questionID))
	}
	if err := clearSettledDecision(session, questionID, docs); err != nil {
		return authoring.WithQuestionID(questionID, err)
	}
	if session.Interview.Metadata == nil {
		session.Interview.Metadata = map[string]string{}
	}
	session.Interview.Metadata[revisionPendingMetadataPrefix+questionID] = "true"
	session.Interview.Nodes[nodeIndex].Status = publicinterview.StatusOpen
	session.Interview.Deferrals = removeDeferralForNode(session.Interview.Deferrals, questionID)
	session.Interview.Round++
	session.Interview.NoProgressRounds = 0
	session.Interview.Evidence = append(session.Interview.Evidence, publicinterview.Evidence{
		ID:   "evidence.reopen." + fmt.Sprintf("%03d", session.Interview.Round) + "." + questionID,
		Kind: publicinterview.EvidenceUserDecision, NodeID: questionID,
		Summary: "The operator explicitly reopened this settled decision for revision.",
		Source:  "user",
	})
	session.Normalize()
	return nil
}

func reopenableDecision(questionID string) bool {
	switch questionID {
	case nodeBoundaryOutcome, nodeActiveWorkflow, nodeActorTrigger, nodeSuccessEvidence,
		nodeRemoteLookup, nodeBrowserRegistry, nodeSideEffectPosture, nodeBrowserSession,
		nodeBrowserApproval, nodeFallback, nodeVerification:
		return true
	default:
		return strings.HasPrefix(questionID, "security.alternative.")
	}
}

func settledDecisionSlots(questionID string) []string {
	switch questionID {
	case nodeBoundaryOutcome:
		return []string{"boundary.outcome"}
	case nodeActiveWorkflow:
		return []string{"boundary.active_workflow"}
	case nodeActorTrigger:
		return []string{"boundary.actor", "boundary.trigger"}
	case nodeSuccessEvidence, nodeVerification:
		return []string{"boundary.success_evidence"}
	case nodeRemoteLookup:
		return []string{"source.remote_lookup"}
	case nodeBrowserRegistry:
		return []string{"source.browser_registry_lookup"}
	case nodeSideEffectPosture:
		return []string{"safety"}
	case nodeBrowserSession:
		return []string{"browser.session_posture"}
	case nodeBrowserApproval:
		return []string{"browser.mutation_approval"}
	case nodeFallback:
		return []string{"fallback"}
	default:
		if strings.HasPrefix(questionID, "security.alternative.") {
			return []string{"steps." + strings.TrimPrefix(questionID, "security.alternative.") + ".security_alternative"}
		}
		return nil
	}
}

func settledDecisionImpact(questionID string) string {
	switch questionID {
	case nodeBoundaryOutcome, nodeActiveWorkflow:
		return "Reopening the active boundary may invalidate source, operation, mapping, and preview decisions. The engine will re-run readiness before any replacement is accepted."
	case nodeBrowserSession, nodeBrowserApproval:
		return "Reopening browser posture or approval may make the current proposal incomplete until the exact replacement is reviewed."
	default:
		return "Reopening clears this value and re-runs source discovery, readiness, and preview generation before the replacement round."
	}
}

func clearSettledDecision(session *Session, questionID string, docs []APIDocument) error {
	switch questionID {
	case nodeBoundaryOutcome:
		session.Boundary.Outcome = ""
		session.Project.Goal = ""
		if session.Intent.Workflow != nil {
			session.Intent.Workflow.Description = ""
		}
	case nodeActiveWorkflow:
		delete(session.Interview.Metadata, "active_workflow_selected")
	case nodeActorTrigger:
		session.Boundary.Actor, session.Boundary.Trigger, session.Boundary.Confirmed = "", "", false
	case nodeSuccessEvidence, nodeVerification:
		session.Boundary.SuccessEvidence = nil
		session.Boundary.Confirmed = false
	case nodeRemoteLookup:
		delete(session.Interview.Metadata, "remote_lookup_decision")
	case nodeBrowserRegistry:
		delete(session.Interview.Metadata, "browser_registry_lookup_decision")
	case nodeSideEffectPosture:
		session.SideEffectScope, session.Safety = "", ""
		session.SafetySet = false
		session.Project.SideEffectScope, session.Project.Safety = "", ""
		session.DecisionEvidence = removeDecisionEvidenceForSlot(session.DecisionEvidence, "side_effect_scope")
		session.Interview.Evidence = removeDecisionEvidenceLedgerForSlot(session.Interview.Evidence, "side_effect_scope")
	case nodeBrowserSession:
		session.BrowserSession = ""
	case nodeBrowserApproval:
		step, _, operation := selectedBrowserOperation(*session, docs)
		if step == nil || operation == nil || !browserOperationMutates(operation) {
			return errors.New("the settled browser approval no longer matches an active mutating step")
		}
		session.BrowserApprovals = removeString(session.BrowserApprovals, step.Name)
	case nodeFallback:
		session.Fallback, session.Project.Fallback = "", ""
		session.FallbackSet = false
	default:
		if strings.HasPrefix(questionID, "security.alternative.") {
			for _, step := range session.Intent.Steps {
				if step == nil || "security.alternative."+slugIdent(step.Name) != questionID {
					continue
				}
				delete(session.Interview.Metadata, securityAlternativeMetadataKey(step))
				slot := securityAlternativeSlot(step)
				session.DecisionEvidence = removeDecisionEvidenceForSlot(session.DecisionEvidence, slot)
				session.Interview.Evidence = removeDecisionEvidenceLedgerForSlot(session.Interview.Evidence, slot)
				return nil
			}
			return errors.New("the settled security alternative no longer matches an active step")
		}
		return fmt.Errorf("decision %q is not eligible for browser revision", questionID)
	}
	return nil
}

func hasHumanAnswer(answers []publicinterview.Answer, questionID string) bool {
	answer, ok := latestAnswersByNode(answers)[questionID]
	return ok && strings.TrimSpace(answer.Source) == "user"
}

func latestAnswersByNode(answers []publicinterview.Answer) map[string]publicinterview.Answer {
	type rankedAnswer struct {
		answer publicinterview.Answer
		round  int
		order  int
	}
	latest := map[string]rankedAnswer{}
	for order, answer := range answers {
		nodeID := strings.TrimSpace(answer.NodeID)
		if nodeID == "" {
			continue
		}
		ranked := rankedAnswer{answer: answer, round: answerRound(answer.ID), order: order}
		current, ok := latest[nodeID]
		if !ok || ranked.round > current.round || ranked.round == current.round && ranked.order > current.order {
			latest[nodeID] = ranked
		}
	}
	out := make(map[string]publicinterview.Answer, len(latest))
	for nodeID, ranked := range latest {
		out[nodeID] = ranked.answer
	}
	return out
}

func answerRound(id string) int {
	parts := strings.SplitN(strings.TrimSpace(id), ".", 3)
	if len(parts) != 3 || parts[0] != "answer" {
		return -1
	}
	round, err := strconv.Atoi(parts[1])
	if err != nil || round < 0 {
		return -1
	}
	return round
}

func removeDeferralForNode(deferrals []publicinterview.Deferral, questionID string) []publicinterview.Deferral {
	out := deferrals[:0]
	for _, deferral := range deferrals {
		if deferral.NodeID != questionID {
			out = append(out, deferral)
		}
	}
	return out
}

func removeString(values []string, target string) []string {
	out := values[:0]
	for _, value := range values {
		if value != target {
			out = append(out, value)
		}
	}
	return out
}

func removeDecisionEvidenceForSlot(values []DecisionEvidence, slot string) []DecisionEvidence {
	out := values[:0]
	for _, value := range values {
		if value.Slot != slot {
			out = append(out, value)
		}
	}
	return out
}

func removeDecisionEvidenceLedgerForSlot(values []publicinterview.Evidence, slot string) []publicinterview.Evidence {
	out := values[:0]
	for _, value := range values {
		if value.Attributes[evidenceAttrRecord] == evidenceRecordDecision && value.Attributes[evidenceAttrSlot] == slot {
			continue
		}
		out = append(out, value)
	}
	return out
}

func clearRevisionPending(session *Session, questionID string) {
	if session == nil || session.Interview.Metadata == nil {
		return
	}
	delete(session.Interview.Metadata, revisionPendingMetadataPrefix+questionID)
}

func preservePendingRevisionState(session *Session) {
	if session == nil {
		return
	}
	if session.Interview.Metadata[revisionPendingMetadataPrefix+nodeBoundaryOutcome] == "true" {
		session.Boundary.Outcome = ""
		session.Project.Goal = ""
		if session.Intent.Workflow != nil {
			session.Intent.Workflow.Description = ""
		}
		session.Boundary.Confirmed = false
	}
	if session.Interview.Metadata[revisionPendingMetadataPrefix+nodeSideEffectPosture] == "true" {
		session.SideEffectScope, session.Safety = "", ""
		session.SafetySet = false
		session.Project.SideEffectScope, session.Project.Safety = "", ""
	}
}
