package synthesize

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/genelet/ramen/internal/openapidisco"
	"github.com/genelet/udon/pkg/rollout"
	"github.com/genelet/udon/pkg/runner"
)

const (
	maxPromptOperationsTotal       = 80
	maxPromptDescriptionChars      = 400
	maxPromptOperationSummaryChars = 160
	maxPromptResponseFields        = 12
	intentGenerationModeStructured = "structured"
	intentGenerationModeLegacy     = "legacy"
)

//go:embed prompts/intent_generation.tmpl
var intentGenerationSystemPrompt string

//go:embed prompts/examples/*.json
var intentPromptExamples embed.FS

//go:embed schemas/intent.schema.json
var embeddedIntentSchema []byte

func generateIntent(ctx context.Context, chat rollout.ChatClient, projectText string, candidates []openapidisco.Candidate, primary string, policy projectPolicy, feedback string) (*rollout.Intent, error) {
	intent, _, err := generateIntentWithMode(ctx, chat, projectText, candidates, primary, policy, feedback, nil)
	return intent, err
}

func generateIntentWithMode(ctx context.Context, chat rollout.ChatClient, projectText string, candidates []openapidisco.Candidate, primary string, policy projectPolicy, feedback string, temperature *float64) (*rollout.Intent, string, error) {
	messages := intentPromptMessagesForMode(projectText, candidates, primary, policy, feedback, supportsStructuredChat(chat))
	return generateIntentFromMessagesWithMode(ctx, chat, messages, primary, policy, temperature)
}

func intentPromptMessages(projectText string, candidates []openapidisco.Candidate, primary string, policy projectPolicy, feedback string) []rollout.ChatMessage {
	return intentPromptMessagesForMode(projectText, candidates, primary, policy, feedback, false)
}

func intentPromptMessagesForMode(projectText string, candidates []openapidisco.Candidate, primary string, policy projectPolicy, feedback string, structured bool) []rollout.ChatMessage {
	feedback = strings.TrimSpace(feedback)
	feedbackSection := ""
	if feedback != "" {
		feedbackSection = "\nPrevious quality failure to repair:\n" + feedback + "\n"
	}
	user := fmt.Sprintf("Project brief:\n%s\n\nRuntime and OpenAPI policy:\n%s\n%sAvailable OpenAPI documents:\n%s\n\nPrimary OpenAPI path: %s\n%s\nReturn JSON matching this shape:\n%s",
		projectText,
		runtimePolicyPrompt(policy),
		requiredByProjectPrompt(policy),
		specSummary(candidates),
		primary,
		feedbackSection,
		intentJSONShape(),
	)
	return []rollout.ChatMessage{
		{Role: "system", Content: renderIntentSystemPromptForMode(structured)},
		{Role: "user", Content: user},
	}
}

func generateIntentFromMessages(ctx context.Context, chat rollout.ChatClient, messages []rollout.ChatMessage, primary string, policy projectPolicy) (*rollout.Intent, error) {
	intent, _, err := generateIntentFromMessagesWithMode(ctx, chat, messages, primary, policy, nil)
	return intent, err
}

func generateIntentFromMessagesWithMode(ctx context.Context, chat rollout.ChatClient, messages []rollout.ChatMessage, primary string, policy projectPolicy, temperature *float64) (*rollout.Intent, string, error) {
	if structured, ok := chat.(rollout.StructuredChat); ok {
		raw, err := structured.StructuredChat(ctx, messages, json.RawMessage(embeddedIntentSchema), rollout.StructuredOpts{Temperature: temperature})
		if err == nil {
			intent, err := decodeIntentJSON(raw, primary, policy)
			if err != nil {
				return nil, intentGenerationModeStructured, err
			}
			return intent, intentGenerationModeStructured, nil
		}
		messages = legacyJSONInstructionMessages(messages)
	}
	response, err := chat.Chat(ctx, messages)
	if err != nil {
		return nil, intentGenerationModeLegacy, err
	}
	jsonText, err := extractJSON(response)
	if err != nil {
		return nil, intentGenerationModeLegacy, fmt.Errorf("extract intent JSON: %w", err)
	}
	intent, err := decodeIntentJSON(jsonText, primary, policy)
	if err != nil {
		return nil, intentGenerationModeLegacy, err
	}
	return intent, intentGenerationModeLegacy, nil
}

func legacyJSONInstructionMessages(messages []rollout.ChatMessage) []rollout.ChatMessage {
	const instruction = "Return only JSON. Do not include Markdown."
	out := append([]rollout.ChatMessage(nil), messages...)
	for i := range out {
		if strings.Contains(out[i].Content, instruction) {
			return out
		}
	}
	for i := len(out) - 1; i >= 0; i-- {
		if strings.TrimSpace(out[i].Role) == "user" {
			out[i].Content = strings.TrimSpace(out[i].Content) + "\n\n" + instruction
			return out
		}
	}
	return out
}

func decodeIntentJSON(jsonText string, primary string, policy projectPolicy) (*rollout.Intent, error) {
	var intent rollout.Intent
	if err := json.Unmarshal([]byte(jsonText), &intent); err != nil {
		return nil, fmt.Errorf("decode intent JSON: %w", err)
	}
	if strings.TrimSpace(intent.OpenAPI) == "" && primary != "" && !policy.NoOpenAPI {
		intent.OpenAPI = primary
	}
	intent.EnsureActionDescriptions()
	if _, err := runner.RenderIntentHCL(&intent); err != nil {
		return nil, err
	}
	return &intent, nil
}

func renderIntentSystemPrompt() string {
	return renderIntentSystemPromptForMode(false)
}

func renderIntentSystemPromptForMode(structured bool) string {
	prompt := strings.ReplaceAll(strings.TrimSpace(intentGenerationSystemPrompt), "{{EXAMPLES}}", promptExamplesBlock())
	if structured {
		prompt = strings.ReplaceAll(prompt, "Return only JSON. Do not include Markdown.\n", "")
		prompt = strings.ReplaceAll(prompt, "Return only JSON. Do not include Markdown.", "")
	}
	return strings.TrimSpace(prompt)
}

func supportsStructuredChat(chat rollout.ChatClient) bool {
	_, ok := chat.(rollout.StructuredChat)
	return ok
}

func promptExamplesBlock() string {
	entries, err := intentPromptExamples.ReadDir("prompts/examples")
	if err != nil {
		return ""
	}
	var names []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	var b strings.Builder
	for _, name := range names {
		data, err := intentPromptExamples.ReadFile(filepath.Join("prompts/examples", name))
		if err != nil {
			continue
		}
		fmt.Fprintf(&b, "### %s\n\n```json\n%s\n```\n\n", strings.TrimSuffix(name, ".json"), strings.TrimSpace(string(data)))
	}
	return strings.TrimSpace(b.String())
}

func renderPromptSnapshot(messages []rollout.ChatMessage) string {
	var b strings.Builder
	for _, message := range messages {
		fmt.Fprintf(&b, "## %s\n\n%s\n\n", strings.ToUpper(strings.TrimSpace(message.Role)), strings.TrimSpace(message.Content))
	}
	return strings.TrimSpace(b.String())
}

func requiredByProjectPrompt(policy projectPolicy) string {
	var lines []string
	for _, input := range policy.Inputs {
		parts := []string{input.Name}
		if input.Type != "" {
			parts = append(parts, input.Type)
		}
		if input.Required {
			parts = append(parts, "required")
		}
		if input.Description != "" {
			parts = append(parts, input.Description)
		}
		lines = append(lines, "- Input: "+strings.Join(parts, ", "))
	}
	for _, output := range policy.Outputs {
		if output.From != "" {
			lines = append(lines, fmt.Sprintf("- Output: %s from %s", output.Name, output.From))
		} else if output.Description != "" {
			lines = append(lines, fmt.Sprintf("- Output: %s (%s)", output.Name, output.Description))
		}
	}
	for _, hint := range policy.BindingHints {
		lines = append(lines, fmt.Sprintf("- Step `%s` MUST receive `%s` as input `%s`.", hint.To, hint.From, hint.Field))
	}
	for _, contract := range policy.FunctionContracts {
		if contract.Name == "" {
			continue
		}
		if len(contract.Inputs) > 0 {
			lines = append(lines, fmt.Sprintf("- Function `%s` inputs: %s", contract.Name, strings.Join(contract.Inputs, ", ")))
		}
		if len(contract.Outputs) > 0 {
			lines = append(lines, fmt.Sprintf("- Function `%s` outputs: %s", contract.Name, strings.Join(contract.Outputs, ", ")))
		}
		if contract.SideEffects != "" {
			lines = append(lines, fmt.Sprintf("- Function `%s` side effects: %s", contract.Name, contract.SideEffects))
		}
	}
	if len(lines) == 0 {
		return "Required by project.md:\n- No structured requirements were extracted.\n\n"
	}
	return "Required by project.md:\n" + strings.Join(lines, "\n") + "\n\n"
}

func specSummary(candidates []openapidisco.Candidate) string {
	if len(candidates) == 0 {
		return "No OpenAPI documents are available.\n"
	}
	var b strings.Builder
	remaining := maxPromptOperationsTotal
	for _, candidate := range candidates {
		title := candidate.Title
		if title == "" {
			title = candidate.RelativePath
		}
		fmt.Fprintf(&b, "- path: %s\n  title: %s\n", candidate.RelativePath, title)
		if candidate.Description != "" {
			fmt.Fprintf(&b, "  description: %s\n", trimForPrompt(candidate.Description, maxPromptDescriptionChars))
		}
		spec, err := rollout.LoadOpenAPISpec(candidate.Path)
		if err != nil {
			continue
		}
		ops := append([]*rollout.OperationInfo(nil), spec.Operations...)
		sort.SliceStable(ops, func(i, j int) bool {
			return ops[i].OperationID < ops[j].OperationID
		})
		limit := len(ops)
		if limit > remaining {
			limit = remaining
		}
		for i := 0; i < limit; i++ {
			op := ops[i]
			fmt.Fprintf(&b, "  operation: %s %s %s", op.OperationID, op.Method, op.Path)
			if op.Summary != "" {
				fmt.Fprintf(&b, " - %s", trimForPrompt(op.Summary, maxPromptOperationSummaryChars))
			}
			b.WriteString("\n")
			required := requiredOpenAPIParams(op)
			if len(required) > 0 {
				fmt.Fprintf(&b, "    required_parameters: %s\n", strings.Join(required, ", "))
			}
			responseFields := responseFieldNames(op, maxPromptResponseFields)
			if len(responseFields) > 0 {
				fmt.Fprintf(&b, "    response_fields: %s\n", strings.Join(responseFields, ", "))
			}
		}
		remaining -= limit
		if omitted := len(ops) - limit; omitted > 0 {
			fmt.Fprintf(&b, "  omitted_operations: %d\n", omitted)
		}
	}
	return b.String()
}

func requiredOpenAPIParams(op *rollout.OperationInfo) []string {
	if op == nil {
		return nil
	}
	var out []string
	for _, param := range op.Parameters {
		if param != nil && param.Required {
			out = append(out, param.Name)
		}
	}
	sort.Strings(out)
	return out
}

func responseFieldNames(op *rollout.OperationInfo, limit int) []string {
	if op == nil {
		return nil
	}
	fields := map[string]bool{}
	for _, response := range op.Responses {
		collectSchemaFieldNames(response.Schema, "", fields, 0)
	}
	out := make([]string, 0, len(fields))
	for field := range fields {
		out = append(out, field)
	}
	sort.Strings(out)
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

func collectSchemaFieldNames(schema map[string]any, prefix string, out map[string]bool, depth int) {
	if schema == nil || depth > 2 {
		return
	}
	if props, ok := schema["properties"].(map[string]any); ok {
		for name, raw := range props {
			field := name
			if prefix != "" {
				field = prefix + "." + name
			}
			out[field] = true
			if child, ok := raw.(map[string]any); ok {
				collectSchemaFieldNames(child, field, out, depth+1)
			}
		}
	}
	if items, ok := schema["items"].(map[string]any); ok {
		collectSchemaFieldNames(items, prefix+"[]", out, depth+1)
	}
}

func intentJSONShape() string {
	return `{
  "openapi": "<primary openapi path provided above>",
  "workflow": {"name": "workflow_name", "description": "short description"},
  "inputs": [{"name": "input_name", "type": "string", "required": true}],
  "steps": [
    {
      "name": "step_name",
      "type": "http",
      "do": "natural-language action",
      "operation": "optional_operationId",
      "depends_on": ["prior_step"],
      "with": {"query.id": "prior_step.received_body.id"}
    }
  ],
  "outputs": [{"name": "result", "from": "step_name.received_body"}]
}`
}

func extractJSON(response string) (string, error) {
	response = strings.TrimSpace(response)
	if response == "" {
		return "", fmt.Errorf("empty model response")
	}
	if strings.HasPrefix(response, "```") {
		lines := strings.Split(response, "\n")
		if len(lines) >= 3 {
			lines = lines[1 : len(lines)-1]
			return strings.TrimSpace(strings.Join(lines, "\n")), nil
		}
	}
	start := strings.Index(response, "{")
	end := strings.LastIndex(response, "}")
	if start < 0 || end <= start {
		return "", fmt.Errorf("no JSON object found in model response")
	}
	return response[start : end+1], nil
}
