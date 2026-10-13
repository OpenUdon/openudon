# Authored browser authentication timeout contract

P12.1 freezes the approved P12 producer contract. This document specifies the
P12.2 implementation and P12.3 compatibility evidence; it does not claim that
either has passed. The [retired full specification and status](../tabilet/docs/history/status-P12.md)
retain the authoritative execution, acceptance and action requirements.

## Frozen inputs and observed conflict

The input producer is accepted/published OpenUdon P11
`ec2fa2666dad604a4512f8fff4155d269c8597b3`, ordinary module
`v0.1.1-0.20261013095506-ec2fa2666dad`. Public browser profile reproduction uses
the existing UWS C10 module
`github.com/OpenUdon/uws v0.0.0-20261009220914-f01a2542410c` and Browsertools M33
module `github.com/OpenUdon/browsertools v0.0.0-20261010115646-1859f5e34367`.
The independent UWS HCL codec remains
`v0.0.0-20261008043726-b099f6803277`; no dependency changes are required.

The read-only native fixture input is accepted Udon
`5bbf2924d85b9f13b89bb8c000cb1b12e6c56491`, ordinary module
`github.com/genelet/udon v0.0.0-20261010194342-5bbf2924d85b`. Its
`pkg/uwsprofile/execution_profile.go` authentication validator requires an
operation timeout in `(0,600]` seconds. Its
`pkg/execute/browser_preflight.go` requires an established preceding
authentication for browser actions in the selected native profile.

The input public [flat operation predicate](../packagev3/execution_plan.go)
requires `op.Timeout == nil`. Both browser supplement construction in
[browser.go](../packagev3/browser.go) and execution-plan construction invoke
that predicate. Thus an authentication operation with a native-required timeout
cannot pass the frozen public browser package path, while an untimed operation
cannot pass native authentication compilation. F01 is the confirmed local P1
that P12 repairs. Static source inspection establishes this conflict; it is no
claim about an installed runtime or a physical browser journey.

## Authored field and exact admissibility

Use the existing public UWS `OperationExecutionFields.Timeout *float64`, encoded
as the optional operation-level `timeout` number. The unit is seconds. A
freshly authored timed authentication operation must have a finite value `t`
with `0 < t && t <= 600`. Fractional seconds are allowed; `600` is included.
Zero, negative zero, negative numbers, values above `600`, NaN and infinities
are outside this contract. Invalid numeric encodings and non-number values are
refused by the existing strict byte/schema boundaries or the bounded predicate.
The existing UWS version/schema requirements still apply.

The timeout exception applies only to these exact, case-sensitive operation
profile names:

- `uws.browser-authentication-call.1.0`
- `uws.browser-authentication-call.1.1`

The operation must still resolve as a validated browser authentication call:
the corresponding exact `x-uws-browser-authentication` extension, independently
reproduced source/flow shape, matching profile version, symbolic credential
slots and session ordering remain required. A profile string alone cannot turn
an HTTP, function, browser action, registration or malformed operation into an
authentication call. Existing closed extension and source-binding checks apply.

Add a browser-specific admissibility predicate at both supplement construction
and execution-plan construction, before browser plan/runtime admission. It
preserves the generic `flatOperation` predicate and all of its other conditions.
Only `Timeout` receives the bounded authentication exception. In particular,
operation dependencies, `when`, `forEach`, `wait`, parallel groups and
success/failure controls retain their existing restrictions. All step,
workflow, control-flow, extension and complete browser-only inventory checks
remain in force. Missing or inconsistent verified supplement/call provenance
must never allow a timed operation into a different executable profile.

This does not allow operation timeouts on HTTP/API calls, function calls,
browser actions or registration calls, or timeouts on steps/workflows. Their
public flat execution restrictions remain unchanged. Generic UWS and native
runtime semantics remain with their owners; no runtime format, public wire or
schema version is added.

## Untimed compatibility and fresh confirmation

The public field remains optional. An omitted timeout (`Timeout == nil` after
the existing decoder) retains the same public Build/Assess/Verify and
InspectHistory behavior as before P12. The new predicate does not make a
timeout mandatory for historical public proof. Existing untimed execution-plan
metadata also retains its prior behavior with the same supplied admission
adapter; such metadata does not establish native support.

Native Compile/CheckSupported must report the genuine unsupported result for
untimed authentication. A runtime admission adapter must not turn that refusal
into successful executable compatibility. Byte inspection and historical proof
remain distinct from executable admission, fresh approval and host authority.

Build, verification, assessment and plan derivation must never inject a default
timeout, clear an authored timeout or rewrite an approved workflow. The author
supplies the timing in new workflow bytes. Those exact bytes are retained for
proof, independent native compilation and subsequent review.

Kinet M52 adopts the accepted/published P12 successor in a freshly authored
package, presents the authored operation timeout in its exact confirmation,
and obtains fresh package/plan/receipt/Authority identities. Accepted W20
source, old approvals, frozen goldens and prior outcomes remain historical
evidence at their original identities. A timed successor never inherits an
untimed package's approval or a consumed grant. W21/U15/M53/A32 reconcile to
the accepted producer through their own pending work; physical browser proof
belongs only to M53.4.

## Existing byte and hash bindings

Timing is already covered by the complete workflow bytes; add no parallel
timeout field to the browser supplement, execution leaf, Config or receipt.

1. Build retains `WorkflowYAML` unchanged. The manifest workflow SHA256 hashes
   those complete bytes, and `BrowserSupplementV1.WorkflowSHA256` binds that
   exact workflow. Source and shape hashes keep their independent profile
   provenance; changing operation timing does not itself change profile bytes.
2. The supplement artifact, manifest input inventory, assessment, handoff and
   package digest retain their existing byte/hash relationships. Verify
   reproduces the supplement and assessment from an independent byte snapshot.
3. `ExecutionOperation.ConstraintsSHA256` covers the complete lossless raw
   operation and step, data digest and worker identity. The raw operation
   includes the authored timeout, preserving its existing numeric projection.
   The ordered execution plan digest therefore binds that timing as well.
4. Browser Config, finalized approval and Authority already bind workflow,
   supplement, inputs, package, handoff and confirmed plan identities. A
   timeout change requires rebuilding/reverifying and fresh exact confirmation;
   an old plan, receipt, Config or Authority must refuse the changed successor.

The bounded predicate uses the typed float value only for admissibility. Existing raw numeric
lexemes, extension values, named outputs and diagnostic ordering keep their
current lossless binding behavior. A caller changing an exported copy does not
change a VerifiedPackage's private snapshot; submitting changed bytes requires
verification against their own current identities.

## Required regression matrix for P12.2 and P12.3

These are required future checks, not P12.1 passing test results. Use separate
additive fixtures and retain all old golden files and accepted evidence.

| Case | Public proof/plan requirement | Native fixture requirement |
|---|---|---|
| Auth call 1.0 and 1.1, positive fractional timeout, ordinary positive timeout and exactly 600 | Build/Assess/Verify reproduce the complete package and supplement; derive/check the plan, browser approval and Authority with valid independent inputs | Feed the exact public workflow bytes to Compile and CheckSupported with authentication followed by read and write leaves; support succeeds with zero effect callbacks |
| Untimed auth call 1.0 and 1.1 | Preserve existing public proof/history and prior metadata behavior; do not insert a timeout | Compile/CheckSupported refuses the missing required timeout; an admission-backed public plan must propagate refusal |
| Zero, negative zero, negative, just above 600 and an oversized finite timeout | Refuse at the relevant schema/admissibility boundary; no executable plan or authority | No supported result; validly encoded invalid timing may be checked by the pure native refusal path |
| NaN, either infinity, malformed numeric text, non-number values and invalid numeric encodings | Refuse at strict byte/schema decoding or bounded predicate; exercise direct nonfinite predicate inputs where applicable | Publicly refused byte inputs supply no native compatibility or effect evidence |
| Timed action, registration, HTTP/API or function; timed step/workflow; wrong or case-variant auth call profile | Keep existing flat/profile restrictions; a timing exception must not escape authenticated browser provenance | No new native capability is claimed |
| Timed authentication with other flat controls, extra/disallowed extensions, incompatible source/flow version or broken session order | Refuse under the existing independent checks, including before runtime admission where already required | No effect callbacks; retain genuine pure native unsupported behavior |
| Timeout-only byte tamper under an old manifest/package identity | Verify refuses changed inventory/proof; changing one digest cannot replace the complete chain | Never feed silently repaired bytes to native compilation |
| Legitimately rebuilt timeout successor with an old constraints/plan hash, Config, receipt or Authority | Derive the new bound identities and refuse every old approval/authority pairing | Compare the same fresh workflow byte digest used by public proof and native compilation |
| Numeric lexemes, open extension values, named output ownership/resolution, old untimed corpus and non-browser golden bytes | Preserve existing number/extension/output semantics, assessment ordering and frozen goldens; additive timed fixtures cover their combination | Native fixture is limited to supported authentication/read/write inventory |

Assessment security/type uncertainty remains independent. A valid timeout does
not upgrade an indeterminate assessment to compatible, grant credentials, allow
session saving/reuse or relax browser approval checks.

## Isolated pure native proof and readiness boundary

P12.3 uses a separate external fixture whose private dependency graph may import
the exact accepted native module. OpenUdon's public graph must continue to
exclude private Udon and retained OpenUdon internal packages. The fixture records
exact source/module identities and workflow bytes/digests, calls only pure
Compile/CheckSupported, and asserts zero effect callbacks. It must not call Run,
launch a browser, resolve real credentials, make requests or execute leaves.
Pre-publication source fixtures are explicitly producer compatibility evidence;
ordinary published-module verification and actual Kinet worker adoption remain
separate gates.

P12.1's cheap cached readiness check at the planning source baseline
`9fa3bfc2d7cc77b8c621001025fb32d714dd45b5` used cached Go 1.26.6 and the retained
owner module/build caches. Offline `go list -deps` over `packagev3`,
`browsercontract`, `browsertransaction`, `browserverify`, `approval`, `authority`,
`runevidence`, `handoff`, `trust`, `wire`, `digest` and `udonreport` succeeded:
373 packages, 35 reached modules, exact C10/M33/APItools owner versions and zero
private Udon or retained OpenUdon internal imports. Workspace resolution,
module proxies, VCS acquisition, checksum-network access, authentication helpers
and toolchain downloads were disabled; the check used existing caches without
creating a new cache or copying source trees.
This is package-resolution readiness only. It establishes neither compiled
closure qualification nor P12.2/P12.3 acceptance, publication, consumer adoption
or physical browser proof.

## Retirement link maintenance

Only the opening active-file links were reconciled to the permanent retired
record after P12 acceptance. The original qualification used SHA256
`3be7b99ba08415151f9fe6c141f7819e445d5035dbe87459bdc85a62da75295a`;
its exact bytes remain in reviewed/published Git history and retained coordinator
evidence. This link-only maintenance changes no authored contract or qualified
fixture. [Retirement provenance](p12-retirement-links.md) records the distinction.
