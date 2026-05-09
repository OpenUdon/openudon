package tfconvert

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/OpenUdon/apitools"
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
	Diagnostics     []Diagnostic
	StrictFailed    bool
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
	conversion.sortAll()

	result := &Result{
		OutDir:          opts.OutDir,
		ProjectPath:     filepath.Join(opts.OutDir, "project.md"),
		IntentPath:      filepath.Join(opts.OutDir, workflowintent.IntentPath),
		DiagnosticsJSON: filepath.Join(opts.OutDir, "expected", "diagnostics.json"),
		DiagnosticsMD:   filepath.Join(opts.OutDir, "expected", "diagnostics.md"),
		ReviewPath:      filepath.Join(opts.OutDir, "expected", "review.md"),
		Diagnostics:     conversion.diagnostics,
		StrictFailed:    opts.Strict && hasStrictFailure(conversion.diagnostics),
	}

	if err := writeArtifacts(result, conversion); err != nil {
		return result, err
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
	ID    string
	Path  string
	Index apitools.OperationIndex
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
		c.openAPIs = append(c.openAPIs, apiDoc{ID: input.ID, Path: input.Path, Index: index})
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

func (c *conversionState) mapObjectPurpose(obj selectedObject, purpose, action string) bool {
	candidates := c.operationCandidates()
	selection := apitools.SelectOperationByHints(apitools.OperationSelectionHints{
		Provider: providerLocalName(obj.Provider),
		Purpose:  purpose,
		Target:   strings.Join([]string{obj.Address, obj.Type, obj.Name}, " "),
	}, candidates)
	mapping := objectMapping{Object: obj, Purpose: purpose, Action: action}
	switch {
	case selection.Found:
		mapping.OpenAPIID = firstNonEmpty(selection.Operation.DocumentName, openAPIIDForPath(c.openAPIs, selection.Operation.DocumentPath))
		mapping.OperationID = selection.Operation.OperationID
		mapping.Auth = apitools.AuthRequirementsForOperation(providerLocalName(obj.Provider), selection.Operation)
		c.mappings = append(c.mappings, mapping)
		return true
	case selection.Ambiguous:
		mapping.Ambiguous = true
		mapping.TodoID = todoID(obj.Address, purpose, action)
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
		left := []string{c.mappings[i].Object.Address, c.mappings[i].Purpose, c.mappings[i].OpenAPIID, c.mappings[i].OperationID, c.mappings[i].TodoID}
		right := []string{c.mappings[j].Object.Address, c.mappings[j].Purpose, c.mappings[j].OpenAPIID, c.mappings[j].OperationID, c.mappings[j].TodoID}
		return strings.Join(left, "\x00") < strings.Join(right, "\x00")
	})
	sortDiagnostics(c.diagnostics)
}

func writeArtifacts(result *Result, c conversionState) error {
	if err := os.MkdirAll(filepath.Join(result.OutDir, "workflows"), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(result.OutDir, "expected"), 0o755); err != nil {
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

func renderProject(c conversionState) string {
	var b strings.Builder
	b.WriteString("# OpenUdon Terraform Conversion Draft\n\n")
	b.WriteString("This package is unapproved review scaffolding generated from static Terraform/OpenTofu facts. It does not execute Terraform, providers, OpenAPI operations, or UWS workflows.\n\n")
	fmt.Fprintf(&b, "- Config directory: `%s`\n", c.opts.ConfigDir)
	fmt.Fprintf(&b, "- Action: `%s`\n", firstNonEmpty(c.opts.Action, "none"))
	fmt.Fprintf(&b, "- Strict mode: `%t`\n", c.opts.Strict)
	b.WriteString("\n## OpenAPI Inputs\n\n")
	for _, doc := range c.openAPIs {
		fmt.Fprintf(&b, "- `%s`: `%s`\n", doc.ID, doc.Path)
	}
	if len(c.openAPIs) == 0 {
		b.WriteString("- none loaded\n")
	}
	b.WriteString("\n## Selected Objects\n\n")
	for _, obj := range c.selected {
		fmt.Fprintf(&b, "- `%s` %s provider `%s`\n", obj.Address, obj.Kind, firstNonEmpty(obj.Binding, "default"))
	}
	if len(c.selected) == 0 {
		b.WriteString("- none\n")
	}
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
	if len(intent.Locals) == 0 {
		intent.Locals = nil
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
			OpenAPI:   mapping.OpenAPIID,
			Operation: mapping.OperationID,
			With:      map[string]string{},
		}
		if step.Operation == "" {
			step.Operation = mapping.TodoID
		}
		for _, attr := range mapping.Object.Config {
			step.With[attr.Path] = attr.Value
		}
		intent.Steps = append(intent.Steps, step)
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
			ref = mapping.OpenAPIID + ":" + mapping.OperationID
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

func openAPIIDForPath(docs []apiDoc, path string) string {
	for _, doc := range docs {
		if doc.Path == path {
			return doc.ID
		}
	}
	return ""
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
