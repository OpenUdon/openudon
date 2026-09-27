//go:build uws_pending_step_c07_2

package stepauthoringcontract

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/OpenUdon/openudon/internal/stepauthoring"
	"github.com/OpenUdon/uws/uws1"
)

func TestSharedContractFieldsMapLosslesslyToUWSPendingStep(t *testing.T) {
	root := filepath.Join("..", "..", "docs", "examples", "step-authoring", "v1")
	requestBytes, err := os.ReadFile(filepath.Join(root, "requests", "step-candidates.json"))
	if err != nil {
		t.Fatal(err)
	}
	var request stepauthoring.CandidatesRequest
	if err := json.Unmarshal(requestBytes, &request); err != nil {
		t.Fatal(err)
	}
	fixtureBytes, err := os.ReadFile(filepath.Join(root, "uws-c07-2-pending-fields-draft.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixture uws1.PendingStep
	if err := json.Unmarshal(fixtureBytes, &fixture); err != nil {
		t.Fatalf("decode pending-step fixture using UWS C07.2: %v", err)
	}
	contract := request.Contract
	if fixture.Purpose != contract.Purpose || fixture.Effect != uws1.OperationEffect(contract.Effect) ||
		!reflect.DeepEqual(fixture.Inputs, contract.Inputs) || !reflect.DeepEqual(fixture.Outputs, contract.Outputs) {
		t.Fatalf("pending-step fields do not losslessly map to the public step contract\nfixture: %#v\ncontract: %#v", fixture, contract)
	}

	roundTrip, err := json.Marshal(fixture)
	if err != nil {
		t.Fatal(err)
	}
	var gotValue, wantValue any
	if err := json.Unmarshal(roundTrip, &gotValue); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(fixtureBytes, &wantValue); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotValue, wantValue) {
		t.Fatalf("UWS pending-step JSON round trip changed shared fields\ngot: %#v\nwant: %#v", gotValue, wantValue)
	}
}
