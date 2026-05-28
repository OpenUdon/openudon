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
	"sort"
	"strings"
	"time"

	"github.com/OpenUdon/openudon/internal/authoring"
	evalpkg "github.com/OpenUdon/openudon/internal/eval"
	"github.com/OpenUdon/openudon/internal/icot/elicitor"
	"github.com/OpenUdon/openudon/internal/projectwizard"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
	runner "github.com/OpenUdon/openudon/internal/workflowintent"
	"gopkg.in/yaml.v3"
)

func Main(args []string, in io.Reader, out, errOut io.Writer) int {
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
	yes := fs.Bool("yes", false, "Accept overwrite prompts without asking")
	printOnly := fs.Bool("print", false, "Render project.md and workflows/intent.hcl to stdout without writing files")
	fromExample := fs.String("from-example", "", "Seed answers from an existing example directory")
	answersFile := fs.String("answers", "", "Path to YAML or JSON session/answers file; suppresses interactive prompts when complete")
	noLLM := fs.Bool("no-llm", false, "Disable optional LLM extraction assistance")
	noTranscript := fs.Bool("no-transcript", false, "Do not save local .icot transcript history")
	promptMode := fs.String("prompt-mode", "full", "Prompt mode: full, normal, or fast. full asks every question; normal accepts defaults visibly; fast skips defaulted questions")
	reviewRepair := fs.Bool("review-repair", false, "Experimental: apply up to two bounded repairs from pre-final flow review suggestions")
	agentMode := fs.Bool("agent", false, "Run noninteractively and return needs_input instead of prompting when authoring is incomplete")
	jsonOutput := fs.Bool("json", false, "Write a structured JSON report to stdout")
	reportPath := fs.String("report", "", "Write a structured JSON report to this path")
	provider := fs.String("provider", "", "LLM provider for optional extraction: copilot-api, openai, anthropic, or gemini")
	model := fs.String("model", "", "LLM model for optional extraction")
	temperature := fs.Float64("temperature", 0.2, "LLM extraction temperature")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: icot --example examples/<name> [--dir examples/<name>] [--force] [--yes] [--print] [--from-example examples/<seed>] [--answers answers.yaml] [--prompt-mode full|normal|fast]\n")
		fmt.Fprintf(fs.Output(), "\nInteractively writes project.md and workflows/intent.hcl with the standard OpenUdon authoring sections.\n")
		fmt.Fprintf(fs.Output(), "It also creates openapi/, workflows/, and expected/ when missing.\n")
		fmt.Fprintf(fs.Output(), "\nPipeline: goal -> catalog plan -> API source retrieval -> operation selection -> request mappings -> draft -> advisory flow review -> final confirmation.\n")
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

	exampleDir := firstNonEmpty(*example, *dirAlias)
	if exampleDir == "" {
		fmt.Fprintln(errOut, "--example is required")
		return 2
	}
	if *agentMode {
		return runAgentAuthor(agentAuthorOptions{
			ExampleDir:    exampleDir,
			DirAlias:      *dirAlias,
			Force:         *force,
			Yes:           *yes,
			FromExample:   *fromExample,
			AnswersFile:   *answersFile,
			NoTranscript:  *noTranscript,
			JSONOutput:    *jsonOutput,
			ReportPath:    *reportPath,
			PromptMode:    *promptMode,
			DefaultMode:   defaultMode,
			ReviewRepair:  *reviewRepair,
			NoLLM:         *noLLM,
			Provider:      *provider,
			Model:         *model,
			Temperature:   *temperature,
			OriginalInput: in,
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
	var artifacts elicitor.Artifacts
	complete := completeSession(seed)
	if complete && (source != seedSourceDraft || *printOnly) {
		artifacts, err = elicitor.RenderArtifacts(seed)
	} else {
		artifacts, err = elicitor.Run(context.Background(), input, out, seed, elicitor.Options{
			ExampleDir:     exampleDir,
			NoLLM:          *noLLM || !usingLLM,
			Extractor:      extractor,
			DraftPath:      draftPath,
			TranscriptPath: transcriptPath,
			DisableAIDraft: source == seedSourceDraft,
			VerifyOnly:     complete && source == seedSourceDraft,
			DefaultMode:    defaultMode,
			ReviewRepair:   *reviewRepair,
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
	if err := writeArtifacts(projectPath, intentPath, artifacts, *force, *yes, input, out); err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	if err := copySeedSourceArtifacts(*fromExample, exampleDir, *force); err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	if err := elicitor.DeleteDraft(draftPath); err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	fmt.Fprintf(out, "icot: wrote %s\n", projectPath)
	fmt.Fprintf(out, "icot: wrote %s\n", intentPath)
	if transcriptPath != "" {
		fmt.Fprintf(out, "icot: wrote %s\n", transcriptPath)
	}
	fmt.Fprintf(out, "next: openudon build --example %s\n", exampleDir)
	return 0
}

type agentAuthorOptions struct {
	ExampleDir    string
	DirAlias      string
	Force         bool
	Yes           bool
	FromExample   string
	AnswersFile   string
	NoTranscript  bool
	JSONOutput    bool
	ReportPath    string
	PromptMode    string
	DefaultMode   authoring.PromptDefaultMode
	ReviewRepair  bool
	NoLLM         bool
	Provider      string
	Model         string
	Temperature   float64
	OriginalInput io.Reader
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
	docs := agentReadinessDocs(exampleDir, opts.FromExample, projectText)
	issues := elicitor.CheckReadiness(seed, docs)
	report.ReadinessIssues = issues
	if top := topBlockingReadinessIssue(issues); top != nil {
		report.TopIssue = top
		report.SuggestedAnswer = top.SuggestedAnswer
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
	artifacts, err := elicitor.RenderArtifacts(seed)
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
	if err := writeArtifacts(projectPath, intentPath, artifacts, opts.Force, true, strings.NewReader(""), io.Discard); err != nil {
		report.Status = statusFail
		report.Error = err.Error()
		report.FailureFamily = failureIntentParse
		if err := writeAuthorReport(report, opts, out); err != nil {
			fmt.Fprintln(errOut, err)
		}
		fmt.Fprintln(errOut, err)
		return 1
	}
	if err := copySeedSourceArtifacts(opts.FromExample, exampleDir, opts.Force); err != nil {
		report.Status = statusFail
		report.Error = err.Error()
		report.FailureFamily = failureUnknown
		if err := writeAuthorReport(report, opts, out); err != nil {
			fmt.Fprintln(errOut, err)
		}
		fmt.Fprintln(errOut, err)
		return 1
	}
	if source == seedSourceDraft {
		if err := elicitor.DeleteDraft(elicitor.DraftPath(exampleDir)); err != nil {
			report.Status = statusFail
			report.Error = err.Error()
			report.FailureFamily = failureUnknown
			if reportErr := writeAuthorReport(report, opts, out); reportErr != nil {
				fmt.Fprintln(errOut, reportErr)
			}
			fmt.Fprintln(errOut, err)
			return 1
		}
	}
	report.Status = statusPass
	report.GeneratedProject = projectPath
	report.GeneratedIntent = intentPath
	if err := writeAuthorReport(report, opts, out); err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	if !opts.JSONOutput {
		fmt.Fprintf(out, "icot: wrote %s\n", projectPath)
		fmt.Fprintf(out, "icot: wrote %s\n", intentPath)
	}
	return 0
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

func agentReadinessDocs(exampleDir, fromExample, projectText string) []elicitor.APIDocument {
	docs, _ := elicitor.DiscoverLocalAPIs(exampleDir, projectText)
	seedDir := strings.TrimSpace(fromExample)
	if seedDir == "" || filepath.Clean(seedDir) == filepath.Clean(exampleDir) {
		return docs
	}
	seedDocs, _ := elicitor.DiscoverLocalAPIs(seedDir, projectText)
	return append(docs, seedDocs...)
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

func runReconcile(args []string, in io.Reader, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("icot reconcile", flag.ContinueOnError)
	fs.SetOutput(out)
	example := fs.String("example", "", "Example directory containing workflows/intent.hcl")
	yes := fs.Bool("yes", false, "Overwrite project.md without asking")
	printOnly := fs.Bool("print", false, "Print regenerated project.md without writing files")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: icot reconcile --example examples/<name> [--print] [--yes]\n\n")
		fmt.Fprintf(fs.Output(), "Regenerates project.md from workflows/intent.hcl while preserving existing project policy text.\n\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	exampleDir := strings.TrimSpace(*example)
	if exampleDir == "" {
		fmt.Fprintln(errOut, "--example is required")
		return 2
	}
	projectPath := filepath.Join(exampleDir, "project.md")
	intentPath := filepath.Join(exampleDir, "workflows", "intent.hcl")
	intent, err := rollout.ParseIntentFile(intentPath)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	project := projectwizard.Answers{}
	if data, err := os.ReadFile(projectPath); err == nil {
		project, err = projectwizard.LoadAnswersFromMarkdown(string(data))
		if err != nil {
			fmt.Fprintln(errOut, err)
			return 1
		}
	} else if !os.IsNotExist(err) {
		fmt.Fprintln(errOut, err)
		return 1
	}
	session := elicitor.SessionFromIntent(intent, project)
	artifacts, err := elicitor.RenderArtifacts(session)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	if *printOnly {
		fmt.Fprint(out, artifacts.ProjectMD)
		if !strings.HasSuffix(artifacts.ProjectMD, "\n") {
			fmt.Fprintln(out)
		}
		return 0
	}
	input := bufio.NewReader(in)
	if _, err := os.Stat(projectPath); err == nil && !*yes {
		ok, err := confirm(input, out, fmt.Sprintf("Overwrite %s?", projectPath), false)
		if err != nil {
			fmt.Fprintln(errOut, err)
			return 1
		}
		if !ok {
			return 0
		}
	} else if err != nil && !os.IsNotExist(err) {
		fmt.Fprintln(errOut, err)
		return 1
	}
	if err := writeGeneratedFilesAtomic([]generatedFile{{Path: projectPath, Content: artifacts.ProjectMD}}, true); err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	fmt.Fprintf(out, "icot: reconciled %s from %s\n", projectPath, intentPath)
	return 0
}

type replayEvalReport struct {
	Provider string             `json:"provider,omitempty"`
	Model    string             `json:"model,omitempty"`
	Root     string             `json:"root"`
	OutDir   string             `json:"out_dir"`
	Passed   bool               `json:"passed"`
	Results  []replayEvalResult `json:"results"`
}

type replayEvalResult struct {
	Name               string                 `json:"name"`
	Passed             bool                   `json:"passed"`
	Error              string                 `json:"error,omitempty"`
	ReferenceIssues    []evalpkg.CompareIssue `json:"reference_issues,omitempty"`
	Blocking           int                    `json:"blocking"`
	Warning            int                    `json:"warning"`
	Advisory           int                    `json:"advisory"`
	PromptMode         string                 `json:"prompt_mode,omitempty"`
	PromptCount        int                    `json:"prompt_count,omitempty"`
	AutoAccepted       int                    `json:"auto_accepted,omitempty"`
	LLMCallCount       int                    `json:"llm_call_count,omitempty"`
	RepairAttempts     int                    `json:"repair_attempts,omitempty"`
	RepairRejected     int                    `json:"repair_rejected,omitempty"`
	UnresolvedReview   int                    `json:"unresolved_review_warnings,omitempty"`
	TranscriptPath     string                 `json:"transcript_path,omitempty"`
	ICOTTranscriptPath string                 `json:"icot_transcript_path,omitempty"`
	StdoutPath         string                 `json:"stdout_path,omitempty"`
	GeneratedIntent    string                 `json:"generated_intent,omitempty"`
	GeneratedProject   string                 `json:"generated_project,omitempty"`
	LLMCalls           []replayLLMCall        `json:"llm_calls,omitempty"`
	Turns              []elicitor.ReplayTurn  `json:"turns,omitempty"`
}

type replayLLMCall struct {
	Kind     string                `json:"kind"`
	Messages []rollout.ChatMessage `json:"messages"`
	Response string                `json:"response,omitempty"`
	Error    string                `json:"error,omitempty"`
}

func runReplayEval(args []string, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("icot replay-eval", flag.ContinueOnError)
	fs.SetOutput(out)
	root := fs.String("root", "examples/eval", "Directory containing eval example subdirectories")
	name := fs.String("name", "", "Run a single eval fixture by directory name")
	provider := fs.String("provider", "copilot-api", "LLM provider for iCoT extraction")
	model := fs.String("model", "gpt-5.4-mini", "LLM model for iCoT extraction")
	temperature := fs.Float64("temperature", 0.2, "LLM extraction temperature")
	promptMode := fs.String("prompt-mode", "fast", "Prompt mode for replayed iCoT loop: full, normal, or fast")
	reviewRepair := fs.Bool("review-repair", false, "Enable experimental bounded review repair during replay")
	timeout := fs.Duration("timeout", 2*time.Minute, "Timeout per fixture replay")
	outDir := fs.String("out-dir", filepath.Join("eval", "runs", "icot-replay-"+time.Now().UTC().Format("20060102T150405Z")), "Directory for replay transcripts and generated artifacts")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: icot replay-eval [--root examples/eval] [--provider copilot-api --model gpt-5.4-mini] [--out-dir eval/runs/icot-replay-<ts>]\n\n")
		fmt.Fprintf(fs.Output(), "Replays eval reference intents through the real iCoT chat loop with LLM extraction enabled and writes transcripts.\n\n")
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
	fixtures := discoverReplayFixtures(*root, *name)
	if len(fixtures) == 0 {
		fmt.Fprintf(errOut, "no eval fixtures found under %s\n", *root)
		return 1
	}
	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	report := replayEvalReport{
		Provider: strings.TrimSpace(*provider),
		Model:    strings.TrimSpace(*model),
		Root:     *root,
		OutDir:   *outDir,
		Passed:   true,
	}
	for _, exampleDir := range fixtures {
		result := runReplayFixture(exampleDir, *provider, *model, *temperature, *timeout, *outDir, *promptMode, defaultMode, *reviewRepair)
		report.Results = append(report.Results, result)
		if !result.Passed {
			report.Passed = false
		}
		status := "pass"
		if !result.Passed {
			status = "fail"
		}
		fmt.Fprintf(out, "icot replay-eval: %s %s", status, result.Name)
		if result.Error != "" {
			fmt.Fprintf(out, " - %s", result.Error)
		}
		fmt.Fprintln(out)
	}
	reportPath := filepath.Join(*outDir, "report.json")
	if err := writeJSONFile(reportPath, report); err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	fmt.Fprintf(out, "icot replay-eval: wrote %s\n", reportPath)
	if !report.Passed {
		return 1
	}
	return 0
}

func discoverReplayFixtures(root, name string) []string {
	if strings.TrimSpace(name) != "" {
		path := filepath.Join(root, strings.TrimSpace(name))
		if _, err := os.Stat(filepath.Join(path, "reference", "intent.hcl")); err == nil {
			return []string{path}
		}
		return nil
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	var out []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(root, entry.Name())
		if _, err := os.Stat(filepath.Join(path, "reference", "intent.hcl")); err == nil {
			out = append(out, path)
		}
	}
	sort.Strings(out)
	return out
}

func runReplayFixture(exampleDir, provider, model string, temperature float64, timeout time.Duration, outDir, promptMode string, defaultMode authoring.PromptDefaultMode, reviewRepair bool) replayEvalResult {
	name := filepath.Base(filepath.Clean(exampleDir))
	fixtureDir := filepath.Join(outDir, name)
	result := replayEvalResult{Name: name, PromptMode: promptMode}
	if err := os.MkdirAll(fixtureDir, 0o755); err != nil {
		result.Error = err.Error()
		return result
	}
	referencePath := filepath.Join(exampleDir, "reference", "intent.hcl")
	reference, err := rollout.ParseIntentFile(referencePath)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	script, err := elicitor.BuildProgressiveReplayScript(exampleDir, reference)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	calls := []replayLLMCall{}
	extractor, err := replayExtractor(provider, model, temperature, &calls)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	var stdout strings.Builder
	icotTranscriptPath := filepath.Join(fixtureDir, "icot-transcript.json")
	artifacts, err := elicitor.Run(ctx, strings.NewReader(script.Input), &stdout, elicitor.Session{}, elicitor.Options{
		ExampleDir:     exampleDir,
		NoLLM:          false,
		Extractor:      extractor,
		DisableAIDraft: true,
		DefaultMode:    defaultMode,
		ReviewRepair:   reviewRepair,
		TranscriptPath: icotTranscriptPath,
	})
	result.Turns = script.Turns
	result.LLMCalls = calls
	result.LLMCallCount = len(calls)
	result.ICOTTranscriptPath = icotTranscriptPath
	if metrics := replayTranscriptMetrics(icotTranscriptPath, stdout.String()); metrics != nil {
		result.Turns = metrics.Turns
		result.PromptCount = len(metrics.Turns)
		result.AutoAccepted = metrics.AutoAccepted
		result.RepairAttempts = metrics.RepairAttempts
		result.RepairRejected = metrics.RepairRejected
		result.UnresolvedReview = metrics.UnresolvedReview
	}
	if writeErr := os.WriteFile(filepath.Join(fixtureDir, "stdout.txt"), []byte(stdout.String()), 0o644); writeErr == nil {
		result.StdoutPath = filepath.Join(fixtureDir, "stdout.txt")
	}
	if err != nil {
		result.Error = err.Error()
		_ = writeJSONFile(filepath.Join(fixtureDir, "transcript.json"), result)
		return result
	}
	if defaultMode == authoring.PromptDefaultsAsk {
		if labelErr := elicitor.AssertReplayLabelsInOrder(stdout.String(), result.Turns); labelErr != nil {
			result.Error = labelErr.Error()
		}
	}
	generatedDir := filepath.Join(fixtureDir, "generated")
	_ = os.MkdirAll(generatedDir, 0o755)
	intentPath := filepath.Join(generatedDir, "intent.hcl")
	projectPath := filepath.Join(generatedDir, "project.md")
	if err := os.WriteFile(intentPath, []byte(artifacts.IntentHCL), 0o644); err == nil {
		result.GeneratedIntent = intentPath
	}
	if err := os.WriteFile(projectPath, []byte(artifacts.ProjectMD), 0o644); err == nil {
		result.GeneratedProject = projectPath
	}
	policy, _ := evalpkg.ReadReferencePolicy(filepath.Join(exampleDir, "reference", "policy.json"))
	issues, compareErr := evalpkg.CompareIntentFiles(intentPath, referencePath, policy)
	if compareErr != nil {
		if result.Error == "" {
			result.Error = compareErr.Error()
		}
	} else {
		result.ReferenceIssues = issues
		for _, issue := range issues {
			switch issue.Severity {
			case "blocking":
				result.Blocking++
			case "warning":
				result.Warning++
			case "advisory":
				result.Advisory++
			}
		}
	}
	result.Passed = replayPassesPolicy(result, policy)
	transcriptPath := filepath.Join(fixtureDir, "transcript.json")
	if err := writeJSONFile(transcriptPath, result); err == nil {
		result.TranscriptPath = transcriptPath
		_ = writeJSONFile(transcriptPath, result)
	}
	return result
}

type replayMetrics struct {
	RepairAttempts   int
	RepairRejected   int
	UnresolvedReview int
	AutoAccepted     int
	Turns            []elicitor.ReplayTurn
}

func replayTranscriptMetrics(path, stdout string) *replayMetrics {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var transcript struct {
		Turns  []elicitor.ReplayTurn `json:"turns,omitempty"`
		Events []struct {
			Kind string          `json:"kind"`
			Data json.RawMessage `json:"data,omitempty"`
		} `json:"events,omitempty"`
	}
	if err := json.Unmarshal(data, &transcript); err != nil {
		return nil
	}
	metrics := &replayMetrics{Turns: append([]elicitor.ReplayTurn(nil), transcript.Turns...)}
	metrics.AutoAccepted = countAutoAcceptedTurns(stdout, transcript.Turns)
	for _, event := range transcript.Events {
		switch event.Kind {
		case "draft_repair_attempt":
			metrics.RepairAttempts++
		case "draft_repair_rejected":
			metrics.RepairRejected++
		case "draft_flow_review_result":
			var payload struct {
				Issues []any `json:"issues"`
			}
			if err := json.Unmarshal(event.Data, &payload); err == nil {
				metrics.UnresolvedReview = len(payload.Issues)
			}
		}
	}
	return metrics
}

func countAutoAcceptedTurns(stdout string, turns []elicitor.ReplayTurn) int {
	if len(turns) == 0 {
		return 0
	}
	offset := 0
	visible := 0
	for _, turn := range turns {
		label := strings.TrimSpace(turn.Label)
		if label == "" {
			continue
		}
		index := strings.Index(stdout[offset:], label)
		if index < 0 {
			continue
		}
		visible++
		offset += index + len(label)
	}
	auto := len(turns) - visible
	if auto < 0 {
		return 0
	}
	return auto
}

func replayPassesPolicy(result replayEvalResult, policy evalpkg.ReferencePolicy) bool {
	if result.Error != "" {
		return false
	}
	if policy.MaxBlocking != nil {
		if result.Blocking > *policy.MaxBlocking {
			return false
		}
	} else if result.Blocking > 0 {
		return false
	}
	if policy.MaxWarning != nil && result.Warning > *policy.MaxWarning {
		return false
	}
	if policy.MaxAdvisory != nil && result.Advisory > *policy.MaxAdvisory {
		return false
	}
	if policy.MaxUnresolvedReview != nil && result.UnresolvedReview > *policy.MaxUnresolvedReview {
		return false
	}
	return true
}

func replayExtractor(provider, model string, temperature float64, calls *[]replayLLMCall) (elicitor.Extractor, error) {
	llm, actualProvider, _, err := runner.NewLLMClientFromEnvWithOptions(provider, model, runner.LLMOptions{
		Temperature: &temperature,
	})
	if err != nil {
		return nil, err
	}
	chat, ok := llm.(rollout.ChatClient)
	if !ok {
		return nil, fmt.Errorf("provider %s does not support chat", actualProvider)
	}
	return elicitor.NewChatExtractor(&recordingChatClient{base: chat, calls: calls}, &temperature), nil
}

type recordingChatClient struct {
	base  rollout.ChatClient
	calls *[]replayLLMCall
}

func (c *recordingChatClient) Chat(ctx context.Context, messages []rollout.ChatMessage) (string, error) {
	response, err := c.base.Chat(ctx, messages)
	call := replayLLMCall{Kind: "chat", Messages: append([]rollout.ChatMessage(nil), messages...), Response: response}
	if err != nil {
		call.Error = err.Error()
	}
	*c.calls = append(*c.calls, call)
	return response, err
}

func (c *recordingChatClient) StructuredChat(ctx context.Context, messages []rollout.ChatMessage, schema json.RawMessage, opts rollout.StructuredOpts) (string, error) {
	structured, ok := c.base.(rollout.StructuredChat)
	if !ok {
		return "", errors.New("structured chat unavailable")
	}
	response, err := structured.StructuredChat(ctx, messages, schema, opts)
	call := replayLLMCall{Kind: "structured_chat", Messages: append([]rollout.ChatMessage(nil), messages...), Response: response}
	if err != nil {
		call.Error = err.Error()
	}
	*c.calls = append(*c.calls, call)
	return response, err
}

func writeJSONFile(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return os.WriteFile(path, data, 0o644)
}

func authorAnswers(answersFile, fromExample, exampleDir string, force bool, in io.Reader, out io.Writer) (projectwizard.Answers, error) {
	if strings.TrimSpace(answersFile) != "" {
		return loadAnswersFile(answersFile)
	}
	seed, err := loadSeed(fromExample, exampleDir, force)
	if err != nil {
		return projectwizard.Answers{}, err
	}
	return projectwizard.PromptWithDefaults(in, out, seed)
}

type seedSource string

const (
	seedSourceEmpty   seedSource = ""
	seedSourceAnswers seedSource = "answers"
	seedSourceSeed    seedSource = "seed"
	seedSourceDraft   seedSource = "draft"
)

func authorSession(answersFile, fromExample, exampleDir string, force bool, allowDraft bool) (elicitor.Session, seedSource, error) {
	if strings.TrimSpace(answersFile) != "" {
		session, err := loadSessionFile(answersFile)
		return session, seedSourceAnswers, err
	}
	if allowDraft {
		if session, ok, err := elicitor.LoadDraft(elicitor.DraftPath(exampleDir)); err != nil {
			return elicitor.Session{}, seedSourceEmpty, err
		} else if ok {
			return session, seedSourceDraft, nil
		}
	}
	session, err := loadSeedSession(fromExample, exampleDir, force)
	source := seedSourceEmpty
	if elicitor.LooksLikeSession(session) {
		source = seedSourceSeed
	}
	return session, source, err
}

func loadSessionFile(path string) (elicitor.Session, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return elicitor.Session{}, err
	}
	if looksLikeLegacyAnswers(data, strings.ToLower(filepath.Ext(path))) {
		answers, err := loadAnswersFile(path)
		if err != nil {
			return elicitor.Session{}, err
		}
		session := elicitor.NewSessionFromAnswers(answers)
		if answers.UsesOpenAPI {
			fmt.Fprintln(os.Stderr, "icot: legacy answers file does not include intent operation details; fill missing intent slots interactively or provide the new session shape")
		}
		return session, nil
	}
	session, sessionErr := elicitor.DecodeSession(data, strings.ToLower(filepath.Ext(path)))
	if sessionErr == nil && elicitor.LooksLikeSession(session) {
		session.Normalize()
		return session, nil
	}
	answers, answerErr := loadAnswersFile(path)
	if answerErr != nil {
		if sessionErr != nil {
			return elicitor.Session{}, fmt.Errorf("parse session: %w", sessionErr)
		}
		return elicitor.Session{}, answerErr
	}
	session = elicitor.NewSessionFromAnswers(answers)
	if len(session.Intent.Steps) == 1 && session.Intent.Steps[0] != nil && strings.TrimSpace(session.Intent.Steps[0].Operation) == "" && answers.UsesOpenAPI {
		fmt.Fprintf(os.Stderr, "icot: legacy answers file did not include intent operation details; fill missing intent slots interactively or provide the new session shape\n")
	}
	return session, nil
}

func looksLikeLegacyAnswers(data []byte, ext string) bool {
	var raw map[string]any
	if strings.EqualFold(ext, ".json") {
		if err := json.Unmarshal(data, &raw); err != nil {
			return false
		}
	} else if err := yaml.Unmarshal(data, &raw); err != nil {
		return false
	}
	if _, ok := raw["project_name"]; ok {
		return true
	}
	if _, ok := raw["uses_openapi"]; ok {
		return true
	}
	if _, ok := raw["goal"]; ok {
		_, hasIntent := raw["intent"]
		_, hasProject := raw["project"]
		return !hasIntent && !hasProject
	}
	return false
}

func loadAnswersFile(path string) (projectwizard.Answers, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return projectwizard.Answers{}, err
	}
	var answers projectwizard.Answers
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json":
		if err := json.Unmarshal(data, &answers); err != nil {
			return projectwizard.Answers{}, fmt.Errorf("parse answers JSON: %w", err)
		}
	default:
		if err := yaml.Unmarshal(data, &answers); err != nil {
			return projectwizard.Answers{}, fmt.Errorf("parse answers YAML: %w", err)
		}
	}
	return answers, nil
}

func loadSeed(fromExample, exampleDir string, force bool) (projectwizard.Answers, error) {
	seedDir := strings.TrimSpace(fromExample)
	if seedDir == "" && force {
		seedDir = exampleDir
	}
	if seedDir == "" {
		return projectwizard.Answers{}, nil
	}
	data, err := os.ReadFile(filepath.Join(seedDir, "project.md"))
	if err != nil {
		if os.IsNotExist(err) && strings.TrimSpace(fromExample) == "" {
			return projectwizard.Answers{}, nil
		}
		return projectwizard.Answers{}, err
	}
	return projectwizard.LoadAnswersFromMarkdown(string(data))
}

func loadSeedSession(fromExample, exampleDir string, force bool) (elicitor.Session, error) {
	seedDir := strings.TrimSpace(fromExample)
	if seedDir == "" && force {
		seedDir = exampleDir
	}
	if seedDir == "" {
		return elicitor.Session{}, nil
	}
	var project projectwizard.Answers
	projectData, projectErr := os.ReadFile(filepath.Join(seedDir, "project.md"))
	if projectErr == nil {
		loaded, err := projectwizard.LoadAnswersFromMarkdown(string(projectData))
		if err != nil {
			return elicitor.Session{}, err
		}
		project = loaded
	} else if !os.IsNotExist(projectErr) || strings.TrimSpace(fromExample) != "" {
		return elicitor.Session{}, projectErr
	}
	intent, intentErr := parseSeedIntent(seedDir)
	if intentErr == nil {
		return elicitor.SessionFromIntent(intent, project), nil
	}
	if projectErr == nil {
		return elicitor.NewSessionFromAnswers(project), nil
	}
	if strings.TrimSpace(fromExample) != "" {
		return elicitor.Session{}, intentErr
	}
	return elicitor.Session{}, nil
}

func parseSeedIntent(seedDir string) (*rollout.Intent, error) {
	workflowPath := filepath.Join(seedDir, "workflows", "intent.hcl")
	intent, err := rollout.ParseIntentFile(workflowPath)
	if err == nil {
		return intent, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	referencePath := filepath.Join(seedDir, "reference", "intent.hcl")
	intent, referenceErr := rollout.ParseIntentFile(referencePath)
	if referenceErr == nil {
		return intent, nil
	}
	if errors.Is(referenceErr, os.ErrNotExist) {
		return nil, err
	}
	return nil, referenceErr
}

func copySeedSourceArtifacts(fromExample, exampleDir string, force bool) error {
	seedDir := strings.TrimSpace(fromExample)
	if seedDir == "" || filepath.Clean(seedDir) == filepath.Clean(exampleDir) {
		return nil
	}
	for _, dir := range []string{"openapi", "google-discovery", "aws-smithy", "asyncapi", "graphql", "openrpc", "grpc-protobuf", "odata"} {
		if err := copySeedArtifactDir(filepath.Join(seedDir, dir), filepath.Join(exampleDir, dir), force); err != nil {
			return err
		}
	}
	return nil
}

func copySeedArtifactDir(srcDir, dstDir string, force bool) error {
	info, err := os.Stat(srcDir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("seed artifact path %s is not a directory", srcDir)
	}
	return filepath.WalkDir(srcDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return os.MkdirAll(dstDir, 0o755)
		}
		target := filepath.Join(dstDir, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("seed artifact path %s is not a regular file", path)
		}
		if _, err := os.Stat(target); err == nil && !force {
			return nil
		} else if err != nil && !os.IsNotExist(err) {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode().Perm())
	})
}

func previewAndConfirm(in io.Reader, out io.Writer, rendered string) (bool, error) {
	fmt.Fprintln(out, "\n----- project.md preview -----")
	fmt.Fprint(out, rendered)
	if !strings.HasSuffix(rendered, "\n") {
		fmt.Fprintln(out)
	}
	fmt.Fprintln(out, "----- end preview -----")
	for {
		fmt.Fprint(out, "Save project.md? [Y/n/cancel]: ")
		line, err := readLine(in)
		if err != nil && err != io.EOF {
			return false, err
		}
		if err == io.EOF && strings.TrimSpace(line) == "" {
			return false, io.ErrUnexpectedEOF
		}
		value := strings.ToLower(strings.TrimSpace(line))
		if value == "" || value == "y" || value == "yes" || value == "save" {
			return true, nil
		}
		if value == "n" || value == "no" || value == "cancel" || value == "q" || value == "quit" {
			return false, nil
		}
		if err == io.EOF {
			return false, io.ErrUnexpectedEOF
		}
	}
}

func writeProject(projectPath, rendered string, force, yes bool, in io.Reader, out io.Writer) error {
	return writeGeneratedFile(projectPath, rendered, force, yes, in, out)
}

func writeArtifacts(projectPath, intentPath string, artifacts elicitor.Artifacts, force, yes bool, in io.Reader, out io.Writer) error {
	for _, path := range []string{projectPath, intentPath} {
		if _, err := os.Stat(path); err == nil && !force {
			return fmt.Errorf("%s already exists; pass --force to overwrite it", path)
		} else if err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	if err := confirmOverwrites([]string{projectPath, intentPath}, force, yes, in, out); err != nil {
		return err
	}
	return writeGeneratedFilesAtomic([]generatedFile{
		{Path: projectPath, Content: artifacts.ProjectMD},
		{Path: intentPath, Content: artifacts.IntentHCL},
	}, force)
}

func writeGeneratedFile(path, rendered string, force, yes bool, in io.Reader, out io.Writer) error {
	if _, err := os.Stat(path); err == nil && !force {
		return fmt.Errorf("%s already exists; pass --force to overwrite it", path)
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := confirmOverwrites([]string{path}, force, yes, in, out); err != nil {
		return err
	}
	return writeGeneratedFilesAtomic([]generatedFile{{Path: path, Content: rendered}}, force)
}

func confirmOverwrites(paths []string, force, yes bool, in io.Reader, out io.Writer) error {
	if !force || yes {
		return nil
	}
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			ok, err := confirm(in, out, fmt.Sprintf("Overwrite %s?", path), false)
			if err != nil {
				return err
			}
			if !ok {
				return errors.New("overwrite canceled")
			}
		} else if err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

type generatedFile struct {
	Path    string
	Content string
}

type fileBackup struct {
	path       string
	backupPath string
	existed    bool
}

func writeGeneratedFilesAtomic(files []generatedFile, force bool) error {
	for _, file := range files {
		if err := validateGeneratedFile(file); err != nil {
			return err
		}
		if err := scaffoldDirs(exampleDirForGenerated(file.Path)); err != nil {
			return err
		}
		if _, err := os.Stat(file.Path); err == nil && !force {
			return fmt.Errorf("%s already exists; pass --force to overwrite it", file.Path)
		} else if err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	tmpPaths := map[string]string{}
	for _, file := range files {
		tmp, err := os.CreateTemp(filepath.Dir(file.Path), "."+filepath.Base(file.Path)+".tmp.")
		if err != nil {
			cleanupTemps(tmpPaths)
			return err
		}
		tmpPath := tmp.Name()
		tmpPaths[file.Path] = tmpPath
		_, writeErr := tmp.WriteString(file.Content)
		closeErr := tmp.Close()
		if writeErr != nil {
			cleanupTemps(tmpPaths)
			return writeErr
		}
		if closeErr != nil {
			cleanupTemps(tmpPaths)
			return closeErr
		}
	}
	backups := map[string]fileBackup{}
	for _, file := range files {
		if _, err := os.Stat(file.Path); err == nil {
			backupPath, err := backupFilePath(file.Path)
			if err != nil {
				cleanupTemps(tmpPaths)
				return err
			}
			backups[file.Path] = fileBackup{path: file.Path, backupPath: backupPath, existed: true}
		} else if err != nil && !os.IsNotExist(err) {
			cleanupTemps(tmpPaths)
			return err
		}
	}
	var renamed []string
	for _, file := range files {
		if err := os.Rename(tmpPaths[file.Path], file.Path); err != nil {
			restoreBackups(backups, renamed)
			cleanupTemps(tmpPaths)
			return err
		}
		renamed = append(renamed, file.Path)
	}
	return nil
}

func validateGeneratedFile(file generatedFile) error {
	if strings.TrimSpace(file.Path) == "" {
		return errors.New("empty output path")
	}
	if filepath.Base(file.Path) == "intent.hcl" {
		_, err := rollout.ParseIntent([]byte(file.Content), file.Path)
		if err != nil {
			return err
		}
	}
	return nil
}

func cleanupTemps(paths map[string]string) {
	for _, path := range paths {
		_ = os.Remove(path)
	}
}

func restoreBackups(backups map[string]fileBackup, renamed []string) {
	for i := len(renamed) - 1; i >= 0; i-- {
		path := renamed[i]
		backup := backups[path]
		if backup.existed {
			_ = copyFile(backup.backupPath, path)
		} else {
			_ = os.Remove(path)
		}
	}
}

func confirm(in io.Reader, out io.Writer, prompt string, defaultYes bool) (bool, error) {
	suffix := "[y/N]"
	if defaultYes {
		suffix = "[Y/n]"
	}
	fmt.Fprintf(out, "%s %s: ", prompt, suffix)
	line, err := readLine(in)
	if err != nil && err != io.EOF {
		return false, err
	}
	value := strings.ToLower(strings.TrimSpace(line))
	if value == "" {
		return defaultYes, nil
	}
	return value == "y" || value == "yes", nil
}

func readLine(in io.Reader) (string, error) {
	if reader, ok := in.(*bufio.Reader); ok {
		return reader.ReadString('\n')
	}
	return bufio.NewReader(in).ReadString('\n')
}

func backupProject(projectPath string) error {
	return backupFile(projectPath)
}

func backupFile(path string) error {
	_, err := backupFilePath(path)
	return err
}

func backupFilePath(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	base := fmt.Sprintf("%s.bak.%d", path, time.Now().UnixNano())
	for i := 0; ; i++ {
		backupPath := base
		if i > 0 {
			backupPath = fmt.Sprintf("%s.%d", base, i)
		}
		file, err := os.OpenFile(backupPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if errors.Is(err, os.ErrExist) {
			continue
		}
		if err != nil {
			return "", err
		}
		_, writeErr := file.Write(data)
		closeErr := file.Close()
		if writeErr != nil {
			return "", writeErr
		}
		if closeErr != nil {
			return "", closeErr
		}
		return backupPath, nil
	}
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o644)
}

func exampleDirForGenerated(path string) string {
	dir := filepath.Dir(path)
	if filepath.Base(dir) == "workflows" || filepath.Base(dir) == "openapi" || filepath.Base(dir) == "expected" {
		return filepath.Dir(dir)
	}
	return dir
}

func scaffoldDirs(exampleDir string) error {
	for _, dir := range []string{
		exampleDir,
		filepath.Join(exampleDir, "openapi"),
		filepath.Join(exampleDir, "workflows"),
		filepath.Join(exampleDir, "expected"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return nil
}

func completeSession(session elicitor.Session) bool {
	_, err := elicitor.RenderArtifacts(session)
	return err == nil
}

func printArtifacts(out io.Writer, artifacts elicitor.Artifacts) {
	fmt.Fprintln(out, "----- project.md -----")
	fmt.Fprint(out, artifacts.ProjectMD)
	if !strings.HasSuffix(artifacts.ProjectMD, "\n") {
		fmt.Fprintln(out)
	}
	fmt.Fprintln(out, "----- workflows/intent.hcl -----")
	fmt.Fprint(out, artifacts.IntentHCL)
	if !strings.HasSuffix(artifacts.IntentHCL, "\n") {
		fmt.Fprintln(out)
	}
}

func resolveExtractor(noLLM bool, provider, model string, temperature float64, out io.Writer) (elicitor.Extractor, bool) {
	if noLLM {
		return elicitor.NewNoopExtractor(), false
	}
	resolvedProvider := strings.TrimSpace(provider)
	if resolvedProvider == "" {
		resolvedProvider = providerFromEnv()
	}
	if resolvedProvider == "" {
		return elicitor.NewNoopExtractor(), false
	}
	llm, actualProvider, actualModel, err := runner.NewLLMClientFromEnvWithOptions(resolvedProvider, model, runner.LLMOptions{
		Temperature: &temperature,
	})
	if err != nil {
		fmt.Fprintf(out, "icot: LLM extraction unavailable: %v\n", err)
		return elicitor.NewNoopExtractor(), false
	}
	chat, ok := llm.(rollout.ChatClient)
	if !ok {
		fmt.Fprintf(out, "icot: LLM extraction unavailable: provider %s does not support chat\n", actualProvider)
		return elicitor.NewNoopExtractor(), false
	}
	fmt.Fprintf(out, "icot: using LLM extraction with %s/%s\n", actualProvider, actualModel)
	return elicitor.NewChatExtractor(chat, &temperature), true
}

func providerFromEnv() string {
	if os.Getenv("OPENUDON_LLM_PROVIDER") != "" {
		return strings.ToLower(strings.TrimSpace(os.Getenv("OPENUDON_LLM_PROVIDER")))
	}
	return "copilot-api"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
