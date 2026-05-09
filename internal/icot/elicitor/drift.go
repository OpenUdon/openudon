package elicitor

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/OpenUdon/openudon/internal/projectdoc"
	"github.com/OpenUdon/openudon/internal/projectwizard"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

type DriftCheck struct {
	Code    string
	Message string
	Detail  string
}

func CompareProjectIntentDrift(projectText string, intent *rollout.Intent) []DriftCheck {
	if intent == nil {
		return nil
	}
	project, _ := projectwizard.LoadAnswersFromMarkdown(projectText)

	var checks []DriftCheck
	addMismatch := func(code, message, detail string) {
		checks = append(checks, DriftCheck{Code: code, Message: message, Detail: detail})
	}

	if intent.Workflow != nil && textMeaningfullyDiffers(project.Goal, intent.Workflow.Description) {
		addMismatch("icot.drift.goal", "project.md goal differs from workflows/intent.hcl", fmt.Sprintf("project goal %q vs intent goal %q", project.Goal, intent.Workflow.Description))
	}
	if refs := intentOpenAPIRefs(*intent); len(refs) > 0 {
		for _, ref := range refs {
			if !strings.Contains(projectText, ref) {
				addMismatch("icot.drift.openapi", "project.md OpenAPI refs differ from workflows/intent.hcl", fmt.Sprintf("missing OpenAPI ref %q in project.md", ref))
			}
		}
	} else if !projectdoc.NoOpenAPIRequired(projectText) {
		addMismatch("icot.drift.openapi", "project.md OpenAPI policy differs from workflows/intent.hcl", "intent.hcl has no OpenAPI refs; project.md should say `OpenAPI: none required` or reconcile")
	}
	for _, input := range intent.Inputs {
		if input != nil && input.Name != "" && !sectionContainsToken(projectText, "Inputs", input.Name) {
			addMismatch("icot.drift.inputs", "project.md inputs differ from workflows/intent.hcl", fmt.Sprintf("missing input %q in project.md", input.Name))
		}
	}
	for _, output := range intent.Outputs {
		if output != nil && output.Name != "" && !sectionContainsToken(projectText, "Outputs", output.Name) {
			addMismatch("icot.drift.outputs", "project.md outputs differ from workflows/intent.hcl", fmt.Sprintf("missing output %q in project.md", output.Name))
		}
	}
	for _, phrase := range splitDriftPhrases(dataFlowText(intent.Steps)) {
		if !containsLoose(projectdoc.Section(projectText, "Data Flow"), phrase) {
			addMismatch("icot.drift.data_flow", "project.md data-flow hints differ from workflows/intent.hcl", fmt.Sprintf("missing data-flow hint %q", phrase))
		}
	}
	for _, phrase := range splitDriftPhrases(functionText(intent.Steps)) {
		name := functionNameFromPhrase(phrase)
		if name != "" && !sectionContainsToken(projectText, "Function Contracts", name) {
			addMismatch("icot.drift.functions", "project.md function contracts differ from workflows/intent.hcl", fmt.Sprintf("missing function step %q", name))
		}
	}
	intentCreds := intentCredentialRefs(intent)
	projectCreds := stringSet(project.Credentials)
	for _, cred := range sortedKeys(intentCreds) {
		if !projectCreds[cred] {
			addMismatch("icot.drift.credentials", "project.md credential bindings differ from workflows/intent.hcl", fmt.Sprintf("intent references credential binding %q not declared in project.md", cred))
		}
	}
	if usesRuntime(intent.Steps, "cmd") && !project.CmdApproved {
		addMismatch("icot.drift.runtime.cmd", "project.md runtime approvals differ from workflows/intent.hcl", "intent.hcl uses cmd but project.md does not explicitly approve cmd")
	}
	if usesRuntime(intent.Steps, "ssh") && !project.SSHApproved {
		addMismatch("icot.drift.runtime.ssh", "project.md runtime approvals differ from workflows/intent.hcl", "intent.hcl uses ssh but project.md does not explicitly approve ssh")
	}
	return dedupeDrift(checks)
}

func textMeaningfullyDiffers(a, b string) bool {
	a = normalizeDriftText(a)
	b = normalizeDriftText(b)
	if a == "" || b == "" {
		return false
	}
	return !strings.Contains(a, b) && !strings.Contains(b, a)
}

func sectionContainsToken(text, heading, token string) bool {
	return containsLoose(projectdoc.Section(text, heading), token)
}

func containsLoose(text, token string) bool {
	return strings.Contains(normalizeDriftText(text), normalizeDriftText(token))
}

var driftTokenRE = regexp.MustCompile(`[^a-zA-Z0-9_./-]+`)

func normalizeDriftText(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "`", "")
	value = driftTokenRE.ReplaceAllString(value, " ")
	return strings.Join(strings.Fields(value), " ")
}

func splitDriftPhrases(value string) []string {
	var out []string
	for _, part := range strings.Split(value, ";") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func functionNameFromPhrase(phrase string) string {
	phrase = strings.TrimSpace(phrase)
	if strings.HasPrefix(phrase, "`") {
		rest := strings.TrimPrefix(phrase, "`")
		if idx := strings.Index(rest, "`"); idx >= 0 {
			return rest[:idx]
		}
	}
	fields := strings.Fields(phrase)
	if len(fields) == 0 {
		return ""
	}
	return strings.Trim(fields[0], "`:")
}

func intentCredentialRefs(intent *rollout.Intent) map[string]bool {
	out := map[string]bool{}
	if intent == nil {
		return out
	}
	for _, sec := range intent.Security {
		if sec == nil {
			continue
		}
		for _, value := range []string{sec.Name, sec.TokenFrom} {
			for _, cred := range credentialCandidates(value) {
				out[cred] = true
			}
		}
	}
	walkStepValues(intent.Steps, func(value string) {
		for _, cred := range credentialCandidates(value) {
			out[cred] = true
		}
	})
	return out
}

func intentOpenAPIRefs(intent rollout.Intent) []string {
	var refs []string
	if strings.TrimSpace(intent.OpenAPI) != "" {
		refs = append(refs, strings.TrimSpace(intent.OpenAPI))
	}
	walkSteps(intent.Steps, func(step *rollout.Step) {
		if strings.TrimSpace(step.OpenAPI) != "" {
			refs = append(refs, strings.TrimSpace(step.OpenAPI))
		}
	})
	return dedupeStrings(refs)
}

func credentialCandidates(value string) []string {
	value = strings.TrimSpace(strings.Trim(value, "`'\""))
	if strings.HasPrefix(value, "credentials.") {
		return []string{strings.TrimPrefix(value, "credentials.")}
	}
	if strings.HasPrefix(value, "credential.") {
		return []string{strings.TrimPrefix(value, "credential.")}
	}
	if strings.HasPrefix(value, "secrets.") {
		return []string{strings.TrimPrefix(value, "secrets.")}
	}
	if strings.Contains(strings.ToLower(value), "token") || strings.Contains(strings.ToLower(value), "api_key") || strings.Contains(strings.ToLower(value), "apikey") {
		if !strings.HasPrefix(value, "inputs.") && !strings.Contains(value, " ") {
			return []string{value}
		}
	}
	return nil
}

func walkStepValues(steps []*rollout.Step, visit func(string)) {
	walkSteps(steps, func(step *rollout.Step) {
		for _, value := range step.With {
			visit(value)
		}
		for _, bind := range step.Binds {
			if bind == nil {
				continue
			}
			for _, value := range bind.Fields {
				visit(value)
			}
		}
	})
}

func usesRuntime(steps []*rollout.Step, runtime string) bool {
	found := false
	walkSteps(steps, func(step *rollout.Step) {
		if strings.EqualFold(strings.TrimSpace(step.Type), runtime) {
			found = true
		}
	})
	return found
}

func stringSet(values []string) map[string]bool {
	out := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out[value] = true
		}
	}
	return out
}

func sortedKeys(values map[string]bool) []string {
	var out []string
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func dedupeDrift(checks []DriftCheck) []DriftCheck {
	seen := map[string]bool{}
	var out []DriftCheck
	for _, check := range checks {
		key := check.Code + "\x00" + check.Detail
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, check)
	}
	return out
}
