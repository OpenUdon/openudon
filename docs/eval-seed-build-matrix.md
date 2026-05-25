# Eval Seed/Build Matrix

The eval corpus now carries an explicit offline seed/build contract in each
`examples/eval/*/reference/policy.json`.

The matrix covers this deterministic path:

```bash
go run ./cmd/icot --no-llm --no-transcript --from-example ./examples/eval/<fixture> --example <tmp>/<fixture>
go run ./cmd/openudon build --example <tmp>/<fixture>
```

The repository test uses the same behavior through package APIs and writes only to a temporary
directory. It does not call an LLM provider, retrieve remote OpenAPI documents, or execute generated
workflows.

Run the matrix directly with:

```bash
make eval-seed-build
```

## Policy Fields

`seed_build.expected` declares the required outcome:

- `pass`: iCoT seeding and build must both succeed.
- `build_fail`: iCoT seeding should succeed, but build is expected to reject the package.
- `icot_fail`: iCoT seeding itself is expected to fail.

`seed_build.class` declares how the fixture should be interpreted:

- `strict-positive`: golden OpenUdon-native behavior. These fixtures must build green.
- `expected-negative`: a deliberate rejection or repair fixture. Failure is acceptable only when it
  matches the declared expectation.
- `advisory`: reducibility or drift evidence. Advisory fixtures are useful, but they do not block
  strict positive coverage.

`seed_build.allowed_failure_codes` is optional. When present, at least one observed failure code must
match the allow-list. Current build errors that happen before a full quality report are represented
as `build:error`.

## Current Split

Strict positive fixtures must pass the matrix. This includes the native SaaS, pagination,
credential-binding, multi-service, runtime-only, and helper-backed examples.

Expected-negative fixtures either build a reviewable package that documents the problem or fail for
the declared policy reason. For example, `cmd-disallowed-deploy` is expected to fail build because
the project policy denies `cmd`.

Advisory n8n reducibility fixtures remain separated from strict native behavior. They currently
seed and build cleanly from bounded, package-local OpenAPI evidence, but they remain advisory until
the project deliberately graduates them into strict OpenUdon-native coverage.

## Maintenance

When adding or changing an eval fixture:

- Add `reference/policy.json` with `seed_build.expected`, `seed_build.class`, and a short reason.
- Keep strict positive fixtures buildable from package-local artifacts.
- Use `advisory` only when a fixture is evidence for reducibility or upstream drift rather than a
  strict OpenUdon-native contract.
- Use `expected-negative` only for deliberate rejection, clarification, or repair behavior.
- Keep `allowed_failure_codes` narrow, and remove allowances when the gap is fixed.
