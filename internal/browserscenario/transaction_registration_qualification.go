package browserscenario

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/OpenUdon/browsertools/registrationprofile"
	"github.com/OpenUdon/openudon/internal/browsercandidate"
	"github.com/OpenUdon/openudon/internal/browsertransaction"
	"github.com/OpenUdon/openudon/internal/icot/artifactwriter"
	"github.com/OpenUdon/openudon/internal/icot/browserauthor"
	"github.com/OpenUdon/openudon/internal/icot/elicitor"
	icotengine "github.com/OpenUdon/openudon/internal/icot/engine"
	"github.com/OpenUdon/openudon/internal/packagepipeline"
	"github.com/OpenUdon/openudon/internal/synthesize"
	"github.com/OpenUdon/openudon/internal/trustedrunner"
	"github.com/OpenUdon/openudon/internal/udonrunner"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

// BRPQualificationEvidence is path-free evidence from one exact real
// Browsertools registration producer and OpenUdon package lifecycle. The
// network counters are independently observed by the loopback fixture.
type BRPQualificationEvidence struct {
	ProducerResultSHA256 string
	TransactionSHA256    string
	PreparationSHA256    string
	QualificationSHA256  string
	GenerationSHA256     string
	SelectionSHA256      string
	PackageSHA256        string
	HandoffSHA256        string
	WorkflowSHA256       string
	EvidenceCount        int
	Methods              []string
	Requests             int
	GETRequests          int
	HEADRequests         int
	MutationRequests     int
	SubmitExecuted       bool
	AccountCreated       bool
	SessionEstablished   bool
	RuntimeSupported     bool
	ExecutorInvoked      bool
}

// RunBRPQualification exercises only a deterministic loopback registration
// page. It observes and reviews a submit control but never activates it.
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
	exampleDir := filepath.Join(root, "example")
	privateRoot := filepath.Join(root, "private")
	scratch := filepath.Join(root, "scratch")
	store := filepath.Join(root, "store")
	for _, directory := range []string{exampleDir, privateRoot, scratch, store} {
		if err := os.Mkdir(directory, 0o700); err != nil {
			return evidence, errors.New("create BRP qualification directory")
		}
	}
	profile, err := registrationQualificationProfile(fixture.Origin(), time.Now().UTC().Truncate(time.Second))
	if err != nil {
		return evidence, errors.New("construct BRP qualification profile")
	}
	candidate, err := runRegistrationQualificationProducer(ctx, executor.browsertools, privateRoot, fixture, profile)
	if err != nil || candidate == nil {
		return evidence, fmt.Errorf("BRP producer or adoption failed: %s", closedBRPProducerFailure(err))
	}
	reviewed, err := candidate.ReviewedTransaction()
	if err != nil || reviewed.Session != "" || reviewed.State != browsertransaction.StateReviewed {
		return evidence, errors.New("BRP review transition failed")
	}
	transactionSHA256, err := browsertransaction.Digest(reviewed)
	if err != nil {
		return evidence, err
	}
	packageAt := time.Now().UTC().Round(0)
	if err := materializeBRPPackage(exampleDir, candidate, packageAt); err != nil {
		return evidence, errors.New("materialize BRP qualification package")
	}
	built, quality, err := synthesize.PackageFromIntent(ctx, synthesize.Options{ExampleDir: exampleDir})
	if err != nil || built == nil || quality == nil {
		return evidence, errors.New("BRP qualification package construction failed")
	}
	if !quality.Passed() {
		return evidence, fmt.Errorf("BRP qualification package quality failed: %s", closedQualityFailureIDs(quality))
	}
	prepared, err := packagepipeline.PrepareCurrent(ctx, packagepipeline.PrepareOptions{ExampleDir: exampleDir, Scope: "qualification/brp"})
	if err != nil {
		return evidence, errors.New("BRP qualification package preparation failed")
	}
	qualified, err := packagepipeline.Qualify(ctx, prepared, packagepipeline.QualifyOptions{ScratchParent: scratch, Now: packageAt})
	if err != nil {
		if code, ok := packagepipeline.QualificationFailureCode(err); ok {
			return evidence, fmt.Errorf("BRP qualification package qualification failed: %s", code)
		}
		return evidence, errors.New("BRP qualification package qualification failed: unclassified")
	}
	baselineDir := filepath.Join(root, "baseline")
	if err := os.CopyFS(baselineDir, os.DirFS(filepath.Join(environment.RepoRoot, "examples", "support-priority-routing"))); err != nil {
		return evidence, errors.New("BRP qualification baseline copy failed")
	}
	if _, err := synthesize.Build(ctx, synthesize.Options{ExampleDir: baselineDir}); err != nil {
		return evidence, errors.New("BRP qualification baseline build failed")
	}
	baseline, err := packagepipeline.PromoteCurrent(ctx, packagepipeline.CurrentOptions{
		ExampleDir: baselineDir,
		Scope:      "examples/support-priority-routing", ScratchParent: scratch, StoreDir: store,
	})
	if err != nil {
		return evidence, fmt.Errorf("BRP qualification baseline promotion failed: %s", closedPackageLifecycleFailure(err))
	}
	promoted, err := packagepipeline.Promote(ctx, qualified, packagepipeline.PromotionOptions{StoreDir: store})
	if err != nil {
		return evidence, fmt.Errorf("BRP qualification package promotion failed: %s", closedPackageLifecycleFailure(err))
	}
	selection := promoted.Selection()
	if selection.PriorGenerationSHA256 != baseline.Selection().SelectedGenerationSHA256 || selection.PriorGenerationSHA256 == "" {
		return evidence, errors.New("BRP qualification prior generation was not preserved")
	}
	inspection, err := packagepipeline.InspectSelected(ctx, store, selection.SelectionSHA256)
	if err != nil || inspection.PackageSHA256 != prepared.Manifest().PackageSHA256 || inspection.HandoffSHA256 != prepared.Manifest().HandoffSHA256 {
		return evidence, errors.New("BRP selected package review failed")
	}
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
	executorInvoked := false
	if result, runErr := packagepipeline.RunSelected(ctx, store, selection.SelectionSHA256, trustedrunner.Options{
		Tier: trustedrunner.TierSandbox, ApprovalPath: approvalPath, WorkDir: filepath.Join(root, "live-run"),
		Env: []string{}, Now: func() time.Time { return packageAt.Add(3 * time.Minute) },
		Invoke: func(context.Context, udonrunner.Invocation) error {
			executorInvoked = true
			return errors.New("registration executor sentinel invoked")
		},
	}); runErr == nil || result != nil || executorInvoked || !strings.Contains(runErr.Error(), "browser registration execution is unsupported") {
		return evidence, errors.New("BRP runtime rejection boundary failed")
	}
	workflowSHA256, err := digestQualificationFile(built.UWSPath)
	if err != nil {
		return evidence, err
	}
	network := fixture.Evidence()
	evidence = BRPQualificationEvidence{
		ProducerResultSHA256: taggedQualificationSHA256(reviewed.Provenance.ResultSHA256), TransactionSHA256: taggedQualificationSHA256(transactionSHA256),
		PreparationSHA256: taggedQualificationSHA256(prepared.Manifest().ManifestSHA256), QualificationSHA256: taggedQualificationSHA256(qualified.Report().QualificationSHA256),
		GenerationSHA256: taggedQualificationSHA256(selection.SelectedGenerationSHA256), SelectionSHA256: taggedQualificationSHA256(selection.SelectionSHA256),
		PackageSHA256: taggedQualificationSHA256(selection.PackageSHA256), HandoffSHA256: taggedQualificationSHA256(inspection.HandoffSHA256),
		WorkflowSHA256: taggedQualificationSHA256(workflowSHA256), EvidenceCount: 9,
		Methods: network.Methods, Requests: network.Requests, GETRequests: network.GETRequests, HEADRequests: network.HEADRequests,
		MutationRequests: network.MutationRequests, SubmitExecuted: false, AccountCreated: network.AccountCreated,
		SessionEstablished: false, RuntimeSupported: false, ExecutorInvoked: executorInvoked,
	}
	if err := ValidateBRPQualificationEvidence(evidence); err != nil {
		return BRPQualificationEvidence{}, err
	}
	return evidence, nil
}

func runRegistrationQualificationProducer(ctx context.Context, executable, privateRoot string, fixture *registrationQualificationFixture, profile []byte) (*browsercandidate.Registration, error) {
	session, err := browserauthor.StartExternalRegistration(ctx, browserauthor.RegistrationConfig{
		PrivateRoot: privateRoot, TransactionID: "qualification-brp", OperatorIdle: time.Minute, Absolute: 5 * time.Minute,
	}, executable)
	if err != nil {
		return nil, errors.New("worker_start")
	}
	defer session.Cancel()
	if _, err := awaitRegistrationQualificationEvent(ctx, session, "ready"); err != nil {
		return nil, errors.New("worker_ready")
	}
	if err := session.Send(ctx, browserauthor.RegistrationCommand{
		Type: "start", ProfileID: "loopback_registration", URL: fixture.URL(), Origins: []string{fixture.Origin()},
	}); err != nil {
		return nil, errors.New("start_write")
	}
	if _, err := awaitRegistrationQualificationEvent(ctx, session, "observing"); err != nil {
		return nil, errors.New("start_response")
	}
	if err := session.Send(ctx, browserauthor.RegistrationCommand{Type: "observe"}); err != nil {
		return nil, errors.New("first_observe_write")
	}
	if event, err := awaitRegistrationQualificationEvent(ctx, session, "observation"); err != nil {
		return nil, errors.New("first_observe_response_" + closedBRPControllerFailure(event.ErrorCode))
	}
	if err := session.Send(ctx, browserauthor.RegistrationCommand{Type: "navigate", Method: http.MethodHead, URL: fixture.URL()}); err != nil {
		return nil, errors.New("head_write")
	}
	if _, err := awaitRegistrationQualificationEvent(ctx, session, "observing"); err != nil {
		return nil, errors.New("head_response")
	}
	if err := session.Send(ctx, browserauthor.RegistrationCommand{Type: "observe"}); err != nil {
		return nil, errors.New("second_observe_write")
	}
	observed, err := awaitRegistrationQualificationEvent(ctx, session, "observation")
	if err != nil || observed.Observation == nil {
		return nil, errors.New("second_observe_response")
	}
	candidateID := ""
	for _, candidate := range observed.Observation.Candidates {
		if candidate.Role == "button" && candidate.Label == "Register" && candidate.Matches == 1 {
			if candidateID != "" {
				return nil, errors.New("submit_candidate_ambiguous")
			}
			candidateID = candidate.ID
		}
	}
	if candidateID == "" {
		return nil, errors.New("submit_candidate_unavailable")
	}
	bindings := []browsertransaction.CredentialBinding{
		{Slot: "identifier", Binding: "registration_identifier"},
		{Slot: "password", Binding: "reg_password"},
	}
	if err := session.Send(ctx, browserauthor.RegistrationCommand{
		Type: "review", Confirmed: true, Profile: profile, CandidateIDs: []string{candidateID},
		Flow: "create_dedicated_test_user", CleanupDisposition: "delete_separately", CredentialBindings: bindings,
	}); err != nil {
		return nil, errors.New("review_write")
	}
	if _, err := awaitRegistrationQualificationEvent(ctx, session, "reviewed"); err != nil {
		return nil, errors.New("review_response")
	}
	if err := session.Send(ctx, browserauthor.RegistrationCommand{Type: "finish", Confirmed: true}); err != nil {
		return nil, errors.New("finish_write")
	}
	if _, err := awaitRegistrationQualificationEvent(ctx, session, "closed"); err != nil {
		return nil, errors.New("finish_response")
	}
	adopted, err := awaitRegistrationQualificationEvent(ctx, session, "candidate")
	if err != nil || adopted.Candidate == nil {
		return nil, errors.New("candidate_adoption")
	}
	return adopted.Candidate, nil
}

func closedBRPProducerFailure(err error) string {
	if err == nil {
		return "candidate_unavailable"
	}
	code := err.Error()
	for _, allowed := range []string{
		"worker_start", "worker_ready", "start_write", "start_response",
		"first_observe_write", "first_observe_response", "head_write", "head_response",
		"second_observe_write", "second_observe_response", "submit_candidate_unavailable",
		"submit_candidate_ambiguous", "review_write", "review_response", "finish_write",
		"finish_response", "candidate_adoption",
	} {
		if code == allowed {
			return code
		}
	}
	if suffix, ok := strings.CutPrefix(code, "first_observe_response_"); ok && suffix == closedBRPControllerFailure(suffix) {
		return code
	}
	return "unclassified"
}

func closedBRPControllerFailure(code string) string {
	for _, allowed := range []string{
		"protocol_negotiation", "invalid_response", "worker_write", "protocol_mismatch",
		"malformed_diagnostic", "worker_failed", "worker_protocol", "worker_exit",
		"review_missing", "candidate_rejected", "worker_teardown", "operator_idle_timeout",
	} {
		if code == allowed {
			return code
		}
	}
	return "unclassified"
}

func awaitRegistrationQualificationEvent(ctx context.Context, session *browserauthor.RegistrationSession, state string) (browserauthor.RegistrationEvent, error) {
	select {
	case event, ok := <-session.Events():
		if !ok || event.State != state {
			return event, errors.New("BRP qualification producer transition failed")
		}
		return event, nil
	case <-ctx.Done():
		return browserauthor.RegistrationEvent{}, errors.New("BRP qualification producer timed out")
	}
}

func registrationQualificationProfile(origin string, at time.Time) ([]byte, error) {
	value, err := registrationprofile.Parse([]byte(fmt.Sprintf(`profile: uws.browser-registration.1.0
info:
  title: Loopback registration qualification
  applicationOrigins: [%q]
  registrationOrigins: [%q]
observationKind: accessibility_snapshot
evidence: {learnedAt: %q, source: deterministic_loopback}
confidence: high
expiresAfter: P30D
verification: {lastVerifiedAt: %q}
credentialSlots:
  identifier: {kind: identifier}
  password: {kind: password}
flows:
  create_dedicated_test_user:
    sequence:
      - navigate: %s/register
      - type_credential: {locator: {role: textbox, name: Email}, slot: identifier}
      - type_credential: {locator: {role: textbox, name: Password}, slot: password}
      - submit: {locator: {role: button, name: Register}}
      - wait_for: {locator: {role: status, name: Registration complete}}
    effects: [creates_account]
    confirmationPolicy: {required: true}
    success: {origin: %s, path: /registration-complete, locator: {role: status, name: Registration complete}}
`, origin, origin, at.Format(time.RFC3339), at.Format(time.RFC3339), origin, origin)))
	if err != nil {
		return nil, err
	}
	return registrationprofile.MarshalJSON(value)
}

func materializeBRPPackage(exampleDir string, candidate *browsercandidate.Registration, at time.Time) error {
	input, err := icotengine.RegistrationVirtualBrowserTransaction(candidate, true)
	if err != nil {
		return err
	}
	discovery, err := elicitor.DiscoverVirtualBrowserSources([]elicitor.VirtualBrowserTransactionInput{input}, at)
	if err != nil || len(discovery.Candidates) != 1 {
		return errors.New("BRP virtual source discovery failed")
	}
	session, err := elicitor.SelectVirtualBrowserSources(elicitor.Session{}, discovery, []string{discovery.Candidates[0].ID})
	if err != nil || len(session.SourcePlan) != 1 || session.SourcePlan[0].Kind != "browser-registration" {
		return errors.New("BRP virtual source selection failed")
	}
	bindings := transactionBindings(input.Transaction)
	timeout := 300.0
	intent := &rollout.Intent{
		Workflow: &rollout.WorkflowMeta{Name: "browser_registration_qualification", Description: "Prepare one reviewed loopback registration without submitting it."},
		Steps: []*rollout.Step{{
			Name: "register", Type: "browser_registration", Do: "Create one account only after separate exact approval.",
			Source: session.SourcePlan[0].TargetPath, RegistrationFlow: candidate.Flow(), RegistrationApproval: "register",
			DuplicatePrevention: "operator_attestation", OnDuplicate: "fail", AmbiguousOutcome: "stop_without_retry",
			CleanupDisposition: candidate.CleanupDisposition(), CredentialBindings: bindings, Timeout: &timeout,
		}},
	}
	intentHCL, err := rollout.RenderIntentHCL(intent)
	if err != nil {
		return err
	}
	session.Version = elicitor.SessionVersion
	session.Intent = *intent
	session.BrowserRoute = "browser"
	session.BrowserSession = "none"
	for _, binding := range input.Transaction.CredentialBindings {
		session.Credentials = append(session.Credentials, binding.Binding)
	}
	sort.Strings(session.Credentials)
	session.CredentialsSet = true
	session.Safety = "Generate and validate only; account creation is side-effectful and requires separate exact approval plus a sandbox proof run through the trusted runtime."
	session.SafetySet = true
	session.Fallback = "Stop without submission or account creation when review, qualification, or runtime support is unavailable."
	session.FallbackSet = true
	prepared, err := artifactwriter.Prepare(exampleDir, elicitor.Artifacts{
		ProjectMD: brpQualificationProject(), IntentHCL: intentHCL, Session: session,
	}, false, at)
	if err != nil {
		return err
	}
	_, err = artifactwriter.CommitChecked(prepared, false, nil)
	return err
}

func brpQualificationProject() string {
	return `# Browser Registration Qualification

## Goal

Prepare one reviewed loopback registration without submitting it.

## Inputs

- No workflow inputs.

## Outputs

- No workflow outputs.

## External Systems and OpenAPI

OpenAPI: none required

- Use only the reviewed deterministic loopback browser registration profile.

## Data Flow

- Carry symbolic registration bindings into an inert package.

## Runtime Policy

- browser_registration is allowed only for an explicitly approved sandbox proof run through a trusted runtime.
- Account creation requires exact approval; package and dry-run qualification do not execute it.

## Function Contracts

- No function runtime is required.

## Credentials and Secrets

- Use only symbolic runtime bindings; never store their values.

## Safety and Approval Boundary

- Generate and validate artifacts only.
- Any side-effectful workflow requires approved trusted runner execution.
- Use sandbox endpoints for proof runs before production handoff.

## Fallback Behavior

- Stop without submission or account creation if review, qualification, or runtime support is unavailable.
`
}

// ValidateBRPQualificationEvidence enforces the complete zero-submit posture
// as well as the nine exact lifecycle digests.
func ValidateBRPQualificationEvidence(evidence BRPQualificationEvidence) error {
	values := []string{
		evidence.ProducerResultSHA256, evidence.TransactionSHA256, evidence.PreparationSHA256,
		evidence.QualificationSHA256, evidence.GenerationSHA256, evidence.SelectionSHA256,
		evidence.PackageSHA256, evidence.HandoffSHA256, evidence.WorkflowSHA256,
	}
	if err := validateQualificationDigests(evidence.EvidenceCount, values); err != nil {
		return errors.New("BRP qualification digest evidence is invalid")
	}
	if len(evidence.Methods) != 2 || evidence.Methods[0] != http.MethodGet || evidence.Methods[1] != http.MethodHead ||
		evidence.GETRequests <= 0 || evidence.HEADRequests <= 0 || evidence.Requests != evidence.GETRequests+evidence.HEADRequests ||
		evidence.MutationRequests != 0 || evidence.SubmitExecuted || evidence.AccountCreated || evidence.SessionEstablished ||
		evidence.RuntimeSupported || evidence.ExecutorInvoked {
		return errors.New("BRP qualification no-submit evidence is invalid")
	}
	return nil
}

type registrationQualificationNetworkEvidence struct {
	Methods          []string
	Requests         int
	GETRequests      int
	HEADRequests     int
	MutationRequests int
	AccountCreated   bool
}

type registrationQualificationFixture struct {
	server  *httptest.Server
	mu      sync.Mutex
	counts  map[string]int
	mutated bool
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
	if request.Method != http.MethodGet && request.Method != http.MethodHead {
		http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	if request.Method == http.MethodHead {
		return
	}
	_, _ = writer.Write([]byte(`<!doctype html><html><body><main><h1>Create account</h1><form method="post" action="/registration-complete"><label>Email<input autocomplete="email"></label><label>Password<input type="password"></label><button type="submit">Register</button><p role="status">Registration is ready</p></form></main></body></html>`))
}

func (fixture *registrationQualificationFixture) URL() string {
	return fixture.server.URL + "/register"
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
		MutationRequests: mutations, AccountCreated: fixture.mutated,
	}
}
