package synthesize

import (
	"sort"
	"strings"

	"github.com/OpenUdon/openudon/internal/openapidisco"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

type sideEffectProfile struct {
	SideEffectful       bool
	HasApprovalPolicy   bool
	HasSandboxPolicy    bool
	HasProductionPolicy bool
	ProductionEndpoint  bool
	Reasons             []string
	Effects             []sideEffectEvidence
}

type sideEffectEvidence struct {
	Step      string
	Kind      string
	Source    string
	Operation string
	Method    string
	Path      string
	Risk      string
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
	profile.HasProductionPolicy = containsAny(policyText, []string{
		"production handoff", "approved production", "approved_for_production", "production execution requires human approval", "production requires human approval",
	})
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
		profile.Effects = append(profile.Effects, sideEffectEvidence{
			Step:   name,
			Kind:   "fnct",
			Source: "function contract",
			Risk:   sideEffects,
		})
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
			profile.Effects = append(profile.Effects, sideEffectEvidence{
				Step:   name,
				Kind:   kind,
				Source: "intent runtime",
				Risk:   kind + " runtime can affect local or remote systems",
			})
			if policy.AllowedRuntime[kind] {
				profile.HasApprovalPolicy = true
			}
			return
		}
		text := strings.ToLower(strings.Join([]string{step.Name, step.Do, step.Operation, step.Set}, " "))
		if containsSideEffectVerb(text) {
			profile.SideEffectful = true
			profile.Reasons = append(profile.Reasons, name+" appears side-effectful")
			profile.Effects = append(profile.Effects, sideEffectEvidence{
				Step:      name,
				Kind:      firstNonEmpty(kind, "intent"),
				Source:    "intent language",
				Operation: strings.TrimSpace(step.Operation),
				Risk:      "intent text contains create/send/write/update/delete/post style behavior",
			})
		}
		if containsCustomerCommunicationTerm(text) {
			profile.SideEffectful = true
			profile.Reasons = append(profile.Reasons, name+" sends customer communications")
			profile.Effects = append(profile.Effects, sideEffectEvidence{
				Step:      name,
				Kind:      firstNonEmpty(kind, "intent"),
				Source:    "intent language",
				Operation: strings.TrimSpace(step.Operation),
				Risk:      "customer communication or notification behavior requires approval",
			})
		}
	})
	if containsAny(policyText, []string{"side-effectful", "side effectful", "sends email", "send email", "deploy workflow"}) {
		profile.SideEffectful = true
		if len(profile.Reasons) == 0 {
			profile.Reasons = append(profile.Reasons, "project policy mentions side effects")
		}
		profile.Effects = append(profile.Effects, sideEffectEvidence{
			Step:   "project policy",
			Kind:   "policy",
			Source: "project.md",
			Risk:   "project policy mentions side-effectful behavior",
		})
	}
	profile.Reasons = sortedUnique(profile.Reasons)
	profile.Effects = sortedUniqueSideEffectEvidence(profile.Effects)
	return profile
}

func sideEffectProfileForOpenAPI(policy projectPolicy, intent *rollout.Intent, candidates []openapidisco.Candidate, primary string) sideEffectProfile {
	profile := sideEffectProfileFor(policy, intent)
	ops := openAPIOperationIndex(candidates)
	servers := openAPIServerIndex(candidates)
	walkIntentSteps(intentSteps(intent), func(step *rollout.Step) {
		if step == nil || strings.TrimSpace(step.Operation) == "" {
			return
		}
		name := strings.TrimSpace(step.Name)
		if name == "" {
			name = "<unnamed>"
		}
		specPath := intentStepOpenAPIPath(intent, step, primary)
		op := ops[operationKey(specPath, step.Operation)]
		if op != nil {
			if openAPIMethodIsSideEffectful(op.Method) {
				profile.SideEffectful = true
				profile.Reasons = append(profile.Reasons, name+" uses "+strings.ToUpper(op.Method)+" "+op.Path)
				profile.Effects = append(profile.Effects, sideEffectEvidence{
					Step:      name,
					Kind:      "openapi",
					Source:    specPath,
					Operation: strings.TrimSpace(step.Operation),
					Method:    strings.ToUpper(strings.TrimSpace(op.Method)),
					Path:      strings.TrimSpace(op.Path),
					Risk:      "write-class HTTP operation requires review and approved trusted-runner handoff",
				})
			}
			text := strings.ToLower(strings.Join([]string{op.OperationID, op.Summary, op.Description, strings.Join(op.Tags, " ")}, " "))
			if containsSideEffectVerb(text) || containsCustomerCommunicationTerm(text) {
				profile.SideEffectful = true
				profile.Reasons = append(profile.Reasons, name+" OpenAPI operation appears side-effectful")
				profile.Effects = append(profile.Effects, sideEffectEvidence{
					Step:      name,
					Kind:      "openapi",
					Source:    specPath,
					Operation: strings.TrimSpace(step.Operation),
					Method:    strings.ToUpper(strings.TrimSpace(op.Method)),
					Path:      strings.TrimSpace(op.Path),
					Risk:      "OpenAPI operation text indicates write, send, notification, or customer communication behavior",
				})
			}
		}
		if server := servers[specPath]; productionEndpointURL(server) {
			profile.ProductionEndpoint = true
			profile.Reasons = append(profile.Reasons, name+" uses production endpoint "+server)
			profile.Effects = append(profile.Effects, sideEffectEvidence{
				Step:   name,
				Kind:   "endpoint",
				Source: specPath,
				Risk:   "production endpoint requires explicit production handoff approval",
			})
		}
	})
	if intent != nil && productionEndpointURL(intent.ServerURL) {
		profile.ProductionEndpoint = true
		profile.Reasons = append(profile.Reasons, "intent uses production endpoint "+intent.ServerURL)
		profile.Effects = append(profile.Effects, sideEffectEvidence{
			Step:   "workflow",
			Kind:   "endpoint",
			Source: "intent server",
			Risk:   "production endpoint requires explicit production handoff approval",
		})
	}
	profile.Reasons = sortedUnique(profile.Reasons)
	profile.Effects = sortedUniqueSideEffectEvidence(profile.Effects)
	return profile
}

func sideEffectsNone(value string) bool {
	value = strings.ToLower(strings.Trim(strings.TrimSpace(value), ".,;:"))
	return value == "none" || value == "no side effects" || strings.Contains(value, "side effects: none")
}

func containsSideEffectVerb(value string) bool {
	for _, token := range sideEffectTokens(value) {
		switch token {
		case "create", "created", "creates", "send", "sends", "sent", "write", "writes", "update", "updates", "delete", "deletes", "deploy", "post", "put", "patch":
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

func containsCustomerCommunicationTerm(value string) bool {
	return containsAny(value, []string{"email", "sms", "text message", "webhook", "customer message", "notify customer", "notification"})
}

func openAPIMethodIsSideEffectful(method string) bool {
	switch strings.ToUpper(strings.TrimSpace(method)) {
	case "POST", "PUT", "PATCH", "DELETE":
		return true
	default:
		return false
	}
}

func openAPIServerIndex(candidates []openapidisco.Candidate) map[string]string {
	out := map[string]string{}
	for _, candidate := range candidates {
		spec, err := rollout.LoadOpenAPISpec(candidate.Path)
		if err != nil || spec == nil {
			continue
		}
		out[candidate.RelativePath] = strings.TrimSpace(spec.ServerURL)
	}
	return out
}

func productionEndpointURL(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	if lower == "" {
		return false
	}
	if strings.Contains(lower, "localhost") || strings.Contains(lower, "127.0.0.1") || strings.Contains(lower, ".test") ||
		strings.Contains(lower, "sandbox") || strings.Contains(lower, "staging") || strings.Contains(lower, "example.") {
		return false
	}
	return strings.Contains(lower, "production") || strings.Contains(lower, "://prod.") ||
		strings.Contains(lower, "://prod-") || strings.Contains(lower, ".prod.") || strings.Contains(lower, "-prod.")
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

func sortedUniqueSideEffectEvidence(effects []sideEffectEvidence) []sideEffectEvidence {
	seen := map[string]bool{}
	var out []sideEffectEvidence
	for _, effect := range effects {
		effect.Step = strings.TrimSpace(effect.Step)
		effect.Kind = strings.TrimSpace(effect.Kind)
		effect.Source = strings.TrimSpace(effect.Source)
		effect.Operation = strings.TrimSpace(effect.Operation)
		effect.Method = strings.TrimSpace(effect.Method)
		effect.Path = strings.TrimSpace(effect.Path)
		effect.Risk = strings.TrimSpace(effect.Risk)
		key := strings.Join([]string{effect.Step, effect.Kind, effect.Source, effect.Operation, effect.Method, effect.Path, effect.Risk}, "\x00")
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, effect)
	}
	sort.SliceStable(out, func(i, j int) bool {
		left := strings.Join([]string{out[i].Step, out[i].Kind, out[i].Source, out[i].Operation, out[i].Method, out[i].Path, out[i].Risk}, "\x00")
		right := strings.Join([]string{out[j].Step, out[j].Kind, out[j].Source, out[j].Operation, out[j].Method, out[j].Path, out[j].Risk}, "\x00")
		return left < right
	})
	return out
}
