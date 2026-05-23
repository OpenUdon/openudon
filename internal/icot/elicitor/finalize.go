package elicitor

import (
	"strings"

	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

const weatherReportPlaceholderName = "render_weather_report"

func finalizeICoTIntent(session *Session) {
	if session == nil {
		return
	}
	addWeatherReportPlaceholder(session)
	ensureDeliverySinksFollowTerminalProducer(session)
	cleanupTopLevelDependsOn(session.Intent.Steps)
	session.Intent.Steps = topologicalIntentSteps(session.Intent.Steps)
	removeUnusedBodyInput(session)
}

func addWeatherReportPlaceholder(session *Session) bool {
	if session == nil {
		return false
	}
	if !weatherReportEmailWorkflow(*session) {
		return false
	}
	weatherSteps := weatherProducerSteps(session.Intent.Steps)
	if len(weatherSteps) != 1 {
		return false
	}
	gmailSteps := gmailDeliverySteps(session.Intent.Steps)
	if len(gmailSteps) == 0 {
		return false
	}
	weather := weatherSteps[0]
	render := stepByName(session.Intent.Steps, weatherReportPlaceholderName)
	changed := false
	if render == nil {
		render = &rollout.Step{
			Name: weatherReportPlaceholderName,
		}
		insertStepAfter(session, weather, render)
		changed = true
	}
	if render.Type != "fnct" {
		render.Type = "fnct"
		changed = true
	}
	if strings.TrimSpace(render.Do) == "" {
		render.Do = "Render a reviewable local weather report from the weather response before Gmail delivery."
		changed = true
	}
	if strings.Join(render.DependsOn, ",") != weather.Name {
		render.DependsOn = []string{weather.Name}
		changed = true
	}
	if render.With == nil {
		render.With = map[string]string{}
	}
	if render.With["input"] != weather.Name+".received_body" {
		render.With["input"] = weather.Name + ".received_body"
		changed = true
	}
	for _, gmail := range gmailSteps {
		if gmail.With == nil {
			gmail.With = map[string]string{}
		}
		nextDeps := replaceStepDependency(gmail.DependsOn, weather.Name, render.Name)
		if strings.Join(gmail.DependsOn, ",") != strings.Join(nextDeps, ",") {
			gmail.DependsOn = nextDeps
			changed = true
		}
		field := gmailBodyField(gmail)
		if gmail.With[field] != render.Name+".received_body" {
			gmail.With[field] = render.Name + ".received_body"
			changed = true
		}
		if strings.TrimSpace(gmail.With["body"]) != "" && field != "body" {
			delete(gmail.With, "body")
			changed = true
		}
		if strings.TrimSpace(gmail.With["userId"]) == "" {
			gmail.With["userId"] = "me"
			changed = true
		}
	}
	if ensureGmailCredentialBinding(session, gmailSteps) {
		changed = true
	}
	if updateWeatherReportOutputs(session, weather, render, gmailSteps) {
		changed = true
	}
	if changed {
		addDecisionEvidence(session, DecisionEvidence{
			Stage:                decisionStageDraftReview,
			Slot:                 "steps." + render.Name,
			Value:                render.Name + " consumes " + weather.Name + ".received_body",
			Source:               mappingSourceDeterministic,
			Confidence:           mappingConfidenceReview,
			Reason:               "Inserted or reconciled a reviewable local fnct placeholder for the narrow weather report to Gmail workflow pattern.",
			Evidence:             "Weather response content must be formatted before Gmail delivery.",
			RequiresConfirmation: true,
		})
		addMappingClassification(session, MappingClassification{
			Slot:                 "steps." + render.Name,
			Value:                render.Name + " consumes " + weather.Name + ".received_body",
			Source:               mappingSourceDeterministic,
			Confidence:           mappingConfidenceReview,
			Evidence:             "Weather report Gmail delivery requires a local formatter placeholder.",
			Reason:               "The formatter is explicit review evidence and must be implemented before trusted execution.",
			RequiresConfirmation: true,
		})
	}
	return changed
}

func weatherReportEmailWorkflow(session Session) bool {
	text := strings.ToLower(strings.Join([]string{
		session.Project.Goal,
		session.Project.Outputs,
		session.Project.DataFlow,
		session.Project.FunctionContracts,
		draftSessionDescription(session),
	}, " "))
	return strings.Contains(text, "weather") &&
		strings.Contains(text, "report") &&
		(strings.Contains(text, "gmail") || strings.Contains(text, "email") || strings.Contains(text, "mail"))
}

func weatherProducerSteps(steps []*rollout.Step) []*rollout.Step {
	var out []*rollout.Step
	for _, step := range steps {
		if isWeatherProducerStep(step) {
			out = append(out, step)
		}
	}
	return out
}

func isWeatherProducerStep(step *rollout.Step) bool {
	if step == nil {
		return false
	}
	stepType := strings.ToLower(strings.TrimSpace(step.Type))
	if stepType != "http" && stepType != "openapi" {
		return false
	}
	if isGmailDeliveryStep(step) {
		return false
	}
	text := stepSearchText(step)
	if strings.Contains(text, "geocode") || strings.Contains(text, "reverse") || strings.Contains(text, "zip") {
		return false
	}
	return strings.Contains(text, "openweathermap") || strings.Contains(text, "weather")
}

func gmailDeliverySteps(steps []*rollout.Step) []*rollout.Step {
	var out []*rollout.Step
	for _, step := range steps {
		if isGmailDeliveryStep(step) {
			out = append(out, step)
		}
	}
	return out
}

func isGmailDeliveryStep(step *rollout.Step) bool {
	if step == nil {
		return false
	}
	stepType := strings.ToLower(strings.TrimSpace(step.Type))
	if stepType != "http" && stepType != "openapi" {
		return false
	}
	text := stepSearchText(step)
	return strings.Contains(text, "gmail") &&
		(strings.Contains(text, "send") || strings.Contains(text, "message") || strings.Contains(text, "mail"))
}

func gmailBodyField(step *rollout.Step) string {
	if step != nil {
		if _, ok := step.With["raw"]; ok {
			return "raw"
		}
		if _, ok := step.With["body"]; ok {
			return "body"
		}
	}
	return "raw"
}

func updateWeatherReportOutputs(session *Session, weather, render *rollout.Step, gmailSteps []*rollout.Step) bool {
	if session == nil || weather == nil || render == nil {
		return false
	}
	changed := false
	renderSource := render.Name + ".received_body"
	if len(session.Intent.Outputs) == 0 {
		session.Intent.Outputs = []*rollout.Output{{Name: "weather_report", From: renderSource}}
		return true
	}
	gmailNames := map[string]bool{}
	for _, gmail := range gmailSteps {
		if gmail != nil {
			gmailNames[gmail.Name] = true
		}
	}
	for _, output := range session.Intent.Outputs {
		if output == nil {
			continue
		}
		sourceStep := sourceStepName(output.From)
		if sourceStep == render.Name || explicitGmailMetadataOutput(output, gmailNames) {
			continue
		}
		name := strings.ToLower(strings.TrimSpace(output.Name))
		if sourceStep == weather.Name || name == "result" || strings.Contains(name, "report") {
			if output.From != renderSource {
				output.From = renderSource
				changed = true
			}
		}
	}
	return changed
}

func explicitGmailMetadataOutput(output *rollout.Output, gmailNames map[string]bool) bool {
	if output == nil || !gmailNames[sourceStepName(output.From)] {
		return false
	}
	name := strings.ToLower(strings.TrimSpace(output.Name))
	for _, token := range []string{"sent", "send", "message", "gmail", "thread", "id"} {
		if strings.Contains(name, token) {
			return true
		}
	}
	return false
}

func replaceStepDependency(deps []string, oldName, newName string) []string {
	var out []string
	replaced := false
	for _, dep := range deps {
		dep = strings.TrimSpace(dep)
		switch {
		case dep == "":
			continue
		case dep == oldName:
			out = append(out, newName)
			replaced = true
		default:
			out = append(out, dep)
		}
	}
	if !replaced {
		out = append(out, newName)
	}
	return dedupeStrings(out)
}

func ensureGmailCredentialBinding(session *Session, gmailSteps []*rollout.Step) bool {
	if session == nil || len(gmailSteps) == 0 {
		return false
	}
	for _, gmail := range gmailSteps {
		if gmail != nil && strings.TrimSpace(gmail.Operation) == "gmail_users_messages_send" {
			before := strings.Join(session.Credentials, ",")
			ensureCredentialBinding(session, "gmail_oauth_token", mappingSourceDeterministic, "Gmail message send requires the Gmail OAuth credential binding.")
			return strings.Join(session.Credentials, ",") != before
		}
	}
	return false
}

func ensureDeliverySinksFollowTerminalProducer(session *Session) {
	if session == nil {
		return
	}
	for _, sink := range session.Intent.Steps {
		if sink == nil {
			continue
		}
		if !deliverySinkStep(sink, "") {
			continue
		}
		producer := singleTerminalProducerForDelivery(session.Intent.Steps, sink)
		if producer == nil {
			continue
		}
		if sourceReferencesStep(sink, producer.Name) {
			continue
		}
		sink.DependsOn = appendUniqueString(sink.DependsOn, producer.Name)
	}
}

func singleTerminalProducerForDelivery(steps []*rollout.Step, sink *rollout.Step) *rollout.Step {
	consumed := map[string]bool{}
	var candidates []*rollout.Step
	for _, step := range steps {
		if step == nil || step == sink {
			continue
		}
		if canProduceFnctRemediationInput(step, nil) && !deliverySinkStep(step, "") {
			candidates = append(candidates, step)
		}
		for _, dep := range step.DependsOn {
			if strings.TrimSpace(dep) != "" {
				consumed[strings.TrimSpace(dep)] = true
			}
		}
		for _, source := range step.With {
			addSourceStepReference(consumed, source)
		}
		for _, bind := range step.Binds {
			if bind != nil && strings.TrimSpace(bind.From) != "" {
				consumed[strings.TrimSpace(bind.From)] = true
			}
		}
	}
	var terminal []*rollout.Step
	for _, candidate := range candidates {
		if candidate != nil && !consumed[candidate.Name] {
			terminal = append(terminal, candidate)
		}
	}
	if len(terminal) == 1 {
		return terminal[0]
	}
	return nil
}

func cleanupTopLevelDependsOn(steps []*rollout.Step) {
	known := map[string]bool{}
	for _, step := range steps {
		if step != nil && strings.TrimSpace(step.Name) != "" {
			known[step.Name] = true
		}
	}
	for _, step := range steps {
		if step == nil {
			continue
		}
		var deps []string
		for _, dep := range step.DependsOn {
			dep = strings.TrimSpace(dep)
			if dep != "" && dep != step.Name && known[dep] {
				deps = append(deps, dep)
			}
		}
		for _, source := range step.With {
			if dep := sourceStepName(source); dep != "" && dep != step.Name && known[dep] {
				deps = append(deps, dep)
			}
		}
		for _, bind := range step.Binds {
			if bind != nil && strings.TrimSpace(bind.From) != "" && bind.From != step.Name && known[bind.From] {
				deps = append(deps, bind.From)
			}
		}
		step.DependsOn = dedupeStrings(deps)
	}
}

func topologicalIntentSteps(steps []*rollout.Step) []*rollout.Step {
	if len(steps) < 2 {
		return steps
	}
	known := map[string]bool{}
	for _, step := range steps {
		if step != nil && strings.TrimSpace(step.Name) != "" {
			known[step.Name] = true
		}
	}
	done := map[string]bool{}
	used := map[int]bool{}
	out := make([]*rollout.Step, 0, len(steps))
	for len(out) < len(steps) {
		progress := false
		for i, step := range steps {
			if used[i] {
				continue
			}
			if step == nil || depsSatisfied(step.DependsOn, known, done) {
				out = append(out, step)
				used[i] = true
				if step != nil {
					done[step.Name] = true
				}
				progress = true
			}
		}
		if progress {
			continue
		}
		for i, step := range steps {
			if !used[i] {
				out = append(out, step)
				used[i] = true
			}
		}
	}
	return out
}

func depsSatisfied(deps []string, known, done map[string]bool) bool {
	for _, dep := range deps {
		dep = strings.TrimSpace(dep)
		if dep != "" && known[dep] && !done[dep] {
			return false
		}
	}
	return true
}

func removeUnusedBodyInput(session *Session) {
	if session == nil {
		return
	}
	var out []*rollout.Input
	for _, input := range session.Intent.Inputs {
		if input == nil {
			continue
		}
		if input.Name == "body" && !inputReferenced(session, "body") {
			continue
		}
		out = append(out, input)
	}
	session.Intent.Inputs = out
}

func inputReferenced(session *Session, name string) bool {
	if session == nil || strings.TrimSpace(name) == "" {
		return false
	}
	needle := "inputs." + name
	found := false
	walkSteps(session.Intent.Steps, func(step *rollout.Step) {
		if found || step == nil {
			return
		}
		for _, source := range step.With {
			if strings.TrimSpace(source) == needle || strings.HasPrefix(strings.TrimSpace(source), needle+".") {
				found = true
				return
			}
		}
		for _, bind := range step.Binds {
			if bind == nil {
				continue
			}
			for _, source := range bind.Fields {
				if strings.TrimSpace(source) == needle || strings.HasPrefix(strings.TrimSpace(source), needle+".") {
					found = true
					return
				}
			}
		}
	})
	if found {
		return true
	}
	for _, output := range session.Intent.Outputs {
		if output != nil && (strings.TrimSpace(output.From) == needle || strings.HasPrefix(strings.TrimSpace(output.From), needle+".")) {
			return true
		}
	}
	return false
}

func sourceReferencesStep(step *rollout.Step, name string) bool {
	if step == nil || strings.TrimSpace(name) == "" {
		return false
	}
	for _, dep := range step.DependsOn {
		if strings.TrimSpace(dep) == name {
			return true
		}
	}
	for _, source := range step.With {
		if sourceStepName(source) == name {
			return true
		}
	}
	for _, bind := range step.Binds {
		if bind != nil && strings.TrimSpace(bind.From) == name {
			return true
		}
	}
	return false
}

func annotateIntentHCLWithPlaceholderWarnings(intentHCL string, session Session) string {
	step := stepByName(session.Intent.Steps, weatherReportPlaceholderName)
	if step == nil {
		return intentHCL
	}
	lines := strings.Split(intentHCL, "\n")
	anchor := -1
	target := `step "` + weatherReportPlaceholderName + `"`
	for i, line := range lines {
		if strings.Contains(line, target) {
			anchor = i
			break
		}
	}
	if anchor < 0 {
		return intentHCL
	}
	comments := []string{
		"# iCoT review warning (reviewable_fnct_placeholder)",
		"# render_weather_report is a local formatting placeholder, not an API operation.",
		"# Review and implement this fnct contract before trusted execution.",
		"# Decision evidence: weather report content is rendered from " + strings.Join(step.DependsOn, ", ") + ".received_body before Gmail delivery.",
	}
	out := make([]string, 0, len(lines)+len(comments))
	out = append(out, lines[:anchor]...)
	out = append(out, comments...)
	out = append(out, lines[anchor:]...)
	return strings.Join(out, "\n")
}

func stepSearchText(step *rollout.Step) string {
	if step == nil {
		return ""
	}
	return strings.ToLower(strings.Join([]string{
		step.Name,
		step.Do,
		step.Provider,
		step.Source,
		step.OpenAPI,
		step.Operation,
	}, " "))
}
