package icot

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	transactionengine "github.com/OpenUdon/openudon/internal/browsertransaction/engine"
	"github.com/OpenUdon/openudon/internal/icot/elicitor"
	"github.com/OpenUdon/openudon/internal/icot/engine"
	uiserver "github.com/OpenUdon/openudon/internal/icot/ui"
	"github.com/OpenUdon/openudon/internal/packagepipeline"
)

var runUIServer = uiserver.Run

func runUI(args []string, out, errOut io.Writer) int {
	return runApplication(args, nil, out, errOut)
}

func runApplication(args []string, input io.Reader, out, errOut io.Writer) int {
	commandName := "icot ui"
	if input != nil {
		commandName = "icot control"
	}
	fs := flag.NewFlagSet(commandName, flag.ContinueOnError)
	fs.SetOutput(out)
	example := fs.String("example", "", "Example directory for the single UI workspace")
	fromExample := fs.String("from-example", "", "Seed authoring from an existing example directory")
	answersFile := fs.String("answers", "", "Path to an openudon.icot-session.v2 YAML or JSON file")
	var apiSourceFlags repeatedFlag
	var openAPIFlags repeatedFlag
	var browserProfileFlags repeatedFlag
	var browserVerificationFlags repeatedFlag
	var browserRegistryFlags repeatedFlag
	var sourceRootFlags repeatedFlag
	fs.Var(&apiSourceFlags, "api-source", "Explicit API document KIND:ID=PATH; repeat for multiple sources")
	fs.Var(&openAPIFlags, "openapi", "OpenAPI shorthand ID=PATH; repeat for multiple sources")
	fs.Var(&browserProfileFlags, "browser-profile", "Verified browser capability/authentication profile, capability bundle, or guided-authoring result ID=PATH; repeat for multiple sources")
	fs.Var(&browserVerificationFlags, "browser-verification", "Value-free Browsertools live-check or portability report path; repeat for multiple reports")
	fs.Var(&browserRegistryFlags, "browser-registry", "Static Browsertools registry directory or HTTPS URL; repeat for multiple registries")
	fs.Var(&sourceRootFlags, "source-root", "Explicit bounded local source root; repeat for multiple roots")
	network := fs.String("network", "", "Remote lookup policy: never, ask, or allow")
	protocol := fs.String("protocol", uiserver.RegistrationControlVersion, "Private control protocol version (control only)")
	port := fs.Int("port", 0, "Loopback TCP port; 0 selects an ephemeral port")
	noOpen := fs.Bool("no-open", false, "Do not open the bootstrap URL in the platform browser")
	privateRoot := fs.String("private-root", "", "absolute mode-0700 private root required only for upload, browser capture, or registration authoring")
	registrationAuthorityPath := fs.String("registration-authority", "", "Optional owner-only file fixing one consumer registration-authoring authority")
	driverDir := fs.String("driver-dir", "", "optional installed Playwright-Go driver directory for browser capture")
	browserTransactionPath := fs.String("browser-transaction", "", "optional public browser-profile transaction v1/v2 JSON file")
	packageScope := fs.String("package-scope", "", "portable package scope for browser-transaction preparation")
	packageScratch := fs.String("package-scratch", "", "existing absolute restrictive-scratch parent for browser-transaction preparation")
	packageStore := fs.String("package-store", "", "existing generation store for browser-transaction promotion")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: "+commandName+" --example DIR [--from-example DIR | --answers FILE] [--api-source KIND:ID=PATH] [--openapi ID=PATH] [--browser-profile ID=PATH] [--browser-verification PATH] [--browser-registry LOCATION] [--source-root DIR] [--network never|ask|allow] [--package-scope PORTABLE --package-scratch DIR --package-store DIR [--browser-transaction FILE]] [--port PORT] [--no-open]")
		if input == nil {
			fmt.Fprintln(fs.Output(), "\nServes one explicitly named workspace on 127.0.0.1 with a per-process capability token.")
		} else {
			fmt.Fprintln(fs.Output(), "\nRequires --no-open and port 0. Uses private NDJSON pipes without an HTTP listener.")
		}
		fmt.Fprintln(fs.Output(), "The embedded shell supports acquisition, revision-protected authoring, reviewed package build, and handoff over experimental API v4.")
		fmt.Fprintln(fs.Output(), "External changes to engine-owned files preserve cached inspection but require a process restart before mutation.")
		fmt.Fprintln(fs.Output())
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(errOut, commandName+": unexpected positional arguments")
		return 2
	}
	if (*protocol != uiserver.RegistrationControlVersion && *protocol != uiserver.ApplicationControlVersion) || (input == nil && *protocol != uiserver.RegistrationControlVersion) {
		fmt.Fprintln(errOut, commandName+": unsupported control protocol")
		return 2
	}
	if input != nil && (*port != 0 || !*noOpen) {
		fmt.Fprintln(errOut, "icot control: requires --no-open and no port")
		return 2
	}
	exampleDir := strings.TrimSpace(*example)
	if exampleDir == "" {
		fmt.Fprintln(errOut, commandName+": --example is required")
		return 2
	}
	if strings.TrimSpace(*answersFile) != "" && strings.TrimSpace(*fromExample) != "" {
		fmt.Fprintln(errOut, commandName+": --answers and --from-example are mutually exclusive")
		return 2
	}
	if *port < 0 || *port > 65535 {
		fmt.Fprintln(errOut, commandName+": --port must be between 0 and 65535")
		return 2
	}
	transactionPath := strings.TrimSpace(*browserTransactionPath)
	packageOptions := []string{strings.TrimSpace(*packageScope), strings.TrimSpace(*packageScratch), strings.TrimSpace(*packageStore)}
	configuredPackageOptions := 0
	for _, value := range packageOptions {
		if value != "" {
			configuredPackageOptions++
		}
	}
	if (configuredPackageOptions != 0 && configuredPackageOptions != len(packageOptions)) || (transactionPath != "" && configuredPackageOptions != len(packageOptions)) {
		fmt.Fprintln(errOut, commandName+": --package-scope, --package-scratch, and --package-store must be supplied together; --browser-transaction additionally requires that package configuration")
		return 2
	}
	localSources, err := parseLocalSourceFlags(apiSourceFlags, openAPIFlags)
	if err != nil {
		fmt.Fprintln(errOut, commandName+":", err)
		return 2
	}
	browserSources, err := parseBrowserSourceFlags(browserProfileFlags)
	if err != nil {
		fmt.Fprintln(errOut, commandName+":", err)
		return 2
	}
	networkPolicy, err := resolveNetworkPolicy(*network, false)
	if err != nil {
		fmt.Fprintln(errOut, commandName+":", err)
		return 2
	}

	engineConfig := engine.Config{
		ExampleDir:           exampleDir,
		LocalSources:         localSources,
		BrowserSources:       browserSources,
		BrowserVerifications: append([]string(nil), browserVerificationFlags...),
		BrowserRegistries:    append([]string(nil), browserRegistryFlags...),
		SourceRoots:          append([]string(nil), sourceRootFlags...),
		NetworkPolicy:        networkPolicy,
		PrivateRoot:          strings.TrimSpace(*privateRoot),
		DriverDir:            strings.TrimSpace(*driverDir),
	}
	configureUISeed(&engineConfig, strings.TrimSpace(*answersFile), strings.TrimSpace(*fromExample))

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	var browserTransactions uiserver.BrowserTransactionEngine
	if configuredPackageOptions == len(packageOptions) {
		var transactionEngine *transactionengine.Engine
		var err error
		if transactionPath != "" {
			transactionEngine, _, err = openBrowserTransaction(ctx, transactionPath, browserTransactionPackageOptions{
				exampleDir: exampleDir, scope: packageOptions[0], scratchParent: packageOptions[1], storeDir: packageOptions[2],
			})
		} else {
			transactionEngine, _, err = transactionengine.New(transactionengine.Config{Package: packagepipeline.CurrentOptions{
				ExampleDir: exampleDir, Scope: packageOptions[0], ScratchParent: packageOptions[1], StoreDir: packageOptions[2],
			}})
		}
		if err != nil {
			_, code, operation, _, ok := transactionengine.ErrorDetails(err)
			if ok {
				fmt.Fprintf(errOut, commandName+": browser transaction initialization failed: %s/%s\n", operation, code)
			} else {
				fmt.Fprintln(errOut, commandName+": browser transaction initialization failed")
			}
			return 1
		}
		browserTransactions = transactionEngine
	}
	config := uiserver.RunConfig{
		EngineConfig: engineConfig, Port: *port, NoOpen: *noOpen, Out: out, ErrOut: errOut,
		PrepareCapture:      prepareUICaptureStage,
		BrowserTransactions: browserTransactions,
	}
	if *registrationAuthorityPath != "" {
		authority, err := uiserver.ReadRegistrationAuthority(*registrationAuthorityPath, time.Now())
		if err != nil {
			fmt.Fprintln(errOut, "icot: invalid registration authority")
			return 1
		}
		config.RegistrationAuthority = authority
	}
	runner := runUIServer
	if input != nil {
		runner = func(ctx context.Context, config uiserver.RunConfig) error {
			owned, ok := input.(io.ReadCloser)
			if !ok {
				owned = io.NopCloser(input)
			}
			if *protocol == uiserver.ApplicationControlVersion {
				return uiserver.RunApplicationControl(ctx, config, owned)
			}
			return uiserver.RunRegistrationControl(ctx, config, owned)
		}
	}
	if err := runner(ctx, config); err != nil {
		fmt.Fprintln(errOut, commandName+":", err)
		return 1
	}
	return 0
}

func prepareUICaptureStage(request uiserver.CaptureStageRequest) (engine.BrowserCaptureStage, error) {
	start := request.Start
	goalURL := strings.TrimSpace(start.GoalOrigin) + strings.TrimSpace(start.GoalPath)
	cfg := liveAuthorConfig{
		ExampleDir: request.ExampleDir, PrivateRoot: request.PrivateRoot,
		URL: start.URL, DashboardURL: start.DashboardURL, GoalURL: goalURL, Goal: start.Goal,
		Origins: append([]string(nil), start.Origins...), ProfileID: start.ProfileID,
		AfterAuthentication: "navigate_absolute", GoalRole: start.GoalRole, GoalLabel: start.GoalLabel, GoalContext: start.GoalContext,
	}
	if err := normalizeLiveAuthorConfig(&cfg); err != nil {
		return engine.BrowserCaptureStage{}, err
	}
	prepared, err := prepareAttestedAuthenticatedAuthoringImport(cfg, liveProtocolResult{
		ArtifactPath: request.Result.ArtifactPath, Digest: request.Result.Digest, Attestation: request.Attestation,
	}, time.Now().UTC())
	if err != nil {
		return engine.BrowserCaptureStage{}, err
	}
	stage := engine.BrowserCaptureStage{ProfileID: cfg.ProfileID, Candidate: prepared.Candidate}
	for _, file := range prepared.Files {
		switch filepath.Clean(file.Path) {
		case filepath.Join(cfg.ExampleDir, filepath.FromSlash(prepared.AuthenticationTarget)):
			stage.Authentication = []byte(file.Content)
		case filepath.Join(cfg.ExampleDir, filepath.FromSlash(prepared.CapabilityTarget)):
			stage.Capability = []byte(file.Content)
		case filepath.Join(cfg.ExampleDir, ".icot", "authenticated-browser-authoring.json"):
			stage.SafeReview = []byte(file.Content)
		}
	}
	return stage, nil
}

func configureUISeed(config *engine.Config, answersFile, fromExample string) {
	if answersFile != "" {
		config.SessionPath = answersFile
		return
	}
	if fromExample != "" {
		config.FromExample = fromExample
		return
	}
	if pathExists(elicitor.DraftPath(config.ExampleDir)) {
		return
	}
	if pathExists(filepath.Join(config.ExampleDir, "project.md")) || pathExists(filepath.Join(config.ExampleDir, "workflows", "intent.hcl")) {
		config.LoadExisting = true
	}
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
