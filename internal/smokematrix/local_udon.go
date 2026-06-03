package smokematrix

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/OpenUdon/openudon/internal/trustedrunner"
)

type LocalUdonSmokeOptions struct {
	RepoRoot     string
	UdonRepo     string
	WorkDir      string
	OutPath      string
	Now          func() time.Time
	RunCommand   func(context.Context, string, ...string) error
	BuildCommand func(context.Context, string, string) error
}

func RunLocalUdonSmoke(ctx context.Context, opts LocalUdonSmokeOptions) (*Report, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	repoRoot, err := filepath.Abs(defaultString(opts.RepoRoot, "."))
	if err != nil {
		return nil, err
	}
	workdir, err := filepath.Abs(defaultString(opts.WorkDir, filepath.Join(repoRoot, ".openudon-run", "local-udon-smoke")))
	if err != nil {
		return nil, err
	}
	bin := filepath.Join(workdir, "bin", localUdonBinaryName())
	if err := os.MkdirAll(filepath.Dir(bin), 0o755); err != nil {
		return nil, err
	}
	udonRepo := strings.TrimSpace(opts.UdonRepo)
	if udonRepo == "" {
		udonRepo = filepath.Join(repoRoot, "..", "udon")
	}
	udonRepo, err = filepath.Abs(udonRepo)
	if err != nil {
		return nil, err
	}
	build := opts.BuildCommand
	if build == nil {
		build = buildLocalUdonBinary
	}
	if err := build(ctx, udonRepo, bin); err != nil {
		return nil, fmt.Errorf("build local udon executor: %w", err)
	}
	if info, err := os.Stat(bin); err != nil || info.IsDir() {
		return nil, fmt.Errorf("local udon executor was not built at %s", bin)
	}
	oldExecutor, hadExecutor := os.LookupEnv("OPENUDON_EXECUTOR")
	if err := os.Setenv("OPENUDON_EXECUTOR", bin); err != nil {
		return nil, err
	}
	defer func() {
		if hadExecutor {
			_ = os.Setenv("OPENUDON_EXECUTOR", oldExecutor)
		} else {
			_ = os.Unsetenv("OPENUDON_EXECUTOR")
		}
	}()
	report, err := Run(ctx, Options{
		RepoRoot:   repoRoot,
		WorkDir:    filepath.Join(workdir, "matrix"),
		OutPath:    defaultString(opts.OutPath, filepath.Join(workdir, "summary.json")),
		Mode:       ModeLive,
		Now:        opts.Now,
		RunCommand: opts.RunCommand,
		Scenarios: []Scenario{{
			ID:       "local-udon-runtime-only",
			Fixture:  "runtime-only-render",
			Sentence: "Render a provider-free runtime-only audit note through the sibling udon executor.",
			LiveKind: "local-udon",
			Overlay:  "local-udon",
			Inputs: map[string]any{
				"summary": "OpenUdon local udon smoke provider-free runtime-only execution.",
			},
		}},
	})
	if err != nil {
		return report, err
	}
	if report == nil || len(report.Scenarios) != 1 || strings.TrimSpace(report.Scenarios[0].RunEvidencePath) == "" {
		return report, fmt.Errorf("local udon smoke did not produce run evidence")
	}
	if _, err := trustedrunner.VerifyRunEvidenceFile(filepath.FromSlash(report.Scenarios[0].RunEvidencePath)); err != nil {
		return report, fmt.Errorf("verify local udon smoke run evidence: %w", err)
	}
	return report, nil
}

func buildLocalUdonBinary(ctx context.Context, repo, out string) error {
	cmd := exec.CommandContext(ctx, "go", "build", "-o", out, "./cmd/udon")
	cmd.Dir = repo
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func localUdonBinaryName() string {
	if runtime.GOOS == "windows" {
		return "udon.exe"
	}
	return "udon"
}
