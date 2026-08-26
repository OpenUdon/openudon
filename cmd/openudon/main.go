package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/OpenUdon/openudon/internal/browserintegrationeval"
	"github.com/OpenUdon/openudon/internal/browserscenario"
	"github.com/OpenUdon/openudon/internal/browsertransactioneval"
	"github.com/OpenUdon/openudon/internal/buildinfo"
	"github.com/OpenUdon/openudon/internal/config"
	evalpkg "github.com/OpenUdon/openudon/internal/eval"
	"github.com/OpenUdon/openudon/internal/localcheck"
	"github.com/OpenUdon/openudon/internal/n8nbridge"
	"github.com/OpenUdon/openudon/internal/packagepipeline"
	"github.com/OpenUdon/openudon/internal/qualityremediation"
	"github.com/OpenUdon/openudon/internal/readiness"
	"github.com/OpenUdon/openudon/internal/releaseevidence"
	"github.com/OpenUdon/openudon/internal/smokematrix"
	"github.com/OpenUdon/openudon/internal/synthesize"
	"github.com/OpenUdon/openudon/internal/trustedrunner"
	openudonvalidation "github.com/OpenUdon/openudon/internal/validation"
)

// version is replaced in release archives with -ldflags. Module-installed
// binaries fall back to debug.BuildInfo's main module version.
var version = "devel"

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: openudon <command>\n\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Commands:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  check     verify required sibling repositories are present\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  assess    assess existing example artifacts and write quality reports\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  approval-template print approval JSON for a validated handoff package\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  build     regenerate workflow/UWS from an existing intent.hcl\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  browser-integration-eval run or verify provider-free cross-repo browser evidence\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  browser-scenario-eval run or verify deterministic loopback/journey/public browser scenarios\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  browser-transaction-eval verify value-free cross-package transaction qualification evidence\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  catalog   inspect first-class provider catalog metadata\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  check-apitools-boundary verify OpenUdon repository boundaries\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  check-doc-memory verify local memory-bank and evolution harness files\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  eval      run synthesis eval briefs and write pass/fail reports\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  n8n-bridge validate review-first n8n pattern summary evidence\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  package   prepare, promote, inspect, or reconcile immutable package generations\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  local-udon-smoke build sibling udon and run provider-free executor smoke\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  promote   export/validate UWS from an existing workflow.hcl\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  readiness write local private-checkout and deterministic-gate readiness report\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  release-evidence run local udon smoke, archive, and release-note evidence flow\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  release-notes draft local release evidence notes from run evidence\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  run       validate approval gates and invoke a trusted executor handoff\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  run-evidence keygen/verify/archive run evidence, signatures, and sidecar digests\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  smoke-matrix run provider-free or opt-in live product smoke scenarios\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  synthesize generate intent, workflow, UWS, and review artifacts for an example\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  validate  validate one UWS JSON/YAML file or a directory of UWS artifacts\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  version   print version\n")
	}
	flag.Parse()

	command := "check"
	if flag.NArg() > 0 {
		command = flag.Arg(0)
	}

	switch command {
	case "check":
		if err := config.CheckSiblings("."); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("openudon: required sibling repositories found")
	case "check-apitools-boundary":
		if err := localcheck.CheckAPIToolsBoundary("."); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("openudon: repository boundary check passed")
	case "check-doc-memory":
		if err := runCheckDocMemory(".", os.Stdout, os.Stderr); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "catalog":
		runCatalogCommand(flag.Args()[1:])
	case "browser-integration-eval":
		runBrowserIntegrationEvalCommand(flag.Args()[1:])
	case "browser-scenario-eval":
		runBrowserScenarioEvalCommand(flag.Args()[1:])
	case "browser-transaction-eval":
		runBrowserTransactionEvalCommand(flag.Args()[1:])
	case "validate":
		if flag.NArg() < 2 {
			fmt.Fprintln(os.Stderr, "usage: openudon validate [--allow-empty] <uws-file-or-dir>")
			os.Exit(2)
		}
		runValidateCommand(flag.Args()[1:])
	case "synthesize", "build", "promote", "assess":
		runArtifactCommand(command, flag.Args()[1:])
	case "run":
		runTrustedCommand(flag.Args()[1:])
	case "run-evidence":
		runEvidenceCommand(flag.Args()[1:])
	case "release-notes":
		runReleaseNotesCommand(flag.Args()[1:])
	case "local-udon-smoke":
		runLocalUdonSmokeCommand(flag.Args()[1:])
	case "smoke-matrix":
		runSmokeMatrixCommand(flag.Args()[1:])
	case "approval-template":
		runApprovalTemplateCommand(flag.Args()[1:])
	case "package":
		runPackageCommand(flag.Args()[1:])
	case "eval":
		runEvalCommand(flag.Args()[1:])
	case "n8n-bridge":
		runN8nBridgeCommand(flag.Args()[1:])
	case "readiness":
		runReadinessCommand(flag.Args()[1:])
	case "release-evidence":
		runReleaseEvidenceCommand(flag.Args()[1:])
	case "version":
		runVersionCommand(flag.Args()[1:])
	case "-h", "--help", "help":
		flag.Usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", command)
		flag.Usage()
		os.Exit(2)
	}
}

func runBrowserTransactionEvalCommand(args []string) {
	fs := flag.NewFlagSet("browser-transaction-eval", flag.ExitOnError)
	verify := fs.String("verify", "", "Verify one canonical qualification report and SHA-256 sidecar")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: openudon browser-transaction-eval --verify REPORT\n\n")
		fmt.Fprintf(fs.Output(), "Verifies the canonical, value-free cross-package browser transaction qualification report. The report contains only closed gate outcomes, exact public/local commit classifications, lifecycle digests, and loopback/sandbox posture; it cannot carry paths, subprocess output, browser content, account identifiers, or credential values.\n\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if fs.NArg() != 0 || strings.TrimSpace(*verify) == "" {
		fmt.Fprintln(os.Stderr, "browser-transaction-eval: --verify is required")
		fs.Usage()
		os.Exit(2)
	}
	report, err := browsertransactioneval.VerifyFile(*verify, true)
	if err != nil {
		fmt.Fprintln(os.Stderr, "openudon browser-transaction-eval verify: fail -", err)
		os.Exit(1)
	}
	fmt.Printf("openudon browser-transaction-eval verify: %s (%d passed, %d failed)\n", report.Status, report.Summary.Passed, report.Summary.Failed)
}

func runBrowserScenarioEvalCommand(args []string) {
	fs := flag.NewFlagSet("browser-scenario-eval", flag.ExitOnError)
	suite := fs.String("suite", "", "Scenario suite: loopback, journey, or public")
	browsertoolsRepo := fs.String("browsertools-repo", "../browsertools", "Sibling Browsertools repository")
	uwsRepo := fs.String("uws-repo", "../uws", "Sibling UWS repository")
	udonRepo := fs.String("udon-repo", "../udon", "Sibling Udon repository")
	browserdriverRepo := fs.String("browserdriver-repo", "../browserdriver", "Sibling Browserdriver repository")
	out := fs.String("out", "", "Value-free report JSON path")
	verify := fs.String("verify", "", "Verify an existing report and SHA-256 sidecar instead of running scenarios")
	requireReady := fs.Bool("require-ready", false, "Fail instead of skipping when installed browser dependencies are unavailable")
	allowNetwork := fs.Bool("allow-network", false, "Allow the fixed anonymous public target inventory; required for --suite public")
	var scenarios repeatedStringFlag
	fs.Var(&scenarios, "scenario", "Repeatable embedded scenario ID to run instead of the full suite")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: openudon browser-scenario-eval --suite loopback|journey|public [--scenario ID]... --out REPORT [--require-ready] [repository flags]\n")
		fmt.Fprintf(fs.Output(), "       openudon browser-scenario-eval --verify REPORT\n\n")
		fmt.Fprintf(fs.Output(), "Runs three complementary strict suites. Loopback uses real Browsertools author-session v2 and Udon/Browserdriver v3 replay. Journey imports reviewed Browsertools guided-authoring bundles and runs realistic local read/write workflows through UWS 1.8, Udon v3, and headless Chromium. Public uses value-free Browsertools live checks and credential-free Udon/Browserdriver v2 presence replay against only the embedded anonymous targets; it requires --allow-network. Reports never retain credential values, page content, or subprocess output.\n\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if fs.NArg() != 0 {
		fs.Usage()
		os.Exit(2)
	}
	if strings.TrimSpace(*verify) != "" {
		invalid := ""
		fs.Visit(func(value *flag.Flag) {
			if value.Name != "verify" && invalid == "" {
				invalid = value.Name
			}
		})
		if invalid != "" {
			fmt.Fprintf(os.Stderr, "browser-scenario-eval: --verify cannot be combined with --%s\n", invalid)
			os.Exit(2)
		}
		report, err := browserscenario.VerifyReportFile(*verify, true)
		if err != nil {
			fmt.Fprintln(os.Stderr, "openudon browser-scenario-eval verify: fail -", err)
			os.Exit(1)
		}
		fmt.Printf("openudon browser-scenario-eval verify: %s %s (%d passed, %d skipped, %d quarantined)\n", report.Status, *verify, report.Summary.Passed, report.Summary.Skipped, report.Summary.Quarantined)
		return
	}
	if strings.TrimSpace(*suite) == "" || strings.TrimSpace(*out) == "" {
		fmt.Fprintln(os.Stderr, "browser-scenario-eval: --suite and --out are required")
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	report, err := browserscenario.Run(ctx, browserscenario.Options{
		RepoRoot: ".", BrowsertoolsRepo: *browsertoolsRepo, UWSRepo: *uwsRepo, UdonRepo: *udonRepo,
		BrowserdriverRepo: *browserdriverRepo, Suite: *suite, ScenarioIDs: []string(scenarios),
		OutPath: *out, RequireReady: *requireReady, AllowNetwork: *allowNetwork,
	})
	if report != nil {
		for _, result := range report.Scenarios {
			fmt.Printf("  %-36s %s: %s\n", result.ID, result.Status, result.Detail)
		}
		fmt.Printf("openudon browser-scenario-eval: %s (%d passed, %d failed, %d skipped, %d quarantined)\n", report.Status, report.Summary.Passed, report.Summary.Failed, report.Summary.Skipped, report.Summary.Quarantined)
		fmt.Printf("  report: %s\n", *out)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runBrowserIntegrationEvalCommand(args []string) {
	fs := flag.NewFlagSet("browser-integration-eval", flag.ExitOnError)
	browsertoolsRepo := fs.String("browsertools-repo", "../browsertools", "Sibling Browsertools repository")
	uwsRepo := fs.String("uws-repo", "../uws", "Sibling UWS repository")
	udonRepo := fs.String("udon-repo", "../udon", "Sibling Udon repository")
	browserdriverRepo := fs.String("browserdriver-repo", "../browserdriver", "Sibling Browserdriver repository")
	out := fs.String("out", "eval/runs/browser-integration-local/report.json", "Value-free report JSON path")
	verify := fs.String("verify", "", "Verify an existing report and SHA-256 sidecar instead of running gates")
	installedEngines := fs.Bool("installed-engines", false, "Opt in to loopback-only installed Chromium, Firefox, and WebKit checks")
	headedAuth := fs.Bool("headed-auth", false, "Opt in to the loopback-only headed authentication check")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: openudon browser-integration-eval [--browsertools-repo DIR] [--uws-repo DIR] [--udon-repo DIR] [--browserdriver-repo DIR] [--out FILE] [--installed-engines] [--headed-auth]\n")
		fmt.Fprintf(fs.Output(), "       openudon browser-integration-eval --verify FILE\n\n")
		fmt.Fprintf(fs.Output(), "Runs the provider-free browser authoring-to-handoff matrix across OpenUdon, Browsertools, UWS, Udon, and Browserdriver. The default does not launch a browser, contact a target, read credential values, or retain subprocess output. Installed-engine checks are explicit loopback-only opt-ins.\n\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if fs.NArg() != 0 {
		fs.Usage()
		os.Exit(2)
	}
	if strings.TrimSpace(*verify) != "" {
		invalidVerifyFlag := ""
		fs.Visit(func(value *flag.Flag) {
			if value.Name != "verify" && invalidVerifyFlag == "" {
				invalidVerifyFlag = value.Name
			}
		})
		if invalidVerifyFlag != "" {
			fmt.Fprintf(os.Stderr, "browser-integration-eval: --verify cannot be combined with --%s\n", invalidVerifyFlag)
			os.Exit(2)
		}
		report, err := browserintegrationeval.VerifyPassingFile(*verify)
		if err != nil {
			fmt.Fprintln(os.Stderr, "openudon browser-integration-eval verify: fail -", err)
			os.Exit(1)
		}
		fmt.Printf("openudon browser-integration-eval verify: %s %s (%d passed, %d skipped)\n", report.Status, *verify, report.Summary.Passed, report.Summary.Skipped)
		return
	}
	if strings.TrimSpace(*out) == "" {
		fmt.Fprintln(os.Stderr, "browser-integration-eval: --out is required")
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	report, err := browserintegrationeval.Run(ctx, browserintegrationeval.Options{
		RepoRoot: ".", BrowsertoolsRepo: *browsertoolsRepo, UWSRepo: *uwsRepo,
		UdonRepo: *udonRepo, BrowserdriverRepo: *browserdriverRepo, OutPath: *out,
		InstalledEngines: *installedEngines, HeadedAuth: *headedAuth,
	})
	if report != nil {
		for _, result := range report.Results {
			fmt.Printf("  %-30s %s: %s\n", result.ID, result.Status, result.Detail)
		}
		fmt.Printf("openudon browser-integration-eval: %s (%d passed, %d failed, %d skipped)\n", report.Status, report.Summary.Passed, report.Summary.Failed, report.Summary.Skipped)
		if strings.TrimSpace(*out) != "" {
			fmt.Printf("  report: %s\n", *out)
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

type versionInfo = buildinfo.Info

func runVersionCommand(args []string) {
	fs := flag.NewFlagSet("version", flag.ExitOnError)
	jsonOutput := fs.Bool("json", false, "Print version and local build metadata as JSON")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: openudon version [--json]\n")
		fmt.Fprintf(fs.Output(), "\nPrints the OpenUdon CLI version. With --json, prints local build metadata only; it does not check networks, releases, updates, or telemetry.\n\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if fs.NArg() != 0 {
		fs.Usage()
		os.Exit(2)
	}
	info := collectVersionInfo()
	if *jsonOutput {
		data, err := json.MarshalIndent(info, "", "  ")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println(string(data))
		return
	}
	fmt.Println(info.Version)
}

func collectVersionInfo() versionInfo {
	return buildinfo.Current(version)
}

func runReleaseEvidenceCommand(args []string) {
	fs := flag.NewFlagSet("release-evidence", flag.ExitOnError)
	udonRepo := fs.String("udon-repo", "../udon", "Sibling udon repository to build")
	workdir := fs.String("workdir", ".openudon-run/release-evidence", "Local release evidence work directory")
	archiveDir := fs.String("archive-dir", "", "Archive directory, default <workdir>/archive")
	releaseNotes := fs.String("release-notes", "", "Release-note draft path, default <workdir>/release-notes.md")
	summaryJSON := fs.String("summary-json", "", "Summary JSON path, default <workdir>/summary.json")
	summaryMD := fs.String("summary-md", "", "Summary Markdown path, default <workdir>/summary.md")
	commit := fs.String("commit", "", "Release commit revision; defaults to the current Git commit")
	var gates repeatedStringFlag
	fs.Var(&gates, "gate", "Repeatable gate result entry such as 'go test ./...=pass'")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: openudon release-evidence [--udon-repo ../udon] [--workdir .openudon-run/release-evidence] [--commit REVISION] [--gate name=status]\n\n")
		fmt.Fprintf(fs.Output(), "Runs the provider-free local udon smoke, archives and verifies run evidence, drafts release notes, and writes local JSON/Markdown release-evidence summaries. It does not tag, publish, or commit artifacts.\n\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	summary, err := releaseevidence.Run(ctx, releaseevidence.Options{
		RepoRoot:     ".",
		UdonRepo:     *udonRepo,
		WorkDir:      *workdir,
		ArchiveDir:   *archiveDir,
		ReleaseNotes: *releaseNotes,
		SummaryJSON:  *summaryJSON,
		SummaryMD:    *summaryMD,
		Commit:       *commit,
		Gates:        []string(gates),
	})
	if summary != nil {
		fmt.Printf("openudon release-evidence: %s\n", summary.Status)
		fmt.Printf("  summary:  %s\n", summary.SummaryJSON)
		fmt.Printf("  markdown: %s\n", summary.SummaryMD)
		if strings.TrimSpace(summary.ReleaseNotes) != "" {
			fmt.Printf("  notes:    %s\n", summary.ReleaseNotes)
		}
		if strings.TrimSpace(summary.ArchivedRun) != "" {
			fmt.Printf("  evidence: %s\n", summary.ArchivedRun)
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runEvidenceCommand(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: openudon run-evidence verify --file run-evidence.json")
		fmt.Fprintln(os.Stderr, "       openudon run-evidence archive --file run-evidence.json --out archive/dir")
		os.Exit(2)
	}
	switch args[0] {
	case "keygen":
		runEvidenceKeygenCommand(args[1:])
	case "verify":
		runEvidenceVerifyCommand(args[1:])
	case "archive":
		runEvidenceArchiveCommand(args[1:])
	default:
		fmt.Fprintln(os.Stderr, "Usage: openudon run-evidence verify --file run-evidence.json")
		fmt.Fprintln(os.Stderr, "       openudon run-evidence archive --file run-evidence.json --out archive/dir")
		os.Exit(2)
	}
}

func runEvidenceKeygenCommand(args []string) {
	fs := flag.NewFlagSet("run-evidence keygen", flag.ExitOnError)
	privateKey := fs.String("private-key", "", "Output PKCS#8 PEM Ed25519 private key")
	publicKey := fs.String("public-key", "", "Output PKIX PEM Ed25519 public key")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: openudon run-evidence keygen --private-key operator.key --public-key operator.pub\n\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if fs.NArg() != 0 {
		fs.Usage()
		os.Exit(2)
	}
	if err := trustedrunner.GenerateSigningKey(*privateKey, *publicKey); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("openudon run-evidence keygen: wrote %s and %s\n", *privateKey, *publicKey)
}

func runEvidenceVerifyCommand(args []string) {
	fs := flag.NewFlagSet("run-evidence verify", flag.ExitOnError)
	file := fs.String("file", "", "run-evidence.json file to verify")
	trustedPublicKey := fs.String("trusted-public-key", "", "PKIX PEM public key required to match the evidence signer")
	requireSignature := fs.Bool("require-signature", false, "Reject unsigned run evidence")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: openudon run-evidence verify --file run-evidence.json\n\n")
		fmt.Fprintf(fs.Output(), "Verifies %s, async sidecar relative paths, sidecar SHA-256 digests, record counts, and neutral async record shapes.\n\n", trustedrunner.RunEvidenceVersion)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	result, err := trustedrunner.VerifyRunEvidenceFileWithOptions(*file, trustedrunner.VerifyRunEvidenceOptions{
		TrustedPublicKey: *trustedPublicKey,
		RequireSignature: *requireSignature,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "openudon run-evidence verify: fail %s - %v\n", *file, err)
		os.Exit(1)
	}
	fmt.Printf("openudon run-evidence verify: pass %s (%d async sidecar file(s))\n", result.RunEvidencePath, len(result.AsyncEvidenceFiles))
}

func runEvidenceArchiveCommand(args []string) {
	fs := flag.NewFlagSet("run-evidence archive", flag.ExitOnError)
	file := fs.String("file", "", "run-evidence.json file to archive")
	out := fs.String("out", "", "Archive directory")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: openudon run-evidence archive --file run-evidence.json --out archive/dir\n\n")
		fmt.Fprintf(fs.Output(), "Copies run-evidence.json, referenced async sidecars, and the udon executor report when present, then verifies the archived run evidence.\n\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	result, err := trustedrunner.ArchiveRunEvidence(trustedrunner.ArchiveOptions{RunEvidencePath: *file, ArchiveDir: *out})
	if err != nil {
		fmt.Fprintf(os.Stderr, "openudon run-evidence archive: fail - %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("openudon run-evidence archive: pass %s (%d async sidecar file(s))\n", result.ArchiveDir, result.VerifiedSidecars)
	fmt.Printf("  evidence: %s\n", result.RunEvidencePath)
	for _, path := range result.AsyncEvidence {
		fmt.Printf("  async:    %s\n", path)
	}
	if strings.TrimSpace(result.ExecutorReport) != "" {
		fmt.Printf("  report:   %s\n", result.ExecutorReport)
	}
}

func runReleaseNotesCommand(args []string) {
	if len(args) == 0 || args[0] != "draft" {
		fmt.Fprintln(os.Stderr, "Usage: openudon release-notes draft --run-evidence run-evidence.json --out release-notes.md")
		os.Exit(2)
	}
	fs := flag.NewFlagSet("release-notes draft", flag.ExitOnError)
	runEvidence := fs.String("run-evidence", "", "Verified run-evidence.json file")
	out := fs.String("out", "", "Release-note draft markdown output path")
	verifierOutput := fs.String("verifier-output", "", "Optional file containing captured verifier output")
	commit := fs.String("commit", "", "Release commit revision; defaults to the current Git commit")
	var gates repeatedStringFlag
	fs.Var(&gates, "gate", "Repeatable gate result entry such as 'go test ./...=pass'")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: openudon release-notes draft --run-evidence run-evidence.json --out release-notes.md [--commit REVISION] [--gate name=status] [--verifier-output verify.txt]\n\n")
		fmt.Fprintf(fs.Output(), "Writes a local release-note evidence draft with current commit, gate results, verifier output, and sidecar/report paths.\n\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args[1:]); err != nil {
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	result, err := trustedrunner.WriteReleaseNotesDraft(ctx, trustedrunner.ReleaseNotesOptions{
		RepoRoot:           ".",
		RunEvidencePath:    *runEvidence,
		OutPath:            *out,
		Commit:             *commit,
		Gates:              []string(gates),
		VerifierOutputPath: *verifierOutput,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "openudon release-notes draft: fail - %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("openudon release-notes draft: wrote %s (commit %s)\n", result.Path, result.Commit)
}

func runLocalUdonSmokeCommand(args []string) {
	fs := flag.NewFlagSet("local-udon-smoke", flag.ExitOnError)
	udonRepo := fs.String("udon-repo", "../udon", "Sibling udon repository to build")
	workdir := fs.String("workdir", ".openudon-run/local-udon-smoke", "Local smoke work directory")
	out := fs.String("out", ".openudon-run/local-udon-smoke/summary.json", "Write smoke summary JSON")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: openudon local-udon-smoke [--udon-repo ../udon] [--workdir .openudon-run/local-udon-smoke] [--out .openudon-run/local-udon-smoke/summary.json]\n\n")
		fmt.Fprintf(fs.Output(), "Builds the sibling udon CLI and runs a provider-free non-dry-run trusted executor proof that emits executor-report.json and expanded async observations.\n\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	report, err := smokematrix.RunLocalUdonSmoke(ctx, smokematrix.LocalUdonSmokeOptions{
		RepoRoot: ".",
		UdonRepo: *udonRepo,
		WorkDir:  *workdir,
		OutPath:  *out,
	})
	if report != nil && len(report.Scenarios) > 0 {
		fmt.Printf("openudon local-udon-smoke: %s\n", report.Status)
		fmt.Printf("  summary:  %s\n", *out)
		fmt.Printf("  evidence: %s\n", report.Scenarios[0].RunEvidencePath)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runCheckDocMemory(root string, out, errOut io.Writer) error {
	result, err := localcheck.CheckDocMemory(root)
	if err != nil {
		return err
	}
	fmt.Fprintln(out, "openudon: doc memory check passed")
	for _, file := range result.CheckedFiles {
		fmt.Fprintf(out, "openudon: checked %s\n", file)
	}
	for _, warning := range result.Warnings {
		fmt.Fprintf(errOut, "openudon: warning: %s\n", warning)
	}
	return nil
}

type repeatedStringFlag []string

func (f *repeatedStringFlag) String() string {
	return strings.Join(*f, ",")
}

func (f *repeatedStringFlag) Set(value string) error {
	*f = append(*f, value)
	return nil
}

func runN8nBridgeCommand(args []string) {
	if len(args) == 0 || args[0] != "validate" {
		fmt.Fprintln(os.Stderr, "usage: openudon n8n-bridge validate [--root examples/eval] [--file examples/eval/<name>/reference/n8n-bridge.json]")
		os.Exit(2)
	}
	runN8nBridgeValidateCommand(args[1:])
}

func runN8nBridgeValidateCommand(args []string) {
	fs := flag.NewFlagSet("n8n-bridge validate", flag.ExitOnError)
	root := fs.String("root", "examples/eval", "Eval fixture root to scan for reference/n8n-bridge.json summaries")
	file := fs.String("file", "", "Validate one n8n bridge summary file instead of scanning --root")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: openudon n8n-bridge validate [--root examples/eval] [--file examples/eval/<name>/reference/n8n-bridge.json]\n")
		fmt.Fprintf(fs.Output(), "\nValidates %s evidence. The bridge is authoring assistance only: it does not import, execute, or emulate n8n workflows.\n\n", n8nbridge.SummaryVersion)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	var (
		results []n8nbridge.ValidationResult
		err     error
	)
	if strings.TrimSpace(*file) != "" {
		var result n8nbridge.ValidationResult
		result, err = n8nbridge.ValidateFile(*file)
		results = []n8nbridge.ValidationResult{result}
	} else {
		results, err = n8nbridge.ValidateRoot(*root)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if len(results) == 0 {
		fmt.Fprintf(os.Stderr, "no n8n bridge summaries found under %s\n", *root)
		os.Exit(1)
	}
	fmt.Printf("openudon: n8n bridge validated %d summary file(s)\n", len(results))
	for _, result := range results {
		fmt.Printf("openudon: checked %s (%s, %s)\n", result.Path, result.Summary.Fixture, result.Summary.Validation.Status)
	}
}

func runValidateCommand(args []string) {
	fs := flag.NewFlagSet("validate", flag.ExitOnError)
	allowEmpty := fs.Bool("allow-empty", false, "Allow directory validation to pass when no UWS artifacts are found")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: openudon validate [--allow-empty] <uws-file-or-dir>\n")
		fmt.Fprintf(fs.Output(), "\nValidates one UWS JSON/YAML file or every *.uws.json/*.uws.yaml/*.uws.yml artifact under a directory.\n\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if fs.NArg() != 1 {
		fs.Usage()
		os.Exit(2)
	}
	if err := validateUWSPath(fs.Arg(0), os.Stdout, *allowEmpty); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateUWSPath(target string, out io.Writer, allowEmpty bool) error {
	return openudonvalidation.ValidatePath(target, out, allowEmpty)
}

func validateUWSPathWithSchema(target string, out io.Writer, schemaForFile func(string) string, allowEmpty bool) error {
	return openudonvalidation.ValidatePathWithSchema(target, out, schemaForFile, allowEmpty)
}

func runReadinessCommand(args []string) {
	fs := flag.NewFlagSet("readiness", flag.ExitOnError)
	out := fs.String("out", "", "Write readiness JSON to this path instead of stdout")
	runGates := fs.Bool("run-gates", false, "Run deterministic gates: go test ./..., go vet ./..., make check, and git diff --check")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: openudon readiness [--out eval/readiness/<name>.json] [--run-gates]\n")
		fmt.Fprintf(fs.Output(), "\nWrites %s JSON for XRD-007 local optional-sibling checkout readiness without printing secret values.\n", readiness.ReportVersion)
		fmt.Fprintf(fs.Output(), "By default, deterministic gates are marked skipped; pass --run-gates for release-readiness evidence.\n\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	report, err := readiness.Build(ctx, readiness.Options{
		RepoRoot: ".",
		RunGates: *runGates,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if strings.TrimSpace(*out) != "" {
		if err := readiness.WriteFile(*out, report); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Printf("openudon: readiness %s wrote %s\n", report.Status, *out)
	} else if err := readiness.Write(os.Stdout, report); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if report.Status == "fail" {
		os.Exit(1)
	}
}

func runTrustedCommand(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	example := fs.String("example", "", "Example directory containing generated OpenUdon artifacts")
	tier := fs.String("tier", "", "Execution tier: sandbox or production")
	approval := fs.String("approval", "", "Approval JSON file")
	workdir := fs.String("workdir", "", "executor work directory; defaults to .openudon-run/<example>")
	dryRun := fs.Bool("dry-run", false, "Validate gates, stage the package, verify the staged digest, and write run evidence without invoking the executor")
	signingKey := fs.String("signing-key", "", "Optional PKCS#8 PEM Ed25519 key used only to sign the completed run evidence")
	browserDriver := fs.String("browser-driver", "", "Absolute trusted browser-driver executable path (or absolute in-image path for Docker)")
	packageStore := fs.String("package-store", "", "Use an exact atomically selected package generation store")
	selection := fs.String("selection", "", "Exact current selection SHA-256 required with --package-store")
	var browserDriverArgs repeatedStringFlag
	fs.Var(&browserDriverArgs, "browser-driver-arg", "Repeatable non-secret argument passed to the trusted browser driver")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: openudon run --example examples/<name> --tier sandbox|production --approval approvals/<name>.json [--browser-driver /absolute/path] [--browser-driver-arg value] [--workdir .openudon-run/<name>] [--dry-run]\n")
		fmt.Fprintf(fs.Output(), "       openudon run --package-store /absolute/store --selection sha256:... --tier sandbox|production --approval /outside/store/approval.json --workdir /outside/store/run [--dry-run]\n")
		fmt.Fprintf(fs.Output(), "\nValidates the OpenUdon handoff package, current quality gates, approval scope, approval digest, tier/state compatibility, runner config, package staging, and staged digest before writing %s run evidence, an async evidence sidecar, and invoking the trusted executor runner.\n", trustedrunner.RunEvidenceVersion)
		fmt.Fprintf(fs.Output(), "\nTier rules:\n")
		fmt.Fprintf(fs.Output(), "  sandbox accepts approved_for_sandbox or approved_for_production\n")
		fmt.Fprintf(fs.Output(), "  production accepts approved_for_production only\n\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	runOptions := trustedrunner.Options{
		RepoRoot:          ".",
		ExampleDir:        *example,
		Tier:              *tier,
		ApprovalPath:      *approval,
		WorkDir:           *workdir,
		DryRun:            *dryRun,
		RunnerPath:        os.Getenv("OPENUDON_UDON_RUNNER"),
		Stdout:            os.Stdout,
		Stderr:            os.Stderr,
		SigningKey:        *signingKey,
		BrowserDriver:     *browserDriver,
		BrowserDriverArgs: []string(browserDriverArgs),
	}
	var result *trustedrunner.RunResult
	var err error
	if strings.TrimSpace(*packageStore) != "" || strings.TrimSpace(*selection) != "" {
		if strings.TrimSpace(*packageStore) == "" || strings.TrimSpace(*selection) == "" || strings.TrimSpace(*example) != "" {
			fmt.Fprintln(os.Stderr, "--package-store and --selection must be used together and replace --example")
			os.Exit(2)
		}
		result, err = packagepipeline.RunSelected(ctx, *packageStore, *selection, runOptions)
	} else {
		result, err = trustedrunner.Run(ctx, runOptions)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if result.DryRun {
		fmt.Printf("openudon: run dry-run passed for %s (%s)\n", result.Scope, result.Tier)
	} else {
		fmt.Printf("openudon: run completed for %s (%s)\n", result.Scope, result.Tier)
	}
	fmt.Printf("  workflow: %s\n", result.WorkflowPath)
	fmt.Printf("  config:   %s\n", result.RunConfigPath)
	fmt.Printf("  evidence: %s\n", result.RunEvidencePath)
	if strings.TrimSpace(result.AsyncEvidencePath) != "" {
		fmt.Printf("  async:    %s\n", result.AsyncEvidencePath)
	}
	fmt.Printf("  workdir:  %s\n", result.WorkDir)
	fmt.Printf("  stage:    %s\n", result.StagePath)
	fmt.Printf("  digest:   %s\n", result.PackageSHA256)
}

func runSmokeMatrixCommand(args []string) {
	fs := flag.NewFlagSet("smoke-matrix", flag.ExitOnError)
	mode := fs.String("mode", smokematrix.ModeDryRun, "Smoke mode: dry-run or live")
	workdir := fs.String("workdir", ".openudon-run/product-smoke", "Ignored smoke work directory")
	out := fs.String("out", ".openudon-run/product-smoke/summary.json", "Write product smoke summary JSON")
	var scenarios repeatedStringFlag
	fs.Var(&scenarios, "scenario", "Repeatable scenario ID to run instead of the full matrix")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: openudon smoke-matrix [--mode dry-run|live] [--scenario ID] [--workdir .openudon-run/product-smoke] [--out .openudon-run/product-smoke/summary.json]\n")
		fmt.Fprintf(fs.Output(), "\nBuilds M37 product smoke packages from reviewed eval fixtures, runs trusted-runner dry-runs, and in live mode invokes only scenarios with explicit live policy and complete local environment. Live provider calls are local maintainer evidence, not CI gates.\n\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	report, err := smokematrix.Run(ctx, smokematrix.Options{
		RepoRoot:    ".",
		Mode:        *mode,
		WorkDir:     *workdir,
		OutPath:     *out,
		ScenarioIDs: []string(scenarios),
		Stdout:      os.Stdout,
		Stderr:      os.Stderr,
	})
	if report != nil {
		fmt.Printf("openudon: product smoke matrix %s (%s)\n", report.Status, report.Mode)
		fmt.Printf("  summary: %s\n", *out)
		for _, scenario := range report.Scenarios {
			fmt.Printf("  - %s: %s", scenario.ID, scenario.Status)
			if len(scenario.MissingEnv) > 0 {
				fmt.Printf(" missing_env=%s", strings.Join(scenario.MissingEnv, ","))
			}
			if scenario.PackageSHA256 != "" {
				fmt.Printf(" digest=%s", scenario.PackageSHA256)
			}
			if scenario.Detail != "" {
				fmt.Printf(" detail=%s", scenario.Detail)
			}
			fmt.Println()
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runApprovalTemplateCommand(args []string) {
	fs := flag.NewFlagSet("approval-template", flag.ExitOnError)
	example := fs.String("example", "", "Example directory containing generated OpenUdon artifacts")
	state := fs.String("state", "", "Approval state: approved_for_sandbox or approved_for_production")
	reviewer := fs.String("reviewer", "", "Reviewer name recorded in the approval JSON")
	notes := fs.String("notes", "", "Optional approval notes")
	packageStore := fs.String("package-store", "", "Use an exact atomically selected package generation store")
	selection := fs.String("selection", "", "Exact current selection SHA-256 required with --package-store")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: openudon approval-template --example examples/<name> --state approved_for_sandbox|approved_for_production --reviewer <name> [--notes <text>]\n")
		fmt.Fprintf(fs.Output(), "       openudon approval-template --package-store /absolute/store --selection sha256:... --state approved_for_sandbox|approved_for_production --reviewer <name> [--notes <text>]\n")
		fmt.Fprintf(fs.Output(), "\nPrints %s JSON to stdout with the current handoff package SHA-256 digest.\n", trustedrunner.ApprovalVersion)
		fmt.Fprintf(fs.Output(), "Schema fields: version, scope, state, reviewer, approved_at, expires_at, package_sha256, notes.\n\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	templateOptions := trustedrunner.TemplateOptions{
		RepoRoot:   ".",
		ExampleDir: *example,
		State:      *state,
		Reviewer:   *reviewer,
		Notes:      *notes,
	}
	var approval trustedrunner.Approval
	var err error
	if strings.TrimSpace(*packageStore) != "" || strings.TrimSpace(*selection) != "" {
		if strings.TrimSpace(*packageStore) == "" || strings.TrimSpace(*selection) == "" || strings.TrimSpace(*example) != "" {
			fmt.Fprintln(os.Stderr, "--package-store and --selection must be used together and replace --example")
			os.Exit(2)
		}
		approval, err = packagepipeline.ApprovalTemplateSelected(ctx, *packageStore, *selection, templateOptions)
	} else {
		approval, err = trustedrunner.ApprovalTemplate(ctx, templateOptions)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := trustedrunner.WriteApproval(os.Stdout, approval); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

type packagePreparationOutput struct {
	Version       string                              `json:"version"`
	Preparation   packagepipeline.Manifest            `json:"preparation"`
	Qualification packagepipeline.QualificationReport `json:"qualification"`
	Selection     *packagepipeline.Selection          `json:"selection,omitempty"`
}

const packageCommandVersion = "openudon.package-command.v1"

func runPackageCommand(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: openudon package prepare|promote|inspect|recover [options]")
		os.Exit(2)
	}
	switch args[0] {
	case "-h", "--help", "help":
		fmt.Fprintln(os.Stdout, "Usage: openudon package prepare|promote|inspect|recover [options]")
		fmt.Fprintln(os.Stdout, "Prepare retains no filesystem changes, promote requires explicit confirmation, inspect is read-only, and recover requires an exact observed digest before cleanup.")
	case "prepare", "promote":
		runPackageMutationCommand(args[0], args[1:])
	case "inspect":
		fs := flag.NewFlagSet("package inspect", flag.ExitOnError)
		store := fs.String("store", "", "Existing package generation store")
		fs.Usage = func() {
			fmt.Fprintln(fs.Output(), "Usage: openudon package inspect --store /absolute/store")
			fs.PrintDefaults()
		}
		if err := fs.Parse(args[1:]); err != nil {
			os.Exit(2)
		}
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		current, err := packagepipeline.ReadCurrent(ctx, *store)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		writePackageJSON(current.Selection())
	case "recover":
		fs := flag.NewFlagSet("package recover", flag.ExitOnError)
		store := fs.String("store", "", "Existing package generation store")
		accept := fs.String("accept", "", "Exact recovery report SHA-256 to reconcile; omit for read-only inspection")
		fs.Usage = func() {
			fmt.Fprintln(fs.Output(), "Usage: openudon package recover --store /absolute/store [--accept sha256:...]")
			fs.PrintDefaults()
		}
		if err := fs.Parse(args[1:]); err != nil {
			os.Exit(2)
		}
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		if strings.TrimSpace(*accept) == "" {
			report, err := packagepipeline.InspectRecovery(ctx, *store)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			writePackageJSON(report)
			return
		}
		reconciled, err := packagepipeline.Reconcile(ctx, packagepipeline.ReconcileOptions{StoreDir: *store, ExpectedRecoverySHA256: *accept})
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		writePackageJSON(reconciled)
	default:
		fmt.Fprintf(os.Stderr, "unknown package command %q\n", args[0])
		os.Exit(2)
	}
}

func runPackageMutationCommand(command string, args []string) {
	fs := flag.NewFlagSet("package "+command, flag.ExitOnError)
	example := fs.String("example", "", "Reviewed package directory")
	scope := fs.String("scope", "", "Explicit portable package scope")
	scratch := fs.String("scratch", "", "Existing absolute restrictive-scratch parent")
	expectedInput := fs.String("expected-input", "", "Optional exact preparation input SHA-256")
	store := fs.String("store", "", "Existing package generation store; required for promote")
	confirmed := fs.Bool("confirmed", false, "Explicitly authorize atomic selection; required for promote")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: openudon package %s --example DIR --scope PORTABLE --scratch /absolute/dir", command)
		if command == "promote" {
			fmt.Fprint(fs.Output(), " --store /absolute/store --confirmed")
		}
		fmt.Fprintln(fs.Output())
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if command == "prepare" && (strings.TrimSpace(*store) != "" || *confirmed) {
		fmt.Fprintln(os.Stderr, "package prepare accepts no store or promotion confirmation")
		os.Exit(2)
	}
	if command == "promote" && (!*confirmed || strings.TrimSpace(*store) == "") {
		fmt.Fprintln(os.Stderr, "package promote requires --store and explicit --confirmed")
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	qualified, err := packagepipeline.PrepareAndQualifyCurrent(ctx, packagepipeline.CurrentOptions{
		ExampleDir: *example, Scope: *scope, ExpectedInputSHA256: *expectedInput, ScratchParent: *scratch,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	output := packagePreparationOutput{
		Version: packageCommandVersion, Preparation: qualified.Prepared().Manifest(), Qualification: qualified.Report(),
	}
	if command == "promote" {
		promoted, err := packagepipeline.Promote(ctx, qualified, packagepipeline.PromotionOptions{StoreDir: *store})
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		selection := promoted.Selection()
		output.Selection = &selection
	}
	writePackageJSON(output)
}

func writePackageJSON(value any) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runEvalCommand(args []string) {
	fs := flag.NewFlagSet("eval", flag.ExitOnError)
	root := fs.String("root", "examples/eval", "Directory containing eval example subdirectories")
	name := fs.String("name", "", "Run a single eval brief by directory name")
	provider := fs.String("provider", "", "LLM provider: copilot-api, openai, anthropic, or gemini")
	model := fs.String("model", "", "LLM model")
	timeout := fs.Duration("timeout", 2*time.Minute, "LLM generation timeout")
	maxAttempts := fs.Int("max-attempts", 5, "Maximum refinement attempts")
	temperature := fs.Float64("temperature", 0.2, "Intent generation temperature")
	concurrency := fs.Int("concurrency", 2, "Maximum concurrent eval runs")
	releaseGate := fs.Bool("release-gate", false, "Fail unless eval results meet local release criteria")
	minBriefs := fs.Int("min-briefs", 0, "Minimum eval brief count required by --release-gate")
	compare := fs.String("compare", "", "Compare this eval report against a specific previous JSON report")
	noCompare := fs.Bool("no-compare", false, "Disable previous-run comparison")
	archiveDir := fs.String("archive-dir", "", "Copy generated eval workspaces under this directory for manual inspection")
	out := fs.String("out", evalpkg.DefaultOutputPath(time.Now()), "JSON report output path")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: openudon eval [--root examples/eval] [--name support-email] [--out eval/runs/<ts>.json] [--release-gate] [--min-briefs N] [--compare eval/runs/<previous>.json] [--no-compare] [--archive-dir eval/artifacts]\n")
		fmt.Fprintf(fs.Output(), "\nRuns synthesis against temporary copies of eval briefs and writes JSON/Markdown reports with optional run comparison.\n")
		fmt.Fprintf(fs.Output(), "Normal evals print comparison regressions for review but exit successfully when synthesis completes.\n")
		fmt.Fprintf(fs.Output(), "With --release-gate, absolute release criteria and comparison regressions fail the command.\n")
		fmt.Fprintf(fs.Output(), "\nExamples:\n")
		fmt.Fprintf(fs.Output(), "  openudon eval --root examples/eval --provider copilot-api --model gpt-5.4-mini\n")
		fmt.Fprintf(fs.Output(), "  openudon eval --root examples/eval --name support-email --provider copilot-api --model gpt-5.4-mini\n")
		fmt.Fprintf(fs.Output(), "  openudon eval --root examples/eval --provider copilot-api --model gpt-5.4-mini --release-gate\n\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	opts := synthesize.Options{
		Provider:          *provider,
		Model:             *model,
		Timeout:           *timeout,
		MaxAttempts:       *maxAttempts,
		IntentTemperature: temperature,
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	runID := runIDFromOutput(*out)
	var results []evalpkg.EvalResult
	var evalErr error
	if strings.TrimSpace(*name) != "" {
		var result evalpkg.EvalResult
		if strings.TrimSpace(*archiveDir) == "" {
			result = evalpkg.RunOne(ctx, filepath.Join(*root, strings.TrimSpace(*name)), opts)
		} else {
			result, evalErr = evalpkg.RunOneArchived(ctx, filepath.Join(*root, strings.TrimSpace(*name)), opts, *archiveDir, runID)
		}
		results = []evalpkg.EvalResult{result}
	} else {
		if strings.TrimSpace(*archiveDir) == "" {
			results = evalpkg.RunAll(ctx, *root, opts, *concurrency)
		} else {
			results, evalErr = evalpkg.RunAllArchived(ctx, *root, opts, *concurrency, *archiveDir, runID)
		}
	}
	if evalErr != nil {
		fmt.Fprintln(os.Stderr, evalErr)
		os.Exit(1)
	}
	if len(results) == 0 {
		fmt.Fprintf(os.Stderr, "no eval briefs found under %s\n", *root)
		os.Exit(1)
	}
	commit, dirty := gitMetadata()
	metadata := evalpkg.RunMetadata{
		RunID:       runID,
		Commit:      commit,
		Dirty:       dirty,
		EvalRoot:    *root,
		OutputPath:  *out,
		Provider:    strings.TrimSpace(*provider),
		Model:       strings.TrimSpace(*model),
		ReleaseGate: *releaseGate,
		MinBriefs:   *minBriefs,
		ArchiveDir:  strings.TrimSpace(*archiveDir),
	}
	var comparison *evalpkg.RunComparison
	if !*noCompare {
		previousPath := strings.TrimSpace(*compare)
		if previousPath == "" {
			var err error
			previousPath, err = evalpkg.FindPreviousRun(*out)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		}
		if previousPath != "" {
			previous, err := evalpkg.ReadResults(previousPath)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			cmp := evalpkg.CompareRuns(results, previous, previousPath)
			comparison = &cmp
			metadata.ComparePath = previousPath
		}
	}
	report := evalpkg.BuildRunReport(results, evalpkg.ReportOptions{
		Metadata:   metadata,
		Comparison: comparison,
	})
	if err := evalpkg.WriteReport(*out, report); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("openudon: eval wrote %s\n", *out)
	fmt.Print(evalpkg.MarkdownReport(report))
	if *releaseGate {
		criteria := evalpkg.DefaultReleaseCriteria()
		criteria.MinBriefs = *minBriefs
		if err := evalpkg.ReleaseCriteriaError(results, criteria); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := evalpkg.ComparisonRegressionError(comparison); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}

func runIDFromOutput(outPath string) string {
	base := strings.TrimSuffix(filepath.Base(outPath), filepath.Ext(outPath))
	if strings.TrimSpace(base) == "" {
		return time.Now().UTC().Format("20060102T150405Z")
	}
	return base
}

func gitMetadata() (string, bool) {
	commitBytes, err := exec.Command("git", "rev-parse", "--short=12", "HEAD").Output()
	if err != nil {
		return "", false
	}
	statusBytes, err := exec.Command("git", "status", "--porcelain").Output()
	if err != nil {
		return strings.TrimSpace(string(commitBytes)), false
	}
	return strings.TrimSpace(string(commitBytes)), strings.TrimSpace(string(statusBytes)) != ""
}

func runArtifactCommand(command string, args []string) {
	fs := flag.NewFlagSet(command, flag.ExitOnError)
	example := fs.String("example", "", "Example directory containing project.md and artifact subdirectories")
	provider := fs.String("provider", "", "LLM provider for synthesize; optional review-evidence label for build")
	model := fs.String("model", "", "LLM model for synthesize; optional review-evidence label for build")
	timeout := fs.Duration("timeout", 2*time.Minute, "LLM generation timeout for synthesize")
	maxAttempts := fs.Int("max-attempts", 5, "Maximum refinement attempts for synthesize")
	temperature := fs.Float64("temperature", 0.2, "Intent generation temperature for synthesize")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: %s\n", artifactCommandUsage(command))
		fmt.Fprintf(fs.Output(), "\n%s\n", artifactCommandDescription(command))
		fmt.Fprintf(fs.Output(), "\nExamples:\n")
		switch command {
		case "synthesize":
			fmt.Fprintf(fs.Output(), "  openudon synthesize --example examples/support-email --provider copilot-api --model gpt-5.4-mini --max-attempts 5\n")
		case "build":
			fmt.Fprintf(fs.Output(), "  openudon build --example examples/support-email\n")
		case "promote":
			fmt.Fprintf(fs.Output(), "  openudon promote --example examples/support-email\n")
		case "assess":
			fmt.Fprintf(fs.Output(), "  openudon assess --example examples/support-email\n")
		}
		fmt.Fprintf(fs.Output(), "\nArtifacts:\n")
		fmt.Fprintf(fs.Output(), "  workflows/intent.hcl        structured intent generated from project.md\n")
		fmt.Fprintf(fs.Output(), "  workflows/workflow.hcl      public UWS HCL artifact\n")
		fmt.Fprintf(fs.Output(), "  workflows/workflow.uws.yaml public UWS YAML artifact\n")
		fmt.Fprintf(fs.Output(), "  expected/plan.json          expected operations, bindings, credentials, and control flow\n")
		fmt.Fprintf(fs.Output(), "  expected/review.md          trusted execution review evidence and handoff notes\n")
		fmt.Fprintf(fs.Output(), "  expected/review-handoff.json machine-readable review approval handoff\n")
		fmt.Fprintf(fs.Output(), "  expected/quality.json       deterministic quality gate results\n\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	opts := synthesize.Options{
		ExampleDir:        *example,
		Provider:          *provider,
		Model:             *model,
		Timeout:           *timeout,
		MaxAttempts:       *maxAttempts,
		IntentTemperature: temperature,
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	var result *synthesize.Result
	var report *synthesize.QualityReport
	var err error
	switch command {
	case "synthesize":
		result, err = synthesize.Synthesize(ctx, opts)
	case "build":
		result, err = synthesize.Build(ctx, opts)
	case "promote":
		result, err = synthesize.Promote(ctx, opts)
	case "assess":
		report, err = synthesize.AssessContext(ctx, opts)
		if err == nil {
			printQuality(report)
			if !report.Passed() {
				os.Exit(1)
			}
			return
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		if report, qerr := synthesize.Assess(opts); qerr == nil {
			fmt.Fprintf(os.Stderr, "quality report: %s\n", report.Artifacts.QualityJSONPath)
		}
		os.Exit(1)
	}
	printResult(command, result)
}

func artifactCommandUsage(command string) string {
	switch command {
	case "synthesize":
		return "openudon synthesize --example examples/<name> [--provider copilot-api --model gpt-5.4-mini]"
	case "build":
		return "openudon build --example examples/<name> [--provider label --model label]"
	case "promote":
		return "openudon promote --example examples/<name>"
	case "assess":
		return "openudon assess --example examples/<name>"
	default:
		return "openudon " + command + " --example examples/<name>"
	}
}

func printResult(command string, result *synthesize.Result) {
	if result == nil {
		return
	}
	fmt.Printf("openudon: %s %s\n", command, result.ExampleDir)
	if result.PrimaryOpenAPI != "" {
		fmt.Printf("  openapi:  %s\n", result.PrimaryOpenAPI)
	}
	fmt.Printf("  intent:   %s\n", result.IntentPath)
	fmt.Printf("  workflow: %s\n", result.WorkflowPath)
	fmt.Printf("  uws:      %s\n", result.UWSPath)
	fmt.Printf("  plan:     %s\n", result.PlanJSONPath)
	fmt.Printf("  refine:   %s\n", result.RefinementJSONPath)
	fmt.Printf("  review:   %s\n", result.ReviewPath)
	fmt.Printf("  quality:  %s\n", result.QualityJSONPath)
}

func artifactCommandDescription(command string) string {
	switch command {
	case "synthesize":
		return "Generate intent, workflow, UWS, plan, review evidence, refinement report, and quality report from project.md."
	case "build":
		return "Deterministically regenerate workflow, UWS, review evidence, and quality reports from an existing workflows/intent.hcl; no LLM is required."
	case "promote":
		return "Export and validate workflows/workflow.uws.yaml from an existing workflows/workflow.hcl."
	case "assess":
		return "Run deterministic quality gates against existing artifacts and rewrite expected/quality.{json,md}."
	default:
		return "Run a OpenUdon artifact command."
	}
}

func printQuality(report *synthesize.QualityReport) {
	if report == nil {
		return
	}
	fmt.Printf("openudon: quality %s\n", report.Status)
	fmt.Printf("  report: %s\n", report.Artifacts.QualityJSONPath)
	for _, check := range report.Checks {
		fmt.Printf("  %s: %s\n", check.Code, check.Status)
		if check.Status == "fail" {
			if check.Detail != "" {
				fmt.Printf("    detail: %s\n", check.Detail)
			}
			if next := qualityremediation.NextAction(check.Code); next != "" {
				fmt.Printf("    next: %s\n", next)
			}
		}
	}
}

func nextActionForQualityCheck(code string) string {
	return qualityremediation.NextAction(code)
}
