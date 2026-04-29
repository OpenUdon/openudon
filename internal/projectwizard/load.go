package projectwizard

import (
	"regexp"
	"strings"
)

func LoadAnswersFromMarkdown(text string) (Answers, error) {
	return Answers{
		ProjectName:       markdownTitle(text),
		Goal:              sectionAnswer(text, "Goal"),
		Inputs:            sectionAnswer(text, "Inputs"),
		Outputs:           sectionAnswer(text, "Outputs"),
		DataFlow:          sectionAnswer(text, "Data Flow"),
		FunctionContracts: sectionAnswer(text, "Function Contracts"),
		UsesOpenAPI:       !noOpenAPIRequired(text),
		OpenAPI:           openAPIAnswer(text),
		CmdApproved:       runtimeApproved(text, "cmd"),
		SSHApproved:       runtimeApproved(text, "ssh"),
		Credentials:       credentialBindings(markdownSection(text, "Credentials and Secrets")),
		Safety:            sectionAnswer(text, "Safety and Approval Boundary"),
		Fallback:          sectionAnswer(text, "Fallback Behavior"),
	}, nil
}

func markdownTitle(text string) string {
	for _, line := range strings.Split(text, "\n") {
		level, title, ok := parseHeading(line)
		if ok && level == 1 {
			return title
		}
	}
	return ""
}

func openAPIAnswer(text string) string {
	if noOpenAPIRequired(text) {
		return ""
	}
	return sectionAnswer(text, "External Systems and OpenAPI")
}

func sectionAnswer(text, heading string) string {
	section := markdownSection(text, heading)
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

var noOpenAPIRequiredRE = regexp.MustCompile(`(?im)^\s*(?:[-*]\s*)?openapi\s*:\s*none\s+required\s*$`)

func noOpenAPIRequired(text string) bool {
	return noOpenAPIRequiredRE.MatchString(text)
}

func runtimeApproved(text, runtime string) bool {
	section := strings.ToLower(markdownSection(text, "Runtime Policy"))
	runtime = strings.ToLower(runtime)
	approved := "`" + runtime + "`"
	if !strings.Contains(section, approved) {
		return false
	}
	if strings.Contains(section, approved+" is explicitly approved") ||
		strings.Contains(section, approved+" is explicitly allowed") {
		return true
	}
	if regexp.MustCompile("`" + regexp.QuoteMeta(runtime) + "`[^.\n]*not allowed").MatchString(section) {
		return false
	}
	return regexp.MustCompile("`" + regexp.QuoteMeta(runtime) + "`[^.\n]*(approved|allowed)").MatchString(section)
}

func markdownSection(text, heading string) string {
	lines := strings.Split(text, "\n")
	target := normalizeHeading(heading)
	start := -1
	level := 0
	for i, line := range lines {
		lvl, title, ok := parseHeading(line)
		if !ok {
			continue
		}
		if start == -1 {
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
	if start == -1 {
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
	return strings.Join(strings.Fields(strings.ToLower(value)), " ")
}
