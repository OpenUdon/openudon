package elicitor

import (
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/OpenUdon/apitools"
	"github.com/OpenUdon/openudon/internal/projectwizard"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

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
			if isBrowserDocument(doc) {
				session.BrowserRoute = "browser"
			} else {
				session.BrowserRoute = "api"
			}
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
	case strings.Contains(slotText, "security_alternative"):
		target := targetStepForPlan(session, plan)
		if target == nil {
			return
		}
		op, ok := operationForStep(*session, docs, target)
		if !ok {
			return
		}
		selectSecurityAlternative(session, target, op, answer)
	case strings.Contains(slotText, "authentication_flow"):
		doc, operation := selectBrowserAuthenticationFlow(answer, docs)
		if operation == nil {
			return
		}
		target := targetStepForPlan(session, plan)
		if target != nil && strings.EqualFold(strings.TrimSpace(target.Type), "browser_authentication") {
			target.Source = doc.RelativePath
			target.OpenAPI = ""
			target.AuthenticationFlow = operation.OperationID
		} else {
			insertBrowserAuthenticationStep(session, target, doc, operation)
		}
	case strings.Contains(slotText, "credential_bindings"):
		bindings := parseAssignments(answer)
		if target := targetStepForPlan(session, plan); target != nil && strings.EqualFold(strings.TrimSpace(target.Type), "browser_authentication") {
			target.CredentialBindings = bindings
		}
	case strings.Contains(slotText, "authentication_approval"):
		if !strings.HasPrefix(strings.ToLower(answer), "approve ") {
			return
		}
		name := strings.TrimSpace(answer[len("approve "):])
		if target := targetStepForPlan(session, plan); target != nil && strings.EqualFold(strings.TrimSpace(target.Type), "browser_authentication") && name == target.Name {
			session.BrowserAuthenticationApprovals = dedupeStrings(append(session.BrowserAuthenticationApprovals, target.Name))
		}
	case strings.HasSuffix(slotText, ".timeout") && strings.Contains(slotText, "steps."):
		seconds, err := strconv.ParseFloat(answer, 64)
		if err != nil || seconds <= 0 || seconds > 600 {
			return
		}
		if target := targetStepForPlan(session, plan); target != nil && strings.EqualFold(strings.TrimSpace(target.Type), "browser_authentication") {
			target.Timeout = &seconds
		}
	case strings.Contains(slotText, "browser_session"):
		if target := targetStepForPlan(session, plan); target != nil && strings.EqualFold(strings.TrimSpace(target.Type), "browser_authentication") {
			if browserBindingNamePattern.MatchString(answer) {
				target.BrowserSession = answer
				for _, candidate := range session.Intent.Steps {
					if candidate != nil && strings.EqualFold(strings.TrimSpace(candidate.Type), "browser") && strings.TrimSpace(candidate.BrowserSession) == "" {
						candidate.BrowserSession = answer
					}
				}
			}
			return
		}
		posture := strings.ToLower(strings.TrimSpace(answer))
		switch posture {
		case "none", "opaque-runtime-binding-required":
			session.BrowserSession = posture
		default:
			return
		}
	case strings.Contains(slotText, "browser_approval"):
		value := strings.TrimSpace(answer)
		if !strings.HasPrefix(strings.ToLower(value), "approve ") {
			return
		}
		name := strings.TrimSpace(value[len("approve "):])
		if step := targetStepForPlan(session, plan); step != nil && name == step.Name {
			session.BrowserApprovals = dedupeStrings(append(session.BrowserApprovals, step.Name))
		}
	case strings.Contains(slotText, "operation") || strings.Contains(slotText, "intent.steps"):
		if doc, op := matchOperationAnswerForPlan(session, plan, answer, docs); op != nil {
			if intentAPISourceRef(session.Intent) == "" {
				setIntentAPISourceFromDoc(session, doc)
			}
			if isBrowserDocument(doc) {
				session.BrowserRoute = "browser"
				if isBrowserAuthenticationOperationSummary(op) {
					session.BrowserSession = "none"
				}
			}
			target := targetStepForPlan(session, plan)
			if len(session.Intent.Steps) == 0 {
				step := stepFromOperation(doc, op)
				setStepAPISourceFromDoc(step, doc)
				session.Intent.Steps = []*rollout.Step{step}
			} else {
				if target == nil {
					target = session.Intent.Steps[0]
				}
				if isBrowserAuthenticationOperationSummary(op) {
					target.Type = "browser_authentication"
					target.AuthenticationFlow = op.OperationID
					target.Operation = ""
					session.BrowserRoute = "browser"
					session.BrowserSession = "none"
				} else if isBrowserDocument(doc) {
					target.Type = "browser"
					session.BrowserRoute = "browser"
				} else {
					target.Type = firstNonEmpty(target.Type, "http")
					session.BrowserRoute = "api"
				}
				target.Do = firstNonEmpty(target.Do, op.Summary, operationLabel(*op))
				if !isBrowserAuthenticationOperationSummary(op) {
					target.Operation = op.OperationID
				}
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
			addDecisionEvidence(session, DecisionEvidence{
				Stage:                decisionStageSideEffect,
				Slot:                 "side_effect_scope",
				Value:                scope,
				Source:               mappingSourceUser,
				Confidence:           mappingConfidenceHigh,
				Reason:               "User confirmed the workflow side-effect boundary.",
				Evidence:             answer,
				RequiresConfirmation: false,
			})
		}
		session.Safety = answer
		session.SafetySet = true
	default:
		if len(plan.Slots) == 1 {
			addDecisionEvidence(session, DecisionEvidence{
				Stage:                decisionStageForSlot(plan.Slots[0]),
				Slot:                 plan.Slots[0],
				Value:                answer,
				Source:               mappingSourceUser,
				Confidence:           mappingConfidenceHigh,
				Reason:               "User confirmed this authoring decision.",
				Evidence:             answer,
				RequiresConfirmation: false,
			})
		}
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
	return stepType == "http" || stepType == "openapi" || stepType == "browser" || strings.TrimSpace(step.Provider) != ""
}

func targetStepForPlan(session *Session, plan QuestionPlan) *rollout.Step {
	if session == nil {
		return nil
	}
	for _, slot := range plan.Slots {
		if !strings.HasPrefix(slot, "steps.") {
			continue
		}
		rest := strings.TrimPrefix(slot, "steps.")
		separator := strings.Index(rest, ".")
		if separator <= 0 {
			continue
		}
		name := rest[:separator]
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
