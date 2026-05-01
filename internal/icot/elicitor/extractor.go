package elicitor

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"strings"

	"github.com/genelet/ramen/internal/projectwizard"
	"github.com/genelet/udon/pkg/rollout"
	"github.com/tabilet/apitools"
)

const PromptVersion = "icot-extractor.v1"

//go:embed prompts/kickoff.txt
var kickoffPrompt string

//go:embed prompts/refine.txt
var refinePrompt string

//go:embed prompts/disambiguate.txt
var disambiguatePrompt string

//go:embed prompts/draft.txt
var draftPrompt string

// Extractor provides optional, bounded LLM assistance for the interactive
// authoring loop. Implementations must return partial drafts only; the loop
// still asks the user to confirm the final intent before saving.
type Extractor interface {
	Kickoff(context.Context, string) (Session, error)
	Draft(context.Context, DraftRequest) (Session, error)
	Refine(context.Context, Session) (Session, error)
	Disambiguate(context.Context, string, []APIDocument) ([]string, error)
}

type DraftRequest struct {
	Opening           string           `json:"opening"`
	Session           Session          `json:"session"`
	Docs              []APIDocument    `json:"docs"`
	TranscriptTurns   []ReplayTurn     `json:"transcript_turns,omitempty"`
	ReadinessFeedback []ReadinessIssue `json:"readiness_feedback,omitempty"`
}

type noopExtractor struct{}

func NewNoopExtractor() Extractor {
	return noopExtractor{}
}

func (noopExtractor) Kickoff(context.Context, string) (Session, error) {
	return Session{}, nil
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
	messages := []rollout.ChatMessage{
		{
			Role:    "system",
			Content: kickoffPrompt,
		},
		{Role: "user", Content: opening},
	}
	raw, err := e.structured(ctx, messages, kickoffSchema)
	if err != nil {
		raw, err = e.client.Chat(ctx, messages)
	}
	if err != nil {
		return Session{}, err
	}
	var session Session
	if err := decodeJSONBlock(raw, &session); err != nil {
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

func (e *chatExtractor) Draft(ctx context.Context, request DraftRequest) (Session, error) {
	request.Opening = strings.TrimSpace(request.Opening)
	currentDescription := ""
	if request.Session.Intent.Workflow != nil {
		currentDescription = request.Session.Intent.Workflow.Description
	}
	if request.Opening == "" && strings.TrimSpace(currentDescription) == "" {
		return Session{}, nil
	}
	data, err := json.Marshal(draftPromptRequest(request))
	if err != nil {
		return Session{}, err
	}
	messages := []rollout.ChatMessage{
		{
			Role:    "system",
			Content: draftPrompt,
		},
		{Role: "user", Content: string(data)},
	}
	raw, err := e.structured(ctx, messages, draftSchema)
	if err != nil {
		raw, err = e.client.Chat(ctx, messages)
	}
	if err != nil {
		return Session{}, err
	}
	var session Session
	if err := decodeJSONBlock(raw, &session); err != nil {
		return Session{}, err
	}
	return sanitizeDraft(request, session), nil
}

func (e *chatExtractor) Refine(ctx context.Context, session Session) (Session, error) {
	data, err := json.Marshal(session)
	if err != nil {
		return session, err
	}
	messages := []rollout.ChatMessage{
		{
			Role:    "system",
			Content: refinePrompt,
		},
		{Role: "user", Content: string(data)},
	}
	raw, err := e.structured(ctx, messages, kickoffSchema)
	if err != nil {
		raw, err = e.client.Chat(ctx, messages)
	}
	if err != nil {
		return session, err
	}
	var refined Session
	if err := decodeJSONBlock(raw, &refined); err != nil {
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
	messages := []rollout.ChatMessage{
		{
			Role:    "system",
			Content: disambiguatePrompt,
		},
		{Role: "user", Content: b.String()},
	}
	raw, err := e.structured(ctx, messages, disambiguateSchema)
	if err != nil {
		raw, err = e.client.Chat(ctx, messages)
	}
	if err != nil {
		return nil, err
	}
	var decoded struct {
		Paths []string `json:"paths"`
	}
	if err := decodeJSONBlock(raw, &decoded); err != nil {
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

func (e *chatExtractor) structured(ctx context.Context, messages []rollout.ChatMessage, schema string) (string, error) {
	structured, ok := e.client.(rollout.StructuredChat)
	if !ok {
		return "", errors.New("structured chat unavailable")
	}
	return structured.StructuredChat(ctx, messages, json.RawMessage(schema), rollout.StructuredOpts{
		Temperature: e.temperature,
		MaxTokens:   1200,
	})
}

func decodeJSONBlock(raw string, target any) error {
	return apitools.DecodeJSONBlock(raw, target)
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

func draftPromptRequest(request DraftRequest) map[string]any {
	rankedDocs := rankedDraftDocuments(request)
	docs := make([]map[string]any, 0, len(rankedDocs))
	for _, doc := range rankedDocs {
		ops := make([]operationPromptContext, 0, len(doc.Operations))
		for _, op := range doc.Operations {
			if op == nil {
				continue
			}
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
		"transcript_turns":   request.TranscriptTurns,
		"readiness_feedback": request.ReadinessFeedback,
	}
}

func sanitizeDraft(request DraftRequest, draft Session) Session {
	allowedDocs := map[string]bool{}
	allowedOps := map[string]bool{}
	for _, doc := range request.Docs {
		allowedDocs[doc.RelativePath] = true
		for _, op := range doc.Operations {
			if op != nil && op.OperationID != "" {
				allowedOps[doc.RelativePath+"\x00"+op.OperationID] = true
				allowedOps["\x00"+op.OperationID] = true
			}
		}
	}
	if draft.Intent.OpenAPI != "" && !allowedDocs[draft.Intent.OpenAPI] {
		draft.Intent.OpenAPI = ""
	}
	walkSteps(draft.Intent.Steps, func(step *rollout.Step) {
		docPath := firstNonEmpty(step.OpenAPI, draft.Intent.OpenAPI)
		if step.OpenAPI != "" && !allowedDocs[step.OpenAPI] {
			step.OpenAPI = ""
			docPath = draft.Intent.OpenAPI
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
    "assumptions": {"type": "array", "items": {"type": "object", "additionalProperties": true}}
  }
}`
