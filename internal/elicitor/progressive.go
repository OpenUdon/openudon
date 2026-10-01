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
	"time"

	"github.com/OpenUdon/apitools"
	"github.com/OpenUdon/apitools/catalog"
	publicinterview "github.com/OpenUdon/authoring/interview"
	"github.com/OpenUdon/openudon/internal/authoring"
	"github.com/OpenUdon/openudon/internal/browserverify"
	"github.com/OpenUdon/openudon/internal/projectdoc"
	"github.com/OpenUdon/openudon/internal/projectwizard"
	"github.com/OpenUdon/openudon/internal/sourcecatalog"
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
	statusOut := out
	if opts.DefaultMode == authoring.PromptDefaultsSilent {
		statusOut = io.Discard
	}

	projectText := projectwizard.Render(session.Project)
	refreshedSources, err := RefreshSessionSources(ctx, session, SourceRefreshOptions{
		ExampleDir: opts.ExampleDir, Query: projectText, LocalSources: opts.LocalSources, SourceRoots: opts.SourceRoots,
		BrowserSources: opts.BrowserSources, BrowserRegistries: opts.BrowserRegistries, BrowserVerifications: opts.BrowserVerifications,
		NetworkPolicy: opts.NetworkPolicy, At: time.Now().UTC(), RejectIncomplete: true,
	})
	if err != nil {
		return Artifacts{}, err
	}
	session, discovery, registryReport := refreshedSources.Session, refreshedSources.Discovery, refreshedSources.Registry
	docs := discovery.Docs
	openingBrief := ""
	if session.Intent.Workflow != nil {
		openingBrief = strings.TrimSpace(session.Intent.Workflow.Description)
	}
	draftJSONErrorReported := false
	skipNextDraft := opts.DisableAIDraft
	reportedDraftEvents := 0
	questionDrafted := map[string]bool{}
	reviewedFinalDrafts := map[string][]DraftReviewIssue{}
	remoteLookupAttempted := false
	attemptRemoteLookup := func(session Session) {
		if remoteLookupAttempted || len(docs) > 0 {
			return
		}
		policy := firstNonEmpty(opts.NetworkPolicy, "ask")
		approved := policy == "allow" || session.Interview.Metadata["remote_lookup_decision"] == "allow"
		if policy == "ask" && !approved {
			return
		}
		remoteLookupAttempted = true
		report, lookupErr := DiscoverRemoteSourceHints(ctx, firstNonEmpty(session.Boundary.Outcome, openingBrief), RemoteSourceLookupOptions{Policy: policy, Approved: approved})
		if lookupErr != nil {
			fmt.Fprintf(statusOut, "icot: bounded remote source lookup failed: %v\n", lookupErr)
			return
		}
		for _, candidate := range report.Candidates {
			fmt.Fprintf(statusOut, "icot: remote source candidate %s:%s %s (%s)\n", candidate.Kind, candidate.ID, candidate.URL, candidate.Provenance)
		}
		if report.Blocker != nil {
			fmt.Fprintf(statusOut, "icot: deferable source blocker: %s\n", report.Blocker.Message)
		}
	}
	attemptRemoteLookup(session)
	for _, blocker := range registryReport.Blockers {
		fmt.Fprintf(statusOut, "icot: deferable browser registry blocker: %s\n", blocker.Message)
	}
	nextSessionEvents := func(session Session) []authoring.PromptEvent {
		return catalogPlanEvents(session, &reportedDraftEvents)
	}
	interviewBinding := openUdonInterviewBinding(docs)
	hooks := authoring.ProgressiveLoopHooks[Session, APIDocument, Artifacts]{
		Session:       session,
		Documents:     docs,
		Opening:       openingBrief,
		Brief:         projectText,
		NoLLM:         opts.NoLLM,
		DefaultMode:   opts.DefaultMode,
		OpeningLabel:  "Workflow goal",
		Interview:     &interviewBinding,
		OpeningPrompt: "Tell me what you want this API/workflow to accomplish. Include inputs, API actions, outputs, and safety constraints if you know them. For send/create/update/delete/post/upload/notify actions, explicitly name the provider and action, for example \"send the report using Google Gmail\". Do not paste secrets.",
		Extractor:     extractor,
		Normalize: func(session *Session) {
			session.Normalize()
		},
		ApplyOpeningAnswer: func(session *Session, answer string, docs []APIDocument) error {
			if err := applyProgressiveAnswerChecked(session, QuestionPlan{Slots: []string{"workflow.goal"}}, answer, docs); err != nil {
				return err
			}
			hints, err := BuildCatalogHints(answer, opts.CatalogHintOptions)
			if err != nil {
				fmt.Fprintf(statusOut, "icot: apitools catalog advisory skipped: %v\n", err)
			} else {
				printCatalogHints(statusOut, hints)
				addCatalogPlanSteps(session, hints)
			}
			return nil
		},
		OpeningEvents: nextSessionEvents,
		RefreshDocuments: func(session Session, docs []APIDocument) ([]APIDocument, error) {
			attemptRemoteLookup(session)
			projectText := projectwizard.Render(session.Project)
			refreshed, err := RefreshSessionSources(ctx, session, SourceRefreshOptions{
				ExampleDir: opts.ExampleDir, Query: projectText, LocalSources: opts.LocalSources, SourceRoots: opts.SourceRoots,
				BrowserSources: opts.BrowserSources, BrowserRegistries: opts.BrowserRegistries, BrowserVerifications: opts.BrowserVerifications,
				NetworkPolicy: opts.NetworkPolicy, At: time.Now().UTC(), RejectIncomplete: true,
			})
			if err != nil {
				return nil, err
			}
			discovery, registryReport = refreshed.Discovery, refreshed.Registry
			return refreshed.Discovery.Docs, nil
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
			return draftRequestMappings(ctx, statusOut, extractor, session, docs, issues, question)
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
			if opts.DefaultMode == authoring.PromptDefaultsSilent {
				return
			}
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
		Ready: func(session Session, issues []ReadinessIssue) bool {
			if opts.VerifyOnly {
				_, err := RenderArtifacts(session)
				return err == nil
			}
			return progressiveReady(session, issues)
		},
		AfterRound: func(session *Session, docs []APIDocument) error {
			defaultSingleOpenAPIDoc(session, docs)
			session.SourcePlan = syncSelectedSourcePlansWithBrowser(*session, discovery.Plans, opts.LocalSources, opts.BrowserSources)
			var err error
			session.SourcePlan, err = AttachBrowserVerifications(session.SourcePlan, opts.BrowserVerifications, time.Now().UTC())
			return err
		},
		FinalConfirm: func(prompts *authoring.PromptSession, session *Session, docs []APIDocument, events *[]authoring.PromptEvent) (Artifacts, error) {
			var review func(context.Context, *Session, Artifacts, []ReadinessIssue) []DraftReviewIssue
			if !opts.VerifyOnly {
				review = func(ctx context.Context, session *Session, artifacts Artifacts, issues []ReadinessIssue) []DraftReviewIssue {
					key := finalDraftReviewKey(artifacts)
					if key == "" {
						return nil
					}
					if cached, ok := reviewedFinalDrafts[key]; ok {
						return cached
					}
					var reviewExtractor Extractor
					if !opts.NoLLM && !opts.VerifyOnly {
						reviewExtractor = extractor
					}
					reviewIssues := reviewFinalDraft(ctx, statusOut, reviewExtractor, session, docs, issues, events)
					reviewedFinalDrafts[key] = reviewIssues
					return reviewIssues
				}
			}
			reviewRepair := opts.ReviewRepair && !opts.VerifyOnly
			return finalProgressiveConfirmationLoop(ctx, out, &prompter{PromptSession: prompts, out: out}, session, docs, opts.DraftPath, events, opts.DefaultMode != authoring.PromptDefaultsSilent, reviewRepair, review, opts.VerifyOnly, opts.AutoApprove)
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
		TranscriptVersion: TranscriptVersion,
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

func progressiveSessionAfterDiscovery(session Session, plans []SourceMaterialization, explicit []apitools.LocalSource, networkPolicy string) Session {
	return progressiveSessionAfterDiscoveryV2(session, plans, explicit, nil, networkPolicy)
}

func progressiveSessionAfterDiscoveryV2(session Session, plans []SourceMaterialization, explicit []apitools.LocalSource, browserExplicit []BrowserSourceInput, networkPolicy string) Session {
	session.SourcePlan = mergeSelectedSourcePlansWithBrowser(session, plans, explicit, browserExplicit)
	session.Normalize()
	if session.Interview.Metadata == nil {
		session.Interview.Metadata = map[string]string{}
	}
	session.Interview.Metadata["network_policy"] = firstNonEmpty(networkPolicy, "ask")
	return session
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

type finalDraftReviewer func(context.Context, *Session, Artifacts, []ReadinessIssue) []DraftReviewIssue

func finalProgressiveConfirmationLoop(ctx context.Context, out io.Writer, p *prompter, session *Session, docs []APIDocument, draftPath string, events *[]TranscriptEvent, showAssumptions bool, reviewRepair bool, review finalDraftReviewer, verifyOnly bool, autoApprove bool) (Artifacts, error) {
	repairAttempts := 0
	askedReviewQuestions := map[string]bool{}
	for {
		artifacts, err := RenderArtifacts(*session)
		if err != nil && len(session.Interview.Deferrals) > 0 {
			artifacts, err = RenderDraftArtifacts(*session)
		}
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
		*session = artifacts.Session
		issues := CheckReadiness(artifacts.Session, docs)
		if !verifyOnly && !artifacts.Incomplete && firstFinalRepairIssue(issues).Code != "" {
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
		if review != nil && !artifacts.Incomplete {
			reviewIssues := review(ctx, session, artifacts, issues)
			if reviewRepair && len(reviewIssues) > 0 && repairAttempts < 2 {
				repairAttempts++
				changed, rejected := applyDraftReviewRemediations(session, reviewIssues, docs)
				if events != nil {
					*events = append(*events, TranscriptEvent{Kind: "draft_repair_attempt", Data: map[string]any{
						"attempt":  repairAttempts,
						"changed":  changed,
						"rejected": rejected,
					}})
				}
				if changed {
					session.Normalize()
					if err := autosave(draftPath, *session); err != nil {
						return Artifacts{}, err
					}
					continue
				}
				if len(rejected) > 0 && events != nil {
					*events = append(*events, TranscriptEvent{Kind: "draft_repair_rejected", Data: rejected})
				}
			}
			forcedAnswerChanged := false
			for i := range reviewIssues {
				issue := reviewIssues[i]
				question := draftReviewForcedQuestion(issue)
				if question == "" {
					continue
				}
				key := issue.Code + "\x00" + issue.Slot + "\x00" + question
				if askedReviewQuestions[key] {
					continue
				}
				askedReviewQuestions[key] = true
				answer, err := p.askDefaultForced(question, issue.SuggestedAnswer)
				if err != nil {
					return Artifacts{}, err
				}
				applied, clarification := applyForcedDraftReviewAnswer(session, issue, answer)
				if applied {
					forcedAnswerChanged = true
				}
				if clarification != "" {
					reviewIssues[i].SuggestedAnswer = strings.TrimSpace(strings.Join([]string{reviewIssues[i].SuggestedAnswer, clarification}, " "))
				}
				addDecisionEvidence(session, DecisionEvidence{
					Stage:                decisionStageDraftReview,
					Slot:                 firstNonEmpty(issue.Slot, "draft_review."+issue.Code),
					Value:                strings.TrimSpace(answer),
					Source:               mappingSourceUser,
					Confidence:           mappingConfidenceReview,
					Reason:               forcedDraftReviewAnswerReason(applied),
					Evidence:             question,
					RequiresConfirmation: false,
				})
				if events != nil {
					*events = append(*events, TranscriptEvent{Kind: "draft_flow_review_question", Data: map[string]any{
						"code":     issue.Code,
						"slot":     issue.Slot,
						"question": question,
					}})
					*events = append(*events, TranscriptEvent{Kind: "draft_flow_review_answer", Data: map[string]any{
						"code":    issue.Code,
						"slot":    issue.Slot,
						"answer":  strings.TrimSpace(answer),
						"applied": applied,
					}})
				}
			}
			if forcedAnswerChanged {
				session.Normalize()
				if err := autosave(draftPath, *session); err != nil {
					return Artifacts{}, err
				}
				var renderErr error
				artifacts, renderErr = RenderArtifacts(*session)
				if renderErr != nil {
					return Artifacts{}, renderErr
				}
				*session = artifacts.Session
				issues = CheckReadiness(artifacts.Session, docs)
			}
			if reviewRepair && len(reviewIssues) > 0 && repairAttempts >= 2 {
				fmt.Fprintln(out, "icot: review repair still has unresolved flow warnings; refine the workflow goal or choose better API artifacts if these warnings are not acceptable.")
			}
			issues = sortReadinessIssues(append(issues, draftReviewIssuesToReadiness(reviewIssues)...))
			artifacts.Session = *session
			artifacts.IntentHCL = annotateIntentHCLWithFlowReviewWarnings(artifacts.IntentHCL, reviewIssues)
		}
		fmt.Fprintln(out, "\n----- complete authoring proposal -----")
		printProposal(out, artifacts)
		printReadinessWarnings(out, issues)
		if showAssumptions {
			printAssumptions(out, artifacts.Session.Assumptions)
		}
		if len(artifacts.Session.Annotations) > 0 {
			fmt.Fprintln(out, "LLM-prefilled values are recorded in the interview evidence ledger and require this final confirmation.")
		}
		if autoApprove {
			return artifacts, nil
		}
		answer, err := p.askDefaultForced("Type approve, edit <slot>, explain <assumption-id>, or cancel", "approve")
		if err != nil {
			return Artifacts{}, err
		}
		answer = strings.TrimSpace(strings.ToLower(answer))
		switch {
		case answer == "" || answer == "approve" || answer == "save":
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
			fmt.Fprintln(out, "Please type approve, edit <slot>, explain <assumption-id>, or cancel.")
		}
	}
}

func printProposal(out io.Writer, artifacts Artifacts) {
	session := artifacts.Session
	printSummary(out, session)
	fmt.Fprintln(out, "\nActive boundary:")
	fmt.Fprintf(out, "- Outcome: %s\n", session.Boundary.Outcome)
	fmt.Fprintf(out, "- Actor/trigger: %s / %s\n", session.Boundary.Actor, session.Boundary.Trigger)
	fmt.Fprintf(out, "- Success evidence: %s\n", strings.Join(session.Boundary.SuccessEvidence, "; "))
	fmt.Fprintf(out, "- Non-goals: %s\n", firstNonEmpty(strings.Join(session.Boundary.NonGoals, "; "), "none declared"))
	if len(session.CandidateWorkflows) > 0 {
		fmt.Fprintln(out, "Candidate workflows (not implemented):")
		for _, candidate := range session.CandidateWorkflows {
			fmt.Fprintf(out, "- %s: %s; deferred because %s; promote when %s\n", candidate.Title, candidate.Outcome, candidate.DeferralReason, candidate.PromotionTrigger)
		}
	}
	if len(session.SourcePlan) > 0 {
		fmt.Fprintln(out, "Selected sources:")
		for _, source := range session.SourcePlan {
			fmt.Fprintf(out, "- %s:%s %s -> %s sha256:%s (%s)\n", source.Kind, source.ID, source.SourcePath, source.TargetPath, source.SHA256, source.Provenance)
			if source.Kind == browserSourceFamily {
				fmt.Fprintf(out, "  actions=%s origins=%s lifecycle=%s expires=%s login-session-required=%t\n", strings.Join(source.Actions, ","), strings.Join(source.Origins, ","), source.Lifecycle, source.ExpiresAt, source.LoginStateRequired)
				for _, verification := range source.BrowserVerifications {
					if verification.Summary.ReportVersion == browserverify.LiveCheckVersion {
						fmt.Fprintf(out, "  current-page-verification engine=chromium ok=%t checked=%s actions=%s report=%s\n", verification.Summary.OK, verification.Summary.CheckedAt, strings.Join(verification.Summary.Actions, ","), verification.Summary.SourceSHA256)
						continue
					}
					engines := make([]string, 0, len(verification.Summary.Engines))
					for _, engine := range verification.Summary.Engines {
						engines = append(engines, engine.Engine+":"+engine.Status)
					}
					fmt.Fprintf(out, "  portability-verification ok=%t checked=%s actions=%s engines=%s report=%s (optional confidence only)\n", verification.Summary.OK, verification.Summary.CheckedAt, strings.Join(verification.Summary.Actions, ","), strings.Join(engines, ","), verification.Summary.SourceSHA256)
				}
			}
			if source.Kind == browserAuthenticationSourceFamily {
				fmt.Fprintf(out, "  authentication-flows=%s origins=%s lifecycle=%s expires=%s credential-slots=%s\n", strings.Join(source.Flows, ","), strings.Join(source.Origins, ","), source.Lifecycle, source.ExpiresAt, formatBrowserFlowSlots(source.FlowCredentialSlots))
			}
		}
	}
	if session.BrowserRoute == "browser" {
		fmt.Fprintf(out, "Browser route: session posture=%s; authoring mutation approvals=%s (runtime approval remains separate)\n", firstNonEmpty(session.BrowserSession, "unresolved"), firstNonEmpty(strings.Join(session.BrowserApprovals, ","), "none"))
		fmt.Fprintf(out, "Browser authentication authoring approvals=%s (credentials and live sessions remain runtime-private)\n", firstNonEmpty(strings.Join(session.BrowserAuthenticationApprovals, ","), "none"))
	}
	if len(session.Interview.Deferrals) > 0 {
		fmt.Fprintln(out, "Unresolved technical deferrals:")
		for _, deferral := range session.Interview.Deferrals {
			fmt.Fprintf(out, "- %s owner=%s impact=%s unblock=%s next=%s\n", deferral.NodeID, deferral.Owner, deferral.Impact, deferral.UnblockCondition, deferral.SuggestedNextAction)
		}
	}
	fmt.Fprintln(out, "Exact file actions:")
	fmt.Fprintln(out, "- write project.md")
	if artifacts.Incomplete {
		fmt.Fprintln(out, "- write workflows/intent.draft.hcl (never workflows/intent.hcl)")
		fmt.Fprintln(out, "- write .icot/session.yaml and .icot/readiness.json")
	} else {
		fmt.Fprintln(out, "- write workflows/intent.hcl")
		fmt.Fprintln(out, "- remove obsolete generated draft/readiness files after promotion")
	}
	for _, source := range session.SourcePlan {
		fmt.Fprintf(out, "- materialize %s\n", source.TargetPath)
	}
	for _, source := range session.SourcePlan {
		if source.Kind == browserSourceFamily {
			fmt.Fprintln(out, "- write .icot/browser-sources.json")
			break
		}
	}
	for _, source := range session.SourcePlan {
		if source.Kind == browserAuthenticationSourceFamily {
			fmt.Fprintln(out, "- write .icot/browser-authentication.json")
			break
		}
	}
}

// PrintProposal renders the complete reviewed boundary and exact file actions
// before a caller requests approval to write a pre-completed session.
func PrintProposal(out io.Writer, artifacts Artifacts) {
	printProposal(out, artifacts)
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
	base.DraftOperations = append([]OperationDetailRef(nil), base.DraftOperations...)
	base.DraftEvents = cloneDraftEventsForMerge(base.DraftEvents)
	overlay.DraftEvents = cloneDraftEventsForMerge(overlay.DraftEvents)
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
	base.Boundary.Outcome = firstNonEmpty(base.Boundary.Outcome, overlay.Boundary.Outcome)
	base.Boundary.Actor = firstNonEmpty(base.Boundary.Actor, overlay.Boundary.Actor)
	base.Boundary.Trigger = firstNonEmpty(base.Boundary.Trigger, overlay.Boundary.Trigger)
	base.Boundary.SuccessEvidence = dedupeStrings(append(base.Boundary.SuccessEvidence, overlay.Boundary.SuccessEvidence...))
	base.Boundary.NonGoals = dedupeStrings(append(base.Boundary.NonGoals, overlay.Boundary.NonGoals...))
	base.Boundary.Confirmed = base.Boundary.Confirmed || overlay.Boundary.Confirmed
	base.CandidateWorkflows = projectdoc.NormalizeCandidateWorkflows(append(base.CandidateWorkflows, overlay.CandidateWorkflows...))
	base.Interview.Evidence = mergeInterviewEvidence(base.Interview.Evidence, overlay.Interview.Evidence)
	base.Annotations = append(base.Annotations, overlay.Annotations...)
	base.Assumptions = mergeAssumptions(base.Assumptions, overlay.Assumptions)
	base.DraftOperations = appendOperationDetailRefs(base.DraftOperations, overlay.DraftOperations)
	base.DraftEvents = append(base.DraftEvents, overlay.DraftEvents...)
	base.Normalize()
	return base
}

func mergeInterviewEvidence(base, overlay []publicinterview.Evidence) []publicinterview.Evidence {
	out := append([]publicinterview.Evidence(nil), base...)
	seen := make(map[string]bool, len(out))
	for _, evidence := range out {
		seen[strings.TrimSpace(evidence.ID)] = true
	}
	for _, evidence := range overlay {
		if id := strings.TrimSpace(evidence.ID); id == "" || seen[id] {
			continue
		} else {
			seen[id] = true
		}
		out = append(out, evidence)
	}
	return out
}

func defaultSingleOpenAPIDoc(session *Session, docs []APIDocument) {
	if session == nil || intentAPISourceRef(session.Intent) != "" || len(docs) != 1 || !session.Intent.RequiresOpenAPI() {
		return
	}
	setIntentAPISourceFromDoc(session, docs[0])
	if isBrowserDocument(docs[0]) {
		session.BrowserRoute = "browser"
	}
	addMappingClassification(session, MappingClassification{
		Slot:                 "intent.source",
		Value:                docs[0].RelativePath,
		Source:               mappingSourceFallbackDefault,
		Confidence:           mappingConfidenceReview,
		Evidence:             docs[0].RelativePath,
		Reason:               "Only one validated local source document is available for source-backed steps.",
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
		if candidate.Kind == catalog.SpecKind("security-overlay") {
			fmt.Fprintf(out, "icot: using existing %s security overlay sidecar %s\n", candidate.ProviderName, candidate.RelativePath)
			continue
		}
		fmt.Fprintf(out, "icot: using existing apitools API document %s\n", candidate.RelativePath)
	}
	for _, candidate := range result.Copied {
		if candidate.Kind == catalog.SpecKind("security-overlay") {
			fmt.Fprintf(out, "icot: retrieved %s security overlay from apitools to %s\n", candidate.ProviderName, candidate.RelativePath)
			continue
		}
		if candidate.Kind == catalog.SpecKind("advisory-overlay") {
			fmt.Fprintf(out, "icot: retrieved %s advisory OpenAPI overlay from apitools to %s\n", candidate.ProviderName, candidate.RelativePath)
			continue
		}
		fmt.Fprintf(out, "icot: retrieved %s API document from apitools to %s\n", candidate.ProviderName, candidate.RelativePath)
	}
	for _, note := range result.Notes {
		fmt.Fprintf(out, "icot: catalog retrieval note: %s\n", note)
	}
	for _, hint := range result.Missing {
		fmt.Fprintf(out, "icot: no first-class OpenAPI is available for %s; cannot continue to operation selection until an artifact is generated/provided.\n", firstNonEmpty(hint.Provider.DisplayName, hint.Provider.ID))
	}
	return nil
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
				fmt.Fprintf(out, "icot: local API document not found: %s. Create that file first, or enter an existing file under %s/.\n", path, strings.Join(sourcecatalog.API(), "/, "))
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
			if candidate.Kind == catalog.SpecKind("security-overlay") {
				fmt.Fprintf(out, "icot: migrated %s security overlay sidecar to %s\n", candidate.ProviderName, candidate.RelativePath)
				continue
			}
			fmt.Fprintf(out, "icot: migrated %s API document to %s\n", candidate.ProviderName, candidate.RelativePath)
		}
		for _, note := range result.Notes {
			fmt.Fprintf(out, "icot: catalog retrieval note: %s\n", note)
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
