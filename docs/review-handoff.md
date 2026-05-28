# Review Handoff

OpenUdon packages are reviewed through generated evidence and a machine-readable handoff manifest
before `openudon run` can invoke a trusted executor.

For SaaS-specific review evidence, credential scope, side-effect risk, approval
JSON, and dry-run guidance, see [SaaS Review And Trusted Handoff](saas-review-handoff.md).

## Handoff Manifest

`expected/review-handoff.json` uses the stable `apitools.review-handoff.v1` wire version. The
manifest records package inputs, approval state, owner split, execution policy, credential binding
names, and trusted-runner metadata.

The manifest is evidence for an external reviewer or orchestrator. It does not grant approval by itself.
Generation normally leaves side-effectful packages in `generated` or review-required state.

## Required Package Inputs

The package digest covers the required handoff inventory:

```text
project.md
workflows/intent.hcl
workflows/workflow.hcl
workflows/workflow.uws.yaml
expected/plan.json
expected/quality.json
expected/refinement.json
expected/review.md
expected/review-handoff.json
expected/data.hcl when runtime inputs are declared
openapi/... regular files used by the package
google-discovery/... regular files used by the package
aws-smithy/... regular files used by the package
asyncapi/... regular files used by the package
graphql/... regular files used by the package
openrpc/... regular files used by the package
grpc-protobuf/... regular files used by the package
odata/... regular files used by the package
associated advisory security sidecars
```

Unsafe relative paths, symlinks, directories, special files, missing files, and unstated required
inputs are rejected before execution.

`expected/data.hcl` may include `ENVIRONMENT` markers for values owned by the
operator environment. Review packages should contain the marker names, not
plaintext credential values.

## Approval JSON

Create approval JSON only after reviewing the current package:

```bash
mkdir -p approvals
go run ./cmd/openudon approval-template \
  --example ./examples/eval/support-email \
  --state approved_for_sandbox \
  --reviewer "Reviewer Name" \
  > approvals/support-email-sandbox.json
```

Approval JSON uses `openudon.approval.v1` and includes:

```text
version
scope
state
reviewer
approved_at
expires_at
package_sha256
notes
```

The approved digest must match the package at run time. If any digest-covered file changes,
generate a new approval after review.

## Trusted Runner Config And Evidence

Validate the package and write a non-secret run config without invoking the executor:

```bash
go run ./cmd/openudon run \
  --example ./examples/eval/support-email \
  --tier sandbox \
  --approval approvals/support-email-sandbox.json \
  --dry-run
```

`openudon run` checks the handoff manifest, stored and current quality, approval scope, approval
state, expiry, package digest, tier compatibility, credential-value policy, and direct-production
policy. The resulting `openudon.executor-run.v1` config includes the UWS artifact, API source files,
sorted package paths, package digest, tier, workdir, and credential binding names.

Dry runs stage digest-covered files into a fresh workdir and recompute the package digest without
requiring credential values or invoking the executor. Both dry runs and real handoffs write
`openudon.run-evidence.v1` at `<workdir>/run-evidence.json` with package paths, staged paths, gate
outcomes, and credential binding names only. `OPENUDON_EXECUTOR` selects the final executor as an
absolute binary path or `docker://<image>`.

If `OPENUDON_UDON_RUNNER` overrides the outer runner shim, OpenUdon evidence marks the staged path as
`stage_kind: preflight`. That proves OpenUdon's package validation before handing the config to the
external runner; the external runner still owns its final executor-visible stage and invocation.
