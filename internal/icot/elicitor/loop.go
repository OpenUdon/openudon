package elicitor

import (
	"bufio"
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
	"github.com/OpenUdon/uws/uws1"
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
	if !opts.NoLLM && !opts.VerifyOnly {
		return runProgressive(ctx, in, out, seed, opts)
	}
	return runManual(ctx, in, out, seed, opts)
}

func runManual(ctx context.Context, in io.Reader, out io.Writer, seed Session, opts Options) (Artifacts, error) {
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
	promptSession := authoring.NewPromptSession(reader, out)
	promptSession.SetDefaultMode(opts.DefaultMode)
	p := &prompter{PromptSession: promptSession, out: out}
	statusOut := out
	if opts.DefaultMode == authoring.PromptDefaultsSilent {
		statusOut = io.Discard
	}
	openingBrief := ""
	if opts.VerifyOnly {
		projectText := projectwizard.Render(session.Project)
		docs, err := DiscoverLocalAPIs(opts.ExampleDir, projectText)
		if err != nil {
			return Artifacts{}, err
		}
		if shouldRetrieveCatalogArtifacts(session, docs) {
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
		artifacts, err := finalVerificationLoop(out, p, &session, docs, opts.DraftPath, opts.DefaultMode != authoring.PromptDefaultsSilent)
		if err == nil {
			if saveErr := SaveTranscript(opts.TranscriptPath, p.turns(), artifacts.Session); saveErr != nil {
				return artifacts, saveErr
			}
		}
		return artifacts, err
	}

	if session.Intent.Workflow == nil || strings.TrimSpace(session.Intent.Workflow.Description) == "" {
		fmt.Fprintln(out, "Describe the workflow in one or two sentences. Mention APIs/files if known. Do not paste secrets.")
		opening, err := p.ask("Workflow brief")
		if err != nil {
			return Artifacts{}, err
		}
		openingBrief = strings.TrimSpace(opening)
		if strings.TrimSpace(opening) != "" {
			if !opts.NoLLM {
				prefill, err := extractor.Kickoff(ctx, opening)
				if err == nil {
					session = mergeSessions(session, prefill)
				} else {
					fmt.Fprintf(statusOut, "icot: LLM prefill skipped: %v\n", err)
				}
			}
			if session.Intent.Workflow == nil {
				session.Intent.Workflow = &rollout.WorkflowMeta{}
			}
			session.Intent.Workflow.Description = firstNonEmpty(session.Intent.Workflow.Description, opening)
			session.Intent.Workflow.Name = firstNonEmpty(session.Intent.Workflow.Name, actionName(opening))
			session.Project.Goal = firstNonEmpty(session.Project.Goal, opening)
			session.Normalize()
			if err := autosave(opts.DraftPath, session); err != nil {
				return Artifacts{}, err
			}
			printSummary(out, session)
			PrintCatalogHints(statusOut, opening)
		}
	}

	var err error
	if session.Intent.Workflow == nil {
		session.Intent.Workflow = &rollout.WorkflowMeta{}
	}
	session.Intent.Workflow.Name, err = p.askDefault("Workflow name", firstNonEmpty(session.Intent.Workflow.Name, slug(session.Project.ProjectName)))
	if err != nil {
		return Artifacts{}, err
	}
	session.Intent.Workflow.Name = slug(session.Intent.Workflow.Name)
	session.Intent.Workflow.Description, err = p.askDefault("Workflow goal", session.Intent.Workflow.Description)
	if err != nil {
		return Artifacts{}, err
	}
	if err := p.collectWorkflowMetadata(session.Intent.Workflow); err != nil {
		return Artifacts{}, err
	}
	session.Project.ProjectName = humanTitle(session.Intent.Workflow.Name)
	session.Project.Goal = session.Intent.Workflow.Description
	session.Normalize()
	if err := autosave(opts.DraftPath, session); err != nil {
		return Artifacts{}, err
	}
	printSummary(out, session)

	projectText := projectwizard.Render(session.Project)
	docs, err := DiscoverLocalAPIs(opts.ExampleDir, projectText)
	if err != nil {
		return Artifacts{}, err
	}
	if shouldRetrieveCatalogArtifacts(session, docs) {
		if err := retrieveCatalogArtifactsForSession(statusOut, session, opts.ExampleDir, opts.CatalogHintOptions); err != nil {
			return Artifacts{}, err
		}
		projectText = projectwizard.Render(session.Project)
		docs, err = DiscoverLocalAPIs(opts.ExampleDir, projectText)
		if err != nil {
			return Artifacts{}, err
		}
		clearUnavailableAPIDocumentRefs(&session, docs)
		if issue := blockingAPIDocumentIssue(session, docs); issue.Code != "" {
			fmt.Fprintf(out, "Intent is incomplete: %s\n", issue.Message)
			return Artifacts{}, errors.New(issue.Message)
		}
	}
	if !opts.NoLLM && len(docs) > 1 {
		if ranked, err := extractor.Disambiguate(ctx, session.Intent.Workflow.Description, docs); err == nil {
			docs = rankDocuments(docs, ranked)
		} else {
			fmt.Fprintf(statusOut, "icot: OpenAPI ranking skipped: %v\n", err)
		}
	}
	if !opts.NoLLM && !opts.DisableAIDraft && len(session.Intent.Steps) == 0 {
		draft, err := extractor.Draft(ctx, DraftRequest{
			Opening: openingBrief,
			Session: session,
			Docs:    docs,
		})
		if err == nil && LooksLikeSession(draft) {
			session = mergeSessions(session, draft)
			fmt.Fprintln(statusOut, "icot: drafted intent defaults from brief and local metadata; final save confirms listed assumptions")
			session.Normalize()
			if err := autosave(opts.DraftPath, session); err != nil {
				return Artifacts{}, err
			}
			printSummary(out, session)
		} else if err != nil {
			fmt.Fprintf(statusOut, "icot: AI draft skipped: %v\n", err)
		}
	}
	usesAPIDefault := session.Intent.RequiresOpenAPI() || len(docs) > 0
	usesAPI, err := p.askYesNo("Use OpenAPI/API steps?", usesAPIDefault)
	if err != nil {
		return Artifacts{}, err
	}
	if usesAPI {
		if len(docs) == 0 {
			if issue := blockingAPIDocumentIssue(session, docs); issue.Code != "" {
				fmt.Fprintf(out, "Intent is incomplete: %s\n", issue.Message)
				return Artifacts{}, errors.New(issue.Message)
			}
			apiPath, err := p.askDefault("OpenAPI document path or URL", session.Intent.OpenAPI)
			if err != nil {
				return Artifacts{}, err
			}
			session.Intent.OpenAPI = strings.TrimSpace(apiPath)
		} else {
			useDefaultDoc := true
			if len(docs) > 1 {
				useDefaultDoc, err = p.askYesNo("Use one default OpenAPI document for all API steps?", strings.TrimSpace(session.Intent.OpenAPI) != "")
				if err != nil {
					return Artifacts{}, err
				}
			}
			if useDefaultDoc {
				doc, err := p.chooseDocument("OpenAPI document", docs, session.Intent.OpenAPI)
				if err != nil {
					return Artifacts{}, err
				}
				session.Intent.OpenAPI = doc.RelativePath
			} else {
				session.Intent.OpenAPI = ""
			}
		}
	} else {
		session.Intent.OpenAPI = ""
		clearAPISteps(session.Intent.Steps)
	}
	session.Normalize()
	if err := autosave(opts.DraftPath, session); err != nil {
		return Artifacts{}, err
	}
	printSummary(out, session)

	if len(session.Intent.Inputs) == 0 {
		value, err := p.askOptionalDefault("Runtime inputs (name:type, comma-separated; blank for none)", "")
		if err != nil {
			return Artifacts{}, err
		}
		session.Intent.Inputs = parseInputs(value)
		session.Normalize()
		if err := autosave(opts.DraftPath, session); err != nil {
			return Artifacts{}, err
		}
		printSummary(out, session)
	}

	if len(session.Intent.Steps) == 0 {
		steps, err := p.collectSteps(usesAPI, session.Intent.OpenAPI, docs, session.Intent.Inputs, nil)
		if err != nil {
			return Artifacts{}, err
		}
		session.Intent.Steps = steps
		session.Normalize()
		if err := autosave(opts.DraftPath, session); err != nil {
			return Artifacts{}, err
		}
		printSummary(out, session)
	}

	if len(session.Intent.Outputs) == 0 {
		last := lastStepName(session.Intent.Steps)
		defaultSource := defaultOutputSource(last)
		name, err := p.askDefault("Output name", "result")
		if err != nil {
			return Artifacts{}, err
		}
		source, err := p.askDefault("Output source", defaultSource)
		if err != nil {
			return Artifacts{}, err
		}
		session.Intent.Outputs = []*rollout.Output{{Name: slugIdent(name), From: source}}
		session.Normalize()
		if err := autosave(opts.DraftPath, session); err != nil {
			return Artifacts{}, err
		}
		printSummary(out, session)
	}

	if hasRuntime(session.Intent.Steps, "cmd") {
		session.Project.CmdApproved, err = p.askYesNo("Approve cmd runtime for this project?", session.Project.CmdApproved)
		if err != nil {
			return Artifacts{}, err
		}
		session.Normalize()
		if err := autosave(opts.DraftPath, session); err != nil {
			return Artifacts{}, err
		}
	}
	if hasRuntime(session.Intent.Steps, "ssh") {
		session.Project.SSHApproved, err = p.askYesNo("Approve ssh runtime for this project?", session.Project.SSHApproved)
		if err != nil {
			return Artifacts{}, err
		}
		session.Normalize()
		if err := autosave(opts.DraftPath, session); err != nil {
			return Artifacts{}, err
		}
	}

	sideEffectDefault := session.SideEffectScope
	if sideEffectDefault == "" {
		sideEffectDefault = projectwizard.InferSideEffectScope(session.Safety)
	}
	session.SideEffectScope, err = p.askSideEffectScope(sideEffectDefault)
	if err != nil {
		return Artifacts{}, err
	}
	session.Normalize()
	if err := autosave(opts.DraftPath, session); err != nil {
		return Artifacts{}, err
	}

	credentialAnswer, err := p.askOptionalDefault("Credential binding names only (comma-separated; blank for none)", strings.Join(session.Credentials, ", "))
	if err != nil {
		return Artifacts{}, err
	}
	session.Credentials = credentialBindings(credentialAnswer)
	session.CredentialsSet = true
	session.Normalize()
	if err := autosave(opts.DraftPath, session); err != nil {
		return Artifacts{}, err
	}
	session.Safety, err = p.askOptionalDefault("Safety and approval notes", session.Safety)
	if err != nil {
		return Artifacts{}, err
	}
	session.Safety = clearablePolicyAnswer(session.Safety)
	session.SafetySet = true
	session.Normalize()
	if err := autosave(opts.DraftPath, session); err != nil {
		return Artifacts{}, err
	}
	session.Fallback, err = p.askOptionalDefault("Fallback behavior", session.Fallback)
	if err != nil {
		return Artifacts{}, err
	}
	session.Fallback = clearablePolicyAnswer(session.Fallback)
	session.FallbackSet = true
	session.Normalize()
	if !opts.NoLLM {
		refined, err := extractor.Refine(ctx, session)
		if err == nil {
			session = refined
			session.Normalize()
		} else {
			fmt.Fprintf(statusOut, "icot: LLM prose refinement skipped: %v\n", err)
		}
	}
	if err := autosave(opts.DraftPath, session); err != nil {
		return Artifacts{}, err
	}

	artifacts, err := finalVerificationLoop(out, p, &session, docs, opts.DraftPath, opts.DefaultMode != authoring.PromptDefaultsSilent)
	transcriptSession := session
	if err == nil {
		transcriptSession = artifacts.Session
		if saveErr := SaveTranscript(opts.TranscriptPath, p.turns(), transcriptSession); saveErr != nil {
			return artifacts, saveErr
		}
	}
	return artifacts, err
}

func finalVerificationLoop(out io.Writer, p *prompter, session *Session, docs []APIDocument, draftPath string, showAssumptions bool) (Artifacts, error) {
	for {
		artifacts, err := RenderArtifacts(*session)
		if err != nil {
			if handled, handleErr := answerFinalBlockingQuestion(out, p, session, docs, draftPath); handled || handleErr != nil {
				if handleErr != nil {
					return Artifacts{}, handleErr
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
		*session = artifacts.Session
		if blocking := firstFinalRepairIssue(CheckReadiness(artifacts.Session, docs)); blocking.Code != "" {
			if handled, handleErr := answerFinalBlockingQuestion(out, p, session, docs, draftPath); handled || handleErr != nil {
				if handleErr != nil {
					return Artifacts{}, handleErr
				}
				continue
			}
		}
		fmt.Fprintln(out, "\n----- current draft -----")
		printSummary(out, artifacts.Session)
		if showAssumptions {
			printAssumptions(out, artifacts.Session.Assumptions)
		}
		if len(artifacts.Session.Annotations) > 0 {
			fmt.Fprintln(out, "LLM-prefilled values are marked in the session annotations and require this final confirmation.")
		}
		answer, err := p.askDefault("Type save, edit <slot>, explain <assumption-id>, regenerate, or cancel", "save")
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
			if err := autosave(draftPath, *session); err != nil {
				return Artifacts{}, err
			}
		case strings.HasPrefix(answer, "explain"):
			id := strings.TrimSpace(strings.TrimPrefix(answer, "explain"))
			printAssumptionExplanation(out, *session, id)
		case answer == "regenerate":
			fmt.Fprintln(out, "Regenerate is available by rerunning iCoT from the saved draft or editing a slot before save.")
		default:
			fmt.Fprintln(out, "Please type save, edit <slot>, explain <assumption-id>, regenerate, or cancel.")
		}
	}
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

func blockingAPIDocumentIssue(session Session, docs []APIDocument) ReadinessIssue {
	missingRefs := missingLocalAPIDocumentRefs(session, docs)
	if len(missingRefs) > 0 && len(CatalogHintsForSession(session)) > 0 {
		clone := session
		clearUnavailableAPIDocumentRefs(&clone, docs)
		message := missingAPIDocMessage(clone, docs)
		if strings.Contains(message, "No first-class OpenAPI is available") {
			return ReadinessIssue{
				Code:            "missing_api_doc",
				Slot:            "intent.openapi",
				Severity:        readinessBlocking,
				Message:         message,
				SuggestedAnswer: "Generate/provide the missing API artifact, then rerun iCoT.",
			}
		}
	}
	for _, issue := range CheckReadiness(session, docs) {
		if issue.Code == "missing_api_doc" && issue.Severity == readinessBlocking {
			if len(missingRefs) > 0 {
				return issue
			}
			if len(docs) == 0 && len(CatalogHintsForSession(session)) > 0 {
				return issue
			}
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

func (p *prompter) ask(label string) (string, error) {
	return p.Ask(label)
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

func (p *prompter) askYesNo(label string, defaultYes bool) (bool, error) {
	return p.AskYesNo(label, defaultYes)
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

func (p *prompter) collectWorkflowMetadata(workflow *rollout.WorkflowMeta) error {
	if workflow == nil {
		return nil
	}
	timeout, err := p.askOptionalSeconds("Workflow timeout seconds (blank for none)", workflow.Timeout)
	if err != nil {
		return err
	}
	workflow.Timeout = timeout
	currentKey := ""
	if workflow.Idempotency != nil {
		currentKey = workflow.Idempotency.Key
	}
	key, err := p.askOptionalDefault("Workflow idempotency key (blank for none)", currentKey)
	if err != nil {
		return err
	}
	key = strings.TrimSpace(key)
	if key == "" || strings.EqualFold(key, "none") || strings.EqualFold(key, "clear") {
		workflow.Idempotency = nil
		return nil
	}
	currentConflict := ""
	currentTTL := (*float64)(nil)
	if workflow.Idempotency != nil {
		currentConflict = workflow.Idempotency.OnConflict
		currentTTL = workflow.Idempotency.TTL
	}
	conflict, err := p.askOptionalDefault("Workflow idempotency onConflict (blank/reject/returnPrevious)", currentConflict)
	if err != nil {
		return err
	}
	conflict = strings.TrimSpace(conflict)
	if conflict != "" && conflict != "reject" && conflict != "returnPrevious" {
		return fmt.Errorf("workflow idempotency onConflict must be blank, reject, or returnPrevious")
	}
	ttl, err := p.askOptionalSeconds("Workflow idempotency ttl seconds (blank for none)", currentTTL)
	if err != nil {
		return err
	}
	workflow.Idempotency = &uws1.Idempotency{Key: key, OnConflict: conflict, TTL: ttl}
	return nil
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

func (p *prompter) turns() []ReplayTurn {
	return p.Turns()
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

func clearAPISteps(steps []*rollout.Step) {
	walkSteps(steps, func(step *rollout.Step) {
		step.OpenAPI = ""
		step.Operation = ""
	})
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

func hasRuntime(steps []*rollout.Step, runtime string) bool {
	found := false
	walkSteps(steps, func(step *rollout.Step) {
		if strings.EqualFold(strings.TrimSpace(step.Type), runtime) {
			found = true
		}
	})
	return found
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
		if ident := slugIdent(part); ident != "" {
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
