package udonreport

import (
	"fmt"
	"github.com/OpenUdon/openudon/internal/evidencefile"
	"github.com/OpenUdon/uws/convert"
	"github.com/OpenUdon/uws/uws1"
)

// InventoryFromWorkflowV5 derives identities from exact reviewed workflow bytes.
// It is a handoff shape check, not a private compiler or UWS semantic extension.
func InventoryFromWorkflowV5(data []byte, format, runID string) (InventoryV5, error) {
	var d uws1.Document
	var err error
	switch format {
	case "uws-yaml", "yaml":
		err = convert.UnmarshalYAML(data, &d)
	case "uws-json", "json":
		err = convert.UnmarshalJSON(data, &d)
	case "uws-hcl", "hcl":
		err = convert.UnmarshalHCL(data, &d)
	default:
		return InventoryV5{}, fmt.Errorf("unsupported v5 workflow format")
	}
	if err != nil {
		return InventoryV5{}, fmt.Errorf("invalid v5 workflow")
	}
	if len(d.Workflows) != 1 || len(d.Triggers) != 0 {
		return InventoryV5{}, fmt.Errorf("v5 requires one flat HTTP sequence")
	}
	w := d.Workflows[0]
	if w == nil || !ValidIdentifier(w.WorkflowID) || w.Type != "sequence" || len(w.Steps) == 0 || len(w.Steps) > MaxSteps || w.When != "" || w.ForEach != "" || len(w.DependsOn) != 0 || w.Wait != "" || w.Items != "" || w.Mode != "" || w.BatchSize != "" || len(w.Cases) != 0 || len(w.Default) != 0 {
		return InventoryV5{}, fmt.Errorf("unsupported v5 workflow shape")
	}
	sources := map[string]bool{}
	for _, s := range d.SourceDescriptions {
		if s != nil {
			sources[s.Name] = s.EffectiveType() == "openapi"
		}
	}
	operations := map[string]bool{}
	for _, op := range d.Operations {
		if op == nil || !ValidIdentifier(op.OperationID) || operations[op.OperationID] || !sources[op.SourceDescription] || op.When != "" || op.ForEach != "" || len(op.DependsOn) != 0 || op.Wait != "" || op.ParallelGroup != "" || len(op.OnFailure) != 0 || len(op.OnSuccess) != 0 {
			return InventoryV5{}, fmt.Errorf("unsupported v5 operation")
		}
		operations[op.OperationID] = true
	}
	i := InventoryV5{RunID: runID, WorkflowID: w.WorkflowID, WorkflowDigest: "sha256:" + evidencefile.SHA256(data)}
	steps, used := map[string]bool{}, map[string]bool{}
	for _, s := range w.Steps {
		if s == nil || !ValidIdentifier(s.StepID) || !operations[s.OperationRef] || steps[s.StepID] || used[s.OperationRef] || s.Workflow != "" || s.When != "" || s.ForEach != "" || s.Wait != "" || s.ParallelGroup != "" || s.Items != "" || s.Mode != "" || s.BatchSize != "" || len(s.Steps) != 0 || len(s.Cases) != 0 || len(s.Default) != 0 || (s.Type != "" && s.Type != "operation") {
			return InventoryV5{}, fmt.Errorf("unsupported v5 invocation")
		}
		for _, dep := range s.DependsOn {
			if !steps[dep] {
				return InventoryV5{}, fmt.Errorf("v5 requires backward step dependencies")
			}
		}
		steps[s.StepID], used[s.OperationRef] = true, true
		i.Steps = append(i.Steps, StepV5{StepID: s.StepID, OperationID: s.OperationRef, InvocationID: s.StepID, Outcome: "not_started"})
	}
	if len(used) != len(operations) {
		return InventoryV5{}, fmt.Errorf("incomplete v5 operation inventory")
	}
	return i, i.Validate()
}
