package browsertransactioneval

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
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
	makeExecutable, err := exec.LookPath("make")
	if err != nil {
		return nil, errors.New("browser transaction qualification toolchain is unavailable")
	}
	goExecutable, err := exec.LookPath("go")
	if err != nil {
		return nil, errors.New("browser transaction qualification toolchain is unavailable")
	}
	if err := runQualificationCommandSilent(ctx, 10*time.Minute, roots.openudon,
		[]string{makeExecutable, "browser-transaction-adversarial"}, qualificationAdversarialEnvironment(roots.browsertools, goExecutable)); err != nil {
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
		{ID: GateBRPRuntime, Status: StatusPass, EvidenceCount: 4},
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
			RegistrationRuntimePostRequests:   brp.RuntimePOSTRequests,
			RegistrationSubmitApproved:        brp.SubmitApproved,
			AccountCreated:                    brp.AccountCreated, ExecutorInvokedForRegistration: brp.ExecutorInvoked,
			RegistrationSessionEstablished: brp.SessionEstablished,
			RegistrationResultVersion:      "browsertools.registration-authoring.v2",
			TransactionVersion:             "openudon.browser-profile-transaction.v2",
			BrowserDriverProtocol:          "udon.browser-driver.v4",
			UdonExecutionReportVersion:     "udon.execution-report.v3",
			ContainsPrivateMaterial:        false, ValueFree: true,
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
		brp.AttestationSHA256, brp.ExecutionReportSHA256,
	}
	artifacts := make([]ArtifactDigest, 0, len(bapArtifactKindOrder)+len(brpArtifactKindOrder))
	for caseIndex, caseID := range []string{CaseBAPBCP, CaseBRP} {
		values := bapValues
		kinds := bapArtifactKindOrder
		if caseIndex == 1 {
			values = brpValues
			kinds = brpArtifactKindOrder
		}
		for index, kind := range kinds {
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
		{"openudon", roots.openudon}, {"browsertools", roots.browsertools}, {"browserdriver", roots.browserdriver},
		{"udon", roots.udon}, {"uws", roots.uws},
	}
	repositories := make([]RepositoryRevision, 0, len(items))
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
		component := locked[item.name]
		if item.name != "openudon" && commit != component.Commit {
			return nil, fmt.Errorf("%s qualification revision does not match the lock", item.name)
		}
		branch, branchErr := qualificationCommandOutput(ctx, time.Minute, item.root, []string{"git", "symbolic-ref", "--quiet", "--short", "HEAD"})
		originMain, originErr := qualificationCommandOutput(ctx, time.Minute, item.root, []string{"git", "rev-parse", "refs/remotes/origin/main"})
		remote, remoteErr := qualificationCommandOutput(ctx, time.Minute, item.root, []string{"git", "ls-remote", "--exit-code", "origin", "refs/heads/main"})
		_, localDescendantErr := qualificationCommandOutput(ctx, time.Minute, item.root, []string{"git", "merge-base", "--is-ancestor", originMain, commit})
		_, publishedErr := qualificationCommandOutput(ctx, time.Minute, item.root, []string{"git", "merge-base", "--is-ancestor", commit, originMain})
		published, related := qualificationPublicationState(localDescendantErr == nil, publishedErr == nil)
		if branchErr != nil || branch != "main" || originErr != nil || !evidencefile.ValidGitObject(originMain) ||
			remoteErr != nil || remote != originMain+"\trefs/heads/main" || !related {
			return nil, fmt.Errorf("%s qualification revision must be published main or local main descended from the independently resolved origin", item.name)
		}
		if item.name == "uws" && !published {
			return nil, errors.New("UWS qualification revision must remain the unchanged published lock")
		}
		repositories = append(repositories, RepositoryRevision{
			Name: item.name, Commit: commit, ModuleVersion: component.Version, Published: published,
		})
	}
	return repositories, nil
}

func qualificationPublicationState(localDescendsOrigin, originDescendsLocal bool) (published, related bool) {
	if !localDescendsOrigin && !originDescendsLocal {
		return false, false
	}
	return originDescendsLocal, true
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
	if len(args) > 0 && args[0] == "git" {
		args = append([]string{"git", "--no-replace-objects"}, args[1:]...)
	}
	output := &boundedQualificationWriter{remaining: qualificationCommandOutputLimit}
	err := processgroup.Run(ctx, timeout, processgroup.Invocation{
		Args: args, Dir: directory, Env: qualificationGitEnvironment(), Stdout: output, Stderr: io.Discard,
	})
	if err != nil || output.exceeded {
		return "", errors.New("qualification command failed")
	}
	return strings.TrimSpace(output.buffer.String()), nil
}

func runQualificationCommandSilent(ctx context.Context, timeout time.Duration, directory string, args, environment []string) error {
	return processgroup.Run(ctx, timeout, processgroup.Invocation{
		Args: args, Dir: directory, Env: environment, Stdout: io.Discard, Stderr: io.Discard,
	})
}

func qualificationAdversarialEnvironment(browsertoolsRepo, goExecutable string) []string {
	blocked := map[string]bool{
		"BASH_ENV": true, "ENV": true, "GO": true, "GO111MODULE": true, "GOARCH": true,
		"CGO_ENABLED": true, "GOENV": true, "GOEXPERIMENT": true, "GOFLAGS": true,
		"GOROOT": true, "GOTOOLCHAIN": true, "GOOS": true, "GOWORK": true,
		"GNUMAKEFLAGS": true, "MAKEFILES": true, "MAKEFLAGS": true, "MFLAGS": true,
		"OPENUDON_BROWSERTOOLS_REPO": true, "PATH": true,
	}
	environment := make([]string, 0, len(os.Environ())+3)
	for _, item := range os.Environ() {
		name, _, ok := strings.Cut(item, "=")
		if ok && !blocked[name] {
			environment = append(environment, item)
		}
	}
	path := filepath.Dir(goExecutable)
	if inherited := os.Getenv("PATH"); inherited != "" {
		path += string(os.PathListSeparator) + inherited
	}
	return append(environment,
		"GOENV=off",
		"OPENUDON_BROWSERTOOLS_REPO="+browsertoolsRepo,
		"PATH="+path,
	)
}

func qualificationGitEnvironment() []string {
	blocked := map[string]bool{
		"GIT_ALTERNATE_OBJECT_DIRECTORIES": true, "GIT_CEILING_DIRECTORIES": true,
		"GIT_COMMON_DIR": true, "GIT_DIR": true, "GIT_DISCOVERY_ACROSS_FILESYSTEM": true,
		"GIT_INDEX_FILE": true, "GIT_NAMESPACE": true, "GIT_NO_REPLACE_OBJECTS": true,
		"GIT_OBJECT_DIRECTORY": true, "GIT_REPLACE_REF_BASE": true, "GIT_WORK_TREE": true,
		"GIT_CONFIG_GLOBAL": true, "GIT_CONFIG_NOSYSTEM": true, "GIT_CONFIG_SYSTEM": true,
	}
	environment := make([]string, 0, len(os.Environ()))
	for _, item := range os.Environ() {
		name, _, ok := strings.Cut(item, "=")
		if ok && !blocked[name] && !strings.HasPrefix(name, "GIT_CONFIG_") {
			environment = append(environment, item)
		}
	}
	return environment
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
