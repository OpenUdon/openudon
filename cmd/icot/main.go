package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/genelet/ramen/internal/projectwizard"
	"github.com/genelet/ramen/internal/synthesize"
	"gopkg.in/yaml.v3"
)

func main() {
	args := os.Args[1:]
	if len(args) > 0 && args[0] == "lint" {
		runLint(args[1:])
		return
	}
	runAuthor(args)
}

func runAuthor(args []string) {
	fs := flag.NewFlagSet("icot", flag.ExitOnError)
	example := fs.String("example", "", "Example directory where project.md will be created")
	dirAlias := fs.String("dir", "", "Alias for --example")
	force := fs.Bool("force", false, "Overwrite an existing project.md")
	yes := fs.Bool("yes", false, "Accept overwrite prompts without asking")
	printOnly := fs.Bool("print", false, "Render project.md to stdout without writing files")
	fromExample := fs.String("from-example", "", "Seed answers from an existing example directory")
	answersFile := fs.String("answers", "", "Path to YAML or JSON answers file; suppresses interactive prompts")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: icot --example examples/<name> [--dir examples/<name>] [--force] [--yes] [--print] [--from-example examples/<seed>] [--answers answers.yaml]\n")
		fmt.Fprintf(fs.Output(), "\nInteractively writes project.md with the standard Ramen authoring sections.\n")
		fmt.Fprintf(fs.Output(), "It also creates openapi/, workflows/, and expected/ when missing.\n")
		fmt.Fprintf(fs.Output(), "Next step: ramen synthesize --example examples/<name>\n\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	exampleDir := firstNonEmpty(*example, *dirAlias)
	if exampleDir == "" {
		fmt.Fprintln(os.Stderr, "--example is required")
		os.Exit(2)
	}
	projectPath := filepath.Join(exampleDir, "project.md")
	input := bufio.NewReader(os.Stdin)

	answers, err := authorAnswers(*answersFile, *fromExample, exampleDir, *force, input, os.Stdout)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	rendered := projectwizard.Render(answers)
	if *printOnly {
		fmt.Print(rendered)
		return
	}
	if *answersFile == "" {
		save, err := previewAndConfirm(input, os.Stdout, rendered)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if !save {
			return
		}
	}
	if err := writeProject(projectPath, rendered, *force, *yes, input, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("icot: wrote %s\n", projectPath)
	fmt.Printf("next: ramen synthesize --example %s\n", exampleDir)
}

func runLint(args []string) {
	fs := flag.NewFlagSet("icot lint", flag.ExitOnError)
	example := fs.String("example", "", "Example directory containing project.md")
	file := fs.String("file", "", "Path to a project.md file")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: icot lint --example examples/<name>\n")
		fmt.Fprintf(fs.Output(), "       icot lint --file path/to/project.md\n\n")
		fmt.Fprintf(fs.Output(), "Runs deterministic project.md authoring checks without LLM or udon execution.\n\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	path := strings.TrimSpace(*file)
	if path == "" && strings.TrimSpace(*example) != "" {
		path = filepath.Join(strings.TrimSpace(*example), "project.md")
	}
	if path == "" {
		fmt.Fprintln(os.Stderr, "--example or --file is required")
		os.Exit(2)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	checks := synthesize.LintProjectMarkdown(string(data))
	failed := printLint(os.Stdout, checks)
	if failed {
		os.Exit(1)
	}
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
	if _, err := os.Stat(projectPath); err == nil {
		if !force {
			return fmt.Errorf("%s already exists; pass --force to overwrite it", projectPath)
		}
		if !yes {
			ok, err := confirm(in, out, fmt.Sprintf("Overwrite %s?", projectPath), false)
			if err != nil {
				return err
			}
			if !ok {
				return errors.New("overwrite canceled")
			}
		}
		if err := backupProject(projectPath); err != nil {
			return err
		}
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := scaffoldDirs(filepath.Dir(projectPath)); err != nil {
		return err
	}
	return os.WriteFile(projectPath, []byte(rendered), 0o644)
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
	data, err := os.ReadFile(projectPath)
	if err != nil {
		return err
	}
	backupPath := fmt.Sprintf("%s.bak.%d", projectPath, time.Now().Unix())
	return os.WriteFile(backupPath, data, 0o644)
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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
