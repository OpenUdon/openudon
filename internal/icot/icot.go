package icot

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/OpenUdon/apitools"
	"github.com/OpenUdon/openudon/internal/authoring"
	"github.com/OpenUdon/openudon/internal/icot/artifactwriter"
	"github.com/OpenUdon/openudon/internal/icot/elicitor"
	"github.com/OpenUdon/openudon/internal/projectwizard"
	"github.com/OpenUdon/openudon/internal/sourcecatalog"
)

func Main(args []string, in io.Reader, out, errOut io.Writer) int {
	if len(args) > 0 && args[0] == "ui" {
		return runUI(args[1:], out, errOut)
	}
	if len(args) > 0 && args[0] == "browser-author" {
		return runBrowserAuthorLive(args[1:], in, out, errOut)
	}
	if len(args) > 0 && args[0] == "browser-authoring" {
		return runBrowserAuthoring(args[1:], out, errOut)
	}
	if len(args) > 0 && args[0] == "lint" {
		return runLint(args[1:], out, errOut)
	}
	if len(args) > 0 && args[0] == "reconcile" {
		return runReconcile(args[1:], in, out, errOut)
	}
	if len(args) > 0 && args[0] == "replay-eval" {
		return runReplayEval(args[1:], out, errOut)
	}
	if len(args) > 0 && args[0] == "scorecard" {
		return runScorecard(args[1:], out, errOut)
	}
	if len(args) > 0 && args[0] == "variants" {
		return runVariants(args[1:], out, errOut)
	}
	if len(args) > 0 && args[0] == "authoring-eval" {
		return runAuthoringEval(args[1:], out, errOut)
	}
	if len(args) > 0 && args[0] == "report" {
		return runReport(args[1:], out, errOut)
	}
	if len(args) > 0 && args[0] == "repair" {
		return runRepair(args[1:], out, errOut)
	}
	return runAuthor(args, in, out, errOut)
}

func runAuthor(args []string, in io.Reader, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("icot", flag.ContinueOnError)
	fs.SetOutput(out)
	example := fs.String("example", "", "Example directory where project.md will be created")
	dirAlias := fs.String("dir", "", "Alias for --example")
	force := fs.Bool("force", false, "Overwrite an existing project.md")
	yes := fs.Bool("yes", false, "Explicitly approve the complete proposal and accept overwrite prompts")
	printOnly := fs.Bool("print", false, "Render project.md and workflows/intent.hcl to stdout without writing files")
	fromExample := fs.String("from-example", "", "Seed answers from an existing example directory")
	answersFile := fs.String("answers", "", "Path to an openudon.icot-session.v2 YAML or JSON file")
	noLLM := fs.Bool("no-llm", false, "Disable optional LLM extraction assistance")
	noTranscript := fs.Bool("no-transcript", false, "Do not save local .icot transcript history")
	promptMode := fs.String("prompt-mode", "full", "Prompt mode: full, normal, or fast. full asks every question; normal accepts defaults visibly; fast skips defaulted questions")
	reviewRepair := fs.Bool("review-repair", false, "Experimental: apply up to two bounded repairs from pre-final flow review suggestions")
	agentMode := fs.Bool("agent", false, "Return the noninteractive frontier/source/file-action report without writing deliverables")
	jsonOutput := fs.Bool("json", false, "Write a structured JSON report to stdout")
	reportPath := fs.String("report", "", "Write a structured JSON report to this path")
	provider := fs.String("provider", "", "LLM provider for optional extraction: copilot-api, openai, anthropic, or gemini")
	model := fs.String("model", "", "LLM model for optional extraction")
	temperature := fs.Float64("temperature", 0.2, "LLM extraction temperature")
	var apiSourceFlags repeatedFlag
	var openAPIFlags repeatedFlag
	var browserProfileFlags repeatedFlag
	var browserVerificationFlags repeatedFlag
	var browserRegistryFlags repeatedFlag
	var browserAuthoringOriginFlags repeatedFlag
	var sourceRootFlags repeatedFlag
	fs.Var(&apiSourceFlags, "api-source", "Explicit API document KIND:ID=PATH; repeat for multiple sources")
	fs.Var(&openAPIFlags, "openapi", "OpenAPI shorthand ID=PATH; repeat for multiple sources")
	fs.Var(&browserProfileFlags, "browser-profile", "Verified browser capability/authentication profile, capability bundle, or guided-authoring result ID=PATH; repeat for multiple sources")
	fs.Var(&browserVerificationFlags, "browser-verification", "Value-free Browsertools live-check or portability report path; repeat for multiple reports")
	fs.Var(&browserRegistryFlags, "browser-registry", "Static Browsertools registry directory or HTTPS URL; repeat for multiple registries")
	browserAuthoringURL := fs.String("browser-authoring-url", "", "Explicit clean HTTPS or loopback URL for a non-executing Browsertools authoring handoff")
	browserAuthoringID := fs.String("browser-authoring-id", "", "Stable profile ID for the Browsertools authoring handoff")
	browserAuthoringAction := fs.String("browser-authoring-action", "", "Stable action hint for the Browsertools authoring handoff")
	browserAuthoringLogin := fs.String("browser-authoring-login", "", "Login posture for the handoff: required or not-required")
	browserAuthoringPrivateRoot := fs.String("browser-authoring-private-root", "", "Private Browsertools work/cache root outside the OpenUdon example")
	fs.Var(&browserAuthoringOriginFlags, "browser-authoring-origin", "Exact origin approved for the Browsertools authoring handoff; repeat for multiple origins")
	fs.Var(&sourceRootFlags, "source-root", "Explicit bounded local source root; repeat for multiple roots")
	network := fs.String("network", "", "Remote lookup policy: never, ask, or allow")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: icot --example examples/<name> [--dir examples/<name>] [--force] [--yes] [--print] [--from-example examples/<seed>] [--answers session.yaml] [--api-source KIND:ID=PATH] [--browser-profile ID=PATH] [--browser-verification PATH] [--browser-registry URL] [--source-root PATH] [--network never|ask|allow] [--prompt-mode full|normal|fast]\n")
		fmt.Fprintf(fs.Output(), "\nInteractively writes project.md and workflows/intent.hcl with the standard OpenUdon authoring sections.\n")
		fmt.Fprintf(fs.Output(), "It also creates selected source directories such as openapi/, browser-profiles/, and browser-authentication/, plus workflows/ and expected/, when missing.\n")
		fmt.Fprintf(fs.Output(), "\nPipeline: local source inspection -> active workflow boundary -> dependency frontier rounds -> complete proposal -> explicit approval.\n")
		fmt.Fprintf(fs.Output(), "\nSubcommands:\n")
		fmt.Fprintf(fs.Output(), "  icot reconcile --example examples/<name>  Regenerate project.md from workflows/intent.hcl.\n")
		fmt.Fprintf(fs.Output(), "  icot lint --example examples/<name>       Check project.md quality, intent parseability, and drift.\n")
		fmt.Fprintf(fs.Output(), "  icot scorecard --root examples/eval      Run the provider-free iCoT reliability scorecard.\n")
		fmt.Fprintf(fs.Output(), "  icot variants validate --root examples/eval Validate authoring variant metadata.\n")
		fmt.Fprintf(fs.Output(), "  icot variants coverage --root examples/eval Check provider-family variant class coverage.\n")
		fmt.Fprintf(fs.Output(), "  icot repair --example examples/<name>    Apply bounded mapping/output/dependency repairs.\n")
		fmt.Fprintf(fs.Output(), "  icot replay-eval --root examples/eval    Replay eval references through the iCoT chat loop.\n")
		fmt.Fprintf(fs.Output(), "  icot authoring-eval --root examples/eval Run optional real-LLM natural-language authoring evidence.\n")
		fmt.Fprintf(fs.Output(), "  icot report verify --file report.json    Verify scorecard or authoring-eval report JSON and digest.\n")
		fmt.Fprintf(fs.Output(), "  icot ui --example examples/<name>       Serve the read-only loopback status shell and experimental API v2.\n")
		fmt.Fprintf(fs.Output(), "  icot browser-author live ...             Run disclosure-gated authenticated Chromium authoring.\n")
		fmt.Fprintf(fs.Output(), "  icot browser-authoring plan ...          Emit a non-executing Browsertools authoring handoff.\n")
		fmt.Fprintf(fs.Output(), "\nSee docs/icot.md, docs/icot-session-schema.md, and docs/icot-transcript.md for file formats.\n")
		fmt.Fprintf(fs.Output(), "Next step: openudon build --example examples/<name>\n\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	defaultMode, err := promptDefaultMode(*promptMode)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 2
	}
	if *agentMode {
		defaultMode = authoring.PromptDefaultsSilent
		*promptMode = "fast"
	}
	localSources, err := parseLocalSourceFlags(apiSourceFlags, openAPIFlags)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 2
	}
	browserSources, err := parseBrowserSourceFlags(browserProfileFlags)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 2
	}
	browserAuthoringInput := browserAuthoringPlanInput{
		ExampleDir: exampleDirForPlan(firstNonEmpty(*example, *dirAlias)),
		TargetURL:  *browserAuthoringURL, Origins: append([]string(nil), browserAuthoringOriginFlags...),
		ProfileID: *browserAuthoringID, ActionHint: *browserAuthoringAction,
		LoginState: *browserAuthoringLogin, PrivateRoot: *browserAuthoringPrivateRoot,
	}
	if browserAuthoringInputProvided(browserAuthoringInput) && !*agentMode {
		fmt.Fprintln(errOut, "browser authoring flags are available only with --agent; use `icot browser-authoring plan` for an operator handoff")
		return 2
	}
	browserAuthoring, err := optionalBrowserAuthoringPlan(browserAuthoringInput)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 2
	}
	networkPolicy, err := resolveNetworkPolicy(*network, *agentMode)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 2
	}

	exampleDir := firstNonEmpty(*example, *dirAlias)
	if exampleDir == "" {
		fmt.Fprintln(errOut, "--example is required")
		return 2
	}
	if *agentMode {
		return runAgentAuthor(agentAuthorOptions{
			ExampleDir:           exampleDir,
			DirAlias:             *dirAlias,
			Force:                *force,
			Yes:                  *yes,
			FromExample:          *fromExample,
			AnswersFile:          *answersFile,
			NoTranscript:         *noTranscript,
			JSONOutput:           *jsonOutput,
			ReportPath:           *reportPath,
			PromptMode:           *promptMode,
			DefaultMode:          defaultMode,
			ReviewRepair:         *reviewRepair,
			NoLLM:                *noLLM,
			Provider:             *provider,
			Model:                *model,
			Temperature:          *temperature,
			OriginalInput:        in,
			LocalSources:         localSources,
			BrowserSources:       browserSources,
			BrowserVerifications: append([]string(nil), browserVerificationFlags...),
			BrowserRegistries:    append([]string(nil), browserRegistryFlags...),
			SourceRoots:          append([]string(nil), sourceRootFlags...),
			NetworkPolicy:        networkPolicy,
			BrowserAuthoring:     browserAuthoring,
		}, out, errOut)
	}
	projectPath := filepath.Join(exampleDir, "project.md")
	intentPath := filepath.Join(exampleDir, "workflows", "intent.hcl")
	input := bufio.NewReader(in)

	draftPath := ""
	loadDraft := strings.TrimSpace(*answersFile) == "" && strings.TrimSpace(*fromExample) == ""
	seed, source, err := authorSession(*answersFile, *fromExample, exampleDir, *force, loadDraft)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	if loadDraft {
		draftPath = elicitor.DraftPath(exampleDir)
		if source == seedSourceDraft {
			fmt.Fprintf(out, "icot: resumed draft %s\n", draftPath)
		}
	}
	if *printOnly {
		draftPath = ""
	}
	transcriptPath := ""
	if !*printOnly && !*noTranscript {
		transcriptPath = filepath.Join(exampleDir, ".icot", "transcript.json")
	}
	statusOut := out
	if defaultMode == authoring.PromptDefaultsSilent {
		statusOut = io.Discard
	}
	extractor, usingLLM := resolveExtractor(*noLLM, *provider, *model, *temperature, statusOut)
	if !usingLLM {
		fmt.Fprintln(statusOut, "icot: running without LLM extraction; continuing with manual slot filling")
	}
	authorSourceRoots := append([]string(nil), sourceRootFlags...)
	if strings.TrimSpace(*fromExample) != "" && filepath.Clean(*fromExample) != filepath.Clean(exampleDir) {
		authorSourceRoots = appendSeedSourceRoots(authorSourceRoots, *fromExample)
	}
	var artifacts elicitor.Artifacts
	complete := completeSession(seed)
	if complete {
		discovery, discoveryErr := elicitor.DiscoverAuthoringSourcesWithBrowser(context.Background(), exampleDir, projectwizard.Render(seed.Project), localSources, authorSourceRoots, browserSources, time.Now().UTC())
		if discoveryErr != nil {
			fmt.Fprintln(errOut, discoveryErr)
			return 1
		}
		if discovery.Report.Truncated || len(discovery.Report.Ambiguous) > 0 {
			fmt.Fprintln(errOut, "local source discovery is incomplete; narrow --source-root or declare ambiguous files with --api-source KIND:ID=PATH")
			return 1
		}
		if len(discovery.BrowserReport.Truncated) > 0 || len(discovery.BrowserReport.Ambiguous) > 0 {
			fmt.Fprintln(errOut, "browser source discovery is incomplete; narrow --source-root or declare the profile with --browser-profile ID=PATH")
			return 1
		}
		if len(browserRegistryFlags) > 0 && (len(discovery.Docs) == 0 || seed.BrowserRoute == "browser" || sessionUsesBrowserRegistry(seed)) {
			registryDiscovery, registryErr := elicitor.DiscoverBrowserRegistrySources(context.Background(), browserRegistryFlags, seed.Boundary.Outcome, networkPolicy, browserRegistryLookupApproved(seed, networkPolicy), time.Now().UTC())
			if registryErr != nil {
				fmt.Fprintln(errOut, registryErr)
				return 1
			}
			discovery = elicitor.MergeBrowserRegistrySources(discovery, registryDiscovery)
			if sessionUsesBrowserRegistry(seed) && len(registryDiscovery.Plans) == 0 {
				fmt.Fprintln(errOut, "selected browser registry profile could not be revalidated; use --network allow for HTTPS registries or provide the profile with --browser-profile")
				return 1
			}
		}
		seed.SourcePlan = elicitor.SyncSelectedSourcePlansWithBrowser(seed, discovery.Plans, localSources, browserSources)
		seed.SourcePlan, err = elicitor.AttachBrowserVerifications(seed.SourcePlan, browserVerificationFlags, time.Now().UTC())
		if err != nil {
			fmt.Fprintln(errOut, err)
			return 1
		}
	}
	if complete && (source != seedSourceDraft || *printOnly) {
		artifacts, err = elicitor.RenderArtifacts(seed)
	} else {
		artifacts, err = elicitor.Run(context.Background(), input, out, seed, elicitor.Options{
			ExampleDir:           exampleDir,
			NoLLM:                *noLLM || !usingLLM,
			Extractor:            extractor,
			DraftPath:            draftPath,
			TranscriptPath:       transcriptPath,
			DisableAIDraft:       source == seedSourceDraft,
			VerifyOnly:           complete && source == seedSourceDraft,
			DefaultMode:          defaultMode,
			ReviewRepair:         *reviewRepair,
			LocalSources:         localSources,
			BrowserSources:       browserSources,
			BrowserVerifications: append([]string(nil), browserVerificationFlags...),
			BrowserRegistries:    append([]string(nil), browserRegistryFlags...),
			SourceRoots:          authorSourceRoots,
			NetworkPolicy:        networkPolicy,
			AutoApprove:          *yes,
		})
	}
	if *printOnly {
		if err != nil {
			fmt.Fprintln(errOut, err)
			return 1
		}
		printArtifacts(out, artifacts)
		return 0
	}
	if err == nil && complete && source != seedSourceDraft {
		fmt.Fprintln(out, "\n----- complete authoring proposal -----")
		elicitor.PrintProposal(out, artifacts)
		if !*yes {
			approved, approvalErr := confirmProposalApproval(input, out)
			if approvalErr != nil {
				fmt.Fprintln(errOut, approvalErr)
				return 1
			}
			if !approved {
				return 0
			}
		}
	}
	if errors.Is(err, elicitor.ErrCanceled) {
		if deleteErr := elicitor.DeleteDraft(draftPath); deleteErr != nil {
			fmt.Fprintln(errOut, deleteErr)
			return 1
		}
		return 0
	}
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	artifacts.Session.Boundary.Confirmed = true
	if err := writeApprovedArtifacts(exampleDir, artifacts, *force, *yes, input, out); err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	if !artifacts.Incomplete {
		if err := elicitor.DeleteDraft(draftPath); err != nil {
			fmt.Fprintln(errOut, err)
			return 1
		}
	}
	fmt.Fprintf(out, "icot: wrote %s\n", projectPath)
	if artifacts.Incomplete {
		fmt.Fprintf(out, "icot: wrote %s\n", filepath.Join(exampleDir, "workflows", "intent.draft.hcl"))
	} else {
		fmt.Fprintf(out, "icot: wrote %s\n", intentPath)
	}
	if transcriptPath != "" {
		fmt.Fprintf(out, "icot: wrote %s\n", transcriptPath)
	}
	fmt.Fprintf(out, "next: openudon build --example %s\n", exampleDir)
	return 0
}

func confirmProposalApproval(in io.Reader, out io.Writer) (bool, error) {
	reader, ok := in.(*bufio.Reader)
	if !ok {
		reader = bufio.NewReader(in)
	}
	for {
		fmt.Fprint(out, "Type approve to write the proposal, or cancel: ")
		value, err := reader.ReadString('\n')
		value = strings.ToLower(strings.TrimSpace(value))
		switch value {
		case "approve":
			return true, nil
		case "cancel", "q", "quit":
			return false, nil
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return false, io.ErrUnexpectedEOF
			}
			return false, err
		}
		fmt.Fprintln(out, "Please type approve or cancel.")
	}
}

type agentAuthorOptions struct {
	ExampleDir           string
	DirAlias             string
	Force                bool
	Yes                  bool
	FromExample          string
	AnswersFile          string
	NoTranscript         bool
	JSONOutput           bool
	ReportPath           string
	PromptMode           string
	DefaultMode          authoring.PromptDefaultMode
	ReviewRepair         bool
	NoLLM                bool
	Provider             string
	Model                string
	Temperature          float64
	OriginalInput        io.Reader
	LocalSources         []apitools.LocalSource
	BrowserSources       []elicitor.BrowserSourceInput
	BrowserVerifications []string
	BrowserRegistries    []string
	SourceRoots          []string
	NetworkPolicy        string
	BrowserAuthoring     *browserAuthoringPlan
}

func runAgentAuthor(opts agentAuthorOptions, out, errOut io.Writer) int {
	exampleDir := strings.TrimSpace(opts.ExampleDir)
	projectPath := filepath.Join(exampleDir, "project.md")
	intentPath := filepath.Join(exampleDir, "workflows", "intent.hcl")
	report := authorReport{
		Version:     authorReportVersion,
		Status:      statusNeedsInput,
		Example:     exampleDir,
		ProjectPath: projectPath,
		IntentPath:  intentPath,
	}
	loadDraft := strings.TrimSpace(opts.AnswersFile) == "" && strings.TrimSpace(opts.FromExample) == ""
	seed, source, err := authorSession(opts.AnswersFile, opts.FromExample, exampleDir, opts.Force, loadDraft)
	if err != nil {
		report.Status = statusFail
		report.Error = err.Error()
		report.FailureFamily = failureUnknown
		if reportErr := writeAuthorReport(report, opts, out); reportErr != nil {
			fmt.Fprintln(errOut, reportErr)
		}
		fmt.Fprintln(errOut, err)
		return 1
	}
	projectText := agentProjectText(projectPath, seed)
	discoveryRoots := append([]string(nil), opts.SourceRoots...)
	if strings.TrimSpace(opts.FromExample) != "" && filepath.Clean(opts.FromExample) != filepath.Clean(exampleDir) {
		discoveryRoots = appendSeedSourceRoots(discoveryRoots, opts.FromExample)
	}
	discovery, discoveryErr := elicitor.DiscoverAuthoringSourcesWithBrowser(context.Background(), exampleDir, projectText, opts.LocalSources, discoveryRoots, opts.BrowserSources, time.Now().UTC())
	if discoveryErr != nil {
		report.Status = statusFail
		report.Error = discoveryErr.Error()
		report.FailureFamily = failureMissingAPISource
		_ = writeAuthorReport(report, opts, out)
		fmt.Fprintln(errOut, discoveryErr)
		return 1
	}
	registryDiscovery := elicitor.BrowserRegistryDiscovery{Candidates: []elicitor.BrowserRegistryCandidate{}, Blockers: []elicitor.BrowserRegistryBlocker{}}
	if len(opts.BrowserRegistries) > 0 && (len(discovery.Docs) == 0 || seed.BrowserRoute == "browser") {
		registryDiscovery, err = elicitor.DiscoverBrowserRegistrySources(context.Background(), opts.BrowserRegistries, firstNonEmpty(seed.Boundary.Outcome, projectText), opts.NetworkPolicy, opts.NetworkPolicy == "allow", time.Now().UTC())
		if err != nil {
			report.Status = statusFail
			report.Error = err.Error()
			report.FailureFamily = failureMissingAPISource
			_ = writeAuthorReport(report, opts, out)
			fmt.Fprintln(errOut, err)
			return 1
		}
		discovery = elicitor.MergeBrowserRegistrySources(discovery, registryDiscovery)
	}
	docs := discovery.Docs
	seed.SourcePlan = elicitor.SyncSelectedSourcePlansWithBrowser(seed, discovery.Plans, opts.LocalSources, opts.BrowserSources)
	seed.SourcePlan, err = elicitor.AttachBrowserVerifications(seed.SourcePlan, opts.BrowserVerifications, time.Now().UTC())
	if err != nil {
		report.Status = statusFail
		report.Error = err.Error()
		report.FailureFamily = failureMissingAPISource
		_ = writeAuthorReport(report, opts, out)
		fmt.Fprintln(errOut, err)
		return 1
	}
	if seed.Interview.Metadata == nil {
		seed.Interview.Metadata = map[string]string{}
	}
	seed.Interview.Metadata["network_policy"] = opts.NetworkPolicy
	if len(opts.BrowserRegistries) > 0 {
		seed.Interview.Metadata["browser_registry_configured"] = "true"
	}
	if len(docs) == 0 && seed.Intent.RequiresOpenAPI() {
		remote, remoteErr := elicitor.DiscoverRemoteSourceHints(context.Background(), seed.Boundary.Outcome, elicitor.RemoteSourceLookupOptions{Policy: opts.NetworkPolicy, Approved: opts.NetworkPolicy == "allow"})
		if remoteErr != nil {
			report.Status = statusFail
			report.Error = remoteErr.Error()
			report.FailureFamily = failureMissingAPISource
			_ = writeAuthorReport(report, opts, out)
			fmt.Fprintln(errOut, remoteErr)
			return 1
		}
		report.RemoteCandidates = remote.Candidates
		report.RemoteBlocker = remote.Blocker
	}
	issues := elicitor.CheckReadiness(seed, docs)
	if browserAuthoringSourceGap(issues) {
		report.BrowserAuthoring = cloneBrowserAuthoringPlan(opts.BrowserAuthoring)
	}
	frontier, frontierErr := elicitor.PlanFrontier(&seed, docs, issues)
	if frontierErr != nil {
		report.Status = statusFail
		report.Error = frontierErr.Error()
		report.FailureFamily = failureAmbiguousUserIntent
		_ = writeAuthorReport(report, opts, out)
		fmt.Fprintln(errOut, frontierErr)
		return 1
	}
	report.ReadinessIssues = issues
	report.Boundary = seed.Boundary
	report.InterviewVersion = seed.Interview.Version
	report.Frontier = frontier
	report.CandidateWorkflows = append([]elicitor.CandidateWorkflow(nil), seed.CandidateWorkflows...)
	report.SourceCandidates = discovery.Report.Candidates
	report.SourceRejected = discovery.Report.Rejected
	report.SourceAmbiguous = discovery.Report.Ambiguous
	report.SourceDiagnostics = discovery.Report.Diagnostics
	report.BrowserSourceCandidates = discovery.BrowserReport.Candidates
	report.BrowserSourceRejected = discovery.BrowserReport.Rejected
	report.BrowserSourceAmbiguous = discovery.BrowserReport.Ambiguous
	report.BrowserSourceTruncated = discovery.BrowserReport.Truncated
	report.BrowserRegistryCandidates = registryDiscovery.Candidates
	report.BrowserRegistryBlockers = registryDiscovery.Blockers
	report.ProposedFileActions = proposedAuthorFileActions(exampleDir, seed, topBlockingReadinessIssue(issues) == nil)
	if discovery.Report.Truncated || len(discovery.Report.Ambiguous) > 0 {
		issue := elicitor.ReadinessIssue{Code: "source_discovery_blocked", Severity: "blocking", Slot: "source.selection", Message: "Local source discovery is incomplete; narrow roots or declare ambiguous documents with --api-source KIND:ID=PATH."}
		report.ReadinessIssues = append([]elicitor.ReadinessIssue{issue}, report.ReadinessIssues...)
		report.TopIssue = &issue
		report.FailureFamily = failureMissingAPISource
		if err := writeAuthorReport(report, opts, out); err != nil {
			fmt.Fprintln(errOut, err)
			return 1
		}
		return 0
	}
	if len(discovery.BrowserReport.Truncated) > 0 || len(discovery.BrowserReport.Ambiguous) > 0 {
		issue := elicitor.ReadinessIssue{Code: "browser_source_discovery_blocked", Severity: "blocking", Slot: "source.browser", Message: "Browser source discovery is incomplete; narrow roots or declare a verified profile with --browser-profile ID=PATH."}
		report.ReadinessIssues = append([]elicitor.ReadinessIssue{issue}, report.ReadinessIssues...)
		report.TopIssue = &issue
		report.FailureFamily = failureMissingAPISource
		if err := writeAuthorReport(report, opts, out); err != nil {
			fmt.Fprintln(errOut, err)
			return 1
		}
		return 0
	}
	if top := topBlockingReadinessIssue(issues); top != nil {
		report.TopIssue = top
		report.SuggestedAnswer = agentSuggestedAnswer(top)
		report.FailureFamily = failureFamilyForReadiness(top.Code)
		if err := writeAuthorReport(report, opts, out); err != nil {
			fmt.Fprintln(errOut, err)
			return 1
		}
		if !opts.JSONOutput {
			fmt.Fprintf(out, "icot: needs input: %s\n", report.TopIssue.Message)
			if report.SuggestedAnswer != "" {
				fmt.Fprintf(out, "suggested answer: %s\n", report.SuggestedAnswer)
			}
		}
		return 0
	}
	_, err = elicitor.RenderArtifacts(seed)
	if err != nil {
		report.Status = statusFail
		report.Error = err.Error()
		report.FailureFamily = failureIntentParse
		if err := writeAuthorReport(report, opts, out); err != nil {
			fmt.Fprintln(errOut, err)
		}
		fmt.Fprintln(errOut, err)
		return 1
	}
	_ = source
	report.Status = statusNeedsInput
	report.SuggestedAnswer = "Approve the complete proposal in an interactive run before writing deliverables."
	approvalIssue := elicitor.ReadinessIssue{Code: "proposal_approval_required", Severity: "blocking", Slot: elicitor.FinalApprovalNodeID(), Message: report.SuggestedAnswer, SuggestedAnswer: "approve"}
	report.ReadinessIssues = append(report.ReadinessIssues, approvalIssue)
	report.TopIssue = &approvalIssue
	if err := writeAuthorReport(report, opts, out); err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	if !opts.JSONOutput {
		fmt.Fprintln(out, "icot: complete proposal ready for approval; agent mode did not write deliverables")
	}
	return 0
}

func agentSuggestedAnswer(issue *elicitor.ReadinessIssue) string {
	if issue == nil {
		return ""
	}
	if strings.TrimSpace(issue.SuggestedAnswer) != "" {
		return issue.SuggestedAnswer
	}
	switch issue.Code {
	case "missing_goal":
		return "Describe the workflow goal and expected output."
	default:
		return ""
	}
}

func browserAuthoringSourceGap(issues []elicitor.ReadinessIssue) bool {
	for _, issue := range issues {
		if issue.Code == "missing_api_doc" && strings.EqualFold(issue.Severity, "blocking") {
			return true
		}
	}
	return false
}

func agentProjectText(projectPath string, seed elicitor.Session) string {
	if data, err := os.ReadFile(projectPath); err == nil {
		return string(data)
	}
	if strings.TrimSpace(seed.Project.Goal) != "" || strings.TrimSpace(seed.Project.ProjectName) != "" {
		return projectwizard.Render(seed.Project)
	}
	return ""
}

func proposedAuthorFileActions(exampleDir string, session elicitor.Session, complete bool) []elicitor.FileAction {
	return artifactwriter.PotentialFileActions(exampleDir, session, complete)
}

func topBlockingReadinessIssue(issues []elicitor.ReadinessIssue) *elicitor.ReadinessIssue {
	for i := range issues {
		if strings.EqualFold(issues[i].Severity, "blocking") {
			return &issues[i]
		}
	}
	return nil
}

func writeAuthorReport(report authorReport, opts agentAuthorOptions, out io.Writer) error {
	if err := validateAuthorReportContract(report); err != nil {
		return err
	}
	if strings.TrimSpace(opts.ReportPath) != "" {
		if err := writeJSONFile(opts.ReportPath, report); err != nil {
			return err
		}
	}
	if opts.JSONOutput {
		if err := json.NewEncoder(out).Encode(report); err != nil {
			return err
		}
	}
	return nil
}

func promptDefaultMode(mode string) (authoring.PromptDefaultMode, error) {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "", "full":
		return authoring.PromptDefaultsAsk, nil
	case "normal":
		return authoring.PromptDefaultsShow, nil
	case "fast":
		return authoring.PromptDefaultsSilent, nil
	default:
		return authoring.PromptDefaultsAsk, fmt.Errorf("--prompt-mode must be full, normal, or fast")
	}
}

type repeatedFlag []string

func (values *repeatedFlag) String() string { return strings.Join(*values, ",") }

func (values *repeatedFlag) Set(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return errors.New("flag value may not be empty")
	}
	*values = append(*values, value)
	return nil
}

func parseLocalSourceFlags(apiSources, openAPIs []string) ([]apitools.LocalSource, error) {
	values := make([]string, 0, len(apiSources)+len(openAPIs))
	values = append(values, apiSources...)
	for _, value := range openAPIs {
		values = append(values, "openapi:"+value)
	}
	out := make([]apitools.LocalSource, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		kindAndRest := strings.SplitN(value, ":", 2)
		if len(kindAndRest) != 2 || strings.TrimSpace(kindAndRest[0]) == "" {
			return nil, fmt.Errorf("invalid --api-source %q; use KIND:ID=PATH", value)
		}
		idAndPath := strings.SplitN(kindAndRest[1], "=", 2)
		if len(idAndPath) != 2 || strings.TrimSpace(idAndPath[0]) == "" || strings.TrimSpace(idAndPath[1]) == "" {
			return nil, fmt.Errorf("invalid API source %q; use KIND:ID=PATH", value)
		}
		source := apitools.LocalSource{Kind: strings.TrimSpace(kindAndRest[0]), ID: strings.TrimSpace(idAndPath[0]), Path: strings.TrimSpace(idAndPath[1])}
		key := strings.ToLower(source.Kind) + "\x00" + source.ID + "\x00" + filepath.Clean(source.Path)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, source)
	}
	return out, nil
}

func parseBrowserSourceFlags(values []string) ([]elicitor.BrowserSourceInput, error) {
	out := make([]elicitor.BrowserSourceInput, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		idAndPath := strings.SplitN(value, "=", 2)
		if len(idAndPath) != 2 || strings.TrimSpace(idAndPath[0]) == "" || strings.TrimSpace(idAndPath[1]) == "" {
			return nil, fmt.Errorf("invalid --browser-profile %q; use ID=PATH", value)
		}
		source := elicitor.BrowserSourceInput{ID: strings.TrimSpace(idAndPath[0]), Path: strings.TrimSpace(idAndPath[1])}
		key := source.ID + "\x00" + filepath.Clean(source.Path)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, source)
	}
	return out, nil
}

func resolveNetworkPolicy(value string, agent bool) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		if agent {
			return "never", nil
		}
		return "ask", nil
	}
	switch value {
	case "never", "ask", "allow":
		if agent && value != "allow" {
			return "never", nil
		}
		return value, nil
	default:
		return "", fmt.Errorf("--network must be never, ask, or allow")
	}
}

func appendSeedSourceRoots(roots []string, seedDir string) []string {
	for _, dir := range sourcecatalog.All() {
		path := filepath.Join(seedDir, dir)
		if info, err := os.Lstat(path); err == nil && info.IsDir() && info.Mode()&os.ModeSymlink == 0 {
			roots = append(roots, path)
		}
	}
	return roots
}

func sessionUsesBrowserRegistry(session elicitor.Session) bool {
	for _, source := range session.SourcePlan {
		if source.Kind == "browser-profile" && strings.TrimSpace(source.Registry) != "" {
			return true
		}
	}
	return false
}

func browserRegistryLookupApproved(session elicitor.Session, networkPolicy string) bool {
	return strings.EqualFold(strings.TrimSpace(networkPolicy), "allow") ||
		strings.EqualFold(strings.TrimSpace(session.Interview.Metadata["browser_registry_lookup_decision"]), "allow")
}
