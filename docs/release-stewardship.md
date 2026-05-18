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

`make release-check` is the deterministic local pre-tag gate:

```bash
make release-check
go run ./cmd/openudon validate ./examples/uws-validation
go run ./cmd/openudon check-doc-memory
go run ./cmd/openudon n8n-bridge validate --root examples/eval
mkdocs build --strict --site-dir /tmp/openudon-mkdocs-release
```

`check-doc-memory` is intentionally local. It verifies ignored memory-bank and
evolution harness files in maintainer checkouts and warns when milestone changes
may need a new evolution record. It is not a public CI gate.

For the SaaS release story, also run the provider-free dry-run demo in
[SaaS Operator Release Path](saas-operator-release.md). The selected examples
are:

- `gmail-send-audit-receipt` for a single-service side-effectful send-and-audit
  workflow;
- `order-fulfillment-chain` for a multi-service lookup-and-create workflow.

The demo must use ignored `.openudon-run/...` output, sandbox approval JSON, and
`openudon run --dry-run`. Do not commit approval JSON, run configs, transcripts,
or real-provider outputs.

Real-provider evals remain opt-in local evidence:

```bash
make release-eval
```

Record provider, model, corpus size, comparison baseline, provider drift status,
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
- Symphony may route review from OpenUdon evidence, but Symphony-managed
  identity, state transitions, and audit persistence stay outside OpenUdon.
- Udon or another trusted executor receives a package only through
  `openudon run` after approval and digest validation.
