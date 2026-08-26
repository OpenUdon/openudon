package browsercandidate

import (
	"bytes"
	"testing"
)

func TestCanonicalAuthenticatedProfileJSONPreservesProducerContextOrder(t *testing.T) {
	producer := []byte(`{"contexts":{"popup_1":{"kind":"popup","parent":"main","origin":"https://app.example.test"}},"profile":"uws.browser.1.6"}`)
	canonical, err := canonicalAuthenticatedProfileJSON(producer)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(canonical, producer) {
		t.Fatalf("producer context encoding was changed: %s", canonical)
	}
	reordered := []byte(`{"contexts":{"popup_1":{"kind":"popup","origin":"https://app.example.test","parent":"main"}},"profile":"uws.browser.1.6"}`)
	canonical, err = canonicalAuthenticatedProfileJSON(reordered)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(canonical, reordered) || !bytes.Equal(canonical, producer) {
		t.Fatalf("alternate context ordering was accepted: %s", canonical)
	}
}
