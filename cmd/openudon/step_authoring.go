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

	"github.com/OpenUdon/openudon/internal/evidencefile"
	"github.com/OpenUdon/openudon/internal/stepauthoring"
)

func runStepCommand(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: openudon step {candidates|bind|check|source add} --example DIR --request FILE|-")
		return 2
	}
	switch args[0] {
	case "candidates":
		return runStepCandidatesCommand(args[1:], os.Stdin, os.Stdout, os.Stderr)
	case "check":
		return runStepCheckCommand(args[1:], os.Stdin, os.Stdout, os.Stderr)
	case "bind":
		return runStepBindCommand(args[1:], os.Stdin, os.Stdout, os.Stderr)
	case "source":
		return runStepSourceCommand(args[1:], os.Stdin, os.Stdout, os.Stderr)
	default:
		fmt.Fprintln(os.Stderr, "usage: openudon step {candidates|bind|check|source add} --example DIR --request FILE|-")
		return 2
	}
}

func runStepSourceCommand(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "add" {
		fmt.Fprintln(stderr, "usage: openudon step source add --example DIR --request FILE|-")
		return 2
	}
	fs := flag.NewFlagSet("openudon step source add", flag.ContinueOnError)
	fs.SetOutput(stderr)
	example := fs.String("example", "", "Workflow package directory")
	requestPath := fs.String("request", "", "Versioned JSON request file, or - for stdin")
	fs.Usage = func() {
		fmt.Fprintln(stderr, "Usage: openudon step source add --example DIR --request FILE|-")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return writeStepSourceAddResult(stdout, failedStepSourceAdd("request.invalid", "The step-source-add command arguments are invalid."))
	}
	if fs.NArg() != 0 || *example == "" || *requestPath == "" {
		return writeStepSourceAddResult(stdout, failedStepSourceAdd("request.invalid", "The step-source-add command requires a package and request."))
	}
	data, err := readStepCheckRequest(*requestPath, stdin)
	if err != nil || !stepauthoring.ValidUTF8Request(data) || !json.Valid(data) {
		return writeStepSourceAddResult(stdout, failedStepSourceAdd("request.invalid_json", "The step-source-add request must be bounded UTF-8 JSON."))
	}
	var request stepauthoring.SourceAddRequest
	if err := evidencefile.DecodeStrict(data, &request); err != nil {
		return writeStepSourceAddResult(stdout, failedStepSourceAdd("request.invalid", "The step-source-add request does not match the strict v1 schema."))
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return writeStepSourceAddResult(stdout, stepauthoring.AddSources(ctx, *example, request))
}

func failedStepSourceAdd(code, message string) stepauthoring.SourceAddOutcome {
	return stepauthoring.SourceAddOutcome{
		Result: stepauthoring.SourceAddWireResult{
			Version: stepauthoring.WireVersion, Kind: "result", Command: stepauthoring.SourceAddCommand,
			Status: "failed", Diagnostics: []stepauthoring.Diagnostic{{Code: code, Severity: "error", Message: message}},
		},
		ExitCode: 2,
	}
}

func runStepCandidatesCommand(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("openudon step candidates", flag.ContinueOnError)
	fs.SetOutput(stderr)
	example := fs.String("example", "", "Example directory containing local API source files")
	requestPath := fs.String("request", "", "Versioned JSON request file, or - for stdin")
	fs.Usage = func() {
		fmt.Fprintln(stderr, "Usage: openudon step candidates --example DIR --request FILE|-")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return writeStepCandidatesResult(stdout, stepauthoring.CandidatesOutcome{Result: stepauthoring.CandidatesWireResult{
			Version: stepauthoring.WireVersion, Kind: "result", Command: stepauthoring.CandidatesCommand,
			Status: "failed", Diagnostics: []stepauthoring.Diagnostic{{Code: "request.invalid", Severity: "error", Message: "The step-candidates command arguments are invalid."}},
		}, ExitCode: 2})
	}
	if fs.NArg() != 0 || *example == "" || *requestPath == "" {
		return writeStepCandidatesResult(stdout, stepauthoring.CandidatesOutcome{Result: stepauthoring.CandidatesWireResult{
			Version: stepauthoring.WireVersion, Kind: "result", Command: stepauthoring.CandidatesCommand,
			Status: "failed", Diagnostics: []stepauthoring.Diagnostic{{Code: "request.invalid", Severity: "error", Message: "The step-candidates command requires an example and request."}},
		}, ExitCode: 2})
	}
	data, err := readStepCheckRequest(*requestPath, stdin)
	if err != nil || !stepauthoring.ValidUTF8Request(data) || !json.Valid(data) {
		return writeStepCandidatesResult(stdout, stepauthoring.CandidatesOutcome{Result: stepauthoring.CandidatesWireResult{
			Version: stepauthoring.WireVersion, Kind: "result", Command: stepauthoring.CandidatesCommand,
			Status: "failed", Diagnostics: []stepauthoring.Diagnostic{{Code: "request.invalid_json", Severity: "error", Message: "The step-candidates request must be bounded UTF-8 JSON."}},
		}, ExitCode: 2})
	}
	var request stepauthoring.CandidatesRequest
	if err := evidencefile.DecodeStrict(data, &request); err != nil {
		return writeStepCandidatesResult(stdout, stepauthoring.CandidatesOutcome{Result: stepauthoring.CandidatesWireResult{
			Version: stepauthoring.WireVersion, Kind: "result", Command: stepauthoring.CandidatesCommand,
			Status: "failed", Diagnostics: []stepauthoring.Diagnostic{{Code: "request.invalid", Severity: "error", Message: "The step-candidates request does not match the strict v1 schema."}},
		}, ExitCode: 2})
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return writeStepCandidatesResult(stdout, stepauthoring.Candidates(ctx, *example, request))
}

func runFlowReviewCommand(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("openudon flow-review", flag.ContinueOnError)
	fs.SetOutput(stderr)
	example := fs.String("example", "", "Example directory containing workflows/intent.hcl")
	requestPath := fs.String("request", "", "Versioned JSON request file, or - for stdin")
	fs.Usage = func() {
		fmt.Fprintln(stderr, "Usage: openudon flow-review --example DIR --request FILE|-")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return writeFlowReviewResult(stdout, stepauthoring.FlowReviewOutcome{Result: stepauthoring.FlowReviewWireResult{
			Version: stepauthoring.WireVersion, Kind: "result", Command: stepauthoring.FlowReviewCommand,
			Status: "failed", Diagnostics: []stepauthoring.Diagnostic{{Code: "request.invalid", Severity: "error", Message: "The flow-review command arguments are invalid."}},
		}, ExitCode: 2})
	}
	if fs.NArg() != 0 || *example == "" || *requestPath == "" {
		return writeFlowReviewResult(stdout, stepauthoring.FlowReviewOutcome{Result: stepauthoring.FlowReviewWireResult{
			Version: stepauthoring.WireVersion, Kind: "result", Command: stepauthoring.FlowReviewCommand,
			Status: "failed", Diagnostics: []stepauthoring.Diagnostic{{Code: "request.invalid", Severity: "error", Message: "The flow-review command requires an example and request."}},
		}, ExitCode: 2})
	}
	data, err := readStepCheckRequest(*requestPath, stdin)
	if err != nil || !stepauthoring.ValidUTF8Request(data) || !json.Valid(data) {
		return writeFlowReviewResult(stdout, stepauthoring.FlowReviewOutcome{Result: stepauthoring.FlowReviewWireResult{
			Version: stepauthoring.WireVersion, Kind: "result", Command: stepauthoring.FlowReviewCommand,
			Status: "failed", Diagnostics: []stepauthoring.Diagnostic{{Code: "request.invalid_json", Severity: "error", Message: "The flow-review request must be bounded UTF-8 JSON."}},
		}, ExitCode: 2})
	}
	if !stepauthoring.ValidFlowReviewRequestJSON(data) {
		return writeFlowReviewResult(stdout, stepauthoring.FlowReviewOutcome{Result: stepauthoring.FlowReviewWireResult{
			Version: stepauthoring.WireVersion, Kind: "result", Command: stepauthoring.FlowReviewCommand,
			Status: "failed", Diagnostics: []stepauthoring.Diagnostic{{Code: "request.invalid", Severity: "error", Message: "The flow-review request does not match the strict v1 schema."}},
		}, ExitCode: 2})
	}
	var request stepauthoring.FlowReviewRequest
	if err := evidencefile.DecodeStrict(data, &request); err != nil {
		return writeFlowReviewResult(stdout, stepauthoring.FlowReviewOutcome{Result: stepauthoring.FlowReviewWireResult{
			Version: stepauthoring.WireVersion, Kind: "result", Command: stepauthoring.FlowReviewCommand,
			Status: "failed", Diagnostics: []stepauthoring.Diagnostic{{Code: "request.invalid", Severity: "error", Message: "The flow-review request does not match the strict v1 schema."}},
		}, ExitCode: 2})
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return writeFlowReviewResult(stdout, stepauthoring.ReviewFlow(ctx, *example, request, nil))
}

func runStepBindCommand(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("openudon step bind", flag.ContinueOnError)
	fs.SetOutput(stderr)
	example := fs.String("example", "", "Example directory containing workflows/intent.hcl")
	requestPath := fs.String("request", "", "Versioned JSON request file, or - for stdin")
	fs.Usage = func() {
		fmt.Fprintln(stderr, "Usage: openudon step bind --example DIR --request FILE|-")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return writeStepBindResult(stdout, stepauthoring.BindOutcome{Result: stepauthoring.BindWireResult{
			Version: stepauthoring.WireVersion, Kind: "result", Command: stepauthoring.BindCommand,
			Status: "failed", Diagnostics: []stepauthoring.Diagnostic{{Code: "request.invalid", Severity: "error", Message: "The step-bind command arguments are invalid."}},
		}, ExitCode: 2})
	}
	if fs.NArg() != 0 || *example == "" || *requestPath == "" {
		return writeStepBindResult(stdout, stepauthoring.BindOutcome{Result: stepauthoring.BindWireResult{
			Version: stepauthoring.WireVersion, Kind: "result", Command: stepauthoring.BindCommand,
			Status: "failed", Diagnostics: []stepauthoring.Diagnostic{{Code: "request.invalid", Severity: "error", Message: "The step-bind command requires an example and request."}},
		}, ExitCode: 2})
	}
	data, err := readStepCheckRequest(*requestPath, stdin)
	if err != nil || !stepauthoring.ValidUTF8Request(data) || !json.Valid(data) {
		return writeStepBindResult(stdout, stepauthoring.BindOutcome{Result: stepauthoring.BindWireResult{
			Version: stepauthoring.WireVersion, Kind: "result", Command: stepauthoring.BindCommand,
			Status: "failed", Diagnostics: []stepauthoring.Diagnostic{{Code: "request.invalid_json", Severity: "error", Message: "The step-bind request must be bounded UTF-8 JSON."}},
		}, ExitCode: 2})
	}
	var request stepauthoring.BindRequest
	if err := evidencefile.DecodeStrict(data, &request); err != nil {
		return writeStepBindResult(stdout, stepauthoring.BindOutcome{Result: stepauthoring.BindWireResult{
			Version: stepauthoring.WireVersion, Kind: "result", Command: stepauthoring.BindCommand,
			Status: "failed", Diagnostics: []stepauthoring.Diagnostic{{Code: "request.invalid", Severity: "error", Message: "The step-bind request does not match the strict v1 schema."}},
		}, ExitCode: 2})
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return writeStepBindResult(stdout, stepauthoring.Bind(ctx, *example, request))
}

func runStepCheckCommand(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("openudon step check", flag.ContinueOnError)
	fs.SetOutput(stderr)
	example := fs.String("example", "", "Example directory containing workflows/intent.hcl")
	requestPath := fs.String("request", "", "Versioned JSON request file, or - for stdin")
	fs.Usage = func() {
		fmt.Fprintln(stderr, "Usage: openudon step check --example DIR --request FILE|-")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return writeStepCheckResult(stdout, stepauthoring.FailedResult("request.invalid", "The step-check command arguments are invalid.", 2))
	}
	if fs.NArg() != 0 || *example == "" || *requestPath == "" {
		return writeStepCheckResult(stdout, stepauthoring.FailedResult("request.invalid", "The step-check command requires an example and request.", 2))
	}
	data, err := readStepCheckRequest(*requestPath, stdin)
	if err != nil {
		return writeStepCheckResult(stdout, stepauthoring.FailedResult("request.invalid_json", "The step-check request could not be read as bounded JSON.", 2))
	}
	if !stepauthoring.ValidUTF8Request(data) {
		return writeStepCheckResult(stdout, stepauthoring.FailedResult("request.invalid_json", "The step-check request must be valid UTF-8 JSON.", 2))
	}
	if !json.Valid(data) {
		return writeStepCheckResult(stdout, stepauthoring.FailedResult("request.invalid_json", "The step-check request must contain one JSON value.", 2))
	}
	var request stepauthoring.CheckRequest
	if err := evidencefile.DecodeStrict(data, &request); err != nil {
		return writeStepCheckResult(stdout, stepauthoring.FailedResult("request.invalid", "The step-check request does not match the strict v1 schema.", 2))
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	outcome := stepauthoring.Check(ctx, *example, request)
	return writeStepCheckResult(stdout, outcome)
}

func readStepCheckRequest(path string, stdin io.Reader) ([]byte, error) {
	if path == "-" {
		data, err := io.ReadAll(io.LimitReader(stdin, stepauthoring.MaxRequestBytes+1))
		if err != nil {
			return nil, err
		}
		if len(data) > stepauthoring.MaxRequestBytes {
			return nil, fmt.Errorf("request exceeds bound")
		}
		return data, nil
	}
	data, _, err := evidencefile.ReadRegular(path, stepauthoring.MaxRequestBytes)
	return data, err
}

func writeStepCheckResult(stdout io.Writer, outcome stepauthoring.Outcome) int {
	encoded, err := json.Marshal(outcome.Result)
	if err != nil || len(encoded) > stepauthoring.MaxResultBytes {
		fallback, _ := json.Marshal(stepauthoring.FailedResult("request.invalid", "The step-check result could not be encoded within its output bound.", 1).Result)
		_, _ = io.Copy(stdout, bytes.NewReader(append(fallback, '\n')))
		return 1
	}
	if _, err := stdout.Write(append(encoded, '\n')); err != nil {
		return 1
	}
	return outcome.ExitCode
}

func writeStepBindResult(stdout io.Writer, outcome stepauthoring.BindOutcome) int {
	encoded, err := json.Marshal(outcome.Result)
	if err != nil || len(encoded) > stepauthoring.MaxResultBytes {
		fallback := []byte(`{"version":"openudon.step-authoring.v1","kind":"result","command":"step.bind","status":"failed","diagnostics":[{"code":"result.encoding_failed","severity":"error","message":"The step-bind result could not be encoded within its output bound."}]}` + "\n")
		_, _ = stdout.Write(fallback)
		return 1
	}
	if _, err := stdout.Write(append(encoded, '\n')); err != nil {
		return 1
	}
	return outcome.ExitCode
}

func writeStepSourceAddResult(stdout io.Writer, outcome stepauthoring.SourceAddOutcome) int {
	encoded, err := json.Marshal(outcome.Result)
	if err != nil || len(encoded) > stepauthoring.MaxResultBytes {
		fallback, _ := json.Marshal(stepauthoring.SourceAddWireResult{
			Version: stepauthoring.WireVersion, Kind: "result", Command: stepauthoring.SourceAddCommand,
			Status: "failed", Diagnostics: []stepauthoring.Diagnostic{{Code: "result.encoding_failed", Severity: "error", Message: "The step-source-add result could not be encoded within its output bound."}},
		})
		_, _ = io.Copy(stdout, bytes.NewReader(append(fallback, '\n')))
		return 1
	}
	if _, err := stdout.Write(append(encoded, '\n')); err != nil {
		return 1
	}
	return outcome.ExitCode
}

func writeStepCandidatesResult(stdout io.Writer, outcome stepauthoring.CandidatesOutcome) int {
	encoded, err := json.Marshal(outcome.Result)
	if err != nil || len(encoded) > stepauthoring.MaxResultBytes {
		fallback, _ := json.Marshal(stepauthoring.CandidatesWireResult{
			Version: stepauthoring.WireVersion, Kind: "result", Command: stepauthoring.CandidatesCommand,
			Status: "failed", Diagnostics: []stepauthoring.Diagnostic{{Code: "result.encoding_failed", Severity: "error", Message: "The step-candidates result could not be encoded within its output bound."}},
		})
		_, _ = io.Copy(stdout, bytes.NewReader(append(fallback, '\n')))
		return 1
	}
	if _, err := stdout.Write(append(encoded, '\n')); err != nil {
		return 1
	}
	return outcome.ExitCode
}

func writeFlowReviewResult(stdout io.Writer, outcome stepauthoring.FlowReviewOutcome) int {
	encoded, err := json.Marshal(outcome.Result)
	if err != nil || len(encoded) > stepauthoring.MaxResultBytes {
		fallback := []byte(`{"version":"openudon.step-authoring.v1","kind":"result","command":"flow-review","status":"failed","diagnostics":[{"code":"result.encoding_failed","severity":"error","message":"The flow-review result could not be encoded within its output bound."}]}` + "\n")
		_, _ = stdout.Write(fallback)
		return 1
	}
	if _, err := stdout.Write(append(encoded, '\n')); err != nil {
		return 1
	}
	return outcome.ExitCode
}
