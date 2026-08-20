// Package qualityremediation maps deterministic quality failures to concise
// operator remediation without coupling that policy to CLI presentation.
package qualityremediation

import "strings"

func NextAction(code string) string {
	switch {
	case code == "project.present":
		return "Create project.md from templates/project.md, then rerun synthesize or assess."
	case strings.HasPrefix(code, "project.authoring."):
		return "Fill the missing project.md section so synthesis decisions are auditable."
	case strings.HasPrefix(code, "openapi."):
		return "Add or fix OpenAPI documents under openapi/ with operation IDs, request fields, response schemas, and security schemes; or declare OpenAPI: none required when no API is needed."
	case code == "plan.gaps":
		return "Resolve missing operations, required parameters, or credential bindings in project.md or intent.hcl."
	case code == "intent.data_flow.required_params":
		return "Map every required OpenAPI path, query, header, or body field to an input, safe literal, prior-step bind, or credential binding name; document SaaS request mappings in Data Flow."
	case code == "intent.data_flow.response_paths":
		return "Use response fields present in the OpenAPI schema or update Outputs and Data Flow; avoid guessing SaaS response paths."
	case code == "intent.data_flow.explicit":
		return "Add Data Flow guidance with request field sources, prior-step bindings, credential binding names, and final output sources."
	case code == "intent.openapi_operations":
		return "Select only operationId values listed in local OpenAPI documents and document unresolved SaaS capability gaps instead of inventing provider operations."
	case strings.HasPrefix(code, "intent."):
		return "Inspect workflows/intent.hcl and project.md; rerun synthesize when the brief needs regeneration."
	case code == "credentials.security_schemes":
		return "Declare symbolic credential binding names for required OpenAPI security schemes in project.md, then rerun synthesize or build."
	case code == "credentials.bindings", code == "workflow.credentials_bound":
		return "Name runtime credential bindings in project.md and ensure workflow request fields reference binding names, never secret values."
	case strings.HasPrefix(code, "workflow."):
		return "Inspect workflows/workflow.hcl against expected/plan.md, then rerun build or synthesize."
	case strings.HasPrefix(code, "uws."):
		return "Inspect workflows/workflow.uws.yaml, then rerun promote or build after fixing workflow.hcl."
	case code == "side_effects.environment":
		return "Use sandbox/test endpoints or add explicit production handoff approval language to Safety and Approval Boundary."
	case code == "side_effects.policy":
		return "Add approval, trusted-runtime, and sandbox proof-run policy to Safety and Approval Boundary."
	case code == "review.credential_bindings":
		return "Update Credentials and Secrets with binding names only, then regenerate review evidence with build/synthesize."
	case code == "review.approval_states":
		return "State generated, review_required, approved_for_sandbox, and approved_for_production approval requirements in review evidence."
	case code == "review.sandbox_handoff":
		return "Scope trusted-runner handoff to approved sandbox or proof runs before production handoff."
	case code == "review.trusted_runner":
		return "Regenerate review evidence so expected/review.md includes the trusted-runner handoff command."
	case code == "review.trusted_runner_dry_run":
		return "Regenerate review evidence so expected/review.md includes the trusted-runner dry-run command and run-config boundary."
	case code == "review.production_boundary":
		return "Regenerate review evidence so it states OpenUdon synthesis does not directly execute production workflows."
	case code == "review.approval_artifact":
		return "Regenerate review evidence so it describes approval JSON fields, tier state, expiry, and package_sha256 requirements."
	case code == "review.credential_scope":
		return "Regenerate review evidence so it includes the credential scope matrix for declared and expected bindings."
	case code == "review.side_effect_risk":
		return "Regenerate review evidence so it lists side-effect risk and approved sandbox/production handoff states."
	case strings.HasPrefix(code, "review."):
		return "Update Safety and Approval Boundary or regenerate review evidence with build/synthesize."
	case strings.HasPrefix(code, "review_handoff."):
		return "Regenerate expected/review-handoff.json with build/synthesize so the review handoff approval contract can be checked."
	case code == "artifacts.no_secrets":
		return "Remove literal secret-like values from artifacts; keep only credential binding names."
	default:
		return "Inspect expected/quality.md for details, fix the referenced artifact, and rerun assess."
	}
}
