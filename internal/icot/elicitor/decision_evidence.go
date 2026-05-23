package elicitor

import (
	"sort"
	"strings"
)

const (
	decisionStageCatalogPlan    = "catalog_plan"
	decisionStageOperation      = "operation_selection"
	decisionStageRequestMapping = "request_mapping"
	decisionStageOutput         = "output_selection"
	decisionStageSideEffect     = "side_effect_scope"
	decisionStageDraftReview    = "draft_review"
)

type DecisionAlternative struct {
	Value  string `json:"value,omitempty" yaml:"value,omitempty"`
	Reason string `json:"reason,omitempty" yaml:"reason,omitempty"`
}

type DecisionEvidence struct {
	Stage                string                `json:"stage,omitempty" yaml:"stage,omitempty"`
	Slot                 string                `json:"slot,omitempty" yaml:"slot,omitempty"`
	Value                string                `json:"value,omitempty" yaml:"value,omitempty"`
	Source               string                `json:"source,omitempty" yaml:"source,omitempty"`
	Confidence           string                `json:"confidence,omitempty" yaml:"confidence,omitempty"`
	Reason               string                `json:"reason,omitempty" yaml:"reason,omitempty"`
	Evidence             string                `json:"evidence,omitempty" yaml:"evidence,omitempty"`
	Alternatives         []DecisionAlternative `json:"alternatives,omitempty" yaml:"alternatives,omitempty"`
	RequiresConfirmation bool                  `json:"requires_confirmation,omitempty" yaml:"requires_confirmation,omitempty"`
}

func addDecisionEvidence(session *Session, evidence DecisionEvidence) {
	if session == nil {
		return
	}
	evidence = normalizeDecisionEvidence(evidence)
	if evidence.Stage == "" || evidence.Slot == "" || evidence.Value == "" {
		return
	}
	if evidence.Source == mappingSourceUser {
		session.DecisionEvidence = pruneSupersededDecisionEvidence(session.DecisionEvidence, evidence)
	}
	session.DecisionEvidence = mergeDecisionEvidence(session.DecisionEvidence, []DecisionEvidence{evidence})
}

func addDecisionEvidenceFromMapping(session *Session, classification MappingClassification) {
	classification = normalizeMappingClassification(classification)
	if classification.Slot == "" || classification.Value == "" || classification.Source == "" {
		return
	}
	addDecisionEvidence(session, DecisionEvidence{
		Stage:                decisionStageForSlot(classification.Slot),
		Slot:                 classification.Slot,
		Value:                classification.Value,
		Source:               classification.Source,
		Confidence:           classification.Confidence,
		Reason:               classification.Reason,
		Evidence:             classification.Evidence,
		RequiresConfirmation: classification.RequiresConfirmation,
	})
}

func mergeDecisionEvidence(base, overlay []DecisionEvidence) []DecisionEvidence {
	seen := map[string]int{}
	out := make([]DecisionEvidence, 0, len(base)+len(overlay))
	for _, evidence := range append(append([]DecisionEvidence(nil), base...), overlay...) {
		evidence = normalizeDecisionEvidence(evidence)
		if evidence.Stage == "" || evidence.Slot == "" || evidence.Value == "" {
			continue
		}
		key := evidence.Stage + "\x00" + evidence.Slot + "\x00" + evidence.Value + "\x00" + evidence.Source
		if existing, ok := seen[key]; ok {
			out[existing] = mergeDecisionEvidenceRecord(out[existing], evidence)
			continue
		}
		seen[key] = len(out)
		out = append(out, evidence)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Stage == out[j].Stage {
			if out[i].Slot == out[j].Slot {
				if out[i].Value == out[j].Value {
					return out[i].Source < out[j].Source
				}
				return out[i].Value < out[j].Value
			}
			return out[i].Slot < out[j].Slot
		}
		return out[i].Stage < out[j].Stage
	})
	return out
}

func normalizeDecisionEvidence(evidence DecisionEvidence) DecisionEvidence {
	evidence.Stage = strings.ToLower(slugIdent(strings.TrimSpace(evidence.Stage)))
	evidence.Slot = strings.TrimSpace(evidence.Slot)
	evidence.Value = strings.TrimSpace(evidence.Value)
	evidence.Source = normalizeMappingSource(evidence.Source)
	evidence.Confidence = normalizeMappingConfidence(evidence.Confidence)
	evidence.Reason = truncateForPrompt(strings.TrimSpace(evidence.Reason), 240)
	evidence.Evidence = truncateForPrompt(strings.TrimSpace(evidence.Evidence), 240)
	evidence.Alternatives = normalizeDecisionAlternatives(evidence.Alternatives)
	return evidence
}

func normalizeDecisionAlternatives(alternatives []DecisionAlternative) []DecisionAlternative {
	seen := map[string]bool{}
	out := make([]DecisionAlternative, 0, len(alternatives))
	for _, alternative := range alternatives {
		alternative.Value = strings.TrimSpace(alternative.Value)
		alternative.Reason = truncateForPrompt(strings.TrimSpace(alternative.Reason), 160)
		if alternative.Value == "" || seen[alternative.Value] {
			continue
		}
		seen[alternative.Value] = true
		out = append(out, alternative)
		if len(out) >= 5 {
			break
		}
	}
	return out
}

func mergeDecisionEvidenceRecord(base, overlay DecisionEvidence) DecisionEvidence {
	if base.Confidence != mappingConfidenceConflict && overlay.Confidence == mappingConfidenceConflict {
		base.Confidence = mappingConfidenceConflict
	}
	if base.Confidence == "" {
		base.Confidence = overlay.Confidence
	}
	if base.Reason == "" {
		base.Reason = overlay.Reason
	}
	if base.Evidence == "" {
		base.Evidence = overlay.Evidence
	}
	base.Alternatives = normalizeDecisionAlternatives(append(base.Alternatives, overlay.Alternatives...))
	base.RequiresConfirmation = base.RequiresConfirmation || overlay.RequiresConfirmation
	return base
}

func pruneSupersededDecisionEvidence(evidence []DecisionEvidence, user DecisionEvidence) []DecisionEvidence {
	out := evidence[:0]
	for _, record := range evidence {
		record = normalizeDecisionEvidence(record)
		if record.Stage == user.Stage && record.Slot == user.Slot && record.Source != mappingSourceUser {
			continue
		}
		out = append(out, record)
	}
	return out
}

func decisionStageForSlot(slot string) string {
	switch {
	case strings.HasPrefix(slot, "sources.") || slot == "intent.source" || slot == "intent.openapi":
		return decisionStageCatalogPlan
	case strings.HasSuffix(slot, ".operation") || slot == "intent.steps":
		return decisionStageOperation
	case strings.Contains(slot, ".with.") || strings.HasSuffix(slot, ".with") || strings.Contains(slot, ".bind."):
		return decisionStageRequestMapping
	case strings.HasPrefix(slot, "intent.outputs.") || strings.HasPrefix(slot, "outputs."):
		return decisionStageOutput
	case slot == "side_effect_scope" || slot == "safety":
		return decisionStageSideEffect
	case strings.HasPrefix(slot, "draft_review."):
		return decisionStageDraftReview
	default:
		return "authoring"
	}
}

func decisionEvidenceIssues(session Session) []ReadinessIssue {
	var issues []ReadinessIssue
	classificationSlots := map[string]bool{}
	for _, classification := range normalizeMappingClassifications(session.Classifications) {
		if classification.Confidence == mappingConfidenceLow || classification.Confidence == mappingConfidenceConflict {
			classificationSlots[classification.Slot] = true
		}
	}
	for _, evidence := range normalizeDecisionEvidenceList(session.DecisionEvidence) {
		if classificationSlots[evidence.Slot] {
			continue
		}
		switch evidence.Confidence {
		case mappingConfidenceConflict:
			issues = append(issues, ReadinessIssue{
				Code:            "conflicting_decision_evidence",
				Slot:            evidence.Slot,
				Severity:        readinessBlocking,
				Message:         "Decision " + evidence.Slot + " has conflicting evidence and needs confirmation.",
				SuggestedAnswer: evidence.Value,
			})
		case mappingConfidenceLow:
			issues = append(issues, ReadinessIssue{
				Code:            "low_confidence_decision",
				Slot:            evidence.Slot,
				Severity:        readinessBlocking,
				Message:         "Decision " + evidence.Slot + " needs confirmation because its evidence is low confidence.",
				SuggestedAnswer: evidence.Value,
			})
		}
	}
	return issues
}

func normalizeDecisionEvidenceList(in []DecisionEvidence) []DecisionEvidence {
	return mergeDecisionEvidence(in, nil)
}
