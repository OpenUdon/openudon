package browserscenario

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/OpenUdon/openudon/internal/processgroup"
	"golang.org/x/mod/modfile"
)

// Options fixes the complete authority of one browser-scenario evaluation.
// Public targets remain unavailable unless AllowNetwork is explicitly set.
type Options struct {
	RepoRoot          string
	BrowsertoolsRepo  string
	UWSRepo           string
	UdonRepo          string
	BrowserdriverRepo string
	Suite             string
	ScenarioIDs       []string
	OutPath           string
	AllowNetwork      bool
	RequireReady      bool
	Now               func() time.Time
	Executor          ScenarioExecutor
}

// Environment contains only resolved local repositories and locked dependency
// facts. It deliberately contains no credential values or captured output.
type Environment struct {
	RepoRoot          string
	BrowsertoolsRepo  string
	UWSRepo           string
	UdonRepo          string
	BrowserdriverRepo string
	Lock              CompatibilityLock
	Now               time.Time
}

// ScenarioExecutor runs one already-validated manifest. Implementations must
// return only the closed, value-free result vocabulary accepted by Report.
type ScenarioExecutor interface {
	Execute(context.Context, Manifest, Environment) ScenarioResult
}

type ScenarioExecutorFunc func(context.Context, Manifest, Environment) ScenarioResult

// Run validates the embedded corpus and compatibility lock before any browser
// process or network authority can be exercised.
func Run(ctx context.Context, options Options) (*Report, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	now := time.Now().UTC().Round(0)
	if options.Now != nil {
		now = options.Now().UTC().Round(0)
	}
	if now.IsZero() {
		return nil, fmt.Errorf("browser scenario clock is unavailable")
	}
	if options.Suite != SuiteLoopback && options.Suite != SuiteJourney && options.Suite != SuitePublic {
		return nil, fmt.Errorf("browser scenario suite must be loopback, journey, or public")
	}
	if options.Suite == SuitePublic && !options.AllowNetwork {
		return nil, fmt.Errorf("public browser scenarios require explicit --allow-network")
	}
	if options.Suite != SuitePublic && options.AllowNetwork {
		return nil, fmt.Errorf("local browser scenarios do not accept network authority")
	}
	if strings.TrimSpace(options.OutPath) == "" {
		return nil, fmt.Errorf("browser scenario report output is required")
	}

	manifests, err := LoadManifests(now)
	if err != nil {
		return nil, err
	}
	selected, err := SelectManifests(manifests, options.Suite, options.ScenarioIDs)
	if err != nil {
		return nil, err
	}
	lock, err := LoadCompatibilityLock()
	if err != nil {
		return nil, err
	}
	environment, repositories, dependencies, err := resolveEnvironment(ctx, options, lock, now)
	if err != nil {
		return nil, err
	}
	executor := options.Executor
	if executor == nil {
		executor = NewRealExecutor()
	}
	if closer, ok := executor.(interface{ Close() error }); ok {
		defer closer.Close()
	}
	results := make([]ScenarioResult, 0, len(selected))
	for _, manifest := range selected {
		if manifest.Quarantine != nil {
			results = append(results, ScenarioResult{ID: manifest.ID, Status: StatusQuarantined, Detail: "quarantined"})
			continue
		}
		result := executor.Execute(ctx, manifest, environment)
		result.ID = manifest.ID
		result.Assertions = canonicalAssertions(result.Assertions)
		if options.RequireReady && result.Status == StatusSkipped {
			result.Status = StatusFail
			result.Detail = "dependency_unavailable"
			for index := range result.Phases {
				if result.Phases[index].Status == StatusSkipped {
					result.Phases[index].Status = StatusFail
				}
			}
		}
		results = append(results, result)
	}
	report := NewReport(options.Suite, now, repositories, dependencies, results)
	if err := ValidateReport(report); err != nil {
		return report, err
	}
	if err := WriteReport(options.OutPath, report); err != nil {
		return report, err
	}
	if report.Status == StatusFail {
		return report, fmt.Errorf("browser scenario evaluation failed")
	}
	return report, nil
}

func resolveEnvironment(ctx context.Context, options Options, lock CompatibilityLock, now time.Time) (Environment, []RepositoryRevision, []DependencyRevision, error) {
	root, err := absoluteDirectory(defaultPath(options.RepoRoot, "."), "openudon")
	if err != nil {
		return Environment{}, nil, nil, err
	}
	browsertoolsRepo, err := absoluteDirectory(defaultPath(options.BrowsertoolsRepo, filepath.Join(root, "..", "browsertools")), "browsertools")
	if err != nil {
		return Environment{}, nil, nil, err
	}
	uwsRepo, err := absoluteDirectory(defaultPath(options.UWSRepo, filepath.Join(root, "..", "uws")), "uws")
	if err != nil {
		return Environment{}, nil, nil, err
	}
	udonRepo, err := absoluteDirectory(defaultPath(options.UdonRepo, filepath.Join(root, "..", "udon")), "udon")
	if err != nil {
		return Environment{}, nil, nil, err
	}
	browserdriverRepo, err := absoluteDirectory(defaultPath(options.BrowserdriverRepo, filepath.Join(root, "..", "browserdriver")), "browserdriver")
	if err != nil {
		return Environment{}, nil, nil, err
	}

	repoPaths := []struct{ name, path string }{
		{"openudon", root}, {"browsertools", browsertoolsRepo}, {"uws", uwsRepo}, {"udon", udonRepo}, {"browserdriver", browserdriverRepo},
	}
	repositories := make([]RepositoryRevision, 0, len(repoPaths))
	for _, repo := range repoPaths {
		commit, dirty, revisionErr := gitRevision(ctx, repo.path)
		if revisionErr != nil {
			return Environment{}, nil, nil, fmt.Errorf("resolve %s browser-scenario revision", repo.name)
		}
		repositories = append(repositories, RepositoryRevision{Name: repo.name, Commit: commit, Dirty: dirty})
	}
	locked := make(map[string]LockedRevision, len(lock.Components))
	for _, component := range lock.Components {
		locked[component.Name] = component
	}
	states := make(map[string]RepositoryState, len(repositories)-1)
	for _, revision := range repositories[1:] {
		states[revision.Name] = RepositoryState{Commit: revision.Commit, Dirty: revision.Dirty}
	}
	if err := ValidateRepositoryStates(lock, states); err != nil {
		return Environment{}, nil, nil, err
	}
	if err := ValidateGoModulePins(root, browsertoolsRepo, lock); err != nil {
		return Environment{}, nil, nil, err
	}
	dependencies := []DependencyRevision{
		{Module: "github.com/OpenUdon/browsertools", Version: locked["browsertools"].Version},
		{Module: "github.com/OpenUdon/uws", Version: locked["uws"].Version},
	}
	return Environment{
		RepoRoot: root, BrowsertoolsRepo: browsertoolsRepo, UWSRepo: uwsRepo, UdonRepo: udonRepo,
		BrowserdriverRepo: browserdriverRepo, Lock: lock, Now: now,
	}, repositories, dependencies, nil
}

func absoluteDirectory(value, name string) (string, error) {
	path, err := filepath.Abs(value)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("%s repository is unavailable at %s", name, path)
	}
	return filepath.Clean(path), nil
}

func gitRevision(ctx context.Context, directory string) (string, bool, error) {
	var stdout strings.Builder
	err := processgroup.Run(ctx, probeDeadline, processgroup.Invocation{
		Args: []string{"git", "rev-parse", "HEAD"}, Dir: directory, Env: os.Environ(), Stdout: &stdout,
	})
	output := stdout.String()
	commit := strings.TrimSpace(string(output))
	if err != nil || !commitPattern.MatchString(commit) {
		return "", false, fmt.Errorf("git revision unavailable")
	}
	stdout.Reset()
	err = processgroup.Run(ctx, probeDeadline, processgroup.Invocation{
		Args: []string{"git", "status", "--porcelain=v1", "--untracked-files=all", "--", ".", ":(exclude)site", ":(exclude)site/**"}, Dir: directory, Env: os.Environ(), Stdout: &stdout,
	})
	output = stdout.String()
	if err != nil {
		return "", false, err
	}
	return commit, strings.TrimSpace(string(output)) != "", nil
}

func goModRequires(root, module, version string) bool {
	filename := filepath.Join(root, "go.mod")
	data, err := os.ReadFile(filename)
	if err != nil {
		return false
	}
	parsed, err := modfile.Parse(filename, data, nil)
	if err != nil {
		return false
	}
	for _, requirement := range parsed.Require {
		if requirement.Mod.Path == module && requirement.Mod.Version == version {
			return true
		}
	}
	return false
}

func defaultPath(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
