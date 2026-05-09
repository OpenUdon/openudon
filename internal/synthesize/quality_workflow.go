package synthesize

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/OpenUdon/uws/uws1"
	"github.com/OpenUdon/openudon/internal/openapidisco"
	uwsprofile "github.com/OpenUdon/openudon/internal/uwsexec"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

func assessWorkflowPlan(report *QualityReport, result Result) *WorkflowPlan {
	plan, err := loadWorkflowPlan(result.PlanJSONPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if generatedArtifactsExist(result) {
				report.add("plan.present", "fail", "expected workflow plan is missing", "Run `openudon synthesize` or `openudon build` to create expected/plan.json.")
			} else {
				report.add("plan.present", "warn", "expected workflow plan is missing", "Run `openudon synthesize` or `openudon build` to create expected/plan.json.")
			}
			return nil
		}
		report.add("plan.parse", "fail", "expected workflow plan is invalid", err.Error())
		return nil
	}
	if strings.TrimSpace(plan.Version) != workflowPlanVersion {
		report.add("plan.version", "warn", "expected workflow plan has an unknown version", plan.Version)
	} else {
		report.add("plan.version", "pass", "expected workflow plan version is supported", "")
	}
	if len(plan.Gaps) > 0 {
		var details []string
		for _, gap := range plan.Gaps {
			details = append(details, fmt.Sprintf("%s: %s", gap.Code, gap.Detail))
		}
		report.add("plan.gaps", "fail", "expected workflow plan has unresolved synthesis gaps", strings.Join(details, "; "))
	} else {
		report.add("plan.gaps", "pass", "expected workflow plan has no unresolved gaps", "")
	}
	return plan
}

func generatedArtifactsExist(result Result) bool {
	for _, path := range []string{result.IntentPath, result.WorkflowPath, result.UWSPath} {
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}
	return false
}

func assessDiscoveryReport(report *QualityReport, path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			report.add("openapi.discovery", "warn", "OpenAPI discovery report is missing", "Run `openudon synthesize` or `openudon build` to record OpenAPI discovery attempts.")
		} else {
			report.add("openapi.discovery", "warn", "OpenAPI discovery report could not be read", err.Error())
		}
		return
	}
	var discovery openapidisco.DiscoveryReport
	if err := json.Unmarshal(data, &discovery); err != nil {
		report.add("openapi.discovery", "warn", "OpenAPI discovery report is invalid", err.Error())
		return
	}
	var failures []string
	for _, attempt := range discovery.Attempts {
		if attempt.Status == "fail" {
			failures = append(failures, fmt.Sprintf("%s %s: %s", attempt.Kind, attempt.Source, attempt.Detail))
		}
	}
	if len(failures) > 0 {
		report.add("openapi.discovery", "warn", "some OpenAPI discovery attempts failed", strings.Join(sortedCopy(failures), "; "))
		return
	}
	report.add("openapi.discovery", "pass", "OpenAPI discovery attempts are recorded", "")
}

func assessWorkflow(report *QualityReport, path, exampleDir string, intent *rollout.Intent, policy projectPolicy, expectedPlan *WorkflowPlan) bool {
	source, err := os.ReadFile(path)
	if err != nil {
		report.add("workflow.present", "fail", "workflow.hcl is required", err.Error())
		return false
	}
	report.add("workflow.present", "pass", "workflow.hcl is readable", "")
	if err := rollout.ValidateHCL(string(source)); err != nil {
		report.add("workflow.hcl_syntax", "fail", "workflow.hcl has invalid HCL syntax", err.Error())
		return false
	}
	report.add("workflow.hcl_syntax", "pass", "workflow.hcl syntax is valid", "")
	if policy.NoOpenAPI && workflowReferencesOpenAPI(source) {
		report.add("workflow.openapi_refs", "fail", "workflow.hcl references OpenAPI even though project.md declares OpenAPI: none required", "")
		return false
	}
	if policy.NoOpenAPI {
		report.add("workflow.openapi_refs", "pass", "workflow.hcl has no OpenAPI references", "")
	}
	doc, err := loadUWSDocumentFile(path)
	if err != nil {
		report.add("workflow.uws_parse", "fail", "workflow.hcl is not a valid public UWS document", err.Error())
		return false
	}
	report.add("workflow.uws_parse", "pass", "workflow.hcl parses as a public UWS document", "")
	if intent != nil {
		missing := missingIntentSteps(intent, doc.Workflows)
		if len(missing) > 0 {
			report.add("workflow.intent_coverage", "fail", "workflow.hcl does not represent every intent step", strings.Join(missing, ", "))
			return false
		}
		report.add("workflow.intent_coverage", "pass", "workflow.hcl represents intent steps", "")
	}
	if expectedPlan != nil && !validateWorkflowAgainstExpectedPlan(report, doc, expectedPlan) {
		return false
	}
	return true
}

func workflowReferencesOpenAPI(source []byte) bool {
	return regexp.MustCompile(`(?im)\bopenapi\s*=`).Match(source)
}

func validateWorkflowAgainstExpectedPlan(report *QualityReport, compiled *uws1.Document, expected *WorkflowPlan) bool {
	if expected == nil {
		return true
	}
	return validateWorkflowAgainstExpectedPlanWithIndex(report, expected, func() (map[string]*compiledOperation, error) {
		return compiledOperationIndex(compiled)
	})
}

func validateWorkflowAgainstExpectedPlanWithIndex(report *QualityReport, expected *WorkflowPlan, operationIndex func() (map[string]*compiledOperation, error)) bool {
	if expected == nil {
		return true
	}
	ops, err := operationIndex()
	if err != nil {
		report.add("workflow.request_evidence", "fail", "workflow.hcl request evidence could not be projected", err.Error())
		return false
	}
	var missing, runtimeMismatch, operationMismatch, dependsMismatch, timeoutMismatch, controlMismatch, actionMismatch, requestMismatch, bindingSourceMismatch, credentialMismatch []string
	for _, step := range expected.Steps {
		name := strings.TrimSpace(step.Name)
		if name == "" {
			continue
		}
		op := ops[name]
		if op == nil {
			missing = append(missing, name)
			continue
		}
		wantRuntime := strings.ToLower(strings.TrimSpace(step.Runtime))
		if wantRuntime == "" {
			wantRuntime = strings.ToLower(strings.TrimSpace(step.Type))
		}
		gotRuntime := strings.ToLower(strings.TrimSpace(op.ServiceType))
		if wantRuntime != "" && gotRuntime != "" && wantRuntime != gotRuntime && !equivalentWorkflowRuntime(wantRuntime, gotRuntime) {
			runtimeMismatch = append(runtimeMismatch, fmt.Sprintf("%s expected %s got %s", name, wantRuntime, gotRuntime))
		}
		if strings.TrimSpace(step.Operation) != "" && strings.TrimSpace(op.OpenAPIOperationID) != strings.TrimSpace(step.Operation) {
			operationMismatch = append(operationMismatch, fmt.Sprintf("%s expected %s got %s", name, step.Operation, op.OpenAPIOperationID))
		}
		if step.Timeout != nil && !floatPtrEqual(op.Timeout, step.Timeout) {
			timeoutMismatch = append(timeoutMismatch, fmt.Sprintf("%s expected timeout %g got %s", name, *step.Timeout, formatFloatPtr(op.Timeout)))
		}
		for _, dep := range step.DependsOn {
			if !containsString(op.DependsOn, dep) {
				dependsMismatch = append(dependsMismatch, fmt.Sprintf("%s missing dependency %s", name, dep))
			}
		}
		if strings.TrimSpace(step.Parent) != "" && strings.TrimSpace(op.Parent) != strings.TrimSpace(step.Parent) {
			controlMismatch = append(controlMismatch, fmt.Sprintf("%s expected parent %s got %s", name, step.Parent, op.Parent))
		}
		if strings.TrimSpace(step.Branch) != "" && strings.TrimSpace(op.Branch) != strings.TrimSpace(step.Branch) {
			controlMismatch = append(controlMismatch, fmt.Sprintf("%s expected branch %s got %s", name, step.Branch, op.Branch))
		}
		if strings.TrimSpace(step.BranchWhen) != "" && !planControlValuesEqual(op.BranchWhen, step.BranchWhen) {
			controlMismatch = append(controlMismatch, fmt.Sprintf("%s expected branch condition %s got %s", name, step.BranchWhen, op.BranchWhen))
		}
		if strings.TrimSpace(step.When) != "" && !planControlValuesEqual(op.When, step.When) {
			controlMismatch = append(controlMismatch, fmt.Sprintf("%s expected when %s got %s", name, step.When, op.When))
		}
		if strings.TrimSpace(step.ForEach) != "" && !planControlValuesEqual(op.ForEach, step.ForEach) {
			controlMismatch = append(controlMismatch, fmt.Sprintf("%s expected for_each %s got %s", name, step.ForEach, op.ForEach))
		}
		if strings.TrimSpace(step.Items) != "" && !planControlValuesEqual(op.Items, step.Items) {
			controlMismatch = append(controlMismatch, fmt.Sprintf("%s expected items %s got %s", name, step.Items, op.Items))
		}
		if strings.TrimSpace(step.Mode) != "" && !planControlValuesEqual(op.Mode, step.Mode) {
			controlMismatch = append(controlMismatch, fmt.Sprintf("%s expected mode %s got %s", name, step.Mode, op.Mode))
		}
		if strings.TrimSpace(step.BatchSize) != "" && !planControlValuesEqual(op.BatchSize, step.BatchSize) {
			controlMismatch = append(controlMismatch, fmt.Sprintf("%s expected batch_size %s got %s", name, step.BatchSize, op.BatchSize))
		}
		if expectedStepHasActions(step) && !compiledActionsMatch(op, step) {
			actionMismatch = append(actionMismatch, fmt.Sprintf("%s expected actions %s got %s", name, planStepActionsSummary(step), compiledActionsSummary(op)))
		}
		for _, param := range step.RequestParams {
			if !param.Required {
				continue
			}
			evidence, ok := requestAttributeEvidence(op.Request, paramCandidateNames(param.Name))
			if !ok {
				if param.Credential {
					credentialMismatch = append(credentialMismatch, fmt.Sprintf("%s missing credential request field %s", name, param.Name))
				} else {
					requestMismatch = append(requestMismatch, fmt.Sprintf("%s missing required request field %s", name, param.Name))
				}
				continue
			}
			if param.Credential {
				if param.ExpectedCredential != "" && !strings.Contains(evidence.Expression, param.ExpectedCredential) {
					credentialMismatch = append(credentialMismatch, fmt.Sprintf("%s.%s expected credential binding %s got %s", name, param.Name, param.ExpectedCredential, evidence.Expression))
				}
				continue
			}
			if param.ExpectedSource == "" {
				continue
			}
			switch param.SourceKind {
			case "input":
				if !expressionReferencesInputSource(evidence.Expression, param.ExpectedSource) {
					if _, ok := requestAttributeEvidenceMatching(op.Request, paramCandidateNames(param.Name), func(candidate requestAttribute) bool {
						return expressionReferencesInputSource(candidate.Expression, param.ExpectedSource)
					}); ok {
						continue
					}
					bindingSourceMismatch = append(bindingSourceMismatch, fmt.Sprintf("%s.%s expected input source %s got %s", name, param.Name, param.ExpectedSource, evidence.Expression))
				}
			case "binding":
				if !expressionReferencesExpectedSource(evidence.Expression, param.ExpectedSource) {
					if _, ok := requestAttributeEvidenceMatching(op.Request, paramCandidateNames(param.Name), func(candidate requestAttribute) bool {
						return expressionReferencesExpectedSource(candidate.Expression, param.ExpectedSource)
					}); ok {
						continue
					}
					bindingSourceMismatch = append(bindingSourceMismatch, fmt.Sprintf("%s.%s expected source %s got %s", name, param.Name, param.ExpectedSource, evidence.Expression))
				}
			case "literal":
				if normalizeBindingExpression(evidence.Expression) != normalizeBindingExpression(param.ExpectedSource) {
					if _, ok := requestAttributeEvidenceMatching(op.Request, paramCandidateNames(param.Name), func(candidate requestAttribute) bool {
						return normalizeBindingExpression(candidate.Expression) == normalizeBindingExpression(param.ExpectedSource)
					}); ok {
						continue
					}
					bindingSourceMismatch = append(bindingSourceMismatch, fmt.Sprintf("%s.%s expected literal source %s got %s", name, param.Name, param.ExpectedSource, evidence.Expression))
				}
			default:
				if !expressionReferencesExpectedSource(evidence.Expression, param.ExpectedSource) {
					if _, ok := requestAttributeEvidenceMatching(op.Request, paramCandidateNames(param.Name), func(candidate requestAttribute) bool {
						return expressionReferencesExpectedSource(candidate.Expression, param.ExpectedSource)
					}); ok {
						continue
					}
					bindingSourceMismatch = append(bindingSourceMismatch, fmt.Sprintf("%s.%s expected source %s got %s", name, param.Name, param.ExpectedSource, evidence.Expression))
				}
			}
		}
		for _, binding := range step.Bindings {
			if strings.TrimSpace(binding.Target) == "" {
				continue
			}
			evidence, ok := requestAttributeEvidence(op.Request, paramCandidateNames(binding.Target))
			if !ok {
				if !bindingRequestEvidenceRequired(step) {
					continue
				}
				if credentialLikeParam(binding.Target) {
					credentialMismatch = append(credentialMismatch, fmt.Sprintf("%s missing credential request field %s", name, binding.Target))
				} else {
					requestMismatch = append(requestMismatch, fmt.Sprintf("%s missing bound request field %s", name, binding.Target))
				}
				continue
			}
			expectedSource := bindingExpectedSource(binding)
			if expectedSource != "" && !expressionReferencesExpectedSource(evidence.Expression, expectedSource) {
				if _, ok := requestAttributeEvidenceMatching(op.Request, paramCandidateNames(binding.Target), func(candidate requestAttribute) bool {
					return expressionReferencesExpectedSource(candidate.Expression, expectedSource)
				}); ok {
					continue
				}
				bindingSourceMismatch = append(bindingSourceMismatch, fmt.Sprintf("%s.%s expected source %s got %s", name, binding.Target, expectedSource, evidence.Expression))
			}
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		report.add("workflow.plan_coverage", "fail", "workflow.hcl does not include every planned step", strings.Join(missing, ", "))
		return false
	}
	report.add("workflow.plan_coverage", "pass", "workflow.hcl includes every planned step", "")
	if len(runtimeMismatch) > 0 || len(operationMismatch) > 0 || len(dependsMismatch) > 0 || len(timeoutMismatch) > 0 || len(controlMismatch) > 0 || len(actionMismatch) > 0 || len(requestMismatch) > 0 {
		var details []string
		details = append(details, sortedCopy(runtimeMismatch)...)
		details = append(details, sortedCopy(operationMismatch)...)
		details = append(details, sortedCopy(dependsMismatch)...)
		details = append(details, sortedCopy(timeoutMismatch)...)
		details = append(details, sortedCopy(controlMismatch)...)
		details = append(details, sortedCopy(actionMismatch)...)
		details = append(details, sortedCopy(requestMismatch)...)
		report.add("workflow.plan_match", "fail", "workflow.hcl diverges from the expected plan", strings.Join(details, "; "))
		return false
	}
	report.add("workflow.plan_match", "pass", "workflow.hcl preserves planned runtimes, operations, dependencies, actions, and request mappings", "")
	if len(bindingSourceMismatch) > 0 {
		report.add("workflow.binding_sources", "fail", "workflow.hcl request fields do not preserve planned data sources", strings.Join(sortedCopy(bindingSourceMismatch), "; "))
		return false
	}
	report.add("workflow.binding_sources", "pass", "workflow.hcl request fields preserve planned data sources", "")
	if len(credentialMismatch) > 0 {
		report.add("workflow.credentials_bound", "fail", "workflow.hcl does not bind required credential-like parameters", strings.Join(sortedCopy(credentialMismatch), "; "))
		return false
	}
	report.add("workflow.credentials_bound", "pass", "workflow.hcl binds required credential-like parameters", "")
	return true
}

func bindingRequestEvidenceRequired(step PlanStep) bool {
	return !strings.EqualFold(strings.TrimSpace(step.Runtime), "fnct") && !strings.EqualFold(strings.TrimSpace(step.Type), "fnct")
}

func planControlValuesEqual(got, want string) bool {
	return normalizeBindingExpression(got) == normalizeBindingExpression(want)
}

func expectedStepHasActions(step PlanStep) bool {
	return len(step.SuccessCriteria) > 0 || len(step.OnFailure) > 0 || len(step.OnSuccess) > 0
}

func compiledActionsMatch(got *compiledOperation, want PlanStep) bool {
	if got == nil {
		return !expectedStepHasActions(want)
	}
	return canonicalActionJSON(got.SuccessCriteria, got.OnFailure, got.OnSuccess) ==
		canonicalActionJSON(want.SuccessCriteria, want.OnFailure, want.OnSuccess)
}

func planStepActionsSummary(step PlanStep) string {
	return canonicalActionJSON(step.SuccessCriteria, step.OnFailure, step.OnSuccess)
}

func compiledActionsSummary(op *compiledOperation) string {
	if op == nil {
		return "{}"
	}
	return canonicalActionJSON(op.SuccessCriteria, op.OnFailure, op.OnSuccess)
}

func canonicalActionJSON(criteria []*uws1.Criterion, failure []*uws1.FailureAction, success []*uws1.SuccessAction) string {
	payload := struct {
		SuccessCriteria []*uws1.Criterion     `json:"successCriteria,omitempty"`
		OnFailure       []*uws1.FailureAction `json:"onFailure,omitempty"`
		OnSuccess       []*uws1.SuccessAction `json:"onSuccess,omitempty"`
	}{
		SuccessCriteria: criteria,
		OnFailure:       failure,
		OnSuccess:       success,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "{}"
	}
	return string(data)
}

type compiledOperation struct {
	ServiceType        string
	OpenAPIOperationID string
	DependsOn          []string
	Parent             string
	Branch             string
	BranchWhen         string
	When               string
	ForEach            string
	Items              string
	Mode               string
	BatchSize          string
	Timeout            *float64
	Request            requestEvidence
	SuccessCriteria    []*uws1.Criterion
	OnFailure          []*uws1.FailureAction
	OnSuccess          []*uws1.SuccessAction
}

func compiledOperationIndex(doc *uws1.Document) (map[string]*compiledOperation, error) {
	return compiledOperationIndexWithRequestProjection(doc, nil)
}

func compiledOperationIndexWithRequestProjection(doc *uws1.Document, projectRequests func(*uws1.Document) (map[string]map[string]any, error)) (map[string]*compiledOperation, error) {
	out := map[string]*compiledOperation{}
	if doc == nil {
		return out, nil
	}
	operationEvidence := uwsOperationRequestEvidence(doc)
	if projectRequests != nil {
		compiledRequests, err := projectRequests(doc)
		if err != nil {
			return nil, err
		}
		for name, request := range compiledRequests {
			if strings.TrimSpace(name) == "" {
				continue
			}
			operationEvidence[strings.TrimSpace(name)] = requestEvidenceFromMap(request)
		}
	}
	collectCompiledUWSSteps(doc, out, operationEvidence)
	return out, nil
}

func collectCompiledUWSSteps(doc *uws1.Document, out map[string]*compiledOperation, operationEvidence map[string]requestEvidence) {
	if doc == nil {
		return
	}
	for _, workflow := range doc.Workflows {
		if workflow == nil {
			continue
		}
		collectCompiledUWSStepList(workflow.Steps, out, operationEvidence, operationIndexForDocument(doc), "", "", "")
		for _, branch := range workflow.Cases {
			if branch != nil {
				collectCompiledUWSStepList(branch.Steps, out, operationEvidence, operationIndexForDocument(doc), workflow.WorkflowID, strings.TrimSpace(branch.Name), strings.TrimSpace(branch.When))
			}
		}
		collectCompiledUWSStepList(workflow.Default, out, operationEvidence, operationIndexForDocument(doc), workflow.WorkflowID, "default", "")
	}
}

func collectCompiledUWSStepList(steps []*uws1.Step, out map[string]*compiledOperation, operationEvidence map[string]requestEvidence, operations map[string]*uws1.Operation, parent, branch, branchWhen string) {
	for _, step := range steps {
		if step == nil {
			continue
		}
		name := strings.TrimSpace(step.StepID)
		if name != "" {
			op := operations[strings.TrimSpace(step.OperationRef)]
			out[name] = &compiledOperation{
				ServiceType:        compiledServiceType(step, op),
				OpenAPIOperationID: compiledOpenAPIOperationID(step, op),
				DependsOn:          compiledDependsOn(step, op),
				Parent:             strings.TrimSpace(parent),
				Branch:             strings.TrimSpace(branch),
				BranchWhen:         strings.TrimSpace(branchWhen),
				When:               strings.TrimSpace(step.When),
				ForEach:            strings.TrimSpace(step.ForEach),
				Items:              strings.TrimSpace(step.Items),
				Mode:               strings.TrimSpace(step.Mode),
				BatchSize:          strings.TrimSpace(step.BatchSize),
				Timeout:            compiledTimeout(step, op),
				Request:            requestEvidenceForStep(step, operationEvidence),
				SuccessCriteria:    compiledCriteria(op),
				OnFailure:          compiledFailureActions(op),
				OnSuccess:          compiledSuccessActions(op),
			}
		}
		childParent := name
		if childParent == "" {
			childParent = parent
		}
		collectCompiledUWSStepList(step.Steps, out, operationEvidence, operations, childParent, "", "")
		for _, nestedBranch := range step.Cases {
			if nestedBranch != nil {
				collectCompiledUWSStepList(nestedBranch.Steps, out, operationEvidence, operations, childParent, strings.TrimSpace(nestedBranch.Name), strings.TrimSpace(nestedBranch.When))
			}
		}
		collectCompiledUWSStepList(step.Default, out, operationEvidence, operations, childParent, "default", "")
	}
}

func operationIndexForDocument(doc *uws1.Document) map[string]*uws1.Operation {
	out := map[string]*uws1.Operation{}
	if doc == nil {
		return out
	}
	for _, op := range doc.Operations {
		if op != nil && strings.TrimSpace(op.OperationID) != "" {
			out[strings.TrimSpace(op.OperationID)] = op
		}
	}
	return out
}

func compiledServiceType(step *uws1.Step, op *uws1.Operation) string {
	if step != nil && strings.TrimSpace(step.Type) != "" {
		return strings.TrimSpace(step.Type)
	}
	if op == nil {
		return ""
	}
	if op.HasOpenAPIBinding() {
		return "http"
	}
	if runtime, ok, err := uwsprofile.ReadOperationRuntime(op.Extensions); err == nil && ok {
		return runtime.Type
	}
	if strings.TrimSpace(op.ExtensionProfile()) != "" {
		return "fnct"
	}
	return ""
}

func compiledOpenAPIOperationID(step *uws1.Step, op *uws1.Operation) string {
	if op != nil && strings.TrimSpace(op.OpenAPIOperationID) != "" {
		return strings.TrimSpace(op.OpenAPIOperationID)
	}
	if step != nil {
		return strings.TrimSpace(step.OperationRef)
	}
	return ""
}

func compiledDependsOn(step *uws1.Step, op *uws1.Operation) []string {
	if op != nil && len(op.DependsOn) > 0 {
		return append([]string(nil), op.DependsOn...)
	}
	if step != nil {
		return append([]string(nil), step.DependsOn...)
	}
	return nil
}

func compiledTimeout(step *uws1.Step, op *uws1.Operation) *float64 {
	if op != nil && op.Timeout != nil {
		return cloneFloat64Ptr(op.Timeout)
	}
	if step != nil {
		return cloneFloat64Ptr(step.Timeout)
	}
	return nil
}

func compiledCriteria(op *uws1.Operation) []*uws1.Criterion {
	if op == nil {
		return nil
	}
	return cloneCriteria(op.SuccessCriteria)
}

func compiledFailureActions(op *uws1.Operation) []*uws1.FailureAction {
	if op == nil {
		return nil
	}
	return cloneFailureActions(op.OnFailure)
}

func compiledSuccessActions(op *uws1.Operation) []*uws1.SuccessAction {
	if op == nil {
		return nil
	}
	return cloneSuccessActions(op.OnSuccess)
}

type requestAttribute struct {
	Name       string
	Expression string
}

type requestEvidence []requestAttribute

func uwsOperationRequestEvidence(doc *uws1.Document) map[string]requestEvidence {
	out := map[string]requestEvidence{}
	if doc == nil {
		return out
	}
	for _, op := range doc.Operations {
		if op == nil || strings.TrimSpace(op.OperationID) == "" {
			continue
		}
		out[strings.TrimSpace(op.OperationID)] = requestEvidenceFromMap(op.Request)
	}
	return out
}

func requestEvidenceForStep(step *uws1.Step, operationEvidence map[string]requestEvidence) requestEvidence {
	if step == nil {
		return nil
	}
	for _, key := range []string{step.OperationRef, step.StepID} {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if evidence := operationEvidence[key]; len(evidence) > 0 {
			return evidence
		}
	}
	return nil
}

func requestEvidenceFromMap(request map[string]any) requestEvidence {
	var out requestEvidence
	for _, key := range sortedAnyKeys(request) {
		collectRequestEvidence(&out, []string{key}, request[key])
	}
	return uniqueRequestEvidence(out)
}

func collectRequestEvidence(out *requestEvidence, path []string, value any) {
	if len(path) == 0 {
		return
	}
	if values, ok := anyStringMap(value); ok {
		for _, key := range sortedAnyKeys(values) {
			collectRequestEvidence(out, appendPath(path, key), values[key])
		}
		return
	}
	if values, ok := anySlice(value); ok {
		for index, child := range values {
			collectRequestEvidence(out, appendPath(path, strconv.Itoa(index)), child)
		}
		return
	}
	name := strings.Join(path, ".")
	expression := requestValueExpression(value)
	if strings.TrimSpace(name) == "" || expression == "" {
		return
	}
	*out = append(*out, requestAttribute{Name: name, Expression: expression})
	for _, alias := range requestPathAliases(path) {
		*out = append(*out, requestAttribute{Name: strings.Join(alias, "."), Expression: expression})
	}
	leaf := strings.TrimSpace(path[len(path)-1])
	if leaf != "" && leaf != name {
		*out = append(*out, requestAttribute{Name: leaf, Expression: expression})
	}
}

func appendPath(path []string, next string) []string {
	out := append([]string(nil), path...)
	return append(out, next)
}

func requestPathAliases(path []string) [][]string {
	if len(path) == 0 {
		return nil
	}
	var out [][]string
	for _, alias := range requestSectionAliases(path[0]) {
		if alias == path[0] {
			continue
		}
		aliasPath := append([]string{alias}, path[1:]...)
		out = append(out, aliasPath)
	}
	return out
}

func requestSectionAliases(section string) []string {
	switch strings.TrimSpace(section) {
	case "path", "path_pars":
		return []string{"path", "path_pars"}
	case "query", "query_pars":
		return []string{"query", "query_pars"}
	case "header", "header_pars":
		return []string{"header", "header_pars"}
	case "cookie", "cookie_pars":
		return []string{"cookie", "cookie_pars"}
	case "body", "payload", "payload_pars":
		return []string{"body", "payload", "payload_pars"}
	default:
		return nil
	}
}

func sortedAnyKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func anyStringMap(value any) (map[string]any, bool) {
	switch typed := value.(type) {
	case map[string]any:
		return typed, true
	case map[any]any:
		out := make(map[string]any, len(typed))
		for key, child := range typed {
			out[strings.TrimSpace(fmt.Sprint(key))] = child
		}
		return out, true
	default:
		return nil, false
	}
}

func anySlice(value any) ([]any, bool) {
	switch typed := value.(type) {
	case []any:
		return typed, true
	default:
		return nil, false
	}
}

func requestValueExpression(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(typed)
	case fmt.Stringer:
		return strings.TrimSpace(typed.String())
	default:
		data, err := json.Marshal(typed)
		if err == nil && strings.TrimSpace(string(data)) != "" {
			return strings.TrimSpace(string(data))
		}
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}

func uniqueRequestEvidence(values requestEvidence) requestEvidence {
	if len(values) < 2 {
		return values
	}
	seen := map[string]bool{}
	out := values[:0]
	for _, value := range values {
		key := value.Name + "\x00" + value.Expression
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, value)
	}
	return out
}

func requestAttributeEvidence(evidence requestEvidence, names []string) (requestAttribute, bool) {
	if len(evidence) == 0 {
		return requestAttribute{}, false
	}
	for _, name := range names {
		for _, candidate := range evidence {
			if candidate.Name == name {
				return candidate, true
			}
		}
	}
	return requestAttribute{}, false
}

func requestAttributeEvidenceMatching(evidence requestEvidence, names []string, match func(requestAttribute) bool) (requestAttribute, bool) {
	if len(evidence) == 0 || match == nil {
		return requestAttribute{}, false
	}
	for _, name := range names {
		for _, candidate := range requestAttributeEvidences(evidence, name) {
			if match(candidate) {
				return candidate, true
			}
		}
	}
	return requestAttribute{}, false
}

func requestAttributeEvidences(evidence requestEvidence, name string) []requestAttribute {
	var out []requestAttribute
	for _, candidate := range evidence {
		if candidate.Name == name {
			out = append(out, candidate)
		}
	}
	return out
}

func equivalentWorkflowRuntime(want, got string) bool {
	return want == "openapi" && got == "http"
}

func bindingExpectedSource(binding PlanBinding) string {
	source := strings.TrimSpace(binding.Source)
	from := strings.TrimSpace(binding.From)
	if source == "" {
		return ""
	}
	if from != "" && !strings.HasPrefix(source, from+".") {
		return from + "." + source
	}
	return source
}

func expressionReferencesExpectedSource(expression, expected string) bool {
	expression = normalizeBindingExpression(expression)
	expected = normalizeBindingExpression(expected)
	if expected == "" {
		return true
	}
	if strings.Contains(expression, expected) {
		return true
	}
	if strings.Contains(expected, ".body") {
		alt := strings.Replace(expected, ".body", ".received_body", 1)
		if strings.Contains(expression, alt) {
			return true
		}
	}
	return false
}

func expressionReferencesInputSource(expression, expected string) bool {
	expression = normalizeBindingExpression(expression)
	expected = normalizeBindingExpression(expected)
	if expected == "" {
		return true
	}
	return expression == expected ||
		strings.Contains(expression, "input."+expected) ||
		strings.Contains(expression, "inputs."+expected) ||
		strings.Contains(expression, "var."+expected)
}

func normalizeBindingExpression(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, `"`)
	value = strings.TrimPrefix(value, "${")
	value = strings.TrimSuffix(value, "}")
	value = strings.ReplaceAll(value, "received_body", "body")
	value = regexp.MustCompile(`\[\s*([0-9]+)\s*\]`).ReplaceAllString(value, ".$1")
	value = regexp.MustCompile(`\[\s*"([^"]+)"\s*\]`).ReplaceAllString(value, ".$1")
	value = regexp.MustCompile(`\[\s*'([^']+)'\s*\]`).ReplaceAllString(value, ".$1")
	value = strings.ReplaceAll(value, "/", ".")
	for strings.Contains(value, "..") {
		value = strings.ReplaceAll(value, "..", ".")
	}
	value = strings.Trim(value, ".")
	return value
}

func paramCandidateNames(name string) []string {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	out := []string{name}
	if strings.Contains(name, ".") {
		parts := strings.Split(name, ".")
		last := strings.TrimSpace(parts[len(parts)-1])
		if last != "" {
			out = append(out, last)
		}
	}
	for _, prefix := range []string{"query.", "path.", "header.", "cookie.", "body.", "payload.", "query_pars.", "path_pars.", "header_pars.", "cookie_pars.", "payload_pars."} {
		if !strings.HasPrefix(name, prefix) {
			out = append(out, prefix+name)
		}
	}
	return sortedUnique(out)
}

func containsString(values []string, want string) bool {
	want = strings.TrimSpace(want)
	for _, value := range values {
		if strings.TrimSpace(value) == want {
			return true
		}
	}
	return false
}
