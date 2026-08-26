package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/OpenUdon/openudon/internal/trustedrunner"
)

func main() {
	configPath := flag.String("config", "", "Path to openudon.executor-run.v2 JSON")
	configSHA256 := flag.String("config-sha256", "", "Exact SHA-256 of the config bytes validated by openudon")
	approvalPath := flag.String("approval", "", "Path to the approval JSON bound by the run config")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: udon-runner --config <run-config.json> --config-sha256 <hex> --approval <approval.json>\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	if *configPath == "" || *configSHA256 == "" || *approvalPath == "" || flag.NArg() != 0 {
		flag.Usage()
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if _, err := trustedrunner.RunExternal(ctx, trustedrunner.ExternalOptions{
		ConfigPath:                  *configPath,
		ConfigSHA256:                *configSHA256,
		ApprovalPath:                *approvalPath,
		RegistrationAttestationPath: os.Getenv("OPENUDON_BROWSER_REGISTRATION_ATTESTATION"),
		RegistrationSubmitApproval:  os.Getenv("OPENUDON_BROWSER_REGISTRATION_SUBMIT_APPROVAL"),
		Stdout:                      os.Stdout,
		Stderr:                      os.Stderr,
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
