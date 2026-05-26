package elicitor

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/OpenUdon/apitools"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

type RequestMappingRequest struct {
	Opening          string                    `json:"opening,omitempty"`
	SessionSummary   RequestMappingSession     `json:"session_summary,omitempty"`
	Question         string                    `json:"question,omitempty"`
	ReadinessIssues  []ReadinessIssue          `json:"readiness_issues,omitempty"`
	DecisionEvidence []DecisionEvidence        `json:"decision_evidence,omitempty"`
	Steps            []RequestMappingStep      `json:"steps"`
	AvailableInputs  []string                  `json:"available_inputs,omitempty"`
	AvailableSecrets []string                  `json:"available_credentials,omitempty"`
	PriorSteps       []RequestMappingPriorStep `json:"prior_steps,omitempty"`
}

type RequestMappingSession struct {
	Workflow string `json:"workflow,omitempty"`
	Safety   string `json:"safety,omitempty"`
}

type RequestMappingPriorStep struct {
	Name      string `json:"name"`
	Provider  string `json:"provider,omitempty"`
	Operation string `json:"operationId,omitempty"`
}

type RequestMappingStep struct {
	Name          string                 `json:"name"`
	Provider      string                 `json:"provider,omitempty"`
	Source        string                 `json:"source,omitempty"`
	OpenAPI       string                 `json:"openapi,omitempty"`
	OperationID   string                 `json:"operationId"`
	Do            string                 `json:"do,omitempty"`
	MissingFields []string               `json:"missing_fields"`
	KnownFields   []string               `json:"known_fields,omitempty"`
	Operation     operationPromptContext `json:"operation"`
}

type RequestMappingResponse struct {
	Steps       []RequestMappingStepResponse `json:"steps,omitempty"`
	Assumptions []string                     `json:"assumptions,omitempty"`
	Blockers    []string                     `json:"blockers,omitempty"`
}

func (response *RequestMappingResponse) UnmarshalJSON(data []byte) error {
	var raw struct {
		Steps       json.RawMessage `json:"steps,omitempty"`
		Assumptions []string        `json:"assumptions,omitempty"`
		Blockers    []string        `json:"blockers,omitempty"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("parse request-mapping response: %w", err)
	}
	response.Steps = decodeRequestMappingSteps(raw.Steps)
	response.Assumptions = raw.Assumptions
	response.Blockers = raw.Blockers
	return nil
}

type RequestMappingStepResponse struct {
	Name string            `json:"name"`
	With map[string]string `json:"with,omitempty"`
}

func decodeRequestMappingSteps(raw json.RawMessage) []RequestMappingStepResponse {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var list []RequestMappingStepResponse
	if err := json.Unmarshal(raw, &list); err == nil {
		return list
	}
	var keyed map[string]json.RawMessage
	if err := json.Unmarshal(raw, &keyed); err != nil {
		return nil
	}
	steps := make([]RequestMappingStepResponse, 0, len(keyed))
	for name, data := range keyed {
		var step RequestMappingStepResponse
		if err := json.Unmarshal(data, &step); err != nil {
			step.With = decodeRequestMappingWith(data)
		}
		if step.Name == "" {
			step.Name = strings.TrimSpace(name)
		}
		if len(step.With) == 0 {
			step.With = decodeRequestMappingWith(data)
		}
		steps = append(steps, step)
	}
	return steps
}

func (step *RequestMappingStepResponse) UnmarshalJSON(data []byte) error {
	var raw struct {
		Name string          `json:"name"`
		With json.RawMessage `json:"with,omitempty"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("parse request-mapping step: %w", err)
	}
	step.Name = raw.Name
	step.With = decodeRequestMappingWith(raw.With)
	return nil
}

func decodeRequestMappingWith(raw json.RawMessage) map[string]string {
	out := map[string]string{}
	if len(raw) == 0 || string(raw) == "null" {
		return out
	}
	var object map[string]any
	if err := json.Unmarshal(raw, &object); err == nil {
		for field, value := range object {
			if text := requestMappingValueString(value); text != "" {
				out[field] = text
			}
		}
		return out
	}
	var entries []map[string]any
	if err := json.Unmarshal(raw, &entries); err != nil {
		return out
	}
	for _, entry := range entries {
		field := firstNonEmpty(
			requestMappingValueString(entry["field"]),
			requestMappingValueString(entry["name"]),
			requestMappingValueString(entry["key"]),
		)
		value := firstNonEmpty(
			requestMappingValueString(entry["value"]),
			requestMappingValueString(entry["source"]),
			requestMappingValueString(entry["mapping"]),
		)
		if field != "" && value != "" {
			out[field] = value
		}
	}
	return out
}

func requestMappingValueString(value any) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(v)
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func BuildRequestMappingRequest(opening string, session Session, docs []APIDocument, issues []ReadinessIssue, question QuestionPlan) RequestMappingRequest {
	request := RequestMappingRequest{
		Opening:          strings.TrimSpace(opening),
		SessionSummary:   requestMappingSessionSummary(session),
		Question:         strings.TrimSpace(question.Prompt),
		ReadinessIssues:  requestMappingIssues(issues),
		DecisionEvidence: compactDecisionEvidence(session.DecisionEvidence),
		AvailableInputs:  requestMappingInputNames(session.Intent.Inputs),
		PriorSteps:       requestMappingPriorSteps(session.Intent.Steps),
	}
	for _, credential := range session.Credentials {
		credential = strings.TrimSpace(credential)
		if credential != "" {
			request.AvailableSecrets = append(request.AvailableSecrets, credential)
		}
	}
	for _, step := range targetStepsForWithPlan(&session, question) {
		if step == nil {
			continue
		}
		op, ok := operationForStep(session, docs, step)
		if !ok {
			continue
		}
		missing := missingRequiredFields(step, op)
		if len(missing) == 0 {
			continue
		}
		request.Steps = append(request.Steps, RequestMappingStep{
			Name:          firstNonEmpty(step.Name, "step"),
			Provider:      strings.TrimSpace(step.Provider),
			Source:        stepAPISourceRef(session, step),
			OpenAPI:       strings.TrimSpace(firstNonEmpty(step.OpenAPI, session.Intent.OpenAPI)),
			OperationID:   strings.TrimSpace(step.Operation),
			Do:            strings.TrimSpace(step.Do),
			MissingFields: append([]string(nil), missing...),
			KnownFields:   qualifiedRequestFieldsForPrompt(op),
			Operation:     operationPrompt(*op),
		})
	}
	return request
}

func applyRequestMappingResponse(session *Session, request RequestMappingRequest, response RequestMappingResponse) requestMappingApplication {
	application := requestMappingApplication{}
	if session == nil {
		return application
	}
	allowed := map[string]map[string]bool{}
	for _, step := range request.Steps {
		name := strings.TrimSpace(step.Name)
		if name == "" {
			continue
		}
		allowed[name] = map[string]bool{}
		for _, field := range step.MissingFields {
			field = strings.TrimSpace(field)
			if field != "" {
				allowed[name][field] = true
			}
		}
	}
	steps := stepsByName(session.Intent.Steps)
	for _, proposed := range response.Steps {
		name := strings.TrimSpace(proposed.Name)
		fields, ok := allowed[name]
		if !ok {
			application.Rejected = append(application.Rejected, "unknown step "+name)
			continue
		}
		step := steps[name]
		if step == nil {
			application.Rejected = append(application.Rejected, "missing local step "+name)
			continue
		}
		for field, source := range proposed.With {
			field = strings.TrimSpace(field)
			source = strings.TrimSpace(source)
			if field == "" || source == "" {
				continue
			}
			normalizedField, ok := normalizeProposedRequestField(field, fields)
			if !ok {
				application.Rejected = append(application.Rejected, "unknown field "+name+"."+field+"; use declared fields such as path.<name>, query.<name>, header.<name>, or body.<name> when aliases are ambiguous")
				continue
			}
			if !safeRequestMappingValue(source) {
				application.Rejected = append(application.Rejected, "unsafe value "+name+"."+normalizedField)
				continue
			}
			if setStepWithIfEmpty(step, normalizedField, source) {
				application.Applied++
				addRequestMappingClassification(session, step, normalizedField, source)
				addRequestMappingAssumption(session, step, normalizedField, source)
				addInputsAndCredentialsFromLLMMapping(session, source)
			}
		}
	}
	if len(response.Assumptions) > 0 || len(response.Blockers) > 0 || len(application.Rejected) > 0 || application.Applied > 0 {
		session.DraftEvents = append(session.DraftEvents, TranscriptEvent{Kind: "request_mapping_draft_result", Data: map[string]any{
			"applied":     application.Applied,
			"rejected":    application.Rejected,
			"assumptions": response.Assumptions,
			"blockers":    response.Blockers,
		}})
	}
	return application
}

// NormalizeRequestMappingField maps a proposed request field, including
// qualified aliases such as path.id, to one of the declared allowed fields.
func NormalizeRequestMappingField(field string, allowedFields []string) (string, bool) {
	allowed := map[string]bool{}
	for _, allowedField := range allowedFields {
		allowedField = strings.TrimSpace(allowedField)
		if allowedField != "" {
			allowed[allowedField] = true
		}
	}
	return normalizeProposedRequestField(field, allowed)
}

func normalizeProposedRequestField(field string, allowed map[string]bool) (string, bool) {
	field = strings.TrimSpace(field)
	if field == "" {
		return "", false
	}
	if allowed[field] {
		return field, true
	}
	for _, prefix := range []string{"path.", "query.", "header.", "body."} {
		if strings.HasPrefix(field, prefix) {
			unqualified := strings.TrimPrefix(field, prefix)
			if allowed[unqualified] {
				return unqualified, true
			}
		}
	}
	return "", false
}

func qualifiedRequestFieldsForPrompt(op *apitools.OperationSummary) []string {
	if op == nil {
		return nil
	}
	var out []string
	for _, parameter := range op.Parameters {
		name := strings.TrimSpace(parameter.Name)
		if name == "" {
			continue
		}
		location := strings.TrimSpace(parameter.In)
		if location == "" {
			location = "query"
		}
		out = append(out, location+"."+name)
	}
	if op.RequestBody != nil {
		for _, field := range op.RequestBody.Fields {
			path := strings.TrimSpace(field.Path)
			if path != "" {
				out = append(out, "body."+path)
			}
		}
	}
	return dedupeStrings(out)
}

type requestMappingApplication struct {
	Applied  int
	Rejected []string
}

func requestMappingSessionSummary(session Session) RequestMappingSession {
	summary := RequestMappingSession{Safety: strings.TrimSpace(session.Safety)}
	if session.Intent.Workflow != nil {
		summary.Workflow = strings.TrimSpace(session.Intent.Workflow.Description)
	}
	return summary
}

func requestMappingIssues(issues []ReadinessIssue) []ReadinessIssue {
	var out []ReadinessIssue
	for _, issue := range issues {
		if issue.Code == "missing_required_request_values" {
			out = append(out, issue)
		}
	}
	return out
}

func requestMappingInputNames(inputs []*rollout.Input) []string {
	var names []string
	for _, input := range inputs {
		if input != nil && strings.TrimSpace(input.Name) != "" {
			names = append(names, strings.TrimSpace(input.Name))
		}
	}
	return names
}

func requestMappingPriorSteps(steps []*rollout.Step) []RequestMappingPriorStep {
	var out []RequestMappingPriorStep
	for _, step := range steps {
		if step == nil || strings.TrimSpace(step.Name) == "" {
			continue
		}
		out = append(out, RequestMappingPriorStep{
			Name:      strings.TrimSpace(step.Name),
			Provider:  strings.TrimSpace(step.Provider),
			Operation: strings.TrimSpace(step.Operation),
		})
	}
	return out
}

func safeRequestMappingValue(source string) bool {
	source = strings.TrimSpace(source)
	if source == "" || len(source) > 240 {
		return false
	}
	if strings.ContainsAny(source, "\r\n") {
		return false
	}
	lower := strings.ToLower(source)
	if strings.Contains(lower, "sk-") || strings.Contains(lower, "secret=") || strings.Contains(lower, "token=") {
		return false
	}
	return true
}

func addRequestMappingClassification(session *Session, step *rollout.Step, field, source string) {
	addMappingClassification(session, MappingClassification{
		Slot:                 stepWithSlot(step, field),
		Value:                source,
		Source:               mappingSourceLLM,
		Confidence:           mappingConfidenceReview,
		Evidence:             source,
		Reason:               "LLM drafted this request field mapping from the selected operation metadata and workflow brief.",
		RequiresConfirmation: true,
	})
}

func addRequestMappingAssumption(session *Session, step *rollout.Step, field, source string) {
	slot := stepWithSlot(step, field)
	session.Assumptions = mergeAssumptions(session.Assumptions, []Assumption{{
		ID:                   "request_mapping_draft_" + slugIdent(slot),
		Slot:                 slot,
		Value:                field + "=" + source,
		Reason:               "LLM inferred this required request field mapping from the selected operation metadata and workflow brief.",
		Evidence:             source,
		Risk:                 "review",
		RequiresConfirmation: true,
	}})
}

func addInputsAndCredentialsFromLLMMapping(session *Session, source string) {
	source = strings.TrimSpace(source)
	if strings.HasPrefix(source, "inputs.") {
		name := strings.TrimSpace(strings.TrimPrefix(source, "inputs."))
		if name != "" {
			session.Intent.Inputs = mergeInputsByName(session.Intent.Inputs, []*rollout.Input{{Name: name, Type: "string", Required: true}})
		}
	}
	for _, credential := range credentialCandidates(source) {
		credential = strings.TrimSpace(credential)
		if credential == "" {
			continue
		}
		session.Credentials = dedupeStrings(append(session.Credentials, credential))
		session.CredentialsSet = true
		addMappingClassification(session, MappingClassification{
			Slot:                 "credentials",
			Value:                credential,
			Source:               mappingSourceLLM,
			Confidence:           mappingConfidenceReview,
			Evidence:             source,
			Reason:               "LLM drafted a symbolic credential binding referenced by a required request field.",
			RequiresConfirmation: true,
		})
	}
}
