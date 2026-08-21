// Package browserworkflow owns document-order browser session analysis shared
// by authoring, quality, and trusted execution.
package browserworkflow

import (
	"sort"
	"strings"

	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

// RuntimeOperationID mirrors the stable identifier lowering used for UWS
// operation IDs. Review evidence names authoring steps; trusted executor
// approvals must name the lowered operation actually seen by Udon.
func RuntimeOperationID(value string) string {
	var out strings.Builder
	lastUnderscore := false
	for _, ch := range strings.TrimSpace(value) {
		valid := ch >= 'A' && ch <= 'Z' || ch >= 'a' && ch <= 'z' || ch >= '0' && ch <= '9' || ch == '_'
		if valid {
			out.WriteRune(ch)
			lastUnderscore = ch == '_'
		} else if !lastUnderscore {
			out.WriteByte('_')
			lastUnderscore = true
		}
	}
	result := strings.Trim(out.String(), "_")
	if result != "" && result[0] >= '0' && result[0] <= '9' {
		result = "_" + result
	}
	return result
}

// Analysis records whether each browser action has a guaranteed named session
// and which named sessions must be supplied by the runtime.
type Analysis struct {
	establishedBefore map[*rollout.Step]bool
	external          map[string]bool
}

// Analyze conservatively walks the executable document order. Authentication
// inside a conditional, loop, or parallel branch never becomes guaranteed
// after that construct.
func Analyze(intent *rollout.Intent) Analysis {
	result := Analysis{establishedBefore: map[*rollout.Step]bool{}, external: map[string]bool{}}
	if intent != nil {
		result.walkList(intent.Steps, map[string]bool{})
	}
	return result
}

// EstablishedBefore reports whether action's named session is guaranteed by
// an earlier authentication step on every path reaching the action.
func (a Analysis) EstablishedBefore(action *rollout.Step) bool {
	return action != nil && a.establishedBefore[action]
}

// ExternalSessions returns the sorted named sessions required before the
// workflow starts.
func (a Analysis) ExternalSessions() []string {
	out := make([]string, 0, len(a.external))
	for name := range a.external {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func (a Analysis) walkList(steps []*rollout.Step, incoming map[string]bool) map[string]bool {
	current := cloneSet(incoming)
	for _, step := range steps {
		if step == nil {
			continue
		}
		kind := strings.ToLower(strings.TrimSpace(step.Type))
		session := strings.TrimSpace(step.BrowserSession)
		if kind == "browser" && session != "" {
			a.establishedBefore[step] = current[session]
			if !current[session] {
				a.external[session] = true
			}
		}

		conditional := strings.TrimSpace(step.When) != ""
		loop := strings.TrimSpace(step.ForEach) != "" || kind == "foreach" || kind == "for" || kind == "while" || kind == "loop"
		parallel := kind == "parallel"
		hasCases := len(step.Cases) != 0 || step.Default != nil || kind == "switch"

		switch {
		case parallel:
			for _, nested := range step.Steps {
				a.walkList([]*rollout.Step{nested}, current)
			}
		case hasCases:
			for _, branch := range step.Cases {
				if branch != nil {
					a.walkList(branch.Steps, current)
				}
			}
			if step.Default != nil {
				a.walkList(step.Default.Steps, current)
			}
			if len(step.Steps) != 0 {
				a.walkList(step.Steps, current)
			}
		case loop || conditional:
			a.walkList(step.Steps, current)
		default:
			current = a.walkList(step.Steps, current)
		}

		if kind == "browser_authentication" && session != "" && !conditional && !loop {
			current[session] = true
		}
	}
	return current
}

// EffectiveSource resolves the source inherited by one step using the same
// precedence as UWS lowering: explicit source, legacy openapi, then parent.
func EffectiveSource(step *rollout.Step, inherited string) string {
	if step == nil {
		return strings.TrimSpace(inherited)
	}
	for _, value := range []string{step.Source, step.OpenAPI, inherited} {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

// WalkEffectiveSources visits every nested step, case, and default with the
// exact source context inherited by that step during lowering.
func WalkEffectiveSources(intent *rollout.Intent, visit func(*rollout.Step, string)) {
	if intent == nil || visit == nil {
		return
	}
	inherited := ""
	for _, value := range []string{intent.Source, intent.OpenAPI} {
		if value = strings.TrimSpace(value); value != "" {
			inherited = value
			break
		}
	}
	walkEffectiveSources(intent.Steps, inherited, visit)
}

func walkEffectiveSources(steps []*rollout.Step, inherited string, visit func(*rollout.Step, string)) {
	for _, step := range steps {
		if step == nil {
			continue
		}
		source := EffectiveSource(step, inherited)
		visit(step, source)
		walkEffectiveSources(step.Steps, source, visit)
		for _, branch := range step.Cases {
			if branch != nil {
				walkEffectiveSources(branch.Steps, source, visit)
			}
		}
		if step.Default != nil {
			walkEffectiveSources(step.Default.Steps, source, visit)
		}
	}
}

func cloneSet(input map[string]bool) map[string]bool {
	out := make(map[string]bool, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}
