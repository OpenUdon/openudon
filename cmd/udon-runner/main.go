package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/OpenUdon/openudon/internal/udonrunner"
)

func main() {
	configPath := flag.String("config", "", "Path to openudon.executor-run.v1 JSON")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: udon-runner --config <run-config.json>\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	if *configPath == "" || flag.NArg() != 0 {
		flag.Usage()
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if _, err := udonrunner.RunConfig(ctx, udonrunner.Options{
		ConfigPath: *configPath,
		RepoRoot:   ".",
		Stdout:     os.Stdout,
		Stderr:     os.Stderr,
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
