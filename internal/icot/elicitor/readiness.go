package elicitor

import "strings"

func CheckReadiness(session Session, docs []APIDocument) []ReadinessIssue {
	rawGoalMissing := session.Intent.Workflow == nil || strings.TrimSpace(firstNonEmpty(session.Intent.Workflow.Description, session.Project.Goal)) == ""
	session.Normalize()
	var issues []ReadinessIssue
	add := func(code, slot, severity, message, suggested string) {
		issues = append(issues, ReadinessIssue{
			Code:            code,
			Slot:            slot,
			Severity:        severity,
			Message:         message,
			SuggestedAnswer: suggested,
		})
	}
	if rawGoalMissing {
		add("missing_goal", "workflow.description", readinessBlocking, "Describe the business goal for the workflow.", "")
	}
	for _, issue := range mappingClassificationIssues(session) {
		add(issue.Code, issue.Slot, issue.Severity, issue.Message, issue.SuggestedAnswer)
	}
	if missingRefs := missingLocalAPIDocumentRefs(session, docs); len(missingRefs) > 0 {
		add("missing_api_doc", "intent.source", readinessBlocking, "Local API document path is not available: "+strings.Join(missingRefs, ", ")+". Generate or provide that artifact before selecting operationIds.", "Generate/provide the missing API artifact, then rerun iCoT.")
	} else if needsAPIDoc(session, docs) {
		add("missing_api_doc", "intent.source", readinessBlocking, missingAPIDocMessage(session, docs), suggestedAPIDocAnswer(session, docs))
	}
	if len(session.Intent.Steps) == 0 {
		add("missing_operation", "intent.steps", readinessBlocking, missingOperationMessage(docs), suggestedOperationAnswer(docs))
	} else {
		for _, step := range session.Intent.Steps {
			if step == nil {
				continue
			}
			slotPrefix := "steps." + firstNonEmpty(step.Name, "step")
			stepType := strings.ToLower(strings.TrimSpace(step.Type))
			if (stepType == "http" || stepType == "openapi" || strings.TrimSpace(step.Operation) != "") && strings.TrimSpace(step.Operation) == "" {
				add("missing_operation", slotPrefix+".operation", readinessBlocking, "Choose the listed OpenAPI operationId for "+firstNonEmpty(step.Name, "this step")+"; leave the capability unresolved if no listed operation matches. "+operationChoiceHintForStep(session, docs, step), suggestedOperationAnswerForStep(session, docs, step))
				continue
			}
			if op, ok := operationForStep(session, docs, step); ok {
				missingFields := missingRequiredFields(step, op)
				if len(missingFields) > 0 {
					add("missing_required_request_values", slotPrefix+".with", readinessBlocking, "Provide sources for the required path/query/header/body fields: "+strings.Join(missingFields, ", ")+". Use inputs.<name>, safe literals, prior-step outputs, or credentials.<binding>.", suggestedFieldAssignments(session, docs, step, op, missingFields))
				}
				if operationNeedsCredential(op) && len(session.Credentials) == 0 {
					add("missing_credential_bindings", "credentials", readinessBlocking, "Name the symbolic credential binding to use for this API; do not paste a secret value.", suggestedCredentialNameForOperation(session, docs, step, op))
				}
				for _, issue := range validateOpenAPIRequestMappings(session, step, op, slotPrefix) {
					add(issue.Code, issue.Slot, issue.Severity, issue.Message, issue.SuggestedAnswer)
				}
			} else if strings.TrimSpace(step.Operation) != "" && (stepType == "http" || stepType == "openapi") {
				add("missing_operation", slotPrefix+".operation", readinessBlocking, "Selected operationId "+step.Operation+" is not available for "+firstNonEmpty(step.Provider, step.Name, "this step")+". "+operationChoiceHintForStep(session, docs, step), suggestedOperationAnswerForStep(session, docs, step))
			}
		}
	}
	missingInputs := missingRuntimeInputs(session)
	if len(missingInputs) > 0 {
		add("missing_runtime_inputs", "intent.inputs", readinessBlocking, "Declare runtime inputs used by the workflow: "+strings.Join(missingInputs, ", ")+".", suggestedRuntimeInputs(missingInputs))
	}
	if len(session.Intent.Outputs) == 0 {
		add("missing_outputs", "intent.outputs", readinessBlocking, "Name the workflow output and its response path or function output source; do not guess provider response fields.", suggestedOutputAnswer(session))
	}
	if strings.TrimSpace(session.Safety) == "" && strings.TrimSpace(session.Project.Safety) == "" {
		add("missing_side_effect_policy", "safety", readinessWarning, "Confirm whether this workflow is read-only, sandbox-only, or after-approval before any side-effectful execution.", suggestedPolicyAnswer(session))
	}
	if referencesOptionalControls(session) {
		if session.Intent.Workflow == nil || (session.Intent.Workflow.Timeout == nil && session.Intent.Workflow.Idempotency == nil) {
			add("optional_timeout_idempotency_controls", "workflow.controls", readinessWarning, "Timeout or idempotency controls were mentioned but not configured.", "")
		}
	}
	if firstBlockingIssue(issues).Code == "" {
		if _, err := RenderArtifacts(session); err != nil {
			add("intent_render_invalid", "intent", readinessBlocking, "The intent still needs one concrete workflow detail before it can be saved: "+err.Error()+".", "")
		}
	}
	return sortReadinessIssues(issues)
}

func PlanNextQuestion(session Session, docs []APIDocument, issues []ReadinessIssue) QuestionPlan {
	blocking := firstBlockingIssue(issues)
	if blocking.Code == "" {
		return QuestionPlan{
			Prompt:          "Confirm any remaining warnings or assumptions",
			SuggestedAnswer: "save",
			Slots:           []string{"warnings"},
		}
	}
	plan := QuestionPlan{
		SuggestedAnswer: blocking.SuggestedAnswer,
		Slots:           []string{blocking.Slot},
	}
	switch blocking.Code {
	case "missing_goal":
		plan.Prompt = "What should this workflow accomplish for the business?"
	case "missing_api_doc":
		plan.Prompt = missingAPIDocPrompt(session, docs)
	case "missing_operation":
		plan.Prompt = missingOperationPrompt(session, docs, blocking.Slot)
	case "missing_required_request_values":
		stepName := stepNameForQuestionSlot(blocking.Slot)
		if stepName != "" {
			plan.Prompt = "What values should the required request fields use? Step: " + stepName + ". Map each field to inputs.<name>, a safe literal, a prior-step output, or credentials.<binding>."
		} else {
			plan.Prompt = "What values should the required request fields use? Map each field to inputs.<name>, a safe literal, a prior-step output, or credentials.<binding>."
		}
		plan.Grouped = true
	case "missing_credential_bindings":
		plan.Prompt = "What credential binding name should the workflow reference? Use a symbolic name only."
		plan.Grouped = true
	case "missing_runtime_inputs":
		plan.Prompt = "What runtime inputs should the operator provide?"
		plan.Grouped = true
	case "missing_outputs":
		plan.Prompt = "What should the workflow return as its output? Use a known response path or function output."
	case "missing_side_effect_policy":
		plan.Prompt = "What side-effect and approval boundary should apply? Choose read-only, sandbox-only, or after-approval."
	case "optional_timeout_idempotency_controls":
		plan.Prompt = "Should this workflow use timeout or idempotency controls?"
	case "conflicting_mapping":
		plan.Prompt = "Which mapping value should " + blocking.Slot + " use?"
		plan.Grouped = strings.Contains(blocking.Slot, ".with.")
	case "low_confidence_mapping":
		plan.Prompt = "Confirm the mapping value for " + blocking.Slot + "."
		plan.Grouped = strings.Contains(blocking.Slot, ".with.")
	default:
		plan.Prompt = blocking.Message
	}
	if plan.SuggestedAnswer == "" {
		plan.SuggestedAnswer = suggestedAnswerForCode(blocking.Code, session, docs)
	}
	return plan
}

func stepNameForQuestionSlot(slot string) string {
	if name, ok := stepNameFromWithSlot(slot); ok {
		return name
	}
	return ""
}

func progressiveReady(session Session, issues []ReadinessIssue) bool {
	if _, err := RenderArtifacts(session); err != nil {
		return false
	}
	return firstBlockingIssue(issues).Code == ""
}
