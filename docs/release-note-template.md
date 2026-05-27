# OpenUdon Release Note Template

## Candidate

- Date:
- Commit:
- Dirty worktree state:
- Eval corpus size:
- Minimum eval brief gate:
- Provider:
- Model:
- Prompt version:
- Comparison baseline:
- Structured mode count:
- Legacy fallback count:
- Pass rate:
- Maximum attempts for any brief:
- Blocking reference issues:
- Secret-scan failures:
- Release-gate result:
- Expanded-corpus evidence exception:
- SaaS operator demo fixtures:
- SaaS operator demo dry-run result:
- Product smoke dry-run result:
- Product smoke live result:
- Product smoke summary JSON:
- Required Slack live smoke result:
- Optional provider live skips:
- n8n bridge validation result:
- Optional iCoT authoring-eval report:
- Optional iCoT authoring-eval provider/model:
- Optional iCoT authoring-eval pass summary:
- Optional iCoT authoring-eval credential-scan result:

## Local Checks

- `go test ./...`:
- `go vet ./...`:
- `make check`:
- `git diff --check`:
- `make release-check`:
- `make release-saas-check`:
- `make icot-variants-validate`:
- `make icot-authoring-scorecard`:
- `make product-smoke-check`:
- `make product-smoke-live`:
- `mkdocs build --strict`:
- `openudon validate ./examples/uws-validation`:
- `openudon check-doc-memory`:
- `openudon n8n-bridge validate --root examples/eval`:
- Selected SaaS fixture lint:
- SaaS demo trusted dry-run:
- Sibling checkout layout (`../uws`, `../apitools`, optional `../udon`):
- `openudon readiness --run-gates`:
- Readiness JSON:

## Real-LLM Smoke

Command:

```bash
make release-eval
```

`make release-eval` is opt-in real-provider evidence. Keep `make release-check` as the deterministic
pre-tag gate and record provider credentials/model context separately from deterministic evidence.

Report paths:

- JSON:
- Markdown:
- Optional iCoT authoring-eval JSON:

Comparison:

- Baseline JSON:
- Pass-rate delta:
- Brief regressions:
- Attempt regressions:
- Blocking-reference regressions:
- Failing-check regressions:

## Provider Drift Watch

- Status:
- Structured fallback count:
- Structured fallback delta:
- Rate/transient failures:
- Model availability:
- Maximum attempts-to-pass:
- Attempt regressions:
- Release-gate failures:
- Eval JSON `provider_drift_watch`:
- Rerun evidence, if provider drift was suspected:

## SaaS Operator Demo

Demo fixtures:

- Single-service:
- Multi-service:

Provider-free commands:

```bash
# See docs/saas-operator-release.md for the ignored-workdir script.
```

Evidence:

- Quality Markdown paths:
- Review Markdown paths:
- Approval JSON paths kept local/ignored:
- Dry-run config paths kept local/ignored:
- Boundary notes:

## Product Smoke Matrix

Commands:

```bash
make product-smoke-check
OPENUDON_EXECUTOR=/absolute/path/to/udon make product-smoke-live
```

Evidence:

- Summary JSON kept local/ignored:
- Slack live message/channel confirmation:
- Local stub-backed live scenarios:
- Optional provider live scenarios run:
- Optional provider live scenarios skipped for missing env:
- Failed or blocked scenarios:

## Known External Blockers

- 

## Decision

- Release decision:
- Reviewer:
- Notes:
