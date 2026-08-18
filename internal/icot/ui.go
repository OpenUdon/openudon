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

	"github.com/OpenUdon/openudon/internal/icot/elicitor"
	"github.com/OpenUdon/openudon/internal/icot/engine"
	uiserver "github.com/OpenUdon/openudon/internal/icot/ui"
)

var runUIServer = uiserver.Run

func runUI(args []string, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("icot ui", flag.ContinueOnError)
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
	port := fs.Int("port", 0, "Loopback TCP port; 0 selects an ephemeral port")
	noOpen := fs.Bool("no-open", false, "Do not open the bootstrap URL in the platform browser")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: icot ui --example DIR [--from-example DIR | --answers FILE] [--api-source KIND:ID=PATH] [--openapi ID=PATH] [--browser-profile ID=PATH] [--browser-verification PATH] [--browser-registry LOCATION] [--source-root DIR] [--network never|ask|allow] [--port PORT] [--no-open]")
		fmt.Fprintln(fs.Output(), "\nServes one explicitly named workspace on 127.0.0.1 with a per-process capability token.")
		fmt.Fprintln(fs.Output(), "The embedded Phase B shell polls experimental API v2 and remains read-only; mutations require exact revisions.")
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
		fmt.Fprintln(errOut, "icot ui: unexpected positional arguments")
		return 2
	}
	exampleDir := strings.TrimSpace(*example)
	if exampleDir == "" {
		fmt.Fprintln(errOut, "icot ui: --example is required")
		return 2
	}
	if strings.TrimSpace(*answersFile) != "" && strings.TrimSpace(*fromExample) != "" {
		fmt.Fprintln(errOut, "icot ui: --answers and --from-example are mutually exclusive")
		return 2
	}
	if *port < 0 || *port > 65535 {
		fmt.Fprintln(errOut, "icot ui: --port must be between 0 and 65535")
		return 2
	}
	localSources, err := parseLocalSourceFlags(apiSourceFlags, openAPIFlags)
	if err != nil {
		fmt.Fprintln(errOut, "icot ui:", err)
		return 2
	}
	browserSources, err := parseBrowserSourceFlags(browserProfileFlags)
	if err != nil {
		fmt.Fprintln(errOut, "icot ui:", err)
		return 2
	}
	networkPolicy, err := resolveNetworkPolicy(*network, false)
	if err != nil {
		fmt.Fprintln(errOut, "icot ui:", err)
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
	}
	configureUISeed(&engineConfig, strings.TrimSpace(*answersFile), strings.TrimSpace(*fromExample))

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if err := runUIServer(ctx, uiserver.RunConfig{
		EngineConfig: engineConfig, Port: *port, NoOpen: *noOpen, Out: out, ErrOut: errOut,
	}); err != nil {
		fmt.Fprintln(errOut, "icot ui:", err)
		return 1
	}
	return 0
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
