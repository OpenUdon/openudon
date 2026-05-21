package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"github.com/OpenUdon/apitools"
	"github.com/OpenUdon/apitools/catalog"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

func runCatalogCommand(args []string) {
	if len(args) == 0 {
		printCatalogUsage(os.Stderr)
		os.Exit(2)
	}
	switch args[0] {
	case "list":
		runCatalogListCommand(args[1:])
	case "inspect":
		runCatalogInspectCommand(args[1:])
	case "advisory":
		runCatalogAdvisoryCommand(args[1:])
	case "specs":
		runCatalogSpecsCommand(args[1:])
	case "security-report":
		runCatalogSecurityReportCommand(args[1:])
	case "import-openapi":
		runCatalogImportOpenAPICommand(args[1:])
	case "-h", "--help", "help":
		printCatalogUsage(os.Stdout)
	default:
		fmt.Fprintf(os.Stderr, "unknown catalog command %q\n", args[0])
		printCatalogUsage(os.Stderr)
		os.Exit(2)
	}
}

func printCatalogUsage(out io.Writer) {
	fmt.Fprintln(out, "Usage: openudon catalog <command>")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Commands:")
	fmt.Fprintln(out, "  list             list first-class provider catalog metadata")
	fmt.Fprintln(out, "  inspect          inspect one provider and resolution status")
	fmt.Fprintln(out, "  advisory         render provider or example catalog advisory summaries")
	fmt.Fprintln(out, "  specs            list directly refreshable built-in spec references")
	fmt.Fprintln(out, "  security-report  report auth/security status across providers")
	fmt.Fprintln(out, "  import-openapi   import a provider-owned OpenAPI spec into an example")
}

func runCatalogListCommand(args []string) {
	fs := flag.NewFlagSet("catalog list", flag.ExitOnError)
	jsonOut := fs.Bool("json", false, "Write JSON output")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: openudon catalog list [--json]\n")
		fmt.Fprintf(fs.Output(), "\nLists built-in provider catalog metadata. Catalog metadata is advisory and does not execute API operations.\n\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "unexpected argument %q\n", fs.Arg(0))
		os.Exit(2)
	}
	report, err := catalog.BuiltInSecurityReport()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	statusByProvider := map[string]catalog.AuthCompletenessStatus{}
	for _, row := range catalog.SecurityReportRows(report) {
		statusByProvider[row.ProviderID] = row.Status
	}
	rows := make([]catalogListRow, 0, len(catalog.BuiltInProviders()))
	for _, provider := range catalog.BuiltInProviders() {
		rows = append(rows, catalogListRow{
			ID:                  provider.ID,
			DisplayName:         provider.DisplayName,
			Category:            provider.Category,
			OpenAPIAvailability: provider.OfficialOpenAPIAvailability,
			MachineAvailability: provider.OfficialMachineSpecAvailability,
			UserOpenAPINeed:     provider.UserOpenAPINeed,
			AuthStatus:          statusByProvider[provider.ID],
		})
	}
	if *jsonOut {
		writeCLIJSONOrExit(rows)
		return
	}
	fmt.Printf("%-22s %-28s %-18s %-18s %-18s %s\n", "ID", "NAME", "OPENAPI", "MACHINE", "USER_OPENAPI", "AUTH")
	for _, row := range rows {
		fmt.Printf("%-22s %-28s %-18s %-18s %-18s %s\n", row.ID, row.DisplayName, row.OpenAPIAvailability, row.MachineAvailability, row.UserOpenAPINeed, row.AuthStatus)
	}
}

type catalogListRow struct {
	ID                  string                         `json:"id"`
	DisplayName         string                         `json:"display_name"`
	Category            string                         `json:"category,omitempty"`
	OpenAPIAvailability catalog.SpecAvailability       `json:"openapi_availability"`
	MachineAvailability catalog.SpecAvailability       `json:"machine_availability"`
	UserOpenAPINeed     catalog.UserOpenAPINeed        `json:"user_openapi_need"`
	AuthStatus          catalog.AuthCompletenessStatus `json:"auth_status"`
}

func runCatalogInspectCommand(args []string) {
	var providerKey string
	parseArgs := args
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		providerKey = args[0]
		parseArgs = args[1:]
	}
	fs := flag.NewFlagSet("catalog inspect", flag.ExitOnError)
	userOpenAPI := fs.String("openapi", "", "User-provided OpenAPI path or URL; overrides built-in specs")
	userOverlay := fs.String("security-overlay", "", "User-provided security overlay path or URL; overrides built-in security overlays")
	localOpenAPI := fs.String("local-openapi", "", "Project-local OpenAPI path; used before built-in specs")
	jsonOut := fs.Bool("json", false, "Write JSON output")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: openudon catalog inspect <provider> [--openapi path-or-url] [--security-overlay path-or-url] [--local-openapi path] [--json]\n")
		fmt.Fprintf(fs.Output(), "\nInspects provider catalog metadata and resolution precedence without reading secrets or executing API operations.\n\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(parseArgs); err != nil {
		os.Exit(2)
	}
	if providerKey == "" && fs.NArg() > 0 {
		providerKey = fs.Arg(0)
	}
	if providerKey == "" || fs.NArg() > 1 {
		fs.Usage()
		os.Exit(2)
	}
	resolved, err := catalog.ResolveProvider(catalog.ResolveProviderOptions{
		ProviderKey:         providerKey,
		UserOpenAPI:         *userOpenAPI,
		UserSecurityOverlay: *userOverlay,
		ProjectLocalOpenAPI: *localOpenAPI,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *jsonOut {
		writeCLIJSONOrExit(resolved)
		return
	}
	writeCatalogInspectText(os.Stdout, resolved)
}

func writeCatalogInspectText(out io.Writer, resolved catalog.ResolvedProvider) {
	fmt.Fprintf(out, "Provider: %s (%s)\n", resolved.Provider.DisplayName, resolved.Provider.ID)
	fmt.Fprintf(out, "Category: %s\n", resolved.Provider.Category)
	fmt.Fprintf(out, "OpenAPI availability: %s\n", resolved.Provider.OfficialOpenAPIAvailability)
	fmt.Fprintf(out, "Machine spec availability: %s\n", resolved.Provider.OfficialMachineSpecAvailability)
	fmt.Fprintf(out, "User OpenAPI need: %s\n", resolved.Provider.UserOpenAPINeed)
	fmt.Fprintf(out, "Resolved API metadata: %s", resolved.OpenAPI.Source)
	if resolved.OpenAPI.Value != "" {
		fmt.Fprintf(out, " %s", resolved.OpenAPI.Value)
	}
	if resolved.OpenAPI.SpecRefID != "" {
		fmt.Fprintf(out, " (%s)", resolved.OpenAPI.SpecRefID)
	}
	fmt.Fprintln(out)
	fmt.Fprintf(out, "Auth/security status: %s\n", resolved.SecurityStatus)
	fmt.Fprintf(out, "Resolved security: %s", resolved.Security.Source)
	if resolved.Security.Value != "" {
		fmt.Fprintf(out, " %s", resolved.Security.Value)
	}
	if resolved.Security.OverlayID != "" {
		fmt.Fprintf(out, " (%s)", resolved.Security.OverlayID)
	}
	fmt.Fprintln(out)
	for _, note := range nonEmptyCatalogNotes(resolved.OpenAPI.SourceNote, resolved.Security.SourceNote) {
		fmt.Fprintf(out, "Source note: %s\n", note)
	}
	if len(resolved.CatalogSpecReferences) > 0 {
		fmt.Fprintln(out, "Spec references:")
		for _, ref := range resolved.CatalogSpecReferences {
			fmt.Fprintf(out, "  - %s %s %s\n", ref.ID, ref.Kind, ref.URL)
		}
	}
	if len(resolved.CatalogSecurityOverlays) > 0 {
		fmt.Fprintln(out, "Security overlays:")
		for _, overlay := range resolved.CatalogSecurityOverlays {
			fmt.Fprintf(out, "  - %s %s\n", overlay.ID, overlay.Status)
		}
	}
}

func runCatalogAdvisoryCommand(args []string) {
	var provider string
	parseArgs := args
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		provider = args[0]
		parseArgs = args[1:]
	}
	fs := flag.NewFlagSet("catalog advisory", flag.ExitOnError)
	example := fs.String("example", "", "Example directory whose workflows/intent.hcl should receive catalog advice")
	jsonOut := fs.Bool("json", false, "Write JSON output")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: openudon catalog advisory [provider] [--example examples/<name>] [--json]\n")
		fmt.Fprintf(fs.Output(), "\nRenders advisory provider/spec/security metadata. Advisory output never imports n8n behavior, resolves credentials, or executes API operations.\n\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(parseArgs); err != nil {
		os.Exit(2)
	}
	if provider == "" && fs.NArg() > 0 {
		provider = fs.Arg(0)
	}
	if fs.NArg() > 1 {
		fmt.Fprintf(os.Stderr, "unexpected argument %q\n", fs.Arg(1))
		os.Exit(2)
	}
	if strings.TrimSpace(*example) != "" {
		if strings.TrimSpace(provider) != "" {
			fmt.Fprintln(os.Stderr, "--example cannot be combined with a provider argument")
			os.Exit(2)
		}
		runCatalogExampleAdvisory(*example, *jsonOut)
		return
	}
	report, err := catalog.BuiltInProviderAdvisoryReport(catalog.ProviderAdvisoryOptions{ProviderKey: provider})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *jsonOut {
		writeCLIJSONOrExit(report)
		return
	}
	writeCatalogProviderAdvisoryText(os.Stdout, report)
}

func runCatalogExampleAdvisory(example string, jsonOut bool) {
	intentPath := filepath.Join(example, "workflows", "intent.hcl")
	intent, err := rollout.ParseIntentFile(intentPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	report, err := rollout.CatalogAdviceForIntent(intent, rollout.CatalogAdviceOptions{})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if jsonOut {
		writeCLIJSONOrExit(report)
		return
	}
	if markdown := rollout.RenderCatalogAdviceMarkdown(report); markdown != "" {
		fmt.Print(markdown)
		return
	}
	fmt.Printf("openudon: no catalog provider matches found for %s\n", intentPath)
}

func writeCatalogProviderAdvisoryText(out io.Writer, report catalog.ProviderAdvisoryReport) {
	if len(report.Providers) == 0 {
		fmt.Fprintln(out, "No catalog advisory providers matched.")
		return
	}
	for _, provider := range report.Providers {
		fmt.Fprintf(out, "Provider: %s (%s)\n", provider.DisplayName, provider.ProviderID)
		fmt.Fprintf(out, "  OpenAPI: %s; machine spec: %s; user OpenAPI: %s\n", provider.OpenAPIAvailability, provider.MachineSpecAvailability, provider.UserOpenAPINeed)
		fmt.Fprintf(out, "  Auth/security: %s\n", provider.AuthStatus)
		if provider.ResolvedOpenAPI.Source != catalog.ResolutionSourceNone {
			fmt.Fprintf(out, "  Resolved API metadata: %s", provider.ResolvedOpenAPI.Source)
			if provider.ResolvedOpenAPI.Value != "" {
				fmt.Fprintf(out, " %s", provider.ResolvedOpenAPI.Value)
			}
			if provider.ResolvedOpenAPI.SpecRefID != "" {
				fmt.Fprintf(out, " (%s)", provider.ResolvedOpenAPI.SpecRefID)
			}
			fmt.Fprintln(out)
		}
		if provider.ResolvedSecurity.Source != catalog.ResolutionSourceNone {
			fmt.Fprintf(out, "  Resolved security: %s", provider.ResolvedSecurity.Source)
			if provider.ResolvedSecurity.OverlayID != "" {
				fmt.Fprintf(out, " (%s)", provider.ResolvedSecurity.OverlayID)
			}
			fmt.Fprintln(out)
		}
		for _, followUp := range provider.ManualFollowUps {
			fmt.Fprintf(out, "  Follow-up: %s\n", followUp)
		}
	}
}

func runCatalogSpecsCommand(args []string) {
	fs := flag.NewFlagSet("catalog specs", flag.ExitOnError)
	jsonOut := fs.Bool("json", false, "Write JSON output")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: openudon catalog specs [--json]\n")
		fmt.Fprintf(fs.Output(), "\nLists directly refreshable built-in catalog spec references. OpenUdon can import only actual OpenAPI references into package openapi/ directories.\n\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "unexpected argument %q\n", fs.Arg(0))
		os.Exit(2)
	}
	rows := catalog.BuiltInRefreshableSpecReferences(nil)
	if *jsonOut {
		writeCLIJSONOrExit(rows)
		return
	}
	fmt.Printf("%-22s %-38s %-18s %-18s %s\n", "PROVIDER", "SPEC_REF", "KIND", "SOURCE", "URL")
	for _, row := range rows {
		fmt.Printf("%-22s %-38s %-18s %-18s %s\n", row.ProviderID, row.SpecRefID, row.Kind, row.SourceAuthority, row.URL)
	}
}

func runCatalogSecurityReportCommand(args []string) {
	fs := flag.NewFlagSet("catalog security-report", flag.ExitOnError)
	jsonOut := fs.Bool("json", false, "Write JSON output")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: openudon catalog security-report [--json]\n")
		fmt.Fprintf(fs.Output(), "\nReports catalog auth/security metadata completeness without resolving credentials.\n\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "unexpected argument %q\n", fs.Arg(0))
		os.Exit(2)
	}
	report, err := catalog.BuiltInSecurityReport()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	rows := catalog.SecurityReportRows(report)
	if *jsonOut {
		writeCLIJSONOrExit(rows)
		return
	}
	fmt.Printf("%-22s %-20s %s\n", "PROVIDER", "AUTH_STATUS", "OVERLAYS")
	for _, row := range rows {
		fmt.Printf("%-22s %-20s %s\n", row.ProviderID, row.Status, strings.Join(row.OverlayIDs, ","))
	}
}

func runCatalogImportOpenAPICommand(args []string) {
	fs := flag.NewFlagSet("catalog import-openapi", flag.ExitOnError)
	providerKey := fs.String("provider", "", "Provider ID, display name, or alias")
	specRefID := fs.String("spec", "", "Catalog spec reference ID; required when provider has multiple OpenAPI specs")
	example := fs.String("example", "", "Example directory whose openapi/ directory should receive the imported spec")
	name := fs.String("name", "", "Suggested imported filename stem; defaults to provider ID")
	jsonOut := fs.Bool("json", false, "Write JSON output")
	timeout := fs.Duration("timeout", apitools.DefaultTimeout, "Download timeout")
	maxBytes := fs.Int64("max-bytes", apitools.DefaultMaxBytes, "Maximum bytes to download")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: openudon catalog import-openapi --provider <provider> --example examples/<name> [--spec <spec-ref>] [--name <stem>] [--json]\n")
		fmt.Fprintf(fs.Output(), "\nImports a provider-owned OpenAPI spec directly from the built-in catalog. Discovery, Smithy, Stone, and human-docs refs remain advisory and are not written as OpenAPI.\n\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "unexpected argument %q\n", fs.Arg(0))
		os.Exit(2)
	}
	ref, provider, err := selectCatalogOpenAPIReference(*providerKey, *specRefID)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	exampleDir := strings.TrimSpace(*example)
	if exampleDir == "" {
		fmt.Fprintln(os.Stderr, "missing --example")
		os.Exit(2)
	}
	targetDir := filepath.Join(exampleDir, "openapi")
	stem := strings.TrimSpace(*name)
	if stem == "" {
		stem = provider.ID
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	client := &apitools.Client{
		Timeout:  *timeout,
		MaxBytes: *maxBytes,
	}
	imported, err := client.Import(ctx, apitools.ImportOptions{
		URL:  ref.URL,
		Dir:  targetDir,
		Name: stem,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	result := catalogImportResult{
		ProviderID:   provider.ID,
		ProviderName: provider.DisplayName,
		SpecRefID:    ref.ID,
		Kind:         ref.Kind,
		SourceURL:    ref.URL,
		Path:         filepath.ToSlash(imported.Path),
		FileName:     imported.Name,
		SHA256:       imported.SHA256,
	}
	if *jsonOut {
		writeCLIJSONOrExit(result)
		return
	}
	fmt.Printf("openudon: imported %s (%s) OpenAPI spec to %s\n", provider.DisplayName, ref.ID, imported.Path)
}

type catalogImportResult struct {
	ProviderID   string           `json:"provider_id"`
	ProviderName string           `json:"provider_name"`
	SpecRefID    string           `json:"spec_ref_id"`
	Kind         catalog.SpecKind `json:"kind"`
	SourceURL    string           `json:"source_url"`
	Path         string           `json:"path"`
	FileName     string           `json:"file_name"`
	SHA256       string           `json:"sha256"`
}

func selectCatalogOpenAPIReference(providerKey, specRefID string) (catalog.SpecReference, catalog.Provider, error) {
	providerKey = strings.TrimSpace(providerKey)
	if providerKey == "" {
		return catalog.SpecReference{}, catalog.Provider{}, fmt.Errorf("missing --provider")
	}
	provider, ok := catalog.FindBuiltInProvider(providerKey)
	if !ok {
		return catalog.SpecReference{}, catalog.Provider{}, fmt.Errorf("unknown provider %q", providerKey)
	}
	specRefID = strings.TrimSpace(specRefID)
	var refs []catalog.SpecReference
	for _, ref := range provider.SpecReferences {
		if ref.Kind != catalog.SpecKindOpenAPI {
			continue
		}
		if specRefID != "" && ref.ID != specRefID {
			continue
		}
		refs = append(refs, ref)
	}
	if len(refs) == 0 {
		if specRefID != "" {
			return catalog.SpecReference{}, provider, fmt.Errorf("provider %q has no OpenAPI spec reference %q", provider.ID, specRefID)
		}
		return catalog.SpecReference{}, provider, fmt.Errorf("provider %q has no directly importable OpenAPI spec; inspect catalog metadata for Discovery, Smithy, Stone, human-docs, or user-provided OpenAPI guidance", provider.ID)
	}
	if specRefID == "" && len(refs) > 1 {
		var ids []string
		for _, ref := range refs {
			ids = append(ids, ref.ID)
		}
		sort.Strings(ids)
		return catalog.SpecReference{}, provider, fmt.Errorf("provider %q has multiple OpenAPI specs; pass --spec (%s)", provider.ID, strings.Join(ids, ", "))
	}
	return refs[0], provider, nil
}

func nonEmptyCatalogNotes(values ...string) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func writeCLIJSONOrExit(value any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(value); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
