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
