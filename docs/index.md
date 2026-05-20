# OpenUdon

<p align="center">
  <img src="assets/openudon.png" alt="OpenUdon logo" width=800>
</p>

OpenUdon is the public UWS workflow authoring, review, package, and executor-handoff tool. It turns
reviewed project briefs into deterministic workflow packages and hands approved packages to a trusted
executor boundary.

OpenUdon can be used directly by an operator or under optional Symphony-managed orchestration. In
both modes, generated artifacts stay untrusted until validation, review, approval, and package digest
checks pass.

## What OpenUdon Owns

- Project briefs, templates, guided iCoT authoring, and eval fixtures.
- OpenAPI-bound UWS artifact generation from reviewed inputs, including Discovery/Smithy sources
  after upstream lowering.
- Review evidence, quality reports, approval templates, package digests, and handoff manifests.
- Local trusted-runner enforcement before invoking an external executor.

OpenUdon does not own public workflow semantics, generic OpenAPI/UWS execution, Symphony workflow
state, or concrete infrastructure authoring. Those boundaries are summarized on the
[Related](related.md) page.

## Basic Flow

```text
project.md
  -> workflows/intent.hcl
  -> workflows/workflow.hcl and workflows/workflow.uws.yaml
  -> expected plan, quality, review, and handoff artifacts
  -> approval JSON with package digest
  -> openudon run trusted executor handoff
```

## Operator Commands

```bash
go run ./cmd/icot --example ./examples/<name>
go run ./cmd/openudon synthesize --example ./examples/<name>
go run ./cmd/openudon build --example ./examples/<name>
go run ./cmd/openudon assess --example ./examples/<name>
go run ./cmd/openudon approval-template --example ./examples/<name> --state approved_for_sandbox --reviewer "Reviewer Name"
go run ./cmd/openudon run --example ./examples/<name> --tier sandbox --approval approvals/<name>.json --dry-run
```

Use [Authoring](authoring.md) for the two authoring paths, [Tutorial](tutorial-weather.md) for
fixture-based walkthroughs, [SaaS Operator Release Path](saas-operator-release.md) for the
provider-free release demo, and [Handoff](safety.md) for the review and execution boundary.
