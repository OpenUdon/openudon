# iCoT journey and consumer inventory (M91)

## Decision boundary

This is the M91.1 inventory for the confirmed Stage 5 goal. It describes current
capabilities and proposed replacement ownership, not implemented replacements.
All capabilities remain available during 5A. No additional capability is
proposed for discontinuation. M95 removes the old iCoT entry points only after
Gate 5B, W8M W28 adoption, and evidence for each replacement. The retained human
inventory-disposition checkpoint in Kinet's launch reference still applies;
automatic Git publication approval does not approve removals or Gate 5B.

Baseline source: `e12a6488b86cafddb9298c7917de84fbc1cc85ff`; inventory
inspection also includes approved Stage 5 planning and prerequisite closure.
E21/P07/P08 are retired without reopening their task rows or review counters.
P08's v11 selection is current; P07's v10 preparation result retains its
historical scope. Udon M45's published source/build is recorded in M92, whose
adoption remains pending. W8M W27 retains its own owner and live-operation gates.

## Retained journeys and proposed replacements

The identifiers below identify inventory entries, not milestone IDs. “Replace”
means replace the entry point while retaining its behavior. Each owner must
supply exact replacement commands/protocols and checks before M95 can close.

| Entry | Capability and current source | Disposition | Replacement owner and required evidence |
| --- | --- | --- | --- |
| J01 | Terminal project brief, outcome, boundaries and intent authoring: `cmd/icot`, `internal/icot/icot.go`, `internal/projectwizard` | Replace entry; retain authoring | Kinet W08/W10/U07 with existing OpenUdon step/build/package commands. Prove exact proposal confirmation and rejected proposals do not publish ledger/package writes. |
| J02 | Seeded answers, from-example, print-only/noninteractive reports, prompt modes, local drafts and transcripts: `internal/icot/elicitor`, `internal/authoring` | Retain reusable logic; replace interactive entry | M91.3 extracts neutral records/discovery; Kinet owns its conversation and persistence. M95 must map print/report and seed/replay cases to explicit commands or review an additional disposition; no silent loss. Existing `.icot` artifacts remain readable. |
| J03 | Dependency frontier, readiness, missing bindings, decision evidence, deferral and bounded repair: `internal/icot/elicitor`, `internal/icot/engine` | Retain semantics; replace interaction | M91.2–M91.3 and Kinet W08/W10. Preserve atomic frontier validation, cancellation, readiness errors, hypothetical/pending labels and repair bounds in deterministic fixtures. Public UWS owns execution semantics. |
| J04 | Transactional file planning, digest recheck, rollback, indeterminate recovery and redacted review: `internal/icot/artifactwriter`, `internal/authoring`, `internal/packagepipeline` | Retain; extract shared implementation | M91.2; step source/bind and browser imports consume neutral code. Prove byte-equivalent artifacts, zero-write conflict/refusal, symlink containment and exact recovery identity; no second writer implementation. |
| J05 | Expert `step source add`, `step candidates`, `step bind`, `step check`, `flow-review`: `internal/stepauthoring` | Retain existing public commands | M91.2–M91.3 remove iCoT implementation dependencies. Existing versioned step conformance fixtures and command JSON remain byte-identical. |
| J06 | Explicit local source roots, native source selection, security alternatives and source refresh: `internal/icot/elicitor`, `internal/sourcecatalog` | Retain; extract | M91.3 and M94. Retain source provenance, root bounds, credential aliases, provider constraints and freshness checks. APItools owns metadata interpretation. |
| J07 | Catalog planner sessions, corpus/provider discovery and opt-in remote lookup: `internal/icot/elicitor/catalog_plan.go`, `internal/sourcecatalog` | Retain; replace caller | M91.3/M94 and Kinet W10. Consume published APItools M81/M80; preserve insufficient/ambiguous/blocked outcomes, explicit network consent and digest-bound export. Never turn an incomplete lookup into proof of absence. |
| J08 | Static browser profiles/registries, guided bundles, virtual sources and attached verification: `internal/icot/elicitor`, `internal/browserverify` | Retain; extract shared helpers | M91.3/M91.4 and M93. Preserve profile identity, allowed origins, dependency traversal, source freshness and independent result validation. Mock browser output is not page replay evidence. |
| J09 | Non-executing browser-authoring plan/handoff: `internal/icot/browser_authoring.go`, `internal/icot/browserauthor` | Replace entry; retain handoff | M91.4/M93 commands and Kinet W09. Prove that planning grants no browser action, account mutation or runtime approval. |
| J10 | Authenticated browser capture, reviewed goal/dashboard/origin, human sign-in and MFA including TOTP: `internal/icot/browser_author_live.go`, `internal/icot/browserauthor`, Browsertools author worker | Replace entry; retain full journey | M91.4/M93 versioned capture, Kinet W09/M19/U07, W8M W28/W29. Demonstrate the authenticated/TOTP journey visibly at M93.5 and freshly in consumer qualification. Keep private codes/credentials outside application payloads and logs. |
| J11 | Registration capture, public wizard definitions, verification approvals and registration authority: `internal/icot/browserauthor`, `internal/browsertransaction`, `internal/icot/ui` | Replace entry; retain full journey | M91.4/M93, Kinet W09/M19/U07 and W8M W28/W29. Demonstrate registration separately from authenticated capture. W8M's cancelled production-registration operation is not reopened. |
| J12 | Reduced observations, navigation/preview, issued action IDs/revisions, blocked-script diagnostics and disclosure consent | Retain capture protections | M93/Kinet W09. Cover stale or mismatched decisions, exact origins, deadlines, bounded POSTs, consent, human-guided default and value-free diagnostics for both modes. Source: `internal/icot/browserauthor`, `internal/icot/browser_author_live.go`. |
| J13 | Browser package transaction review/prepare/promote/recover: `internal/icot/browser_transaction_terminal.go`, `internal/browsertransaction`, `internal/packagepipeline` | Replace entry; retain transaction engine | M91.2/M91.4/M93 and Kinet W09. Retain digest/revision binding, explicit action approval, indeterminate recovery, atomic package selection and cancellation. |
| J14 | Bundled Browsertools workers, subprocess groups, readiness, timeouts and teardown: `internal/icot/browser_worker.go`, `internal/processgroup` | Retain; move dispatch | M91.4/M93 adds the worker under `openudon`; iCoT uses the same implementation until M95. Prove no orphan workers, no public listeners and no private executor import. |
| J15 | Loopback UI and application/control APIs: `internal/icot/ui`, `internal/icot/ui.go`, `docs/application-control.md` | Replace old transports; retain user capability | Kinet W08–W10/M19/U07 and M93 capture protocol. M95 removes old UI/control endpoints only after replacement session, state, approval, concurrency, cancellation and recovery checks. Versioned historical evidence retains its original meaning. |
| J16 | Lint, drift/reconcile, bounded mapping/output/dependency repair, review/report generation: `internal/icot/lint.go`, `repair.go`, `reconcile_replay.go`, `report.go` | Retain; expose through neutral commands | M91.2/M91.5 then M95. Map each expert command and preserve fixture outcomes, report schemas/digests and no-write modes before removal. |
| J17 | Provider-free reliability scorecards, variants validation/coverage, deterministic replay and reference seed matrix: `internal/icot/scorecard.go`, `variants.go`, `internal/eval`, `internal/icotreport` | Retain evaluation; replace command caller | M91.5. Run the same fixture corpus and compare expected classes, counts, results and verification. Preserve immutable reports and distinguish historical evidence from new qualification. |
| J18 | Optional real-model authoring/replay evaluation and provider/model configuration: `internal/icot/authoring_eval.go`, `reconcile_replay.go`, `internal/workflowintent/provider_client.go` | Retain explicit evidence path; replace command caller | M91.5/M95, with Kinet's configured provider on the consumer path. Fake/model-free tests are default; inventory approval and source publication authorize no billable or live provider run. |
| J19 | Approval-template, build/assess/package/promote, trusted sandbox dry/real execution and evidence on legacy packages: `internal/trustedrunner`, `internal/udonrunner`, `internal/udonreport` | Retain existing OpenUdon entry points | M92 pending refusal and Kinet W08/W10/M20. Legacy declared versions and historical `.icot` files stay intact; no rewrite grants new approval. Preserve report v1–v5 meanings and explicit real-run authority. |
| J20 | Browser 1.10 persistent v11 dispatch, older v10 pairing and registration exclusions: P08, `internal/trustedrunner/browser.go`, `internal/udonrunner/browser.go` | Retain exact runtime boundary | M91.4/M91.6/M95. Verify supported/inactive/mixed profile choices and exact local/container arguments. Udon and Browserdriver own runtime semantics. |
| J21 | Native integration/scenario/journey qualification and E23/E24 actual tool/source/namespace bindings: `internal/browsersystem`, `internal/browserscenario`, `internal/browserintegrationeval` | Retain; rebase helpers | M91.4/M91.6/M95. Preserve frozen readers/locks and current-stack inputs; extraction changes source identity and needs actual new qualification. Kinet's fixed-M90 script does not qualify M91; W08 later qualifies adopted M92. |
| J22 | W8M authenticated/TOTP and registration packets, constrained destination, aliases, claims, deadlines, diagnostics and teardown | Replace authoring launch; retain consumer authority | Kinet M19 and W8M W28/W29. Inspect `browser-workflows/check/internal/authoringcontract` and W8M `browser-workflows/check/internal/operating/author.go`; independently validate result/package identity. Kinet never runs W8M prepare/verify-registration/count operations. Preserve W27's consumed outcomes and ownership transfer rule. |
| J23 | Release binaries, standalone builds, test/docs/tutorial and qualification command references: `.github/workflows`, `Makefile`, `docs` | Retain gates/docs; replace iCoT entry references | M91.5/M91.6/M95. Publish and test replacement commands before removing the iCoT binary/embedded UI; no release gate quietly loses coverage. |

## Extraction seams verified in current code

- `stepauthoring/source.go` and `bind.go` import iCoT's artifactwriter;
  `stepauthoring/flow_review.go` imports its elicitor review. Move shared code
  once and rebase callers; keep package transactions and review checks intact.
- `browserscenario` imports iCoT scenario/diagnostic/transaction/UI helpers.
  Qualification coverage must migrate with those helpers, including both
  capture modes and preserved report readers.
- `internal/authoring/{types,interactive,progressive_lifecycle}.go` imports
  Authoring's `icot`. Its readiness/question aliases already have neutral
  upstream counterparts in `session`/`readiness`, while progressive interactive
  adapters remain intertwined with the terminal loop. M91.3 must separate
  transport-specific adapters from neutral consumers and reuse public Authoring
  primitives; copying the generic loop would create duplicate implementation.
- `internal/eval/authoring_variants.go` consumes report/corpus code, and the
  Makefile still invokes iCoT for scorecards, variants, replay and lint.
  Preserve those noninteractive expert/evaluation capabilities explicitly.
- `internal/icot/browser_worker.go` dispatches Browsertools workers. Moving
  it must preserve both registration and authenticated workers and private
  environment/action bounds; moving only UI assets is insufficient.

## Acceptance record for each retained entry

Before M95 closes, augment each entry with exact replacement command/protocol,
accepted producer revision, conformance fixture or test, and downstream
qualification where required. A pending implementation is not a replacement
pass. No unchecked entry becomes discontinued by default. Changes to these
retained dispositions require explicit user review. Gate 5B also requires all
5A milestones accepted/published and both capture journeys demonstrated.

M91.1's inventory review is separate from the later milestone deep review and
Gate 5B. The proposed inventory currently has 23 retained/replaced entries and
zero proposed capability discontinuations. Approving it permits extraction
under M91's existing scope; it does not accept unfinished replacement evidence.

## Inventory approval

The user explicitly approved all 23 dispositions before extraction. Later
Gate 5B, human-visible qualification and each entry's replacement evidence
remain required. No additional capability discontinuation was approved.

## M95.1 verified migration map — 2026-10-02

Consumer acceptance: W8M executed source46accdb39f57597c3dd50640e8c2b2f13a834e69,
acceptance7cbea4933adf6a8c55864ddb6257e8d33edd0950, closurefd571ef0db8ab438a33cf454dd859f0fbb91fc5d;
Kinet U07 applicationd3589d4742272b3d024328192768374da4c0c637; OpenUdon M96
applicationeed683f27d448ca96af90e7bc5987967a6cd0335. Complete actual build/proof
identities and private evidence digests are in status-M95.md. Gate5B and all23
dispositions are approved; zero extra discontinuations. This table records
existing seams and required removal checks; planned entries are not passing
replacement evidence. Final M95.4/.5 must supply its new qualified revision.

| Entry | Actual replacement / retained implementation | Verification owner before closure |
| --- | --- | --- |
| J01 | Kinet workflow jobs/chat, OpenUdon step/build/package | Kinet U07/W28 accepted; M95 retained package checks |
| J02 | Neutral authoring/elicitor records; planned closed `openudon authoring draft` for seeded/from-example/print/report/prompt modes | M95.2 draft conformance, noninteractive no-input refusal and legacy draft bytes |
| J03 | `internal/authoringengine`, `elicitor`, step pending/check; Kinet conversation | Native readiness/frontier/refusal tests and neutral draft checks |
| J04 | `internal/artifactwriter`, `packagepipeline`, step source/bind | Atomic/rollback/symlink/optimistic guard and package recovery tests |
| J05 | `openudon step source add/candidates/bind/check`, `flow-review` | Existing step v1 conformance and real-main dispatch |
| J06 | `internal/sourcecatalog`, step source add/discover/provision | Source root/provenance/credential alias/freshness tests |
| J07 | `openudon step discover/provision`, APItools native catalog, neutral elicitor planner records | M94 accepted contracts; draft/source import and five-outcome conformance |
| J08 | Browser profiles/registry/verification helpers in elicitor/browserverify | Profile traversal/identity/refusal tests; mock versus actual evidence labels |
| J09 | Existing browserauthoring.Plan; planned expert non-executing plan entry | M95.2 public plan read-only/no-action test |
| J10 | `openudon browser-capture`, Kinet capture/external sessions | M93/M96/U07/W28 both-mode/TOTP accepted; fresh neutral producer qualification |
| J11 | Same public capture with registration protocol and explicit authority | Separate verification refusal/approval, no production registration reopened |
| J12 | `internal/browsercapture`, `browserauthor`, `browserauthoring` | Issued action/revision/deadline/origin/disclosure/stale-decision tests |
| J13 | `openudon browser-author plan/apply`, `package prepare/promote/inspect/reconcile`; Kinet exact decisions | Native writer/package recovery and lost-output fixtures |
| J14 | `openudon __browsertools-worker`, `internal/processgroup` | Worker dispatch, bounded environment, joined teardown |
| J15 | Kinet embedded UI/public jobs/external v1; no OpenUdon listener needed | U07/W28 current controls/recovery accepted; M95 removal/dependency exclusion |
| J16 | `openudon authoring lint/reconcile/repair/report` | Retained exact reports and mutation/no-write tests |
| J17 | `openudon authoring scorecard/variants/replay-eval` and eval corpus | M95.2 closed draft path; deterministic classes/counts/report verification |
| J18 | `openudon authoring authoring-eval/replay-eval`, explicit existing provider configuration | Fake/model-free tests; no paid/live provider run authorized |
| J19 | `openudon build/assess/approval-template/run/package` | Legacy packages/.icot unchanged and real external executor checks |
| J20 | `internal/trustedrunner`, `udonrunner` supported v10/v11 dispatch | Exact profile/protocol pairing, local/container args and executor boundaries |
| J21 | Retained historical readers; planned neutral current native/scenario/integration selectors | M95.2–.4 new source-bound qualification; no old UI gate silently omitted |
| J22 | W8M constrained Kinet packets, independent native/result selection | W28 accepted both modes/counts; W29 final pins independently qualified later |
| J23 | Two retained release binaries openudon/udon-runner; neutral expert and capture gates | M95.3 CI/release/docs checks and M95.4 actual frozen qualification |

Historical inventory source references above are preserved as baseline evidence.
M95 does not delete historical package .icot/session/transcript artifacts, alter
old report schemas/locks or retire Authoring/udon-ui. No capability is silently
dropped because its old transport is removed.

## M95 final replacement evidence map — 2026-10-02

This additive map preserves the approved historical inventory above. No extra
capability is discontinued. The first review-fix revision was
`8ce4dd2ac7d74a656c7fecbe0bc42d2fa459e4c5`; its extra agent flag refusal was
corrected during review2. Final exact application and closing acceptance are
recorded separately in M95's ledger. Native browser application qualification is honestly
bound to `bb9863ed097a56460082640668f7cc0a121002ac`, CLI SHA256
`dfb0e9f0764aecf7ee448d5c6d19adea6945bf79244123231c5f39524176ece3`.
The later application delta only refuses conflicting draft print/publication
flags; its focused/full/offline/integration results are separate. Native
owners, entry, fixtures, locks and modules are unchanged; the original native
report is never relabeled as a run at the final revision.

Full native v6:39/39 in three fresh complete passes, independently verified;
integration v7:17 required passed,3 optional unrequested; offline v6:4 passed.
Original qualification summary SHA256
`ea80c3d40ce3c544f33845e931d50dc17cb8a318e466ba9fd9c04ef6b601e201`.
Final full/default and integration, legacy CLI and closing-review results are
recorded in the permanent M95 record when accepted. Existing U07/W28 migration
is accepted evidence; final Kinet M20 and W8M W29 remain pending consumer gates.

| Entry | Replacement command/protocol | Actual fixture/check and downstream boundary |
| --- | --- | --- |
| J01 | Kinet chat/jobs plus `openudon step`, build/package | Accepted Kinet U07/W28 exact proposal and no-write refusals; M95 full step/package tests. M20/W29 requalify final pins. |
| J02 | `authoring draft --answers/--from-example --print/--agent`; explicit `--yes` publication | `authoringcli/draft_test.go` seed corpus, no terminal reads, frontier, no-write print/conflict and transcript checks; retained session schemas. Kinet owns new conversation persistence. |
| J03 | Neutral `authoringengine`, elicitor and public step check/pending | Full authoringengine/elicitor/stepauthoring readiness, pending/deferral, evidence and bounded repair tests; no implicit execution. |
| J04 | Neutral artifactwriter/packagepipeline and browser-package commands | Full atomic/rollback/symlink/conflict tests; browserpackage lost-output, cancellation and exact native receipt tests; no copied writer. |
| J05 | `step source add`, `step candidates`, `step bind`, `step check`, `flow-review` | Full stepauthoring conformance and cmd/openudon dispatch tests; versioned fixtures retained. |
| J06 | `step source add/discover/provision`, neutral sourcecatalog/elicitor | Full source-root, provenance, digest, credential-alias, freshness and inactive-browser-source refusal tests. |
| J07 | `step discover/provision`, native APItools catalog records | M94 accepted five-outcome/source-import fixtures; full catalog/step tests at M95. No implicit remote lookup or incomplete negative. |
| J08 | Elicitor/browserverify profiles, registry and guided verification | Full traversal, profile identity/expiry and source validation; current integration/native suite separately proves actual browser behavior. |
| J09 | `authoring browser-plan` | authoringcli expert conformance and browserauthoring tests: inert handoff, no writer/browser/runtime authority. |
| J10 | `browser-capture` authenticated mode, `browser-author`, package lifecycle | Fresh native v6 `public_authenticated_package`, actual login/TOTP/goal/dashboard and selected native package; M20/W29 actual final consumer gates pending. |
| J11 | Same public commands with registration mode; `authoring registration-draft` | Fresh native `registration_capture_handoff` and `public_registration_package`: typed field/history, verification refusal/grant, review/finish/import; no production registration. |
| J12 | Issued capture action/revision/digest protocol | Full browsercapture action replay, changed observation, deadline, disclosure and value-free diagnostic tests; both actual native modes. |
| J13 | Browser-author plan/apply; package prepare/promote/inspect/recover | Full browserpackage/browsertransaction/packagepipeline conformance, partial-write/recovery refusal and exact selection; native BAP/BCP transaction and both packages. |
| J14 | Public capture worker/processgroup owner | Native lifecycle/teardown, private TCP-disabled display join/auth deletion; full cancellation/stream/processgroup tests. |
| J15 | Kinet owner sessions/jobs/UI; public OpenUdon capture/package | U07/W28 accepted session/approval/reconnect; full capture and integration boundary checks. No OpenUdon UI/control listener remains. M20/W29 final transport adoption pending. |
| J16 | `authoring lint/reconcile/repair/report` | Full expert regression/replay/report tests; reconcile requires explicit publication, print stays read-only. Historical schema labels remain readable. |
| J17 | `authoring scorecard/variants/replay-eval` and make aliases | Provider-free retained corpus103/103,0failed/unsafe; variants validation/coverage and reference-seed/replay fake checks. Original reports retain identity. |
| J18 | Explicit `authoring authoring-eval/replay-eval` | Fake-provider/model-configuration tests and real main dispatch; no billable provider run selected. Closed draft rejects model-extraction options with replacement command. |
| J19 | Existing build/assess/approval-template/run/package | Actual declared1.11 public rebuild/assess/exact sandbox approval/dry-run with executor.invoked=false; opaque .icot bytes/modes unchanged. M20 owns final Kinet accept check. |
| J20 | Existing trustedrunner/udonrunner v11 dispatch | Full supported/inactive/mixed profile and argument tests; native Udon browser contract/CLI stages at exact M45 closure. Older paired readers/locks unchanged. |
| J21 | Current native v6/integration v7; earlier readers unchanged | Fresh39 native stages,17 integration gates and independent verification; old W28 v5 independent verification only, no historical re-execution/relabel. |
| J22 | Kinet external-authoring v1 and W8M authoringcontract | Accepted U07/W28 packet/owner/destination/TOTP/native result inspection; producer strict capture/package conformance. M20 and W29 qualify actual final binaries; W27/live authority unchanged. |
| J23 | Two CLI release binaries, neutral Make/CI/docs gates | All12 Linux/macOS/Windows builds, strict docs/memory, retained corpus; locally executed CI16-repo/14-replacement preparation and conflict/token-persistence refusal. No remote CI/tag/release claimed. |

Replacing terminal/UI transports transfers interaction to Kinet. It does not
remove neutral authoring/evaluation, rewrite .icot histories, grant runtime
approval, retire Authoring/udon-ui, or accept final downstream qualifications.
