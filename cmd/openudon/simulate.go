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
	"github.com/OpenUdon/openudon/internal/simulation"
	"github.com/OpenUdon/uws/mockruntime"
)

func runSimulateCommand(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("openudon simulate", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	example := fs.String("example", "", "Reviewed package directory")
	inputPath := fs.String("input", "", "Optional openudon.simulate-input.v1 JSON with inputs and response examples/schemas")
	fixturePath := fs.String("fixtures", "", "Optional public UWS Mock Fixture Format 1.0 JSON")
	fallback := fs.Bool("allow-generated-fallback", false, "Explicitly permit examples/synthesis after an exact fixture miss")
	emit := func(report simulation.Report) int {
		encoded, err := json.Marshal(report)
		if err != nil || len(encoded) > simulation.MaxResultBytes {
			report = simulation.Failed("result.encoding", "The bounded preview could not be encoded.")
			encoded, _ = json.Marshal(report)
		}
		if _, err := stdout.Write(append(encoded, '\n')); err != nil {
			return 1
		}
		if report.Status != "completed" {
			return 1
		}
		return 0
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fs.SetOutput(stderr)
			fs.PrintDefaults()
			return 0
		}
		return emit(simulation.Failed("request.invalid", "The simulation arguments are invalid."))
	}
	if fs.NArg() != 0 || *example == "" {
		return emit(simulation.Failed("request.invalid", "Simulation requires exactly one --example package."))
	}
	repo, err := os.Getwd()
	if err != nil {
		return emit(simulation.Failed("workspace.unavailable", "The current repository root is unavailable."))
	}
	options := simulation.Options{RepoRoot: repo, ExampleDir: *example, AllowGeneratedFallback: *fallback}
	if *inputPath != "" {
		data, _, err := evidencefile.ReadRegular(*inputPath, 256<<10)
		if err != nil {
			return emit(simulation.Failed("input.invalid", "The bounded simulation input is unavailable or unsafe."))
		}
		var input simulation.Input
		if evidencefile.DecodeStrictNumbers(data, &input) != nil || input.Version != simulation.InputVersion {
			return emit(simulation.Failed("input.invalid", "The simulation input does not match its versioned contract."))
		}
		options.Inputs = input.Inputs
		options.Responses = input.Responses
	}
	if *fixturePath != "" {
		data, _, err := evidencefile.ReadRegular(*fixturePath, mockruntime.MaxFixtureSetBytes)
		if err != nil {
			return emit(simulation.Failed("fixtures.invalid", "The bounded fixture file is unavailable or unsafe."))
		}
		options.Fixtures, err = mockruntime.DecodeFixtures(data)
		if err != nil {
			return emit(simulation.Failed("fixtures.invalid", "The fixture file does not match public Mock Fixture Format 1.0."))
		}
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return emit(simulation.Run(ctx, options))
}
