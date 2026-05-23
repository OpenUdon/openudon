package elicitor

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/OpenUdon/apitools"
	"github.com/OpenUdon/openudon/internal/authoring"
	"github.com/OpenUdon/openudon/internal/projectwizard"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

const PromptVersion = "icot-extractor.v1"

// Limit detail retries so the draft loop can ask for missing operation context
// without turning into an unbounded catalog crawl.
const maxDraftDetailRounds = 2

//go:embed prompts/kickoff.txt
var kickoffPrompt string

//go:embed prompts/refine.txt
var refinePrompt string

//go:embed prompts/disambiguate.txt
var disambiguatePrompt string

//go:embed prompts/draft.txt
var draftPrompt string

//go:embed prompts/catalog_plan.txt
var catalogPlanPrompt string

//go:embed prompts/request_mappings.txt
var requestMappingsPrompt string

//go:embed prompts/draft_review.txt
var draftReviewPrompt string

// Extractor provides optional, bounded LLM assistance for the interactive
// authoring loop. Implementations must return partial drafts only; the loop
// still asks the user to confirm the final intent before saving.
type Extractor interface {
	Kickoff(context.Context, string) (Session, error)
	CatalogPlan(context.Context, CatalogPlanRequest) (CatalogPlanResponse, error)
	RequestMappings(context.Context, RequestMappingRequest) (RequestMappingResponse, error)
	ReviewDraft(context.Context, DraftReviewRequest) (DraftReviewResponse, error)
	Draft(context.Context, DraftRequest) (Session, error)
	Refine(context.Context, Session) (Session, error)
	Disambiguate(context.Context, string, []APIDocument) ([]string, error)
}

type DraftRequest = authoring.InteractiveDraftRequest[Session, APIDocument]

type noopExtractor struct{}

func NewNoopExtractor() Extractor {
	return noopExtractor{}
}

func (noopExtractor) Kickoff(context.Context, string) (Session, error) {
	return Session{}, nil
}

func (noopExtractor) CatalogPlan(context.Context, CatalogPlanRequest) (CatalogPlanResponse, error) {
	return CatalogPlanResponse{}, nil
}

func (noopExtractor) RequestMappings(context.Context, RequestMappingRequest) (RequestMappingResponse, error) {
	return RequestMappingResponse{}, nil
}

func (noopExtractor) ReviewDraft(context.Context, DraftReviewRequest) (DraftReviewResponse, error) {
	return DraftReviewResponse{}, nil
}

func (noopExtractor) Draft(context.Context, DraftRequest) (Session, error) {
	return Session{}, nil
}

func (noopExtractor) Refine(_ context.Context, session Session) (Session, error) {
	return session, nil
}

func (noopExtractor) Disambiguate(context.Context, string, []APIDocument) ([]string, error) {
	return nil, nil
}

type chatExtractor struct {
	client      rollout.ChatClient
	temperature *float64
}

func NewChatExtractor(client rollout.ChatClient, temperature *float64) Extractor {
	if client == nil {
		return NewNoopExtractor()
	}
	return &chatExtractor{client: client, temperature: temperature}
}

func (e *chatExtractor) Kickoff(ctx context.Context, opening string) (Session, error) {
	opening = strings.TrimSpace(opening)
	if opening == "" {
		return Session{}, nil
	}
	var session Session
	if err := e.completeJSON(ctx, "kickoff extraction", kickoffPrompt, opening, kickoffSchema, &session, 1200); err != nil {
		return Session{}, err
	}
	session = sanitizeKickoff(session)
	if !emptySession(session) {
		session.Annotations = append(session.Annotations, SourceAnnotation{
			Slot:          "kickoff",
			Source:        "llm",
			PromptVersion: PromptVersion,
			Evidence:      firstLine(opening),
		})
	}
	return session, nil
}

func (e *chatExtractor) CatalogPlan(ctx context.Context, request CatalogPlanRequest) (CatalogPlanResponse, error) {
	request.Opening = strings.TrimSpace(request.Opening)
	if request.Opening == "" || len(request.Candidates) == 0 {
		return CatalogPlanResponse{}, nil
	}
	data, err := json.Marshal(request)
	if err != nil {
		return CatalogPlanResponse{}, err
	}
	var response CatalogPlanResponse
	if err := e.completeJSON(ctx, "catalog planning", catalogPlanPrompt, string(data), catalogPlanSchema, &response, 1000); err != nil {
		return CatalogPlanResponse{}, err
	}
	return response, nil
}

func (e *chatExtractor) RequestMappings(ctx context.Context, request RequestMappingRequest) (RequestMappingResponse, error) {
	request.Opening = strings.TrimSpace(request.Opening)
	if request.Opening == "" || len(request.Steps) == 0 {
		return RequestMappingResponse{}, nil
	}
	data, err := json.Marshal(request)
	if err != nil {
		return RequestMappingResponse{}, err
	}
	var response RequestMappingResponse
	if err := e.completeJSON(ctx, "request mapping draft", requestMappingsPrompt, string(data), requestMappingsSchema, &response, 900); err != nil {
		return RequestMappingResponse{}, err
	}
	return response, nil
}

func (e *chatExtractor) ReviewDraft(ctx context.Context, request DraftReviewRequest) (DraftReviewResponse, error) {
	if strings.TrimSpace(request.Goal) == "" || len(request.Steps) == 0 {
		return DraftReviewResponse{}, nil
	}
	data, err := json.Marshal(request)
	if err != nil {
		return DraftReviewResponse{}, err
	}
	var response DraftReviewResponse
	if err := e.completeJSON(ctx, "draft flow review", draftReviewPrompt, string(data), draftReviewSchema, &response, 800); err != nil {
		return DraftReviewResponse{}, err
	}
	return sanitizeDraftReviewResponse(response), nil
}

func (e *chatExtractor) Draft(ctx context.Context, request DraftRequest) (Session, error) {
	request.Opening = strings.TrimSpace(request.Opening)
	currentDescription := ""
	if request.Session.Intent.Workflow != nil {
		currentDescription = request.Session.Intent.Workflow.Description
	}
	if request.Opening == "" && strings.TrimSpace(currentDescription) == "" {
		return Session{}, nil
	}
	var requested []OperationDetailRef
	var warnings []Assumption
	var events []TranscriptEvent
	for round := 0; round <= maxDraftDetailRounds; round++ {
		detailRefs := draftDetailRefs(request, requested)
		data, err := json.Marshal(draftPromptRequestWithDetails(request, detailRefs))
		if err != nil {
			return Session{}, err
		}
		var completion draftCompletion
		if err := e.completeJSON(ctx, "workflow draft", draftPrompt, string(data), draftSchema, &completion, 1200); err != nil {
			return Session{}, err
		}
		valid, rejected, capped := resolveDraftDetailRequests(request.Docs, completion.RequestedOperationIDs, detailRefs)
		if capped {
			warnings = append(warnings, draftWarningAssumption("operation_detail_request_capped", "Only the first requested operationIds were considered for detail lookup."))
		}
		if len(rejected) > 0 {
			warnings = append(warnings, draftWarningAssumption("operation_detail_request_rejected", "Rejected unknown or unavailable requested operationIds: "+strings.Join(rejected, ", ")+"."))
			events = append(events, TranscriptEvent{Kind: "operation_detail_rejected", Data: map[string]any{
				"operation_ids": rejected,
			}})
		}
		if len(valid) > 0 {
			events = append(events, TranscriptEvent{Kind: "operation_detail_request", Data: map[string]any{
				"operation_ids":         operationRefIDs(valid),
				"detail_request_reason": strings.TrimSpace(completion.DetailRequestReason),
				"round":                 round + 1,
			}})
			if round == maxDraftDetailRounds {
				warnings = append(warnings, draftWarningAssumption("operation_detail_round_limit", "Additional operation detail requests were ignored after the bounded draft detail loop."))
				break
			}
			requested = appendOperationDetailRefs(requested, valid)
			events = append(events, TranscriptEvent{Kind: "operation_detail_fulfilled", Data: map[string]any{
				"operation_ids": operationRefIDs(valid),
				"round":         round + 1,
			}})
			continue
		}
		session := sanitizeDraftWithDetails(request, completion.Session, detailRefs)
		session.Assumptions = mergeAssumptions(session.Assumptions, warnings)
		session.DraftOperations = detailRefs
		session.DraftEvents = events
		return session, nil
	}
	session := sanitizeDraftWithDetails(request, Session{}, draftDetailRefs(request, requested))
	session.Assumptions = mergeAssumptions(session.Assumptions, warnings)
	session.DraftEvents = events
	return session, nil
}

func (e *chatExtractor) Refine(ctx context.Context, session Session) (Session, error) {
	data, err := json.Marshal(session)
	if err != nil {
		return session, err
	}
	var refined Session
	if err := e.completeJSON(ctx, "prose refinement", refinePrompt, string(data), kickoffSchema, &refined, 1200); err != nil {
		return session, err
	}
	return sanitizeRefine(session, refined), nil
}

func (e *chatExtractor) Disambiguate(ctx context.Context, need string, docs []APIDocument) ([]string, error) {
	if strings.TrimSpace(need) == "" || len(docs) == 0 {
		return nil, nil
	}
	var b strings.Builder
	b.WriteString("Need: ")
	b.WriteString(need)
	b.WriteString("\nAvailable OpenAPI documents:\n")
	for _, doc := range docs {
		b.WriteString("- ")
		b.WriteString(doc.RelativePath)
		if doc.Title != "" {
			b.WriteString(": ")
			b.WriteString(doc.Title)
		}
		b.WriteByte('\n')
	}
	var decoded struct {
		Paths []string `json:"paths"`
	}
	if err := e.completeJSON(ctx, "operation disambiguation", disambiguatePrompt, b.String(), disambiguateSchema, &decoded, 1200); err != nil {
		return nil, err
	}
	allowed := map[string]bool{}
	for _, doc := range docs {
		allowed[doc.RelativePath] = true
	}
	var out []string
	for _, path := range decoded.Paths {
		path = strings.TrimSpace(path)
		if allowed[path] {
			out = append(out, path)
		}
	}
	return out, nil
}

func (e *chatExtractor) completeJSON(ctx context.Context, stage, systemPrompt, userPayload, schema string, out any, maxTokens int) error {
	_, err := authoring.CompleteJSONWithFallback(ctx, rolloutChatAdapter{
		Client:      e.client,
		Temperature: e.temperature,
		MaxTokens:   maxTokens,
	}, []authoring.TranscriptTurn{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPayload},
	}, json.RawMessage(schema), out, authoring.JSONCompletionOptions{FallbackOnStructuredError: true})
	if err != nil {
		return fmt.Errorf("%s: %w", stage, err)
	}
	return nil
}

type rolloutChatAdapter struct {
	Client      rollout.ChatClient
	Temperature *float64
	MaxTokens   int
}

func (adapter rolloutChatAdapter) Complete(ctx context.Context, transcript []authoring.TranscriptTurn) (authoring.TranscriptTurn, error) {
	if adapter.Client == nil {
		return authoring.TranscriptTurn{}, fmt.Errorf("rollout chat client is required")
	}
	messages := make([]rollout.ChatMessage, 0, len(transcript))
	for _, turn := range transcript {
		messages = append(messages, rollout.ChatMessage{Role: turn.Role, Content: turn.Content})
	}
	content, err := adapter.Client.Chat(ctx, messages)
	if err != nil {
		return authoring.TranscriptTurn{}, err
	}
	return authoring.TranscriptTurn{Role: "assistant", Content: content}, nil
}

func (adapter rolloutChatAdapter) CompleteStructured(ctx context.Context, transcript []authoring.TranscriptTurn, schema any, out any) error {
	if adapter.Client == nil {
		return fmt.Errorf("rollout chat client is required")
	}
	structured, ok := adapter.Client.(rollout.StructuredChat)
	if !ok {
		return fmt.Errorf("structured chat unavailable")
	}
	rawSchema, err := authoring.RawSchema(schema)
	if err != nil {
		return err
	}
	messages := make([]rollout.ChatMessage, 0, len(transcript))
	for _, turn := range transcript {
		messages = append(messages, rollout.ChatMessage{Role: turn.Role, Content: turn.Content})
	}
	raw, err := structured.StructuredChat(ctx, messages, rawSchema, rollout.StructuredOpts{
		Temperature: adapter.Temperature,
		MaxTokens:   adapter.MaxTokens,
	})
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(raw), out)
}

func firstLine(value string) string {
	value = strings.TrimSpace(value)
	if idx := strings.IndexAny(value, "\r\n"); idx >= 0 {
		value = value[:idx]
	}
	if len(value) > 140 {
		return value[:137] + "..."
	}
	return value
}

func sanitizeKickoff(session Session) Session {
	workflow := session.Intent.Workflow
	session.Intent = rollout.Intent{Workflow: workflow}
	session.Project.Inputs = ""
	session.Project.Outputs = ""
	session.Project.DataFlow = ""
	session.Project.FunctionContracts = ""
	session.Project.OpenAPI = ""
	session.Project.UsesOpenAPI = false
	session.Project.CmdApproved = false
	session.Project.SSHApproved = false
	session.Credentials = credentialBindings(strings.Join(session.Credentials, ","))
	session.Project.Credentials = credentialBindings(strings.Join(session.Project.Credentials, ","))
	session.SideEffectScope = projectwizard.NormalizeSideEffectScope(session.SideEffectScope)
	return session
}

type draftCompletion struct {
	Session
	RequestedOperationIDs []string `json:"requested_operation_ids,omitempty"`
	DetailRequestReason   string   `json:"detail_request_reason,omitempty"`
}

func draftPromptRequest(request DraftRequest) map[string]any {
	return draftPromptRequestWithDetails(request, draftDetailRefs(request, nil))
}

func draftPromptRequestWithDetails(request DraftRequest, detailRefs []OperationDetailRef) map[string]any {
	draftDocs := detailDocuments(request, detailRefs)
	if len(draftDocs) == 0 {
		draftDocs = rankedDraftDocuments(request)
	}
	docs := make([]map[string]any, 0, len(draftDocs))
	for _, doc := range draftDocs {
		ops := make([]operationPromptContext, 0, len(doc.Operations))
		for _, op := range doc.Operations {
			ops = append(ops, operationPrompt(op))
		}
		docs = append(docs, map[string]any{
			"path":        doc.RelativePath,
			"title":       doc.Title,
			"description": doc.Description,
			"operations":  ops,
		})
	}
	return map[string]any{
		"opening":            request.Opening,
		"session":            request.Session,
		"docs":               docs,
		"operation_catalog":  operationCatalog(request.Docs),
		"transcript_turns":   request.TranscriptTurns,
		"readiness_feedback": request.ReadinessFeedback,
	}
}

func draftDetailRefs(request DraftRequest, requested []OperationDetailRef) []OperationDetailRef {
	refs := operationRefsFromDocuments(selectedDraftDocuments(request))
	if len(refs) == 0 {
		refs = operationRefsFromDocuments(rankedDraftDocuments(request))
	}
	return appendOperationDetailRefs(refs, requested)
}

func detailDocuments(request DraftRequest, refs []OperationDetailRef) []APIDocument {
	if len(refs) == 0 {
		return nil
	}
	docByPath := map[string]APIDocument{}
	opByKey := map[string]apitoolsOperation{}
	for docIndex, doc := range request.Docs {
		docByPath[doc.RelativePath] = doc
		for _, op := range doc.Operations {
			opByKey[operationCandidateKey(doc.RelativePath, op.OperationID)] = apitoolsOperation{docIndex: docIndex, op: op}
			if _, ok := opByKey["\x00"+op.OperationID]; !ok {
				opByKey["\x00"+op.OperationID] = apitoolsOperation{docIndex: docIndex, op: op}
			}
		}
	}
	var out []APIDocument
	docIndex := map[string]int{}
	for _, ref := range refs {
		found, ok := opByKey[operationCandidateKey(ref.DocumentPath, ref.OperationID)]
		if !ok && strings.TrimSpace(ref.DocumentPath) == "" {
			found, ok = opByKey["\x00"+ref.OperationID]
		}
		if !ok {
			continue
		}
		doc := request.Docs[found.docIndex]
		index, ok := docIndex[doc.RelativePath]
		if !ok {
			copyDoc := docByPath[doc.RelativePath]
			copyDoc.Operations = nil
			out = append(out, copyDoc)
			index = len(out) - 1
			docIndex[doc.RelativePath] = index
		}
		out[index].Operations = append(out[index].Operations, found.op)
	}
	return out
}

type apitoolsOperation struct {
	docIndex int
	op       apitools.OperationSummary
}

func operationRefsFromDocuments(docs []APIDocument) []OperationDetailRef {
	var refs []OperationDetailRef
	for _, doc := range docs {
		for _, op := range doc.Operations {
			if strings.TrimSpace(op.OperationID) == "" {
				continue
			}
			refs = append(refs, OperationDetailRef{DocumentPath: doc.RelativePath, OperationID: op.OperationID})
		}
	}
	return refs
}

func appendOperationDetailRefs(base []OperationDetailRef, refs []OperationDetailRef) []OperationDetailRef {
	seen := map[string]bool{}
	out := make([]OperationDetailRef, 0, len(base)+len(refs))
	for _, ref := range append(append([]OperationDetailRef(nil), base...), refs...) {
		ref.DocumentPath = strings.TrimSpace(ref.DocumentPath)
		ref.OperationID = strings.TrimSpace(ref.OperationID)
		if ref.OperationID == "" {
			continue
		}
		key := operationCandidateKey(ref.DocumentPath, ref.OperationID)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, ref)
	}
	return out
}

func resolveDraftDetailRequests(docs []APIDocument, ids []string, existing []OperationDetailRef) ([]OperationDetailRef, []string, bool) {
	existingKeys := map[string]bool{}
	for _, ref := range existing {
		existingKeys[operationCandidateKey(ref.DocumentPath, ref.OperationID)] = true
	}
	var requested []string
	seenIDs := map[string]bool{}
	capped := false
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" || seenIDs[id] {
			continue
		}
		if len(requested) >= maxDraftRequestedOperations {
			capped = true
			continue
		}
		seenIDs[id] = true
		requested = append(requested, id)
	}
	var valid []OperationDetailRef
	var rejected []string
	for _, id := range requested {
		ref, ok := findOperationDetailRef(docs, id)
		if !ok {
			rejected = append(rejected, id)
			continue
		}
		if existingKeys[operationCandidateKey(ref.DocumentPath, ref.OperationID)] {
			continue
		}
		valid = append(valid, ref)
	}
	return valid, rejected, capped
}

func findOperationDetailRef(docs []APIDocument, operationID string) (OperationDetailRef, bool) {
	for _, doc := range docs {
		for _, op := range doc.Operations {
			if op.OperationID == operationID {
				return OperationDetailRef{DocumentPath: doc.RelativePath, OperationID: op.OperationID}, true
			}
		}
	}
	return OperationDetailRef{}, false
}

func operationRefIDs(refs []OperationDetailRef) []string {
	out := make([]string, 0, len(refs))
	for _, ref := range refs {
		out = append(out, ref.OperationID)
	}
	return out
}

func selectedDraftDocuments(request DraftRequest) []APIDocument {
	var out []APIDocument
	docIndex := map[string]int{}
	seenOps := map[string]bool{}
	walkSteps(request.Session.Intent.Steps, func(step *rollout.Step) {
		op, ok := operationForStep(request.Session, request.Docs, step)
		if !ok {
			return
		}
		doc, ok := documentForStep(request.Session, request.Docs, step, op)
		if !ok || doc.RelativePath == "" {
			return
		}
		index, ok := docIndex[doc.RelativePath]
		if !ok {
			copyDoc := doc
			copyDoc.Operations = nil
			out = append(out, copyDoc)
			index = len(out) - 1
			docIndex[doc.RelativePath] = index
		}
		key := doc.RelativePath + "\x00" + op.OperationID
		if seenOps[key] {
			return
		}
		seenOps[key] = true
		out[index].Operations = append(out[index].Operations, *op)
	})
	return out
}

func sanitizeDraft(request DraftRequest, draft Session) Session {
	return sanitizeDraftWithDetails(request, draft, draftDetailRefs(request, draft.DraftOperations))
}

func sanitizeDraftWithDetails(request DraftRequest, draft Session, detailRefs []OperationDetailRef) Session {
	allowedDocs := map[string]bool{}
	allowedOps := map[string]bool{}
	for _, doc := range request.Docs {
		allowedDocs[doc.RelativePath] = true
	}
	if len(detailRefs) == 0 {
		detailRefs = draftDetailRefs(request, nil)
	}
	for _, ref := range detailRefs {
		if strings.TrimSpace(ref.OperationID) == "" {
			continue
		}
		if ref.DocumentPath != "" {
			allowedOps[operationCandidateKey(ref.DocumentPath, ref.OperationID)] = true
		}
		allowedOps["\x00"+ref.OperationID] = true
	}
	for _, doc := range request.Docs {
		for _, op := range doc.Operations {
			if op.OperationID == "" {
				continue
			}
			if allowedOps[operationCandidateKey(doc.RelativePath, op.OperationID)] || allowedOps["\x00"+op.OperationID] {
				allowedOps[operationCandidateKey(doc.RelativePath, op.OperationID)] = true
			}
		}
	}
	if len(allowedDocs) > 0 && draft.Intent.Source != "" && !allowedDocs[draft.Intent.Source] {
		draft.Intent.Source = ""
	}
	if len(allowedDocs) > 0 && draft.Intent.OpenAPI != "" && !allowedDocs[draft.Intent.OpenAPI] {
		draft.Intent.OpenAPI = ""
	}
	walkSteps(draft.Intent.Steps, func(step *rollout.Step) {
		docPath := firstNonEmpty(step.Source, step.OpenAPI, draft.Intent.Source, draft.Intent.OpenAPI)
		if len(allowedDocs) > 0 && step.Source != "" && !allowedDocs[step.Source] {
			step.Source = ""
			docPath = firstNonEmpty(step.OpenAPI, draft.Intent.Source, draft.Intent.OpenAPI)
		}
		if len(allowedDocs) > 0 && step.OpenAPI != "" && !allowedDocs[step.OpenAPI] {
			step.OpenAPI = ""
			docPath = firstNonEmpty(step.Source, draft.Intent.Source, draft.Intent.OpenAPI)
		}
		if step.Operation != "" && !allowedOps[docPath+"\x00"+step.Operation] && !allowedOps["\x00"+step.Operation] {
			step.Operation = ""
		}
	})
	draft.Credentials = credentialBindings(strings.Join(draft.Credentials, ","))
	draft.Project.Credentials = credentialBindings(strings.Join(draft.Project.Credentials, ","))
	draft.SideEffectScope = projectwizard.NormalizeSideEffectScope(draft.SideEffectScope)
	if len(draft.Assumptions) == 0 && !emptySession(draft) {
		draft.Assumptions = []Assumption{{
			ID:                   "ai_draft",
			Slot:                 "intent",
			Value:                "AI-assisted draft",
			Reason:               "Drafted from the workflow brief and available local OpenAPI metadata.",
			Evidence:             draftEvidence(request),
			Risk:                 "review",
			RequiresConfirmation: true,
		}}
	}
	if !emptySession(draft) {
		draft.Annotations = append(draft.Annotations, SourceAnnotation{
			Slot:          "draft",
			Source:        "llm",
			PromptVersion: PromptVersion,
			Evidence:      draftEvidence(request),
		})
	}
	draft.Normalize()
	return draft
}

func draftWarningAssumption(id, message string) Assumption {
	return Assumption{
		ID:                   id,
		Slot:                 "draft.requested_operation_ids",
		Value:                "operation detail request rejected",
		Reason:               message,
		Evidence:             "LLM requested operation details outside the allowed local catalog or loop budget.",
		Risk:                 "warning",
		RequiresConfirmation: true,
	}
}

func draftEvidence(request DraftRequest) string {
	description := ""
	if request.Session.Intent.Workflow != nil {
		description = request.Session.Intent.Workflow.Description
	}
	return firstLine(firstNonEmpty(request.Opening, description))
}

func sanitizeRefine(base, refined Session) Session {
	out := base
	out.Project = projectProse(base.Project, refined.Project)
	out.Safety = firstNonEmpty(refined.Safety, base.Safety)
	out.Fallback = firstNonEmpty(refined.Fallback, base.Fallback)
	if out.Intent.Workflow != nil && refined.Intent.Workflow != nil {
		out.Intent.Workflow.Description = firstNonEmpty(refined.Intent.Workflow.Description, out.Intent.Workflow.Description)
	}
	out.Project.Goal = firstNonEmpty(refined.Project.Goal, out.Project.Goal)
	out.Project.Safety = firstNonEmpty(refined.Project.Safety, out.Project.Safety)
	out.Project.Fallback = firstNonEmpty(refined.Project.Fallback, out.Project.Fallback)
	out.Annotations = append(out.Annotations, SourceAnnotation{
		Slot:          "refine",
		Source:        "llm",
		PromptVersion: PromptVersion,
	})
	return out
}

func projectProse(base, refined projectwizard.Answers) projectwizard.Answers {
	base.Goal = firstNonEmpty(refined.Goal, base.Goal)
	base.Safety = firstNonEmpty(refined.Safety, base.Safety)
	base.Fallback = firstNonEmpty(refined.Fallback, base.Fallback)
	return base
}

const kickoffSchema = `{
  "type": "object",
  "additionalProperties": false,
  "properties": {
    "project": {"type": "object", "additionalProperties": true},
    "intent": {"type": "object", "additionalProperties": true},
    "credentials": {"type": "array", "items": {"type": "string"}},
    "safety": {"type": "string"},
    "fallback": {"type": "string"},
    "side_effect_scope": {"type": "string"},
    "annotations": {"type": "array", "items": {"type": "object", "additionalProperties": true}}
  }
}`

const disambiguateSchema = `{
  "type": "object",
  "additionalProperties": false,
  "properties": {
    "paths": {"type": "array", "items": {"type": "string"}}
  }
}`

const catalogPlanSchema = `{
  "type": "object",
  "additionalProperties": false,
  "properties": {
    "selected_artifacts": {
      "type": "array",
      "items": {
        "type": "object",
        "additionalProperties": false,
        "properties": {
          "provider_id": {"type": "string"},
          "artifact_key": {"type": "string"},
          "reason": {"type": "string"}
        }
      }
    },
    "proposed_steps": {
      "type": "array",
      "items": {
        "type": "object",
        "additionalProperties": false,
        "properties": {
          "name": {"type": "string"},
          "type": {"type": "string"},
          "provider": {"type": "string"},
          "openapi": {"type": "string"},
          "do": {"type": "string"},
          "depends_on": {"type": "array", "items": {"type": "string"}}
        }
      }
    },
    "blockers": {"type": "array", "items": {"type": "string"}},
    "assumptions": {"type": "array", "items": {"type": "string"}}
  }
}`

const requestMappingsSchema = `{
  "type": "object",
  "additionalProperties": false,
  "properties": {
    "steps": {
      "type": "array",
      "items": {
        "type": "object",
        "additionalProperties": false,
        "properties": {
          "name": {"type": "string"},
          "with": {"type": "object", "additionalProperties": {"type": "string"}}
        }
      }
    },
    "assumptions": {"type": "array", "items": {"type": "string"}},
    "blockers": {"type": "array", "items": {"type": "string"}}
  }
}`

const draftReviewSchema = `{
  "type": "object",
  "additionalProperties": false,
  "properties": {
    "issues": {
      "type": "array",
      "items": {
        "type": "object",
        "additionalProperties": false,
        "properties": {
          "severity": {"type": "string"},
          "code": {"type": "string"},
          "message": {"type": "string"},
          "slot": {"type": "string"},
          "suggested_answer": {"type": "string"},
          "evidence": {"type": "string"}
        }
      }
    }
  }
}`

const draftSchema = `{
  "type": "object",
  "additionalProperties": false,
  "properties": {
    "project": {"type": "object", "additionalProperties": true},
    "intent": {"type": "object", "additionalProperties": true},
    "credentials": {"type": "array", "items": {"type": "string"}},
    "credentials_set": {"type": "boolean"},
    "safety": {"type": "string"},
    "safety_set": {"type": "boolean"},
    "fallback": {"type": "string"},
    "fallback_set": {"type": "boolean"},
    "side_effect_scope": {"type": "string"},
    "annotations": {"type": "array", "items": {"type": "object", "additionalProperties": true}},
    "assumptions": {"type": "array", "items": {"type": "object", "additionalProperties": true}},
    "requested_operation_ids": {"type": "array", "items": {"type": "string"}},
    "detail_request_reason": {"type": "string"}
  }
}`
