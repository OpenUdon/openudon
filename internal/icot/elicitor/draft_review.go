package elicitor

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"strings"

	"github.com/OpenUdon/apitools"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

type DraftReviewRequest struct {
	Goal             string                    `json:"goal,omitempty"`
	Workflow         string                    `json:"workflow,omitempty"`
	Inputs           []string                  `json:"inputs,omitempty"`
	Outputs          []DraftReviewOutput       `json:"outputs,omitempty"`
	Credentials      []string                  `json:"credentials,omitempty"`
	Warnings         []ReadinessIssue          `json:"warnings,omitempty"`
	DecisionEvidence []DecisionEvidence        `json:"decision_evidence,omitempty"`
	Steps            []DraftReviewStep         `json:"steps"`
	PriorSteps       []RequestMappingPriorStep `json:"prior_steps,omitempty"`
}

type DraftReviewStep struct {
	Name        string                 `json:"name"`
	Type        string                 `json:"type,omitempty"`
	Provider    string                 `json:"provider,omitempty"`
	Source      string                 `json:"source,omitempty"`
	OperationID string                 `json:"operationId,omitempty"`
	Do          string                 `json:"do,omitempty"`
	DependsOn   []string               `json:"depends_on,omitempty"`
	With        map[string]string      `json:"with,omitempty"`
	Binds       []*rollout.StepBind    `json:"binds,omitempty"`
	Operation   operationPromptContext `json:"operation,omitempty"`
}

type DraftReviewOutput struct {
	Name string `json:"name"`
	From string `json:"from"`
}

type DraftReviewResponse struct {
	Issues []DraftReviewIssue `json:"issues,omitempty"`
}

type DraftReviewIssue struct {
	Severity           string `json:"severity,omitempty"`
	Code               string `json:"code,omitempty"`
	Message            string `json:"message,omitempty"`
	Slot               string `json:"slot,omitempty"`
	SuggestedAnswer    string `json:"suggested_answer,omitempty"`
	Evidence           string `json:"evidence,omitempty"`
	GapKind            string `json:"gap_kind,omitempty"`
	RemediationAction  string `json:"remediation_action,omitempty"`
	ClarifyingQuestion string `json:"clarifying_question,omitempty"`
}

const (
	flowGapMissingTransformStep     = "missing_transform_step"
	flowGapMissingAPIPrework        = "missing_api_prework"
	flowGapDisconnectedNotification = "disconnected_notification"
	flowGapAmbiguousOutput          = "ambiguous_output"
	flowGapOperationMismatch        = "operation_mismatch"
	flowGapUnavailableSource        = "unavailable_source"
	flowGapUnclearIntent            = "unclear_intent"
	flowGapNarrowRepair             = "narrow_repair"

	remediationApplyNarrowRepair = "apply_narrow_repair"
	remediationProposeFnctStep   = "propose_fnct_step"
	remediationProposeAPIPrework = "propose_api_prework"
	remediationAskUser           = "ask_user"
	remediationCommentOnly       = "comment_only"
)

func BuildDraftReviewRequest(session Session, docs []APIDocument, issues []ReadinessIssue) DraftReviewRequest {
	session.Normalize()
	request := DraftReviewRequest{
		Goal:             strings.TrimSpace(session.Project.Goal),
		Workflow:         draftSessionDescription(session),
		Inputs:           requestMappingInputNames(session.Intent.Inputs),
		Credentials:      append([]string(nil), session.Credentials...),
		Warnings:         draftReviewWarnings(issues),
		DecisionEvidence: compactDecisionEvidence(session.DecisionEvidence),
		PriorSteps:       requestMappingPriorSteps(session.Intent.Steps),
	}
	if request.Goal == "" && session.Intent.Workflow != nil {
		request.Goal = strings.TrimSpace(session.Intent.Workflow.Description)
	}
	for _, output := range session.Intent.Outputs {
		if output == nil {
			continue
		}
		name := strings.TrimSpace(output.Name)
		from := strings.TrimSpace(output.From)
		if name != "" || from != "" {
			request.Outputs = append(request.Outputs, DraftReviewOutput{Name: name, From: from})
		}
	}
	for _, step := range session.Intent.Steps {
		if step == nil {
			continue
		}
		reviewStep := DraftReviewStep{
			Name:        firstNonEmpty(step.Name, "step"),
			Type:        strings.TrimSpace(step.Type),
			Provider:    strings.TrimSpace(step.Provider),
			Source:      stepAPISourceRef(session, step),
			OperationID: strings.TrimSpace(step.Operation),
			Do:          strings.TrimSpace(step.Do),
			DependsOn:   append([]string(nil), step.DependsOn...),
			With:        cloneStringMap(step.With),
			Binds:       cloneStepBinds(step.Binds),
		}
		if op, ok := operationForStep(session, docs, step); ok {
			reviewStep.Operation = operationPrompt(*op)
		}
		request.Steps = append(request.Steps, reviewStep)
	}
	return request
}

func compactDecisionEvidence(in []DecisionEvidence) []DecisionEvidence {
	normalized := normalizeDecisionEvidenceList(in)
	if len(normalized) <= 12 {
		return normalized
	}
	return normalized[len(normalized)-12:]
}

func draftReviewWarnings(issues []ReadinessIssue) []ReadinessIssue {
	var out []ReadinessIssue
	for _, issue := range issues {
		if issue.Severity == readinessWarning {
			out = append(out, issue)
		}
	}
	return out
}

func sanitizeDraftReviewResponse(response DraftReviewResponse) DraftReviewResponse {
	var out DraftReviewResponse
	for _, issue := range response.Issues {
		code := strings.ToLower(slugIdent(firstNonEmpty(issue.Code, "flow_review")))
		if !strings.HasPrefix(code, "llm_flow_review_") {
			code = "llm_flow_review_" + code
		}
		message := strings.TrimSpace(issue.Message)
		if code == "" || message == "" {
			continue
		}
		sanitized := DraftReviewIssue{
			Severity:           readinessWarning,
			Code:               code,
			Message:            truncateForPrompt(message, 240),
			Slot:               truncateForPrompt(strings.TrimSpace(issue.Slot), 120),
			SuggestedAnswer:    truncateForPrompt(strings.TrimSpace(issue.SuggestedAnswer), 240),
			Evidence:           truncateForPrompt(strings.TrimSpace(issue.Evidence), 240),
			ClarifyingQuestion: truncateForPrompt(strings.TrimSpace(issue.ClarifyingQuestion), 220),
		}
		kind, action, ok := normalizeReviewRemediation(issue.GapKind, issue.RemediationAction)
		if !ok {
			kind, action = classifyDraftReviewIssue(sanitized)
		}
		sanitized.GapKind = kind
		sanitized.RemediationAction = action
		if sanitized.RemediationAction != remediationAskUser {
			sanitized.ClarifyingQuestion = ""
		}
		out.Issues = append(out.Issues, sanitized)
		if len(out.Issues) >= 5 {
			break
		}
	}
	return out
}

func normalizeReviewRemediation(kind, action string) (string, string, bool) {
	kind = strings.TrimSpace(kind)
	action = strings.TrimSpace(action)
	if !validFlowGapKind(kind) || !validRemediationAction(action) {
		return "", "", false
	}
	if !remediationActionAllowedForGap(kind, action) {
		return "", "", false
	}
	return kind, action, true
}

func remediationActionAllowedForGap(kind, action string) bool {
	switch kind {
	case flowGapMissingTransformStep:
		return action == remediationProposeFnctStep || action == remediationCommentOnly || action == remediationAskUser
	case flowGapMissingAPIPrework:
		return action == remediationProposeAPIPrework || action == remediationCommentOnly || action == remediationAskUser
	case flowGapDisconnectedNotification:
		return action == remediationCommentOnly || action == remediationApplyNarrowRepair || action == remediationAskUser
	case flowGapAmbiguousOutput:
		return action == remediationAskUser || action == remediationCommentOnly
	case flowGapOperationMismatch:
		return action == remediationCommentOnly || action == remediationAskUser
	case flowGapUnavailableSource:
		return action == remediationCommentOnly
	case flowGapUnclearIntent:
		return action == remediationCommentOnly || action == remediationAskUser
	case flowGapNarrowRepair:
		return action == remediationApplyNarrowRepair
	default:
		return false
	}
}

func validFlowGapKind(kind string) bool {
	switch kind {
	case flowGapMissingTransformStep, flowGapMissingAPIPrework, flowGapDisconnectedNotification, flowGapAmbiguousOutput, flowGapOperationMismatch, flowGapUnavailableSource, flowGapUnclearIntent, flowGapNarrowRepair:
		return true
	default:
		return false
	}
}

func validRemediationAction(action string) bool {
	switch action {
	case remediationApplyNarrowRepair, remediationProposeFnctStep, remediationProposeAPIPrework, remediationAskUser, remediationCommentOnly:
		return true
	default:
		return false
	}
}

func classifyDraftReviewIssue(issue DraftReviewIssue) (string, string) {
	text := strings.ToLower(strings.Join([]string{issue.Code, issue.Message, issue.Slot, issue.Evidence, issue.SuggestedAnswer}, " "))
	switch {
	case isNarrowDraftRepairIssue(issue):
		return flowGapNarrowRepair, remediationApplyNarrowRepair
	case strings.Contains(text, "operation") && (strings.Contains(text, "wrong") || strings.Contains(text, "mismatch") || strings.Contains(text, "incompatible") || strings.Contains(text, "does not match")):
		return flowGapOperationMismatch, remediationCommentOnly
	case strings.Contains(text, "unavailable") || strings.Contains(text, "missing api") || strings.Contains(text, "missing source") || strings.Contains(text, "missing artifact") || strings.Contains(text, "not available"):
		return flowGapUnavailableSource, remediationCommentOnly
	case strings.Contains(text, "prework") || strings.Contains(text, "lookup") || strings.Contains(text, "fetch before") || strings.Contains(text, "read before") || strings.Contains(text, "resolve before"):
		return flowGapMissingAPIPrework, remediationProposeAPIPrework
	case notificationFlowText(text):
		return flowGapDisconnectedNotification, remediationCommentOnly
	case transformFlowText(text):
		return flowGapMissingTransformStep, remediationProposeFnctStep
	case strings.Contains(text, "output") && (strings.Contains(text, "ambiguous") || strings.Contains(text, "which") || strings.Contains(text, "unclear") || strings.Contains(text, "raw") || strings.Contains(text, "transport")):
		return flowGapAmbiguousOutput, remediationAskUser
	case strings.Contains(text, "unclear") || strings.Contains(text, "ambiguous") || strings.Contains(text, "underspecified"):
		return flowGapUnclearIntent, remediationCommentOnly
	default:
		return flowGapUnclearIntent, remediationCommentOnly
	}
}

func isNarrowDraftRepairIssue(issue DraftReviewIssue) bool {
	if strings.TrimSpace(issue.SuggestedAnswer) == "" {
		return false
	}
	slot := strings.TrimSpace(issue.Slot)
	return strings.Contains(slot, ".with.") || strings.HasSuffix(slot, ".depends_on") || strings.HasPrefix(slot, "intent.outputs.") || strings.HasPrefix(slot, "outputs[") || strings.HasPrefix(slot, "outputs.")
}

func notificationFlowText(text string) bool {
	transport := strings.Contains(text, "email") || strings.Contains(text, "gmail") || strings.Contains(text, "slack") || strings.Contains(text, "message") || strings.Contains(text, "notification") || strings.Contains(text, "send")
	disconnected := strings.Contains(text, "disconnect") || strings.Contains(text, "does not consume") || strings.Contains(text, "unrelated") || strings.Contains(text, "missing content")
	return transport && disconnected
}

func transformFlowText(text string) bool {
	for _, token := range []string{"report", "receipt", "summary", "summar", "render", "transform", "normalize", "normalise", "enrich", "produced content", "generated content"} {
		if strings.Contains(text, token) {
			return true
		}
	}
	return false
}

func draftReviewReadinessIssues(response DraftReviewResponse) []ReadinessIssue {
	return draftReviewIssuesToReadiness(sanitizeDraftReviewResponse(response).Issues)
}

func draftReviewIssuesToReadiness(issues []DraftReviewIssue) []ReadinessIssue {
	var out []ReadinessIssue
	for _, issue := range issues {
		out = append(out, ReadinessIssue{
			Severity:        issue.Severity,
			Code:            issue.Code,
			Message:         issue.Message,
			Slot:            issue.Slot,
			SuggestedAnswer: issue.SuggestedAnswer,
		})
	}
	return out
}

func annotateIntentHCLWithFlowReviewWarnings(intentHCL string, issues []DraftReviewIssue) string {
	intentHCL = strings.TrimRight(intentHCL, "\n")
	var filtered []DraftReviewIssue
	for _, issue := range issues {
		if issue.Severity == readinessWarning && strings.HasPrefix(issue.Code, "llm_flow_review_") && strings.TrimSpace(issue.Message) != "" {
			filtered = append(filtered, issue)
		}
	}
	if intentHCL == "" || len(filtered) == 0 {
		if intentHCL == "" {
			return ""
		}
		return intentHCL + "\n"
	}
	lines := strings.Split(intentHCL, "\n")
	commentsByLine := map[int][]string{}
	for _, issue := range filtered {
		line := flowReviewCommentAnchor(lines, issue.Slot)
		commentsByLine[line] = append(commentsByLine[line], flowReviewComment(issue))
	}
	var out []string
	for i, line := range lines {
		if comments, ok := commentsByLine[i]; ok {
			out = append(out, comments...)
		}
		out = append(out, line)
	}
	if comments, ok := commentsByLine[len(lines)]; ok {
		out = append(out, comments...)
	}
	return strings.TrimRight(strings.Join(out, "\n"), "\n") + "\n"
}

func flowReviewComment(issue DraftReviewIssue) string {
	lines := []string{fmt.Sprintf("# iCoT flow review warning (%s): %s", issue.Code, reviewCommentText(issue.Message))}
	if strings.TrimSpace(issue.GapKind) != "" {
		lines = append(lines, "# Gap kind: "+reviewCommentText(issue.GapKind))
	}
	if strings.TrimSpace(issue.RemediationAction) != "" {
		lines = append(lines, "# Remediation: "+reviewCommentText(issue.RemediationAction))
	}
	if strings.TrimSpace(issue.Slot) != "" {
		lines = append(lines, "# Slot: "+reviewCommentText(issue.Slot))
	}
	if strings.TrimSpace(issue.Evidence) != "" {
		lines = append(lines, "# Evidence: "+reviewCommentText(issue.Evidence))
	}
	if strings.TrimSpace(issue.SuggestedAnswer) != "" {
		lines = append(lines, "# Suggested review: "+reviewCommentText(issue.SuggestedAnswer))
	}
	if strings.TrimSpace(issue.ClarifyingQuestion) != "" {
		lines = append(lines, "# Clarifying question: "+reviewCommentText(issue.ClarifyingQuestion))
	}
	return strings.Join(lines, "\n")
}

func flowReviewCommentAnchor(lines []string, slot string) int {
	slot = strings.Trim(strings.TrimSpace(slot), ".")
	if slot == "" {
		return 0
	}
	parts := strings.Split(slot, ".")
	for i := 0; i < len(parts)-1; i++ {
		switch parts[i] {
		case "steps":
			if line := findHCLBlockLine(lines, "step", parts[i+1]); line >= 0 {
				return line
			}
		case "outputs":
			if line := findHCLBlockLine(lines, "output", parts[i+1]); line >= 0 {
				return line
			}
		case "inputs":
			if line := findHCLBlockLine(lines, "input", parts[i+1]); line >= 0 {
				return line
			}
		case "workflow":
			if line := findHCLBlockLine(lines, "workflow", ""); line >= 0 {
				return line
			}
		}
	}
	return 0
}

func findHCLBlockLine(lines []string, blockType, label string) int {
	prefix := blockType + " "
	if label != "" {
		prefix += fmt.Sprintf("%q ", label)
	}
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if label == "" {
			if trimmed == blockType+" {" {
				return i
			}
			continue
		}
		if strings.HasPrefix(trimmed, prefix) && strings.HasSuffix(trimmed, "{") {
			return i
		}
	}
	return -1
}

func reviewCommentText(value string) string {
	value = strings.TrimSpace(truncateForPrompt(value, 220))
	return strings.ReplaceAll(value, "\n", " ")
}

func reviewFinalDraft(ctx context.Context, out io.Writer, extractor Extractor, session *Session, docs []APIDocument, issues []ReadinessIssue, events *[]TranscriptEvent) []DraftReviewIssue {
	if session == nil {
		return nil
	}
	if noSourceCapabilityGapFallbackSession(*session) {
		if events != nil {
			*events = append(*events, TranscriptEvent{Kind: "draft_flow_review_result", Data: map[string]any{
				"issues":  []DraftReviewIssue{},
				"skipped": "no-source capability gap fallback",
			}})
		}
		return nil
	}
	localIssues := localDraftReviewIssues(*session, docs)
	if extractor == nil {
		return localIssues
	}
	request := BuildDraftReviewRequest(*session, docs, issues)
	if len(request.Steps) == 0 {
		return localIssues
	}
	if events != nil {
		*events = append(*events, TranscriptEvent{Kind: "draft_flow_review_call", Data: map[string]any{
			"steps":   draftReviewStepNames(request.Steps),
			"outputs": request.Outputs,
		}})
	}
	response, err := extractor.ReviewDraft(ctx, request)
	if err != nil {
		if events != nil {
			*events = append(*events, TranscriptEvent{Kind: "draft_flow_review_error", Data: err.Error()})
		}
		fmt.Fprintf(out, "icot: draft flow review skipped: %v\n", err)
		return nil
	}
	sanitized := sanitizeDraftReviewResponse(response)
	sanitized.Issues = filterDraftReviewIssues(*session, mergeDraftReviewIssues(localIssues, sanitized.Issues))
	for _, issue := range sanitized.Issues {
		addDecisionEvidence(session, DecisionEvidence{
			Stage:                decisionStageDraftReview,
			Slot:                 firstNonEmpty(issue.Slot, "draft_review."+issue.Code),
			Value:                issue.Message,
			Source:               mappingSourceLLM,
			Confidence:           mappingConfidenceReview,
			Reason:               "Pre-final flow review reported a cross-step consistency issue.",
			Evidence:             issue.Evidence,
			RequiresConfirmation: true,
		})
	}
	if events != nil {
		*events = append(*events, TranscriptEvent{Kind: "draft_flow_review_result", Data: map[string]any{
			"issues": sanitized.Issues,
		}})
	}
	return sanitized.Issues
}

func localDraftReviewIssues(session Session, docs []APIDocument) []DraftReviewIssue {
	if !goalAllowsLocalFnctRemediation(session) {
		return nil
	}
	var issues []DraftReviewIssue
	for _, step := range session.Intent.Steps {
		if step == nil {
			continue
		}
		op, ok := operationForStep(session, docs, step)
		if !ok {
			continue
		}
		field := missingRenderedRequestBodyField(step, op)
		if field == "" || !deliverySinkStep(step, field) {
			continue
		}
		issue := DraftReviewIssue{
			Severity:          readinessWarning,
			Code:              "llm_flow_review_missing_rendered_request_body",
			Message:           fmt.Sprintf("%s requires rendered request body content for %q, but the current draft does not provide it.", firstNonEmpty(step.Name, "step"), field),
			Slot:              "steps." + firstNonEmpty(step.Name, "step") + ".with." + field,
			SuggestedAnswer:   field + "=<local render fnct output>",
			Evidence:          "Selected operation request-body metadata requires downstream content.",
			GapKind:           flowGapMissingTransformStep,
			RemediationAction: remediationProposeFnctStep,
		}
		issues = append(issues, issue)
	}
	return issues
}

func missingRenderedRequestBodyField(step *rollout.Step, op *apitools.OperationSummary) string {
	if step == nil || op == nil || op.RequestBody == nil {
		return ""
	}
	for _, field := range renderedRequestBodyFields(op) {
		if !stepHasRequestField(step, field) {
			return field
		}
	}
	return ""
}

func renderedRequestBodyFields(op *apitools.OperationSummary) []string {
	if op == nil || op.RequestBody == nil {
		return nil
	}
	var fields []string
	for _, field := range op.RequestBody.Fields {
		name := strings.TrimSpace(field.Path)
		if name != "" && field.Required {
			fields = append(fields, name)
		}
	}
	if len(fields) > 0 {
		return fields
	}
	preferred := []string{"raw", "message", "text", "body", "content", "html", "payload"}
	for _, want := range preferred {
		for _, field := range op.RequestBody.Fields {
			name := strings.TrimSpace(field.Path)
			if strings.EqualFold(name, want) {
				return []string{name}
			}
		}
	}
	if op.RequestBody.Required || strings.TrimSpace(op.RequestBody.Ref) != "" {
		return []string{"body"}
	}
	return nil
}

func stepHasRequestField(step *rollout.Step, field string) bool {
	field = strings.TrimSpace(field)
	if step == nil || field == "" {
		return false
	}
	for _, key := range []string{field, "body." + field} {
		if strings.TrimSpace(step.With[key]) != "" {
			return true
		}
	}
	if field == "body" && strings.TrimSpace(step.With["body"]) != "" {
		return true
	}
	for _, bind := range step.Binds {
		if bind == nil {
			continue
		}
		for _, key := range []string{field, "body." + field} {
			if strings.TrimSpace(bind.Fields[key]) != "" {
				return true
			}
		}
	}
	return false
}

func deliverySinkStep(step *rollout.Step, field string) bool {
	text := strings.ToLower(strings.Join([]string{
		step.Name,
		step.Do,
		step.Provider,
		step.Operation,
		field,
	}, " "))
	for _, token := range []string{"send", "email", "gmail", "message", "notification", "notify", "slack", "raw"} {
		if strings.Contains(text, token) {
			return true
		}
	}
	return false
}

func mergeDraftReviewIssues(primary, secondary []DraftReviewIssue) []DraftReviewIssue {
	if len(primary) == 0 {
		return secondary
	}
	out := append([]DraftReviewIssue(nil), primary...)
	seen := map[string]bool{}
	for _, issue := range out {
		seen[draftReviewIssueKey(issue)] = true
	}
	for _, issue := range secondary {
		key := draftReviewIssueKey(issue)
		if seen[key] {
			continue
		}
		out = append(out, issue)
		seen[key] = true
	}
	return out
}

func draftReviewIssueKey(issue DraftReviewIssue) string {
	return strings.TrimSpace(issue.Slot) + "\x00" + strings.TrimSpace(issue.GapKind) + "\x00" + strings.TrimSpace(issue.RemediationAction)
}

func filterDraftReviewIssues(session Session, issues []DraftReviewIssue) []DraftReviewIssue {
	if len(issues) == 0 {
		return nil
	}
	out := make([]DraftReviewIssue, 0, len(issues))
	for _, issue := range issues {
		if localFnctTransportOutputFalsePositive(session, issue) {
			continue
		}
		out = append(out, issue)
	}
	return out
}

func localFnctTransportOutputFalsePositive(session Session, issue DraftReviewIssue) bool {
	if issue.GapKind != flowGapAmbiguousOutput && issue.GapKind != flowGapMissingTransformStep {
		return false
	}
	text := strings.ToLower(strings.Join([]string{
		issue.Code,
		issue.Message,
		issue.Evidence,
		issue.SuggestedAnswer,
	}, " "))
	if !strings.Contains(text, "transport") && !strings.Contains(text, "raw") && !strings.Contains(text, "received_body") {
		return false
	}
	outputName, _, ok := parseOutputSlot(&session, issue.Slot)
	if !ok {
		return false
	}
	for _, output := range session.Intent.Outputs {
		if output == nil || output.Name != outputName {
			continue
		}
		stepName := sourceStepName(output.From)
		if stepName == "" {
			return false
		}
		step := stepByName(session.Intent.Steps, stepName)
		if step == nil || strings.ToLower(strings.TrimSpace(step.Type)) != "fnct" {
			return false
		}
		if strings.TrimSpace(step.Operation) != "" || strings.TrimSpace(firstNonEmpty(step.Source, step.OpenAPI)) != "" {
			return false
		}
		return strings.HasPrefix(strings.TrimSpace(output.From), step.Name+".received_body")
	}
	return false
}

func noSourceCapabilityGapFallbackSession(session Session) bool {
	if len(session.Intent.Steps) != 1 || session.Intent.Steps[0] == nil {
		return false
	}
	step := session.Intent.Steps[0]
	if step.Name != "render_capability_gap" || strings.ToLower(strings.TrimSpace(step.Type)) != "fnct" {
		return false
	}
	if strings.TrimSpace(step.Operation) != "" || strings.TrimSpace(firstNonEmpty(step.Source, step.OpenAPI)) != "" {
		return false
	}
	hasOutput := false
	for _, output := range session.Intent.Outputs {
		if output != nil && output.Name == "gap_report" && output.From == "render_capability_gap.received_body" {
			hasOutput = true
			break
		}
	}
	if !hasOutput {
		return false
	}
	for _, evidence := range session.DecisionEvidence {
		if evidence.Slot == "intent.steps.render_capability_gap" && evidence.Value == "no-source capability gap fallback" {
			return true
		}
	}
	return false
}

func finalDraftReviewKey(artifacts Artifacts) string {
	if artifacts.IntentHCL == "" && artifacts.ProjectMD == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(artifacts.IntentHCL + "\x00" + artifacts.ProjectMD))
	return hex.EncodeToString(sum[:])
}

func draftReviewStepNames(steps []DraftReviewStep) []string {
	names := make([]string, 0, len(steps))
	for _, step := range steps {
		if strings.TrimSpace(step.Name) != "" {
			names = append(names, strings.TrimSpace(step.Name))
		}
	}
	return names
}

func cloneStepBinds(in []*rollout.StepBind) []*rollout.StepBind {
	if len(in) == 0 {
		return nil
	}
	out := make([]*rollout.StepBind, 0, len(in))
	for _, bind := range in {
		if bind == nil {
			continue
		}
		out = append(out, &rollout.StepBind{
			From:   bind.From,
			Fields: cloneStringMap(bind.Fields),
		})
	}
	return out
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
