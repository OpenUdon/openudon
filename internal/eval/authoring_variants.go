package eval

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/OpenUdon/openudon/internal/icotreport"
)

const AuthoringVariantsVersion = "openudon.icot-authoring-variants.v1"

type AuthoringVariants struct {
	Version          string             `json:"version"`
	ProviderFamilies []string           `json:"provider_families,omitempty"`
	Variants         []AuthoringVariant `json:"variants"`
}

type AuthoringVariant struct {
	ID                    string   `json:"id"`
	Brief                 string   `json:"brief"`
	Class                 string   `json:"class"`
	ExpectedOutcome       string   `json:"expected_outcome"`
	ExpectedFailureFamily string   `json:"expected_failure_family,omitempty"`
	SeedFromReference     bool     `json:"seed_from_reference,omitempty"`
	ClearFields           []string `json:"clear_fields,omitempty"`
	ClearSlots            []string `json:"clear_slots,omitempty"`
	Tags                  []string `json:"tags,omitempty"`
	Reason                string   `json:"reason,omitempty"`
}

func ReadAuthoringVariants(path string) (AuthoringVariants, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return AuthoringVariants{}, err
	}
	var variants AuthoringVariants
	if err := json.Unmarshal(data, &variants); err != nil {
		return AuthoringVariants{}, fmt.Errorf("parse authoring variants: %w", err)
	}
	if err := normalizeAuthoringVariants(&variants); err != nil {
		return AuthoringVariants{}, err
	}
	return variants, nil
}

func normalizeAuthoringVariants(variants *AuthoringVariants) error {
	variants.Version = strings.TrimSpace(variants.Version)
	if variants.Version != AuthoringVariantsVersion {
		return fmt.Errorf("authoring variants version = %q, want %q", variants.Version, AuthoringVariantsVersion)
	}
	variants.ProviderFamilies = normalizeStringList(variants.ProviderFamilies)
	seen := map[string]bool{}
	for i := range variants.Variants {
		variant := &variants.Variants[i]
		variant.ID = strings.TrimSpace(variant.ID)
		variant.Brief = strings.TrimSpace(variant.Brief)
		variant.Class = normalizeAuthoringVariantClass(variant.Class)
		variant.ExpectedOutcome = normalizeAuthoringVariantOutcome(variant.ExpectedOutcome)
		variant.ExpectedFailureFamily = strings.TrimSpace(variant.ExpectedFailureFamily)
		variant.ClearFields = normalizeStringList(variant.ClearFields)
		variant.ClearSlots = normalizeStringList(variant.ClearSlots)
		variant.Tags = normalizeStringList(variant.Tags)
		variant.Reason = strings.TrimSpace(variant.Reason)
		if variant.ID == "" {
			return fmt.Errorf("authoring variant at index %d is missing id", i)
		}
		if seen[variant.ID] {
			return fmt.Errorf("duplicate authoring variant id %q", variant.ID)
		}
		seen[variant.ID] = true
		if variant.Brief == "" {
			return fmt.Errorf("authoring variant %q is missing brief", variant.ID)
		}
		if variant.Class == "" {
			return fmt.Errorf("authoring variant %q has unsupported class", variant.ID)
		}
		if variant.ExpectedOutcome == "" {
			return fmt.Errorf("authoring variant %q has unsupported expected_outcome", variant.ID)
		}
		if variant.ExpectedFailureFamily != "" && !icotreport.IsValidFailureFamily(variant.ExpectedFailureFamily) {
			return fmt.Errorf("authoring variant %q has unsupported expected_failure_family %q", variant.ID, variant.ExpectedFailureFamily)
		}
		if (len(variant.ClearFields) > 0 || len(variant.ClearSlots) > 0) && !variant.SeedFromReference {
			return fmt.Errorf("authoring variant %q uses clear_fields or clear_slots without seed_from_reference", variant.ID)
		}
	}
	if len(variants.Variants) == 0 {
		return fmt.Errorf("authoring variants file must contain at least one variant")
	}
	return nil
}

func normalizeAuthoringVariantClass(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "positive":
		return "positive"
	case "missing-detail", "missing_detail":
		return "missing-detail"
	case "unsafe-negative", "unsafe_negative", "negative":
		return "unsafe-negative"
	default:
		return ""
	}
}

func normalizeAuthoringVariantOutcome(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "pass":
		return "pass"
	case "needs_input", "needs-input":
		return "needs_input"
	case "build_fail", "build-fail", "fail":
		return "build_fail"
	case "icot_fail", "icot-fail":
		return "icot_fail"
	default:
		return ""
	}
}
