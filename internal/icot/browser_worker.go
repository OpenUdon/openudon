package icot

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/OpenUdon/browsertools/authorworker"
	"github.com/OpenUdon/browsertools/capture"
	"github.com/OpenUdon/browsertools/registrationauthorworker"
)

// runBundledBrowserWorker is intentionally absent from public help. The parent
// iCoT process copies its own executable into the private root and re-executes
// this entry point so Playwright never runs in the authoring engine or HTTP
// server process.
func runBundledBrowserWorker(args []string, in io.Reader, out, errOut io.Writer) int {
	if len(args) >= 2 && args[0] == "playwright-doctor" && args[1] == "chromium" {
		return runBundledBrowserDoctor(args[2:], out, errOut)
	}
	if len(args) < 2 || (args[0] != "author-session" && args[0] != "registration-author-session") || args[1] != "chromium" {
		fmt.Fprintln(errOut, "browser worker: unsupported invocation")
		return 2
	}
	fs := flag.NewFlagSet("browser worker", flag.ContinueOnError)
	fs.SetOutput(errOut)
	privateRoot := fs.String("private-root", "", "")
	driverDirectory := fs.String("driver-dir", "", "")
	if err := fs.Parse(args[2:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 || *privateRoot == "" {
		fmt.Fprintln(errOut, "browser worker: invalid invocation")
		return 2
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	stdin, ok := in.(io.ReadCloser)
	if !ok {
		stdin = io.NopCloser(in)
	}
	var runErr error
	if args[0] == "registration-author-session" {
		runErr = registrationauthorworker.Run(ctx, registrationauthorworker.Options{
			PrivateRoot: *privateRoot, DriverDirectory: *driverDirectory, Stdin: stdin, Stdout: out,
		})
	} else {
		runErr = authorworker.Run(ctx, authorworker.Options{
			PrivateRoot: *privateRoot, DriverDirectory: *driverDirectory, Stdin: stdin, Stdout: out,
		})
	}
	if runErr != nil {
		fmt.Fprintln(errOut, "browser worker: session failed closed")
		return 1
	}
	return 0
}

func runBundledBrowserDoctor(args []string, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("browser worker doctor", flag.ContinueOnError)
	fs.SetOutput(errOut)
	driverDirectory := fs.String("driver-dir", "", "")
	if err := fs.Parse(args); err != nil || fs.NArg() != 0 {
		fmt.Fprintln(errOut, "browser worker: invalid doctor invocation")
		return 2
	}
	parent, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()
	report, err := capture.Doctor(ctx, capture.NewPlaywrightRuntime(*driverDirectory), capture.EngineChromium)
	if encodeErr := json.NewEncoder(out).Encode(report); encodeErr != nil {
		fmt.Fprintln(errOut, "browser worker: doctor report failed closed")
		return 1
	}
	if err != nil {
		fmt.Fprintln(errOut, "browser worker: Chromium readiness check failed")
		return 1
	}
	return 0
}
