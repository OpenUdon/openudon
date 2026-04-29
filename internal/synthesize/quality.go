package synthesize

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/genelet/hcllight/light"
	"github.com/genelet/ramen/internal/openapidisco"
	"github.com/genelet/ramen/internal/uwsvalidate"
	"github.com/genelet/udon/generator"
	"github.com/genelet/udon/pkg/rollout"
	"github.com/genelet/udon/pkg/runtimeplan"
	"github.com/genelet/udon/pkg/uwsprofile"
	"github.com/tabilet/uws/uws1"
	"gopkg.in/yaml.v3"
)

type QualityReport struct {
	Status    string         `json:"status"`
	Example   string         `json:"example"`
	Artifacts Result         `json:"artifacts"`
	Checks    []QualityCheck `json:"checks"`
}

type QualityCheck struct {
	Code    string `json:"code"`
	Status  string `json:"status"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

func (r *QualityReport) Passed() bool {
	return r != nil && r.Status == "pass"
}

func Assess(opts Options) (*QualityReport, error) {
	return AssessContext(context.Background(), opts)
}

func AssessContext(ctx context.Context, opts Options) (*QualityReport, error) {
	return assessContext(ctx, opts, true)
}

func AssessCurrent(ctx context.Context, opts Options) (*QualityReport, error) {
	return assessContext(ctx, opts, false)
}

func assessContext(ctx context.Context, opts Options, writeReport bool) (*QualityReport, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	exampleDir, err := resolveExampleDir(opts.ExampleDir)
	if err != nil {
		return nil, err
	}
	result := resultPaths(exampleDir)
	report := &QualityReport{
		Status:    "pass",
		Example:   exampleDir,
		Artifacts: result,
	}

	projectBytes, projectErr := os.ReadFile(result.ProjectPath)
	projectText := string(projectBytes)
	policy := analyzeProject(projectText)
	if projectErr != nil {
		report.add("project.present", "fail", "project.md is required", projectErr.Error())
	} else {
		report.add("project.present", "pass", "project.md is readable", "")
		addProjectAuthoringChecks(report, projectText)
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}
	candidates, err := openapidisco.LocalFiles(filepath.Join(exampleDir, "openapi"), exampleDir, projectText)
	if err != nil && !(policy.NoOpenAPI && errors.Is(err, os.ErrNotExist)) {
		report.add("openapi.local", "fail", "OpenAPI directory could not be scanned", err.Error())
	} else if policy.NoOpenAPI {
		result.OpenAPICandidates = candidates
		report.Artifacts = result
		report.add("openapi.local", "pass", "project explicitly declares OpenAPI is not required", candidateList(candidates))
	} else if len(candidates) == 0 {
		report.add("openapi.local", "fail", "no local OpenAPI documents are available", "Add a valid .json, .yaml, or .yml OpenAPI document under openapi/, or rerun synthesize with an explicit URL in project.md.")
	} else {
		report.add("openapi.local", "pass", fmt.Sprintf("%d OpenAPI document(s) available", len(candidates)), candidateList(candidates))
		result.OpenAPICandidates = candidates
		if primary, err := openapidisco.SelectPrimary(candidates); err == nil {
			result.PrimaryOpenAPI = primary.RelativePath
		}
		report.Artifacts = result
	}

	expectedPlan := assessWorkflowPlan(report, result)
	assessDiscoveryReport(report, result.DiscoveryJSONPath)
	intent, intentOK := assessIntent(report, result.IntentPath, candidates, result.PrimaryOpenAPI, policy)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	planOK := assessWorkflow(report, result.WorkflowPath, exampleDir, intent, policy, expectedPlan)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	assessUWS(report, result.UWSPath, opts.SchemaPath, exampleDir, expectedPlan)
	sideEffects := sideEffectProfileForOpenAPI(policy, intent, candidates, result.PrimaryOpenAPI)
	assessSideEffectProfile(report, sideEffects)
	assessSideEffectRetryPolicy(report, sideEffects, policy, expectedPlan)
	assessReview(report, result.ReviewPath, sideEffects, policy, expectedPlan)
	assessSymphonyHandoff(report, result.SymphonyHandoffPath, sideEffects, policy, expectedPlan)
	assessSecrets(report, result)

	if intentOK && planOK {
		report.add("quality.review", "pass", "workflow.hcl passed deterministic v1 quality gates", "")
	}
	report.finalize()
	if writeReport {
		if err := writeQualityFiles(result, report); err != nil {
			return nil, err
		}
	}
	return report, nil
}

func assessIntent(report *QualityReport, path string, candidates []openapidisco.Candidate, primary string, policy projectPolicy) (*rollout.Intent, bool) {
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
	if err := validateIntentOpenAPIRefs(intent, candidates, primary, policy.NoOpenAPI); err != nil {
		report.add("intent.openapi_refs", "fail", "intent.hcl references unavailable OpenAPI documents", err.Error())
		return intent, false
	}
	report.add("intent.openapi_refs", "pass", "intent.hcl OpenAPI references are available", "")
	if err := validateIntentOpenAPIOperations(intent, candidates, primary); err != nil {
		report.add("intent.openapi_operations", "fail", "intent.hcl references unavailable OpenAPI operations", err.Error())
		return intent, false
	}
	report.add("intent.openapi_operations", "pass", "intent.hcl OpenAPI operation references are available", "")
	if err := validateIntentRequiredParameters(intent, candidates, primary); err != nil {
		report.add("intent.data_flow.required_params", "fail", "required OpenAPI parameters are not satisfied", err.Error())
		return intent, false
	}
	report.add("intent.data_flow.required_params", "pass", "required OpenAPI parameters are satisfied or credential-bound", "")
	if err := validateIntentCredentialPolicy(intent, candidates, primary, policy); err != nil {
		report.add("credentials.bindings", "fail", "credential-like parameters require declared credential policy", err.Error())
		return intent, false
	}
	report.add("credentials.bindings", "pass", "credential-like parameters are covered by project credential policy or not required", "")
	if err := validateIntentOpenAPISecurity(intent, candidates, primary, policy); err != nil {
		report.add("credentials.security_schemes", "fail", "OpenAPI security requirements need credential bindings", err.Error())
		return intent, false
	}
	report.add("credentials.security_schemes", "pass", "OpenAPI security requirements are covered by credential policy or not required", "")
	if err := validateIntentDataFlowSources(intent); err != nil {
		report.add("intent.data_flow.sources", "fail", "intent.hcl references unresolved data-flow sources", err.Error())
		return intent, false
	}
	report.add("intent.data_flow.sources", "pass", "intent.hcl data-flow references resolve to known steps or inputs", "")
	responsePathResult := validateIntentResponsePaths(intent, candidates, primary)
	if len(responsePathResult.Failures) > 0 {
		report.add("intent.data_flow.response_paths", "fail", "intent.hcl references response fields absent from OpenAPI schemas", strings.Join(sortedCopy(responsePathResult.Failures), "; "))
		return intent, false
	}
	if len(responsePathResult.Warnings) > 0 {
		report.add("intent.data_flow.response_paths", "warn", "some intent.hcl response paths could not be proven from OpenAPI schemas", strings.Join(sortedCopy(responsePathResult.Warnings), "; "))
	} else {
		report.add("intent.data_flow.response_paths", "pass", "intent.hcl response paths match available OpenAPI response schemas", "")
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

func assessWorkflowPlan(report *QualityReport, result Result) *WorkflowPlan {
	plan, err := loadWorkflowPlan(result.PlanJSONPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if generatedArtifactsExist(result) {
				report.add("plan.present", "fail", "expected workflow plan is missing", "Run `ramen synthesize` or `ramen build` to create expected/plan.json.")
			} else {
				report.add("plan.present", "warn", "expected workflow plan is missing", "Run `ramen synthesize` or `ramen build` to create expected/plan.json.")
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
			report.add("openapi.discovery", "warn", "OpenAPI discovery report is missing", "Run `ramen synthesize` or `ramen build` to record OpenAPI discovery attempts.")
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
	compiledPlan, err := generator.NewRuntimePlanFromWorkflowFile(path, exampleDir)
	if err != nil {
		report.add("workflow.udon_compile", "fail", "workflow.hcl does not compile through udon", err.Error())
		return false
	}
	report.add("workflow.udon_compile", "pass", "workflow.hcl compiles through udon", "")
	if intent != nil {
		missing := missingIntentSteps(intent, compiledPlan.Document().Workflows)
		if len(missing) > 0 {
			report.add("workflow.intent_coverage", "fail", "workflow.hcl does not represent every intent step", strings.Join(missing, ", "))
			return false
		}
		report.add("workflow.intent_coverage", "pass", "workflow.hcl represents intent steps", "")
	}
	if expectedPlan != nil && !validateWorkflowAgainstExpectedPlan(report, compiledPlan, expectedPlan) {
		return false
	}
	return true
}

func validateIntentOpenAPIOperations(intent *rollout.Intent, candidates []openapidisco.Candidate, primary string) error {
	if intent == nil {
		return nil
	}
	ops := openAPIOperationIndex(candidates)
	var missing []string
	var omitted []string
	walkIntentSteps(intent.Steps, func(step *rollout.Step) {
		if step == nil {
			return
		}
		operation := strings.TrimSpace(step.Operation)
		specPath := intentStepOpenAPIPath(intent, step, primary)
		if operation == "" {
			if intentStepRequiresOpenAPIOperation(intent, step, primary) {
				name := strings.TrimSpace(step.Name)
				if name == "" {
					name = "<unnamed>"
				}
				omitted = append(omitted, fmt.Sprintf("%s in %q", name, specPath))
			}
			return
		}
		if op := ops[operationKey(specPath, operation)]; op == nil {
			name := strings.TrimSpace(step.Name)
			if name == "" {
				name = "<unnamed>"
			}
			missing = append(missing, fmt.Sprintf("%s operation %q in %q", name, operation, specPath))
		}
	})
	if len(omitted) > 0 || len(missing) > 0 {
		sort.Strings(omitted)
		sort.Strings(missing)
		var details []string
		for _, item := range omitted {
			details = append(details, "missing operation for "+item)
		}
		for _, item := range missing {
			details = append(details, "missing OpenAPI operation "+item)
		}
		return fmt.Errorf("%s", strings.Join(details, "; "))
	}
	return nil
}

func intentStepRequiresOpenAPIOperation(intent *rollout.Intent, step *rollout.Step, primary string) bool {
	if step == nil {
		return false
	}
	kind := strings.ToLower(strings.TrimSpace(step.Type))
	if kind != "" && kind != "http" && kind != "openapi" {
		return false
	}
	return strings.TrimSpace(intentStepOpenAPIPath(intent, step, primary)) != ""
}

func validateIntentRequiredParameters(intent *rollout.Intent, candidates []openapidisco.Candidate, primary string) error {
	if intent == nil {
		return nil
	}
	ops := openAPIOperationIndex(candidates)
	inputs := intentInputNames(intent)
	var missing []string
	walkIntentSteps(intent.Steps, func(step *rollout.Step) {
		if step == nil {
			return
		}
		operation := strings.TrimSpace(step.Operation)
		if operation == "" {
			return
		}
		specPath := intentStepOpenAPIPath(intent, step, primary)
		op := ops[operationKey(specPath, operation)]
		if op == nil {
			return
		}
		for _, param := range op.Parameters {
			if param == nil || !param.Required || credentialLikeParam(param.Name) {
				continue
			}
			if stepSatisfiesParam(step, param, inputs) {
				continue
			}
			name := strings.TrimSpace(step.Name)
			if name == "" {
				name = "<unnamed>"
			}
			missing = append(missing, fmt.Sprintf("%s.%s requires %s parameter %q", name, operation, param.In, param.Name))
		}
	})
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("%s. Add literals, inputs, bind/with mappings, or import a complementary OpenAPI document that produces the missing values.", strings.Join(missing, "; "))
	}
	return nil
}

func validateIntentCredentialPolicy(intent *rollout.Intent, candidates []openapidisco.Candidate, primary string, policy projectPolicy) error {
	if intent == nil {
		return nil
	}
	ops := openAPIOperationIndex(candidates)
	inputs := intentInputNames(intent)
	var required, missingBinding []string
	walkIntentSteps(intent.Steps, func(step *rollout.Step) {
		if step == nil {
			return
		}
		operation := strings.TrimSpace(step.Operation)
		if operation == "" {
			return
		}
		specPath := intentStepOpenAPIPath(intent, step, primary)
		op := ops[operationKey(specPath, operation)]
		if op == nil {
			return
		}
		for _, param := range op.Parameters {
			if param == nil || !param.Required || !credentialLikeParam(param.Name) {
				continue
			}
			name := strings.TrimSpace(step.Name)
			if name == "" {
				name = "<unnamed>"
			}
			required = append(required, fmt.Sprintf("%s.%s requires credential-like parameter %q", name, operation, param.Name))
			if stepSatisfiesParam(step, param, inputs) {
				continue
			}
			if credentialDeclaredForParam(policy, param.Name) {
				continue
			}
			missingBinding = append(missingBinding, fmt.Sprintf("%s.%s has no auditable credential binding for %q", name, operation, param.Name))
		}
	})
	if len(required) == 0 {
		return nil
	}
	if strings.TrimSpace(policy.CredentialSection) == "" {
		sort.Strings(required)
		return fmt.Errorf("%s. Add a Credentials and Secrets section that names runtime credential bindings, never literal secrets.", strings.Join(required, "; "))
	}
	if len(missingBinding) > 0 {
		sort.Strings(missingBinding)
		return fmt.Errorf("%s. Add a with/bind request mapping or name a credential binding that includes the parameter name.", strings.Join(missingBinding, "; "))
	}
	return nil
}

func validateIntentOpenAPISecurity(intent *rollout.Intent, candidates []openapidisco.Candidate, primary string, policy projectPolicy) error {
	if intent == nil {
		return nil
	}
	security := openAPISecurityIndex(candidates)
	var required, missing []string
	walkIntentSteps(intent.Steps, func(step *rollout.Step) {
		if step == nil || strings.TrimSpace(step.Operation) == "" {
			return
		}
		reqs := security[operationKey(intentStepOpenAPIPath(intent, step, primary), step.Operation)]
		if len(reqs) == 0 {
			return
		}
		name := strings.TrimSpace(step.Name)
		if name == "" {
			name = "<unnamed>"
		}
		for _, req := range reqs {
			label := req.label()
			required = append(required, fmt.Sprintf("%s.%s requires OpenAPI security %q", name, step.Operation, label))
			if intentSecurityCoversRequirement(intent, req) || stepCoversSecurityRequirement(step, req, policy) || credentialDeclaredForSecurity(policy, req) {
				continue
			}
			missing = append(missing, fmt.Sprintf("%s.%s has no auditable credential binding for OpenAPI security %q", name, step.Operation, label))
		}
	})
	if len(required) == 0 {
		return nil
	}
	if strings.TrimSpace(policy.CredentialSection) == "" {
		return fmt.Errorf("%s. Add a Credentials and Secrets section that names security credential bindings, never literal secrets.", strings.Join(sortedCopy(required), "; "))
	}
	if len(missing) > 0 {
		return fmt.Errorf("%s. Bind the security field by credential binding name or add a matching credential binding policy.", strings.Join(sortedCopy(missing), "; "))
	}
	return nil
}

func credentialDeclaredForParam(policy projectPolicy, name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return false
	}
	for _, binding := range credentialBindingNames(policy) {
		if strings.Contains(strings.ToLower(binding), name) {
			return true
		}
	}
	return false
}

func credentialDeclaredForSecurity(policy projectPolicy, req openAPISecurityRequirement) bool {
	for _, binding := range credentialBindingNames(policy) {
		if securityBindingMatches(binding, req) {
			return true
		}
	}
	return false
}

func securityBindingMatches(binding string, req openAPISecurityRequirement) bool {
	binding = strings.ToLower(strings.TrimSpace(binding))
	if binding == "" {
		return false
	}
	for _, candidate := range req.bindingCandidates() {
		candidate = strings.ToLower(strings.TrimSpace(candidate))
		if candidate != "" && (strings.Contains(binding, candidate) || strings.Contains(candidate, binding)) {
			return true
		}
	}
	return false
}

func intentSecurityCoversRequirement(intent *rollout.Intent, req openAPISecurityRequirement) bool {
	if intent == nil {
		return false
	}
	for _, security := range intent.Security {
		if security == nil {
			continue
		}
		for _, candidate := range []string{security.Name, security.TokenFrom} {
			if securityBindingMatches(candidate, req) {
				return true
			}
		}
	}
	return false
}

func stepCoversSecurityRequirement(step *rollout.Step, req openAPISecurityRequirement, policy projectPolicy) bool {
	if step == nil {
		return false
	}
	names := req.fieldNames()
	for _, name := range names {
		if source := strings.TrimSpace(step.With[name]); source != "" && securityCredentialSourceAllowed(source, req, policy) {
			return true
		}
		for _, bind := range step.Binds {
			if bind == nil {
				continue
			}
			if source := strings.TrimSpace(bind.Fields[name]); source != "" && securityCredentialSourceAllowed(source, req, policy) {
				return true
			}
		}
	}
	return false
}

func securityCredentialSourceAllowed(source string, req openAPISecurityRequirement, policy projectPolicy) bool {
	if securityBindingMatches(source, req) {
		return true
	}
	for _, binding := range credentialBindingNames(policy) {
		if strings.EqualFold(strings.TrimSpace(source), strings.TrimSpace(binding)) {
			return true
		}
	}
	return false
}

func openAPIOperationIndex(candidates []openapidisco.Candidate) map[string]*rollout.OperationInfo {
	out := map[string]*rollout.OperationInfo{}
	for _, candidate := range candidates {
		spec, err := rollout.LoadOpenAPISpec(candidate.Path)
		if err != nil {
			continue
		}
		for _, op := range spec.Operations {
			if op == nil || strings.TrimSpace(op.OperationID) == "" {
				continue
			}
			out[operationKey(candidate.RelativePath, op.OperationID)] = op
		}
	}
	return out
}

type openAPISecurityRequirement struct {
	Scheme string
	Name   string
	In     string
	Type   string
}

func (r openAPISecurityRequirement) label() string {
	if strings.TrimSpace(r.Scheme) != "" {
		return strings.TrimSpace(r.Scheme)
	}
	if strings.TrimSpace(r.Name) != "" {
		return strings.TrimSpace(r.Name)
	}
	return "security"
}

func (r openAPISecurityRequirement) fieldNames() []string {
	var out []string
	for _, name := range []string{r.Name, r.Scheme} {
		name = strings.TrimSpace(name)
		if name != "" {
			out = append(out, name)
		}
	}
	if strings.EqualFold(r.Type, "http") || strings.EqualFold(r.Scheme, "bearer") || strings.Contains(strings.ToLower(r.Scheme), "bearer") {
		out = append(out, "Authorization", "authorization", "header.Authorization", "header.authorization", "header_pars.Authorization", "header_pars.authorization")
	}
	switch strings.ToLower(strings.TrimSpace(r.In)) {
	case "query":
		for _, name := range []string{r.Name, r.Scheme} {
			if strings.TrimSpace(name) != "" {
				out = append(out, "query."+name, "query_pars."+name)
			}
		}
	case "header":
		for _, name := range []string{r.Name, r.Scheme} {
			if strings.TrimSpace(name) != "" {
				out = append(out, "header."+name, "header_pars."+name)
			}
		}
	}
	return sortedUnique(out)
}

func (r openAPISecurityRequirement) bindingCandidates() []string {
	return sortedUnique([]string{r.Scheme, r.Name, strings.ReplaceAll(r.Name, "-", "_"), strings.ReplaceAll(r.Scheme, "-", "_")})
}

func openAPISecurityIndex(candidates []openapidisco.Candidate) map[string][]openAPISecurityRequirement {
	out := map[string][]openAPISecurityRequirement{}
	for _, candidate := range candidates {
		doc, err := readOpenAPISecurityDocument(candidate.Path)
		if err != nil {
			continue
		}
		schemes := openAPISecuritySchemes(doc)
		global := openAPISecurityRequirements(asMap(doc["security"]), schemes)
		paths := asMap(doc["paths"])
		for path, rawPathItem := range paths {
			pathItem := asMap(rawPathItem)
			for _, method := range []string{"get", "post", "put", "patch", "delete", "head", "options"} {
				rawOp, ok := pathItem[method]
				if !ok {
					continue
				}
				op := asMap(rawOp)
				operationID := strings.TrimSpace(asString(op["operationId"]))
				if operationID == "" {
					continue
				}
				requirements := global
				if rawSecurity, ok := op["security"]; ok {
					requirements = openAPISecurityRequirements(asMap(rawSecurity), schemes)
				}
				if len(requirements) == 0 {
					continue
				}
				_ = path
				out[operationKey(candidate.RelativePath, operationID)] = requirements
			}
		}
	}
	return out
}

func readOpenAPISecurityDocument(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	return asMap(raw), nil
}

func openAPISecuritySchemes(doc map[string]any) map[string]openAPISecurityRequirement {
	out := map[string]openAPISecurityRequirement{}
	components := asMap(doc["components"])
	schemes := asMap(components["securitySchemes"])
	if len(schemes) == 0 {
		schemes = asMap(doc["securityDefinitions"])
	}
	for name, raw := range schemes {
		scheme := asMap(raw)
		out[name] = openAPISecurityRequirement{
			Scheme: name,
			Name:   asString(scheme["name"]),
			In:     asString(scheme["in"]),
			Type:   asString(scheme["type"]),
		}
	}
	return out
}

func openAPISecurityRequirements(raw map[string]any, schemes map[string]openAPISecurityRequirement) []openAPISecurityRequirement {
	var out []openAPISecurityRequirement
	for _, item := range asSlice(raw) {
		req := asMap(item)
		for name := range req {
			if scheme, ok := schemes[name]; ok {
				out = append(out, scheme)
				continue
			}
			out = append(out, openAPISecurityRequirement{Scheme: name})
		}
	}
	return sortedSecurityRequirements(out)
}

func sortedSecurityRequirements(values []openAPISecurityRequirement) []openAPISecurityRequirement {
	sort.SliceStable(values, func(i, j int) bool {
		if values[i].Scheme != values[j].Scheme {
			return values[i].Scheme < values[j].Scheme
		}
		return values[i].Name < values[j].Name
	})
	return values
}

func asMap(value any) map[string]any {
	switch typed := value.(type) {
	case map[string]any:
		return typed
	case map[any]any:
		out := map[string]any{}
		for key, val := range typed {
			out[fmt.Sprint(key)] = val
		}
		return out
	case []any:
		out := map[string]any{}
		for i, item := range typed {
			out[fmt.Sprint(i)] = item
		}
		return out
	default:
		return nil
	}
}

func asSlice(value any) []any {
	switch typed := value.(type) {
	case []any:
		return typed
	case map[string]any:
		if len(typed) == 0 {
			return nil
		}
		out := make([]any, 0, len(typed))
		keys := sortedMapKeys(typed)
		for _, key := range keys {
			out = append(out, typed[key])
		}
		return out
	default:
		return nil
	}
}

func asString(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	default:
		return strings.TrimSpace(fmt.Sprint(value))
	}
}

func sortedMapKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func operationKey(specPath, operation string) string {
	return strings.TrimSpace(specPath) + "\x00" + strings.TrimSpace(operation)
}

func intentStepOpenAPIPath(intent *rollout.Intent, step *rollout.Step, primary string) string {
	if step != nil && strings.TrimSpace(step.OpenAPI) != "" {
		return strings.TrimSpace(step.OpenAPI)
	}
	if intent != nil && strings.TrimSpace(intent.OpenAPI) != "" {
		return strings.TrimSpace(intent.OpenAPI)
	}
	return strings.TrimSpace(primary)
}

func intentInputNames(intent *rollout.Intent) map[string]bool {
	out := map[string]bool{}
	if intent == nil {
		return out
	}
	for _, input := range intent.Inputs {
		if input != nil && strings.TrimSpace(input.Name) != "" {
			out[strings.TrimSpace(input.Name)] = true
		}
	}
	return out
}

func credentialLikeParam(name string) bool {
	lower := strings.ToLower(strings.TrimSpace(name))
	for _, token := range []string{"key", "token", "secret", "password", "appid", "api_key", "apikey", "authorization"} {
		if strings.Contains(lower, token) {
			return true
		}
	}
	return false
}

func stepSatisfiesParam(step *rollout.Step, param *rollout.ParameterInfo, inputs map[string]bool) bool {
	if step == nil || param == nil {
		return false
	}
	names := paramTargetNames(param)
	for _, name := range names {
		if step.With[name] != "" {
			return true
		}
		for _, bind := range step.Binds {
			if bind != nil && bind.Fields[name] != "" {
				return true
			}
		}
	}
	if inputs[param.Name] {
		return true
	}
	for _, value := range step.With {
		if referencesInputName(value, param.Name) {
			return true
		}
	}
	return false
}

func paramTargetNames(param *rollout.ParameterInfo) []string {
	name := strings.TrimSpace(param.Name)
	if name == "" {
		return nil
	}
	var out []string
	out = append(out, name)
	if param.In != "" {
		out = append(out, strings.TrimSpace(param.In)+"."+name)
	}
	if param.In == "query" {
		out = append(out, "query_pars."+name)
	}
	if param.In == "path" {
		out = append(out, "path_pars."+name)
	}
	return out
}

func referencesInputName(value, name string) bool {
	value = strings.TrimSpace(value)
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	return value == name || strings.Contains(value, "inputs."+name) || strings.Contains(value, "input."+name)
}

func validateIntentDataFlowSources(intent *rollout.Intent) error {
	if intent == nil {
		return nil
	}
	stepNames := intentStepNameSet(intent)
	inputs := intentInputNames(intent)
	var unresolved []string
	walkIntentSteps(intent.Steps, func(step *rollout.Step) {
		if step == nil {
			return
		}
		name := strings.TrimSpace(step.Name)
		if name == "" {
			name = "<unnamed>"
		}
		for _, dep := range step.DependsOn {
			dep = strings.TrimSpace(dep)
			if dep != "" && !stepNames[dep] {
				unresolved = append(unresolved, fmt.Sprintf("%s depends_on %q", name, dep))
			}
		}
		for target, source := range step.With {
			for _, ref := range unresolvedDataFlowReferences(source, stepNames, inputs) {
				unresolved = append(unresolved, fmt.Sprintf("%s.%s references %q", name, target, ref))
			}
		}
		for _, bind := range step.Binds {
			if bind == nil {
				continue
			}
			from := strings.TrimSpace(bind.From)
			if from != "" && !stepNames[from] {
				unresolved = append(unresolved, fmt.Sprintf("%s bind.from %q", name, from))
			}
			for target, source := range bind.Fields {
				for _, ref := range unresolvedDataFlowReferences(source, stepNames, inputs) {
					unresolved = append(unresolved, fmt.Sprintf("%s.%s references %q", name, target, ref))
				}
			}
		}
		for label, source := range map[string]string{
			"when":       step.When,
			"for_each":   step.ForEach,
			"items":      step.Items,
			"batch_size": step.BatchSize,
		} {
			for _, ref := range unresolvedDataFlowReferences(source, stepNames, inputs) {
				unresolved = append(unresolved, fmt.Sprintf("%s %s references %q", name, label, ref))
			}
		}
	})
	for _, output := range intent.Outputs {
		if output == nil {
			continue
		}
		for _, ref := range unresolvedDataFlowReferences(output.From, stepNames, inputs) {
			name := strings.TrimSpace(output.Name)
			if name == "" {
				name = "<unnamed>"
			}
			unresolved = append(unresolved, fmt.Sprintf("output %s references %q", name, ref))
		}
	}
	if len(unresolved) > 0 {
		return fmt.Errorf("%s. Use declared step names, inputs, or credential binding names only.", strings.Join(sortedCopy(unresolved), "; "))
	}
	return nil
}

func unresolvedDataFlowReferences(source string, stepNames, inputs map[string]bool) []string {
	var out []string
	for _, ref := range dataFlowReferencePrefixes(source) {
		lower := strings.ToLower(ref)
		if stepNames[ref] || inputs[ref] ||
			lower == "input" || lower == "inputs" || lower == "var" || lower == "vars" ||
			lower == "each" ||
			lower == "workflow" || lower == "trigger" || lower == "security" || lower == "credentials" ||
			lower == "body" || lower == "received_body" || lower == "request" || lower == "response" {
			continue
		}
		out = append(out, ref)
	}
	return sortedUnique(out)
}

func dataFlowReferencePrefixes(source string) []string {
	source = strings.TrimSpace(source)
	if source == "" {
		return nil
	}
	re := regexp.MustCompile(`\b([A-Za-z_][A-Za-z0-9_-]*)\s*\.`)
	matches := re.FindAllStringSubmatchIndex(source, -1)
	var out []string
	for _, match := range matches {
		if len(match) < 4 || dataFlowReferenceIsLiteralDomain(source, match[0]) {
			continue
		}
		out = append(out, source[match[2]:match[3]])
	}
	return sortedUnique(out)
}

func dataFlowReferenceIsLiteralDomain(source string, start int) bool {
	if start <= 0 || start > len(source) {
		return false
	}
	switch source[start-1] {
	case '@', '/', ':', '.':
		return true
	default:
		return false
	}
}

func intentStepNameSet(intent *rollout.Intent) map[string]bool {
	out := map[string]bool{}
	if intent == nil {
		return out
	}
	walkIntentSteps(intent.Steps, func(step *rollout.Step) {
		if step != nil && strings.TrimSpace(step.Name) != "" {
			out[strings.TrimSpace(step.Name)] = true
		}
	})
	return out
}

type responsePathValidation struct {
	Failures []string
	Warnings []string
}

func validateIntentResponsePaths(intent *rollout.Intent, candidates []openapidisco.Candidate, primary string) responsePathValidation {
	var result responsePathValidation
	if intent == nil {
		return result
	}
	ops := openAPIOperationIndex(candidates)
	stepOps := map[string]*rollout.OperationInfo{}
	walkIntentSteps(intent.Steps, func(step *rollout.Step) {
		if step == nil || strings.TrimSpace(step.Name) == "" || strings.TrimSpace(step.Operation) == "" {
			return
		}
		op := ops[operationKey(intentStepOpenAPIPath(intent, step, primary), step.Operation)]
		if op != nil {
			stepOps[strings.TrimSpace(step.Name)] = op
		}
	})
	walkIntentSteps(intent.Steps, func(step *rollout.Step) {
		if step == nil {
			return
		}
		name := strings.TrimSpace(step.Name)
		if name == "" {
			name = "<unnamed>"
		}
		for target, source := range step.With {
			result.addResponsePathChecks(fmt.Sprintf("%s.%s", name, target), source, stepOps)
		}
		for _, bind := range step.Binds {
			if bind == nil {
				continue
			}
			for target, source := range bind.Fields {
				checkSource := strings.TrimSpace(source)
				from := strings.TrimSpace(bind.From)
				if from != "" && (strings.HasPrefix(checkSource, "body") || strings.HasPrefix(checkSource, "received_body")) {
					checkSource = from + "." + checkSource
				}
				result.addResponsePathChecks(fmt.Sprintf("%s.%s", name, target), checkSource, stepOps)
			}
		}
	})
	for _, output := range intent.Outputs {
		if output == nil {
			continue
		}
		name := strings.TrimSpace(output.Name)
		if name == "" {
			name = "<unnamed>"
		}
		result.addResponsePathChecks("output "+name, output.From, stepOps)
	}
	return result
}

func (r *responsePathValidation) addResponsePathChecks(label, source string, stepOps map[string]*rollout.OperationInfo) {
	for _, ref := range responsePathReferences(source) {
		op := stepOps[ref.Step]
		if op == nil {
			continue
		}
		switch responsePathStatus(op, ref.Path) {
		case "missing":
			r.Failures = append(r.Failures, fmt.Sprintf("%s references missing response path %s.%s", label, ref.Step, ref.Path))
		case "opaque":
			r.Warnings = append(r.Warnings, fmt.Sprintf("%s references unverified response path %s.%s", label, ref.Step, ref.Path))
		}
	}
}

type responsePathReference struct {
	Step string
	Path string
}

func responsePathReferences(source string) []responsePathReference {
	source = strings.TrimSpace(source)
	if source == "" {
		return nil
	}
	matches := regexp.MustCompile(`\b([A-Za-z_][A-Za-z0-9_-]*)\.(?:received_body|body)([A-Za-z0-9_\.\[\]-]*)`).FindAllStringSubmatch(source, -1)
	var out []responsePathReference
	for _, match := range matches {
		if len(match) < 3 {
			continue
		}
		path := strings.TrimPrefix(match[2], ".")
		if path != "" {
			out = append(out, responsePathReference{Step: match[1], Path: path})
		}
	}
	return out
}

func responsePathStatus(op *rollout.OperationInfo, path string) string {
	schema := preferredResponseSchema(op)
	if len(schema) == 0 {
		return "opaque"
	}
	if schemaHasPath(schema, responsePathTokens(path)) {
		return "present"
	}
	return "missing"
}

func preferredResponseSchema(op *rollout.OperationInfo) map[string]any {
	if op == nil {
		return nil
	}
	for _, code := range []string{"200", "201", "202", "204", "default"} {
		if response := op.Responses[code]; response != nil && len(response.Schema) > 0 {
			return response.Schema
		}
	}
	codes := make([]string, 0, len(op.Responses))
	for code := range op.Responses {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	for _, code := range codes {
		if response := op.Responses[code]; response != nil && len(response.Schema) > 0 {
			return response.Schema
		}
	}
	return nil
}

func responsePathTokens(path string) []string {
	path = strings.Trim(strings.TrimSpace(path), ".")
	if path == "" {
		return nil
	}
	path = regexp.MustCompile(`\[[^\]]+\]`).ReplaceAllString(path, "")
	path = strings.Trim(path, ".")
	if path == "" {
		return nil
	}
	return strings.Split(path, ".")
}

func schemaHasPath(schema map[string]any, tokens []string) bool {
	if len(tokens) == 0 {
		return true
	}
	if len(schema) == 0 {
		return false
	}
	if strings.EqualFold(asString(schema["type"]), "array") {
		return schemaHasPath(asMap(schema["items"]), tokens)
	}
	props := asMap(schema["properties"])
	if len(props) == 0 {
		return false
	}
	next, ok := props[tokens[0]]
	if !ok {
		return false
	}
	return schemaHasPath(asMap(next), tokens[1:])
}

func workflowReferencesOpenAPI(source []byte) bool {
	return regexp.MustCompile(`(?im)\bopenapi\s*=`).Match(source)
}

func validateWorkflowAgainstExpectedPlan(report *QualityReport, compiled *runtimeplan.Plan, expected *WorkflowPlan) bool {
	if expected == nil {
		return true
	}
	ops := compiledOperationIndex(compiled)
	var missing, runtimeMismatch, operationMismatch, dependsMismatch, controlMismatch, actionMismatch, requestMismatch, bindingSourceMismatch, credentialMismatch []string
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
	if len(runtimeMismatch) > 0 || len(operationMismatch) > 0 || len(dependsMismatch) > 0 || len(controlMismatch) > 0 || len(actionMismatch) > 0 || len(requestMismatch) > 0 {
		var details []string
		details = append(details, sortedCopy(runtimeMismatch)...)
		details = append(details, sortedCopy(operationMismatch)...)
		details = append(details, sortedCopy(dependsMismatch)...)
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
	Request            *light.Body
	SuccessCriteria    []*uws1.Criterion
	OnFailure          []*uws1.FailureAction
	OnSuccess          []*uws1.SuccessAction
}

func compiledOperationIndex(plan *runtimeplan.Plan) map[string]*compiledOperation {
	out := map[string]*compiledOperation{}
	if plan == nil {
		return out
	}
	collectCompiledUWSSteps(plan.Document(), out)
	if plan.ExecCache() == nil {
		return out
	}
	for _, op := range plan.ExecCache().Operations {
		if op == nil || strings.TrimSpace(op.Name) == "" {
			continue
		}
		name := strings.TrimSpace(op.Name)
		compiled := out[name]
		if compiled == nil {
			compiled = &compiledOperation{}
			out[name] = compiled
		}
		compiled.ServiceType = op.ServiceType
		compiled.OpenAPIOperationID = op.OpenAPIOperationID
		compiled.DependsOn = append([]string(nil), op.DependsOn...)
		compiled.Request = op.Request
		compiled.SuccessCriteria = cloneCriteria(op.SuccessCriteria)
		compiled.OnFailure = cloneFailureActions(op.OnFailure)
		compiled.OnSuccess = cloneSuccessActions(op.OnSuccess)
	}
	return out
}

func collectCompiledUWSSteps(doc *uws1.Document, out map[string]*compiledOperation) {
	if doc == nil {
		return
	}
	for _, workflow := range doc.Workflows {
		if workflow == nil {
			continue
		}
		collectCompiledUWSStepList(workflow.Steps, out, "", "", "")
		for _, branch := range workflow.Cases {
			if branch != nil {
				collectCompiledUWSStepList(branch.Steps, out, workflow.WorkflowID, strings.TrimSpace(branch.Name), strings.TrimSpace(branch.When))
			}
		}
		collectCompiledUWSStepList(workflow.Default, out, workflow.WorkflowID, "default", "")
	}
}

func collectCompiledUWSStepList(steps []*uws1.Step, out map[string]*compiledOperation, parent, branch, branchWhen string) {
	for _, step := range steps {
		if step == nil {
			continue
		}
		name := strings.TrimSpace(step.StepID)
		if name != "" {
			out[name] = &compiledOperation{
				ServiceType:        strings.TrimSpace(step.Type),
				OpenAPIOperationID: strings.TrimSpace(step.OperationRef),
				DependsOn:          append([]string(nil), step.DependsOn...),
				Parent:             strings.TrimSpace(parent),
				Branch:             strings.TrimSpace(branch),
				BranchWhen:         strings.TrimSpace(branchWhen),
				When:               strings.TrimSpace(step.When),
				ForEach:            strings.TrimSpace(step.ForEach),
				Items:              strings.TrimSpace(step.Items),
				Mode:               strings.TrimSpace(step.Mode),
				BatchSize:          strings.TrimSpace(step.BatchSize),
			}
		}
		childParent := name
		if childParent == "" {
			childParent = parent
		}
		collectCompiledUWSStepList(step.Steps, out, childParent, "", "")
		for _, nestedBranch := range step.Cases {
			if nestedBranch != nil {
				collectCompiledUWSStepList(nestedBranch.Steps, out, childParent, strings.TrimSpace(nestedBranch.Name), strings.TrimSpace(nestedBranch.When))
			}
		}
		collectCompiledUWSStepList(step.Default, out, childParent, "default", "")
	}
}

type requestAttribute struct {
	Name       string
	Expression string
}

func requestAttributeEvidence(body *light.Body, names []string) (requestAttribute, bool) {
	if body == nil {
		return requestAttribute{}, false
	}
	for _, name := range names {
		if attr := body.Attributes[name]; attr != nil {
			return requestAttribute{Name: name, Expression: attributeExpression(attr)}, true
		}
		if block, child, ok := strings.Cut(name, "."); ok {
			if evidence, found := requestBlockAttributeEvidence(body, block, child); found {
				return evidence, true
			}
		}
		if evidence, found := requestNestedAttributeEvidence(body, name); found {
			return evidence, true
		}
	}
	return requestAttribute{}, false
}

func requestAttributeEvidenceMatching(body *light.Body, names []string, match func(requestAttribute) bool) (requestAttribute, bool) {
	if body == nil || match == nil {
		return requestAttribute{}, false
	}
	for _, name := range names {
		for _, evidence := range requestAttributeEvidences(body, name) {
			if match(evidence) {
				return evidence, true
			}
		}
	}
	return requestAttribute{}, false
}

func requestAttributeEvidences(body *light.Body, name string) []requestAttribute {
	if body == nil {
		return nil
	}
	var out []requestAttribute
	if attr := body.Attributes[name]; attr != nil {
		out = append(out, requestAttribute{Name: name, Expression: attributeExpression(attr)})
	}
	if block, child, ok := strings.Cut(name, "."); ok {
		out = append(out, requestBlockAttributeEvidences(body, block, child)...)
	}
	out = append(out, requestNestedAttributeEvidences(body, name)...)
	return out
}

func requestBlockAttributeEvidence(body *light.Body, blockType, name string) (requestAttribute, bool) {
	evidences := requestBlockAttributeEvidences(body, blockType, name)
	if len(evidences) == 0 {
		return requestAttribute{}, false
	}
	return evidences[0], true
}

func requestBlockAttributeEvidences(body *light.Body, blockType, name string) []requestAttribute {
	var out []requestAttribute
	if attr := body.Attributes[blockType]; attr != nil {
		expression := attributeExpression(attr)
		if requestMapExpressionContainsKey(expression, name) {
			out = append(out, requestAttribute{Name: blockType + "." + name, Expression: expression})
		}
	}
	for _, block := range body.Blocks {
		if block == nil || block.Bdy == nil {
			continue
		}
		if block.Type == blockType {
			if attr := block.Bdy.Attributes[name]; attr != nil {
				out = append(out, requestAttribute{Name: blockType + "." + name, Expression: attributeExpression(attr)})
			}
		}
		out = append(out, requestBlockAttributeEvidences(block.Bdy, blockType, name)...)
	}
	return out
}

func equivalentWorkflowRuntime(want, got string) bool {
	return want == "openapi" && got == "http"
}

func requestMapExpressionContainsKey(expression, key string) bool {
	expression = strings.TrimSpace(expression)
	key = strings.TrimSpace(key)
	if expression == "" || key == "" {
		return false
	}
	for _, pattern := range []string{
		key + " =",
		`"` + key + `"`,
		key + ":",
	} {
		if strings.Contains(expression, pattern) {
			return true
		}
	}
	return false
}

func requestNestedAttributeEvidence(body *light.Body, name string) (requestAttribute, bool) {
	evidences := requestNestedAttributeEvidences(body, name)
	if len(evidences) == 0 {
		return requestAttribute{}, false
	}
	return evidences[0], true
}

func requestNestedAttributeEvidences(body *light.Body, name string) []requestAttribute {
	var out []requestAttribute
	for _, block := range body.Blocks {
		if block == nil || block.Bdy == nil {
			continue
		}
		if attr := block.Bdy.Attributes[name]; attr != nil {
			out = append(out, requestAttribute{Name: block.Type + "." + name, Expression: attributeExpression(attr)})
		}
		out = append(out, requestNestedAttributeEvidences(block.Bdy, name)...)
	}
	return out
}

func attributeExpression(attr *light.Attribute) string {
	if attr == nil || attr.Expr == nil {
		return ""
	}
	if hcl, err := attr.Expr.HclExpression(); err == nil && strings.TrimSpace(hcl) != "" {
		return strings.TrimSpace(hcl)
	}
	if value := light.ExprToString(attr.Expr); value != "" {
		return value
	}
	return strings.TrimSpace(fmt.Sprint(attr.Expr))
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
	for _, prefix := range []string{"query.", "path.", "header.", "cookie.", "payload.", "query_pars.", "path_pars.", "header_pars.", "cookie_pars.", "payload_pars."} {
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

func assessUWS(report *QualityReport, path, schemaPath, exampleDir string, expectedPlan *WorkflowPlan) {
	if _, err := os.Stat(path); err != nil {
		report.add("uws.present", "fail", "workflow.uws.yaml is required", err.Error())
		return
	}
	report.add("uws.present", "pass", "workflow.uws.yaml is present", "")
	if strings.TrimSpace(schemaPath) == "" {
		schemaPath = defaultSchemaPath(exampleDir)
	}
	if err := uwsvalidate.ValidateFile(schemaPath, path); err != nil {
		report.add("uws.schema", "fail", "workflow.uws.yaml fails public UWS schema validation", err.Error())
		return
	}
	report.add("uws.schema", "pass", "workflow.uws.yaml validates against public UWS schema", "")
	doc, err := uwsprofile.LoadDocumentFile(path, uwsprofile.DocumentFormatYAML)
	if err != nil {
		report.add("uws.execution_profile", "fail", "workflow.uws.yaml could not be loaded by udon profile helpers", err.Error())
		return
	}
	if err := uwsprofile.ValidateForExecution(doc); err != nil {
		report.add("uws.execution_profile", "fail", "workflow.uws.yaml fails udon execution-profile validation", err.Error())
		return
	}
	report.add("uws.execution_profile", "pass", "workflow.uws.yaml passes udon execution-profile validation", "")
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

func assessSideEffectPolicy(report *QualityReport, policy projectPolicy, intent *rollout.Intent) {
	assessSideEffectProfile(report, sideEffectProfileFor(policy, intent))
}

func assessSideEffectProfile(report *QualityReport, profile sideEffectProfile) {
	if profile.ProductionEndpoint {
		if !profile.HasProductionPolicy {
			report.add("side_effects.environment", "fail", "production endpoint usage requires approved production handoff policy", "Add production handoff approval language to the Safety and Approval Boundary. Detected: "+strings.Join(profile.Reasons, "; "))
			return
		}
		report.add("side_effects.environment", "pass", "production endpoint usage has approved production handoff policy", "")
	}
	if !profile.SideEffectful {
		report.add("side_effects.policy", "pass", "no side-effectful workflow behavior inferred", "")
		return
	}
	if profile.HasApprovalPolicy && profile.HasSandboxPolicy {
		report.add("side_effects.policy", "pass", "side-effectful workflow has approval/trusted-runtime and sandbox proof-run policy", strings.Join(profile.Reasons, "; "))
		return
	}
	var missing []string
	if !profile.HasApprovalPolicy {
		missing = append(missing, "approval or trusted-runtime policy")
	}
	if !profile.HasSandboxPolicy {
		missing = append(missing, "sandbox/test proof-run policy")
	}
	report.add("side_effects.policy", "fail", "side-effectful workflow lacks required execution-boundary policy", "Add "+strings.Join(missing, " and ")+" to the Safety and Approval Boundary or Function Contracts section. Detected: "+strings.Join(profile.Reasons, "; "))
}

func assessSideEffectRetryPolicy(report *QualityReport, profile sideEffectProfile, policy projectPolicy, expectedPlan *WorkflowPlan) {
	if !profile.SideEffectful || !planHasRetryAction(expectedPlan) {
		report.add("side_effects.retry_policy", "pass", "retry action policy is not required", "")
		return
	}
	policyText := strings.ToLower(strings.Join([]string{
		policy.SafetySection,
		policy.FunctionSection,
		policy.RuntimeSection,
		policy.DataFlowSection,
	}, "\n"))
	if containsAny(policyText, []string{"idempotent", "idempotency", "safe to retry", "retry approved", "explicit retry", "bounded retry"}) {
		report.add("side_effects.retry_policy", "pass", "side-effectful retry actions have explicit retry/idempotency policy", "")
		return
	}
	report.add("side_effects.retry_policy", "fail", "side-effectful retry actions require explicit retry/idempotency policy", "Add retry or idempotency language to project.md before using onFailure retry actions on side-effectful workflows.")
}

func planHasRetryAction(plan *WorkflowPlan) bool {
	if plan == nil {
		return false
	}
	for _, step := range plan.Steps {
		for _, action := range step.OnFailure {
			if action != nil && strings.EqualFold(strings.TrimSpace(action.Type), "retry") {
				return true
			}
		}
	}
	return false
}

func assessReview(report *QualityReport, path string, profile sideEffectProfile, policy projectPolicy, expectedPlan *WorkflowPlan) {
	data, err := os.ReadFile(path)
	if err != nil {
		report.add("review.present", "fail", "review evidence is required", err.Error())
		return
	}
	text := string(data)
	if !strings.Contains(text, "Side-effectful execution was skipped") {
		report.add("review.execution_boundary", "fail", "review evidence must record skipped side-effectful execution", "")
		return
	}
	report.add("review.execution_boundary", "pass", "review evidence records skipped side-effectful execution", "")
	if !reviewContainsMinimumPackage(text) {
		report.add("review.package", "fail", "review evidence must list the minimum trusted-execution review package", "")
		return
	}
	report.add("review.package", "pass", "review evidence lists the minimum trusted-execution review package", "")
	if !strings.Contains(text, "Trusted proof run") || !strings.Contains(text, "./scripts/run-udon.sh") {
		report.add("review.trusted_runner", "fail", "review evidence must include trusted-runner handoff command", "")
		return
	}
	report.add("review.trusted_runner", "pass", "review evidence includes trusted-runner handoff command", "")
	if !strings.Contains(text, "Direct production execution: not performed by Ramen synthesis") {
		report.add("review.production_boundary", "fail", "review evidence must state that synthesis does not directly execute production workflows", "")
		return
	}
	report.add("review.production_boundary", "pass", "review evidence records the production execution boundary", "")
	if !strings.Contains(text, "Credential binding audit") {
		report.add("review.credential_audit", "fail", "review evidence must record credential binding audit requirements", "")
		return
	}
	report.add("review.credential_audit", "pass", "review evidence records credential binding audit requirements", "")
	if !reviewContainsApprovalStates(text, profile) {
		report.add("review.approval_states", "fail", "review evidence must state the Symphony approval states required before execution", "")
		return
	}
	report.add("review.approval_states", "pass", "review evidence records approval-state requirements", "")
	if !reviewContainsSandboxHandoff(text, profile) {
		report.add("review.sandbox_handoff", "fail", "review evidence must scope trusted-runner handoff to approved sandbox proof runs before production", "")
		return
	}
	report.add("review.sandbox_handoff", "pass", "review evidence scopes trusted-runner handoff to the approved execution state", "")
	if !reviewContainsCredentialBindings(text, policy, expectedPlan) {
		report.add("review.credential_bindings", "fail", "review evidence must list declared and expected credential bindings or explicitly state that none are required", "")
		return
	}
	report.add("review.credential_bindings", "pass", "review evidence records credential-binding inventory", "")
	if !strings.Contains(text, "## Side-Effect Summary") {
		report.add("review.side_effect_summary", "fail", "review evidence must summarize inferred side effects", "")
		return
	}
	if profile.SideEffectful && !strings.Contains(text, "Side-effectful workflow: yes") {
		report.add("review.side_effect_summary", "fail", "review evidence must mark side-effectful workflows explicitly", "")
		return
	}
	report.add("review.side_effect_summary", "pass", "review evidence summarizes inferred side effects", "")
	if !strings.Contains(text, "## Unresolved Risks") {
		report.add("review.unresolved_risks", "fail", "review evidence must record unresolved risks or lack of known risks", "")
		return
	}
	report.add("review.unresolved_risks", "pass", "review evidence records unresolved risks", "")
}

func reviewContainsMinimumPackage(text string) bool {
	if !strings.Contains(text, "## Minimum Review Package") {
		return false
	}
	required := []string{
		"Project brief",
		"Intent HCL",
		"Workflow HCL",
		"UWS artifact",
		"Expected plan",
		"Quality report",
		"Refinement report",
		"Review evidence",
		"Symphony handoff manifest",
	}
	for _, item := range required {
		if !strings.Contains(text, item) {
			return false
		}
	}
	return true
}

func assessSymphonyHandoff(report *QualityReport, path string, profile sideEffectProfile, policy projectPolicy, expectedPlan *WorkflowPlan) {
	data, err := os.ReadFile(path)
	if err != nil {
		report.add("symphony_handoff.present", "fail", "Symphony handoff manifest is required", err.Error())
		return
	}
	var manifest SymphonyHandoff
	if err := json.Unmarshal(data, &manifest); err != nil {
		report.add("symphony_handoff.present", "fail", "Symphony handoff manifest must be valid JSON", err.Error())
		return
	}
	report.add("symphony_handoff.present", "pass", "Symphony handoff manifest is readable", "")
	if manifest.Version != symphonyHandoffVersion || manifest.GeneratedState != "generated" {
		report.add("symphony_handoff.contract", "fail", "Symphony handoff manifest must declare the expected version and generated state", "")
		return
	}
	if !symphonyHandoffHasRequiredInputs(manifest) {
		report.add("symphony_handoff.contract", "fail", "Symphony handoff manifest must list every required handoff input", "")
		return
	}
	if !symphonyHandoffHasApprovalStates(manifest) {
		report.add("symphony_handoff.contract", "fail", "Symphony handoff manifest must list the required approval states", "")
		return
	}
	if !symphonyHandoffExecutionPolicyMatches(manifest, profile) {
		report.add("symphony_handoff.contract", "fail", "Symphony handoff execution policy must match inferred side-effect requirements", "")
		return
	}
	if !symphonyHandoffCredentialBindingsMatch(manifest, policy, expectedPlan) {
		report.add("symphony_handoff.contract", "fail", "Symphony handoff credential bindings must match declared and expected binding names", "")
		return
	}
	if manifest.CredentialBindings.ValuesAllowedInArtifacts || manifest.ExecutionPolicy.DirectProductionExecution {
		report.add("symphony_handoff.contract", "fail", "Symphony handoff must prohibit credential values and direct production execution", "")
		return
	}
	report.add("symphony_handoff.contract", "pass", "Symphony handoff manifest records package, state, execution, and credential contracts", "")
}

func symphonyHandoffHasRequiredInputs(manifest SymphonyHandoff) bool {
	required := map[string]bool{
		"project.md":                     false,
		"workflows/intent.hcl":           false,
		"workflows/workflow.hcl":         false,
		"workflows/workflow.uws.yaml":    false,
		"expected/plan.json":             false,
		"expected/quality.json":          false,
		"expected/refinement.json":       false,
		"expected/review.md":             false,
		"expected/symphony-handoff.json": false,
	}
	for _, input := range manifest.HandoffInputs {
		if !input.Required {
			continue
		}
		if _, ok := required[input.Path]; ok {
			required[input.Path] = true
		}
	}
	for _, found := range required {
		if !found {
			return false
		}
	}
	return true
}

func symphonyHandoffHasApprovalStates(manifest SymphonyHandoff) bool {
	states := map[string]bool{}
	for _, state := range manifest.ApprovalStates {
		states[state.Name] = true
	}
	for _, state := range []string{"generated", "validated", "review_required", "approved_for_sandbox", "approved_for_production", "rejected"} {
		if !states[state] {
			return false
		}
	}
	return true
}

func symphonyHandoffExecutionPolicyMatches(manifest SymphonyHandoff, profile sideEffectProfile) bool {
	policy := manifest.ExecutionPolicy
	if policy.SideEffectful != profile.SideEffectful {
		return false
	}
	if !profile.SideEffectful {
		return policy.RequiredNextState == "" && policy.SandboxProofRunState == "" && policy.ProductionExecutionState == ""
	}
	return policy.RequiredNextState == "review_required" &&
		policy.SandboxProofRunState == "approved_for_sandbox" &&
		policy.ProductionExecutionState == "approved_for_production" &&
		manifest.TrustedRunner.SandboxOnly
}

func symphonyHandoffCredentialBindingsMatch(manifest SymphonyHandoff, policy projectPolicy, expectedPlan *WorkflowPlan) bool {
	declared := credentialBindingNames(policy)
	expected := []string(nil)
	if expectedPlan != nil {
		seen := map[string]bool{}
		for _, step := range expectedPlan.Steps {
			for _, credential := range step.Credentials {
				credential = strings.TrimSpace(credential)
				if credential != "" {
					seen[credential] = true
				}
			}
			for _, param := range step.RequestParams {
				if !param.Credential {
					continue
				}
				for _, credential := range []string{param.ExpectedCredential, param.ExpectedSource} {
					credential = strings.TrimSpace(credential)
					if credential != "" && param.SourceKind == "credential" {
						seen[credential] = true
					}
				}
			}
		}
		for credential := range seen {
			expected = append(expected, credential)
		}
		sort.Strings(expected)
	}
	return stringSlicesEqual(declared, manifest.CredentialBindings.Declared) &&
		stringSlicesEqual(expected, manifest.CredentialBindings.ExpectedFromPlan)
}

func stringSlicesEqual(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func reviewContainsApprovalStates(text string, profile sideEffectProfile) bool {
	if !strings.Contains(text, "## Approval State Requirements") || !strings.Contains(text, "`generated`") {
		return false
	}
	for _, state := range []string{"`validated`", "`review_required`", "`approved_for_sandbox`", "`approved_for_production`", "`rejected`"} {
		if !strings.Contains(text, state) {
			return false
		}
	}
	if !profile.SideEffectful {
		return strings.Contains(text, "not required unless future changes add side effects")
	}
	return true
}

func reviewContainsSandboxHandoff(text string, profile sideEffectProfile) bool {
	if !strings.Contains(text, "## Trusted Execution Handoff") {
		return false
	}
	if !profile.SideEffectful {
		return strings.Contains(text, "Sandbox/test proof run is optional unless future changes add side effects")
	}
	return strings.Contains(text, "Trusted proof run command is for sandbox/test execution only after `approved_for_sandbox`") &&
		strings.Contains(text, "Production execution requires `approved_for_production`")
}

func reviewContainsCredentialBindings(text string, policy projectPolicy, expectedPlan *WorkflowPlan) bool {
	if !strings.Contains(text, "## Credential Binding Audit") {
		return false
	}
	expected := map[string]bool{}
	for _, credential := range credentialBindingNames(policy) {
		expected[credential] = true
	}
	if expectedPlan != nil {
		for _, step := range expectedPlan.Steps {
			for _, credential := range step.Credentials {
				credential = strings.TrimSpace(credential)
				if credential != "" {
					expected[credential] = true
				}
			}
			for _, param := range step.RequestParams {
				if !param.Credential {
					continue
				}
				for _, credential := range []string{param.ExpectedCredential, param.ExpectedSource} {
					credential = strings.TrimSpace(credential)
					if credential != "" && param.SourceKind == "credential" {
						expected[credential] = true
					}
				}
			}
		}
	}
	if len(expected) == 0 {
		return strings.Contains(text, "No credential bindings declared or required.")
	}
	for credential := range expected {
		if !strings.Contains(text, "`"+credential+"`") {
			return false
		}
	}
	return true
}
