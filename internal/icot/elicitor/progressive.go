package elicitor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/OpenUdon/apitools"
	"github.com/OpenUdon/apitools/catalog"
	"github.com/OpenUdon/openudon/internal/authoring"
	"github.com/OpenUdon/openudon/internal/projectwizard"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

type ReadinessIssue = authoring.ReadinessIssue
type QuestionPlan = authoring.InteractiveQuestion

const (
	readinessBlocking = "blocking"
	readinessWarning  = "warning"
)

func runProgressive(ctx context.Context, in io.Reader, out io.Writer, seed Session, opts Options) (Artifacts, error) {
	extractor := opts.Extractor
	if extractor == nil {
		extractor = NewNoopExtractor()
	}
	session := seed
	session.Normalize()

	projectText := projectwizard.Render(session.Project)
	docs, err := DiscoverLocalAPIs(opts.ExampleDir, projectText)
	if err != nil {
		return Artifacts{}, err
	}
	openingBrief := ""
	if session.Intent.Workflow != nil {
		openingBrief = strings.TrimSpace(session.Intent.Workflow.Description)
	}
	draftJSONErrorReported := false
	skipNextDraft := opts.DisableAIDraft
	catalogRetrievalAttempted := false
	reportedDraftEvents := 0
	questionDrafted := map[string]bool{}
	if openingBrief != "" && shouldRetrieveCatalogArtifacts(session, docs) {
		catalogRetrievalAttempted = true
		if err := retrieveCatalogArtifactsForSession(out, session, opts.ExampleDir, opts.CatalogHintOptions); err != nil {
			return Artifacts{}, err
		}
		projectText = projectwizard.Render(session.Project)
		docs, err = DiscoverLocalAPIs(opts.ExampleDir, projectText)
		if err != nil {
			return Artifacts{}, err
		}
		clearUnavailableAPIDocumentRefs(&session, docs)
	}
	attemptCatalogRetrieval := func(session Session) error {
		if catalogRetrievalAttempted {
			return nil
		}
		if !shouldRetrieveCatalogArtifacts(session, docs) {
			return nil
		}
		catalogRetrievalAttempted = true
		return retrieveCatalogArtifactsForSession(out, session, opts.ExampleDir, opts.CatalogHintOptions)
	}
	nextSessionEvents := func(session Session) []authoring.PromptEvent {
		return catalogPlanEvents(session, &reportedDraftEvents)
	}
	hooks := authoring.ProgressiveLoopHooks[Session, APIDocument, Artifacts]{
		Session:       session,
		Documents:     docs,
		Opening:       openingBrief,
		Brief:         projectText,
		NoLLM:         opts.NoLLM,
		MaxAttempts:   20,
		OpeningPrompt: "Tell me what you want this API/workflow to accomplish. Include inputs, API actions, outputs, and safety constraints if you know them. Do not paste secrets.",
		Extractor:     extractor,
		Normalize: func(session *Session) {
			session.Normalize()
		},
		ApplyOpeningAnswer: func(session *Session, answer string, docs []APIDocument) error {
			applyProgressiveAnswer(session, QuestionPlan{Slots: []string{"workflow.goal"}}, answer, docs)
			hints, err := BuildCatalogHints(answer, opts.CatalogHintOptions)
			if err != nil {
				fmt.Fprintf(out, "icot: apitools catalog advisory skipped: %v\n", err)
			} else {
				printCatalogHints(out, hints)
				applied, err := planOpeningCatalogArtifacts(ctx, out, extractor, session, answer, hints, opts)
				if err != nil {
					return err
				}
				if applied {
					catalogRetrievalAttempted = true
				} else {
					addCatalogPlanSteps(session, hints)
				}
			}
			return nil
		},
		OpeningEvents: nextSessionEvents,
		RefreshDocuments: func(session Session, docs []APIDocument) ([]APIDocument, error) {
			if err := attemptCatalogRetrieval(session); err != nil {
				return nil, err
			}
			projectText := projectwizard.Render(session.Project)
			return DiscoverLocalAPIs(opts.ExampleDir, projectText)
		},
		ShouldDraft: func(session Session, docs []APIDocument, issues []ReadinessIssue) bool {
			if skipNextDraft {
				skipNextDraft = false
				return false
			}
			return readyForSelectedOperationDraft(session, docs, issues)
		},
		DraftQuestion: func(ctx context.Context, session *Session, docs []APIDocument, issues []ReadinessIssue, question QuestionPlan) (bool, error) {
			key := questionDraftKey(question)
			if key == "" || questionDrafted[key] {
				return false, nil
			}
			if !questionTargetsRequestMappings(question) {
				return false, nil
			}
			questionDrafted[key] = true
			if !readyForSelectedOperationDraft(*session, docs, issues) {
				return false, nil
			}
			return draftRequestMappings(ctx, out, extractor, session, docs, issues, question)
		},
		RankDocuments: rankDocuments,
		DeterministicPrefill: func(session *Session, docs []APIDocument) bool {
			return deterministicPrefill(session, docs)
		},
		LooksLikeSession: LooksLikeSession,
		MergeDraft: func(base, draft Session, docs []APIDocument) Session {
			merged := mergeProgressiveSessions(base, draft, docs)
			defaultSingleOpenAPIDoc(&merged, docs)
			return merged
		},
		DraftResultSummary: func(session Session) any {
			return map[string]any{
				"steps":       len(session.Intent.Steps),
				"inputs":      len(session.Intent.Inputs),
				"outputs":     len(session.Intent.Outputs),
				"assumptions": session.Assumptions,
			}
		},
		DraftEvents: func(session Session) []authoring.PromptEvent {
			return nextSessionEvents(session)
		},
		AfterDraft: func(session Session) error {
			printSummary(out, session)
			return nil
		},
		OnDraftError: func(err error) {
			if strings.Contains(err.Error(), "OpenAPI ranking skipped") {
				fmt.Fprintf(out, "icot: %v\n", err)
				return
			}
			message, isJSON := progressiveDraftErrorMessage(err)
			if isJSON {
				if draftJSONErrorReported {
					return
				}
				draftJSONErrorReported = true
			}
			fmt.Fprintf(out, "icot: AI draft skipped: %s\n", message)
		},
		CheckReadiness: CheckReadiness,
		Ready:          progressiveReady,
		PlanQuestion:   PlanNextQuestion,
		ApplyAnswer: func(session *Session, plan QuestionPlan, answer string, docs []APIDocument) error {
			handled, err := applyCatalogDocumentAnswer(out, session, plan, answer, docs, opts.ExampleDir)
			if err != nil {
				return err
			}
			if handled {
				return nil
			}
			applyProgressiveAnswer(session, plan, answer, docs)
			defaultSingleOpenAPIDoc(session, docs)
			return nil
		},
		FinalConfirm: func(prompts *authoring.PromptSession, session *Session, docs []APIDocument, events *[]authoring.PromptEvent) (Artifacts, error) {
			return finalProgressiveConfirmationLoop(out, &prompter{PromptSession: prompts, out: out}, session, docs, opts.DraftPath, events)
		},
		FinalResultSummary: func(artifacts Artifacts) any {
			return map[string]any{
				"intent_hcl_bytes": len(artifacts.IntentHCL),
				"project_md_bytes": len(artifacts.ProjectMD),
				"assumptions":      artifacts.Session.Assumptions,
			}
		},
	}
	artifacts, err := authoring.RunProgressiveWithLifecycle(ctx, in, out, hooks, authoring.ProgressiveLifecycleOptions[Session, APIDocument, Artifacts]{
		DraftPath:         opts.DraftPath,
		TranscriptPath:    opts.TranscriptPath,
		TranscriptVersion: "openudon.icot-transcript.v1",
		Normalize: func(session *Session) {
			session.Normalize()
		},
		LooksLikeSession: LooksLikeSession,
		Opening: func(session Session) string {
			if session.Intent.Workflow == nil {
				return ""
			}
			return strings.TrimSpace(session.Intent.Workflow.Description)
		},
		TranscriptSession: func(artifacts Artifacts) any {
			return artifacts.Session
		},
	})
	if errors.Is(err, authoring.ErrCanceled) {
		return artifacts, ErrCanceled
	}
	return artifacts, err
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

func progressiveDraftErrorMessage(err error) (string, bool) {
	if err == nil {
		return "", false
	}
	text := err.Error()
	switch {
	case strings.Contains(text, "cannot unmarshal") || strings.Contains(text, "invalid character") || strings.Contains(text, "no JSON object found"):
		return "model returned invalid draft JSON; continuing with deterministic questions", true
	default:
		return text, false
	}
}

func finalProgressiveConfirmationLoop(out io.Writer, p *prompter, session *Session, docs []APIDocument, draftPath string, events *[]TranscriptEvent) (Artifacts, error) {
	for {
		artifacts, err := RenderArtifacts(*session)
		if err != nil {
			if handled, handleErr := answerFinalBlockingQuestion(out, p, session, docs, draftPath); handled || handleErr != nil {
				if handleErr != nil {
					return Artifacts{}, handleErr
				}
				if events != nil {
					*events = append(*events, TranscriptEvent{Kind: "confirmation_repair", Data: ""})
				}
				continue
			}
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
		if firstFinalRepairIssue(issues).Code != "" {
			if handled, handleErr := answerFinalBlockingQuestion(out, p, session, docs, draftPath); handled || handleErr != nil {
				if handleErr != nil {
					return Artifacts{}, handleErr
				}
				if events != nil {
					*events = append(*events, TranscriptEvent{Kind: "confirmation_repair", Data: ""})
				}
				continue
			}
		}
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
	base.Intent.Source = firstNonEmpty(base.Intent.Source, overlay.Intent.Source)
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
	base.DraftOperations = appendOperationDetailRefs(base.DraftOperations, overlay.DraftOperations)
	base.DraftEvents = append(base.DraftEvents, overlay.DraftEvents...)
	base.Normalize()
	return base
}

func defaultSingleOpenAPIDoc(session *Session, docs []APIDocument) {
	if session == nil || intentAPISourceRef(session.Intent) != "" || len(docs) != 1 || !session.Intent.RequiresOpenAPI() {
		return
	}
	setIntentAPISourceFromDoc(session, docs[0])
	addMappingClassification(session, MappingClassification{
		Slot:                 "intent.source",
		Value:                docs[0].RelativePath,
		Source:               mappingSourceFallbackDefault,
		Confidence:           mappingConfidenceReview,
		Evidence:             docs[0].RelativePath,
		Reason:               "Only one local API source document is available for API-backed steps.",
		RequiresConfirmation: true,
	})
}

func addCatalogPlanSteps(session *Session, hints []CatalogHint) {
	if session == nil || len(hints) == 0 || len(session.Intent.Steps) > 0 {
		return
	}
	for _, hint := range hints {
		name := slugIdent(firstNonEmpty(hint.Provider.ID, hint.Provider.DisplayName))
		if name == "" {
			continue
		}
		session.Intent.Steps = append(session.Intent.Steps, &rollout.Step{
			Name:     name,
			Type:     "http",
			Do:       firstNonEmpty(hint.Provider.DisplayName, hint.Provider.ID) + " API action for this workflow.",
			Provider: hint.Provider.ID,
		})
	}
	if len(session.Intent.Steps) == 0 {
		return
	}
	session.Assumptions = mergeAssumptions(session.Assumptions, []Assumption{{
		ID:                   "catalog_provider_order",
		Slot:                 "intent.steps",
		Value:                strings.Join(CatalogProviderPlan(hints), " -> "),
		Reason:               "The workflow brief mentions first-class catalog providers in this order; iCoT will ask for a concrete operationId for each provider.",
		Evidence:             strings.Join(CatalogProviderPlan(hints), " -> "),
		Risk:                 "medium",
		RequiresConfirmation: true,
	}})
}

func planOpeningCatalogArtifacts(ctx context.Context, out io.Writer, extractor Extractor, session *Session, opening string, hints []CatalogHint, opts Options) (bool, error) {
	if opts.NoLLM || extractor == nil || session == nil || len(hints) == 0 {
		return false, nil
	}
	request := BuildCatalogPlanRequest(opening, *session, hints, opts.ExampleDir)
	if len(request.Candidates) == 0 {
		return false, nil
	}
	session.DraftEvents = append(session.DraftEvents, TranscriptEvent{Kind: "catalog_plan_call", Data: map[string]any{
		"opening":    request.Opening,
		"candidates": request.Candidates,
	}})
	response, err := extractor.CatalogPlan(ctx, request)
	if err != nil {
		fmt.Fprintf(out, "icot: catalog plan unavailable: %v\n", err)
		session.DraftEvents = append(session.DraftEvents, TranscriptEvent{Kind: "catalog_plan_rejected", Data: map[string]any{
			"error": err.Error(),
		}})
		return false, nil
	}
	application, err := applyCatalogPlanResponse(out, session, hints, opts.ExampleDir, response)
	if err != nil {
		return false, err
	}
	return application.Applied, nil
}

func retrieveCatalogArtifactsForSession(out io.Writer, session Session, exampleDir string, opts CatalogHintOptions) error {
	result, err := MigrateCatalogArtifacts(catalogQueryForSession(session), exampleDir, opts)
	if err != nil {
		return err
	}
	for _, candidate := range result.Existing {
		fmt.Fprintf(out, "icot: using existing apitools API document %s\n", candidate.RelativePath)
	}
	for _, candidate := range result.Copied {
		if candidate.Kind == catalog.SpecKind("advisory-overlay") {
			fmt.Fprintf(out, "icot: retrieved %s advisory OpenAPI overlay from apitools to %s\n", candidate.ProviderName, candidate.RelativePath)
			continue
		}
		fmt.Fprintf(out, "icot: retrieved %s API document from apitools to %s\n", candidate.ProviderName, candidate.RelativePath)
	}
	for _, hint := range result.Missing {
		fmt.Fprintf(out, "icot: no first-class OpenAPI is available for %s; cannot continue to operation selection until an artifact is generated/provided.\n", firstNonEmpty(hint.Provider.DisplayName, hint.Provider.ID))
	}
	return nil
}

func shouldRetrieveCatalogArtifacts(session Session, docs []APIDocument) bool {
	hints := CatalogHintsForSession(session)
	return shouldRetrieveCatalogArtifactsForHints(session, docs, hints)
}

func shouldRetrieveCatalogArtifactsForHints(session Session, docs []APIDocument, hints []CatalogHint) bool {
	if len(hints) == 0 {
		return false
	}
	if len(missingLocalAPIDocumentRefs(session, docs)) > 0 {
		return true
	}
	if len(docs) == 0 {
		return true
	}
	for _, hint := range hints {
		if catalogProviderHasLocalDoc(hint, docs) {
			continue
		}
		if len(CatalogMigrationCandidates([]CatalogHint{hint}, "")) > 0 {
			return true
		}
	}
	return false
}

func readyForSelectedOperationDraft(session Session, docs []APIDocument, issues []ReadinessIssue) bool {
	for _, issue := range issues {
		if issue.Severity != readinessBlocking {
			continue
		}
		switch issue.Code {
		case "missing_goal", "missing_api_doc", "missing_operation":
			return false
		}
	}
	return true
}

func draftRequestMappings(ctx context.Context, out io.Writer, extractor Extractor, session *Session, docs []APIDocument, issues []ReadinessIssue, question QuestionPlan) (bool, error) {
	if session == nil || extractor == nil {
		return false, nil
	}
	request := BuildRequestMappingRequest(draftSessionDescription(*session), *session, docs, issues, question)
	if len(request.Steps) == 0 {
		return false, nil
	}
	session.DraftEvents = append(session.DraftEvents, TranscriptEvent{Kind: "request_mapping_draft_call", Data: map[string]any{
		"question": question.Prompt,
		"steps":    requestMappingStepNames(request.Steps),
	}})
	response, err := extractor.RequestMappings(ctx, request)
	if err != nil {
		session.DraftEvents = append(session.DraftEvents, TranscriptEvent{Kind: "request_mapping_draft_error", Data: err.Error()})
		if os.Getenv("OPENUDON_ICOT_DEBUG_JSON") != "" {
			fmt.Fprintf(out, "icot: AI request mapping skipped: %v\n", err)
		} else if message, report := progressiveDraftErrorMessage(err); report {
			fmt.Fprintf(out, "icot: AI request mapping skipped: %s\n", message)
		}
		return false, nil
	}
	application := applyRequestMappingResponse(session, request, response)
	for _, rejected := range application.Rejected {
		fmt.Fprintf(out, "icot: rejected AI request mapping: %s\n", rejected)
	}
	if application.Applied == 0 {
		return false, nil
	}
	fmt.Fprintf(out, "icot: drafted request mappings for %s from selected operation metadata\n", strings.Join(requestMappingStepNames(request.Steps), ", "))
	return true, nil
}

func requestMappingStepNames(steps []RequestMappingStep) []string {
	var names []string
	for _, step := range steps {
		name := strings.TrimSpace(step.Name)
		if name != "" {
			names = append(names, name)
		}
	}
	return names
}

func applyCatalogDocumentAnswer(out io.Writer, session *Session, plan QuestionPlan, answer string, docs []APIDocument, exampleDir string) (bool, error) {
	if session == nil || !questionTargetsOpenAPI(plan) {
		return false, nil
	}
	answer = strings.TrimSpace(answer)
	if answer == "" {
		answer = strings.TrimSpace(plan.SuggestedAnswer)
	}
	if isLocalAPIDocumentRef(answer) {
		if doc := matchDocAnswer(answer, docs); doc.RelativePath != "" {
			setIntentAPISourceFromDoc(session, doc)
			markAPIDocsAccepted(session, "local_api_docs_accepted", "User accepted local API documents for operation selection.")
			return true, nil
		}
		path := filepath.Join(exampleDir, filepath.FromSlash(answer))
		if _, err := os.Stat(path); err != nil {
			clearUnavailableAPIDocumentRefs(session, docs)
			clearMissingStepAPIDocumentRefs(session, answer)
			if os.IsNotExist(err) {
				fmt.Fprintf(out, "icot: local API document not found: %s. Create that file first, or enter an existing file under openapi/, google-discovery/, aws-smithy/, or discovery/.\n", path)
				return true, nil
			}
			return true, err
		}
		fmt.Fprintf(out, "icot: %s exists but is not in local API metadata yet. Check that it is valid OpenAPI/Discovery JSON or YAML, then answer with the same path again.\n", answer)
		session.Intent.Source = filepath.ToSlash(answer)
		return true, nil
	}
	if !isAffirmativeAnswer(answer) {
		return false, nil
	}
	hints := CatalogHintsForSession(*session)
	candidates := CatalogMigrationCandidates(hints, exampleDir)
	if len(candidates) > 0 && len(docs) == 0 {
		result, err := MigrateCatalogArtifactsForSession(*session, exampleDir)
		if err != nil {
			return true, err
		}
		for _, candidate := range result.Existing {
			fmt.Fprintf(out, "icot: using existing catalog API document %s\n", candidate.RelativePath)
		}
		for _, candidate := range result.Copied {
			fmt.Fprintf(out, "icot: migrated %s API document to %s\n", candidate.ProviderName, candidate.RelativePath)
		}
		for _, hint := range result.Missing {
			fmt.Fprintf(out, "icot: no migratable first-class API document found for %s; provide a local OpenAPI file or lowering output before synthesis.\n", firstNonEmpty(hint.Provider.DisplayName, hint.Provider.ID))
		}
		markAPIDocsAccepted(session, "catalog_api_docs_migrated", "Migrated available first-class catalog API documents into this workflow.")
		return true, nil
	}
	if len(docs) > 0 {
		markAPIDocsAccepted(session, "local_api_docs_accepted", "User accepted local API documents for operation selection.")
		if len(docs) == 1 && intentAPISourceRef(session.Intent) == "" {
			setIntentAPISourceFromDoc(session, docs[0])
		}
		return true, nil
	}
	return false, nil
}

func questionTargetsOpenAPI(plan QuestionPlan) bool {
	for _, slot := range plan.Slots {
		if strings.Contains(slot, "intent.openapi") || strings.Contains(slot, "intent.source") {
			return true
		}
	}
	return false
}

func questionTargetsRequestMappings(plan QuestionPlan) bool {
	for _, slot := range plan.Slots {
		if strings.Contains(slot, ".with") {
			return true
		}
	}
	return false
}

func questionDraftKey(plan QuestionPlan) string {
	var parts []string
	parts = append(parts, plan.Slots...)
	sort.Strings(parts)
	key := strings.Join(parts, "|")
	if key == "" {
		key = strings.TrimSpace(plan.Prompt)
	}
	return key
}

func isAffirmativeAnswer(answer string) bool {
	switch strings.ToLower(strings.TrimSpace(answer)) {
	case "y", "yes", "use", "use it", "use them", "migrate", "copy", "ok", "okay":
		return true
	default:
		return false
	}
}

func markAPIDocsAccepted(session *Session, id, reason string) {
	if session == nil {
		return
	}
	session.Assumptions = mergeAssumptions(session.Assumptions, []Assumption{{
		ID:                   id,
		Slot:                 "intent.openapi",
		Value:                "accepted",
		Reason:               reason,
		Evidence:             "user confirmation",
		Risk:                 "low",
		RequiresConfirmation: true,
	}})
}

func clearMissingStepAPIDocumentRefs(session *Session, ref string) {
	if session == nil {
		return
	}
	ref = filepath.ToSlash(strings.TrimSpace(ref))
	walkSteps(session.Intent.Steps, func(step *rollout.Step) {
		if step == nil {
			return
		}
		if filepath.ToSlash(strings.TrimSpace(step.Source)) == ref {
			step.Source = ""
		}
		if filepath.ToSlash(strings.TrimSpace(step.OpenAPI)) == ref {
			step.OpenAPI = ""
		}
	})
}

func clearUnavailableAPIDocumentRefs(session *Session, docs []APIDocument) {
	if session == nil {
		return
	}
	available := map[string]bool{}
	for _, doc := range docs {
		if doc.RelativePath != "" {
			available[filepath.ToSlash(doc.RelativePath)] = true
		}
	}
	if ref := filepath.ToSlash(strings.TrimSpace(session.Intent.Source)); isLocalAPIDocumentRef(ref) && !available[ref] {
		session.Intent.Source = ""
	}
	if ref := filepath.ToSlash(strings.TrimSpace(session.Intent.OpenAPI)); isLocalAPIDocumentRef(ref) && !available[ref] {
		session.Intent.OpenAPI = ""
	}
	walkSteps(session.Intent.Steps, func(step *rollout.Step) {
		if step == nil {
			return
		}
		if ref := filepath.ToSlash(strings.TrimSpace(step.Source)); isLocalAPIDocumentRef(ref) && !available[ref] {
			step.Source = ""
		}
		if ref := filepath.ToSlash(strings.TrimSpace(step.OpenAPI)); isLocalAPIDocumentRef(ref) && !available[ref] {
			step.OpenAPI = ""
		}
	})
}

func apiDocsAccepted(session Session) bool {
	for _, assumption := range session.Assumptions {
		switch assumption.ID {
		case "local_api_docs_accepted", "catalog_api_docs_migrated", "catalog_plan_api_docs_migrated":
			return true
		}
	}
	return false
}

func isOpenAPIDocument(doc APIDocument) bool {
	return strings.HasPrefix(filepath.ToSlash(doc.RelativePath), "openapi/")
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
	base.Source = firstNonEmpty(base.Source, overlay.Source)
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
	case strings.Contains(slotText, "intent.openapi") || strings.Contains(slotText, "intent.source"):
		if doc := matchDocAnswer(answer, docs); doc.RelativePath != "" {
			setIntentAPISourceFromDoc(session, doc)
		} else {
			session.Intent.Source = answer
		}
		addMappingClassification(session, MappingClassification{
			Slot:                 "intent.source",
			Value:                intentAPISourceRef(session.Intent),
			Source:               mappingSourceUser,
			Confidence:           mappingConfidenceHigh,
			Evidence:             answer,
			Reason:               "User selected the API source document.",
			RequiresConfirmation: false,
		})
	case strings.Contains(slotText, "operation") || strings.Contains(slotText, "intent.steps"):
		if doc, op := matchOperationAnswerForPlan(session, plan, answer, docs); op != nil {
			if intentAPISourceRef(session.Intent) == "" {
				setIntentAPISourceFromDoc(session, doc)
			}
			target := targetStepForPlan(session, plan)
			if len(session.Intent.Steps) == 0 {
				step := stepFromOperation(op)
				setStepAPISourceFromDoc(step, doc)
				session.Intent.Steps = []*rollout.Step{step}
			} else {
				if target == nil {
					target = session.Intent.Steps[0]
				}
				target.Type = firstNonEmpty(target.Type, "http")
				target.Do = firstNonEmpty(target.Do, op.Summary, operationLabel(*op))
				target.Operation = op.OperationID
				if strings.TrimSpace(firstNonEmpty(target.Source, target.OpenAPI)) == "" {
					setStepAPISourceFromDoc(target, doc)
				}
			}
			selectedStep := target
			if selectedStep == nil && len(session.Intent.Steps) > 0 {
				selectedStep = session.Intent.Steps[0]
			}
			addMappingClassification(session, MappingClassification{
				Slot:                 stepOperationSlot(selectedStep),
				Value:                op.OperationID,
				Source:               mappingSourceUser,
				Confidence:           mappingConfidenceHigh,
				Evidence:             answer,
				Reason:               "User selected the API operation.",
				RequiresConfirmation: false,
			})
		} else if len(session.Intent.Steps) == 0 || !questionTargetsExistingAPIStep(session, plan) {
			stepType := "fnct"
			operation := ""
			if intentAPISourceRef(session.Intent) != "" {
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
		for _, step := range targetStepsForWithPlan(session, plan) {
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

func targetStepsForWithPlan(session *Session, plan QuestionPlan) []*rollout.Step {
	if session == nil {
		return nil
	}
	for _, slot := range plan.Slots {
		name, ok := stepNameFromWithSlot(slot)
		if !ok {
			continue
		}
		for _, step := range session.Intent.Steps {
			if step != nil && firstNonEmpty(step.Name, "step") == name {
				return []*rollout.Step{step}
			}
		}
	}
	return session.Intent.Steps
}

func stepNameFromWithSlot(slot string) (string, bool) {
	slot = strings.TrimSpace(slot)
	if !strings.HasPrefix(slot, "steps.") {
		return "", false
	}
	rest := strings.TrimPrefix(slot, "steps.")
	if idx := strings.Index(rest, ".with"); idx > 0 {
		return rest[:idx], true
	}
	return "", false
}

func questionTargetsExistingAPIStep(session *Session, plan QuestionPlan) bool {
	step := targetStepForPlan(session, plan)
	if step == nil {
		return false
	}
	stepType := strings.ToLower(strings.TrimSpace(step.Type))
	return stepType == "http" || stepType == "openapi" || strings.TrimSpace(step.Provider) != ""
}

func targetStepForPlan(session *Session, plan QuestionPlan) *rollout.Step {
	if session == nil {
		return nil
	}
	for _, slot := range plan.Slots {
		if !strings.HasPrefix(slot, "steps.") || !strings.HasSuffix(slot, ".operation") {
			continue
		}
		name := strings.TrimSuffix(strings.TrimPrefix(slot, "steps."), ".operation")
		for _, step := range session.Intent.Steps {
			if step != nil && firstNonEmpty(step.Name, "step") == name {
				return step
			}
		}
	}
	return nil
}

func matchOperationAnswerForPlan(session *Session, plan QuestionPlan, answer string, docs []APIDocument) (APIDocument, *apitools.OperationSummary) {
	step := targetStepForPlan(session, plan)
	if step == nil {
		return matchOperationAnswer(answer, docs)
	}
	filtered := filterDocsForStep(session, docs, step)
	if len(filtered) == 0 {
		return APIDocument{}, nil
	}
	return matchOperationAnswer(answer, filtered)
}

func filterDocsForStep(session *Session, docs []APIDocument, step *rollout.Step) []APIDocument {
	if step == nil {
		return docs
	}
	docPath := strings.TrimSpace(firstNonEmpty(step.Source, step.OpenAPI))
	if docPath == "" && session != nil {
		docPath = intentAPISourceRef(session.Intent)
	}
	if docPath != "" {
		var filtered []APIDocument
		for _, doc := range docs {
			if doc.RelativePath == docPath {
				filtered = append(filtered, doc)
			}
		}
		if len(filtered) > 0 {
			return filtered
		}
	}
	provider := normalizeToken(firstNonEmpty(step.Provider, step.Name))
	if provider == "" {
		return docs
	}
	var filtered []APIDocument
	for _, doc := range docs {
		if docMatchesProvider(doc, provider) {
			filtered = append(filtered, doc)
		}
	}
	sortAPIDocumentsByPriority(filtered)
	return filtered
}

func sortAPIDocumentsByPriority(docs []APIDocument) {
	sort.SliceStable(docs, func(i, j int) bool {
		if apiDocumentPriority(docs[i]) != apiDocumentPriority(docs[j]) {
			return apiDocumentPriority(docs[i]) < apiDocumentPriority(docs[j])
		}
		return docs[i].RelativePath < docs[j].RelativePath
	})
}

func docMatchesProvider(doc APIDocument, provider string) bool {
	provider = normalizeToken(provider)
	if provider == "" {
		return false
	}
	haystack := tokenSet(doc.RelativePath + " " + doc.Title + " " + doc.Description)
	if haystack[provider] {
		return true
	}
	normalizedPath := normalizeSearchText(doc.RelativePath + doc.Title + doc.Description)
	return strings.Contains(normalizedPath, provider)
}

func deterministicPrefill(session *Session, docs []APIDocument) bool {
	if session == nil {
		return false
	}
	changed := false
	if addDeterministicPreworkSteps(session, docs) {
		changed = true
	}
	walkSteps(session.Intent.Steps, func(step *rollout.Step) {
		if step == nil {
			return
		}
		if strings.TrimSpace(step.Operation) == "" {
			choices := rankedOperationChoicesForStep(*session, docs, step)
			if len(choices) == 1 && strings.TrimSpace(choices[0].Op.OperationID) != "" {
				step.Operation = choices[0].Op.OperationID
				if stepAPISourceRef(*session, step) == "" {
					setStepAPISourceFromDoc(step, choices[0].Doc)
				}
				addMappingClassification(session, MappingClassification{
					Slot:                 "steps." + firstNonEmpty(step.Name, "step") + ".operation",
					Value:                step.Operation,
					Source:               mappingSourceFallbackDefault,
					Confidence:           mappingConfidenceReview,
					Evidence:             operationLabel(choices[0].Op),
					Reason:               "Only one listed operationId is available for this API step.",
					RequiresConfirmation: true,
				})
				changed = true
			}
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

func addDeterministicPreworkSteps(session *Session, docs []APIDocument) bool {
	if session == nil {
		return false
	}
	changed := false
	for _, step := range append([]*rollout.Step(nil), session.Intent.Steps...) {
		if addOpenWeatherMapGeocodePrework(session, docs, step) {
			changed = true
		}
	}
	return changed
}

func addOpenWeatherMapGeocodePrework(session *Session, docs []APIDocument, weatherStep *rollout.Step) bool {
	if session == nil || weatherStep == nil {
		return false
	}
	if strings.TrimSpace(weatherStep.Operation) != "getOpenWeatherMapOneCall3" {
		return false
	}
	weatherOp, ok := operationForStep(*session, docs, weatherStep)
	if !ok {
		return false
	}
	missing := map[string]bool{}
	for _, field := range missingRequiredFields(weatherStep, weatherOp) {
		missing[field] = true
	}
	if !missing["lat"] || !missing["lon"] {
		return false
	}
	if hasDependencyForFields(weatherStep, "lat", "lon") {
		return false
	}
	location := locationLiteralFromWorkflow(*session)
	if location == "" {
		return false
	}
	doc, ok := documentForStep(*session, docs, weatherStep, weatherOp)
	if !ok {
		return false
	}
	geocodeOp, ok := operationByID([]APIDocument{doc}, doc.RelativePath, "geocodeOpenWeatherMapLocationName")
	if !ok {
		return false
	}
	geocodeName := uniqueStepName(session.Intent.Steps, "geocode_openweathermap_location")
	credential := ensureCredentialBinding(session, suggestedCredentialNameForOperation(*session, docs, weatherStep, weatherOp), mappingSourceDeterministic, "OpenWeatherMap geocoding and weather steps require the same symbolic API credential binding.")
	geocodeStep := &rollout.Step{
		Name:      geocodeName,
		Type:      "http",
		Do:        "Resolve " + location + " to OpenWeatherMap coordinates.",
		Provider:  firstNonEmpty(weatherStep.Provider, "openweathermap"),
		OpenAPI:   doc.RelativePath,
		Operation: geocodeOp.OperationID,
		With:      map[string]string{},
	}
	setStepWithIfEmpty(geocodeStep, "q", location)
	addMappingClassification(session, MappingClassification{
		Slot:                 stepWithSlot(geocodeStep, "q"),
		Value:                location,
		Source:               mappingSourceDeterministic,
		Confidence:           mappingConfidenceReview,
		Evidence:             draftSessionDescription(*session),
		Reason:               "The workflow brief names a location and the local OpenWeatherMap overlay includes a geocoding operation.",
		RequiresConfirmation: true,
	})
	addDeterministicCredentialMappings(session, docs, geocodeStep, geocodeOp, credential, "The geocoding prework step uses the same OpenWeatherMap credential binding as the weather step.")
	addDeterministicCredentialMappings(session, docs, weatherStep, weatherOp, credential, "The selected weather operation requires the OpenWeatherMap credential binding.")
	insertStepBefore(session, weatherStep, geocodeStep)
	weatherStep.DependsOn = appendUniqueString(weatherStep.DependsOn, geocodeName)
	weatherStep.Binds = append(weatherStep.Binds, &rollout.StepBind{
		From: geocodeName,
		Fields: map[string]string{
			"lat": "received_body[0].lat",
			"lon": "received_body[0].lon",
		},
	})
	recordPreworkAssumption(session, geocodeStep, weatherStep, location)
	return true
}

func hasDependencyForFields(step *rollout.Step, fields ...string) bool {
	if step == nil {
		return false
	}
	needed := map[string]bool{}
	for _, field := range fields {
		needed[field] = true
	}
	for _, bind := range step.Binds {
		if bind == nil {
			continue
		}
		for field, source := range bind.Fields {
			if needed[field] && strings.TrimSpace(source) != "" {
				delete(needed, field)
			}
		}
	}
	return len(needed) == 0
}

func locationLiteralFromWorkflow(session Session) string {
	description := draftSessionDescription(session)
	if description == "" {
		return ""
	}
	lower := strings.ToLower(description)
	for _, marker := range []string{"weather of ", "weather in ", "weather for "} {
		idx := strings.Index(lower, marker)
		if idx < 0 {
			continue
		}
		start := idx + len(marker)
		rest := description[start:]
		restLower := lower[start:]
		end := len(rest)
		for _, stop := range []string{", and then", " and then", ", then", " then ", ". ", ";"} {
			if stopIdx := strings.Index(restLower, stop); stopIdx >= 0 && stopIdx < end {
				end = stopIdx
			}
		}
		location := strings.Trim(rest[:end], " ,.;")
		location = strings.TrimPrefix(location, "the ")
		if location != "" && !strings.Contains(strings.ToLower(location), "lat") && !strings.Contains(strings.ToLower(location), "lon") {
			return location
		}
	}
	return ""
}

func ensureCredentialBinding(session *Session, credential, source, reason string) string {
	credential = strings.TrimSpace(credential)
	if session == nil || credential == "" {
		return credential
	}
	session.Credentials = dedupeStrings(append(session.Credentials, credential))
	session.CredentialsSet = true
	addMappingClassification(session, MappingClassification{
		Slot:                 "credentials",
		Value:                credential,
		Source:               source,
		Confidence:           mappingConfidenceReview,
		Evidence:             credential,
		Reason:               reason,
		RequiresConfirmation: true,
	})
	return credential
}

func addDeterministicCredentialMappings(session *Session, docs []APIDocument, step *rollout.Step, op *apitools.OperationSummary, credential, reason string) bool {
	if session == nil || step == nil || op == nil || credential == "" {
		return false
	}
	changed := false
	source := "credentials." + credential
	for _, field := range requiredMappingFields(op) {
		if !suggestedCredentialField(field, op) && !apiKeyParameterField(field, op) {
			continue
		}
		if setStepWithIfEmpty(step, field, source) {
			addDeterministicPrefillAssumption(session, step, field, source, "credential binding", reason)
			addMappingClassification(session, MappingClassification{
				Slot:                 stepWithSlot(step, field),
				Value:                source,
				Source:               mappingSourceDeterministic,
				Confidence:           mappingConfidenceReview,
				Evidence:             "credential binding " + source,
				Reason:               reason,
				RequiresConfirmation: true,
			})
			changed = true
		}
	}
	return changed
}

func apiKeyParameterField(field string, op *apitools.OperationSummary) bool {
	if op == nil {
		return false
	}
	for _, security := range op.Security {
		if strings.EqualFold(security.Type, "apiKey") && strings.TrimSpace(security.ParameterName) != "" && field == security.ParameterName {
			return true
		}
	}
	return false
}

func insertStepBefore(session *Session, before, inserted *rollout.Step) {
	if session == nil || before == nil || inserted == nil {
		return
	}
	for i, step := range session.Intent.Steps {
		if step == before {
			next := append([]*rollout.Step{}, session.Intent.Steps[:i]...)
			next = append(next, inserted)
			next = append(next, session.Intent.Steps[i:]...)
			session.Intent.Steps = next
			return
		}
	}
	session.Intent.Steps = append([]*rollout.Step{inserted}, session.Intent.Steps...)
}

func uniqueStepName(steps []*rollout.Step, base string) string {
	base = slugIdent(base)
	if base == "" {
		base = "step"
	}
	used := map[string]bool{}
	walkSteps(steps, func(step *rollout.Step) {
		if step != nil && strings.TrimSpace(step.Name) != "" {
			used[step.Name] = true
		}
	})
	if !used[base] {
		return base
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s_%d", base, i)
		if !used[candidate] {
			return candidate
		}
	}
}

func appendUniqueString(values []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return values
	}
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func recordPreworkAssumption(session *Session, geocodeStep, weatherStep *rollout.Step, location string) {
	if session == nil || geocodeStep == nil || weatherStep == nil {
		return
	}
	session.Assumptions = mergeAssumptions(session.Assumptions, []Assumption{{
		ID:                   "deterministic_prework_" + slugIdent(geocodeStep.Name),
		Slot:                 "steps." + geocodeStep.Name,
		Value:                geocodeStep.Name + " -> " + weatherStep.Name,
		Reason:               "The workflow names a location, while the selected weather operation requires latitude and longitude. A local OpenWeatherMap geocoding operation can produce those values as a legal workflow step.",
		Evidence:             location,
		Risk:                 "review",
		RequiresConfirmation: true,
	}})
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
	if len(missingLocalAPIDocumentRefs(session, docs)) > 0 {
		return true
	}
	if intentAPISourceRef(session.Intent) != "" {
		return false
	}
	hints := CatalogHintsForSession(session)
	if len(catalogProvidersMissingLocalDocs(hints, docs)) > 0 {
		return true
	}
	if len(hints) > 0 && len(docs) > 0 && !apiDocsAccepted(session) && intentAPISourceRef(session.Intent) == "" {
		return true
	}
	if apiDocsAccepted(session) && len(docs) > 0 {
		return false
	}
	if len(docs) == 1 && session.Intent.RequiresOpenAPI() {
		return false
	}
	if session.Intent.RequiresOpenAPI() {
		return true
	}
	if len(docs) == 0 && len(hints) > 0 {
		return true
	}
	for _, step := range session.Intent.Steps {
		if step == nil {
			continue
		}
		stepType := strings.ToLower(strings.TrimSpace(step.Type))
		if (stepType == "http" || stepType == "openapi") && stepAPISourceRef(session, step) == "" {
			return true
		}
	}
	return false
}

func missingAPIDocMessage(session Session, docs []APIDocument) string {
	if missingRefs := missingLocalAPIDocumentRefs(session, docs); len(missingRefs) > 0 {
		return "Local API document path is not available: " + strings.Join(missingRefs, ", ") + ". Add the file under the workflow example before selecting operationIds."
	}
	if len(docs) > 0 {
		if hints := CatalogHintsForSession(session); len(hints) > 0 {
			if missing := catalogProvidersMissingLocalDocs(hints, docs); len(missing) > 0 {
				return "No first-class OpenAPI is available for " + strings.Join(missing, ", ") + "; cannot continue to operation selection until an artifact is generated/provided. Local API documents already available: " + strings.Join(apiDocumentLabels(docs), ", ") + "."
			}
		}
		return "Local API documents are available: " + strings.Join(apiDocumentLabels(docs), ", ") + ". Confirm whether to use them for operationId selection."
	}
	if hints := CatalogHintsForSession(session); len(hints) > 0 {
		available := CatalogProvidersWithMigratableDocs(hints, "")
		missing := CatalogProvidersMissingMigratableDocs(hints, "")
		switch {
		case len(available) > 0 && len(missing) == 0:
			return "First-class API documents were found in ../apitools for " + strings.Join(available, " -> ") + ", but they are not local to this workflow yet."
		case len(available) > 0:
			return "First-class API documents were found in ../apitools for " + strings.Join(available, " -> ") + "; no migratable API document was found for " + strings.Join(missing, ", ") + "."
		default:
			return "No first-class OpenAPI is available for " + strings.Join(CatalogProviderPlan(hints), " -> ") + "; cannot continue to operation selection until an artifact is generated/provided."
		}
	}
	return "Identify the local OpenAPI document for API-backed SaaS steps, or say none only when no API call is needed."
}

func missingAPIDocPrompt(session Session, docs []APIDocument) string {
	if missingRefs := missingLocalAPIDocumentRefs(session, docs); len(missingRefs) > 0 {
		return "The local API document is missing: " + strings.Join(missingRefs, ", ") + ". Generate or provide that artifact, then rerun iCoT."
	}
	if len(docs) > 0 {
		if hints := CatalogHintsForSession(session); len(hints) > 0 {
			if missing := catalogProvidersMissingLocalDocs(hints, docs); len(missing) > 0 {
				return "No first-class OpenAPI is available for " + strings.Join(missing, ", ") + "; cannot continue to operation selection until an artifact is generated/provided."
			}
		}
		return "Local API documents found: " + strings.Join(apiDocumentLabels(docs), ", ") + ". Use these for operation selection?"
	}
	if hints := CatalogHintsForSession(session); len(hints) > 0 {
		available := CatalogProvidersWithMigratableDocs(hints, "")
		missing := CatalogProvidersMissingMigratableDocs(hints, "")
		if len(available) > 0 && len(missing) == 0 {
			return "All first-class API documents were found in ../apitools for " + strings.Join(available, " -> ") + ". Migrate them into this workflow?"
		}
		if len(available) > 0 {
			return "First-class API documents were found in ../apitools for " + strings.Join(available, " -> ") + ", but " + strings.Join(missing, ", ") + " still needs a local OpenAPI file or lowering output. Migrate the available documents now?"
		}
		return "No first-class OpenAPI is available for " + strings.Join(CatalogProviderPlan(hints), " -> ") + "; cannot continue to operation selection until an artifact is generated/provided."
	}
	return "Which local OpenAPI document should this SaaS workflow use?"
}

func missingLocalAPIDocumentRefs(session Session, docs []APIDocument) []string {
	available := map[string]bool{}
	for _, doc := range docs {
		if doc.RelativePath != "" {
			available[doc.RelativePath] = true
		}
	}
	seen := map[string]bool{}
	var missing []string
	add := func(ref string) {
		ref = filepath.ToSlash(strings.TrimSpace(ref))
		if ref == "" || !isLocalAPIDocumentRef(ref) || available[ref] || seen[ref] {
			return
		}
		seen[ref] = true
		missing = append(missing, ref)
	}
	add(session.Intent.Source)
	add(session.Intent.OpenAPI)
	walkSteps(session.Intent.Steps, func(step *rollout.Step) {
		if step != nil {
			add(step.Source)
			add(step.OpenAPI)
		}
	})
	sort.Strings(missing)
	return missing
}

func isLocalAPIDocumentRef(ref string) bool {
	ref = filepath.ToSlash(strings.TrimSpace(ref))
	if ref == "" {
		return false
	}
	if strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") {
		return false
	}
	return strings.HasPrefix(ref, "openapi/") || strings.HasPrefix(ref, "google-discovery/") || strings.HasPrefix(ref, "aws-smithy/") || strings.HasPrefix(ref, "discovery/")
}

func catalogProvidersMissingLocalDocs(hints []CatalogHint, docs []APIDocument) []string {
	var missing []string
	for _, hint := range hints {
		if catalogProviderHasLocalDoc(hint, docs) {
			continue
		}
		if len(CatalogMigrationCandidates([]CatalogHint{hint}, "")) > 0 {
			continue
		}
		missing = append(missing, firstNonEmpty(hint.Provider.DisplayName, hint.Provider.ID))
	}
	return missing
}

func catalogProviderHasLocalDoc(hint CatalogHint, docs []APIDocument) bool {
	providerTerms := []string{hint.Provider.ID, hint.Provider.DisplayName}
	providerTerms = append(providerTerms, hint.Provider.Aliases...)
	for _, doc := range docs {
		haystack := strings.ToLower(doc.RelativePath + " " + doc.Title + " " + doc.Description)
		docTokens := tokenSet(haystack)
		for _, term := range providerTerms {
			if phraseTokensMatch(docTokens, term) {
				return true
			}
		}
		for _, artifact := range hint.SpecArtifacts {
			if strings.TrimSpace(artifact.SpecRef.ID) != "" && strings.Contains(haystack, strings.ToLower(artifact.SpecRef.ID)) {
				return true
			}
		}
	}
	return false
}

func apiDocumentLabels(docs []APIDocument) []string {
	var labels []string
	for _, doc := range docs {
		label := doc.RelativePath
		if title := strings.TrimSpace(doc.Title); title != "" && title != label {
			label += " (" + title + ")"
		}
		labels = append(labels, label)
	}
	return labels
}

func operationForStep(session Session, docs []APIDocument, step *rollout.Step) (*apitools.OperationSummary, bool) {
	if step == nil || strings.TrimSpace(step.Operation) == "" {
		return nil, false
	}
	docPath := stepAPISourceRef(session, step)
	searchDocs := docs
	if strings.TrimSpace(step.Provider) != "" || strings.TrimSpace(firstNonEmpty(step.Source, step.OpenAPI)) != "" {
		searchDocs = filterDocsForStep(&session, docs, step)
		if len(searchDocs) == 0 {
			return nil, false
		}
	}
	if docPath == "" && len(searchDocs) == 1 {
		docPath = searchDocs[0].RelativePath
	}
	if op, ok := operationByID(searchDocs, docPath, step.Operation); ok {
		return op, true
	}
	for _, doc := range searchDocs {
		for i := range doc.Operations {
			if doc.Operations[i].OperationID == step.Operation {
				return &doc.Operations[i], true
			}
		}
	}
	return nil, false
}

func missingRequiredFields(step *rollout.Step, op *apitools.OperationSummary) []string {
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

func requiredMappingFields(op *apitools.OperationSummary) []string {
	if op == nil {
		return nil
	}
	return apitools.RequiredOperationFields(*op)
}

func validateOpenAPIRequestMappings(session Session, step *rollout.Step, op *apitools.OperationSummary, slotPrefix string) []ReadinessIssue {
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

func openAPIRequestFieldTypes(op *apitools.OperationSummary) map[string]openAPIRequestFieldInfo {
	out := map[string]openAPIRequestFieldInfo{}
	if op == nil {
		return out
	}
	for field, info := range apitools.OperationRequestFieldTypes(*op) {
		out[field] = openAPIRequestFieldInfo{Type: info.Type, Body: info.Body}
	}
	for _, field := range apitools.RequiredOperationFields(*op) {
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
		return "yes"
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

func suggestedOperationAnswer(docs []APIDocument) string {
	for _, doc := range docs {
		for _, op := range doc.Operations {
			if op.OperationID != "" {
				return op.OperationID
			}
		}
	}
	return "Describe the action in business terms."
}

func suggestedOperationAnswerForStep(session Session, docs []APIDocument, step *rollout.Step) string {
	choices := rankedOperationChoicesForStep(session, docs, step)
	if len(choices) == 0 {
		return "Describe the action in business terms."
	}
	return choices[0].Op.OperationID
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
	for _, choice := range choices[:limit] {
		label := choice.Op.OperationID
		if desc := firstNonEmpty(choice.Op.Summary, choice.Op.Description); desc != "" {
			label += " (" + truncateForPrompt(desc, 80) + ")"
		}
		if choice.Doc.RelativePath != "" {
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
	if op == nil || len(op.Security) == 0 {
		return "api_token"
	}
	return slugIdent(op.Security[0].Name)
}

func suggestedCredentialNameForOperation(session Session, docs []APIDocument, step *rollout.Step, op *apitools.OperationSummary) string {
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
	if op == nil || len(op.Security) == 0 {
		return false
	}
	for _, security := range op.Security {
		if field == apitools.SecurityCredentialFieldName(security) {
			return true
		}
	}
	return strings.EqualFold(field, "Authorization")
}

func safeLiteralDefault(field string, op *apitools.OperationSummary) (string, bool) {
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
		for _, field := range apitools.RequiredOperationFields(*op) {
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
	for _, security := range op.Security {
		if field == apitools.SecurityCredentialFieldName(security) {
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

func stepFromOperation(op *apitools.OperationSummary) *rollout.Step {
	return &rollout.Step{
		Name:      camelToSnake(firstNonEmpty(op.OperationID, op.Summary, op.Path)),
		Type:      "http",
		Do:        firstNonEmpty(op.Summary, operationLabel(*op)),
		Operation: op.OperationID,
	}
}
