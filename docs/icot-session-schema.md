# iCoT v2 Session Files

`icot --answers <file>` accepts YAML or JSON using the durable
`openudon.icot-session.v2` contract. v1 sessions and the old brief-only answers
shape are rejected with a version error; there is no compatibility decoder.

The session records one confirmed active workflow, the generic dependency graph
used by the interview, a unified evidence ledger, reviewed source plans, and
future candidate workflows that are deliberately not implemented.

```yaml
version: openudon.icot-session.v2
boundary:
  outcome: Resolve Toronto weather and prepare a reviewed report.
  actor: operator
  trigger: on demand
  success_evidence:
    - output report is produced from render_report.received_body
  non_goals:
    - sending email
  confirmed: true
interview:
  version: authoring.interview.v1
  round: 2
  nodes:
    - id: boundary.outcome
      title: Active outcome
      status: settled
      required: true
    - id: source.selection
      title: API source
      status: deferred
      dependencies: [boundary.outcome]
      deferrable: true
  evidence:
    - id: evidence.boundary
      kind: user_decision
      node_id: boundary.outcome
      summary: The operator selected the weather-report workflow.
      value: Resolve Toronto weather and prepare a reviewed report.
      source: operator
      attributes:
        confidence: high
  deferrals:
    - id: deferral.source
      node_id: source.selection
      owner: API maintainer
      impact: Provider operation selection cannot be verified.
      unblock_condition: A reviewed API document is available.
      suggested_next_action: Rerun with --api-source or --source-root.
project:
  project_name: Toronto Weather Report
  goal: Resolve Toronto weather and prepare a reviewed report.
  side_effect_scope: read-only
  safety: Generate and validate artifacts only.
  fallback: Stop if geocoding or weather lookup fails.
intent:
  workflow:
    name: toronto_weather_report
    description: Resolve Toronto weather and prepare a reviewed report.
  steps:
    - name: render_report
      type: fnct
      do: Render the reviewed weather report.
  outputs:
    - name: report
      from: render_report.received_body
source_plan:
  - kind: browser-profile
    source_kind: capability_bundle
    id: status
    release: 1.0.0
    source_path: https://profiles.example.org/blobs/sha256/...
    target_path: browser-profiles/status.json
    sha256: 0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef
    source_sha256: abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789
    title: Reviewed Status UI
    operation_count: 1
    actions: [read_status]
    origins: [https://status.example.org]
    lifecycle: active
    expires_at: 2026-09-14T00:00:00Z
    provenance: reviewed registry contribution
    registry: https://profiles.example.org/
    registry_coordinate: example/status@1.0.0
    browser_verifications:
      - source_path: /reviewed/status.live-check.json
        summary:
          report_version: browsertools.live-check.v1
          source_sha256: sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef
          profile_digest: sha256:abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789
          checked_at: 2026-08-16T12:00:00Z
          origin: https://status.example.org
          actions: [read_status]
          ok: true
          engine: chromium
          checks:
            - kind: output
              path: actions.read_status.outputs.status
              ok: true
              matches: 1
              expectedType: string
              observedType: string
              message: declared output source and JSON type matched
  - kind: browser-authentication
    source_kind: browser_authentication_profile
    id: member-auth
    source_path: /reviewed/member-auth.yaml
    target_path: browser-authentication/member-auth.yaml
    sha256: 123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0
    title: Reviewed Member Sign-In
    operation_count: 1
    flows: [member_login_push]
    flow_credential_slots:
      member_login_push: [username, password]
    origins: [https://members.example.org, https://login.example.org]
    lifecycle: active
    expires_at: 2026-09-14T00:00:00Z
    provenance: reviewed local observation
browser_route: browser
browser_session_posture: none
browser_mutation_approvals: []
browser_authentication_approvals: [authenticate_member]
fallback: Stop if geocoding or weather lookup fails.
fallback_set: true
side_effect_scope: read-only
candidate_workflows:
  - title: Email Weather Report
    outcome: Send the reviewed report through Gmail.
    deferral_reason: Delivery is outside the active workflow boundary.
    promotion_trigger: The reporting workflow and Gmail approval posture are approved.
```

## Contract

- `boundary` must contain the active outcome, actor, trigger, success evidence,
  non-goals, and confirmation state. Boundary and side-effect posture cannot be
  deferred.
- `interview` uses `authoring.interview.v1`. Node states are `open`, `settled`,
  `deferred`, and `inapplicable`. Dependencies must exist and be acyclic.
- `interview.evidence` is the only durable decision ledger. Evidence kinds are
  `observed_fact`, `user_decision`, `recommendation`, `assumption`,
  `open_decision`, `deferral`, and `inapplicable_branch`. It contains concise
  public rationale, never hidden chain-of-thought. Optional string-valued
  `attributes` retain machine-readable confidence and confirmation qualifiers
  required to reproduce readiness and safety checks after resume.
- A technical deferral must name its owner, impact, unblock condition, and next
  action. Source, operation, mapping, and output leaves may be deferred.
- `candidate_workflows` are unnumbered future directions. Each has a title,
  outcome, deferral reason, and promotion trigger, and must not contain sources,
  operations, mappings, or implementation steps.
- `source_plan` entries, when present, include kind, stable ID, inspected source
  path, package target, SHA-256 digest, title/operation count, and provenance.
  Sources are not copied until the proposal is approved. Identical entries for
  one target are reused; conflicting selected digests for one target are
  rejected before transaction staging.
- Browser source entries additionally bind the package-local profile digest to
  reviewed action names, origins, active lifecycle/expiry, login-state
  requirement, provenance, and optional immutable registry coordinate. The
  source bundle digest and materialized profile digest are distinct when a
  capability bundle is used.
- `browser_verifications` is optional resumable local state for explicit
  value-free Browsertools live/portability reports. `source_path` is reopened
  at approval but is not copied into package review metadata. Its `summary` is
  strict, bounded, profile/action/origin/lifecycle-bound, and contains only
  declared paths, match/type/reachability facts, fixed diagnostics, and the raw
  report SHA-256. Conflicting or changed sources fail closed. Package metadata
  retains the summary without `source_path`; portability remains optional.
- Browser authentication entries bind a secret-free
  `uws.browser-authentication.1.0` profile to its reviewed flows, exact
  per-flow credential-slot names, origins, active lifecycle/expiry, digest,
  and provenance. `browser_authentication_approvals` contains exact
  authentication step names. Credential values, MFA responses, cookies,
  storage state, and live browser handles are prohibited.
- `browser_route` is `browser` only after an explicit or fallback route
  selection. `browser_session_posture` is `none` or
  `opaque-runtime-binding-required`; it never stores a browser session value.
  `browser_mutation_approvals` contains exact workflow step names, not a global
  bypass. Separate interview metadata records API remote-lookup and static
  browser-registry lookup decisions.
- `intent`, symbolic credential bindings, `fallback`, and `side_effect_scope`
  retain their existing OpenUdon meanings. Never store credential values.

Incomplete approved work is saved as `workflows/intent.draft.hcl` plus
`.icot/session.yaml` and `.icot/readiness.json`; it never creates
`workflows/intent.hcl`. Completing and approving that draft promotes it
atomically and removes the obsolete generated draft/readiness files.

`--yes` is the explicit noninteractive proposal approval for a complete v2
session. Without it, interactive runs show the full proposal and require
`approve`; agent mode never writes deliverables.
