package browserscenario

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/OpenUdon/browsertools/registrationprofile"
	"github.com/OpenUdon/openudon/internal/evidencefile"
	"github.com/OpenUdon/openudon/internal/processgroup"
	"github.com/mxschmitt/playwright-go"
)

// All values here are fixed synthetic fixture data. The separate Udon CLI and
// shipped private UI are exercised without importing private runtime packages.
type registrationInputQualification struct {
	child           *processgroup.InteractiveChild
	drained         chan struct{}
	pw              *playwright.Playwright
	browser         playwright.Browser
	page            playwright.Page
	endpoint        string
	initialIdentity string
}

const registrationInputQualificationToken = "synthetic-registration-private-form-capability"

func startRegistrationInputQualification(ctx context.Context, root, udon string, authority registrationQualificationRuntime, environment []string) (_ *registrationInputQualification, resultErr error) {
	q := &registrationInputQualification{}
	defer func() {
		if resultErr != nil {
			_ = q.Close()
		}
	}()
	bad := func(stage string) error { return errors.New("registration private UI qualification: " + stage) }
	profile, err := registrationprofile.Parse(authority.profile)
	if err != nil {
		return nil, bad("profile")
	}
	data, err := registrationprofile.MarshalJSON(profile)
	if err != nil {
		return nil, bad("profile")
	}
	profilePath := filepath.Join(root, "private-form-profile.json")
	descriptor := filepath.Join(root, "private-form-service.json")
	if os.WriteFile(profilePath, data, 0600) != nil {
		return nil, bad("profile")
	}
	args := []string{udon, "registration-input", "serve", "--profile", profilePath, "--operation", authority.operation, "--flow", authority.flow, "--binding", authority.binding, "--token-env", "UDON_REGISTRATION_INPUT_TOKEN", "--descriptor", descriptor}
	q.child, err = processgroup.StartInteractiveIn(ctx, root, args, append(environment, "UDON_REGISTRATION_INPUT_TOKEN="+registrationInputQualificationToken), io.Discard)
	if err != nil {
		return nil, bad("service")
	}
	_ = q.child.Input().Close()
	q.drained = make(chan struct{})
	go func() { _, _ = io.Copy(io.Discard, q.child.Output()); close(q.drained) }()
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for q.endpoint == "" {
		select {
		case <-ctx.Done():
			return nil, bad("service deadline")
		case <-q.drained:
			return nil, bad("service stopped")
		case <-ticker.C:
		}
		data, _, err := evidencefile.ReadRegular(descriptor, 4096)
		if err != nil {
			continue
		}
		var service struct {
			Version          string `json:"version"`
			URL              string `json:"url"`
			TokenEnvironment string `json:"tokenEnvironment"`
			OperationID      string `json:"operationId"`
			Binding          string `json:"binding"`
			Flow             string `json:"flow"`
		}
		if evidencefile.DecodeStrict(data, &service) != nil || service.Version != "udon.registration-input-service.v1" || service.OperationID != authority.operation || service.Flow != authority.flow || service.Binding != authority.binding || !strings.HasPrefix(service.URL, "http://127.0.0.1:") {
			return nil, bad("service descriptor")
		}
		q.endpoint = service.URL
	}
	q.pw, err = playwright.Run()
	if err != nil {
		return nil, bad("playwright")
	}
	q.browser, err = q.pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{Headless: playwright.Bool(true), ChromiumSandbox: playwright.Bool(true)})
	if err != nil {
		return nil, bad("browser")
	}
	q.page, err = q.browser.NewPage(playwright.BrowserNewPageOptions{ExtraHttpHeaders: map[string]string{"Authorization": "Bearer " + registrationInputQualificationToken}})
	if err != nil {
		return nil, bad("page")
	}
	q.page.SetDefaultTimeout(15000)
	if err = q.page.Route("**/*", func(route playwright.Route) {
		if !strings.HasPrefix(route.Request().URL(), q.endpoint) {
			_ = route.Abort()
			return
		}
		_ = route.Continue()
	}); err != nil {
		return nil, bad("origin")
	}
	if _, err = q.page.Goto(q.endpoint); err != nil {
		return nil, bad("navigation")
	}
	for _, field := range []struct{ name, value string }{{"identifier", "dedicated-test@example.test"}, {"password", "qualification-password-value"}, {"contact_name", "Synthetic Å member"}} {
		if q.page.Locator("#private-field-"+field.name).Fill(field.value) != nil {
			return nil, bad("initial fields")
		}
	}
	choice := []string{"1"}
	if _, err = q.page.Locator("#private-field-account_kind").SelectOption(playwright.SelectOptionValues{Values: &choice}); err != nil {
		return nil, bad("public choice")
	}
	if q.page.Locator("#private-field-company").Fill("Synthetic company") != nil {
		return nil, bad("conditional field")
	}
	if q.button("Start").Click() != nil {
		return nil, bad("Start")
	}
	sum := sha256.Sum256(data)
	body, _ := json.Marshal(map[string]string{"operationId": authority.operation, "flow": authority.flow, "binding": authority.binding, "profileSha256": hex.EncodeToString(sum[:])})
	client := &http.Client{Timeout: 2 * time.Second, Transport: &http.Transport{Proxy: nil}, CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("redirect") }}
	defer client.CloseIdleConnections()
	for q.initialIdentity == "" {
		select {
		case <-ctx.Done():
			return nil, bad("readiness deadline")
		case <-ticker.C:
		}
		request, _ := http.NewRequestWithContext(ctx, "POST", q.endpoint+"runtime/readiness", bytes.NewReader(body))
		request.Header.Set("Authorization", "Bearer "+registrationInputQualificationToken)
		request.Header.Set("Content-Type", "application/json")
		response, err := client.Do(request)
		if err != nil {
			return nil, bad("readiness")
		}
		accepted, err := io.ReadAll(io.LimitReader(response.Body, 4097))
		_ = response.Body.Close()
		if err != nil {
			return nil, bad("readiness")
		}
		if response.StatusCode != 200 {
			continue
		}
		var identity struct {
			Revision uint64 `json:"inputRevision"`
			SHA256   string `json:"inputSha256"`
		}
		if json.Unmarshal(accepted, &identity) != nil || identity.Revision != 1 || len(identity.SHA256) != 64 {
			return nil, bad("readiness identity")
		}
		q.initialIdentity = identity.SHA256
	}
	return q, nil
}
func (q *registrationInputQualification) button(name string) playwright.Locator {
	return q.page.GetByRole("button", playwright.PageGetByRoleOptions{Name: name, Exact: playwright.Bool(true)})
}
func (q *registrationInputQualification) Environment(environment []string) []string {
	expected, _ := json.Marshal(map[string]any{"inputRevision": 1, "inputSha256": q.initialIdentity})
	return append(environment, "UDON_REGISTRATION_INPUT_TOKEN="+registrationInputQualificationToken, "UDON_REGISTRATION_INPUT_EXPECTED="+string(expected))
}
func (q *registrationInputQualification) Continue() error {
	bad := errors.New("registration private UI checkpoint qualification")
	for _, field := range []struct{ name, value string }{{"quantity", "0"}, {"ratio", "0.0000001"}} {
		if q.page.Locator("#private-field-"+field.name).Fill(field.value) != nil {
			return bad
		}
	}
	choice := []string{"1"}
	if _, err := q.page.Locator("#private-field-updates").SelectOption(playwright.SelectOptionValues{Values: &choice}); err != nil {
		return bad
	}
	if q.button("Apply").Click() != nil || q.button("Approve registration").Click() != nil || q.button("Continue").Click() != nil {
		return bad
	}
	return nil
}
func (q *registrationInputQualification) Close() error {
	bad := false
	if q.browser != nil && q.browser.Close() != nil {
		bad = true
	}
	if q.pw != nil && q.pw.Stop() != nil {
		bad = true
	}
	if q.child != nil {
		err := q.child.Terminate()
		var exited *exec.ExitError
		if err != nil && (!errors.As(err, &exited) || errors.Is(err, processgroup.ErrTerminationTimeout)) {
			bad = true
		}
	}
	if q.drained != nil {
		select {
		case <-q.drained:
		case <-time.After(5 * time.Second):
			bad = true
		}
	}
	if bad {
		return errors.New("registration private UI teardown")
	}
	return nil
}

func (q *registrationInputQualification) VerifyComplete(directory string, files map[string][]byte) error {
	bad := errors.New("registration private UI completion or artifact privacy failed")
	if q.page.GetByText("Registration completed.", playwright.PageGetByTextOptions{Exact: playwright.Bool(true)}).WaitFor() != nil {
		return bad
	}
	if count, err := q.page.Locator("#fields input, #fields select").Count(); err != nil || count != 0 {
		return bad
	}
	check := func(data []byte) error {
		for _, value := range []string{q.initialIdentity, registrationInputQualificationToken, "dedicated-test@example.test", "qualification-password-value", "Synthetic Å member", "Synthetic company"} {
			if bytes.Contains(data, []byte(value)) {
				return bad
			}
		}
		return nil
	}
	for _, data := range files {
		if check(data) != nil {
			return bad
		}
	}
	return filepath.WalkDir(directory, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return bad
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			return bad
		}
		data, _, err := evidencefile.ReadRegular(path, 8<<20)
		if err != nil {
			return bad
		}
		return check(data)
	})
}
