# OpenUdon authoring after iCoT retirement

Kinet owns chat, interviews and browser UI. OpenUdon is an external CLI/artifact
producer; it has no terminal interview, UI listener or application-control server.
This migration implements the approved 23 retained capabilities in
[the inventory](icot-retirement-inventory.md). M95 acceptance and final M20/W29
consumer adoption remain separate ledger gates.

| Previous entry | Current entry and owner |
| --- | --- |
| `icot` interview, UI and control | Kinet workflow chat/jobs/UI; OpenUdon step and capture commands |
| Seed/from-example/print/agent | `openudon authoring draft`; local structured seeds, no terminal reads |
| `icot browser-authoring plan` | `openudon authoring browser-plan`; inert plan only |
| Registration field definitions | `openudon authoring registration-draft`; [strict pure contract](registration-draft.md) |
| Browser capture/live | [Public supervised browser capture](browser-capture-protocol.md); both authentication/TOTP and registration |
| Browser transaction UI/terminal | [Browser-author and package lifecycle](browser-package-handoff.md), orchestrated by Kinet |
| Lint/reconcile/repair/report | `openudon authoring lint|reconcile|repair|report` |
| Scorecards/variants/replay/model evaluation | `openudon authoring scorecard|variants|replay-eval|authoring-eval` |
| Build/assess/approval-template/run | Existing OpenUdon commands; separate digest-bound approval |
| UI native qualification selectors | [Native v6 and integration v7](browser-system-eval.md); retained earlier readers |
| Three release binaries | Current `openudon` and `udon-runner`; earlier published archives unchanged |

## Draft publication

`--print` renders without state writes; combining it with `--agent --yes` or a
`--report` output path is refused before effects. Read-only `--agent --print` retains its frontier report. Agent mode reports by default;
its explicit `--yes` publication remains a separate operator choice. Missing mandatory input returns the
existing structured frontier without publication. `--yes` explicitly authorizes
publication of a complete seed; `--force` selects replacement source precedence,
not execution authority. Fast from-example mode reuses existing deterministic
elicitor defaults with a private empty reader, without model calls or autosave.
Closed draft creates no terminal transcript and never claims one was written;
existing `.icot` histories remain readable and consumer conversation persistence
belongs to Kinet. Reconcile publication requires `--yes`, while `--print` is
read-only. Model/repair options on closed draft refuse with the explicit expert
evaluation replacements. Explicit expert evaluation retains fake-provider/default offline tests and
separately invoked real-model evaluation. Native source, credential, rollback,
optimistic conflict and safety rules remain unchanged.

The `.icot` artifact paths and `openudon.icot-*` report schema names remain
readable. Never rename or rewrite historical artifacts or relabel a retained
report as newly executed evidence. Authoring and udon-ui retirement is deferred.

## Capture and runtime

Use exact issued actions/digests/revisions; review, finish, import, author,
prepare and promote are distinct decisions. Native receipt and selected package
inspection must agree. A reviewed transaction is not a promoted transaction.
Capture credentials are entered directly in the isolated worker; no credential,
cookie or session value enters a package, audit or public result. A planned or
promoted package grants no runtime or target authority. Browser registration
success remains deferred until separately approved execution proves it.

## Owner checks

`make check`, `make standalone-build`, `make authoring-variants-validate`,
`make authoring-variants-coverage`, `make authoring-scorecard` and retained
reference-seed/replay fake tests are browser/model free. `make browser-capture-check`
requires an explicitly selected synthetic sandboxed display; it exercises both
actual public native package journeys. `make qualify` selects a complete current
native gate with three fresh passes and exact dependencies, never a development
cache. Keep private failure diagnostics and joined process evidence.

Current archives still build for Linux/macOS/Windows. Reviewed browser-package
admission requires qualified Unix owner/mode/hard-link checks. Non-Unix platforms
refuse that operation until ACL equivalence qualifies; other public commands remain
buildable. This does not weaken Unix admission or claim a Windows browser journey.
