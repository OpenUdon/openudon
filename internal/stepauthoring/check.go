package stepauthoring

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/OpenUdon/apitools"
	"github.com/OpenUdon/openudon/internal/authoring"
	"github.com/OpenUdon/openudon/internal/evidencefile"
	"github.com/OpenUdon/openudon/internal/packageartifacts"
	"github.com/OpenUdon/openudon/internal/workflowintent"
	"github.com/OpenUdon/uws/uws1"
)

const (
	WireVersion       = "openudon.step-authoring.v1"
	CheckCommand      = "step.check"
	MaxRequestBytes   = 256 << 10
	MaxSourceBytes    = 8 << 20
	MaxSourceOps      = 1000
	MaxSchemaNodes    = 2048
	MaxSchemaDepth    = 32
	MaxChecks         = 64
	MaxUnresolved     = 32
	MaxResultBytes    = 256 << 10
	defaultIntentPath = "workflows/intent.hcl"
)

var (
	sha256Pattern       = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
	sourceIDPattern     = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)
	stepOutputReference = regexp.MustCompile(`\b([A-Za-z][A-Za-z0-9_-]*)\.received_body(?:\.([A-Za-z][A-Za-z0-9_.-]*))?`)
)

// OperationRef binds a selected source operation to its exact local bytes.
type OperationRef struct {
	SourceKind     string `json:"source_kind"`
	SourceID       string `json:"source_id"`
	SourceSHA256   string `json:"source_sha256"`
	NativeSelector string `json:"native_selector"`
	OperationKey   string `json:"operation_key"`
	OperationID    string `json:"operation_id,omitempty"`
}

type StepContract struct {
	ID                     string            `json:"id"`
	Purpose                string            `json:"purpose"`
	Inputs                 *uws1.ParamSchema `json:"inputs"`
	Outputs                *uws1.ParamSchema `json:"outputs"`
	Effect                 string            `json:"effect"`
	AccountConstraints     []string          `json:"account_constraints,omitempty"`
	DestinationConstraints []string          `json:"destination_constraints,omitempty"`
}

type CheckRequest struct {
	Version        string            `json:"version"`
	Kind           string            `json:"kind"`
	Command        string            `json:"command"`
	StepID         string            `json:"step_id"`
	Contract       StepContract      `json:"contract"`
	OperationRef   OperationRef      `json:"operation_ref"`
	IntentSHA256   string            `json:"intent_sha256"`
	OutputMappings map[string]string `json:"output_mappings,omitempty"`
}

type CheckItem struct {
	Code    string `json:"code"`
	Status  string `json:"status"`
	Message string `json:"message"`
	Pointer string `json:"pointer,omitempty"`
}

type Diagnostic struct {
	Code        string `json:"code"`
	Severity    string `json:"severity"`
	Message     string `json:"message"`
	Remediation string `json:"remediation,omitempty"`
}

type CheckResult struct {
	StepID              string       `json:"step_id"`
	IntentSHA256        string       `json:"intent_sha256"`
	OperationRef        OperationRef `json:"operation_ref"`
	Assessment          string       `json:"assessment"`
	Checks              []CheckItem  `json:"checks"`
	UnresolvedQuestions []string     `json:"unresolved_questions"`
}

type Result struct {
	Version     string       `json:"version"`
	Kind        string       `json:"kind"`
	Command     string       `json:"command"`
	Status      string       `json:"status"`
	Diagnostics []Diagnostic `json:"diagnostics"`
	Result      *CheckResult `json:"result,omitempty"`
}

type Outcome struct {
	Result   Result
	ExitCode int
}

func FailedResult(code, message string, exitCode int) Outcome {
	return Outcome{
		Result: Result{
			Version: WireVersion, Kind: "result", Command: CheckCommand,
			Status: "failed", Diagnostics: []Diagnostic{{Code: code, Severity: "error", Message: message}},
		},
		ExitCode: exitCode,
	}
}

// Check performs the read-only step.check protocol. It never returns source
// prose, request values, or local paths in its result.
func Check(ctx context.Context, exampleDir string, request CheckRequest) Outcome {
	if request.Version != "" && request.Version != WireVersion {
		return FailedResult("request.unsupported_version", "The step-check request version is not supported.", 2)
	}
	if err := validateRequest(request); err != nil {
		return FailedResult("request.invalid", "The step-check request is invalid.", 2)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return blocked("request.cancelled", "The step check was cancelled before it started.")
	}
	root, err := resolveExampleRoot(exampleDir)
	if err != nil {
		return blocked("example.invalid", "The selected example is unavailable or unsafe.")
	}
	intentBytes, err := readWithin(root, defaultIntentPath, evidencefile.DefaultMaxBytes)
	if err != nil {
		return blocked("intent.invalid", "The workflow intent is unavailable or unsafe to read.")
	}
	intentDigest := "sha256:" + evidencefile.SHA256(intentBytes)
	if request.IntentSHA256 != intentDigest {
		return conflict("intent.stale", "The workflow intent changed after selection.")
	}
	intent, err := workflowintent.ParseIntent(intentBytes, defaultIntentPath)
	if err != nil {
		return blocked("intent.invalid", "The workflow intent could not be validated.")
	}
	steps := findSteps(intent.Steps, request.StepID)
	if len(steps) != 1 {
		code, message := "step.not_found", "The selected step does not exist exactly once."
		if len(steps) > 1 {
			code, message = "step.ambiguous", "The selected step name is ambiguous in the workflow."
		}
		check := CheckItem{Code: code, Status: "fail", Message: message}
		return completedCheck(request, intentDigest, []CheckItem{check}, nil)
	}
	step := steps[0]
	check := newCheckAccumulator()
	check.add("intent.digest_match", "pass", "The expected intent digest matches the current bytes.")

	relSource, sourceState := selectedSource(intent, step)
	if sourceState != "ok" {
		check.add("source.selection", "fail", "The selected step does not resolve to one local API source.")
		return completedCheck(request, intentDigest, check.items, unresolvedForContract(request.Contract, true, true))
	}
	if packageartifacts.IsAdvisorySecuritySidecarPath(relSource) {
		check.add("source.selection", "fail", "The selected step references a security sidecar rather than an API source document.")
		return completedCheck(request, intentDigest, check.items, unresolvedForContract(request.Contract, true, true))
	}
	sourceBytes, err := readWithin(root, relSource, MaxSourceBytes)
	if err != nil {
		return blocked("source.unavailable", "The selected source is unavailable or unsafe to read.")
	}
	gotSourceDigest := "sha256:" + evidencefile.SHA256(sourceBytes)
	if gotSourceDigest != request.OperationRef.SourceSHA256 {
		return conflict("source.digest_mismatch", "The selected API source changed after operation selection.")
	}
	check.add("source.digest_match", "pass", "The selected source digest matches the current bytes.")
	if sourceIDForPath(relSource) != request.OperationRef.SourceID {
		check.add("source.identity_match", "fail", "The selected source identity does not match the step source.")
	}

	candidate, candidateErr := resolveOperationCandidate(ctx, relSource, sourceBytes, request.OperationRef, request.Contract)
	if err := ctx.Err(); err != nil {
		return blocked("request.cancelled", "The source check was cancelled before operation inspection completed.")
	}
	if errors.Is(candidateErr, errOperationMissing) || errors.Is(candidateErr, errOperationAmbiguous) {
		check.add("operation.exact_match", "fail", "The selected operation reference does not resolve in the current source.")
		return completedCheck(request, intentDigest, check.items, unresolvedForContract(request.Contract, true, true))
	}
	if candidateErr != nil {
		return blocked("source.unsupported", "The selected source could not be inspected safely.")
	}
	op := candidate.Operation
	check.add("operation.exact_match", "pass", "The selected operation identity and native selector match one source operation.")
	if !stepOperationMatches(step.Operation, op, request.OperationRef) {
		check.add("operation.intent_binding", "fail", "The selected step is not bound to the requested source operation.")
	} else {
		check.add("operation.intent_binding", "pass", "The selected step names the requested source operation.")
	}
	if !validateInputReferenceValues(intent, stepInputMappingValues(step)) {
		check.add("mapping.workflow_inputs", "fail", "A step mapping refers to an undeclared workflow input.")
	}

	checkRequiredMappings(step, request.Contract, op, check)
	_, unsupportedInputs, unsupportedOutputs := mapStepContract(request.Contract)
	inputStatus, inputPointer := checkRequestMappingCompatibility(step, request.Contract, op, !unsupportedInputs && operationCapability(candidate, "inputs") == apitools.OperationCapabilitySupported)
	if inputStatus == "indeterminate" {
		if pointer := rootContractExtensionPointer("inputs", request.Contract.Inputs); pointer != "" {
			inputPointer = pointer
		}
	}
	checkMappedWorkflowValues(intent, step, request.Contract, op, check)
	checkInlineCredentialMappings(step, check)
	outputStatus, outputPointer := checkOutputs(intent, step, request.Contract, op, request.OutputMappings, !unsupportedOutputs && operationCapability(candidate, "outputs") == apitools.OperationCapabilitySupported, check)
	if outputStatus == "indeterminate" {
		if pointer := rootContractExtensionPointer("outputs", request.Contract.Outputs); pointer != "" {
			outputPointer = pointer
		}
	}
	checkAuthentication(intent, step, op, check)
	checkDependencies(intent, step, request.StepID, check)
	if dependencyCycleFrom(intent.Steps, request.StepID) {
		check.add("dependency.cycle", "fail", "The selected step participates in a workflow dependency cycle or self-reference.")
	}
	addMappingCompatibilityCheck(check, "mapping.input_contract", "input", inputStatus, inputPointer)
	addMappingCompatibilityCheck(check, "mapping.output_contract", "output", outputStatus, outputPointer)
	effectStatus := checkEffect(request.Contract, candidate, true, check)

	return completedCheck(request, intentDigest, check.items, unresolvedForContract(request.Contract, false, effectStatus == "indeterminate"))
}

func validateRequest(request CheckRequest) error {
	if request.Version != WireVersion || request.Kind != "request" || request.Command != CheckCommand {
		return fmt.Errorf("wire envelope is invalid")
	}
	if !symbol(request.StepID) || !symbol(request.Contract.ID) {
		return fmt.Errorf("step identity is invalid")
	}
	if strings.TrimSpace(request.Contract.Purpose) == "" || len(request.Contract.Purpose) > 2048 {
		return fmt.Errorf("contract purpose is invalid")
	}
	if request.Contract.Effect != "read" && request.Contract.Effect != "write" && request.Contract.Effect != "unknown" {
		return fmt.Errorf("contract effect is invalid")
	}
	if err := validateFieldSet(request.Contract.Inputs); err != nil {
		return err
	}
	if err := validateFieldSet(request.Contract.Outputs); err != nil {
		return err
	}
	if len(request.OutputMappings) > 64 {
		return fmt.Errorf("output mappings exceed limits")
	}
	for path, target := range request.OutputMappings {
		if !mappingFieldName.MatchString(path) || strings.Contains(path, "..") || !bindOutputReference.MatchString(target) {
			return fmt.Errorf("output mapping is invalid")
		}
	}
	ref := request.OperationRef
	if !sourceIDPattern.MatchString(ref.SourceID) || !sha256Pattern.MatchString(ref.SourceSHA256) ||
		!sourceKindSupported(ref.SourceKind) || !safeNativeIdentity(ref.NativeSelector, 512) ||
		!safeNativeIdentity(ref.OperationKey, 256) || (ref.OperationID != "" && !safeNativeIdentity(ref.OperationID, 256)) ||
		!sha256Pattern.MatchString(request.IntentSHA256) {
		return fmt.Errorf("operation reference or intent digest is invalid")
	}
	return nil
}

// safeNativeIdentity preserves source-family identifiers without letting
// control characters, invalid UTF-8, or unbounded text enter the public wire.
// Native selectors and operation keys are opaque and may contain punctuation
// that differs by source family (for example Smithy shape IDs use '#').
func safeNativeIdentity(value string, maxRunes int) bool {
	if strings.TrimSpace(value) != value || value == "" || !utf8.ValidString(value) || utf8.RuneCountInString(value) > maxRunes {
		return false
	}
	for _, r := range value {
		if !unicode.IsPrint(r) {
			return false
		}
	}
	return true
}

func validateFieldSet(schema *uws1.ParamSchema) error {
	if schema == nil || schema.Type != "object" {
		return fmt.Errorf("field set must be an object schema")
	}
	count := 0
	var walk func(*uws1.ParamSchema, int) error
	walk = func(node *uws1.ParamSchema, depth int) error {
		if node == nil || depth > MaxSchemaDepth {
			return fmt.Errorf("field schema is invalid")
		}
		count++
		if count > MaxSchemaNodes || len(node.Properties) > MaxSchemaNodes || len(node.Required) > MaxSchemaNodes {
			return fmt.Errorf("field schema exceeds limits")
		}
		required := map[string]bool{}
		for _, name := range node.Required {
			if strings.TrimSpace(name) == "" || required[name] {
				return fmt.Errorf("field schema has invalid required names")
			}
			required[name] = true
			if _, ok := node.Properties[name]; !ok {
				return fmt.Errorf("field schema requires an undeclared property")
			}
		}
		for name := range node.Properties {
			if strings.TrimSpace(name) == "" || len(name) > 256 {
				return fmt.Errorf("field schema has an invalid property name")
			}
		}
		for _, child := range node.Properties {
			if err := walk(child, depth+1); err != nil {
				return err
			}
		}
		if node.Items != nil {
			if err := walk(node.Items, depth+1); err != nil {
				return err
			}
		}
		for _, list := range [][]*uws1.ParamSchema{node.AllOf, node.OneOf, node.AnyOf} {
			for _, child := range list {
				if err := walk(child, depth+1); err != nil {
					return err
				}
			}
		}
		return nil
	}
	return walk(schema, 0)
}

func sourceKindSupported(kind string) bool {
	switch kind {
	case "openapi", "google-discovery", "aws-smithy", "asyncapi", "graphql", "openrpc", "grpc-protobuf", "odata":
		return true
	default:
		return false
	}
}

func symbol(value string) bool {
	if len(value) == 0 || len(value) > 64 || value[0] < 'a' || value[0] > 'z' {
		return false
	}
	for _, c := range value[1:] {
		if !(c >= 'a' && c <= 'z') && !(c >= '0' && c <= '9') && c != '_' && c != '-' {
			return false
		}
	}
	return true
}

func validCredentialSymbol(value string) bool {
	return symbol(value) && value != "none" && value != "clear"
}

func validCredentialReference(value string) bool {
	return strings.HasPrefix(value, "credentials.") && validCredentialSymbol(strings.TrimPrefix(value, "credentials."))
}

func invalidInlineCredentialReference(value string) bool {
	value = strings.TrimSpace(value)
	return (value == "credentials" || strings.HasPrefix(value, "credentials.") || strings.HasPrefix(value, "credentials/") || strings.HasPrefix(value, "credentials:")) && !validCredentialReference(value)
}

func completedCheck(request CheckRequest, intentDigest string, items []CheckItem, unresolved []string) Outcome {
	if len(items) > MaxChecks {
		items = items[:MaxChecks]
	}
	if len(unresolved) > MaxUnresolved {
		unresolved = unresolved[:MaxUnresolved]
	}
	if items == nil {
		items = []CheckItem{}
	}
	if unresolved == nil {
		unresolved = []string{}
	}
	assessment := "compatible"
	for _, item := range items {
		if item.Status == "fail" {
			assessment = "incompatible"
			break
		}
		if item.Status == "indeterminate" {
			assessment = "indeterminate"
		}
	}
	result := CheckResult{
		StepID: request.StepID, IntentSHA256: intentDigest, OperationRef: request.OperationRef,
		Assessment: assessment, Checks: items, UnresolvedQuestions: unresolved,
	}
	return Outcome{Result: Result{Version: WireVersion, Kind: "result", Command: CheckCommand, Status: "completed", Diagnostics: []Diagnostic{}, Result: &result}}
}

func blocked(code, message string) Outcome {
	return Outcome{Result: Result{Version: WireVersion, Kind: "result", Command: CheckCommand, Status: "blocked", Diagnostics: []Diagnostic{{Code: code, Severity: "error", Message: message}}}, ExitCode: 4}
}

func conflict(code, message string) Outcome {
	return Outcome{Result: Result{Version: WireVersion, Kind: "result", Command: CheckCommand, Status: "conflict", Diagnostics: []Diagnostic{{Code: code, Severity: "error", Message: message}}}, ExitCode: 3}
}

type checkAccumulator struct{ items []CheckItem }

func newCheckAccumulator() *checkAccumulator {
	return &checkAccumulator{items: make([]CheckItem, 0, 16)}
}
func (c *checkAccumulator) add(code, status, message string) {
	c.addAt(code, status, message, "")
}

func (c *checkAccumulator) addAt(code, status, message, pointer string) {
	if len(c.items) >= MaxChecks {
		return
	}
	if len(pointer) > 256 {
		pointer = ""
	}
	c.items = append(c.items, CheckItem{Code: code, Status: status, Message: message, Pointer: pointer})
}

func resolveExampleRoot(example string) (string, error) {
	if strings.TrimSpace(example) == "" {
		return "", fmt.Errorf("missing example")
	}
	abs, err := filepath.Abs(example)
	if err != nil {
		return "", err
	}
	real, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(real)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("not a directory")
	}
	return filepath.Clean(real), nil
}

func readWithin(root, relative string, limit int64) ([]byte, error) {
	if strings.TrimSpace(relative) == "" || strings.Contains(relative, "\\") || filepath.IsAbs(relative) {
		return nil, fmt.Errorf("invalid relative path")
	}
	clean := filepath.Clean(filepath.FromSlash(relative))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return nil, fmt.Errorf("path escapes example")
	}
	parts := strings.Split(clean, string(filepath.Separator))
	current := root
	for i, part := range parts {
		if part == "" || part == "." || part == ".." {
			return nil, fmt.Errorf("invalid path segment")
		}
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil {
			return nil, err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("symlink component")
		}
		if i < len(parts)-1 && !info.IsDir() {
			return nil, fmt.Errorf("non-directory component")
		}
	}
	data, _, err := evidencefile.ReadRegular(current, limit)
	return data, err
}

func selectedSource(intent *workflowintent.Intent, step *workflowintent.Step) (string, string) {
	if intent == nil || step == nil {
		return "", "missing"
	}
	stepRef, stepState := uniqueSourceRef(step.OpenAPI, step.Source)
	if stepState != "ok" && stepState != "missing" {
		return "", stepState
	}
	rootRef, rootState := uniqueSourceRef(intent.OpenAPI, intent.Source)
	if rootState != "ok" && rootState != "missing" {
		return "", rootState
	}
	// A step-level source explicitly overrides the workflow default. Multi-
	// service workflows commonly use one root default and per-step sources.
	ref := stepRef
	if ref == "" {
		ref = rootRef
	}
	if ref == "" {
		return "", "missing"
	}
	if filepath.IsAbs(ref) || strings.Contains(ref, "\\") {
		return "", "unsafe"
	}
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(ref)))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || strings.HasPrefix(clean, "/") {
		return "", "unsafe"
	}
	return clean, "ok"
}

func uniqueSourceRef(primary, secondary string) (string, string) {
	primary = strings.TrimSpace(primary)
	secondary = strings.TrimSpace(secondary)
	if primary != "" && secondary != "" && normalizeSourceRef(primary) != normalizeSourceRef(secondary) {
		return "", "ambiguous"
	}
	if primary != "" {
		return primary, "ok"
	}
	if secondary != "" {
		return secondary, "ok"
	}
	return "", "missing"
}

func normalizeSourceRef(value string) string {
	return filepath.ToSlash(filepath.Clean(filepath.FromSlash(strings.TrimSpace(value))))
}

func sourceIDForPath(relative string) string {
	pathDigest := sha256.Sum256([]byte(filepath.ToSlash(relative)))
	return "src-" + hex.EncodeToString(pathDigest[:12])
}

func stepOperationMatches(stepOperation string, operation apitools.OperationSummary, ref OperationRef) bool {
	stepOperation = strings.TrimSpace(stepOperation)
	if stepOperation == "" {
		return false
	}
	return stepOperation == operation.ID || (operation.OperationID != "" && stepOperation == operation.OperationID) || stepOperation == ref.NativeSelector
}

func findSteps(steps []*workflowintent.Step, name string) []*workflowintent.Step {
	var found []*workflowintent.Step
	var walk func([]*workflowintent.Step)
	walk = func(values []*workflowintent.Step) {
		for _, step := range values {
			if step == nil {
				continue
			}
			if step.Name == name {
				found = append(found, step)
			}
			walk(step.Steps)
			for _, branch := range step.Cases {
				if branch != nil {
					walk(branch.Steps)
				}
			}
			if step.Default != nil {
				walk(step.Default.Steps)
			}
		}
	}
	walk(steps)
	return found
}

func checkRequiredMappings(step *workflowintent.Step, contract StepContract, operation apitools.OperationSummary, check *checkAccumulator) {
	with := make(map[string]string, len(step.With))
	for name, value := range step.With {
		with[strings.TrimSpace(name)] = strings.TrimSpace(value)
	}
	for _, bind := range step.Binds {
		if bind != nil {
			for name, value := range bind.Fields {
				if strings.TrimSpace(value) != "" {
					with[strings.TrimSpace(name)] = strings.TrimSpace(value)
				}
			}
		}
	}
	locations := declaredRequestLocations(operation)
	misplaced, ambiguous := false, false
	for key, value := range with {
		if value == "" {
			continue
		}
		section, name, qualified := requestMappingLocation(key)
		declared := locations[name]
		if qualified && len(declared) > 0 && !declared[section] {
			misplaced = true
		}
		if !qualified && len(declared) > 1 {
			ambiguous = true
		}
	}
	if misplaced {
		check.add("mapping.location", "fail", "A qualified request mapping uses a location different from the selected source.")
	}
	if ambiguous {
		check.add("mapping.ambiguous", "fail", "An unqualified request field occurs in more than one source location.")
	}
	missing := false
	missingPointer := ""
	for _, name := range requiredContractPaths(contract.Inputs) {
		if !contractInputMapped(with, name) {
			missing = true
			if missingPointer == "" {
				missingPointer = mappingPointer("/contract/inputs", name)
			}
		}
	}
	for _, parameter := range operation.Parameters {
		if !parameter.Required {
			continue
		}
		section := sourceParameterSection(parameter.In)
		if section == "" {
			if !hasInputMapping(with, parameter.Name) {
				missing = true
				if missingPointer == "" {
					missingPointer = "/operation/inputs/" + escapeJSONPointer(parameter.Name)
				}
			}
			continue
		}
		if !hasLocatedInputMapping(with, locations, section, parameter.Name) {
			missing = true
			if missingPointer == "" {
				missingPointer = "/operation/inputs/" + escapeJSONPointer(section) + "/" + escapeJSONPointer(parameter.Name)
			}
		}
	}
	if operation.RequestBody != nil {
		for _, path := range operation.RequestBody.RequiredFieldPaths {
			if !hasLocatedInputMapping(with, locations, "body", path) {
				missing = true
				if missingPointer == "" {
					missingPointer = "/operation/inputs/body/" + escapeJSONPointer(path)
				}
			}
		}
		for _, field := range operation.RequestBody.Fields {
			if field.Required && !hasLocatedInputMapping(with, locations, "body", field.Path) {
				missing = true
				if missingPointer == "" {
					missingPointer = "/operation/inputs/body/" + escapeJSONPointer(field.Path)
				}
			}
		}
		if operation.RequestBody.Required && len(operation.RequestBody.Fields) == 0 && len(operation.RequestBody.RequiredFieldPaths) == 0 && !hasLocatedInputMapping(with, locations, "body", "body") {
			check.addAt("mapping.request_body_evidence", "indeterminate", "The required request body has no inspectable field mapping evidence.", "/operation/inputs/body")
		}
	}
	if missing {
		check.addAt("mapping.incomplete", "fail", "One or more required input mappings are missing.", missingPointer)
	} else {
		check.add("mapping.required_inputs", "pass", "Required contract and operation inputs have mappings.")
	}
}

func sourceParameterSection(value string) string {
	switch strings.TrimSpace(value) {
	case "path", "query", "header", "cookie":
		return strings.TrimSpace(value)
	case "odata-query-option":
		return "query"
	case "graphql-variable", "json-rpc", "odata-parameter":
		return "body"
	default:
		return ""
	}
}

func declaredRequestLocations(operation apitools.OperationSummary) map[string]map[string]bool {
	locations := map[string]map[string]bool{}
	add := func(name, section string) {
		name = strings.TrimSpace(name)
		if name == "" || section == "" {
			return
		}
		if locations[name] == nil {
			locations[name] = map[string]bool{}
		}
		locations[name][section] = true
	}
	for _, parameter := range operation.Parameters {
		add(parameter.Name, sourceParameterSection(parameter.In))
	}
	if operation.RequestBody != nil {
		add("body", "body")
		for _, field := range operation.RequestBody.Fields {
			add(field.Path, "body")
		}
		for _, path := range operation.RequestBody.RequiredFieldPaths {
			add(path, "body")
		}
	}
	for _, alternative := range operation.SecurityRequirementSets {
		for _, requirement := range alternative.Requirements {
			section := sourceParameterSection(requirement.In)
			add(requirement.ParameterName, section)
			add(requirement.Name, section)
		}
	}
	return locations
}

func requestMappingLocation(key string) (section, name string, qualified bool) {
	for _, prefix := range []string{"body.", "query.", "path.", "header.", "cookie."} {
		if strings.HasPrefix(key, prefix) {
			return strings.TrimSuffix(prefix, "."), strings.TrimPrefix(key, prefix), true
		}
	}
	return "", key, false
}

func hasInputMapping(mappings map[string]string, name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	for _, key := range []string{name, "body." + name, "query." + name, "path." + name, "header." + name, "cookie." + name} {
		if strings.TrimSpace(mappings[key]) != "" {
			return true
		}
	}
	return false
}

// A contract/source comparison cannot prove a binding when the actual workflow
// input supplies a different type, or may be absent at execution time.
func checkMappedWorkflowValues(intent *workflowintent.Intent, step *workflowintent.Step, contract StepContract, operation apitools.OperationSummary, check *checkAccumulator) {
	inputs := map[string]*workflowintent.Input{}
	for _, input := range intent.Inputs {
		if input != nil {
			inputs[input.Name] = input
		}
	}
	mappings := combineStepMappings(step)
	locations := declaredRequestLocations(operation)
	fail, unknown := false, false
	for key, raw := range mappings {
		section, name, qualified := requestMappingLocation(strings.TrimSpace(key))
		if !qualified {
			if len(locations[name]) == 1 {
				for section = range locations[name] {
				}
			}
		}
		source := requestFieldEvidence(operation, section, name)
		path, isInput := inputContractPath(raw)
		if !isInput {
			if source.found {
				unknown = true
			}
			continue
		}
		rootName := strings.Split(path, ".")[0]
		input := inputs[rootName]
		if input == nil {
			continue // the existing workflow-input reference check reports this
		}
		contractRoot := contractRootField(contract.Inputs, path)
		if contractRoot == nil || contractRoot.Type == "" || input.Type == "" {
			unknown = true
		} else if compatible, known := workflowInputTypeCompatible(input.Type, contractRoot.Type); !known {
			unknown = true
		} else if !compatible {
			fail = true
		}
		_, _, contractFound := contractSchemaField(contract.Inputs, path)
		if !contractFound {
			unknown = true
		}
		contractRootRequired := schemaRequired(contract.Inputs, rootName)
		if (contractRootRequired || source.required) && !input.Required {
			if input.Default == "" {
				fail = true
			} else {
				unknown = true // the untyped default has not been checked here
			}
		}
	}
	if fail {
		check.add("mapping.workflow_value_types", "fail", "A mapped workflow input has an incompatible type or is optional where a value is required.")
	} else if unknown {
		check.add("mapping.workflow_value_types", "indeterminate", "The mapped value's type or availability cannot be established from the workflow intent.")
	}
}

func workflowInputTypeCompatible(actual, expected string) (compatible, known bool) {
	if actual == "" || expected == "" {
		return false, false
	}
	if actual == expected || actual == "integer" && expected == "number" {
		return true, true
	}
	for _, value := range []string{actual, expected} {
		switch value {
		case "string", "integer", "number", "boolean", "object", "array":
		default:
			return false, false
		}
	}
	return false, true
}

func checkOutputs(intent *workflowintent.Intent, step *workflowintent.Step, contract StepContract, operation apitools.OperationSummary, requestedMappings map[string]string, representable bool, check *checkAccumulator) (string, string) {
	declared := contract.Outputs.Properties
	mappings := effectiveOutputMappings(contract.Outputs, requestedMappings)
	if outputsReferenceMissingDeclaredField(intent, step, contract.Outputs, mappings) {
		check.addAt("mapping.output_references", "fail", "A workflow output reference selects a field outside the step contract.", "/intent/outputs")
	} else {
		check.add("mapping.output_references", "pass", "Relevant workflow outputs stay within the step contract.")
	}
	if len(declared) == 0 {
		check.add("mapping.outputs", "pass", "The step contract declares no output fields.")
		if requestedMappings != nil && len(requestedMappings) > 0 {
			return "incompatible", "/output_mappings"
		}
		return "compatible", ""
	}
	if operation.ResponseBody == nil || len(operation.ResponseBody.Fields) == 0 && operation.ResponseBody.Schema == nil {
		check.addAt("mapping.outputs", "indeterminate", "The selected operation has no inspectable response-field summary.", "/operation/outputs")
		return "indeterminate", "/operation/outputs"
	}
	status := "compatible"
	pointer := ""
	seenTargets := map[string]string{}
	for contractPath, target := range requestedMappings {
		if _, _, exists := contractSchemaField(contract.Outputs, contractPath); !exists {
			status = aggregateMappingStatus(status, "incompatible")
			if pointer == "" {
				pointer = mappingPointer("/contract/outputs", contractPath)
			}
			continue
		}
		responsePath, ok := responseReferencePath(target)
		if !ok {
			status = aggregateMappingStatus(status, "incompatible")
			if pointer == "" {
				pointer = mappingPointer("/contract/outputs", contractPath)
			}
			continue
		}
		if previous, exists := seenTargets[responsePath]; exists && previous != contractPath {
			status = aggregateMappingStatus(status, "incompatible")
			if pointer == "" {
				pointer = mappingPointer("/contract/outputs", contractPath)
			}
		}
		seenTargets[responsePath] = contractPath
	}
	missingRequired, missingOptional := false, false
	missingPointer := ""
	paths := make([]string, 0, len(contractOutputPaths(declared)))
	for name := range contractOutputPaths(declared) {
		paths = append(paths, name)
	}
	sort.Strings(paths)
	for _, name := range paths {
		field, _, exists := contractSchemaField(contract.Outputs, name)
		if !exists {
			status = aggregateMappingStatus(status, "indeterminate")
			if pointer == "" {
				pointer = mappingPointer("/contract/outputs", name)
			}
			continue
		}
		required := schemaRequiredPath(contract.Outputs, name)
		target, mapped := effectiveOutputTarget(name, mappings)
		if !mapped {
			if required {
				missingRequired = true
			} else {
				missingOptional = true
			}
			if missingPointer == "" {
				missingPointer = mappingPointer("/contract/outputs", name)
			}
			continue
		}
		responsePath, ok := responseReferencePath(target)
		if !ok {
			status = aggregateMappingStatus(status, "incompatible")
			if pointer == "" {
				pointer = mappingPointer("/contract/outputs", name)
			}
			continue
		}
		source := responseFieldEvidence(operation, responsePath)
		if !source.found {
			if required {
				missingRequired = true
			} else {
				missingOptional = true
			}
			if missingPointer == "" {
				missingPointer = mappingPointer("/contract/outputs", name)
			}
			continue
		}
		fieldStatus := fieldTypeStatus(field, source, false)
		if required && source.requiredKnown && !source.required {
			fieldStatus = "incompatible"
		}
		if required && !source.requiredKnown {
			fieldStatus = aggregateMappingStatus(fieldStatus, "indeterminate")
		}
		if source.nullable {
			fieldStatus = aggregateMappingStatus(fieldStatus, "indeterminate")
		}
		if fieldStatus != "compatible" && (status == "compatible" || fieldStatus == "incompatible" && status != "incompatible") {
			pointer = mappingPointer("/contract/outputs", name)
		}
		status = aggregateMappingStatus(status, fieldStatus)
	}
	if missingRequired {
		if pointer == "" {
			pointer = missingPointer
		}
		check.addAt("mapping.outputs", "fail", "One or more required contract outputs are absent from the selected response.", pointer)
		status = aggregateMappingStatus(status, "incompatible")
	} else if missingOptional {
		if pointer == "" {
			pointer = missingPointer
		}
		check.addAt("mapping.outputs", "indeterminate", "One or more optional contract outputs are not confirmed by the response summary.", pointer)
		status = aggregateMappingStatus(status, "indeterminate")
	} else {
		check.add("mapping.outputs", "pass", "Declared contract outputs are present in the selected response summary.")
	}
	if !representable {
		status = aggregateMappingStatus(status, "indeterminate")
		if pointer == "" {
			pointer = "/operation/capabilities/outputs"
		}
	}
	return status, pointer
}

func schemaRequired(schema *uws1.ParamSchema, name string) bool {
	for _, value := range schema.Required {
		if value == name {
			return true
		}
	}
	return false
}

func schemaRequiredPath(schema *uws1.ParamSchema, path string) bool {
	parts := strings.Split(path, ".")
	current := schema
	allRequired := true
	for index, part := range parts {
		if current == nil {
			return false
		}
		if current.Type == "array" {
			current = current.Items
			if current == nil {
				return false
			}
		}
		if !schemaRequired(current, part) {
			allRequired = false
		}
		current = current.Properties[part]
		if current == nil && index < len(parts)-1 {
			return false
		}
	}
	return allRequired
}

func contractOutputPaths(properties map[string]*uws1.ParamSchema) map[string]bool {
	out := map[string]bool{}
	var walk func(string, *uws1.ParamSchema)
	walk = func(prefix string, schema *uws1.ParamSchema) {
		if schema == nil {
			return
		}
		out[prefix] = true
		for name, child := range schema.Properties {
			next := name
			if prefix != "" {
				next = prefix + "." + name
			}
			walk(next, child)
		}
		if schema.Items != nil {
			walk(prefix, schema.Items)
		}
	}
	for name, schema := range properties {
		walk(name, schema)
	}
	return out
}

func outputsReferenceMissingDeclaredField(intent *workflowintent.Intent, selected *workflowintent.Step, declared *uws1.ParamSchema, mappings map[string]string) bool {
	if intent == nil || selected == nil {
		return false
	}
	for _, output := range intent.Outputs {
		if output == nil {
			continue
		}
		if root, field, ok := outputRef(output.From); ok && root == selected.Name && field != "" && !outputPathCoveredByContract(field, declared, mappings) {
			return true
		}
	}
	var invalid bool
	var inspect func([]*workflowintent.Step)
	inspect = func(steps []*workflowintent.Step) {
		for _, step := range steps {
			if step == nil {
				continue
			}
			for _, value := range step.With {
				if root, field, ok := outputRef(value); ok && root == selected.Name && field != "" && !outputPathCoveredByContract(field, declared, mappings) {
					invalid = true
				}
			}
			for _, bind := range step.Binds {
				if bind != nil {
					for _, value := range bind.Fields {
						if root, field, ok := outputRef(value); ok && root == selected.Name && field != "" && !outputPathCoveredByContract(field, declared, mappings) {
							invalid = true
						}
					}
				}
			}
			inspect(step.Steps)
			for _, branch := range step.Cases {
				if branch != nil {
					inspect(branch.Steps)
				}
			}
			if step.Default != nil {
				inspect(step.Default.Steps)
			}
		}
	}
	inspect(intent.Steps)
	return invalid
}

func outputRef(value string) (string, string, bool) {
	match := stepOutputReference.FindStringSubmatch(strings.TrimSpace(value))
	if len(match) != 3 {
		return "", "", false
	}
	return match[1], match[2], true
}

func contractHasOutputPath(properties map[string]*uws1.ParamSchema, path string) bool {
	parts := strings.Split(path, ".")
	node := properties[parts[0]]
	if node == nil {
		return false
	}
	for _, part := range parts[1:] {
		if node.Type == "array" {
			node = node.Items
			if node == nil {
				return false
			}
		}
		node = node.Properties[part]
		if node == nil {
			return false
		}
	}
	return true
}

func checkInlineCredentialMappings(step *workflowintent.Step, check *checkAccumulator) {
	for _, value := range step.With {
		if invalidInlineCredentialReference(value) {
			check.add("mapping.credential_reference", "fail", "An inline credential reference is malformed or uses a reserved symbol.")
			return
		}
	}
	for _, binding := range step.Binds {
		if binding != nil {
			for _, value := range binding.Fields {
				if invalidInlineCredentialReference(value) {
					check.add("mapping.credential_reference", "fail", "An inline credential reference is malformed or uses a reserved symbol.")
					return
				}
			}
		}
	}
}

func checkAuthentication(intent *workflowintent.Intent, step *workflowintent.Step, operation apitools.OperationSummary, check *checkAccumulator) {
	sets := operation.SecurityRequirementSets
	if len(sets) == 0 {
		check.add("authentication.evidence", "indeterminate", "The source does not provide an explicit authentication alternative.")
		return
	}
	for _, alternative := range sets {
		if len(alternative.Requirements) == 0 {
			check.add("authentication.alternative", "pass", "An explicit anonymous alternative is available.")
			return
		}
		allBound := true
		for _, requirement := range alternative.Requirements {
			if !authenticationBound(intent, step, requirement) {
				allBound = false
				break
			}
		}
		if allBound {
			check.add("authentication.alternative", "pass", "At least one complete authentication alternative is configured.")
			return
		}
	}
	check.add("authentication.alternative", "fail", "No complete source authentication alternative is mapped.")
}

func authenticationBound(intent *workflowintent.Intent, step *workflowintent.Step, requirement apitools.SecuritySummary) bool {
	for _, declaration := range intent.Security {
		if declaration == nil || declaration.TokenFrom == "" {
			continue
		}
		if strings.EqualFold(declaration.Name, requirement.Name) || strings.EqualFold(declaration.TokenFrom, requirement.Name) {
			if !authoring.ContainsLikelyCredentialValue([]byte(declaration.TokenFrom)) {
				return true
			}
		}
	}
	fields := []string{requirement.ParameterName, requirement.Name}
	if strings.EqualFold(requirement.Scheme, "bearer") || strings.EqualFold(requirement.Scheme, "basic") {
		fields = append(fields, "Authorization")
	}
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		if value := strings.TrimSpace(step.With[field]); value != "" && !invalidInlineCredentialReference(value) && !authoring.ContainsLikelyCredentialValue([]byte(value)) {
			return true
		}
		for _, binding := range step.Binds {
			if binding != nil {
				if value := strings.TrimSpace(binding.Fields[field]); value != "" && !invalidInlineCredentialReference(value) && !authoring.ContainsLikelyCredentialValue([]byte(value)) {
					return true
				}
			}
		}
	}
	return false
}

func checkDependencies(intent *workflowintent.Intent, selected *workflowintent.Step, selectedID string, check *checkAccumulator) {
	all := map[string]int{}
	var walk func([]*workflowintent.Step)
	walk = func(steps []*workflowintent.Step) {
		for _, step := range steps {
			if step == nil {
				continue
			}
			all[step.Name]++
			walk(step.Steps)
			for _, branch := range step.Cases {
				if branch != nil {
					walk(branch.Steps)
				}
			}
			if step.Default != nil {
				walk(step.Default.Steps)
			}
		}
	}
	walk(intent.Steps)
	dependencies := map[string]bool{}
	missing := false
	for _, dependency := range selected.DependsOn {
		dependencies[dependency] = true
		if all[dependency] != 1 || dependency == selectedID {
			missing = true
		}
	}
	refs := map[string]bool{}
	for _, value := range selected.With {
		collectStepRefs(value, refs)
	}
	for _, binding := range selected.Binds {
		if binding != nil {
			if binding.From != "" {
				refs[binding.From] = true
			}
			for _, value := range binding.Fields {
				collectStepRefs(value, refs)
			}
		}
	}
	for ref := range refs {
		if ref == selectedID || !dependencies[ref] {
			missing = true
		}
	}
	if missing {
		check.add("dependency.mapping", "fail", "A referenced workflow dependency is missing or undeclared.")
	} else {
		check.add("dependency.mapping", "pass", "Declared and referenced step dependencies resolve.")
	}
}

// dependencyCycleFrom checks the selected step and its transitive prerequisites.
// It includes both declared dependencies and step-output references so a
// self-reference or a back-edge cannot be hidden by an incomplete depends_on.
func dependencyCycleFrom(steps []*workflowintent.Step, selectedID string) bool {
	counts := map[string]int{}
	var nodes []*workflowintent.Step
	var collect func([]*workflowintent.Step)
	collect = func(values []*workflowintent.Step) {
		for _, step := range values {
			if step == nil {
				continue
			}
			nodes = append(nodes, step)
			counts[step.Name]++
			collect(step.Steps)
			for _, branch := range step.Cases {
				if branch != nil {
					collect(branch.Steps)
				}
			}
			if step.Default != nil {
				collect(step.Default.Steps)
			}
		}
	}
	collect(steps)
	if counts[selectedID] != 1 {
		return false
	}

	graph := make(map[string][]string, len(nodes))
	for _, step := range nodes {
		if counts[step.Name] != 1 {
			continue
		}
		dependencies := map[string]bool{}
		for _, name := range step.DependsOn {
			dependencies[strings.TrimSpace(name)] = true
		}
		refs := map[string]bool{}
		for _, value := range step.With {
			collectStepRefs(value, refs)
		}
		for _, binding := range step.Binds {
			if binding == nil {
				continue
			}
			collectStepRefs(binding.From, refs)
			for _, value := range binding.Fields {
				collectStepRefs(value, refs)
			}
		}
		for name := range refs {
			dependencies[name] = true
		}
		for name := range dependencies {
			if counts[name] == 1 {
				graph[step.Name] = append(graph[step.Name], name)
			}
		}
		sort.Strings(graph[step.Name])
	}

	type frame struct {
		name string
		next int
	}
	state := map[string]uint8{selectedID: 1}
	stack := []frame{{name: selectedID}}
	for len(stack) > 0 {
		current := &stack[len(stack)-1]
		edges := graph[current.name]
		if current.next >= len(edges) {
			state[current.name] = 2
			stack = stack[:len(stack)-1]
			continue
		}
		next := edges[current.next]
		current.next++
		if state[next] == 1 {
			return true
		}
		if state[next] == 0 {
			state[next] = 1
			stack = append(stack, frame{name: next})
		}
	}
	return false
}

func collectStepRefs(value string, refs map[string]bool) {
	for _, match := range stepOutputReference.FindAllStringSubmatch(value, -1) {
		if len(match) > 1 {
			refs[match[1]] = true
		}
	}
}

func resolveOperationCandidate(ctx context.Context, relative string, source []byte, ref OperationRef, stepContract StepContract) (apitools.OperationCandidate, error) {
	contract, _, _ := mapStepContract(stepContract)
	report, err := apitools.BuildOperationCandidates(ctx, apitools.OperationCandidateOptions{
		Sources: []apitools.OperationSourceInput{{
			Kind: apitools.OperationSourceKind(ref.SourceKind), Name: ref.SourceID,
			Path: relative, Content: source,
		}},
		Contract: contract,
		MaxBytes: MaxSourceBytes, MaxOperations: MaxSourceOps, MaxCandidates: MaxCandidateShortlist,
		PromptBudget: apitools.PromptBudget{MaxTextRunes: maxCandidateSummaryTextRunes},
	})
	if err != nil || report.Truncated {
		return apitools.OperationCandidate{}, errSourceUnsupported
	}
	if hasErrorDiagnostics(mapAPIDiagnostics(report.Diagnostics)) {
		return apitools.OperationCandidate{}, errSourceUnsupported
	}
	expectedDigest := strings.TrimPrefix(ref.SourceSHA256, "sha256:")
	var found []apitools.OperationCandidate
	for _, candidate := range report.Candidates {
		if candidate.Source.Kind == apitools.OperationSourceKind(ref.SourceKind) && candidate.Operation.DocumentName == ref.SourceID &&
			candidate.Source.SHA256 == expectedDigest && candidate.Source.Selector == ref.NativeSelector &&
			candidate.Operation.ID == ref.OperationKey &&
			(candidate.Operation.OperationID == ref.OperationID || ref.OperationID == "") {
			found = append(found, candidate)
		}
	}
	if len(found) == 0 {
		return apitools.OperationCandidate{}, errOperationMissing
	}
	if len(found) != 1 {
		return apitools.OperationCandidate{}, errOperationAmbiguous
	}
	return found[0], nil
}

func checkEffect(contract StepContract, candidate apitools.OperationCandidate, available bool, check *checkAccumulator) string {
	if contract.Effect == "unknown" {
		check.add("effect.evidence", "indeterminate", "The step contract does not declare a read/write effect constraint.")
		return "indeterminate"
	}
	if !available || (candidate.Effect.Class != apitools.OperationEffectRead && candidate.Effect.Class != apitools.OperationEffectWrite) || len(candidate.Effect.Evidence) == 0 {
		check.add("effect.evidence", "indeterminate", "The selected source operation has no usable effect classification evidence.")
		return "indeterminate"
	}
	if candidate.Effect.Class != apitools.OperationEffect(contract.Effect) {
		check.add("effect.evidence", "fail", "The selected source operation's classified effect conflicts with the step contract.")
		return "fail"
	}
	check.add("effect.evidence", "pass", "Source metadata supports the selected operation's effect class.")
	return "pass"
}

func unresolvedForContract(contract StepContract, includeOperation, includeEffect bool) []string {
	questions := []string{"Does the selected operation fulfill the natural-language purpose?"}
	if len(contract.AccountConstraints) > 0 {
		questions = append(questions, "Does the configured account satisfy the reviewed account constraint?")
	}
	if len(contract.DestinationConstraints) > 0 {
		questions = append(questions, "Does the configured destination satisfy the reviewed destination constraint?")
	}
	if contract.Effect != "unknown" && includeEffect {
		questions = append(questions, "Is the selected operation effect acceptable? Source metadata has no usable effect classification evidence.")
	}
	if includeOperation {
		questions = append(questions, "Does the step's source reference identify the intended local API document?")
	}
	sort.Strings(questions)
	return questions
}

func sourceDigest(data []byte) string {
	digest := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func ValidUTF8Request(data []byte) bool { return utf8.Valid(data) }

// Prevent compile-time drift in the source digest implementation used by
// fixtures and the CLI.
var _ = fmt.Sprintf
