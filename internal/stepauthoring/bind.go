package stepauthoring

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/OpenUdon/apitools"
	"github.com/OpenUdon/openudon/internal/authoring"
	"github.com/OpenUdon/openudon/internal/evidencefile"
	"github.com/OpenUdon/openudon/internal/icot/artifactwriter"
	"github.com/OpenUdon/openudon/internal/packageartifacts"
	"github.com/OpenUdon/openudon/internal/workflowintent"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

const BindCommand = "step.bind"

var (
	bindOutputReference = regexp.MustCompile(`^received_body(?:\.[A-Za-z][A-Za-z0-9_.-]*|\[[0-9]+\](?:\.[A-Za-z][A-Za-z0-9_.-]*)*)?$`)
	mappingFieldName    = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_.-]{0,255}$`)
)

type IntentRevision struct {
	State  string `json:"state"`
	SHA256 string `json:"sha256,omitempty"`
}

type BindScaffold struct {
	Workflow BindScaffoldWorkflow `json:"workflow"`
	Inputs   []BindField          `json:"inputs,omitempty"`
	Outputs  []BindOutput         `json:"outputs,omitempty"`
}

type BindScaffoldWorkflow struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type BindField struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Format      string `json:"format,omitempty"`
	Required    bool   `json:"required,omitempty"`
	Description string `json:"description,omitempty"`
}

type BindOutput struct {
	Name        string `json:"name"`
	From        string `json:"from"`
	Description string `json:"description,omitempty"`
}

type BindRequest struct {
	Version                   string            `json:"version"`
	Kind                      string            `json:"kind"`
	Command                   string            `json:"command"`
	StepID                    string            `json:"step_id"`
	Contract                  StepContract      `json:"contract"`
	OperationRef              OperationRef      `json:"operation_ref"`
	RequestMappings           map[string]string `json:"request_mappings"`
	OutputMappings            map[string]string `json:"output_mappings"`
	AuthenticationAlternative *int              `json:"authentication_alternative,omitempty"`
	CredentialBindings        map[string]string `json:"credential_bindings,omitempty"`
	DependsOn                 []string          `json:"depends_on,omitempty"`
	IntentRevision            IntentRevision    `json:"intent_revision"`
	Scaffold                  *BindScaffold     `json:"scaffold,omitempty"`
}

type BindResult struct {
	StepID       string `json:"step_id"`
	WriteOutcome string `json:"write_outcome"`
	IntentSHA256 string `json:"intent_sha256,omitempty"`
}

type BindWireResult struct {
	Version     string       `json:"version"`
	Kind        string       `json:"kind"`
	Command     string       `json:"command"`
	Status      string       `json:"status"`
	Diagnostics []Diagnostic `json:"diagnostics"`
	Result      *BindResult  `json:"result,omitempty"`
}

type BindOutcome struct {
	Result   BindWireResult
	ExitCode int
}

// Bind validates one explicit step contract and writes only workflows/intent.hcl.
// All rejection paths are read-only; the shared artifact writer performs the
// final optimistic-revision check and rollback-capable replacement.
func Bind(ctx context.Context, exampleDir string, request BindRequest) BindOutcome {
	if request.Version != "" && request.Version != WireVersion {
		return bindFailure("failed", "request.unsupported_version", "The step-bind request version is not supported.", 2)
	}
	if request.Version == "" || request.Kind != "request" || request.Command != BindCommand || !symbol(request.StepID) {
		return bindFailure("failed", "request.invalid", "The step-bind request is invalid.", 2)
	}
	checkRequest := CheckRequest{
		Version: WireVersion, Kind: "request", Command: CheckCommand,
		StepID: request.StepID, Contract: request.Contract,
		OperationRef: request.OperationRef, IntentSHA256: "sha256:" + strings.Repeat("0", 64),
	}
	if err := validateRequest(checkRequest); err != nil || !validateBindMappings(request) {
		return bindFailure("failed", "request.invalid", "The step-bind request is invalid.", 2)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return bindFailure("blocked", "request.cancelled", "The step-bind request was cancelled before it started.", 4)
	}
	root, err := resolveExampleRoot(exampleDir)
	if err != nil {
		return bindFailure("blocked", "example.invalid", "The selected example is unavailable or unsafe.", 4)
	}

	intentPath := filepath.Join(root, filepath.FromSlash(defaultIntentPath))
	intentBytes, readErr := readWithin(root, defaultIntentPath, evidencefile.DefaultMaxBytes)
	intentPresent := readErr == nil
	var intent *workflowintent.Intent
	if intentPresent {
		currentDigest := sourceDigest(intentBytes)
		if request.IntentRevision.State != "present" || request.IntentRevision.SHA256 != currentDigest {
			return bindFailure("conflict", "intent.stale", "The workflow intent changed after it was selected.", 3)
		}
		intent, err = workflowintent.ParseIntent(intentBytes, defaultIntentPath)
		if err != nil {
			return bindFailure("blocked", "intent.invalid", "The workflow intent could not be validated safely.", 4)
		}
	} else if errors.Is(readErr, fs.ErrNotExist) {
		if request.IntentRevision.State != "missing" || request.IntentRevision.SHA256 != "" || request.Scaffold == nil {
			return bindFailure("conflict", "intent.missing", "The selected workflow intent is missing or no longer matches the request.", 3)
		}
		intent, err = intentFromScaffold(request.Scaffold)
		if err != nil {
			return bindFailure("failed", "scaffold.invalid", "The explicit workflow scaffold is invalid.", 2)
		}
	} else {
		return bindFailure("blocked", "intent.unavailable", "The workflow intent is unavailable or unsafe to read.", 4)
	}

	steps := findSteps(intent.Steps, request.StepID)
	if len(steps) > 1 {
		return bindFailure("blocked", "step.ambiguous", "The selected step name is not unique in the workflow.", 4)
	}

	sourceRel, sourceBytes, sourceErr := locateSource(root, request.OperationRef)
	if sourceErr != nil {
		if errors.Is(sourceErr, errSourceChanged) {
			return bindFailure("conflict", "source.stale", "The selected API source changed after operation selection.", 3)
		}
		return bindFailure("blocked", "source.unavailable", "The selected API source is unavailable or unsafe to inspect.", 4)
	}
	candidate, err := resolveOperationCandidate(ctx, sourceRel, sourceBytes, request.OperationRef, request.Contract)
	if err != nil {
		if ctx.Err() != nil {
			return bindFailure("blocked", "request.cancelled", "The step-bind request was cancelled during source inspection.", 4)
		}
		if errors.Is(err, errOperationMissing) || errors.Is(err, errOperationAmbiguous) {
			return bindFailure("conflict", "operation.stale", "The selected operation no longer matches the source.", 3)
		}
		return bindFailure("blocked", "source.unsupported", "The selected API source cannot be validated safely.", 4)
	}
	operation := candidate.Operation
	if err := ctx.Err(); err != nil {
		return bindFailure("blocked", "request.cancelled", "The step-bind request was cancelled before effect and authentication checks completed.", 4)
	}
	if mapCandidateAuthentication(candidate).Status != "known" {
		return bindFailure("needs_input", "authentication.unknown", "The selected operation has no complete source authentication alternative to bind.", 4)
	}
	_, unsupportedInputs, unsupportedOutputs := mapStepContract(request.Contract)
	inputStatus := candidateContractStatus(candidate.Match.Inputs, !unsupportedInputs, true)
	outputStatus := candidateContractStatus(candidate.Match.Outputs, !unsupportedOutputs, true)
	if inputStatus != "compatible" || outputStatus != "compatible" {
		code := "mapping.incomplete"
		message := "Source metadata cannot establish complete input and output compatibility with the step contract."
		if inputStatus == "incompatible" || outputStatus == "incompatible" {
			code = "mapping.contract_mismatch"
			message = "The selected operation's input or output types conflict with the step contract."
		}
		return bindFailure("needs_input", code, message, 4)
	}
	effectCheck := newCheckAccumulator()
	effectStatus := checkEffect(request.Contract, candidate, true, effectCheck)
	if effectStatus != "pass" {
		code, message := "effect.unknown", "The selected operation's effect is not sufficiently confirmed for this step contract."
		if effectStatus == "fail" {
			code = "effect.conflict"
			message = "The selected operation's classified effect conflicts with the step contract."
		}
		return bindFailure("needs_input", code, message, 4)
	}
	if err := validateDependencies(intent, request.StepID, request.DependsOn, request.RequestMappings); err != nil {
		return bindFailure("needs_input", "dependency.invalid", "Step dependencies must name unique existing steps and cover referenced outputs.", 4)
	}
	if !validateInputReferences(intent, request.RequestMappings) {
		return bindFailure("needs_input", "mapping.input_reference", "Input mappings must refer only to declared workflow inputs.", 4)
	}

	step, err := bindStep(request, sourceRel, operation)
	if err != nil {
		return bindFailure("needs_input", "binding.incomplete", "The selected operation needs complete request, output, or authentication mappings before it can be bound.", 4)
	}
	check := newCheckAccumulator()
	checkRequiredMappings(step, request.Contract, operation, check)
	for _, item := range check.items {
		if item.Status != "pass" {
			return bindFailure("needs_input", "mapping.incomplete", "Required operation inputs need explicit symbolic mappings before the step can be bound.", 4)
		}
	}
	if err := validateOutputMappings(request, operation); err != nil {
		return bindFailure("needs_input", "mapping.outputs", "Required output mappings are absent or do not match the selected response summary.", 4)
	}
	trialIntent, err := intent.Clone()
	if err != nil {
		return bindFailure("blocked", "intent.invalid", "The workflow intent could not be reviewed safely.", 4)
	}
	if !replaceIntentStep(trialIntent.Steps, request.StepID, step) {
		trialIntent.Steps = append(trialIntent.Steps, step)
	}
	if dependencyCycleFrom(trialIntent.Steps, request.StepID) {
		return bindFailure("needs_input", "dependency.cycle", "The selected step would create or retain a workflow dependency cycle.", 4)
	}
	outputChecks := newCheckAccumulator()
	checkOutputs(trialIntent, step, request.Contract, operation, outputChecks)
	for _, item := range outputChecks.items {
		if item.Status == "fail" {
			return bindFailure("needs_input", "mapping.output_reference", "Existing workflow outputs must remain within the selected step contract.", 4)
		}
	}

	var nextBytes []byte
	if !intentPresent {
		intent.Steps = append(intent.Steps, step)
		rendered, renderErr := workflowintent.RenderIntentHCL(intent)
		if renderErr != nil {
			return bindFailure("failed", "intent.render_failed", "The explicit scaffold and selected step could not be rendered as valid intent.", 1)
		}
		nextBytes = []byte(rendered)
	} else {
		parsed, parseDiags := hclwrite.ParseConfig(intentBytes, defaultIntentPath, hcl.InitialPos)
		if parseDiags.HasErrors() {
			return bindFailure("blocked", "intent.invalid", "The workflow intent could not be edited safely.", 4)
		}
		blocks := findHCLSteps(parsed.Body(), request.StepID)
		if len(blocks) == 1 {
			if hasNestedSteps(blocks[0].Body()) {
				return bindFailure("needs_input", "step.composite_unsupported", "A composite step cannot be replaced by a single API operation.", 4)
			}
			writeStepFields(blocks[0].Body(), step)
		} else {
			writeStepFields(parsed.Body().AppendNewBlock("step", []string{request.StepID}).Body(), step)
		}
		nextBytes = parsed.Bytes()
	}
	if _, err := workflowintent.ParseIntent(nextBytes, defaultIntentPath); err != nil {
		return bindFailure("needs_input", "intent.invalid_after_bind", "The proposed step does not satisfy the workflow intent contract.", 4)
	}
	if bytesEqual(intentBytes, nextBytes) {
		digest := sourceDigest(intentBytes)
		return bindSuccess(request.StepID, "unchanged", digest)
	}
	if err := ctx.Err(); err != nil {
		return bindFailure("blocked", "request.cancelled", "The step-bind request was cancelled before the write.", 4)
	}

	file := artifactwriter.GeneratedFile{
		Path: intentPath, Content: string(nextBytes), Action: "write",
		Reason:         "bind one explicitly confirmed step to a source operation",
		AllowOverwrite: intentPresent,
	}
	if intentPresent {
		file.ExpectedCurrentSHA256 = sourceDigest(intentBytes)
	}
	beforeReplace := bindBeforeReplace(root, sourceRel, request.OperationRef.SourceSHA256, intentPath, intentPresent)
	_, err = artifactwriter.CommitChecked(artifactwriter.Prepared{ExampleRoot: root, Files: []artifactwriter.GeneratedFile{file}}, false, beforeReplace)
	if err != nil {
		if errors.Is(err, errSourceChanged) {
			return bindFailure("conflict", "source.stale", "The selected API source changed before the intent could be replaced.", 3)
		}
		var transactionErr *artifactwriter.TransactionError
		if errors.As(err, &transactionErr) {
			return bindFailureWithResult("failed", "write.indeterminate", "The intent write outcome is indeterminate; inspect the exact file revision before retrying.", 1, &BindResult{StepID: request.StepID, WriteOutcome: "indeterminate"})
		}
		if current, currentErr := readWithin(root, defaultIntentPath, evidencefile.DefaultMaxBytes); currentErr == nil && sourceDigest(current) != sourceDigest(intentBytes) {
			return bindFailure("conflict", "intent.stale", "The workflow intent changed before the atomic write.", 3)
		}
		return bindFailure("failed", "write.failed", "The workflow intent was not changed because the atomic write failed.", 1)
	}
	return bindSuccess(request.StepID, "written", sourceDigest(nextBytes))
}

var (
	errSourceChanged      = errors.New("selected source digest changed")
	errOperationMissing   = errors.New("selected operation is missing")
	errOperationAmbiguous = errors.New("selected operation is ambiguous")
	errSourceUnsupported  = errors.New("selected source cannot be inspected safely")
)

func verifySourceRevision(root, relative, expectedDigest string) error {
	data, err := readWithin(root, relative, MaxSourceBytes)
	if err != nil || sourceDigest(data) != expectedDigest {
		return errSourceChanged
	}
	return nil
}

func bindBeforeReplace(root, sourceRel, expectedSourceDigest, intentPath string, intentPresent bool) func() error {
	return func() error {
		if err := verifySourceRevision(root, sourceRel, expectedSourceDigest); err != nil {
			return errSourceChanged
		}
		if intentPresent {
			return nil
		}
		_, statErr := os.Lstat(intentPath)
		if errors.Is(statErr, os.ErrNotExist) {
			return nil
		}
		if statErr != nil {
			return statErr
		}
		return fmt.Errorf("intent appeared after the missing revision was checked")
	}
}

func validateBindMappings(request BindRequest) bool {
	if request.RequestMappings == nil || request.OutputMappings == nil {
		return false
	}
	if request.IntentRevision.State != "present" && request.IntentRevision.State != "missing" {
		return false
	}
	if request.IntentRevision.State == "present" && !sha256Pattern.MatchString(request.IntentRevision.SHA256) {
		return false
	}
	if request.IntentRevision.State == "missing" && request.IntentRevision.SHA256 != "" {
		return false
	}
	if len(request.RequestMappings) > 64 || len(request.OutputMappings) > 64 || len(request.CredentialBindings) > 16 || len(request.DependsOn) > 64 {
		return false
	}
	if !safeDescription(request.Contract.Purpose, 2048) {
		return false
	}
	if len(request.Contract.AccountConstraints) > 16 || len(request.Contract.DestinationConstraints) > 16 {
		return false
	}
	for _, constraint := range append(append([]string(nil), request.Contract.AccountConstraints...), request.Contract.DestinationConstraints...) {
		if !safeDescription(constraint, 256) {
			return false
		}
	}
	for key, value := range request.RequestMappings {
		if !mappingFieldName.MatchString(key) || strings.Contains(key, "..") || !safeMapping(value) {
			return false
		}
	}
	for key, value := range request.OutputMappings {
		if !mappingFieldName.MatchString(key) || strings.Contains(key, "..") || !bindOutputReference.MatchString(value) {
			return false
		}
	}
	for key, value := range request.CredentialBindings {
		if !symbol(key) || !symbol(value) || value == "none" || value == "clear" {
			return false
		}
	}
	seen := map[string]bool{}
	for _, dependency := range request.DependsOn {
		if !symbol(dependency) || dependency == request.StepID || seen[dependency] {
			return false
		}
		seen[dependency] = true
	}
	if request.Scaffold != nil && (request.IntentRevision.State != "missing" || !validateScaffold(request.Scaffold)) {
		return false
	}
	return true
}

func safeMapping(value string) bool {
	if value == "" || len(value) > 512 || !utf8.ValidString(value) || authoring.ContainsLikelyCredentialValue([]byte(value)) {
		return false
	}
	for _, r := range value {
		if r < 0x20 || r == 0x7f {
			return false
		}
	}
	return true
}

func safeDescription(value string, max int) bool {
	if strings.TrimSpace(value) == "" || len(value) > max || !utf8.ValidString(value) || authoring.ContainsLikelyCredentialValue([]byte(value)) {
		return false
	}
	for _, r := range value {
		if r < 0x20 && r != '\n' && r != '\t' || r == 0x7f {
			return false
		}
	}
	return true
}

func validateScaffold(scaffold *BindScaffold) bool {
	if scaffold == nil || !symbol(scaffold.Workflow.Name) || !safeDescription(scaffold.Workflow.Description, 2048) || len(scaffold.Inputs) > 64 || len(scaffold.Outputs) > 64 {
		return false
	}
	seen := map[string]bool{}
	for _, input := range scaffold.Inputs {
		if !symbol(input.Name) || seen[input.Name] || !validFieldType(input.Type) || len(input.Description) > 1024 || (input.Description != "" && !safeDescription(input.Description, 1024)) || len(input.Format) > 64 {
			return false
		}
		seen[input.Name] = true
	}
	seen = map[string]bool{}
	for _, output := range scaffold.Outputs {
		if !symbol(output.Name) || seen[output.Name] || output.From == "" || len(output.From) > 512 || len(output.Description) > 1024 || (output.Description != "" && !safeDescription(output.Description, 1024)) || !safeMapping(output.From) {
			return false
		}
		seen[output.Name] = true
	}
	return true
}

func validFieldType(value string) bool {
	switch value {
	case "string", "integer", "number", "boolean", "object", "array":
		return true
	default:
		return false
	}
}

func intentFromScaffold(scaffold *BindScaffold) (*workflowintent.Intent, error) {
	if !validateScaffold(scaffold) {
		return nil, fmt.Errorf("invalid scaffold")
	}
	intent := &workflowintent.Intent{Workflow: &workflowintent.WorkflowMeta{
		Name: scaffold.Workflow.Name, Description: scaffold.Workflow.Description,
	}}
	for _, input := range scaffold.Inputs {
		intent.Inputs = append(intent.Inputs, &workflowintent.Input{
			Name: input.Name, Type: input.Type, Description: input.Description, Required: input.Required,
		})
	}
	for _, output := range scaffold.Outputs {
		intent.Outputs = append(intent.Outputs, &workflowintent.Output{
			Name: output.Name, From: output.From, Description: output.Description,
		})
	}
	return intent, nil
}

func locateSource(root string, ref OperationRef) (string, []byte, error) {
	directories := sourceDirectories(ref.SourceKind)
	if len(directories) == 0 {
		return "", nil, fmt.Errorf("unsupported source kind")
	}
	var identityPath string
	var matchedPath string
	var matchedBytes []byte
	identityCount := 0
	visited := 0
	for _, directory := range directories {
		start := filepath.Join(root, directory)
		info, err := os.Lstat(start)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return "", nil, fmt.Errorf("source directory unavailable")
		}
		err = filepath.WalkDir(start, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			visited++
			if visited > 4096 {
				return fmt.Errorf("source inventory exceeds bound")
			}
			if packageartifacts.IsAdvisorySecuritySidecarPath(path) {
				return nil
			}
			if entry.Type()&os.ModeSymlink != 0 {
				if entry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if entry.IsDir() || !entry.Type().IsRegular() {
				return nil
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			if sourceIDForPath(rel) != ref.SourceID {
				return nil
			}
			identityPath = rel
			identityCount++
			data, err := readWithin(root, rel, MaxSourceBytes)
			if err != nil {
				return err
			}
			if sourceDigest(data) == ref.SourceSHA256 {
				matchedPath, matchedBytes = rel, data
			}
			return nil
		})
		if err != nil {
			return "", nil, err
		}
	}
	if identityCount == 0 {
		return "", nil, fmt.Errorf("source identity unavailable")
	}
	if matchedPath == "" {
		if identityPath != "" {
			return "", nil, errSourceChanged
		}
		return "", nil, fmt.Errorf("source identity unavailable")
	}
	if identityCount != 1 || matchedPath == "" {
		return "", nil, fmt.Errorf("source identity is ambiguous")
	}
	return matchedPath, matchedBytes, nil
}

func sourceDirectories(kind string) []string {
	switch kind {
	case "openapi", "aws-smithy", "asyncapi", "graphql", "openrpc", "grpc-protobuf", "odata":
		return []string{kind}
	case "google-discovery":
		return []string{"google-discovery", "discovery"}
	default:
		return nil
	}
}

func bindStep(request BindRequest, source string, operation apitools.OperationSummary) (*workflowintent.Step, error) {
	step := &workflowintent.Step{
		Name: request.StepID, Type: "http", Do: request.Contract.Purpose,
		Source: source, Operation: operation.OperationID, With: map[string]string{},
		DependsOn: append([]string(nil), request.DependsOn...),
	}
	if step.Operation == "" {
		step.Operation = operation.ID
	}
	for key, value := range request.RequestMappings {
		step.With[key] = value
	}
	sets := operation.SecurityRequirementSets
	if len(sets) == 0 {
		if request.AuthenticationAlternative != nil || len(request.CredentialBindings) > 0 {
			return nil, fmt.Errorf("source has no explicit authentication alternatives")
		}
	} else {
		alternative := 0
		if request.AuthenticationAlternative != nil {
			alternative = *request.AuthenticationAlternative
		} else if len(sets) > 1 {
			return nil, fmt.Errorf("authentication alternative must be selected")
		}
		if alternative < 0 || alternative >= len(sets) {
			return nil, fmt.Errorf("authentication alternative is outside source bounds")
		}
		selected := sets[alternative]
		allowed := map[string]bool{}
		if len(selected.Requirements) == 0 && len(request.CredentialBindings) > 0 {
			return nil, fmt.Errorf("anonymous alternative cannot accept credentials")
		}
		for _, requirement := range selected.Requirements {
			field := requirement.ParameterName
			if field == "" && (strings.EqualFold(requirement.Scheme, "bearer") || strings.EqualFold(requirement.Scheme, "basic")) {
				field = "Authorization"
			}
			if field == "" {
				field = requirement.Name
			}
			slot := credentialSlotName(requirement)
			if symbol(requirement.Name) {
				allowed[requirement.Name] = true
			}
			if symbol(slot) {
				allowed[slot] = true
			}
			nameBinding := request.CredentialBindings[requirement.Name]
			slotBinding := request.CredentialBindings[slot]
			if nameBinding != "" && slotBinding != "" && nameBinding != slotBinding {
				return nil, fmt.Errorf("credential aliases conflict")
			}
			binding := nameBinding
			if binding == "" {
				binding = slotBinding
			}
			if binding != "" {
				value := "credentials." + binding
				if existing := step.With[field]; existing != "" && existing != value {
					return nil, fmt.Errorf("credential mapping conflicts")
				}
				step.With[field] = value
			} else if !strings.HasPrefix(step.With[field], "credentials.") {
				return nil, fmt.Errorf("required authentication mapping is missing")
			}
		}
		for key := range request.CredentialBindings {
			if !allowed[key] {
				return nil, fmt.Errorf("credential mapping does not belong to selected alternative")
			}
		}
	}
	if len(step.With) == 0 {
		step.With = nil
	}
	return step, nil
}

func validateOutputMappings(request BindRequest, operation apitools.OperationSummary) error {
	declared := request.Contract.Outputs
	if len(declared.Properties) == 0 {
		if len(request.OutputMappings) != 0 {
			return fmt.Errorf("no outputs are declared")
		}
		return nil
	}
	if operation.ResponseBody == nil || len(operation.ResponseBody.Fields) == 0 {
		return fmt.Errorf("operation response has no inspectable output fields")
	}
	available := map[string]bool{}
	for _, field := range operation.ResponseBody.Fields {
		available[strings.TrimPrefix(strings.TrimSpace(field.Path), "$")] = true
	}
	for name, target := range request.OutputMappings {
		if !contractHasOutputPath(declared.Properties, name) {
			return fmt.Errorf("output mapping is outside the contract")
		}
		path := strings.TrimPrefix(target, "received_body.")
		if path == target {
			if target != "received_body" {
				return fmt.Errorf("output mapping is not a response reference")
			}
			continue
		}
		if !available[path] {
			return fmt.Errorf("output mapping is absent from response summary")
		}
	}
	for name := range contractOutputPaths(declared.Properties) {
		if schemaRequiredPath(declared, name) && request.OutputMappings[name] == "" {
			return fmt.Errorf("required output mapping is missing")
		}
	}
	return nil
}

func validateDependencies(intent *workflowintent.Intent, stepID string, dependsOn []string, mappings map[string]string) error {
	counts := map[string]int{}
	var walk func([]*workflowintent.Step)
	walk = func(steps []*workflowintent.Step) {
		for _, step := range steps {
			if step == nil {
				continue
			}
			counts[step.Name]++
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
	if counts[stepID] > 1 {
		return fmt.Errorf("duplicate target step")
	}
	declared := map[string]bool{}
	for _, name := range dependsOn {
		if counts[name] != 1 || name == stepID {
			return fmt.Errorf("dependency is missing or ambiguous")
		}
		declared[name] = true
	}
	refs := map[string]bool{}
	for _, value := range mappings {
		collectStepRefs(value, refs)
	}
	for ref := range refs {
		if ref != stepID && !declared[ref] {
			return fmt.Errorf("mapping references an undeclared dependency")
		}
	}
	return nil
}

func validateInputReferences(intent *workflowintent.Intent, mappings map[string]string) bool {
	values := make([]string, 0, len(mappings))
	for _, value := range mappings {
		values = append(values, value)
	}
	return validateInputReferenceValues(intent, values)
}

func stepInputMappingValues(step *workflowintent.Step) []string {
	if step == nil {
		return nil
	}
	values := make([]string, 0, len(step.With))
	for _, value := range step.With {
		values = append(values, value)
	}
	for _, binding := range step.Binds {
		if binding == nil {
			continue
		}
		if binding.From != "" {
			values = append(values, binding.From)
		}
		for _, value := range binding.Fields {
			values = append(values, value)
		}
	}
	return values
}

func validateInputReferenceValues(intent *workflowintent.Intent, values []string) bool {
	declared := map[string]bool{}
	for _, input := range intent.Inputs {
		if input != nil {
			declared[input.Name] = true
		}
	}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if !strings.HasPrefix(value, "inputs.") {
			continue
		}
		field := strings.TrimPrefix(value, "inputs.")
		if dot := strings.IndexAny(field, ".[\""); dot >= 0 {
			field = field[:dot]
		}
		if !declared[field] {
			return false
		}
	}
	return true
}

func replaceIntentStep(steps []*workflowintent.Step, name string, replacement *workflowintent.Step) bool {
	for index, step := range steps {
		if step == nil {
			continue
		}
		if step.Name == name {
			steps[index] = replacement
			return true
		}
		if replaceIntentStep(step.Steps, name, replacement) {
			return true
		}
		for _, branch := range step.Cases {
			if branch != nil && replaceIntentStep(branch.Steps, name, replacement) {
				return true
			}
		}
		if step.Default != nil && replaceIntentStep(step.Default.Steps, name, replacement) {
			return true
		}
	}
	return false
}

func findHCLSteps(body *hclwrite.Body, name string) []*hclwrite.Block {
	var found []*hclwrite.Block
	var walk func(*hclwrite.Body)
	walk = func(current *hclwrite.Body) {
		for _, block := range current.Blocks() {
			if block.Type() == "step" && len(block.Labels()) == 1 && block.Labels()[0] == name {
				found = append(found, block)
			}
			walk(block.Body())
		}
	}
	walk(body)
	return found
}

func hasNestedSteps(body *hclwrite.Body) bool {
	for _, block := range body.Blocks() {
		if block.Type() == "step" {
			return true
		}
		if hasNestedSteps(block.Body()) {
			return true
		}
	}
	return false
}

func writeStepFields(body *hclwrite.Body, step *workflowintent.Step) {
	body.SetAttributeValue("type", cty.StringVal(step.Type))
	body.SetAttributeValue("do", cty.StringVal(step.Do))
	body.RemoveAttribute("openapi")
	body.RemoveAttribute("provider")
	for _, name := range []string{
		"using", "set", "items", "mode", "batch_size", "authentication_flow",
		"registration_flow", "input_binding", "registration_approval",
		"duplicate_prevention", "on_duplicate", "ambiguous_outcome",
		"cleanup_disposition", "browser_session", "credential_bindings",
	} {
		body.RemoveAttribute(name)
	}
	body.SetAttributeValue("source", cty.StringVal(step.Source))
	body.SetAttributeValue("operation", cty.StringVal(step.Operation))
	if step.With == nil {
		body.RemoveAttribute("with")
	} else {
		values := make(map[string]cty.Value, len(step.With))
		for key, value := range step.With {
			values[key] = cty.StringVal(value)
		}
		body.SetAttributeValue("with", cty.ObjectVal(values))
	}
	if len(step.DependsOn) == 0 {
		body.RemoveAttribute("depends_on")
	} else {
		values := make([]cty.Value, len(step.DependsOn))
		for i, value := range step.DependsOn {
			values[i] = cty.StringVal(value)
		}
		body.SetAttributeValue("depends_on", cty.TupleVal(values))
	}
	for _, block := range body.Blocks() {
		if block.Type() == "bind" {
			body.RemoveBlock(block)
		}
	}
}

func bytesEqual(left, right []byte) bool {
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

func bindSuccess(stepID, outcome, digest string) BindOutcome {
	return BindOutcome{Result: BindWireResult{
		Version: WireVersion, Kind: "result", Command: BindCommand, Status: "completed",
		Diagnostics: []Diagnostic{}, Result: &BindResult{StepID: stepID, WriteOutcome: outcome, IntentSHA256: digest},
	}}
}

func bindFailure(status, code, message string, exitCode int) BindOutcome {
	return bindFailureWithResult(status, code, message, exitCode, nil)
}

func bindFailureWithResult(status, code, message string, exitCode int, result *BindResult) BindOutcome {
	return BindOutcome{ExitCode: exitCode, Result: BindWireResult{
		Version: WireVersion, Kind: "result", Command: BindCommand, Status: status,
		Diagnostics: []Diagnostic{{Code: code, Severity: "error", Message: message}}, Result: result,
	}}
}
