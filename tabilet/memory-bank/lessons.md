# Lessons

Keep concise, reusable lessons that still affect decisions. Consult the topics
relevant to the current task before substantial changes; this is not a session
log or a requirement to produce one lesson per milestone.

Maintain relevant lessons when ordinary work produces reusable, evidence-backed
learning, and consolidate them during milestone closure. A still-applicable
lesson stays here even after its supporting milestone retires; closing a task
or reaching an arbitrary file size is not a reason to discard useful knowledge.

For each lesson, use a descriptive heading and record when it applies, the
lesson, why it matters, and links to supporting tasks, verification, or retired
records. Keep product facts in `product.md`, system contracts in
`architecture.md`, and commands in `tech-stack.md` rather than repeating them.

Merge duplicates. Before materially replacing or removing an obsolete lesson,
append its previous wording, source, reason, and replacement reference to
`tabilet/docs/history/knowledge.md` under the retirement rules in
[milestone.md](milestone.md#long-term-memory-and-retirement). Preserve evidence
links when merging lessons. Revalidate historical evidence before applying it
to current work. Routine wording edits need no journal entry or archive run.

## Bind retained report verification to a versioned lock

When report verifiers consult embedded compatibility data, make the report
version select an immutable lock snapshot. Updating a mutable current lock can
otherwise invalidate retained reports even when their bytes and recorded
digests are unchanged. E21 froze the M86 scenario and integration lock bytes
before advancing current-stack support; the original integration and two
scenario v2 reports continue to verify. See [status-E21](../docs/history/status-E21.md) and
[M86](../docs/history/status-M86.md).

Advancing a report version must carry its supported phase/assertion vocabulary
as well as its pins and required markers. Test both retained and new versions
with positive count evidence and negative non-count evidence. M92 review R3
found that all fourteen native journeys ran successfully while the outer v5
verifier still rejected the retained `udon_v11` phase through a v4-only branch.
Evidence: `TestBrowser110CurrentReportRequiresVersionedCountEvidence` and
M92's status record, resolved through the milestone/history index.

## Bind advertised coverage to versioned qualification selectors

When release guidance claims a feature is covered, its versioned gate must
require the named producer, schema, consumer and runtime tests for that feature.
A passing package suite or a separate end-to-end journey does not prove that
the integration report attests the advertised seams. Preserve the old selector
for existing reports and add the explicit markers to a new report version.
E22 review iteration 2 found and corrected this gap for Browser 1.10 count
evidence; see the [E22 history record](../docs/history/status-E22.md).

M91's frozen integration detected a stale v10-only marker after P08 had
broadened the real test to v10/v11. Relocate current paths and require the
actual modern marker in a new report version; preserve all historical
selectors/locks. Compare fixture/asset bytes and shared function bodies as
well as counters, then run fresh native qualification at the extracted
source. See M91's closing record (resolved through the history index).

## Exercise browser profiles through trusted package preparation

When adding a browser profile version, test the package lifecycle and trusted
dry-run that consume it, not only its producer, schema, and browser journey.
E22's v4 count qualification passed while the trusted-runner rank-10 dispatcher
still rejected `uws.browser.1.10` during W8M package preparation. Preserve
existing profile-version mappings and verify the restricted dry-run without
invoking an executor or browser. See [P07](../docs/history/status-P07.md) and the W8M W22.5
package-preflight record in that repository (`tabilet/memory-bank/status-W22.md`).

Preparation alone missed a second boundary: P07 selected v10 for Browser 1.10,
so W8M W24.5 authenticated but Udon rejected the count action before DOM
extraction. Test the exact prepared run-config's profile/protocol pairing
through the external executor handoff as well. P08 selects v11 and rejects
incompatible active mixes; see [P08](../docs/history/status-P08.md).

## Validate task tables with the installed runner

When repairing or adding a status ledger, use outer `|` table delimiters and
the exact backticked markers defined in `AGENTS.md`. A visually readable table
without outer delimiters was invisible to the installed API runner, while
`[x]` was rejected as an unknown state. This left historical rows uncounted
and hid in-progress work. The repaired [A12 ledger](../docs/history/status-A12.md) and
[A21 ledger](../docs/history/status-A21.md) show the accepted syntax; the runner entry point is
recorded in [tech-stack.md](tech-stack.md#harness-runner). Check the full active
ledger with that runner's read-only parser before launching an execution loop.

## Keep qualification chronology with its evidence owner

When a cross-repo qualification closes, retain exact pins, failed attempts,
report digests, and review history in its milestone or status record. Product,
architecture, and tech-stack should state the current fact each owns and link
to that evidence. Copying release chronology into all routine-read documents
repeated long passages and left older "current" headings beside newer work.
The [knowledge journal](../docs/history/knowledge.md) preserves the wording
removed during the September 24 consolidation, and [E20](../docs/history/status-E20.md)
illustrates the owning task evidence.

## Preserve uncertainty across executor report boundaries

When deciding whether an approved run can be repeated, compare the report's
unique attempt, workflow-byte digest and complete ordered inventory with the
reviewed staged plan. A missing/rejected report proves no unstarted step;
an incomplete run can retain truthful complete inventory. Leaf results and
later run failures are separate observations. Preparation refusal must not
claim executor invocation. Evidence: M90's strict report/invocation checks and
`TestPublishedM44Qualification` (failed read, killed write, checkpoint failure,
missing/stale report and duplicate refusal), [M90 history](../docs/history/status-M90.md).

## Validate the exact JSON wire before interpreting typed records

Go's encoding/json accepts case aliases and converts null strings to zero
values. DisallowUnknownFields alone does not enforce a canonical closed wire
schema. Reject aliases, nulls and forbidden optional-field presence before
interpreting report outcomes; reject timestamp precision the consumer cannot
compare faithfully. Evidence: `internal/udonreport/v5.go` and its alias/null,
presence, identity and timestamp mutation tests; M90 review finding R90-3 in
[M90 history](../docs/history/status-M90.md). Keep legacy wire behavior under its own version.

Capture's M93.1 uses the same exact-schema rule: the public resource is
embedded, duplicate keys/unknown nested fields are rejected before interpreting
the closed union, and conformance uses the actual decoders. Evidence:
`internal/browsercapture/conformance_test.go` and
`docs/browser-capture-protocol.md`. Its process-local gate additionally binds
review to immutable issued arguments and consumes approval/refusal once; a
failed new-state publication must destroy old action authority.

## Keep hypothetical execution separate from executable approval

Project unresolved contracts only in memory for preview, using public runtime
semantics and explicit response fixtures/examples/schemas. Never publish or
approve a fabricated operation. Reserve every public identity namespace,
including parallel groups, before choosing synthetic names. Pending admission
must check every workflow/branch and both generated artifacts before stored
quality, credentials or executor dispatch. Evidence: M92 pending refusal,
public orchestration and namespace regression tests, resolved through its
milestone/history record. Export provenance and labels without private values;
preview success grants no execution authority.


## Preserve an explicit empty choice at a native JSON boundary

A reviewed empty selection is distinct from absent authority. A pointer to a
nil Go slice marshals as JSON null and decodes as an absent pointer; cloning a
reviewed selection must retain a non-nil zero-length slice when empty is valid.
Test the native worker JSON round trip as well as adapter method calls. Evidence:
M93.5 actual public authenticated/TOTP capture with no outputs; shared
browserauthor.checkpointResponse and
TestCompletionRetainsExplicitEmptySelectionAcrossWorkerJSON. This retains the
worker's refusal of missing/unconfirmed authority instead of relaxing it.

## Bind catalog selection through native export and package publication

An indexed discovery result proves only its reported scope. Preserve its native
artifact references, raw identity and selectors through selected export, then
independently validate selector binding before a confirmed package transaction.
Keep every selected provider link and only applicable advisory overlays; advice
does not authorize runtime behavior. Stage privately and commit source bytes,
manifest and provenance with the existing atomic writer, so refusals cannot
leave a partially provisioned package. Evidence: M94 catalog round-trip,
wrong-selector/drift/collision/cancellation/schema tests and source-backed
public fixtures in docs/fixtures/catalog-discovery-v1. Read-only native SQLite
may maintain transient WAL/shared-memory files; assert unchanged registration
data/index/raw bytes rather than confusing those locks with application writes.

## Separate setup seeds, current codes and recovery codes

For TOTP authentication, capture a value-free challenge while the person
enters a current code; resolve the setup seed only at the trusted runtime.
A list of recovery codes is a different credential type. Make the human
prompt identify which input is needed, and verify that clipboard/display
isolation permits the selected entry method. Issuer-specific shape rules
belong to the consumer, not the portable profile schema. The
[received consumer handoff](../../docs/consumer-totp-qualification-handoff.md)
records existing coverage and owner dispositions without secret values.

## Inspect declared bindings before consuming an attempt

Credential-section prose can accidentally declare a quoted kind token as
a binding. Compare the actual declared/expected inventory against the exact
workflow and profile, then run native preparation with appropriate transient
inputs before execution. Keep any diagnostic reproduction separate from
the original error: discarded child stderr cannot be reconstructed by a
later offline probe. See the [synthetic before/after reproduction](../../docs/consumer-totp-qualification-handoff.md#inspect-the-actual-declared-credential-inventory-before-execution).

## Check native reuse against the actual host and report roles

Freeze and recheck the admitted source/dependency/tool/display/environment
and workstation inputs across long qualification stages. A package update
can invalidate reuse while sources stay unchanged; an empty ignored runtime
directory can fail native cleanliness. Cache misses stop without implicit
full-suite fallback. Compare full producer execution evidence for freshness,
not a reduced output digest that can repeat across runs. The
[received handoff](../../docs/consumer-totp-qualification-handoff.md) links
the integrated E23/E24 guard regressions and qualified consumer evidence.

## Carry exact native bytes and original capture identity across adapters

Nested raw JSON compaction/re-indentation can invalidate a receipt hash even
when the parsed object is unchanged. Encode exact native start bytes as base64,
then reuse native schema/candidate validation after decoding. Treat receipt
hashes as bindings to original approved capture evidence, not signatures over
replacement records. Match available native login/dashboard/goal/origin facts;
do not invent missing registration profile-ID/initial-URL constraints. Evidence:
M96 exact-byte/case/numeric-array/start-substitution conformance and both native
modes/TOTP in the source-bound capture/package journeys.

## Bind native writer transients and preserve qualification execution identity

A full inventory guard must exclude only exact temporary/backup paths reported
by the existing writer, never filename suffixes. Keep the writer's legacy
callback semantics and rollback/cleanup; inspect partial build/output loss before
a new proposal. Serialize native and integration gates on supplied snapshots;
record/remove only known owned empty generated directories before strict source
checks. Preserve failed and superseded execution identities and independently
verify process/display teardown. Tests-only deltas get their own checks and do
not relabel native source/time. Evidence: M96 guard/refusal/partial-build/replay
regressions, frozen native39/integration17, failed preflight and superseded bundles.

## Bind current CI setup to actual native lock snapshots

Do not infer executor prerequisites from only the four primary browser repos.
Prepare all current replacement inputs from the versioned build lock, reject
conflicting identities before cloning, and verify clean exact heads. GitHub read
headers belong only in transient child environments, never persisted config.
Evidence: M95 local CI preparation executes the actual workflow body against
local exact objects, verifies16 distinct repositories/14 offline Udon replacements
and conflict refusal/no credential persistence. Native acceptance uses its own
recorded clean source and independent reports.

## Wait for an actual private input checkpoint

Synthetic runtime tests must wait for Apply readiness before filling fields
disabled during package/runtime startup. A short browser action timeout is not
a workflow readiness deadline. Keep the existing runtime limit and approval
policy, and expose only fixed phase codes when a component fails. Evidence:
M95 bounded diagnosis, readiness/privacy regression and corrected native39
qualification; the earlier failure remains preserved, not retrospectively explained.
