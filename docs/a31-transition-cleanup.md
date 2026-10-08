# Stage 11 transition cleanup inventory

A31.1 scans all73 current owner packages using standalone offline `go list -json
./...`, including production imports, transitive Deps and test imports. The
machine-readable inventory retains complete edges and source/graph hashes.
This is an owner source inventory, not runtime or live-operation authority.

| Private package | Direct production consumers | Transitive owner consumers | Disposition |
|---|---|---|---|
| `internal/synthesize` | `cmd/openudon`, `internal/authoringcli`, `internal/browserpackage`, `internal/browserscenario`, `internal/eval`, `internal/packagepipeline`, `internal/smokematrix`, `internal/trustedrunner`, `internal/udonrunner` | 17 | Retain through its Stage12 gate. |
| `internal/workflowintent` | `cmd/openudon`, `internal/artifactwriter`, `internal/authoringcli`, `internal/authoringengine`, `internal/browserauthoring`, `internal/browserpackage`, `internal/browserscenario`, `internal/browserworkflow`, `internal/elicitor`, `internal/eval`, `internal/stepauthoring`, `internal/synthesize`, `internal/trustedrunner`, `internal/udonrunner` | 25 | Retain through its Stage12 gate. |
| `internal/elicitor` | `internal/artifactwriter`, `internal/authoringcli`, `internal/authoringengine`, `internal/browserauthoring`, `internal/browsercapture`, `internal/browserpackage`, `internal/browserscenario`, `internal/stepauthoring` | 13 | Retain through its Stage12 gate. |
| `internal/projectwizard` | `internal/authoringcli`, `internal/authoringengine`, `internal/browserpackage`, `internal/elicitor`, `internal/stepauthoring` | 14 | Retain through its Stage12 gate. |
| `internal/udonrunner` | `internal/releaseevidence`, `internal/smokematrix`, `internal/trustedrunner` | 13 | Retain through its Stage12 gate. |
| `internal/trustedrunner` | `cmd/openudon`, `cmd/udon-runner`, `internal/browserscenario`, `internal/browsertransaction/engine`, `internal/packagepipeline`, `internal/releaseevidence`, `internal/smokematrix` | 12 | Retain through its Stage12 gate. |

## Public and consumer boundaries

M98 supported neutral approval/authority/digest/handoff/trust/wire/udonreport/
runevidence remains public. P09 packagev3 and credentialpolicy remain supported;
the full public transitive graph imports no OpenUdon internal or private executor
module. Horizon/HCL dependencies in that public graph are declared public codec
implementation and cannot be removed solely because legacy HCL also uses them.
Synthesis-coupled v2 build/assess/simulation stays private compatibility without
a new supported v2 library promise. Current public surface/wire manifest tests
remain the deletion gate for these contracts.

Current Kinet public author imports digest/packagev3/trust/wire; its isolated
private executor imports approval/authority/packagev3/wire. Parent Kinet imports
no OpenUdon internals. Qualified workers keep their exact P09 source and binaries;
A31 source publication cannot silently change their module or runtime pins.
The separately pinned browser CLI atc2f161d762bc9f2217bbf0c34b00cdef64b0f7d0 and
frozen W8M/Ramen consumers remain untouched.

## Command and legacy HCL closure

`cmd/openudon` retains synthesize/build/promote/assess, explicit step commands,
package prepare/promote/inspect/recover, approval/run/run-evidence, authoring,
eval/smoke/release checks and browser capture/author/qualification entrypoints.
`cmd/udon-runner` retains trusted runtime handoff. Browserpackage and browser
scenario/system/transaction qualification transitively use synthesis, elicitor,
projectwizard, workflowintent and trustedrunner/udonrunner. Step authoring also
uses elicitor/projectwizard and legacy intent/HCL output. Removing any whole
listed package would break those existing source entrypoints or fixtures.

Legacy HCL parsers/renderers remain in workflowintent, synthesize runtime_data,
stepauthoring and Udon-runner format dispatch, alongside public codec dependencies.
The accepted new primary non-browser Kinet path uses explicit packagev3 construction;
that ownership change does not make independently callable OpenUdon CLI behavior
or its browser qualification dead. Safe cleanup is limited to truly unused
private forwarding code; remaining removal is expressly deferred.

## Safe cleanup and Stage12 deletion gates

The unexported trustedrunner.validateTierState forwarding helper has no tracked
production/test reference beyond its declaration. M98 already moved real tier
validation into public approval.ValidateTierState/Validate; its public behavior
and schema remain. A31.2 may remove this dead private adapter only.

Stage12 OpenUdon owns deletion of the retained private synthesis/intent/elicitor/
projectwizard/runner/browser adapters after Kinet W19/U14 browser replacement is
accepted and its exact runtime/authoring/capture/report/approval consumers are
reconciled. For each removal: eliminate every production/test/command edge in
this inventory, preserve public M98/P09 surface/wire fixtures, qualify all
remaining browser/capture/package/transaction paths, and record any public CLI
retirement in an explicitly approved Stage12 scope. Frozen historical pins stay
readable and do not move. Remaining live CLI/eval/packagepipeline consumers also
need an explicit owner replacement or supported-retirement decision; browser
completion alone cannot justify deleting them. No deferred package removal is
claimed delivered by A31.
