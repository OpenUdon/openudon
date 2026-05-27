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

For iCoT reliability reporting, run the provider-free scorecard:

```bash
go run ./cmd/icot scorecard --root examples/eval --out eval/runs/icot-scorecard-local
```

The scorecard writes `openudon.icot-scorecard.v1` JSON with the expected outcome, observed outcome,
fixture class, first failure family, and failure codes for each fixture. It uses the same no-LLM,
package-local seed/build path as the matrix.

For M40 natural-language authoring coverage, include checked-in variant metadata:

```bash
go run ./cmd/icot variants validate --root examples/eval
go run ./cmd/icot scorecard --root examples/eval --include-variants --out eval/runs/icot-authoring-scorecard-local
```

Variant files live at `examples/eval/*/reference/authoring-variants.json`. Positive variants reuse
the fixture's reviewed reference workflow with a different operator brief and must still build
provider-free. Missing-detail and unsafe-negative variants must stop with the declared
`needs_input`, `build_fail`, or `icot_fail` outcome and failure family. The variant scorecard adds
provider-family, variant-class, provider/failure-family, and top readiness issue summaries without
changing the default seed/build contract. Missing-detail variants may set `seed_from_reference` plus
`clear_fields` or `clear_slots` so the deterministic path preserves the reviewed source/operation
and removes only the intended business/request detail.

`icot variants validate` is a fast metadata check for the same files. It catches schema errors,
unknown expected failure families, duplicate IDs, and reference-seeded clear slots that no longer
match the reviewed reference intent.

This scorecard remains provider-free reference/variant package evidence. It does not show that a
live LLM generated the workflow from the variant brief. For optional real authoring evidence, run:

```bash
go run ./cmd/icot authoring-eval --root examples/eval --include-variants --provider copilot-api --model gpt-5.4-mini --out eval/runs/icot-authoring-eval-local
```

`icot authoring-eval` writes `openudon.icot-authoring-eval.v1` with provider/model, prompt version,
LLM call count, generated paths, first failure family, drift counts, and per-variant pass/fail. Keep
that report local/manual unless it has been reviewed for release-note evidence.

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
- Add `reference/authoring-variants.json` when the fixture is part of natural-language authoring
  coverage; keep variants provider-free and free of secrets, channel IDs, email addresses, or live
  provider outputs.
- Keep strict positive fixtures buildable from package-local artifacts.
- Use `advisory` only when a fixture is evidence for reducibility or upstream drift rather than a
  strict OpenUdon-native contract.
- Use `expected-negative` only for deliberate rejection, clarification, or repair behavior.
- Keep `allowed_failure_codes` narrow, and remove allowances when the gap is fixed.
