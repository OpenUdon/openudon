package authoringcard_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/OpenUdon/openudon/authoringcard"
)

func TestRefuseUnpairedUnicodeEscapes(t *testing.T) {
	data, err := json.Marshal(knownCard())
	if err != nil {
		t.Fatal(err)
	}
	for _, escape := range []string{`\ud800`, `\udc00`, `\uD800x`, `\ud800\ud800`, `\udc00\ud800`, `\ud800\u0041`} {
		for _, target := range []string{`"id":"items"`, `"id":"read-item"`, `"key":"readItem"`, `"name":"id"`, `"text":"Read a public item"`} {
			t.Run(escape+target, func(t *testing.T) {
				replacement := append([]byte{}, []byte(target[:len(target)-1])...)
				replacement = append(replacement, []byte(escape+`"`)...)
				input := bytes.Replace(data, []byte(target), replacement, 1)
				card, err := authoringcard.Parse(input)
				if err != authoringcard.ErrCard || card.Version != "" {
					t.Fatalf("unpaired Unicode escape changed exact review data: %v", err)
				}
			})
		}
		for _, literal := range []string{`"` + escape + `"`, `{"` + escape + `":true}`, `{"value":"` + escape + `"}`} {
			t.Run("literal/"+literal, func(t *testing.T) {
				card := knownCard()
				card.Outline[0].Mappings[1].Value.Literal = json.RawMessage(literal)
				if card.Validate() != authoringcard.ErrCard {
					t.Fatal("typed validation accepted a lossy raw literal or key")
				}
				if text, err := authoringcard.RenderText(card); err != authoringcard.ErrCard || text != "" {
					t.Fatal("unpaired raw literal produced review text")
				}
				input, err := json.Marshal(card)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := authoringcard.Parse(input); err != authoringcard.ErrCard {
					t.Fatal("parsing accepted a lossy raw literal or key")
				}
			})
		}
	}
}

func TestPreserveValidUnicodeAndLiteralBackslashes(t *testing.T) {
	data, err := json.Marshal(knownCard())
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		raw     string
		decoded string
	}{
		{`items\ud83d\ude80`, "items🚀"},
		{`items\uD800\uDC00`, "items𐀀"},
		{`items\udbff\udfff`, "items\U0010ffff"},
		{`items�`, "items�"},
		{`items\uFFFD`, "items�"},
		{`items\\ud800`, `items\ud800`},
		{`items\\udc00`, `items\udc00`},
		{`items\\\ud83d\ude80`, "items\\🚀"},
	} {
		t.Run(test.raw, func(t *testing.T) {
			input := bytes.Replace(data, []byte(`"id":"items"`), []byte(`"id":"`+test.raw+`"`), 1)
			card, err := authoringcard.Parse(input)
			if err != nil || card.Outline[0].Operation.Identity.Source.ID != test.decoded {
				t.Fatalf("valid Unicode identity not preserved: %v", err)
			}
			text, err := authoringcard.RenderText(card)
			if err != nil || !strings.Contains(text, "source ") {
				t.Fatal("valid Unicode card lost its review")
			}
			card.Outline[0].Mappings[1].Value.Literal = json.RawMessage(`{"` + test.raw + `":"` + test.raw + `"}`)
			if err := card.Validate(); err != nil {
				t.Fatal("valid Unicode raw literal/key rejected", err)
			}
			text, err = authoringcard.RenderText(card)
			if err != nil || !strings.Contains(text, "literal-json ") {
				t.Fatal("valid Unicode literal lost its exact review")
			}
		})
	}
}
