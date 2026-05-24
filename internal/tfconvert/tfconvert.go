package tfconvert

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/OpenUdon/apitools"
	"github.com/OpenUdon/openudon/internal/packageartifacts"
	"github.com/OpenUdon/openudon/internal/synthesize"
	"github.com/OpenUdon/openudon/internal/workflowintent"
	"github.com/OpenUdon/tfconfig"
)

const (
	defaultOutDir = "./.openudon/convert"
)

type Options struct {
	ConfigDir string
	OpenAPIs  []OpenAPIInput
	Action    string
	Targets   []string
	OutDir    string
	Strict    bool
}

type OpenAPIInput struct {
	ID   string
	Path string
}

type Result struct {
	OutDir          string
	ProjectPath     string
	IntentPath      string
	DiagnosticsJSON string
	DiagnosticsMD   string
	ReviewPath      string
	WorkflowPath    string
	UWSPath         string
	PlanJSONPath    string
	PlanMDPath      string
	DiscoveryPath   string
	RefinementPath  string
	HandoffPath     string
	QualityJSONPath string
	QualityMDPath   string
	Diagnostics     []Diagnostic
	StrictFailed    bool
	QualityPassed   bool
}

type Diagnostic struct {
	Code          string       `json:"code"`
	Severity      string       `json:"severity"`
	Message       string       `json:"message"`
	Address       string       `json:"address,omitempty"`
	ModuleAddress string       `json:"module_address,omitempty"`
	SourceRange   *SourceRange `json:"source_range,omitempty"`
	TodoID        string       `json:"todo_id,omitempty"`
	StrictFailure bool         `json:"strict_failure,omitempty"`
}

type SourceRange struct {
	SourceID string   `json:"source_id,omitempty"`
	Path     string   `json:"path,omitempty"`
	Start    Position `json:"start"`
	End      Position `json:"end"`
}

type Position struct {
	Line   int `json:"line"`
	Column int `json:"column"`
	Byte   int `json:"byte,omitempty"`
}

type strictFailureError struct {
	diagnostics []Diagnostic
}

func (e strictFailureError) Error() string {
	return fmt.Sprintf("strict mode failed with %d diagnostic(s)", len(e.diagnostics))
}

func Convert(ctx context.Context, opts Options) (*Result, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	opts = normalizeOptions(opts)

	doc, loadErr := tfconfig.LoadDir(opts.ConfigDir)
	conversion := conversionState{
		opts: opts,
		doc:  doc,
	}
	if loadErr != nil {
		conversion.addDiagnostic(Diagnostic{
			Code:          "tfconfig.load_error",
			Severity:      "error",
			Message:       loadErr.Error(),
			StrictFailure: true,
		})
	}
	conversion.addTFDiagnostics(doc.Diagnostics)
	for _, mod := range doc.Modules {
		conversion.addTFDiagnostics(mod.Diagnostics)
	}

	conversion.loadOpenAPIs(ctx)
	conversion.collectBindings()
	conversion.collectSymbols()
	conversion.selectObjects()
	conversion.validateAction()
	conversion.mapObjects()
	conversion.ensureCredentialBindings()
	conversion.sortAll()

	result := &Result{
		OutDir:          opts.OutDir,
		ProjectPath:     filepath.Join(opts.OutDir, "project.md"),
		IntentPath:      filepath.Join(opts.OutDir, workflowintent.IntentPath),
		DiagnosticsJSON: filepath.Join(opts.OutDir, "expected", "diagnostics.json"),
		DiagnosticsMD:   filepath.Join(opts.OutDir, "expected", "diagnostics.md"),
		ReviewPath:      filepath.Join(opts.OutDir, "expected", "review.md"),
		WorkflowPath:    filepath.Join(opts.OutDir, "workflows", "workflow.hcl"),
		UWSPath:         filepath.Join(opts.OutDir, "workflows", "workflow.uws.yaml"),
		PlanJSONPath:    filepath.Join(opts.OutDir, "expected", "plan.json"),
		PlanMDPath:      filepath.Join(opts.OutDir, "expected", "plan.md"),
		DiscoveryPath:   filepath.Join(opts.OutDir, "expected", "discovery.json"),
		RefinementPath:  filepath.Join(opts.OutDir, "expected", "refinement.json"),
		HandoffPath:     filepath.Join(opts.OutDir, filepath.FromSlash(packageartifacts.ReviewHandoffPath)),
		QualityJSONPath: filepath.Join(opts.OutDir, "expected", "quality.json"),
		QualityMDPath:   filepath.Join(opts.OutDir, "expected", "quality.md"),
		Diagnostics:     conversion.diagnostics,
		StrictFailed:    opts.Strict && hasStrictFailure(conversion.diagnostics),
	}

	if err := writeArtifacts(result, conversion); err != nil {
		return result, err
	}
	if packageResult, quality, err := synthesize.PackageFromIntent(ctx, synthesize.Options{ExampleDir: opts.OutDir}); err != nil {
		return result, err
	} else {
		result.WorkflowPath = packageResult.WorkflowPath
		result.UWSPath = packageResult.UWSPath
		result.PlanJSONPath = packageResult.PlanJSONPath
		result.PlanMDPath = packageResult.PlanMDPath
		result.DiscoveryPath = packageResult.DiscoveryJSONPath
		result.RefinementPath = packageResult.RefinementJSONPath
		result.HandoffPath = packageResult.ReviewHandoffPath
		result.QualityJSONPath = packageResult.QualityJSONPath
		result.QualityMDPath = packageResult.QualityMDPath
		result.QualityPassed = quality != nil && quality.Passed()
	}
	if result.StrictFailed {
		return result, strictFailureError{diagnostics: strictDiagnostics(conversion.diagnostics)}
	}
	return result, nil
}

func IsStrictFailure(err error) bool {
	_, ok := err.(strictFailureError)
	return ok
}

type conversionState struct {
	opts        Options
	doc         tfconfig.Document
	openAPIs    []apiDoc
	bindings    []binding
	symbols     []symbolFact
	selected    []selectedObject
	mappings    []objectMapping
	diagnostics []Diagnostic
}

type apiDoc struct {
	ID          string
	Path        string
	PackagePath string
	Index       apitools.OperationIndex
}

type binding struct {
	Name          string
	Address       string
	ModuleAddress string
	LocalName     string
	Alias         string
	Config        []attributeFact
}

type symbolFact struct {
	Kind          string
	Name          string
	ModuleAddress string
	Value         string
	Sensitive     bool
}

type selectedObject struct {
	Kind          string
	Address       string
	ModuleAddress string
	Type          string
	Name          string
	Provider      string
	Binding       string
	Config        []attributeFact
	Range         *tfconfig.SourceRange
}

type attributeFact struct {
	Path      string
	Value     string
	Sensitive bool
	TodoID    string
}

type objectMapping struct {
	Object      selectedObject
	Purpose     string
	Action      string
	OpenAPIID   string
	OpenAPIPath string
	OperationID string
	TodoID      string
	Ambiguous   bool
	Auth        []apitools.AuthRequirementSummary
}

func normalizeOptions(opts Options) Options {
	if strings.TrimSpace(opts.ConfigDir) == "" {
		opts.ConfigDir = "."
	}
	if strings.TrimSpace(opts.OutDir) == "" {
		opts.OutDir = defaultOutDir
	}
	opts.Action = strings.ToLower(strings.TrimSpace(opts.Action))
	for i := range opts.OpenAPIs {
		opts.OpenAPIs[i].ID = strings.TrimSpace(opts.OpenAPIs[i].ID)
		opts.OpenAPIs[i].Path = strings.TrimSpace(opts.OpenAPIs[i].Path)
	}
	sort.Slice(opts.OpenAPIs, func(i, j int) bool {
		if opts.OpenAPIs[i].ID != opts.OpenAPIs[j].ID {
			return opts.OpenAPIs[i].ID < opts.OpenAPIs[j].ID
		}
		return opts.OpenAPIs[i].Path < opts.OpenAPIs[j].Path
	})
	for i := range opts.Targets {
		opts.Targets[i] = strings.TrimSpace(opts.Targets[i])
	}
	sort.Strings(opts.Targets)
	return opts
}

func (c *conversionState) loadOpenAPIs(ctx context.Context) {
	if len(c.opts.OpenAPIs) == 0 {
		c.addDiagnostic(Diagnostic{
			Code:          "openapi.missing",
			Severity:      "error",
			Message:       "at least one --openapi id=PATH input is required",
			StrictFailure: true,
		})
		return
	}
	seen := map[string]bool{}
	seenPackagePaths := map[string]string{}
	for _, input := range c.opts.OpenAPIs {
		switch {
		case input.ID == "":
			c.addDiagnostic(Diagnostic{Code: "openapi.invalid", Severity: "error", Message: "--openapi ID is required", StrictFailure: true})
			continue
		case input.Path == "":
			c.addDiagnostic(Diagnostic{Code: "openapi.invalid", Severity: "error", Message: fmt.Sprintf("--openapi %s path is required", input.ID), StrictFailure: true})
			continue
		case seen[input.ID]:
			c.addDiagnostic(Diagnostic{Code: "openapi.duplicate_id", Severity: "error", Message: fmt.Sprintf("OpenAPI ID %q is duplicated", input.ID), StrictFailure: true})
			continue
		}
		seen[input.ID] = true
		packagePath := packageOpenAPIPath(input.ID, input.Path)
		if previousID, ok := seenPackagePaths[packagePath]; ok {
			c.addDiagnostic(Diagnostic{
				Code:          "openapi.package_path_collision",
				Severity:      "error",
				Message:       fmt.Sprintf("OpenAPI IDs %q and %q both stage to %q", previousID, input.ID, packagePath),
				StrictFailure: true,
			})
			continue
		}
		seenPackagePaths[packagePath] = input.ID
		inventory, err := apitools.BuildOperationInventory(ctx, apitools.InventoryOptions{
			Documents: []apitools.InventoryDocument{{Name: input.ID, Path: input.Path}},
		})
		if err != nil {
			c.addDiagnostic(Diagnostic{Code: "openapi.load_error", Severity: "error", Message: err.Error(), StrictFailure: true})
			continue
		}
		for _, diag := range inventory.Diagnostics {
			c.addDiagnostic(Diagnostic{
				Code:          "openapi." + strings.ReplaceAll(diag.Code, ".", "_"),
				Severity:      normalizeSeverity(diag.Severity),
				Message:       diag.Message,
				SourceRange:   &SourceRange{Path: diag.Path},
				StrictFailure: diag.Severity == "error",
			})
		}
		index, err := apitools.NewOperationIndex(inventory)
		if err != nil {
			c.addDiagnostic(Diagnostic{Code: "openapi.index_error", Severity: "error", Message: fmt.Sprintf("%s: %v", input.ID, err), StrictFailure: true})
			continue
		}
		c.openAPIs = append(c.openAPIs, apiDoc{ID: input.ID, Path: input.Path, PackagePath: packagePath, Index: index})
	}
}

func (c *conversionState) collectBindings() {
	for _, mod := range c.doc.Modules {
		for _, cfg := range mod.ProviderConfigs {
			b := binding{
				Name:          normalizeBindingName(cfg.Address),
				Address:       cfg.Address,
				ModuleAddress: mod.Address,
				LocalName:     cfg.LocalName,
				Alias:         cfg.Alias,
				Config:        c.attributes(fullAddress(mod.Address, cfg.Address), mod.Address, cfg.Config),
			}
			c.bindings = append(c.bindings, b)
		}
	}
}

func (c *conversionState) collectSymbols() {
	for _, mod := range c.doc.Modules {
		for _, v := range mod.Variables {
			fact := symbolFact{Kind: "variable", Name: v.Name, ModuleAddress: mod.Address, Sensitive: v.Sensitive}
			if v.Default != nil {
				fact.Value = valueText(*v.Default)
				fact.Sensitive = fact.Sensitive || valueSensitive(*v.Default)
				c.maybeSensitiveDiagnostic(fullAddress(mod.Address, "var."+v.Name), mod.Address, "variable default", *v.Default)
			}
			c.symbols = append(c.symbols, fact)
		}
		for _, l := range mod.Locals {
			fact := symbolFact{Kind: "local", Name: l.Name, ModuleAddress: mod.Address}
			if l.Value != nil {
				fact.Value = valueText(*l.Value)
				fact.Sensitive = valueSensitive(*l.Value)
				c.maybeSensitiveDiagnostic(fullAddress(mod.Address, "local."+l.Name), mod.Address, "local value", *l.Value)
			}
			c.symbols = append(c.symbols, fact)
		}
		for _, out := range mod.Outputs {
			fact := symbolFact{Kind: "output", Name: out.Name, ModuleAddress: mod.Address, Sensitive: out.Sensitive}
			if out.Value != nil {
				fact.Value = valueText(*out.Value)
				fact.Sensitive = fact.Sensitive || valueSensitive(*out.Value)
				c.maybeSensitiveDiagnostic(fullAddress(mod.Address, "output."+out.Name), mod.Address, "output value", *out.Value)
			}
			c.symbols = append(c.symbols, fact)
		}
	}
}

func (c *conversionState) selectObjects() {
	targets := map[string]bool{}
	for _, target := range c.opts.Targets {
		if target != "" {
			targets[target] = false
		}
	}
	selectAll := len(targets) == 0
	for _, mod := range c.doc.Modules {
		for _, res := range mod.Resources {
			addr := fullAddress(mod.Address, res.Address)
			if selectAll || targetSelected(targets, addr) {
				obj := selectedObject{
					Kind:          "resource",
					Address:       addr,
					ModuleAddress: mod.Address,
					Type:          res.Type,
					Name:          res.Name,
					Provider:      providerAddress(res.Provider),
					Binding:       normalizeBindingName(providerAddress(res.Provider)),
					Config:        c.attributes(addr, mod.Address, res.Config),
					Range:         res.Range,
				}
				c.selected = append(c.selected, obj)
			}
		}
		for _, ds := range mod.DataSources {
			addr := fullAddress(mod.Address, ds.Address)
			if selectAll || targetSelected(targets, addr) {
				obj := selectedObject{
					Kind:          "data_source",
					Address:       addr,
					ModuleAddress: mod.Address,
					Type:          ds.Type,
					Name:          ds.Name,
					Provider:      providerAddress(ds.Provider),
					Binding:       normalizeBindingName(providerAddress(ds.Provider)),
					Config:        c.attributes(addr, mod.Address, ds.Config),
					Range:         ds.Range,
				}
				c.selected = append(c.selected, obj)
			}
		}
	}
	for target, matched := range targets {
		if !matched {
			c.addDiagnostic(Diagnostic{
				Code:          "target.unmatched",
				Severity:      "error",
				Message:       fmt.Sprintf("target %q did not match a managed resource or data source", target),
				Address:       target,
				StrictFailure: true,
			})
		}
	}
}

func targetSelected(targets map[string]bool, addr string) bool {
	if _, ok := targets[addr]; ok {
		targets[addr] = true
		return true
	}
	return false
}

func (c *conversionState) validateAction() {
	if c.opts.Action != "" && !validAction(c.opts.Action) {
		c.addDiagnostic(Diagnostic{
			Code:          "action.invalid",
			Severity:      "error",
			Message:       fmt.Sprintf("action %q is invalid; use create, update, delete, or replace", c.opts.Action),
			StrictFailure: true,
		})
		return
	}
	for _, obj := range c.selected {
		if obj.Kind == "resource" && c.opts.Action == "" {
			c.addDiagnostic(Diagnostic{
				Code:          "action.required",
				Severity:      "error",
				Message:       "managed resources require --action create, update, delete, or replace",
				Address:       obj.Address,
				ModuleAddress: obj.ModuleAddress,
				SourceRange:   convertRange(obj.Range),
				StrictFailure: true,
			})
		}
	}
}

func (c *conversionState) mapObjects() {
	for _, obj := range c.selected {
		if isProviderLocalDataSource(obj) {
			continue
		}
		switch obj.Kind {
		case "data_source":
			if !c.mapObjectPurpose(obj, "read", "read") {
				c.mapObjectPurpose(obj, "list", "list")
			}
		case "resource":
			if !validAction(c.opts.Action) {
				continue
			}
			if c.opts.Action == "replace" {
				c.mapObjectPurpose(obj, "delete", "replace")
				c.mapObjectPurpose(obj, "create", "replace")
				continue
			}
			c.mapObjectPurpose(obj, c.opts.Action, c.opts.Action)
		}
	}
}

func isProviderLocalDataSource(obj selectedObject) bool {
	if obj.Kind != "data_source" || objectProviderLocalName(obj) != "aws" {
		return false
	}
	switch obj.Type {
	case "aws_iam_policy_document", "aws_partition", "aws_region":
		return true
	default:
		return false
	}
}

func (c *conversionState) mapObjectPurpose(obj selectedObject, purpose, action string) bool {
	candidates := c.operationCandidates()
	provider := objectProviderLocalName(obj)
	if operationID := awsOperationIDForObject(obj, purpose, action); operationID != "" {
		if operation, ok := findOperationByID(candidates, operationID); ok {
			mapping := objectMapping{Object: obj, Purpose: purpose, Action: action}
			doc := openAPIForOperation(c.openAPIs, operation)
			mapping.OpenAPIID = firstNonEmpty(operation.DocumentName, doc.ID)
			mapping.OpenAPIPath = doc.PackagePath
			mapping.OperationID = operation.OperationID
			mapping.Auth = apitools.AuthRequirementsForOperation(provider, operation)
			c.mappings = append(c.mappings, mapping)
			return true
		}
	}
	selection := apitools.SelectOperationByHints(apitools.OperationSelectionHints{
		Provider: provider,
		Purpose:  purpose,
		Target:   strings.Join([]string{obj.Address, obj.Type, obj.Name}, " "),
	}, candidates)
	mapping := objectMapping{Object: obj, Purpose: purpose, Action: action}
	switch {
	case selection.Found:
		doc := openAPIForOperation(c.openAPIs, selection.Operation)
		mapping.OpenAPIID = firstNonEmpty(selection.Operation.DocumentName, doc.ID)
		mapping.OpenAPIPath = doc.PackagePath
		mapping.OperationID = selection.Operation.OperationID
		mapping.Auth = apitools.AuthRequirementsForOperation(provider, selection.Operation)
		c.mappings = append(c.mappings, mapping)
		return true
	case selection.Ambiguous:
		mapping.Ambiguous = true
		mapping.TodoID = todoID(obj.Address, purpose, action)
		mapping.OpenAPIPath = defaultOpenAPIPath(c.openAPIs)
		c.addDiagnostic(Diagnostic{
			Code:          "operation.ambiguous",
			Severity:      "warning",
			Message:       fmt.Sprintf("multiple OpenAPI operations may match %s %s for %s", purpose, obj.Kind, obj.Address),
			Address:       obj.Address,
			ModuleAddress: obj.ModuleAddress,
			SourceRange:   convertRange(obj.Range),
			TodoID:        mapping.TodoID,
			StrictFailure: true,
		})
	default:
		mapping.TodoID = todoID(obj.Address, purpose, action)
		mapping.OpenAPIPath = defaultOpenAPIPath(c.openAPIs)
		c.addDiagnostic(Diagnostic{
			Code:          "operation.unresolved",
			Severity:      "warning",
			Message:       fmt.Sprintf("no confident OpenAPI operation match for %s %s %s", purpose, obj.Kind, obj.Address),
			Address:       obj.Address,
			ModuleAddress: obj.ModuleAddress,
			SourceRange:   convertRange(obj.Range),
			TodoID:        mapping.TodoID,
			StrictFailure: true,
		})
	}
	c.mappings = append(c.mappings, mapping)
	return false
}

func findOperationByID(candidates []apitools.OperationSummary, operationID string) (apitools.OperationSummary, bool) {
	for _, candidate := range candidates {
		if candidate.OperationID == operationID {
			return candidate, true
		}
	}
	return apitools.OperationSummary{}, false
}

func (c *conversionState) operationCandidates() []apitools.OperationSummary {
	var out []apitools.OperationSummary
	for _, doc := range c.openAPIs {
		ops := apitools.SortedOperationSummaries(doc.Index.OperationIDs)
		for _, op := range ops {
			if op.DocumentName == "" {
				op.DocumentName = doc.ID
			}
			out = append(out, op)
		}
	}
	return out
}

func (c *conversionState) attributes(address, moduleAddress string, attrs []tfconfig.Attribute) []attributeFact {
	out := make([]attributeFact, 0, len(attrs))
	for _, attr := range attrs {
		fact := attributeFact{Path: attr.Path, Value: valueText(attr.Value), Sensitive: valueSensitive(attr.Value)}
		if fact.Sensitive {
			fact.TodoID = todoID(address+"."+attr.Path, "redaction", "review")
		}
		c.maybeSensitiveDiagnostic(address, moduleAddress, attr.Path, attr.Value)
		out = append(out, fact)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

func (c *conversionState) maybeSensitiveDiagnostic(address, moduleAddress, path string, value tfconfig.Value) {
	if !valueSensitive(value) {
		return
	}
	reason := "sensitive value"
	if value.SensitiveCandidate != nil {
		reason = value.SensitiveCandidate.Reason
	}
	c.addDiagnostic(Diagnostic{
		Code:          "redaction.review_required",
		Severity:      "warning",
		Message:       fmt.Sprintf("%s at %s is redacted and requires review", reason, path),
		Address:       address,
		ModuleAddress: moduleAddress,
		SourceRange:   convertRange(value.Range),
		TodoID:        todoID(address+"."+path, "redaction", "review"),
		StrictFailure: true,
	})
}

func (c *conversionState) addTFDiagnostics(diags []tfconfig.Diagnostic) {
	for _, diag := range diags {
		c.addDiagnostic(Diagnostic{
			Code:          firstNonEmpty(diag.Code, "tfconfig.diagnostic"),
			Severity:      normalizeSeverity(string(diag.Severity)),
			Message:       diagnosticMessage(diag),
			Address:       diag.Address,
			ModuleAddress: diag.ModuleAddress,
			SourceRange:   convertRange(diag.Range),
			StrictFailure: diag.Severity == tfconfig.DiagnosticError,
		})
	}
}

func (c *conversionState) addDiagnostic(diag Diagnostic) {
	diag.Code = strings.TrimSpace(diag.Code)
	diag.Severity = normalizeSeverity(diag.Severity)
	diag.Message = strings.TrimSpace(diag.Message)
	c.diagnostics = append(c.diagnostics, diag)
}

func (c *conversionState) ensureCredentialBindings() {
	existing := map[string]bool{}
	for _, binding := range c.bindings {
		existing[binding.Name] = true
	}
	for _, mapping := range c.mappings {
		for _, auth := range mapping.Auth {
			name := credentialBindingName(mapping.Object, auth)
			if name == "" || existing[name] {
				continue
			}
			c.bindings = append(c.bindings, binding{
				Name:      name,
				Address:   credentialBindingAddress(mapping.Object, auth),
				LocalName: objectProviderLocalName(mapping.Object),
			})
			existing[name] = true
		}
	}
}

func (c *conversionState) sortAll() {
	sort.Slice(c.bindings, func(i, j int) bool {
		if c.bindings[i].Name != c.bindings[j].Name {
			return c.bindings[i].Name < c.bindings[j].Name
		}
		return c.bindings[i].Address < c.bindings[j].Address
	})
	sort.Slice(c.symbols, func(i, j int) bool {
		return strings.Join([]string{c.symbols[i].ModuleAddress, c.symbols[i].Kind, c.symbols[i].Name}, "\x00") <
			strings.Join([]string{c.symbols[j].ModuleAddress, c.symbols[j].Kind, c.symbols[j].Name}, "\x00")
	})
	sort.Slice(c.selected, func(i, j int) bool { return c.selected[i].Address < c.selected[j].Address })
	sort.Slice(c.mappings, func(i, j int) bool {
		left := []string{c.mappings[i].Object.Address, c.mappings[i].Purpose, c.mappings[i].OpenAPIID, c.mappings[i].OpenAPIPath, c.mappings[i].OperationID, c.mappings[i].TodoID}
		right := []string{c.mappings[j].Object.Address, c.mappings[j].Purpose, c.mappings[j].OpenAPIID, c.mappings[j].OpenAPIPath, c.mappings[j].OperationID, c.mappings[j].TodoID}
		return strings.Join(left, "\x00") < strings.Join(right, "\x00")
	})
	sortDiagnostics(c.diagnostics)
}

func writeArtifacts(result *Result, c conversionState) error {
	if err := validateOpenAPIStagingSafety(result.OutDir, c.openAPIs); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(result.OutDir, "workflows"), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(result.OutDir, "expected"), 0o755); err != nil {
		return err
	}
	if err := resetOpenAPIStagingDir(result.OutDir); err != nil {
		return err
	}
	if err := copyOpenAPIDocuments(result.OutDir, c.openAPIs); err != nil {
		return err
	}
	if err := writeFile(result.ProjectPath, renderProject(c)); err != nil {
		return err
	}
	intentHCL, err := renderIntent(c)
	if err != nil {
		return err
	}
	if err := writeFile(result.IntentPath, intentHCL); err != nil {
		return err
	}
	diagJSON, err := json.MarshalIndent(c.diagnostics, "", "  ")
	if err != nil {
		return err
	}
	diagJSON = append(diagJSON, '\n')
	if err := os.WriteFile(result.DiagnosticsJSON, diagJSON, 0o644); err != nil {
		return err
	}
	if err := writeFile(result.DiagnosticsMD, renderDiagnosticsMarkdown(c.diagnostics)); err != nil {
		return err
	}
	return writeFile(result.ReviewPath, renderReview(c))
}

func writeFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func resetOpenAPIStagingDir(outDir string) error {
	openAPIDir := filepath.Join(outDir, "openapi")
	if err := os.RemoveAll(openAPIDir); err != nil {
		return fmt.Errorf("reset staged OpenAPI directory: %w", err)
	}
	return nil
}

func validateOpenAPIStagingSafety(outDir string, docs []apiDoc) error {
	stagingDir, err := filepath.Abs(filepath.Join(outDir, "openapi"))
	if err != nil {
		return err
	}
	stagingDir = filepath.Clean(stagingDir)
	for _, doc := range docs {
		sourcePath, err := filepath.Abs(doc.Path)
		if err != nil {
			return fmt.Errorf("resolve OpenAPI %s source path: %w", doc.ID, err)
		}
		sourcePath = filepath.Clean(sourcePath)
		if pathWithin(sourcePath, stagingDir) {
			return fmt.Errorf("stage OpenAPI %s: source %s is inside generated OpenAPI staging directory %s; choose an --out directory outside OpenAPI inputs", doc.ID, sourcePath, stagingDir)
		}
	}
	return nil
}

func pathWithin(path, dir string) bool {
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

func copyOpenAPIDocuments(outDir string, docs []apiDoc) error {
	for _, doc := range docs {
		if strings.TrimSpace(doc.PackagePath) == "" {
			continue
		}
		dst := filepath.Join(outDir, filepath.FromSlash(doc.PackagePath))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		if err := copyRegularFile(doc.Path, dst); err != nil {
			return fmt.Errorf("stage OpenAPI %s: %w", doc.ID, err)
		}
	}
	return nil
}

func copyRegularFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	info, err := in.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("source is not a regular file: %s", src)
	}
	tmp := dst + ".tmp"
	out, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return closeErr
	}
	return os.Rename(tmp, dst)
}

func renderProject(c conversionState) string {
	var b strings.Builder
	b.WriteString("# OpenUdon Terraform Conversion Draft\n\n")
	b.WriteString("This package is unapproved review scaffolding generated from static Terraform/OpenTofu facts. It does not execute Terraform, providers, OpenAPI operations, or UWS workflows.\n\n")
	b.WriteString("```openudon-policy\n")
	b.WriteString("runtimes:\n")
	b.WriteString("  openapi: true\n")
	b.WriteString("  http: true\n")
	b.WriteString("  fnct: true\n")
	b.WriteString("  cmd: false\n")
	b.WriteString("  ssh: false\n")
	b.WriteString("```\n\n")
	b.WriteString("## Goal\n\n")
	b.WriteString("Review static Terraform/OpenTofu configuration facts against local OpenAPI operation candidates and produce a normal OpenUdon package candidate for human review.\n\n")
	fmt.Fprintf(&b, "- Config directory: `%s`\n", c.opts.ConfigDir)
	fmt.Fprintf(&b, "- Action: `%s`\n", firstNonEmpty(c.opts.Action, "none"))
	fmt.Fprintf(&b, "- Strict mode: `%t`\n", c.opts.Strict)
	b.WriteString("\n## Inputs\n\n")
	for _, sym := range c.symbols {
		if sym.Kind != "variable" {
			continue
		}
		required := "required"
		if strings.TrimSpace(sym.Value) != "" {
			required = "optional default preserved symbolically"
		}
		sensitive := ""
		if sym.Sensitive {
			sensitive = " sensitive"
		}
		fmt.Fprintf(&b, "- `%s`: string, %s%s Terraform variable.\n", normalizeName(fullAddress(sym.ModuleAddress, sym.Name)), required, sensitive)
	}
	if len(c.symbols) == 0 {
		b.WriteString("- No Terraform variables were selected.\n")
	}
	b.WriteString("\n## Outputs\n\n")
	b.WriteString("- `review_package`: generated OpenUdon package artifacts for review; no operational result is produced by conversion.\n")
	b.WriteString("\n## External Systems and OpenAPI\n\n")
	for _, doc := range c.openAPIs {
		fmt.Fprintf(&b, "- `%s`: source `%s`, staged package path `%s`.\n", doc.ID, doc.Path, doc.PackagePath)
	}
	if len(c.openAPIs) == 0 {
		b.WriteString("- none loaded\n")
	}
	b.WriteString("\n## Runtime Policy\n\n")
	b.WriteString("- Only `openapi`, `http`, and `fnct` runtime artifacts are allowed in generated package output.\n")
	b.WriteString("- `cmd` and `ssh` are not allowed by this conversion package.\n")
	b.WriteString("\n## Data Flow\n\n")
	for _, obj := range c.selected {
		if isProviderLocalDataSource(obj) {
			fmt.Fprintf(&b, "- Terraform `%s` `%s` is provider-local metadata preserved symbolically; no OpenAPI operation is generated.\n", obj.Kind, obj.Address)
			for _, attr := range obj.Config {
				fmt.Fprintf(&b, "- `%s.%s`: symbolic Terraform expression `%s`.\n", obj.Address, attr.Path, attr.Value)
			}
			continue
		}
		fmt.Fprintf(&b, "- Terraform `%s` `%s` maps to a symbolic OpenUdon review step using provider binding `%s`.\n", obj.Kind, obj.Address, firstNonEmpty(obj.Binding, "default"))
		for _, attr := range obj.Config {
			if attr.Sensitive {
				fmt.Fprintf(&b, "- `%s.%s`: sensitive symbolic value, review TODO `%s`.\n", obj.Address, attr.Path, attr.TodoID)
				continue
			}
			fmt.Fprintf(&b, "- `%s.%s`: symbolic Terraform expression `%s`.\n", obj.Address, attr.Path, attr.Value)
		}
	}
	if len(c.selected) == 0 {
		b.WriteString("- none\n")
	}
	b.WriteString("\n## Function Contracts\n\n")
	b.WriteString("- No custom function adapters are generated by Terraform conversion.\n")
	b.WriteString("\n## Credentials and Secrets\n\n")
	if len(c.bindings) == 0 {
		b.WriteString("- No provider credential bindings were declared in the selected Terraform configuration.\n")
	} else {
		for _, binding := range c.bindings {
			fmt.Fprintf(&b, "- `%s`: symbolic provider credential binding for `%s`; credential values must be supplied outside generated artifacts.\n", binding.Name, binding.Address)
		}
	}
	b.WriteString("- Sensitive or secret-like Terraform values are redacted into symbolic review inputs and must not appear as literals in generated artifacts.\n")
	b.WriteString("\n## Safety and Approval Boundary\n\n")
	b.WriteString("- Generated artifacts are unapproved by default and require human review before trusted-runner handoff.\n")
	b.WriteString("- Side-effectful OpenAPI operations require review, sandbox proof-run approval, and trusted-runtime approval before production execution.\n")
	b.WriteString("- Direct production execution is not performed by conversion or synthesis.\n")
	b.WriteString("\n## Fallback Behavior\n\n")
	b.WriteString("- Unmatched Terraform targets, missing OpenAPI inputs, ambiguous operation matches, unresolved operation TODOs, and sensitive redaction TODOs remain diagnostics.\n")
	b.WriteString("- Strict mode fails when strict-failure diagnostics remain.\n")
	b.WriteString("- Normal package quality fails unresolved conversion diagnostics so unsafe assumptions are visible to reviewers.\n")
	b.WriteString("\n## Diagnostics\n\n")
	if len(c.diagnostics) == 0 {
		b.WriteString("- none\n")
	} else {
		for _, diag := range c.diagnostics {
			fmt.Fprintf(&b, "- `%s` %s: %s\n", diag.Code, diag.Severity, diag.Message)
		}
	}
	return b.String()
}

func renderIntent(c conversionState) (string, error) {
	intent := &workflowintent.Intent{
		OpenAPI: defaultOpenAPIPath(c.openAPIs),
		Workflow: &workflowintent.WorkflowMeta{
			Name:        "terraform_conversion_draft",
			Description: "Draft review scaffold generated from static Terraform/OpenTofu configuration.",
		},
		Locals: map[string]string{},
	}
	for _, sym := range c.symbols {
		if sym.Kind != "variable" {
			intent.Locals[normalizeName(sym.ModuleAddress+"_"+sym.Kind+"_"+sym.Name)] = sym.Value
			continue
		}
		input := &workflowintent.Input{
			Name:        normalizeName(fullAddress(sym.ModuleAddress, sym.Name)),
			Type:        "string",
			Description: fmt.Sprintf("Symbolic Terraform variable %s", fullAddress(sym.ModuleAddress, sym.Name)),
			Required:    true,
			Sensitive:   sym.Sensitive,
		}
		if !sym.Sensitive && safeDefault(sym.Value) {
			input.Default = sym.Value
		}
		intent.Inputs = append(intent.Inputs, input)
	}
	for _, binding := range c.bindings {
		intent.Security = append(intent.Security, &workflowintent.SecurityIntent{
			Name:        binding.Name,
			Description: fmt.Sprintf("Symbolic provider binding for %s", binding.Address),
			TokenFrom:   "review:" + binding.Name,
		})
	}
	for _, mapping := range c.mappings {
		step := &workflowintent.Step{
			Name:      normalizeName(mapping.Object.Address + "_" + mapping.Purpose),
			Type:      "openapi",
			Do:        fmt.Sprintf("Review %s %s for Terraform %s %s", mapping.Purpose, mapping.Action, mapping.Object.Kind, mapping.Object.Address),
			Provider:  mapping.Object.Binding,
			OpenAPI:   mapping.OpenAPIPath,
			Operation: mapping.OperationID,
		}
		if step.Operation == "" {
			step.Operation = mapping.TodoID
		}
		for _, attr := range mapping.Object.Config {
			if strings.TrimSpace(attr.Path) == "" {
				continue
			}
			localName := normalizeName(mapping.Object.Address + "_" + mapping.Purpose + "_" + attr.Path)
			intent.Locals[localName] = attr.Value
			if step.With == nil {
				step.With = map[string]string{}
			}
			step.With[terraformAttributeReviewKey(attr.Path)] = localName
			for _, requestKey := range terraformOpenAPIRequestKeys(mapping, attr.Path) {
				step.With[requestKey] = localName
			}
		}
		for requestKey, value := range awsQueryProtocolStaticBindings(mapping) {
			if step.With == nil {
				step.With = map[string]string{}
			}
			if strings.TrimSpace(step.With[requestKey]) == "" {
				step.With[requestKey] = value
			}
		}
		for _, auth := range mapping.Auth {
			bindingName := credentialBindingName(mapping.Object, auth)
			if bindingName == "" {
				continue
			}
			if step.With == nil {
				step.With = map[string]string{}
			}
			for _, requestKey := range credentialRequestKeys(auth) {
				if strings.TrimSpace(step.With[requestKey]) == "" {
					step.With[requestKey] = bindingName
				}
			}
		}
		intent.Steps = append(intent.Steps, step)
	}
	if len(intent.Locals) == 0 {
		intent.Locals = nil
	}
	sort.Slice(intent.Inputs, func(i, j int) bool { return intent.Inputs[i].Name < intent.Inputs[j].Name })
	sort.Slice(intent.Security, func(i, j int) bool { return intent.Security[i].Name < intent.Security[j].Name })
	return workflowintent.RenderIntentHCL(intent)
}

func renderDiagnosticsMarkdown(diags []Diagnostic) string {
	var b strings.Builder
	b.WriteString("# Terraform Conversion Diagnostics\n\n")
	if len(diags) == 0 {
		b.WriteString("No diagnostics.\n")
		return b.String()
	}
	for _, diag := range diags {
		fmt.Fprintf(&b, "## %s\n\n", diag.Code)
		fmt.Fprintf(&b, "- Severity: `%s`\n", diag.Severity)
		if diag.Address != "" {
			fmt.Fprintf(&b, "- Address: `%s`\n", diag.Address)
		}
		if diag.ModuleAddress != "" {
			fmt.Fprintf(&b, "- Module: `%s`\n", diag.ModuleAddress)
		}
		if diag.TodoID != "" {
			fmt.Fprintf(&b, "- TODO: `%s`\n", diag.TodoID)
		}
		fmt.Fprintf(&b, "- Strict failure: `%t`\n", diag.StrictFailure)
		fmt.Fprintf(&b, "\n%s\n\n", diag.Message)
	}
	return b.String()
}

func renderReview(c conversionState) string {
	var b strings.Builder
	b.WriteString("# Terraform Conversion Review\n\n")
	b.WriteString("Generated artifacts are draft review scaffolding and are not approved for trusted execution.\n\n")
	b.WriteString("## Operation Mappings\n\n")
	if len(c.mappings) == 0 {
		b.WriteString("- none\n")
	}
	for _, mapping := range c.mappings {
		ref := mapping.TodoID
		if mapping.OperationID != "" {
			ref = mapping.OpenAPIPath + ":" + mapping.OperationID
		}
		fmt.Fprintf(&b, "- `%s` %s/%s -> `%s`\n", mapping.Object.Address, mapping.Action, mapping.Purpose, ref)
		for _, auth := range mapping.Auth {
			fmt.Fprintf(&b, "  - Auth `%s`: %s\n", auth.Scheme, auth.Description)
		}
	}
	b.WriteString("\n## Provider Bindings\n\n")
	if len(c.bindings) == 0 {
		b.WriteString("- none\n")
	}
	for _, binding := range c.bindings {
		fmt.Fprintf(&b, "- `%s` from `%s`\n", binding.Name, binding.Address)
	}
	b.WriteString("\n## Symbolic Facts\n\n")
	for _, sym := range c.symbols {
		fmt.Fprintf(&b, "- %s `%s` = `%s`\n", sym.Kind, fullAddress(sym.ModuleAddress, sym.Name), sym.Value)
	}
	b.WriteString("\n## Redaction\n\n")
	wrote := false
	for _, diag := range c.diagnostics {
		if strings.HasPrefix(diag.Code, "redaction.") {
			fmt.Fprintf(&b, "- `%s`: %s\n", diag.TodoID, diag.Message)
			wrote = true
		}
	}
	if !wrote {
		b.WriteString("- none\n")
	}
	return b.String()
}

func fullAddress(moduleAddress, objectAddress string) string {
	moduleAddress = strings.TrimSpace(moduleAddress)
	objectAddress = strings.TrimSpace(objectAddress)
	if moduleAddress == "" {
		return objectAddress
	}
	if objectAddress == "" {
		return moduleAddress
	}
	return moduleAddress + "." + objectAddress
}

func providerAddress(ref *tfconfig.ProviderRef) string {
	if ref == nil {
		return ""
	}
	return ref.Address
}

func providerLocalName(address string) string {
	address = strings.TrimPrefix(strings.TrimSpace(address), "provider.")
	if head, _, ok := strings.Cut(address, "."); ok {
		return head
	}
	return address
}

func objectProviderLocalName(obj selectedObject) string {
	if provider := providerLocalName(obj.Provider); provider != "" {
		return provider
	}
	if provider, _, ok := strings.Cut(strings.TrimSpace(obj.Type), "_"); ok {
		return provider
	}
	return ""
}

func awsOperationIDForObject(obj selectedObject, purpose, action string) string {
	if objectProviderLocalName(obj) != "aws" {
		return ""
	}
	purpose = strings.ToLower(strings.TrimSpace(purpose))
	action = strings.ToLower(strings.TrimSpace(action))
	switch obj.Type {
	case "aws_s3_bucket":
		if obj.Kind == "resource" && purpose == "create" && (action == "create" || action == "replace") {
			return "CreateBucket"
		}
		if obj.Kind == "data_source" && purpose == "read" {
			return "GetBucketLocation"
		}
		if obj.Kind == "data_source" && purpose == "list" {
			return "ListBuckets"
		}
	case "aws_s3_bucket_accelerate_configuration":
		if obj.Kind == "resource" && purpose == "create" && (action == "create" || action == "replace") {
			return "PutBucketAccelerateConfiguration"
		}
	case "aws_caller_identity":
		if obj.Kind == "data_source" && purpose == "read" {
			return "POST_GetCallerIdentity"
		}
	case "aws_iam_role":
		if obj.Kind == "resource" && purpose == "create" && (action == "create" || action == "replace") {
			return "POST_CreateRole"
		}
		if obj.Kind == "resource" && purpose == "delete" {
			return "POST_DeleteRole"
		}
	case "aws_iam_role_policy":
		if obj.Kind == "resource" && (purpose == "create" || purpose == "update") && (action == "create" || action == "update" || action == "replace") {
			return "POST_PutRolePolicy"
		}
		if obj.Kind == "resource" && purpose == "delete" {
			return "POST_DeleteRolePolicy"
		}
	case "aws_lambda_function":
		if obj.Kind == "resource" && purpose == "create" && (action == "create" || action == "replace") {
			return "CreateFunction"
		}
		if obj.Kind == "resource" && purpose == "delete" {
			return "DeleteFunction"
		}
	case "aws_lambda_function_url":
		if obj.Kind == "resource" && purpose == "create" && (action == "create" || action == "replace") {
			return "CreateFunctionUrlConfig"
		}
		if obj.Kind == "resource" && purpose == "update" {
			return "UpdateFunctionUrlConfig"
		}
		if obj.Kind == "resource" && purpose == "delete" {
			return "DeleteFunctionUrlConfig"
		}
	}
	return ""
}

func awsQueryProtocolStaticBindings(mapping objectMapping) map[string]string {
	if objectProviderLocalName(mapping.Object) != "aws" {
		return nil
	}
	action := awsQueryProtocolAction(mapping.OperationID)
	version := awsQueryProtocolVersion(mapping.OpenAPIID, mapping.OpenAPIPath)
	if action == "" || version == "" {
		return nil
	}
	return map[string]string{
		"Action":  action,
		"Version": version,
	}
}

func awsQueryProtocolAction(operationID string) string {
	operationID = strings.TrimSpace(operationID)
	for _, prefix := range []string{"GET_", "POST_"} {
		if strings.HasPrefix(operationID, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(operationID, prefix))
		}
	}
	return ""
}

func awsQueryProtocolVersion(openAPIID, openAPIPath string) string {
	normalized := strings.ToLower(strings.Join([]string{openAPIID, openAPIPath}, " "))
	switch {
	case strings.Contains(normalized, "iam"):
		return "2010-05-08"
	case strings.Contains(normalized, "sts"):
		return "2011-06-15"
	default:
		return ""
	}
}

func terraformOpenAPIRequestKeys(mapping objectMapping, attrPath string) []string {
	attrPath = strings.TrimSpace(attrPath)
	if attrPath == "" {
		return nil
	}
	if objectProviderLocalName(mapping.Object) != "aws" {
		return nil
	}
	switch mapping.OperationID {
	case "CreateFunctionUrlConfig", "UpdateFunctionUrlConfig", "DeleteFunctionUrlConfig":
		if attrPath == "function_name" {
			return []string{"FunctionName"}
		}
	}
	return nil
}

func credentialBindingName(obj selectedObject, auth apitools.AuthRequirementSummary) string {
	if auth.Kind != "aws_signature" {
		return ""
	}
	provider := credentialProviderName(obj)
	scheme := firstNonEmpty(auth.Scheme, "sigv4")
	return normalizeName(provider + "_" + scheme)
}

func credentialBindingAddress(obj selectedObject, auth apitools.AuthRequirementSummary) string {
	provider := strings.TrimPrefix(firstNonEmpty(strings.TrimSpace(obj.Provider), "provider."+credentialProviderName(obj)), "provider.")
	scheme := firstNonEmpty(auth.Scheme, "sigv4")
	return "provider." + provider + "." + scheme
}

func credentialProviderName(obj selectedObject) string {
	if binding := strings.TrimSpace(obj.Binding); binding != "" && binding != "default" {
		return binding
	}
	return firstNonEmpty(objectProviderLocalName(obj), "aws")
}

func credentialRequestKeys(auth apitools.AuthRequirementSummary) []string {
	if auth.Kind != "aws_signature" {
		return nil
	}
	return []string{firstNonEmpty(auth.ParameterName, auth.Scheme, "Authorization"), "Authorization"}
}

func normalizeBindingName(address string) string {
	address = strings.TrimPrefix(strings.TrimSpace(address), "provider.")
	return normalizeName(address)
}

var invalidNameChars = regexp.MustCompile(`[^A-Za-z0-9_]+`)

func normalizeName(value string) string {
	value = strings.Trim(value, ".")
	value = strings.ReplaceAll(value, ".", "_")
	value = invalidNameChars.ReplaceAllString(value, "_")
	value = strings.Trim(value, "_")
	if value == "" {
		return "default"
	}
	if value[0] >= '0' && value[0] <= '9' {
		value = "n_" + value
	}
	return strings.ToLower(value)
}

func valueText(value tfconfig.Value) string {
	if valueSensitive(value) {
		return "${sensitive." + firstNonEmpty(sensitiveCandidatePath(value), "value") + "}"
	}
	if value.Expression != "" {
		return value.Expression
	}
	if value.UnknownReason != "" {
		return "${unknown:" + value.UnknownReason + "}"
	}
	if value.Literal != nil {
		data, err := json.Marshal(value.Literal)
		if err == nil {
			return string(data)
		}
	}
	return string(value.Kind)
}

func valueSensitive(value tfconfig.Value) bool {
	return value.Sensitive || value.Redacted || value.SensitiveCandidate != nil
}

func safeDefault(value string) bool {
	return value != "" && !strings.HasPrefix(value, "${") && len(value) < 160
}

func sensitiveCandidatePath(value tfconfig.Value) string {
	if value.SensitiveCandidate != nil && value.SensitiveCandidate.AttributePath != "" {
		return normalizeName(value.SensitiveCandidate.AttributePath)
	}
	return ""
}

func todoID(address, purpose, action string) string {
	return "todo." + normalizeName(address) + "." + normalizeName(purpose) + "." + normalizeName(action)
}

func terraformAttributeReviewKey(path string) string {
	path = strings.Trim(strings.TrimSpace(path), ".")
	if path == "" {
		return ""
	}
	return "body.terraform." + path
}

func validAction(action string) bool {
	switch action {
	case "create", "update", "delete", "replace":
		return true
	default:
		return false
	}
}

func normalizeSeverity(severity string) string {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "error", "warning", "info":
		return strings.ToLower(strings.TrimSpace(severity))
	case "warn":
		return "warning"
	default:
		return "warning"
	}
}

func diagnosticMessage(diag tfconfig.Diagnostic) string {
	if diag.Detail == "" {
		return diag.Summary
	}
	return diag.Summary + ": " + diag.Detail
}

func convertRange(rng *tfconfig.SourceRange) *SourceRange {
	if rng == nil {
		return nil
	}
	return &SourceRange{
		SourceID: rng.SourceID,
		Path:     rng.Path,
		Start:    Position{Line: rng.Start.Line, Column: rng.Start.Column, Byte: rng.Start.Byte},
		End:      Position{Line: rng.End.Line, Column: rng.End.Column, Byte: rng.End.Byte},
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func packageOpenAPIPath(id, sourcePath string) string {
	ext := strings.ToLower(filepath.Ext(sourcePath))
	switch ext {
	case ".json", ".yaml", ".yml":
	default:
		ext = ".yaml"
	}
	return filepath.ToSlash(filepath.Join("openapi", normalizeName(id)+ext))
}

func defaultOpenAPIPath(docs []apiDoc) string {
	if len(docs) == 0 {
		return ""
	}
	return docs[0].PackagePath
}

func openAPIForOperation(docs []apiDoc, operation apitools.OperationSummary) apiDoc {
	for _, doc := range docs {
		if operation.DocumentName != "" && doc.ID == operation.DocumentName {
			return doc
		}
		if operation.DocumentPath != "" && doc.Path == operation.DocumentPath {
			return doc
		}
	}
	return apiDoc{}
}

func sortDiagnostics(diags []Diagnostic) {
	sort.Slice(diags, func(i, j int) bool {
		left := []string{diags[i].Code, diags[i].Address, diags[i].ModuleAddress, diags[i].TodoID, diags[i].Message}
		right := []string{diags[j].Code, diags[j].Address, diags[j].ModuleAddress, diags[j].TodoID, diags[j].Message}
		return strings.Join(left, "\x00") < strings.Join(right, "\x00")
	})
}

func hasStrictFailure(diags []Diagnostic) bool {
	for _, diag := range diags {
		if diag.StrictFailure {
			return true
		}
	}
	return false
}

func strictDiagnostics(diags []Diagnostic) []Diagnostic {
	var out []Diagnostic
	for _, diag := range diags {
		if diag.StrictFailure {
			out = append(out, diag)
		}
	}
	return out
}
