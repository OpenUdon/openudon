package synthesize

import (
	"fmt"
	"os"
	"strings"

	"github.com/OpenUdon/uws/uws1"
	uwsprofile "github.com/genelet/ramen/internal/uwsexec"
	"github.com/genelet/ramen/internal/uwsvalidate"
)

func assessUWS(report *QualityReport, path, schemaPath, exampleDir string, expectedPlan *WorkflowPlan) {
	if _, err := os.Stat(path); err != nil {
		report.add("uws.present", "fail", "workflow.uws.yaml is required", err.Error())
		return
	}
	report.add("uws.present", "pass", "workflow.uws.yaml is present", "")
	if strings.TrimSpace(schemaPath) == "" {
		schemaPath = defaultSchemaPathForDocument(exampleDir, path)
	}
	if err := uwsvalidate.ValidateFile(schemaPath, path); err != nil {
		report.add("uws.schema", "fail", "workflow.uws.yaml fails public UWS schema validation", err.Error())
		return
	}
	report.add("uws.schema", "pass", "workflow.uws.yaml validates against public UWS schema", "")
	doc, err := uwsprofile.LoadDocumentFile(path, uwsprofile.DocumentFormatYAML)
	if err != nil {
		report.add("uws.execution_profile", "fail", "workflow.uws.yaml could not be loaded by local execution-profile helpers", err.Error())
		return
	}
	if err := uwsprofile.ValidateForExecution(doc); err != nil {
		report.add("uws.execution_profile", "fail", "workflow.uws.yaml fails local execution-profile validation", err.Error())
		return
	}
	report.add("uws.execution_profile", "pass", "workflow.uws.yaml passes local execution-profile validation", "")
	if expectedPlan != nil && len(expectedPlan.Results) > 0 {
		if err := validateUWSStructuralResults(doc, expectedPlan.Results); err != nil {
			report.add("uws.structural_results", "fail", "workflow.uws.yaml does not preserve planned structural results", err.Error())
			return
		}
		report.add("uws.structural_results", "pass", "workflow.uws.yaml preserves planned structural results", "")
	}
	if expectedPlan != nil && planHasActions(expectedPlan) {
		if err := validateUWSOperationActions(doc, expectedPlan); err != nil {
			report.add("uws.operation_actions", "fail", "workflow.uws.yaml does not preserve planned operation actions", err.Error())
			return
		}
		report.add("uws.operation_actions", "pass", "workflow.uws.yaml preserves planned operation actions", "")
	}
	if expectedPlan != nil && planHasTimeoutOrIdempotency(expectedPlan) {
		if err := validateUWSTimeoutAndIdempotency(doc, expectedPlan); err != nil {
			report.add("uws.timeout_idempotency", "fail", "workflow.uws.yaml does not preserve planned timeout/idempotency metadata", err.Error())
			return
		}
		report.add("uws.timeout_idempotency", "pass", "workflow.uws.yaml preserves planned timeout/idempotency metadata", "")
	}
}

func validateUWSStructuralResults(doc *uws1.Document, expected []PlanResult) error {
	if len(expected) == 0 {
		return nil
	}
	got := map[string]*uws1.StructuralResult{}
	if doc != nil {
		for _, result := range doc.Results {
			if result != nil && strings.TrimSpace(result.Name) != "" {
				got[strings.TrimSpace(result.Name)] = result
			}
		}
	}
	var missing, mismatched []string
	for _, want := range expected {
		name := strings.TrimSpace(want.Name)
		if name == "" {
			continue
		}
		result := got[name]
		if result == nil {
			missing = append(missing, name)
			continue
		}
		if strings.TrimSpace(result.Kind) != strings.TrimSpace(want.Kind) ||
			strings.TrimSpace(result.From) != strings.TrimSpace(want.From) ||
			(strings.TrimSpace(want.Value) != "" && strings.TrimSpace(result.Value) != strings.TrimSpace(want.Value)) {
			mismatched = append(mismatched, fmt.Sprintf("%s expected kind=%s from=%s value=%s got kind=%s from=%s value=%s", name, want.Kind, want.From, want.Value, result.Kind, result.From, result.Value))
		}
	}
	if len(missing) == 0 && len(mismatched) == 0 {
		return nil
	}
	var details []string
	if len(missing) > 0 {
		details = append(details, "missing "+strings.Join(sortedCopy(missing), ", "))
	}
	details = append(details, sortedCopy(mismatched)...)
	return fmt.Errorf("%s", strings.Join(details, "; "))
}

func validateUWSOperationActions(doc *uws1.Document, expected *WorkflowPlan) error {
	if expected == nil {
		return nil
	}
	ops := map[string]*uws1.Operation{}
	if doc != nil {
		for _, op := range doc.Operations {
			if op != nil && strings.TrimSpace(op.OperationID) != "" {
				ops[strings.TrimSpace(op.OperationID)] = op
			}
		}
	}
	var mismatches []string
	for _, step := range expected.Steps {
		if !expectedStepHasActions(step) {
			continue
		}
		name := strings.TrimSpace(step.Name)
		if name == "" {
			continue
		}
		op := ops[name]
		if op == nil {
			mismatches = append(mismatches, name+" missing operation")
			continue
		}
		got := canonicalActionJSON(op.SuccessCriteria, op.OnFailure, op.OnSuccess)
		want := planStepActionsSummary(step)
		if got != want {
			mismatches = append(mismatches, fmt.Sprintf("%s expected %s got %s", name, want, got))
		}
	}
	if len(mismatches) == 0 {
		return nil
	}
	return fmt.Errorf("%s", strings.Join(sortedCopy(mismatches), "; "))
}

func planHasActions(plan *WorkflowPlan) bool {
	if plan == nil {
		return false
	}
	for _, step := range plan.Steps {
		if expectedStepHasActions(step) {
			return true
		}
	}
	return false
}

func planHasTimeoutOrIdempotency(plan *WorkflowPlan) bool {
	if plan == nil {
		return false
	}
	if plan.Timeout != nil || plan.Idempotency != nil {
		return true
	}
	for _, step := range plan.Steps {
		if step.Timeout != nil {
			return true
		}
	}
	return false
}

func validateUWSTimeoutAndIdempotency(doc *uws1.Document, expected *WorkflowPlan) error {
	if expected == nil {
		return nil
	}
	root := rootUWSWorkflow(doc)
	var mismatches []string
	if expected.Timeout != nil {
		if root == nil || !floatPtrEqual(root.Timeout, expected.Timeout) {
			mismatches = append(mismatches, fmt.Sprintf("workflow timeout expected %g got %s", *expected.Timeout, formatFloatPtr(workflowTimeout(root))))
		}
	}
	if expected.Idempotency != nil {
		if root == nil || !idempotencyEqual(root.Idempotency, expected.Idempotency) {
			mismatches = append(mismatches, fmt.Sprintf("workflow idempotency expected %s got %s", idempotencySummary(expected.Idempotency), idempotencySummary(idempotencyFromWorkflow(root))))
		}
	}
	ops := map[string]*uws1.Operation{}
	stepIndex := map[string]*uws1.Step{}
	if doc != nil {
		for _, op := range doc.Operations {
			if op != nil && strings.TrimSpace(op.OperationID) != "" {
				ops[strings.TrimSpace(op.OperationID)] = op
			}
		}
		for _, wf := range doc.Workflows {
			if wf != nil {
				indexUWSSteps(stepIndex, wf.Steps)
				for _, branch := range wf.Cases {
					if branch != nil {
						indexUWSSteps(stepIndex, branch.Steps)
					}
				}
				indexUWSSteps(stepIndex, wf.Default)
			}
		}
	}
	for _, step := range expected.Steps {
		if step.Timeout == nil || strings.TrimSpace(step.Name) == "" {
			continue
		}
		name := strings.TrimSpace(step.Name)
		gotStep := stepIndex[name]
		if gotStep != nil && strings.TrimSpace(gotStep.OperationRef) == "" {
			if !floatPtrEqual(gotStep.Timeout, step.Timeout) {
				mismatches = append(mismatches, fmt.Sprintf("%s step timeout expected %g got %s", name, *step.Timeout, formatFloatPtr(gotStep.Timeout)))
			}
			continue
		}
		op := ops[name]
		if op == nil && gotStep != nil && strings.TrimSpace(gotStep.OperationRef) != "" {
			op = ops[strings.TrimSpace(gotStep.OperationRef)]
		}
		if op == nil || !floatPtrEqual(op.Timeout, step.Timeout) {
			var got *float64
			if op != nil {
				got = op.Timeout
			}
			mismatches = append(mismatches, fmt.Sprintf("%s operation timeout expected %g got %s", name, *step.Timeout, formatFloatPtr(got)))
		}
	}
	if len(mismatches) > 0 {
		return fmt.Errorf("%s", strings.Join(sortedCopy(mismatches), "; "))
	}
	return nil
}

func rootUWSWorkflow(doc *uws1.Document) *uws1.Workflow {
	if doc == nil {
		return nil
	}
	for _, wf := range doc.Workflows {
		if wf != nil && strings.TrimSpace(wf.WorkflowID) == "main" {
			return wf
		}
	}
	if len(doc.Workflows) == 1 {
		return doc.Workflows[0]
	}
	return nil
}

func workflowTimeout(wf *uws1.Workflow) *float64 {
	if wf == nil {
		return nil
	}
	return wf.Timeout
}

func idempotencyFromWorkflow(wf *uws1.Workflow) *uws1.Idempotency {
	if wf == nil {
		return nil
	}
	return wf.Idempotency
}

func indexUWSSteps(out map[string]*uws1.Step, steps []*uws1.Step) {
	for _, step := range steps {
		if step == nil {
			continue
		}
		if name := strings.TrimSpace(step.StepID); name != "" {
			out[name] = step
		}
		indexUWSSteps(out, step.Steps)
		for _, branch := range step.Cases {
			if branch != nil {
				indexUWSSteps(out, branch.Steps)
			}
		}
		indexUWSSteps(out, step.Default)
	}
}

func idempotencyEqual(left, right *uws1.Idempotency) bool {
	if left == nil || right == nil {
		return left == right
	}
	return strings.TrimSpace(left.Key) == strings.TrimSpace(right.Key) &&
		strings.TrimSpace(left.OnConflict) == strings.TrimSpace(right.OnConflict) &&
		floatPtrEqual(left.TTL, right.TTL)
}

func idempotencySummary(value *uws1.Idempotency) string {
	if value == nil {
		return "missing"
	}
	parts := []string{"key=" + strings.TrimSpace(value.Key)}
	if value.OnConflict != "" {
		parts = append(parts, "onConflict="+strings.TrimSpace(value.OnConflict))
	}
	if value.TTL != nil {
		parts = append(parts, fmt.Sprintf("ttl=%g", *value.TTL))
	}
	return strings.Join(parts, ",")
}
