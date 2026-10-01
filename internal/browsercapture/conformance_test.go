package browsercapture

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/OpenUdon/openudon/internal/evidencefile"
)

func TestPublishedCaptureWireConformance(t *testing.T) {
	for _, folder := range []string{"valid", "invalid"} {
		paths, err := filepath.Glob("../../docs/examples/browser-capture/v1/" + folder + "/*.json")
		if err != nil || len(paths) == 0 {
			t.Fatalf("missing %s fixtures: %v", folder, err)
		}
		for _, path := range paths {
			t.Run(folder+"/"+filepath.Base(path), func(t *testing.T) {
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				var kind struct {
					Type string `json:"type"`
				}
				if err = json.Unmarshal(data, &kind); err != nil {
					t.Fatal(err)
				}
				switch kind.Type {
				case "propose":
					_, err = DecodeProposal(data)
				case "decide":
					_, err = DecodeDecision(data)
				case "cancel":
					_, err = DecodeCancellation(data)
				default:
					_, err = DecodeEvent(data)
				}
				if (folder == "valid") != (err == nil) {
					t.Fatalf("fixture disposition mismatch: %v", err)
				}
			})
		}
	}
}

func TestPublishedReviewCardHasExactCommandDigest(t *testing.T) {
	data, err := os.ReadFile("../../docs/examples/browser-capture/v1/valid/authenticated-review-card.json")
	if err != nil {
		t.Fatal(err)
	}
	card, err := DecodeEvent(data)
	if err != nil {
		t.Fatal(err)
	}
	command, err := json.Marshal(card.Action.Command)
	if err != nil {
		t.Fatal(err)
	}
	if card.Action.CommandSHA256 != evidencefile.SHA256(command) {
		t.Fatal("published review card digest is not the actual command digest")
	}
}

func TestRegistrationApprovalUsesSameBoundedGate(t *testing.T) {
	_, _, now := testGate(t)
	g, err := NewGate(Registration, now.Add(time.Minute), func(view View, command Command) error {
		if view.State != "observing" || command.Registration == nil || command.Registration.Type != "observe" {
			return errors.New("not offered")
		}
		return nil
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	event, err := g.Observe("state", View{State: "observing"}, now)
	if err != nil {
		t.Fatal(err)
	}
	card, err := g.Propose(Proposal{Binding: event.Binding, Type: "propose", Command: Command{Registration: &RegistrationCommand{Type: "observe"}}}, now)
	if err != nil {
		t.Fatal(err)
	}
	command, err := g.Decide(decisionFor(card, true), now)
	if err != nil || command == nil || command.Registration.Type != "observe" {
		t.Fatalf("registration decision: %v %v", command, err)
	}
}
