package authoringcard

import (
	"encoding/json"
	"errors"
	"strconv"
	"sync"
	"unicode/utf8"

	"github.com/OpenUdon/openudon/docs/schemas"
	"github.com/OpenUdon/openudon/wire"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

// ErrCard is deliberately value-free. It does not reveal rejected authoring data.
var ErrCard = errors.New("invalid or unsupported authoring card")

type closedLoader struct{}

func (closedLoader) Load(string) (any, error) { return nil, ErrCard }

var cardSchema = sync.OnceValue(func() *jsonschema.Schema {
	data, err := schemas.AuthoringCardResources.ReadFile("openudon.authoring-card.v1.schema.json")
	if err != nil {
		return nil
	}
	var resource any
	if wire.DecodeStrictNumbers(data, &resource) != nil {
		return nil
	}
	compiler := jsonschema.NewCompiler()
	compiler.UseLoader(closedLoader{})
	const location = "https://openudon.dev/schemas/openudon.authoring-card.v1.schema.json"
	if compiler.AddResource(location, resource) != nil {
		return nil
	}
	compiled, err := compiler.Compile(location)
	if err != nil {
		return nil
	}
	return compiled
})

// Parse decodes one bounded UTF-8 card, preserving literal numeric tokens and
// rejecting unknown/noncanonical fields, duplicate keys, unpaired Unicode
// escapes and unsupported data before any lossy string decoding.
func Parse(data []byte) (Card, error) {
	var card Card
	if len(data) > MaxBytes || !utf8.Valid(data) || validateJSON(data) != nil || wire.DecodeStrictNumbers(data, &card) != nil || card.Validate() != nil {
		return Card{}, ErrCard
	}
	return card, nil
}

func validateJSON(data []byte) error {
	if !validUnicodeEscapes(data) {
		return ErrCard
	}
	var value any
	if wire.DecodeStrictNumbers(data, &value) != nil {
		return ErrCard
	}
	schema := cardSchema()
	if schema == nil || schema.Validate(value) != nil {
		return ErrCard
	}
	return nil
}

// Go's JSON decoder replaces unpaired UTF-16 escapes. Check string tokens before
// decoding so identities, values and object keys cannot silently change. This
// scan only checks Unicode escapes; the JSON decoder still owns JSON grammar.
func validUnicodeEscapes(data []byte) bool {
	inString := false
	for i := 0; i < len(data); i++ {
		if data[i] == '"' {
			inString = !inString
			continue
		}
		if !inString || data[i] != '\\' {
			continue
		}
		i++
		if i >= len(data) {
			return false
		}
		if data[i] != 'u' {
			continue
		}
		if i+4 >= len(data) {
			return false
		}
		first, err := strconv.ParseUint(string(data[i+1:i+5]), 16, 16)
		if err != nil || first >= 0xdc00 && first <= 0xdfff {
			return false
		}
		i += 4
		if first < 0xd800 || first > 0xdbff {
			continue
		}
		if i+6 >= len(data) || data[i+1] != '\\' || data[i+2] != 'u' {
			return false
		}
		second, err := strconv.ParseUint(string(data[i+3:i+7]), 16, 16)
		if err != nil || second < 0xdc00 || second > 0xdfff {
			return false
		}
		i += 6
	}
	return true
}

// Validate checks structure, boundedness and internal consistency only. It does
// not reproduce source bytes, validate API schemas, evaluate bindings or approve
// a draft. Invalid/unresolved authoring values are legal explicit review data.
func (c Card) Validate() error {
	if !c.bounded() {
		return ErrCard
	}
	data, err := json.Marshal(c)
	if err != nil || len(data) > MaxBytes || validateJSON(data) != nil {
		return ErrCard
	}
	unfinished := c.Intent.State != Known || c.Outcome.State != Known || len(c.Outline) == 0
	steps := map[string]bool{}
	sources := map[string]Source{}
	operations := map[string]Operation{}
	for _, step := range c.Outline {
		if steps[step.ID] {
			return ErrCard
		}
		steps[step.ID] = true
		unfinished = unfinished || step.Operation.State != Known
		if op := step.Operation.Identity; op != nil {
			if previous, exists := sources[op.Source.ID]; exists && previous != op.Source {
				return ErrCard
			}
			if previous, exists := operations[op.ID]; exists && previous != *op {
				return ErrCard
			}
			sources[op.Source.ID] = op.Source
			operations[op.ID] = *op
		}
		targets := map[[2]string]bool{}
		for _, mapping := range step.Mappings {
			target := [2]string{mapping.Location, mapping.Name}
			if targets[target] {
				return ErrCard
			}
			targets[target] = true
			unfinished = unfinished || mapping.Value.State != Known
		}
	}
	switch c.Validation.Outcome {
	case "valid":
		if unfinished || len(c.Validation.Findings) != 0 {
			return ErrCard
		}
	case "invalid":
		if len(c.Validation.Findings) == 0 {
			return ErrCard
		}
	case "unresolved":
		if !unfinished && len(c.Validation.Findings) == 0 {
			return ErrCard
		}
	}
	return nil
}

// Bound before marshaling, including each raw literal, so typed callers cannot
// allocate an unbounded JSON projection. Schema validation owns tighter bounds.
func (c Card) bounded() bool {
	if len(c.Outline) > 100 || len(c.Validation.Findings) > 128 {
		return false
	}
	size := 0
	texts := func(values ...string) bool {
		for _, value := range values {
			if len(value) > MaxBytes-size || !utf8.ValidString(value) {
				return false
			}
			size += len(value)
		}
		return true
	}
	if !texts(c.Version, string(c.Intent.State), c.Intent.Text, c.Intent.Detail, string(c.Outcome.State), c.Outcome.Text, c.Outcome.Detail, c.Validation.Outcome) {
		return false
	}
	for _, step := range c.Outline {
		if len(step.Mappings) > 128 || !texts(step.ID, step.Title, string(step.Operation.State), step.Operation.Detail) {
			return false
		}
		if op := step.Operation.Identity; op != nil && !texts(op.ID, op.Source.ID, op.Source.Kind, op.Source.SHA256, op.Source.URL, op.Selector.Kind, op.Selector.Value, op.Selector.Key) {
			return false
		}
		for _, mapping := range step.Mappings {
			v := mapping.Value
			if !texts(mapping.Location, mapping.Name, string(v.State), v.Binding, v.Detail) || v.Literal != nil && len(v.Literal) == 0 || len(v.Literal) > MaxBytes-size || !utf8.Valid(v.Literal) {
				return false
			}
			size += len(v.Literal)
		}
	}
	for _, finding := range c.Validation.Findings {
		if !texts(finding.Path, finding.Code, finding.Message) {
			return false
		}
	}
	return true
}
