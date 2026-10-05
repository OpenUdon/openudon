# M97 qualification and consumer handoff

## Qualified source and artifacts

Qualified application source: `f4127c159e18fa66619659bc3c4b8757b7022267`.
The clean detached exports contain no worktree overlay or ambient go.work.
The owner records added after qualification change documentation only; they do
not change this application, module, schema or fixture selection. Acceptance
and publication are recorded separately in the owner milestone record.

Source archive SHA-256: `87b14b27bf5c1b72280cdaab3293ec8207809e10780eace66885070c0cbcb5ec`.
Qualified closure SHA-256: `5fb02718562932acd64c2e1a19245e0773441a0e718ddba577aa90b15694035c`.
Build-input inventory SHA-256: `38a2b26a19ab073fddb497331bd6877b43756e18ddbb3ef110c73d923436a324`.

| Artifact | SHA-256 |
|---|---|
| openudon | `4f0c6cad518331f4dcf61de986c4b21fbda9aadaf1065a2efaafbda2f1220c17` |
| udon-runner | `5e0cf3623db21a2aef72e7f5daefa34ae8672306658c35222995e4af3735407e` |
| docs/fixtures/broker-execution-v1/manifest.json | `dd3357dc4a61e15f006e92f3d24fdbb72f88f9368336ee55375b08ed9ed0294f` |
| docs/fixtures/broker-handoff-v1/manifest.json | `903b0d5b885edc46df87e9158ee76fd7a3f422a62846dd4fed3d3f9584925539` |
| docs/fixtures/udon-http-broker-v1/manifest.json | `7289085b14f744504019ae6d607d351f74522ed3f930ecbbfd159200fd2ddb5f` |
| docs/schemas/openudon.broker-authority.v1.schema.json | `1fafe34b8de49b79b461879ab6e14da91f0b9f1616c89d7d75e90b5e5bf899e6` |
| docs/schemas/openudon.approval.v2.schema.json | `e269e2fe1d1dc3371819e205d0c8296821c80692d59f3f6ced424b3180b4216d` |
| docs/schemas/openudon.executor-run.v3.schema.json | `1db06ae5a484888c1e9aef9dac7a4bd31b3ff185b9c9b91ff489630bf164cbb1` |
| docs/schemas/openudon.run-evidence.v4.schema.json | `ce96682064c9d9bcb9e4e0b998b68d21e3ad379223ac598af1840fdaef68768c` |

Both CLI build records bind the full qualified source with vcs.modified=false.
The inventory binds cached Go 1.26.6, exact module/sum bytes, 48 modules,
2,784 compiled source/embed inputs and seventeen clean exported repositories.
Native reports contain nineteen source entries because Browsertools/UWS also
occur in the fourteen-source historical M45 browser closure. Broker execution
uses the separate accepted M46 executor below; the browser closure is not
relabeled as M46. No private executor module import or download was used.

## Passing verification at the exact source

- Full make fast and focused broker/authority race tests passed.
- Clean-export make check passed: standalone build/default tests, sibling checks
  and APItools boundary. Clean-export go vet ./... passed.
- Seven of seven exact published M46 runtime cases passed: success, lost write,
  wrong identity, unknown write, refused read, failed write and cancelled write.
- Fresh affected registration smoke passed (95.048 seconds), with reuse=false;
  this is development evidence and does not itself qualify a runtime.
- Full make qualify passed: four offline gates and three fresh native repeats,
  thirteen stages each (39/39), followed by independent report verification.
- The owned display process joined, its socket disappeared and private temporary
  X authority was removed. Chromium sandboxing remained enabled throughout.

| Evidence | SHA-256 |
|---|---|
| native-current-loopback.json | `0e49eff3de0d8c21bac7742316ee7c239adce4111b3d48c723daacab8e153676` |
| native-current-offline.json | `0fda632a7dbdcec15df8229adf2387b4d940f189865b725d0400ba7a8660c231` |
| Fresh smoke report | `36414dd2895d9183feeba1ca04d7add53139bba6d2edd0400917076455938c76` |
| Launcher and verified teardown record | `be0008b454ea9779aa6a4704d680767bf508539b9e4c8dd474fb975e4404fc63` |

Private frozen evidence root:
`/var/tmp/kinet-stage9-p_9plhx7/m97-final/`.
`qualified-build.json` binds artifacts, source selection, native reports,
smoke/teardown records and exact-source verification log digests. Passing native
reports are under qualification-2/. No failed or diagnostic result substitutes
for them. The resource launcher invokes the exact compiled CLI and cached Go,
with disk-backed temporary files and two build workers. It changes no validator,
stage count, source bytes or sandbox rule.

## Producer provenance and consumer requirements

Udon M46 accepted source: `95c5850fd446e06ac6f79d943774db67e417c989`.
Completed owner publication: `71071537890599e98541abe8ca564490660dfd16`.
Executor SHA-256: `53bb9e8976f67c6a5880f07248a99cd195e6285ce7a5a68239793dd5dc6eb429`.
Closure SHA-256: `a286f21a5dcd4180b7ec19bf0630188ddacec26e0532198db14decd0c86b2069`.
Original fixture manifest: `7289085b14f744504019ae6d607d351f74522ed3f930ecbbfd159200fd2ddb5f`.

Portable OpenUdon examples have independent manifest hashes and never replace
actual runtime proof. [The handoff contract](broker-execution-handoff.md)
defines versions, digest bytes and supported bounded HTTP shapes. Both main
and async evidence redact the private broker configuration path; the regression
assertion failed before the correction and passed afterward. Additional tests
preserve public sandbox refusal and production approval requirements and cover
API-key header/query metadata without passing credential values.

Kinet M35, W14 and M37 must bind the actual accepted source, independently
verified publication, both CLI artifacts and this closure before adoption.
The trusted host owns current grants/credential revisions, destination and
request constraints, finite quotas and durable claims before every I/O.
Unknown writes are neither retried nor reported as safe failure. Inspection's
RawMessage constraints must be json.Compact-ed before digest checking, preserving
member order and number spelling. Broker binding revisions hash revision
metadata, never secret values.

Existing authoring/capture pin `c2f161d762bc9f2217bbf0c34b00cdef64b0f7d0`
and frozen browser evidence remain unchanged. No W8M adoption, public-service
access, API/model call, deployment, install or customer data is included.

## Earlier attempts (excluded from acceptance)

The pre-fix source `c55fb8eb208802955c8aac3e8ec6dc61b2d80bfc` and its
artifacts remain under the separate m97-native/ root. Initial preflights refused
inherited umask 0002; process-local 0022 passed without changing global policy.
Early offline attempts hit the existing /tmp user quota; a resource launcher
initially lost precedence to Go's automatic GOROOT PATH prefix. Two disposable
launcher typos stopped before qualification stages. All display teardowns passed.

Pre-fix attempt 4 passed its four offline gates and ten native stages, then
failed registration_capture_handoff. Focused diagnostics passed afterward;
the failure was not reproduced and no cause is claimed. Attempt 5 was stopped
before native stages after task inspection found the async evidence redaction
defect. Private logger overlays and all partial/failed reports are excluded.
The corrected source passed a fresh smoke and the full fresh qualification
above; no old smoke/cache result was consumed.
