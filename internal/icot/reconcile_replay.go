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
	"github.com/OpenUdon/openudon/internal/authoring/atomicfile"
	evalpkg "github.com/OpenUdon/openudon/internal/eval"
	"github.com/OpenUdon/openudon/internal/evidencefile"
	"github.com/OpenUdon/openudon/internal/icot/artifactwriter"
	"github.com/OpenUdon/openudon/internal/icot/elicitor"
	"github.com/OpenUdon/openudon/internal/projectwizard"
	"github.com/OpenUdon/openudon/internal/sourcecatalog"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
	runner "github.com/OpenUdon/openudon/internal/workflowintent"
)

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
	session, err := elicitor.SessionFromIntent(intent, project)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
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
	session, err := elicitor.DecodeSession(data, strings.ToLower(filepath.Ext(path)))
	if err != nil {
		return elicitor.Session{}, fmt.Errorf("parse v2 session: %w", err)
	}
	if !elicitor.LooksLikeSession(session) {
		return elicitor.Session{}, errors.New("v2 session has no workflow state")
	}
	return session, nil
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
		return elicitor.SessionFromIntent(intent, project)
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
	for _, dir := range sourcecatalog.All() {
		if err := copySeedArtifactDir(filepath.Join(seedDir, dir), filepath.Join(exampleDir, dir), force); err != nil {
			return err
		}
	}
	return nil
}

func copySeedArtifactDir(srcDir, dstDir string, force bool) error {
	info, err := os.Lstat(srcDir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("seed artifact path %s must be a real directory", srcDir)
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
			return ensureSeedDestinationDirectory(dstDir)
		}
		target := filepath.Join(dstDir, rel)
		if d.IsDir() {
			return ensureSeedDestinationDirectory(target)
		}
		data, sourceInfo, err := evidencefile.ReadRegular(path, evidencefile.DefaultMaxBytes)
		if err != nil {
			return fmt.Errorf("read seed artifact %s: %w", path, err)
		}
		targetInfo, err := os.Lstat(target)
		if err == nil {
			if targetInfo.Mode()&os.ModeSymlink != 0 || !targetInfo.Mode().IsRegular() {
				return fmt.Errorf("seed artifact destination %s must be a regular file", target)
			}
			if !force {
				return nil
			}
		} else if !os.IsNotExist(err) {
			return err
		}
		if err := ensureSeedDestinationDirectory(filepath.Dir(target)); err != nil {
			return err
		}
		return atomicfile.Write(target, data, sourceInfo.Mode().Perm())
	})
}

func ensureSeedDestinationDirectory(path string) error {
	if err := os.MkdirAll(path, 0o755); err != nil {
		return err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("seed artifact destination %s must be a real directory", path)
	}
	return nil
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

func writeApprovedArtifacts(exampleDir string, artifacts elicitor.Artifacts, force, yes bool, in io.Reader, out io.Writer) error {
	prepared, err := artifactwriter.Prepare(exampleDir, artifacts, force, time.Now().UTC())
	if err != nil {
		return err
	}
	paths := make([]string, 0, len(prepared.Files))
	for _, file := range prepared.Files {
		paths = append(paths, file.Path)
	}
	if err := confirmOverwrites(paths, force, yes, in, out); err != nil {
		return err
	}
	_, err = artifactwriter.Commit(prepared, force)
	return err
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

type generatedFile = artifactwriter.GeneratedFile

func writeGeneratedFilesAtomic(files []generatedFile, force bool) error {
	return artifactwriter.WriteFilesAtomic(files, force)
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
