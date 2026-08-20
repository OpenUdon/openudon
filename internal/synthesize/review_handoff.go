package synthesize

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/OpenUdon/openudon/internal/authoring"
	"github.com/OpenUdon/openudon/internal/authoring/atomicfile"
	"github.com/OpenUdon/openudon/internal/evidencefile"
	"github.com/OpenUdon/openudon/internal/packageartifacts"
)

const reviewHandoffVersion = authoring.ReviewHandoffVersion

type ReviewHandoff = authoring.ReviewHandoff
type ReviewHandoffInput = authoring.ReviewHandoffInput
type ReviewApprovalState = authoring.ReviewApprovalState
type ReviewOwnerSplit = authoring.ReviewOwnerSplit
type ReviewExecutionPolicy = authoring.ReviewExecutionPolicy
type ReviewCredentialBindings = authoring.ReviewCredentialBindings
type ReviewTrustedRunner = authoring.ReviewTrustedRunner

func writeReviewHandoff(result Result, policy projectPolicy, profile sideEffectProfile) error {
	expectedCredentials := expectedPlanCredentialNames(result.PlanJSONPath)
	return writeReviewHandoffWithCredentials(result, policy, profile, expectedCredentials)
}

func writeReviewHandoffWithCredentials(result Result, policy projectPolicy, profile sideEffectProfile, expectedCredentials []string) error {
	if err := ensureArtifactDirs(result); err != nil {
		return err
	}
	manifest, err := buildReviewHandoffWithCredentials(result, policy, profile, expectedCredentials)
	if err != nil {
		return err
	}
	manifest, err = bindReviewHandoffDigests(result, manifest)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal review handoff: %w", err)
	}
	return atomicfile.Write(result.ReviewHandoffPath, append(data, '\n'), 0o644)
}

func bindReviewHandoffDigests(result Result, manifest ReviewHandoff) (ReviewHandoff, error) {
	selfPath := filepath.ToSlash(relOrAbs(result.ExampleDir, result.ReviewHandoffPath))
	for i := range manifest.HandoffInputs {
		input := &manifest.HandoffInputs[i]
		if input.Path == selfPath {
			continue
		}
		path := filepath.Join(result.ExampleDir, filepath.FromSlash(input.Path))
		data, _, err := evidencefile.ReadRegular(path, evidencefile.DefaultMaxBytes)
		if err != nil {
			if os.IsNotExist(err) && (input.Path == "expected/quality.json" || input.Path == "expected/refinement.json") {
				input.SHA256 = fmt.Sprintf("%064x", 0)
				continue
			}
			return ReviewHandoff{}, fmt.Errorf("read review handoff input %s: %w", input.Path, err)
		}
		input.SHA256 = evidencefile.SHA256(data)
	}
	selfDigest, err := authoring.ReviewHandoffSelfDigest(manifest, selfPath)
	if err != nil {
		return ReviewHandoff{}, err
	}
	for i := range manifest.HandoffInputs {
		if manifest.HandoffInputs[i].Path == selfPath {
			manifest.HandoffInputs[i].SHA256 = selfDigest
		}
	}
	return manifest, nil
}

func refreshReviewHandoffDigests(result Result) error {
	data, _, err := evidencefile.ReadRegular(result.ReviewHandoffPath, evidencefile.DefaultMaxBytes)
	if err != nil {
		return err
	}
	var manifest ReviewHandoff
	if err := evidencefile.DecodeStrict(data, &manifest); err != nil {
		return fmt.Errorf("parse review handoff for digest refresh: %w", err)
	}
	manifest, err = bindReviewHandoffDigests(result, manifest)
	if err != nil {
		return err
	}
	data, err = json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return atomicfile.Write(result.ReviewHandoffPath, append(data, '\n'), 0o644)
}

func buildReviewHandoff(result Result, policy projectPolicy, profile sideEffectProfile) (ReviewHandoff, error) {
	return buildReviewHandoffWithCredentials(result, policy, profile, expectedPlanCredentialNames(result.PlanJSONPath))
}

func buildReviewHandoffWithCredentials(result Result, policy projectPolicy, profile sideEffectProfile, expectedCredentials []string) (ReviewHandoff, error) {
	bindingContract := authoring.BuildBindingContract(authoring.BindingContractOptions{
		BindingNames:         credentialBindingNames(policy),
		ExpectedBindingNames: append([]string(nil), expectedCredentials...),
	})
	inputs, err := reviewHandoffInputs(result)
	if err != nil {
		return ReviewHandoff{}, err
	}
	return authoring.NewReviewHandoff(authoring.ReviewHandoffOptions{
		Version:        reviewHandoffVersion,
		GeneratedState: string(authoring.ReviewStateGenerated),
		HandoffInputs:  inputs,
		ApprovalStates: authoring.DefaultReviewStateMachine(),
		OwnerSplit: ReviewOwnerSplit{
			"openudon": {
				"artifact generation",
				"deterministic validation",
				"review evidence",
				"credential-binding inventory",
				"trusted-runner command text",
			},
			"external_review_orchestration": {
				"review routing",
				"reviewer identity",
				"audit trail",
				"workspace linkage",
				"state transitions",
				"production-execution enforcement",
			},
		},
		ExecutionPolicy:    authoring.DefaultReviewExecutionPolicy(profile.SideEffectful),
		CredentialBindings: bindingContract.ReviewCredentialBindings(),
		TrustedRunner: ReviewTrustedRunner{
			Command:     fmt.Sprintf("openudon run --example %s --tier sandbox --approval approvals/%s.json", relOrAbs(filepath.Dir(result.ExampleDir), result.ExampleDir), filepath.Base(result.ExampleDir)),
			SandboxOnly: profile.SideEffectful,
		},
	}), nil
}

func reviewHandoffInputs(result Result) ([]ReviewHandoffInput, error) {
	artifacts := []authoring.ReviewArtifactInput{
		{Path: relOrAbs(result.ExampleDir, result.ProjectPath), Purpose: "Source brief, integration policy, runtime policy, credentials policy, safety boundary, and fallback behavior.", Required: true},
		{Path: relOrAbs(result.ExampleDir, result.IntentPath), Purpose: "Structured intent extracted from the project brief.", Required: true},
		{Path: relOrAbs(result.ExampleDir, result.WorkflowPath), Purpose: "Public UWS HCL document produced from intent.", Required: true},
		{Path: relOrAbs(result.ExampleDir, result.UWSPath), Purpose: "Public UWS YAML artifact validated against the public UWS schema and local execution-profile checks.", Required: true},
		{Path: relOrAbs(result.ExampleDir, result.PlanJSONPath), Purpose: "Machine-readable expected steps, bindings, credentials, control flow, and side-effect hints.", Required: true},
		{Path: relOrAbs(result.ExampleDir, result.QualityJSONPath), Purpose: "Deterministic quality gate result.", Required: true},
		{Path: relOrAbs(result.ExampleDir, result.RefinementJSONPath), Purpose: "Generation/refinement attempts, failed checks, and stop reason.", Required: true},
		{Path: relOrAbs(result.ExampleDir, result.ReviewPath), Purpose: "Human review evidence, unresolved risks, skipped execution notes, and trusted-runner command text.", Required: true},
		{Path: relOrAbs(result.ExampleDir, result.ReviewHandoffPath), Purpose: "Machine-readable review handoff manifest for reviewer or orchestrator routing.", Required: true},
	}
	if runtimeDataPathExists(result) {
		artifacts = append(artifacts, authoring.ReviewArtifactInput{
			Path:     packageartifacts.RuntimeDataPath,
			Purpose:  "Reviewed non-secret runtime input values staged with the trusted executor package.",
			Required: true,
		})
	}
	openAPIPaths, err := packageartifacts.CollectAPISourcePaths(result.ExampleDir)
	if err != nil {
		return nil, err
	}
	for _, path := range openAPIPaths {
		artifacts = append(artifacts, authoring.ReviewArtifactInput{
			Path:     path,
			Purpose:  "Reviewed API source contract staged with the trusted executor package.",
			Required: true,
		})
	}
	browserPaths, err := packageartifacts.CollectBrowserProfilePaths(result.ExampleDir)
	if err != nil {
		return nil, err
	}
	for _, path := range browserPaths {
		artifacts = append(artifacts, authoring.ReviewArtifactInput{
			Path:     path,
			Purpose:  "Verified browser capability profile staged with the trusted Udon package.",
			Required: true,
		})
	}
	if len(browserPaths) > 0 {
		artifacts = append(artifacts, authoring.ReviewArtifactInput{
			Path:     packageartifacts.BrowserSourceReviewPath,
			Purpose:  "Browser source digest, origin, lifecycle, session-posture, and mutation-approval review evidence.",
			Required: true,
		})
	}
	authenticationPaths, err := packageartifacts.CollectBrowserAuthenticationProfilePaths(result.ExampleDir)
	if err != nil {
		return nil, err
	}
	for _, path := range authenticationPaths {
		artifacts = append(artifacts, authoring.ReviewArtifactInput{
			Path: path, Purpose: "Portable secret-free browser authentication profile staged for the trusted Udon runtime.", Required: true,
		})
	}
	if len(authenticationPaths) > 0 {
		artifacts = append(artifacts, authoring.ReviewArtifactInput{
			Path:    packageartifacts.BrowserAuthenticationReviewPath,
			Purpose: "Authentication profile digest, flow, named-session, symbolic-credential, and authoring-approval review evidence.", Required: true,
		})
	}
	securitySidecars, err := packageartifacts.CollectAdvisorySecuritySidecarPaths(result.ExampleDir)
	if err != nil {
		return nil, err
	}
	for _, path := range securitySidecars {
		artifacts = append(artifacts, authoring.ReviewArtifactInput{
			Path:     path,
			Purpose:  "Advisory security metadata used for build credential review.",
			Required: true,
		})
	}
	return authoring.ReviewHandoffInputsFromArtifacts(artifacts), nil
}
