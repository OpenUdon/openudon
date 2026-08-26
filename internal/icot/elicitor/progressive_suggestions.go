package elicitor

import (
	"fmt"
	"strings"

	"github.com/OpenUdon/apitools"
	"github.com/OpenUdon/openudon/internal/projectwizard"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

func operationSelectionRankingText(session Session, step *rollout.Step) string {
	var parts []string
	if session.Intent.Workflow != nil {
		parts = append(parts, session.Intent.Workflow.Name, session.Intent.Workflow.Description)
	}
	parts = append(parts,
		session.Project.Goal,
		session.Project.DataFlow,
		session.Project.Outputs,
		intentAPISourceRef(session.Intent),
	)
	if step != nil {
		parts = append(parts, step.Name, step.Do, step.Provider, firstNonEmpty(step.Source, step.OpenAPI))
		for field, value := range step.With {
			parts = append(parts, field, value)
		}
	}
	text := strings.Join(parts, " ")
	lower := strings.ToLower(text)
	if strings.Contains(lower, "gmail me") || strings.Contains(lower, "email me") || strings.Contains(lower, "mail me") || strings.Contains(lower, "send") {
		text += " send email mail message create"
	}
	if strings.Contains(lower, "weather") {
		text += " current weather forecast conditions"
	}
	return text
}

func operationChoicesHint(choices []rankedOperationChoice) string {
	if len(choices) == 0 {
		return "Add local OpenAPI metadata when this is an API-backed SaaS step."
	}
	var labels []string
	limit := len(choices)
	if limit > 12 {
		limit = 12
	}
	includeDocPath := multipleChoiceDocuments(choices[:limit])
	for _, choice := range choices[:limit] {
		label := choice.Op.OperationID
		if desc := firstNonEmpty(choice.Op.Summary, choice.Op.Description); desc != "" {
			label += " (" + truncateForPrompt(desc, 80) + ")"
		}
		if includeDocPath && choice.Doc.RelativePath != "" {
			label += " [" + choice.Doc.RelativePath + "]"
		}
		labels = append(labels, label)
	}
	suffix := "."
	if len(choices) > limit {
		suffix = fmt.Sprintf("; and %d more in local API metadata.", len(choices)-limit)
	}
	return "Available candidate operationIds: " + strings.Join(labels, "; ") + suffix
}

func multipleChoiceDocuments(choices []rankedOperationChoice) bool {
	var first string
	for _, choice := range choices {
		path := strings.TrimSpace(choice.Doc.RelativePath)
		if path == "" {
			continue
		}
		if first == "" {
			first = path
			continue
		}
		if path != first {
			return true
		}
	}
	return false
}

func operationChoiceHint(docs []APIDocument) string {
	var groups []string
	for _, doc := range docs {
		var choices []string
		for _, op := range doc.Operations {
			if strings.TrimSpace(op.OperationID) == "" {
				continue
			}
			label := op.OperationID
			if desc := firstNonEmpty(op.Summary, op.Description); desc != "" {
				label += " (" + truncateForPrompt(desc, 80) + ")"
			}
			choices = append(choices, label)
			if len(choices) >= 6 {
				break
			}
		}
		if len(choices) > 0 {
			groups = append(groups, doc.RelativePath+": "+strings.Join(choices, "; "))
		}
		if len(groups) >= 4 {
			break
		}
	}
	if len(groups) == 0 {
		return "Add local OpenAPI metadata when this is an API-backed SaaS step."
	}
	return "Available operationIds by API document: " + strings.Join(groups, " | ") + "."
}

func truncateForPrompt(value string, max int) string {
	value = strings.Join(strings.Fields(value), " ")
	if len(value) <= max {
		return value
	}
	if max <= 3 {
		return value[:max]
	}
	return strings.TrimSpace(value[:max-3]) + "..."
}

func suggestedFieldAssignments(session Session, docs []APIDocument, step *rollout.Step, op *apitools.OperationSummary, fields []string) string {
	var parts []string
	for _, field := range fields {
		parts = append(parts, field+"="+suggestedFieldSource(session, docs, step, op, field))
	}
	return strings.Join(parts, ", ")
}

func suggestedCredentialName(op *apitools.OperationSummary) string {
	if op == nil || len(op.SecurityRequirementSets) == 0 {
		return "api_token"
	}
	for _, set := range op.SecurityRequirementSets {
		for _, requirement := range set.Requirements {
			if name := strings.TrimSpace(requirement.Name); name != "" {
				return name
			}
		}
	}
	return "api_token"
}

func suggestedCredentialNameForOperation(session Session, docs []APIDocument, step *rollout.Step, op *apitools.OperationSummary) string {
	if selected, ok := selectedSecurityAlternative(session, step, op); ok {
		for _, requirement := range selected.Requirements {
			if name := strings.TrimSpace(requirement.Name); name != "" {
				return name
			}
		}
	}
	if name := suggestedCredentialName(op); name != "" && name != "api_token" {
		return name
	}
	if len(session.Credentials) == 1 {
		return session.Credentials[0]
	}
	doc, ok := documentForStep(session, docs, step, op)
	if ok {
		if name := credentialNameFromDocument(doc); name != "" {
			return name
		}
	}
	return suggestedCredentialName(op)
}

func suggestedFieldSource(session Session, docs []APIDocument, step *rollout.Step, op *apitools.OperationSummary, field string) string {
	field = strings.TrimSpace(field)
	if field == "" {
		return ""
	}
	if input, ok := exactInputMatch(session.Intent.Inputs, field); ok {
		return "inputs." + input
	}
	if input, ok := inputMatchByLeafOrDescription(session.Intent.Inputs, field, op); ok {
		return "inputs." + input
	}
	if suggestedCredentialField(field, op) {
		return "credentials." + suggestedCredentialNameForOperation(session, docs, step, op)
	}
	if value, ok := safeLiteralDefault(field, op); ok {
		return value
	}
	return "inputs." + slugIdent(field)
}

func inputMatchByLeafOrDescription(inputs []*rollout.Input, field string, op *apitools.OperationSummary) (string, bool) {
	leaf := slugIdent(fieldLeaf(field))
	for _, input := range inputs {
		if input == nil || strings.TrimSpace(input.Name) == "" {
			continue
		}
		if slugIdent(input.Name) == leaf {
			return input.Name, true
		}
	}
	description := requestFieldDescription(op, field)
	fieldTokens := rankingTokenWeights(strings.Join([]string{field, fieldLeaf(field), description}, " "))
	bestName := ""
	bestScore := 0
	for _, input := range inputs {
		if input == nil || strings.TrimSpace(input.Name) == "" {
			continue
		}
		score := rankingMatchScore(fieldTokens, input.Name+" "+input.Description, 1)
		if score > bestScore {
			bestScore = score
			bestName = input.Name
		} else if score == bestScore {
			bestName = ""
		}
	}
	if bestScore > 0 && bestName != "" {
		return bestName, true
	}
	return "", false
}

func fieldLeaf(field string) string {
	field = strings.TrimSpace(strings.ReplaceAll(field, "[]", ""))
	if idx := strings.LastIndex(field, "."); idx >= 0 {
		return field[idx+1:]
	}
	return field
}

func requestFieldDescription(op *apitools.OperationSummary, field string) string {
	if op == nil {
		return ""
	}
	for _, parameter := range op.Parameters {
		if parameter.Name == field {
			return parameter.Description
		}
	}
	if op.RequestBody != nil {
		for _, bodyField := range op.RequestBody.Fields {
			if bodyField.Path == field {
				return bodyField.Description
			}
		}
	}
	return ""
}

func suggestedCredentialField(field string, op *apitools.OperationSummary) bool {
	if op == nil || len(op.SecurityRequirementSets) == 0 {
		return false
	}
	for _, security := range allSecurityRequirements(op) {
		if field == apitools.SecurityCredentialFieldName(security) {
			return true
		}
	}
	return strings.EqualFold(field, "Authorization")
}

func safeLiteralDefault(field string, op *apitools.OperationSummary) (string, bool) {
	return "", false
}

func safeScalarType(typ string) bool {
	switch strings.ToLower(strings.TrimSpace(typ)) {
	case "", "string", "integer", "int", "number", "float", "double", "boolean", "bool":
		return true
	default:
		return false
	}
}

func documentForStep(session Session, docs []APIDocument, step *rollout.Step, op *apitools.OperationSummary) (APIDocument, bool) {
	if op == nil {
		return APIDocument{}, false
	}
	docPath := intentAPISourceRef(session.Intent)
	if step != nil {
		docPath = stepAPISourceRef(session, step)
	}
	for _, doc := range docs {
		if docPath != "" && doc.RelativePath != docPath {
			continue
		}
		for _, candidate := range doc.Operations {
			if candidate.OperationID == op.OperationID {
				return doc, true
			}
		}
	}
	if docPath == "" {
		for _, doc := range docs {
			for _, candidate := range doc.Operations {
				if candidate.OperationID == op.OperationID {
					return doc, true
				}
			}
		}
	}
	return APIDocument{}, false
}

func credentialNameFromDocument(doc APIDocument) string {
	base := slug(strings.TrimSuffix(strings.TrimSpace(doc.Title), " API"))
	if base == "" {
		path := strings.TrimSpace(doc.RelativePath)
		if idx := strings.LastIndex(path, "/"); idx >= 0 {
			path = path[idx+1:]
		}
		for _, ext := range []string{".yaml", ".yml", ".json"} {
			path = strings.TrimSuffix(path, ext)
		}
		base = slug(path)
	}
	if base == "" {
		return ""
	}
	if strings.HasSuffix(base, "_api") {
		return base + "_token"
	}
	return base + "_api_token"
}

func suggestedRuntimeInputs(inputs []string) string {
	var parts []string
	for _, input := range inputs {
		parts = append(parts, input+":string")
	}
	return strings.Join(parts, ", ")
}

func suggestedOutputAnswer(session Session) string {
	stepName := lastStepName(session.Intent.Steps)
	if stepName == "" {
		return "result"
	}
	return "result=" + stepName + ".received_body"
}

func suggestedPolicyAnswer(session Session) string {
	scope := session.SideEffectScope
	if scope == "" {
		if sessionAppearsReadOnly(session) {
			return projectwizard.SideEffectReadOnly
		}
		scope = projectwizard.SideEffectSandboxOnly
	}
	return scope
}

func sessionAppearsReadOnly(session Session) bool {
	foundExecutable := false
	for _, step := range session.Intent.Steps {
		if step == nil {
			continue
		}
		stepType := strings.ToLower(strings.TrimSpace(step.Type))
		operation := strings.ToLower(strings.TrimSpace(step.Operation))
		text := strings.Join([]string{stepType, operation, strings.ToLower(step.Name), strings.ToLower(step.Do)}, " ")
		switch stepType {
		case "http", "openapi":
			foundExecutable = true
			if operation != "" && readOnlyOperationName(operation) {
				continue
			}
			if containsMutationHint(text) {
				return false
			}
			if operation == "" {
				continue
			}
			return false
		case "browser":
			foundExecutable = true
			// Browser safety is defined by the selected profile action, not by
			// operation-name heuristics. Keep the automatic policy conservative;
			// readiness asks the author to confirm the actual posture.
			return false
		case "fnct", "":
			if containsMutationHint(text) {
				return false
			}
			if strings.TrimSpace(step.Name) != "" || strings.TrimSpace(step.Do) != "" {
				foundExecutable = true
			}
		default:
			foundExecutable = true
			return false
		}
	}
	return foundExecutable
}

func readOnlyOperationName(operation string) bool {
	operation = strings.ToLower(strings.TrimSpace(operation))
	for _, prefix := range []string{"get", "list", "read", "fetch", "search", "describe", "lookup"} {
		if strings.HasPrefix(operation, prefix) {
			return true
		}
	}
	return false
}

func containsMutationHint(text string) bool {
	for _, hint := range []string{"post", "send", "create", "update", "delete", "upload", "write", "archive", "notify", "approve", "deploy", "provision", "close", "modify"} {
		if strings.Contains(text, hint) {
			return true
		}
	}
	return false
}

func parseAssignments(value string) map[string]string {
	out := map[string]string{}
	for _, item := range splitList(value) {
		name, rest := splitNameRest(item)
		name = slugIdent(name)
		rest = strings.TrimSpace(rest)
		if name == "" || rest == "" {
			continue
		}
		out[name] = rest
	}
	return out
}

func fieldFromWithSlot(slot string) string {
	parts := strings.Split(strings.TrimSpace(slot), ".with.")
	if len(parts) != 2 {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func addInputsFromAssignments(session *Session, assignments map[string]string) {
	for _, source := range assignments {
		source = strings.TrimSpace(source)
		if !strings.HasPrefix(source, "inputs.") {
			continue
		}
		name := strings.TrimPrefix(source, "inputs.")
		session.Intent.Inputs = mergeInputsByName(session.Intent.Inputs, []*rollout.Input{{Name: name, Type: "string", Required: true}})
	}
}

func addCredentialsFromAssignments(session *Session, assignments map[string]string) {
	for _, source := range assignments {
		for _, credential := range credentialCandidates(source) {
			if credential == "" {
				continue
			}
			session.Credentials = dedupeStrings(append(session.Credentials, credential))
			session.CredentialsSet = true
			addMappingClassification(session, MappingClassification{
				Slot:                 "credentials",
				Value:                credential,
				Source:               mappingSourceUser,
				Confidence:           mappingConfidenceHigh,
				Evidence:             source,
				Reason:               "User accepted a request mapping that references this credential binding.",
				RequiresConfirmation: false,
			})
		}
	}
}

func fillCredentialFields(session *Session, docs []APIDocument, credential string) {
	walkSteps(session.Intent.Steps, func(step *rollout.Step) {
		op, ok := operationForStep(*session, docs, step)
		if !ok || !operationNeedsCredential(op) {
			return
		}
		if step.With == nil {
			step.With = map[string]string{}
		}
		securityFields, selected := selectedSecurityCredentialFields(*session, step, op)
		if !selected {
			return
		}
		for _, field := range securityFields {
			if step.With[field] == "" && looksCredentialField(field, op) {
				step.With[field] = "credentials." + credential
				addMappingClassification(session, MappingClassification{
					Slot:                 stepWithSlot(step, field),
					Value:                step.With[field],
					Source:               mappingSourceUser,
					Confidence:           mappingConfidenceHigh,
					Evidence:             "credential binding " + credential,
					Reason:               "User provided a single credential binding for the API credential field.",
					RequiresConfirmation: false,
				})
			}
		}
	})
}

func looksCredentialField(field string, op *apitools.OperationSummary) bool {
	lowerField := strings.ToLower(field)
	if strings.Contains(lowerField, "auth") || strings.Contains(lowerField, "token") || strings.Contains(lowerField, "key") {
		return true
	}
	for _, security := range allSecurityRequirements(op) {
		if field == apitools.SecurityCredentialFieldName(security) {
			return true
		}
	}
	return false
}

func matchDocAnswer(answer string, docs []APIDocument) APIDocument {
	answer = strings.TrimSpace(answer)
	if strings.EqualFold(answer, "yes") && len(docs) > 0 {
		return docs[0]
	}
	for i, doc := range docs {
		if answer == doc.RelativePath || answer == fmt.Sprint(i+1) || strings.EqualFold(answer, doc.Title) {
			return doc
		}
	}
	return APIDocument{}
}

func matchOperationAnswer(answer string, docs []APIDocument) (APIDocument, *apitools.OperationSummary) {
	answer = strings.TrimSpace(answer)
	for _, doc := range docs {
		for i := range doc.Operations {
			op := &doc.Operations[i]
			if answer == op.OperationID || answer == fmt.Sprint(i+1) || strings.Contains(strings.ToLower(operationLabel(*op)), strings.ToLower(answer)) {
				return doc, op
			}
		}
	}
	return APIDocument{}, nil
}

func stepFromOperation(doc APIDocument, op *apitools.OperationSummary) *rollout.Step {
	stepType := "http"
	if isBrowserAuthenticationOperationSummary(op) {
		return &rollout.Step{
			Name: camelToSnake(firstNonEmpty("authenticate_"+doc.ID, op.OperationID)),
			Type: "browser_authentication", Do: firstNonEmpty(op.Summary, "Establish the browser session."),
			AuthenticationFlow: op.OperationID,
		}
	}
	if isBrowserRegistrationOperationSummary(op) {
		return browserRegistrationStepFromOperation(doc, op)
	}
	if isBrowserDocument(doc) {
		stepType = "browser"
	}
	return &rollout.Step{
		Name:      camelToSnake(firstNonEmpty(op.OperationID, op.Summary, op.Path)),
		Type:      stepType,
		Do:        firstNonEmpty(op.Summary, operationLabel(*op)),
		Operation: op.OperationID,
	}
}
