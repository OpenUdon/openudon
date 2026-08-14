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
  public rationale, never hidden chain-of-thought.
- A technical deferral must name its owner, impact, unblock condition, and next
  action. Source, operation, mapping, and output leaves may be deferred.
- `candidate_workflows` are unnumbered future directions. Each has a title,
  outcome, deferral reason, and promotion trigger, and must not contain sources,
  operations, mappings, or implementation steps.
- `source_plan` entries, when present, include kind, stable ID, inspected source
  path, package target, SHA-256 digest, title/operation count, and provenance.
  Sources are not copied until the proposal is approved.
- `intent`, symbolic credential bindings, `fallback`, and `side_effect_scope`
  retain their existing OpenUdon meanings. Never store credential values.

Incomplete approved work is saved as `workflows/intent.draft.hcl` plus
`.icot/session.yaml` and `.icot/readiness.json`; it never creates
`workflows/intent.hcl`. Completing and approving that draft promotes it
atomically and removes the obsolete generated draft/readiness files.

`--yes` is the explicit noninteractive proposal approval for a complete v2
session. Without it, interactive runs show the full proposal and require
`approve`; agent mode never writes deliverables.
