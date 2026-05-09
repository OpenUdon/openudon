package projectwizard

import (
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/OpenUdon/openudon/internal/authoring"
)

type Answers struct {
	ProjectName       string   `json:"project_name" yaml:"project_name"`
	Goal              string   `json:"goal" yaml:"goal"`
	Inputs            string   `json:"inputs" yaml:"inputs"`
	Outputs           string   `json:"outputs" yaml:"outputs"`
	DataFlow          string   `json:"data_flow" yaml:"data_flow"`
	FunctionContracts string   `json:"function_contracts" yaml:"function_contracts"`
	UsesOpenAPI       bool     `json:"uses_openapi" yaml:"uses_openapi"`
	OpenAPI           string   `json:"openapi" yaml:"openapi"`
	CmdApproved       bool     `json:"cmd_approved" yaml:"cmd_approved"`
	SSHApproved       bool     `json:"ssh_approved" yaml:"ssh_approved"`
	SideEffectScope   string   `json:"side_effect_scope" yaml:"side_effect_scope"`
	Credentials       []string `json:"credentials" yaml:"credentials"`
	Safety            string   `json:"safety" yaml:"safety"`
	Fallback          string   `json:"fallback" yaml:"fallback"`
}

func Run(in io.Reader, out io.Writer) (string, error) {
	answers, err := Prompt(in, out)
	if err != nil {
		return "", err
	}
	return Render(answers), nil
}

func Prompt(in io.Reader, out io.Writer) (Answers, error) {
	return PromptWithDefaults(in, out, Answers{})
}

func PromptWithDefaults(in io.Reader, out io.Writer, defaults Answers) (Answers, error) {
	p := authoring.NewPromptSession(in, out)
	answers := defaults
	var err error
	if answers.ProjectName, err = p.AskDefault("Project name", answers.ProjectName); err != nil {
		return answers, err
	}
	if answers.Goal, err = p.AskDefault("Goal", answers.Goal); err != nil {
		return answers, err
	}
	if answers.Inputs, err = p.AskDefault("Inputs", answers.Inputs); err != nil {
		return answers, err
	}
	if answers.Outputs, err = p.AskDefault("Outputs", answers.Outputs); err != nil {
		return answers, err
	}
	if answers.DataFlow, err = p.AskDefault("Data flow", answers.DataFlow); err != nil {
		return answers, err
	}
	if answers.FunctionContracts, err = p.AskDefault("Function contracts", answers.FunctionContracts); err != nil {
		return answers, err
	}
	if answers.UsesOpenAPI, err = p.AskYesNo("Does this project need API/OpenAPI integration?", answers.UsesOpenAPI); err != nil {
		return answers, err
	}
	if answers.UsesOpenAPI {
		if answers.OpenAPI, err = p.AskDefault("OpenAPI files, URLs, or service hints", answers.OpenAPI); err != nil {
			return answers, err
		}
	} else {
		answers.OpenAPI = ""
	}
	if answers.CmdApproved, err = p.AskYesNo("Approve cmd runtime?", answers.CmdApproved); err != nil {
		return answers, err
	}
	if answers.SSHApproved, err = p.AskYesNo("Approve ssh runtime?", answers.SSHApproved); err != nil {
		return answers, err
	}
	if answers.SideEffectScope, err = askSideEffectScope(p, out, answers.SideEffectScope); err != nil {
		return answers, err
	}
	credentialAnswer, err := p.AskDefault("Credential binding names only", strings.Join(answers.Credentials, ", "))
	if err != nil {
		return answers, err
	}
	answers.Credentials = credentialBindings(credentialAnswer)
	if answers.Safety, err = p.AskDefault("Safety and approval notes", answers.Safety); err != nil {
		return answers, err
	}
	if answers.Fallback, err = p.AskDefault("Fallback behavior", answers.Fallback); err != nil {
		return answers, err
	}
	return answers, nil
}

func Render(answers Answers) string {
	var b strings.Builder
	title := strings.TrimSpace(answers.ProjectName)
	if title == "" {
		title = "Project Name"
	}
	fmt.Fprintf(&b, "# %s\n\n", title)

	writeSection(&b, "Goal", answers.Goal, "none declared")
	writeSection(&b, "Inputs", answers.Inputs, "none declared")
	writeSection(&b, "Outputs", answers.Outputs, "none declared")
	writeSection(&b, "Data Flow", answers.DataFlow, "none declared")
	writeSection(&b, "Function Contracts", answers.FunctionContracts, "none declared")

	b.WriteString("## External Systems and OpenAPI\n\n")
	if answers.UsesOpenAPI {
		writeListOrNone(&b, answers.OpenAPI, "none declared")
	} else {
		b.WriteString("OpenAPI: none required\n")
	}
	b.WriteByte('\n')

	b.WriteString("## Runtime Policy\n\n")
	b.WriteString("- Allowed runtimes: `openapi`, `http`, `fnct`.\n")
	if answers.CmdApproved {
		b.WriteString("- `cmd` is explicitly approved for this project.\n")
	} else {
		b.WriteString("- `cmd` is not allowed unless explicitly approved here.\n")
	}
	if answers.SSHApproved {
		b.WriteString("- `ssh` is explicitly approved for this project.\n")
	} else {
		b.WriteString("- `ssh` is not allowed unless explicitly approved here.\n")
	}
	b.WriteByte('\n')

	b.WriteString("## Credentials and Secrets\n\n")
	b.WriteString("- Name credential bindings only.\n")
	b.WriteString("- Do not include secret values.\n")
	if len(answers.Credentials) == 0 {
		b.WriteString("- Credential bindings: none declared.\n")
	} else {
		for _, binding := range answers.Credentials {
			fmt.Fprintf(&b, "- Use credential binding `%s`.\n", binding)
		}
	}
	b.WriteByte('\n')

	b.WriteString("## Safety and Approval Boundary\n\n")
	writeSafetyScope(&b, answers.SideEffectScope)
	writeOptionalList(&b, answers.Safety)
	b.WriteByte('\n')

	b.WriteString("## Fallback Behavior\n\n")
	if strings.TrimSpace(answers.Fallback) == "" {
		b.WriteString("- Stop if required OpenAPI documents, runtime capabilities, or credential bindings are missing.\n")
	} else {
		writeListOrNone(&b, answers.Fallback, "none declared")
	}

	return b.String()
}

func askSideEffectScope(p *authoring.PromptSession, out io.Writer, current string) (string, error) {
	current = NormalizeSideEffectScope(current)
	if current == "" {
		current = SideEffectAfterApproval
	}
	for {
		value, err := p.AskDefault("Side-effect scope (read-only/sandbox-only/after-approval)", current)
		if err != nil {
			return "", err
		}
		value = NormalizeSideEffectScope(value)
		if value != "" {
			return value, nil
		}
		fmt.Fprintln(out, "Use read-only, sandbox-only, or after-approval.")
	}
}

func writeSection(b *strings.Builder, heading, value, empty string) {
	fmt.Fprintf(b, "## %s\n\n", heading)
	writeListOrNone(b, value, empty)
	b.WriteByte('\n')
}

func writeListOrNone(b *strings.Builder, value, empty string) {
	items := answerItems(value)
	if len(items) == 0 {
		fmt.Fprintf(b, "- %s.\n", strings.TrimSuffix(empty, "."))
		return
	}
	for _, item := range items {
		fmt.Fprintf(b, "- %s\n", item)
	}
}

func writeOptionalList(b *strings.Builder, value string) {
	for _, item := range answerItems(value) {
		fmt.Fprintf(b, "- %s\n", item)
	}
}

func answerItems(value string) []string {
	var out []string
	for _, item := range splitAnswer(value) {
		item = strings.TrimSpace(item)
		item = strings.TrimPrefix(item, "-")
		item = strings.TrimSpace(item)
		if item != "" {
			out = append(out, strings.TrimSuffix(item, ".")+".")
		}
	}
	return out
}

const (
	SideEffectReadOnly      = "read-only"
	SideEffectSandboxOnly   = "sandbox-only"
	SideEffectAfterApproval = "after-approval"
)

func NormalizeSideEffectScope(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "_", "-")
	switch value {
	case "read", "readonly", "read-only", "no-side-effects", "none":
		return SideEffectReadOnly
	case "sandbox", "sandbox-only", "sandboxed", "test-only":
		return SideEffectSandboxOnly
	case "", "approval", "after-approval", "approved", "production", "production-after-approval":
		if value == "" {
			return ""
		}
		return SideEffectAfterApproval
	default:
		return ""
	}
}

func InferSideEffectScope(safety string) string {
	lower := strings.ToLower(safety)
	switch {
	case strings.Contains(lower, "read-only") || strings.Contains(lower, "no side effects") || strings.Contains(lower, "do not execute workflows"):
		return SideEffectReadOnly
	case strings.Contains(lower, "production is not approved") || strings.Contains(lower, "sandbox-only") || strings.Contains(lower, "sandbox only"):
		return SideEffectSandboxOnly
	case strings.Contains(lower, "approved_for_production") || strings.Contains(lower, "production execution") || strings.Contains(lower, "after approval"):
		return SideEffectAfterApproval
	case strings.Contains(lower, "approved_for_sandbox") || strings.Contains(lower, "sandbox proof"):
		return SideEffectSandboxOnly
	default:
		return SideEffectAfterApproval
	}
}

func writeSafetyScope(b *strings.Builder, scope string) {
	switch NormalizeSideEffectScope(scope) {
	case SideEffectReadOnly:
		b.WriteString("- Generate and validate artifacts only.\n")
		b.WriteString("- Do not execute workflows, call external systems, write remote state, or perform other side effects.\n")
	case SideEffectSandboxOnly:
		b.WriteString("- Generate and validate artifacts only.\n")
		b.WriteString("- Sandbox proof runs require Symphony state `approved_for_sandbox`, approved credential bindings, and a trusted runner.\n")
		b.WriteString("- Production execution is not approved by this project contract.\n")
	case SideEffectAfterApproval:
		fallthrough
	default:
		b.WriteString("- Generate and validate artifacts only.\n")
		b.WriteString("- Do not directly execute production workflows.\n")
		b.WriteString("- Sandbox proof runs require Symphony state `approved_for_sandbox`.\n")
		b.WriteString("- Production execution requires Symphony state `approved_for_production`.\n")
		b.WriteString("- Side-effectful execution requires explicit approval, approved credential bindings, and a trusted runner.\n")
		b.WriteString("- Trusted runner required for approved sandbox or production execution.\n")
	}
}

func splitAnswer(value string) []string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	fields := strings.FieldsFunc(value, func(r rune) bool {
		return r == '\n' || r == ';'
	})
	if len(fields) > 1 {
		return fields
	}
	return []string{value}
}

var bindingTokenRE = regexp.MustCompile(`[A-Za-z][A-Za-z0-9_.-]*`)

func credentialBindings(value string) []string {
	var out []string
	seen := map[string]bool{}
	for _, item := range splitCredentialAnswer(value) {
		binding := credentialBinding(item)
		if binding == "" || seen[binding] {
			continue
		}
		seen[binding] = true
		out = append(out, binding)
	}
	return out
}

func splitCredentialAnswer(value string) []string {
	return strings.FieldsFunc(value, func(r rune) bool {
		return r == '\n' || r == ',' || r == ';'
	})
}

func credentialBinding(value string) string {
	value = strings.TrimSpace(strings.Trim(value, "`'\""))
	if value == "" {
		return ""
	}
	if idx := strings.Index(value, "="); idx >= 0 {
		value = value[:idx]
	} else if idx := strings.Index(value, ":"); idx >= 0 {
		left := strings.TrimSpace(value[:idx])
		right := strings.TrimSpace(value[idx+1:])
		if looksLikeBindingName(right) {
			value = right
		} else {
			value = left
		}
	}
	matches := bindingTokenRE.FindAllString(value, -1)
	var candidate string
	for _, match := range matches {
		lower := strings.ToLower(match)
		if ignoredCredentialToken(lower) {
			continue
		}
		candidate = match
	}
	return candidate
}

func ignoredCredentialToken(value string) bool {
	value = strings.Trim(value, ".,:;")
	switch value {
	case "use", "uses", "binding", "bindings", "credential", "credentials", "secret", "secrets", "value", "values", "name", "names", "only", "none", "declared", "do", "not", "include", "required", "no", "are":
		return true
	default:
		return false
	}
}

func looksLikeBindingName(value string) bool {
	value = strings.TrimSpace(strings.Trim(value, "`'\""))
	return bindingTokenRE.FindString(value) == value && !strings.ContainsAny(value, " \t")
}
