package workflowintent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/genelet/udon/pkg/rollout"
	"github.com/genelet/udon/pkg/runner"
	"github.com/tabilet/apitools"
)

const IntentPath = "workflows/intent.hcl"

type WorkflowAdapter struct{}

func WorkflowFlow() openapisearch.Flow[*rollout.Intent] {
	adapter := WorkflowAdapter{}
	return openapisearch.Flow[*rollout.Intent]{
		Parser:       adapter,
		Renderer:     adapter,
		Validator:    adapter,
		SlotProvider: adapter,
	}
}

func ParseFile(ctx context.Context, path string) (*rollout.Intent, error) {
	draft, diagnostics, err := WorkflowAdapter{}.ParseIntent(ctx, openapisearch.Artifact{Path: path})
	if err != nil {
		return nil, err
	}
	if openapisearch.HasErrors(diagnostics) {
		return nil, openapisearch.DiagnosticError{Diagnostics: diagnostics}
	}
	return draft, nil
}

func RenderHCL(ctx context.Context, draft *rollout.Intent) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	flow := WorkflowFlow()
	diagnostics := flow.Validator.ValidateIntent(ctx, draft)
	if openapisearch.HasErrors(diagnostics) {
		return "", openapisearch.DiagnosticError{Diagnostics: diagnostics}
	}
	set, renderDiagnostics, err := flow.Renderer.RenderIntent(ctx, draft)
	diagnostics = append(diagnostics, renderDiagnostics...)
	if err != nil {
		return "", err
	}
	if openapisearch.HasErrors(diagnostics) {
		return "", openapisearch.DiagnosticError{Diagnostics: diagnostics}
	}
	for _, artifact := range set.Artifacts {
		if artifact.Path == IntentPath || artifact.MediaType == "text/hcl" {
			return string(artifact.Content), nil
		}
	}
	return "", fmt.Errorf("rendered intent artifact %q not found", IntentPath)
}

func ValidateComplete(ctx context.Context, draft *rollout.Intent) []openapisearch.Diagnostic {
	adapter := WorkflowAdapter{}
	diagnostics := adapter.ValidateIntent(ctx, draft)
	for _, slot := range adapter.MissingSlots(ctx, draft) {
		if !slot.Required {
			continue
		}
		diagnostics = append(diagnostics, openapisearch.Diagnostic{
			Severity: "error",
			Code:     "missing_slot",
			Message:  fmt.Sprintf("required slot %q is missing", slot.Name),
			Path:     slot.Name,
		})
	}
	return diagnostics
}

func (WorkflowAdapter) ParseIntent(ctx context.Context, artifact openapisearch.Artifact) (*rollout.Intent, []openapisearch.Diagnostic, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	path := strings.TrimSpace(artifact.Path)
	data := artifact.Content
	if len(data) == 0 && path != "" {
		var err error
		data, err = os.ReadFile(path)
		if err != nil {
			return nil, []openapisearch.Diagnostic{diagnostic("error", "intent_read_failed", err.Error(), path)}, err
		}
	}
	if path == "" {
		path = IntentPath
	}
	draft, err := rollout.ParseIntent(data, path)
	if err != nil {
		return nil, []openapisearch.Diagnostic{diagnostic("error", "intent_parse_failed", err.Error(), path)}, err
	}
	return draft, nil, nil
}

func (WorkflowAdapter) RenderIntent(ctx context.Context, draft *rollout.Intent) (openapisearch.ArtifactSet, []openapisearch.Diagnostic, error) {
	if err := ctx.Err(); err != nil {
		return openapisearch.ArtifactSet{}, nil, err
	}
	hcl, err := runner.RenderIntentHCL(draft)
	if err != nil {
		return openapisearch.ArtifactSet{}, []openapisearch.Diagnostic{diagnostic("error", "intent_render_failed", err.Error(), IntentPath)}, err
	}
	if _, err := rollout.ParseIntent([]byte(hcl), IntentPath); err != nil {
		return openapisearch.ArtifactSet{}, []openapisearch.Diagnostic{diagnostic("error", "intent_render_invalid", err.Error(), IntentPath)}, err
	}
	return openapisearch.ArtifactSet{Artifacts: []openapisearch.Artifact{{
		Path:      IntentPath,
		MediaType: "text/hcl",
		Content:   []byte(hcl),
	}}}, nil, nil
}

func (WorkflowAdapter) ValidateIntent(ctx context.Context, draft *rollout.Intent) []openapisearch.Diagnostic {
	if err := ctx.Err(); err != nil {
		return []openapisearch.Diagnostic{diagnostic("error", "intent_validation_cancelled", err.Error(), "")}
	}
	if draft == nil {
		return []openapisearch.Diagnostic{diagnostic("error", "intent_required", "intent is required", "")}
	}
	hcl, err := runner.RenderIntentHCL(draft)
	if err != nil {
		return []openapisearch.Diagnostic{diagnostic("error", "intent_render_failed", err.Error(), IntentPath)}
	}
	if _, err := rollout.ParseIntent([]byte(hcl), IntentPath); err != nil {
		return []openapisearch.Diagnostic{diagnostic("error", "intent_parse_failed", err.Error(), IntentPath)}
	}
	return nil
}

func (WorkflowAdapter) MissingSlots(ctx context.Context, draft *rollout.Intent) []openapisearch.Slot {
	_ = ctx
	var missing []string
	if draft == nil {
		return []openapisearch.Slot{{Name: "intent", Required: true}}
	}
	if draft.Workflow == nil || strings.TrimSpace(draft.Workflow.Name) == "" {
		missing = append(missing, "workflow name")
	}
	if draft.Workflow == nil || strings.TrimSpace(draft.Workflow.Description) == "" {
		missing = append(missing, "workflow goal")
	}
	missing = append(missing, draft.MissingSlots()...)
	collectOperationSlots(&missing, draft.OpenAPI, draft.Steps)
	if len(draft.Outputs) == 0 {
		missing = append(missing, "at least one output")
	}
	missing = dedupe(missing)
	slots := make([]openapisearch.Slot, 0, len(missing))
	for _, name := range missing {
		slots = append(slots, openapisearch.Slot{Name: name, Required: true})
	}
	return slots
}

type ChatAdapter struct {
	Client      rollout.ChatClient
	Temperature *float64
	MaxTokens   int
}

func (adapter ChatAdapter) Complete(ctx context.Context, transcript []openapisearch.TranscriptTurn) (openapisearch.TranscriptTurn, error) {
	if adapter.Client == nil {
		return openapisearch.TranscriptTurn{}, fmt.Errorf("rollout chat client is required")
	}
	content, err := adapter.Client.Chat(ctx, TranscriptToMessages(transcript))
	if err != nil {
		return openapisearch.TranscriptTurn{}, err
	}
	return openapisearch.TranscriptTurn{Role: "assistant", Content: content}, nil
}

func (adapter ChatAdapter) CompleteStructured(ctx context.Context, transcript []openapisearch.TranscriptTurn, schema any, out any) error {
	if adapter.Client == nil {
		return fmt.Errorf("rollout chat client is required")
	}
	structured, ok := adapter.Client.(rollout.StructuredChat)
	if !ok {
		return fmt.Errorf("structured chat unavailable")
	}
	rawSchema, err := openapisearch.RawSchema(schema)
	if err != nil {
		return err
	}
	raw, err := structured.StructuredChat(ctx, TranscriptToMessages(transcript), rawSchema, rollout.StructuredOpts{
		Temperature: adapter.Temperature,
		MaxTokens:   adapter.MaxTokens,
	})
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(raw), out)
}

func TranscriptToMessages(transcript []openapisearch.TranscriptTurn) []rollout.ChatMessage {
	messages := make([]rollout.ChatMessage, 0, len(transcript))
	for _, turn := range transcript {
		messages = append(messages, rollout.ChatMessage{Role: turn.Role, Content: turn.Content})
	}
	return messages
}

func MessagesToTranscript(messages []rollout.ChatMessage) []openapisearch.TranscriptTurn {
	transcript := make([]openapisearch.TranscriptTurn, 0, len(messages))
	for _, message := range messages {
		transcript = append(transcript, openapisearch.TranscriptTurn{Role: message.Role, Content: message.Content})
	}
	return transcript
}

func collectOperationSlots(missing *[]string, defaultOpenAPI string, steps []*rollout.Step) {
	for _, step := range steps {
		if step == nil {
			continue
		}
		stepOpenAPI := firstNonEmpty(step.OpenAPI, defaultOpenAPI)
		stepType := strings.ToLower(strings.TrimSpace(step.Type))
		if (stepType == "http" || stepType == "openapi") && strings.TrimSpace(stepOpenAPI) != "" && strings.TrimSpace(step.Operation) == "" {
			*missing = append(*missing, "operation for step "+firstNonEmpty(step.Name, "unnamed"))
		}
		collectOperationSlots(missing, stepOpenAPI, step.Steps)
		for _, branch := range step.Cases {
			if branch != nil {
				collectOperationSlots(missing, stepOpenAPI, branch.Steps)
			}
		}
		if step.Default != nil {
			collectOperationSlots(missing, stepOpenAPI, step.Default.Steps)
		}
	}
}

func diagnostic(severity, code, message, path string) openapisearch.Diagnostic {
	return openapisearch.Diagnostic{
		Severity: severity,
		Code:     code,
		Message:  strings.TrimSpace(message),
		Path:     path,
	}
}

func dedupe(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
