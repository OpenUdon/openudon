package synthesize

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/genelet/ramen/internal/openapidisco"
	"github.com/genelet/udon/pkg/rollout"
	"github.com/genelet/udon/pkg/runner"
)

const (
	maxPromptOperationsTotal       = 80
	maxPromptDescriptionChars      = 400
	maxPromptOperationSummaryChars = 160
	maxPromptResponseFields        = 12
	intentGenerationModeStructured = "structured"
	intentGenerationModeLegacy     = "legacy"
)

//go:embed prompts/intent_generation.tmpl
var intentGenerationSystemPrompt string

//go:embed prompts/examples/*.json
var intentPromptExamples embed.FS

//go:embed schemas/intent.schema.json
var embeddedIntentSchema []byte

func generateIntent(ctx context.Context, chat rollout.ChatClient, projectText string, candidates []openapidisco.Candidate, primary string, policy projectPolicy, feedback string) (*rollout.Intent, error) {
	intent, _, err := generateIntentWithMode(ctx, chat, projectText, candidates, primary, policy, feedback, nil)
	return intent, err
}

func generateIntentWithMode(ctx context.Context, chat rollout.ChatClient, projectText string, candidates []openapidisco.Candidate, primary string, policy projectPolicy, feedback string, temperature *float64) (*rollout.Intent, string, error) {
	messages := intentPromptMessagesForMode(projectText, candidates, primary, policy, feedback, supportsStructuredChat(chat))
	return generateIntentFromMessagesWithMode(ctx, chat, messages, primary, policy, temperature)
}

func intentPromptMessages(projectText string, candidates []openapidisco.Candidate, primary string, policy projectPolicy, feedback string) []rollout.ChatMessage {
	return intentPromptMessagesForMode(projectText, candidates, primary, policy, feedback, false)
}

func intentPromptMessagesForMode(projectText string, candidates []openapidisco.Candidate, primary string, policy projectPolicy, feedback string, structured bool) []rollout.ChatMessage {
	feedback = strings.TrimSpace(feedback)
	feedbackSection := ""
	if feedback != "" {
		feedbackSection = "\nPrevious quality failure to repair:\n" + feedback + "\n"
	}
	user := fmt.Sprintf("Project brief:\n%s\n\nRuntime and OpenAPI policy:\n%s\n%sAvailable OpenAPI documents:\n%s\n\nPrimary OpenAPI path: %s\n%s\nReturn JSON matching this shape:\n%s",
		projectText,
		runtimePolicyPrompt(policy),
		requiredByProjectPrompt(policy),
		specSummary(candidates),
		primary,
		feedbackSection,
		intentJSONShape(),
	)
	return []rollout.ChatMessage{
		{Role: "system", Content: renderIntentSystemPromptForMode(structured)},
		{Role: "user", Content: user},
	}
}

func generateIntentFromMessages(ctx context.Context, chat rollout.ChatClient, messages []rollout.ChatMessage, primary string, policy projectPolicy) (*rollout.Intent, error) {
	intent, _, err := generateIntentFromMessagesWithMode(ctx, chat, messages, primary, policy, nil)
	return intent, err
}

func generateIntentFromMessagesWithMode(ctx context.Context, chat rollout.ChatClient, messages []rollout.ChatMessage, primary string, policy projectPolicy, temperature *float64) (*rollout.Intent, string, error) {
	if structured, ok := chat.(rollout.StructuredChat); ok {
		raw, err := structured.StructuredChat(ctx, messages, json.RawMessage(embeddedIntentSchema), rollout.StructuredOpts{Temperature: temperature})
		if err == nil {
			intent, err := decodeIntentJSON(raw, primary, policy)
			if err != nil {
				return nil, intentGenerationModeStructured, err
			}
			return intent, intentGenerationModeStructured, nil
		}
		messages = legacyJSONInstructionMessages(messages)
	}
	response, err := chat.Chat(ctx, messages)
	if err != nil {
		return nil, intentGenerationModeLegacy, err
	}
	jsonText, err := extractJSON(response)
	if err != nil {
		return nil, intentGenerationModeLegacy, fmt.Errorf("extract intent JSON: %w", err)
	}
	intent, err := decodeIntentJSON(jsonText, primary, policy)
	if err != nil {
		return nil, intentGenerationModeLegacy, err
	}
	return intent, intentGenerationModeLegacy, nil
}

func legacyJSONInstructionMessages(messages []rollout.ChatMessage) []rollout.ChatMessage {
	const instruction = "Return only JSON. Do not include Markdown."
	out := append([]rollout.ChatMessage(nil), messages...)
	for i := range out {
		if strings.Contains(out[i].Content, instruction) {
			return out
		}
	}
	for i := len(out) - 1; i >= 0; i-- {
		if strings.TrimSpace(out[i].Role) == "user" {
			out[i].Content = strings.TrimSpace(out[i].Content) + "\n\n" + instruction
			return out
		}
	}
	return out
}

func decodeIntentJSON(jsonText string, primary string, policy projectPolicy) (*rollout.Intent, error) {
	var intent rollout.Intent
	if err := json.Unmarshal([]byte(jsonText), &intent); err != nil {
		return nil, fmt.Errorf("decode intent JSON: %w", err)
	}
	if strings.TrimSpace(intent.OpenAPI) == "" && primary != "" && !policy.NoOpenAPI {
		intent.OpenAPI = primary
	}
	sanitizeGeneratedIntent(&intent, policy)
	intent.EnsureActionDescriptions()
	if _, err := runner.RenderIntentHCL(&intent); err != nil {
		return nil, err
	}
	return &intent, nil
}

func sanitizeGeneratedIntent(intent *rollout.Intent, policy projectPolicy) {
	if intent == nil {
		return
	}
	intent.Steps = foldParameterSetterSteps(intent.Steps)
	names := map[string]bool{}
	var collect func([]*rollout.Step)
	collect = func(steps []*rollout.Step) {
		for _, step := range steps {
			if step == nil {
				continue
			}
			if name := strings.TrimSpace(step.Name); name != "" {
				names[name] = true
			}
			collect(step.Steps)
			for _, branch := range step.Cases {
				if branch != nil {
					collect(branch.Steps)
				}
			}
			if step.Default != nil {
				collect(step.Default.Steps)
			}
		}
	}
	collect(intent.Steps)
	applyProjectBindingHints(intent, policy)
	applyRuntimeInputHints(intent.Steps, policy.Inputs)
	applyCredentialBindingHints(intent.Steps, credentialBindingNames(policy))
	convertCredentialBinds(intent.Steps, names, credentialBindingNames(policy), intentSecurityNames(intent))

	var clean func([]*rollout.Step)
	clean = func(steps []*rollout.Step) {
		for _, step := range steps {
			if step == nil {
				continue
			}
			if len(step.DependsOn) > 0 {
				filtered := make([]string, 0, len(step.DependsOn))
				seen := map[string]bool{}
				for _, dep := range step.DependsOn {
					dep = strings.TrimSpace(dep)
					if dep == "" || dep == strings.TrimSpace(step.Name) || !names[dep] || seen[dep] {
						continue
					}
					seen[dep] = true
					filtered = append(filtered, dep)
				}
				step.DependsOn = filtered
			}
			if len(step.With) == 0 {
				step.With = nil
			}
			if len(step.Binds) == 0 {
				step.Binds = nil
			}
			clean(step.Steps)
			for _, branch := range step.Cases {
				if branch != nil {
					clean(branch.Steps)
				}
			}
			if step.Default != nil {
				clean(step.Default.Steps)
				if len(step.Default.Steps) == 0 {
					step.Default = nil
				}
			}
		}
	}
	clean(intent.Steps)
}

func foldParameterSetterSteps(steps []*rollout.Step) []*rollout.Step {
	stepsByName := map[string]*rollout.Step{}
	var collect func([]*rollout.Step)
	collect = func(items []*rollout.Step) {
		for _, step := range items {
			if step == nil {
				continue
			}
			if name := strings.TrimSpace(step.Name); name != "" {
				stepsByName[name] = step
			}
			collect(step.Steps)
			for _, branch := range step.Cases {
				if branch != nil {
					collect(branch.Steps)
				}
			}
			if step.Default != nil {
				collect(step.Default.Steps)
			}
		}
	}
	collect(steps)
	return foldParameterSetterStepList(steps, stepsByName)
}

func foldParameterSetterStepList(steps []*rollout.Step, stepsByName map[string]*rollout.Step) []*rollout.Step {
	out := make([]*rollout.Step, 0, len(steps))
	for _, step := range steps {
		if step == nil {
			continue
		}
		step.Steps = foldParameterSetterStepList(step.Steps, stepsByName)
		for _, branch := range step.Cases {
			if branch != nil {
				branch.Steps = foldParameterSetterStepList(branch.Steps, stepsByName)
			}
		}
		if step.Default != nil {
			step.Default.Steps = foldParameterSetterStepList(step.Default.Steps, stepsByName)
		}
		if foldedParameterSetterStep(step, stepsByName) {
			continue
		}
		out = append(out, step)
	}
	return out
}

func foldedParameterSetterStep(step *rollout.Step, stepsByName map[string]*rollout.Step) bool {
	if step == nil {
		return false
	}
	targetName, values := parameterSetterTargetAndValues(step)
	if targetName == "" || len(values) == 0 {
		return false
	}
	target := stepsByName[targetName]
	if target == nil || target == step {
		return false
	}
	if target.With == nil {
		target.With = map[string]string{}
	}
	for key, value := range values {
		key = strings.TrimSpace(key)
		if key == "" || strings.TrimSpace(target.With[key]) != "" {
			continue
		}
		target.With[key] = strings.TrimSpace(value)
	}
	return true
}

func parameterSetterTargetAndValues(step *rollout.Step) (string, map[string]string) {
	if step == nil {
		return "", nil
	}
	set := strings.TrimSpace(step.Set)
	targetName := strings.TrimSpace(strings.TrimSuffix(set, ".with"))
	if targetName != "" && targetName != set && len(step.With) > 0 {
		return targetName, step.With
	}
	assignments := map[string]string{}
	var target string
	for _, part := range strings.Split(set, ";") {
		left, right, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		left = strings.TrimSpace(left)
		right = strings.Trim(strings.TrimSpace(right), `"`)
		prefix, field, ok := strings.Cut(left, ".with.")
		if !ok || strings.TrimSpace(prefix) == "" || strings.TrimSpace(field) == "" || right == "" {
			continue
		}
		prefix = strings.TrimSpace(prefix)
		if target == "" {
			target = prefix
		}
		if target != prefix {
			return "", nil
		}
		assignments[strings.TrimSpace(field)] = right
	}
	return target, assignments
}

func applyRuntimeInputHints(steps []*rollout.Step, inputs []InputDecl) {
	if len(steps) == 0 || len(inputs) == 0 {
		return
	}
	var flat []*rollout.Step
	walk := func(items []*rollout.Step) {}
	walk = func(items []*rollout.Step) {
		for _, step := range items {
			if step == nil {
				continue
			}
			flat = append(flat, step)
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
	walk(steps)
	for _, input := range inputs {
		name := strings.TrimSpace(input.Name)
		if name == "" {
			continue
		}
		var candidates []*rollout.Step
		for _, step := range flat {
			if stepLikelyConsumesInput(step, name) {
				candidates = append(candidates, step)
			}
		}
		if len(candidates) == 1 {
			step := candidates[0]
			if step.With == nil {
				step.With = map[string]string{}
			}
			if strings.TrimSpace(step.With[name]) == "" {
				step.With[name] = "inputs." + name
			}
		}
	}
}

func stepLikelyConsumesInput(step *rollout.Step, inputName string) bool {
	if step == nil {
		return false
	}
	kind := strings.ToLower(strings.TrimSpace(step.Type))
	if kind != "http" && kind != "openapi" {
		return false
	}
	input := normalizedIdentifier(inputName)
	if input == "" {
		return false
	}
	combined := normalizedIdentifier(step.Name + " " + step.Do + " " + step.Operation)
	if strings.Contains(combined, input) {
		return true
	}
	stem := strings.TrimSuffix(strings.TrimSuffix(input, "id"), "ids")
	return len(stem) >= 4 && strings.Contains(combined, stem)
}

func applyCredentialBindingHints(steps []*rollout.Step, bindings []string) {
	if len(steps) == 0 || len(bindings) == 0 {
		return
	}
	var flat []*rollout.Step
	var walk func([]*rollout.Step)
	walk = func(items []*rollout.Step) {
		for _, step := range items {
			if step == nil {
				continue
			}
			flat = append(flat, step)
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
	walk(steps)
	for _, binding := range bindings {
		field, selector := credentialFieldAndSelector(binding)
		if field == "" {
			continue
		}
		var candidates []*rollout.Step
		for _, step := range flat {
			if stepLikelyConsumesCredential(step, selector, field) {
				candidates = append(candidates, step)
			}
		}
		if len(candidates) != 1 {
			continue
		}
		step := candidates[0]
		if step.With == nil {
			step.With = map[string]string{}
		}
		if strings.TrimSpace(step.With[field]) == "" {
			step.With[field] = binding
		}
	}
}

func credentialFieldAndSelector(binding string) (string, string) {
	binding = strings.Trim(strings.TrimSpace(binding), ".,;:")
	if binding == "" {
		return "", ""
	}
	parts := strings.FieldsFunc(binding, func(r rune) bool {
		return r == '_' || r == '-' || r == '.' || r == '/'
	})
	if len(parts) == 0 {
		return "", ""
	}
	last := strings.ToLower(parts[len(parts)-1])
	selector := strings.ToLower(parts[0])
	switch {
	case last == "appid":
		return "appid", selector
	case last == "key" && len(parts) >= 2 && strings.EqualFold(parts[len(parts)-2], "api"):
		return "api_key", selector
	case last == "token":
		return "token", selector
	case last == "secret":
		return "secret", selector
	default:
		return "", ""
	}
}

func stepLikelyConsumesCredential(step *rollout.Step, selector, field string) bool {
	if step == nil {
		return false
	}
	kind := strings.ToLower(strings.TrimSpace(step.Type))
	if kind != "http" && kind != "openapi" {
		return false
	}
	combined := strings.ToLower(step.Name + " " + step.Do + " " + step.Operation)
	if selector != "" && strings.Contains(combined, selector) {
		return true
	}
	return strings.Contains(combined, strings.TrimSuffix(field, "_key"))
}

func applyProjectBindingHints(intent *rollout.Intent, policy projectPolicy) {
	if intent == nil || len(policy.BindingHints) == 0 {
		return
	}
	var steps []*rollout.Step
	stepsByName := map[string]*rollout.Step{}
	var collect func([]*rollout.Step)
	collect = func(items []*rollout.Step) {
		for _, step := range items {
			if step == nil {
				continue
			}
			if name := strings.TrimSpace(step.Name); name != "" {
				stepsByName[name] = step
				steps = append(steps, step)
			}
			collect(step.Steps)
			for _, branch := range step.Cases {
				if branch != nil {
					collect(branch.Steps)
				}
			}
			if step.Default != nil {
				collect(step.Default.Steps)
			}
		}
	}
	collect(intent.Steps)
	for _, hint := range policy.BindingHints {
		stepName, field := splitStepFieldTarget(hint.To, hint.Field)
		if stepName == "" || field == "" {
			field = strings.TrimSpace(hint.Field)
		}
		if field == "" {
			continue
		}
		step := resolveBindingHintStep(stepName, hint.StepSelector, field, hint.From, steps, stepsByName)
		if step == nil {
			continue
		}
		if step.With == nil {
			step.With = map[string]string{}
		}
		if strings.TrimSpace(step.With[field]) == "" {
			step.With[field] = normalizeHintSource(strings.TrimSpace(hint.From), steps, stepsByName)
		}
	}
}

func normalizeHintSource(source string, steps []*rollout.Step, stepsByName map[string]*rollout.Step) string {
	source = strings.TrimSpace(source)
	prefix, rest, ok := strings.Cut(source, ".")
	if !ok || prefix == "" || stepsByName[prefix] != nil {
		return source
	}
	step := resolveBindingHintStep(prefix, "", "", "", steps, stepsByName)
	if step == nil || strings.TrimSpace(step.Name) == "" {
		return source
	}
	return strings.TrimSpace(step.Name) + "." + rest
}

func resolveBindingHintStep(stepName, selector, field, source string, steps []*rollout.Step, stepsByName map[string]*rollout.Step) *rollout.Step {
	stepName = strings.TrimSpace(stepName)
	if stepName != "" {
		if step := stepsByName[stepName]; step != nil {
			return step
		}
		var candidates []*rollout.Step
		for _, step := range steps {
			if stepNameMatchesHint(step, stepName) {
				candidates = append(candidates, step)
			}
		}
		if len(candidates) == 1 {
			return candidates[0]
		}
	}
	var candidates []*rollout.Step
	for _, step := range steps {
		if stepMatchesSelector(step, selector, field, source) {
			candidates = append(candidates, step)
		}
	}
	if len(candidates) == 1 {
		return candidates[0]
	}
	return nil
}

func stepNameMatchesHint(step *rollout.Step, hint string) bool {
	if step == nil {
		return false
	}
	hint = normalizedIdentifier(hint)
	if hint == "" {
		return false
	}
	name := normalizedIdentifier(step.Name)
	return name == hint || strings.Contains(name, hint) || strings.Contains(hint, name)
}

func stepMatchesSelector(step *rollout.Step, selector, field, source string) bool {
	if step == nil {
		return false
	}
	selector = strings.ToLower(strings.TrimSpace(selector))
	field = strings.ToLower(strings.TrimSpace(field))
	source = strings.ToLower(strings.TrimSpace(source))
	name := strings.ToLower(strings.TrimSpace(step.Name))
	do := strings.ToLower(strings.TrimSpace(step.Do))
	operation := strings.ToLower(strings.TrimSpace(step.Operation))
	if selector != "" {
		switch selector {
		case "page_1":
			return strings.Contains(name, "page_1") || strings.Contains(do, "first page")
		case "page_2":
			return strings.Contains(name, "page_2") || strings.Contains(do, "second page")
		case "list":
			return strings.Contains(name, "list") || strings.Contains(operation, "list")
		case "coordinate":
			return strings.Contains(name, "coordinate") || (containsResolveVerb(do) && strings.Contains(do, "coordinate")) || operation == "direct_get"
		default:
			return strings.Contains(name, selector) || strings.Contains(do, selector) || strings.Contains(operation, selector)
		}
	}
	if field == "q" && strings.Contains(source, "toronto") {
		return strings.Contains(name, "coordinate") || strings.Contains(do, "coordinate") || operation == "direct_get"
	}
	if field == "page" || field == "limit" {
		return strings.Contains(name, "list") || strings.Contains(operation, "list")
	}
	return false
}

func containsResolveVerb(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	for _, field := range strings.FieldsFunc(value, func(r rune) bool {
		return !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'))
	}) {
		switch field {
		case "resolve", "resolves", "resolving":
			return true
		}
	}
	return false
}

func normalizedIdentifier(value string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(value)) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func splitStepFieldTarget(target, fallbackField string) (string, string) {
	target = strings.Trim(strings.TrimSpace(target), ".,")
	field := strings.Trim(strings.TrimSpace(fallbackField), ".,")
	if target == "" {
		return "", field
	}
	idx := strings.LastIndex(target, ".")
	if idx < 0 {
		return target, field
	}
	step := strings.TrimSpace(target[:idx])
	if field == "" {
		field = strings.TrimSpace(target[idx+1:])
	}
	return step, field
}

func convertCredentialBinds(steps []*rollout.Step, stepNames map[string]bool, policyCredentials, securityNames []string) {
	credentialNames := map[string]string{}
	for _, name := range append(policyCredentials, securityNames...) {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		credentialNames[strings.ToLower(name)] = name
		credentialNames[strings.ToLower("security/"+name)] = name
	}
	for _, step := range steps {
		if step == nil {
			continue
		}
		if len(step.Binds) > 0 {
			filtered := make([]*rollout.StepBind, 0, len(step.Binds))
			for _, bind := range step.Binds {
				if bind == nil {
					continue
				}
				from := strings.TrimSpace(bind.From)
				if stepNames[from] {
					filtered = append(filtered, bind)
					continue
				}
				credential := credentialNameForBindSource(from, credentialNames)
				if credential == "" {
					continue
				}
				if step.With == nil {
					step.With = map[string]string{}
				}
				for target := range bind.Fields {
					target = strings.TrimSpace(target)
					if target == "" || strings.TrimSpace(step.With[target]) != "" {
						continue
					}
					step.With[target] = credential
				}
			}
			step.Binds = filtered
			if len(step.Binds) == 0 {
				step.Binds = nil
			}
		}
		convertCredentialBinds(step.Steps, stepNames, policyCredentials, securityNames)
		for _, branch := range step.Cases {
			if branch != nil {
				convertCredentialBinds(branch.Steps, stepNames, policyCredentials, securityNames)
			}
		}
		if step.Default != nil {
			convertCredentialBinds(step.Default.Steps, stepNames, policyCredentials, securityNames)
		}
	}
}

func credentialNameForBindSource(source string, credentialNames map[string]string) string {
	source = strings.Trim(strings.TrimSpace(source), ".,")
	if source == "" {
		return ""
	}
	lower := strings.ToLower(source)
	if name := credentialNames[lower]; name != "" {
		return name
	}
	if strings.HasPrefix(lower, "security/") {
		return credentialNames[strings.TrimPrefix(lower, "security/")]
	}
	if strings.HasPrefix(lower, "credentials/") {
		return credentialNames[strings.TrimPrefix(lower, "credentials/")]
	}
	return ""
}

func intentSecurityNames(intent *rollout.Intent) []string {
	if intent == nil {
		return nil
	}
	var out []string
	for _, security := range intent.Security {
		if security == nil {
			continue
		}
		if name := strings.TrimSpace(security.Name); name != "" {
			out = append(out, name)
		}
	}
	return out
}

func renderIntentSystemPrompt() string {
	return renderIntentSystemPromptForMode(false)
}

func renderIntentSystemPromptForMode(structured bool) string {
	prompt := strings.ReplaceAll(strings.TrimSpace(intentGenerationSystemPrompt), "{{EXAMPLES}}", promptExamplesBlock())
	if structured {
		prompt = strings.ReplaceAll(prompt, "Return only JSON. Do not include Markdown.\n", "")
		prompt = strings.ReplaceAll(prompt, "Return only JSON. Do not include Markdown.", "")
	}
	return strings.TrimSpace(prompt)
}

func supportsStructuredChat(chat rollout.ChatClient) bool {
	_, ok := chat.(rollout.StructuredChat)
	return ok
}

func promptExamplesBlock() string {
	entries, err := intentPromptExamples.ReadDir("prompts/examples")
	if err != nil {
		return ""
	}
	var names []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	var b strings.Builder
	for _, name := range names {
		data, err := intentPromptExamples.ReadFile(filepath.Join("prompts/examples", name))
		if err != nil {
			continue
		}
		fmt.Fprintf(&b, "### %s\n\n```json\n%s\n```\n\n", strings.TrimSuffix(name, ".json"), strings.TrimSpace(string(data)))
	}
	return strings.TrimSpace(b.String())
}

func renderPromptSnapshot(messages []rollout.ChatMessage) string {
	var b strings.Builder
	for _, message := range messages {
		fmt.Fprintf(&b, "## %s\n\n%s\n\n", strings.ToUpper(strings.TrimSpace(message.Role)), strings.TrimSpace(message.Content))
	}
	return strings.TrimSpace(b.String())
}

func requiredByProjectPrompt(policy projectPolicy) string {
	var lines []string
	for _, input := range policy.Inputs {
		parts := []string{input.Name}
		if input.Type != "" {
			parts = append(parts, input.Type)
		}
		if input.Required {
			parts = append(parts, "required")
		}
		if input.Description != "" {
			parts = append(parts, input.Description)
		}
		lines = append(lines, "- Input: "+strings.Join(parts, ", "))
	}
	for _, output := range policy.Outputs {
		if output.From != "" {
			lines = append(lines, fmt.Sprintf("- Output: %s from %s", output.Name, output.From))
		} else if output.Description != "" {
			lines = append(lines, fmt.Sprintf("- Output: %s (%s)", output.Name, output.Description))
		}
	}
	for _, hint := range policy.BindingHints {
		lines = append(lines, fmt.Sprintf("- Step `%s` MUST receive `%s` as input `%s`.", hint.To, hint.From, hint.Field))
	}
	for _, contract := range policy.FunctionContracts {
		if contract.Name == "" {
			continue
		}
		if len(contract.Inputs) > 0 {
			lines = append(lines, fmt.Sprintf("- Function `%s` inputs: %s", contract.Name, strings.Join(contract.Inputs, ", ")))
		}
		if len(contract.Outputs) > 0 {
			lines = append(lines, fmt.Sprintf("- Function `%s` outputs: %s", contract.Name, strings.Join(contract.Outputs, ", ")))
		}
		if contract.SideEffects != "" {
			lines = append(lines, fmt.Sprintf("- Function `%s` side effects: %s", contract.Name, contract.SideEffects))
		}
	}
	if len(lines) == 0 {
		return "Required by project.md:\n- No structured requirements were extracted.\n\n"
	}
	return "Required by project.md:\n" + strings.Join(lines, "\n") + "\n\n"
}

func specSummary(candidates []openapidisco.Candidate) string {
	if len(candidates) == 0 {
		return "No OpenAPI documents are available.\n"
	}
	var b strings.Builder
	remaining := maxPromptOperationsTotal
	for _, candidate := range candidates {
		title := candidate.Title
		if title == "" {
			title = candidate.RelativePath
		}
		fmt.Fprintf(&b, "- path: %s\n  title: %s\n", candidate.RelativePath, title)
		if candidate.Description != "" {
			fmt.Fprintf(&b, "  description: %s\n", trimForPrompt(candidate.Description, maxPromptDescriptionChars))
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
		if limit > remaining {
			limit = remaining
		}
		for i := 0; i < limit; i++ {
			op := ops[i]
			fmt.Fprintf(&b, "  operation: %s %s %s", op.OperationID, op.Method, op.Path)
			if op.Summary != "" {
				fmt.Fprintf(&b, " - %s", trimForPrompt(op.Summary, maxPromptOperationSummaryChars))
			}
			b.WriteString("\n")
			required := requiredOpenAPIParams(op)
			if len(required) > 0 {
				fmt.Fprintf(&b, "    required_parameters: %s\n", strings.Join(required, ", "))
			}
			responseFields := responseFieldNames(op, maxPromptResponseFields)
			if len(responseFields) > 0 {
				fmt.Fprintf(&b, "    response_fields: %s\n", strings.Join(responseFields, ", "))
			}
		}
		remaining -= limit
		if omitted := len(ops) - limit; omitted > 0 {
			fmt.Fprintf(&b, "  omitted_operations: %d\n", omitted)
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
  "openapi": "<primary openapi path provided above>",
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
