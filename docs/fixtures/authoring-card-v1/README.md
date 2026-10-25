# Authoring card v1 conformance

`manifest.json` binds the new contract source/schema and every case by SHA-256.
The repository commit supplies producer provenance; this manifest supplies file
identity and does not claim remote publication. Consumers pin both. Existing
package, browser, trust and step-authoring fixtures are unchanged.

The 33 JSON cases include nine positive drafts with exact text goldens. Text
goldens were prepared with an independent projection of the documented format,
then compared with the Go implementation; default tests never rewrite fixtures.
They cover both selector kinds, source provenance, ordered mappings, null/false/
zero/empty literals, large integers, numeric lexemes, opaque bindings, unfinished
selection, missing/invalid/unresolved values, validation findings and escaped
multiline/non-ASCII review text.

Each case records two results: `schema_accepted` includes strict duplicate-key
decoding plus the embedded JSON Schema, while `accepted` also includes bounded
identity/consistency validation. Schema-only checking is insufficient: duplicate
steps/targets, source/operation drift and inconsistent validation claims require
the additional checks in `Card.Validate`. Refusals return no partial card or
text. Byte/UTF-8/depth bounds have focused Go regression coverage as well.

The Unicode cases refuse lone high/low surrogate escapes in source IDs, raw
literal values and keys, while preserving valid pairs, literal/escaped U+FFFD
and escaped literal backslashes. These refusals may pass the schema-only check
through a permissive decoder; adapters must check Unicode scalars before any
lossy string decoding. Typed validation/rendering also checks raw literals.
During prepublication preparation, this correction extended P14 v1’s own
corpus and contract hashes without rewriting older public fixtures. Exact e486
is now independently published and ordinarily qualified; see
[publication evidence](../../p14-published-qualification.json).

Kinet may consume the published package and these files. Ramen may mirror the
small types/schema/text behavior and fixture data with the exact producer commit
and file hashes; a mirror imports no OpenUdon Go module. Both adapters must
match JSON refusal and text behavior, retain all admitted operation metadata
and their product-specific review evidence, and keep their existing approval
and execution boundaries. See [adapter guidance](../../authoring-card-v1.md).
