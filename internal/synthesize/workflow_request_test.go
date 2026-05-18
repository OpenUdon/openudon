package synthesize

import (
	"testing"

	"github.com/OpenUdon/apitools"
)

func TestRequestFieldPlacementsExposeBearerAuthorization(t *testing.T) {
	fields, err := requestFieldPlacements(apitools.OperationSummary{
		OperationID: "getCustomer",
		Security: []apitools.SecuritySummary{{
			Name:   "BearerAuth",
			Type:   "http",
			Scheme: "bearer",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	placement, ok := fields["Authorization"]
	if !ok {
		t.Fatalf("Authorization placement missing: %#v", fields)
	}
	if placement.Section != "header" || placement.Name != "Authorization" {
		t.Fatalf("Authorization placement = %#v", placement)
	}
}
