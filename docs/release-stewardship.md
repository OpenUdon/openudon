# Release Stewardship

OpenUdon release checks are split between public, provider-free gates and local
maintainer evidence. Public automation must not depend on ignored memory-bank,
evolution, readiness, eval, approval, or run-workdir files.

## Public Gates

GitHub Actions runs the public Go module with workspace mode disabled:

```bash
GOWORK=off go vet ./...
GOWORK=off go test ./... -count=1 -timeout=5m
GOWORK=off go run ./cmd/openudon check-apitools-boundary
git diff --check
```

Documentation publishing builds the MkDocs site in strict mode before deploy:

```bash
mkdocs build --strict
```

The repository boundary check rejects direct OpenUdon imports of old lifecycle
`apitools` APIs, private udon executor packages, private `genelet/*` executor
modules, Terraform/OpenTofu internals, and `tfconfig/_upstream/...`.

## Local Maintainer Gates

`make release-check` is the fast deterministic local pre-tag gate:

```bash
make release-check
```

`check-doc-memory` is intentionally local. It verifies ignored memory-bank and
evolution harness files in maintainer checkouts and warns when milestone changes
may need a new evolution record. It is not a public CI gate.

For the SaaS release story, run the comprehensive provider-free local gate:

```bash
make release-saas-check
```

`release-saas-check` runs `release-check`, `eval-seed-build`,
`icot-variants-validate`, `icot-variants-coverage`, `icot-authoring-scorecard`, UWS validation,
doc-memory, n8n bridge validation, strict MkDocs build, selected strict SaaS fixture lint, and the
provider-free dry-run demo in
[SaaS Operator Release Path](saas-operator-release.md). `icot-authoring-scorecard` generates the
provider-free scorecard and then runs `icot report verify` against `scorecard.json`, including the
digest sidecar and retention/share-safety metadata. The selected demo
examples are:

- `gmail-send-audit-receipt` for a single-service side-effectful send-and-audit
  workflow;
- `order-fulfillment-chain` for a multi-service lookup-and-create workflow.

The demo must use ignored `.openudon-run/...` output, sandbox approval JSON, and
`openudon run --dry-run`. Do not commit approval JSON, run configs, transcripts,
or real-provider outputs.

Run the eval seed/build matrix directly when changing authoring fixtures or
reference intents:

```bash
make eval-seed-build
make icot-variants-validate
```

For the improved `v0.1.2-a.1` candidate, run the product smoke matrix after the
provider-free release gates:

```bash
make product-smoke-check
OPENUDON_EXECUTOR=/absolute/path/to/udon make product-smoke-live
```

`product-smoke-check` is provider-free and builds ignored scratch packages from
the reviewed eval fixtures. `product-smoke-live` is local maintainer evidence:
Slack live smoke is required before tagging `v0.1.2-a.1`, local synthetic APIs
run against a stub server, and optional OpenWeatherMap live proof runs only when
its complete credential env set is present. Gmail has credential-backed examples
and manual proof-run support, but the product smoke matrix records dry-run
evidence for Gmail unless an operator separately runs and records a reviewed
Gmail proof. Jira currently has fixture/dry-run coverage but no recorded
real-key proof. See
[Product Smoke Matrix](product-smoke-matrix.md).

Real-provider evals remain opt-in local evidence:

```bash
make release-eval
go run ./cmd/icot authoring-eval --root examples/eval --include-variants --provider copilot-api --model gpt-5.4-mini --out eval/runs/icot-authoring-eval-local
go run ./cmd/icot report verify --file eval/runs/icot-authoring-eval-local/authoring-eval.json
```

Record provider, model, corpus size, comparison baseline, provider drift status,
optional authoring-eval report path, authoring-eval pass summary, retention/share-safety metadata,
and known gaps in the release notes.

Provider/model drift is release evidence, not a deterministic gate by itself.
Record transient provider failures and rerun once from a trusted workstation
when availability or rate limits look external.

## Terraform/OpenTofu Conversion

`openudon convert tf` release stewardship uses the same boundaries:

- static Terraform/OpenTofu facts come from `github.com/OpenUdon/tfconfig`;
- OpenAPI operation metadata comes from `github.com/OpenUdon/apitools`;
- generated workflow, review, quality, and handoff artifacts remain unapproved
  until normal OpenUdon review and trusted-runner checks pass;
- Terraform/OpenTofu execution, provider plugins, state, plan/apply, and
  credential resolution stay outside OpenUdon.

## Boundary Recap

OpenUdon's release evidence must keep these ownership boundaries clear:

- n8n and `../try-n8n` provide service-priority and pattern evidence only; they
  are not runtime dependencies or import targets.
- Live SaaS providers are not contacted by build, assess, iCoT, eval, or
  trusted-runner dry-run demo commands.
- External review orchestration may route review from OpenUdon evidence, but
  identity, state transitions, and audit persistence stay outside OpenUdon.
- Udon or another trusted executor receives a package only through
  `openudon run` after approval and digest validation.
