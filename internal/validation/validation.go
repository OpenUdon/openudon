// Package validation owns OpenUdon's reusable public-UWS artifact validation
// policy. Command packages are responsible only for flags and presentation.
package validation

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/OpenUdon/uws/schemas"
	uwsvalidation "github.com/OpenUdon/uws/validation"
)

type SchemaSelector func(string) string

func SchemaForFile(path string) string {
	version := "1.0.0"
	if doc, err := uwsvalidation.LoadDocumentFile(path); err == nil && doc != nil && strings.TrimSpace(doc.UWS) != "" {
		version = strings.TrimSpace(doc.UWS)
	}
	return schemas.PathForVersion(".", version)
}

func ValidatePath(target string, out io.Writer, allowEmpty bool) error {
	return ValidatePathWithSchema(target, out, SchemaForFile, allowEmpty)
}

func ValidatePathWithSchema(target string, out io.Writer, schemaForFile SchemaSelector, allowEmpty bool) error {
	info, err := os.Stat(target)
	if err != nil {
		return fmt.Errorf("target does not exist: %s", target)
	}
	if !info.IsDir() {
		return ValidateFile(target, out, schemaForFile)
	}
	files, err := uwsvalidation.CollectArtifactFiles(target)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		if allowEmpty {
			fmt.Fprintf(out, "no UWS artifacts found under %s\n", target)
			return nil
		}
		return fmt.Errorf("no UWS artifacts found under %s; pass --allow-empty to allow this", target)
	}
	fmt.Fprintf(out, "found %d UWS artifact(s); schema selected from document version\n", len(files))
	for _, file := range files {
		if err := ValidateFile(file, out, schemaForFile); err != nil {
			return err
		}
	}
	return nil
}

func ValidateFile(path string, out io.Writer, schemaForFile SchemaSelector) error {
	if err := uwsvalidation.ValidateFile(schemaForFile(path), path); err != nil {
		return err
	}
	fmt.Fprintf(out, "openudon: %s is valid UWS\n", path)
	return nil
}
