package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/OpenUdon/openudon/internal/evidencefile"
	"github.com/OpenUdon/openudon/internal/stepauthoring"
)

func runStepPendingCommand(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("openudon step pending", flag.ContinueOnError)
	fs.SetOutput(stderr)
	example := fs.String("example", "", "Package directory")
	requestPath := fs.String("request", "", "Versioned pending request file, or - for stdin")
	fail := func(code, message string) int {
		return writeStepBindResult(stdout, stepauthoring.BindOutcome{Result: stepauthoring.BindWireResult{Version: stepauthoring.PendingWireVersion, Kind: "result", Command: stepauthoring.PendingCommand, Status: "failed", Diagnostics: []stepauthoring.Diagnostic{{Code: code, Severity: "error", Message: message}}}, ExitCode: 2})
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return fail("request.invalid", "The pending-step arguments are invalid.")
	}
	if fs.NArg() != 0 || *example == "" || *requestPath == "" {
		return fail("request.invalid", "The pending-step command requires an example and request.")
	}
	data, err := readStepCheckRequest(*requestPath, stdin)
	if err != nil || !stepauthoring.ValidUTF8Request(data) || !json.Valid(data) {
		return fail("request.invalid_json", "The pending-step request must be bounded UTF-8 JSON.")
	}
	var request stepauthoring.PendingRequest
	if evidencefile.DecodeStrict(data, &request) != nil {
		return fail("request.invalid", "The request does not match the pending-step envelope.")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return writeStepBindResult(stdout, stepauthoring.Pending(ctx, *example, request))
}
