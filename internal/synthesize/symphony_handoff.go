package synthesize

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const symphonyHandoffVersion = "ramen.symphony-handoff.v1"

type SymphonyHandoff struct {
	Version            string                     `json:"version"`
	GeneratedState     string                     `json:"generated_state"`
	HandoffInputs      []SymphonyHandoffInput     `json:"handoff_inputs"`
	ApprovalStates     []SymphonyApprovalState    `json:"approval_states"`
	OwnerSplit         SymphonyOwnerSplit         `json:"owner_split"`
	ExecutionPolicy    SymphonyExecutionPolicy    `json:"execution_policy"`
	CredentialBindings SymphonyCredentialBindings `json:"credential_bindings"`
	TrustedRunner      SymphonyTrustedRunner      `json:"trusted_runner"`
}

type SymphonyHandoffInput struct {
	Path     string `json:"path"`
	Purpose  string `json:"purpose"`
	Required bool   `json:"required"`
}

type SymphonyApprovalState struct {
	Name              string   `json:"name"`
	Meaning           string   `json:"meaning"`
	AllowedNextStates []string `json:"allowed_next_states,omitempty"`
}

type SymphonyOwnerSplit struct {
	Ramen    []string `json:"ramen"`
	Symphony []string `json:"symphony"`
}

type SymphonyExecutionPolicy struct {
	SideEffectful             bool   `json:"side_effectful"`
	RequiredNextState         string `json:"required_next_state,omitempty"`
	SandboxProofRunState      string `json:"sandbox_proof_run_state,omitempty"`
	ProductionExecutionState  string `json:"production_execution_state,omitempty"`
	DirectProductionExecution bool   `json:"direct_production_execution"`
}

type SymphonyCredentialBindings struct {
	Declared                 []string `json:"declared,omitempty"`
	ExpectedFromPlan         []string `json:"expected_from_plan,omitempty"`
	ValuesAllowedInArtifacts bool     `json:"values_allowed_in_artifacts"`
}

type SymphonyTrustedRunner struct {
	Command     string `json:"command"`
	SandboxOnly bool   `json:"sandbox_only"`
}

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
	requiredNext := ""
	sandboxState := ""
	productionState := ""
	if profile.SideEffectful {
		requiredNext = "review_required"
		sandboxState = "approved_for_sandbox"
		productionState = "approved_for_production"
	}
	return SymphonyHandoff{
		Version:        symphonyHandoffVersion,
		GeneratedState: "generated",
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
		ApprovalStates: []SymphonyApprovalState{
			{Name: "generated", Meaning: "Ramen emitted artifacts; no approval is implied.", AllowedNextStates: []string{"validated", "rejected"}},
			{Name: "validated", Meaning: "Required validators and quality gates passed, or known warnings are attached.", AllowedNextStates: []string{"review_required", "rejected"}},
			{Name: "review_required", Meaning: "Human review is required before side-effectful execution.", AllowedNextStates: []string{"approved_for_sandbox", "approved_for_production", "rejected"}},
			{Name: "approved_for_sandbox", Meaning: "A reviewer approved sandbox or test-endpoint execution only.", AllowedNextStates: []string{"review_required", "approved_for_production", "rejected"}},
			{Name: "approved_for_production", Meaning: "A reviewer approved production execution through a trusted runner and approved credentials.", AllowedNextStates: []string{"rejected"}},
			{Name: "rejected", Meaning: "A reviewer rejected the artifact or requested regeneration.", AllowedNextStates: []string{"generated"}},
		},
		OwnerSplit: SymphonyOwnerSplit{
			Ramen: []string{
				"artifact generation",
				"deterministic validation",
				"review evidence",
				"credential-binding inventory",
				"trusted-runner command text",
			},
			Symphony: []string{
				"work-item routing",
				"reviewer identity",
				"audit trail",
				"workspace linkage",
				"state transitions",
				"production-execution enforcement",
			},
		},
		ExecutionPolicy: SymphonyExecutionPolicy{
			SideEffectful:             profile.SideEffectful,
			RequiredNextState:         requiredNext,
			SandboxProofRunState:      sandboxState,
			ProductionExecutionState:  productionState,
			DirectProductionExecution: false,
		},
		CredentialBindings: SymphonyCredentialBindings{
			Declared:                 credentialBindingNames(policy),
			ExpectedFromPlan:         expectedPlanCredentialNames(result.PlanJSONPath),
			ValuesAllowedInArtifacts: false,
		},
		TrustedRunner: SymphonyTrustedRunner{
			Command:     fmt.Sprintf("./scripts/run-udon.sh %s %s", relOrAbs(filepath.Dir(result.ExampleDir), result.WorkflowPath), result.ExampleDir),
			SandboxOnly: profile.SideEffectful,
		},
	}
}
