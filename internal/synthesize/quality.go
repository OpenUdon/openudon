package synthesize

import (
	"context"
	"encoding/base64"
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
	assessUWS(report, result.UWSPath, opts.SchemaPath, exampleDir)
	assessSideEffectPolicy(report, policy, intent)
	assessReview(report, result.ReviewPath, sideEffectProfileFor(policy, intent))
	assessSecrets(report, result)

	if intentOK && planOK {
		report.add("quality.review", "pass", "workflow.hcl passed deterministic v1 quality gates", "")
	}
	report.finalize()
	if err := writeQualityFiles(result, report); err != nil {
		return nil, err
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
	addIntentDataFlowWarning(report, intent)
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

func workflowReferencesOpenAPI(source []byte) bool {
	return regexp.MustCompile(`(?im)\bopenapi\s*=`).Match(source)
}

func validateWorkflowAgainstExpectedPlan(report *QualityReport, compiled *runtimeplan.Plan, expected *WorkflowPlan) bool {
	if expected == nil {
		return true
	}
	ops := compiledOperationIndex(compiled)
	var missing, runtimeMismatch, operationMismatch, dependsMismatch, requestMismatch, bindingSourceMismatch, credentialMismatch []string
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
	if len(runtimeMismatch) > 0 || len(operationMismatch) > 0 || len(dependsMismatch) > 0 || len(requestMismatch) > 0 {
		var details []string
		details = append(details, sortedCopy(runtimeMismatch)...)
		details = append(details, sortedCopy(operationMismatch)...)
		details = append(details, sortedCopy(dependsMismatch)...)
		details = append(details, sortedCopy(requestMismatch)...)
		report.add("workflow.plan_match", "fail", "workflow.hcl diverges from the expected plan", strings.Join(details, "; "))
		return false
	}
	report.add("workflow.plan_match", "pass", "workflow.hcl preserves planned runtimes, operations, dependencies, and request mappings", "")
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

type compiledOperation struct {
	ServiceType        string
	OpenAPIOperationID string
	DependsOn          []string
	Request            *light.Body
}

func compiledOperationIndex(plan *runtimeplan.Plan) map[string]*compiledOperation {
	out := map[string]*compiledOperation{}
	if plan == nil || plan.ExecCache() == nil {
		return out
	}
	for _, op := range plan.ExecCache().Operations {
		if op == nil || strings.TrimSpace(op.Name) == "" {
			continue
		}
		out[strings.TrimSpace(op.Name)] = &compiledOperation{
			ServiceType:        op.ServiceType,
			OpenAPIOperationID: op.OpenAPIOperationID,
			DependsOn:          append([]string(nil), op.DependsOn...),
			Request:            op.Request,
		}
	}
	return out
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

func assessUWS(report *QualityReport, path, schemaPath, exampleDir string) {
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
}

func assessSideEffectPolicy(report *QualityReport, policy projectPolicy, intent *rollout.Intent) {
	profile := sideEffectProfileFor(policy, intent)
	if !profile.SideEffectful {
		report.add("side_effects.policy", "pass", "no side-effectful workflow behavior inferred", "")
		return
	}
	if profile.HasApprovalPolicy {
		report.add("side_effects.policy", "pass", "side-effectful workflow has approval or trusted-runtime policy", strings.Join(profile.Reasons, "; "))
		return
	}
	report.add("side_effects.policy", "fail", "side-effectful workflow lacks approval or trusted-runtime policy", "Add approval, trusted runner, or trusted runtime path guidance to the Safety and Approval Boundary or Function Contracts section. Detected: "+strings.Join(profile.Reasons, "; "))
}

func assessReview(report *QualityReport, path string, profile sideEffectProfile) {
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
	if !strings.Contains(text, "Trusted proof run") || !strings.Contains(text, "./scripts/run-udon.sh") {
		report.add("review.trusted_runner", "fail", "review evidence must include trusted-runner handoff command", "")
		return
	}
	report.add("review.trusted_runner", "pass", "review evidence includes trusted-runner handoff command", "")
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

func assessSecrets(report *QualityReport, result Result) {
	paths := []string{result.ProjectPath, result.IntentPath, result.WorkflowPath, result.UWSPath, result.PlanJSONPath, result.PlanMDPath, result.DiscoveryJSONPath, result.RefinementJSONPath, result.RefinementMDPath, result.ReviewPath}
	var hits []string
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		if containsSecretLikeToken(data) {
			hits = append(hits, relOrAbs(result.ExampleDir, path))
		}
	}
	if len(hits) > 0 {
		report.add("artifacts.no_secrets", "fail", "artifacts contain secret-like tokens", strings.Join(hits, ", "))
		return
	}
	report.add("artifacts.no_secrets", "pass", "no obvious secret-like tokens found in artifacts", "")
}

func writeQualityFiles(result Result, report *QualityReport) error {
	if err := os.MkdirAll(filepath.Dir(result.QualityJSONPath), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(result.QualityJSONPath, append(data, '\n'), 0o644); err != nil {
		return err
	}
	return os.WriteFile(result.QualityMDPath, []byte(qualityMarkdown(report)), 0o644)
}

func qualityMarkdown(report *QualityReport) string {
	var b strings.Builder
	b.WriteString("# Ramen Quality Report\n\n")
	fmt.Fprintf(&b, "Status: `%s`\n\n", report.Status)
	for _, check := range report.Checks {
		fmt.Fprintf(&b, "- `%s` %s - %s\n", check.Code, check.Status, check.Message)
		if check.Detail != "" {
			fmt.Fprintf(&b, "  Detail: %s\n", check.Detail)
		}
	}
	return b.String()
}

func missingIntentSteps(intent *rollout.Intent, workflows []*uws1.Workflow) []string {
	stepIDs := map[string]bool{}
	for _, workflow := range workflows {
		if workflow != nil {
			collectUWSStepIDs(workflow.Steps, stepIDs)
		}
	}
	var missing []string
	walkIntentSteps(intent.Steps, func(step *rollout.Step) {
		if step == nil {
			return
		}
		name := strings.TrimSpace(step.Name)
		if name != "" && !stepIDs[name] {
			missing = append(missing, name)
		}
	})
	sort.Strings(missing)
	return missing
}

func collectUWSStepIDs(steps []*uws1.Step, out map[string]bool) {
	for _, step := range steps {
		if step == nil {
			continue
		}
		if strings.TrimSpace(step.StepID) != "" {
			out[strings.TrimSpace(step.StepID)] = true
		}
		collectUWSStepIDs(step.Steps, out)
		for _, branch := range step.Cases {
			if branch != nil {
				collectUWSStepIDs(branch.Steps, out)
			}
		}
		collectUWSStepIDs(step.Default, out)
	}
}

func candidateList(candidates []openapidisco.Candidate) string {
	var items []string
	for _, candidate := range candidates {
		items = append(items, candidate.RelativePath)
	}
	sort.Strings(items)
	return strings.Join(items, ", ")
}

func walkIntentSteps(steps []*rollout.Step, fn func(*rollout.Step)) {
	for _, step := range steps {
		if step == nil {
			continue
		}
		fn(step)
		walkIntentSteps(step.Steps, fn)
		for _, branch := range step.Cases {
			if branch != nil {
				walkIntentSteps(branch.Steps, fn)
			}
		}
		if step.Default != nil {
			walkIntentSteps(step.Default.Steps, fn)
		}
	}
}

func (r *QualityReport) add(code, status, message, detail string) {
	r.Checks = append(r.Checks, QualityCheck{
		Code:    code,
		Status:  status,
		Message: message,
		Detail:  detail,
	})
}

func (r *QualityReport) finalize() {
	for _, check := range r.Checks {
		if check.Status == "fail" {
			r.Status = "fail"
			return
		}
	}
	r.Status = "pass"
}

const (
	minAssignedSecretLength = 12
)

var providerSecretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`AIza[0-9A-Za-z_-]{20,}`),
	regexp.MustCompile(`sk-ant-api[0-9A-Za-z_-]*-[0-9A-Za-z_-]{20,}`),
	regexp.MustCompile(`sk-(?:proj-)?[0-9A-Za-z_-]{20,}`),
	regexp.MustCompile(`ghp_[0-9A-Za-z]{36,}`),
	regexp.MustCompile(`github_pat_[0-9A-Za-z_]{20,}`),
	regexp.MustCompile(`(?:AKIA|ASIA)[0-9A-Z]{16}`),
}

var (
	jwtCandidatePattern         = regexp.MustCompile(`[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+`)
	sensitiveAssignmentPattern  = regexp.MustCompile(`(?i)\b([A-Za-z0-9_.-]*(?:api[_-]?key|apikey|app[_-]?id|appid|token|secret|password|authorization)[A-Za-z0-9_.-]*)\s*[:=]\s*["']([^"'\r\n]+)["']`)
	workflowReferencePattern    = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]*(?:\[[0-9]+\])?(?:\.[A-Za-z_][A-Za-z0-9_-]*(?:\[[0-9]+\])?)*$`)
	tokenShapedValuePattern     = regexp.MustCompile(`^[A-Za-z0-9_+/=-]+$`)
	tokenSourceAssignmentSuffix = regexp.MustCompile(`(?i)(?:^|[_\-.])from$`)
)

func containsSecretLikeToken(data []byte) bool {
	for _, pattern := range providerSecretPatterns {
		if pattern.Match(data) {
			return true
		}
	}
	if containsValidatedJWT(data) {
		return true
	}
	for _, match := range sensitiveAssignmentPattern.FindAllSubmatch(data, -1) {
		if len(match) < 3 {
			continue
		}
		if isSensitiveSourceAssignment(string(match[1])) {
			continue
		}
		if isAssignedSecretLiteral(string(match[2])) {
			return true
		}
	}
	return false
}

func containsValidatedJWT(data []byte) bool {
	for _, candidate := range jwtCandidatePattern.FindAll(data, -1) {
		if isValidatedJWT(string(candidate)) {
			return true
		}
	}
	return false
}

func isValidatedJWT(candidate string) bool {
	parts := strings.Split(candidate, ".")
	if len(parts) != 3 {
		return false
	}
	header := map[string]any{}
	if !decodeBase64URLJSON(parts[0], &header) {
		return false
	}
	if _, ok := header["alg"]; !ok {
		if _, ok := header["typ"]; !ok {
			return false
		}
	}
	payload := map[string]any{}
	return decodeBase64URLJSON(parts[1], &payload)
}

func decodeBase64URLJSON(segment string, out any) bool {
	decoded, err := base64.RawURLEncoding.DecodeString(segment)
	if err != nil {
		decoded, err = base64.URLEncoding.DecodeString(segment)
		if err != nil {
			return false
		}
	}
	if err := json.Unmarshal(decoded, out); err != nil {
		return false
	}
	if object, ok := out.(*map[string]any); ok {
		return len(*object) > 0
	}
	return true
}

func isSensitiveSourceAssignment(key string) bool {
	normalized := strings.ToLower(strings.TrimSpace(key))
	return normalized == "token_from" || tokenSourceAssignmentSuffix.MatchString(normalized)
}

func isAssignedSecretLiteral(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || isWorkflowReferenceOrBindingName(value) {
		return false
	}
	for _, pattern := range providerSecretPatterns {
		if pattern.MatchString(value) {
			return true
		}
	}
	if isValidatedJWT(value) {
		return true
	}
	if len(value) < minAssignedSecretLength {
		return false
	}
	return looksTokenShaped(value)
}

func isWorkflowReferenceOrBindingName(value string) bool {
	value = strings.TrimSpace(value)
	if !strings.ContainsAny(value, "._[]-") {
		return false
	}
	return workflowReferencePattern.MatchString(value)
}

func looksTokenShaped(value string) bool {
	if len(value) < 16 || !tokenShapedValuePattern.MatchString(value) {
		return false
	}
	var hasLetter, hasDigit bool
	for _, r := range value {
		switch {
		case r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z':
			hasLetter = true
		case r >= '0' && r <= '9':
			hasDigit = true
		}
	}
	return hasLetter && hasDigit
}
