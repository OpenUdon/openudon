package ui

import (
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"time"

	"github.com/OpenUdon/browsertools/registrationauthorsession"

	"github.com/OpenUdon/openudon/internal/evidencefile"
)

// RegistrationAuthority is optional consumer-owned authority for one process.
// It restricts the shared application, so HTTP and control cannot bypass it.
// It contains no credentials and does not authorize workflow execution.
type RegistrationAuthority struct {
	Version        string   `json:"version"`
	ProfileID      string   `json:"profile_id"`
	ProfileVersion string   `json:"profile_version"`
	InitialURL     string   `json:"initial_url"`
	Origins        []string `json:"origins"`
	NavigationURLs []string `json:"navigation_urls"`
	ExpiresAt      string   `json:"expires_at"`
}

func ReadRegistrationAuthority(path string, now time.Time) (*RegistrationAuthority, error) {
	info, err := os.Lstat(path)
	if err != nil || !filepath.IsAbs(path) || !info.Mode().IsRegular() || info.Mode().Perm() != 0600 || info.Size() > 32<<10 {
		return nil, errors.New("registration authority file is invalid")
	}
	data, _, err := evidencefile.ReadRegular(path, 32<<10)
	if err != nil {
		return nil, err
	}
	var authority RegistrationAuthority
	if evidencefile.DecodeStrict(data, &authority) != nil || authority.validate(now) != nil {
		return nil, errors.New("registration authority is invalid")
	}
	return &authority, nil
}

func (a *RegistrationAuthority) validate(now time.Time) error {
	bad := errors.New("registration authority is invalid")
	if a == nil {
		return nil
	}
	expires, err := time.Parse(time.RFC3339, a.ExpiresAt)
	if err != nil || !expires.After(now) || expires.Sub(now) > 20*time.Minute || a.Version != "openudon.registration-authority.v1" || !registrationDraftSymbol.MatchString(a.ProfileID) || a.ProfileVersion != "1.0" && a.ProfileVersion != "1.1" || len(a.Origins) == 0 || len(a.Origins) > 8 || !sort.StringsAreSorted(a.Origins) || len(a.NavigationURLs) == 0 || len(a.NavigationURLs) > 32 {
		return bad
	}
	origins := map[string]bool{}
	for _, origin := range a.Origins {
		parsed, err := url.Parse(origin)
		if err != nil || parsed.User != nil || parsed.Scheme != "https" && parsed.Scheme != "http" || parsed.Host == "" || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" || origins[origin] {
			return bad
		}
		origins[origin] = true
	}
	initial := false
	seen := map[string]bool{}
	for _, navigation := range a.NavigationURLs {
		parsed, err := url.Parse(navigation)
		if err != nil || parsed.User != nil || parsed.Fragment != "" || !origins[parsed.Scheme+"://"+parsed.Host] || seen[navigation] {
			return bad
		}
		if _, _, _, err := registrationauthorsession.ValidateNavigationURL(registrationauthorsession.ProtocolV3, navigation); err != nil {
			return bad
		}
		seen[navigation] = true
		initial = initial || navigation == a.InitialURL
	}
	if !initial {
		return bad
	}
	return nil
}
func (a *RegistrationAuthority) allowsStart(r registrationAuthoringStartRequest, now time.Time) bool {
	if a == nil {
		return true
	}
	version := r.ProfileVersion
	if version == "" {
		version = "1.0"
	}
	origins := append([]string(nil), r.Origins...)
	sort.Strings(origins)
	return a.validate(now) == nil && r.ProfileID == a.ProfileID && version == a.ProfileVersion && r.URL == a.InitialURL && reflect.DeepEqual(origins, a.Origins)
}
func (a *RegistrationAuthority) allowsNavigation(target string, now time.Time) bool {
	if a == nil {
		return true
	}
	if a.validate(now) != nil {
		return false
	}
	for _, allowed := range a.NavigationURLs {
		if target == allowed {
			return true
		}
	}
	return false
}
