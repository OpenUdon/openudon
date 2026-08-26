package browsertransactioneval

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/OpenUdon/openudon/internal/browserscenario"
	"github.com/OpenUdon/openudon/internal/evidencefile"
	"github.com/OpenUdon/openudon/internal/processgroup"
)

const qualificationCommandOutputLimit = 64 << 10

// QualificationOptions fixes every local repository and the sole report
// output. Qualification may resolve public Git heads read-only, but browser
// execution remains restricted to embedded loopback fixtures.
type QualificationOptions struct {
	RepoRoot          string
	BrowsertoolsRepo  string
	UWSRepo           string
	UdonRepo          string
	BrowserdriverRepo string
	OutPath           string
	Now               func() time.Time
}

// RunQualification executes the complete value-free qualification. It must
// start from clean exact revisions so every retained digest is commit-bound.
func RunQualification(ctx context.Context, options QualificationOptions) (*Report, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	now := time.Now().UTC().Truncate(time.Second)
	if options.Now != nil {
		now = options.Now().UTC().Truncate(time.Second)
	}
	if now.IsZero() || strings.TrimSpace(options.OutPath) == "" {
		return nil, errors.New("browser transaction qualification authority is invalid")
	}
	roots, err := qualificationRoots(options)
	if err != nil {
		return nil, err
	}
	repositories, err := qualificationRepositories(ctx, roots)
	if err != nil {
		return nil, err
	}
	if err := runQualificationCommandSilent(ctx, 10*time.Minute, roots.openudon,
		[]string{"make", "BROWSERTOOLS_REPO=" + roots.browsertools, "browser-transaction-adversarial"}); err != nil {
		return nil, errors.New("browser transaction adversarial qualification failed")
	}
	scenarioOptions := browserscenario.Options{
		RepoRoot: roots.openudon, BrowsertoolsRepo: roots.browsertools, UWSRepo: roots.uws,
		UdonRepo: roots.udon, BrowserdriverRepo: roots.browserdriver,
	}
	bapBCP, err := browserscenario.RunBAPBCPQualification(ctx, scenarioOptions)
	if err != nil {
		return nil, qualificationStageError(err, "authenticated browser transaction qualification failed")
	}
	brp, err := browserscenario.RunBRPQualification(ctx, scenarioOptions)
	if err != nil {
		return nil, qualificationStageError(err, "registration browser transaction qualification failed")
	}
	currentRepositories, err := qualificationRepositories(ctx, roots)
	if err != nil || !equalRepositoryRevisions(repositories, currentRepositories) {
		return nil, errors.New("browser transaction qualification repository state changed during execution")
	}
	report, err := BuildQualificationReport(now, repositories, bapBCP, brp)
	if err != nil {
		return nil, err
	}
	if err := Write(options.OutPath, report); err != nil {
		return nil, err
	}
	verified, err := VerifyFile(options.OutPath, true)
	if err != nil {
		return nil, err
	}
	return verified, nil
}

func qualificationStageError(err error, fallback string) error {
	if errors.Is(err, browserscenario.ErrSandboxPrerequisiteUnavailable) {
		return browserscenario.ErrSandboxPrerequisiteUnavailable
	}
	return errors.New(fallback)
}

// BuildQualificationReport maps only already-validated path-free evidence to
// the canonical report. Adversarial gate success is represented by the fixed
// pass results because RunQualification constructs this only after that target
// exits successfully.
func BuildQualificationReport(at time.Time, repositories []RepositoryRevision, bapBCP browserscenario.BAPBCPQualificationEvidence, brp browserscenario.BRPQualificationEvidence) (*Report, error) {
	if err := browserscenario.ValidateBAPBCPQualificationEvidence(bapBCP); err != nil {
		return nil, errors.New("authenticated qualification evidence is invalid")
	}
	if err := browserscenario.ValidateBRPQualificationEvidence(brp); err != nil {
		return nil, errors.New("registration qualification evidence is invalid")
	}
	artifacts := qualificationArtifacts(bapBCP, brp)
	results := []GateResult{
		{ID: GateReportSchema, Status: StatusPass, EvidenceCount: 1},
		{ID: GateBAPBCPProducer, Status: StatusPass, EvidenceCount: 1},
		{ID: GateBAPBCPTransaction, Status: StatusPass, EvidenceCount: 2},
		{ID: GateBAPBCPPackage, Status: StatusPass, EvidenceCount: 5},
		{ID: GateBAPBCPHandoff, Status: StatusPass, EvidenceCount: 2},
		{ID: GateBAPBCPReplay, Status: StatusPass, EvidenceCount: 1},
		{ID: GateBRPProducer, Status: StatusPass, EvidenceCount: 1},
		{ID: GateBRPNetwork, Status: StatusPass, EvidenceCount: brp.Requests},
		{ID: GateBRPTransaction, Status: StatusPass, EvidenceCount: 2},
		{ID: GateBRPPackage, Status: StatusPass, EvidenceCount: 5},
		{ID: GateBRPExecutorRejection, Status: StatusPass, EvidenceCount: 2},
		{ID: GateProtocolBounds, Status: StatusPass, EvidenceCount: 1},
		{ID: GateLifecycleDrift, Status: StatusPass, EvidenceCount: 1},
		{ID: GateConcurrentLifecycle, Status: StatusPass, EvidenceCount: 1},
		{ID: GateFilesystemRollback, Status: StatusPass, EvidenceCount: 1},
		{ID: GateIndeterminateRecovery, Status: StatusPass, EvidenceCount: 1},
		{ID: GateFrontendConflicts, Status: StatusPass, EvidenceCount: 1},
		{ID: GateSensitiveArtifactScan, Status: StatusPass, EvidenceCount: 2},
	}
	return NewReport(BuildRequest{
		GeneratedAt: at, Repositories: repositories,
		Posture: Posture{
			SandboxRequired: true, SandboxEnabled: true, LoopbackOnly: true,
			RegistrationAuthoringMethods:      append([]string(nil), brp.Methods...),
			RegistrationAuthoringPostRequests: brp.MutationRequests,
			AccountCreated:                    brp.AccountCreated, ExecutorInvokedForRegistration: brp.ExecutorInvoked,
			ContainsPrivateMaterial: false, ValueFree: true,
		},
		Artifacts: artifacts, Results: results,
	})
}

func qualificationArtifacts(bapBCP browserscenario.BAPBCPQualificationEvidence, brp browserscenario.BRPQualificationEvidence) []ArtifactDigest {
	bapValues := []string{
		bapBCP.ProducerResultSHA256, bapBCP.TransactionSHA256, bapBCP.PreparationSHA256,
		bapBCP.QualificationSHA256, bapBCP.GenerationSHA256, bapBCP.SelectionSHA256,
		bapBCP.PackageSHA256, bapBCP.HandoffSHA256, bapBCP.WorkflowSHA256,
	}
	brpValues := []string{
		brp.ProducerResultSHA256, brp.TransactionSHA256, brp.PreparationSHA256,
		brp.QualificationSHA256, brp.GenerationSHA256, brp.SelectionSHA256,
		brp.PackageSHA256, brp.HandoffSHA256, brp.WorkflowSHA256,
	}
	artifacts := make([]ArtifactDigest, 0, 2*len(artifactKindOrder))
	for caseIndex, caseID := range []string{CaseBAPBCP, CaseBRP} {
		values := bapValues
		if caseIndex == 1 {
			values = brpValues
		}
		for index, kind := range artifactKindOrder {
			artifacts = append(artifacts, ArtifactDigest{Case: caseID, Kind: kind, SHA256: values[index]})
		}
	}
	return artifacts
}

type qualificationRepoRoots struct {
	openudon, browsertools, uws, udon, browserdriver string
}

func qualificationRoots(options QualificationOptions) (qualificationRepoRoots, error) {
	root, err := qualificationDirectory(options.RepoRoot, ".", "OpenUdon")
	if err != nil {
		return qualificationRepoRoots{}, err
	}
	browsertools, err := qualificationDirectory(options.BrowsertoolsRepo, filepath.Join(root, "..", "browsertools"), "Browsertools")
	if err != nil {
		return qualificationRepoRoots{}, err
	}
	uws, err := qualificationDirectory(options.UWSRepo, filepath.Join(root, "..", "uws"), "UWS")
	if err != nil {
		return qualificationRepoRoots{}, err
	}
	udon, err := qualificationDirectory(options.UdonRepo, filepath.Join(root, "..", "udon"), "Udon")
	if err != nil {
		return qualificationRepoRoots{}, err
	}
	browserdriver, err := qualificationDirectory(options.BrowserdriverRepo, filepath.Join(root, "..", "browserdriver"), "Browserdriver")
	if err != nil {
		return qualificationRepoRoots{}, err
	}
	return qualificationRepoRoots{root, browsertools, uws, udon, browserdriver}, nil
}

func qualificationDirectory(value, fallback, name string) (string, error) {
	if strings.TrimSpace(value) == "" {
		value = fallback
	}
	abs, err := filepath.Abs(value)
	if err != nil {
		return "", fmt.Errorf("resolve %s qualification repository", name)
	}
	info, statErr := os.Stat(abs)
	if statErr != nil || !info.IsDir() {
		return "", fmt.Errorf("%s qualification repository is unavailable", name)
	}
	return filepath.Clean(abs), nil
}

func qualificationRepositories(ctx context.Context, roots qualificationRepoRoots) ([]RepositoryRevision, error) {
	lock, err := browserscenario.LoadCompatibilityLock()
	if err != nil {
		return nil, err
	}
	locked := map[string]browserscenario.LockedRevision{}
	for _, component := range lock.Components {
		locked[component.Name] = component
	}
	type repository struct {
		name, root string
	}
	items := []repository{
		{"openudon", roots.openudon}, {"browsertools", roots.browsertools}, {"uws", roots.uws},
		{"udon", roots.udon}, {"browserdriver", roots.browserdriver},
	}
	repositories := make([]RepositoryRevision, 0, 3)
	for _, item := range items {
		commit, err := qualificationCommandOutput(ctx, time.Minute, item.root, []string{"git", "rev-parse", "HEAD"})
		if err != nil || !evidencefile.ValidGitObject(commit) {
			return nil, fmt.Errorf("resolve %s qualification revision", item.name)
		}
		status, err := qualificationCommandOutput(ctx, time.Minute, item.root,
			[]string{"git", "status", "--porcelain=v1", "--untracked-files=all", "--", "."})
		if err != nil || status != "" {
			return nil, fmt.Errorf("%s qualification repository must be clean", item.name)
		}
		if item.name == "openudon" {
			branch, branchErr := qualificationCommandOutput(ctx, time.Minute, item.root, []string{"git", "symbolic-ref", "--quiet", "--short", "HEAD"})
			originMain, originErr := qualificationCommandOutput(ctx, time.Minute, item.root, []string{"git", "rev-parse", "refs/remotes/origin/main"})
			remote, remoteErr := qualificationCommandOutput(ctx, time.Minute, item.root, []string{"git", "ls-remote", "--exit-code", "origin", "refs/heads/main"})
			_, ancestryErr := qualificationCommandOutput(ctx, time.Minute, item.root, []string{"git", "merge-base", "--is-ancestor", originMain, commit})
			if branchErr != nil || branch != "main" || originErr != nil || !evidencefile.ValidGitObject(originMain) ||
				remoteErr != nil || remote != originMain+"\trefs/heads/main" || ancestryErr != nil || originMain == commit {
				return nil, errors.New("OpenUdon qualification revision must remain local and unpublished")
			}
			repositories = append(repositories, RepositoryRevision{Name: item.name, Commit: commit})
			continue
		}
		component := locked[item.name]
		if commit != component.Commit {
			return nil, fmt.Errorf("%s qualification revision does not match the lock", item.name)
		}
		if item.name == "browsertools" || item.name == "uws" {
			remote, err := qualificationCommandOutput(ctx, time.Minute, item.root, []string{"git", "ls-remote", "--exit-code", "origin", "refs/heads/main"})
			if err != nil || remote != component.Commit+"\trefs/heads/main" {
				return nil, fmt.Errorf("%s published qualification revision is not independently resolvable", item.name)
			}
			repositories = append(repositories, RepositoryRevision{
				Name: item.name, Commit: commit, ModuleVersion: component.Version, Published: true,
			})
		}
	}
	return repositories, nil
}

func equalRepositoryRevisions(left, right []RepositoryRevision) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func qualificationCommandOutput(ctx context.Context, timeout time.Duration, directory string, args []string) (string, error) {
	output := &boundedQualificationWriter{remaining: qualificationCommandOutputLimit}
	err := processgroup.Run(ctx, timeout, processgroup.Invocation{
		Args: args, Dir: directory, Env: os.Environ(), Stdout: output, Stderr: io.Discard,
	})
	if err != nil || output.exceeded {
		return "", errors.New("qualification command failed")
	}
	return strings.TrimSpace(output.buffer.String()), nil
}

func runQualificationCommandSilent(ctx context.Context, timeout time.Duration, directory string, args []string) error {
	return processgroup.Run(ctx, timeout, processgroup.Invocation{
		Args: args, Dir: directory, Env: os.Environ(), Stdout: io.Discard, Stderr: io.Discard,
	})
}

type boundedQualificationWriter struct {
	buffer    bytes.Buffer
	remaining int
	exceeded  bool
}

func (writer *boundedQualificationWriter) Write(data []byte) (int, error) {
	length := len(data)
	if length > writer.remaining {
		data = data[:writer.remaining]
		writer.exceeded = true
	}
	_, _ = writer.buffer.Write(data)
	writer.remaining -= len(data)
	return length, nil
}
