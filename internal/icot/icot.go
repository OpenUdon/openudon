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

	"github.com/genelet/ramen/internal/icot/elicitor"
	"github.com/genelet/ramen/internal/projectwizard"
	"github.com/genelet/ramen/internal/synthesize"
	"github.com/genelet/udon/pkg/rollout"
	"github.com/genelet/udon/pkg/runner"
	"gopkg.in/yaml.v3"
)

func Main(args []string, in io.Reader, out, errOut io.Writer) int {
	if len(args) > 0 && args[0] == "lint" {
		return runLint(args[1:], out, errOut)
	}
	if len(args) > 0 && args[0] == "reconcile" {
		return runReconcile(args[1:], in, out, errOut)
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
	provider := fs.String("provider", "", "LLM provider for optional extraction: openai, anthropic, or gemini")
	model := fs.String("model", "", "LLM model for optional extraction")
	temperature := fs.Float64("temperature", 0.2, "LLM extraction temperature")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: icot --example examples/<name> [--dir examples/<name>] [--force] [--yes] [--print] [--from-example examples/<seed>] [--answers answers.yaml]\n")
		fmt.Fprintf(fs.Output(), "\nInteractively writes project.md and workflows/intent.hcl with the standard Ramen authoring sections.\n")
		fmt.Fprintf(fs.Output(), "It also creates openapi/, workflows/, and expected/ when missing.\n")
		fmt.Fprintf(fs.Output(), "Next step: ramen build --example examples/<name>\n\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}

	exampleDir := firstNonEmpty(*example, *dirAlias)
	if exampleDir == "" {
		fmt.Fprintln(errOut, "--example is required")
		return 2
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
	extractor, usingLLM := resolveExtractor(*noLLM, *provider, *model, *temperature, out)
	if !usingLLM {
		fmt.Fprintln(out, "icot: running without LLM extraction; continuing with manual slot filling")
	}
	var artifacts elicitor.Artifacts
	complete := completeSession(seed)
	if complete && (source != seedSourceDraft || *printOnly) {
		artifacts, err = elicitor.RenderArtifacts(seed)
	} else {
		artifacts, err = elicitor.Run(context.Background(), input, out, seed, elicitor.Options{
			ExampleDir: exampleDir,
			NoLLM:      *noLLM || !usingLLM,
			Extractor:  extractor,
			DraftPath:  draftPath,
			VerifyOnly: complete && source == seedSourceDraft,
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
	if err := elicitor.DeleteDraft(draftPath); err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	fmt.Fprintf(out, "icot: wrote %s\n", projectPath)
	fmt.Fprintf(out, "icot: wrote %s\n", intentPath)
	fmt.Fprintf(out, "next: ramen build --example %s\n", exampleDir)
	return 0
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

func runLint(args []string, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("icot lint", flag.ContinueOnError)
	fs.SetOutput(out)
	example := fs.String("example", "", "Example directory containing project.md")
	file := fs.String("file", "", "Path to a project.md file")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: icot lint --example examples/<name>\n")
		fmt.Fprintf(fs.Output(), "       icot lint --file path/to/project.md\n\n")
		fmt.Fprintf(fs.Output(), "Runs deterministic project.md authoring checks without LLM or udon execution.\n\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	path := strings.TrimSpace(*file)
	if path == "" && strings.TrimSpace(*example) != "" {
		path = filepath.Join(strings.TrimSpace(*example), "project.md")
	}
	if path == "" {
		fmt.Fprintln(errOut, "--example or --file is required")
		return 2
	}
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	failed := printLint(out, synthesize.LintProjectMarkdown(string(data)))
	intentExampleDir := strings.TrimSpace(*example)
	if intentExampleDir == "" && filepath.Base(path) == "project.md" {
		intentExampleDir = filepath.Dir(path)
	}
	if intentExampleDir != "" {
		intentPath := filepath.Join(intentExampleDir, "workflows", "intent.hcl")
		if _, statErr := os.Stat(intentPath); statErr == nil {
			intent, err := lintIntent(out, intentPath)
			if err != nil {
				failed = true
			} else {
				printDrift(out, elicitor.CompareProjectIntentDrift(string(data), intent))
			}
		}
	}
	if failed {
		return 1
	}
	return 0
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
	intent, intentErr := rollout.ParseIntentFile(filepath.Join(seedDir, "workflows", "intent.hcl"))
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

func printLint(out io.Writer, checks []synthesize.QualityCheck) bool {
	failed := false
	fmt.Fprintln(out, "icot: lint")
	for _, check := range checks {
		fmt.Fprintf(out, "  %s: %s - %s\n", check.Code, check.Status, check.Message)
		if check.Detail != "" {
			fmt.Fprintf(out, "    detail: %s\n", check.Detail)
		}
		if check.Status == "fail" {
			failed = true
		}
	}
	return failed
}

func lintIntent(out io.Writer, path string) (*rollout.Intent, error) {
	fmt.Fprintf(out, "  intent.parse: ")
	intent, err := rollout.ParseIntentFile(path)
	if err != nil {
		fmt.Fprintf(out, "fail - %v\n", err)
		return nil, err
	}
	missing := intent.MissingSlots()
	if len(missing) > 0 {
		fmt.Fprintf(out, "fail - missing %s\n", strings.Join(missing, ", "))
		return nil, fmt.Errorf("missing %s", strings.Join(missing, ", "))
	}
	fmt.Fprintln(out, "pass - workflows/intent.hcl parses")
	return intent, nil
}

func printDrift(out io.Writer, checks []elicitor.DriftCheck) {
	if len(checks) == 0 {
		fmt.Fprintln(out, "  icot.drift: pass - project.md matches workflows/intent.hcl")
		return
	}
	for _, check := range checks {
		fmt.Fprintf(out, "  %s: warn - %s\n", check.Code, check.Message)
		if check.Detail != "" {
			fmt.Fprintf(out, "    detail: %s\n", check.Detail)
		}
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
	switch {
	case os.Getenv("GEMINI_API_KEY") != "":
		return "gemini"
	case os.Getenv("OPENAI_API_KEY") != "":
		return "openai"
	case os.Getenv("ANTHROPIC_API_KEY") != "":
		return "anthropic"
	default:
		return ""
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
