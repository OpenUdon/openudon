package workflowintent

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/genelet/ramen/internal/authoring"
)

func WorkflowFlow() authoring.Flow[*Intent] {
	adapter := WorkflowAdapter{}
	return authoring.Flow[*Intent]{
		Parser:       adapter,
		Renderer:     adapter,
		Validator:    adapter,
		SlotProvider: adapter,
	}
}

func ParseFile(ctx context.Context, path string) (*Intent, error) {
	draft, diagnostics, err := WorkflowAdapter{}.ParseIntent(ctx, authoring.Artifact{Path: path})
	if err != nil {
		return nil, err
	}
	if authoring.HasErrors(diagnostics) {
		return nil, authoring.DiagnosticError{Diagnostics: diagnostics}
	}
	return draft, nil
}

func RenderHCL(ctx context.Context, draft *Intent) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	flow := WorkflowFlow()
	diagnostics := flow.Validator.ValidateIntent(ctx, draft)
	if authoring.HasErrors(diagnostics) {
		return "", authoring.DiagnosticError{Diagnostics: diagnostics}
	}
	set, renderDiagnostics, err := flow.Renderer.RenderIntent(ctx, draft)
	diagnostics = append(diagnostics, renderDiagnostics...)
	if err != nil {
		return "", err
	}
	if authoring.HasErrors(diagnostics) {
		return "", authoring.DiagnosticError{Diagnostics: diagnostics}
	}
	for _, artifact := range set.Artifacts {
		if artifact.Path == IntentPath || artifact.MediaType == "text/hcl" {
			return string(artifact.Content), nil
		}
	}
	return "", fmt.Errorf("rendered intent artifact %q not found", IntentPath)
}

func ValidateComplete(ctx context.Context, draft *Intent) []authoring.Diagnostic {
	adapter := WorkflowAdapter{}
	diagnostics := adapter.ValidateIntent(ctx, draft)
	for _, slot := range adapter.MissingSlots(ctx, draft) {
		if !slot.Required {
			continue
		}
		diagnostics = append(diagnostics, authoring.Diagnostic{
			Severity: "error",
			Code:     "missing_slot",
			Message:  fmt.Sprintf("required slot %q is missing", slot.Name),
			Path:     slot.Name,
		})
	}
	return diagnostics
}

type WorkflowAdapter struct{}

func (WorkflowAdapter) ParseIntent(ctx context.Context, artifact authoring.Artifact) (*Intent, []authoring.Diagnostic, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	path := strings.TrimSpace(artifact.Path)
	data := artifact.Content
	if len(data) == 0 && path != "" {
		var err error
		data, err = os.ReadFile(path)
		if err != nil {
			return nil, []authoring.Diagnostic{diagnostic("error", "intent_read_failed", err.Error(), path)}, err
		}
	}
	if path == "" {
		path = IntentPath
	}
	draft, err := ParseIntent(data, path)
	if err != nil {
		return nil, []authoring.Diagnostic{diagnostic("error", "intent_parse_failed", err.Error(), path)}, err
	}
	return draft, nil, nil
}

func (WorkflowAdapter) RenderIntent(ctx context.Context, draft *Intent) (authoring.ArtifactSet, []authoring.Diagnostic, error) {
	if err := ctx.Err(); err != nil {
		return authoring.ArtifactSet{}, nil, err
	}
	hcl, err := RenderIntentHCL(draft)
	if err != nil {
		return authoring.ArtifactSet{}, []authoring.Diagnostic{diagnostic("error", "intent_render_failed", err.Error(), IntentPath)}, err
	}
	return authoring.ArtifactSet{Artifacts: []authoring.Artifact{{
		Path:      IntentPath,
		MediaType: "text/hcl",
		Content:   []byte(hcl),
	}}}, nil, nil
}

func (WorkflowAdapter) ValidateIntent(ctx context.Context, draft *Intent) []authoring.Diagnostic {
	if err := ctx.Err(); err != nil {
		return []authoring.Diagnostic{diagnostic("error", "intent_validation_cancelled", err.Error(), "")}
	}
	if draft == nil {
		return []authoring.Diagnostic{diagnostic("error", "intent_required", "intent is required", "")}
	}
	hcl, err := RenderIntentHCL(draft)
	if err != nil {
		return []authoring.Diagnostic{diagnostic("error", "intent_render_failed", err.Error(), IntentPath)}
	}
	if _, err := ParseIntent([]byte(hcl), IntentPath); err != nil {
		return []authoring.Diagnostic{diagnostic("error", "intent_parse_failed", err.Error(), IntentPath)}
	}
	return nil
}

func (WorkflowAdapter) MissingSlots(ctx context.Context, draft *Intent) []authoring.Slot {
	_ = ctx
	var missing []string
	if draft == nil {
		return []authoring.Slot{{Name: "intent", Required: true}}
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
	slots := make([]authoring.Slot, 0, len(missing))
	for _, name := range missing {
		slots = append(slots, authoring.Slot{Name: name, Required: true})
	}
	return slots
}

func collectOperationSlots(missing *[]string, defaultOpenAPI string, steps []*Step) {
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

func diagnostic(severity, code, message, path string) authoring.Diagnostic {
	return authoring.Diagnostic{
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
