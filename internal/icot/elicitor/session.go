package elicitor

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/genelet/ramen/internal/projectwizard"
	"github.com/genelet/udon/pkg/rollout"
	"gopkg.in/yaml.v3"
)

type Session struct {
	Project         projectwizard.Answers   `json:"project,omitempty" yaml:"project,omitempty"`
	Intent          rollout.Intent          `json:"intent,omitempty" yaml:"intent,omitempty"`
	Credentials     []string                `json:"credentials,omitempty" yaml:"credentials,omitempty"`
	CredentialsSet  bool                    `json:"credentials_set,omitempty" yaml:"credentials_set,omitempty"`
	Safety          string                  `json:"safety,omitempty" yaml:"safety,omitempty"`
	SafetySet       bool                    `json:"safety_set,omitempty" yaml:"safety_set,omitempty"`
	Fallback        string                  `json:"fallback,omitempty" yaml:"fallback,omitempty"`
	FallbackSet     bool                    `json:"fallback_set,omitempty" yaml:"fallback_set,omitempty"`
	SideEffectScope string                  `json:"side_effect_scope,omitempty" yaml:"side_effect_scope,omitempty"`
	Annotations     []SourceAnnotation      `json:"annotations,omitempty" yaml:"annotations,omitempty"`
	Assumptions     []Assumption            `json:"assumptions,omitempty" yaml:"assumptions,omitempty"`
	Classifications []MappingClassification `json:"classifications,omitempty" yaml:"classifications,omitempty"`
}

type SourceAnnotation struct {
	Slot          string `json:"slot,omitempty" yaml:"slot,omitempty"`
	Source        string `json:"source,omitempty" yaml:"source,omitempty"`
	PromptVersion string `json:"prompt_version,omitempty" yaml:"prompt_version,omitempty"`
	Evidence      string `json:"evidence,omitempty" yaml:"evidence,omitempty"`
}

type Assumption struct {
	ID                   string `json:"id,omitempty" yaml:"id,omitempty"`
	Slot                 string `json:"slot,omitempty" yaml:"slot,omitempty"`
	Value                string `json:"value,omitempty" yaml:"value,omitempty"`
	Reason               string `json:"reason,omitempty" yaml:"reason,omitempty"`
	Evidence             string `json:"evidence,omitempty" yaml:"evidence,omitempty"`
	Risk                 string `json:"risk,omitempty" yaml:"risk,omitempty"`
	RequiresConfirmation bool   `json:"requires_confirmation,omitempty" yaml:"requires_confirmation,omitempty"`
}

func NewSessionFromAnswers(answers projectwizard.Answers) Session {
	intent := rollout.Intent{
		Workflow: &rollout.WorkflowMeta{
			Name:        slug(answers.ProjectName),
			Description: firstNonEmpty(answers.Goal, answers.ProjectName),
		},
	}
	for _, input := range parseInputs(answers.Inputs) {
		intent.Inputs = append(intent.Inputs, input)
	}
	stepName := "run_workflow"
	stepType := "fnct"
	if answers.UsesOpenAPI {
		stepType = "http"
		intent.OpenAPI = firstOpenAPIPath(answers.OpenAPI)
	}
	step := &rollout.Step{
		Name: stepName,
		Type: stepType,
		Do:   firstNonEmpty(answers.Goal, "Run the workflow."),
		With: map[string]string{},
	}
	if len(intent.Inputs) > 0 {
		step.Name = actionName(firstNonEmpty(answers.Goal, "run workflow"))
		for _, input := range intent.Inputs {
			step.With[input.Name] = "inputs." + input.Name
		}
	}
	if answers.UsesOpenAPI && strings.TrimSpace(intent.OpenAPI) == "" {
		step.OpenAPI = strings.TrimSpace(answers.OpenAPI)
	}
	intent.Steps = []*rollout.Step{step}
	for _, output := range parseOutputs(answers.Outputs, step.Name) {
		intent.Outputs = append(intent.Outputs, output)
	}
	return Session{
		Project:         answers,
		Intent:          intent,
		Credentials:     dedupeStrings(answers.Credentials),
		CredentialsSet:  true,
		Safety:          answers.Safety,
		SafetySet:       true,
		Fallback:        answers.Fallback,
		FallbackSet:     true,
		SideEffectScope: answers.SideEffectScope,
	}
}

func SessionFromIntent(intent *rollout.Intent, project projectwizard.Answers) Session {
	var value rollout.Intent
	if intent != nil {
		value = *intent.Clone()
	}
	session := Session{Project: project, Intent: value}
	if value.Workflow != nil {
		session.Project.ProjectName = humanTitle(value.Workflow.Name)
		session.Project.Goal = value.Workflow.Description
	}
	session.Project.UsesOpenAPI = value.RequiresOpenAPI()
	session.Project.OpenAPI = openAPIText(value)
	session.Project.Inputs = inputsText(value.Inputs)
	session.Project.Outputs = outputsText(value.Outputs)
	session.Project.DataFlow = dataFlowText(value.Steps)
	session.Project.FunctionContracts = functionText(value.Steps)
	session.Credentials = dedupeStrings(project.Credentials)
	session.CredentialsSet = true
	session.Safety = project.Safety
	session.SafetySet = true
	session.Fallback = project.Fallback
	session.FallbackSet = true
	session.SideEffectScope = project.SideEffectScope
	if session.SideEffectScope == "" {
		session.SideEffectScope = projectwizard.InferSideEffectScope(project.Safety)
	}
	return session
}

func (s *Session) Normalize() {
	if emptySession(*s) {
		return
	}
	if s.Intent.Workflow == nil {
		s.Intent.Workflow = &rollout.WorkflowMeta{}
	}
	if strings.TrimSpace(s.Intent.Workflow.Name) == "" {
		s.Intent.Workflow.Name = slug(firstNonEmpty(s.Project.ProjectName, s.Intent.Workflow.Description, "workflow"))
	}
	if strings.TrimSpace(s.Intent.Workflow.Description) == "" {
		s.Intent.Workflow.Description = firstNonEmpty(s.Project.Goal, s.Project.ProjectName, humanTitle(s.Intent.Workflow.Name))
	}
	s.Project.ProjectName = firstNonEmpty(s.Project.ProjectName, humanTitle(s.Intent.Workflow.Name))
	s.Project.Goal = firstNonEmpty(s.Project.Goal, s.Intent.Workflow.Description)
	s.Project.UsesOpenAPI = s.Intent.RequiresOpenAPI()
	s.Project.OpenAPI = openAPIText(s.Intent)
	s.Project.Inputs = inputsText(s.Intent.Inputs)
	s.Project.Outputs = outputsText(s.Intent.Outputs)
	s.Project.DataFlow = dataFlowText(s.Intent.Steps)
	s.Project.FunctionContracts = functionText(s.Intent.Steps)
	if s.CredentialsSet {
		s.Credentials = dedupeStrings(s.Credentials)
		s.Project.Credentials = s.Credentials
	} else {
		s.Project.Credentials = dedupeStrings(append(s.Project.Credentials, s.Credentials...))
		s.Credentials = s.Project.Credentials
	}
	if s.SafetySet {
		s.Project.Safety = strings.TrimSpace(s.Safety)
	} else {
		s.Project.Safety = firstNonEmpty(s.Project.Safety, s.Safety)
		s.Safety = s.Project.Safety
	}
	if s.FallbackSet {
		s.Project.Fallback = strings.TrimSpace(s.Fallback)
	} else {
		s.Project.Fallback = firstNonEmpty(s.Project.Fallback, s.Fallback)
		s.Fallback = s.Project.Fallback
	}
	s.SideEffectScope = projectwizard.NormalizeSideEffectScope(firstNonEmpty(s.SideEffectScope, s.Project.SideEffectScope))
	if s.SideEffectScope == "" {
		s.SideEffectScope = projectwizard.InferSideEffectScope(s.Project.Safety)
	}
	s.Project.SideEffectScope = s.SideEffectScope
	s.Intent.Inputs = dedupeInputs(s.Intent.Inputs)
	s.Intent.Outputs = dedupeOutputs(s.Intent.Outputs)
	s.Classifications = normalizeMappingClassifications(s.Classifications)
	normalizeSteps(s.Intent.Steps)
}

func (s Session) Missing() []string {
	var missing []string
	if s.Intent.Workflow == nil || strings.TrimSpace(s.Intent.Workflow.Name) == "" {
		missing = append(missing, "workflow name")
	}
	if s.Intent.Workflow == nil || strings.TrimSpace(s.Intent.Workflow.Description) == "" {
		missing = append(missing, "workflow goal")
	}
	missing = append(missing, s.Intent.MissingSlots()...)
	for _, step := range s.Intent.Steps {
		collectStepMissing(&missing, s.Intent.OpenAPI, step)
	}
	if len(s.Intent.Outputs) == 0 {
		missing = append(missing, "at least one output")
	}
	return dedupeStrings(missing)
}

func collectStepMissing(missing *[]string, defaultOpenAPI string, step *rollout.Step) {
	if step == nil {
		return
	}
	stepType := strings.ToLower(strings.TrimSpace(step.Type))
	if (stepType == "http" || stepType == "openapi") && (strings.TrimSpace(defaultOpenAPI) != "" || strings.TrimSpace(step.OpenAPI) != "") && strings.TrimSpace(step.Operation) == "" {
		name := firstNonEmpty(step.Name, "unnamed")
		*missing = append(*missing, "operation for step "+name)
	}
	for _, child := range step.Steps {
		collectStepMissing(missing, firstNonEmpty(step.OpenAPI, defaultOpenAPI), child)
	}
	for _, branch := range step.Cases {
		if branch == nil {
			continue
		}
		for _, child := range branch.Steps {
			collectStepMissing(missing, firstNonEmpty(step.OpenAPI, defaultOpenAPI), child)
		}
	}
	if step.Default != nil {
		for _, child := range step.Default.Steps {
			collectStepMissing(missing, firstNonEmpty(step.OpenAPI, defaultOpenAPI), child)
		}
	}
}

func DecodeSession(data []byte, ext string) (Session, error) {
	var session Session
	if strings.EqualFold(ext, ".json") {
		if err := json.Unmarshal(data, &session); err != nil {
			return Session{}, err
		}
		return session, nil
	}
	var generic any
	if err := yaml.Unmarshal(data, &generic); err != nil {
		return Session{}, err
	}
	jsonReady := yamlToJSONReady(generic)
	encoded, err := json.Marshal(jsonReady)
	if err != nil {
		return Session{}, err
	}
	if err := json.Unmarshal(encoded, &session); err != nil {
		return Session{}, err
	}
	return session, nil
}

func LooksLikeSession(s Session) bool {
	return !emptySession(s)
}

func emptySession(s Session) bool {
	return s.Intent.Workflow == nil &&
		len(s.Intent.Steps) == 0 &&
		len(s.Intent.Inputs) == 0 &&
		len(s.Intent.Outputs) == 0 &&
		strings.TrimSpace(s.Project.ProjectName+s.Project.Goal) == "" &&
		len(s.Credentials) == 0 &&
		strings.TrimSpace(s.Safety+s.Fallback+s.SideEffectScope) == ""
}

func mergeSessions(base, overlay Session) Session {
	if base.Intent.Workflow == nil && overlay.Intent.Workflow != nil {
		base.Intent.Workflow = overlay.Intent.Workflow
	} else if base.Intent.Workflow != nil && overlay.Intent.Workflow != nil {
		base.Intent.Workflow.Name = firstNonEmpty(base.Intent.Workflow.Name, overlay.Intent.Workflow.Name)
		base.Intent.Workflow.Description = firstNonEmpty(base.Intent.Workflow.Description, overlay.Intent.Workflow.Description)
	}
	base.Intent.OpenAPI = firstNonEmpty(base.Intent.OpenAPI, overlay.Intent.OpenAPI)
	base.Intent.ServerURL = firstNonEmpty(base.Intent.ServerURL, overlay.Intent.ServerURL)
	if len(base.Intent.Inputs) == 0 {
		base.Intent.Inputs = overlay.Intent.Inputs
	}
	if len(base.Intent.Steps) == 0 {
		base.Intent.Steps = overlay.Intent.Steps
	}
	if len(base.Intent.Outputs) == 0 {
		base.Intent.Outputs = overlay.Intent.Outputs
	}
	if len(base.Intent.Security) == 0 {
		base.Intent.Security = overlay.Intent.Security
	}
	base.Project = mergeAnswers(base.Project, overlay.Project)
	if overlay.CredentialsSet {
		base.Credentials = overlay.Credentials
		base.CredentialsSet = true
	} else {
		base.Credentials = dedupeStrings(append(base.Credentials, overlay.Credentials...))
	}
	if overlay.SafetySet {
		base.Safety = overlay.Safety
		base.SafetySet = true
	} else {
		base.Safety = firstNonEmpty(base.Safety, overlay.Safety)
	}
	if overlay.FallbackSet {
		base.Fallback = overlay.Fallback
		base.FallbackSet = true
	} else {
		base.Fallback = firstNonEmpty(base.Fallback, overlay.Fallback)
	}
	base.SideEffectScope = firstNonEmpty(base.SideEffectScope, overlay.SideEffectScope)
	base.Annotations = append(base.Annotations, overlay.Annotations...)
	base.Assumptions = mergeAssumptions(base.Assumptions, overlay.Assumptions)
	base.Classifications = mergeClassifications(base.Classifications, overlay.Classifications)
	base.Normalize()
	return base
}

func mergeAssumptions(base, overlay []Assumption) []Assumption {
	seen := map[string]bool{}
	var out []Assumption
	for _, assumption := range append(base, overlay...) {
		assumption.ID = strings.TrimSpace(assumption.ID)
		assumption.Slot = strings.TrimSpace(assumption.Slot)
		if assumption.ID == "" {
			assumption.ID = slugIdent(firstNonEmpty(assumption.Slot, assumption.Value, fmt.Sprintf("assumption_%d", len(out)+1)))
		}
		key := assumption.ID
		if key == "" {
			key = assumption.Slot + "\x00" + assumption.Value
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, assumption)
	}
	return out
}

func mergeAnswers(base, overlay projectwizard.Answers) projectwizard.Answers {
	base.ProjectName = firstNonEmpty(base.ProjectName, overlay.ProjectName)
	base.Goal = firstNonEmpty(base.Goal, overlay.Goal)
	base.Inputs = firstNonEmpty(base.Inputs, overlay.Inputs)
	base.Outputs = firstNonEmpty(base.Outputs, overlay.Outputs)
	base.DataFlow = firstNonEmpty(base.DataFlow, overlay.DataFlow)
	base.FunctionContracts = firstNonEmpty(base.FunctionContracts, overlay.FunctionContracts)
	base.UsesOpenAPI = base.UsesOpenAPI || overlay.UsesOpenAPI
	base.OpenAPI = firstNonEmpty(base.OpenAPI, overlay.OpenAPI)
	base.CmdApproved = base.CmdApproved || overlay.CmdApproved
	base.SSHApproved = base.SSHApproved || overlay.SSHApproved
	base.SideEffectScope = firstNonEmpty(base.SideEffectScope, overlay.SideEffectScope)
	base.Credentials = dedupeStrings(append(base.Credentials, overlay.Credentials...))
	base.Safety = firstNonEmpty(base.Safety, overlay.Safety)
	base.Fallback = firstNonEmpty(base.Fallback, overlay.Fallback)
	return base
}

func yamlToJSONReady(value any) any {
	switch v := value.(type) {
	case map[string]any:
		out := map[string]any{}
		for key, val := range v {
			out[key] = yamlToJSONReady(val)
		}
		return out
	case map[any]any:
		out := map[string]any{}
		for key, val := range v {
			out[fmt.Sprint(key)] = yamlToJSONReady(val)
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i, val := range v {
			out[i] = yamlToJSONReady(val)
		}
		return out
	default:
		return value
	}
}

func parseInputs(value string) []*rollout.Input {
	var out []*rollout.Input
	for _, item := range splitList(value) {
		name, rest := splitNameRest(item)
		if name == "" {
			continue
		}
		input := &rollout.Input{Name: slugIdent(name), Type: "string", Required: true}
		lower := strings.ToLower(rest)
		if strings.Contains(lower, "integer") || strings.Contains(lower, "number") || strings.Contains(lower, "int") {
			input.Type = "integer"
		} else if strings.Contains(lower, "bool") {
			input.Type = "boolean"
		} else if strings.Contains(lower, "object") || strings.Contains(lower, "json") {
			input.Type = "object"
		}
		input.Description = strings.TrimSpace(rest)
		out = append(out, input)
	}
	return out
}

func parseOutputs(value, stepName string) []*rollout.Output {
	var out []*rollout.Output
	if strings.TrimSpace(stepName) == "" {
		stepName = "run_workflow"
	}
	for _, item := range splitList(value) {
		name, rest := splitNameRest(item)
		if name == "" {
			name = "result"
		}
		source := strings.TrimSpace(rest)
		if source == "" || !strings.Contains(source, ".") {
			source = stepName + ".received_body"
		}
		out = append(out, &rollout.Output{Name: slugIdent(name), From: source, Description: item})
	}
	if len(out) == 0 && stepName != "" {
		out = append(out, &rollout.Output{Name: "result", From: stepName + ".received_body"})
	}
	return out
}

func splitNameRest(value string) (string, string) {
	value = strings.TrimSpace(strings.Trim(value, "`"))
	for _, sep := range []string{":", "="} {
		if idx := strings.Index(value, sep); idx >= 0 {
			return strings.TrimSpace(strings.Trim(value[:idx], "`")), strings.TrimSpace(value[idx+1:])
		}
	}
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return "", ""
	}
	return strings.Trim(fields[0], "`"), strings.TrimSpace(strings.TrimPrefix(value, fields[0]))
}

func splitList(value string) []string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	fields := strings.FieldsFunc(value, func(r rune) bool {
		return r == '\n' || r == ';' || r == ','
	})
	var out []string
	for _, field := range fields {
		field = strings.TrimSpace(strings.TrimPrefix(field, "-"))
		if field != "" && !strings.Contains(strings.ToLower(field), "none declared") {
			out = append(out, field)
		}
	}
	return out
}

func firstOpenAPIPath(value string) string {
	for _, item := range splitList(value) {
		for _, token := range strings.Fields(item) {
			token = strings.Trim(token, "`'\".,")
			if strings.HasPrefix(token, "openapi/") || strings.Contains(token, "/openapi/") {
				return filepath.ToSlash(token)
			}
			ext := strings.ToLower(filepath.Ext(token))
			if ext == ".yaml" || ext == ".yml" || ext == ".json" {
				return filepath.ToSlash(token)
			}
		}
	}
	return ""
}

func inputsText(inputs []*rollout.Input) string {
	var parts []string
	for _, input := range inputs {
		if input == nil || strings.TrimSpace(input.Name) == "" {
			continue
		}
		req := "optional"
		if input.Required {
			req = "required"
		}
		typ := firstNonEmpty(input.Type, "string")
		parts = append(parts, fmt.Sprintf("`%s`: %s %s", input.Name, req, typ))
	}
	return strings.Join(parts, "; ")
}

func outputsText(outputs []*rollout.Output) string {
	var parts []string
	for _, output := range outputs {
		if output == nil || strings.TrimSpace(output.Name) == "" {
			continue
		}
		parts = append(parts, fmt.Sprintf("`%s` from `%s`", output.Name, output.From))
	}
	return strings.Join(parts, "; ")
}

func dataFlowText(steps []*rollout.Step) string {
	var parts []string
	walkSteps(steps, func(step *rollout.Step) {
		if step == nil {
			return
		}
		for _, dep := range step.DependsOn {
			parts = append(parts, fmt.Sprintf("`%s` depends on `%s`", step.Name, dep))
		}
		for _, bind := range step.Binds {
			if bind == nil {
				continue
			}
			for field, source := range bind.Fields {
				parts = append(parts, fmt.Sprintf("`%s.%s` comes from `%s.%s`", step.Name, field, bind.From, source))
			}
		}
	})
	return strings.Join(parts, "; ")
}

func functionText(steps []*rollout.Step) string {
	var parts []string
	walkSteps(steps, func(step *rollout.Step) {
		if step == nil || strings.TrimSpace(step.Type) != "fnct" {
			return
		}
		parts = append(parts, fmt.Sprintf("`%s`: %s", step.Name, step.Do))
	})
	return strings.Join(parts, "; ")
}

func openAPIText(intent rollout.Intent) string {
	var refs []string
	if strings.TrimSpace(intent.OpenAPI) != "" {
		refs = append(refs, intent.OpenAPI)
	}
	walkSteps(intent.Steps, func(step *rollout.Step) {
		if strings.TrimSpace(step.OpenAPI) != "" {
			refs = append(refs, step.OpenAPI)
		}
	})
	refs = dedupeStrings(refs)
	if len(refs) == 0 {
		return ""
	}
	return strings.Join(refs, "; ")
}

func normalizeSteps(steps []*rollout.Step) {
	seen := map[string]int{}
	for i, step := range steps {
		if step == nil {
			continue
		}
		step.Name = slugIdent(firstNonEmpty(step.Name, step.Do, fmt.Sprintf("step_%d", i+1)))
		seen[step.Name]++
		if seen[step.Name] > 1 {
			step.Name = fmt.Sprintf("%s_%d", step.Name, seen[step.Name])
		}
		step.Type = strings.ToLower(strings.TrimSpace(step.Type))
		if step.Type == "" {
			step.Type = "fnct"
		}
		if step.Do == "" {
			step.Do = humanTitle(step.Name)
		}
		if len(step.With) == 0 {
			step.With = nil
		}
		step.DependsOn = dedupeStrings(step.DependsOn)
		normalizeSteps(step.Steps)
		for _, branch := range step.Cases {
			if branch != nil {
				normalizeSteps(branch.Steps)
			}
		}
		if step.Default != nil {
			normalizeSteps(step.Default.Steps)
		}
	}
}

func walkSteps(steps []*rollout.Step, visit func(*rollout.Step)) {
	for _, step := range steps {
		if step == nil {
			continue
		}
		visit(step)
		walkSteps(step.Steps, visit)
		for _, branch := range step.Cases {
			if branch != nil {
				walkSteps(branch.Steps, visit)
			}
		}
		if step.Default != nil {
			walkSteps(step.Default.Steps, visit)
		}
	}
}

func dedupeInputs(inputs []*rollout.Input) []*rollout.Input {
	seen := map[string]bool{}
	var out []*rollout.Input
	for _, input := range inputs {
		if input == nil {
			continue
		}
		input.Name = slugIdent(input.Name)
		if input.Name == "" || seen[input.Name] {
			continue
		}
		if input.Type == "" {
			input.Type = "string"
		}
		seen[input.Name] = true
		out = append(out, input)
	}
	return out
}

func dedupeOutputs(outputs []*rollout.Output) []*rollout.Output {
	seen := map[string]bool{}
	var out []*rollout.Output
	for _, output := range outputs {
		if output == nil {
			continue
		}
		output.Name = slugIdent(output.Name)
		if output.Name == "" || output.From == "" || seen[output.Name] {
			continue
		}
		seen[output.Name] = true
		out = append(out, output)
	}
	return out
}

func dedupeStrings(values []string) []string {
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
	sort.Strings(out)
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func actionName(value string) string {
	fields := strings.Fields(value)
	if len(fields) > 4 {
		fields = fields[:4]
	}
	return slug(strings.Join(fields, " "))
}

var nonIdentRE = regexp.MustCompile(`[^a-zA-Z0-9_]+`)

func slug(value string) string {
	return slugIdent(strings.ToLower(value))
}

func slugIdent(value string) string {
	value = strings.TrimSpace(strings.Trim(value, "`'\""))
	value = strings.ReplaceAll(value, "-", "_")
	value = nonIdentRE.ReplaceAllString(value, "_")
	value = strings.Trim(value, "_")
	if value == "" {
		return ""
	}
	if value[0] >= '0' && value[0] <= '9' {
		value = "v_" + value
	}
	return value
}

func humanTitle(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	value = strings.ReplaceAll(value, "_", " ")
	words := strings.Fields(value)
	for i, word := range words {
		if len(word) == 0 {
			continue
		}
		words[i] = strings.ToUpper(word[:1]) + word[1:]
	}
	return strings.Join(words, " ")
}
