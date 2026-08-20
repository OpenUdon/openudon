package elicitor

import (
	"sort"
	"strings"

	"github.com/OpenUdon/apitools"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

type openAPIRequestFieldInfo struct {
	Type string
	Body bool
}

func openAPIRequestFieldTypes(session Session, step *rollout.Step, op *apitools.OperationSummary) map[string]openAPIRequestFieldInfo {
	out := map[string]openAPIRequestFieldInfo{}
	if op == nil {
		return out
	}
	for _, parameter := range op.Parameters {
		if field := strings.TrimSpace(parameter.Name); field != "" {
			out[field] = openAPIRequestFieldInfo{Type: parameter.Type}
		}
	}
	if op.RequestBody != nil {
		for _, bodyField := range op.RequestBody.Fields {
			if field := strings.TrimSpace(bodyField.Path); field != "" {
				out[field] = openAPIRequestFieldInfo{Type: bodyField.Type, Body: true}
			}
		}
	}
	if fields, selected := selectedSecurityCredentialFields(session, step, op); selected {
		for _, field := range fields {
			out[field] = openAPIRequestFieldInfo{Type: "string"}
		}
	}
	for _, field := range apitools.RequiredRequestFields(*op) {
		if strings.TrimSpace(field) == "" {
			continue
		}
		if _, ok := out[field]; !ok {
			out[field] = openAPIRequestFieldInfo{Type: "string"}
		}
	}
	return out
}

func invalidBodyPath(field string, fields map[string]openAPIRequestFieldInfo) bool {
	if !strings.Contains(field, ".") && !strings.Contains(field, "[]") && !strings.HasPrefix(field, "body") {
		return false
	}
	for _, info := range fields {
		if info.Body {
			return true
		}
	}
	return false
}

func incompatibleLiteralType(source, wantType string) bool {
	source = strings.TrimSpace(source)
	wantType = strings.ToLower(strings.TrimSpace(wantType))
	if source == "" || wantType == "" || expressionLikeSource(source) {
		return false
	}
	switch wantType {
	case "string":
		return isBoolLiteral(source) || isNumberLiteral(source)
	case "integer", "int":
		return !isIntegerLiteral(source)
	case "number", "float", "double":
		return !isNumberLiteral(source)
	case "boolean", "bool":
		return !isBoolLiteral(source)
	default:
		return false
	}
}

func expressionLikeSource(source string) bool {
	source = strings.TrimSpace(source)
	return strings.HasPrefix(source, "inputs.") ||
		strings.HasPrefix(source, "credentials.") ||
		strings.HasPrefix(source, "credential.") ||
		strings.HasPrefix(source, "received_body") ||
		strings.Contains(source, ".received_") ||
		strings.Contains(source, ".body") ||
		strings.HasPrefix(source, "${")
}

func isBoolLiteral(value string) bool {
	return strings.EqualFold(value, "true") || strings.EqualFold(value, "false")
}

func isIntegerLiteral(value string) bool {
	if value == "" {
		return false
	}
	value = strings.TrimPrefix(strings.TrimPrefix(value, "-"), "+")
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func isNumberLiteral(value string) bool {
	if value == "" {
		return false
	}
	seenDigit := false
	seenDot := false
	value = strings.TrimPrefix(strings.TrimPrefix(value, "-"), "+")
	if value == "" {
		return false
	}
	for _, r := range value {
		switch {
		case r >= '0' && r <= '9':
			seenDigit = true
		case r == '.' && !seenDot:
			seenDot = true
		default:
			return false
		}
	}
	return seenDigit
}

func operationNeedsCredential(op *apitools.OperationSummary) bool {
	return op != nil && apitools.OperationNeedsCredential(*op)
}

func operationNeedsCredentialForStep(session Session, step *rollout.Step, op *apitools.OperationSummary) bool {
	selected, ok := selectedSecurityAlternative(session, step, op)
	if !ok {
		return operationNeedsCredential(op)
	}
	return len(selected.Requirements) > 0
}

func missingRuntimeInputs(session Session) []string {
	declared := map[string]bool{}
	for _, input := range session.Intent.Inputs {
		if input != nil {
			declared[input.Name] = true
		}
	}
	used := map[string]bool{}
	walkSteps(session.Intent.Steps, func(step *rollout.Step) {
		for _, source := range step.With {
			if name := strings.TrimPrefix(strings.TrimSpace(source), "inputs."); name != source && name != "" {
				used[name] = true
			}
		}
	})
	var missing []string
	for name := range used {
		if !declared[name] {
			missing = append(missing, name)
		}
	}
	sort.Strings(missing)
	return missing
}

func referencesOptionalControls(session Session) bool {
	text := strings.ToLower(strings.Join([]string{
		session.Project.Goal,
		session.Project.DataFlow,
		session.Project.FunctionContracts,
		session.Project.Safety,
		session.Safety,
		session.Fallback,
	}, "\n"))
	return strings.Contains(text, "timeout") || strings.Contains(text, "idempot")
}

func firstBlockingIssue(issues []ReadinessIssue) ReadinessIssue {
	for _, issue := range issues {
		if issue.Severity == readinessBlocking {
			return issue
		}
	}
	return ReadinessIssue{}
}

func sortReadinessIssues(issues []ReadinessIssue) []ReadinessIssue {
	priority := map[string]int{
		"missing_goal":                           0,
		"inline_secret_value":                    1,
		"unsafe_review_bypass":                   2,
		"conflicting_mapping":                    3,
		"low_confidence_mapping":                 4,
		"missing_api_doc":                        5,
		readinessUnconfirmedSideEffectCommitment: 6,
		"missing_operation":                      7,
		"missing_browser_session_posture":        8,
		"unconfirmed_browser_mutation":           9,
		"missing_runtime_inputs":                 10,
		"undeclared_credential_reference":        11,
		"invented_request_field":                 12,
		"invalid_request_body_path":              13,
		"incompatible_request_value_type":        14,
		"missing_required_request_values":        15,
		"missing_credential_bindings":            16,
		"missing_outputs":                        17,
		"missing_side_effect_policy":             18,
		"optional_timeout_idempotency_controls":  19,
		"intent_render_invalid":                  20,
	}
	sort.SliceStable(issues, func(i, j int) bool {
		left, ok := priority[issues[i].Code]
		if !ok {
			left = len(priority)
		}
		right, ok := priority[issues[j].Code]
		if !ok {
			right = len(priority)
		}
		return left < right
	})
	return issues
}

func suggestedAnswerForCode(code string, session Session, docs []APIDocument) string {
	switch code {
	case "missing_api_doc":
		return suggestedAPIDocAnswer(session, docs)
	case "missing_operation":
		return suggestedOperationAnswer(docs)
	case "missing_outputs":
		return suggestedOutputAnswer(session)
	case "missing_side_effect_policy":
		return suggestedPolicyAnswer(session)
	default:
		return ""
	}
}

func suggestedAPIDocAnswer(session Session, docs []APIDocument) string {
	if len(missingLocalAPIDocumentRefs(session, docs)) > 0 {
		return "Generate/provide the missing API artifact, then rerun iCoT."
	}
	if len(docs) > 0 {
		if hints := CatalogHintsForSession(session); len(hints) > 0 {
			if len(catalogProvidersMissingLocalDocs(hints, docs)) > 0 {
				return "Generate/provide the missing API artifact, then rerun iCoT."
			}
		}
		return preferredSourceDocument(session, docs).RelativePath
	}
	if hints := CatalogHintsForSession(session); len(CatalogProvidersWithMigratableDocs(hints, "")) > 0 {
		return "yes"
	}
	return suggestedDocAnswer(docs)
}

func suggestedDocAnswer(docs []APIDocument) string {
	if len(docs) == 0 {
		return "openapi/api.yaml"
	}
	return docs[0].RelativePath
}

func preferredSourceDocument(session Session, docs []APIDocument) APIDocument {
	if len(docs) == 0 {
		return APIDocument{}
	}
	query := rankingTokenWeights(strings.Join([]string{session.Boundary.Outcome, session.Project.Goal, workflowDescription(session)}, " "))
	bestAPI, bestBrowser := -1, -1
	var apiDoc, browserDoc APIDocument
	for _, doc := range docs {
		score := 0
		for _, operation := range doc.Operations {
			candidate := rankingMatchScore(query, strings.Join([]string{operation.OperationID, operation.Summary, operation.Description, operation.Path, operationMethodHints(operation.Method)}, " "), 1)
			if candidate > score {
				score = candidate
			}
		}
		if isBrowserDocument(doc) {
			if score > bestBrowser {
				bestBrowser, browserDoc = score, doc
			}
		} else if score > bestAPI {
			bestAPI, apiDoc = score, doc
		}
	}
	if apiDoc.RelativePath != "" && bestAPI > 0 {
		return apiDoc
	}
	if browserDoc.RelativePath != "" && bestBrowser > 0 && bestAPI <= 0 {
		return browserDoc
	}
	if apiDoc.RelativePath != "" {
		return apiDoc
	}
	return browserDoc
}

func suggestedOperationAnswer(docs []APIDocument) string {
	var selected string
	for _, doc := range docs {
		for _, op := range doc.Operations {
			if op.OperationID != "" {
				if selected != "" {
					return ""
				}
				selected = op.OperationID
			}
		}
	}
	return selected
}

func suggestedOperationAnswerForStep(session Session, docs []APIDocument, step *rollout.Step) string {
	choices := rankedOperationChoicesForStep(session, docs, step)
	if len(choices) != 1 {
		intentText := operationSelectionRankingText(session, step)
		if wantsEmailMessageSend(intentText) {
			if operationID, ok := uniqueEmailMessageSendOperationID(choices); ok {
				if !sideEffectCommitmentExplicit(session, step) {
					return ""
				}
				return operationID
			}
		}
		if len(choices) > 1 && confidentOperationDefault(session, step, choices) {
			return choices[0].Op.OperationID
		}
		return ""
	}
	if operationLooksSideEffectful(choices[0].Op) && goalNeedsSideEffectCommitment(session, step) && !sideEffectCommitmentExplicit(session, step) {
		return ""
	}
	return choices[0].Op.OperationID
}

func confidentOperationDefault(session Session, step *rollout.Step, choices []rankedOperationChoice) bool {
	if len(choices) < 2 || strings.TrimSpace(choices[0].Op.OperationID) == "" || choices[0].Score <= 0 {
		return false
	}
	intentText := operationSelectionRankingText(session, step)
	if wantsEmailMessageSend(intentText) && uniqueEmailMessageSendChoice(choices) {
		return sideEffectCommitmentExplicit(session, step)
	}
	if wantsWeatherLookup(intentText) {
		return uniqueWeatherLookupChoice(choices) || preferredCurrentWeatherLookupChoice(choices)
	}
	return false
}

func wantsEmailMessageSend(text string) bool {
	tokens := rankingTokenWeights(text)
	if tokens["send"] == 0 && tokens["mail"] == 0 && tokens["email"] == 0 {
		return false
	}
	return tokens["gmail"] > 0 || tokens["email"] > 0 || tokens["mail"] > 0 || tokens["message"] > 0
}

func uniqueEmailMessageSendChoice(choices []rankedOperationChoice) bool {
	bestID := strings.TrimSpace(choices[0].Op.OperationID)
	if bestID == "" || !operationLooksEmailMessageSend(choices[0].Op) {
		return false
	}
	operationID, ok := uniqueEmailMessageSendOperationID(choices)
	return ok && operationID == bestID
}

func uniqueEmailMessageSendOperationID(choices []rankedOperationChoice) (string, bool) {
	var selected string
	matches := 0
	for _, choice := range choices {
		if operationLooksEmailMessageSend(choice.Op) {
			matches++
			operationID := strings.TrimSpace(choice.Op.OperationID)
			if selected == "" {
				selected = operationID
			} else if operationID != selected {
				return "", false
			}
		}
	}
	return selected, matches == 1 && selected != ""
}

func wantsWeatherLookup(text string) bool {
	tokens := rankingTokenWeights(text)
	return tokens["weather"] > 0
}

func uniqueWeatherLookupChoice(choices []rankedOperationChoice) bool {
	bestID := strings.TrimSpace(choices[0].Op.OperationID)
	if bestID == "" || !operationLooksWeatherLookup(choices[0].Op) {
		return false
	}
	matches := 0
	for _, choice := range choices {
		if operationLooksWeatherLookup(choice.Op) {
			matches++
			if strings.TrimSpace(choice.Op.OperationID) != bestID {
				return false
			}
		}
	}
	return matches == 1
}

func preferredCurrentWeatherLookupChoice(choices []rankedOperationChoice) bool {
	if len(choices) == 0 || !operationLooksCurrentWeatherLookup(choices[0].Op) {
		return false
	}
	for _, choice := range choices[1:] {
		if choice.Score >= choices[0].Score && operationLooksWeatherLookup(choice.Op) && !operationLooksCurrentWeatherLookup(choice.Op) {
			return false
		}
	}
	return true
}

func operationLooksEmailMessageSend(op apitools.OperationSummary) bool {
	text := strings.Join([]string{
		op.OperationID,
		op.Path,
		op.Summary,
		op.Description,
		strings.Join(op.Tags, " "),
	}, " ")
	lower := strings.ToLower(text)
	if strings.Contains(lower, "does not send") || strings.Contains(lower, "not send") {
		return false
	}
	tokens := rankingTokenWeights(text)
	if tokens["send"] == 0 || (tokens["message"] == 0 && tokens["mail"] == 0 && tokens["email"] == 0) {
		return false
	}
	for _, token := range []string{"setting", "settings", "alias", "aliases", "verify", "smime", "cse", "identity", "identities", "draft", "drafts", "import", "insert"} {
		if tokens[token] > 0 {
			return false
		}
	}
	return true
}

func operationLooksWeatherLookup(op apitools.OperationSummary) bool {
	tokens := operationTextTokens(op)
	if tokens["weather"] == 0 {
		return false
	}
	if tokens["geocode"] > 0 || tokens["reverse"] > 0 || tokens["zip"] > 0 {
		return false
	}
	return tokens["get"] > 0 || tokens["current"] > 0 || tokens["forecast"] > 0 || tokens["condition"] > 0 || tokens["conditions"] > 0 || tokens["data"] > 0
}

func operationLooksCurrentWeatherLookup(op apitools.OperationSummary) bool {
	tokens := operationTextTokens(op)
	return operationLooksWeatherLookup(op) && tokens["current"] > 0
}

func operationTextTokens(op apitools.OperationSummary) map[string]int {
	return rankingTokenWeights(strings.Join([]string{
		op.OperationID,
		op.Path,
		op.Summary,
		op.Description,
		strings.Join(op.Tags, " "),
	}, " "))
}

func missingOperationMessage(docs []APIDocument) string {
	return "Choose the API operationId or workflow action to run. " + operationChoiceHint(docs)
}

func missingOperationPrompt(session Session, docs []APIDocument, slot string) string {
	step := stepForOperationSlot(session, slot)
	if step == nil {
		return "Which API action or workflow step should run first? Choose a listed operationId when this is an API-backed SaaS step."
	}
	return "Which operationId should " + firstNonEmpty(step.Name, "this step") + " use? Choose one listed for its API document or provider. " + operationChoiceHintForStep(session, docs, step)
}

func stepForOperationSlot(session Session, slot string) *rollout.Step {
	if !strings.HasPrefix(slot, "steps.") || !strings.HasSuffix(slot, ".operation") {
		return nil
	}
	name := strings.TrimSuffix(strings.TrimPrefix(slot, "steps."), ".operation")
	for _, step := range session.Intent.Steps {
		if step != nil && firstNonEmpty(step.Name, "step") == name {
			return step
		}
	}
	return nil
}

func operationChoiceHintForStep(session Session, docs []APIDocument, step *rollout.Step) string {
	choices := rankedOperationChoicesForStep(session, docs, step)
	if len(choices) == 0 && step != nil {
		provider := firstNonEmpty(step.Provider, step.Name)
		if provider != "" {
			return "No local API document with operations is available for " + provider + "."
		}
	}
	if len(choices) > 0 {
		return operationChoicesHint(choices)
	}
	return operationChoiceHint(nil)
}

type rankedOperationChoice struct {
	Doc   APIDocument
	Op    apitools.OperationSummary
	Score int
}

func rankedOperationChoicesForStep(session Session, docs []APIDocument, step *rollout.Step) []rankedOperationChoice {
	if step == nil {
		return nil
	}
	filtered := filterDocsForStep(&session, docs, step)
	if len(filtered) == 0 {
		return nil
	}
	query := rankingTokenWeights(operationSelectionRankingText(session, step))
	var choices []rankedOperationChoice
	for _, doc := range filtered {
		selectedDoc := strings.TrimSpace(doc.RelativePath) != "" && doc.RelativePath == stepAPISourceRef(session, step)
		for _, op := range doc.Operations {
			if strings.TrimSpace(op.OperationID) == "" {
				continue
			}
			choices = append(choices, rankedOperationChoice{
				Doc:   doc,
				Op:    op,
				Score: operationRankScore(query, doc, op, selectedDoc),
			})
		}
	}
	sort.SliceStable(choices, func(i, j int) bool {
		if choices[i].Score != choices[j].Score {
			return choices[i].Score > choices[j].Score
		}
		if choices[i].Doc.RelativePath != choices[j].Doc.RelativePath {
			return choices[i].Doc.RelativePath < choices[j].Doc.RelativePath
		}
		return choices[i].Op.OperationID < choices[j].Op.OperationID
	})
	return choices
}
