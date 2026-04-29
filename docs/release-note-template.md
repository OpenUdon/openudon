# Ramen Release Note Template

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

## Local Checks

- `go test ./...`:
- `go vet ./...`:
- `make check`:
- `git diff --check`:
- `make release-check`:
- `ramen readiness --run-gates`:
- Readiness JSON:

## Real-LLM Smoke

Command:

```bash
make release-eval
```

Report paths:

- JSON:
- Markdown:

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

## Known External Blockers

- 

## Decision

- Release decision:
- Reviewer:
- Notes:
