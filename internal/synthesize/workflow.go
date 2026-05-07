package synthesize

import (
	"fmt"
	"os"
	"strings"

	"github.com/genelet/ramen/internal/uwsvalidate"
	"github.com/genelet/udon/generator"
	"github.com/genelet/udon/pkg/rollout"
	"github.com/genelet/udon/pkg/uwsprofile"
	"github.com/OpenUdon/uws/uws1"
	"gopkg.in/yaml.v3"
)

func promoteWorkflow(result Result, schemaPath string) error {
	plan, err := generator.NewRuntimePlanFromWorkflowFile(result.WorkflowPath, result.ExampleDir)
	if err != nil {
		return fmt.Errorf("compile workflow through udon: %w", err)
	}
	doc := plan.Document()
	normalizeUWSStepsForSchema(doc)
	if intent, err := rollout.ParseIntentFile(result.IntentPath); err == nil {
		addStructuralResultsFromIntent(doc, intent)
	}
	uwsBytes, err := uwsprofile.MarshalDocument(doc, uwsprofile.DocumentFormatYAML)
	if err != nil {
		return fmt.Errorf("marshal UWS: %w", err)
	}
	uwsBytes, err = pruneEmptyUWSStepTypes(uwsBytes)
	if err != nil {
		return fmt.Errorf("normalize UWS YAML: %w", err)
	}
	if err := ensureArtifactDirs(result); err != nil {
		return err
	}
	if err := os.WriteFile(result.UWSPath, uwsBytes, 0o644); err != nil {
		return err
	}

	schemaPath = strings.TrimSpace(schemaPath)
	if schemaPath == "" {
		schemaPath = defaultSchemaPathForVersion(result.ExampleDir, doc.UWS)
	}
	if err := uwsvalidate.ValidateFile(schemaPath, result.UWSPath); err != nil {
		return fmt.Errorf("validate exported UWS: %w", err)
	}
	return nil
}

func addStructuralResultsFromIntent(doc *uws1.Document, intent *rollout.Intent) {
	if doc == nil || intent == nil {
		return
	}
	for _, result := range structuralPlanResults(intent) {
		if structuralResultExists(doc.Results, result.Name) {
			continue
		}
		doc.Results = append(doc.Results, &uws1.StructuralResult{
			Name:  result.Name,
			Kind:  result.Kind,
			From:  result.From,
			Value: result.Value,
		})
	}
}

func structuralResultExists(results []*uws1.StructuralResult, name string) bool {
	name = strings.TrimSpace(name)
	for _, result := range results {
		if result != nil && strings.TrimSpace(result.Name) == name {
			return true
		}
	}
	return false
}

func pruneEmptyUWSStepTypes(data []byte) ([]byte, error) {
	var value any
	if err := yaml.Unmarshal(data, &value); err != nil {
		return nil, err
	}
	pruneEmptyTypeFields(value)
	return yaml.Marshal(value)
}

func pruneEmptyTypeFields(value any) {
	switch typed := value.(type) {
	case map[string]any:
		if raw, ok := typed["type"]; ok && strings.TrimSpace(fmt.Sprint(raw)) == "" {
			delete(typed, "type")
		}
		for _, child := range typed {
			pruneEmptyTypeFields(child)
		}
	case []any:
		for _, child := range typed {
			pruneEmptyTypeFields(child)
		}
	}
}

func normalizeUWSStepsForSchema(doc *uws1.Document) {
	if doc == nil {
		return
	}
	operationIDs := make(map[string]bool, len(doc.Operations))
	for _, op := range doc.Operations {
		if op != nil && strings.TrimSpace(op.OperationID) != "" {
			operationIDs[strings.TrimSpace(op.OperationID)] = true
		}
	}
	for _, workflow := range doc.Workflows {
		if workflow == nil {
			continue
		}
		normalizeUWSStepList(workflow.Steps, operationIDs)
	}
}

func normalizeUWSStepList(steps []*uws1.Step, operationIDs map[string]bool) {
	for _, step := range steps {
		if step == nil {
			continue
		}
		if strings.TrimSpace(step.OperationRef) == "" && !isUWSStructuralStepType(step.Type) && operationIDs[strings.TrimSpace(step.StepID)] {
			step.OperationRef = strings.TrimSpace(step.StepID)
		}
		if strings.TrimSpace(step.OperationRef) != "" && !isUWSStructuralStepType(step.Type) {
			step.Type = ""
		}
		normalizeUWSStepList(step.Steps, operationIDs)
		for _, branch := range step.Cases {
			if branch != nil {
				normalizeUWSStepList(branch.Steps, operationIDs)
			}
		}
		normalizeUWSStepList(step.Default, operationIDs)
	}
}

func isUWSStructuralStepType(value string) bool {
	switch strings.TrimSpace(value) {
	case "", uws1.WorkflowTypeSequence, uws1.WorkflowTypeParallel, uws1.WorkflowTypeSwitch,
		uws1.WorkflowTypeMerge, uws1.WorkflowTypeLoop, uws1.WorkflowTypeAwait:
		return true
	default:
		return false
	}
}
