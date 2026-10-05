package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/OpenUdon/openudon/internal/trustedrunner"
)

func runBrokerInspectCommand(args []string) {
	fs := flag.NewFlagSet("broker-inspect", flag.ExitOnError)
	example := fs.String("example", "", "Exact reviewed package relative to this repository root")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if fs.NArg() != 0 || *example == "" {
		fs.Usage()
		os.Exit(2)
	}
	inspection, err := trustedrunner.InspectBrokerPackage(context.Background(), trustedrunner.TemplateOptions{RepoRoot: ".", ExampleDir: *example})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(inspection); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
