package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/OpenUdon/apitools"
	"github.com/OpenUdon/apitools/catalog"
	"github.com/OpenUdon/openudon/internal/evidencefile"
	"github.com/OpenUdon/openudon/internal/stepauthoring"
)

// Discovery preserves the native APItools request/report contract. Installation
// flags cannot be supplied by the request; discovery never builds an index.
func runStepDiscoverCommand(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("openudon step discover", flag.ContinueOnError)
	fs.SetOutput(io.Discard) // Flag errors must not echo untrusted values or paths.
	root := fs.String("catalog-root", "", "Explicit prepared catalog root (missing configuration is incomplete evidence)")
	registry := fs.String("catalog-registry", "", "Registry path relative to catalog root; default cache.sqlite")
	index := fs.String("catalog-index", "", "Index path relative to catalog root; default operations.v1.json")
	metadata := fs.String("catalog-metadata", "", "Explicit installation catalog JSON; default built-in catalog")
	remote := fs.Bool("enable-remote", false, "Allow bounded native remote lookup only when the request also opts in")
	requestPath := fs.String("request", "", "apitools.catalog-discovery/v1 request file, or - for stdin")
	fs.Usage = func() {
		fmt.Fprintln(stderr, "Usage: openudon step discover --request FILE|- [--catalog-root DIR] [--catalog-registry REL] [--catalog-index REL] [--catalog-metadata FILE] [--enable-remote]")
	}
	fail := func(code, message string, exit int) int {
		return writeCatalogDiscoveryReport(stdout, apitools.CatalogDiscoveryReport{
			SchemaVersion: apitools.CatalogDiscoverySchemaVersion, Outcome: apitools.CatalogDiscoveryBlocked,
			Scope: apitools.CatalogDiscoveryScope{ProviderIDs: []string{}}, Incomplete: true,
			Diagnostics: []apitools.Diagnostic{{Severity: "error", Code: code, Message: message}},
		}, exit)
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fs.Usage()
			return 0
		}
		return fail("request.invalid", "The discovery arguments are invalid.", 2)
	}
	if fs.NArg() != 0 || *requestPath == "" {
		return fail("request.invalid", "A discovery request is required.", 2)
	}
	data, err := readStepCheckRequest(*requestPath, stdin)
	if err != nil {
		return fail("request.invalid", "The discovery request is unavailable or exceeds its bound.", 2)
	}
	var strict apitools.CatalogDiscoveryRequest
	if !stepauthoring.ValidUTF8Request(data) || evidencefile.DecodeStrict(data, &strict) != nil {
		return fail("request.invalid", "The discovery request must be unambiguous bounded JSON.", 2)
	}
	request, err := apitools.DecodeCatalogDiscoveryRequest(bytes.NewReader(data))
	if err != nil {
		return fail("request.invalid", "The request does not match the native catalog discovery contract.", 2)
	}
	installation := stepauthoring.CatalogInstallation{Root: catalog.RootOptions{Directory: *root, RegistryPath: *registry, IndexPath: *index}, MetadataPath: *metadata, RemoteEnabled: *remote}
	options, err := installation.IndexOptions()
	if err != nil {
		return fail("discovery.catalog_invalid", "Catalog installation metadata is unavailable or invalid.", 4)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	report, err := apitools.DiscoverCatalogOperations(ctx, apitools.CatalogDiscoveryOptions{Request: request, Index: options, RemoteEnabled: installation.RemoteEnabled})
	exit := 0
	if err != nil || report.Outcome == apitools.CatalogDiscoveryBlocked {
		exit = 4
	}
	return writeCatalogDiscoveryReport(stdout, report, exit)
}

func writeCatalogDiscoveryReport(stdout io.Writer, report apitools.CatalogDiscoveryReport, exit int) int {
	data, err := json.Marshal(report)
	if err != nil || len(data) > apitools.MaxCatalogDiscoveryContextBytes {
		data = []byte(`{"schema_version":"apitools.catalog-discovery/v1","outcome":"blocked","scope":{"provider_ids":[],"provider_constrained":false,"complete":false},"limits":{},"examined_operations":0,"qualified_operations":0,"truncated":false,"incomplete":true,"diagnostics":[{"severity":"error","code":"result.encoding_failed","message":"The discovery result exceeds its output bound."}]}`)
		exit = 1
	}
	if _, err := stdout.Write(append(data, '\n')); err != nil {
		return 1
	}
	return exit
}
