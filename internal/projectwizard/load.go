package projectwizard

import (
	"strings"

	"github.com/genelet/ramen/internal/projectdoc"
)

func LoadAnswersFromMarkdown(text string) (Answers, error) {
	return Answers{
		ProjectName:       projectdoc.Title(text),
		Goal:              sectionAnswer(text, "Goal"),
		Inputs:            sectionAnswer(text, "Inputs"),
		Outputs:           sectionAnswer(text, "Outputs"),
		DataFlow:          sectionAnswer(text, "Data Flow"),
		FunctionContracts: sectionAnswer(text, "Function Contracts"),
		UsesOpenAPI:       !projectdoc.NoOpenAPIRequired(text),
		OpenAPI:           openAPIAnswer(text),
		CmdApproved:       runtimeApproved(text, "cmd"),
		SSHApproved:       runtimeApproved(text, "ssh"),
		Credentials:       credentialBindings(projectdoc.Section(text, "Credentials and Secrets")),
		Safety:            sectionAnswer(text, "Safety and Approval Boundary"),
		Fallback:          sectionAnswer(text, "Fallback Behavior"),
	}, nil
}

func openAPIAnswer(text string) string {
	if projectdoc.NoOpenAPIRequired(text) {
		return ""
	}
	return sectionAnswer(text, "External Systems and OpenAPI")
}

func sectionAnswer(text, heading string) string {
	section := projectdoc.Section(text, heading)
	if section == "" {
		return ""
	}
	var out []string
	for _, line := range strings.Split(section, "\n") {
		item := cleanMarkdownLine(line)
		if item == "" || isTemplateInstruction(item) || isGeneratedBoilerplate(item) {
			continue
		}
		out = append(out, item)
	}
	return strings.Join(out, "; ")
}

func cleanMarkdownLine(line string) string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "-")
	line = strings.TrimPrefix(line, "*")
	line = strings.TrimSpace(line)
	return strings.TrimSuffix(line, ".")
}

func isTemplateInstruction(line string) bool {
	lower := strings.ToLower(line)
	return strings.HasPrefix(lower, "describe ") ||
		strings.HasPrefix(lower, "list ") ||
		strings.HasPrefix(lower, "if no api/") ||
		strings.HasPrefix(lower, "if the user-level goal") ||
		strings.Contains(lower, "none declared") ||
		strings.HasPrefix(lower, "`function_name`") ||
		strings.HasPrefix(lower, "inputs: list ") ||
		strings.HasPrefix(lower, "outputs: list ") ||
		strings.HasPrefix(lower, "side effects: none")
}

func isGeneratedBoilerplate(line string) bool {
	lower := strings.ToLower(line)
	switch {
	case lower == "name credential bindings only":
		return true
	case lower == "do not include secret values":
		return true
	case lower == "generate and validate artifacts only":
		return true
	case lower == "do not directly execute production workflows":
		return true
	case strings.Contains(lower, "approved_for_sandbox"):
		return true
	case strings.Contains(lower, "approved_for_production"):
		return true
	case strings.Contains(lower, "trusted runner"):
		return true
	case strings.Contains(lower, "trusted-runner"):
		return true
	case lower == "allowed runtimes: `openapi`, `http`, `fnct`":
		return true
	case strings.Contains(lower, "`cmd` is not allowed"):
		return true
	case strings.Contains(lower, "`ssh` is not allowed"):
		return true
	default:
		return false
	}
}

func runtimeApproved(text, runtime string) bool {
	return projectdoc.RuntimeExplicitlyAllowed(projectdoc.Section(text, "Runtime Policy"), runtime)
}
