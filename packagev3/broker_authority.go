package packagev3

import (
	"context"
	"net/http"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/OpenUdon/apitools"
	"github.com/OpenUdon/openudon/authority"
)

type BrokerOptions struct {
	Execution ExecutionOptions
	// Seed supplies fresh occurrence/owner/grant/time identities only. Package,
	// input, worker and operation fields must be empty and are derived here.
	Seed authority.Authority
	// Credentials supplies symbolic revision metadata only, keyed by the exact
	// source security-scheme name. The host independently checks current custody.
	Credentials map[string]authority.Binding
	Now         time.Time
}

func DeriveBrokerAuthority(ctx context.Context, verified VerifiedPackage, options BrokerOptions) (authority.Authority, error) {
	seed := options.Seed
	if seed.Version != "" && seed.Version != authority.Version || seed.PackageSHA256 != "" || seed.HandoffSHA256 != "" || seed.InputsSHA256 != "" || seed.ExecutorSHA256 != "" || seed.PolicySHA256 != "" || len(seed.Operations) != 0 || options.Now.IsZero() {
		return authority.Authority{}, ErrPackage
	}
	plan, err := DeriveExecutionPlan(ctx, verified, options.Execution)
	if err != nil {
		return authority.Authority{}, err
	}
	if plan.AssessmentOutcome != "compatible" {
		return authority.Authority{}, ErrPackage
	}
	inventories := map[string]apitools.OperationInventory{}
	seed.Version = authority.Version
	seed.PackageSHA256 = plan.PackageSHA256
	seed.HandoffSHA256 = plan.HandoffSHA256
	seed.InputsSHA256 = plan.InputsSHA256
	seed.ExecutorSHA256 = plan.Worker.BinarySHA256
	seed.Operations = []authority.Operation{}
	usedCredentials := map[string]bool{}
	for _, leaf := range plan.Operations {
		if leaf.Kind != "http" || !leaf.MetadataComplete || len(leaf.Security.Alternatives) != 1 {
			return authority.Authority{}, ErrPackage
		}
		inventory, ok := inventories[leaf.Source.ID]
		if !ok {
			var source *Source
			for i := range verified.manifest.Sources {
				if verified.manifest.Sources[i].ID == leaf.Source.ID {
					source = &verified.manifest.Sources[i]
				}
			}
			if source == nil || source.Kind != "openapi" {
				return authority.Authority{}, ErrPackage
			}
			inventory, err = apitools.BuildOperationInventory(ctx, apitools.InventoryOptions{Documents: []apitools.InventoryDocument{{Content: verified.files[source.Artifact.Path]}}, MaxBytes: MaxFileBytes, MaxOperations: 10000})
			if err != nil || inventory.Truncated {
				return authority.Authority{}, ErrPackage
			}
			for _, diagnostic := range inventory.Diagnostics {
				if diagnostic.Severity == "error" {
					return authority.Authority{}, ErrPackage
				}
			}
			inventories[leaf.Source.ID] = inventory
		}
		var native *apitools.OperationSummary
		for i := range inventory.Operations {
			candidate := &inventory.Operations[i]
			if strings.EqualFold(candidate.Method, leaf.Method) && candidate.Path == leaf.PathTemplate {
				if native != nil {
					return authority.Authority{}, ErrPackage
				}
				native = candidate
			}
		}
		if native == nil {
			return authority.Authority{}, ErrPackage
		}
		requirements := leaf.Security.Alternatives[0].Requirements
		nativeByName := map[string]apitools.SecuritySummary{}
		if len(requirements) > 0 {
			if len(native.SecurityRequirementSets) != 1 || len(native.SecurityRequirementSets[0].Requirements) != len(requirements) {
				return authority.Authority{}, ErrPackage
			}
			for _, s := range native.SecurityRequirementSets[0].Requirements {
				nativeByName[s.Name] = s
			}
		} else {
			for _, set := range native.SecurityRequirementSets {
				if len(set.Requirements) > 0 {
					return authority.Authority{}, ErrPackage
				}
			}
		}
		operation := authority.Operation{StepID: leaf.StepID, OperationID: leaf.OperationID, InvocationID: leaf.InvocationID, Method: leaf.Method, Origin: leaf.Origin, ConstraintsSHA256: leaf.ConstraintsSHA256}
		for _, requirement := range requirements {
			if len(requirement.Scopes) != 0 {
				return authority.Authority{}, ErrPackage
			}
			native, ok := nativeByName[requirement.Scheme]
			if !ok || native.Type != requirement.Type || len(native.Scopes) != 0 {
				return authority.Authority{}, ErrPackage
			}
			binding, ok := options.Credentials[requirement.Scheme]
			if !ok || binding.Name != requirement.Scheme || binding.Validate() != nil {
				return authority.Authority{}, ErrPackage
			}
			switch {
			case requirement.Type == "apiKey":
				if native.In != requirement.Location || native.ParameterName != requirement.Name || binding.Kind != "api_key" || binding.In != requirement.Location || binding.Parameter != http.CanonicalHeaderKey(requirement.Name) && binding.In == "header" || binding.In == "query" && binding.Parameter != requirement.Name {
					return authority.Authority{}, ErrPackage
				}
			case requirement.Type == "http" && strings.EqualFold(native.Scheme, "bearer"):
				if binding.Kind != "bearer" || binding.In != "header" || binding.Parameter != "Authorization" {
					return authority.Authority{}, ErrPackage
				}
			default:
				return authority.Authority{}, ErrPackage
			}
			operation.Bindings = append(operation.Bindings, binding)
			usedCredentials[binding.Name] = true
		}
		seed.Operations = append(seed.Operations, operation)
	}
	if len(usedCredentials) != len(options.Credentials) {
		return authority.Authority{}, ErrPackage
	}
	seed.PolicySHA256 = seed.Digest()
	if seed.ValidateAt(options.Now) != nil {
		return authority.Authority{}, ErrPackage
	}
	return seed, nil
}

// CheckBrokerAuthority derives the exact current package/operation/worker
// projection before comparison. It checks supplied symbolic revisions and time;
// current host grant custody/revocation and destination policy remain host-owned.
func CheckBrokerAuthority(ctx context.Context, verified VerifiedPackage, execution ExecutionOptions, expected authority.Authority, credentials map[string]authority.Binding, now time.Time) error {
	if expected.ValidateAt(now) != nil {
		return ErrPackage
	}
	seed := expected
	seed.PackageSHA256 = ""
	seed.HandoffSHA256 = ""
	seed.InputsSHA256 = ""
	seed.ExecutorSHA256 = ""
	seed.PolicySHA256 = ""
	seed.Operations = nil
	actual, err := DeriveBrokerAuthority(ctx, verified, BrokerOptions{Execution: execution, Seed: seed, Credentials: credentials, Now: now})
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(actual, expected) {
		return ErrPackage
	}
	return nil
}

// CredentialNames reports only the independently derived review inventory.
func (v VerifiedPackage) CredentialNames() []string {
	names := append([]string{}, v.handoff.Credentials...)
	sort.Strings(names)
	return names
}
