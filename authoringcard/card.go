// Package authoringcard describes bounded, source-bound authoring data and its
// deterministic review projection. Cards are untrusted data, never approval,
// source admission, an executable plan, or permission to perform an operation.
package authoringcard

import "encoding/json"

// Version is independent of package, approval and historical step-authoring wires.
const Version = "openudon.authoring-card.v1"

// MaxBytes bounds one encoded card, including all literal values.
const MaxBytes = 256 << 10

// State distinguishes usable authoring data from an absent, rejected or
// unresolved value. Known describes supplied data, not independent source proof.
type State string

const (
	Known      State = "known"
	Missing    State = "missing"
	Invalid    State = "invalid"
	Unresolved State = "unresolved"
)

// Card preserves the consumer's outline order. Empty outlines are legal drafts.
// Intent/outcome, mappings and validation remain visible in the exact review.
type Card struct {
	Version    string     `json:"version"`
	Intent     TextValue  `json:"intent"`
	Outcome    TextValue  `json:"outcome"`
	Outline    []Step     `json:"outline"`
	Validation Validation `json:"validation"`
}

// TextValue retains invalid text for review. Missing/unresolved text carries
// detail instead; no placeholder is silently promoted to known text.
type TextValue struct {
	State  State  `json:"state"`
	Text   string `json:"text,omitempty"`
	Detail string `json:"detail,omitempty"`
}

// Step is a consumer-owned outline entry, not a scheduler or UWS constructor.
type Step struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Operation Selection `json:"operation"`
	Mappings  []Mapping `json:"mappings"`
}

// Selection permits an exact known operation or explicit unfinished selection.
// Ambiguous/unsupported selections must remain invalid or unresolved with detail.
type Selection struct {
	State    State      `json:"state"`
	Identity *Operation `json:"identity,omitempty"`
	Detail   string     `json:"detail,omitempty"`
}

// Operation binds the local workflow operation ID to its source-native identity.
// This matches public UWS shape-table source/selector spelling, without importing
// UWS or imposing any product's source-admission or resource-role policy.
type Operation struct {
	ID       string   `json:"id"`
	Source   Source   `json:"source"`
	Selector Selector `json:"selector"`
}

// Source identifies exact bytes. SHA256 is lowercase, unprefixed hexadecimal.
// URL is optional provenance; no card helper fetches it.
type Source struct {
	ID     string `json:"id"`
	Kind   string `json:"kind"`
	SHA256 string `json:"sha256"`
	URL    string `json:"url,omitempty"`
}

// Selector retains both the source ID/reference and the stable operation key.
type Selector struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
	Key   string `json:"key"`
}

// Mapping identifies one input field in the selected source operation.
type Mapping struct {
	Location string `json:"location"`
	Name     string `json:"name"`
	Value    Value  `json:"value"`
}

// Value contains at most one literal or opaque source-backed binding expression.
// A literal null differs from an absent literal. Invalid values may retain their
// original payload; unresolved bindings retain their expression and detail.
// Helpers neither evaluate expressions nor resolve credentials or placeholders.
type Value struct {
	State   State           `json:"state"`
	Literal json.RawMessage `json:"literal,omitempty"`
	Binding string          `json:"binding,omitempty"`
	Detail  string          `json:"detail,omitempty"`
}

// Validation records a consumer-supplied authoring assessment. A valid outcome
// requires known values and no findings, but is still no approval or source proof.
type Validation struct {
	Outcome  string    `json:"outcome"`
	Findings []Finding `json:"findings"`
}

// Finding uses a JSON Pointer into this card. Code/message are public review
// data; consumers must keep credentials and private values outside cards.
type Finding struct {
	Path    string `json:"path"`
	Code    string `json:"code"`
	Message string `json:"message"`
}
