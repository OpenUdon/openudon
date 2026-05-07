package synthesize

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/OpenUdon/apitools"
)

const (
	symphonyHandoffVersion       = apitools.ReviewHandoffVersion
	legacySymphonyHandoffVersion = "ramen.symphony-handoff.v1"
)

type SymphonyHandoff = apitools.ReviewHandoff
type SymphonyHandoffInput = apitools.ReviewHandoffInput
type SymphonyApprovalState = apitools.ReviewApprovalState
type SymphonyOwnerSplit = apitools.ReviewOwnerSplit
type SymphonyExecutionPolicy = apitools.ReviewExecutionPolicy
type SymphonyCredentialBindings = apitools.ReviewCredentialBindings
type SymphonyTrustedRunner = apitools.ReviewTrustedRunner

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
	return apitools.NewReviewHandoff(apitools.ReviewHandoffOptions{
		Version:        symphonyHandoffVersion,
		GeneratedState: string(apitools.ReviewStateGenerated),
		HandoffInputs: []SymphonyHandoffInput{
			{Path: relOrAbs(result.ExampleDir, result.ProjectPath), Purpose: "Source brief, integration policy, runtime policy, credentials policy, safety boundary, and fallback behavior.", Required: true},
			{Path: relOrAbs(result.ExampleDir, result.IntentPath), Purpose: "Structured intent extracted from the project brief.", Required: true},
			{Path: relOrAbs(result.ExampleDir, result.WorkflowPath), Purpose: "udon workflow source produced from intent.", Required: true},
			{Path: relOrAbs(result.ExampleDir, result.UWSPath), Purpose: "Exported UWS artifact validated against the public UWS schema and udon profile checks.", Required: true},
			{Path: relOrAbs(result.ExampleDir, result.PlanJSONPath), Purpose: "Machine-readable expected steps, bindings, credentials, control flow, and side-effect hints.", Required: true},
			{Path: relOrAbs(result.ExampleDir, result.QualityJSONPath), Purpose: "Deterministic quality gate result.", Required: true},
			{Path: relOrAbs(result.ExampleDir, result.RefinementJSONPath), Purpose: "Generation/refinement attempts, failed checks, and stop reason.", Required: true},
			{Path: relOrAbs(result.ExampleDir, result.ReviewPath), Purpose: "Human review evidence, unresolved risks, skipped execution notes, and trusted-runner command text.", Required: true},
			{Path: relOrAbs(result.ExampleDir, result.SymphonyHandoffPath), Purpose: "Machine-readable XRD-005 handoff manifest for Symphony work-item routing.", Required: true},
		},
		ApprovalStates: apitools.DefaultReviewStateMachine(),
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
		ExecutionPolicy: apitools.DefaultReviewExecutionPolicy(profile.SideEffectful),
		CredentialBindings: SymphonyCredentialBindings{
			Declared:                 credentialBindingNames(policy),
			ExpectedFromPlan:         expectedPlanCredentialNames(result.PlanJSONPath),
			ValuesAllowedInArtifacts: false,
		},
		TrustedRunner: SymphonyTrustedRunner{
			Command:     fmt.Sprintf("./scripts/run-udon.sh %s %s", relOrAbs(filepath.Dir(result.ExampleDir), result.WorkflowPath), result.ExampleDir),
			SandboxOnly: profile.SideEffectful,
		},
	})
}
