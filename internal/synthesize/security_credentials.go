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
	normalizeStepSecurityCredentialBindings(intent, intent.Steps, security, primary)
}

func normalizeStepSecurityCredentialBindings(intent *rollout.Intent, steps []*rollout.Step, security map[string][]openAPISecurityRequirement, primary string) {
	for _, step := range steps {
		if step == nil {
			continue
		}
		source := intentStepOpenAPIPath(intent, step, primary)
		for _, req := range security[operationKey(source, step.Operation)] {
			normalizeStepSecurityCredentialBinding(step, req)
		}
		normalizeStepSecurityCredentialBindings(intent, step.Steps, security, primary)
		for _, branch := range step.Cases {
			if branch != nil {
				normalizeStepSecurityCredentialBindings(intent, branch.Steps, security, primary)
			}
		}
		if step.Default != nil {
			normalizeStepSecurityCredentialBindings(intent, step.Default.Steps, security, primary)
		}
	}
}

func normalizeStepSecurityCredentialBinding(step *rollout.Step, req openAPISecurityRequirement) {
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
		if field != primaryField && credentialSourceUsesScheme(current, req) {
			delete(step.With, field)
			current = ""
		}
		if current != "" && shouldReplaceSecurityCredentialSource(current, req) {
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
			if credentialSourceUsesScheme(current, req) || (field != primaryField && shouldReplaceSecurityCredentialSource(current, req)) {
				delete(bind.Fields, field)
				continue
			}
			if current != "" && shouldReplaceSecurityCredentialSource(current, req) {
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
		if shouldReplaceSecurityCredentialSource(step.With[primaryField], req) {
			step.With[primaryField] = source
		}
	}
}

func credentialSourceUsesScheme(current string, req openAPISecurityRequirement) bool {
	current = strings.TrimSpace(current)
	if !strings.HasPrefix(current, "credentials.") {
		return false
	}
	return strings.EqualFold(strings.TrimPrefix(current, "credentials."), strings.TrimSpace(req.label()))
}

func shouldReplaceSecurityCredentialSource(current string, req openAPISecurityRequirement) bool {
	current = strings.TrimSpace(current)
	if current == "" {
		return true
	}
	if !strings.HasPrefix(current, "credentials.") {
		return false
	}
	return !credentialSourceUsesScheme(current, req)
}
