package wire_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/OpenUdon/openudon/wire"
)

func TestBoundedClosedJSON(t *testing.T) {
	var value struct {
		Value any `json:"value"`
	}
	if err := wire.DecodeStrictNumbers([]byte(`{"value":9007199254740993}`), &value); err != nil {
		t.Fatal(err)
	}
	if value.Value != json.Number("9007199254740993") {
		t.Fatal("large number rounded")
	}
	for _, data := range []string{
		`{"value":1,"value":2}`, `{"value":1,"Value":2}`, `{"value":1,"\u0076alue":2}`,
		`{"unknown":1}`, `{"value":1} {}`, strings.Repeat("[", 65) + "0" + strings.Repeat("]", 65),
		"[" + strings.Repeat("0,", wire.MaxNodes) + "0]", strings.Repeat(" ", wire.MaxBytes+1),
	} {
		if wire.DecodeStrictNumbers([]byte(data), &value) == nil {
			t.Fatal("accepted ambiguous or unbounded JSON")
		}
	}
}

func TestTypedUnicodeAliasesAndFreeFormKeys(t *testing.T) {
	type record struct {
		Scope string         `json:"scope"`
		Key   string         `json:"k"`
		Data  map[string]any `json:"data"`
	}
	type envelope struct {
		Rows []*record         `json:"rows"`
		ByID map[string]record `json:"byId"`
	}
	for _, data := range []string{
		`{"rows":[{"scope":"first","ſcope":"last"}]}`,
		`{"byId":{"item":{"k":"first","K":"last"}}}`,
		`{"rows":[{"data":{"id":1,"id":2}}]}`,
		`{"rows":[{"data":{"id":1,"\u0069d":2}}]}`,
	} {
		var out envelope
		if wire.DecodeStrictNumbers([]byte(data), &out) == nil {
			t.Fatalf("accepted duplicate destination: %s", data)
		}
	}
	var out envelope
	data := []byte(`{"rows":[{"ſcope":"review","K":"key","data":{"id":9007199254740993,"ID":9007199254740995,"scope":1,"ſcope":2}}],"byId":{"id":{"scope":"one"},"ID":{"scope":"two"}}}`)
	if err := wire.DecodeStrictNumbers(data, &out); err != nil {
		t.Fatal(err)
	}
	if out.Rows[0].Scope != "review" || out.Rows[0].Key != "key" || out.Rows[0].Data["id"] != json.Number("9007199254740993") || out.Rows[0].Data["ID"] != json.Number("9007199254740995") || len(out.ByID) != 2 {
		t.Fatal("changed Go field matching, numeric text or map keys")
	}
}

func TestExactFieldNamesAndEmbeddedDominanceMatchJSON(t *testing.T) {
	type embedded struct {
		Scope string `json:"scope"`
	}
	type record struct {
		embedded
		Upper string `json:"ID"`
		Mixed string `json:"Id"`
	}
	var out record
	if err := wire.DecodeStrict([]byte(`{"ID":"upper","Id":"mixed","ſcope":"embedded"}`), &out); err != nil {
		t.Fatal(err)
	}
	if out.Upper != "upper" || out.Mixed != "mixed" || out.Scope != "embedded" {
		t.Fatal("exact field matching changed")
	}
	for _, data := range []string{`{"ID":"upper","id":"alias"}`, `{"scope":"first","ſcope":"alias"}`} {
		if wire.DecodeStrict([]byte(data), &out) == nil {
			t.Fatalf("accepted %s", data)
		}
	}
}

func TestPopulatedInterfacesRetainTheirActualJSONDestination(t *testing.T) {
	type record struct {
		Scope string `json:"scope"`
	}
	type envelope struct {
		Value any `json:"value"`
	}
	for _, data := range []string{`{"scope":"first","ſcope":"last"}`, `{"scope":"first","Scope":"last"}`} {
		var root any = &record{}
		if wire.DecodeStrict([]byte(data), &root) == nil {
			t.Fatal("root interface bypassed typed alias check")
		}
		nested := envelope{Value: &record{}}
		if wire.DecodeStrict([]byte(`{"value":`+data+`}`), &nested) == nil {
			t.Fatal("nested interface bypassed typed alias check")
		}
		rows := []any{&record{}}
		if wire.DecodeStrict([]byte(`[`+data+`]`), &rows) == nil {
			t.Fatal("reused slice interface bypassed typed alias check")
		}
	}
	var root any = &record{}
	if err := wire.DecodeStrict([]byte(`{"ſcope":"review"}`), &root); err != nil || root.(*record).Scope != "review" {
		t.Fatal("Go interface destination changed", err)
	}
	// Non-pointer/typed-nil interface contents and fresh map elements are
	// replaced with free-form JSON values by encoding/json, retaining key case.
	for _, seed := range []any{record{}, (*record)(nil)} {
		root := seed
		if err := wire.DecodeStrictNumbers([]byte(`{"id":9007199254740993,"ID":9007199254740995}`), &root); err != nil {
			t.Fatal(err)
		}
		if len(root.(map[string]any)) != 2 {
			t.Fatal("free-form interface keys lost")
		}
	}
	values := map[string]any{"record": &record{}}
	if err := wire.DecodeStrict([]byte(`{"record":{"scope":1,"ſcope":2}}`), &values); err != nil {
		t.Fatal("existing map values are not reused", err)
	}
	if len(values["record"].(map[string]any)) != 2 {
		t.Fatal("map data folded")
	}
	var self any
	self = &self
	if err := wire.DecodeStrict([]byte(`{"id":1,"ID":2}`), &self); err != nil {
		t.Fatal("self-containing interface escape", err)
	}
	if len(self.(map[string]any)) != 2 {
		t.Fatal("self-containing interface changed")
	}
}
