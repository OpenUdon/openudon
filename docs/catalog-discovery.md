# Catalog discovery and source provisioning

OpenUdon delegates catalog metadata, indexing, ranking and scoped discovery to
the published APItools M81/M80 implementation. `step discover` is read-only;
it grants no source, package, browser or execution approval.

## Installation and request

Prepare registrations and the offline index explicitly with APItools:

```sh
apitools catalog index --root /prepared/catalog \
  --registry cache.sqlite --index operations.v1.json
openudon step discover --catalog-root /prepared/catalog \
  --catalog-registry cache.sqlite --catalog-index operations.v1.json \
  --request /private/discover.json
```

The request and output retain APItools' native
`apitools.catalog-discovery/v1` contract, without a copied ranking engine:

```json
{
  "schema_version": "apitools.catalog-discovery/v1",
  "contract": {"purpose": "list notes records", "effect": "read"},
  "provider_keys": ["Example Notes Shared"],
  "remote_lookup": false
}
```

This provider is a synthetic fixture, not a configured real service. Provider
IDs, display names and aliases are exact individual keys; multiword keys are
never split. An omitted/null provider list searches the configured catalog
scope. An explicit empty array remains empty scope and incomplete evidence.

Root, registry, index, catalog metadata and remote installation capability are
CLI configuration, never request fields. Registry/index paths are confined
relative paths under the explicit root. `--catalog-metadata FILE` selects an
explicit bounded installation catalog matching the index, otherwise APItools'
built-in catalog is used. APItools owns all root/index/registration validation.
No directory, registry or index is created, rebuilt, migrated or refreshed by
discovery; no sibling path, working directory or user home is inferred.

## Reports and consumer actions

| Native outcome | Meaning | Consumer action |
| --- | --- | --- |
| `match` | One operation qualifies in the examined scope; gaps remain visible. | Present evidence for intent/source confirmation. |
| `ambiguous` | Multiple operations qualify, even if only one is displayed. | Ask the user to choose; never pick the top rank silently. |
| `no_qualifying_api` | Complete checked scope rules out qualifying operations under the stated constraints. | May offer scoped browser or pending fallback. |
| `insufficient_evidence` | Missing root/index/artifact, weak purpose, stale metadata, limits or unresolved comparison. | Ask for intent/configuration/documents or an explicit route choice. |
| `blocked` | Invalid request/configuration, identity, unsafe path or corrupt index. | Explain refusal; no automatic browser fallback. |

Reports retain coverage, exclusions, native selectors and artifact/catalog
digests, source authority, license/redistribution unknowns, ties and limits.
Unknown evidence is never inferred into permission. A positive result may
coexist with incomplete scope; it does not prove global uniqueness. Only the
native scoped `no_qualifying_api` outcome can support automatic fallback.
Discovery itself never routes, confirms sources or performs API calls.

Output is one bounded JSON report on stdout. Valid match, ambiguity, scoped
no-match and incomplete reports exit 0; native blockers/cancellation exit 4;
malformed input/arguments exit 2; output failure exits 1. Errors omit raw
untrusted arguments and installation paths. Requests must be bounded UTF-8,
single-value JSON without duplicate/unknown fields; native APItools validation
still owns contract versions and semantics. Reports contain source evidence,
not hidden model reasoning.

## Optional remote lookup

Default lookup is offline. Both `--enable-remote` and request
`"remote_lookup": true` are required to use APItools' bounded public-catalog
lookup. This retains native eight-second/three-document/20-MiB limits and
sanitized digest/final-URL provenance. It writes no local index/registration,
does not execute a service operation and cannot establish global absence.
Remote results remain ephemeral until separately selected provisioning.
No custom request-controlled host, endpoint or transport is accepted.

## Source-backed conformance

`cmd/openudon/testdata/catalog-root` retains the exact published producer's
synthetic catalog, registrations and OpenAPI bytes; its provenance file binds
their full source revision and hashes. Tests prepare real native indexes from
those sources and compare complete CLI reports with the native APItools call,
covering all five outcomes, provider constraints, relocation, missing indexes,
unknown licenses and refusal without widening remote authority. Default checks
need no model, credential, browser or external service.

## Confirmed source provisioning

After presenting the exact operation and source proposal to the user, a caller
may submit its confirmed selection to the existing source command:

```sh
openudon step source add --catalog --example /workflow/package \
  --catalog-root /prepared/catalog --catalog-registry cache.sqlite \
  --catalog-index operations.v1.json --request /private/source-selection.json
```

This additive request uses `openudon.step-source-catalog.v1`; its closed request
schema is [published here](schemas/openudon.step-source-catalog.v1.schema.json).
The envelope contains `kind: "request"`, `command: "step.source.add"`,
`confirmed: true`, the exact current `manifest_revision`, and `sources`.
Each source has a package-local `source_id` and the **unchanged native
`CatalogArtifactReference` values** from the selected operation's `references`.
References bind catalog SHA, provider/spec/artifact IDs, source kind, raw SHA,
byte count and native selector. Preserve every selected provider link. One
source's references must identify the same exact raw artifact and operation.
Catalog/request hashes are lowercase untagged SHA-256; package manifest
revisions retain the existing `sha256:` prefix. A missing manifest is
`{"state":"missing"}`; an existing manifest requires its exact current SHA.
Installation paths and remote capability are not request fields.

Discovery scores and a `match` do not constitute confirmation. The caller owns
the exact user decision represented by `confirmed`; this command is an
artifact provisioning boundary, not an authentication or authorization service.
Ephemeral remote candidates without registered exportable native references
require separate explicit acquisition/registration before this command.

OpenUdon asks APItools to export only selected artifacts into disposable private
staging, checks their raw identity and native selector, then reuses its one
atomic source writer. Raw API bytes, the optimistic-revision-bound
`expected/api-source-manifest.json`, selected advisory overlay bytes and
`expected/catalog-sources/<request-sha>/provenance.json` commit together.
The request SHA binds canonical typed JSON, rather than whitespace in the input
file. APItools validates catalog/registration/artifact identity and applies
provider and selected-spec overlay scope. OpenUdon checks the final package and
catalog roots are disjoint. Likely concrete credentials in selected bytes or
provenance are refused; bytes are never silently redacted and relabeled.

Provenance uses `openudon.catalog-source-provenance.v1`, records the canonical
request SHA and catalog SHA, all selected native references with package paths,
registered kinds and sanitized source URLs, and applicable native security
overlay metadata with package-relative paths and digests. Overlays remain
advisory evidence: they do not alter specifications, credential bindings,
runtime behavior, approval gates or license permissions. The source manifest
and package digest continue to use their established contracts.

Catalog mode returns a `openudon.step-source-catalog.v1` result with the existing
source-add status/diagnostic/result fields and an optional
`result.provenance_path`. Local source-add requests and their v1 output remain
unchanged. Completed exits 0, invalid input 2, stale manifest/source collisions
3, unsafe/unavailable/changed catalog identity or cancellation 4, and writer or
output failure 1. Rejection or validation/conflict failures publish no package
files. A writer's `write_outcome: "indeterminate"`, process interruption or lost
result requires inspection of package paths; never automatically replay it.

Bounds are 256 KiB requests/results, 1 MiB provenance, sixteen sources, sixty-four references in
total, 8 MiB per source, sixty-four advisory overlays of at most 2 MiB each,
and 32 MiB combined selected raw sources and overlays. Staging is private and
removed on normal return. This operation performs no network request, browser
action, workflow run or implicit catalog index update.
