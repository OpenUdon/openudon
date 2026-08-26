package browserscenario

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/OpenUdon/openudon/internal/browsercandidate"
	"github.com/OpenUdon/openudon/internal/browsertransaction"
	"github.com/OpenUdon/openudon/internal/evidencefile"
	"github.com/OpenUdon/openudon/internal/icot"
	"github.com/OpenUdon/openudon/internal/icot/artifactwriter"
	"github.com/OpenUdon/openudon/internal/icot/elicitor"
	icotengine "github.com/OpenUdon/openudon/internal/icot/engine"
	"github.com/OpenUdon/openudon/internal/packagepipeline"
	"github.com/OpenUdon/openudon/internal/synthesize"
	"github.com/OpenUdon/openudon/internal/trustedrunner"
	"github.com/OpenUdon/openudon/internal/udonreport"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

const bapBCPQualificationScenario = "mfa-totp-scalars"

// BAPBCPQualificationEvidence is path-free evidence from one exact real
// Browsertools -> OpenUdon -> package lifecycle -> Udon/Browserdriver loop.
type BAPBCPQualificationEvidence struct {
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
}

// RunBAPBCPQualification exercises only the embedded loopback target. It
// never accepts a target URL, credential value, browser session, or output
// path from its caller, and removes all private/scratch/runtime material.
func RunBAPBCPQualification(ctx context.Context, options Options) (BAPBCPQualificationEvidence, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	now := time.Now().UTC().Round(0)
	if options.Now != nil {
		now = options.Now().UTC().Round(0)
	}
	if now.IsZero() || options.AllowNetwork {
		return BAPBCPQualificationEvidence{}, errors.New("BAP+BCP qualification authority is invalid")
	}
	lock, err := LoadCompatibilityLock()
	if err != nil {
		return BAPBCPQualificationEvidence{}, err
	}
	environment, _, _, err := resolveEnvironment(ctx, options, lock, now)
	if err != nil {
		return BAPBCPQualificationEvidence{}, err
	}
	manifests, err := LoadManifests(now)
	if err != nil {
		return BAPBCPQualificationEvidence{}, err
	}
	selected, err := SelectManifests(manifests, SuiteLoopback, []string{bapBCPQualificationScenario})
	if err != nil || len(selected) != 1 {
		return BAPBCPQualificationEvidence{}, errors.New("BAP+BCP qualification manifest is unavailable")
	}
	executor := &realExecutor{}
	defer executor.Close()
	executor.prepare(ctx, environment, SuiteLoopback)
	if executor.unavailable {
		return BAPBCPQualificationEvidence{}, errors.New("BAP+BCP qualification sandbox prerequisite is unavailable")
	}
	if executor.prepareErr != nil {
		return BAPBCPQualificationEvidence{}, errors.New("BAP+BCP qualification dependencies are invalid")
	}
	return executor.runBAPBCPQualification(ctx, environment, selected[0])
}

func (executor *realExecutor) runBAPBCPQualification(ctx context.Context, environment Environment, manifest Manifest) (evidence BAPBCPQualificationEvidence, resultErr error) {
	fixture, err := NewLoopbackFixture(manifest)
	if err != nil {
		return evidence, err
	}
	defer fixture.Close()
	root, err := os.MkdirTemp(executor.root, "transaction-bap-bcp-")
	if err != nil {
		return evidence, errors.New("create BAP+BCP qualification root")
	}
	defer func() {
		if cleanupErr := os.RemoveAll(root); cleanupErr != nil && resultErr == nil {
			resultErr = errors.New("remove BAP+BCP qualification root")
		}
	}()
	exampleDir := filepath.Join(root, "example")
	privateRoot := filepath.Join(root, "private")
	scratch := filepath.Join(root, "scratch")
	store := filepath.Join(root, "store")
	for _, directory := range []string{exampleDir, privateRoot, scratch, store} {
		if err := os.Mkdir(directory, 0o700); err != nil {
			return evidence, errors.New("create BAP+BCP qualification directory")
		}
	}
	var candidate *browsercandidate.AuthenticationCapability
	author, err := icot.RunBrowserScenarioAuthor(ctx, icot.BrowserScenarioAuthorRequest{
		BrowsertoolsPath: executor.browsertools, ExampleDir: exampleDir, PrivateRoot: privateRoot,
		InitialURL: fixture.InitialURL(), AuthenticationURL: fixture.AuthenticationURL(), GoalURL: fixture.GoalURL(),
		GoalContext: manifest.Goal.Context, GoalRole: manifest.Goal.Role, GoalLabel: manifest.Goal.Name,
		ChallengeKind: manifest.Authentication.ChallengeKind, ContextMode: manifest.Authentication.ContextMode,
		Outputs: scenarioAuthorOutputs(manifest.Outputs), Now: time.Now,
		CandidateObserver: func(value *browsercandidate.AuthenticationCapability) error {
			if value == nil || candidate != nil {
				return errors.New("candidate observation is ambiguous")
			}
			candidate = value
			return nil
		},
	})
	if err != nil || candidate == nil || !validAuthorResult(manifest, author) {
		return evidence, errors.New("BAP+BCP producer or adoption failed")
	}
	reviewed, err := candidate.ReviewedTransaction()
	if err != nil || reviewed.Session == "" || reviewed.State != browsertransaction.StateReviewed {
		return evidence, errors.New("BAP+BCP review transition failed")
	}
	transactionSHA256, err := browsertransaction.Digest(reviewed)
	if err != nil {
		return evidence, err
	}
	packageAt := time.Now().UTC().Round(0)
	if err := materializeBAPBCPPackage(exampleDir, candidate, reviewed, manifest, packageAt); err != nil {
		return evidence, err
	}
	built, quality, err := synthesize.PackageFromIntent(ctx, synthesize.Options{ExampleDir: exampleDir})
	if err != nil || built == nil || quality == nil {
		return evidence, errors.New("BAP+BCP qualification package construction failed")
	}
	if !quality.Passed() {
		return evidence, fmt.Errorf("BAP+BCP qualification package quality failed: %s", closedQualityFailureIDs(quality))
	}
	prepared, err := packagepipeline.PrepareCurrent(ctx, packagepipeline.PrepareOptions{ExampleDir: exampleDir, Scope: "qualification/bap-bcp"})
	if err != nil {
		return evidence, errors.New("BAP+BCP qualification package preparation failed")
	}
	qualified, err := packagepipeline.Qualify(ctx, prepared, packagepipeline.QualifyOptions{ScratchParent: scratch, Now: packageAt})
	if err != nil {
		if code, ok := packagepipeline.QualificationFailureCode(err); ok {
			return evidence, fmt.Errorf("BAP+BCP qualification package qualification failed: %s", code)
		}
		return evidence, errors.New("BAP+BCP qualification package qualification failed: unclassified")
	}
	baselineDir := filepath.Join(root, "baseline")
	if err := os.CopyFS(baselineDir, os.DirFS(filepath.Join(environment.RepoRoot, "examples", "support-priority-routing"))); err != nil {
		return evidence, errors.New("BAP+BCP qualification baseline copy failed")
	}
	if _, err := synthesize.Build(ctx, synthesize.Options{ExampleDir: baselineDir}); err != nil {
		return evidence, errors.New("BAP+BCP qualification baseline build failed")
	}
	baseline, err := packagepipeline.PromoteCurrent(ctx, packagepipeline.CurrentOptions{
		ExampleDir: baselineDir,
		Scope:      "examples/support-priority-routing", ScratchParent: scratch, StoreDir: store,
	})
	if err != nil {
		return evidence, fmt.Errorf("BAP+BCP qualification baseline promotion failed: %s", closedPackageLifecycleFailure(err))
	}
	promoted, err := packagepipeline.Promote(ctx, qualified, packagepipeline.PromotionOptions{StoreDir: store})
	if err != nil {
		return evidence, fmt.Errorf("BAP+BCP qualification package promotion failed: %s", closedPackageLifecycleFailure(err))
	}
	selection := promoted.Selection()
	if selection.PriorGenerationSHA256 != baseline.Selection().SelectedGenerationSHA256 || selection.PriorGenerationSHA256 == "" {
		return evidence, errors.New("BAP+BCP qualification prior generation was not preserved")
	}
	current, err := packagepipeline.ReadCurrent(ctx, store)
	if err != nil || current.Selection() != selection {
		return evidence, errors.New("BAP+BCP qualification current selection is inconsistent")
	}
	inspection, err := packagepipeline.InspectSelected(ctx, store, selection.SelectionSHA256)
	if err != nil || inspection.PackageSHA256 != prepared.Manifest().PackageSHA256 || inspection.HandoffSHA256 != prepared.Manifest().HandoffSHA256 {
		return evidence, errors.New("BAP+BCP selected package review failed")
	}
	approval, err := packagepipeline.ApprovalTemplateSelected(ctx, store, selection.SelectionSHA256, trustedrunner.TemplateOptions{
		State: trustedrunner.StateApprovedForSandbox, Reviewer: "Browser transaction qualification",
		Now: func() time.Time { return packageAt.Add(time.Minute) },
	})
	if err != nil {
		return evidence, errors.New("BAP+BCP selected approval failed")
	}
	approvalPath := filepath.Join(root, "approval.json")
	file, err := os.OpenFile(approvalPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return evidence, errors.New("create BAP+BCP qualification approval")
	}
	if err := trustedrunner.WriteApproval(file, approval); err != nil {
		_ = file.Close()
		return evidence, errors.New("write BAP+BCP qualification approval")
	}
	if err := file.Close(); err != nil {
		return evidence, errors.New("close BAP+BCP qualification approval")
	}
	run, err := packagepipeline.RunSelected(ctx, store, selection.SelectionSHA256, trustedrunner.Options{
		Tier: trustedrunner.TierSandbox, ApprovalPath: approvalPath, WorkDir: filepath.Join(root, "dry-run"),
		DryRun: true, Env: []string{}, Now: func() time.Time { return packageAt.Add(2 * time.Minute) },
	})
	if err != nil || run == nil || !run.DryRun || run.PackageSHA256 != selection.PackageSHA256 {
		return evidence, errors.New("BAP+BCP selected trusted handoff failed")
	}
	bindings := transactionBindings(reviewed)
	fixture.SetRuntime(true)
	replay := executor.runUdonWithFormat(ctx, manifest, exampleDir, built.UWSPath, "uws-yaml", author.CredentialSlotKinds, bindings, fixture)
	if replay.failureCode != "" {
		code := closedReplayFailureCode(replay.failureCode)
		if code == udonreport.CodeUnclassified {
			code += "_" + closedExecutionFailureCategory(filepath.Join(exampleDir, "execution-report.json"))
		}
		return evidence, fmt.Errorf("BAP+BCP authenticated replay failed: executor_%s", code)
	}
	if !scenarioOutputsEqual(replay.outputs, fixture.ExpectedOutputs(manifest.Outputs)) {
		return evidence, errors.New("BAP+BCP authenticated replay failed: output_mismatch")
	}
	if !fixture.AuthenticatedReplayObserved() {
		return evidence, errors.New("BAP+BCP authenticated replay failed: authentication_not_observed")
	}
	workflowSHA256, err := digestQualificationFile(built.UWSPath)
	if err != nil {
		return evidence, err
	}
	evidence = BAPBCPQualificationEvidence{
		ProducerResultSHA256: author.EnvelopeDigest, TransactionSHA256: transactionSHA256,
		PreparationSHA256: prepared.Manifest().ManifestSHA256, QualificationSHA256: qualified.Report().QualificationSHA256,
		GenerationSHA256: selection.SelectedGenerationSHA256, SelectionSHA256: selection.SelectionSHA256,
		PackageSHA256: selection.PackageSHA256, HandoffSHA256: inspection.HandoffSHA256,
		WorkflowSHA256: workflowSHA256, EvidenceCount: 9,
	}
	if err := ValidateBAPBCPQualificationEvidence(evidence); err != nil {
		return BAPBCPQualificationEvidence{}, err
	}
	return evidence, nil
}

func materializeBAPBCPPackage(exampleDir string, candidate *browsercandidate.AuthenticationCapability, reviewed browsertransaction.Transaction, manifest Manifest, at time.Time) error {
	input, err := icotengine.AuthenticationCapabilityVirtualBrowserTransaction(candidate, true)
	if err != nil {
		return err
	}
	discovery, err := elicitor.DiscoverVirtualBrowserSources([]elicitor.VirtualBrowserTransactionInput{input}, at)
	if err != nil {
		return err
	}
	capabilityID := ""
	for _, item := range discovery.Candidates {
		if item.Kind == browsertransaction.CandidateCapability {
			capabilityID = item.ID
		}
	}
	if capabilityID == "" {
		return errors.New("BAP+BCP virtual capability is missing")
	}
	session, err := elicitor.SelectVirtualBrowserSources(elicitor.Session{}, discovery, []string{capabilityID})
	if err != nil || len(session.SourcePlan) != 2 {
		return errors.New("BAP+BCP virtual dependency closure failed")
	}
	authenticationTarget, capabilityTarget := "", ""
	for _, source := range session.SourcePlan {
		switch source.Kind {
		case "browser-authentication":
			authenticationTarget = source.TargetPath
		case "browser-profile":
			capabilityTarget = source.TargetPath
		}
	}
	if authenticationTarget == "" || capabilityTarget == "" {
		return errors.New("BAP+BCP virtual source targets are incomplete")
	}
	bindings := transactionBindings(reviewed)
	timeout := 120.0
	intent := &rollout.Intent{
		Workflow: &rollout.WorkflowMeta{Name: "browser_transaction_qualification", Description: "Authenticate and read the reviewed loopback goal."},
		Steps: []*rollout.Step{
			{Name: "authenticate", Type: "browser_authentication", Do: "Establish the reviewed loopback browser session.", Source: authenticationTarget, AuthenticationFlow: candidate.Flow(), BrowserSession: reviewed.Session, CredentialBindings: bindings, Timeout: &timeout},
			{Name: "read", Type: "browser", Source: capabilityTarget, Operation: "reach_authenticated_goal", BrowserSession: reviewed.Session, DependsOn: []string{"authenticate"}},
		},
	}
	for _, output := range manifest.Outputs {
		intent.Outputs = append(intent.Outputs, &rollout.Output{Name: output.Key, From: "read.received_body." + output.Key})
	}
	intentHCL, err := rollout.RenderIntentHCL(intent)
	if err != nil {
		return err
	}
	session.Version = elicitor.SessionVersion
	session.Intent = *intent
	session.BrowserRoute = "browser"
	session.BrowserSession = "none"
	session.BrowserAuthenticationApprovals = []string{"authenticate"}
	for _, binding := range reviewed.CredentialBindings {
		session.Credentials = append(session.Credentials, binding.Binding)
	}
	sort.Strings(session.Credentials)
	session.CredentialsSet = true
	session.Safety = "Use only the deterministic loopback fixture and require the existing trusted browser boundary."
	session.SafetySet = true
	session.Fallback = "Stop without side effects when authentication, review, or replay fails."
	session.FallbackSet = true
	prepared, err := artifactwriter.Prepare(exampleDir, elicitor.Artifacts{
		ProjectMD: bapBCPQualificationProject(manifest), IntentHCL: intentHCL, Session: session,
	}, false, at)
	if err != nil {
		return errors.New("prepare BAP+BCP authoring artifacts")
	}
	if _, err := artifactwriter.CommitChecked(prepared, false, nil); err != nil {
		return errors.New("commit BAP+BCP authoring artifacts")
	}
	return nil
}

func transactionBindings(transaction browsertransaction.Transaction) map[string]string {
	bindings := make(map[string]string, len(transaction.CredentialBindings))
	for _, binding := range transaction.CredentialBindings {
		bindings[binding.Slot] = binding.Binding
	}
	return bindings
}

func bapBCPQualificationProject(manifest Manifest) string {
	outputs := make([]string, 0, len(manifest.Outputs))
	for _, output := range manifest.Outputs {
		outputs = append(outputs, "- `"+output.Key+"`: reviewed typed loopback output.")
	}
	return "# Browser Transaction Qualification\n\n" +
		"## Goal\n\nAuthenticate to the deterministic loopback fixture and read the reviewed goal.\n\n" +
		"## Inputs\n\n- No workflow inputs.\n\n## Outputs\n\n" + strings.Join(outputs, "\n") + "\n\n" +
		"## External Systems and OpenAPI\n\nOpenAPI: none required\n\n- Only the generated loopback browser profiles are used.\n\n" +
		"## Runtime Policy\n\n- Browser execution is allowed only through the trusted Udon and Browserdriver handoff.\n\n" +
		"## Data Flow\n\n- Establish one named session, then read the reviewed goal outputs.\n\n" +
		"## Function Contracts\n\n- No function runtime is required.\n\n" +
		"## Credentials and Secrets\n\n- Credential bindings are symbolic names resolved only by the trusted runtime. Values are never package artifacts.\n\n" +
		"## Safety and Approval Boundary\n\n- Use only the deterministic loopback fixture. Require explicit authentication approval and the trusted runtime boundary. Perform a sandbox proof run before any production execution.\n\n" +
		"## Fallback Behavior\n\n- Stop without side effects when authentication, package review, or replay fails.\n"
}

func ValidateBAPBCPQualificationEvidence(evidence BAPBCPQualificationEvidence) error {
	values := []string{
		evidence.ProducerResultSHA256, evidence.TransactionSHA256, evidence.PreparationSHA256,
		evidence.QualificationSHA256, evidence.GenerationSHA256, evidence.SelectionSHA256,
		evidence.PackageSHA256, evidence.HandoffSHA256, evidence.WorkflowSHA256,
	}
	if evidence.EvidenceCount != len(values) {
		return errors.New("BAP+BCP qualification evidence count is invalid")
	}
	seen := map[string]bool{}
	for _, value := range values {
		if !validScenarioTaggedSHA256(value) || seen[value] {
			return errors.New("BAP+BCP qualification digest evidence is invalid")
		}
		seen[value] = true
	}
	return nil
}

func digestQualificationFile(path string) (string, error) {
	data, _, err := evidencefile.ReadRegular(path, scenarioCommandOutputLimit)
	if err != nil {
		return "", errors.New("read qualification workflow")
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func validScenarioTaggedSHA256(value string) bool {
	if !strings.HasPrefix(value, "sha256:") || len(value) != len("sha256:")+sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil
}

func closedQualityFailureIDs(report *synthesize.QualityReport) string {
	if report == nil {
		return "unclassified"
	}
	var codes []string
	for _, check := range report.Checks {
		if check.Status == "fail" && safeQualityCode(check.Code) {
			codes = append(codes, check.Code)
		}
	}
	sort.Strings(codes)
	if len(codes) == 0 {
		return "unclassified"
	}
	return strings.Join(codes, ",")
}

func safeQualityCode(value string) bool {
	if value == "" || len(value) > 128 {
		return false
	}
	for _, character := range value {
		if character != '.' && character != '_' && character != '-' && (character < 'a' || character > 'z') && (character < '0' || character > '9') {
			return false
		}
	}
	return true
}

func closedPackageLifecycleFailure(err error) string {
	if code, ok := packagepipeline.QualificationFailureCode(err); ok {
		return "qualification_" + string(code)
	}
	if code, ok := packagepipeline.PromotionFailureCode(err); ok {
		return "promotion_" + string(code)
	}
	return "unclassified"
}

func closedReplayFailureCode(value string) string {
	if safeQualityCode(value) {
		return value
	}
	return "unclassified"
}

func closedExecutionFailureCategory(path string) string {
	data, _, err := evidencefile.ReadRegular(path, scenarioCommandOutputLimit)
	if err != nil {
		return "report_unavailable"
	}
	report, err := udonreport.Decode(data)
	if err != nil || report.Status != "error" {
		return "report_invalid"
	}
	return classifyExecutionFailureSummary(report.ErrorSummary)
}

func classifyExecutionFailureSummary(summary string) string {
	value := strings.ToLower(summary)
	for _, match := range []struct {
		contains string
		category string
	}{
		{"browser authentication", "browser_authentication_contract"},
		{"browser-profile", "browser_profile_contract"},
		{"browser profile", "browser_profile_contract"},
		{"browser session", "browser_session_contract"},
		{"browser driver", "browser_driver_contract"},
		{"sourcedescription", "source_contract"},
		{"source description", "source_contract"},
		{"credential", "credential_contract"},
		{"no such file", "artifact_missing"},
		{"not exist", "artifact_missing"},
		{"depends_on cycle", "dependency_cycle"},
		{"circular dependency", "dependency_cycle"},
		{"entry workflow", "entry_workflow_contract"},
		{"multiple workflows", "workflow_inventory"},
		{"reserved entry", "workflow_inventory"},
		{"workflow at index", "workflow_identity"},
		{"workflow references", "workflow_reference"},
		{"workflow expression", "workflow_expression"},
		{"workflow root", "workflow_root"},
		{"root workflow", "workflow_root"},
		{"workflow target", "workflow_target"},
		{"workflow type", "workflow_type"},
		{"workflow step", "workflow_step"},
		{"parallel workflow", "parallel_workflow"},
		{"workflow program", "workflow_program"},
		{"workflow metadata", "workflow_metadata"},
		{"not found", "lookup_failed"},
		{"unsupported", "unsupported_contract"},
		{"control signal", "control_signal"},
		{"required", "required_contract"},
		{"missing", "missing_contract"},
		{"invalid", "invalid_contract"},
		{"parse", "parse_contract"},
		{"decode", "decode_contract"},
		{"failed", "operation_failed"},
		{"uws", "uws_contract"},
		{"request", "request_contract"},
		{"parameter", "request_contract"},
		{"timeout", "timeout"},
		{"output", "output_contract"},
	} {
		if strings.Contains(value, match.contains) {
			return match.category
		}
	}
	if strings.Contains(value, "workflow") {
		return "workflow_keywords_" + allowlistedExecutionFailureKeywords(value)
	}
	return "unclassified"
}

func allowlistedExecutionFailureKeywords(value string) string {
	allowed := map[string]bool{
		"workflow": true, "write": true, "open": true, "read": true, "persist": true,
		"config": true, "program": true, "document": true, "step": true, "operation": true,
		"result": true, "source": true, "entry": true, "type": true, "execute": true,
		"execution": true, "evaluate": true, "expression": true, "convert": true, "load": true,
		"marshal": true, "unmarshal": true, "yaml": true, "json": true, "hcl": true,
		"file": true, "directory": true, "permission": true, "cycle": true, "not": true,
		"found": true, "unsupported": true, "missing": true, "required": true, "invalid": true,
		"failed": true, "cannot": true, "outside": true, "root": true, "path": true,
		"output": true, "outputs": true,
	}
	seen := map[string]bool{}
	var keywords []string
	for _, token := range strings.FieldsFunc(strings.ToLower(value), func(character rune) bool {
		return (character < 'a' || character > 'z') && (character < '0' || character > '9')
	}) {
		if allowed[token] && !seen[token] {
			seen[token] = true
			keywords = append(keywords, token)
			if len(keywords) == 8 {
				break
			}
		}
	}
	if len(keywords) == 0 {
		return "unclassified"
	}
	return strings.Join(keywords, "_")
}
