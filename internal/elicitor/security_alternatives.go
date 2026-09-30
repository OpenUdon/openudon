package elicitor

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/OpenUdon/apitools"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

const securityAlternativeMetadataPrefix = "security_alternative."

func securityAlternativeSlot(step *rollout.Step) string {
	return "steps." + firstNonEmpty(stepName(step), "step") + ".security_alternative"
}

func securityAlternativeMetadataKey(step *rollout.Step) string {
	return securityAlternativeMetadataPrefix + firstNonEmpty(stepName(step), "step")
}

func stepName(step *rollout.Step) string {
	if step == nil {
		return ""
	}
	return strings.TrimSpace(step.Name)
}

func securityAlternativeLabel(set apitools.SecurityRequirementSetSummary) string {
	if len(set.Requirements) == 0 {
		return "anonymous"
	}
	names := make([]string, 0, len(set.Requirements))
	for _, requirement := range set.Requirements {
		name := firstNonEmpty(requirement.Name, requirement.ParameterName, requirement.Scheme, requirement.Type)
		if name != "" {
			name = strings.TrimSpace(name)
			scopes := dedupeStrings(requirement.Scopes)
			sort.Strings(scopes)
			if len(scopes) > 0 {
				name += "[" + strings.Join(scopes, ",") + "]"
			}
			names = append(names, name)
		}
	}
	names = dedupeStrings(names)
	sort.Strings(names)
	if len(names) == 0 {
		return "unnamed credential requirement"
	}
	return strings.Join(names, " + ")
}

func securityAlternativeChoices(op *apitools.OperationSummary) []string {
	if op == nil {
		return nil
	}
	out := make([]string, 0, len(op.SecurityRequirementSets))
	for index, set := range op.SecurityRequirementSets {
		out = append(out, fmt.Sprintf("%d=%s", index+1, securityAlternativeLabel(set)))
	}
	return out
}

func selectedSecurityAlternative(session Session, step *rollout.Step, op *apitools.OperationSummary) (apitools.SecurityRequirementSetSummary, bool) {
	if op == nil || len(op.SecurityRequirementSets) == 0 {
		return apitools.SecurityRequirementSetSummary{}, true
	}
	if len(op.SecurityRequirementSets) == 1 {
		return op.SecurityRequirementSets[0], true
	}
	selected := ""
	if session.Interview.Metadata != nil {
		selected = strings.TrimSpace(session.Interview.Metadata[securityAlternativeMetadataKey(step)])
	}
	if selected == "" {
		return apitools.SecurityRequirementSetSummary{}, false
	}
	for _, set := range op.SecurityRequirementSets {
		if selected == securityAlternativeFingerprint(set) {
			return set, true
		}
	}
	if index, err := strconv.Atoi(selected); err == nil && index > 0 && index <= len(op.SecurityRequirementSets) &&
		legacySecurityAlternativeConfirmed(session, step, op, index-1) {
		return op.SecurityRequirementSets[index-1], true
	}
	return apitools.SecurityRequirementSetSummary{}, false
}

func securityAlternativeFingerprint(set apitools.SecurityRequirementSetSummary) string {
	normalized := set
	normalized.Requirements = append([]apitools.SecuritySummary(nil), set.Requirements...)
	for i := range normalized.Requirements {
		requirement := &normalized.Requirements[i]
		requirement.Flows = sortedSecurityStrings(requirement.Flows)
		requirement.Scopes = sortedSecurityStrings(requirement.Scopes)
		requirement.OAuthFlows = append([]apitools.OAuthFlowSummary(nil), requirement.OAuthFlows...)
		for j := range requirement.OAuthFlows {
			requirement.OAuthFlows[j].Scopes = sortedSecurityStrings(requirement.OAuthFlows[j].Scopes)
		}
		sort.SliceStable(requirement.OAuthFlows, func(a, b int) bool {
			left, _ := json.Marshal(requirement.OAuthFlows[a])
			right, _ := json.Marshal(requirement.OAuthFlows[b])
			return string(left) < string(right)
		})
	}
	sort.SliceStable(normalized.Requirements, func(i, j int) bool {
		left, _ := json.Marshal(normalized.Requirements[i])
		right, _ := json.Marshal(normalized.Requirements[j])
		return string(left) < string(right)
	})
	data, _ := json.Marshal(normalized)
	digest := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func sortedSecurityStrings(values []string) []string {
	out := append([]string(nil), values...)
	sort.Strings(out)
	return out
}

func legacySecurityAlternativeConfirmed(session Session, step *rollout.Step, op *apitools.OperationSummary, index int) bool {
	if index < 0 || index >= len(op.SecurityRequirementSets) {
		return false
	}
	label := securityAlternativeLabel(op.SecurityRequirementSets[index])
	matches := 0
	for _, set := range op.SecurityRequirementSets {
		if strings.EqualFold(label, securityAlternativeLabel(set)) {
			matches++
		}
	}
	if matches != 1 {
		return false
	}
	evidenceMatches := 0
	for _, evidence := range session.DecisionEvidence {
		evidence = normalizeDecisionEvidence(evidence)
		if evidence.Slot == securityAlternativeSlot(step) && strings.EqualFold(evidence.Value, label) &&
			evidence.Source == mappingSourceUser && evidence.Confidence == mappingConfidenceHigh && !evidence.RequiresConfirmation {
			evidenceMatches++
		}
	}
	return evidenceMatches == 1
}

func selectSecurityAlternative(session *Session, step *rollout.Step, op *apitools.OperationSummary, answer string) bool {
	if session == nil || step == nil || op == nil || len(op.SecurityRequirementSets) < 2 {
		return false
	}
	answer = strings.TrimSpace(answer)
	selectedIndex := -1
	if index, err := strconv.Atoi(answer); err == nil && index > 0 && index <= len(op.SecurityRequirementSets) {
		selectedIndex = index - 1
	} else {
		for index, set := range op.SecurityRequirementSets {
			label := securityAlternativeLabel(set)
			if strings.EqualFold(answer, label) {
				if selectedIndex >= 0 {
					return false
				}
				selectedIndex = index
			}
		}
	}
	if selectedIndex < 0 {
		return false
	}
	selected := securityAlternativeLabel(op.SecurityRequirementSets[selectedIndex])
	if session.Interview.Metadata == nil {
		session.Interview.Metadata = map[string]string{}
	}
	session.Interview.Metadata[securityAlternativeMetadataKey(step)] = securityAlternativeFingerprint(op.SecurityRequirementSets[selectedIndex])
	addDecisionEvidence(session, DecisionEvidence{
		Stage:      decisionStageOperation,
		Slot:       securityAlternativeSlot(step),
		Value:      selected,
		Source:     mappingSourceUser,
		Confidence: mappingConfidenceHigh,
		Evidence:   answer,
		Reason:     "User selected one OpenAPI security alternative; requirements within it remain conjunctive.",
	})
	return true
}

func selectedSecurityCredentialFields(session Session, step *rollout.Step, op *apitools.OperationSummary) ([]string, bool) {
	set, ok := selectedSecurityAlternative(session, step, op)
	if !ok {
		return nil, false
	}
	var fields []string
	for _, requirement := range set.Requirements {
		if field := apitools.SecurityCredentialFieldName(requirement); field != "" {
			fields = append(fields, field)
		}
	}
	return dedupeStrings(fields), true
}

func allSecurityRequirements(op *apitools.OperationSummary) []apitools.SecuritySummary {
	if op == nil {
		return nil
	}
	var out []apitools.SecuritySummary
	for _, set := range op.SecurityRequirementSets {
		out = append(out, set.Requirements...)
	}
	return out
}
