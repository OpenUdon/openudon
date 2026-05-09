package synthesize

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/genelet/ramen/internal/authoring"
	"github.com/genelet/ramen/internal/packageartifacts"
	rollout "github.com/genelet/ramen/internal/workflowintent"
)

func assessSideEffectPolicy(report *QualityReport, policy projectPolicy, intent *rollout.Intent) {
	assessSideEffectProfile(report, sideEffectProfileFor(policy, intent))
}

func assessSideEffectProfile(report *QualityReport, profile sideEffectProfile) {
	if profile.ProductionEndpoint {
		if !profile.HasProductionPolicy {
			report.add("side_effects.environment", "fail", "production endpoint usage requires approved production handoff policy", "Add production handoff approval language to the Safety and Approval Boundary. Detected: "+strings.Join(profile.Reasons, "; "))
			return
		}
		report.add("side_effects.environment", "pass", "production endpoint usage has approved production handoff policy", "")
	}
	if !profile.SideEffectful {
		report.add("side_effects.policy", "pass", "no side-effectful workflow behavior inferred", "")
		return
	}
	if profile.HasApprovalPolicy && profile.HasSandboxPolicy {
		report.add("side_effects.policy", "pass", "side-effectful workflow has approval/trusted-runtime and sandbox proof-run policy", strings.Join(profile.Reasons, "; "))
		return
	}
	var missing []string
	if !profile.HasApprovalPolicy {
		missing = append(missing, "approval or trusted-runtime policy")
	}
	if !profile.HasSandboxPolicy {
		missing = append(missing, "sandbox/test proof-run policy")
	}
	report.add("side_effects.policy", "fail", "side-effectful workflow lacks required execution-boundary policy", "Add "+strings.Join(missing, " and ")+" to the Safety and Approval Boundary or Function Contracts section. Detected: "+strings.Join(profile.Reasons, "; "))
}

func assessSideEffectRetryPolicy(report *QualityReport, profile sideEffectProfile, policy projectPolicy, expectedPlan *WorkflowPlan) {
	if !profile.SideEffectful || !planHasRetryAction(expectedPlan) {
		report.add("side_effects.retry_policy", "pass", "retry action policy is not required", "")
		return
	}
	policyText := strings.ToLower(strings.Join([]string{
		policy.SafetySection,
		policy.FunctionSection,
		policy.RuntimeSection,
		policy.DataFlowSection,
	}, "\n"))
	if containsAny(policyText, []string{"idempotent", "idempotency", "safe to retry", "retry approved", "explicit retry", "bounded retry"}) {
		report.add("side_effects.retry_policy", "pass", "side-effectful retry actions have explicit retry/idempotency policy", "")
		return
	}
	report.add("side_effects.retry_policy", "fail", "side-effectful retry actions require explicit retry/idempotency policy", "Add retry or idempotency language to project.md before using onFailure retry actions on side-effectful workflows.")
}

func planHasRetryAction(plan *WorkflowPlan) bool {
	if plan == nil {
		return false
	}
	for _, step := range plan.Steps {
		for _, action := range step.OnFailure {
			if action != nil && strings.EqualFold(strings.TrimSpace(action.Type), "retry") {
				return true
			}
		}
	}
	return false
}

func assessReview(report *QualityReport, path string, profile sideEffectProfile, policy projectPolicy, expectedPlan *WorkflowPlan) {
	data, err := os.ReadFile(path)
	if err != nil {
		report.add("review.present", "fail", "review evidence is required", err.Error())
		return
	}
	text := string(data)
	if !strings.Contains(text, "Side-effectful execution was skipped") {
		report.add("review.execution_boundary", "fail", "review evidence must record skipped side-effectful execution", "")
		return
	}
	report.add("review.execution_boundary", "pass", "review evidence records skipped side-effectful execution", "")
	if !reviewContainsMinimumPackage(text) {
		report.add("review.package", "fail", "review evidence must list the minimum trusted-execution review package", "")
		return
	}
	report.add("review.package", "pass", "review evidence lists the minimum trusted-execution review package", "")
	if !strings.Contains(text, "Trusted proof run") || !strings.Contains(text, "ramen run") {
		report.add("review.trusted_runner", "fail", "review evidence must include trusted-runner handoff command", "")
		return
	}
	report.add("review.trusted_runner", "pass", "review evidence includes trusted-runner handoff command", "")
	if !strings.Contains(text, "Direct production execution: not performed by Ramen synthesis") {
		report.add("review.production_boundary", "fail", "review evidence must state that synthesis does not directly execute production workflows", "")
		return
	}
	report.add("review.production_boundary", "pass", "review evidence records the production execution boundary", "")
	if !strings.Contains(text, "Credential binding audit") {
		report.add("review.credential_audit", "fail", "review evidence must record credential binding audit requirements", "")
		return
	}
	report.add("review.credential_audit", "pass", "review evidence records credential binding audit requirements", "")
	if !reviewContainsApprovalStates(text, profile) {
		report.add("review.approval_states", "fail", "review evidence must state the Symphony approval states required before execution", "")
		return
	}
	report.add("review.approval_states", "pass", "review evidence records approval-state requirements", "")
	if !reviewContainsSandboxHandoff(text, profile) {
		report.add("review.sandbox_handoff", "fail", "review evidence must scope trusted-runner handoff to approved sandbox proof runs before production", "")
		return
	}
	report.add("review.sandbox_handoff", "pass", "review evidence scopes trusted-runner handoff to the approved execution state", "")
	if !reviewContainsCredentialBindings(text, policy, expectedPlan) {
		report.add("review.credential_bindings", "fail", "review evidence must list declared and expected credential bindings or explicitly state that none are required", "")
		return
	}
	report.add("review.credential_bindings", "pass", "review evidence records credential-binding inventory", "")
	if !strings.Contains(text, "## Side-Effect Summary") {
		report.add("review.side_effect_summary", "fail", "review evidence must summarize inferred side effects", "")
		return
	}
	if profile.SideEffectful && !strings.Contains(text, "Side-effectful workflow: yes") {
		report.add("review.side_effect_summary", "fail", "review evidence must mark side-effectful workflows explicitly", "")
		return
	}
	report.add("review.side_effect_summary", "pass", "review evidence summarizes inferred side effects", "")
	if !strings.Contains(text, "## Unresolved Risks") {
		report.add("review.unresolved_risks", "fail", "review evidence must record unresolved risks or lack of known risks", "")
		return
	}
	report.add("review.unresolved_risks", "pass", "review evidence records unresolved risks", "")
}

func reviewContainsMinimumPackage(text string) bool {
	if !strings.Contains(text, "## Minimum Review Package") {
		return false
	}
	required := []string{
		"Project brief",
		"Intent HCL",
		"Workflow HCL",
		"UWS artifact",
		"Expected plan",
		"Quality report",
		"Refinement report",
		"Review evidence",
		"Symphony handoff manifest",
	}
	for _, item := range required {
		if !strings.Contains(text, item) {
			return false
		}
	}
	return true
}

func assessSymphonyHandoff(report *QualityReport, path string, profile sideEffectProfile, policy projectPolicy, expectedPlan *WorkflowPlan) {
	data, err := os.ReadFile(path)
	if err != nil {
		report.add("symphony_handoff.present", "fail", "Symphony handoff manifest is required", err.Error())
		return
	}
	var manifest SymphonyHandoff
	if err := json.Unmarshal(data, &manifest); err != nil {
		report.add("symphony_handoff.present", "fail", "Symphony handoff manifest must be valid JSON", err.Error())
		return
	}
	report.add("symphony_handoff.present", "pass", "Symphony handoff manifest is readable", "")
	allowedVersions := []string{symphonyHandoffVersion, legacySymphonyHandoffVersion}
	if diagnostics := authoring.ValidateReviewHandoff(manifest, authoring.ReviewHandoffValidationOptions{AllowedVersions: allowedVersions}); len(diagnostics) > 0 {
		report.add("symphony_handoff.contract", "fail", "Symphony handoff manifest must satisfy the review handoff contract", diagnostics[0].Message)
		return
	}
	requiredOK, requiredErr := symphonyHandoffHasRequiredInputs(filepath.Dir(filepath.Dir(path)), manifest)
	if requiredErr != nil {
		report.add("symphony_handoff.contract", "fail", "Symphony handoff manifest required inputs could not be checked", requiredErr.Error())
		return
	}
	if !requiredOK {
		report.add("symphony_handoff.contract", "fail", "Symphony handoff manifest must list every required handoff input", "")
		return
	}
	if !symphonyHandoffHasApprovalStates(manifest) {
		report.add("symphony_handoff.contract", "fail", "Symphony handoff manifest must list the required approval states", "")
		return
	}
	if !symphonyHandoffExecutionPolicyMatches(manifest, profile) {
		report.add("symphony_handoff.contract", "fail", "Symphony handoff execution policy must match inferred side-effect requirements", "")
		return
	}
	if !symphonyHandoffCredentialBindingsMatch(manifest, policy, expectedPlan) {
		report.add("symphony_handoff.contract", "fail", "Symphony handoff credential bindings must match declared and expected binding names", "")
		return
	}
	if manifest.CredentialBindings.ValuesAllowedInArtifacts || manifest.ExecutionPolicy.DirectProductionExecution {
		report.add("symphony_handoff.contract", "fail", "Symphony handoff must prohibit credential values and direct production execution", "")
		return
	}
	report.add("symphony_handoff.contract", "pass", "Symphony handoff manifest records package, state, execution, and credential contracts", "")
}

func symphonyHandoffHasRequiredInputs(exampleDir string, manifest SymphonyHandoff) (bool, error) {
	_, err := packageartifacts.RequiredManifestPaths(exampleDir, packageArtifactManifestInputs(manifest))
	if err != nil {
		if strings.Contains(err.Error(), "missing required input") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func packageArtifactManifestInputs(manifest SymphonyHandoff) []packageartifacts.ManifestInput {
	inputs := make([]packageartifacts.ManifestInput, 0, len(manifest.HandoffInputs))
	for _, input := range manifest.HandoffInputs {
		inputs = append(inputs, packageartifacts.ManifestInput{
			Path:     input.Path,
			Required: input.Required,
		})
	}
	return inputs
}

func symphonyHandoffHasApprovalStates(manifest SymphonyHandoff) bool {
	return authoring.ReviewStateMachineHasRequiredStates(manifest.ApprovalStates)
}

func symphonyHandoffExecutionPolicyMatches(manifest SymphonyHandoff, profile sideEffectProfile) bool {
	policy := manifest.ExecutionPolicy
	if policy.SideEffectful != profile.SideEffectful {
		return false
	}
	if !profile.SideEffectful {
		return policy.RequiredNextState == "" && policy.SandboxProofRunState == "" && policy.ProductionExecutionState == ""
	}
	return policy.RequiredNextState == string(authoring.ReviewStateReviewRequired) &&
		policy.SandboxProofRunState == string(authoring.ReviewStateApprovedForSandbox) &&
		policy.ProductionExecutionState == string(authoring.ReviewStateApprovedForProduction) &&
		manifest.TrustedRunner.SandboxOnly
}

func symphonyHandoffCredentialBindingsMatch(manifest SymphonyHandoff, policy projectPolicy, expectedPlan *WorkflowPlan) bool {
	declared := credentialBindingNames(policy)
	expected := []string(nil)
	if expectedPlan != nil {
		seen := map[string]bool{}
		for _, step := range expectedPlan.Steps {
			for _, credential := range step.Credentials {
				credential = strings.TrimSpace(credential)
				if credential != "" {
					seen[credential] = true
				}
			}
			for _, param := range step.RequestParams {
				if !param.Credential {
					continue
				}
				for _, credential := range []string{param.ExpectedCredential, param.ExpectedSource} {
					credential = strings.TrimSpace(credential)
					if credential != "" && param.SourceKind == "credential" {
						seen[credential] = true
					}
				}
			}
		}
		for credential := range seen {
			expected = append(expected, credential)
		}
		sort.Strings(expected)
	}
	return stringSlicesEqual(declared, manifest.CredentialBindings.Declared) &&
		stringSlicesEqual(expected, manifest.CredentialBindings.ExpectedFromPlan)
}

func stringSlicesEqual(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func reviewContainsApprovalStates(text string, profile sideEffectProfile) bool {
	if !strings.Contains(text, "## Approval State Requirements") || !strings.Contains(text, "`generated`") {
		return false
	}
	for _, state := range []string{"`validated`", "`review_required`", "`approved_for_sandbox`", "`approved_for_production`", "`rejected`"} {
		if !strings.Contains(text, state) {
			return false
		}
	}
	if !profile.SideEffectful {
		return strings.Contains(text, "not required unless future changes add side effects")
	}
	return true
}

func reviewContainsSandboxHandoff(text string, profile sideEffectProfile) bool {
	if !strings.Contains(text, "## Trusted Execution Handoff") {
		return false
	}
	if !profile.SideEffectful {
		return strings.Contains(text, "Sandbox/test proof run is optional unless future changes add side effects")
	}
	return strings.Contains(text, "Trusted proof run command is for sandbox/test execution only after `approved_for_sandbox`") &&
		strings.Contains(text, "Production execution requires `approved_for_production`")
}

func reviewContainsCredentialBindings(text string, policy projectPolicy, expectedPlan *WorkflowPlan) bool {
	if !strings.Contains(text, "## Credential Binding Audit") {
		return false
	}
	expected := map[string]bool{}
	for _, credential := range credentialBindingNames(policy) {
		expected[credential] = true
	}
	if expectedPlan != nil {
		for _, step := range expectedPlan.Steps {
			for _, credential := range step.Credentials {
				credential = strings.TrimSpace(credential)
				if credential != "" {
					expected[credential] = true
				}
			}
			for _, param := range step.RequestParams {
				if !param.Credential {
					continue
				}
				for _, credential := range []string{param.ExpectedCredential, param.ExpectedSource} {
					credential = strings.TrimSpace(credential)
					if credential != "" && param.SourceKind == "credential" {
						expected[credential] = true
					}
				}
			}
		}
	}
	if len(expected) == 0 {
		return strings.Contains(text, "No credential bindings declared or required.")
	}
	for credential := range expected {
		if !strings.Contains(text, "`"+credential+"`") {
			return false
		}
	}
	return true
}
