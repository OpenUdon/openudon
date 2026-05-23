package elicitor

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"strings"

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
	Severity        string `json:"severity,omitempty"`
	Code            string `json:"code,omitempty"`
	Message         string `json:"message,omitempty"`
	Slot            string `json:"slot,omitempty"`
	SuggestedAnswer string `json:"suggested_answer,omitempty"`
	Evidence        string `json:"evidence,omitempty"`
}

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
		out.Issues = append(out.Issues, DraftReviewIssue{
			Severity:        readinessWarning,
			Code:            code,
			Message:         truncateForPrompt(message, 240),
			Slot:            truncateForPrompt(strings.TrimSpace(issue.Slot), 120),
			SuggestedAnswer: truncateForPrompt(strings.TrimSpace(issue.SuggestedAnswer), 240),
			Evidence:        truncateForPrompt(strings.TrimSpace(issue.Evidence), 240),
		})
		if len(out.Issues) >= 5 {
			break
		}
	}
	return out
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
	if strings.TrimSpace(issue.Slot) != "" {
		lines = append(lines, "# Slot: "+reviewCommentText(issue.Slot))
	}
	if strings.TrimSpace(issue.Evidence) != "" {
		lines = append(lines, "# Evidence: "+reviewCommentText(issue.Evidence))
	}
	if strings.TrimSpace(issue.SuggestedAnswer) != "" {
		lines = append(lines, "# Suggested review: "+reviewCommentText(issue.SuggestedAnswer))
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
	if session == nil || extractor == nil {
		return nil
	}
	request := BuildDraftReviewRequest(*session, docs, issues)
	if len(request.Steps) == 0 {
		return nil
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
