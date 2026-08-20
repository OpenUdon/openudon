package elicitor

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	publicinterview "github.com/OpenUdon/authoring/interview"
	"github.com/OpenUdon/openudon/internal/authoring"
	"github.com/OpenUdon/openudon/internal/projectwizard"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

const (
	nodeBoundaryOutcome     = "boundary.outcome"
	nodeActiveWorkflow      = "boundary.active_workflow"
	nodeActorTrigger        = "boundary.actor_trigger"
	nodeSuccessEvidence     = "boundary.success_evidence"
	nodeRemoteLookup        = "source.remote_lookup"
	nodeBrowserRegistry     = "source.browser_registry_lookup"
	nodeSideEffectPosture   = "safety.side_effect_posture"
	nodeWorkflowSteps       = "workflow.steps"
	nodeSourceSelection     = "source.selection"
	nodeBrowserSession      = "browser.session_posture"
	nodeBrowserApproval     = "browser.mutation_approval"
	nodeCredentials         = "security.credentials"
	nodeOutputs             = "workflow.outputs"
	nodeFallback            = "workflow.fallback"
	nodeVerification        = "workflow.verification"
	nodeFinalIntentApproval = "intent.final_approval"
)

// PlanFrontier returns every independent, dependency-ready question and stores
// the corresponding design tree in the durable v2 session.
func PlanFrontier(session *Session, docs []APIDocument, issues []ReadinessIssue) ([]QuestionPlan, error) {
	if session == nil {
		return nil, nil
	}
	session.Normalize()
	state, questions := buildInterviewState(*session, docs, issues)
	session.Interview = state
	binding := openUdonInterviewBinding(docs)
	binding.Prepare = nil
	binding.Question = func(_ Session, _ []APIDocument, node publicinterview.Node) QuestionPlan {
		question, ok := questions[node.ID]
		if !ok {
			question = QuestionPlan{ID: node.ID, Prompt: node.Prompt, Required: node.Required, Recommendation: node.Recommendation, Priority: node.Priority, Rationale: node.Rationale, EvidenceRefs: node.EvidenceRefs}
		}
		return question
	}
	return binding.Plan(session, docs)
}

func buildInterviewState(session Session, docs []APIDocument, issues []ReadinessIssue) (publicinterview.State, map[string]QuestionPlan) {
	prior := publicinterview.Normalize(session.Interview)
	priorStatus := map[string]string{}
	for _, node := range prior.Nodes {
		priorStatus[node.ID] = node.Status
	}
	questions := map[string]QuestionPlan{}
	var nodes []publicinterview.Node
	add := func(node publicinterview.Node, question QuestionPlan, settled bool) {
		switch {
		case settled:
			node.Status = publicinterview.StatusSettled
		case priorStatus[node.ID] == publicinterview.StatusDeferred:
			node.Status = publicinterview.StatusDeferred
		default:
			node.Status = publicinterview.StatusOpen
		}
		question.ID = node.ID
		question.Required = node.Required
		question.Forced = question.Forced || (node.Required && node.Recommendation == "" && question.Recommendation == "" && question.SuggestedAnswer == "")
		question.Recommendation = firstNonEmpty(question.Recommendation, question.SuggestedAnswer, node.Recommendation)
		question.Priority = node.Priority
		question.Rationale = firstNonEmpty(question.Rationale, node.Rationale)
		question.EvidenceRefs = append([]string(nil), node.EvidenceRefs...)
		node.Prompt = firstNonEmpty(node.Prompt, question.Prompt)
		node.Recommendation = firstNonEmpty(node.Recommendation, question.Recommendation)
		nodes = append(nodes, node)
		questions[node.ID] = question
	}

	outcomeSettled := strings.TrimSpace(session.Boundary.Outcome) != ""
	add(publicinterview.Node{ID: nodeBoundaryOutcome, Title: "Active outcome", Required: true, Priority: 100, Rationale: "One explicit outcome bounds the active workflow."}, QuestionPlan{Prompt: "What single outcome should the active workflow deliver?", Slots: []string{"boundary.outcome"}}, outcomeSettled)
	boundaryRoot := nodeBoundaryOutcome
	if len(session.CandidateWorkflows) > 0 {
		var choices []string
		if session.Boundary.Outcome != "" {
			choices = append(choices, session.Boundary.Outcome)
		}
		for _, candidate := range session.CandidateWorkflows {
			choices = append(choices, candidate.Title+": "+candidate.Outcome)
		}
		add(publicinterview.Node{ID: nodeActiveWorkflow, Title: "Active workflow", Dependencies: []string{nodeBoundaryOutcome}, Required: true, Priority: 99, Rationale: "Only one workflow receives sources, operations, mappings, and implementation detail."}, QuestionPlan{Prompt: "Choose the one active workflow to author: " + strings.Join(choices, " | "), Slots: []string{"boundary.active_workflow"}, Forced: true}, strings.TrimSpace(prior.Metadata["active_workflow_selected"]) != "")
		boundaryRoot = nodeActiveWorkflow
	}
	add(publicinterview.Node{ID: nodeActorTrigger, Title: "Actor and trigger", Dependencies: []string{boundaryRoot}, Required: true, Priority: 90, Recommendation: "operator | on demand", Rationale: "The actor and trigger define when the workflow boundary begins."}, QuestionPlan{Prompt: "Who starts the workflow, and what triggers it? Use actor | trigger.", Slots: []string{"boundary.actor", "boundary.trigger"}}, session.Boundary.Actor != "" && session.Boundary.Trigger != "")
	successRecommendation := "the workflow returns its reviewed output without an unapproved side effect"
	if len(session.Intent.Outputs) > 0 && session.Intent.Outputs[0] != nil {
		successRecommendation = "output " + session.Intent.Outputs[0].Name + " is produced from " + session.Intent.Outputs[0].From
	}
	add(publicinterview.Node{ID: nodeSuccessEvidence, Title: "Success evidence", Dependencies: []string{boundaryRoot}, Required: true, Priority: 89, Recommendation: successRecommendation, Rationale: "Observable evidence makes completion verifiable."}, QuestionPlan{Prompt: "What evidence proves this workflow succeeded?", Slots: []string{"boundary.success_evidence"}}, len(session.Boundary.SuccessEvidence) > 0)

	browserStep, browserDoc, browserOperation := selectedBrowserOperation(session, docs)
	browserActionNode := ""
	if browserStep != nil {
		add(publicinterview.Node{ID: nodeSourceSelection, Title: "Browser source", Dependencies: []string{boundaryRoot}, Required: true, Priority: 75, Rationale: "A verified browser profile is required before selecting one of its reviewed actions."}, QuestionPlan{Prompt: "Select the verified browser profile for the active capability.", Slots: []string{"intent.source"}}, browserDoc.RelativePath != "")
		browserActionNode = "browser.action." + slugIdent(browserStep.Name)
		add(publicinterview.Node{ID: browserActionNode, Title: "Browser action", Dependencies: []string{nodeSourceSelection}, Required: true, Priority: 70, Rationale: "The action must exist in the selected verified profile."}, QuestionPlan{Prompt: "Select a reviewed browser action from the verified profile.", Slots: []string{"steps." + browserStep.Name + ".operation"}}, browserOperation != nil)
	}

	issueMap := map[string]ReadinessIssue{}
	for _, issue := range issues {
		if issue.Severity != readinessBlocking && issue.Code != "missing_side_effect_policy" {
			continue
		}
		if issue.Code == "intent_render_invalid" && (session.Boundary.Actor == "" || session.Boundary.Trigger == "" || len(session.Boundary.SuccessEvidence) == 0 || len(session.CandidateWorkflows) > 0 && prior.Metadata["active_workflow_selected"] == "" || strings.TrimSpace(firstNonEmpty(session.Fallback, session.Project.Fallback)) == "" || projectwizard.NormalizeSideEffectScope(session.SideEffectScope) == "") {
			// Dedicated boundary nodes expose the actionable decisions; the
			// aggregate render error would otherwise ask the same thing twice.
			continue
		}
		issueMap[nodeIDForIssue(issue)] = issue
	}
	missingSource := hasIssueCode(issueMap, "missing_api_doc")
	missingOperation := hasIssueCode(issueMap, "missing_operation") || hasIssueCode(issueMap, readinessUnconfirmedSideEffectCommitment)
	var remoteLookupDependencies []string
	if missingSource && len(docs) == 0 && strings.EqualFold(prior.Metadata["network_policy"], "ask") && prior.Metadata["remote_lookup_decision"] == "" {
		add(publicinterview.Node{ID: nodeRemoteLookup, Title: "Remote API source lookup", Dependencies: []string{boundaryRoot}, Required: true, Priority: 76, Recommendation: "never", Rationale: "Local API evidence is exhausted, so network access requires an explicit decision."}, QuestionPlan{Prompt: "Allow one bounded lookup of curated apitools references and APIs.guru? Answer allow or never.", Slots: []string{"source.remote_lookup"}, Forced: true}, false)
		remoteLookupDependencies = append(remoteLookupDependencies, nodeRemoteLookup)
	}
	if missingSource && len(docs) == 0 && strings.EqualFold(prior.Metadata["network_policy"], "ask") && prior.Metadata["browser_registry_configured"] == "true" && prior.Metadata["browser_registry_lookup_decision"] == "" {
		add(publicinterview.Node{ID: nodeBrowserRegistry, Title: "Static browser registry lookup", Dependencies: []string{boundaryRoot}, Required: true, Priority: 75, Recommendation: "never", Rationale: "A configured static Browsertools registry is a separate remote evidence source and requires its own approval."}, QuestionPlan{Prompt: "Allow one bounded lookup of the configured static Browsertools registries? Answer allow or never.", Slots: []string{"source.browser_registry_lookup"}, Forced: true}, false)
		remoteLookupDependencies = append(remoteLookupDependencies, nodeBrowserRegistry)
	}
	keys := make([]string, 0, len(issueMap))
	for key := range issueMap {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, originalID := range keys {
		issue := issueMap[originalID]
		deps, priority, deferrable := issueDependencies(issue, boundaryRoot, missingSource, missingOperation)
		if deps == nil {
			continue
		}
		id := originalID
		plan := planQuestionForIssue(session, docs, issue)
		switch issue.Code {
		case "missing_side_effect_policy", "unsafe_review_bypass":
			id, deferrable = nodeSideEffectPosture, false
		case "missing_api_doc":
			id = nodeSourceSelection
			if len(remoteLookupDependencies) > 0 {
				deps = append(deps, remoteLookupDependencies...)
			}
		case "missing_credential_bindings", "inline_secret_value":
			id = nodeCredentials
		case "missing_browser_session_posture":
			id, deferrable = nodeBrowserSession, false
			if browserActionNode != "" {
				deps = []string{browserActionNode}
			}
		case "unconfirmed_browser_mutation":
			id, deferrable = nodeBrowserApproval, false
			if browserActionNode != "" {
				deps = []string{browserActionNode}
			}
		case "missing_outputs":
			id = nodeOutputs
			if browserActionNode != "" {
				deps = existingDependencies(nodes, nodeBrowserSession, nodeBrowserApproval, browserActionNode)
			}
		case "missing_operation":
			if issue.Slot == "intent.steps" {
				id = nodeWorkflowSteps
			}
		}
		if browserActionNode != "" && browserStep != nil && strings.Contains(issue.Slot, "steps."+browserStep.Name+".") {
			switch issue.Code {
			case "missing_required_request_values", "conflicting_mapping", "low_confidence_mapping":
				deps = []string{browserActionNode}
			}
		}
		add(publicinterview.Node{ID: id, Title: issue.Code, Dependencies: deps, Required: issue.Severity == readinessBlocking, Deferrable: deferrable, Priority: priority, Rationale: issue.Message, Recommendation: issue.SuggestedAnswer}, plan, false)
	}

	outputDep := ""
	switch {
	case hasNode(nodes, nodeOutputs):
		outputDep = nodeOutputs
	case len(matchingNodeIDs(nodes, "operation.")) > 0:
		ops := matchingNodeIDs(nodes, "operation.")
		outputDep = ops[len(ops)-1]
	case hasNode(nodes, nodeWorkflowSteps):
		outputDep = nodeWorkflowSteps
	}
	if browserActionNode != "" {
		browserDeps := existingDependencies(nodes, nodeBrowserSession, nodeBrowserApproval, browserActionNode)
		if len(browserDeps) > 0 {
			outputDep = ""
		}
		deps := browserDeps
		add(publicinterview.Node{ID: nodeFallback, Title: "Fallback behavior", Dependencies: deps, Deferrable: true, Priority: 20, Recommendation: "stop cleanly and report the failed browser action", Rationale: "Fallback behavior prevents silent partial success."}, QuestionPlan{Prompt: "What should happen when a required browser action fails?", Slots: []string{"fallback"}}, strings.TrimSpace(firstNonEmpty(session.Fallback, session.Project.Fallback)) != "")
		add(publicinterview.Node{ID: nodeVerification, Title: "Verification", Dependencies: deps, Deferrable: true, Priority: 19, Recommendation: successRecommendation, Rationale: "Verification turns the expected result into a reviewable check."}, QuestionPlan{Prompt: "How should the browser result be verified?", Slots: []string{"boundary.success_evidence"}}, len(session.Boundary.SuccessEvidence) > 0)
	}
	if !missingOperation || outputDep != "" {
		deps := existingDependencies(nodes, outputDep)
		if browserActionNode == "" {
			add(publicinterview.Node{ID: nodeFallback, Title: "Fallback behavior", Dependencies: deps, Deferrable: true, Priority: 20, Recommendation: "stop cleanly and report the failed step", Rationale: "Fallback behavior prevents silent partial success."}, QuestionPlan{Prompt: "What should happen when a required step fails?", Slots: []string{"fallback"}}, strings.TrimSpace(firstNonEmpty(session.Fallback, session.Project.Fallback)) != "")
			add(publicinterview.Node{ID: nodeVerification, Title: "Verification", Dependencies: deps, Deferrable: true, Priority: 19, Recommendation: successRecommendation, Rationale: "Verification turns the expected result into a reviewable check."}, QuestionPlan{Prompt: "How should the result be verified?", Slots: []string{"boundary.success_evidence"}}, len(session.Boundary.SuccessEvidence) > 0)
		}
	}

	currentIDs := map[string]bool{}
	for _, node := range nodes {
		currentIDs[node.ID] = true
	}
	for _, historical := range prior.Nodes {
		if currentIDs[historical.ID] {
			continue
		}
		if historical.Status == publicinterview.StatusOpen {
			historical.Status = publicinterview.StatusInapplicable
		}
		nodes = append(nodes, historical)
	}
	prior.Nodes = dedupeInterviewNodes(nodes)
	prior.Deferrals = retainGraphDeferrals(prior.Deferrals, prior.Nodes)
	if prior.Metadata == nil {
		prior.Metadata = map[string]string{}
	}
	return publicinterview.Normalize(prior), questions
}

func issueDependencies(issue ReadinessIssue, boundaryRoot string, missingSource, missingOperation bool) ([]string, int, bool) {
	switch issue.Code {
	case "missing_goal":
		return []string{}, 100, false
	case "missing_side_effect_policy", "unsafe_review_bypass", "inline_secret_value":
		return []string{boundaryRoot}, 80, false
	case "missing_api_doc":
		return []string{boundaryRoot}, 75, true
	case "missing_operation", readinessUnconfirmedSideEffectCommitment:
		if missingSource {
			return nil, 0, false
		}
		return []string{boundaryRoot}, 70, true
	case "missing_security_alternative", "missing_required_request_values", "missing_credential_bindings", "missing_runtime_inputs", "conflicting_mapping", "low_confidence_mapping", "conflicting_decision_evidence", "low_confidence_decision":
		if missingSource || missingOperation {
			return nil, 0, false
		}
		return []string{boundaryRoot}, 60, true
	case readinessMissingBrowserAuthenticationFlow, readinessMissingBrowserAuthenticationSession, readinessMissingBrowserCredentialBindings, readinessMissingBrowserAuthenticationTimeout:
		return []string{boundaryRoot}, 65, true
	case readinessUnconfirmedBrowserAuthentication:
		return []string{boundaryRoot}, 64, false
	case "missing_outputs":
		if missingOperation {
			return nil, 0, false
		}
		return []string{boundaryRoot}, 50, true
	default:
		return []string{boundaryRoot}, 40, strings.HasPrefix(issue.Slot, "steps.") || strings.Contains(issue.Slot, "output") || strings.Contains(issue.Slot, "source")
	}
}

func nodeIDForIssue(issue ReadinessIssue) string {
	slot := strings.NewReplacer(".", "_", "[", "_", "]", "", " ", "_").Replace(strings.ToLower(strings.TrimSpace(issue.Slot)))
	code := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(issue.Code)), ".", "_")
	if strings.HasSuffix(issue.Slot, ".operation") {
		return "operation." + strings.TrimSuffix(strings.TrimPrefix(slot, "steps_"), "_operation")
	}
	if strings.HasSuffix(issue.Slot, ".security_alternative") {
		return "security.alternative." + strings.TrimSuffix(strings.TrimPrefix(slot, "steps_"), "_security_alternative")
	}
	if strings.Contains(issue.Slot, ".with") {
		return "mapping." + strings.TrimPrefix(slot, "steps_")
	}
	return "readiness." + code + "." + slot
}

func hasIssueCode(issues map[string]ReadinessIssue, code string) bool {
	for _, issue := range issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}

func hasNode(nodes []publicinterview.Node, id string) bool {
	for _, node := range nodes {
		if node.ID == id {
			return true
		}
	}
	return false
}

func matchingNodeIDs(nodes []publicinterview.Node, prefix string) []string {
	var out []string
	for _, node := range nodes {
		if strings.HasPrefix(node.ID, prefix) {
			out = append(out, node.ID)
		}
	}
	sort.Strings(out)
	return out
}

func existingDependencies(nodes []publicinterview.Node, ids ...string) []string {
	var out []string
	for _, id := range ids {
		if hasNode(nodes, id) {
			out = append(out, id)
		}
	}
	return out
}

func dedupeInterviewNodes(nodes []publicinterview.Node) []publicinterview.Node {
	byID := map[string]publicinterview.Node{}
	for _, node := range nodes {
		byID[node.ID] = node
	}
	out := make([]publicinterview.Node, 0, len(byID))
	for _, node := range byID {
		out = append(out, node)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func retainGraphDeferrals(deferrals []publicinterview.Deferral, nodes []publicinterview.Node) []publicinterview.Deferral {
	valid := map[string]bool{}
	for _, node := range nodes {
		valid[node.ID] = node.Status == publicinterview.StatusDeferred
	}
	var out []publicinterview.Deferral
	for _, deferral := range deferrals {
		if valid[deferral.NodeID] {
			out = append(out, deferral)
		}
	}
	return out
}

// ApplyFrontierRound delegates clone-based product mutation and atomic
// interview settlement to the shared Authoring binding.
func ApplyFrontierRound(session *Session, answers []authoring.RoundAnswer, docs []APIDocument) error {
	if session == nil {
		return nil
	}
	return openUdonInterviewBinding(docs).Apply(session, answers, docs)
}

func openUdonInterviewBinding(docs []APIDocument) authoring.InterviewBinding[Session, APIDocument] {
	return authoring.InterviewBinding[Session, APIDocument]{
		State: func(session *Session) *publicinterview.State { return &session.Interview },
		Clone: cloneSession,
		Prepare: func(session *Session, currentDocs []APIDocument) error {
			state, _ := buildInterviewState(*session, currentDocs, CheckReadiness(*session, currentDocs))
			session.Interview = state
			return publicinterview.Validate(session.Interview)
		},
		Question: func(session Session, currentDocs []APIDocument, node publicinterview.Node) QuestionPlan {
			_, questions := buildInterviewState(session, currentDocs, CheckReadiness(session, currentDocs))
			if question, ok := questions[node.ID]; ok {
				return question
			}
			return QuestionPlan{ID: node.ID, Prompt: node.Prompt, Required: node.Required, Recommendation: node.Recommendation, Priority: node.Priority, Rationale: node.Rationale, EvidenceRefs: node.EvidenceRefs}
		},
		Resolve: func(session *Session, currentDocs []APIDocument, node publicinterview.Node, answer authoring.RoundAnswer) (publicinterview.Resolution, error) {
			value := strings.TrimSpace(answer.Value)
			if strings.HasPrefix(strings.ToLower(value), "defer:") {
				if !node.Deferrable {
					return publicinterview.Resolution{}, fmt.Errorf("decision %q may not be deferred", node.ID)
				}
				parts := strings.Split(strings.TrimSpace(value[len("defer:"):]), "|")
				if len(parts) != 4 {
					return publicinterview.Resolution{}, fmt.Errorf("defer %q with owner | impact | unblock condition | suggested next action", node.ID)
				}
				deferral := publicinterview.Deferral{
					ID: "deferral." + fmt.Sprintf("%03d", session.Interview.Round+1) + "." + node.ID, NodeID: node.ID,
					Owner: strings.TrimSpace(parts[0]), Impact: strings.TrimSpace(parts[1]), UnblockCondition: strings.TrimSpace(parts[2]), SuggestedNextAction: strings.TrimSpace(parts[3]),
				}
				return publicinterview.Resolution{NodeID: node.ID, Deferral: &deferral}, nil
			}
			if value == "" {
				return publicinterview.Resolution{}, fmt.Errorf("decision %q requires an answer", node.ID)
			}
			if err := applyFrontierValue(session, node.ID, answer, currentDocs); err != nil {
				return publicinterview.Resolution{}, err
			}
			resolved := publicinterview.Answer{ID: "answer." + fmt.Sprintf("%03d", session.Interview.Round+1) + "." + node.ID, NodeID: node.ID, Value: value, Source: answer.Source}
			return publicinterview.Resolution{NodeID: node.ID, Answer: &resolved}, nil
		},
		Normalize: func(session *Session) {
			session.Normalize()
			session.Boundary.Confirmed = session.Boundary.Outcome != "" && session.Boundary.Actor != "" && session.Boundary.Trigger != "" && len(session.Boundary.SuccessEvidence) > 0
		},
	}
}

func applyFrontierValue(session *Session, nodeID string, answer authoring.RoundAnswer, docs []APIDocument) error {
	value := strings.TrimSpace(answer.Value)
	switch nodeID {
	case nodeBoundaryOutcome:
		session.Boundary.Outcome = value
		applyProgressiveAnswer(session, QuestionPlan{Slots: []string{"workflow.goal"}}, value, docs)
	case nodeActiveWorkflow:
		if session.Interview.Metadata == nil {
			session.Interview.Metadata = map[string]string{}
		}
		selected := strings.ToLower(strings.TrimSpace(value))
		matched := selected == strings.ToLower(strings.TrimSpace(session.Boundary.Outcome))
		for index, candidate := range session.CandidateWorkflows {
			if selected != strings.ToLower(candidate.Title) && selected != strings.ToLower(candidate.Outcome) && selected != strings.ToLower(candidate.Title+": "+candidate.Outcome) {
				continue
			}
			previous := CandidateWorkflow{Title: humanTitle(firstNonEmpty(session.IntentName(), "current workflow")), Outcome: session.Boundary.Outcome, DeferralReason: "another workflow was explicitly selected as the active authoring boundary", PromotionTrigger: "the selected active workflow is complete or priorities change"}
			session.Boundary.Outcome = candidate.Outcome
			session.Project.Goal = candidate.Outcome
			if session.Intent.Workflow == nil {
				session.Intent.Workflow = &rollout.WorkflowMeta{}
			}
			session.Intent.Workflow.Name = slug(candidate.Title)
			session.Intent.Workflow.Description = candidate.Outcome
			session.Intent.Source, session.Intent.OpenAPI, session.Intent.ServerURL = "", "", ""
			session.Intent.Inputs, session.Intent.Steps, session.Intent.Security, session.Intent.Outputs = nil, nil, nil, nil
			session.SourcePlan = nil
			session.CandidateWorkflows = append(append(session.CandidateWorkflows[:index:index], session.CandidateWorkflows[index+1:]...), previous)
			matched = true
			break
		}
		if !matched {
			return fmt.Errorf("active workflow %q is not one of the proposed boundaries", value)
		}
		session.Interview.Metadata["active_workflow_selected"] = session.Boundary.Outcome
	case nodeActorTrigger:
		parts := strings.SplitN(value, "|", 2)
		session.Boundary.Actor = strings.TrimSpace(parts[0])
		session.Boundary.Trigger = "on demand"
		if len(parts) == 2 {
			session.Boundary.Trigger = strings.TrimSpace(parts[1])
		}
	case nodeRemoteLookup:
		decision := strings.ToLower(value)
		if decision != "allow" && decision != "never" {
			return fmt.Errorf("remote source lookup decision must be allow or never")
		}
		if session.Interview.Metadata == nil {
			session.Interview.Metadata = map[string]string{}
		}
		session.Interview.Metadata["remote_lookup_decision"] = decision
	case nodeBrowserRegistry:
		decision := strings.ToLower(value)
		if decision != "allow" && decision != "never" {
			return fmt.Errorf("browser registry lookup decision must be allow or never")
		}
		if session.Interview.Metadata == nil {
			session.Interview.Metadata = map[string]string{}
		}
		session.Interview.Metadata["browser_registry_lookup_decision"] = decision
	case nodeBrowserSession:
		posture := strings.ToLower(value)
		if posture != "none" && posture != "opaque-runtime-binding-required" {
			return fmt.Errorf("browser session posture must be none or opaque-runtime-binding-required")
		}
		session.BrowserSession = posture
	case nodeBrowserApproval:
		if !strings.HasPrefix(strings.ToLower(value), "approve ") {
			return fmt.Errorf("browser mutation approval must use approve <operation-step-name>")
		}
		name := strings.TrimSpace(value[len("approve "):])
		step, _, operation := selectedBrowserOperation(*session, docs)
		if step == nil || operation == nil || !browserOperationMutates(operation) || name != step.Name {
			return fmt.Errorf("browser mutation approval %q does not match the selected mutating operation step", value)
		}
		session.BrowserApprovals = dedupeStrings(append(session.BrowserApprovals, step.Name))
	case nodeSuccessEvidence, nodeVerification:
		session.Boundary.SuccessEvidence = dedupeStrings(append(session.Boundary.SuccessEvidence, splitFrontierList(value)...))
	case nodeFallback:
		session.Fallback, session.Project.Fallback, session.FallbackSet = value, value, true
	default:
		if strings.HasPrefix(nodeID, "security.alternative.") {
			for _, step := range session.Intent.Steps {
				if step == nil || nodeIDForIssue(ReadinessIssue{Code: "missing_security_alternative", Slot: securityAlternativeSlot(step)}) != nodeID {
					continue
				}
				op, ok := operationForStep(*session, docs, step)
				if !ok || !selectSecurityAlternative(session, step, op, value) {
					return fmt.Errorf("security alternative %q is not one unambiguous listed choice", value)
				}
				return nil
			}
			return fmt.Errorf("security alternative decision %q does not match an active step", nodeID)
		}
		applyProgressiveAnswer(session, QuestionPlan{ID: nodeID, Slots: append([]string(nil), answer.Slots...)}, value, docs)
	}
	return nil
}

func splitFrontierList(value string) []string {
	value = strings.ReplaceAll(value, "\n", ";")
	var out []string
	for _, item := range strings.Split(value, ";") {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}
	return out
}

func cloneSession(session Session) (Session, error) {
	data, err := json.Marshal(session)
	if err != nil {
		return Session{}, err
	}
	var clone Session
	if err := json.Unmarshal(data, &clone); err != nil {
		return Session{}, err
	}
	clone.Annotations = append([]SourceAnnotation(nil), session.Annotations...)
	clone.Assumptions = append([]Assumption(nil), session.Assumptions...)
	clone.Classifications = append([]MappingClassification(nil), session.Classifications...)
	clone.DecisionEvidence = append([]DecisionEvidence(nil), session.DecisionEvidence...)
	contentBySource := map[string][]byte{}
	for _, source := range session.SourcePlan {
		if len(source.MaterializedContent) == 0 {
			continue
		}
		key := source.TargetPath + "\x00" + source.SHA256
		contentBySource[key] = append([]byte(nil), source.MaterializedContent...)
	}
	for index := range clone.SourcePlan {
		key := clone.SourcePlan[index].TargetPath + "\x00" + clone.SourcePlan[index].SHA256
		clone.SourcePlan[index].MaterializedContent = append([]byte(nil), contentBySource[key]...)
	}
	return clone, nil
}

// FinalApprovalNodeID exposes the non-deferrable proposal approval identity to
// agent reports and proposal renderers.
func FinalApprovalNodeID() string { return nodeFinalIntentApproval }
