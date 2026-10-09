# P10 — Package v3 browser supplement

**Stage:** Kinet STG-12, Phase A. **Owner:** OpenUdon.
**State:** Pending. No row has started. Review 0/10.
**Source baseline:** `477bf1a53591da70c98979a7a351db4fafac0ff9` (clean at planning).
**Coordinator:** [Stage 12 contract](../../../kinet/docs/stage12.md). This
package-local milestone and status own acceptance. Planning was approved on
2026-10-09 (Kinet R65). It authorizes these planning files only.

**Planning reconciliation.** Stage 12 planning review, 2026-10-09; source priority not supplied. Review baseline and current revalidation: `477bf1a53591da70c98979a7a351db4fafac0ff9`; includes the uncommitted Stage 12 planning files. F02 (confirmed P1; runtime owner Udon:M53) requires independent evidence checks in P10.4. Evidence: `../browserdriver/src/driver.ts` executes the action sequence before extracting outputs; a closed failure code alone supplies no non-dispatch witness. This is approved review intake, not a closing-review iteration; all rows and review counters remain pending/0.

**Parallel planning provenance.** Stage 12 parallel execution review,
2026-10-09; source priority and separate review baseline not supplied. Current
revalidation `477bf1a53591da70c98979a7a351db4fafac0ff9`, including uncommitted planning files. F01 (confirmed local P2) adopts scoped workflow ownership; F02 (confirmed Lower) replaces strict ordering with readiness; F04 (confirmed local P2) requires frozen consumer checks. F03 (partially confirmed Lower; owner-selected early freeze) moves report/host contract definition into existing M51 rows and joins actual producer qualification later.
Evidence: the old Kinet goal/launch rules, owning agent rules, existing pending
dependencies and sibling consumer checks; Udon `go.mod`/`pkg/execute` import no
OpenUdon code. This approved intake changes no row state or review counter.

## Dispatch and lease boundaries

**Depends on.** Kinet:M51, UWS:C10, Browsertools:M33. All required prerequisites must have
accepted closure at exact revisions; sibling producer adoption also needs
independently verified publication. A priority position never supplies authority.

**Downstream impacts.** Kinet:M56, Kinet:W20, Kinet:M52, OpenUdon:A32.

**Write set.** The owning `openudon/` package's implementation, tests, ordinary
documentation, manifests and qualification outputs only as required by this
milestone's existing scope, plus `tabilet/memory-bank/status-P10.md` in its
assigned worktree. Excludes `AGENTS.md`, `tabilet/GOAL.md`, shared memory-bank
files, other statuses, evolution, stages, history/knowledge, the package audit
database/sidecars, coordination docs and launch input. The coordinator alone applies shared-memory and closure
changes serially; no child writes a sibling repository or user ledger.

**Contracts read.** Immutable exact prerequisite artifacts listed above, the
M51 native-owner-reviewed contract/fixtures when applicable, the assigned
package baseline and frozen shared-memory/consumer snapshots captured at
dispatch. Cross-package checks use read-only exact snapshots or approved
published module inputs, never changing sibling checkouts. Record full source,
artifact and fixture hashes in the later execution brief; contract drift pauses
affected leases for coordinator reconciliation. Existing no-workspace/no-directory
substitution requirements for ordinary published adoption remain in force.

**Parallel-safe.** yes. Eligible only under the explicit Stage 12 lease opt-in, with no dependency path or bidirectional read/write conflict against any running lease.
At most one live milestone per package. All tests use private lease ports,
disposable stores/caches/browser profiles and unique output directories.

## Dependencies and handoff

**Upstream.**

- [Kinet:M51](../../../kinet/tabilet/memory-bank/status-M51.md) consumer contract.
- Accepted and published
  [UWS:C10](../../../uws/tabilet/memory-bank/status-C10.md) and
  [Browsertools:M33](../../../browsertools/tabilet/memory-bank/status-M33.md).
- Builds on accepted [P09](../docs/history/status-P09.md) package v3. Today it
  refuses browser paths, the `browser-profile` kind and non-http/fnct leaves
  (`packagev3/records.go:89-104`, `execution_plan.go:248-257`).

**Downstream.**

- [Kinet:M56](../../../kinet/tabilet/memory-bank/status-M56.md)
- [Kinet:W20](../../../kinet/tabilet/memory-bank/status-W20.md)
- [Kinet:M52](../../../kinet/tabilet/memory-bank/status-M52.md)
- [OpenUdon:A32](status-A32.md)

Consumers adopt only the exact accepted and independently published revision and
SDK version.

## Tasks

| Item | State | Notes |
|---|---|---|
| P10.1 — Versioned browser supplement | `[ ]` | Admit browser-profile, authentication and registration source artifacts and browser leaves in a versioned package v3 browser supplement. Browser shape tables are untrusted until reproduced through the published Browsertools:M33 verifier. Non-browser v3 package, approval and run-evidence identities stay byte-identical. |
| P10.2 — Public browser verification subset | `[ ]` | Promote only the needed `browserverify` / `browsertransaction` behavior to public packages: profile transaction receipts v1–v4 and capture review evidence. Import nothing from synthesize, workflowintent, elicitor, projectwizard, udonrunner or trustedrunner. |
| P10.3 — Browser approval and Authority | `[ ]` | Approval and Authority bind browser actions, origins, side effects, confirmation policy, credential-slot names and saved-session reuse permission. Changed bytes refuse. No value is carried. |
| P10.4 — Browser run evidence | `[ ]` | Independently verify browser reports/run evidence, including `runevidence.BrowserConfig`, against exact action/dispatch identity. A typed error after authentication, registration or action dispatch cannot establish no effect. Preserve unknown after successful write then failed extraction, lost response, cancellation or crash; reject evidence that relabels uncertainty as safe retry or continuation. Positive non-dispatch proof is explicit and cannot transfer old authority. Evidence stays value-free; no private runtime import. Use M51 frozen native-report/interface fixtures for independent implementation beside Udon:M53. Preserve public/private isolation; M56/M52 subsequently prove actual producer interoperability. Contract drift pauses affected leases for coordinator reconciliation. |
| P10.5 — Qualify and publish | `[ ]` | No Udon import; v2 and v3 history readers and existing wires unchanged; consumer builds (Kinet author/exec workers); SDK publication handoff under named authority. |

## Acceptance and verification

**Acceptance.**

- A browser package with verified shapes builds, assesses and yields exact
  browser Authority.
- Tampered profiles, shapes or approvals refuse.
- Non-browser identities are unchanged.
- The public closure contains no retained-set package.

**Verification.**

- `GOWORK=off go test ./...` and `go vet ./...`.
- `make check`.
- Public-closure and boundary guard.
- Wire and immutability fixtures.
- `git diff --check`.

## Execution policy

One coordinator owns the integrated ledgers, shared memory and serialized
integration/closure. Serial execution remains the default. Concurrent leases
require this milestone's declared safety, frozen inputs, a complete explicit
Kinet goal request and the Stage 12 agent-rule opt-in. Each lease has one
in-progress row and one assigned milestone; its persisted review count survives
resume/rebase. Commit policy comes from that later request. Source publication,
deployment and live operations retain separate named authority. Planning and
status markers grant none; audit stays disabled.

## Review

Whole-milestone review: 0/10, not started.

## Frozen M51 producer input checkpoint — 2026-10-09

Kinet:M51.3 completed at local commit `589b844628b358167fe0a0137eded11c1028875c`. The [native-owner-reviewed contract](../../../kinet/docs/stage12-browser-contract.md), [host ABI](../../../kinet/docs/stage12-browser-host.go.txt), [public declarations](../../../kinet/docs/stage12-browser-public.go.txt) and [fixture manifest](../../../kinet/fixtures/stage12-browser-v1/manifest.json) SHA256 `0c445a5c90d2c09be561e713c364747f4ab9a3698ea8ab46e7b7b774ccf16bad` are exact producer inputs. Both native/public contract reviews pass with zero P1/P2/P3. This is input review, not this producer's implementation, publication or closing review. Kinet:M51 still awaits browser qualification and whole review; no dispatch prerequisite is satisfied by these task commits alone. Rows and persisted review counters remain unchanged.

Use the exact source/extraction/identity maps, per-leaf old driver protocols, complete-plan credential lease, private registration inputs, automatic TOTP versus claimed push continuations, original admitted deadline and separate bounded teardown. The supported consumer profile has one durable session binding per execution and permits other fresh named contexts. M16's immutable host-private save plan preserves v2–v11; candidate creation precedes Join and encrypted host acceptance follows Join/current-generation checks. Preserve report-v5 uncertainty and independently reproduce canonical/golden digests and positive/negative host witnesses. No current artifact claims real conformance or adoption.
