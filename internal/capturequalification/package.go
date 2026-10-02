package capturequalification

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/OpenUdon/browsertools/profile"
	"github.com/OpenUdon/openudon/internal/browsercapture"
	"github.com/OpenUdon/openudon/internal/browserpackage"
	"github.com/OpenUdon/openudon/internal/browsertransaction"
	"github.com/OpenUdon/openudon/internal/evidencefile"
	"github.com/OpenUdon/openudon/internal/packagepipeline"
	"github.com/OpenUdon/openudon/internal/processgroup"
	"io"
	"os/exec"
	"path/filepath"
	"time"
)

// publishQualification consumes the original public capture receipt. Native
// browserpackage and packagepipeline independently validate it; a harness does
// not rewrite the capture transaction into a package lifecycle state.
func publishQualification(ctx context.Context, o RegistrationOptions, startBytes []byte, mode string, result RegistrationResult) (RegistrationResult, error) {
	receiptPath := filepath.Join("expected", "browser-capture", o.ProfileID+".json")
	receipt, _, err := evidencefile.ReadRegular(filepath.Join(o.ExampleDir, receiptPath), browserpackage.MaxRequestBytes)
	if err != nil {
		return result, errors.New("neutral package qualification failed: receipt")
	}
	var r struct {
		Transaction browsertransaction.Transaction `json:"transaction"`
	}
	if json.Unmarshal(receipt, &r) != nil || r.Transaction.State != browsertransaction.StateReviewed {
		return result, errors.New("neutral package qualification failed: reviewed receipt")
	}
	tx, err := browsertransaction.Digest(r.Transaction)
	if err != nil {
		return result, errors.New("neutral package qualification failed: transaction digest")
	}
	input, err := browserpackage.InputDigest(ctx, o.ExampleDir)
	if err != nil {
		return result, errors.New("neutral package qualification failed: input digest")
	}
	totp := mode == browsercapture.Authenticated
	request := browserpackage.Request{Version: browserpackage.Version, Kind: "request", RequestID: "neutral_qualification", Start: startBytes, ReceiptPath: filepath.ToSlash(receiptPath), ReceiptSHA256: evidencefile.SHA256(receipt), TransactionSHA256: tx, InputSHA256: input, ExpectedTOTP: &totp, RegistrationAuthority: "synthetic_qualification", WorkflowName: "qualified_browser", AllowOverwrite: true, Flow: "create_dedicated_test_user", CleanupDisposition: "delete_separately", Inputs: []browserpackage.Input{{Name: "registration_inputs", Type: "object", Sensitive: true}}}
	if mode == browsercapture.Authenticated {
		request.RegistrationAuthority = ""
		request.CleanupDisposition = ""
		request.Inputs = nil
		request.Flow = "authenticated_goal"
		content, _, err := evidencefile.ReadRegular(filepath.Join(o.ExampleDir, "browser-profiles", r.Transaction.ID+".json"), 1<<20)
		if err != nil {
			return result, errors.New("qualification capability profile unavailable")
		}
		capability, err := profile.ParseJSON(content)
		if err != nil || len(capability.SortedActionNames()) != 1 {
			return result, errors.New("qualification capability profile invalid")
		}
		request.Action = capability.SortedActionNames()[0]
	}
	data, err := json.Marshal(request)
	if err != nil {
		return result, errors.New("neutral package qualification failed: request encoding")
	}
	var plan browserpackage.Plan
	args := []string{"browser-author", "plan", "--example", o.ExampleDir, "--request", "-"}
	if err := qualificationCommand(ctx, o, args, data, &plan); err != nil || !plan.Ready {
		return result, errors.New("neutral package qualification failed: public plan")
	}
	var refused bytes.Buffer
	err = processgroup.Run(ctx, 2*time.Minute, processgroup.Invocation{Args: append([]string{o.Executable}, "browser-author", "apply", "--example", o.ExampleDir, "--request", "-", "--expected-plan", plan.PlanSHA256), Dir: filepath.Dir(o.ExampleDir), Env: o.Environment, Stdin: bytes.NewReader(data), Stdout: &refused, Stderr: io.Discard})
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 2 || refused.Len() != 0 {
		return result, errors.New("neutral package qualification failed: missing authoring approval refusal")
	}
	var applied browserpackage.Result
	if err := qualificationCommand(ctx, o, []string{"browser-author", "apply", "--example", o.ExampleDir, "--request", "-", "--expected-plan", plan.PlanSHA256, "--confirmed"}, data, &applied); err != nil || applied.Outcome != "authored" || applied.QualityStatus != "pass" {
		return result, errors.New("neutral package qualification failed: confirmed authoring/build")
	}
	common := []string{"--example", o.ExampleDir, "--scope", o.Scope, "--scratch", o.ScratchParent}
	var prepared packageOutput
	if err := qualificationCommand(ctx, o, append([]string{"package", "prepare"}, common...), nil, &prepared); err != nil || prepared.Selection != nil || prepared.Version != "openudon.package-command.v1" {
		return result, errors.New("neutral package qualification failed: package preparation")
	}
	var promoted packageOutput
	promote := append([]string{"package", "promote"}, common...)
	promote = append(promote, "--store", o.StoreDir, "--expected-input", prepared.Preparation.InputSHA256, "--confirmed")
	if err := qualificationCommand(ctx, o, promote, nil, &promoted); err != nil || promoted.Selection == nil || promoted.Preparation.PackageSHA256 != prepared.Preparation.PackageSHA256 {
		return result, errors.New("neutral package qualification failed: package promotion")
	}
	var selected packagepipeline.Selection
	if err := qualificationCommand(ctx, o, []string{"package", "inspect", "--store", o.StoreDir}, nil, &selected); err != nil || selected != *promoted.Selection || selected.PackageSHA256 != prepared.Preparation.PackageSHA256 {
		return result, errors.New("neutral package qualification failed: independent selection")
	}
	result.Prepared = prepared.Preparation
	result.Qualified = prepared.Qualification
	result.Selection = selected
	profileDirectory := "browser-registration"
	if mode == browsercapture.Authenticated {
		profileDirectory = "browser-profiles"
	}
	profile, _, err := evidencefile.ReadRegular(filepath.Join(o.ExampleDir, profileDirectory, r.Transaction.ID+".json"), 1<<20)
	if err != nil {
		return result, errors.New("neutral package qualification failed: profile read")
	}
	result.Transaction = r.Transaction
	result.TransactionSHA256 = tx
	result.CanonicalProfile = profile
	return result, nil
}

type packageOutput struct {
	Version       string                              `json:"version"`
	Preparation   packagepipeline.Manifest            `json:"preparation"`
	Qualification packagepipeline.QualificationReport `json:"qualification"`
	Selection     *packagepipeline.Selection          `json:"selection,omitempty"`
}
type qualificationOutput struct{ bytes.Buffer }

func (b *qualificationOutput) Write(p []byte) (int, error) {
	if b.Len()+len(p) > browserpackage.MaxReportBytes {
		return 0, errors.New("qualification output bound")
	}
	return b.Buffer.Write(p)
}
func qualificationCommand(ctx context.Context, o RegistrationOptions, args []string, data []byte, target any) error {
	var out qualificationOutput
	if err := processgroup.Run(ctx, 2*time.Minute, processgroup.Invocation{Args: append([]string{o.Executable}, args...), Dir: filepath.Dir(o.ExampleDir), Env: o.Environment, Stdin: bytes.NewReader(data), Stdout: &out, Stderr: io.Discard}); err != nil {
		if errors.Is(err, processgroup.ErrTerminationTimeout) {
			return processgroup.ErrTerminationTimeout
		}
		return errors.New("qualification command failed")
	}
	if err := evidencefile.DecodeStrict(out.Bytes(), target); err != nil {
		return errors.New("qualification command evidence invalid")
	}
	return nil
}
