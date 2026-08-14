package projectdoc

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var noOpenAPIRequiredRE = regexp.MustCompile(`(?im)^\s*(?:[-*]\s*)?openapi\s*:\s*none\s+required\s*$`)

const CandidateWorkflowsHeading = "Candidate Workflows"

// CandidateWorkflow is one intentionally unexpanded future workflow. It is
// project direction, not executable workflow intent.
type CandidateWorkflow struct {
	Title            string `json:"title" yaml:"title"`
	Outcome          string `json:"outcome" yaml:"outcome"`
	DeferralReason   string `json:"deferral_reason" yaml:"deferral_reason"`
	PromotionTrigger string `json:"promotion_trigger" yaml:"promotion_trigger"`
}

// RenderCandidateWorkflows renders a deterministic, explicitly
// non-executable project.md section.
func RenderCandidateWorkflows(candidates []CandidateWorkflow) string {
	candidates = NormalizeCandidateWorkflows(candidates)
	if len(candidates) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("## " + CandidateWorkflowsHeading + "\n\n")
	b.WriteString("<!-- openudon:candidate-workflows:v1; non-executable -->\n\n")
	for _, candidate := range candidates {
		fmt.Fprintf(&b, "- **%s**\n", candidate.Title)
		fmt.Fprintf(&b, "  - Outcome: %s\n", candidate.Outcome)
		fmt.Fprintf(&b, "  - Deferral reason: %s\n", candidate.DeferralReason)
		fmt.Fprintf(&b, "  - Promotion trigger: %s\n", candidate.PromotionTrigger)
	}
	b.WriteByte('\n')
	return b.String()
}

// ParseCandidateWorkflows parses the generated candidate-workflow section and
// rejects partial or hand-shaped entries so lint can surface drift.
func ParseCandidateWorkflows(text string) ([]CandidateWorkflow, error) {
	section := Section(text, CandidateWorkflowsHeading)
	if strings.TrimSpace(section) == "" {
		return nil, nil
	}
	if !strings.Contains(section, "openudon:candidate-workflows:v1") {
		return nil, fmt.Errorf("%s section is missing the v1 non-executable marker", CandidateWorkflowsHeading)
	}
	var out []CandidateWorkflow
	var current *CandidateWorkflow
	flush := func() error {
		if current == nil {
			return nil
		}
		if current.Title == "" || current.Outcome == "" || current.DeferralReason == "" || current.PromotionTrigger == "" {
			return fmt.Errorf("candidate workflow %q must include title, outcome, deferral reason, and promotion trigger", current.Title)
		}
		out = append(out, *current)
		current = nil
		return nil
	}
	for _, line := range strings.Split(section, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "<!--") {
			continue
		}
		if strings.HasPrefix(line, "- **") && strings.HasSuffix(line, "**") {
			if err := flush(); err != nil {
				return nil, err
			}
			current = &CandidateWorkflow{Title: strings.TrimSuffix(strings.TrimPrefix(line, "- **"), "**")}
			continue
		}
		if current == nil {
			return nil, fmt.Errorf("malformed %s entry %q", CandidateWorkflowsHeading, line)
		}
		switch {
		case strings.HasPrefix(line, "- Outcome:"):
			current.Outcome = strings.TrimSpace(strings.TrimPrefix(line, "- Outcome:"))
		case strings.HasPrefix(line, "- Deferral reason:"):
			current.DeferralReason = strings.TrimSpace(strings.TrimPrefix(line, "- Deferral reason:"))
		case strings.HasPrefix(line, "- Promotion trigger:"):
			current.PromotionTrigger = strings.TrimSpace(strings.TrimPrefix(line, "- Promotion trigger:"))
		default:
			return nil, fmt.Errorf("malformed %s field %q", CandidateWorkflowsHeading, line)
		}
	}
	if err := flush(); err != nil {
		return nil, err
	}
	return NormalizeCandidateWorkflows(out), nil
}

// NormalizeCandidateWorkflows trims, deduplicates, and orders candidates.
func NormalizeCandidateWorkflows(candidates []CandidateWorkflow) []CandidateWorkflow {
	byKey := map[string]CandidateWorkflow{}
	for _, candidate := range candidates {
		candidate.Title = strings.TrimSpace(candidate.Title)
		candidate.Outcome = strings.TrimSpace(candidate.Outcome)
		candidate.DeferralReason = strings.TrimSpace(candidate.DeferralReason)
		candidate.PromotionTrigger = strings.TrimSpace(candidate.PromotionTrigger)
		if candidate.Title == "" && candidate.Outcome == "" {
			continue
		}
		key := strings.ToLower(candidate.Title + "\x00" + candidate.Outcome)
		byKey[key] = candidate
	}
	out := make([]CandidateWorkflow, 0, len(byKey))
	for _, candidate := range byKey {
		out = append(out, candidate)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Title != out[j].Title {
			return out[i].Title < out[j].Title
		}
		return out[i].Outcome < out[j].Outcome
	})
	return out
}

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
