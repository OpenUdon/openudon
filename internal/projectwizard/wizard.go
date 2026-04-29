package projectwizard

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strings"
)

type Answers struct {
	ProjectName       string
	Goal              string
	Inputs            string
	Outputs           string
	DataFlow          string
	FunctionContracts string
	UsesOpenAPI       bool
	OpenAPI           string
	CmdApproved       bool
	SSHApproved       bool
	Credentials       []string
	Safety            string
	Fallback          string
}

func Run(in io.Reader, out io.Writer) (string, error) {
	answers, err := Prompt(in, out)
	if err != nil {
		return "", err
	}
	return Render(answers), nil
}

func Prompt(in io.Reader, out io.Writer) (Answers, error) {
	p := prompter{
		scanner: bufio.NewScanner(in),
		out:     out,
	}
	var answers Answers
	var err error
	if answers.ProjectName, err = p.ask("Project name"); err != nil {
		return answers, err
	}
	if answers.Goal, err = p.ask("Goal"); err != nil {
		return answers, err
	}
	if answers.Inputs, err = p.ask("Inputs"); err != nil {
		return answers, err
	}
	if answers.Outputs, err = p.ask("Outputs"); err != nil {
		return answers, err
	}
	if answers.DataFlow, err = p.ask("Data flow"); err != nil {
		return answers, err
	}
	if answers.FunctionContracts, err = p.ask("Function contracts"); err != nil {
		return answers, err
	}
	if answers.UsesOpenAPI, err = p.askYesNo("Does this project need API/OpenAPI integration?", false); err != nil {
		return answers, err
	}
	if answers.UsesOpenAPI {
		if answers.OpenAPI, err = p.ask("OpenAPI files, URLs, or service hints"); err != nil {
			return answers, err
		}
	}
	if answers.CmdApproved, err = p.askYesNo("Approve cmd runtime?", false); err != nil {
		return answers, err
	}
	if answers.SSHApproved, err = p.askYesNo("Approve ssh runtime?", false); err != nil {
		return answers, err
	}
	credentialAnswer, err := p.ask("Credential binding names only")
	if err != nil {
		return answers, err
	}
	answers.Credentials = credentialBindings(credentialAnswer)
	if answers.Safety, err = p.ask("Safety and approval notes"); err != nil {
		return answers, err
	}
	if answers.Fallback, err = p.ask("Fallback behavior"); err != nil {
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
	b.WriteString("- Generate and validate artifacts only.\n")
	b.WriteString("- Do not directly execute production workflows.\n")
	b.WriteString("- Sandbox proof runs require Symphony state `approved_for_sandbox`.\n")
	b.WriteString("- Production execution requires Symphony state `approved_for_production`.\n")
	b.WriteString("- Side-effectful execution requires explicit approval, approved credential bindings, and a trusted runner.\n")
	b.WriteString("- Trusted runner required for approved sandbox or production execution.\n")
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

type prompter struct {
	scanner *bufio.Scanner
	out     io.Writer
}

func (p *prompter) ask(label string) (string, error) {
	fmt.Fprintf(p.out, "%s: ", label)
	return p.next()
}

func (p *prompter) askYesNo(label string, defaultYes bool) (bool, error) {
	defaultText := "y/N"
	if defaultYes {
		defaultText = "Y/n"
	}
	fmt.Fprintf(p.out, "%s [%s]: ", label, defaultText)
	value, err := p.next()
	if err != nil {
		return false, err
	}
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return defaultYes, nil
	}
	return value == "y" || value == "yes" || value == "true" || value == "approved" || value == "allow" || value == "allowed", nil
}

func (p *prompter) next() (string, error) {
	if !p.scanner.Scan() {
		if err := p.scanner.Err(); err != nil {
			return "", err
		}
		return "", io.ErrUnexpectedEOF
	}
	return strings.TrimSpace(p.scanner.Text()), nil
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
		if lower == "use" || lower == "uses" || lower == "binding" || lower == "bindings" || lower == "credential" || lower == "credentials" || lower == "secret" || lower == "secrets" {
			continue
		}
		candidate = match
	}
	return candidate
}

func looksLikeBindingName(value string) bool {
	value = strings.TrimSpace(strings.Trim(value, "`'\""))
	return bindingTokenRE.FindString(value) == value && !strings.ContainsAny(value, " \t")
}
