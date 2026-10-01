package stepauthoring

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"path/filepath"

	"github.com/OpenUdon/openudon/internal/artifactwriter"
	"github.com/OpenUdon/openudon/internal/authoring"
	"github.com/OpenUdon/openudon/internal/evidencefile"
	"github.com/OpenUdon/openudon/internal/uwsexec"
	"github.com/OpenUdon/openudon/internal/workflowintent"
	"github.com/OpenUdon/uws/uws1"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

const PendingWireVersion = "openudon.step-pending.v1"
const PendingCommand = "step.pending"

type PendingRequest struct {
	Version        string         `json:"version"`
	Kind           string         `json:"kind"`
	Command        string         `json:"command"`
	StepID         string         `json:"step_id"`
	Contract       StepContract   `json:"contract"`
	DependsOn      []string       `json:"depends_on,omitempty"`
	IntentRevision IntentRevision `json:"intent_revision"`
	Scaffold       *BindScaffold  `json:"scaffold,omitempty"`
}

func pendingContract(contract StepContract) *uws1.PendingStep {
	return &uws1.PendingStep{Purpose: contract.Purpose, Inputs: contract.Inputs, Outputs: contract.Outputs, Effect: uws1.OperationEffect(contract.Effect)}
}

// Pending authors one unresolved contract, never a guessed operation. Resolve
// it using step bind with the same contract and the current intent revision.
func Pending(ctx context.Context, exampleDir string, request PendingRequest) (outcome BindOutcome) {
	defer func() { outcome.Result.Version = PendingWireVersion; outcome.Result.Command = PendingCommand }()
	fail := func(status, code, message string, exit int) BindOutcome {
		return bindFailure(status, code, message, exit)
	}
	encoded, err := json.Marshal(request)
	if err != nil || len(encoded) > MaxRequestBytes || authoring.ContainsLikelyCredentialValue(encoded) || request.Version != PendingWireVersion || request.Kind != "request" || request.Command != PendingCommand || !symbol(request.StepID) || !symbol(request.Contract.ID) || validateStepContract(request.Contract) != nil || len(request.DependsOn) > 64 {
		return fail("failed", "request.invalid", "The pending-step request or field-set contract is invalid.", 2)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if ctx.Err() != nil {
		return fail("blocked", "request.cancelled", "Pending authoring was cancelled.", 4)
	}
	root, err := resolveExampleRoot(exampleDir)
	if err != nil {
		return fail("blocked", "example.invalid", "The selected package is unavailable or unsafe.", 4)
	}
	versionCheck := func() error {
		version, e := uwsexec.DeclaredVersion(filepath.Join(root, "workflows/workflow.hcl"), filepath.Join(root, "workflows/workflow.uws.yaml"))
		if e != nil {
			return e
		}
		if version != "" && version != "1.12.0" {
			return errors.New("pending contract requires an explicitly declared 1.12 package")
		}
		return nil
	}
	if versionCheck() != nil {
		return fail("blocked", "package.version", "Pending contracts require a new or explicitly declared UWS 1.12 package.", 4)
	}
	path := filepath.Join(root, defaultIntentPath)
	before, readErr := readWithin(root, defaultIntentPath, evidencefile.DefaultMaxBytes)
	present := readErr == nil
	var intent *workflowintent.Intent
	if present {
		if request.IntentRevision.State != "present" || request.IntentRevision.SHA256 != sourceDigest(before) {
			return fail("conflict", "intent.stale", "The selected intent revision changed.", 3)
		}
		intent, err = workflowintent.ParseIntent(before, defaultIntentPath)
	} else if errors.Is(readErr, fs.ErrNotExist) {
		if request.IntentRevision.State != "missing" || request.IntentRevision.SHA256 != "" || request.Scaffold == nil {
			return fail("conflict", "intent.missing", "A missing intent needs an explicit scaffold and missing revision.", 3)
		}
		intent, err = intentFromScaffold(request.Scaffold)
	} else {
		return fail("blocked", "intent.unavailable", "The selected intent is unsafe or unavailable.", 4)
	}
	if err != nil {
		return fail("blocked", "intent.invalid", "The selected intent cannot be validated safely.", 4)
	}
	selected := findSteps(intent.Steps, request.StepID)
	if len(selected) > 1 {
		return fail("blocked", "step.ambiguous", "The selected step is not unique.", 4)
	}
	if len(selected) == 1 && (len(selected[0].Steps) > 0 || len(selected[0].Cases) > 0 || selected[0].Default != nil) {
		return fail("needs_input", "step.composite_unsupported", "A composite step cannot be replaced by one pending contract.", 4)
	}
	if validateDependencies(intent, request.StepID, request.DependsOn, nil) != nil {
		return fail("needs_input", "dependency.invalid", "Pending dependencies must identify unique existing steps.", 4)
	}
	step := &workflowintent.Step{Name: request.StepID, Type: "pending", Do: request.Contract.Purpose, DependsOn: append([]string(nil), request.DependsOn...), Pending: pendingContract(request.Contract)}
	if !replaceIntentStep(intent.Steps, request.StepID, step) {
		intent.Steps = append(intent.Steps, step)
	}
	if dependencyCycleFrom(intent.Steps, request.StepID) {
		return fail("needs_input", "dependency.cycle", "The pending contract would retain or create a dependency cycle.", 4)
	}
	rendered, err := workflowintent.RenderIntentHCL(intent)
	if err != nil {
		return fail("failed", "contract.invalid", "The pending contract fails public UWS validation or intent constraints.", 2)
	}
	next := []byte(rendered)
	if present {
		old, diags := hclwrite.ParseConfig(before, defaultIntentPath, hcl.InitialPos)
		if diags.HasErrors() {
			return fail("blocked", "intent.invalid", "The selected intent cannot be edited safely.", 4)
		}
		candidate, diags := hclwrite.ParseConfig(next, defaultIntentPath, hcl.InitialPos)
		if diags.HasErrors() {
			return fail("failed", "intent.render_failed", "The pending intent could not be rendered.", 1)
		}
		newBlocks := findHCLSteps(candidate.Body(), request.StepID)
		if len(newBlocks) != 1 {
			return fail("failed", "intent.render_failed", "The pending intent is not unique.", 1)
		}
		blocks := findHCLSteps(old.Body(), request.StepID)
		if len(blocks) == 0 {
			old.Body().AppendBlock(newBlocks[0])
		} else {
			blocks[0].Body().Clear()
			blocks[0].Body().AppendUnstructuredTokens(newBlocks[0].Body().BuildTokens(nil))
		}
		next = old.Bytes()
	}
	if _, err := workflowintent.ParseIntent(next, defaultIntentPath); err != nil {
		return fail("failed", "intent.invalid_after_pending", "The pending update does not form a valid intent.", 2)
	}
	if bytesEqual(before, next) {
		return bindSuccess(request.StepID, "unchanged", sourceDigest(next))
	}
	if ctx.Err() != nil {
		return fail("blocked", "request.cancelled", "Pending authoring was cancelled before writing.", 4)
	}
	file := artifactwriter.GeneratedFile{Path: path, Content: string(next), Action: "write", Reason: "author one explicit unresolved step contract", AllowOverwrite: present}
	if present {
		file.ExpectedCurrentSHA256 = sourceDigest(before)
	}
	_, err = artifactwriter.CommitChecked(artifactwriter.Prepared{ExampleRoot: root, Files: []artifactwriter.GeneratedFile{file}}, false, func() error {
		if e := versionCheck(); e != nil {
			return e
		}
		current, e := readWithin(root, defaultIntentPath, evidencefile.DefaultMaxBytes)
		if !present && errors.Is(e, fs.ErrNotExist) {
			return nil
		}
		if e != nil || !present || sourceDigest(current) != sourceDigest(before) {
			return errors.New("intent revision changed")
		}
		return ctx.Err()
	})
	if err != nil {
		var transactionErr *artifactwriter.TransactionError
		if errors.As(err, &transactionErr) {
			return bindFailureWithResult("failed", "write.indeterminate", "Inspect the durable intent revision before retrying the indeterminate write.", 1, &BindResult{StepID: request.StepID, WriteOutcome: "indeterminate"})
		}
		current, e := readWithin(root, defaultIntentPath, evidencefile.DefaultMaxBytes)
		if e == nil && (!present || sourceDigest(current) != sourceDigest(before)) {
			return fail("conflict", "intent.stale", "The intent changed before replacement.", 3)
		}
		return fail("failed", "write.failed", "The pending update could not be written.", 1)
	}
	return bindSuccess(request.StepID, "written", sourceDigest(next))
}
