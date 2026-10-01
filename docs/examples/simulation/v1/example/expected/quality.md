# OpenUdon Quality Report

Status: `fail`

- `project.present` pass - project.md is readable
- `project.authoring.structured_policy` pass - openudon-policy block controls are valid
- `project.authoring.goal` pass - project.md declares the workflow goal
- `project.authoring.integration_policy` warn - External Systems and OpenAPI is missing
  Detail: Add this section to make synthesis decisions auditable.
- `project.authoring.data_flow` warn - Data Flow is missing
  Detail: Add this section when the workflow has multiple steps or fnct adapters.
- `project.authoring.credentials` warn - Credentials and Secrets is missing
  Detail: Add this section when the workflow has multiple steps or fnct adapters.
- `project.authoring.runtime_policy` warn - Runtime Policy is missing
  Detail: Add this section to make synthesis decisions auditable.
- `project.authoring.function_contracts` warn - Function Contracts is missing
  Detail: Add this section when the workflow has multiple steps or fnct adapters.
- `project.authoring.safety` warn - Safety and Approval Boundary is missing
  Detail: Add this section to make synthesis decisions auditable.
- `project.authoring.fallback` warn - Fallback Behavior is missing
  Detail: Add this section to make synthesis decisions auditable.
- `openapi.local` fail - no local API source documents are available
  Detail: Add a valid OpenAPI document under openapi/ or a first-class source under google-discovery/, aws-smithy/, asyncapi/, graphql/, openrpc/, grpc-protobuf/, or odata/.
- `plan.version` pass - expected workflow plan version is supported
- `plan.gaps` pass - expected workflow plan has no unresolved gaps
- `openapi.discovery` warn - OpenAPI discovery report is missing
  Detail: Run `openudon synthesize` or `openudon build` to record OpenAPI discovery attempts.
- `intent.parse` pass - intent.hcl parses
- `intent.slots` pass - intent.hcl has required slots
- `intent.openapi_refs` pass - intent.hcl API source references are available
- `intent.openapi_operations` pass - intent.hcl API source operation references are available
- `intent.data_flow.required_params` pass - required OpenAPI parameters are satisfied or credential-bound
- `credentials.bindings` pass - credential-like parameters are covered by project credential policy or not required
- `credentials.security_schemes` pass - API source security requirements are covered by credential policy or not required
- `intent.data_flow.sources` pass - intent.hcl data-flow references resolve to known steps or inputs
- `intent.data_flow.response_paths` pass - intent.hcl response paths match available API source response schemas
- `intent.data_flow.explicit` pass - single-step intent does not require cross-step data-flow evidence
- `intent.function_contracts` pass - fnct steps match declared project function contracts
- `intent.runtime_policy` pass - intent.hcl respects project runtime policy
- `intent.project_policy` pass - intent.hcl preserves required project controls
- `workflow.present` pass - workflow.hcl is readable
- `workflow.hcl_syntax` pass - workflow.hcl syntax is valid
- `workflow.uws_parse` pass - workflow.hcl parses as a public UWS document
- `workflow.pending_steps` fail - workflow.hcl contains unresolved pending contracts
  Detail: 1 pending step(s); approval and execution refused
- `uws.present` pass - workflow.uws.yaml is present
- `uws.schema` pass - workflow.uws.yaml validates against public UWS schema
- `uws.pending_steps` fail - workflow.uws.yaml contains unresolved pending contracts
  Detail: 1 pending step(s); simulation only, approval and execution refused
- `browser.sources` pass - browser source review is not required
- `browser.authentication.sources` pass - browser authentication review is not required
- `browser.registration.sources` pass - browser registration review is not required
- `side_effects.policy` pass - no side-effectful workflow behavior inferred
- `side_effects.retry_policy` pass - retry action policy is not required
- `review.execution_boundary` pass - review evidence records skipped side-effectful execution
- `review.package` pass - review evidence lists the minimum trusted-execution review package
- `review.trusted_runner` pass - review evidence includes trusted-runner handoff command
- `review.production_boundary` pass - review evidence records the production execution boundary
- `review.credential_audit` pass - review evidence records credential binding audit requirements
- `review.approval_states` pass - review evidence records approval-state requirements
- `review.sandbox_handoff` pass - review evidence scopes trusted-runner handoff to the approved execution state
- `review.credential_bindings` pass - review evidence records credential-binding inventory
- `review.side_effect_summary` pass - review evidence summarizes inferred side effects
- `review.side_effect_risk` pass - review evidence includes side-effect risk review
- `review.approval_artifact` pass - review evidence describes approval artifact requirements
- `review.credential_scope` pass - review evidence includes credential scope matrix
- `review.trusted_runner_dry_run` pass - review evidence includes trusted-runner dry-run evidence
- `review.unresolved_risks` pass - review evidence records unresolved risks
- `review_handoff.present` pass - review handoff manifest is readable
- `review_handoff.contract` pass - review handoff manifest records package, state, execution, and credential contracts
- `artifacts.no_secrets` pass - no obvious secret-like tokens found in artifacts
- `conversion.diagnostics` pass - no conversion diagnostics artifact is present
