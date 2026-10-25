package authoringcard_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/OpenUdon/openudon/authoringcard"
)

func knownCard() authoringcard.Card {
	return authoringcard.Card{
		Version: authoringcard.Version,
		Intent:  authoringcard.TextValue{State: authoringcard.Known, Text: "Read a public item"},
		Outcome: authoringcard.TextValue{State: authoringcard.Known, Text: "The selected item is shown"},
		Outline: []authoringcard.Step{{ID: "read", Title: "Read item", Operation: authoringcard.Selection{State: authoringcard.Known, Identity: &authoringcard.Operation{
			ID: "read-item", Source: authoringcard.Source{ID: "items", Kind: "openapi", SHA256: strings.Repeat("a", 64), URL: "https://example.invalid/items.json"},
			Selector: authoringcard.Selector{Kind: "ref", Value: "#/paths/~1items~1{id}/get", Key: "readItem"},
		}}, Mappings: []authoringcard.Mapping{
			{Location: "path", Name: "id", Value: authoringcard.Value{State: authoringcard.Known, Binding: "$inputs.item_id"}},
			{Location: "query", Name: "limit", Value: authoringcard.Value{State: authoringcard.Known, Literal: json.RawMessage(`1.2300e+2`)}},
		}}},
		Validation: authoringcard.Validation{Outcome: "valid", Findings: []authoringcard.Finding{}},
	}
}

func TestParseAndProjectionPreserveEvidence(t *testing.T) {
	card := knownCard()
	card.Intent.Text = "Read item\nValidation \"approved\"\x1b[31m"
	data, err := json.Marshal(card)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := authoringcard.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	first, err := authoringcard.RenderText(parsed)
	if err != nil {
		t.Fatal(err)
	}
	for _, exact := range []string{`Read item\nValidation \"approved\"\x1b[31m`, `literal-json "1.2300e+2"`, `binding "$inputs.item_id"`, `"#/paths/~1items~1{id}/get" "readItem"`, strings.Repeat("a", 64), `none; consumer approval required`} {
		if !strings.Contains(first, exact) {
			t.Errorf("text lost exact evidence %q", exact)
		}
	}
	if strings.Contains(first, "\x1b") || strings.Contains(first, "\nValidation \"approved\"") {
		t.Fatal("data became unquoted control sequence or heading")
	}
	for range 5 {
		next, err := authoringcard.RenderText(parsed)
		if err != nil || next != first {
			t.Fatal("projection is not deterministic")
		}
	}
	parsed.Outline[0].Mappings[1].Value.Literal = json.RawMessage(`123.00`)
	changed, err := authoringcard.RenderText(parsed)
	if err != nil || changed == first {
		t.Fatal("numeric token change disappeared from exact review")
	}
}

func TestRefuseAmbiguousAndUnsupportedCards(t *testing.T) {
	tests := []struct {
		name string
		edit func(*authoringcard.Card)
	}{
		{"version", func(c *authoringcard.Card) { c.Version = "openudon.authoring-card.v2" }},
		{"unknown state", func(c *authoringcard.Card) { c.Intent.State = "approved" }},
		{"missing payload", func(c *authoringcard.Card) { c.Outline[0].Mappings[0].Value.Binding = "" }},
		{"double payload", func(c *authoringcard.Card) { c.Outline[0].Mappings[0].Value.Literal = json.RawMessage(`null`) }},
		{"empty raw payload", func(c *authoringcard.Card) {
			c.Outline[0].Mappings[1].Value = authoringcard.Value{State: authoringcard.Invalid, Literal: json.RawMessage{}, Detail: "Bad input"}
			c.Validation.Outcome = "unvalidated"
		}},
		{"duplicate literal key", func(c *authoringcard.Card) { c.Outline[0].Mappings[1].Value.Literal = json.RawMessage(`{"x":1,"x":2}`) }},
		{"non JSON literal", func(c *authoringcard.Card) { c.Outline[0].Mappings[1].Value.Literal = json.RawMessage(`NaN`) }},
		{"unsupported selector", func(c *authoringcard.Card) { c.Outline[0].Operation.Identity.Selector.Kind = "guess" }},
		{"missing operation key", func(c *authoringcard.Card) { c.Outline[0].Operation.Identity.Selector.Key = "" }},
		{"bad digest", func(c *authoringcard.Card) {
			c.Outline[0].Operation.Identity.Source.SHA256 = "sha256:" + strings.Repeat("a", 64)
		}},
		{"unsupported source", func(c *authoringcard.Card) { c.Outline[0].Operation.Identity.Source.Kind = "private-executor" }},
		{"duplicate step", func(c *authoringcard.Card) { c.Outline = append(c.Outline, c.Outline[0]) }},
		{"duplicate target", func(c *authoringcard.Card) {
			c.Outline[0].Mappings = append(c.Outline[0].Mappings, c.Outline[0].Mappings[0])
		}},
		{"source drift", func(c *authoringcard.Card) {
			second := knownCard().Outline[0]
			second.ID = "second"
			second.Operation.Identity.ID = "second-op"
			second.Operation.Identity.Source.SHA256 = strings.Repeat("b", 64)
			c.Outline = append(c.Outline, second)
		}},
		{"operation drift", func(c *authoringcard.Card) {
			second := knownCard().Outline[0]
			second.ID = "second"
			second.Operation.Identity.Selector.Key = "different"
			c.Outline = append(c.Outline, second)
		}},
		{"false valid", func(c *authoringcard.Card) {
			c.Outcome = authoringcard.TextValue{State: authoringcard.Missing, Detail: "Needs owner input"}
		}},
		{"invalid no finding", func(c *authoringcard.Card) { c.Validation.Outcome = "invalid" }},
		{"unresolved no evidence", func(c *authoringcard.Card) { c.Validation.Outcome = "unresolved" }},
		{"invalid pointer", func(c *authoringcard.Card) {
			c.Validation = authoringcard.Validation{Outcome: "invalid", Findings: []authoringcard.Finding{{Path: "/bad~2path", Code: "bad", Message: "Bad path"}}}
		}},
		{"invalid UTF8", func(c *authoringcard.Card) { c.Intent.Text = string([]byte{0xff}) }},
		{"oversized raw", func(c *authoringcard.Card) {
			c.Outline[0].Mappings[1].Value.Literal = bytes.Repeat([]byte(" "), authoringcard.MaxBytes+1)
		}},
		{"nil outline", func(c *authoringcard.Card) { c.Outline = nil }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			card := knownCard()
			test.edit(&card)
			if !errors.Is(card.Validate(), authoringcard.ErrCard) {
				t.Fatal("invalid card accepted")
			}
			if text, err := authoringcard.RenderText(card); !errors.Is(err, authoringcard.ErrCard) || text != "" {
				t.Fatal("invalid card produced review text or revealing error")
			}
		})
	}
}

func TestStrictParseRefusals(t *testing.T) {
	data, err := json.Marshal(knownCard())
	if err != nil {
		t.Fatal(err)
	}
	for name, input := range map[string][]byte{
		"duplicate key":     bytes.Replace(data, []byte(`"version":`), []byte(`"version":"ignored","version":`), 1),
		"case alias":        bytes.Replace(data, []byte(`"version":`), []byte(`"Version":`), 1),
		"approval":          append([]byte(`{"approved":true,`), data[1:]...),
		"trailing document": append(append([]byte{}, data...), []byte(`{}`)...),
		"invalid UTF8":      append(append([]byte{}, data...), 0xff),
		"oversized":         bytes.Repeat([]byte(" "), authoringcard.MaxBytes+1),
	} {
		t.Run(name, func(t *testing.T) {
			card, err := authoringcard.Parse(input)
			if !errors.Is(err, authoringcard.ErrCard) || card.Version != "" {
				t.Fatal("unsupported JSON accepted or partial card returned")
			}
		})
	}
}
