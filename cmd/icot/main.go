package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/genelet/ramen/internal/projectwizard"
)

func main() {
	example := flag.String("example", "", "Example directory where project.md will be created")
	force := flag.Bool("force", false, "Overwrite an existing project.md")
	usage := func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: icot --example examples/<name> [--force]\n")
		fmt.Fprintf(flag.CommandLine.Output(), "\nInteractively writes project.md with the standard Ramen authoring sections.\n")
		fmt.Fprintf(flag.CommandLine.Output(), "It also creates openapi/, workflows/, and expected/ when missing.\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Next step: ramen synthesize --example examples/<name>\n\n")
		flag.PrintDefaults()
	}
	flag.Usage = usage
	flag.CommandLine.Usage = usage
	flag.Parse()

	exampleDir := strings.TrimSpace(*example)
	if exampleDir == "" {
		fmt.Fprintln(os.Stderr, "--example is required")
		os.Exit(2)
	}
	projectPath := filepath.Join(exampleDir, "project.md")
	if !*force {
		if _, err := os.Stat(projectPath); err == nil {
			fmt.Fprintf(os.Stderr, "%s already exists; pass --force to overwrite it\n", projectPath)
			os.Exit(1)
		} else if !os.IsNotExist(err) {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	document, err := projectwizard.Run(os.Stdin, os.Stdout)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for _, dir := range []string{
		exampleDir,
		filepath.Join(exampleDir, "openapi"),
		filepath.Join(exampleDir, "workflows"),
		filepath.Join(exampleDir, "expected"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	if err := os.WriteFile(projectPath, []byte(document), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("icot: wrote %s\n", projectPath)
	fmt.Printf("next: ramen synthesize --example %s\n", exampleDir)
}
