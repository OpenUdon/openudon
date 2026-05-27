# SaaS Operator Release Path

This page is the provider-free release demo path for OpenUdon's SaaS authoring
slice. It shows how a new operator can move from reviewed source artifacts to
quality evidence, approval JSON, and trusted-runner dry-run output without live
provider credentials.

Use these demos for release communication:

| Demo | Fixture | Why It Is The Demo |
| --- | --- | --- |
| Single-service send-and-audit | `examples/eval/gmail-send-audit-receipt` | Shows a side-effectful SaaS send operation followed by local audit evidence. |
| Multi-service lookup-and-create | `examples/eval/order-fulfillment-chain` | Shows customer and inventory lookups feeding a sandbox fulfillment create step with mixed credential bindings. |

Both fixtures are strict native SaaS examples. They use local OpenAPI slices and
symbolic credential bindings only.

## Provider-Free Release Gate

Run the complete local SaaS release gate with:

```bash
make release-saas-check
```

The target runs the normal deterministic release check, UWS validation,
the eval seed/build matrix, doc-memory check, n8n bridge validation, strict
MkDocs build, selected strict SaaS fixture lint, and the Gmail/order
fulfillment trusted-runner dry-run demo below. It is local maintainer evidence,
not public CI, and it does not require provider credentials.

## Provider-Free Demo Loop

Run release demos in an ignored workdir so checked-in examples stay clean:

```bash
export DEMO_ROOT=.openudon-run/m23-operator-demo
rm -rf "$DEMO_ROOT"
mkdir -p "$DEMO_ROOT/approvals"
```

For each demo fixture, copy the source fixture and promote the checked-in
reference intent into the working `workflows/` directory:

```bash
for fixture in gmail-send-audit-receipt order-fulfillment-chain; do
  cp -R "examples/eval/$fixture" "$DEMO_ROOT/$fixture"
  mkdir -p "$DEMO_ROOT/$fixture/workflows"
  cp "$DEMO_ROOT/$fixture/reference/intent.hcl" "$DEMO_ROOT/$fixture/workflows/intent.hcl"
done
```

Build, review, approve for sandbox, and dry-run each package:

```bash
for fixture in gmail-send-audit-receipt order-fulfillment-chain; do
  go run ./cmd/openudon build --example "$DEMO_ROOT/$fixture"
  go run ./cmd/openudon assess --example "$DEMO_ROOT/$fixture"
  go run ./cmd/openudon approval-template \
    --example "$DEMO_ROOT/$fixture" \
    --state approved_for_sandbox \
    --reviewer "Release Reviewer" \
    > "$DEMO_ROOT/approvals/$fixture.json"
  go run ./cmd/openudon run \
    --example "$DEMO_ROOT/$fixture" \
    --tier sandbox \
    --approval "$DEMO_ROOT/approvals/$fixture.json" \
    --workdir "$DEMO_ROOT/workdir-$fixture" \
    --dry-run
done
```

The dry run validates current quality, stored quality, approval state, approval
scope, package digest, handoff manifest, credential-value policy, and tier
compatibility. It writes a non-secret `openudon.executor-run.v1` config, stages
the package, verifies the staged digest, writes `openudon.run-evidence.v1`, and
does not invoke the executor.

Archive only reviewed, non-secret summaries such as command transcripts,
quality Markdown, review Markdown, or release-note excerpts. Do not commit
`.openudon-run`, approval JSON, run configs, real-provider eval output, or
credential material.

## Evidence Checklist

For a SaaS release candidate, collect deterministic evidence first:

- `go test ./...`;
- `go vet ./...`;
- `make check`;
- `make release-check`;
- `make eval-seed-build`;
- `make icot-variants-validate`;
- `make icot-variants-coverage`;
- `make icot-authoring-scorecard`, which also verifies the generated scorecard JSON and digest
  sidecar with `icot report verify`;
- `make release-saas-check`;
- `go run ./cmd/openudon validate ./examples/uws-validation`;
- `go run ./cmd/openudon check-doc-memory`;
- `go run ./cmd/openudon n8n-bridge validate --root examples/eval`;
- `mkdocs build --strict`;
- selected strict SaaS fixture lint with `cmd/icot`;
- the two demo dry runs above.

Optional real-provider or real-LLM evidence stays local/manual:

- `make release-eval`;
- `go run ./cmd/icot authoring-eval --root examples/eval --include-variants --provider ... --model ... --out eval/runs/icot-authoring-eval-local`;
- `go run ./cmd/icot report verify --file eval/runs/icot-authoring-eval-local/authoring-eval.json`;
- provider/model name;
- comparison baseline;
- provider drift watch status;
- transient provider errors and rerun evidence.

Do not weaken deterministic release criteria because of a transient provider
failure. Record it as provider drift and rerun from a trusted workstation if it
matters for the release decision.

## Boundary Summary

OpenUdon does:

- help author `project.md` and `workflows/intent.hcl`;
- build UWS/OpenAPI package artifacts from reviewed inputs;
- generate quality, review, handoff, approval, package digest, and run-config
  evidence;
- validate a trusted-runner handoff before any executor receives a package.

OpenUdon does not:

- run n8n workflows or preserve n8n runtime behavior;
- call live SaaS providers during build, assess, iCoT, eval, or dry-run demos;
- execute Terraform/OpenTofu, provider plugins, state, plan, apply, or cloud
  SDK calls;
- implement reviewer identity, routing, managed state, or audit
  persistence;
- compile or execute generic UWS/OpenAPI workflows itself;
- resolve secrets or credential values in committed artifacts.

Use `openudon run` without `--dry-run` only in an operator-controlled
environment with reviewed approval JSON, sandbox or production tier policy,
symbolic credential bindings resolved outside generated artifacts, and a
configured trusted executor.
