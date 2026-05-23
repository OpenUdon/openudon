package synthesize

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/OpenUdon/openudon/internal/openapidisco"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
	"github.com/OpenUdon/uws/uws1"
)

func assessIntent(report *QualityReport, path, exampleDir string, candidates []openapidisco.Candidate, primary string, policy projectPolicy) (*rollout.Intent, bool) {
	intent, err := rollout.ParseIntentFile(path)
	if err != nil {
		report.add("intent.parse", "fail", "intent.hcl is missing or invalid", err.Error())
		return nil, false
	}
	report.add("intent.parse", "pass", "intent.hcl parses", "")
	missing := intent.MissingSlots()
	if len(missing) > 0 && strings.TrimSpace(primary) != "" && strings.TrimSpace(intent.OpenAPI) == "" {
		intent.OpenAPI = strings.TrimSpace(primary)
		missing = intent.MissingSlots()
	}
	if len(missing) > 0 {
		report.add("intent.slots", "fail", "intent.hcl has missing slots", strings.Join(missing, "; "))
		return intent, false
	}
	report.add("intent.slots", "pass", "intent.hcl has required slots", "")
	if err := validateIntentOpenAPIRefs(intent, exampleDir, candidates, primary, policy.NoOpenAPI); err != nil {
		report.add("intent.openapi_refs", "fail", "intent.hcl references unavailable API source documents", err.Error())
		return intent, false
	}
	report.add("intent.openapi_refs", "pass", "intent.hcl API source references are available", "")
	if err := validateIntentOpenAPIOperations(intent, exampleDir, candidates, primary); err != nil {
		report.add("intent.openapi_operations", "fail", "intent.hcl references unavailable API source operations", err.Error())
		return intent, false
	}
	report.add("intent.openapi_operations", "pass", "intent.hcl API source operation references are available", "")
	if err := validateIntentRequiredParameters(intent, exampleDir, candidates, primary); err != nil {
		report.add("intent.data_flow.required_params", "fail", "required OpenAPI parameters are not satisfied", err.Error())
		return intent, false
	}
	report.add("intent.data_flow.required_params", "pass", "required OpenAPI parameters are satisfied or credential-bound", "")
	if err := validateIntentCredentialPolicy(intent, exampleDir, candidates, primary, policy); err != nil {
		report.add("credentials.bindings", "fail", "credential-like parameters require declared credential policy", err.Error())
		return intent, false
	}
	report.add("credentials.bindings", "pass", "credential-like parameters are covered by project credential policy or not required", "")
	if err := validateIntentOpenAPISecurity(intent, exampleDir, candidates, primary, policy); err != nil {
		report.add("credentials.security_schemes", "fail", "API source security requirements need credential bindings", err.Error())
		return intent, false
	}
	report.add("credentials.security_schemes", "pass", "API source security requirements are covered by credential policy or not required", "")
	if err := validateIntentDataFlowSources(intent); err != nil {
		report.add("intent.data_flow.sources", "fail", "intent.hcl references unresolved data-flow sources", err.Error())
		return intent, false
	}
	report.add("intent.data_flow.sources", "pass", "intent.hcl data-flow references resolve to known steps or inputs", "")
	responsePathResult := validateIntentResponsePaths(intent, exampleDir, candidates, primary)
	if len(responsePathResult.Failures) > 0 {
		report.add("intent.data_flow.response_paths", "fail", "intent.hcl references response fields absent from API source schemas", strings.Join(sortedCopy(responsePathResult.Failures), "; "))
		return intent, false
	}
	if len(responsePathResult.Warnings) > 0 {
		report.add("intent.data_flow.response_paths", "warn", "some intent.hcl response paths could not be proven from API source schemas", strings.Join(sortedCopy(responsePathResult.Warnings), "; "))
	} else {
		report.add("intent.data_flow.response_paths", "pass", "intent.hcl response paths match available API source response schemas", "")
	}
	addIntentDataFlowWarning(report, intent)
	if err := validateIntentFunctionContracts(intent, policy); err != nil {
		report.add("intent.function_contracts", "fail", "fnct steps must match declared project function contracts", err.Error())
		return intent, false
	}
	report.add("intent.function_contracts", "pass", "fnct steps match declared project function contracts", "")
	if err := validateIntentRuntimePolicy(intent, policy); err != nil {
		report.add("intent.runtime_policy", "fail", "intent.hcl uses runtimes not allowed by project.md", err.Error())
		return intent, false
	}
	report.add("intent.runtime_policy", "pass", "intent.hcl respects project runtime policy", "")
	if err := validateIntentProjectMetadataPolicy(intent, policy); err != nil {
		report.add("intent.project_policy", "fail", "intent.hcl does not preserve required project controls", err.Error())
		return intent, false
	}
	report.add("intent.project_policy", "pass", "intent.hcl preserves required project controls", "")
	return intent, true
}

func addIntentDataFlowWarning(report *QualityReport, intent *rollout.Intent) {
	if intent == nil || intentStepCount(intent) < 2 {
		report.add("intent.data_flow.explicit", "pass", "single-step intent does not require cross-step data-flow evidence", "")
		return
	}
	if intentHasDataFlowEvidence(intent) {
		report.add("intent.data_flow.explicit", "pass", "multi-step intent includes explicit data-flow evidence", "")
		return
	}
	report.add("intent.data_flow.explicit", "warn", "multi-step intent has no explicit bind, with, or prior-step references", "Add Data Flow guidance to project.md or bind/with hints to intent.hcl.")
}

func validateIntentProjectMetadataPolicy(intent *rollout.Intent, policy projectPolicy) error {
	var mismatches []string
	if policy.WorkflowTimeout != nil {
		var got *float64
		if intent != nil && intent.Workflow != nil {
			got = intent.Workflow.Timeout
		}
		if !floatPtrEqual(got, policy.WorkflowTimeout) {
			mismatches = append(mismatches, fmt.Sprintf("workflow timeout expected %g got %s", *policy.WorkflowTimeout, formatFloatPtr(got)))
		}
	}
	if policy.Idempotency != nil {
		var got *uws1.Idempotency
		if intent != nil && intent.Workflow != nil {
			got = intent.Workflow.Idempotency
		}
		if !idempotencyEqual(got, policy.Idempotency) {
			mismatches = append(mismatches, fmt.Sprintf("workflow idempotency expected %s got %s", idempotencySummary(policy.Idempotency), idempotencySummary(got)))
		}
	}
	if len(policy.StepTimeouts) > 0 {
		steps := map[string]*rollout.Step{}
		if intent != nil {
			walkIntentSteps(intent.Steps, func(step *rollout.Step) {
				if step != nil && strings.TrimSpace(step.Name) != "" {
					steps[strings.TrimSpace(step.Name)] = step
				}
			})
		}
		for _, name := range sortedStepTimeoutKeys(policy.StepTimeouts) {
			expected := policy.StepTimeouts[name]
			var got *float64
			if step := steps[name]; step != nil {
				got = step.Timeout
			}
			if got == nil || !floatPtrEqual(got, &expected) {
				mismatches = append(mismatches, fmt.Sprintf("%s timeout expected %g got %s", name, expected, formatFloatPtr(got)))
			}
		}
	}
	if len(mismatches) > 0 {
		return fmt.Errorf("%s", strings.Join(sortedCopy(mismatches), "; "))
	}
	return nil
}

func intentStepCount(intent *rollout.Intent) int {
	var count int
	walkIntentSteps(intent.Steps, func(step *rollout.Step) {
		count++
	})
	return count
}

func intentHasDataFlowEvidence(intent *rollout.Intent) bool {
	stepNames := map[string]bool{}
	walkIntentSteps(intent.Steps, func(step *rollout.Step) {
		if step != nil && strings.TrimSpace(step.Name) != "" {
			stepNames[strings.TrimSpace(step.Name)] = true
		}
	})
	found := false
	walkIntentSteps(intent.Steps, func(step *rollout.Step) {
		if step == nil || found {
			return
		}
		if len(step.Binds) > 0 {
			found = true
			return
		}
		for _, value := range step.With {
			if referencesStep(value, stepNames) {
				found = true
				return
			}
		}
	})
	return found
}

func referencesStep(value string, stepNames map[string]bool) bool {
	for name := range stepNames {
		if strings.Contains(value, name+".") || strings.Contains(value, "${"+name+".") {
			return true
		}
	}
	return false
}

func validateIntentFunctionContracts(intent *rollout.Intent, policy projectPolicy) error {
	if intent == nil {
		return nil
	}
	fnctSteps := intentFunctionSteps(intent)
	if len(fnctSteps) == 0 {
		return nil
	}
	if functionSectionForbidsSteps(policy.FunctionSection) {
		return fmt.Errorf("project Function Contracts says no function steps are expected, but intent has fnct step(s): %s", strings.Join(intentStepNames(fnctSteps), ", "))
	}
	contracts := functionContractIndex(policy.FunctionContracts)
	var missingContract, missingEvidence, undeclaredInputs []string
	for _, step := range fnctSteps {
		name := strings.TrimSpace(step.Name)
		if name == "" {
			name = "<unnamed>"
		}
		contract := contracts[name]
		if contract == nil {
			missingContract = append(missingContract, name)
			continue
		}
		inputTargets := fnctInputTargets(step)
		if functionContractRequiresInputs(*contract) && len(inputTargets) == 0 && !fnctStepReferencesPriorStep(step) {
			missingEvidence = append(missingEvidence, name)
		}
		allowed := simpleFunctionContractInputs(*contract)
		if len(allowed) == 0 || len(inputTargets) == 0 {
			continue
		}
		for target := range inputTargets {
			if !allowed[target] {
				undeclaredInputs = append(undeclaredInputs, fmt.Sprintf("%s.%s", name, target))
			}
		}
	}
	if len(missingContract) == 0 && len(missingEvidence) == 0 && len(undeclaredInputs) == 0 {
		return nil
	}
	var details []string
	if len(missingContract) > 0 {
		details = append(details, "missing Function Contracts entries for "+strings.Join(sortedCopy(missingContract), ", "))
	}
	if len(missingEvidence) > 0 {
		details = append(details, "declared function inputs have no with/bind/prior-step evidence for "+strings.Join(sortedCopy(missingEvidence), ", "))
	}
	if len(undeclaredInputs) > 0 {
		details = append(details, "fnct inputs not declared by contract: "+strings.Join(sortedCopy(undeclaredInputs), ", "))
	}
	return fmt.Errorf("%s. Update the Function Contracts section or repair intent.hcl bindings.", strings.Join(details, "; "))
}

func intentFunctionSteps(intent *rollout.Intent) []*rollout.Step {
	if intent == nil {
		return nil
	}
	var out []*rollout.Step
	walkIntentSteps(intent.Steps, func(step *rollout.Step) {
		if step != nil && strings.EqualFold(strings.TrimSpace(step.Type), "fnct") {
			out = append(out, step)
		}
	})
	return out
}

func functionSectionForbidsSteps(section string) bool {
	lower := strings.ToLower(section)
	return strings.Contains(lower, "no function steps") || strings.Contains(lower, "no fnct steps")
}

func functionContractIndex(contracts []FunctionContract) map[string]*FunctionContract {
	out := map[string]*FunctionContract{}
	for i := range contracts {
		name := strings.TrimSpace(contracts[i].Name)
		if name == "" || strings.Contains(strings.ToLower(name), "no function steps") {
			continue
		}
		out[name] = &contracts[i]
	}
	return out
}

func functionContractRequiresInputs(contract FunctionContract) bool {
	if len(contract.Inputs) == 0 {
		return false
	}
	for _, input := range contract.Inputs {
		if !contractInputMeansNone(input) {
			return true
		}
	}
	return false
}

func contractInputMeansNone(input string) bool {
	lower := strings.ToLower(strings.TrimSpace(strings.Trim(input, "`.")))
	return lower == "" || lower == "none" || lower == "no inputs" || lower == "no input"
}

func fnctInputTargets(step *rollout.Step) map[string]bool {
	out := map[string]bool{}
	if step == nil {
		return out
	}
	for target := range step.With {
		target = strings.TrimSpace(target)
		if target != "" {
			out[target] = true
		}
	}
	for _, bind := range step.Binds {
		if bind == nil {
			continue
		}
		for target := range bind.Fields {
			target = strings.TrimSpace(target)
			if target != "" {
				out[target] = true
			}
		}
	}
	return out
}

func fnctStepReferencesPriorStep(step *rollout.Step) bool {
	if step == nil {
		return false
	}
	for _, value := range step.With {
		if referencesKnownStep(value, step) {
			return true
		}
	}
	for _, bind := range step.Binds {
		if bind != nil && strings.TrimSpace(bind.From) != "" {
			return true
		}
	}
	return false
}

func simpleFunctionContractInputs(contract FunctionContract) map[string]bool {
	out := map[string]bool{}
	for _, input := range contract.Inputs {
		input = strings.TrimSpace(strings.Trim(input, "` ."))
		if contractInputMeansNone(input) {
			continue
		}
		if !regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]*$`).MatchString(input) {
			continue
		}
		out[input] = true
	}
	return out
}

func intentStepNames(steps []*rollout.Step) []string {
	var out []string
	for _, step := range steps {
		if step == nil {
			continue
		}
		name := strings.TrimSpace(step.Name)
		if name == "" {
			name = "<unnamed>"
		}
		out = append(out, name)
	}
	return sortedCopy(out)
}
