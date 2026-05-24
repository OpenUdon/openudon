package elicitor

import (
	"strings"
	"unicode"

	"github.com/OpenUdon/apitools"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

const readinessUnconfirmedSideEffectCommitment = "unconfirmed_side_effect_commitment"

func unconfirmedSideEffectCommitmentIssue(session Session, docs []APIDocument, step *rollout.Step, op *apitools.OperationSummary) (ReadinessIssue, bool) {
	if step == nil {
		return ReadinessIssue{}, false
	}
	if op != nil {
		if !operationLooksSideEffectful(*op) ||
			!goalNeedsSideEffectCommitment(session, step) ||
			sideEffectCommitmentExplicit(session, step) ||
			userConfirmedOperation(session, step, op.OperationID) ||
			!inferredOperationNeedsCommitmentConfirmation(session, step, op.OperationID) {
			return ReadinessIssue{}, false
		}
		return sideEffectCommitmentReadinessIssue(session, docs, step, op.OperationID), true
	}
	choices := rankedOperationChoicesForStep(session, docs, step)
	if !goalNeedsSideEffectCommitment(session, step) {
		return ReadinessIssue{}, false
	}
	suggested, found := likelySideEffectOperationID(choices)
	if !found || sideEffectCommitmentExplicit(session, step) {
		return ReadinessIssue{}, false
	}
	return sideEffectCommitmentReadinessIssue(session, docs, step, suggested), true
}

func sideEffectCommitmentReadinessIssue(session Session, docs []APIDocument, step *rollout.Step, suggested string) ReadinessIssue {
	name := firstNonEmpty(step.Name, "this step")
	return ReadinessIssue{
		Code:            readinessUnconfirmedSideEffectCommitment,
		Slot:            "steps." + name + ".operation",
		Severity:        readinessBlocking,
		Message:         "The workflow goal suggests a side-effectful provider action for " + name + ", but the wording is ambiguous. Confirm the explicit provider and action before selecting an operationId. " + operationChoiceHintForStep(session, docs, step),
		SuggestedAnswer: suggested,
	}
}

func likelySideEffectOperationID(choices []rankedOperationChoice) (string, bool) {
	if operationID, ok := uniqueEmailMessageSendOperationID(choices); ok {
		return operationID, true
	}
	selected := ""
	matches := 0
	for _, choice := range choices {
		if !operationLooksSideEffectful(choice.Op) {
			continue
		}
		matches++
		if selected == "" {
			selected = strings.TrimSpace(choice.Op.OperationID)
		}
	}
	if matches == 1 {
		return selected, true
	}
	return "", matches > 0
}

func operationLooksSideEffectful(op apitools.OperationSummary) bool {
	if operationLooksEmailMessageSend(op) {
		return true
	}
	switch strings.ToUpper(strings.TrimSpace(op.Method)) {
	case "POST", "PUT", "PATCH", "DELETE":
		return true
	}
	tokens := operationTextTokens(op)
	for _, token := range []string{"send", "sent", "email", "mail", "post", "create", "update", "delete", "upload", "notify", "notification", "invite", "share", "publish"} {
		if tokens[token] > 0 {
			return true
		}
	}
	return false
}

func goalNeedsSideEffectCommitment(session Session, step *rollout.Step) bool {
	text := workflowGoalText(session)
	if goalContainsSideEffectAction(text) {
		return true
	}
	return providerAsVerbAmbiguity(text, providerCommitmentAliases(step))
}

func sideEffectCommitmentExplicit(session Session, step *rollout.Step) bool {
	text := workflowGoalText(session)
	if !goalContainsSideEffectAction(text) {
		return false
	}
	aliases := providerCommitmentAliases(step)
	if len(aliases) == 0 || !goalMentionsProvider(text, aliases) {
		return false
	}
	tokens := commitmentTokens(text)
	for i, token := range tokens {
		if !providerAliasToken(token, aliases) {
			continue
		}
		if hasConnectorBefore(tokens, i) || actionBeforeProvider(tokens, i) || providerBeforeAction(tokens, i) {
			return true
		}
	}
	return false
}

func workflowGoalText(session Session) string {
	var parts []string
	if session.Intent.Workflow != nil {
		parts = append(parts, session.Intent.Workflow.Description)
	}
	parts = append(parts, session.Project.Goal)
	return strings.Join(parts, " ")
}

func goalContainsSideEffectAction(text string) bool {
	for _, token := range commitmentTokens(text) {
		if sideEffectActionToken(token) {
			return true
		}
	}
	return false
}

func providerAsVerbAmbiguity(text string, aliases []string) bool {
	if len(aliases) == 0 || goalContainsSideEffectAction(text) {
		return false
	}
	tokens := commitmentTokens(text)
	for i, token := range tokens {
		if !providerAliasToken(token, aliases) {
			continue
		}
		if i == 0 || previousProviderVerbBoundary(tokens, i) || nextProviderVerbObject(tokens, i) {
			return true
		}
	}
	return false
}

func goalMentionsProvider(text string, aliases []string) bool {
	for _, token := range commitmentTokens(text) {
		if providerAliasToken(token, aliases) {
			return true
		}
	}
	return false
}

func providerCommitmentAliases(step *rollout.Step) []string {
	if step == nil {
		return nil
	}
	var out []string
	for _, value := range []string{step.Provider, step.Name, step.Source, step.OpenAPI} {
		for _, token := range rankingTokens(value) {
			if token == "" || sideEffectActionToken(token) || genericProviderToken(token) {
				continue
			}
			out = append(out, token)
		}
	}
	return dedupeStrings(out)
}

func providerAliasToken(token string, aliases []string) bool {
	token = strings.TrimSpace(token)
	if token == "" {
		return false
	}
	for _, alias := range aliases {
		if token == alias {
			return true
		}
	}
	return false
}

func sideEffectActionToken(token string) bool {
	switch token {
	case "send", "sent", "email", "mail", "message", "post", "create", "update", "delete", "upload", "notify", "share", "invite", "publish":
		return true
	default:
		return false
	}
}

func genericProviderToken(token string) bool {
	switch token {
	case "api", "openapi", "google", "discovery", "aws", "smithy", "report", "summary", "weather", "workflow", "step":
		return true
	default:
		return false
	}
}

func hasConnectorBefore(tokens []string, providerIndex int) bool {
	start := providerIndex - 4
	if start < 0 {
		start = 0
	}
	for i := start; i < providerIndex; i++ {
		switch tokens[i] {
		case "using", "via", "through", "with", "to", "in", "on", "by":
			return true
		}
	}
	return false
}

func actionBeforeProvider(tokens []string, providerIndex int) bool {
	start := providerIndex - 4
	if start < 0 {
		start = 0
	}
	for i := start; i < providerIndex; i++ {
		if sideEffectActionToken(tokens[i]) {
			return true
		}
	}
	return false
}

func providerBeforeAction(tokens []string, providerIndex int) bool {
	end := providerIndex + 5
	if end > len(tokens) {
		end = len(tokens)
	}
	seenConnector := false
	for i := providerIndex + 1; i < end; i++ {
		if tokens[i] == "to" || tokens[i] == "for" {
			seenConnector = true
			continue
		}
		if seenConnector && sideEffectActionToken(tokens[i]) {
			return true
		}
	}
	return false
}

func previousProviderVerbBoundary(tokens []string, index int) bool {
	if index == 0 {
		return true
	}
	switch tokens[index-1] {
	case "and", "then", "also":
		return true
	default:
		return false
	}
}

func nextProviderVerbObject(tokens []string, index int) bool {
	if index+1 >= len(tokens) {
		return false
	}
	switch tokens[index+1] {
	case "me", "us", "him", "her", "them", "it", "the", "a", "an", "this", "that", "my", "our":
		return true
	default:
		return false
	}
}

func commitmentTokens(text string) []string {
	var normalized []rune
	for _, r := range strings.ToLower(text) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			normalized = append(normalized, r)
		} else {
			normalized = append(normalized, ' ')
		}
	}
	return strings.Fields(string(normalized))
}

func userConfirmedOperation(session Session, step *rollout.Step, operationID string) bool {
	slot := stepOperationSlot(step)
	for _, classification := range normalizeMappingClassifications(session.Classifications) {
		if classification.Slot == slot && classification.Value == operationID && classification.Source == mappingSourceUser {
			return true
		}
	}
	for _, evidence := range normalizeDecisionEvidenceList(session.DecisionEvidence) {
		if evidence.Slot == slot && evidence.Value == operationID && evidence.Source == mappingSourceUser {
			return true
		}
	}
	return false
}

func inferredOperationNeedsCommitmentConfirmation(session Session, step *rollout.Step, operationID string) bool {
	slot := stepOperationSlot(step)
	for _, classification := range normalizeMappingClassifications(session.Classifications) {
		if classification.Slot == slot && classification.Value == operationID && classification.Source != mappingSourceUser && classification.RequiresConfirmation {
			return true
		}
	}
	for _, evidence := range normalizeDecisionEvidenceList(session.DecisionEvidence) {
		if evidence.Slot == slot && evidence.Value == operationID && evidence.Source != mappingSourceUser && evidence.RequiresConfirmation {
			return true
		}
	}
	return false
}
