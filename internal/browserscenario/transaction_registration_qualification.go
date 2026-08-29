package browserscenario

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/OpenUdon/browsertools/registrationprofile"
	"github.com/OpenUdon/openudon/internal/browsertransaction"
	"github.com/OpenUdon/openudon/internal/browserworkflow"
	"github.com/OpenUdon/openudon/internal/evidencefile"
	icotui "github.com/OpenUdon/openudon/internal/icot/ui"
	"github.com/OpenUdon/openudon/internal/packageartifacts"
	"github.com/OpenUdon/openudon/internal/packagepipeline"
	"github.com/OpenUdon/openudon/internal/registrationattestation"
	"github.com/OpenUdon/openudon/internal/synthesize"
	"github.com/OpenUdon/openudon/internal/trustedrunner"
	"github.com/OpenUdon/openudon/internal/udonrunner"
)

// BRPQualificationEvidence is path-free evidence from one exact real
// Browsertools registration producer and OpenUdon package lifecycle. The
// network counters are independently observed by the loopback fixture.
type BRPQualificationEvidence struct {
	ProducerResultSHA256  string
	TransactionSHA256     string
	PreparationSHA256     string
	QualificationSHA256   string
	GenerationSHA256      string
	SelectionSHA256       string
	PackageSHA256         string
	HandoffSHA256         string
	WorkflowSHA256        string
	AttestationSHA256     string
	ExecutionReportSHA256 string
	EvidenceCount         int
	Methods               []string
	Requests              int
	GETRequests           int
	HEADRequests          int
	MutationRequests      int
	RuntimePOSTRequests   int
	SubmitApproved        bool
	SubmitExecuted        bool
	AccountCreated        bool
	SessionEstablished    bool
	RuntimeSupported      bool
	ExecutorInvoked       bool
}

// RunBRPQualification exercises only a deterministic loopback registration
// page. Authoring remains GET/HEAD-only; the separately attested runtime path
// performs exactly one approved submit against the same fixture.
func RunBRPQualification(ctx context.Context, options Options) (BRPQualificationEvidence, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	now := time.Now().UTC().Round(0)
	if options.Now != nil {
		now = options.Now().UTC().Round(0)
	}
	if now.IsZero() || options.AllowNetwork {
		return BRPQualificationEvidence{}, errors.New("BRP qualification authority is invalid")
	}
	lock, err := LoadCompatibilityLock()
	if err != nil {
		return BRPQualificationEvidence{}, err
	}
	environment, _, _, err := resolveEnvironment(ctx, options, lock, now)
	if err != nil {
		return BRPQualificationEvidence{}, err
	}
	if err := ValidateQualificationBuildInputs(ctx, environment.UdonRepo, lock); err != nil {
		return BRPQualificationEvidence{}, errors.New("BRP qualification build inputs are invalid")
	}
	environment.CommitBoundBuild = true
	executor := &realExecutor{}
	defer executor.Close()
	executor.prepare(ctx, environment, SuiteLoopback)
	if executor.unavailable {
		return BRPQualificationEvidence{}, fmt.Errorf("BRP qualification: %w", ErrSandboxPrerequisiteUnavailable)
	}
	if executor.prepareErr != nil {
		return BRPQualificationEvidence{}, errors.New("BRP qualification dependencies are invalid")
	}
	return executor.runBRPQualification(ctx, environment)
}

func (executor *realExecutor) runBRPQualification(ctx context.Context, environment Environment) (evidence BRPQualificationEvidence, resultErr error) {
	fixture := newRegistrationQualificationFixture()
	defer fixture.Close()
	root, err := os.MkdirTemp(executor.root, "transaction-brp-")
	if err != nil {
		return evidence, errors.New("create BRP qualification root")
	}
	defer func() {
		if cleanupErr := os.RemoveAll(root); cleanupErr != nil && resultErr == nil {
			resultErr = errors.New("remove BRP qualification root")
		}
	}()
	privateRoot := filepath.Join(root, "private")
	scratch := filepath.Join(root, "scratch")
	store := filepath.Join(root, "store")
	for _, directory := range []string{privateRoot, scratch, store} {
		if err := os.Mkdir(directory, 0o700); err != nil {
			return evidence, errors.New("create BRP qualification directory")
		}
	}
	exampleParent := filepath.Join(environment.RepoRoot, "eval", "runs")
	if err := os.MkdirAll(exampleParent, 0o700); err != nil {
		return evidence, errors.New("create BRP qualification example parent")
	}
	exampleDir, err := os.MkdirTemp(exampleParent, ".e11-brp-")
	if err != nil {
		return evidence, errors.New("create BRP qualification example")
	}
	defer func() {
		if cleanupErr := os.RemoveAll(exampleDir); cleanupErr != nil && resultErr == nil {
			resultErr = errors.New("remove BRP qualification example")
		}
	}()
	packageAt := time.Now().UTC().Round(0)
	baselineDir := filepath.Join(root, "baseline")
	if err := copyQualificationBaseline(baselineDir, environment.RepoRoot); err != nil {
		return evidence, errors.New("BRP qualification baseline copy failed")
	}
	if _, err := synthesize.Build(ctx, synthesize.Options{ExampleDir: baselineDir}); err != nil {
		return evidence, errors.New("BRP qualification baseline build failed")
	}
	baseline, err := packagepipeline.PromoteCurrent(ctx, packagepipeline.CurrentOptions{
		ExampleDir: baselineDir,
		Scope:      qualificationBaselineScope, ScratchParent: scratch, StoreDir: store,
	})
	if err != nil {
		return evidence, fmt.Errorf("BRP qualification baseline promotion failed: %s", closedPackageLifecycleFailure(err))
	}
	qualifiedUI, err := icotui.RunRegistrationQualification(ctx, icotui.RegistrationQualificationOptions{
		RepoRoot: environment.RepoRoot, BrowsertoolsExecutable: executor.browsertools,
		ExampleDir: exampleDir, PrivateRoot: privateRoot, ScratchParent: scratch, StoreDir: store, Scope: "qualification/brp",
		ProfileID: "qualification_brp", InitialURL: fixture.URL(), Origin: fixture.Origin(), Now: func() time.Time { return time.Now().UTC() },
	})
	if err != nil || qualifiedUI.Snapshot.Transaction == nil || qualifiedUI.Snapshot.Preparation == nil || qualifiedUI.Snapshot.Promotion == nil || !qualifiedUI.RetainedQuery {
		return evidence, errors.New("BRP iCoT wizard qualification failed")
	}
	authoringNetwork := fixture.Evidence()
	if len(authoringNetwork.Methods) != 2 || authoringNetwork.Methods[0] != http.MethodGet || authoringNetwork.Methods[1] != http.MethodHead ||
		authoringNetwork.MutationRequests != 0 || authoringNetwork.AccountCreated {
		return evidence, errors.New("BRP producer exceeded its GET/HEAD-only authority")
	}
	reviewed := *qualifiedUI.Snapshot.Transaction
	if reviewed.Version != browsertransaction.VersionV2 || reviewed.Session != "" || reviewed.State != browsertransaction.StatePromoted ||
		reviewed.Provenance.ResultVersion != browsertransaction.ResultRegistrationAuthoringV2 {
		return evidence, errors.New("BRP iCoT transaction-v2 transition failed")
	}
	promoted, err := packagepipeline.ReadCurrent(ctx, store)
	if err != nil {
		return evidence, errors.New("BRP promoted package is unavailable")
	}
	runtimeAuthority, err := registrationQualificationAuthority(promoted, qualifiedUI.CanonicalProfile)
	if err != nil {
		return evidence, errors.New("BRP promoted registration authority is invalid")
	}
	selection := promoted.Selection()
	if selection.PriorGenerationSHA256 != baseline.Selection().SelectedGenerationSHA256 || selection.PriorGenerationSHA256 == "" ||
		qualifiedUI.Snapshot.Promotion.PriorGenerationSHA256 != selection.PriorGenerationSHA256 ||
		qualifiedUI.Snapshot.Promotion.SelectionSHA256 != selection.SelectionSHA256 {
		return evidence, errors.New("BRP qualification prior generation was not preserved")
	}
	inspection, err := packagepipeline.InspectSelected(ctx, store, selection.SelectionSHA256)
	if err != nil || taggedQualificationSHA256(inspection.PackageSHA256) != taggedQualificationSHA256(qualifiedUI.Snapshot.Preparation.PackageSHA256) ||
		taggedQualificationSHA256(inspection.HandoffSHA256) != taggedQualificationSHA256(qualifiedUI.Snapshot.Preparation.HandoffSHA256) {
		return evidence, errors.New("BRP selected package review failed")
	}
	packageAt = time.Now().UTC().Round(0)
	approval, err := packagepipeline.ApprovalTemplateSelected(ctx, store, selection.SelectionSHA256, trustedrunner.TemplateOptions{
		State: trustedrunner.StateApprovedForSandbox, Reviewer: "Browser transaction qualification",
		Now: func() time.Time { return packageAt.Add(time.Minute) },
	})
	if err != nil {
		return evidence, errors.New("BRP selected approval failed")
	}
	approvalPath := filepath.Join(root, "approval.json")
	file, err := os.OpenFile(approvalPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return evidence, errors.New("create BRP qualification approval")
	}
	if err := trustedrunner.WriteApproval(file, approval); err != nil {
		_ = file.Close()
		return evidence, errors.New("write BRP qualification approval")
	}
	if err := file.Close(); err != nil {
		return evidence, errors.New("close BRP qualification approval")
	}
	dryRun, err := packagepipeline.RunSelected(ctx, store, selection.SelectionSHA256, trustedrunner.Options{
		Tier: trustedrunner.TierSandbox, ApprovalPath: approvalPath, WorkDir: filepath.Join(root, "dry-run"),
		DryRun: true, Env: []string{}, Now: func() time.Time { return packageAt.Add(2 * time.Minute) },
	})
	if err != nil || dryRun == nil || !dryRun.DryRun || dryRun.PackageSHA256 != selection.PackageSHA256 {
		return evidence, errors.New("BRP selected trusted handoff failed")
	}
	attestationPath, attestationSHA256, err := writeRegistrationQualificationAttestation(
		root, runtimeAuthority.profile, selection.PackageSHA256, runtimeAuthority.operation,
		runtimeAuthority.flow, runtimeAuthority.cleanup, packageAt,
	)
	if err != nil {
		return evidence, errors.New("BRP registration attestation failed")
	}
	runtimeEnv := registrationQualificationRuntimeEnvironment(executor.udon)
	run, runErr := packagepipeline.RunSelected(ctx, store, selection.SelectionSHA256, trustedrunner.Options{
		Tier: trustedrunner.TierSandbox, ApprovalPath: approvalPath, WorkDir: filepath.Join(root, "live-run"),
		Env: runtimeEnv, Now: func() time.Time { return packageAt.Add(3 * time.Minute) }, Stdout: io.Discard, Stderr: io.Discard,
		BrowserDriver: executor.node, BrowserDriverArgs: []string{executor.driverEntry, "--headed"},
		RegistrationAttestationPath: attestationPath, RegistrationSubmitApproval: runtimeAuthority.operation,
	})
	if runErr != nil || run == nil || run.DryRun || run.PackageSHA256 != selection.PackageSHA256 {
		return evidence, errors.New("BRP attested runtime execution failed")
	}
	var runEvidence trustedrunner.RunEvidence
	runEvidenceData, _, err := evidencefile.ReadRegular(run.RunEvidencePath, evidencefile.DefaultMaxBytes)
	if err != nil || evidencefile.DecodeStrict(runEvidenceData, &runEvidence) != nil || !runEvidence.Executor.Invoked || runEvidence.Executor.ReportSHA256 == "" {
		return evidence, errors.New("BRP execution evidence is invalid")
	}
	workflowSHA256, err := digestQualificationFile(filepath.Join(exampleDir, "workflows", "workflow.uws.yaml"))
	if err != nil {
		return evidence, err
	}
	network := fixture.Evidence()
	if len(network.Methods) != 3 || network.Methods[0] != http.MethodGet || network.Methods[1] != http.MethodHead || network.Methods[2] != http.MethodPost ||
		network.POSTRequests != 1 || network.MutationRequests != 1 || !network.AccountCreated {
		return evidence, errors.New("BRP runtime exceeded its one-POST authority")
	}
	submitApproved := runEvidence.Browser != nil && runEvidence.Browser.Protocol == "v4" &&
		len(runEvidence.Browser.ApprovedRegistration) == 1 && runEvidence.Browser.ApprovedRegistration[0] == runtimeAuthority.operation
	evidence = BRPQualificationEvidence{
		ProducerResultSHA256: taggedQualificationSHA256(reviewed.Provenance.ResultSHA256), TransactionSHA256: taggedQualificationSHA256(qualifiedUI.Snapshot.TransactionSHA256),
		PreparationSHA256: taggedQualificationSHA256(qualifiedUI.Snapshot.Preparation.PreparationSHA256), QualificationSHA256: taggedQualificationSHA256(qualifiedUI.Snapshot.Preparation.QualificationSHA256),
		GenerationSHA256: taggedQualificationSHA256(selection.SelectedGenerationSHA256), SelectionSHA256: taggedQualificationSHA256(selection.SelectionSHA256),
		PackageSHA256: taggedQualificationSHA256(selection.PackageSHA256), HandoffSHA256: taggedQualificationSHA256(inspection.HandoffSHA256),
		WorkflowSHA256: taggedQualificationSHA256(workflowSHA256), AttestationSHA256: taggedQualificationSHA256(attestationSHA256),
		ExecutionReportSHA256: taggedQualificationSHA256(runEvidence.Executor.ReportSHA256), EvidenceCount: 11,
		Methods: authoringNetwork.Methods, Requests: authoringNetwork.Requests, GETRequests: authoringNetwork.GETRequests, HEADRequests: authoringNetwork.HEADRequests,
		MutationRequests: authoringNetwork.MutationRequests, RuntimePOSTRequests: network.POSTRequests,
		SubmitApproved: submitApproved, SubmitExecuted: network.POSTRequests == 1, AccountCreated: network.AccountCreated,
		SessionEstablished: false, RuntimeSupported: true, ExecutorInvoked: runEvidence.Executor.Invoked,
	}
	if err := ValidateBRPQualificationEvidence(evidence); err != nil {
		return BRPQualificationEvidence{}, err
	}
	return evidence, nil
}

// ValidateBRPQualificationEvidence enforces a zero-submit authoring posture,
// exactly one separately attested runtime submit, and all lifecycle digests.
func ValidateBRPQualificationEvidence(evidence BRPQualificationEvidence) error {
	values := []string{
		evidence.ProducerResultSHA256, evidence.TransactionSHA256, evidence.PreparationSHA256,
		evidence.QualificationSHA256, evidence.GenerationSHA256, evidence.SelectionSHA256,
		evidence.PackageSHA256, evidence.HandoffSHA256, evidence.WorkflowSHA256,
		evidence.AttestationSHA256, evidence.ExecutionReportSHA256,
	}
	if err := validateQualificationDigests(evidence.EvidenceCount, values); err != nil {
		return errors.New("BRP qualification digest evidence is invalid")
	}
	if len(evidence.Methods) != 2 || evidence.Methods[0] != http.MethodGet || evidence.Methods[1] != http.MethodHead ||
		evidence.GETRequests <= 0 || evidence.HEADRequests <= 0 || evidence.Requests != evidence.GETRequests+evidence.HEADRequests ||
		evidence.MutationRequests != 0 || evidence.RuntimePOSTRequests != 1 || !evidence.SubmitApproved || !evidence.SubmitExecuted ||
		!evidence.AccountCreated || evidence.SessionEstablished || !evidence.RuntimeSupported || !evidence.ExecutorInvoked {
		return errors.New("BRP qualification authoring/runtime evidence is invalid")
	}
	return nil
}

type registrationQualificationNetworkEvidence struct {
	Methods          []string
	Requests         int
	GETRequests      int
	HEADRequests     int
	POSTRequests     int
	MutationRequests int
	AccountCreated   bool
}

type registrationQualificationFixture struct {
	server         *httptest.Server
	mu             sync.Mutex
	counts         map[string]int
	mutated        bool
	accountCreated bool
}

type registrationQualificationRuntime struct {
	operation string
	flow      string
	cleanup   string
	profile   []byte
}

func registrationQualificationAuthority(promoted packagepipeline.Promoted, canonicalProfile []byte) (registrationQualificationRuntime, error) {
	files := promoted.Files()
	data, ok := files[packageartifacts.BrowserRegistrationReviewPath]
	if !ok {
		return registrationQualificationRuntime{}, errors.New("registration review is unavailable")
	}
	var review struct {
		Version string `json:"version"`
		Calls   []struct {
			Step                string            `json:"step"`
			Source              string            `json:"source"`
			Flow                string            `json:"flow"`
			CredentialBindings  map[string]string `json:"credential_bindings"`
			Approval            string            `json:"approval"`
			DuplicatePrevention string            `json:"duplicate_prevention"`
			OnDuplicate         string            `json:"on_duplicate"`
			AmbiguousOutcome    string            `json:"ambiguous_outcome"`
			CleanupDisposition  string            `json:"cleanup_disposition"`
			Timeout             float64           `json:"timeout"`
		} `json:"registration_calls"`
		Sources []json.RawMessage `json:"sources"`
	}
	if evidencefile.DecodeStrict(data, &review) != nil || review.Version != "openudon.browser-registration-review.v1" || len(review.Calls) != 1 {
		return registrationQualificationRuntime{}, errors.New("registration review is invalid")
	}
	call := review.Calls[0]
	operation := browserworkflow.RuntimeOperationID(call.Step)
	if operation == "" || browserworkflow.RuntimeOperationID(call.Approval) != operation || call.Flow != "create_dedicated_test_user" || call.CleanupDisposition != "delete_separately" {
		return registrationQualificationRuntime{}, errors.New("registration call authority is invalid")
	}
	profile, ok := files[call.Source]
	if !ok {
		return registrationQualificationRuntime{}, errors.New("registration profile is unavailable")
	}
	packaged, err := registrationprofile.Parse(profile)
	if err != nil {
		return registrationQualificationRuntime{}, err
	}
	reviewed, err := registrationprofile.Parse(canonicalProfile)
	if err != nil {
		return registrationQualificationRuntime{}, err
	}
	packagedDigest, err := registrationprofile.Digest(packaged)
	if err != nil {
		return registrationQualificationRuntime{}, err
	}
	reviewedDigest, err := registrationprofile.Digest(reviewed)
	if err != nil || reviewedDigest != packagedDigest {
		return registrationQualificationRuntime{}, errors.New("registration profile changed after UI review")
	}
	return registrationQualificationRuntime{
		operation: operation, flow: call.Flow, cleanup: call.CleanupDisposition, profile: append([]byte(nil), profile...),
	}, nil
}

func newRegistrationQualificationFixture() *registrationQualificationFixture {
	fixture := &registrationQualificationFixture{counts: map[string]int{}}
	fixture.server = httptest.NewServer(http.HandlerFunc(fixture.serveHTTP))
	return fixture
}

func (fixture *registrationQualificationFixture) serveHTTP(writer http.ResponseWriter, request *http.Request) {
	fixture.mu.Lock()
	fixture.counts[request.Method]++
	if request.Method != http.MethodGet && request.Method != http.MethodHead {
		fixture.mutated = true
	}
	fixture.mu.Unlock()
	if request.Method == http.MethodPost && request.URL.Path == "/registration-complete" {
		if err := request.ParseForm(); err != nil || request.Form.Get("identifier") != "dedicated-test@example.test" || request.Form.Get("password") != "qualification-password-value" {
			http.Error(writer, "invalid registration", http.StatusUnauthorized)
			return
		}
		fixture.mu.Lock()
		fixture.accountCreated = true
		fixture.mu.Unlock()
		writer.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = writer.Write([]byte(`<!doctype html><html><body><p role="status" aria-label="Registration complete">Registration complete</p></body></html>`))
		return
	}
	if request.Method != http.MethodGet && request.Method != http.MethodHead {
		http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	if request.Method == http.MethodHead {
		return
	}
	_, _ = writer.Write([]byte(`<!doctype html><html><body><main><h1>Create account</h1><form method="post" action="/registration-complete"><label>Email<input name="identifier" autocomplete="email"></label><label>Password<input name="password" type="password"></label><button type="submit">Register</button><p role="status" aria-label="Registration complete">Registration proof marker</p></form></main></body></html>`))
}

func (fixture *registrationQualificationFixture) URL() string {
	return fixture.server.URL + "/register?action=startnew"
}
func (fixture *registrationQualificationFixture) Origin() string { return fixture.server.URL }
func (fixture *registrationQualificationFixture) Close()         { fixture.server.Close() }

func (fixture *registrationQualificationFixture) Evidence() registrationQualificationNetworkEvidence {
	fixture.mu.Lock()
	defer fixture.mu.Unlock()
	methods := make([]string, 0, len(fixture.counts))
	requests, mutations := 0, 0
	for method, count := range fixture.counts {
		if count <= 0 {
			continue
		}
		methods = append(methods, method)
		requests += count
		if method != http.MethodGet && method != http.MethodHead {
			mutations += count
		}
	}
	sort.Strings(methods)
	return registrationQualificationNetworkEvidence{
		Methods: methods, Requests: requests, GETRequests: fixture.counts[http.MethodGet], HEADRequests: fixture.counts[http.MethodHead],
		POSTRequests: fixture.counts[http.MethodPost], MutationRequests: mutations, AccountCreated: fixture.accountCreated,
	}
}

func writeRegistrationQualificationAttestation(root string, profile []byte, packageSHA256, operation, flow, cleanup string, now time.Time) (string, string, error) {
	value, err := registrationprofile.Parse(profile)
	if err != nil {
		return "", "", err
	}
	profileSHA256, err := registrationprofile.Digest(value)
	if err != nil {
		return "", "", err
	}
	artifact := registrationattestation.Artifact{
		Version: registrationattestation.Version, PackageSHA256: taggedQualificationSHA256(packageSHA256), ProfileSHA256: profileSHA256,
		Operation: operation, Flow: flow, PriorAttempts: 0, DedicatedTest: true,
		CleanupDisposition: cleanup, Reviewer: "Browser transaction qualification",
		ExpiresAt: now.Add(time.Hour).UTC().Truncate(time.Second).Format(time.RFC3339),
	}
	data, err := json.Marshal(artifact)
	if err != nil {
		return "", "", err
	}
	path := filepath.Join(root, "registration-attestation.json")
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", "", err
	}
	if _, err := file.Write(append(data, '\n')); err != nil {
		_ = file.Close()
		return "", "", err
	}
	if err := file.Close(); err != nil {
		return "", "", err
	}
	_, digest, err := registrationattestation.ReadOutsideRepo(path, filepath.Join(root, "store"), registrationattestation.Expected{
		PackageSHA256: artifact.PackageSHA256, ProfileSHA256: profileSHA256, Operation: artifact.Operation,
		Flow: artifact.Flow, CleanupDisposition: artifact.CleanupDisposition,
	}, now)
	return path, digest, err
}

func registrationQualificationRuntimeEnvironment(udonPath string) []string {
	values := []string{
		"OPENUDON_EXECUTOR=" + udonPath,
		udonrunner.CredentialEnvironmentName("registration_identifier") + "=dedicated-test@example.test",
		udonrunner.CredentialEnvironmentName("reg_password") + "=qualification-password-value",
	}
	for _, name := range []string{"PATH", "HOME", "TMPDIR", "TMP", "TEMP", "DISPLAY", "WAYLAND_DISPLAY", "XAUTHORITY", "DBUS_SESSION_BUS_ADDRESS", "LANG", "LC_ALL", "LC_CTYPE", "PLAYWRIGHT_BROWSERS_PATH"} {
		if value := os.Getenv(name); value != "" {
			values = append(values, name+"="+value)
		}
	}
	return values
}
