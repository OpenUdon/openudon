# Ramen Release Note Template

## Candidate

- Date:
- Commit:
- Eval corpus size:
- Provider:
- Model:
- Prompt version:
- Structured mode count:
- Legacy fallback count:
- Pass rate:
- Maximum attempts for any brief:
- Blocking reference issues:
- Secret-scan failures:

## Local Checks

- `go test ./...`:
- `go vet ./...`:
- `make check`:

## Real-LLM Smoke

Command:

```bash
go run ./cmd/ramen eval --root ./examples/eval --provider gemini --model gemini-2.5-flash --release-gate
```

Report paths:

- JSON:
- Markdown:

## Known Gaps

- 

## Decision

- Release decision:
- Reviewer:
- Notes:
