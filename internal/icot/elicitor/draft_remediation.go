package elicitor

import (
	"fmt"
	"os"
	"strings"

	"github.com/OpenUdon/apitools"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
	"gopkg.in/yaml.v3"
)

func applyDraftReviewRemediations(session *Session, issues []DraftReviewIssue, docs ...[]APIDocument) (bool, []string) {
	if session == nil {
		return false, nil
	}
	var apiDocs []APIDocument
	if len(docs) > 0 {
		apiDocs = docs[0]
	}
	sanitized := sanitizeDraftReviewResponse(DraftReviewResponse{Issues: issues}).Issues
	changed, rejected := applyDraftReviewRepairs(session, repairableDraftReviewIssues(sanitized))
	for _, issue := range sanitized {
		if issue.RemediationAction == remediationProposeAPIPrework {
			rejected = append(rejected, firstNonEmpty(issue.Slot, issue.Code)+" (api prework remediation is not implemented for review repair yet)")
			continue
		}
		if issue.RemediationAction != remediationProposeFnctStep {
			continue
		}
		applied, reason := applyDraftReviewFnctRemediation(session, issue, apiDocs)
		if applied {
			changed = true
			continue
		}
		if reason != "" {
			rejected = append(rejected, firstNonEmpty(issue.Slot, issue.Code)+" ("+reason+")")
		}
	}
	return changed, dedupeStrings(rejected)
}

func repairableDraftReviewIssues(issues []DraftReviewIssue) []DraftReviewIssue {
	var out []DraftReviewIssue
	for _, issue := range issues {
		if issue.RemediationAction == remediationApplyNarrowRepair {
			out = append(out, issue)
		}
	}
	return out
}

func applyDraftReviewFnctRemediation(session *Session, issue DraftReviewIssue, docs []APIDocument) (bool, string) {
	if session == nil {
		return false, "missing session"
	}
	if !goalAllowsLocalFnctRemediation(*session) {
		return false, "goal does not clearly ask for local produced content"
	}
	sink, field := sinkStepAndFieldForIssue(session, issue)
	producers := candidateProducerSteps(session, sink)
	if len(producers) != 1 {
		if len(producers) == 0 {
			return false, "no single producer step"
		}
		return false, "ambiguous producer step"
	}
	producer := producers[0]
	name := uniqueStepName(session.Intent.Steps, fnctRemediationStepBase(*session, issue))
	do := fnctRemediationDoString(name, producer.Name)
	fnct := &rollout.Step{
		Name:      name,
		Type:      "fnct",
		Do:        do,
		DependsOn: []string{producer.Name},
	}
	fnct.With, fnct.Binds = fnctRemediationInputBindings(*session, docs, producer)
	insertStepAfter(session, producer, fnct)
	if sink != nil {
		sink.DependsOn = appendUniqueString(sink.DependsOn, fnct.Name)
		if field != "" {
			if sink.With == nil {
				sink.With = map[string]string{}
			}
			sink.With[field] = fnct.Name + ".received_body"
		}
	}
	ensureFnctRemediationOutput(session, issue, fnct, producer)
	addDecisionEvidence(session, DecisionEvidence{
		Stage:                decisionStageDraftReview,
		Slot:                 "steps." + fnct.Name,
		Value:                fnct.Do,
		Source:               mappingSourceDeterministic,
		Confidence:           mappingConfidenceReview,
		Reason:               "Added a local fnct step because the pre-final flow review found a missing transform/report step and exactly one prior producer step was available.",
		Evidence:             issue.Message,
		RequiresConfirmation: true,
	})
	addMappingClassification(session, MappingClassification{
		Slot:                 "steps." + fnct.Name,
		Value:                fnct.Name + " consumes " + producer.Name + ".received_body",
		Source:               mappingSourceDeterministic,
		Confidence:           mappingConfidenceReview,
		Evidence:             issue.Message,
		Reason:               "Pre-final flow remediation added a reviewable local transform step without changing API operations or credentials.",
		RequiresConfirmation: true,
	})
	return true, ""
}

func fnctRemediationInputBindings(session Session, docs []APIDocument, producer *rollout.Step) (map[string]string, []*rollout.StepBind) {
	fields := responseFieldPathsForStep(session, docs, producer)
	if len(fields) == 0 {
		return map[string]string{"input": producer.Name + ".received_body"}, []*rollout.StepBind{{
			From: producer.Name,
			Fields: map[string]string{
				"input": "received_body",
			},
		}}
	}
	bindFields := map[string]string{}
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		bindFields[slugIdent(field)] = "received_body." + field
	}
	if len(bindFields) == 0 {
		return map[string]string{"input": producer.Name + ".received_body"}, []*rollout.StepBind{{
			From: producer.Name,
			Fields: map[string]string{
				"input": "received_body",
			},
		}}
	}
	return nil, []*rollout.StepBind{{From: producer.Name, Fields: bindFields}}
}

func goalAllowsLocalFnctRemediation(session Session) bool {
	text := strings.ToLower(strings.Join([]string{
		session.Project.Goal,
		session.Project.Outputs,
		session.Project.DataFlow,
		session.Project.FunctionContracts,
		draftSessionDescription(session),
	}, " "))
	for _, token := range []string{"produced content", "generated content", "transform", "summary", "summarize", "summarise", "normalize", "normalise", "report", "receipt", "render", "enrich", "compose"} {
		if strings.Contains(text, token) {
			return true
		}
	}
	return false
}

func sinkStepAndFieldForIssue(session *Session, issue DraftReviewIssue) (*rollout.Step, string) {
	stepName, field, ok := parseStepWithSlot(issue.Slot)
	if !ok {
		return nil, ""
	}
	return stepByName(session.Intent.Steps, stepName), field
}

func candidateProducerSteps(session *Session, sink *rollout.Step) []*rollout.Step {
	if session == nil {
		return nil
	}
	referenced := fnctRemediationReferencedSteps(session, sink)
	var out []*rollout.Step
	for _, step := range session.Intent.Steps {
		if step == nil || step == sink {
			break
		}
		if canProduceFnctRemediationInput(step, referenced) {
			out = append(out, step)
		}
	}
	if sink == nil {
		out = nil
		for _, step := range session.Intent.Steps {
			if canProduceFnctRemediationInput(step, referenced) {
				out = append(out, step)
			}
		}
	}
	return out
}

func fnctRemediationReferencedSteps(session *Session, sink *rollout.Step) map[string]bool {
	referenced := map[string]bool{}
	if session == nil {
		return referenced
	}
	for _, output := range session.Intent.Outputs {
		if output != nil {
			addSourceStepReference(referenced, output.From)
		}
	}
	if sink != nil {
		for _, source := range sink.With {
			addSourceStepReference(referenced, source)
		}
		for _, bind := range sink.Binds {
			if bind != nil {
				referenced[strings.TrimSpace(bind.From)] = true
			}
		}
	}
	return referenced
}

func addSourceStepReference(referenced map[string]bool, source string) {
	name := sourceStepName(source)
	if name != "" {
		referenced[name] = true
	}
}

func sourceStepName(source string) string {
	source = strings.TrimSpace(source)
	if source == "" || strings.HasPrefix(source, "inputs.") || strings.HasPrefix(source, "credentials.") {
		return ""
	}
	idx := strings.IndexAny(source, ".[")
	if idx <= 0 {
		return ""
	}
	return source[:idx]
}

func canProduceFnctRemediationInput(step *rollout.Step, referenced map[string]bool) bool {
	if step == nil {
		return false
	}
	stepType := strings.ToLower(strings.TrimSpace(step.Type))
	if stepType == "fnct" {
		return true
	}
	if (stepType == "http" || stepType == "openapi") && strings.TrimSpace(step.Operation) != "" {
		return true
	}
	if referenced[strings.TrimSpace(step.Name)] {
		return true
	}
	return false
}

func fnctRemediationStepBase(session Session, issue DraftReviewIssue) string {
	text := strings.ToLower(strings.Join([]string{session.Project.Goal, draftSessionDescription(session), issue.Message, issue.Evidence}, " "))
	switch {
	case strings.Contains(text, "audit") && strings.Contains(text, "receipt"):
		return "render_audit_receipt"
	case strings.Contains(text, "incident") && (strings.Contains(text, "summary") || strings.Contains(text, "summar")):
		return "summarize_incident"
	case strings.Contains(text, "normalize") || strings.Contains(text, "normalise"):
		return "normalize_record"
	case strings.Contains(text, "report"):
		return "render_report"
	case strings.Contains(text, "receipt"):
		return "render_receipt"
	case strings.Contains(text, "summary") || strings.Contains(text, "summar"):
		return "summarize_result"
	case strings.Contains(text, "enrich"):
		return "enrich_record"
	default:
		return "render_content"
	}
}

func fnctRemediationDoString(name, producer string) string {
	label := strings.ReplaceAll(name, "_", " ")
	return fmt.Sprintf("Locally %s from %s output for review before downstream delivery or workflow output.", label, producer)
}

func ensureFnctRemediationOutput(session *Session, issue DraftReviewIssue, fnct, producer *rollout.Step) {
	if session == nil || fnct == nil {
		return
	}
	source := fnct.Name + ".received_body"
	if name, _, ok := parseOutputSlot(session, issue.Slot); ok {
		for _, output := range session.Intent.Outputs {
			if output != nil && output.Name == name {
				if defaultRawResultOutput(output, producer) {
					output.Name = outputNameFromFnctStep(fnct.Name)
				}
				output.From = source
				return
			}
		}
		session.Intent.Outputs = append(session.Intent.Outputs, &rollout.Output{Name: name, From: source})
		return
	}
	if len(session.Intent.Outputs) == 0 {
		session.Intent.Outputs = append(session.Intent.Outputs, &rollout.Output{Name: outputNameFromFnctStep(fnct.Name), From: source})
	}
}

func defaultRawResultOutput(output *rollout.Output, producer *rollout.Step) bool {
	if output == nil || strings.TrimSpace(output.Name) != "result" || producer == nil {
		return false
	}
	return strings.TrimSpace(output.From) == producer.Name+".received_body"
}

func outputNameFromFnctStep(stepName string) string {
	name := slugIdent(stepName)
	for _, prefix := range []string{"render_", "compose_", "generate_", "create_", "prepare_"} {
		if strings.HasPrefix(name, prefix) {
			name = strings.TrimPrefix(name, prefix)
			break
		}
	}
	if name == "" {
		return "result"
	}
	return name
}

func responseFieldPathsForStep(session Session, docs []APIDocument, step *rollout.Step) []string {
	if step == nil || strings.TrimSpace(step.Operation) == "" {
		return nil
	}
	op, ok := operationForStep(session, docs, step)
	if !ok {
		return nil
	}
	doc, ok := documentForStep(session, docs, step, op)
	if !ok || strings.TrimSpace(doc.Path) == "" {
		return nil
	}
	return openAPIResponseFieldPaths(doc.Path, *op)
}

func openAPIResponseFieldPaths(path string, op apitools.OperationSummary) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var root map[string]any
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil
	}
	paths := anyMap(root["paths"])
	pathItem := anyMap(paths[op.Path])
	operation := anyMap(pathItem[strings.ToLower(op.Method)])
	if len(operation) == 0 {
		return nil
	}
	responses := anyMap(operation["responses"])
	for _, code := range []string{"200", "201", "202", "default"} {
		response := anyMap(responses[code])
		if len(response) == 0 {
			continue
		}
		if fields := responseFieldsFromResponse(response); len(fields) > 0 {
			return fields
		}
	}
	return nil
}

func responseFieldsFromResponse(response map[string]any) []string {
	content := anyMap(response["content"])
	for _, contentType := range []string{"application/json", "application/*+json"} {
		mediaType := anyMap(content[contentType])
		if len(mediaType) == 0 {
			continue
		}
		if fields := responseFieldsFromSchema(anyMap(mediaType["schema"])); len(fields) > 0 {
			return fields
		}
	}
	for _, raw := range content {
		mediaType := anyMap(raw)
		if fields := responseFieldsFromSchema(anyMap(mediaType["schema"])); len(fields) > 0 {
			return fields
		}
	}
	return nil
}

func responseFieldsFromSchema(schema map[string]any) []string {
	properties := anyMap(schema["properties"])
	if len(properties) == 0 {
		return nil
	}
	var fields []string
	for name, raw := range properties {
		if len(fields) >= 8 {
			break
		}
		if strings.TrimSpace(name) == "" {
			continue
		}
		fieldSchema := anyMap(raw)
		if typ := strings.TrimSpace(fmt.Sprint(fieldSchema["type"])); typ != "" && !safeScalarType(typ) {
			continue
		}
		fields = append(fields, name)
	}
	return dedupeStrings(fields)
}

func anyMap(value any) map[string]any {
	switch typed := value.(type) {
	case map[string]any:
		return typed
	case map[any]any:
		out := map[string]any{}
		for key, val := range typed {
			out[fmt.Sprint(key)] = val
		}
		return out
	default:
		return nil
	}
}

func insertStepAfter(session *Session, after, inserted *rollout.Step) {
	if session == nil || after == nil || inserted == nil {
		return
	}
	for i, step := range session.Intent.Steps {
		if step == after {
			next := append([]*rollout.Step{}, session.Intent.Steps[:i+1]...)
			next = append(next, inserted)
			next = append(next, session.Intent.Steps[i+1:]...)
			session.Intent.Steps = next
			return
		}
	}
	session.Intent.Steps = append(session.Intent.Steps, inserted)
}

func draftReviewForcedQuestion(issue DraftReviewIssue) string {
	if issue.RemediationAction == remediationAskUser && strings.TrimSpace(issue.ClarifyingQuestion) != "" {
		return strings.TrimSpace(issue.ClarifyingQuestion)
	}
	if issue.RemediationAction == remediationAskUser {
		return "What exact workflow output or produced content should replace the ambiguous draft output?"
	}
	return ""
}

func applyForcedDraftReviewAnswer(session *Session, issue DraftReviewIssue, answer string) (bool, string) {
	answer = strings.TrimSpace(answer)
	if session == nil || answer == "" {
		return false, ""
	}
	name, field, ok := parseOutputSlot(session, issue.Slot)
	if !ok || (field != "" && field != "from") {
		return false, forcedDraftReviewClarification(answer)
	}
	from := forcedDraftReviewOutputSource(name, answer)
	if !safeForcedOutputSource(session, from) {
		return false, forcedDraftReviewClarification(answer)
	}
	for _, output := range session.Intent.Outputs {
		if output == nil || output.Name != name {
			continue
		}
		if strings.TrimSpace(output.From) == from {
			return false, ""
		}
		output.From = from
		addMappingClassification(session, MappingClassification{
			Slot:                 "intent.outputs." + output.Name,
			Value:                output.Name + "=" + output.From,
			Source:               mappingSourceUser,
			Confidence:           mappingConfidenceReview,
			Evidence:             answer,
			Reason:               "Operator answered a forced pre-final ambiguity question with a known safe output source.",
			RequiresConfirmation: false,
		})
		return true, ""
	}
	session.Intent.Outputs = append(session.Intent.Outputs, &rollout.Output{Name: name, From: from})
	addMappingClassification(session, MappingClassification{
		Slot:                 "intent.outputs." + name,
		Value:                name + "=" + from,
		Source:               mappingSourceUser,
		Confidence:           mappingConfidenceReview,
		Evidence:             answer,
		Reason:               "Operator answered a forced pre-final ambiguity question with a known safe output source.",
		RequiresConfirmation: false,
	})
	return true, ""
}

func forcedDraftReviewOutputSource(outputName, answer string) string {
	answer = strings.TrimSpace(answer)
	assignments := parseAssignments(answer)
	if value, ok := assignments[outputName]; ok {
		return strings.TrimSpace(value)
	}
	if value, ok := assignments["from"]; ok {
		return strings.TrimSpace(value)
	}
	if len(assignments) == 1 {
		for _, value := range assignments {
			return strings.TrimSpace(value)
		}
	}
	return answer
}

func safeForcedOutputSource(session *Session, source string) bool {
	source = strings.TrimSpace(source)
	if source == "" || strings.ContainsAny(source, " \t\r\n") {
		return false
	}
	lower := strings.ToLower(source)
	for _, token := range []string{"operation", "source", "credential", "side-effect", "side_effect", "side effect", "openapi", "provider"} {
		if strings.Contains(lower, token) {
			return false
		}
	}
	stepName := sourceStepName(source)
	return stepName != "" && stepByName(session.Intent.Steps, stepName) != nil
}

func forcedDraftReviewClarification(answer string) string {
	answer = strings.TrimSpace(answer)
	if answer == "" {
		return ""
	}
	return "Operator clarification: " + truncateForPrompt(answer, 180)
}

func forcedDraftReviewAnswerReason(applied bool) string {
	if applied {
		return "Operator answered a forced pre-final flow ambiguity question and the answer was applied to a safe output source."
	}
	return "Operator answered a forced pre-final flow ambiguity question; the answer was kept as review evidence because it was not a safe output-source mapping."
}
