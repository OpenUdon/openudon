package sourcecatalog

import (
	"reflect"
	"testing"
)

func TestCanonicalSourceDirectories(t *testing.T) {
	wantAPI := []string{"openapi", "google-discovery", "discovery", "aws-smithy", "asyncapi", "graphql", "openrpc", "grpc-protobuf", "odata"}
	wantBrowser := []string{"browser-profiles", "browser-authentication", "browser-registration", "capability-bundles"}
	if got := API(); !reflect.DeepEqual(got, wantAPI) {
		t.Fatalf("API directories = %#v, want %#v", got, wantAPI)
	}
	if got := Browser(); !reflect.DeepEqual(got, wantBrowser) {
		t.Fatalf("browser directories = %#v, want %#v", got, wantBrowser)
	}
	wantAll := append(append([]string(nil), wantAPI...), wantBrowser...)
	got := All()
	if !reflect.DeepEqual(got, wantAll) {
		t.Fatalf("all directories = %#v, want %#v", got, wantAll)
	}
	got[0] = "mutated"
	if API()[0] != "openapi" {
		t.Fatal("source directory caller mutated the canonical list")
	}
}

func TestAPIPathClassificationUsesCanonicalFamilies(t *testing.T) {
	for _, value := range []string{"graphql/schema.graphql", "/workspace/openrpc/math.json", `C:\\workspace\\grpc-protobuf\\service.proto`} {
		if !ContainsAPIPath(value) {
			t.Errorf("ContainsAPIPath(%q) = false", value)
		}
	}
	if !IsAPIPath("odata/service.xml") || IsAPIPath("workspace/odata/service.xml") {
		t.Fatal("IsAPIPath did not require a canonical leading directory")
	}
}
