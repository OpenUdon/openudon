// Package registrationdiscovery owns private authoring inventory, not portable
// workflow semantics or browser authority. It never contacts a target.
package registrationdiscovery

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/url"
	"regexp"
	"slices"
	"strings"
)

const Version = "openudon.registration-discovery.v1"

var (
	ErrInvalid = errors.New("invalid discovery request")
	ErrStale   = errors.New("stale discovery revision")
	ErrStorage = errors.New("private discovery storage unavailable")
	symbol     = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)
)

// Candidate contains an authoring hint, never a verified registration recipe.
// Browser-observed URLs contain only the observed origin and path; query data
// must be supplied and reviewed separately through the existing wizard.
type Candidate struct {
	ID               string `json:"id"`
	RegistrationType string `json:"registration_type"`
	URL              string `json:"url"`
	Source           string `json:"source"`
}

type Inventory struct {
	Version          string      `json:"version"`
	Sequence         int         `json:"sequence"`
	PreviousRevision string      `json:"previous_revision,omitempty"`
	Revision         string      `json:"revision"`
	Coverage         string      `json:"coverage"`
	Limitations      []string    `json:"limitations"`
	OwnerReview      string      `json:"owner_review"`
	Candidates       []Candidate `json:"candidates"`
	SelectedID       string      `json:"selected_id,omitempty"`
}

// Change is an exact-revision operation. Source is deliberately absent: only
// the application may attach a current native observation to add_observed.
type Change struct {
	Revision         string   `json:"revision"`
	Action           string   `json:"action"`
	ID               string   `json:"id,omitempty"`
	RegistrationType string   `json:"registration_type,omitempty"`
	URL              string   `json:"url,omitempty"`
	Coverage         string   `json:"coverage,omitempty"`
	Limitations      []string `json:"limitations,omitempty"`
}

func empty() Inventory {
	i := Inventory{Version: Version, Coverage: "unknown", OwnerReview: "pending", Limitations: []string{"unknown_routes"}, Candidates: []Candidate{}}
	i.Revision = digest(i)
	return i
}

func digest(i Inventory) string {
	i.Revision = ""
	b, _ := json.Marshal(i)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func validURL(raw string) bool {
	if len(raw) == 0 || len(raw) > 2048 || strings.TrimSpace(raw) != raw || strings.ContainsAny(raw, "\r\n\t\\") {
		return false
	}
	u, err := url.Parse(raw)
	return err == nil && (u.Scheme == "https" || u.Scheme == "http") && u.Hostname() != "" && u.User == nil && u.Fragment == "" && u.Opaque == "" && u.String() == raw
}

func validLimits(limits []string) bool {
	allowed := []string{"authentication_required", "invitation_required", "conditional_flow", "unreachable_page", "unknown_routes"}
	if limits == nil || len(limits) > len(allowed) {
		return false
	}
	seen := map[string]bool{}
	for _, limit := range limits {
		if !slices.Contains(allowed, limit) || seen[limit] {
			return false
		}
		seen[limit] = true
	}
	return true
}

func valid(i Inventory) bool {
	if i.Version != Version || i.Sequence < 0 || i.Sequence > 512 || (i.Coverage != "unknown" && i.Coverage != "partial") || (i.OwnerReview != "pending" && i.OwnerReview != "reviewed") || !validLimits(i.Limitations) || i.Candidates == nil || len(i.Candidates) > 32 {
		return false
	}
	seen := map[string]bool{}
	for _, c := range i.Candidates {
		if !symbol.MatchString(c.ID) || !symbol.MatchString(c.RegistrationType) || !validURL(c.URL) || seen[c.ID] || (c.Source != "operator_supplied" && c.Source != "browser_observed") {
			return false
		}
		seen[c.ID] = true
	}
	return i.SelectedID == "" || seen[i.SelectedID]
}

func apply(i Inventory, c Change, observedURL string) (Inventory, error) {
	if c.Revision != i.Revision {
		return Inventory{}, ErrStale
	}
	// Reject fields belonging to a different operation rather than silently
	// accepting ignored authority, URL or coverage claims.
	switch c.Action {
	case "add", "add_observed":
		if c.Coverage != "" || c.Limitations != nil || !symbol.MatchString(c.ID) || !symbol.MatchString(c.RegistrationType) {
			return Inventory{}, ErrInvalid
		}
		for _, entry := range i.Candidates {
			if entry.ID == c.ID {
				return Inventory{}, ErrInvalid
			}
		}
		entry := Candidate{ID: c.ID, RegistrationType: c.RegistrationType, URL: c.URL, Source: "operator_supplied"}
		if c.Action == "add_observed" {
			if c.URL != "" || observedURL == "" {
				return Inventory{}, ErrInvalid
			}
			entry.URL, entry.Source = observedURL, "browser_observed"
		}
		i.Candidates = append(slices.Clone(i.Candidates), entry)
		i.OwnerReview = "pending"
	case "remove", "select":
		if c.URL != "" || c.RegistrationType != "" || c.Coverage != "" || c.Limitations != nil {
			return Inventory{}, ErrInvalid
		}
		index := slices.IndexFunc(i.Candidates, func(entry Candidate) bool { return entry.ID == c.ID })
		if index < 0 {
			return Inventory{}, ErrInvalid
		}
		if c.Action == "remove" {
			i.Candidates = slices.Delete(slices.Clone(i.Candidates), index, index+1)
			if i.SelectedID == c.ID {
				i.SelectedID = ""
			}
		} else {
			i.SelectedID = c.ID
		}
		i.OwnerReview = "pending"
	case "coverage":
		if c.ID != "" || c.RegistrationType != "" || c.URL != "" || !validLimits(c.Limitations) {
			return Inventory{}, ErrInvalid
		}
		i.Coverage, i.Limitations, i.OwnerReview = c.Coverage, slices.Clone(c.Limitations), "pending"
	case "review":
		if c.ID != "" || c.RegistrationType != "" || c.URL != "" || c.Coverage != "" || c.Limitations != nil {
			return Inventory{}, ErrInvalid
		}
		i.OwnerReview = "reviewed"
	default:
		return Inventory{}, ErrInvalid
	}
	i.Sequence++
	i.PreviousRevision, i.Revision = i.Revision, ""
	if !valid(i) {
		return Inventory{}, ErrInvalid
	}
	i.Revision = digest(i)
	return i, nil
}
