# Review Handoff

OpenUdon packages are reviewed through generated evidence and a machine-readable handoff manifest
before `openudon run` can invoke a trusted executor.

## Handoff Manifest

`expected/symphony-handoff.json` uses the stable `apitools.review-handoff.v1` wire version. The
manifest records package inputs, approval state, owner split, execution policy, credential binding
names, and trusted-runner metadata.

The manifest is evidence for Symphony or another reviewer. It does not grant approval by itself.
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
expected/symphony-handoff.json
openapi/... regular files used by the package
```

Unsafe relative paths, symlinks, directories, special files, missing files, and unstated required
inputs are rejected before execution.

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

## Trusted Runner Config

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
policy. The resulting `openudon.executor-run.v1` config includes the UWS artifact, OpenAPI files,
sorted package paths, package digest, tier, workdir, and credential binding names.

The runner stages digest-covered files into a fresh workdir and recomputes the package digest before
executor invocation. `OPENUDON_EXECUTOR` selects the final executor as an absolute binary path or
`docker://<image>`.
