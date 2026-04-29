package projectdoc

import (
	"regexp"
	"strings"
)

var noOpenAPIRequiredRE = regexp.MustCompile(`(?im)^\s*(?:[-*]\s*)?openapi\s*:\s*none\s+required\s*$`)

func Title(text string) string {
	for _, line := range strings.Split(text, "\n") {
		level, title, ok := ParseHeading(line)
		if ok && level == 1 {
			return title
		}
	}
	return ""
}

func Section(text, heading string) string {
	lines := strings.Split(text, "\n")
	target := NormalizeHeading(heading)
	start := -1
	level := 0
	for i, line := range lines {
		lvl, title, ok := ParseHeading(line)
		if !ok {
			continue
		}
		if start == -1 {
			if NormalizeHeading(title) == target {
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

func HasSection(text, heading string) bool {
	return Section(text, heading) != ""
}

func ParseHeading(line string) (int, string, bool) {
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

func NormalizeHeading(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "&", "and")
	return strings.Join(strings.Fields(value), " ")
}

func NoOpenAPIRequired(text string) bool {
	return noOpenAPIRequiredRE.MatchString(text)
}

func RuntimeExplicitlyAllowed(section, runtime string) bool {
	runtime = strings.ToLower(strings.TrimSpace(runtime))
	for _, line := range strings.Split(section, "\n") {
		lower := strings.ToLower(line)
		if !ContainsRuntimeToken(lower, runtime) {
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

func ContainsRuntimeToken(line, runtime string) bool {
	runtime = strings.ToLower(strings.TrimSpace(runtime))
	for _, field := range strings.FieldsFunc(strings.ToLower(line), func(r rune) bool {
		return !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9')
	}) {
		if field == runtime {
			return true
		}
	}
	return false
}
