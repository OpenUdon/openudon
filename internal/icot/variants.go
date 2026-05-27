package icot

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	evalpkg "github.com/OpenUdon/openudon/internal/eval"
)

const variantsValidationReportVersion = "openudon.icot-variants-validation.v1"

type variantsValidationReport struct {
	Version string                     `json:"version"`
	Status  string                     `json:"status"`
	Root    string                     `json:"root"`
	Results []variantsValidationResult `json:"results"`
}

type variantsValidationResult struct {
	Fixture      string   `json:"fixture"`
	Path         string   `json:"path"`
	Status       string   `json:"status"`
	VariantCount int      `json:"variant_count,omitempty"`
	Errors       []string `json:"errors,omitempty"`
}

func runVariants(args []string, out, errOut io.Writer) int {
	if len(args) == 0 || args[0] != "validate" {
		fmt.Fprintln(errOut, "Usage: icot variants validate [--root examples/eval] [--name fixture] [--json]")
		return 2
	}
	return runVariantsValidate(args[1:], out, errOut)
}

func runVariantsValidate(args []string, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("icot variants validate", flag.ContinueOnError)
	fs.SetOutput(out)
	root := fs.String("root", "examples/eval", "Directory containing eval example subdirectories")
	name := fs.String("name", "", "Validate one eval fixture by directory name")
	jsonOutput := fs.Bool("json", false, "Write openudon.icot-variants-validation.v1 JSON to stdout")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: icot variants validate [--root examples/eval] [--name fixture] [--json]\n\n")
		fmt.Fprintf(fs.Output(), "Validates authoring variant metadata, expected failure families, and reference-seeded clear slots without running the scorecard.\n\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	fixtures := discoverScorecardFixtures(*root, *name)
	if len(fixtures) == 0 {
		fmt.Fprintf(errOut, "no eval fixtures found under %s\n", *root)
		return 1
	}
	report := variantsValidationReport{
		Version: variantsValidationReportVersion,
		Status:  statusPass,
		Root:    *root,
	}
	for _, fixture := range fixtures {
		result := validateAuthoringVariantsFixture(fixture)
		if result.Status == "" {
			continue
		}
		report.Results = append(report.Results, result)
		if result.Status != statusPass {
			report.Status = statusFail
		}
		if !*jsonOutput {
			if result.Status == statusPass {
				fmt.Fprintf(out, "icot variants validate: pass %s (%d variant(s))\n", result.Fixture, result.VariantCount)
			} else {
				fmt.Fprintf(out, "icot variants validate: fail %s - %s\n", result.Fixture, strings.Join(result.Errors, "; "))
			}
		}
	}
	if len(report.Results) == 0 {
		fmt.Fprintf(errOut, "no authoring variant files found under %s\n", *root)
		return 1
	}
	if *jsonOutput {
		if err := json.NewEncoder(out).Encode(report); err != nil {
			fmt.Fprintln(errOut, err)
			return 1
		}
	}
	if report.Status != statusPass {
		return 1
	}
	return 0
}

func validateAuthoringVariantsFixture(fixture string) variantsValidationResult {
	fixtureName := filepath.Base(filepath.Clean(fixture))
	path := filepath.Join(fixture, "reference", "authoring-variants.json")
	result := variantsValidationResult{
		Fixture: fixtureName,
		Path:    path,
		Status:  statusPass,
	}
	variants, err := evalpkg.ReadAuthoringVariants(path)
	if os.IsNotExist(err) {
		return variantsValidationResult{}
	}
	if err != nil {
		result.Status = statusFail
		result.Errors = append(result.Errors, err.Error())
		return result
	}
	result.VariantCount = len(variants.Variants)
	for _, variant := range variants.Variants {
		if variant.SeedFromReference {
			if _, err := scorecardVariantSession(fixture, fixtureName, variant); err != nil {
				result.Status = statusFail
				result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", variant.ID, err))
			}
		}
		if variant.ExpectedOutcome == statusNeedsInput {
			if variant.ExpectedFailureFamily == "" {
				result.Status = statusFail
				result.Errors = append(result.Errors, fmt.Sprintf("%s: needs_input variants must declare expected_failure_family", variant.ID))
			}
			if variant.ExpectedTopIssueCode == "" {
				result.Status = statusFail
				result.Errors = append(result.Errors, fmt.Sprintf("%s: needs_input variants must declare expected_top_issue_code", variant.ID))
			}
			if variant.ExpectedTopIssueSlot == "" {
				result.Status = statusFail
				result.Errors = append(result.Errors, fmt.Sprintf("%s: needs_input variants must declare expected_top_issue_slot", variant.ID))
			}
		}
		if variant.SeedFromReference && len(variant.ClearFields) == 0 && len(variant.ClearSlots) == 0 {
			result.Status = statusFail
			result.Errors = append(result.Errors, fmt.Sprintf("%s: seed_from_reference requires clear_fields or clear_slots", variant.ID))
		}
	}
	return result
}
