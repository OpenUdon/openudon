package synthesize

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/genelet/ramen/internal/authoring"
)

const (
	symphonyHandoffVersion       = authoring.ReviewHandoffVersion
	legacySymphonyHandoffVersion = "ramen.symphony-handoff.v1"
)

type SymphonyHandoff = authoring.ReviewHandoff
type SymphonyHandoffInput = authoring.ReviewHandoffInput
type SymphonyApprovalState = authoring.ReviewApprovalState
type SymphonyOwnerSplit = authoring.ReviewOwnerSplit
type SymphonyExecutionPolicy = authoring.ReviewExecutionPolicy
type SymphonyCredentialBindings = authoring.ReviewCredentialBindings
type SymphonyTrustedRunner = authoring.ReviewTrustedRunner

func writeSymphonyHandoff(result Result, policy projectPolicy, profile sideEffectProfile) error {
	if err := ensureArtifactDirs(result); err != nil {
		return err
	}
	manifest := buildSymphonyHandoff(result, policy, profile)
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal Symphony handoff: %w", err)
	}
	return os.WriteFile(result.SymphonyHandoffPath, append(data, '\n'), 0o644)
}

func buildSymphonyHandoff(result Result, policy projectPolicy, profile sideEffectProfile) SymphonyHandoff {
	bindingContract := authoring.BuildBindingContract(authoring.BindingContractOptions{
		BindingNames:         credentialBindingNames(policy),
		ExpectedBindingNames: expectedPlanCredentialNames(result.PlanJSONPath),
	})
	return authoring.NewReviewHandoff(authoring.ReviewHandoffOptions{
		Version:        symphonyHandoffVersion,
		GeneratedState: string(authoring.ReviewStateGenerated),
		HandoffInputs:  symphonyHandoffInputs(result),
		ApprovalStates: authoring.DefaultReviewStateMachine(),
		OwnerSplit: SymphonyOwnerSplit{
			"ramen": {
				"artifact generation",
				"deterministic validation",
				"review evidence",
				"credential-binding inventory",
				"trusted-runner command text",
			},
			"symphony": {
				"work-item routing",
				"reviewer identity",
				"audit trail",
				"workspace linkage",
				"state transitions",
				"production-execution enforcement",
			},
		},
		ExecutionPolicy:    authoring.DefaultReviewExecutionPolicy(profile.SideEffectful),
		CredentialBindings: bindingContract.ReviewCredentialBindings(),
		TrustedRunner: SymphonyTrustedRunner{
			Command:     fmt.Sprintf("./scripts/run-udon.sh %s %s", relOrAbs(filepath.Dir(result.ExampleDir), result.WorkflowPath), result.ExampleDir),
			SandboxOnly: profile.SideEffectful,
		},
	})
}

func symphonyHandoffInputs(result Result) []SymphonyHandoffInput {
	return authoring.ReviewHandoffInputsFromArtifacts([]authoring.ReviewArtifactInput{
		{Path: relOrAbs(result.ExampleDir, result.ProjectPath), Purpose: "Source brief, integration policy, runtime policy, credentials policy, safety boundary, and fallback behavior.", Required: true},
		{Path: relOrAbs(result.ExampleDir, result.IntentPath), Purpose: "Structured intent extracted from the project brief.", Required: true},
		{Path: relOrAbs(result.ExampleDir, result.WorkflowPath), Purpose: "udon workflow source produced from intent.", Required: true},
		{Path: relOrAbs(result.ExampleDir, result.UWSPath), Purpose: "Exported UWS artifact validated against the public UWS schema and udon profile checks.", Required: true},
		{Path: relOrAbs(result.ExampleDir, result.PlanJSONPath), Purpose: "Machine-readable expected steps, bindings, credentials, control flow, and side-effect hints.", Required: true},
		{Path: relOrAbs(result.ExampleDir, result.QualityJSONPath), Purpose: "Deterministic quality gate result.", Required: true},
		{Path: relOrAbs(result.ExampleDir, result.RefinementJSONPath), Purpose: "Generation/refinement attempts, failed checks, and stop reason.", Required: true},
		{Path: relOrAbs(result.ExampleDir, result.ReviewPath), Purpose: "Human review evidence, unresolved risks, skipped execution notes, and trusted-runner command text.", Required: true},
		{Path: relOrAbs(result.ExampleDir, result.SymphonyHandoffPath), Purpose: "Machine-readable XRD-005 handoff manifest for Symphony work-item routing.", Required: true},
	})
}
