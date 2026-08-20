package catalogpolicy

import (
	"strings"
	"testing"
)

func TestSelectOpenAPIReferenceFailsClosedForNonOpenAPIProvider(t *testing.T) {
	_, provider, err := SelectOpenAPIReference("gmail", "")
	if provider.ID != "gmail" || err == nil || !strings.Contains(err.Error(), "no directly importable OpenAPI") {
		t.Fatalf("selection = %#v, %v", provider, err)
	}
}

func TestSelectOpenAPIReferenceRejectsUnknownProvider(t *testing.T) {
	if _, _, err := SelectOpenAPIReference("missing", ""); err == nil || !strings.Contains(err.Error(), "unknown provider") {
		t.Fatalf("error = %v", err)
	}
}
