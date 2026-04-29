package elicitor

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/genelet/ramen/internal/projectwizard"
	"github.com/genelet/udon/pkg/rollout"
)

type ReadinessIssue struct {
	Code            string `json:"code"`
	Slot            string `json:"slot,omitempty"`
	Severity        string `json:"severity"`
	Message         string `json:"message"`
	SuggestedAnswer string `json:"suggested_answer,omitempty"`
}

type QuestionPlan struct {
	Prompt          string   `json:"prompt"`
	SuggestedAnswer string   `json:"suggested_answer,omitempty"`
	Slots           []string `json:"slots,omitempty"`
	Grouped         bool     `json:"grouped,omitempty"`
}

const (
	readinessBlocking = "blocking"
	readinessWarning  = "warning"
)

func runProgressive(ctx context.Context, in io.Reader, out io.Writer, seed Session, opts Options) (Artifacts, error) {
	reader, ok := in.(*bufio.Reader)
	if !ok {
		reader = bufio.NewReader(in)
	}
	extractor := opts.Extractor
	if extractor == nil {
		extractor = NewNoopExtractor()
	}
	session := seed
	session.Normalize()
	p := &prompter{reader: reader, out: out}
	var events []TranscriptEvent
	record := func(kind string, data any) {
		events = append(events, TranscriptEvent{Kind: kind, Data: data})
	}

	projectText := projectwizard.Render(session.Project)
	docs, err := DiscoverLocalAPIs(opts.ExampleDir, projectText)
	if err != nil {
		return Artifacts{}, err
	}
	openingBrief := ""
	if session.Intent.Workflow != nil {
		openingBrief = strings.TrimSpace(session.Intent.Workflow.Description)
	}
	if openingBrief == "" {
		fmt.Fprintln(out, "Tell me what you want this API/workflow to accomplish. Include inputs, API actions, outputs, and safety constraints if you know them. Do not paste secrets.")
		answer, err := p.ask("Workflow goal")
		if err != nil {
			return Artifacts{}, err
		}
		openingBrief = strings.TrimSpace(answer)
		applyProgressiveAnswer(&session, QuestionPlan{Slots: []string{"workflow.goal"}}, openingBrief, docs)
		session.Normalize()
		if err := autosave(opts.DraftPath, session); err != nil {
			return Artifacts{}, err
		}
		record("progressive_question", QuestionPlan{Prompt: "Workflow goal", Slots: []string{"workflow.goal"}})
		record("progressive_answer", ReplayTurn{Label: "Workflow goal", Answer: answer})
	}
	if !opts.NoLLM && len(docs) > 1 && openingBrief != "" {
		if ranked, err := extractor.Disambiguate(ctx, openingBrief, docs); err == nil {
			docs = rankDocuments(docs, ranked)
		} else {
			fmt.Fprintf(out, "icot: OpenAPI ranking skipped: %v\n", err)
		}
	}

	var issues []ReadinessIssue
	for attempt := 0; attempt < 20; attempt++ {
		if deterministicPrefill(&session, docs) {
			session.Normalize()
		}
		request := DraftRequest{
			Opening:           openingBrief,
			Session:           session,
			Docs:              docs,
			TranscriptTurns:   append([]ReplayTurn(nil), p.turns...),
			ReadinessFeedback: append([]ReadinessIssue(nil), issues...),
		}
		record("model_draft_call", map[string]any{
			"opening":          request.Opening,
			"turn_count":       len(request.TranscriptTurns),
			"readiness_issues": request.ReadinessFeedback,
		})
		draft, draftErr := extractor.Draft(ctx, request)
		if draftErr == nil && LooksLikeSession(draft) {
			session = mergeProgressiveSessions(session, draft, docs)
			defaultSingleOpenAPIDoc(&session, docs)
			session.Normalize()
			if deterministicPrefill(&session, docs) {
				session.Normalize()
			}
			record("model_draft_result", map[string]any{
				"steps":       len(session.Intent.Steps),
				"inputs":      len(session.Intent.Inputs),
				"outputs":     len(session.Intent.Outputs),
				"assumptions": session.Assumptions,
			})
			if err := autosave(opts.DraftPath, session); err != nil {
				return Artifacts{}, err
			}
			printSummary(out, session)
		} else if draftErr != nil {
			record("model_draft_error", draftErr.Error())
			fmt.Fprintf(out, "icot: AI draft skipped: %v\n", draftErr)
		}

		issues = CheckReadiness(session, docs)
		record("readiness_decision", issues)
		if progressiveReady(session, issues) {
			record("next_question_decision", QuestionPlan{
				Prompt:          "Confirm first valid intent",
				SuggestedAnswer: "save",
				Slots:           []string{"confirmation"},
			})
			artifacts, err := finalProgressiveConfirmationLoop(out, p, &session, docs, opts.DraftPath, &events)
			if err == nil {
				record("final_generated_artifacts", map[string]any{
					"intent_hcl_bytes": len(artifacts.IntentHCL),
					"project_md_bytes": len(artifacts.ProjectMD),
					"assumptions":      artifacts.Session.Assumptions,
				})
				if saveErr := SaveTranscriptWithEvents(opts.TranscriptPath, p.turns, events, artifacts.Session); saveErr != nil {
					return artifacts, saveErr
				}
			}
			return artifacts, err
		}

		plan := PlanNextQuestion(session, docs, issues)
		record("next_question_decision", plan)
		answer, err := p.askDefault(plan.Prompt, plan.SuggestedAnswer)
		if err != nil {
			return Artifacts{}, err
		}
		trimmed := strings.TrimSpace(answer)
		if strings.EqualFold(trimmed, "cancel") {
			return Artifacts{}, ErrCanceled
		}
		applyProgressiveAnswer(&session, plan, answer, docs)
		defaultSingleOpenAPIDoc(&session, docs)
		session.Normalize()
		if deterministicPrefill(&session, docs) {
			session.Normalize()
		}
		record("progressive_question", plan)
		record("progressive_answer", ReplayTurn{Label: plan.Prompt, Answer: answer})
		if err := autosave(opts.DraftPath, session); err != nil {
			return Artifacts{}, err
		}
	}
	return Artifacts{}, fmt.Errorf("progressive iCoT could not reach a valid intent after 20 draft attempts")
}

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
	if needsAPIDoc(session, docs) {
		add("missing_api_doc", "intent.openapi", readinessBlocking, "Identify the OpenAPI document for API-backed steps.", suggestedDocAnswer(docs))
	}
	if len(session.Intent.Steps) == 0 {
		add("missing_operation", "intent.steps", readinessBlocking, "Choose the API operation or workflow action to run.", suggestedOperationAnswer(docs))
	} else {
		for _, step := range session.Intent.Steps {
			if step == nil {
				continue
			}
			slotPrefix := "steps." + firstNonEmpty(step.Name, "step")
			stepType := strings.ToLower(strings.TrimSpace(step.Type))
			if (stepType == "http" || stepType == "openapi" || strings.TrimSpace(step.Operation) != "") && strings.TrimSpace(step.Operation) == "" {
				add("missing_operation", slotPrefix+".operation", readinessBlocking, "Choose the API operation for "+firstNonEmpty(step.Name, "this step")+".", suggestedOperationAnswer(docs))
				continue
			}
			if op, ok := operationForStep(session, docs, step); ok {
				missingFields := missingRequiredFields(step, op)
				if len(missingFields) > 0 {
					add("missing_required_request_values", slotPrefix+".with", readinessBlocking, "Provide values for the required API request fields: "+strings.Join(missingFields, ", ")+".", suggestedFieldAssignments(session, docs, step, op, missingFields))
				}
				if operationNeedsCredential(op) && len(session.Credentials) == 0 {
					add("missing_credential_bindings", "credentials", readinessBlocking, "Name the credential binding to use for this API.", suggestedCredentialNameForOperation(session, docs, step, op))
				}
				for _, issue := range validateOpenAPIRequestMappings(session, step, op, slotPrefix) {
					add(issue.Code, issue.Slot, issue.Severity, issue.Message, issue.SuggestedAnswer)
				}
			}
		}
	}
	missingInputs := missingRuntimeInputs(session)
	if len(missingInputs) > 0 {
		add("missing_runtime_inputs", "intent.inputs", readinessBlocking, "Declare runtime inputs used by the workflow: "+strings.Join(missingInputs, ", ")+".", suggestedRuntimeInputs(missingInputs))
	}
	if len(session.Intent.Outputs) == 0 {
		add("missing_outputs", "intent.outputs", readinessBlocking, "Name the workflow output and where it comes from.", suggestedOutputAnswer(session))
	}
	if strings.TrimSpace(session.Safety) == "" && strings.TrimSpace(session.Project.Safety) == "" {
		add("missing_side_effect_policy", "safety", readinessWarning, "Confirm the side-effect and approval policy.", suggestedPolicyAnswer(session))
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
		plan.Prompt = "Which API document should this workflow use?"
	case "missing_operation":
		plan.Prompt = "Which API action or workflow step should run first?"
	case "missing_required_request_values":
		plan.Prompt = "What values should the required request fields use?"
		plan.Grouped = true
	case "missing_credential_bindings":
		plan.Prompt = "What credential binding name should the workflow reference?"
		plan.Grouped = true
	case "missing_runtime_inputs":
		plan.Prompt = "What runtime inputs should the operator provide?"
		plan.Grouped = true
	case "missing_outputs":
		plan.Prompt = "What should the workflow return as its output?"
	case "missing_side_effect_policy":
		plan.Prompt = "What side-effect and approval boundary should apply?"
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

func progressiveReady(session Session, issues []ReadinessIssue) bool {
	if _, err := RenderArtifacts(session); err != nil {
		return false
	}
	return firstBlockingIssue(issues).Code == ""
}

func finalProgressiveConfirmationLoop(out io.Writer, p *prompter, session *Session, docs []APIDocument, draftPath string, events *[]TranscriptEvent) (Artifacts, error) {
	for {
		artifacts, err := RenderArtifacts(*session)
		if err != nil {
			fmt.Fprintf(out, "Intent is incomplete: %v\n", err)
			slot, slotErr := p.askDefault("Edit slot", "steps")
			if slotErr != nil {
				return Artifacts{}, slotErr
			}
			if err := editSlot(p, session, strings.TrimSpace(slot), docs); err != nil {
				return Artifacts{}, err
			}
			if err := autosave(draftPath, *session); err != nil {
				return Artifacts{}, err
			}
			continue
		}
		issues := CheckReadiness(artifacts.Session, docs)
		fmt.Fprintln(out, "\n----- current draft -----")
		printSummary(out, artifacts.Session)
		printReadinessWarnings(out, issues)
		printAssumptions(out, artifacts.Session.Assumptions)
		if len(artifacts.Session.Annotations) > 0 {
			fmt.Fprintln(out, "LLM-prefilled values are marked in the session annotations and require this final confirmation.")
		}
		answer, err := p.askDefault("Type save, edit <slot>, explain <assumption-id>, or cancel", "save")
		if err != nil {
			return Artifacts{}, err
		}
		answer = strings.TrimSpace(strings.ToLower(answer))
		switch {
		case answer == "" || answer == "save":
			return artifacts, nil
		case answer == "cancel":
			return Artifacts{}, ErrCanceled
		case strings.HasPrefix(answer, "edit"):
			slot := strings.TrimSpace(strings.TrimPrefix(answer, "edit"))
			if slot == "" {
				slot, err = p.askDefault("Edit slot", "steps")
				if err != nil {
					return Artifacts{}, err
				}
			}
			if err := editSlot(p, session, slot, docs); err != nil {
				return Artifacts{}, err
			}
			session.Normalize()
			if events != nil {
				*events = append(*events, TranscriptEvent{Kind: "confirmation_edit", Data: slot})
			}
			if err := autosave(draftPath, *session); err != nil {
				return Artifacts{}, err
			}
		case strings.HasPrefix(answer, "explain"):
			id := strings.TrimSpace(strings.TrimPrefix(answer, "explain"))
			printAssumptionExplanation(out, *session, id)
		default:
			fmt.Fprintln(out, "Please type save, edit <slot>, explain <assumption-id>, or cancel.")
		}
	}
}

func printReadinessWarnings(out io.Writer, issues []ReadinessIssue) {
	var warnings []ReadinessIssue
	for _, issue := range issues {
		if issue.Severity == readinessWarning {
			warnings = append(warnings, issue)
		}
	}
	if len(warnings) == 0 {
		return
	}
	fmt.Fprintln(out, "Remaining warnings:")
	for _, warning := range warnings {
		fmt.Fprintf(out, "- %s: %s\n", warning.Code, warning.Message)
	}
	fmt.Fprintln(out)
}

func mergeProgressiveSessions(base, overlay Session, docs []APIDocument) Session {
	overlay = sanitizeDraft(DraftRequest{Session: base, Docs: docs}, overlay)
	before := base
	recordLLMOverlayClassifications(&base, before, overlay)
	if base.Intent.Workflow == nil && overlay.Intent.Workflow != nil {
		base.Intent.Workflow = overlay.Intent.Workflow
	} else if base.Intent.Workflow != nil && overlay.Intent.Workflow != nil {
		base.Intent.Workflow.Name = firstNonEmpty(base.Intent.Workflow.Name, overlay.Intent.Workflow.Name)
		base.Intent.Workflow.Description = firstNonEmpty(base.Intent.Workflow.Description, overlay.Intent.Workflow.Description)
		if base.Intent.Workflow.Timeout == nil {
			base.Intent.Workflow.Timeout = overlay.Intent.Workflow.Timeout
		}
		if base.Intent.Workflow.Idempotency == nil {
			base.Intent.Workflow.Idempotency = overlay.Intent.Workflow.Idempotency
		}
	}
	base.Intent.OpenAPI = firstNonEmpty(base.Intent.OpenAPI, overlay.Intent.OpenAPI)
	base.Intent.ServerURL = firstNonEmpty(base.Intent.ServerURL, overlay.Intent.ServerURL)
	base.Intent.Inputs = mergeInputsByName(base.Intent.Inputs, overlay.Intent.Inputs)
	base.Intent.Steps = mergeStepsByName(base.Intent.Steps, overlay.Intent.Steps)
	base.Intent.Outputs = mergeOutputsByName(base.Intent.Outputs, overlay.Intent.Outputs)
	if len(base.Intent.Security) == 0 {
		base.Intent.Security = overlay.Intent.Security
	}
	base.Project = mergeAnswers(base.Project, overlay.Project)
	if overlay.CredentialsSet {
		base.Credentials = overlay.Credentials
		base.CredentialsSet = true
	} else {
		base.Credentials = dedupeStrings(append(base.Credentials, overlay.Credentials...))
	}
	if overlay.SafetySet {
		base.Safety = overlay.Safety
		base.SafetySet = true
	} else {
		base.Safety = firstNonEmpty(base.Safety, overlay.Safety)
	}
	if overlay.FallbackSet {
		base.Fallback = overlay.Fallback
		base.FallbackSet = true
	} else {
		base.Fallback = firstNonEmpty(base.Fallback, overlay.Fallback)
	}
	base.SideEffectScope = firstNonEmpty(base.SideEffectScope, overlay.SideEffectScope)
	base.Annotations = append(base.Annotations, overlay.Annotations...)
	base.Assumptions = mergeAssumptions(base.Assumptions, overlay.Assumptions)
	base.Normalize()
	return base
}

func defaultSingleOpenAPIDoc(session *Session, docs []APIDocument) {
	if session == nil || strings.TrimSpace(session.Intent.OpenAPI) != "" || len(docs) != 1 || !session.Intent.RequiresOpenAPI() {
		return
	}
	session.Intent.OpenAPI = docs[0].RelativePath
	addMappingClassification(session, MappingClassification{
		Slot:                 "intent.openapi",
		Value:                docs[0].RelativePath,
		Source:               mappingSourceFallbackDefault,
		Confidence:           mappingConfidenceReview,
		Evidence:             docs[0].RelativePath,
		Reason:               "Only one local OpenAPI document is available for API-backed steps.",
		RequiresConfirmation: true,
	})
}

func mergeInputsByName(base, overlay []*rollout.Input) []*rollout.Input {
	out := append([]*rollout.Input(nil), base...)
	index := map[string]int{}
	for i, input := range out {
		if input != nil {
			index[input.Name] = i
		}
	}
	for _, input := range overlay {
		if input == nil || input.Name == "" {
			continue
		}
		if existing, ok := index[input.Name]; ok {
			if out[existing].Type == "" {
				out[existing].Type = input.Type
			}
			if out[existing].Description == "" {
				out[existing].Description = input.Description
			}
			out[existing].Required = out[existing].Required || input.Required
			continue
		}
		index[input.Name] = len(out)
		out = append(out, input)
	}
	return out
}

func mergeOutputsByName(base, overlay []*rollout.Output) []*rollout.Output {
	if len(base) == 0 {
		return overlay
	}
	out := append([]*rollout.Output(nil), base...)
	index := map[string]int{}
	for i, output := range out {
		if output != nil {
			index[output.Name] = i
		}
	}
	for _, output := range overlay {
		if output == nil || output.Name == "" {
			continue
		}
		if existing, ok := index[output.Name]; ok {
			if out[existing].From == "" {
				out[existing].From = output.From
			}
			if out[existing].Description == "" {
				out[existing].Description = output.Description
			}
			continue
		}
		out = append(out, output)
	}
	return out
}

func mergeStepsByName(base, overlay []*rollout.Step) []*rollout.Step {
	if len(base) == 0 {
		return overlay
	}
	out := append([]*rollout.Step(nil), base...)
	index := map[string]int{}
	for i, step := range out {
		if step != nil {
			index[step.Name] = i
		}
	}
	for _, step := range overlay {
		if step == nil || step.Name == "" {
			continue
		}
		if existing, ok := index[step.Name]; ok {
			mergeStep(out[existing], step)
			continue
		}
		out = append(out, step)
	}
	return out
}

func mergeStep(base, overlay *rollout.Step) {
	base.Type = firstNonEmpty(base.Type, overlay.Type)
	base.Do = firstNonEmpty(base.Do, overlay.Do)
	base.Using = firstNonEmpty(base.Using, overlay.Using)
	base.Set = firstNonEmpty(base.Set, overlay.Set)
	base.When = firstNonEmpty(base.When, overlay.When)
	base.ForEach = firstNonEmpty(base.ForEach, overlay.ForEach)
	base.Provider = firstNonEmpty(base.Provider, overlay.Provider)
	base.OpenAPI = firstNonEmpty(base.OpenAPI, overlay.OpenAPI)
	base.Operation = firstNonEmpty(base.Operation, overlay.Operation)
	if base.Timeout == nil {
		base.Timeout = overlay.Timeout
	}
	if len(base.With) == 0 {
		base.With = overlay.With
	} else {
		for k, v := range overlay.With {
			if strings.TrimSpace(base.With[k]) == "" {
				base.With[k] = v
			}
		}
	}
	base.Binds = append(base.Binds, overlay.Binds...)
	base.DependsOn = dedupeStrings(append(base.DependsOn, overlay.DependsOn...))
}

func applyProgressiveAnswer(session *Session, plan QuestionPlan, answer string, docs []APIDocument) {
	answer = strings.TrimSpace(answer)
	if answer == "" {
		answer = strings.TrimSpace(plan.SuggestedAnswer)
	}
	if answer == "" {
		return
	}
	slotText := strings.Join(plan.Slots, " ")
	switch {
	case strings.Contains(slotText, "workflow.goal") || strings.Contains(slotText, "workflow.description"):
		if session.Intent.Workflow == nil {
			session.Intent.Workflow = &rollout.WorkflowMeta{}
		}
		session.Intent.Workflow.Description = firstNonEmpty(session.Intent.Workflow.Description, answer)
		session.Intent.Workflow.Name = firstNonEmpty(session.Intent.Workflow.Name, actionName(answer))
		session.Project.Goal = firstNonEmpty(session.Project.Goal, answer)
	case strings.Contains(slotText, "intent.openapi"):
		if doc := matchDocAnswer(answer, docs); doc.RelativePath != "" {
			session.Intent.OpenAPI = doc.RelativePath
		} else {
			session.Intent.OpenAPI = answer
		}
		addMappingClassification(session, MappingClassification{
			Slot:                 "intent.openapi",
			Value:                session.Intent.OpenAPI,
			Source:               mappingSourceUser,
			Confidence:           mappingConfidenceHigh,
			Evidence:             answer,
			Reason:               "User selected the OpenAPI document.",
			RequiresConfirmation: false,
		})
	case strings.Contains(slotText, "operation") || strings.Contains(slotText, "intent.steps"):
		if doc, op := matchOperationAnswer(answer, docs); op != nil {
			session.Intent.OpenAPI = firstNonEmpty(session.Intent.OpenAPI, doc.RelativePath)
			if len(session.Intent.Steps) == 0 {
				session.Intent.Steps = []*rollout.Step{stepFromOperation(op)}
			} else {
				session.Intent.Steps[0].Type = firstNonEmpty(session.Intent.Steps[0].Type, "http")
				session.Intent.Steps[0].Do = firstNonEmpty(session.Intent.Steps[0].Do, op.Summary, operationLabel(op))
				session.Intent.Steps[0].Operation = op.OperationID
			}
			addMappingClassification(session, MappingClassification{
				Slot:                 stepOperationSlot(session.Intent.Steps[0]),
				Value:                op.OperationID,
				Source:               mappingSourceUser,
				Confidence:           mappingConfidenceHigh,
				Evidence:             answer,
				Reason:               "User selected the API operation.",
				RequiresConfirmation: false,
			})
		} else if len(session.Intent.Steps) == 0 {
			stepType := "fnct"
			operation := ""
			if strings.TrimSpace(session.Intent.OpenAPI) != "" {
				stepType = "http"
				operation = slugIdent(answer)
			}
			session.Intent.Steps = []*rollout.Step{{
				Name:      actionName(answer),
				Type:      stepType,
				Do:        answer,
				Operation: operation,
			}}
			if operation != "" {
				addMappingClassification(session, MappingClassification{
					Slot:                 stepOperationSlot(session.Intent.Steps[0]),
					Value:                operation,
					Source:               mappingSourceUser,
					Confidence:           mappingConfidenceHigh,
					Evidence:             answer,
					Reason:               "User provided the API operation.",
					RequiresConfirmation: false,
				})
			}
		}
	case strings.Contains(slotText, ".with"):
		assignments := parseAssignments(answer)
		if len(assignments) == 0 && len(plan.Slots) == 1 && strings.Contains(plan.Slots[0], ".with.") {
			if field := fieldFromWithSlot(plan.Slots[0]); field != "" {
				assignments[field] = answer
			}
		}
		for _, step := range session.Intent.Steps {
			if step == nil {
				continue
			}
			if step.With == nil {
				step.With = map[string]string{}
			}
			for field, source := range assignments {
				step.With[field] = source
				addMappingClassification(session, MappingClassification{
					Slot:                 stepWithSlot(step, field),
					Value:                source,
					Source:               mappingSourceUser,
					Confidence:           mappingConfidenceHigh,
					Evidence:             answer,
					Reason:               "User provided the request field mapping.",
					RequiresConfirmation: false,
				})
			}
		}
		addInputsFromAssignments(session, assignments)
		addCredentialsFromAssignments(session, assignments)
	case strings.Contains(slotText, "credentials"):
		session.Credentials = credentialBindings(answer)
		session.CredentialsSet = true
		for _, credential := range session.Credentials {
			addMappingClassification(session, MappingClassification{
				Slot:                 "credentials",
				Value:                credential,
				Source:               mappingSourceUser,
				Confidence:           mappingConfidenceHigh,
				Evidence:             answer,
				Reason:               "User provided the credential binding name.",
				RequiresConfirmation: false,
			})
		}
		if len(session.Credentials) == 1 {
			fillCredentialFields(session, docs, session.Credentials[0])
		}
	case strings.Contains(slotText, "intent.inputs"):
		session.Intent.Inputs = mergeInputsByName(session.Intent.Inputs, parseInputs(answer))
	case strings.Contains(slotText, "intent.outputs"):
		session.Intent.Outputs = mergeOutputsByName(session.Intent.Outputs, parseOutputs(answer, lastStepName(session.Intent.Steps)))
		for _, output := range session.Intent.Outputs {
			if output == nil || strings.TrimSpace(output.Name) == "" || strings.TrimSpace(output.From) == "" {
				continue
			}
			addMappingClassification(session, MappingClassification{
				Slot:                 "intent.outputs." + output.Name,
				Value:                output.Name + "=" + output.From,
				Source:               mappingSourceUser,
				Confidence:           mappingConfidenceHigh,
				Evidence:             answer,
				Reason:               "User provided the workflow output mapping.",
				RequiresConfirmation: false,
			})
		}
	case strings.Contains(slotText, "safety"):
		if scope := projectwizard.NormalizeSideEffectScope(answer); scope != "" {
			session.SideEffectScope = scope
		}
		session.Safety = answer
		session.SafetySet = true
	}
}

func deterministicPrefill(session *Session, docs []APIDocument) bool {
	if session == nil {
		return false
	}
	changed := false
	walkSteps(session.Intent.Steps, func(step *rollout.Step) {
		if step == nil {
			return
		}
		op, ok := operationForStep(*session, docs, step)
		if !ok {
			return
		}
		for _, field := range missingRequiredFields(step, op) {
			if looksCredentialField(field, op) {
				if len(session.Credentials) != 1 {
					continue
				}
				source := "credentials." + session.Credentials[0]
				if setStepWithIfEmpty(step, field, source) {
					addDeterministicPrefillAssumption(session, step, field, source, "credential binding", "The selected operation security metadata identifies this request field as a credential field.")
					addMappingClassification(session, MappingClassification{
						Slot:                 stepWithSlot(step, field),
						Value:                source,
						Source:               mappingSourceDeterministic,
						Confidence:           mappingConfidenceHigh,
						Evidence:             "credential binding " + source,
						Reason:               "The selected operation security metadata identifies this request field as a credential field.",
						RequiresConfirmation: false,
					})
					changed = true
				}
				continue
			}
			inputName, ok := exactInputMatch(session.Intent.Inputs, field)
			if !ok {
				continue
			}
			source := "inputs." + inputName
			if setStepWithIfEmpty(step, field, source) {
				addDeterministicPrefillAssumption(session, step, field, source, "runtime input", "A declared runtime input exactly matches the required request field.")
				addMappingClassification(session, MappingClassification{
					Slot:                 stepWithSlot(step, field),
					Value:                source,
					Source:               mappingSourceDeterministic,
					Confidence:           mappingConfidenceHigh,
					Evidence:             "runtime input " + source,
					Reason:               "A declared runtime input exactly matches the required request field.",
					RequiresConfirmation: false,
				})
				changed = true
			}
		}
	})
	if len(session.Intent.Outputs) == 0 {
		if output, ok := deterministicSingleStepOutput(session.Intent.Steps); ok {
			session.Intent.Outputs = []*rollout.Output{output}
			addDeterministicOutputAssumption(session, output)
			addMappingClassification(session, MappingClassification{
				Slot:                 "intent.outputs." + output.Name,
				Value:                output.Name + "=" + output.From,
				Source:               mappingSourceFallbackDefault,
				Confidence:           mappingConfidenceReview,
				Evidence:             output.From,
				Reason:               "A single executable step can expose its received body as the workflow result.",
				RequiresConfirmation: true,
			})
			changed = true
		}
	}
	return changed
}

func setStepWithIfEmpty(step *rollout.Step, field, source string) bool {
	field = strings.TrimSpace(field)
	source = strings.TrimSpace(source)
	if step == nil || field == "" || source == "" {
		return false
	}
	if step.With == nil {
		step.With = map[string]string{}
	}
	if strings.TrimSpace(step.With[field]) != "" {
		return false
	}
	step.With[field] = source
	return true
}

func exactInputMatch(inputs []*rollout.Input, field string) (string, bool) {
	field = strings.TrimSpace(field)
	slugged := slugIdent(field)
	matches := map[string]bool{}
	for _, input := range inputs {
		if input == nil {
			continue
		}
		name := strings.TrimSpace(input.Name)
		if name == "" {
			continue
		}
		if name == field || name == slugged {
			matches[name] = true
		}
	}
	if len(matches) != 1 {
		return "", false
	}
	for name := range matches {
		return name, true
	}
	return "", false
}

func deterministicSingleStepOutput(steps []*rollout.Step) (*rollout.Output, bool) {
	if len(steps) != 1 || !prefillOutputStep(steps[0]) {
		return nil, false
	}
	stepName := strings.TrimSpace(steps[0].Name)
	if stepName == "" {
		return nil, false
	}
	return &rollout.Output{Name: "result", From: stepName + ".received_body"}, true
}

func prefillOutputStep(step *rollout.Step) bool {
	if step == nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(step.Type)) {
	case "switch", "merge", "loop", "branch":
		return false
	case "http", "openapi":
		return strings.TrimSpace(step.Operation) != ""
	default:
		return strings.TrimSpace(step.Name) != ""
	}
}

func addDeterministicPrefillAssumption(session *Session, step *rollout.Step, field, source, sourceKind, reason string) {
	stepName := firstNonEmpty(step.Name, "step")
	slot := "steps." + stepName + ".with." + field
	assumption := Assumption{
		ID:                   "deterministic_prefill_" + slugIdent(slot),
		Slot:                 slot,
		Value:                field + "=" + source,
		Reason:               reason,
		Evidence:             sourceKind + " " + source,
		Risk:                 "low",
		RequiresConfirmation: true,
	}
	session.Assumptions = mergeAssumptions(session.Assumptions, []Assumption{assumption})
}

func addDeterministicOutputAssumption(session *Session, output *rollout.Output) {
	if output == nil {
		return
	}
	assumption := Assumption{
		ID:                   "deterministic_prefill_output_" + slugIdent(output.Name),
		Slot:                 "intent.outputs." + output.Name,
		Value:                output.Name + "=" + output.From,
		Reason:               "A single executable step can expose its received body as the workflow result.",
		Evidence:             output.From,
		Risk:                 "low",
		RequiresConfirmation: true,
	}
	session.Assumptions = mergeAssumptions(session.Assumptions, []Assumption{assumption})
}

func needsAPIDoc(session Session, docs []APIDocument) bool {
	if strings.TrimSpace(session.Intent.OpenAPI) != "" {
		return false
	}
	if len(docs) == 1 && session.Intent.RequiresOpenAPI() {
		return false
	}
	if session.Intent.RequiresOpenAPI() {
		return true
	}
	for _, step := range session.Intent.Steps {
		if step == nil {
			continue
		}
		stepType := strings.ToLower(strings.TrimSpace(step.Type))
		if (stepType == "http" || stepType == "openapi") && strings.TrimSpace(step.OpenAPI) == "" {
			return true
		}
	}
	return false
}

func operationForStep(session Session, docs []APIDocument, step *rollout.Step) (*rollout.OperationInfo, bool) {
	if step == nil || strings.TrimSpace(step.Operation) == "" {
		return nil, false
	}
	docPath := firstNonEmpty(step.OpenAPI, session.Intent.OpenAPI)
	if docPath == "" && len(docs) == 1 {
		docPath = docs[0].RelativePath
	}
	if op, ok := operationByID(docs, docPath, step.Operation); ok {
		return op, true
	}
	for _, doc := range docs {
		for _, op := range doc.Operations {
			if op != nil && op.OperationID == step.Operation {
				return op, true
			}
		}
	}
	return nil, false
}

func missingRequiredFields(step *rollout.Step, op *rollout.OperationInfo) []string {
	available := map[string]bool{}
	for field, value := range step.With {
		if strings.TrimSpace(value) != "" {
			available[field] = true
		}
	}
	for _, bind := range step.Binds {
		if bind == nil {
			continue
		}
		for field, value := range bind.Fields {
			if strings.TrimSpace(value) != "" {
				available[field] = true
			}
		}
	}
	var missing []string
	for _, field := range requiredMappingFields(op) {
		if !available[field] {
			missing = append(missing, field)
		}
	}
	return missing
}

func requiredMappingFields(op *rollout.OperationInfo) []string {
	if op == nil {
		return nil
	}
	var out []string
	for _, parameter := range op.Parameters {
		if parameter != nil && parameter.Required {
			out = append(out, parameter.Name)
		}
	}
	if op.RequestBody != nil && op.RequestBody.Required {
		bodyFields := requiredLeafBodyFields(op.RequestBody)
		if len(bodyFields) == 0 {
			out = append(out, "body")
		} else {
			out = append(out, bodyFields...)
		}
	}
	for _, security := range op.Security {
		if field := securityFieldName(security); field != "" {
			out = append(out, field)
		}
	}
	return dedupeStrings(out)
}

func requiredLeafBodyFields(body *rollout.RequestBodyInfo) []string {
	var required []requestBodyFieldContext
	for _, field := range flattenRequestBodyFields(body) {
		if field.Required {
			required = append(required, field)
		}
	}
	var out []string
	for _, field := range required {
		if requestBodyFieldHasRequiredDescendant(field.Path, required) {
			continue
		}
		out = append(out, field.Path)
	}
	sort.Strings(out)
	return out
}

func requestBodyFieldHasRequiredDescendant(path string, fields []requestBodyFieldContext) bool {
	prefix := path + "."
	arrayPrefix := path + "[]."
	for _, field := range fields {
		if field.Path == path {
			continue
		}
		if strings.HasPrefix(field.Path, prefix) || strings.HasPrefix(field.Path, arrayPrefix) {
			return true
		}
	}
	return false
}

func validateOpenAPIRequestMappings(session Session, step *rollout.Step, op *rollout.OperationInfo, slotPrefix string) []ReadinessIssue {
	if step == nil || op == nil {
		return nil
	}
	fields := openAPIRequestFieldTypes(op)
	credentialSet := map[string]bool{}
	for _, credential := range session.Credentials {
		if credential != "" {
			credentialSet[credential] = true
		}
	}
	var issues []ReadinessIssue
	add := func(code, slot, message string) {
		issues = append(issues, ReadinessIssue{
			Code:     code,
			Slot:     slot,
			Severity: readinessBlocking,
			Message:  message,
		})
	}
	validate := func(field, source, slot string) {
		field = strings.TrimSpace(field)
		source = strings.TrimSpace(source)
		if field == "" || source == "" {
			return
		}
		for _, credential := range credentialCandidates(source) {
			if !credentialSet[credential] {
				add("undeclared_credential_reference", slot, "Request field "+field+" references undeclared credential binding "+credential+".")
			}
		}
		info, ok := fields[field]
		if !ok {
			if invalidBodyPath(field, fields) {
				add("invalid_request_body_path", slot, "Request body path "+field+" is not present in the selected operation schema.")
			} else {
				add("invented_request_field", slot, "Request field "+field+" is not defined by the selected OpenAPI operation.")
			}
			return
		}
		if incompatibleLiteralType(source, info.Type) {
			add("incompatible_request_value_type", slot, "Request field "+field+" expects "+info.Type+" but is mapped from incompatible literal "+source+".")
		}
	}
	for field, source := range step.With {
		validate(field, source, slotPrefix+".with."+field)
	}
	for i, bind := range step.Binds {
		if bind == nil {
			continue
		}
		for field, source := range bind.Fields {
			validate(field, source, fmt.Sprintf("%s.bind.%d.%s", slotPrefix, i+1, field))
		}
	}
	return issues
}

type openAPIRequestFieldInfo struct {
	Type string
	Body bool
}

func openAPIRequestFieldTypes(op *rollout.OperationInfo) map[string]openAPIRequestFieldInfo {
	out := map[string]openAPIRequestFieldInfo{}
	if op == nil {
		return out
	}
	for _, parameter := range op.Parameters {
		if parameter == nil || strings.TrimSpace(parameter.Name) == "" {
			continue
		}
		out[parameter.Name] = openAPIRequestFieldInfo{Type: parameter.Type}
	}
	for _, security := range op.Security {
		if field := securityFieldName(security); field != "" {
			out[field] = openAPIRequestFieldInfo{Type: "string"}
		}
	}
	if op.RequestBody != nil {
		for _, field := range flattenRequestBodyFields(op.RequestBody) {
			if strings.TrimSpace(field.Path) == "" {
				continue
			}
			out[field.Path] = openAPIRequestFieldInfo{Type: field.Type, Body: true}
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

func operationNeedsCredential(op *rollout.OperationInfo) bool {
	return op != nil && len(op.Security) > 0
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
		"missing_goal":                          0,
		"conflicting_mapping":                   1,
		"low_confidence_mapping":                2,
		"missing_api_doc":                       3,
		"missing_operation":                     4,
		"undeclared_credential_reference":       5,
		"invented_request_field":                6,
		"invalid_request_body_path":             7,
		"incompatible_request_value_type":       8,
		"missing_required_request_values":       9,
		"missing_credential_bindings":           10,
		"missing_runtime_inputs":                11,
		"missing_outputs":                       12,
		"missing_side_effect_policy":            13,
		"optional_timeout_idempotency_controls": 14,
		"intent_render_invalid":                 15,
	}
	sort.SliceStable(issues, func(i, j int) bool {
		return priority[issues[i].Code] < priority[issues[j].Code]
	})
	return issues
}

func suggestedAnswerForCode(code string, session Session, docs []APIDocument) string {
	switch code {
	case "missing_api_doc":
		return suggestedDocAnswer(docs)
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

func suggestedDocAnswer(docs []APIDocument) string {
	if len(docs) == 0 {
		return "openapi/<api>.yaml"
	}
	return docs[0].RelativePath
}

func suggestedOperationAnswer(docs []APIDocument) string {
	for _, doc := range docs {
		for _, op := range doc.Operations {
			if op != nil && op.OperationID != "" {
				return op.OperationID
			}
		}
	}
	return "Describe the action in business terms."
}

func suggestedFieldAssignments(session Session, docs []APIDocument, step *rollout.Step, op *rollout.OperationInfo, fields []string) string {
	var parts []string
	for _, field := range fields {
		parts = append(parts, field+"="+suggestedFieldSource(session, docs, step, op, field))
	}
	return strings.Join(parts, ", ")
}

func suggestedCredentialName(op *rollout.OperationInfo) string {
	if op == nil || len(op.Security) == 0 {
		return "api_token"
	}
	return slugIdent(op.Security[0])
}

func suggestedCredentialNameForOperation(session Session, docs []APIDocument, step *rollout.Step, op *rollout.OperationInfo) string {
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

func suggestedFieldSource(session Session, docs []APIDocument, step *rollout.Step, op *rollout.OperationInfo, field string) string {
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
		if len(session.Credentials) == 1 {
			return "credentials." + session.Credentials[0]
		}
		return "credentials." + suggestedCredentialNameForOperation(session, docs, step, op)
	}
	if value, ok := safeLiteralDefault(field, op); ok {
		return value
	}
	return "inputs." + slugIdent(field)
}

func inputMatchByLeafOrDescription(inputs []*rollout.Input, field string, op *rollout.OperationInfo) (string, bool) {
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

func requestFieldDescription(op *rollout.OperationInfo, field string) string {
	if op == nil {
		return ""
	}
	for _, parameter := range op.Parameters {
		if parameter != nil && parameter.Name == field {
			return parameter.Description
		}
	}
	if op.RequestBody != nil {
		for _, bodyField := range flattenRequestBodyFields(op.RequestBody) {
			if bodyField.Path == field {
				return bodyField.Description
			}
		}
	}
	return ""
}

func suggestedCredentialField(field string, op *rollout.OperationInfo) bool {
	if op == nil || len(op.Security) == 0 {
		return false
	}
	for _, security := range op.Security {
		if field == securityFieldName(security) {
			return true
		}
	}
	return strings.EqualFold(field, "Authorization")
}

func safeLiteralDefault(field string, op *rollout.OperationInfo) (string, bool) {
	if op == nil || op.RequestBody == nil || secretLikeField(field) {
		return "", false
	}
	for _, bodyField := range flattenRequestBodyFields(op.RequestBody) {
		if bodyField.Path != field || !safeScalarType(bodyField.Type) {
			continue
		}
		for _, value := range []any{bodyField.Default, firstEnumValue(bodyField.Enum), bodyField.Example} {
			if formatted, ok := formatSafeLiteral(value, bodyField.Type); ok {
				return formatted, true
			}
		}
	}
	return "", false
}

func firstEnumValue(values []any) any {
	if len(values) == 0 {
		return nil
	}
	return values[0]
}

func safeScalarType(typ string) bool {
	switch strings.ToLower(strings.TrimSpace(typ)) {
	case "", "string", "integer", "int", "number", "float", "double", "boolean", "bool":
		return true
	default:
		return false
	}
}

func formatSafeLiteral(value any, typ string) (string, bool) {
	if value == nil {
		return "", false
	}
	text := strings.TrimSpace(fmt.Sprint(value))
	if text == "" {
		return "", false
	}
	switch strings.ToLower(strings.TrimSpace(typ)) {
	case "integer", "int":
		if !isIntegerLiteral(text) {
			return "", false
		}
	case "number", "float", "double":
		if !isNumberLiteral(text) {
			return "", false
		}
	case "boolean", "bool":
		if !isBoolLiteral(text) {
			return "", false
		}
		text = strings.ToLower(text)
	}
	if expressionLikeSource(text) {
		return "", false
	}
	return text, true
}

func secretLikeField(field string) bool {
	normalized := strings.ToLower(field)
	for _, token := range []string{"authorization", "auth", "token", "secret", "password", "passwd", "api_key", "apikey", "key"} {
		if strings.Contains(normalized, token) {
			return true
		}
	}
	return false
}

func documentForStep(session Session, docs []APIDocument, step *rollout.Step, op *rollout.OperationInfo) (APIDocument, bool) {
	if op == nil {
		return APIDocument{}, false
	}
	docPath := session.Intent.OpenAPI
	if step != nil {
		docPath = firstNonEmpty(step.OpenAPI, docPath)
	}
	for _, doc := range docs {
		if docPath != "" && doc.RelativePath != docPath {
			continue
		}
		for _, candidate := range doc.Operations {
			if candidate == op || (candidate != nil && candidate.OperationID == op.OperationID) {
				return doc, true
			}
		}
	}
	if docPath == "" {
		for _, doc := range docs {
			for _, candidate := range doc.Operations {
				if candidate == op || (candidate != nil && candidate.OperationID == op.OperationID) {
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
		scope = projectwizard.SideEffectSandboxOnly
	}
	return scope
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
		for _, field := range requiredFields(op) {
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

func looksCredentialField(field string, op *rollout.OperationInfo) bool {
	lowerField := strings.ToLower(field)
	if strings.Contains(lowerField, "auth") || strings.Contains(lowerField, "token") || strings.Contains(lowerField, "key") {
		return true
	}
	for _, security := range op.Security {
		if field == securityFieldName(security) {
			return true
		}
	}
	return false
}

func matchDocAnswer(answer string, docs []APIDocument) APIDocument {
	answer = strings.TrimSpace(answer)
	for i, doc := range docs {
		if answer == doc.RelativePath || answer == fmt.Sprint(i+1) || strings.EqualFold(answer, doc.Title) {
			return doc
		}
	}
	return APIDocument{}
}

func matchOperationAnswer(answer string, docs []APIDocument) (APIDocument, *rollout.OperationInfo) {
	answer = strings.TrimSpace(answer)
	for _, doc := range docs {
		for i, op := range doc.Operations {
			if op == nil {
				continue
			}
			if answer == op.OperationID || answer == fmt.Sprint(i+1) || strings.Contains(strings.ToLower(operationLabel(op)), strings.ToLower(answer)) {
				return doc, op
			}
		}
	}
	return APIDocument{}, nil
}

func stepFromOperation(op *rollout.OperationInfo) *rollout.Step {
	return &rollout.Step{
		Name:      actionName(firstNonEmpty(op.OperationID, op.Summary, op.Path)),
		Type:      "http",
		Do:        firstNonEmpty(op.Summary, operationLabel(op)),
		Operation: op.OperationID,
	}
}
