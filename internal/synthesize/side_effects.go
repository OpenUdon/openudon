package synthesize

import (
	"strings"

	"github.com/genelet/udon/pkg/rollout"
)

type sideEffectProfile struct {
	SideEffectful     bool
	HasApprovalPolicy bool
	HasSandboxPolicy  bool
	Reasons           []string
}

func sideEffectProfileFor(policy projectPolicy, intent *rollout.Intent) sideEffectProfile {
	var profile sideEffectProfile
	policyText := strings.ToLower(strings.Join([]string{
		policy.RuntimeSection,
		policy.FunctionSection,
		policy.SafetySection,
	}, "\n"))
	profile.HasApprovalPolicy = containsAny(policyText, []string{
		"approved", "approval", "trusted runner", "trusted runtime", "trusted proof run", "explicitly approved",
	})
	profile.HasSandboxPolicy = containsAny(policyText, []string{"sandbox", "test endpoint", "test endpoints", "proof run"})
	for _, contract := range policy.FunctionContracts {
		sideEffects := strings.TrimSpace(contract.SideEffects)
		if sideEffects == "" || sideEffectsNone(sideEffects) {
			continue
		}
		profile.SideEffectful = true
		name := strings.TrimSpace(contract.Name)
		if name == "" {
			name = "fnct"
		}
		profile.Reasons = append(profile.Reasons, name+" side effects: "+sideEffects)
	}
	walkIntentSteps(intentSteps(intent), func(step *rollout.Step) {
		if step == nil {
			return
		}
		kind := strings.ToLower(strings.TrimSpace(step.Type))
		name := strings.TrimSpace(step.Name)
		if name == "" {
			name = "<unnamed>"
		}
		if kind == "cmd" || kind == "ssh" {
			profile.SideEffectful = true
			profile.Reasons = append(profile.Reasons, name+" uses "+kind+" runtime")
			if policy.AllowedRuntime[kind] {
				profile.HasApprovalPolicy = true
			}
			return
		}
		text := strings.ToLower(strings.Join([]string{step.Name, step.Do, step.Operation, step.Set}, " "))
		if containsSideEffectVerb(text) {
			profile.SideEffectful = true
			profile.Reasons = append(profile.Reasons, name+" appears side-effectful")
		}
	})
	if containsAny(policyText, []string{"side-effectful", "side effectful", "sends email", "send email", "deploy workflow"}) {
		profile.SideEffectful = true
		if len(profile.Reasons) == 0 {
			profile.Reasons = append(profile.Reasons, "project policy mentions side effects")
		}
	}
	profile.Reasons = sortedUnique(profile.Reasons)
	return profile
}

func sideEffectsNone(value string) bool {
	value = strings.ToLower(strings.Trim(strings.TrimSpace(value), ".,;:"))
	return value == "none" || value == "no side effects" || strings.Contains(value, "side effects: none")
}

func containsSideEffectVerb(value string) bool {
	for _, token := range sideEffectTokens(value) {
		switch token {
		case "create", "created", "creates", "send", "sends", "sent", "write", "writes", "update", "updates", "delete", "deletes", "deploy":
			return true
		}
		if strings.HasPrefix(token, "create") || strings.HasPrefix(token, "update") || strings.HasPrefix(token, "delete") {
			return true
		}
	}
	return false
}

func sideEffectTokens(value string) []string {
	var out []string
	for _, field := range strings.FieldsFunc(strings.ToLower(value), func(r rune) bool {
		return !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'))
	}) {
		if field != "" {
			out = append(out, field)
		}
	}
	return out
}

func containsAny(value string, needles []string) bool {
	value = strings.ToLower(value)
	for _, needle := range needles {
		if strings.Contains(value, strings.ToLower(needle)) {
			return true
		}
	}
	return false
}
