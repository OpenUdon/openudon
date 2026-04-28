package synthesize

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/genelet/ramen/internal/openapidisco"
	"github.com/genelet/ramen/internal/uwsvalidate"
	"github.com/genelet/udon/generator"
	"github.com/genelet/udon/pkg/rollout"
	"github.com/genelet/udon/pkg/runner"
	"github.com/genelet/udon/pkg/uwsprofile"
	"github.com/tabilet/uws/uws1"
	"gopkg.in/yaml.v3"
)

type Options struct {
	ExampleDir string
	Provider   string
	Model      string
	Timeout    time.Duration

	Discoverer *openapidisco.Discoverer
	LLMClient  rollout.LLMClient
	ChatClient rollout.ChatClient
	SchemaPath string
}

type Result struct {
	ExampleDir        string
	ProjectPath       string
	IntentPath        string
	WorkflowPath      string
	UWSPath           string
	PlanJSONPath      string
	PlanMDPath        string
	DiscoveryJSONPath string
	ReviewPath        string
	QualityJSONPath   string
	QualityMDPath     string
	PrimaryOpenAPI    string
	OpenAPICandidates []openapidisco.Candidate
	DiscoveryReport   openapidisco.DiscoveryReport
}

func Run(ctx context.Context, opts Options) (*Result, error) {
	return Synthesize(ctx, opts)
}

func Synthesize(ctx context.Context, opts Options) (*Result, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	exampleDir, err := resolveExampleDir(opts.ExampleDir)
	if err != nil {
		return nil, err
	}
	result := resultPaths(exampleDir)
	projectBytes, err := os.ReadFile(result.ProjectPath)
	if err != nil {
		return nil, fmt.Errorf("read project brief: %w", err)
	}
	projectText := string(projectBytes)
	policy := analyzeProject(projectText)

	if opts.Discoverer == nil {
		opts.Discoverer = &openapidisco.Discoverer{}
	}
	var candidates []openapidisco.Candidate
	var primaryPath string
	var discoveryReport openapidisco.DiscoveryReport
	if !policy.NoOpenAPI {
		candidates, discoveryReport, err = opts.Discoverer.DiscoverWithReport(ctx, exampleDir, projectText)
		result.DiscoveryReport = discoveryReport
		_ = writeDiscoveryReport(result, discoveryReport)
		if err != nil {
			return nil, fmt.Errorf("discover OpenAPI documents: %w", err)
		}
		primary, err := openapidisco.SelectPrimary(candidates)
		if err != nil {
			return nil, err
		}
		primaryPath = primary.RelativePath
	}
	result.PrimaryOpenAPI = primaryPath
	result.OpenAPICandidates = candidates
	result.DiscoveryReport = discoveryReport

	llm, chat, provider, model, err := resolveClients(opts)
	if err != nil {
		return nil, err
	}

	intent, err := generateIntent(ctx, chat, projectText, candidates, primaryPath, policy)
	if err != nil {
		return nil, fmt.Errorf("generate intent: %w", err)
	}
	if !policy.NoOpenAPI {
		if refreshed, changed := discoverComplementaryOpenAPI(ctx, opts.Discoverer, exampleDir, projectText, candidates, intent, policy); changed {
			candidates = refreshed
			result.OpenAPICandidates = candidates
			if primary, err := openapidisco.SelectPrimary(candidates); err == nil {
				primaryPath = primary.RelativePath
				result.PrimaryOpenAPI = primaryPath
			}
			intent, err = generateIntent(ctx, chat, projectText, candidates, primaryPath, policy)
			if err != nil {
				return nil, fmt.Errorf("regenerate intent after complementary OpenAPI discovery: %w", err)
			}
		}
	}
	if err := validateIntentOpenAPIRefs(intent, candidates, primaryPath, policy.NoOpenAPI); err != nil {
		return nil, err
	}
	if err := validateIntentRuntimePolicy(intent, policy); err != nil {
		return nil, err
	}
	workflowPlan := buildWorkflowPlan(result, intent, candidates, policy)
	intentHCL, err := runner.RenderIntentHCL(intent)
	if err != nil {
		return nil, fmt.Errorf("render intent HCL: %w", err)
	}

	if err := ensureArtifactDirs(result); err != nil {
		return nil, err
	}
	if err := os.WriteFile(result.IntentPath, []byte(intentHCL), 0o644); err != nil {
		return nil, err
	}
	if err := writeWorkflowPlan(result, workflowPlan); err != nil {
		return nil, err
	}
	if err := generateWorkflow(ctx, result, intent, llm, provider, model, opts.Timeout); err != nil {
		return nil, err
	}
	if err := promoteWorkflow(result, opts.SchemaPath); err != nil {
		return nil, err
	}
	if err := writeReview(result, provider, model); err != nil {
		return nil, err
	}
	report, err := Assess(opts)
	if err != nil {
		return nil, err
	}
	if !report.Passed() {
		return &result, fmt.Errorf("quality gate failed; see %s", result.QualityJSONPath)
	}
	return &result, nil
}

func Build(ctx context.Context, opts Options) (*Result, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	exampleDir, err := resolveExampleDir(opts.ExampleDir)
	if err != nil {
		return nil, err
	}
	result := resultPaths(exampleDir)
	projectBytes, _ := os.ReadFile(result.ProjectPath)
	policy := analyzeProject(string(projectBytes))
	if opts.Discoverer == nil {
		opts.Discoverer = &openapidisco.Discoverer{}
	}
	var candidates []openapidisco.Candidate
	var discoveryReport openapidisco.DiscoveryReport
	if !policy.NoOpenAPI {
		candidates, discoveryReport, err = opts.Discoverer.DiscoverWithReport(ctx, exampleDir, string(projectBytes))
		result.DiscoveryReport = discoveryReport
		_ = writeDiscoveryReport(result, discoveryReport)
		if err != nil {
			return nil, fmt.Errorf("discover OpenAPI documents: %w", err)
		}
	}
	intent, err := rollout.ParseIntentFile(result.IntentPath)
	if err != nil {
		return nil, fmt.Errorf("parse intent.hcl: %w", err)
	}
	primary := strings.TrimSpace(intent.OpenAPI)
	if primary == "" && !policy.NoOpenAPI {
		selected, err := openapidisco.SelectPrimary(candidates)
		if err != nil {
			return nil, err
		}
		primary = selected.RelativePath
		intent.OpenAPI = primary
		intentHCL, err := runner.RenderIntentHCL(intent)
		if err != nil {
			return nil, err
		}
		if err := ensureArtifactDirs(result); err != nil {
			return nil, err
		}
		if err := os.WriteFile(result.IntentPath, []byte(intentHCL), 0o644); err != nil {
			return nil, err
		}
	}
	if err := validateIntentOpenAPIRefs(intent, candidates, primary, policy.NoOpenAPI); err != nil {
		return nil, err
	}
	if err := validateIntentRuntimePolicy(intent, policy); err != nil {
		return nil, err
	}
	workflowPlan := buildWorkflowPlan(result, intent, candidates, policy)
	result.PrimaryOpenAPI = primary
	result.OpenAPICandidates = candidates
	result.DiscoveryReport = discoveryReport
	if err := writeWorkflowPlan(result, workflowPlan); err != nil {
		return nil, err
	}
	llm, _, provider, model, err := resolveClients(opts)
	if err != nil {
		return nil, err
	}
	if err := generateWorkflow(ctx, result, intent, llm, provider, model, opts.Timeout); err != nil {
		return nil, err
	}
	if err := promoteWorkflow(result, opts.SchemaPath); err != nil {
		return nil, err
	}
	if err := writeReview(result, provider, model); err != nil {
		return nil, err
	}
	report, err := Assess(opts)
	if err != nil {
		return nil, err
	}
	if !report.Passed() {
		return &result, fmt.Errorf("quality gate failed; see %s", result.QualityJSONPath)
	}
	return &result, nil
}

func Promote(ctx context.Context, opts Options) (*Result, error) {
	exampleDir, err := resolveExampleDir(opts.ExampleDir)
	if err != nil {
		return nil, err
	}
	result := resultPaths(exampleDir)
	projectBytes, _ := os.ReadFile(result.ProjectPath)
	candidates, _ := openapidisco.LocalFiles(filepath.Join(exampleDir, "openapi"), exampleDir, string(projectBytes))
	result.OpenAPICandidates = candidates
	result.DiscoveryReport = openapidisco.DiscoveryReport{Attempts: []openapidisco.DiscoveryAttempt{{
		Kind:   "local",
		Source: filepath.ToSlash(filepath.Join(exampleDir, "openapi")),
		Status: "pass",
		Detail: fmt.Sprintf("%d local OpenAPI document(s)", len(candidates)),
	}}}
	_ = writeDiscoveryReport(result, result.DiscoveryReport)
	if len(candidates) > 0 {
		if primary, err := openapidisco.SelectPrimary(candidates); err == nil {
			result.PrimaryOpenAPI = primary.RelativePath
		}
	}
	if intent, err := rollout.ParseIntentFile(result.IntentPath); err == nil {
		policy := analyzeProject(string(projectBytes))
		if err := writeWorkflowPlan(result, buildWorkflowPlan(result, intent, candidates, policy)); err != nil {
			return nil, err
		}
	}
	if err := promoteWorkflow(result, opts.SchemaPath); err != nil {
		return nil, err
	}
	if err := writeReview(result, opts.Provider, opts.Model); err != nil {
		return nil, err
	}
	report, err := Assess(opts)
	if err != nil {
		return nil, err
	}
	if !report.Passed() {
		return &result, fmt.Errorf("quality gate failed; see %s", result.QualityJSONPath)
	}
	return &result, nil
}

func generateWorkflow(ctx context.Context, result Result, intent *rollout.Intent, llm rollout.LLMClient, provider, model string, timeout time.Duration) error {
	if timeout == 0 {
		timeout = 2 * time.Minute
	}
	workflowHCL, err := runner.GenerateConfigFromIntentWithClient(ctx, intent, result.PrimaryOpenAPI, llm, runner.GenerateOptions{
		Provider: provider,
		Model:    model,
		WorkDir:  result.ExampleDir,
		Validate: true,
		Format:   true,
		Timeout:  timeout,
	})
	if err != nil {
		return fmt.Errorf("generate workflow HCL: %w", err)
	}
	if err := ensureArtifactDirs(result); err != nil {
		return err
	}
	return os.WriteFile(result.WorkflowPath, []byte(workflowHCL), 0o644)
}

func promoteWorkflow(result Result, schemaPath string) error {
	plan, err := generator.NewRuntimePlanFromWorkflowFile(result.WorkflowPath, result.ExampleDir)
	if err != nil {
		return fmt.Errorf("compile workflow through udon: %w", err)
	}
	doc := plan.Document()
	normalizeUWSStepsForSchema(doc)
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
		schemaPath = defaultSchemaPath(result.ExampleDir)
	}
	if err := uwsvalidate.ValidateFile(schemaPath, result.UWSPath); err != nil {
		return fmt.Errorf("validate exported UWS: %w", err)
	}
	return nil
}

func writeReview(result Result, provider, model string) error {
	if err := ensureArtifactDirs(result); err != nil {
		return err
	}
	return os.WriteFile(result.ReviewPath, []byte(reviewMarkdown(result, provider, model)), 0o644)
}

func resolveExampleDir(example string) (string, error) {
	example = strings.TrimSpace(example)
	if example == "" {
		return "", fmt.Errorf("--example is required")
	}
	return filepath.Abs(example)
}

func resultPaths(exampleDir string) Result {
	workflowsDir := filepath.Join(exampleDir, "workflows")
	expectedDir := filepath.Join(exampleDir, "expected")
	return Result{
		ExampleDir:        exampleDir,
		ProjectPath:       filepath.Join(exampleDir, "project.md"),
		IntentPath:        filepath.Join(workflowsDir, "intent.hcl"),
		WorkflowPath:      filepath.Join(workflowsDir, "workflow.hcl"),
		UWSPath:           filepath.Join(workflowsDir, "workflow.uws.yaml"),
		PlanJSONPath:      filepath.Join(expectedDir, "plan.json"),
		PlanMDPath:        filepath.Join(expectedDir, "plan.md"),
		DiscoveryJSONPath: filepath.Join(expectedDir, "discovery.json"),
		ReviewPath:        filepath.Join(expectedDir, "review.md"),
		QualityJSONPath:   filepath.Join(expectedDir, "quality.json"),
		QualityMDPath:     filepath.Join(expectedDir, "quality.md"),
		OpenAPICandidates: nil,
	}
}

func writeDiscoveryReport(result Result, report openapidisco.DiscoveryReport) error {
	if len(report.Attempts) == 0 {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(result.DiscoveryJSONPath), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(result.DiscoveryJSONPath, append(data, '\n'), 0o644)
}

func ensureArtifactDirs(result Result) error {
	if err := os.MkdirAll(filepath.Dir(result.IntentPath), 0o755); err != nil {
		return err
	}
	return os.MkdirAll(filepath.Dir(result.ReviewPath), 0o755)
}

func defaultSchemaPath(exampleDir string) string {
	if _, file, _, ok := runtime.Caller(0); ok {
		repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
		schema := filepath.Join(repoRoot, "..", "uws", "versions", "1.0.0.json")
		if _, err := os.Stat(schema); err == nil {
			return schema
		}
	}
	return filepath.Join(exampleDir, "..", "..", "..", "uws", "versions", "1.0.0.json")
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

func resolveClients(opts Options) (rollout.LLMClient, rollout.ChatClient, string, string, error) {
	if opts.LLMClient != nil && opts.ChatClient != nil {
		return opts.LLMClient, opts.ChatClient, strings.TrimSpace(opts.Provider), strings.TrimSpace(opts.Model), nil
	}
	llm, provider, model, err := runner.NewLLMClientFromEnv(opts.Provider, opts.Model)
	if err != nil {
		return nil, nil, "", "", err
	}
	chat, ok := llm.(rollout.ChatClient)
	if !ok {
		return nil, nil, "", "", fmt.Errorf("selected provider %s does not support chat", provider)
	}
	return llm, chat, provider, model, nil
}

func generateIntent(ctx context.Context, chat rollout.ChatClient, projectText string, candidates []openapidisco.Candidate, primary string, policy projectPolicy) (*rollout.Intent, error) {
	system := `You turn a natural-language workflow brief into Udon rollout intent JSON.
Return only JSON. Do not include Markdown.
Use only the OpenAPI relative paths listed by the user. Do not invent OpenAPI filenames.
Prefer concise step names in snake_case. Use operation when an operationId is clear from the listed metadata.
Expand one business action into multiple technical steps when OpenAPI metadata requires intermediate calls.
For example, if weather requires lat/lon and the user provided a city, add a coordinate lookup step when an allowed OpenAPI operation exists.
If an intermediate operation is needed but no listed OpenAPI operation or approved fnct adapter exists, do not invent it.
Use bind blocks only when the brief clearly describes step-to-step data flow.
Use bind blocks for inferred hidden technical steps so required parameters are auditable.
When a later step depends on a prior step's output, preserve both depends_on and with/bind field mappings.
Never include secrets or credential values. Use security/token_from names instead.`
	user := fmt.Sprintf("Project brief:\n%s\n\nRuntime and OpenAPI policy:\n%s\nAvailable OpenAPI documents:\n%s\n\nPrimary OpenAPI path: %s\n\nReturn JSON matching this shape:\n%s",
		projectText,
		runtimePolicyPrompt(policy),
		specSummary(candidates),
		primary,
		intentJSONShape(),
	)
	response, err := chat.Chat(ctx, []rollout.ChatMessage{
		{Role: "system", Content: system},
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
	for _, candidate := range candidates {
		title := candidate.Title
		if title == "" {
			title = candidate.RelativePath
		}
		fmt.Fprintf(&b, "- path: %s\n  title: %s\n", candidate.RelativePath, title)
		if candidate.Description != "" {
			fmt.Fprintf(&b, "  description: %s\n", trimForPrompt(candidate.Description, 400))
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
		if limit > 80 {
			limit = 80
		}
		for i := 0; i < limit; i++ {
			op := ops[i]
			fmt.Fprintf(&b, "  operation: %s %s %s", op.OperationID, op.Method, op.Path)
			if op.Summary != "" {
				fmt.Fprintf(&b, " - %s", trimForPrompt(op.Summary, 160))
			}
			b.WriteString("\n")
			required := requiredOpenAPIParams(op)
			if len(required) > 0 {
				fmt.Fprintf(&b, "    required_parameters: %s\n", strings.Join(required, ", "))
			}
			responseFields := responseFieldNames(op, 12)
			if len(responseFields) > 0 {
				fmt.Fprintf(&b, "    response_fields: %s\n", strings.Join(responseFields, ", "))
			}
		}
		if len(ops) > limit {
			fmt.Fprintf(&b, "  omitted_operations: %d\n", len(ops)-limit)
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

func validateIntentOpenAPIRefs(intent *rollout.Intent, candidates []openapidisco.Candidate, primary string, noOpenAPI bool) error {
	if intent == nil {
		return nil
	}
	if noOpenAPI {
		var refs []string
		if strings.TrimSpace(intent.OpenAPI) != "" {
			refs = append(refs, "top-level openapi")
		}
		walkIntentSteps(intent.Steps, func(step *rollout.Step) {
			if step == nil {
				return
			}
			if strings.TrimSpace(step.OpenAPI) != "" {
				refs = append(refs, fmt.Sprintf("%s.openapi", strings.TrimSpace(step.Name)))
			}
			if strings.TrimSpace(step.Operation) != "" {
				refs = append(refs, fmt.Sprintf("%s.operation", strings.TrimSpace(step.Name)))
			}
		})
		if len(refs) > 0 {
			sort.Strings(refs)
			return fmt.Errorf("project declares OpenAPI: none required but intent references OpenAPI metadata: %s", strings.Join(refs, ", "))
		}
		return nil
	}
	allowed := map[string]bool{}
	if strings.TrimSpace(primary) != "" {
		allowed[primary] = true
	}
	for _, candidate := range candidates {
		allowed[candidate.RelativePath] = true
	}
	if strings.TrimSpace(intent.OpenAPI) == "" && primary != "" {
		intent.OpenAPI = primary
	}
	if strings.TrimSpace(intent.OpenAPI) != "" && !allowed[intent.OpenAPI] {
		return fmt.Errorf("generated intent referenced unavailable OpenAPI document %q", intent.OpenAPI)
	}
	var bad []string
	var walk func([]*rollout.Step)
	walk = func(steps []*rollout.Step) {
		for _, step := range steps {
			if step == nil {
				continue
			}
			if step.OpenAPI != "" && !allowed[step.OpenAPI] {
				bad = append(bad, step.OpenAPI)
			}
			walk(step.Steps)
			for _, branch := range step.Cases {
				if branch != nil {
					walk(branch.Steps)
				}
			}
			if step.Default != nil {
				walk(step.Default.Steps)
			}
		}
	}
	walk(intent.Steps)
	if len(bad) > 0 {
		sort.Strings(bad)
		return fmt.Errorf("generated intent referenced unavailable step OpenAPI documents: %s", strings.Join(bad, ", "))
	}
	return nil
}

func validateIntentRuntimePolicy(intent *rollout.Intent, policy projectPolicy) error {
	if intent == nil {
		return nil
	}
	var blocked []string
	walkIntentSteps(intent.Steps, func(step *rollout.Step) {
		if step == nil {
			return
		}
		typ := strings.ToLower(strings.TrimSpace(step.Type))
		if typ == "" || allowedIntentRuntimeType(typ) {
			if (typ == "cmd" || typ == "ssh") && !policy.AllowedRuntime[typ] {
				name := strings.TrimSpace(step.Name)
				if name == "" {
					name = "<unnamed>"
				}
				blocked = append(blocked, fmt.Sprintf("%s uses %s", name, typ))
			}
			return
		}
		name := strings.TrimSpace(step.Name)
		if name == "" {
			name = "<unnamed>"
		}
		blocked = append(blocked, fmt.Sprintf("%s uses unsupported runtime %s", name, typ))
	})
	if len(blocked) > 0 {
		sort.Strings(blocked)
		return fmt.Errorf("intent uses runtime not allowed by project policy: %s", strings.Join(blocked, "; "))
	}
	return nil
}

func allowedIntentRuntimeType(typ string) bool {
	switch strings.ToLower(strings.TrimSpace(typ)) {
	case "", "http", "openapi", "fnct", "cmd", "ssh",
		"sequence", "parallel", "switch", "merge", "loop", "await":
		return true
	default:
		return false
	}
}

func reviewMarkdown(result Result, provider, model string) string {
	var b strings.Builder
	b.WriteString("# Ramen Review Evidence\n\n")
	fmt.Fprintf(&b, "- Project brief: `%s`\n", relOrAbs(result.ExampleDir, result.ProjectPath))
	fmt.Fprintf(&b, "- Intent HCL: `%s`\n", relOrAbs(result.ExampleDir, result.IntentPath))
	fmt.Fprintf(&b, "- Workflow HCL: `%s`\n", relOrAbs(result.ExampleDir, result.WorkflowPath))
	fmt.Fprintf(&b, "- UWS artifact: `%s`\n", relOrAbs(result.ExampleDir, result.UWSPath))
	fmt.Fprintf(&b, "- Expected plan: `%s`\n", relOrAbs(result.ExampleDir, result.PlanJSONPath))
	fmt.Fprintf(&b, "- Discovery report: `%s`\n", relOrAbs(result.ExampleDir, result.DiscoveryJSONPath))
	fmt.Fprintf(&b, "- Primary OpenAPI: `%s`\n", result.PrimaryOpenAPI)
	if provider != "" || model != "" {
		fmt.Fprintf(&b, "- LLM: `%s` `%s`\n", provider, model)
	}
	b.WriteString("\n## OpenAPI Candidates\n\n")
	for _, candidate := range result.OpenAPICandidates {
		fmt.Fprintf(&b, "- `%s`", candidate.RelativePath)
		if candidate.Title != "" {
			fmt.Fprintf(&b, " - %s", candidate.Title)
		}
		if candidate.Source != "" {
			fmt.Fprintf(&b, " (%s)", candidate.Source)
		}
		b.WriteString("\n")
	}
	if len(result.DiscoveryReport.Attempts) > 0 {
		b.WriteString("\n## OpenAPI Discovery Attempts\n\n")
		for _, attempt := range result.DiscoveryReport.Attempts {
			fmt.Fprintf(&b, "- `%s` %s", attempt.Kind, attempt.Status)
			if attempt.Source != "" {
				fmt.Fprintf(&b, " `%s`", attempt.Source)
			}
			if attempt.Detail != "" {
				fmt.Fprintf(&b, " - %s", attempt.Detail)
			}
			b.WriteString("\n")
		}
	}
	if intent, err := rollout.ParseIntentFile(result.IntentPath); err == nil {
		b.WriteString("\n## Inferred Steps And Data Flow\n\n")
		writeIntentDataFlowReview(&b, intent)
	}
	b.WriteString("\n## Validation\n\n")
	b.WriteString("- Generated intent.hcl from project.md.\n")
	b.WriteString("- Generated workflow.hcl through udon rollout generation.\n")
	b.WriteString("- Compiled workflow.hcl through udon runtime plan generation.\n")
	b.WriteString("- Exported workflow.uws.yaml and validated it against the UWS schema.\n")
	b.WriteString("- Side-effectful execution was skipped.\n\n")
	b.WriteString("Trusted proof run, only when explicitly approved:\n\n")
	fmt.Fprintf(&b, "```bash\n./scripts/run-udon.sh %s %s\n```\n", relOrAbs(filepath.Dir(result.ExampleDir), result.WorkflowPath), result.ExampleDir)
	return b.String()
}

func writeIntentDataFlowReview(b *strings.Builder, intent *rollout.Intent) {
	if intent == nil || len(intent.Steps) == 0 {
		b.WriteString("- No intent steps were available for review.\n")
		return
	}
	var wrote bool
	walkIntentSteps(intent.Steps, func(step *rollout.Step) {
		if step == nil {
			return
		}
		name := strings.TrimSpace(step.Name)
		if name == "" {
			name = "<unnamed>"
		}
		typ := strings.TrimSpace(step.Type)
		if typ == "" {
			typ = "unspecified"
		}
		fmt.Fprintf(b, "- `%s` (%s)", name, typ)
		if step.Operation != "" {
			fmt.Fprintf(b, " operation `%s`", step.Operation)
		}
		if step.Do != "" {
			fmt.Fprintf(b, ": %s", strings.Join(strings.Fields(step.Do), " "))
		}
		b.WriteString("\n")
		wrote = true
		for _, bind := range step.Binds {
			if bind == nil {
				continue
			}
			fmt.Fprintf(b, "  - bind from `%s`", bind.From)
			if len(bind.Fields) == 0 {
				b.WriteString("\n")
				continue
			}
			keys := make([]string, 0, len(bind.Fields))
			for key := range bind.Fields {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				fmt.Fprintf(b, ": `%s <- %s`", key, bind.Fields[key])
			}
			b.WriteString("\n")
		}
	})
	if !wrote {
		b.WriteString("- No leaf intent steps were available for review.\n")
	}
}

func relOrAbs(base, path string) string {
	rel, err := filepath.Rel(base, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(rel)
}

func trimForPrompt(value string, max int) string {
	value = strings.Join(strings.Fields(value), " ")
	if len(value) <= max {
		return value
	}
	return value[:max] + "..."
}
