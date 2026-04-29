# Ramen CI

Ramen deterministic CI needs private repository access because `go.mod` uses local sibling
replacements for `../udon`, `../grand`, `../golet`, `../hcllight`, `../horizon`, `../molecule`,
`../uws`, and `../arazzo`, and the imported dependency graph also contains private
`github.com/genelet/*` and `github.com/tabilet/*` modules.

## Required Secret

Set these Ramen repository or organization secrets:

- `RAMEN_CI_GENELET_TOKEN`: read-only contents access to private `genelet/*` dependency repos.
- `RAMEN_CI_TABILET_TOKEN`: read-only contents access to private `tabilet/*` dependency repos.

A classic PAT with both owners may be stored in both secrets. Fine-grained PATs usually need separate
owner-scoped tokens.

The tokens need read-only contents access to these private dependency repositories:

- `genelet/udon`
- `genelet/grand`
- `genelet/golet`
- `genelet/hcllight`
- `genelet/horizon`
- `genelet/molecule`
- private `genelet/*` modules imported transitively, including `genelet/determined`
- `tabilet/uws`
- `tabilet/arazzo`
- private `tabilet/*` modules imported transitively, including `tabilet/oas`, `tabilet/schema`, and
  `tabilet/sqlmeta`

Some older transitive module paths still reference `github.com/genelet/oas`,
`github.com/genelet/schema`, or `github.com/genelet/sqlmeta`. Those repositories have moved under
the `tabilet` owner, so CI checks the canonical `tabilet/*` repositories and rewrites those legacy
Git fetches to use `RAMEN_CI_TABILET_TOKEN`.

The workflow configures:

```text
GOPRIVATE=github.com/genelet/*,github.com/tabilet/*
GONOSUMDB=github.com/genelet/*,github.com/tabilet/*
GONOPROXY=github.com/genelet/*,github.com/tabilet/*
```

It also configures GitHub HTTPS module fetches to use the owner-specific CI tokens.

## Layout

The workflow checks out Ramen and sibling repos into one parent workspace:

```text
ramen/
udon/
grand/
golet/
hcllight/
horizon/
molecule/
uws/
arazzo/
```

This matches the local `replace ../...` layout and lets `./scripts/check-siblings.sh` validate the
workspace before tests run.

## Common Failure

Before checkout, the workflow calls the GitHub repository API for each required private repository.
If this preflight fails, the annotation identifies the token name and repository that needs access.

If a checkout step fails with:

```text
Not Found - https://docs.github.com/rest/repos/repos#get-a-repository
```

the token used by that step cannot read the requested `owner/repo`, or the repository name is wrong.
For example, failures in `Check out udon` point at `RAMEN_CI_GENELET_TOKEN`; failures in
`Check out uws` point at `RAMEN_CI_TABILET_TOKEN`.

## Commands

The workflow intentionally runs the deterministic local gates:

```bash
./scripts/check-siblings.sh
go test ./...
go vet ./...
make check
git diff --check
```

It does not run `go mod download all` or `go list -m all`; the current private sibling graph has
placeholder transitive versions that make blanket module-graph commands a different problem from
Ramen deterministic build/test coverage.

Real-provider evals remain local/manual through `make release-eval`; CI does not receive provider
API keys.
