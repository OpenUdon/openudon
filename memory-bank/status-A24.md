# A24 Status - Honest Guided Registration Profile Completion

State markers and commit rules are defined in [milestone.md](milestone.md).

## State

Complete and published at OpenUdon
`bf90537be965ace7885d43de53dab1f9eb2f2ab9` after one separately authorized
downstream no-submit observation exposed a generic guided-wizard mismatch.
Independent `origin/main` resolution returned that exact commit. No additional
public target contact, registration submission, runtime execution, release, or
deployment occurred during A24 implementation and verification.

## Task Ledger

| Item | State | Notes |
|---|---|---|
| A24.1 Make symbolic registration inputs and success proof honest | `[+]` | Published at OpenUdon `bf90537be965ace7885d43de53dab1f9eb2f2ab9`. The common value-free `contact_name` slot is a default; credential steps select declared slots and safely reuse one password slot for confirmation; success is a separately reviewed origin/optional-path/role/name declaration marked `operator_reviewed_deferred`, unobserved during authoring, and runtime-proof-required. The server preserves schema-valid BRP construction, current-generation macro/submit candidate binding, no-submit authority, and fail-closed validation. |

## Acceptance

- The default guided form includes `identifier`, `password`, and
  `contact_name` symbolic slots without accepting any value.
- Credential macro steps choose only declared slot symbols and may reference
  one slot more than once; duplicate password use does not duplicate or expose
  a binding.
- A success locator is supplied as an explicit operator-reviewed role/name,
  is never described or retained as observed by the no-submit session, and is
  disclosed as deferred until a separately approved runtime proves it.
- Only current-generation observed candidates may supply macro locators; the
  exact submit control remains bound to one observed unique candidate.
- Focused builder/API/UI tests, browser-free qualification, full offline tests,
  vet, documentation-memory, JavaScript syntax, and diff checks pass.

## Non-Goals

- No additional target observation during A24 implementation, registration
  submission, credential entry, account creation, attestation, transaction
  adoption, package write, runtime execution, release, or deployment.
- No UWS or Browsertools wire change. Their existing contracts already permit
  an explicit reviewed success condition and repeated references to one
  symbolic credential slot.

## Verification

- Pinned Go 1.26.6 with `GOWORK=off`, `GOPROXY=off`, `GOSUMDB=off`, and
  `GOTOOLCHAIN=local`: focused `./internal/icot/ui` tests, full `go vet ./...`,
  and `make check` all pass.
- The `icot_ui_browser` test set compiles with no tests executed; Chromium was
  not launched. `node --check internal/icot/ui/assets/app.js` passes.
- Documentation-memory and all three scoped repository diff checks pass.
- Review iteration 1 found and corrected two honesty drifts: the success help
  now requires separately reviewed documentation or a separately reviewed
  local contract, and an unknown post-submit path no longer defaults to `/`.
  The tagged UI test now asserts a genuinely pre-submit observed label rather
  than the declared post-submit success label. No P1/P2 finding remains.

## Publication State

- The scoped OpenUdon implementation commit
  `bf90537be965ace7885d43de53dab1f9eb2f2ab9` is published at exact
  `origin/main`. Tofu implementation record
  `e7f2bd0adba328e5a95319c5f89f380986cc4f67` precedes this publication-state
  closeout. W8M must still adopt the final published Tofu closure and pass a
  fresh exact-pin preflight before another target session.
