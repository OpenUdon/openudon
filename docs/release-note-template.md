# Ramen Release Note Template

## Candidate

- Date:
- Commit:
- Dirty worktree state:
- Eval corpus size:
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

## Local Checks

- `go test ./...`:
- `go vet ./...`:
- `make check`:
- `git diff --check`:
- `make release-check`:

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

## Known External Blockers

- 

## Decision

- Release decision:
- Reviewer:
- Notes:
