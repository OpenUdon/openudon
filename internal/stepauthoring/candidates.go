package stepauthoring

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/OpenUdon/apitools"
	"github.com/OpenUdon/openudon/internal/packageartifacts"
	"github.com/OpenUdon/uws/uws1"
)

const (
	CandidatesCommand               = "step.candidates"
	MaxCandidateSources             = 32
	MaxCandidateSourceEntries       = 4096
	MaxCandidateSourceBytesTotal    = 32 << 20
	MaxCandidateShortlist           = 100
	MaxCandidateDiagnostics         = 32
	MaxCandidateValues              = 64
	MaxCandidateEvidence            = 16
	MaxCandidateStrings             = 64
	maxCandidateSummaryTextRunes    = 512
	maxCandidateSummaryFieldRunes   = 256
	maxCandidateSummaryLocationSize = 64
)

var candidateDiagnosticCode = regexp.MustCompile(`^[a-z][a-z0-9_.-]{0,127}$`)
var contractExtensionName = regexp.MustCompile(`^x-[A-Za-z0-9][A-Za-z0-9._-]{0,125}$`)

type CandidateSourceFilter struct {
	SourceKind   string `json:"source_kind"`
	SourceID     string `json:"source_id"`
	SourceSHA256 string `json:"source_sha256"`
}

type CandidatesRequest struct {
	Version       string                  `json:"version"`
	Kind          string                  `json:"kind"`
	Command       string                  `json:"command"`
	Contract      StepContract            `json:"contract"`
	SourceFilters []CandidateSourceFilter `json:"source_filters,omitempty"`
	Limit         *int                    `json:"limit,omitempty"`
}

type CandidateEvidence struct {
	Kind      string `json:"kind"`
	Reference string `json:"reference"`
}

type CandidateValue struct {
	Name        string              `json:"name"`
	Location    string              `json:"location,omitempty"`
	Type        string              `json:"type,omitempty"`
	Format      string              `json:"format,omitempty"`
	Required    *bool               `json:"required,omitempty"`
	Description string              `json:"description,omitempty"`
	Evidence    []CandidateEvidence `json:"evidence"`
}

type CandidateSummary struct {
	Description string              `json:"description"`
	Inputs      []CandidateValue    `json:"inputs,omitempty"`
	Outputs     []CandidateValue    `json:"outputs,omitempty"`
	Evidence    []CandidateEvidence `json:"evidence"`
	Gaps        []string            `json:"gaps"`
}

type CandidateCompatibility struct {
	Status    string              `json:"status"`
	Score     int                 `json:"score"`
	Evidence  []CandidateEvidence `json:"evidence,omitempty"`
	Reasons   []string            `json:"reasons,omitempty"`
	Missing   []string            `json:"missing,omitempty"`
	Conflicts []string            `json:"conflicts,omitempty"`
	Gaps      []string            `json:"gaps,omitempty"`
}

type CandidateMatch struct {
	Score   int                    `json:"score"`
	Purpose CandidateCompatibility `json:"purpose"`
	Inputs  CandidateCompatibility `json:"inputs"`
	Outputs CandidateCompatibility `json:"outputs"`
	Effect  CandidateCompatibility `json:"effect"`
}

type CandidateEffect struct {
	Class    string              `json:"class"`
	Evidence []CandidateEvidence `json:"evidence"`
	Reasons  []string            `json:"reasons"`
}

type CandidateAuthRequirement struct {
	Name           string   `json:"name"`
	Type           string   `json:"type,omitempty"`
	Scheme         string   `json:"scheme,omitempty"`
	Location       string   `json:"location,omitempty"`
	ParameterName  string   `json:"parameter_name,omitempty"`
	CredentialSlot string   `json:"credential_slot,omitempty"`
	Flows          []string `json:"flows,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
}

type CandidateAuthAlternative struct {
	Requirements []CandidateAuthRequirement `json:"requirements"`
}

type CandidateAuthentication struct {
	Status       string                     `json:"status"`
	Alternatives []CandidateAuthAlternative `json:"alternatives,omitempty"`
}

type CandidateSourceCapability struct {
	Dimension string              `json:"dimension"`
	Status    string              `json:"status"`
	Evidence  []CandidateEvidence `json:"evidence,omitempty"`
	Gaps      []string            `json:"gaps,omitempty"`
}

type CandidateSourceReport struct {
	SourceID       string                      `json:"source_id"`
	SourceKind     string                      `json:"source_kind"`
	OperationCount int                         `json:"operation_count"`
	Capabilities   []CandidateSourceCapability `json:"capabilities"`
	Diagnostics    []Diagnostic                `json:"diagnostics"`
}

type StepCandidate struct {
	Rank           int                     `json:"rank"`
	OperationRef   OperationRef            `json:"operation_ref"`
	Summary        CandidateSummary        `json:"summary"`
	Match          CandidateMatch          `json:"match"`
	Authentication CandidateAuthentication `json:"authentication"`
	Effect         CandidateEffect         `json:"effect"`
}

type CandidatesData struct {
	Sources    []CandidateSourceReport `json:"sources,omitempty"`
	Candidates []StepCandidate         `json:"candidates"`
	Truncated  bool                    `json:"truncated"`
}

type CandidatesWireResult struct {
	Version     string          `json:"version"`
	Kind        string          `json:"kind"`
	Command     string          `json:"command"`
	Status      string          `json:"status"`
	Diagnostics []Diagnostic    `json:"diagnostics"`
	Result      *CandidatesData `json:"result,omitempty"`
}

type CandidatesOutcome struct {
	Result   CandidatesWireResult
	ExitCode int
}

type localCandidateSource struct {
	kind           string
	relPath        string
	id             string
	expectedDigest string
	digest         string
	content        []byte
}

type sourceScanFailure struct {
	code     string
	conflict bool
}

var candidateSourceKinds = []string{
	"openapi", "google-discovery", "aws-smithy", "asyncapi",
	"graphql", "openrpc", "grpc-protobuf", "odata",
}

type candidateSourceDirectory struct {
	kind      string
	directory string
}

var candidateSourceDirectories = []candidateSourceDirectory{
	{kind: "openapi", directory: "openapi"},
	{kind: "google-discovery", directory: "google-discovery"},
	{kind: "google-discovery", directory: "discovery"}, // legacy package directory
	{kind: "aws-smithy", directory: "aws-smithy"},
	{kind: "asyncapi", directory: "asyncapi"},
	{kind: "graphql", directory: "graphql"},
	{kind: "openrpc", directory: "openrpc"},
	{kind: "grpc-protobuf", directory: "grpc-protobuf"},
	{kind: "odata", directory: "odata"},
}

// Candidates ranks exact local source operations against one declared step.
// It never fetches source URLs, resolves credentials, or chooses an account.
func Candidates(ctx context.Context, exampleDir string, request CandidatesRequest) CandidatesOutcome {
	if err := validateCandidatesRequest(request); err != nil {
		code := "request.invalid"
		if request.Version != "" && request.Version != WireVersion {
			code = "request.unsupported_version"
		}
		return candidatesFailure("failed", code, "The step-candidates request is invalid or uses an unsupported version.", 2)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return candidatesFailure("blocked", "request.cancelled", "Candidate discovery was cancelled before it started.", 4)
	}
	root, err := resolveExampleRoot(exampleDir)
	if err != nil {
		return candidatesFailure("blocked", "example.invalid", "The selected example is unavailable or unsafe.", 4)
	}

	sources, failure := loadCandidateSources(ctx, root, request.SourceFilters)
	if failure != nil {
		if failure.conflict {
			return candidatesFailure("conflict", failure.code, candidateFailureMessage(failure.code), 3)
		}
		return candidatesFailure("blocked", failure.code, candidateFailureMessage(failure.code), 4)
	}
	if len(sources) == 0 {
		data := CandidatesData{Sources: []CandidateSourceReport{}, Candidates: []StepCandidate{}, Truncated: false}
		return candidatesComplete(data, []Diagnostic{{Code: "source.no_documents", Severity: "warning", Message: "No supported local API source documents were found in the selected example."}})
	}
	if err := ctx.Err(); err != nil {
		return candidatesFailure("blocked", "request.cancelled", "Candidate discovery was cancelled before ranking completed.", 4)
	}

	contract, unsupportedInputs, unsupportedOutputs := mapStepContract(request.Contract)
	inputGaps := unsupportedContractDiagnostics("inputs", unsupportedInputs, rootContractExtensionNames(request.Contract.Inputs))
	outputGaps := unsupportedContractDiagnostics("outputs", unsupportedOutputs, rootContractExtensionNames(request.Contract.Outputs))

	apiSources := make([]apitools.OperationSourceInput, 0, len(sources))
	for _, source := range sources {
		apiSources = append(apiSources, apitools.OperationSourceInput{
			Kind: apitools.OperationSourceKind(source.kind), Name: source.id,
			Path: source.relPath, Content: source.content,
		})
	}
	apiReport, apiErr := apitools.BuildOperationCandidates(ctx, apitools.OperationCandidateOptions{
		Sources: apiSources, Contract: contract, MaxBytes: MaxSourceBytes,
		MaxOperations: MaxSourceOps, MaxCandidates: MaxCandidateShortlist,
		PromptBudget: apitools.PromptBudget{MaxTextRunes: maxCandidateSummaryTextRunes},
	})
	if err := ctx.Err(); err != nil {
		return candidatesFailure("blocked", "request.cancelled", "Candidate ranking was cancelled before it completed.", 4)
	}
	apiDiagnostics := mapAPIDiagnostics(apiReport.Diagnostics)
	if apiErr != nil || apiReport.Truncated || hasErrorDiagnostics(apiDiagnostics) {
		if len(apiDiagnostics) == 0 {
			apiDiagnostics = []Diagnostic{{Code: "source.unavailable", Severity: "error", Message: "One or more selected source documents could not be ranked completely."}}
		}
		return candidatesFailureWithDiagnostics("blocked", apiDiagnostics, 4)
	}

	data, mapDiagnostics, mapErr := mapCandidateReport(apiReport, request.Limit)
	if mapErr != nil {
		return candidatesFailure("failed", "result.invalid", "API source metadata could not be represented safely in the step-candidates result.", 1)
	}
	applyUnsupportedDimensions(&data, unsupportedInputs, unsupportedOutputs)
	diagnostics := append(apiDiagnostics, inputGaps...)
	diagnostics = append(diagnostics, outputGaps...)
	diagnostics = append(diagnostics, mapDiagnostics...)
	if len(request.Contract.AccountConstraints) > 0 || len(request.Contract.DestinationConstraints) > 0 {
		diagnostics = append(diagnostics, Diagnostic{
			Code: "contract.constraints_not_ranked", Severity: "info",
			Message: "Account and destination constraints are not operation metadata and were not ranked.",
		})
	}
	return candidatesComplete(data, diagnostics)
}

func validateCandidatesRequest(request CandidatesRequest) error {
	if request.Version != WireVersion || request.Kind != "request" || request.Command != CandidatesCommand {
		return fmt.Errorf("invalid envelope")
	}
	if !symbol(request.Contract.ID) || strings.TrimSpace(request.Contract.Purpose) == "" ||
		len(request.Contract.Purpose) > 2048 || !utf8.ValidString(request.Contract.Purpose) {
		return fmt.Errorf("invalid contract identity or purpose")
	}
	if request.Contract.Effect != "read" && request.Contract.Effect != "write" && request.Contract.Effect != "unknown" {
		return fmt.Errorf("invalid expected effect")
	}
	if err := validateFieldSet(request.Contract.Inputs); err != nil {
		return err
	}
	if err := validateFieldSet(request.Contract.Outputs); err != nil {
		return err
	}
	if len(request.Contract.AccountConstraints) > 16 || len(request.Contract.DestinationConstraints) > 16 {
		return fmt.Errorf("too many account or destination constraints")
	}
	for _, value := range append(append([]string(nil), request.Contract.AccountConstraints...), request.Contract.DestinationConstraints...) {
		if strings.TrimSpace(value) == "" || len(value) > 256 || !utf8.ValidString(value) {
			return fmt.Errorf("invalid account or destination constraint")
		}
	}
	if request.Limit != nil && (*request.Limit < 1 || *request.Limit > MaxCandidateShortlist) {
		return fmt.Errorf("candidate limit is outside the supported range")
	}
	if len(request.SourceFilters) > MaxCandidateSources {
		return fmt.Errorf("too many source filters")
	}
	seen := map[string]bool{}
	for _, filter := range request.SourceFilters {
		if !sourceKindSupported(filter.SourceKind) || !sourceIDPattern.MatchString(filter.SourceID) || !sha256Pattern.MatchString(filter.SourceSHA256) {
			return fmt.Errorf("invalid source filter")
		}
		key := filter.SourceKind + "\x00" + filter.SourceID
		if seen[key] {
			return fmt.Errorf("duplicate source filter")
		}
		seen[key] = true
	}
	return nil
}

func loadCandidateSources(ctx context.Context, root string, filters []CandidateSourceFilter) ([]localCandidateSource, *sourceScanFailure) {
	directories := append([]candidateSourceDirectory(nil), candidateSourceDirectories...)
	if len(filters) > 0 {
		kindSet := map[string]bool{}
		for _, filter := range filters {
			kindSet[filter.SourceKind] = true
		}
		directories = directories[:0]
		for _, source := range candidateSourceDirectories {
			if kindSet[source.kind] {
				directories = append(directories, source)
			}
		}
	}
	files := make([]localCandidateSource, 0)
	visited := 0
	for _, source := range directories {
		if err := ctx.Err(); err != nil {
			return nil, &sourceScanFailure{code: "request.cancelled"}
		}
		start := filepath.Join(root, source.directory)
		info, err := os.Lstat(start)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return nil, &sourceScanFailure{code: "source.unavailable"}
		}
		err = filepath.WalkDir(start, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			visited++
			if visited > MaxCandidateSourceEntries {
				return errors.New("candidate source entry limit")
			}
			if packageartifacts.IsAdvisorySecuritySidecarPath(path) {
				return nil
			}
			if entry.Type()&os.ModeSymlink != 0 {
				return errors.New("candidate source symlink")
			}
			entryInfo, err := entry.Info()
			if err != nil {
				return err
			}
			if entryInfo.IsDir() {
				return nil
			}
			if !entryInfo.Mode().IsRegular() {
				return errors.New("candidate source is not a regular file")
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			files = append(files, localCandidateSource{kind: source.kind, relPath: rel, id: sourceIDForPath(rel)})
			if len(filters) == 0 && len(files) > MaxCandidateSources {
				return errors.New("candidate source count limit")
			}
			return nil
		})
		if err != nil {
			code := "source.unavailable"
			if strings.Contains(err.Error(), "entry limit") || strings.Contains(err.Error(), "source count limit") {
				code = "source.limit_reached"
			}
			return nil, &sourceScanFailure{code: code}
		}
	}
	sort.SliceStable(files, func(i, j int) bool {
		if files[i].kind != files[j].kind {
			return files[i].kind < files[j].kind
		}
		return files[i].relPath < files[j].relPath
	})
	for i := 1; i < len(files); i++ {
		if files[i-1].kind == files[i].kind && files[i-1].id == files[i].id {
			return nil, &sourceScanFailure{code: "source.ambiguous"}
		}
	}

	if len(filters) > 0 {
		selected := make([]localCandidateSource, 0, len(filters))
		for _, filter := range filters {
			var found []localCandidateSource
			for _, file := range files {
				if filter.SourceKind == file.kind && filter.SourceID == file.id {
					found = append(found, file)
				}
			}
			if len(found) != 1 {
				return nil, &sourceScanFailure{code: "source.unavailable"}
			}
			found[0].expectedDigest = filter.SourceSHA256
			selected = append(selected, found[0])
		}
		files = selected
	}
	if len(files) > MaxCandidateSources {
		return nil, &sourceScanFailure{code: "source.limit_reached"}
	}
	totalBytes := 0
	for i := range files {
		if err := ctx.Err(); err != nil {
			return nil, &sourceScanFailure{code: "request.cancelled"}
		}
		content, err := readWithin(root, files[i].relPath, MaxSourceBytes)
		if err != nil {
			return nil, &sourceScanFailure{code: "source.unavailable"}
		}
		totalBytes += len(content)
		if totalBytes > MaxCandidateSourceBytesTotal {
			return nil, &sourceScanFailure{code: "source.byte_limit"}
		}
		files[i].content = content
		files[i].digest = sourceDigest(content)
		if files[i].expectedDigest != "" && files[i].expectedDigest != files[i].digest {
			return nil, &sourceScanFailure{code: "source.digest_mismatch", conflict: true}
		}
	}
	return files, nil
}

func candidateFailureMessage(code string) string {
	switch code {
	case "request.cancelled":
		return "Candidate discovery was cancelled before it completed."
	case "source.limit_reached":
		return "Local source discovery reached a file or entry limit; narrow the source filters and retry."
	case "source.byte_limit":
		return "Selected local source documents exceed the combined byte limit; narrow the source filters."
	case "source.digest_mismatch":
		return "A selected source no longer matches its requested digest."
	default:
		return "A requested local source is unavailable, unsafe, or ambiguous."
	}
}

func mapStepContract(contract StepContract) (apitools.StepContract, bool, bool) {
	inputs, unsupportedInputs := mapFieldSet(contract.Inputs)
	outputs, unsupportedOutputs := mapFieldSet(contract.Outputs)
	return apitools.StepContract{
		Purpose: contract.Purpose, Inputs: inputs, Outputs: outputs,
		Effect: apitools.OperationEffect(contract.Effect),
	}, unsupportedInputs, unsupportedOutputs
}

func mapFieldSet(schema *uws1.ParamSchema) (map[string]apitools.ContractValue, bool) {
	values := make(map[string]apitools.ContractValue)
	unsupported := false
	if schema == nil {
		return values, true
	}
	if schema.Ref != "" || len(schema.AllOf) > 0 || len(schema.OneOf) > 0 || len(schema.AnyOf) > 0 || schema.Format != "" {
		unsupported = true
	}
	if len(schema.Extensions) > 0 {
		if _, ok := schema.Extensions["x-openudon-description"]; ok {
			unsupported = true
		}
		for name := range schema.Extensions {
			if name != "x-openudon-description" {
				unsupported = true
			}
		}
	}
	required := make(map[string]bool, len(schema.Required))
	for _, name := range schema.Required {
		required[name] = true
	}
	names := make([]string, 0, len(schema.Properties))
	for name := range schema.Properties {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		value, unsupportedValue := mapParamSchema(schema.Properties[name], required[name], true)
		values[name] = value
		unsupported = unsupported || unsupportedValue
	}
	return values, unsupported
}

func mapParamSchema(schema *uws1.ParamSchema, required bool, hasRequired bool) (apitools.ContractValue, bool) {
	value := apitools.ContractValue{}
	unsupported := false
	if schema == nil {
		return value, true
	}
	value.Type = schema.Type
	value.Format = schema.Format
	if hasRequired {
		value.Required = candidateBoolPointer(required)
	}
	if schema.Ref != "" || len(schema.AllOf) > 0 || len(schema.OneOf) > 0 || len(schema.AnyOf) > 0 {
		unsupported = true
	}
	for name, raw := range schema.Extensions {
		if name != "x-openudon-description" {
			unsupported = true
			continue
		}
		description, ok := raw.(string)
		if !ok {
			unsupported = true
			continue
		}
		value.Description = description
	}
	if len(schema.Required) > 0 && len(schema.Properties) == 0 {
		unsupported = true
	}
	if len(schema.Properties) > 0 {
		value.Properties = make(map[string]apitools.ContractValue, len(schema.Properties))
		requiredNames := make(map[string]bool, len(schema.Required))
		for _, name := range schema.Required {
			requiredNames[name] = true
		}
		names := make([]string, 0, len(schema.Properties))
		for name := range schema.Properties {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			child, childUnsupported := mapParamSchema(schema.Properties[name], requiredNames[name], true)
			value.Properties[name] = child
			unsupported = unsupported || childUnsupported
		}
	}
	if schema.Items != nil {
		item, itemUnsupported := mapParamSchema(schema.Items, false, false)
		value.Items = &item
		unsupported = unsupported || itemUnsupported
	}
	return value, unsupported
}

func unsupportedContractDiagnostics(dimension string, unsupported bool, rootExtensions []string) []Diagnostic {
	if !unsupported {
		return nil
	}
	diagnostics := []Diagnostic{{
		Code: "contract.schema_unsupported", Severity: "warning",
		Message: "Some declared " + dimension + " schema constructs are not represented by APItools; affected matches are indeterminate.",
	}}
	if len(rootExtensions) > 0 {
		diagnostics = append(diagnostics, Diagnostic{
			Code: "contract.root_extension_unsupported", Severity: "warning",
			Message: unsupportedRootExtensionMessage(dimension, rootExtensions),
		})
	}
	return diagnostics
}

func rootContractExtensionNames(schema *uws1.ParamSchema) []string {
	if schema == nil || len(schema.Extensions) == 0 {
		return nil
	}
	names := make([]string, 0, len(schema.Extensions))
	for name := range schema.Extensions {
		if contractExtensionName.MatchString(name) {
			names = append(names, name)
		} else {
			names = append(names, "invalid x-* extension name")
		}
	}
	sort.Strings(names)
	return names
}

func unsupportedRootExtensionMessage(dimension string, names []string) string {
	if len(names) == 0 {
		return "The " + dimension + " contract root contains an unsupported extension field."
	}
	message := "The " + dimension + " contract root contains unsupported extension field " + names[0] + "."
	if len(names) > 1 {
		message += fmt.Sprintf(" %d additional extension field(s) are unsupported.", len(names)-1)
	}
	return message
}

func rootContractExtensionPointer(dimension string, schema *uws1.ParamSchema) string {
	if schema == nil {
		return ""
	}
	var names []string
	for name := range schema.Extensions {
		if contractExtensionName.MatchString(name) {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return ""
	}
	sort.Strings(names)
	return "/contract/" + dimension + "/" + escapeJSONPointer(names[0])
}

func mapCandidateReport(report apitools.OperationCandidateReport, limit *int) (CandidatesData, []Diagnostic, error) {
	data := CandidatesData{Sources: make([]CandidateSourceReport, 0, len(report.Sources)), Candidates: make([]StepCandidate, 0, len(report.Candidates))}
	for _, source := range report.Sources {
		sourceID := strings.TrimSpace(source.Name)
		if !sourceIDPattern.MatchString(sourceID) || !sourceKindSupported(string(source.Kind)) {
			return CandidatesData{}, nil, fmt.Errorf("unsafe source identity")
		}
		mapped := CandidateSourceReport{
			SourceID: sourceID, SourceKind: string(source.Kind), OperationCount: source.OperationCount,
			Capabilities: mapCandidateCapabilities(source.Capabilities), Diagnostics: mapAPIDiagnostics(source.Diagnostics),
		}
		data.Sources = append(data.Sources, mapped)
	}
	for index, candidate := range report.Candidates {
		mapped, err := mapStepCandidate(index+1, candidate)
		if err != nil {
			return CandidatesData{}, nil, err
		}
		data.Candidates = append(data.Candidates, mapped)
	}
	shortlistLimit := MaxCandidateShortlist
	if limit != nil {
		shortlistLimit = *limit
	}
	var diagnostics []Diagnostic
	if len(data.Candidates) > shortlistLimit {
		data.Candidates = data.Candidates[:shortlistLimit]
		data.Truncated = true
		diagnostics = append(diagnostics, Diagnostic{
			Code: "candidate.result_truncated", Severity: "warning",
			Message: "The requested candidate limit shortened the ranked list; review the truncated flag before selection.",
		})
	}
	return data, diagnostics, nil
}

func mapStepCandidate(rank int, candidate apitools.OperationCandidate) (StepCandidate, error) {
	sourceID := strings.TrimSpace(candidate.Operation.DocumentName)
	if !sourceIDPattern.MatchString(sourceID) || !sourceKindSupported(string(candidate.Source.Kind)) ||
		!safeNativeIdentity(candidate.Source.Selector, 512) || !safeNativeIdentity(candidate.Operation.ID, 256) ||
		(candidate.Operation.OperationID != "" && !safeNativeIdentity(candidate.Operation.OperationID, 256)) ||
		len(candidate.Source.SHA256) != 64 {
		return StepCandidate{}, fmt.Errorf("unsafe operation identity")
	}
	for _, digit := range candidate.Source.SHA256 {
		if !strings.ContainsRune("0123456789abcdef", digit) {
			return StepCandidate{}, fmt.Errorf("invalid operation digest")
		}
	}

	summary := CandidateSummary{
		Description: candidate.Summary.Description,
		Inputs:      make([]CandidateValue, 0, len(candidate.Summary.Inputs)),
		Outputs:     make([]CandidateValue, 0, len(candidate.Summary.Outputs)),
		Evidence:    mapCandidateEvidence(candidate.Summary.Evidence),
		Gaps:        append([]string{}, candidate.Summary.Gaps...),
	}
	for _, value := range candidate.Summary.Inputs {
		summary.Inputs = append(summary.Inputs, mapCandidateValue(value))
	}
	for _, value := range candidate.Summary.Outputs {
		summary.Outputs = append(summary.Outputs, mapCandidateValue(value))
	}
	if summary.Description == "" {
		return StepCandidate{}, fmt.Errorf("missing summary description")
	}
	match := CandidateMatch{
		Score:   candidate.Match.Score,
		Purpose: mapCandidateMatch(candidate.Match.Purpose),
		Inputs:  mapCandidateMatch(candidate.Match.Inputs),
		Outputs: mapCandidateMatch(candidate.Match.Outputs),
		Effect:  mapCandidateMatch(candidate.Match.Effect),
	}
	return StepCandidate{
		Rank: rank,
		OperationRef: OperationRef{
			SourceKind: string(candidate.Source.Kind), SourceID: sourceID,
			SourceSHA256:   "sha256:" + candidate.Source.SHA256,
			NativeSelector: candidate.Source.Selector, OperationKey: candidate.Operation.ID,
			OperationID: candidate.Operation.OperationID,
		},
		Summary: summary, Match: match,
		Authentication: mapCandidateAuthentication(candidate),
		Effect:         CandidateEffect{Class: string(candidate.Effect.Class), Evidence: mapCandidateEvidence(candidate.Effect.Evidence), Reasons: append([]string{}, candidate.Effect.Reasons...)},
	}, nil
}

func mapCandidateValue(value apitools.OperationValueSummary) CandidateValue {
	return CandidateValue{
		Name: value.Name, Location: value.Location, Type: value.Type, Format: value.Format,
		Required: value.Required, Description: value.Description, Evidence: mapCandidateEvidence(value.Evidence),
	}
}

func mapCandidateEvidence(values []apitools.OperationEvidence) []CandidateEvidence {
	out := make([]CandidateEvidence, 0, len(values))
	for _, value := range values {
		out = append(out, CandidateEvidence{Kind: value.Kind, Reference: value.Reference})
	}
	return out
}

func mapCandidateMatch(value apitools.ContractDimensionMatch) CandidateCompatibility {
	return CandidateCompatibility{
		Status: string(value.Status), Score: value.Score,
		Evidence:  mapCandidateEvidence(value.Evidence),
		Reasons:   append([]string{}, value.Reasons...),
		Missing:   append([]string{}, value.Missing...),
		Conflicts: append([]string{}, value.Conflicts...),
		Gaps:      append([]string{}, value.Gaps...),
	}
}

func mapCandidateCapabilities(values []apitools.OperationCapability) []CandidateSourceCapability {
	out := make([]CandidateSourceCapability, 0, len(values))
	for _, value := range values {
		out = append(out, CandidateSourceCapability{
			Dimension: value.Dimension, Status: string(value.Status),
			Evidence: mapCandidateEvidence(value.Evidence), Gaps: append([]string{}, value.Gaps...),
		})
	}
	return out
}

func mapCandidateAuthentication(candidate apitools.OperationCandidate) CandidateAuthentication {
	capability, found := candidateCapability(candidate.Capabilities, "auth")
	sets := candidate.Operation.SecurityRequirementSets
	if !found || capability.Status != apitools.OperationCapabilitySupported || len(sets) == 0 {
		return CandidateAuthentication{Status: "unknown"}
	}
	out := CandidateAuthentication{Status: "known", Alternatives: make([]CandidateAuthAlternative, 0, len(sets))}
	for _, set := range sets {
		alternative := CandidateAuthAlternative{Requirements: make([]CandidateAuthRequirement, 0, len(set.Requirements))}
		for _, requirement := range set.Requirements {
			if strings.TrimSpace(requirement.Name) == "" || utf8.RuneCountInString(requirement.Name) > 128 ||
				utf8.RuneCountInString(requirement.Type) > 64 || utf8.RuneCountInString(requirement.Scheme) > 64 ||
				utf8.RuneCountInString(requirement.ParameterName) > 512 {
				return CandidateAuthentication{Status: "unknown"}
			}
			location := requirement.In
			if location != "header" && location != "query" && location != "cookie" {
				location = "other"
			}
			flows := append([]string(nil), requirement.Flows...)
			for _, flow := range requirement.OAuthFlows {
				flows = append(flows, flow.Name)
			}
			flows = candidateUniqueSortedStrings(flows)
			slot := credentialSlotName(requirement)
			if slot == "" {
				return CandidateAuthentication{Status: "unknown"}
			}
			alternative.Requirements = append(alternative.Requirements, CandidateAuthRequirement{
				Name: requirement.Name, Type: requirement.Type, Scheme: requirement.Scheme,
				Location: location, ParameterName: requirement.ParameterName, CredentialSlot: slot,
				Flows: flows, Scopes: candidateUniqueSortedStrings(requirement.Scopes),
			})
		}
		out.Alternatives = append(out.Alternatives, alternative)
	}
	return out
}

func credentialSlotName(requirement apitools.SecuritySummary) string {
	slot := apitools.SecurityCredentialFieldName(requirement)
	if !symbol(slot) {
		slot = strings.ToLower(strings.ReplaceAll(requirement.Name, "-", "_"))
	}
	if !symbol(slot) {
		return ""
	}
	return slot
}

func candidateCapability(values []apitools.OperationCapability, dimension string) (apitools.OperationCapability, bool) {
	for _, value := range values {
		if value.Dimension == dimension {
			return value, true
		}
	}
	return apitools.OperationCapability{}, false
}

func mapAPIDiagnostics(values []apitools.Diagnostic) []Diagnostic {
	out := make([]Diagnostic, 0, len(values))
	for _, value := range values {
		code := value.Code
		if !candidateDiagnosticCode.MatchString(code) {
			code = "source.metadata"
		}
		severity := value.Severity
		if severity != "error" && severity != "warning" && severity != "info" {
			severity = "warning"
		}
		message := safeFindingMessage(value.Message)
		if message == "" {
			message = "A source metadata diagnostic was omitted because it was unsafe to display."
		}
		remediation := safeFindingMessage(value.Remediation)
		out = append(out, Diagnostic{Code: code, Severity: severity, Message: message, Remediation: remediation})
	}
	sortCandidateDiagnostics(out)
	return out
}

func hasErrorDiagnostics(values []Diagnostic) bool {
	for _, value := range values {
		if value.Severity == "error" {
			return true
		}
	}
	return false
}

func candidatesComplete(data CandidatesData, diagnostics []Diagnostic) CandidatesOutcome {
	diagnostics = boundedCandidateDiagnostics(diagnostics)
	return CandidatesOutcome{Result: CandidatesWireResult{
		Version: WireVersion, Kind: "result", Command: CandidatesCommand,
		Status: "completed", Diagnostics: diagnostics, Result: &data,
	}}
}

func candidatesFailure(status, code, message string, exitCode int) CandidatesOutcome {
	return candidatesFailureWithDiagnostics(status, []Diagnostic{{Code: code, Severity: "error", Message: message}}, exitCode)
}

func candidatesFailureWithDiagnostics(status string, diagnostics []Diagnostic, exitCode int) CandidatesOutcome {
	diagnostics = boundedCandidateDiagnostics(diagnostics)
	return CandidatesOutcome{Result: CandidatesWireResult{
		Version: WireVersion, Kind: "result", Command: CandidatesCommand,
		Status: status, Diagnostics: diagnostics,
	}, ExitCode: exitCode}
}

func boundedCandidateDiagnostics(values []Diagnostic) []Diagnostic {
	values = append([]Diagnostic(nil), values...)
	sortCandidateDiagnostics(values)
	if len(values) > MaxCandidateDiagnostics {
		values = values[:MaxCandidateDiagnostics]
	}
	if values == nil {
		return []Diagnostic{}
	}
	return values
}

func sortCandidateDiagnostics(values []Diagnostic) {
	sort.SliceStable(values, func(i, j int) bool {
		if values[i].Code != values[j].Code {
			return values[i].Code < values[j].Code
		}
		if values[i].Severity != values[j].Severity {
			return values[i].Severity < values[j].Severity
		}
		return values[i].Message < values[j].Message
	})
}

func applyUnsupportedDimensions(data *CandidatesData, inputs, outputs bool) {
	for i := range data.Candidates {
		if inputs {
			markCandidateMatchIndeterminate(&data.Candidates[i].Match.Inputs, "Some declared input schema constructs are not represented by APItools.")
		}
		if outputs {
			markCandidateMatchIndeterminate(&data.Candidates[i].Match.Outputs, "Some declared output schema constructs are not represented by APItools.")
		}
	}
}

func markCandidateMatchIndeterminate(match *CandidateCompatibility, reason string) {
	if match.Status != "incompatible" {
		match.Status = "indeterminate"
	}
	match.Gaps = append(match.Gaps, reason)
	match.Gaps = candidateUniqueSortedStrings(match.Gaps)
}

func candidateBoolPointer(value bool) *bool { return &value }

func candidateUniqueSortedStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	out := append([]string(nil), values...)
	sort.Strings(out)
	unique := out[:1]
	for _, value := range out[1:] {
		if value != unique[len(unique)-1] {
			unique = append(unique, value)
		}
	}
	return unique
}
