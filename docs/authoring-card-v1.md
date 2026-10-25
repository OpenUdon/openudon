# Authoring card v1

`authoringcard.Card` and the embedded
[`openudon.authoring-card.v1.schema.json`](schemas/openudon.authoring-card.v1.schema.json)
describe a single authoring draft. The new discriminator is
`openudon.authoring-card.v1`; historical step-authoring v1, package v3, approval,
handoff, browser and history contracts keep their existing versions and bytes.

`Parse([]byte)` returns a fully checked card or a zero card plus the fixed
value-free `ErrCard`. Typed callers use `Card.Validate()`. `RenderText(Card)`
validates first and returns either the complete projection or empty text plus
`ErrCard`. These helpers mutate no input and perform no external action. The
[versioned conformance corpus](fixtures/authoring-card-v1/README.md) separates
schema results from identity/consistency results and supplies exact text goldens
for independent adapters.

One card contains intent, desired outcome, an ordered outline, exact operation
selections, input mappings, and a supplied validation assessment. Reviewing the
whole card supplies the authoring review; there is no approval or confirmation
field. Outline/mapping/finding order is evidence and is preserved. Consumers
choose their business order, resource roles and which stages to show.

Values have explicit `known`, `missing`, `invalid` or `unresolved` states.
Known intent/outcome retain nonempty text. Known selections retain one exact
operation identity. Other selections have detail and no guessed identity.
Known mapped values have exactly one JSON literal or opaque binding expression.
Literal `null`, `false`, zero and empty string remain distinct from missing.
Invalid mappings may retain a literal or binding for review; unresolved mappings
may retain a binding placeholder. Missing values have no payload. Every
non-known state requires detail. Known values have no detail. A binding is
review data, never evaluated or substituted by this package.

An operation preserves its local workflow operation ID plus source ID, kind,
lowercase unprefixed SHA-256, optional provenance URL, and source selector
`kind` (`id` or `ref`), `value` and stable `key`. These mirror public UWS
shape-table identity spelling. The adapter must independently reproduce and
admit the source; an exact-looking identity alone supplies no proof. No helper
reads files, fetches provenance URLs, discovers operations or invokes a provider.

Validation outcomes are `unvalidated`, `valid`, `invalid` and `unresolved`.
`valid` requires a nonempty outline, all known authoring values and no findings.
`invalid` requires at least one finding. `unresolved` requires a non-known value
or finding. Findings identify a card location using a JSON Pointer, a stable
consumer code and a public message. Validation of this data contract does not
establish API field types, executable support or consumer admission.

One UTF-8 card is at most 256 KiB. The schema bounds text and collections;
the helpers also reject duplicate or noncanonical JSON keys, ambiguous duplicate
step IDs or mapping targets, conflicting source identities, unknown versions,
invalid UTF-8 and unsupported shapes. No payload is truncated. Text projection
quotes all strings and preserves every field, literal numeric token and ordered
entry; it supplies no new default or hidden decision. `literal-json` quotes the
raw JSON value bytes, including any internal whitespace, instead of interpreting
them as review markup. Parse treats whitespace outside a JSON value as framing.

All strings and object keys must contain Unicode scalar values. Before decoding,
the contract refuses unpaired UTF-16 surrogate escapes anywhere in a card,
including inside raw literal values and keys; a permissive JSON decoder must
never replace them silently. Valid escaped surrogate pairs, literal or escaped
U+FFFD, and literal backslash-u text remain valid. Typed validation/rendering
apply the same rule to raw literals. Ordinary typed Go strings must be valid
UTF-8; an intentionally supplied U+FFFD is preserved. This is an additional
predecode check beyond JSON Schema and changes no historical wire helper.

Kinet's adapter retains its business sequencing, source admission, exact
rendered approval, receipts, ledger confirmation and worker isolation. Its
`internal/cardfixture` is formatter/frontend evidence only. A card does not
replace those formats. Ramen's independent adapter retains resource identity,
state/graph/diff, admission and its own plan/apply confirmation. Ramen may mirror
these types, schema and versioned JSON/text fixtures with exact producer/file
identities; a data mirror makes no compiled OpenUdon dependency claim. Neither
product imports the other, and no shared UI, planner, transport or private
executor dependency is introduced.

The card's mapping target/value is not a replacement for admitted operation
metadata. For example, Kinet's retained `catalog-1` and `workflow-repair` card
fixtures also review contract/operation types and requiredness, source evidence,
effect/capability reasons and incompleteness. Adapters must retain that evidence
from their independently admitted exact source/shape snapshot and APItools
eligibility/mapping reports, alongside this card in their own review. Ramen
similarly adds lifecycle/resource-role and state/diff evidence. `RenderText` is
the complete projection of the P14 card, not a sufficient product approval
prompt. No adapter may discard existing review evidence merely because the
common data contract has fewer fields.

Adapters must keep credentials and private values outside cards. Review text
contains the complete supplied public authoring values; render it as text with
the consumer's existing escaping. When an adapter subsequently changes a card,
it must repeat its own exact review and approval; old approval is never carried
forward by this package.
