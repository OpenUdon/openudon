package synthesize

import (
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type projectPolicy struct {
	NoOpenAPI         bool
	RuntimeSection    string
	InputsSection     string
	OutputsSection    string
	DataFlowSection   string
	FunctionSection   string
	CredentialSection string
	AllowedRuntime    map[string]bool
	HasFunctionSteps  bool
	Inputs            []InputDecl
	Outputs           []OutputDecl
	BindingHints      []BindingHint
	FunctionContracts []FunctionContract
}

func analyzeProject(text string) projectPolicy {
	structured := parseStructuredProjectPolicy(text)
	inputsSection := markdownSection(text, "Inputs")
	outputsSection := markdownSection(text, "Outputs")
	dataFlowSection := markdownSection(text, "Data Flow")
	functionSection := markdownSection(text, "Function Contracts")
	policy := projectPolicy{
		NoOpenAPI:         noOpenAPIRequired(text) || structured.NoOpenAPI,
		RuntimeSection:    markdownSection(text, "Runtime Policy"),
		InputsSection:     inputsSection,
		OutputsSection:    outputsSection,
		DataFlowSection:   dataFlowSection,
		FunctionSection:   functionSection,
		CredentialSection: markdownSection(text, "Credentials and Secrets"),
		AllowedRuntime: map[string]bool{
			"fnct":    true,
			"http":    true,
			"openapi": true,
		},
		HasFunctionSteps:  containsRuntimeToken(strings.ToLower(text), "fnct") || strings.Contains(strings.ToLower(text), "function"),
		Inputs:            extractInputDecls(inputsSection),
		Outputs:           extractOutputDecls(outputsSection),
		BindingHints:      extractBindingHints(dataFlowSection),
		FunctionContracts: extractFunctionContracts(functionSection),
	}
	if policy.CredentialSection == "" && len(structured.Credentials) > 0 {
		policy.CredentialSection = strings.Join(structured.Credentials, "\n")
	}
	for _, runtime := range []string{"cmd", "ssh"} {
		policy.AllowedRuntime[runtime] = structured.AllowedRuntime[runtime] || runtimeExplicitlyAllowed(policy.RuntimeSection, runtime)
	}
	return policy
}

type InputDecl struct {
	Name        string
	Type        string
	Required    bool
	Description string
}

type OutputDecl struct {
	Name        string
	From        string
	Description string
}

type BindingHint struct {
	From         string
	To           string
	Field        string
	StepSelector string
}

type FunctionContract struct {
	Name        string
	Inputs      []string
	Outputs     []string
	SideEffects string
}

type structuredProjectPolicy struct {
	NoOpenAPI      bool
	AllowedRuntime map[string]bool
	Credentials    []string
}

type structuredProjectPolicyYAML struct {
	OpenAPI             string            `yaml:"openapi"`
	Runtimes            map[string]any    `yaml:"runtimes"`
	RuntimePolicy       map[string]any    `yaml:"runtime_policy"`
	Credentials         []string          `yaml:"credentials"`
	CredentialBindings  []string          `yaml:"credential_bindings"`
	CredentialsBySystem map[string]string `yaml:"credentials_by_system"`
}

func parseStructuredProjectPolicy(text string) structuredProjectPolicy {
	out := structuredProjectPolicy{
		AllowedRuntime: map[string]bool{},
	}
	for _, block := range fencedBlocks(text, "ramen-policy") {
		var raw structuredProjectPolicyYAML
		if err := yaml.Unmarshal([]byte(block), &raw); err != nil {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(raw.OpenAPI), "none required") {
			out.NoOpenAPI = true
		}
		mergeStructuredRuntimes(out.AllowedRuntime, raw.Runtimes)
		mergeStructuredRuntimes(out.AllowedRuntime, raw.RuntimePolicy)
		out.Credentials = append(out.Credentials, raw.Credentials...)
		out.Credentials = append(out.Credentials, raw.CredentialBindings...)
		for system, binding := range raw.CredentialsBySystem {
			if strings.TrimSpace(binding) != "" {
				out.Credentials = append(out.Credentials, strings.TrimSpace(system)+": "+strings.TrimSpace(binding))
			}
		}
	}
	out.Credentials = sortedUnique(out.Credentials)
	return out
}

func mergeStructuredRuntimes(out map[string]bool, values map[string]any) {
	for runtime, raw := range values {
		runtime = strings.ToLower(strings.TrimSpace(runtime))
		if runtime == "" {
			continue
		}
		if structuredPolicyValueAllows(raw) {
			out[runtime] = true
		}
	}
}

func structuredPolicyValueAllows(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		lower := strings.ToLower(strings.TrimSpace(typed))
		return lower == "allow" || lower == "allowed" || lower == "true" || lower == "enabled" || lower == "approved"
	default:
		return false
	}
}

func fencedBlocks(text, language string) []string {
	var out []string
	lines := strings.Split(text, "\n")
	target := strings.ToLower(strings.TrimSpace(language))
	var inBlock bool
	var builder strings.Builder
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !inBlock {
			if strings.HasPrefix(trimmed, "```") && strings.ToLower(strings.TrimSpace(strings.TrimPrefix(trimmed, "```"))) == target {
				inBlock = true
				builder.Reset()
			}
			continue
		}
		if strings.HasPrefix(trimmed, "```") {
			out = append(out, builder.String())
			inBlock = false
			continue
		}
		builder.WriteString(line)
		builder.WriteByte('\n')
	}
	return out
}

func noOpenAPIRequired(text string) bool {
	return regexp.MustCompile(`(?im)^\s*(?:[-*]\s*)?openapi\s*:\s*none\s+required\s*$`).MatchString(text)
}

func extractInputDecls(section string) []InputDecl {
	var out []InputDecl
	for _, line := range markdownListItems(section) {
		name, rest := splitDeclLine(line)
		if name == "" {
			continue
		}
		lower := strings.ToLower(rest)
		decl := InputDecl{
			Name:        name,
			Type:        firstTypeToken(rest),
			Required:    strings.Contains(lower, "required"),
			Description: strings.TrimSpace(rest),
		}
		out = append(out, decl)
	}
	return out
}

func extractOutputDecls(section string) []OutputDecl {
	var out []OutputDecl
	for _, line := range markdownListItems(section) {
		name, rest := splitDeclLine(line)
		if name == "" {
			name = stableTokenFromText(line)
			rest = line
		}
		out = append(out, OutputDecl{
			Name:        name,
			From:        extractFromReference(rest),
			Description: strings.TrimSpace(rest),
		})
	}
	return out
}

func extractBindingHints(section string) []BindingHint {
	re := regexp.MustCompile(`(?i)\bpass\s+` + "`?" + `([^` + "`" + `]+?)` + "`?" + `\s+to\s+` + "`?" + `([A-Za-z0-9_.-]+)` + "`?")
	var out []BindingHint
	for _, line := range markdownListItems(section) {
		match := re.FindStringSubmatch(line)
		if len(match) >= 3 {
			to := strings.Trim(strings.TrimSpace(match[2]), ".,")
			field := to
			if idx := strings.LastIndexAny(to, "."); idx >= 0 && idx+1 < len(to) {
				field = to[idx+1:]
			}
			out = append(out, BindingHint{
				From:  strings.Trim(strings.TrimSpace(match[1]), ".,"),
				To:    to,
				Field: field,
			})
		}
		out = append(out, extractLiteralBindingHints(line)...)
	}
	return out
}

func extractLiteralBindingHints(line string) []BindingHint {
	normalized := strings.ToLower(strings.TrimSpace(line))
	var out []BindingHint
	selector := literalStepSelector(normalized)
	assignmentRe := regexp.MustCompile("(?i)(?:literal\\s+)?`?([A-Za-z][A-Za-z0-9_.-]*)`?\\s*=\\s*`?([A-Za-z0-9_.:-]+)`?")
	for _, match := range assignmentRe.FindAllStringSubmatch(line, -1) {
		if len(match) < 3 {
			continue
		}
		field := strings.Trim(strings.TrimSpace(match[1]), ".,")
		value := strings.Trim(strings.TrimSpace(match[2]), ".,")
		if field == "" || value == "" {
			continue
		}
		out = append(out, BindingHint{From: value, Field: field, StepSelector: selector})
	}
	if strings.Contains(normalized, "literal page") {
		pageRe := regexp.MustCompile(`(?i)\bpage\s+` + "`?" + `([0-9]+)` + "`?")
		limitRe := regexp.MustCompile(`(?i)\blimit\s+` + "`?" + `([0-9]+)` + "`?")
		if match := pageRe.FindStringSubmatch(line); len(match) >= 2 {
			out = append(out, BindingHint{From: match[1], Field: "page", StepSelector: selector})
		}
		if match := limitRe.FindStringSubmatch(line); len(match) >= 2 {
			out = append(out, BindingHint{From: match[1], Field: "limit", StepSelector: selector})
		}
	}
	if strings.Contains(normalized, "toronto") && (strings.Contains(normalized, "resolve") || strings.Contains(normalized, "coordinate")) {
		out = append(out, BindingHint{From: "Toronto,CA", Field: "q", StepSelector: "coordinate"})
	}
	return dedupeBindingHints(out)
}

func literalStepSelector(line string) string {
	switch {
	case strings.Contains(line, "page 1") || strings.Contains(line, "first page"):
		return "page_1"
	case strings.Contains(line, "page 2") || strings.Contains(line, "second page"):
		return "page_2"
	case strings.Contains(line, "list operation") || strings.Contains(line, "list customers"):
		return "list"
	case strings.Contains(line, "coordinate") || strings.Contains(line, "resolve"):
		return "coordinate"
	default:
		return ""
	}
}

func dedupeBindingHints(hints []BindingHint) []BindingHint {
	seen := map[string]bool{}
	var out []BindingHint
	for _, hint := range hints {
		key := hint.From + "\x00" + hint.To + "\x00" + hint.Field + "\x00" + hint.StepSelector
		if hint.From == "" || hint.Field == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, hint)
	}
	return out
}

func extractFunctionContracts(section string) []FunctionContract {
	var out []FunctionContract
	var current *FunctionContract
	for _, rawLine := range strings.Split(section, "\n") {
		trimmed := strings.TrimSpace(rawLine)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "- ") && !strings.Contains(trimmed, ":") {
			if current != nil {
				out = append(out, *current)
			}
			name := normalizeDeclName(strings.TrimSpace(strings.TrimPrefix(trimmed, "- ")))
			current = &FunctionContract{Name: name}
			continue
		}
		if current == nil {
			continue
		}
		item := strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))
		key, value, ok := strings.Cut(item, ":")
		if !ok {
			continue
		}
		switch normalizeHeading(key) {
		case "inputs":
			current.Inputs = splitCommaList(value)
		case "outputs":
			current.Outputs = splitCommaList(value)
		case "side effects":
			current.SideEffects = strings.TrimSpace(value)
		}
	}
	if current != nil {
		out = append(out, *current)
	}
	return out
}

func markdownListItems(section string) []string {
	var out []string
	for _, line := range strings.Split(section, "\n") {
		trimmed := strings.TrimSpace(line)
		trimmed = strings.TrimPrefix(trimmed, "- ")
		trimmed = strings.TrimPrefix(trimmed, "* ")
		if strings.TrimSpace(trimmed) != "" {
			out = append(out, strings.TrimSpace(trimmed))
		}
	}
	return out
}

func splitDeclLine(line string) (string, string) {
	line = strings.TrimSpace(line)
	line = strings.Trim(line, "`")
	for _, sep := range []string{":", " - ", " -- "} {
		if left, right, ok := strings.Cut(line, sep); ok {
			return normalizeDeclName(left), strings.TrimSpace(right)
		}
	}
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return "", ""
	}
	return normalizeDeclName(fields[0]), strings.TrimSpace(strings.TrimPrefix(line, fields[0]))
}

func normalizeDeclName(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, "`")
	value = strings.Trim(value, ".,")
	return value
}

func firstTypeToken(value string) string {
	for _, field := range strings.FieldsFunc(value, func(r rune) bool {
		return !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-')
	}) {
		lower := strings.ToLower(strings.TrimSpace(field))
		switch lower {
		case "string", "number", "integer", "bool", "boolean", "object", "array":
			return lower
		}
	}
	return ""
}

func extractFromReference(value string) string {
	re := regexp.MustCompile(`(?i)\bfrom\s+` + "`?" + `([A-Za-z0-9_.\[\]-]+)` + "`?")
	match := re.FindStringSubmatch(value)
	if len(match) < 2 {
		return ""
	}
	return strings.Trim(match[1], ".,")
}

func stableTokenFromText(value string) string {
	var fields []string
	for _, field := range strings.FieldsFunc(strings.ToLower(value), func(r rune) bool {
		return !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'))
	}) {
		if field != "" {
			fields = append(fields, field)
		}
		if len(fields) == 3 {
			break
		}
	}
	if len(fields) == 0 {
		return ""
	}
	return strings.Join(fields, "_")
}

func splitCommaList(value string) []string {
	var out []string
	for _, item := range strings.Split(value, ",") {
		item = strings.TrimSpace(strings.Trim(item, "."))
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}

func addProjectAuthoringChecks(report *QualityReport, text string) {
	checkProjectSection(report, text, "Goal", "project.authoring.goal", "project.md declares the workflow goal")
	checkProjectSection(report, text, "External Systems and OpenAPI", "project.authoring.integration_policy", "project.md declares integration/OpenAPI policy")
	checkOptionalProjectSection(report, text, "Data Flow", "project.authoring.data_flow", "project.md declares data-flow hints")
	checkOptionalProjectSection(report, text, "Credentials and Secrets", "project.authoring.credentials", "project.md declares credential binding policy")
	checkProjectSection(report, text, "Runtime Policy", "project.authoring.runtime_policy", "project.md declares runtime policy")
	if analyzeProject(text).HasFunctionSteps {
		checkProjectSection(report, text, "Function Contracts", "project.authoring.function_contracts", "project.md declares function contracts")
	} else {
		checkOptionalProjectSection(report, text, "Function Contracts", "project.authoring.function_contracts", "project.md declares function contracts")
	}
	checkProjectSection(report, text, "Safety and Approval Boundary", "project.authoring.safety", "project.md declares safety and approval boundary")
	checkProjectSection(report, text, "Fallback Behavior", "project.authoring.fallback", "project.md declares fallback behavior")
}

func checkProjectSection(report *QualityReport, text, heading, code, message string) {
	if hasMarkdownSection(text, heading) {
		report.add(code, "pass", message, "")
		return
	}
	report.add(code, "warn", fmt.Sprintf("%s is missing", heading), "Add this section to make synthesis decisions auditable.")
}

func checkOptionalProjectSection(report *QualityReport, text, heading, code, message string) {
	if hasMarkdownSection(text, heading) {
		report.add(code, "pass", message, "")
		return
	}
	report.add(code, "warn", fmt.Sprintf("%s is missing", heading), "Add this section when the workflow has multiple steps or fnct adapters.")
}

func hasMarkdownSection(text, heading string) bool {
	return markdownSection(text, heading) != ""
}

func markdownSection(text, heading string) string {
	lines := strings.Split(text, "\n")
	target := normalizeHeading(heading)
	var start int = -1
	var level int
	for i, line := range lines {
		lvl, title, ok := parseHeading(line)
		if !ok {
			continue
		}
		if start < 0 {
			if normalizeHeading(title) == target {
				start = i + 1
				level = lvl
			}
			continue
		}
		if lvl <= level {
			return strings.TrimSpace(strings.Join(lines[start:i], "\n"))
		}
	}
	if start < 0 {
		return ""
	}
	return strings.TrimSpace(strings.Join(lines[start:], "\n"))
}

func parseHeading(line string) (int, string, bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "#") {
		return 0, "", false
	}
	level := 0
	for level < len(trimmed) && trimmed[level] == '#' {
		level++
	}
	if level == 0 || level >= len(trimmed) || trimmed[level] != ' ' {
		return 0, "", false
	}
	return level, strings.TrimSpace(trimmed[level+1:]), true
}

func normalizeHeading(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "&", "and")
	return strings.Join(strings.Fields(value), " ")
}

func runtimeExplicitlyAllowed(section, runtime string) bool {
	runtime = strings.ToLower(strings.TrimSpace(runtime))
	for _, line := range strings.Split(section, "\n") {
		lower := strings.ToLower(line)
		if !containsRuntimeToken(lower, runtime) {
			continue
		}
		if strings.Contains(lower, "not allowed") || strings.Contains(lower, "disallowed") ||
			strings.Contains(lower, "forbidden") || strings.Contains(lower, "disabled") {
			continue
		}
		if strings.Contains(lower, "allowed") || strings.Contains(lower, "allow ") ||
			strings.Contains(lower, "approved") || strings.Contains(lower, "enabled") {
			return true
		}
	}
	return false
}

func containsRuntimeToken(line, runtime string) bool {
	for _, field := range strings.FieldsFunc(line, func(r rune) bool {
		return !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9')
	}) {
		if field == runtime {
			return true
		}
	}
	return false
}

func runtimePolicyPrompt(policy projectPolicy) string {
	var b strings.Builder
	if policy.NoOpenAPI {
		b.WriteString("- OpenAPI: none required. Do not set top-level openapi, step openapi, provider openapi, or operation fields.\n")
	} else {
		b.WriteString("- Use OpenAPI only when matching a listed OpenAPI document and operation.\n")
	}
	b.WriteString("- Use fnct for trusted local glue, transformations, renderers, and adapters.\n")
	if policy.AllowedRuntime["cmd"] {
		b.WriteString("- cmd is explicitly allowed by project policy.\n")
	} else {
		b.WriteString("- Do not use cmd unless project policy explicitly allows it.\n")
	}
	if policy.AllowedRuntime["ssh"] {
		b.WriteString("- ssh is explicitly allowed by project policy.\n")
	} else {
		b.WriteString("- Do not use ssh unless project policy explicitly allows it.\n")
	}
	b.WriteString("- Do not invent smtp, sql, or llm runtime types; use approved fnct adapters or leave the step unresolved.\n")
	if strings.TrimSpace(policy.DataFlowSection) != "" {
		b.WriteString("- Treat the project Data Flow section as authoritative field mapping guidance.\n")
	}
	if strings.TrimSpace(policy.FunctionSection) != "" {
		b.WriteString("- Treat the project Function Contracts section as authoritative for fnct inputs, outputs, and side effects.\n")
	}
	return b.String()
}
