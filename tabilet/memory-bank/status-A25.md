# A25 Status - Guided Registration Symbolic-Binding False-Positive Closure

State markers and commit rules are defined in [milestone.md](milestone.md).

## State

Complete and published at OpenUdon
`561fd933097abe90cdbe2b59b56e0fa33d4d41c1`. It corrects intermediate revision
`e5b8a87d00ffd158dc7b71e7b5c264d320bf425f` in the newly published history after
deep review found and closed a P1 letters-only opaque-token bypass. A downstream
A24 no-submit session had reached local draft construction but the pinned
OpenUdon validator mistook valid namespaced symbolic runtime binding names for
credential values. The session failed closed and retained no candidate.
Implementation and verification remained offline.

## Task Ledger

| Item | State | Notes |
|---|---|---|
| A25.1 Align guided registration binding validation with symbolic-name authority | `[+]` | Published at OpenUdon `561fd933097abe90cdbe2b59b56e0fa33d4d41c1`. Deep review found that the first local implementation treated arbitrary lowercase runs as descriptive words, allowing letters-only opaque values into transaction metadata. The final classifier requires at least two closed-vocabulary purpose words plus at most one short digit-bearing namespace. Builder and API-v4 coverage exercise exact-slot names, W8M-style names, a natural non-slot name, digit-bearing and letters-only opaque values, known token formats, value-free diagnostics, and canonical BRP separation. Full offline verification and bounded review pass. |

## Acceptance

- Descriptive high-entropy bindings are accepted as symbolic names whether or
  not they end in the exact declared slot only when they contain at least two
  closed-vocabulary purpose words and at most one short digit-bearing namespace.
- Known credential formats, short opaque-prefix suffix bypasses, opaque
  digit-bearing and letters-only multi-token bindings, invalid symbols,
  duplicate slots, and duplicate bindings remain rejected with a value-free
  binding-specific API diagnostic.
- Canonical BRP bytes remain credential-free; bindings remain transaction-only
  symbolic metadata.
- Focused and full offline tests, vet, repository checks,
  documentation-memory validation, JavaScript syntax, diff checks, and one
  bounded review pass.

## Non-Goals

- No public UWS, Browsertools, transaction, or API wire change.
- No target contact, browser session, registration submission, candidate,
  transaction adoption, credential entry, account action, runtime execution,
  downstream pin adoption, release, or deployment.

## Verification

- Pinned Go 1.26.6 with `GOWORK=off`, `GOPROXY=off`, `GOSUMDB=off`,
  `GOENV=off`, and `GOTOOLCHAIN=local`: focused builder/API tests, full
  `go test ./...`, full `go vet ./...`, and `make check` pass.
- The `icot_ui_browser` test set compiles with no tests executed; Chromium was
  not launched. `node --check internal/icot/ui/assets/app.js` passes.
- Repository-boundary and documentation-memory checks pass. OpenUdon and Tofu
  `git diff --check` pass; no dependency or public wire changed.
- Review iteration 3 found one P1: arbitrary lowercase components let a
  letters-only opaque token bypass the value-oriented entropy rejection and
  enter transaction metadata. The current working tree replaces that open
  structure with a closed purpose-word vocabulary, requires at least two such
  words, and retains at most one short digit-bearing namespace. Review iteration
  4 rechecked accepted W8M-style/exact-slot/natural non-slot names, both opaque
  token families, known formats, invalid/duplicate input, value-free API
  diagnostics, canonical BRP isolation, transaction staging, documentation,
  and the full automated gate; no unresolved P1/P2 remains.
- No target contact, browser session, submission, candidate, transaction,
  account action, runtime execution, downstream pin adoption, release, or
  deployment occurred. Separately authorized publication advanced OpenUdon
  `origin/main` to exact commit
  `561fd933097abe90cdbe2b59b56e0fa33d4d41c1`.

## Publication State

- Independent `origin/main` resolution returned exact OpenUdon commit
  `561fd933097abe90cdbe2b59b56e0fa33d4d41c1`. This Tofu closeout records that
  publication; downstream pin adoption remains a separate decision.
