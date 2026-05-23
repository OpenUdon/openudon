package elicitor

import (
	"strconv"
	"strings"

	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

func applyDraftReviewRepairs(session *Session, issues []DraftReviewIssue) (bool, []string) {
	if session == nil {
		return false, nil
	}
	changed := false
	var rejected []string
	for _, issue := range sanitizeDraftReviewResponse(DraftReviewResponse{Issues: issues}).Issues {
		if strings.TrimSpace(issue.SuggestedAnswer) == "" {
			rejected = append(rejected, firstNonEmpty(issue.Slot, issue.Code)+" (missing suggested answer)")
			continue
		}
		ok := false
		switch {
		case draftRepairWithMapping(session, issue):
			ok = true
		case draftRepairOutput(session, issue):
			ok = true
		case draftRepairDependsOn(session, issue):
			ok = true
		}
		if ok {
			changed = true
			addDecisionEvidence(session, DecisionEvidence{
				Stage:                decisionStageDraftReview,
				Slot:                 firstNonEmpty(issue.Slot, "draft_review."+issue.Code),
				Value:                issue.SuggestedAnswer,
				Source:               mappingSourceDeterministic,
				Confidence:           mappingConfidenceReview,
				Reason:               "Applied an allowed pre-final flow-review repair suggestion.",
				Evidence:             issue.Message,
				RequiresConfirmation: true,
			})
			continue
		}
		rejected = append(rejected, firstNonEmpty(issue.Slot, issue.Code)+" (outside allowed repair fields)")
	}
	return changed, dedupeStrings(rejected)
}

func draftRepairWithMapping(session *Session, issue DraftReviewIssue) bool {
	stepName, field, ok := parseStepWithSlot(issue.Slot)
	if !ok {
		return false
	}
	step := stepByName(session.Intent.Steps, stepName)
	if step == nil {
		return false
	}
	source := strings.TrimSpace(issue.SuggestedAnswer)
	assignments := parseAssignments(source)
	if value, ok := assignments[field]; ok {
		source = value
	} else if len(assignments) == 1 {
		for _, value := range assignments {
			source = value
		}
	}
	if source == "" || strings.Contains(source, "operation") || strings.Contains(source, "credential") {
		return false
	}
	if step.With == nil {
		step.With = map[string]string{}
	}
	if strings.TrimSpace(step.With[field]) == source {
		return false
	}
	step.With[field] = source
	addMappingClassification(session, MappingClassification{
		Slot:                 stepWithSlot(step, field),
		Value:                source,
		Source:               mappingSourceDeterministic,
		Confidence:           mappingConfidenceReview,
		Evidence:             issue.Message,
		Reason:               "Pre-final flow review suggested this allowed request mapping repair.",
		RequiresConfirmation: true,
	})
	return true
}

func draftRepairOutput(session *Session, issue DraftReviewIssue) bool {
	name, field, ok := parseOutputSlot(session, issue.Slot)
	if !ok {
		return false
	}
	if field != "" && field != "from" {
		return false
	}
	assignments := parseAssignments(issue.SuggestedAnswer)
	from := strings.TrimSpace(issue.SuggestedAnswer)
	if value, ok := assignments[name]; ok {
		from = value
	} else if value, ok := assignments["from"]; ok {
		from = value
	} else if len(assignments) == 1 {
		for _, value := range assignments {
			from = value
		}
	}
	if from == "" || strings.Contains(from, "operation") || strings.Contains(from, "credential") {
		return false
	}
	for _, output := range session.Intent.Outputs {
		if output == nil || output.Name != name {
			continue
		}
		if strings.TrimSpace(output.From) == from {
			return false
		}
		output.From = from
		addMappingClassification(session, MappingClassification{
			Slot:                 "intent.outputs." + output.Name,
			Value:                output.Name + "=" + output.From,
			Source:               mappingSourceDeterministic,
			Confidence:           mappingConfidenceReview,
			Evidence:             issue.Message,
			Reason:               "Pre-final flow review suggested this allowed output repair.",
			RequiresConfirmation: true,
		})
		return true
	}
	session.Intent.Outputs = append(session.Intent.Outputs, &rollout.Output{Name: name, From: from})
	return true
}

func draftRepairDependsOn(session *Session, issue DraftReviewIssue) bool {
	slot := strings.TrimSpace(issue.Slot)
	if !strings.HasPrefix(slot, "steps.") || !strings.HasSuffix(slot, ".depends_on") {
		return false
	}
	stepName := strings.TrimSuffix(strings.TrimPrefix(slot, "steps."), ".depends_on")
	step := stepByName(session.Intent.Steps, stepName)
	if step == nil {
		return false
	}
	answer := strings.TrimSpace(issue.SuggestedAnswer)
	assignments := parseAssignments(answer)
	if value, ok := assignments["depends_on"]; ok {
		answer = value
	}
	values := splitComma(answer)
	if len(values) == 0 {
		return false
	}
	known := map[string]bool{}
	for _, candidate := range flattenSteps(session.Intent.Steps) {
		if candidate != nil && strings.TrimSpace(candidate.Name) != "" {
			known[candidate.Name] = true
		}
	}
	var deps []string
	for _, value := range values {
		if known[value] && value != step.Name {
			deps = append(deps, value)
		}
	}
	deps = dedupeStrings(deps)
	if len(deps) == 0 || strings.Join(step.DependsOn, ",") == strings.Join(deps, ",") {
		return false
	}
	step.DependsOn = deps
	return true
}

func parseStepWithSlot(slot string) (string, string, bool) {
	slot = strings.TrimSpace(slot)
	if !strings.HasPrefix(slot, "steps.") {
		return "", "", false
	}
	rest := strings.TrimPrefix(slot, "steps.")
	idx := strings.Index(rest, ".with.")
	if idx <= 0 {
		return "", "", false
	}
	stepName := rest[:idx]
	field := strings.TrimSpace(rest[idx+len(".with."):])
	return stepName, field, stepName != "" && field != ""
}

func parseOutputSlot(session *Session, slot string) (string, string, bool) {
	slot = strings.TrimSpace(slot)
	for _, prefix := range []string{"intent.outputs.", "outputs."} {
		if strings.HasPrefix(slot, prefix) {
			name := strings.TrimSpace(strings.TrimPrefix(slot, prefix))
			field := ""
			if strings.HasSuffix(name, ".from") {
				name = strings.TrimSuffix(name, ".from")
				field = "from"
			}
			return name, field, name != ""
		}
	}
	if strings.HasPrefix(slot, "outputs[") {
		end := strings.Index(slot, "]")
		if end <= len("outputs[") {
			return "", "", false
		}
		index, err := strconv.Atoi(strings.TrimSpace(slot[len("outputs["):end]))
		if err != nil || index < 0 || session == nil || index >= len(session.Intent.Outputs) {
			return "", "", false
		}
		field := strings.TrimPrefix(strings.TrimSpace(slot[end+1:]), ".")
		output := session.Intent.Outputs[index]
		if output == nil || strings.TrimSpace(output.Name) == "" {
			return "", "", false
		}
		return strings.TrimSpace(output.Name), field, true
	}
	return "", "", false
}

func stepByName(steps []*rollout.Step, name string) *rollout.Step {
	for _, step := range flattenSteps(steps) {
		if step != nil && step.Name == name {
			return step
		}
	}
	return nil
}

func splitComma(value string) []string {
	var out []string
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
