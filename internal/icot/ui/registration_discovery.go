package ui

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"

	"github.com/OpenUdon/openudon/internal/registrationdiscovery"
)

type registrationDiscoveryRequest struct {
	registrationdiscovery.Change
	RegistrationRevision string `json:"registration_revision,omitempty"`
}

// RegistrationDiscovery is a separate private resource, explicitly requested
// through either adapter. Ordinary snapshots, transcripts, drafts and package
// resources do not contain its candidate URLs or review history.
func (app Application) RegistrationDiscovery(ctx context.Context, request *registrationDiscoveryRequest) (result applicationResult) {
	s := app.server
	s.mu.Lock()
	defer s.mu.Unlock()
	if ctx.Err() != nil {
		result.fail(http.StatusConflict, "discovery_canceled", "discovery operation canceled", false, "")
		return
	}
	store, err := registrationdiscovery.Open(s.privateRoot, s.exampleDir)
	if err != nil {
		result.fail(http.StatusConflict, "discovery_storage", "discovery requires available owner-only storage outside Git and the example", false, "")
		return
	}
	defer store.Close()
	var inventory registrationdiscovery.Inventory
	if request == nil {
		inventory, err = store.Read()
	} else {
		observedURL := ""
		if request.Action == "add_observed" {
			if request.RegistrationRevision != s.registrationRevision || s.registrationAuthoring == nil || s.registrationAuthoring.Observation == nil {
				result.fail(http.StatusConflict, "discovery_observation", "a current registration observation revision is required", false, "")
				return
			}
			observation := s.registrationAuthoring.Observation
			observedURL = observation.Origin + observation.Path
			u, parseErr := url.Parse(observedURL)
			if parseErr != nil || u.RawQuery != "" || u.Fragment != "" {
				result.fail(http.StatusConflict, "discovery_observation", "a reduced origin and path observation is required", false, "")
				return
			}
		} else if request.RegistrationRevision != "" {
			result.fail(http.StatusBadRequest, "discovery_invalid", "unexpected observation revision", false, "")
			return
		}
		inventory, err = store.Change(request.Change, observedURL)
	}
	if err != nil {
		code, status := "discovery_storage", http.StatusConflict
		if errors.Is(err, registrationdiscovery.ErrInvalid) {
			code, status = "discovery_invalid", http.StatusBadRequest
		}
		if errors.Is(err, registrationdiscovery.ErrStale) {
			code = "discovery_stale"
		}
		result.fail(status, code, "discovery operation rejected; reload the private inventory before continuing", false, "")
		return
	}
	result.respond(http.StatusOK, inventory)
	// Detached bytes are exposed by control only for explicit discovery replies.
	result.Discovery = append(json.RawMessage(nil), result.Data.(json.RawMessage)...)
	return
}

func (s *Server) serveRegistrationDiscovery(w http.ResponseWriter, r *http.Request, cookieScoped bool, requestID string) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		s.methodNotAllowed(w, "GET, POST", requestID)
		return
	}
	if !s.authenticated(r, cookieScoped) {
		s.writeError(w, http.StatusUnauthorized, "unauthorized", "a valid UI capability token is required", false, requestID, "")
		return
	}
	var request *registrationDiscoveryRequest
	if r.Method == http.MethodPost {
		request = &registrationDiscoveryRequest{}
		if err := decodeJSONRequest(w, r, request); err != nil {
			s.writeRequestError(w, err, requestID)
			return
		}
	}
	s.writeApplicationResult(w, r, requestID, (Application{server: s}).RegistrationDiscovery(r.Context(), request))
}
