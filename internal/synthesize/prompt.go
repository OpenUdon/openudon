package synthesize

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
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
)

//go:embed prompts/intent_generation.tmpl
var intentGenerationSystemPrompt string

func generateIntent(ctx context.Context, chat rollout.ChatClient, projectText string, candidates []openapidisco.Candidate, primary string, policy projectPolicy, feedback string) (*rollout.Intent, error) {
	feedback = strings.TrimSpace(feedback)
	feedbackSection := ""
	if feedback != "" {
		feedbackSection = "\nPrevious quality failure to repair:\n" + feedback + "\n"
	}
	user := fmt.Sprintf("Project brief:\n%s\n\nRuntime and OpenAPI policy:\n%s\nAvailable OpenAPI documents:\n%s\n\nPrimary OpenAPI path: %s\n%s\nReturn JSON matching this shape:\n%s",
		projectText,
		runtimePolicyPrompt(policy),
		specSummary(candidates),
		primary,
		feedbackSection,
		intentJSONShape(),
	)
	response, err := chat.Chat(ctx, []rollout.ChatMessage{
		{Role: "system", Content: strings.TrimSpace(intentGenerationSystemPrompt)},
		{Role: "user", Content: user},
	})
	if err != nil {
		return nil, err
	}
	jsonText, err := extractJSON(response)
	if err != nil {
		return nil, err
	}
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
  "openapi": "openapi/example.yaml",
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
