package elicitor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/OpenUdon/apitools"
	"github.com/OpenUdon/openudon/internal/authoring"
	"github.com/OpenUdon/openudon/internal/projectwizard"
	"github.com/OpenUdon/openudon/internal/workflowintent"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

type Options struct {
	ExampleDir         string
	NoLLM              bool
	Extractor          Extractor
	DraftPath          string
	TranscriptPath     string
	DisableAIDraft     bool
	VerifyOnly         bool
	DefaultMode        authoring.PromptDefaultMode
	ReviewRepair       bool
	CatalogHintOptions CatalogHintOptions
}

type Artifacts struct {
	ProjectMD string
	IntentHCL string
	Session   Session
}

func Run(ctx context.Context, in io.Reader, out io.Writer, seed Session, opts Options) (Artifacts, error) {
	if opts.VerifyOnly {
		opts.DisableAIDraft = true
	}
	return runProgressive(ctx, in, out, seed, opts)
}

func answerFinalBlockingQuestion(out io.Writer, p *prompter, session *Session, docs []APIDocument, draftPath string) (bool, error) {
	issues := CheckReadiness(*session, docs)
	blocking := firstFinalRepairIssue(issues)
	if blocking.Code == "" {
		return false, nil
	}
	if blocking.Code == "missing_api_doc" {
		if len(docs) > 0 && len(missingLocalAPIDocumentRefs(*session, docs)) == 0 {
			return false, nil
		}
		fmt.Fprintf(out, "Intent is incomplete: %s\n", blocking.Message)
		return true, errors.New(blocking.Message)
	}
	plan := PlanNextQuestion(*session, docs, issues)
	var answer string
	var err error
	if plan.ForceAsk {
		answer, err = p.askDefaultForced(plan.Prompt, plan.SuggestedAnswer)
	} else {
		answer, err = p.askDefault(plan.Prompt, plan.SuggestedAnswer)
	}
	if err != nil {
		return true, err
	}
	applyProgressiveAnswer(session, plan, answer, docs)
	session.Normalize()
	if err := autosave(draftPath, *session); err != nil {
		return true, err
	}
	return true, nil
}

func firstFinalRepairIssue(issues []ReadinessIssue) ReadinessIssue {
	for _, issue := range issues {
		if issue.Severity != readinessBlocking {
			continue
		}
		switch issue.Code {
		case "missing_api_doc", "missing_operation", readinessUnconfirmedSideEffectCommitment:
			return issue
		}
	}
	return ReadinessIssue{}
}

var ErrCanceled = errors.New("authoring canceled")

func RenderArtifacts(session Session) (Artifacts, error) {
	session.Normalize()
	finalizeICoTIntent(&session)
	session.Normalize()
	if err := session.Validate(); err != nil {
		return Artifacts{}, err
	}
	intentHCL, err := workflowintent.RenderHCL(context.Background(), &session.Intent)
	if err != nil {
		return Artifacts{}, err
	}
	intentHCL = annotateIntentHCLWithPlaceholderWarnings(intentHCL, session)
	if _, err := rollout.ParseIntent([]byte(intentHCL), "intent.hcl"); err != nil {
		return Artifacts{}, err
	}
	return Artifacts{
		ProjectMD: projectwizard.Render(session.Project),
		IntentHCL: intentHCL,
		Session:   session,
	}, nil
}

type prompter struct {
	*authoring.PromptSession
	out io.Writer
}

func (p *prompter) askDefault(label, current string) (string, error) {
	return p.AskDefault(label, current)
}

func (p *prompter) askDefaultForced(label, current string) (string, error) {
	return p.AskDefaultForced(label, current)
}

func (p *prompter) askOptionalDefault(label, current string) (string, error) {
	return p.AskOptionalDefault(label, current)
}

func (p *prompter) askSideEffectScope(current string) (string, error) {
	current = projectwizard.NormalizeSideEffectScope(current)
	if current == "" {
		current = projectwizard.SideEffectAfterApproval
	}
	for {
		value, err := p.askDefault("Side-effect scope (read-only/sandbox-only/after-approval)", current)
		if err != nil {
			return "", err
		}
		value = projectwizard.NormalizeSideEffectScope(value)
		if value != "" {
			return value, nil
		}
		fmt.Fprintln(p.out, "Use read-only, sandbox-only, or after-approval.")
	}
}

func (p *prompter) chooseDocument(label string, docs []APIDocument, current string) (APIDocument, error) {
	if len(docs) == 0 {
		return APIDocument{}, errors.New("no OpenAPI documents available")
	}
	fmt.Fprintf(p.out, "%s:\n", label)
	defaultIndex := 0
	if strings.TrimSpace(current) == "" {
		for i, doc := range docs {
			if isAdvisoryAPIDocument(doc) {
				defaultIndex = i
				break
			}
		}
	}
	for i, doc := range docs {
		if doc.RelativePath == current {
			defaultIndex = i
		}
		title := firstNonEmpty(doc.Title, doc.RelativePath)
		fmt.Fprintf(p.out, "  %d. %s (%s)\n", i+1, title, doc.RelativePath)
	}
	for {
		answer, err := p.askDefault("Choose document number", strconv.Itoa(defaultIndex+1))
		if err != nil {
			return APIDocument{}, err
		}
		index, err := strconv.Atoi(strings.TrimSpace(answer))
		if err == nil && index >= 1 && index <= len(docs) {
			return docs[index-1], nil
		}
		for _, doc := range docs {
			if strings.TrimSpace(answer) == doc.RelativePath {
				return doc, nil
			}
		}
		fmt.Fprintln(p.out, "Choose one of the listed document numbers.")
	}
}

func (p *prompter) chooseOperation(doc APIDocument, current string, step *rollout.Step) (*apitools.OperationSummary, error) {
	if len(doc.Operations) == 0 {
		return nil, fmt.Errorf("%s has no operations", doc.RelativePath)
	}
	fmt.Fprintf(p.out, "Operations in %s:\n", doc.RelativePath)
	defaultIndex := defaultOperationIndex(doc, current, step)
	for i, op := range doc.Operations {
		label := operationLabel(op)
		if desc := firstNonEmpty(op.Summary, op.Description); desc != "" && !strings.Contains(label, desc) {
			label += " - " + truncateForPrompt(desc, 120)
		}
		fmt.Fprintf(p.out, "  %d. %s\n", i+1, label)
	}
	for {
		answer, err := p.askDefault("Choose operation number", strconv.Itoa(defaultIndex+1))
		if err != nil {
			return nil, err
		}
		index, err := strconv.Atoi(strings.TrimSpace(answer))
		if err == nil && index >= 1 && index <= len(doc.Operations) {
			return &doc.Operations[index-1], nil
		}
		for i := range doc.Operations {
			if strings.TrimSpace(answer) == doc.Operations[i].OperationID {
				return &doc.Operations[i], nil
			}
		}
		fmt.Fprintln(p.out, "Choose one of the listed operation numbers.")
	}
}

func defaultOperationIndex(doc APIDocument, current string, step *rollout.Step) int {
	current = strings.TrimSpace(current)
	for i, op := range doc.Operations {
		if op.OperationID == current {
			return i
		}
	}
	if step == nil {
		return 0
	}
	query := rankingTokenWeights(strings.Join([]string{step.Name, step.Do, step.Provider, step.OpenAPI}, " "))
	bestIndex := 0
	bestScore := -1
	for i, op := range doc.Operations {
		score := operationRankScore(query, doc, op, false)
		if score > bestScore {
			bestScore = score
			bestIndex = i
		}
	}
	return bestIndex
}

func (p *prompter) collectSteps(usesAPI bool, defaultOpenAPI string, docs []APIDocument, inputs []*rollout.Input, currentSteps []*rollout.Step) ([]*rollout.Step, error) {
	var steps []*rollout.Step
	for {
		defaultName := ""
		var current *rollout.Step
		if len(currentSteps) > len(steps) {
			current = currentSteps[len(steps)]
		}
		if current != nil {
			defaultName = current.Name
		}
		if len(steps) == 0 {
			defaultName = firstNonEmpty(defaultName, "run_workflow")
		}
		name, err := p.askOptionalDefault("Step name (blank when done)", defaultName)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(name) == "" {
			if len(steps) == 0 {
				fmt.Fprintln(p.out, "At least one step is required.")
				continue
			}
			break
		}
		stepTypeDefault := "fnct"
		if usesAPI {
			stepTypeDefault = "http"
		}
		if current != nil && strings.TrimSpace(current.Type) != "" {
			stepTypeDefault = current.Type
		}
		stepType, err := p.askDefault("Step type (http/openapi/fnct/cmd/ssh)", stepTypeDefault)
		if err != nil {
			return nil, err
		}
		step := &rollout.Step{Name: slugIdent(name), Type: strings.ToLower(strings.TrimSpace(stepType))}
		if current != nil {
			step.Provider = current.Provider
			step.OpenAPI = current.OpenAPI
			step.Operation = current.Operation
			step.Timeout = current.Timeout
		}
		step.Do, err = p.askDefault("Step action", firstNonEmpty(step.Do, currentStepAction(current), humanTitle(step.Name)))
		if err != nil {
			return nil, err
		}
		step.Timeout, err = p.askOptionalSeconds("Step timeout seconds (blank for none)", step.Timeout)
		if err != nil {
			return nil, err
		}
		if step.Type == "http" || step.Type == "openapi" {
			docPath := defaultOpenAPI
			if current != nil {
				docPath = firstNonEmpty(current.OpenAPI, docPath)
			}
			var doc APIDocument
			candidateDocs := docs
			if current != nil && strings.TrimSpace(firstNonEmpty(current.Provider, current.OpenAPI)) != "" {
				currentForFilter := *current
				currentForFilter.OpenAPI = firstNonEmpty(currentForFilter.OpenAPI, defaultOpenAPI)
				filtered := filterDocsForStep(nil, docs, &currentForFilter)
				if len(filtered) > 0 {
					candidateDocs = filtered
				} else if strings.TrimSpace(currentForFilter.OpenAPI) != "" {
					candidateDocs = nil
				}
			}
			if len(candidateDocs) > 0 {
				doc, err = p.chooseDocument("OpenAPI document for step", candidateDocs, docPath)
				if err != nil {
					return nil, err
				}
				op, err := p.chooseOperation(doc, step.Operation, step)
				if err != nil {
					return nil, err
				}
				step.Operation = op.OperationID
				op, ok := operationByID([]APIDocument{doc}, doc.RelativePath, step.Operation)
				if !ok {
					return nil, fmt.Errorf("operationId %s is not available in %s", step.Operation, doc.RelativePath)
				}
				docPath = doc.RelativePath
				if docPath != defaultOpenAPI {
					step.OpenAPI = docPath
				}
				fields, err := p.stepFields(apitools.RequiredOperationFields(*op))
				if err != nil {
					return nil, err
				}
				step.With, step.Binds, step.DependsOn, err = p.collectFields(fields, inputs, steps)
				if err != nil {
					return nil, err
				}
			} else {
				step.OpenAPI = docPath
				for {
					step.Operation, err = p.askDefault("Operation ID", step.Operation)
					if err != nil {
						return nil, err
					}
					if strings.TrimSpace(step.Operation) != "" {
						break
					}
					fmt.Fprintln(p.out, "Operation ID is required for API steps.")
				}
				fields, err := p.askOptionalDefault("Required request fields (comma-separated; blank for none)", "")
				if err != nil {
					return nil, err
				}
				selectedFields, err := p.stepFields(splitList(fields))
				if err != nil {
					return nil, err
				}
				step.With, step.Binds, step.DependsOn, err = p.collectFields(selectedFields, inputs, steps)
				if err != nil {
					return nil, err
				}
			}
		} else {
			fields, err := p.askOptionalDefault("Step input fields (comma-separated; blank for none)", inputNames(inputs))
			if err != nil {
				return nil, err
			}
			step.With, step.Binds, step.DependsOn, err = p.collectFields(splitList(fields), inputs, steps)
			if err != nil {
				return nil, err
			}
		}
		steps = append(steps, step)
	}
	return steps, nil
}

func currentStepAction(step *rollout.Step) string {
	if step == nil {
		return ""
	}
	return step.Do
}

func (p *prompter) askOptionalSeconds(label string, current *float64) (*float64, error) {
	defaultValue := ""
	if current != nil {
		defaultValue = strconv.FormatFloat(*current, 'f', -1, 64)
	}
	value, err := p.askOptionalDefault(label, defaultValue)
	if err != nil {
		return nil, err
	}
	value = strings.TrimSpace(value)
	if value == "" || strings.EqualFold(value, "none") || strings.EqualFold(value, "clear") {
		return nil, nil
	}
	seconds, err := strconv.ParseFloat(value, 64)
	if err != nil || seconds <= 0 {
		return nil, fmt.Errorf("%s must be a positive number of seconds", label)
	}
	return &seconds, nil
}

func (p *prompter) stepFields(required []string) ([]string, error) {
	extra, err := p.askOptionalDefault("Additional step fields (comma-separated; blank for none)", "")
	if err != nil {
		return nil, err
	}
	return dedupeStrings(append(required, splitList(extra)...)), nil
}

func (p *prompter) collectFields(fields []string, inputs []*rollout.Input, prior []*rollout.Step) (map[string]string, []*rollout.StepBind, []string, error) {
	with := map[string]string{}
	bindFields := map[string]map[string]string{}
	deps := []string{}
	for _, raw := range fields {
		field := normalizeRequestPromptField(raw)
		if field == "" {
			continue
		}
		defaultSource := defaultFieldSource(field, inputs)
		var source string
		for {
			var err error
			source, err = p.askDefault(fmt.Sprintf("Value for `%s` (runtime input, literal, credential binding, or prior step output)", field), defaultSource)
			if err != nil {
				return nil, nil, nil, err
			}
			source = strings.TrimSpace(source)
			if source != "" {
				break
			}
			fmt.Fprintln(p.out, "A value source is required for this field.")
		}
		stepName, path, ok := parsePriorStepSource(source, prior)
		if ok {
			if bindFields[stepName] == nil {
				bindFields[stepName] = map[string]string{}
			}
			bindFields[stepName][field] = path
			deps = append(deps, stepName)
			continue
		}
		with[field] = source
	}
	if len(with) == 0 {
		with = nil
	}
	var binds []*rollout.StepBind
	for from, fields := range bindFields {
		binds = append(binds, &rollout.StepBind{From: from, Fields: fields})
	}
	return with, binds, dedupeStrings(deps), nil
}

func editSlot(p *prompter, session *Session, slot string, docs []APIDocument) error {
	slot = strings.ToLower(strings.TrimSpace(slot))
	switch slot {
	case "workflow", "name", "goal":
		if session.Intent.Workflow == nil {
			session.Intent.Workflow = &rollout.WorkflowMeta{}
		}
		name, err := p.askDefault("Workflow name", session.Intent.Workflow.Name)
		if err != nil {
			return err
		}
		goal, err := p.askDefault("Workflow goal", session.Intent.Workflow.Description)
		if err != nil {
			return err
		}
		session.Intent.Workflow.Name = slug(name)
		session.Intent.Workflow.Description = goal
	case "inputs":
		value, err := p.askDefault("Runtime inputs (name:type, comma-separated; blank for none)", inputsText(session.Intent.Inputs))
		if err != nil {
			return err
		}
		session.Intent.Inputs = parseInputs(value)
	case "steps":
		steps, err := p.collectSteps(session.Intent.RequiresOpenAPI(), session.Intent.OpenAPI, docs, session.Intent.Inputs, session.Intent.Steps)
		if err != nil {
			return err
		}
		session.Intent.Steps = steps
	case "outputs":
		last := lastStepName(session.Intent.Steps)
		name, err := p.askDefault("Output name", "result")
		if err != nil {
			return err
		}
		source, err := p.askDefault("Output source", defaultOutputSource(last))
		if err != nil {
			return err
		}
		session.Intent.Outputs = []*rollout.Output{{Name: slugIdent(name), From: source}}
	case "credentials":
		value, err := p.askDefault("Credential binding names only", strings.Join(session.Credentials, ", "))
		if err != nil {
			return err
		}
		session.Credentials = credentialBindings(value)
		session.CredentialsSet = true
	case "side-effect", "side-effects", "scope", "side-effect-scope":
		value, err := p.askSideEffectScope(session.SideEffectScope)
		if err != nil {
			return err
		}
		session.SideEffectScope = value
	case "safety":
		value, err := p.askDefault("Safety and approval notes", session.Safety)
		if err != nil {
			return err
		}
		session.Safety = clearablePolicyAnswer(value)
		session.SafetySet = true
	case "fallback":
		value, err := p.askDefault("Fallback behavior", session.Fallback)
		if err != nil {
			return err
		}
		session.Fallback = clearablePolicyAnswer(value)
		session.FallbackSet = true
	case "openapi", "api":
		if len(docs) > 0 {
			doc, err := p.chooseDocument("OpenAPI document", docs, session.Intent.OpenAPI)
			if err != nil {
				return err
			}
			session.Intent.OpenAPI = doc.RelativePath
		} else {
			value, err := p.askDefault("OpenAPI document path or URL", session.Intent.OpenAPI)
			if err != nil {
				return err
			}
			session.Intent.OpenAPI = value
		}
	default:
		fmt.Fprintf(p.out, "Unknown slot %q. Use workflow, openapi, inputs, steps, outputs, side-effect-scope, credentials, safety, or fallback.\n", slot)
	}
	session.Normalize()
	return nil
}

func printSummary(out io.Writer, session Session) {
	session.Normalize()
	fmt.Fprintln(out, "\nSummary:")
	if session.Intent.Workflow != nil {
		fmt.Fprintf(out, "- Workflow: %s - %s\n", session.Intent.Workflow.Name, session.Intent.Workflow.Description)
	}
	if session.Intent.OpenAPI != "" {
		fmt.Fprintf(out, "- OpenAPI: %s\n", session.Intent.OpenAPI)
	} else if hints := CatalogHintsForSession(session); len(hints) > 0 {
		fmt.Fprintf(out, "- API documents: not local yet; catalog providers matched %s\n", strings.Join(CatalogProviderPlan(hints), " -> "))
	} else {
		fmt.Fprintln(out, "- OpenAPI: none required")
	}
	if len(session.Intent.Inputs) > 0 {
		fmt.Fprintf(out, "- Inputs: %s\n", inputsText(session.Intent.Inputs))
	}
	if len(session.Intent.Steps) > 0 {
		var names []string
		for _, step := range session.Intent.Steps {
			if step != nil {
				names = append(names, step.Name+"("+step.Type+")")
			}
		}
		fmt.Fprintf(out, "- Steps: %s\n", strings.Join(names, ", "))
	}
	if len(session.Intent.Outputs) > 0 {
		fmt.Fprintf(out, "- Outputs: %s\n", outputsText(session.Intent.Outputs))
	}
	if len(session.Credentials) > 0 {
		fmt.Fprintf(out, "- Credential bindings: %s\n", strings.Join(session.Credentials, ", "))
	}
	if session.SideEffectScope != "" {
		fmt.Fprintf(out, "- Side-effect scope: %s\n", session.SideEffectScope)
	}
	fmt.Fprintln(out)
}

func printAssumptions(out io.Writer, assumptions []Assumption) {
	if len(assumptions) == 0 {
		return
	}
	fmt.Fprintln(out, "Assumptions to confirm:")
	for _, assumption := range assumptions {
		id := firstNonEmpty(assumption.ID, "assumption")
		slot := firstNonEmpty(assumption.Slot, "intent")
		value := firstNonEmpty(assumption.Value, assumption.Reason, "inferred value")
		risk := strings.TrimSpace(assumption.Risk)
		if risk != "" {
			fmt.Fprintf(out, "- %s [%s]: %s (risk: %s)\n", id, slot, value, risk)
		} else {
			fmt.Fprintf(out, "- %s [%s]: %s\n", id, slot, value)
		}
	}
	fmt.Fprintln(out, "Saving confirms these assumptions.")
	fmt.Fprintln(out)
}

func printAssumptionExplanation(out io.Writer, session Session, id string) {
	id = strings.TrimSpace(id)
	if id == "" {
		fmt.Fprintln(out, "Use explain <assumption-id>.")
		return
	}
	for _, assumption := range session.Assumptions {
		if strings.EqualFold(strings.TrimSpace(assumption.ID), id) {
			fmt.Fprintf(out, "%s\n", assumption.ID)
			if assumption.Slot != "" {
				fmt.Fprintf(out, "- Slot: %s\n", assumption.Slot)
			}
			if assumption.Value != "" {
				fmt.Fprintf(out, "- Value: %s\n", assumption.Value)
			}
			if assumption.Reason != "" {
				fmt.Fprintf(out, "- Reason: %s\n", assumption.Reason)
			}
			if assumption.Evidence != "" {
				fmt.Fprintf(out, "- Evidence: %s\n", assumption.Evidence)
			}
			if assumption.Risk != "" {
				fmt.Fprintf(out, "- Risk: %s\n", assumption.Risk)
			}
			printClassificationExplanation(out, session.Classifications, assumption.Slot)
			return
		}
	}
	fmt.Fprintf(out, "No assumption found for %q.\n", id)
}

func printClassificationExplanation(out io.Writer, classifications []MappingClassification, slot string) {
	slot = strings.TrimSpace(slot)
	if slot == "" {
		return
	}
	var matches []MappingClassification
	for _, classification := range normalizeMappingClassifications(classifications) {
		if classification.Slot == slot {
			matches = append(matches, classification)
		}
	}
	if len(matches) == 0 {
		return
	}
	fmt.Fprintln(out, "- Classifications:")
	for _, classification := range matches {
		fmt.Fprintf(out, "  - %s/%s value=%s", classification.Source, classification.Confidence, classification.Value)
		if classification.Evidence != "" {
			fmt.Fprintf(out, " evidence=%s", classification.Evidence)
		}
		if classification.Reason != "" {
			fmt.Fprintf(out, " reason=%s", classification.Reason)
		}
		fmt.Fprintln(out)
	}
}

func autosave(path string, session Session) error {
	if path == "" {
		return nil
	}
	return SaveDraft(path, session)
}

func rankDocuments(docs []APIDocument, ranked []string) []APIDocument {
	if len(ranked) == 0 {
		return docs
	}
	priority := map[string]int{}
	for i, path := range ranked {
		if _, ok := priority[path]; !ok {
			priority[path] = i
		}
	}
	out := append([]APIDocument(nil), docs...)
	sort.SliceStable(out, func(i, j int) bool {
		pi, iOK := priority[out[i].RelativePath]
		pj, jOK := priority[out[j].RelativePath]
		if iOK && jOK {
			return pi < pj
		}
		return iOK && !jOK
	})
	return out
}

func credentialBindings(value string) []string {
	var out []string
	for _, item := range splitList(value) {
		item = strings.TrimSpace(strings.Trim(item, "`'\""))
		if item == "" || strings.EqualFold(item, "none") || strings.EqualFold(item, "clear") {
			continue
		}
		out = append(out, slugIdent(item))
	}
	return dedupeStrings(out)
}

func clearablePolicyAnswer(value string) string {
	value = strings.TrimSpace(value)
	if strings.EqualFold(value, "none") || strings.EqualFold(value, "clear") {
		return ""
	}
	return value
}

func lastStepName(steps []*rollout.Step) string {
	for i := len(steps) - 1; i >= 0; i-- {
		if steps[i] != nil && steps[i].Name != "" {
			return steps[i].Name
		}
	}
	return ""
}

func defaultOutputSource(stepName string) string {
	if strings.TrimSpace(stepName) == "" {
		return "result"
	}
	return stepName + ".received_body"
}

func inputNames(inputs []*rollout.Input) string {
	var names []string
	for _, input := range inputs {
		if input != nil && input.Name != "" {
			names = append(names, input.Name)
		}
	}
	return strings.Join(names, ", ")
}

func defaultFieldSource(field string, inputs []*rollout.Input) string {
	leaf := field
	if _, after, ok := strings.Cut(field, "."); ok {
		leaf = after
	}
	for _, input := range inputs {
		if input != nil && (strings.EqualFold(input.Name, field) || strings.EqualFold(input.Name, leaf)) {
			return "inputs." + input.Name
		}
	}
	return ""
}

func normalizeRequestPromptField(raw string) string {
	raw = strings.TrimSpace(strings.Trim(raw, "`'\""))
	if raw == "" {
		return ""
	}
	section, rest, ok := strings.Cut(raw, ".")
	if !ok {
		return slugIdent(raw)
	}
	section = slugIdent(section)
	if section == "" {
		return slugIdent(raw)
	}
	switch section {
	case "path", "query", "header", "cookie", "body":
	default:
		return slugIdent(raw)
	}
	var parts []string
	for _, part := range strings.Split(rest, ".") {
		ident := slugIdent(part)
		if section == "header" {
			ident = slugHeaderIdent(part)
		}
		if ident != "" {
			parts = append(parts, ident)
		}
	}
	if len(parts) == 0 {
		return section
	}
	return section + "." + strings.Join(parts, ".")
}

func parsePriorStepSource(source string, prior []*rollout.Step) (string, string, bool) {
	source = strings.TrimSpace(source)
	for _, step := range prior {
		if step == nil || step.Name == "" {
			continue
		}
		prefix := step.Name + "."
		if strings.HasPrefix(source, prefix) {
			path := strings.TrimPrefix(source, prefix)
			if path == "" {
				path = "received_body"
			}
			return step.Name, path, true
		}
	}
	return "", "", false
}

func oneLine(value string) string {
	value = strings.Join(strings.Fields(value), " ")
	if len(value) <= 80 {
		return value
	}
	return value[:77] + "..."
}
