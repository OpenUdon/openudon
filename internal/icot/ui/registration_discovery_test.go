package ui

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/OpenUdon/openudon/internal/registrationdiscovery"
)

func discoveryApplication(t *testing.T) Application {
	t.Helper()
	privateRoot := t.TempDir()
	if err := os.Chmod(privateRoot, 0700); err != nil {
		t.Fatal(err)
	}
	fake := &fakeEngine{}
	handler, err := NewHandler(HandlerConfig{Engine: fake, Snapshot: fake.snapshot, ExampleDir: t.TempDir(), PrivateRoot: privateRoot, Token: testToken, AccessCode: testAccessCode, Authority: testAuthority, BrowserTransactions: newFakeBrowserTransactions()})
	if err != nil {
		t.Fatal(err)
	}
	return Application{handler.(*Server)}
}

func readDiscovery(t *testing.T, reply applicationResult) registrationdiscovery.Inventory {
	t.Helper()
	var inventory registrationdiscovery.Inventory
	if reply.Failure != nil || json.Unmarshal(reply.Discovery, &inventory) != nil {
		t.Fatalf("inventory reply rejected: %+v", reply.Failure)
	}
	return inventory
}

func TestRegistrationDiscoveryAdaptersAndSnapshotIsolation(t *testing.T) {
	app := discoveryApplication(t)
	s := app.server
	before := currentResponse(t, s)
	if response := doRequest(s, "GET", "/api/v4/registration-discovery", "", "", false); response.Code != http.StatusUnauthorized {
		t.Fatal("unauthenticated inventory")
	}
	initial := readDiscovery(t, app.RegistrationDiscovery(context.Background(), nil))
	request := registrationDiscoveryRequest{Change: registrationdiscovery.Change{Revision: initial.Revision, Action: "add", ID: "publisher", RegistrationType: "publisher", URL: "https://app.example.test/private-route?action=register"}}
	response := doRequest(s, "POST", "/api/v4/registration-discovery", mustJSON(t, request), "application/json", true)
	var added registrationdiscovery.Inventory
	if response.Code != http.StatusOK || json.Unmarshal(response.Body.Bytes(), &added) != nil {
		t.Fatalf("HTTP add: %s", response.Body.String())
	}
	stale := app.RegistrationDiscovery(context.Background(), &request)
	if stale.Failure == nil || stale.Failure.Code != "discovery_stale" {
		t.Fatal("cross-adapter stale write accepted")
	}
	// Source claims cannot be supplied by an operator through either decoder.
	for _, extra := range []string{`,"source":"browser_observed"`, `,"revision":"duplicate"`} {
		body := strings.TrimSuffix(mustJSON(t, request), "}") + extra + "}"
		response = doRequest(s, "POST", "/api/v4/registration-discovery", body, "application/json", true)
		if response.Code == http.StatusOK {
			t.Fatal("unknown or duplicate field accepted")
		}
	}
	c := applicationClient(t, app)
	frame := registrationControlFrame{Version: ApplicationControlVersion, Operation: "registration.discovery.snapshot"}
	if err := c.encoder.Encode(frame); err != nil {
		t.Fatal(err)
	}
	var state ApplicationControlState
	if err := c.decoder.Decode(&state); err != nil {
		t.Fatal(err)
	}
	var shared registrationdiscovery.Inventory
	if json.Unmarshal(state.RegistrationDiscovery, &shared) != nil || shared.Revision != added.Revision {
		t.Fatal("control resource differs from HTTP")
	}
	if strings.Contains(string(state.Application), "private-route") {
		t.Fatal("inventory leaked into ordinary snapshot")
	}
	changeRequest, _ := json.Marshal(registrationDiscoveryRequest{Change: registrationdiscovery.Change{Revision: shared.Revision, Action: "select", ID: "publisher"}})
	if err := c.encoder.Encode(registrationControlFrame{Version: ApplicationControlVersion, Operation: "registration.discovery.change", Request: changeRequest}); err != nil {
		t.Fatal(err)
	}
	state = ApplicationControlState{}
	if err := c.decoder.Decode(&state); err != nil || state.Failure != "" || json.Unmarshal(state.RegistrationDiscovery, &shared) != nil || shared.SelectedID != "publisher" {
		t.Fatal("control selection failed")
	}
	response = doRequest(s, "GET", "/api/v4/registration-discovery", "", "", true)
	var fromHTTP registrationdiscovery.Inventory
	if response.Code != http.StatusOK || json.Unmarshal(response.Body.Bytes(), &fromHTTP) != nil || fromHTTP.Revision != shared.Revision {
		t.Fatal("control change was not visible to HTTP")
	}
	if err := c.encoder.Encode(registrationControlFrame{Version: ApplicationControlVersion, Operation: "snapshot"}); err != nil {
		t.Fatal(err)
	}
	state = ApplicationControlState{}
	if err := c.decoder.Decode(&state); err != nil || len(state.RegistrationDiscovery) != 0 {
		t.Fatal("explicit resource carried into later snapshot")
	}
	after := currentResponse(t, s)
	if after.Revision != before.Revision || after.RegistrationRevision != before.RegistrationRevision || s.registrationAttemptConsumed || s.registrationSession != nil || s.registrationCandidate != nil {
		t.Fatal("discovery changed browser or workflow authority")
	}
}

func TestRegistrationDiscoveryPreservesConsumedAttemptAndCanonicalDraft(t *testing.T) {
	app := discoveryApplication(t)
	s := app.server
	canonical, _, _, _, err := buildRegistrationDraft(validRegistrationDraftRequest(), registrationAuthoringStartRequest{Origins: []string{"https://app.example.test"}}, registrationDraftObservation(), time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	s.registrationDraft = bytes.Clone(canonical)
	s.registrationAttemptConsumed = true
	s.registrationContainmentFailed = true
	i := readDiscovery(t, app.RegistrationDiscovery(context.Background(), nil))
	change := func(c registrationdiscovery.Change) {
		c.Revision = i.Revision
		i = readDiscovery(t, app.RegistrationDiscovery(context.Background(), &registrationDiscoveryRequest{Change: c}))
	}
	change(registrationdiscovery.Change{Action: "add", ID: "publisher", RegistrationType: "publisher", URL: "https://app.example.test/publisher"})
	change(registrationdiscovery.Change{Action: "select", ID: "publisher"})
	change(registrationdiscovery.Change{Action: "review"})
	if !s.registrationAttemptConsumed || !s.registrationContainmentFailed || s.registrationSession != nil || !bytes.Equal(s.registrationDraft, canonical) {
		t.Fatal("inventory renewed authority or rewrote draft")
	}
	for _, marker := range []string{`"discovery"`, `"owner_review"`, `"candidates"`, "/publisher"} {
		if bytes.Contains(canonical, []byte(marker)) {
			t.Fatal("inventory metadata in canonical export")
		}
	}
}

func TestRegistrationDiscoveryRequiresCurrentNativeObservation(t *testing.T) {
	app := discoveryApplication(t)
	i := readDiscovery(t, app.RegistrationDiscovery(context.Background(), nil))
	r := registrationDiscoveryRequest{Change: registrationdiscovery.Change{Revision: i.Revision, Action: "add_observed", ID: "observed", RegistrationType: "advertiser"}, RegistrationRevision: "stale"}
	if result := app.RegistrationDiscovery(context.Background(), &r); result.Failure == nil {
		t.Fatal("missing observation accepted")
	}
	obs := registrationDraftObservation()
	app.server.registrationAuthoring = &RegistrationAuthoringState{State: "observing", Observation: &obs}
	app.server.updateRevisionLocked()
	if result := app.RegistrationDiscovery(context.Background(), &r); result.Failure == nil {
		t.Fatal("stale observation accepted")
	}
	r.RegistrationRevision = app.server.registrationRevision
	i = readDiscovery(t, app.RegistrationDiscovery(context.Background(), &r))
	if i.Candidates[0].Source != "browser_observed" || i.Candidates[0].URL != obs.Origin+obs.Path || i.Coverage != "unknown" || i.OwnerReview != "pending" {
		t.Fatal("observation overclaimed review or coverage")
	}
}
