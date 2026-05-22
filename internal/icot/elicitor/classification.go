package elicitor

import (
	"fmt"
	"sort"
	"strings"

	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

const (
	mappingSourceDeterministic   = "deterministic"
	mappingSourceLLM             = "llm"
	mappingSourceUser            = "user"
	mappingSourceFallbackDefault = "fallback_default"

	mappingConfidenceHigh     = "high"
	mappingConfidenceReview   = "review"
	mappingConfidenceLow      = "low"
	mappingConfidenceConflict = "conflict"
)

type MappingClassification struct {
	Slot                 string `json:"slot,omitempty" yaml:"slot,omitempty"`
	Value                string `json:"value,omitempty" yaml:"value,omitempty"`
	Source               string `json:"source,omitempty" yaml:"source,omitempty"`
	Confidence           string `json:"confidence,omitempty" yaml:"confidence,omitempty"`
	Evidence             string `json:"evidence,omitempty" yaml:"evidence,omitempty"`
	Reason               string `json:"reason,omitempty" yaml:"reason,omitempty"`
	RequiresConfirmation bool   `json:"requires_confirmation,omitempty" yaml:"requires_confirmation,omitempty"`
}

func addMappingClassification(session *Session, classification MappingClassification) {
	if session == nil {
		return
	}
	classification = normalizeMappingClassification(classification)
	if classification.Slot == "" || classification.Value == "" || classification.Source == "" {
		return
	}
	if classification.Source == mappingSourceUser {
		session.Classifications = pruneSupersededUserClassifications(session.Classifications, classification)
	}
	session.Classifications = normalizeMappingClassifications(append(session.Classifications, classification))
}

func mergeClassifications(base, overlay []MappingClassification) []MappingClassification {
	return normalizeMappingClassifications(append(append([]MappingClassification(nil), base...), overlay...))
}

func normalizeMappingClassifications(classifications []MappingClassification) []MappingClassification {
	seen := map[string]int{}
	out := make([]MappingClassification, 0, len(classifications))
	for _, classification := range classifications {
		classification = normalizeMappingClassification(classification)
		if classification.Slot == "" || classification.Value == "" || classification.Source == "" {
			continue
		}
		key := classification.Slot + "\x00" + classification.Value + "\x00" + classification.Source
		if existing, ok := seen[key]; ok {
			out[existing] = mergeMappingClassification(out[existing], classification)
			continue
		}
		seen[key] = len(out)
		out = append(out, classification)
	}
	markMappingConflicts(out)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Slot == out[j].Slot {
			if out[i].Value == out[j].Value {
				return out[i].Source < out[j].Source
			}
			return out[i].Value < out[j].Value
		}
		return out[i].Slot < out[j].Slot
	})
	return out
}

func normalizeMappingClassification(classification MappingClassification) MappingClassification {
	classification.Slot = strings.TrimSpace(classification.Slot)
	classification.Value = strings.TrimSpace(classification.Value)
	classification.Source = normalizeMappingSource(classification.Source)
	classification.Confidence = normalizeMappingConfidence(classification.Confidence)
	classification.Evidence = strings.TrimSpace(classification.Evidence)
	classification.Reason = strings.TrimSpace(classification.Reason)
	return classification
}

func normalizeMappingSource(source string) string {
	switch strings.TrimSpace(source) {
	case mappingSourceDeterministic, mappingSourceLLM, mappingSourceUser, mappingSourceFallbackDefault:
		return strings.TrimSpace(source)
	default:
		return ""
	}
}

func normalizeMappingConfidence(confidence string) string {
	switch strings.TrimSpace(confidence) {
	case mappingConfidenceHigh, mappingConfidenceReview, mappingConfidenceLow, mappingConfidenceConflict:
		return strings.TrimSpace(confidence)
	default:
		return mappingConfidenceReview
	}
}

func mergeMappingClassification(base, overlay MappingClassification) MappingClassification {
	if base.Confidence != mappingConfidenceConflict && overlay.Confidence == mappingConfidenceConflict {
		base.Confidence = mappingConfidenceConflict
	}
	if base.Confidence == "" {
		base.Confidence = overlay.Confidence
	}
	if base.Evidence == "" {
		base.Evidence = overlay.Evidence
	}
	if base.Reason == "" {
		base.Reason = overlay.Reason
	}
	base.RequiresConfirmation = base.RequiresConfirmation || overlay.RequiresConfirmation
	return base
}

func markMappingConflicts(classifications []MappingClassification) {
	type slotState struct {
		values  map[string]bool
		sources map[string]bool
	}
	bySlot := map[string]*slotState{}
	for _, classification := range classifications {
		if classification.Slot == "" || classification.Value == "" || classification.Source == "" {
			continue
		}
		state := bySlot[classification.Slot]
		if state == nil {
			state = &slotState{values: map[string]bool{}, sources: map[string]bool{}}
			bySlot[classification.Slot] = state
		}
		state.values[classification.Value] = true
		state.sources[classification.Source] = true
	}
	conflictEvidence := map[string]string{}
	for slot, state := range bySlot {
		if len(state.values) <= 1 || len(state.sources) <= 1 {
			continue
		}
		conflictEvidence[slot] = "conflicting values: " + strings.Join(sortedKeys(state.values), ", ")
	}
	for i := range classifications {
		evidence, ok := conflictEvidence[classifications[i].Slot]
		if !ok {
			continue
		}
		classifications[i].Confidence = mappingConfidenceConflict
		classifications[i].RequiresConfirmation = true
		if classifications[i].Reason == "" {
			classifications[i].Reason = "Multiple inference sources proposed different values for this mapping slot."
		}
		if classifications[i].Evidence == "" {
			classifications[i].Evidence = evidence
		}
	}
}

func pruneSupersededUserClassifications(classifications []MappingClassification, user MappingClassification) []MappingClassification {
	out := classifications[:0]
	for _, classification := range classifications {
		classification = normalizeMappingClassification(classification)
		if classification.Slot == user.Slot && classification.Value != user.Value {
			continue
		}
		out = append(out, classification)
	}
	return out
}

func mappingClassificationIssues(session Session) []ReadinessIssue {
	classifications := normalizeMappingClassifications(session.Classifications)
	conflictSlots := map[string][]MappingClassification{}
	lowSlots := map[string]MappingClassification{}
	for _, classification := range classifications {
		switch classification.Confidence {
		case mappingConfidenceConflict:
			conflictSlots[classification.Slot] = append(conflictSlots[classification.Slot], classification)
			lowSlots[classification.Slot] = classification
		case mappingConfidenceLow:
			lowSlots[classification.Slot] = classification
		}
	}
	var issues []ReadinessIssue
	for slot, records := range conflictSlots {
		values := map[string]bool{}
		for _, record := range records {
			values[record.Value] = true
		}
		suggested := currentMappingAssignment(session, slot)
		if suggested == "" && len(records) > 0 {
			suggested = records[0].Value
		}
		issues = append(issues, ReadinessIssue{
			Code:            "conflicting_mapping",
			Slot:            slot,
			Severity:        readinessBlocking,
			Message:         "Mapping " + slot + " has conflicting inferred values: " + strings.Join(sortedKeys(values), ", ") + ".",
			SuggestedAnswer: suggested,
		})
	}
	for _, slot := range sortedKeysFromClassifications(lowSlots) {
		record := lowSlots[slot]
		issues = append(issues, ReadinessIssue{
			Code:            "low_confidence_mapping",
			Slot:            record.Slot,
			Severity:        readinessBlocking,
			Message:         "Mapping " + record.Slot + " needs confirmation because its inferred value is low confidence.",
			SuggestedAnswer: firstNonEmpty(currentMappingAssignment(session, record.Slot), record.Value),
		})
	}
	sort.SliceStable(issues, func(i, j int) bool {
		if issues[i].Code == issues[j].Code {
			return issues[i].Slot < issues[j].Slot
		}
		return issues[i].Code < issues[j].Code
	})
	return issues
}

func currentMappingAssignment(session Session, slot string) string {
	slot = strings.TrimSpace(slot)
	switch {
	case slot == "intent.openapi", slot == "intent.source":
		return intentAPISourceRef(session.Intent)
	case slot == "credentials":
		return strings.Join(session.Credentials, ", ")
	case strings.HasPrefix(slot, "intent.outputs."):
		name := strings.TrimPrefix(slot, "intent.outputs.")
		for _, output := range session.Intent.Outputs {
			if output != nil && output.Name == name {
				return output.Name + "=" + output.From
			}
		}
	case strings.HasPrefix(slot, "steps."):
		parts := strings.Split(slot, ".")
		if len(parts) < 3 {
			return ""
		}
		stepName := parts[1]
		for _, step := range flattenSteps(session.Intent.Steps) {
			if step == nil || step.Name != stepName {
				continue
			}
			if len(parts) == 3 && parts[2] == "operation" {
				return step.Operation
			}
			if len(parts) >= 4 && parts[2] == "with" {
				field := strings.Join(parts[3:], ".")
				if step.With != nil && strings.TrimSpace(step.With[field]) != "" {
					return field + "=" + step.With[field]
				}
			}
		}
	}
	return ""
}

func flattenSteps(steps []*rollout.Step) []*rollout.Step {
	var out []*rollout.Step
	walkSteps(steps, func(step *rollout.Step) {
		out = append(out, step)
	})
	return out
}

func recordLLMOverlayClassifications(base *Session, before, overlay Session) {
	if base == nil {
		return
	}
	evidence := firstLine(firstNonEmpty(overlay.Project.Goal, draftSessionDescription(overlay)))
	beforeSteps := stepsByName(before.Intent.Steps)
	walkSteps(overlay.Intent.Steps, func(step *rollout.Step) {
		if step == nil || strings.TrimSpace(step.Name) == "" {
			return
		}
		beforeStep := beforeSteps[step.Name]
		if strings.TrimSpace(step.Operation) != "" && (beforeStep == nil || strings.TrimSpace(beforeStep.Operation) != step.Operation) {
			addMappingClassification(base, MappingClassification{
				Slot:                 stepOperationSlot(step),
				Value:                step.Operation,
				Source:               mappingSourceLLM,
				Confidence:           mappingConfidenceReview,
				Evidence:             evidence,
				Reason:               "LLM draft selected this operation from the workflow brief and local OpenAPI metadata.",
				RequiresConfirmation: true,
			})
		}
		for field, source := range step.With {
			field = strings.TrimSpace(field)
			source = strings.TrimSpace(source)
			if field == "" || source == "" {
				continue
			}
			if beforeStep != nil && strings.TrimSpace(beforeStep.With[field]) == source {
				continue
			}
			addMappingClassification(base, MappingClassification{
				Slot:                 stepWithSlot(step, field),
				Value:                source,
				Source:               mappingSourceLLM,
				Confidence:           mappingConfidenceReview,
				Evidence:             evidence,
				Reason:               "LLM draft mapped this request field from the workflow brief and local OpenAPI metadata.",
				RequiresConfirmation: true,
			})
		}
		for i, bind := range step.Binds {
			if bind == nil {
				continue
			}
			for field, source := range bind.Fields {
				field = strings.TrimSpace(field)
				source = strings.TrimSpace(source)
				if field == "" || source == "" {
					continue
				}
				value := bind.From + "." + source
				addMappingClassification(base, MappingClassification{
					Slot:                 fmt.Sprintf("steps.%s.bind.%d.%s", step.Name, i+1, field),
					Value:                value,
					Source:               mappingSourceLLM,
					Confidence:           mappingConfidenceReview,
					Evidence:             evidence,
					Reason:               "LLM draft mapped this request field from a prior step output.",
					RequiresConfirmation: true,
				})
			}
		}
	})
	beforeCredentials := stringSet(before.Credentials)
	for _, credential := range overlay.Credentials {
		if strings.TrimSpace(credential) == "" || beforeCredentials[credential] {
			continue
		}
		addMappingClassification(base, MappingClassification{
			Slot:                 "credentials",
			Value:                credential,
			Source:               mappingSourceLLM,
			Confidence:           mappingConfidenceReview,
			Evidence:             evidence,
			Reason:               "LLM draft inferred this credential binding name from the workflow brief and API security metadata.",
			RequiresConfirmation: true,
		})
	}
	beforeOutputs := outputsByName(before.Intent.Outputs)
	for _, output := range overlay.Intent.Outputs {
		if output == nil || strings.TrimSpace(output.Name) == "" || strings.TrimSpace(output.From) == "" {
			continue
		}
		if beforeOutput := beforeOutputs[output.Name]; beforeOutput != nil && strings.TrimSpace(beforeOutput.From) == output.From {
			continue
		}
		addMappingClassification(base, MappingClassification{
			Slot:                 "intent.outputs." + output.Name,
			Value:                output.Name + "=" + output.From,
			Source:               mappingSourceLLM,
			Confidence:           mappingConfidenceReview,
			Evidence:             evidence,
			Reason:               "LLM draft inferred this workflow output from the workflow brief.",
			RequiresConfirmation: true,
		})
	}
}

func stepOperationSlot(step *rollout.Step) string {
	return "steps." + firstNonEmpty(step.Name, "step") + ".operation"
}

func stepWithSlot(step *rollout.Step, field string) string {
	return "steps." + firstNonEmpty(step.Name, "step") + ".with." + strings.TrimSpace(field)
}

func draftSessionDescription(session Session) string {
	if session.Intent.Workflow == nil {
		return ""
	}
	return session.Intent.Workflow.Description
}

func stepsByName(steps []*rollout.Step) map[string]*rollout.Step {
	out := map[string]*rollout.Step{}
	walkSteps(steps, func(step *rollout.Step) {
		if step != nil && strings.TrimSpace(step.Name) != "" {
			out[step.Name] = step
		}
	})
	return out
}

func outputsByName(outputs []*rollout.Output) map[string]*rollout.Output {
	out := map[string]*rollout.Output{}
	for _, output := range outputs {
		if output != nil && strings.TrimSpace(output.Name) != "" {
			out[output.Name] = output
		}
	}
	return out
}

func sortedKeysFromClassifications(values map[string]MappingClassification) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
