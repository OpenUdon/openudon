package readiness

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/genelet/ramen/internal/config"
)

const ReportVersion = "ramen.local-readiness.v1"

type Options struct {
	RepoRoot string
	RunGates bool
	Now      func() time.Time
	Runner   CommandRunner
}

type CommandRunner interface {
	Run(ctx context.Context, dir string, name string, args ...string) CommandResult
}

type CommandResult struct {
	ExitCode int    `json:"exit_code"`
	Output   string `json:"output,omitempty"`
	Error    string `json:"error,omitempty"`
}

type Report struct {
	Version             string             `json:"version"`
	GeneratedAt         string             `json:"generated_at"`
	RepoRoot            string             `json:"repo_root"`
	Status              string             `json:"status"`
	Siblings            []Check            `json:"siblings"`
	DeterministicGates  []Check            `json:"deterministic_gates"`
	Git                 []Check            `json:"git"`
	IgnoredArtifacts    []Check            `json:"ignored_artifacts"`
	ProviderEnvironment []ProviderEnvCheck `json:"provider_environment"`
	AutomationPolicy    AutomationPolicy   `json:"automation_policy"`
}

type Check struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

type ProviderEnvCheck struct {
	Name    string `json:"name"`
	Present bool   `json:"present"`
	Message string `json:"message"`
}

type AutomationPolicy struct {
	HostedCIEnabled        bool   `json:"hosted_ci_enabled"`
	RealProviderAutomation bool   `json:"real_provider_automation"`
	RequiredMode           string `json:"required_mode"`
	Notes                  string `json:"notes"`
}

type execRunner struct{}

func (execRunner) Run(ctx context.Context, dir string, name string, args ...string) CommandResult {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	var combined bytes.Buffer
	cmd.Stdout = &combined
	cmd.Stderr = &combined
	err := cmd.Run()
	result := CommandResult{Output: strings.TrimSpace(combined.String())}
	if err == nil {
		return result
	}
	result.Error = err.Error()
	if exitErr, ok := err.(*exec.ExitError); ok {
		result.ExitCode = exitErr.ExitCode()
	} else {
		result.ExitCode = -1
	}
	return result
}

func Build(ctx context.Context, opts Options) (Report, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	root := strings.TrimSpace(opts.RepoRoot)
	if root == "" {
		root = "."
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return Report{}, fmt.Errorf("resolve repo root: %w", err)
	}
	report := Report{
		Version:     ReportVersion,
		GeneratedAt: resolveNow(opts.Now).UTC().Format(time.RFC3339),
		RepoRoot:    absRoot,
		Status:      "pass",
		AutomationPolicy: AutomationPolicy{
			HostedCIEnabled:        false,
			RealProviderAutomation: false,
			RequiredMode:           "local_manual",
			Notes:                  "Deterministic gates run on a trusted workstation with private siblings; real-provider evals remain local/manual until protected secrets and redaction policy exist.",
		},
	}
	report.Siblings = siblingChecks(absRoot)
	report.IgnoredArtifacts = ignoredArtifactChecks(absRoot)
	report.ProviderEnvironment = providerEnvironmentChecks()
	report.Git = gitChecks(ctx, absRoot, opts.Runner)
	report.DeterministicGates = deterministicGateChecks(ctx, absRoot, opts.RunGates, opts.Runner)
	report.Status = reportStatus(report)
	return report, nil
}

func Write(w io.Writer, report Report) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	_, err = w.Write(append(data, '\n'))
	return err
}

func WriteFile(path string, report Report) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return Write(file, report)
}

func siblingChecks(root string) []Check {
	parent := filepath.Dir(root)
	var checks []Check
	for _, name := range config.RequiredSiblings() {
		path := filepath.Join(parent, name)
		info, err := os.Stat(path)
		if err != nil {
			checks = append(checks, Check{Name: name, Status: "fail", Message: "required sibling is missing", Detail: path})
			continue
		}
		if !info.IsDir() {
			checks = append(checks, Check{Name: name, Status: "fail", Message: "required sibling path is not a directory", Detail: path})
			continue
		}
		checks = append(checks, Check{Name: name, Status: "pass", Message: "required sibling is present", Detail: path})
	}
	return checks
}

func ignoredArtifactChecks(root string) []Check {
	required := []string{
		".ramen-run/",
		"approvals/",
		"eval/artifacts/",
		"eval/readiness/",
		"eval/runs/",
	}
	ignored := gitignoreEntries(filepath.Join(root, ".gitignore"))
	var checks []Check
	for _, entry := range required {
		if ignored[entry] {
			checks = append(checks, Check{Name: entry, Status: "pass", Message: "local artifact path is ignored"})
		} else {
			checks = append(checks, Check{Name: entry, Status: "fail", Message: "local artifact path is not ignored"})
		}
	}
	return checks
}

func gitignoreEntries(path string) map[string]bool {
	out := map[string]bool{}
	data, err := os.ReadFile(path)
	if err != nil {
		return out
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out[line] = true
	}
	return out
}

func providerEnvironmentChecks() []ProviderEnvCheck {
	names := []string{"GEMINI_API_KEY", "OPENAI_API_KEY", "ANTHROPIC_API_KEY"}
	var checks []ProviderEnvCheck
	for _, name := range names {
		present := strings.TrimSpace(os.Getenv(name)) != ""
		message := "not set; real-provider evals that use this provider will require a trusted local environment"
		if present {
			message = "set; value intentionally not reported"
		}
		checks = append(checks, ProviderEnvCheck{Name: name, Present: present, Message: message})
	}
	return checks
}

func gitChecks(ctx context.Context, root string, runner CommandRunner) []Check {
	runner = resolveRunner(runner)
	checks := []Check{commandCheck(ctx, runner, root, "git.diff_check", "git", "diff", "--check")}
	status := runner.Run(ctx, root, "git", "status", "--porcelain")
	if status.ExitCode != 0 || status.Error != "" {
		checks = append(checks, Check{Name: "git.status", Status: "fail", Message: "git status failed", Detail: commandDetail(status)})
		return checks
	}
	if strings.TrimSpace(status.Output) == "" {
		checks = append(checks, Check{Name: "git.status", Status: "pass", Message: "working tree is clean"})
	} else {
		checks = append(checks, Check{Name: "git.status", Status: "warn", Message: "working tree has uncommitted changes", Detail: status.Output})
	}
	return checks
}

func deterministicGateChecks(ctx context.Context, root string, run bool, runner CommandRunner) []Check {
	gates := []struct {
		name string
		args []string
	}{
		{name: "go.test", args: []string{"go", "test", "./..."}},
		{name: "go.vet", args: []string{"go", "vet", "./..."}},
		{name: "make.check", args: []string{"make", "check"}},
	}
	if !run {
		checks := make([]Check, 0, len(gates))
		for _, gate := range gates {
			checks = append(checks, Check{Name: gate.name, Status: "skip", Message: "not run; pass --run-gates for deterministic gate evidence"})
		}
		return checks
	}
	runner = resolveRunner(runner)
	var checks []Check
	for _, gate := range gates {
		checks = append(checks, commandCheck(ctx, runner, root, gate.name, gate.args[0], gate.args[1:]...))
	}
	return checks
}

func commandCheck(ctx context.Context, runner CommandRunner, root, checkName, name string, args ...string) Check {
	result := runner.Run(ctx, root, name, args...)
	if result.ExitCode == 0 && result.Error == "" {
		return Check{Name: checkName, Status: "pass", Message: "command passed"}
	}
	return Check{Name: checkName, Status: "fail", Message: "command failed", Detail: commandDetail(result)}
}

func commandDetail(result CommandResult) string {
	parts := []string{}
	if result.ExitCode != 0 {
		parts = append(parts, fmt.Sprintf("exit=%d", result.ExitCode))
	}
	if strings.TrimSpace(result.Error) != "" {
		parts = append(parts, "error="+strings.TrimSpace(result.Error))
	}
	if strings.TrimSpace(result.Output) != "" {
		parts = append(parts, "output="+strings.TrimSpace(result.Output))
	}
	return strings.Join(parts, "; ")
}

func reportStatus(report Report) string {
	var warned bool
	for _, check := range append(append(append([]Check{}, report.Siblings...), report.IgnoredArtifacts...), append(report.Git, report.DeterministicGates...)...) {
		switch check.Status {
		case "fail":
			return "fail"
		case "warn", "skip":
			warned = true
		}
	}
	if warned {
		return "warn"
	}
	return "pass"
}

func resolveRunner(runner CommandRunner) CommandRunner {
	if runner != nil {
		return runner
	}
	return execRunner{}
}

func resolveNow(now func() time.Time) time.Time {
	if now == nil {
		return time.Now()
	}
	return now()
}

func SortChecks(checks []Check) {
	sort.Slice(checks, func(i, j int) bool {
		return checks[i].Name < checks[j].Name
	})
}
