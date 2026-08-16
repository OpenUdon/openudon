// Package browserintegrationeval runs the provider-free browser authoring,
// package, contract, and runtime-boundary release matrix across sibling repos.
package browserintegrationeval

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/OpenUdon/openudon/internal/browserverify"
)

const (
	ReportVersion  = "openudon.browser-integration-eval.v1"
	StatusPass     = "pass"
	StatusFail     = "fail"
	StatusSkipped  = "skipped"
	maxOutputBytes = 4 << 20
)

type Options struct {
	RepoRoot          string
	BrowsertoolsRepo  string
	UWSRepo           string
	UdonRepo          string
	BrowserdriverRepo string
	OutPath           string
	InstalledEngines  bool
	HeadedAuth        bool
	Now               func() time.Time
	Runner            Runner
}

type Command struct {
	Repository string
	Dir        string
	Args       []string
	Env        map[string]string
}

type CommandOutput struct {
	Stdout string
	Stderr string
	Err    error
}

type Runner func(context.Context, Command) CommandOutput

type Report struct {
	Version                         string               `json:"version"`
	Status                          string               `json:"status"`
	GeneratedAt                     string               `json:"generated_at"`
	Commit                          string               `json:"commit"`
	Command                         string               `json:"command"`
	Repositories                    []RepositoryRevision `json:"repositories"`
	RetentionClass                  string               `json:"retention_class"`
	ContainsProviderOutput          bool                 `json:"contains_provider_output"`
	SafeToArchive                   bool                 `json:"safe_to_archive"`
	RedactionRequiredBeforeShare    bool                 `json:"redaction_required_before_share"`
	ProviderFree                    bool                 `json:"provider_free"`
	BrowserLaunchedByDefault        bool                 `json:"browser_launched_by_default"`
	TargetContactedByICoT           bool                 `json:"target_contacted_by_icot"`
	CredentialEnvironmentReadByICoT bool                 `json:"credential_environment_read_by_icot"`
	PlanningDeliverablesWritten     bool                 `json:"planning_deliverables_written"`
	Summary                         Summary              `json:"summary"`
	Results                         []GateResult         `json:"results"`
}

type Summary struct {
	Total   int `json:"total"`
	Passed  int `json:"passed"`
	Failed  int `json:"failed"`
	Skipped int `json:"skipped"`
}

type RepositoryRevision struct {
	Name   string `json:"name"`
	Commit string `json:"commit"`
	Dirty  bool   `json:"dirty"`
}

type GateResult struct {
	ID            string   `json:"id"`
	Repository    string   `json:"repository"`
	Kind          string   `json:"kind"`
	Status        string   `json:"status"`
	EvidenceCount int      `json:"evidence_count,omitempty"`
	Command       []string `json:"command,omitempty"`
	Assertions    []string `json:"assertions"`
	Detail        string   `json:"detail,omitempty"`
}

type gate struct {
	ID             string
	Repository     string
	Kind           string
	Args           []string
	Env            map[string]string
	Assertions     []string
	Forbidden      []string
	RequiredPasses []string
	RequiredLines  []string
	OptIn          string
}

type doctorReport struct {
	Version             string `json:"version"`
	Engine              string `json:"engine"`
	PlaywrightGoVersion string `json:"playwright_go_version"`
	PlaywrightVersion   string `json:"playwright_version"`
	DriverReady         bool   `json:"driver_ready"`
	BrowserReady        bool   `json:"browser_ready"`
	BrowserExecutable   string `json:"browser_executable,omitempty"`
	Capabilities        []struct {
		Name        string `json:"name"`
		Disposition string `json:"disposition"`
		Reason      string `json:"reason"`
	} `json:"capability_policy"`
	Error string `json:"error,omitempty"`
}

func Run(ctx context.Context, opts Options) (*Report, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	repos, err := resolveRepos(opts)
	if err != nil {
		return nil, err
	}
	runner := opts.Runner
	if runner == nil {
		runner = runCommand
	}
	now := time.Now().UTC()
	if opts.Now != nil {
		now = opts.Now().UTC()
	}
	revisions, err := repositoryRevisions(ctx, repos, runner)
	if err != nil {
		return nil, err
	}
	report := &Report{
		Version: ReportVersion, Status: StatusPass,
		GeneratedAt: now.Format(time.RFC3339), Commit: revisions[0].Commit, Repositories: revisions,
		Command: "openudon browser-integration-eval", RetentionClass: "release_evidence",
		SafeToArchive: true, ProviderFree: true,
		Results: make([]GateResult, 0, len(defaultGates())),
	}
	componentReady := map[string]bool{}
	for _, spec := range defaultGates() {
		enabled := spec.OptIn == "" || (spec.OptIn == "installed" && opts.InstalledEngines) || (spec.OptIn == "headed" && opts.HeadedAuth)
		if !enabled {
			report.Results = append(report.Results, GateResult{
				ID: spec.ID, Repository: spec.Repository, Kind: spec.Kind, Status: StatusSkipped,
				Command: append([]string(nil), spec.Args...), Assertions: append([]string(nil), spec.Assertions...),
				Detail: optInSkipDetail(spec.OptIn),
			})
			continue
		}
		if spec.OptIn != "" && !componentsReadyForOptIn(spec.OptIn, componentReady) {
			report.Results = append(report.Results, GateResult{
				ID: spec.ID, Repository: spec.Repository, Kind: spec.Kind, Status: StatusSkipped,
				Command: append([]string(nil), spec.Args...), Assertions: append([]string(nil), spec.Assertions...),
				Detail: optInUnavailableDetail(spec.OptIn),
			})
			continue
		}
		command := Command{Repository: spec.Repository, Dir: repos[spec.Repository], Args: append([]string(nil), spec.Args...), Env: cloneMap(spec.Env)}
		output := runner(ctx, command)
		result := evaluateGate(spec, output)
		if spec.Kind == "doctor" && result.Status == StatusPass {
			componentReady[doctorEngine(spec)] = doctorBrowserReady(output)
		}
		report.Results = append(report.Results, result)
		if result.Status == StatusFail {
			report.Status = StatusFail
		}
	}
	report.Summary = summarize(report.Results)
	if err := Validate(report); err != nil {
		return report, err
	}
	if strings.TrimSpace(opts.OutPath) != "" {
		if err := Write(opts.OutPath, report); err != nil {
			return report, err
		}
	}
	if report.Status == StatusFail {
		return report, fmt.Errorf("browser integration evaluation failed")
	}
	return report, nil
}

func Write(path string, report *Report) error {
	if report == nil {
		return fmt.Errorf("browser integration report is required")
	}
	if err := Validate(report); err != nil {
		return err
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if err := writeAtomic(path, data, 0o644); err != nil {
		return err
	}
	sum := sha256.Sum256(data)
	line := "sha256:" + hex.EncodeToString(sum[:]) + "  " + filepath.Base(path) + "\n"
	return writeAtomic(path+".sha256", []byte(line), 0o644)
}

func VerifyFile(path string) (*Report, error) {
	data, err := readBoundedRegular(path, maxOutputBytes)
	if err != nil {
		return nil, err
	}
	if err := verifyDigest(path, data); err != nil {
		return nil, err
	}
	var report Report
	if err := browserverify.DecodeStrictJSON(data, &report); err != nil {
		return nil, fmt.Errorf("parse browser integration report: %w", err)
	}
	if err := validateRequiredWireFields(data); err != nil {
		return nil, err
	}
	if err := Validate(&report); err != nil {
		return nil, err
	}
	return &report, nil
}

func VerifyPassingFile(path string) (*Report, error) {
	report, err := VerifyFile(path)
	if err != nil {
		return nil, err
	}
	if report.Status != StatusPass {
		return report, fmt.Errorf("browser integration report status is %s", report.Status)
	}
	return report, nil
}

func Validate(report *Report) error {
	if report == nil {
		return fmt.Errorf("browser integration report is required")
	}
	if report.Version != ReportVersion {
		return fmt.Errorf("browser integration report version must be %q", ReportVersion)
	}
	if report.Status != StatusPass && report.Status != StatusFail {
		return fmt.Errorf("browser integration report status %q is invalid", report.Status)
	}
	if parsed, err := time.Parse(time.RFC3339, report.GeneratedAt); err != nil || report.GeneratedAt != parsed.UTC().Format(time.RFC3339) {
		return fmt.Errorf("browser integration generated_at must be canonical UTC RFC3339")
	}
	if !commitPattern.MatchString(report.Commit) || report.Command != "openudon browser-integration-eval" || report.RetentionClass != "release_evidence" {
		return fmt.Errorf("browser integration provenance is incomplete")
	}
	if err := validateRepositoryRevisions(report.Repositories, report.Commit); err != nil {
		return err
	}
	if report.ContainsProviderOutput || !report.SafeToArchive || report.RedactionRequiredBeforeShare || !report.ProviderFree {
		return fmt.Errorf("browser integration retention or provider-free claims are invalid")
	}
	if report.BrowserLaunchedByDefault || report.TargetContactedByICoT || report.CredentialEnvironmentReadByICoT || report.PlanningDeliverablesWritten {
		return fmt.Errorf("browser integration authoring authority widened")
	}
	expected := defaultGates()
	if len(report.Results) != len(expected) {
		return fmt.Errorf("browser integration result count = %d, want %d", len(report.Results), len(expected))
	}
	for index, result := range report.Results {
		spec := expected[index]
		if result.ID != spec.ID || result.Repository != spec.Repository || result.Kind != spec.Kind || !equalStrings(result.Command, spec.Args) || !equalStrings(result.Assertions, spec.Assertions) {
			return fmt.Errorf("browser integration result %q contract drifted", result.ID)
		}
		if result.Status != StatusPass && result.Status != StatusFail && result.Status != StatusSkipped {
			return fmt.Errorf("browser integration result %q status is invalid", result.ID)
		}
		if spec.OptIn == "" && result.Status == StatusSkipped {
			return fmt.Errorf("required browser integration result %q is skipped", result.ID)
		}
		if result.Status == StatusPass && result.EvidenceCount <= 0 {
			return fmt.Errorf("browser integration result %q has no evidence", result.ID)
		}
		if result.Kind == "go_test" && result.Status == StatusPass && result.EvidenceCount != len(spec.RequiredPasses) {
			return fmt.Errorf("browser integration result %q named-test evidence count drifted", result.ID)
		}
		if result.Kind == "command" && result.Status == StatusPass && result.EvidenceCount != len(spec.RequiredLines) {
			return fmt.Errorf("browser integration result %q command evidence count drifted", result.ID)
		}
		if result.EvidenceCount < 0 || (result.Kind == "go_test" && result.EvidenceCount > len(spec.RequiredPasses)) {
			return fmt.Errorf("browser integration result %q evidence count is invalid", result.ID)
		}
		if result.Status == StatusSkipped && strings.TrimSpace(result.Detail) == "" {
			return fmt.Errorf("browser integration result %q has no skip reason", result.ID)
		}
		if !validResultDetail(spec, result) {
			return fmt.Errorf("browser integration result %q detail is not value-free canonical evidence", result.ID)
		}
	}
	wantSummary := summarize(report.Results)
	if report.Summary != wantSummary {
		return fmt.Errorf("browser integration summary does not match results")
	}
	if (report.Status == StatusPass) != (report.Summary.Failed == 0) {
		return fmt.Errorf("browser integration status does not match failed count")
	}
	return nil
}

func validateRequiredWireFields(data []byte) error {
	root, err := requiredJSONObject(data, "browser integration report",
		"version", "status", "generated_at", "commit", "command", "repositories",
		"retention_class", "contains_provider_output", "safe_to_archive",
		"redaction_required_before_share", "provider_free", "browser_launched_by_default",
		"target_contacted_by_icot", "credential_environment_read_by_icot",
		"planning_deliverables_written", "summary", "results")
	if err != nil {
		return err
	}
	var repositories []json.RawMessage
	if err := json.Unmarshal(root["repositories"], &repositories); err != nil {
		return fmt.Errorf("browser integration repositories: %w", err)
	}
	for index, raw := range repositories {
		if _, err := requiredJSONObject(raw, fmt.Sprintf("browser integration repositories[%d]", index), "name", "commit", "dirty"); err != nil {
			return err
		}
	}
	if _, err := requiredJSONObject(root["summary"], "browser integration summary", "total", "passed", "failed", "skipped"); err != nil {
		return err
	}
	var results []json.RawMessage
	if err := json.Unmarshal(root["results"], &results); err != nil {
		return fmt.Errorf("browser integration results: %w", err)
	}
	for index, raw := range results {
		if _, err := requiredJSONObject(raw, fmt.Sprintf("browser integration results[%d]", index), "id", "repository", "kind", "status", "command", "assertions", "detail"); err != nil {
			return err
		}
	}
	return nil
}

func requiredJSONObject(data []byte, label string, fields ...string) (map[string]json.RawMessage, error) {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(data, &object); err != nil {
		return nil, fmt.Errorf("%s must be an object: %w", label, err)
	}
	for _, field := range fields {
		value, ok := object[field]
		if !ok || string(bytes.TrimSpace(value)) == "null" {
			return nil, fmt.Errorf("%s requires non-null field %q", label, field)
		}
	}
	return object, nil
}

func defaultGates() []gate {
	return []gate{
		{
			ID: "openudon-authoring", Repository: "openudon", Kind: "go_test",
			Args:       []string{"go", "test", "-v", "./internal/icot", "./internal/icot/elicitor", "-run", "Test(BuildBrowserAuthoringPlan|BrowserAuthoring|DiscoverAuthoringSourcesWithBrowserProfileAndAPIPreference|BrowserProfileWinsOnlyForAPICapabilityGap|DiscoverExplicitGuidedAuthoringBundle|GuidedAuthoringAdapter|BrowserAuthenticationDiscoveryAndReadiness)", "-count=1"},
			Assertions: []string{"API operation preference", "anonymous non-executing handoff", "authenticated capture fail-closed", "guided result replay and rejection matrix", "separate authentication and capability profiles"},
			RequiredPasses: []string{
				"TestBuildBrowserAuthoringPlanIsValueFreeAndNonExecuting",
				"TestBuildBrowserAuthoringPlanFailsClosedForAuthenticatedCapture",
				"TestBuildBrowserAuthoringPlanRejectsUnsafeAuthority",
				"TestBrowserAuthoringPlanCLIAndAgentReportDoNotWriteDeliverables",
				"TestDiscoverAuthoringSourcesWithBrowserProfileAndAPIPreference",
				"TestBrowserProfileWinsOnlyForAPICapabilityGap",
				"TestDiscoverExplicitGuidedAuthoringBundleStagesOnlyVerifiedProfile",
				"TestGuidedAuthoringAdapterRejectsStaleUnsupportedAndTrailingInput",
				"TestGuidedAuthoringAdapterRejectsUnsignedSafetyBypasses",
				"TestGuidedAuthoringAdapterRejectsDecisionMismatch",
				"TestDiscoverExplicitGuidedAuthoringBundleDeduplicatesAndOverridesBroadRootAmbiguity",
				"TestGuidedAuthoringAdapterBoundsReplayWork",
				"TestBrowserAuthenticationDiscoveryAndReadiness",
			},
		},
		{
			ID: "openudon-package-handoff", Repository: "openudon", Kind: "go_test",
			Args:       []string{"go", "test", "-v", "./internal/browserverify", "./internal/icot", "./internal/icot/elicitor", "./internal/synthesize", "./internal/trustedrunner", "-run", "Test(Inspect|AttachBrowserVerifications|ValidateBrowserVerificationCoverage|BrowserVerificationAttachment|ApprovedBrowserVerification|MainAttachesExplicitBrowserVerification|ValidateBrowserSourceReview|ValidatePackagedBrowserProfile|BrowserSourceReviewStrict|ReviewMarkdown.*BrowserVerification|ReviewHandoffIncludesBrowser|ValidateBrowserAuthenticationReview|PackageFromIntentBuildsBrowserAuthenticationWorkflow|RunDryRunStagesAndWritesEvidenceWithoutCredentialEnv)", "-count=1"},
			Assertions: []string{"optional live and portability evidence", "tamper/private/stale/mismatch rejection", "value-free review and package inventory", "trusted dry-run handoff"},
			RequiredPasses: []string{
				"TestInspectAndValidateLiveSummary",
				"TestInspectRejectsUnknownPrivateStaleAndMismatchedFacts",
				"TestInspectRejectsDuplicateAndMissingRequiredWireFields",
				"TestInspectPortabilityValidatesExactEnginesAndBaseline",
				"TestInspectRejectsSymlinkAndOversizedReports",
				"TestAttachBrowserVerificationsBindsDeduplicatesAndRevalidates",
				"TestAttachBrowserVerificationsRejectsMismatchPrivateAndConflict",
				"TestAttachBrowserVerificationsRejectsMalformedRetainedAttachment",
				"TestValidateBrowserVerificationCoverageRejectsFailedAndUnrelatedReports",
				"TestBrowserVerificationAttachmentSurvivesDraftRoundTrip",
				"TestApprovedBrowserVerificationStagesOnlySafeSummaryAndRevalidatesSource",
				"TestMainAttachesExplicitBrowserVerificationReport",
				"TestValidatePackagedBrowserProfileRejectsRawOrSecretShapedFields",
				"TestValidateBrowserAuthenticationReview",
				"TestPackageFromIntentBuildsBrowserAuthenticationWorkflow",
				"TestReviewHandoffIncludesBrowserProfileAndReviewEvidence",
				"TestReviewMarkdownDoesNotTrustTamperedBrowserVerification",
				"TestRunDryRunStagesAndWritesEvidenceWithoutCredentialEnv",
			},
		},
		{
			ID: "icot-dependency-boundary", Repository: "openudon", Kind: "dependency_scan",
			Args:       []string{"go", "list", "-deps", "./cmd/icot"},
			Assertions: []string{"iCoT has no Browsertools capture or Playwright implementation dependency"},
			Forbidden:  []string{"github.com/OpenUdon/browsertools/capture", "github.com/OpenUdon/browsertools/adapter/playwright", "github.com/mxschmitt/playwright-go"},
		},
		{
			ID: "openudon-repository-boundary", Repository: "openudon", Kind: "command",
			Args:          []string{"go", "run", "./cmd/openudon", "check-apitools-boundary"},
			Assertions:    []string{"no private executor imports", "no desired-state parser imports", "no removed apitools lifecycle imports"},
			RequiredLines: []string{"openudon: repository boundary check passed"},
		},
		{
			ID: "browsertools-producer", Repository: "browsertools", Kind: "go_test",
			Args:       []string{"go", "test", "-v", "./capture", "./cmd/browsertools", "./guide", "./authassist", "-run", "Test(AuthorBuilds|Check|ComparePortability|ContractPressure|LiveCheck|PortabilityCLI|GuideAuthor|GuidedEvidenceReader|RunObservesSelectedAlternatives|RunClosesAndReturnsNoArtifact|RunDefendsAgainstMisbehavingBrowser|PlaywrightDoctorFailsOfflineWithoutInstalling)", "-count=1"},
			Assertions: []string{"synthetic/fake-engine default", "value-free live and portability wires", "private/auth evidence boundaries", "offline doctor does not install"},
			RequiredPasses: []string{
				"TestCheckUsesBoundedAcquisitionAndReturnsValueFreeReport",
				"TestComparePortabilityUsesFreshExactProbePlans",
				"TestLiveCheckChromiumCLIEmitsOnlyValueFreeReport",
				"TestPortabilityCLIUsesProcessClockAndExplicitFreshEngines",
				"TestAuthorBuildsDeterministicStrictBundle",
				"TestRunObservesSelectedAlternativesAndBuildsValueFreeBundle",
				"TestRunClosesAndReturnsNoArtifactOnFailure",
				"TestRunDefendsAgainstMisbehavingBrowserImplementations",
				"TestPlaywrightDoctorFailsOfflineWithoutInstalling",
			},
		},
		{
			ID: "uws-browser-contract", Repository: "uws", Kind: "go_test",
			Args:       []string{"go", "test", "-v", "./schemas", "./uws1", "./browserauthentication", "-run", "Test(ValidateBrowser|CanonicalBrowser|BrowserSourceProfile|BrowserProfileSchema|BrowserAuthentication|Validate_UWS15BrowserProfile|SessionExtensionRoundTrip)", "-count=1"},
			Assertions: []string{"unchanged browser.1.5 contract", "browser-authentication contract", "workflow source bindings"},
			RequiredPasses: []string{
				"TestValidateBrowserSourceProfileCanonicalFixtures",
				"TestBrowserAuthenticationProfileCanonicalFixture",
				"TestValidate_UWS15BrowserProfileSourceOperationBindings",
				"TestSessionExtensionRoundTrip",
			},
		},
		{
			ID: "udon-browser-consumer", Repository: "udon", Kind: "go_test",
			Args:       []string{"go", "test", "-v", "./internal/sourceloader", "./generator", "./spider", "./pkg/browserdriver", "./pkg/uwsprofile", "./cmd/udon", "-run", "Test(LoadBrowser|BrowserSource|NewRuntimePlanFromUWSFile.*Browser|BrowserRuntime|BrowserAuthentication|SessionIsOpaque|PersistentSubprocess|ValidateAuthenticationRequest|ValidateBrowserAuthenticationCall|ParseExecuteFlagsAcceptsRepeatableBrowserPolicy|BrowserExecutionOptionsRequireExplicitDriver)", "-count=1"},
			Assertions: []string{"trusted runtime keeps source/profile checks", "mutation and authentication approvals", "opaque named sessions", "driver remains explicit"},
			RequiredPasses: []string{
				"TestLoadBrowserSourceAndResolveAction",
				"TestNewRuntimePlanFromUWSFileLoadsBrowserProfilePrivately",
				"TestBrowserRuntimeReadOnlySuccessAndSanitizedPersistence",
				"TestBrowserAuthenticationRequiresApprovalAndRejectsProfileTamper",
				"TestSessionIsOpaque",
				"TestPersistentSubprocessAuthenticationChallengeAndNamedAction",
				"TestValidateAuthenticationRequestRequiresExactSymbolicCredentials",
				"TestValidateBrowserAuthenticationCallRejectsUnknownRawFields",
				"TestParseExecuteFlagsAcceptsRepeatableBrowserPolicy",
				"TestBrowserExecutionOptionsRequireExplicitDriver",
			},
		},
		{
			ID: "browserdriver-runtime", Repository: "browserdriver", Kind: "npm_test",
			Args:       []string{"npm", "test"},
			Assertions: []string{"closed NDJSON runtime protocol", "exact-origin and session isolation", "offline synthetic runtime tests"},
		},
		{
			ID: "browser-component-inventory-chromium", Repository: "browsertools", Kind: "doctor",
			Args:       []string{"go", "run", "./cmd/browsertools", "playwright", "doctor", "--engine", "chromium", "--format", "json"},
			Assertions: []string{"pinned Chromium readiness is observed without installation or launch"},
		},
		{
			ID: "browser-component-inventory-firefox", Repository: "browsertools", Kind: "doctor",
			Args:       []string{"go", "run", "./cmd/browsertools", "playwright", "doctor", "--engine", "firefox", "--format", "json"},
			Assertions: []string{"pinned Firefox readiness is observed without installation or launch"},
		},
		{
			ID: "browser-component-inventory-webkit", Repository: "browsertools", Kind: "doctor",
			Args:       []string{"go", "run", "./cmd/browsertools", "playwright", "doctor", "--engine", "webkit", "--format", "json"},
			Assertions: []string{"pinned WebKit readiness is observed without installation or launch"},
		},
		{
			ID: "installed-headless-opt-in", Repository: "browsertools", Kind: "go_test", OptIn: "installed",
			Args:       []string{"go", "test", "-v", "./capture", "-run", "TestPlaywright(LiveCapture|RichCapture|Portability)LoopbackOptIn", "-count=1"},
			Env:        map[string]string{"BROWSERTOOLS_LIVE_TEST": "1", "BROWSERTOOLS_RICH_LIVE_TEST": "1", "BROWSERTOOLS_PORTABILITY_LIVE_TEST": "1"},
			Assertions: []string{"explicit loopback-only installed Chromium/Firefox/WebKit proof"},
			RequiredPasses: []string{
				"TestPlaywrightLiveCaptureLoopbackOptIn",
				"TestPlaywrightRichCaptureLoopbackOptIn",
				"TestPlaywrightPortabilityLoopbackOptIn",
			},
		},
		{
			ID: "headed-auth-opt-in", Repository: "browsertools", Kind: "go_test", OptIn: "headed",
			Args:           []string{"go", "test", "-v", "./capture", "-run", "TestPlaywrightAuthHeadedLoopbackOptIn", "-count=1"},
			Env:            map[string]string{"BROWSERTOOLS_AUTH_LIVE_TEST": "1"},
			Assertions:     []string{"explicit loopback-only headed authentication proof"},
			RequiredPasses: []string{"TestPlaywrightAuthHeadedLoopbackOptIn"},
		},
	}
}

func evaluateGate(spec gate, output CommandOutput) GateResult {
	result := GateResult{
		ID: spec.ID, Repository: spec.Repository, Kind: spec.Kind,
		Command: append([]string(nil), spec.Args...), Assertions: append([]string(nil), spec.Assertions...),
	}
	combined := output.Stdout + "\n" + output.Stderr
	switch spec.Kind {
	case "go_test":
		passed, skipped := testMarkerCounts(combined, spec.RequiredPasses)
		result.EvidenceCount = passed
		if output.Err == nil && passed == len(spec.RequiredPasses) {
			result.Status = StatusPass
			result.Detail = fmt.Sprintf("%d named provider-free test(s) passed", result.EvidenceCount)
			return result
		}
		if output.Err == nil && spec.OptIn != "" && passed+skipped == len(spec.RequiredPasses) && skipped > 0 {
			result.Status = StatusSkipped
			result.Detail = optInUnavailableDetail(spec.OptIn)
			return result
		}
	case "npm_test":
		result.EvidenceCount = npmTestCount(combined)
		if output.Err == nil && result.EvidenceCount > 0 {
			result.Status = StatusPass
			result.Detail = fmt.Sprintf("%d Browserdriver test(s) passed", result.EvidenceCount)
			return result
		}
	case "dependency_scan":
		lines := nonemptyLines(output.Stdout)
		result.EvidenceCount = len(lines)
		for _, forbidden := range spec.Forbidden {
			if containsExactLine(lines, forbidden) {
				result.Status = StatusFail
				result.Detail = "forbidden live-browser implementation dependency is reachable from iCoT"
				return result
			}
		}
		if output.Err == nil && result.EvidenceCount > 0 {
			result.Status = StatusPass
			result.Detail = fmt.Sprintf("%d dependency path(s) scanned", result.EvidenceCount)
			return result
		}
	case "command":
		lines := nonemptyLines(output.Stdout)
		for _, required := range spec.RequiredLines {
			if containsExactLine(lines, required) {
				result.EvidenceCount++
			}
		}
		if output.Err == nil && result.EvidenceCount == len(spec.RequiredLines) {
			result.Status = StatusPass
			result.Detail = fmt.Sprintf("%d value-free command marker(s) observed", result.EvidenceCount)
			return result
		}
	case "doctor":
		var doctor doctorReport
		if err := browserverify.DecodeStrictJSON([]byte(output.Stdout), &doctor); err == nil && validDoctorReport(doctor, doctorEngine(spec), output.Err) {
			result.Status = StatusPass
			result.EvidenceCount = len(doctor.Capabilities)
			result.Detail = doctorPassDetail(doctor.Engine, doctor.DriverReady && doctor.BrowserReady)
			return result
		}
	}
	result.Status = StatusFail
	result.Detail = commandFailure(output.Err)
	return result
}

func runCommand(ctx context.Context, command Command) CommandOutput {
	if len(command.Args) == 0 {
		return CommandOutput{Err: fmt.Errorf("empty command")}
	}
	cmd := exec.CommandContext(ctx, command.Args[0], command.Args[1:]...)
	cmd.Dir = command.Dir
	if len(command.Env) > 0 {
		cmd.Env = environmentWithOverrides(os.Environ(), command.Env)
	}
	var stdout, stderr boundedBuffer
	stdout.max, stderr.max = maxOutputBytes, maxOutputBytes
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	if stdout.truncated || stderr.truncated {
		if err == nil {
			err = fmt.Errorf("command output exceeded %d bytes", maxOutputBytes)
		}
	}
	return CommandOutput{Stdout: stdout.String(), Stderr: stderr.String(), Err: err}
}

type boundedBuffer struct {
	bytes.Buffer
	max       int
	truncated bool
}

func (b *boundedBuffer) Write(value []byte) (int, error) {
	written := len(value)
	remaining := b.max - b.Len()
	if remaining <= 0 {
		b.truncated = true
		return written, nil
	}
	if len(value) > remaining {
		value = value[:remaining]
		b.truncated = true
	}
	_, _ = b.Buffer.Write(value)
	return written, nil
}

func resolveRepos(opts Options) (map[string]string, error) {
	root, err := filepath.Abs(defaultString(opts.RepoRoot, "."))
	if err != nil {
		return nil, err
	}
	values := map[string]string{
		"openudon":      root,
		"browsertools":  defaultString(opts.BrowsertoolsRepo, filepath.Join(root, "..", "browsertools")),
		"uws":           defaultString(opts.UWSRepo, filepath.Join(root, "..", "uws")),
		"udon":          defaultString(opts.UdonRepo, filepath.Join(root, "..", "udon")),
		"browserdriver": defaultString(opts.BrowserdriverRepo, filepath.Join(root, "..", "browserdriver")),
	}
	for name, path := range values {
		path, err = filepath.Abs(path)
		if err != nil {
			return nil, err
		}
		info, statErr := os.Stat(path)
		if statErr != nil || !info.IsDir() {
			return nil, fmt.Errorf("%s repository is unavailable at %s", name, path)
		}
		values[name] = path
	}
	return values, nil
}

func repositoryRevisions(ctx context.Context, repos map[string]string, runner Runner) ([]RepositoryRevision, error) {
	names := []string{"openudon", "browsertools", "uws", "udon", "browserdriver"}
	revisions := make([]RepositoryRevision, 0, len(names))
	for _, name := range names {
		commitOutput := runner(ctx, Command{Repository: name, Dir: repos[name], Args: []string{"git", "rev-parse", "--short=12", "HEAD"}})
		commit := strings.TrimSpace(commitOutput.Stdout)
		if commitOutput.Err != nil || !commitPattern.MatchString(commit) {
			return nil, fmt.Errorf("resolve %s release-evidence commit", name)
		}
		statusOutput := runner(ctx, Command{Repository: name, Dir: repos[name], Args: []string{"git", "status", "--porcelain=v1", "--untracked-files=all"}})
		if statusOutput.Err != nil {
			return nil, fmt.Errorf("resolve %s release-evidence worktree state", name)
		}
		revisions = append(revisions, RepositoryRevision{Name: name, Commit: commit, Dirty: strings.TrimSpace(statusOutput.Stdout) != ""})
	}
	return revisions, nil
}

func validateRepositoryRevisions(revisions []RepositoryRevision, openUdonCommit string) error {
	names := []string{"openudon", "browsertools", "uws", "udon", "browserdriver"}
	if len(revisions) != len(names) {
		return fmt.Errorf("browser integration repository provenance count = %d, want %d", len(revisions), len(names))
	}
	for index, name := range names {
		if revisions[index].Name != name || !commitPattern.MatchString(revisions[index].Commit) {
			return fmt.Errorf("browser integration repository provenance %d drifted", index)
		}
	}
	if revisions[0].Commit != openUdonCommit {
		return fmt.Errorf("browser integration OpenUdon commit provenance disagrees")
	}
	return nil
}

func verifyDigest(path string, data []byte) error {
	sidecar, err := readBoundedRegular(path+".sha256", 512)
	if err != nil {
		return err
	}
	fields := strings.Fields(string(sidecar))
	if len(fields) != 2 || fields[1] != filepath.Base(path) {
		return fmt.Errorf("browser integration digest sidecar is malformed")
	}
	sum := sha256.Sum256(data)
	if fields[0] != "sha256:"+hex.EncodeToString(sum[:]) {
		return fmt.Errorf("browser integration report digest mismatch")
	}
	return nil
}

func summarize(results []GateResult) Summary {
	summary := Summary{Total: len(results)}
	for _, result := range results {
		switch result.Status {
		case StatusPass:
			summary.Passed++
		case StatusFail:
			summary.Failed++
		case StatusSkipped:
			summary.Skipped++
		}
	}
	return summary
}

var (
	npmPassPattern    = regexp.MustCompile(`(?m)^(?:#|ℹ) pass ([0-9]+)$`)
	commitPattern     = regexp.MustCompile(`^[0-9a-f]{12,64}$`)
	exitDetailPattern = regexp.MustCompile(`^command exited with status [0-9]+; rerun the recorded command locally for private diagnostics$`)
)

func npmTestCount(output string) int {
	matches := npmPassPattern.FindStringSubmatch(output)
	if len(matches) != 2 {
		return 0
	}
	value, _ := strconv.Atoi(matches[1])
	return value
}

func nonemptyLines(value string) []string {
	var result []string
	for _, line := range strings.Split(value, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			result = append(result, line)
		}
	}
	return result
}

func containsExactLine(lines []string, wanted string) bool {
	for _, line := range lines {
		if line == wanted {
			return true
		}
	}
	return false
}

func optInSkipDetail(kind string) string {
	if kind == "headed" {
		return "headed authentication is a separate explicit loopback-only opt-in and was not requested"
	}
	return "installed Chromium/Firefox/WebKit checks are separate explicit loopback-only opt-ins and were not requested"
}

func optInUnavailableDetail(kind string) string {
	if kind == "headed" {
		return "headed authentication was requested but pinned Chromium readiness or a named loopback-only test was unavailable"
	}
	return "installed-engine proof was requested but pinned engine readiness or a named loopback-only test was unavailable"
}

func testMarkerCounts(output string, tests []string) (passed, skipped int) {
	for _, name := range tests {
		if strings.Contains(output, "--- PASS: "+name+" ") {
			passed++
			continue
		}
		if strings.Contains(output, "--- SKIP: "+name+" ") {
			skipped++
		}
	}
	return passed, skipped
}

func validDoctorReport(report doctorReport, expectedEngine string, commandErr error) bool {
	if report.Version != "browsertools.playwright-doctor.v1" || report.Engine != expectedEngine ||
		strings.TrimSpace(report.PlaywrightGoVersion) == "" || strings.TrimSpace(report.PlaywrightVersion) == "" ||
		len(report.Capabilities) == 0 {
		return false
	}
	seen := make(map[string]bool, len(report.Capabilities))
	for _, capability := range report.Capabilities {
		if strings.TrimSpace(capability.Name) == "" || strings.TrimSpace(capability.Disposition) == "" ||
			strings.TrimSpace(capability.Reason) == "" || seen[capability.Name] {
			return false
		}
		seen[capability.Name] = true
	}
	if report.BrowserReady && (!report.DriverReady || strings.TrimSpace(report.BrowserExecutable) == "") {
		return false
	}
	if !report.BrowserReady && strings.TrimSpace(report.BrowserExecutable) != "" {
		return false
	}
	if report.DriverReady && report.BrowserReady {
		return commandErr == nil && report.Error == ""
	}
	return commandErr != nil && strings.TrimSpace(report.Error) != ""
}

func doctorEngine(spec gate) string {
	for index := 0; index+1 < len(spec.Args); index++ {
		if spec.Args[index] == "--engine" {
			return spec.Args[index+1]
		}
	}
	return ""
}

func doctorBrowserReady(output CommandOutput) bool {
	var report doctorReport
	if err := browserverify.DecodeStrictJSON([]byte(output.Stdout), &report); err != nil {
		return false
	}
	return report.DriverReady && report.BrowserReady
}

func doctorPassDetail(engine string, ready bool) string {
	if ready {
		return fmt.Sprintf("pinned %s components are available; live tests remain opt-in", engine)
	}
	return fmt.Sprintf("pinned %s components are absent; dependent live checks remain skipped unless explicitly requested and available", engine)
}

func componentsReadyForOptIn(kind string, ready map[string]bool) bool {
	if kind == "headed" {
		return ready["chromium"]
	}
	return ready["chromium"] && ready["firefox"] && ready["webkit"]
}

func validResultDetail(spec gate, result GateResult) bool {
	switch result.Status {
	case StatusPass:
		switch spec.Kind {
		case "go_test":
			return result.Detail == fmt.Sprintf("%d named provider-free test(s) passed", result.EvidenceCount)
		case "npm_test":
			return result.Detail == fmt.Sprintf("%d Browserdriver test(s) passed", result.EvidenceCount)
		case "dependency_scan":
			return result.Detail == fmt.Sprintf("%d dependency path(s) scanned", result.EvidenceCount)
		case "command":
			return result.Detail == fmt.Sprintf("%d value-free command marker(s) observed", result.EvidenceCount)
		case "doctor":
			engine := doctorEngine(spec)
			return result.Detail == doctorPassDetail(engine, true) || result.Detail == doctorPassDetail(engine, false)
		}
	case StatusSkipped:
		return spec.OptIn != "" && (result.Detail == optInSkipDetail(spec.OptIn) || result.Detail == optInUnavailableDetail(spec.OptIn))
	case StatusFail:
		return result.Detail == "command completed without the required evidence markers" ||
			result.Detail == "command could not be completed; rerun the recorded command locally for private diagnostics" ||
			exitDetailPattern.MatchString(result.Detail) ||
			(spec.Kind == "dependency_scan" && result.Detail == "forbidden live-browser implementation dependency is reachable from iCoT")
	}
	return false
}

func readBoundedRegular(path string, max int64) ([]byte, error) {
	pathInfo, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !pathInfo.Mode().IsRegular() {
		return nil, fmt.Errorf("%s must be a regular non-symlink file", path)
	}
	if pathInfo.Size() > max {
		return nil, fmt.Errorf("%s exceeds %d bytes", path, max)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	openedInfo, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !openedInfo.Mode().IsRegular() || !os.SameFile(pathInfo, openedInfo) {
		return nil, fmt.Errorf("%s changed while being opened", path)
	}
	data, err := io.ReadAll(io.LimitReader(file, max+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > max {
		return nil, fmt.Errorf("%s exceeds %d bytes", path, max)
	}
	return data, nil
}

func writeAtomic(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	temp, err := os.CreateTemp(dir, ".browser-integration-*")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if err := temp.Chmod(mode); err != nil {
		_ = temp.Close()
		return err
	}
	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(tempPath, path)
}

func environmentWithOverrides(base []string, overrides map[string]string) []string {
	keys := make([]string, 0, len(overrides))
	for key := range overrides {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]string, 0, len(base)+len(keys))
	for _, entry := range base {
		key := entry
		if index := strings.IndexByte(entry, '='); index >= 0 {
			key = entry[:index]
		}
		if _, replace := overrides[key]; replace {
			continue
		}
		result = append(result, entry)
	}
	for _, key := range keys {
		result = append(result, key+"="+overrides[key])
	}
	return result
}

func commandFailure(err error) string {
	if err == nil {
		return "command completed without the required evidence markers"
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return fmt.Sprintf("command exited with status %d; rerun the recorded command locally for private diagnostics", exitErr.ExitCode())
	}
	return "command could not be completed; rerun the recorded command locally for private diagnostics"
}

func cloneMap(value map[string]string) map[string]string {
	if len(value) == 0 {
		return nil
	}
	result := make(map[string]string, len(value))
	for key, item := range value {
		result[key] = item
	}
	return result
}

func equalStrings(left, right []string) bool {
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

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

var _ io.Writer = (*boundedBuffer)(nil)
