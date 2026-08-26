package elicitor

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/OpenUdon/apitools"
	"github.com/OpenUdon/openudon/internal/projectwizard"
	"github.com/OpenUdon/openudon/internal/sourcecatalog"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

func deterministicPrefill(session *Session, docs []APIDocument) bool {
	if session == nil {
		return false
	}
	changed := false
	if applyCapabilityGapFallback(session, docs) {
		changed = true
	}
	if addDeterministicPreworkSteps(session, docs) {
		changed = true
	}
	walkSteps(session.Intent.Steps, func(step *rollout.Step) {
		if step == nil {
			return
		}
		if apiBackedStep(step) && strings.TrimSpace(step.Operation) == "" {
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
		for _, field := range missingRequiredFields(*session, step, op) {
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

func apiBackedStep(step *rollout.Step) bool {
	if step == nil {
		return false
	}
	stepType := strings.ToLower(strings.TrimSpace(step.Type))
	return stepType == "http" || stepType == "openapi" || stepType == "browser"
}

func applyCapabilityGapFallback(session *Session, docs []APIDocument) bool {
	if session == nil || !capabilityGapFallbackGoal(*session) || usableOperationCount(docs) > 0 {
		return false
	}
	if len(session.Intent.Steps) == 1 && session.Intent.Steps[0] != nil && session.Intent.Steps[0].Name == "render_capability_gap" {
		return false
	}
	session.Intent.Source = ""
	session.Intent.OpenAPI = ""
	session.Intent.Inputs = mergeInputsByName(session.Intent.Inputs, []*rollout.Input{
		{Name: "provider", Type: "string", Required: true, Description: "Provider or API source with missing capability evidence."},
		{Name: "action", Type: "string", Required: true, Description: "Missing or ambiguous provider action."},
	})
	session.Intent.Steps = []*rollout.Step{{
		Name: "render_capability_gap",
		Type: "fnct",
		Do:   "Render a capability gap report for the missing provider action.",
		With: map[string]string{
			"provider": "inputs.provider",
			"action":   "inputs.action",
		},
	}}
	session.Intent.Outputs = []*rollout.Output{{Name: "gap_report", From: "render_capability_gap.received_body"}}
	session.Credentials = nil
	session.CredentialsSet = true
	session.SideEffectScope = projectwizard.SideEffectReadOnly
	if strings.TrimSpace(session.Safety) == "" && strings.TrimSpace(session.Project.Safety) == "" {
		session.Safety = "Generate a local capability gap report only; do not call external APIs or perform side effects."
		session.SafetySet = true
	}
	addDecisionEvidence(session, DecisionEvidence{
		Stage:                decisionStageCatalogPlan,
		Slot:                 "intent.steps.render_capability_gap",
		Value:                "no-source capability gap fallback",
		Source:               mappingSourceDeterministic,
		Confidence:           mappingConfidenceHigh,
		Reason:               "No usable local API operations are available and the goal explicitly asks to stop and render a missing or ambiguous provider capability gap report.",
		Evidence:             draftSessionDescription(*session),
		RequiresConfirmation: false,
	})
	return true
}

func capabilityGapFallbackGoal(session Session) bool {
	text := strings.ToLower(strings.Join([]string{
		session.Project.Goal,
		session.Project.Outputs,
		session.Project.DataFlow,
		session.Project.FunctionContracts,
		session.Project.OpenAPI,
		session.Project.Fallback,
		session.Fallback,
		draftSessionDescription(session),
	}, " "))
	if !(strings.Contains(text, "gap report") || strings.Contains(text, "capability gap") || (strings.Contains(text, "render") && strings.Contains(text, "gap")) || (strings.Contains(text, "report") && strings.Contains(text, "gap"))) {
		return false
	}
	if !(strings.Contains(text, "stop") || strings.Contains(text, "missing") || strings.Contains(text, "ambiguous") || strings.Contains(text, "no usable") || strings.Contains(text, "lacks usable") || strings.Contains(text, "without openapi") || strings.Contains(text, "without api") || strings.Contains(text, "no api")) {
		return false
	}
	return strings.Contains(text, "provider") || strings.Contains(text, "api") || strings.Contains(text, "source") || strings.Contains(text, "openapi") || strings.Contains(text, "operation") || strings.Contains(text, "action")
}

func usableOperationCount(docs []APIDocument) int {
	total := 0
	for _, doc := range docs {
		for _, op := range doc.Operations {
			if strings.TrimSpace(op.OperationID) != "" {
				total++
			}
		}
	}
	return total
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
	switch strings.TrimSpace(weatherStep.Operation) {
	case "getOpenWeatherMapOneCall3", "getOpenWeatherMapCurrentWeather":
	default:
		return false
	}
	weatherOp, ok := operationForStep(*session, docs, weatherStep)
	if !ok {
		return false
	}
	missing := map[string]bool{}
	for _, field := range missingRequiredFields(*session, weatherStep, weatherOp) {
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
		for _, stop := range []string{", and then", " and then", ", and send", " and send", ", then", " then ", ". ", ";"} {
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
	for _, field := range requiredMappingFieldsForStep(*session, step, op) {
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
	for _, security := range allSecurityRequirements(op) {
		if strings.EqualFold(security.Type, "apiKey") && strings.TrimSpace(security.ParameterName) != "" && field == security.ParameterName {
			return true
		}
	}
	return false
}

func skippableSecurityAliasField(field string, op *apitools.OperationSummary) bool {
	if op == nil {
		return false
	}
	for _, security := range allSecurityRequirements(op) {
		parameterName := strings.TrimSpace(security.ParameterName)
		if parameterName == "" {
			continue
		}
		if field == apitools.SecurityCredentialFieldName(security) && field != parameterName {
			for _, required := range apitools.RequiredRequestFields(*op) {
				if required == parameterName {
					return true
				}
			}
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
	case "http", "openapi", "browser":
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
		if (stepType == "http" || stepType == "openapi" || stepType == "browser") && stepAPISourceRef(session, step) == "" {
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
		return "Validated source documents are available: " + strings.Join(apiDocumentLabels(docs), ", ") + ". API-family sources are preferred when they cover the active capability; choose a browser profile only for an uncovered UI-only capability or an explicit reviewed browser route."
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
		return "Validated source documents found: " + strings.Join(apiDocumentLabels(docs), ", ") + ". Choose the source for operation/action selection; API-family sources are preferred when adequate."
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
	for _, directory := range sourcecatalog.All() {
		if strings.HasPrefix(ref, directory+"/") {
			return true
		}
	}
	return false
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
	if step == nil {
		return nil, false
	}
	operationID := strings.TrimSpace(step.Operation)
	if strings.EqualFold(strings.TrimSpace(step.Type), "browser_registration") {
		operationID = strings.TrimSpace(step.RegistrationFlow)
	}
	if operationID == "" {
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
	if op, ok := operationByID(searchDocs, docPath, operationID); ok {
		return op, true
	}
	for _, doc := range searchDocs {
		for i := range doc.Operations {
			if doc.Operations[i].OperationID == operationID {
				return &doc.Operations[i], true
			}
		}
	}
	return nil, false
}

func missingRequiredFields(session Session, step *rollout.Step, op *apitools.OperationSummary) []string {
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
	for _, field := range requiredMappingFieldsForStep(session, step, op) {
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
	return apitools.RequiredRequestFields(*op)
}

func requiredMappingFieldsForStep(session Session, step *rollout.Step, op *apitools.OperationSummary) []string {
	out := requiredMappingFields(op)
	securityFields, selected := selectedSecurityCredentialFields(session, step, op)
	if !selected {
		return out
	}
	for _, field := range securityFields {
		if skippableSecurityAliasField(field, op) {
			continue
		}
		out = append(out, field)
	}
	return dedupeStrings(out)
}

func validateOpenAPIRequestMappings(session Session, step *rollout.Step, op *apitools.OperationSummary, slotPrefix string) []ReadinessIssue {
	if step == nil || op == nil {
		return nil
	}
	fields := openAPIRequestFieldTypes(session, step, op)
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
		if strings.EqualFold(strings.TrimSpace(op.Provenance), "asyncapi") {
			return
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
