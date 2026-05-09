package main

import (
	"os"

	"github.com/OpenUdon/openudon/internal/icot"
)

func main() {
	os.Exit(icot.Main(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
