# Contributing

OpenUdon accepts focused issues and small pull requests that improve the public UWS authoring,
review, package, and executor-handoff workflow. Roadmap direction, release timing, and trusted
executor integration policy remain maintainer-controlled.

## Before Opening A Pull Request

- Keep public UWS semantics in `github.com/OpenUdon/uws`.
- Keep generic OpenAPI discovery/search/import behavior in `github.com/OpenUdon/apitools`.
- Keep private executor behavior outside OpenUdon; OpenUdon invokes executors through approved
  CLI/Docker handoff only.
- Do not include secrets in prompts, examples, test fixtures, review artifacts, or logs.
- Prefer small changes with focused tests.

## Local Checks

Run these before submitting:

```bash
GOWORK=off go test ./...
GOWORK=off go vet ./...
go test ./...
go vet ./...
go run ./cmd/openudon check-doc-memory
go run ./cmd/openudon validate ./examples/uws-validation
git diff --check
```

`make check` may also be useful in a maintainer workspace with the expected sibling repositories.

## Licensing

OpenUdon is licensed under Apache-2.0. Unless explicitly marked otherwise, any contribution
submitted for inclusion in this repository is submitted under Apache-2.0. New files do not need
per-file copyright headers unless the surrounding package already uses them.

