# Contributing

OpenUdon accepts focused issues and small pull requests that improve the public
UWS package, review, approval and executor-handoff libraries. Roadmap direction, release timing, and trusted
executor integration policy remain maintainer-controlled.

## Before Opening A Pull Request

- Keep public UWS semantics in `github.com/OpenUdon/uws`.
- Keep generic OpenAPI discovery/search/import behavior in `github.com/OpenUdon/apitools`.
- Keep authoring product flows in Kinet and private execution in Udon; OpenUdon
  supplies public package and handoff contracts.
- Do not include secrets in prompts, examples, test fixtures, review artifacts, or logs.
- Prefer small changes with focused tests.

## Local Checks

Run these before submitting:

```bash
GOWORK=off go mod download
GOWORK=off go test ./... -count=1 -timeout=10m
GOWORK=off go vet ./...
GOWORK=off go test ./tests/publicapi -count=1
git diff --check
```

`make check` runs the standalone public-module build, tests and vet checks.

## Licensing

OpenUdon is licensed under Apache-2.0. Unless explicitly marked otherwise, any contribution
submitted for inclusion in this repository is submitted under Apache-2.0. New files do not need
per-file copyright headers unless the surrounding package already uses them.
