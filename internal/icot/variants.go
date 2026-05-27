package icot

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	evalpkg "github.com/OpenUdon/openudon/internal/eval"
)

const (
	variantsValidationReportVersion = "openudon.icot-variants-validation.v1"
	variantsCoverageReportVersion   = "openudon.icot-variants-coverage.v1"
)

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

type variantsCoverageReport struct {
	Version string                   `json:"version"`
	Status  string                   `json:"status"`
	Root    string                   `json:"root"`
	Results []variantsCoverageResult `json:"results"`
	Errors  []string                 `json:"errors,omitempty"`
}

type variantsCoverageResult struct {
	ProviderFamily string   `json:"provider_family"`
	Status         string   `json:"status"`
	Positive       int      `json:"positive"`
	MissingDetail  int      `json:"missing_detail"`
	UnsafeNegative int      `json:"unsafe_negative"`
	Fixtures       []string `json:"fixtures,omitempty"`
	Errors         []string `json:"errors,omitempty"`
}

func runVariants(args []string, out, errOut io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(errOut, "Usage: icot variants <validate|coverage> [--root examples/eval] [--name fixture] [--json]")
		return 2
	}
	switch args[0] {
	case "validate":
		return runVariantsValidate(args[1:], out, errOut)
	case "coverage":
		return runVariantsCoverage(args[1:], out, errOut)
	default:
		fmt.Fprintln(errOut, "Usage: icot variants <validate|coverage> [--root examples/eval] [--name fixture] [--json]")
		return 2
	}
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

func runVariantsCoverage(args []string, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("icot variants coverage", flag.ContinueOnError)
	fs.SetOutput(out)
	root := fs.String("root", "examples/eval", "Directory containing eval example subdirectories")
	name := fs.String("name", "", "Check one eval fixture by directory name")
	jsonOutput := fs.Bool("json", false, "Write openudon.icot-variants-coverage.v1 JSON to stdout")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: icot variants coverage [--root examples/eval] [--name fixture] [--json]\n\n")
		fmt.Fprintf(fs.Output(), "Checks provider-family coverage for positive, missing-detail, and unsafe-negative authoring variants.\n\n")
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
	report := buildVariantsCoverageReport(*root, fixtures)
	if !*jsonOutput {
		for _, result := range report.Results {
			if result.Status == statusPass {
				fmt.Fprintf(out, "icot variants coverage: pass %s (positive=%d missing-detail=%d unsafe-negative=%d)\n", result.ProviderFamily, result.Positive, result.MissingDetail, result.UnsafeNegative)
			} else {
				fmt.Fprintf(out, "icot variants coverage: fail %s - %s\n", result.ProviderFamily, strings.Join(result.Errors, "; "))
			}
		}
		for _, err := range report.Errors {
			fmt.Fprintf(out, "icot variants coverage: fail - %s\n", err)
		}
	}
	if len(report.Results) == 0 && len(report.Errors) == 0 {
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

func buildVariantsCoverageReport(root string, fixtures []string) variantsCoverageReport {
	report := variantsCoverageReport{
		Version: variantsCoverageReportVersion,
		Status:  statusPass,
		Root:    root,
	}
	byProvider := map[string]*variantsCoverageResult{}
	fixtureSeen := map[string]map[string]bool{}
	for _, fixture := range fixtures {
		fixtureName := filepath.Base(filepath.Clean(fixture))
		path := filepath.Join(fixture, "reference", "authoring-variants.json")
		variants, err := evalpkg.ReadAuthoringVariants(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			report.Errors = append(report.Errors, fmt.Sprintf("%s: %v", fixtureName, err))
			report.Status = statusFail
			continue
		}
		if len(variants.ProviderFamilies) == 0 {
			report.Errors = append(report.Errors, fmt.Sprintf("%s: authoring variants missing provider_families", fixtureName))
			report.Status = statusFail
			continue
		}
		for _, provider := range variants.ProviderFamilies {
			result := byProvider[provider]
			if result == nil {
				result = &variantsCoverageResult{
					ProviderFamily: provider,
					Status:         statusPass,
				}
				byProvider[provider] = result
				fixtureSeen[provider] = map[string]bool{}
			}
			if !fixtureSeen[provider][fixtureName] {
				result.Fixtures = append(result.Fixtures, fixtureName)
				fixtureSeen[provider][fixtureName] = true
			}
			for _, variant := range variants.Variants {
				switch variant.Class {
				case "positive":
					result.Positive++
				case "missing-detail":
					result.MissingDetail++
				case "unsafe-negative":
					result.UnsafeNegative++
				}
			}
		}
	}
	var providers []string
	for provider := range byProvider {
		providers = append(providers, provider)
	}
	sort.Strings(providers)
	for _, provider := range providers {
		result := byProvider[provider]
		sort.Strings(result.Fixtures)
		if result.Positive == 0 {
			result.Errors = append(result.Errors, "missing positive variant")
		}
		if result.MissingDetail == 0 {
			result.Errors = append(result.Errors, "missing missing-detail variant")
		}
		if result.UnsafeNegative == 0 {
			result.Errors = append(result.Errors, "missing unsafe-negative variant")
		}
		if len(result.Errors) > 0 {
			result.Status = statusFail
			report.Status = statusFail
		}
		report.Results = append(report.Results, *result)
	}
	if len(report.Errors) > 0 {
		report.Status = statusFail
	}
	return report
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
