# Reviewed capture package handoff

OpenUdon M96 is accepted and published at qualified application `eed683f27d448ca96af90e7bc5987967a6cd0335` through verified source/review publication `d77f6d51262d0f311910070bc4a43f662260cd7e`. Review1 passed; [its retired package-local evidence](../tabilet/docs/history/status-M96.md) preserves qualification and acceptance. Kinet M19 still owns pending consumer delivery.

The published M93 capture contract imports canonical profiles and a receipt. Existing package preparation requires a fully reviewed/built package. At that published baseline, ordinary virtual-source adoption required retained iCoT. The new browser-author commands now bridge capture and ordinary package authoring with separate approval.

M96 adds a bounded non-iCoT command over the existing neutral authoring engine, elicitor and artifact writer. It binds receipt/source/input identities and retains separately issued authoring approval before native build. Capture approval, package promotion and runtime authority remain separate. Both authenticated/TOTP and registration are retained. No duplicate profile/review writer or browser rerun is allowed.

Kinet remains an external CLI consumer. After the exact producer is accepted/published, M19 reconciles its shared pin, narrowly admits native public review artifacts and implements recoverable prepare/promote/inspect/recover delivery. Its external v1 envelope stays unchanged. M95 must retain this replacement when retiring iCoT. Full contract/fixtures are frozen in M96.1; no future revision or acceptance is implied here.

## M96.1 frozen command and wire

Public commands:

```text
openudon browser-author plan --example DIR --request FILE|-
openudon browser-author apply --example DIR --request FILE|- --expected-plan sha256:HEX --confirmed
```

`plan` is read-only, model-free and network-free. `apply` recomputes the plan,
requires exact matching authority and readiness, commits reviewed authoring
through the existing neutral writer and runs native package build. It does not
promote, capture, login, submit registration or execute a workflow. File paths
remain explicit local CLI inputs. The example must be within the working root.

Request version `openudon.browser-author.v1`, kind `request`, max256KiB UTF-8:

| Field | Contract |
| --- | --- |
| `request_id` | Bounded lowercase request identifier |
| `start` | Base64 of the exact UTF-8 native start file bytes; decoding reuses the closed native start schema and its exact receipt hash |
| `receipt_path` | Clean example-relative `expected/browser-capture/ID.json` |
| `receipt_sha256` | Lowercase64hex of the exact receipt file bytes |
| `transaction_sha256` | Native tagged transaction digest |
| `input_sha256` | Optional tagged current package inventory digest for read-only discovery; required for apply |
| `expected_totp` | Required explicit Boolean; true requires captured native TOTP evidence, never a synthesized step |
| `registration_authority` | Required non-secret registration requester identity; absent for authentication; grants no submit permission |
| `workflow_name` | Bounded lowercase name of the proposed workflow |
| `flow`, `action` | Exact native operations. Flow selects authentication/registration; action selects authenticated capability. Missing choices produce a non-ready read-only catalog, never an implicit choice |
| `cleanup_disposition` | Registration only: `delete_separately` or `retain_dedicated_test_identity` |
| `inputs` | At most32 unique `{name,type,sensitive?}` declarations; no defaults or values |
| `input_bindings` | At most32 native parameter→`inputs.NAME` references, declared above; never arbitrary expressions or literals |
| `allow_overwrite` | Explicit permission for the exact authoring conflicts bound into the reviewed plan; default false |

Unknown/duplicate keys and capitalization aliases, mixed modes, native schema violations and unsafe paths
are refused without payload echo. Native canonical receipt/source/review/expiry
and original-start policy validation remain mandatory on each plan/apply. The source pair
and session/credential lowering reuse neutral elicitor semantics. Registration
is an inert unsupported-runtime recipe, retaining all native safety policies.

Plan version above, kind `plan`: request ID, tagged exact request-byte digest,
tagged current input digest, receipt and transaction digests, native candidates,
operation catalog, readiness, blocker codes, optional native preview, native
write conflicts, exact native file actions, package-relative `artifact_sha256` byte digests for every prepared file and `plan_sha256`. Arrays retain native deterministic order.
The plan digest hashes encoding/json's compact typed struct serialization with
its own field empty. Exact request hashes include whitespace; base64-decoded starts retain
the original native bytes for receipt matching regardless of outer JSON formatting. A missing initial
input digest permits catalog inspection only; use the returned digest to form
a bound request, re-plan, then confirm that exact plan.

Receipt digests must come from the separately approved original capture, not
from a replacement receipt supplied by a model. Receipts are content-addressed
local evidence, not signed attestations. The exact original start hash binds all
start fields in both modes. Authentication additionally matches its native
recipe's initial login and dashboard success proof, and its public goal review.
Registration recipes may navigate among the approved origins after the initial
page; their source does not retain the separate capture profile ID or initial
URL. Preserve their original receipt/start binding and native origin/transaction
checks rather than inventing a profile-ID equality or narrowing valid recipes.

Input inventory is sorted package-relative regular-file `{path,sha256,bytes}`
records (tagged byte digests), serialized with encoding/json and tagged SHA-256.
`.git` is excluded; symlinks/special files, foreign ownership, hardlinks and
group/world-writable package files/directories/root are refused. Bounds:512 files,
8MiB per file,32MiB total. No input may change between review and commit. The shared native writer reports
its exact own staged temporary/backup files to the inventory guard; only those
paths are excluded during its pre-replacement comparison, never a wildcard.

Result version above, kind `result`: request ID/digest, approved plan digest,
outcome `authored` or `build_failed`, written relative paths and native quality
status, with `cleanup_required: true` when the native writer reports incomplete temporary cleanup (warning text is not disclosed). Cleanup uncertainty requires inspection before delivery. `authored` proves local authoring/build only. `build_failed` means the
native authoring commit succeeded but build did not pass; it never claims no
writes. Lost output requires inspection of actual artifacts and a fresh proposal,
never automatic apply replay. Replay is refused by changed package inventory.
Package prepare/promote/inspect/recover remains a distinct later authority.
Reports are at most2MiB. Personal previews/candidates stay transient; observers
retain only approved metadata. [Synthetic request fixtures](fixtures/browser-author-v1/README.md)
have placeholder hashes and grant no authority.

The adapter reconstructs imported candidates via native browserauthoring/browsercandidate validation, then uses the pure neutral elicitor workflow lowering and the existing artifact writer directly. Imported profiles already occupy physical source paths; opening the interactive engine would rediscover and collide with those virtual sources. This transport reuses the neutral components without changing that engine collision rule or adding another writer.
