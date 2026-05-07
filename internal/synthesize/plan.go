package synthesize

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/genelet/ramen/internal/openapidisco"
	"github.com/genelet/udon/pkg/rollout"
	"github.com/OpenUdon/uws/uws1"
)

const workflowPlanVersion = "ramen.workflow-plan.v1"

type WorkflowPlan struct {
	Version     string            `json:"version"`
	Example     string            `json:"example,omitempty"`
	Workflow    string            `json:"workflow,omitempty"`
	Summary     string            `json:"summary,omitempty"`
	Timeout     *float64          `json:"timeout,omitempty"`
	Idempotency *uws1.Idempotency `json:"idempotency,omitempty"`
	Steps       []PlanStep        `json:"steps"`
	Results     []PlanResult      `json:"results,omitempty"`
	Gaps        []PlanGap         `json:"gaps,omitempty"`
}

type PlanStep struct {
	Name            string                `json:"name"`
	Type            string                `json:"type,omitempty"`
	Parent          string                `json:"parent,omitempty"`
	Branch          string                `json:"branch,omitempty"`
	BranchWhen      string                `json:"branch_when,omitempty"`
	Inferred        bool                  `json:"inferred,omitempty"`
	OpenAPI         string                `json:"openapi,omitempty"`
	Operation       string                `json:"operation,omitempty"`
	Timeout         *float64              `json:"timeout,omitempty"`
	Runtime         string                `json:"runtime,omitempty"`
	When            string                `json:"when,omitempty"`
	ForEach         string                `json:"for_each,omitempty"`
	Items           string                `json:"items,omitempty"`
	Mode            string                `json:"mode,omitempty"`
	BatchSize       string                `json:"batch_size,omitempty"`
	DependsOn       []string              `json:"depends_on,omitempty"`
	RequiredParams  []string              `json:"required_params,omitempty"`
	RequestParams   []PlanParam           `json:"request_params,omitempty"`
	Bindings        []PlanBinding         `json:"bindings,omitempty"`
	Credentials     []string              `json:"credentials,omitempty"`
	SuccessCriteria []*uws1.Criterion     `json:"successCriteria,omitempty"`
	OnFailure       []*uws1.FailureAction `json:"onFailure,omitempty"`
	OnSuccess       []*uws1.SuccessAction `json:"onSuccess,omitempty"`
}

type PlanParam struct {
	Name               string `json:"name"`
	In                 string `json:"in,omitempty"`
	Required           bool   `json:"required,omitempty"`
	Credential         bool   `json:"credential,omitempty"`
	SourceKind         string `json:"source_kind,omitempty"`
	ExpectedSource     string `json:"expected_source,omitempty"`
	ExpectedCredential string `json:"expected_credential,omitempty"`
}

type PlanBinding struct {
	From   string `json:"from,omitempty"`
	Target string `json:"target"`
	Source string `json:"source"`
}

type PlanResult struct {
	Name  string `json:"name"`
	Kind  string `json:"kind"`
	From  string `json:"from"`
	Value string `json:"value,omitempty"`
}

type PlanGap struct {
	Code   string `json:"code"`
	Step   string `json:"step,omitempty"`
	Detail string `json:"detail"`
	Query  string `json:"query,omitempty"`
}

func buildWorkflowPlan(result Result, intent *rollout.Intent, candidates []openapidisco.Candidate, policy projectPolicy) *WorkflowPlan {
	plan := &WorkflowPlan{
		Version: workflowPlanVersion,
		Example: relOrAbs(filepath.Dir(result.ExampleDir), result.ExampleDir),
	}
	if intent != nil && intent.Workflow != nil {
		plan.Workflow = strings.TrimSpace(intent.Workflow.Name)
		plan.Summary = strings.TrimSpace(intent.Workflow.Description)
		plan.Timeout = cloneFloat64Ptr(intent.Workflow.Timeout)
		plan.Idempotency = cloneIdempotency(intent.Workflow.Idempotency)
	}
	ops := openAPIOperationIndex(candidates)
	security := openAPISecurityIndex(candidates)
	inputs := intentInputNames(intent)
	addStepsToWorkflowPlan(plan, intent, intentSteps(intent), ops, security, inputs, policy, planStepContext{})
	plan.Results = structuralPlanResults(intent)
	sortPlanGaps(plan.Gaps)
	return plan
}

type planStepContext struct {
	Parent     string
	Branch     string
	BranchWhen string
}

func addStepsToWorkflowPlan(plan *WorkflowPlan, intent *rollout.Intent, steps []*rollout.Step, ops map[string]*rollout.OperationInfo, security map[string][]openAPISecurityRequirement, inputs map[string]bool, policy projectPolicy, ctx planStepContext) {
	for _, step := range steps {
		if step == nil {
			continue
		}
		name := strings.TrimSpace(step.Name)
		planStep := PlanStep{
			Name:            name,
			Type:            strings.TrimSpace(step.Type),
			Parent:          ctx.Parent,
			Branch:          ctx.Branch,
			BranchWhen:      ctx.BranchWhen,
			Runtime:         strings.TrimSpace(step.Type),
			Operation:       strings.TrimSpace(step.Operation),
			Timeout:         cloneFloat64Ptr(step.Timeout),
			When:            strings.TrimSpace(step.When),
			ForEach:         strings.TrimSpace(step.ForEach),
			Items:           strings.TrimSpace(step.Items),
			Mode:            strings.TrimSpace(step.Mode),
			BatchSize:       strings.TrimSpace(step.BatchSize),
			DependsOn:       sortedCopy(step.DependsOn),
			Inferred:        true,
			SuccessCriteria: cloneCriteria(step.SuccessCriteria),
			OnFailure:       cloneFailureActions(step.OnFailure),
			OnSuccess:       cloneSuccessActions(step.OnSuccess),
		}
		planStep.OpenAPI = strings.TrimSpace(step.OpenAPI)
		if planStep.OpenAPI == "" && intent != nil {
			planStep.OpenAPI = strings.TrimSpace(intent.OpenAPI)
		}
		for target, source := range step.With {
			if strings.TrimSpace(target) == "" {
				continue
			}
			if strings.EqualFold(strings.TrimSpace(step.Type), "cmd") && strings.EqualFold(strings.TrimSpace(target), "command") {
				continue
			}
			planStep.Bindings = append(planStep.Bindings, PlanBinding{
				Target: strings.TrimSpace(target),
				Source: strings.TrimSpace(source),
			})
		}
		for _, bind := range step.Binds {
			if bind == nil {
				continue
			}
			for target, source := range bind.Fields {
				if strings.TrimSpace(target) == "" {
					continue
				}
				planStep.Bindings = append(planStep.Bindings, PlanBinding{
					From:   strings.TrimSpace(bind.From),
					Target: strings.TrimSpace(target),
					Source: strings.TrimSpace(source),
				})
			}
		}
		sortPlanBindings(planStep.Bindings)
		if planStep.Operation != "" {
			key := operationKey(planStep.OpenAPI, planStep.Operation)
			op := ops[key]
			if op == nil {
				plan.Gaps = append(plan.Gaps, PlanGap{
					Code:   "openapi.missing_operation",
					Step:   planStep.Name,
					Detail: fmt.Sprintf("operation %q is not available in %q", planStep.Operation, planStep.OpenAPI),
					Query:  planStep.Operation,
				})
			} else {
				for _, param := range op.Parameters {
					if param == nil || !param.Required {
						continue
					}
					planStep.RequiredParams = append(planStep.RequiredParams, param.Name)
					planParam := PlanParam{
						Name:       strings.TrimSpace(param.Name),
						In:         strings.TrimSpace(param.In),
						Required:   true,
						Credential: credentialLikeParam(param.Name),
					}
					planParam.SourceKind, planParam.ExpectedSource = expectedSourceForParam(step, param, inputs)
					if credentialLikeParam(param.Name) {
						planStep.Credentials = append(planStep.Credentials, param.Name)
						planParam.ExpectedCredential = expectedCredentialForParam(step, param, policy)
						if planParam.ExpectedCredential != "" {
							planParam.SourceKind = "credential"
						}
						if planParam.ExpectedSource == "" && planParam.ExpectedCredential == "" {
							plan.Gaps = append(plan.Gaps, PlanGap{
								Code:   "credentials.missing_binding",
								Step:   planStep.Name,
								Detail: fmt.Sprintf("operation %q requires credential-like parameter %q but no request mapping or credential binding is auditable", planStep.Operation, param.Name),
								Query:  param.Name,
							})
						}
						planStep.RequestParams = append(planStep.RequestParams, planParam)
						continue
					}
					if !stepSatisfiesParam(step, param, inputs) {
						plan.Gaps = append(plan.Gaps, PlanGap{
							Code:   "data_flow.required_params",
							Step:   planStep.Name,
							Detail: fmt.Sprintf("operation %q requires %s parameter %q", planStep.Operation, param.In, param.Name),
							Query:  param.Name,
						})
					}
					planStep.RequestParams = append(planStep.RequestParams, planParam)
				}
			}
			for _, req := range security[key] {
				planParam := PlanParam{
					Name:               securityRequestFieldName(req),
					In:                 strings.TrimSpace(req.In),
					Required:           true,
					Credential:         true,
					SourceKind:         "credential",
					ExpectedCredential: expectedCredentialForSecurity(step, req, policy),
				}
				if planParam.Name == "" {
					planParam.Name = req.label()
				}
				planStep.Credentials = append(planStep.Credentials, planParam.Name)
				planStep.RequestParams = append(planStep.RequestParams, planParam)
				if planParam.ExpectedCredential == "" {
					plan.Gaps = append(plan.Gaps, PlanGap{
						Code:   "credentials.missing_binding",
						Step:   planStep.Name,
						Detail: fmt.Sprintf("operation %q requires OpenAPI security %q but no credential binding is auditable", planStep.Operation, req.label()),
						Query:  req.label(),
					})
				}
			}
		}
		if len(planStep.Credentials) > 0 && strings.TrimSpace(policy.CredentialSection) == "" {
			plan.Gaps = append(plan.Gaps, PlanGap{
				Code:   "credentials.missing_policy",
				Step:   planStep.Name,
				Detail: fmt.Sprintf("credential-like parameter(s) require a Credentials and Secrets section: %s", strings.Join(sortedCopy(planStep.Credentials), ", ")),
			})
		}
		planStep.RequiredParams = sortedUnique(planStep.RequiredParams)
		planStep.Credentials = sortedUnique(planStep.Credentials)
		sortPlanParams(planStep.RequestParams)
		plan.Steps = append(plan.Steps, planStep)
		addStepsToWorkflowPlan(plan, intent, step.Steps, ops, security, inputs, policy, planStepContext{Parent: name})
		for _, branch := range step.Cases {
			if branch == nil {
				continue
			}
			addStepsToWorkflowPlan(plan, intent, branch.Steps, ops, security, inputs, policy, planStepContext{
				Parent:     name,
				Branch:     strings.TrimSpace(branch.Name),
				BranchWhen: strings.TrimSpace(branch.When),
			})
		}
		if step.Default != nil {
			addStepsToWorkflowPlan(plan, intent, step.Default.Steps, ops, security, inputs, policy, planStepContext{
				Parent: name,
				Branch: "default",
			})
		}
	}
}

func intentSteps(intent *rollout.Intent) []*rollout.Step {
	if intent == nil {
		return nil
	}
	return intent.Steps
}

func structuralPlanResults(intent *rollout.Intent) []PlanResult {
	if intent == nil || len(intent.Outputs) == 0 {
		return nil
	}
	steps := structuralIntentStepIndex(intent)
	var out []PlanResult
	for _, output := range intent.Outputs {
		if output == nil {
			continue
		}
		name := strings.TrimSpace(output.Name)
		from := strings.TrimSpace(output.From)
		source := structuralResultSourceName(from)
		if name == "" || source == "" {
			continue
		}
		kind := steps[source]
		if !uws1.IsStructuralResultKind(kind) {
			continue
		}
		out = append(out, PlanResult{
			Name:  name,
			Kind:  kind,
			From:  "main." + source,
			Value: from,
		})
	}
	sortPlanResults(out)
	return out
}

func structuralIntentStepIndex(intent *rollout.Intent) map[string]string {
	out := map[string]string{}
	if intent == nil {
		return out
	}
	walkIntentSteps(intent.Steps, func(step *rollout.Step) {
		if step == nil {
			return
		}
		name := strings.TrimSpace(step.Name)
		kind := strings.ToLower(strings.TrimSpace(step.Type))
		if name != "" && uws1.IsStructuralResultKind(kind) {
			out[name] = kind
		}
	})
	return out
}

func structuralResultSourceName(from string) string {
	from = strings.TrimSpace(from)
	if from == "" {
		return ""
	}
	beforeDot, _, hasDot := strings.Cut(from, ".")
	if hasDot {
		return strings.TrimSpace(beforeDot)
	}
	return from
}

func writeWorkflowPlan(result Result, plan *WorkflowPlan) error {
	if plan == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(result.PlanJSONPath), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(result.PlanJSONPath, append(data, '\n'), 0o644); err != nil {
		return err
	}
	return os.WriteFile(result.PlanMDPath, []byte(workflowPlanMarkdown(plan)), 0o644)
}

func loadWorkflowPlan(path string) (*WorkflowPlan, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var plan WorkflowPlan
	if err := json.Unmarshal(data, &plan); err != nil {
		return nil, err
	}
	return &plan, nil
}

func workflowPlanMarkdown(plan *WorkflowPlan) string {
	var b strings.Builder
	b.WriteString("# Ramen Workflow Plan\n\n")
	if plan.Workflow != "" {
		fmt.Fprintf(&b, "- Workflow: `%s`\n", plan.Workflow)
	}
	if plan.Summary != "" {
		fmt.Fprintf(&b, "- Summary: %s\n", plan.Summary)
	}
	if plan.Timeout != nil {
		fmt.Fprintf(&b, "- Timeout: `%g` seconds\n", *plan.Timeout)
	}
	if plan.Idempotency != nil {
		fmt.Fprintf(&b, "- Idempotency: key `%s`", plan.Idempotency.Key)
		if plan.Idempotency.OnConflict != "" {
			fmt.Fprintf(&b, " onConflict `%s`", plan.Idempotency.OnConflict)
		}
		if plan.Idempotency.TTL != nil {
			fmt.Fprintf(&b, " ttl `%g` seconds", *plan.Idempotency.TTL)
		}
		b.WriteString("\n")
	}
	fmt.Fprintf(&b, "- Version: `%s`\n\n", plan.Version)
	b.WriteString("## Steps\n\n")
	if len(plan.Steps) == 0 {
		b.WriteString("- No steps planned.\n")
	} else {
		for _, step := range plan.Steps {
			fmt.Fprintf(&b, "- `%s`", step.Name)
			if step.Runtime != "" {
				fmt.Fprintf(&b, " runtime `%s`", step.Runtime)
			}
			if step.Operation != "" {
				fmt.Fprintf(&b, " operation `%s`", step.Operation)
			}
			if step.Timeout != nil {
				fmt.Fprintf(&b, " timeout `%g`s", *step.Timeout)
			}
			b.WriteString("\n")
			if step.Parent != "" {
				fmt.Fprintf(&b, "  - parent: `%s`\n", step.Parent)
			}
			if step.Branch != "" {
				fmt.Fprintf(&b, "  - branch: `%s`", step.Branch)
				if step.BranchWhen != "" {
					fmt.Fprintf(&b, " when `%s`", step.BranchWhen)
				}
				b.WriteString("\n")
			}
			if step.When != "" {
				fmt.Fprintf(&b, "  - when: `%s`\n", step.When)
			}
			if step.ForEach != "" {
				fmt.Fprintf(&b, "  - for_each: `%s`\n", step.ForEach)
			}
			if step.Items != "" {
				fmt.Fprintf(&b, "  - items: `%s`\n", step.Items)
			}
			if step.Mode != "" {
				fmt.Fprintf(&b, "  - mode: `%s`\n", step.Mode)
			}
			if step.BatchSize != "" {
				fmt.Fprintf(&b, "  - batch_size: `%s`\n", step.BatchSize)
			}
			if len(step.DependsOn) > 0 {
				fmt.Fprintf(&b, "  - depends_on: `%s`\n", strings.Join(step.DependsOn, "`, `"))
			}
			if len(step.RequiredParams) > 0 {
				fmt.Fprintf(&b, "  - required_params: `%s`\n", strings.Join(step.RequiredParams, "`, `"))
			}
			for _, param := range step.RequestParams {
				fmt.Fprintf(&b, "  - request_param: `%s`", param.Name)
				if param.In != "" {
					fmt.Fprintf(&b, " in `%s`", param.In)
				}
				if param.Credential {
					b.WriteString(" credential")
				}
				if param.SourceKind != "" {
					fmt.Fprintf(&b, " source_kind `%s`", param.SourceKind)
				}
				if param.ExpectedSource != "" {
					fmt.Fprintf(&b, " source `%s`", param.ExpectedSource)
				}
				if param.ExpectedCredential != "" {
					fmt.Fprintf(&b, " binding `%s`", param.ExpectedCredential)
				}
				b.WriteString("\n")
			}
			for _, binding := range step.Bindings {
				fmt.Fprintf(&b, "  - binding: `%s <- %s`\n", binding.Target, binding.Source)
			}
			if len(step.Credentials) > 0 {
				fmt.Fprintf(&b, "  - credentials: `%s`\n", strings.Join(step.Credentials, "`, `"))
			}
			writePlanActionMarkdown(&b, step)
		}
	}
	if len(plan.Results) > 0 {
		b.WriteString("\n## Structural Results\n\n")
		for _, result := range plan.Results {
			fmt.Fprintf(&b, "- `%s` kind `%s` from `%s`", result.Name, result.Kind, result.From)
			if result.Value != "" {
				fmt.Fprintf(&b, " value `%s`", result.Value)
			}
			b.WriteString("\n")
		}
	}
	if len(plan.Gaps) > 0 {
		b.WriteString("\n## Gaps\n\n")
		for _, gap := range plan.Gaps {
			fmt.Fprintf(&b, "- `%s`", gap.Code)
			if gap.Step != "" {
				fmt.Fprintf(&b, " step `%s`", gap.Step)
			}
			fmt.Fprintf(&b, ": %s\n", gap.Detail)
		}
	}
	return b.String()
}

func writePlanActionMarkdown(b *strings.Builder, step PlanStep) {
	if b == nil {
		return
	}
	for _, criterion := range step.SuccessCriteria {
		if criterion == nil {
			continue
		}
		fmt.Fprintf(b, "  - successCriteria: `%s`", criterion.Condition)
		if criterion.Type != "" {
			fmt.Fprintf(b, " type `%s`", criterion.Type)
		}
		if criterion.Context != "" {
			fmt.Fprintf(b, " context `%s`", criterion.Context)
		}
		b.WriteString("\n")
	}
	for _, action := range step.OnFailure {
		if action == nil {
			continue
		}
		fmt.Fprintf(b, "  - onFailure: `%s` type `%s`", action.Name, action.Type)
		if action.WorkflowID != "" {
			fmt.Fprintf(b, " workflowId `%s`", action.WorkflowID)
		}
		if action.StepID != "" {
			fmt.Fprintf(b, " stepId `%s`", action.StepID)
		}
		if action.RetryLimit > 0 {
			fmt.Fprintf(b, " retryLimit `%d`", action.RetryLimit)
		}
		if action.RetryAfter > 0 {
			fmt.Fprintf(b, " retryAfter `%g`", action.RetryAfter)
		}
		b.WriteString("\n")
	}
	for _, action := range step.OnSuccess {
		if action == nil {
			continue
		}
		fmt.Fprintf(b, "  - onSuccess: `%s` type `%s`", action.Name, action.Type)
		if action.WorkflowID != "" {
			fmt.Fprintf(b, " workflowId `%s`", action.WorkflowID)
		}
		if action.StepID != "" {
			fmt.Fprintf(b, " stepId `%s`", action.StepID)
		}
		b.WriteString("\n")
	}
}

func discoverComplementaryOpenAPI(ctx context.Context, discoverer *openapidisco.Discoverer, exampleDir, projectText string, candidates []openapidisco.Candidate, intent *rollout.Intent, policy projectPolicy) ([]openapidisco.Candidate, []openapidisco.DiscoveryAttempt, bool) {
	if discoverer == nil {
		return candidates, nil, false
	}
	plan := buildWorkflowPlan(resultPaths(exampleDir), intent, candidates, policy)
	query := complementaryDiscoveryQuery(projectText, plan)
	if query == "" {
		return candidates, nil, false
	}
	imported, err := discoverer.ImportBestAPIsGuruMatch(ctx, filepath.Join(exampleDir, "openapi"), exampleDir, query)
	if err != nil || imported.Path == "" {
		detail := "no complementary APIs.guru match"
		if err != nil {
			detail = err.Error()
		}
		return candidates, []openapidisco.DiscoveryAttempt{{
			Kind:   "apis.guru.complementary",
			Status: "fail",
			Detail: detail,
		}}, false
	}
	all := append(append([]openapidisco.Candidate(nil), candidates...), imported)
	all = dedupeCandidates(all)
	return all, []openapidisco.DiscoveryAttempt{{
		Kind:   "apis.guru.complementary",
		Source: imported.Source,
		Status: "pass",
		Detail: imported.RelativePath,
	}}, len(all) > len(candidates)
}

func complementaryDiscoveryQuery(projectText string, plan *WorkflowPlan) string {
	if plan == nil {
		return ""
	}
	var parts []string
	for _, gap := range plan.Gaps {
		switch gap.Code {
		case "openapi.missing_operation", "data_flow.required_params":
			parts = append(parts, gap.Detail, gap.Query)
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return projectText + "\n\nCapability gaps:\n" + strings.Join(parts, "\n")
}

func dedupeCandidates(candidates []openapidisco.Candidate) []openapidisco.Candidate {
	seen := map[string]bool{}
	var out []openapidisco.Candidate
	for _, candidate := range candidates {
		key := strings.TrimSpace(candidate.RelativePath)
		if key == "" {
			key = strings.TrimSpace(candidate.Path)
		}
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, candidate)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		return out[i].RelativePath < out[j].RelativePath
	})
	return out
}

func sortedCopy(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			out = append(out, strings.TrimSpace(value))
		}
	}
	sort.Strings(out)
	return out
}

func sortedUnique(values []string) []string {
	values = sortedCopy(values)
	if len(values) < 2 {
		return values
	}
	out := values[:0]
	var last string
	for _, value := range values {
		if value == last {
			continue
		}
		out = append(out, value)
		last = value
	}
	return out
}

func expectedSourceForParam(step *rollout.Step, param *rollout.ParameterInfo, inputs map[string]bool) (string, string) {
	if step == nil || param == nil {
		return "", ""
	}
	for _, name := range paramTargetNames(param) {
		if source := strings.TrimSpace(step.With[name]); source != "" {
			return sourceKindForValue(source, param.Name, inputs, step), source
		}
		for _, bind := range step.Binds {
			if bind == nil {
				continue
			}
			if source := strings.TrimSpace(bind.Fields[name]); source != "" {
				from := strings.TrimSpace(bind.From)
				if from != "" && !strings.HasPrefix(source, from+".") {
					return "binding", from + "." + source
				}
				return sourceKindForValue(source, param.Name, inputs, step), source
			}
		}
	}
	if inputs[strings.TrimSpace(param.Name)] {
		return "input", strings.TrimSpace(param.Name)
	}
	return "unresolved", ""
}

func expectedCredentialForParam(step *rollout.Step, param *rollout.ParameterInfo, policy projectPolicy) string {
	_, source := expectedSourceForParam(step, param, nil)
	if source != "" && !referencesKnownStep(source, step) {
		return source
	}
	for _, binding := range credentialBindingNames(policy) {
		if strings.Contains(strings.ToLower(binding), strings.ToLower(strings.TrimSpace(param.Name))) {
			return binding
		}
	}
	return ""
}

func expectedCredentialForSecurity(step *rollout.Step, req openAPISecurityRequirement, policy projectPolicy) string {
	for _, name := range req.fieldNames() {
		if source := strings.TrimSpace(step.With[name]); source != "" {
			return source
		}
		for _, bind := range step.Binds {
			if bind == nil {
				continue
			}
			if source := strings.TrimSpace(bind.Fields[name]); source != "" {
				return source
			}
		}
	}
	for _, binding := range credentialBindingNames(policy) {
		if securityBindingMatches(binding, req) {
			return binding
		}
	}
	return ""
}

func securityRequestFieldName(req openAPISecurityRequirement) string {
	if strings.EqualFold(req.Type, "http") || strings.EqualFold(req.Scheme, "bearer") || strings.Contains(strings.ToLower(req.Scheme), "bearer") {
		return "Authorization"
	}
	if strings.TrimSpace(req.Name) != "" {
		return strings.TrimSpace(req.Name)
	}
	return strings.TrimSpace(req.Scheme)
}

func sourceKindForValue(source, paramName string, inputs map[string]bool, step *rollout.Step) string {
	source = strings.TrimSpace(source)
	if source == "" {
		return "unresolved"
	}
	if referencesKnownStep(source, step) {
		return "binding"
	}
	if referencesInputName(source, paramName) || inputs[source] {
		return "input"
	}
	return "literal"
}

func referencesKnownStep(source string, step *rollout.Step) bool {
	source = strings.TrimSpace(source)
	if source == "" || step == nil {
		return false
	}
	for _, dep := range step.DependsOn {
		dep = strings.TrimSpace(dep)
		if dep != "" && strings.Contains(source, dep+".") {
			return true
		}
	}
	for _, bind := range step.Binds {
		if bind == nil {
			continue
		}
		from := strings.TrimSpace(bind.From)
		if from != "" && strings.Contains(source, from+".") {
			return true
		}
	}
	return false
}

func credentialBindingNames(policy projectPolicy) []string {
	section := strings.TrimSpace(policy.CredentialSection)
	if section == "" {
		return nil
	}
	var out []string
	for _, line := range strings.Split(section, "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "-"))
		if line == "" {
			continue
		}
		tokens := credentialBindingTokens(line)
		lower := strings.ToLower(line)
		if strings.Contains(lower, "binding") {
			afterBinding := false
			for _, token := range tokens {
				tokenLower := strings.ToLower(token)
				if tokenLower == "binding" || tokenLower == "bindings" {
					afterBinding = true
					continue
				}
				if !afterBinding {
					continue
				}
				if credentialBindingToken(token) {
					out = append(out, token)
				}
			}
			continue
		}
		if len(tokens) == 1 && credentialBindingToken(tokens[0]) {
			out = append(out, tokens[0])
		}
	}
	return sortedUnique(out)
}

func credentialBindingTokens(value string) []string {
	var out []string
	for _, token := range strings.FieldsFunc(value, func(r rune) bool {
		return !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.')
	}) {
		token = strings.Trim(strings.TrimSpace(token), ".,;:")
		if token != "" {
			out = append(out, token)
		}
	}
	return out
}

func credentialBindingToken(token string) bool {
	token = strings.Trim(strings.TrimSpace(token), ".,;:")
	if token == "" {
		return false
	}
	lower := strings.ToLower(token)
	switch lower {
	case "-", "use", "using", "credential", "credentials", "binding", "bindings", "name", "names",
		"only", "none", "required", "value", "values", "secret", "secrets", "literal", "literals",
		"do", "not", "include", "runtime", "declared", "approved", "artifact", "artifacts":
		return false
	}
	return strings.Contains(token, "_") || strings.Contains(token, "-") || strings.Contains(token, ".")
}

func sortPlanParams(params []PlanParam) {
	sort.SliceStable(params, func(i, j int) bool {
		if params[i].Name != params[j].Name {
			return params[i].Name < params[j].Name
		}
		return params[i].In < params[j].In
	})
}

func sortPlanBindings(bindings []PlanBinding) {
	sort.SliceStable(bindings, func(i, j int) bool {
		if bindings[i].Target != bindings[j].Target {
			return bindings[i].Target < bindings[j].Target
		}
		if bindings[i].From != bindings[j].From {
			return bindings[i].From < bindings[j].From
		}
		return bindings[i].Source < bindings[j].Source
	})
}

func sortPlanResults(results []PlanResult) {
	sort.SliceStable(results, func(i, j int) bool {
		return results[i].Name < results[j].Name
	})
}

func sortPlanGaps(gaps []PlanGap) {
	sort.SliceStable(gaps, func(i, j int) bool {
		if gaps[i].Code != gaps[j].Code {
			return gaps[i].Code < gaps[j].Code
		}
		if gaps[i].Step != gaps[j].Step {
			return gaps[i].Step < gaps[j].Step
		}
		return gaps[i].Detail < gaps[j].Detail
	})
}
