package synthesize

import (
	"strings"

	"github.com/OpenUdon/openudon/internal/openapidisco"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

func normalizeIntentSecurityCredentialBindings(intent *rollout.Intent, candidates []openapidisco.Candidate, primary string) {
	if intent == nil {
		return
	}
	security := openAPISecurityIndex(candidates)
	declared := intentSecurityBindingSet(intent)
	normalizeStepSecurityCredentialBindings(intent, intent.Steps, security, primary, declared)
}

func normalizeStepSecurityCredentialBindings(intent *rollout.Intent, steps []*rollout.Step, security map[string][]openAPISecurityRequirement, primary string, declared map[string]bool) {
	for _, step := range steps {
		if step == nil {
			continue
		}
		source := intentStepOpenAPIPath(intent, step, primary)
		for _, req := range security[operationKey(source, step.Operation)] {
			normalizeStepSecurityCredentialBinding(step, req, declared)
		}
		normalizeStepSecurityCredentialBindings(intent, step.Steps, security, primary, declared)
		for _, branch := range step.Cases {
			if branch != nil {
				normalizeStepSecurityCredentialBindings(intent, branch.Steps, security, primary, declared)
			}
		}
		if step.Default != nil {
			normalizeStepSecurityCredentialBindings(intent, step.Default.Steps, security, primary, declared)
		}
	}
}

func normalizeStepSecurityCredentialBinding(step *rollout.Step, req openAPISecurityRequirement, declared map[string]bool) {
	if step == nil {
		return
	}
	binding := strings.TrimSpace(req.label())
	if binding == "" {
		return
	}
	source := "credentials." + binding
	primaryField := strings.TrimSpace(securityRequestFieldName(req))
	wrotePrimary := false
	for _, field := range req.fieldNames() {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		current := ""
		if step.With != nil {
			current = step.With[field]
		}
		if field != primaryField && shouldDropSecondarySecurityBinding(current, req, declared) {
			delete(step.With, field)
			current = ""
		}
		if current != "" && shouldReplaceSecurityCredentialSource(current, req, declared) {
			step.With[field] = source
			if field == primaryField {
				wrotePrimary = true
			}
		}
		if strings.TrimSpace(current) != "" && field == primaryField {
			wrotePrimary = true
		}
		for _, bind := range step.Binds {
			if bind == nil || bind.Fields == nil {
				continue
			}
			current := bind.Fields[field]
			if shouldDropSecondarySecurityBinding(current, req, declared) {
				delete(bind.Fields, field)
				continue
			}
			if current != "" && shouldReplaceSecurityCredentialSource(current, req, declared) {
				bind.Fields[field] = source
				if field == primaryField {
					wrotePrimary = true
				}
			}
			if strings.TrimSpace(current) != "" && field == primaryField {
				wrotePrimary = true
			}
		}
	}
	if !wrotePrimary && primaryField != "" {
		if step.With == nil {
			step.With = map[string]string{}
		}
		if shouldReplaceSecurityCredentialSource(step.With[primaryField], req, declared) {
			step.With[primaryField] = source
		}
	}
}

func intentSecurityBindingSet(intent *rollout.Intent) map[string]bool {
	if intent == nil || len(intent.Security) == 0 {
		return nil
	}
	out := map[string]bool{}
	for _, security := range intent.Security {
		if security == nil {
			continue
		}
		name := strings.ToLower(strings.TrimSpace(security.Name))
		if name != "" {
			out[name] = true
		}
	}
	return out
}

func shouldDropSecondarySecurityBinding(current string, req openAPISecurityRequirement, declared map[string]bool) bool {
	current = strings.TrimSpace(current)
	if current == "" {
		return false
	}
	return credentialSourceUsesScheme(current, req) || shouldReplaceSecurityCredentialSource(current, req, declared)
}

func credentialSourceUsesScheme(current string, req openAPISecurityRequirement) bool {
	current = strings.TrimSpace(current)
	if !strings.HasPrefix(current, "credentials.") {
		return false
	}
	return strings.EqualFold(strings.TrimPrefix(current, "credentials."), strings.TrimSpace(req.label()))
}

func shouldReplaceSecurityCredentialSource(current string, req openAPISecurityRequirement, declared map[string]bool) bool {
	current = strings.TrimSpace(current)
	if current == "" {
		return true
	}
	if !strings.HasPrefix(current, "credentials.") {
		if declaredPlainCredentialSource(current, declared) {
			return false
		}
		return !looksLikeRuntimeDataSource(current)
	}
	if declaredCredentialSource(current, declared) {
		return false
	}
	return !credentialSourceUsesScheme(current, req)
}

func declaredCredentialSource(current string, declared map[string]bool) bool {
	if len(declared) == 0 {
		return false
	}
	name := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(current, "credentials.")))
	return declared[name]
}

func declaredPlainCredentialSource(current string, declared map[string]bool) bool {
	if len(declared) == 0 {
		return false
	}
	name := strings.ToLower(strings.TrimSpace(current))
	return declared[name]
}

func looksLikeRuntimeDataSource(current string) bool {
	current = strings.TrimSpace(current)
	if strings.HasPrefix(current, "inputs.") {
		return true
	}
	if strings.HasPrefix(current, "variables.") {
		return true
	}
	return looksLikeRuntimeExpression(current)
}
