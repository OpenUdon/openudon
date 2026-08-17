package browserscenario

import (
	"fmt"
	"html"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"
)

const scenarioHTTPTimeout = 5 * time.Second

// LoopbackFixture is one process-local, credential-free browser target. Its
// address is selected by the kernel and never accepted from a caller.
type LoopbackFixture struct {
	server   *httptest.Server
	manifest Manifest
	mu       sync.RWMutex
	variant  string
	runtime  bool
}

func NewLoopbackFixture(manifest Manifest) (*LoopbackFixture, error) {
	if err := ValidateManifest(manifest, time.Now().UTC()); err != nil || manifest.Suite != SuiteLoopback {
		return nil, fmt.Errorf("invalid loopback fixture manifest")
	}
	fixture := &LoopbackFixture{manifest: manifest}
	fixture.server = httptest.NewUnstartedServer(http.HandlerFunc(fixture.serveHTTP))
	fixture.server.Config.ReadHeaderTimeout = scenarioHTTPTimeout
	fixture.server.Config.WriteTimeout = scenarioHTTPTimeout
	fixture.server.Config.IdleTimeout = scenarioHTTPTimeout
	fixture.server.Start()
	return fixture, nil
}

func (fixture *LoopbackFixture) Close() {
	if fixture != nil && fixture.server != nil {
		fixture.server.Close()
	}
}

func (fixture *LoopbackFixture) Origin() string { return fixture.server.URL }

func (fixture *LoopbackFixture) InitialURL() string {
	return fixture.server.URL + fixture.path("login")
}

func (fixture *LoopbackFixture) DashboardURL() string {
	page := "dashboard"
	if fixture.manifest.Authentication.ContextMode == "frame" {
		page = "embedded"
	}
	return fixture.server.URL + fixture.path(page)
}

func (fixture *LoopbackFixture) AuthenticationURL() string {
	if fixture.manifest.Authentication.ContextMode == "popup" {
		return fixture.server.URL + fixture.path("dashboard")
	}
	return fixture.DashboardURL()
}

func (fixture *LoopbackFixture) GoalURL() string {
	if fixture.manifest.Authentication.ContextMode == "popup" {
		return fixture.server.URL + fixture.path("popup")
	}
	return fixture.DashboardURL()
}

func (fixture *LoopbackFixture) SetReplayVariant(variant string) error {
	if variant != "" && !allowedReplayVariants[variant] {
		return fmt.Errorf("unknown loopback replay variant")
	}
	fixture.mu.Lock()
	fixture.variant = variant
	fixture.mu.Unlock()
	return nil
}

func (fixture *LoopbackFixture) SetRuntime(value bool) {
	fixture.mu.Lock()
	fixture.runtime = value
	fixture.mu.Unlock()
}

func (fixture *LoopbackFixture) path(page string) string {
	return "/scenario/" + fixture.manifest.ID + "/" + page
}

func (fixture *LoopbackFixture) serveHTTP(writer http.ResponseWriter, request *http.Request) {
	fixture.mu.RLock()
	allowEscape := fixture.runtime && fixture.manifest.Fault == "origin_escape"
	fixture.mu.RUnlock()
	formAction := "'self'"
	if allowEscape {
		formAction += " http://127.0.0.1:9"
	}
	writer.Header().Set("Cache-Control", "no-store")
	writer.Header().Set("Content-Security-Policy", "default-src 'self'; frame-src 'self'; form-action "+formAction+"; base-uri 'none'")
	writer.Header().Set("X-Content-Type-Options", "nosniff")
	if request.URL.RawQuery != "" || request.URL.Fragment != "" || !strings.HasPrefix(request.URL.Path, "/scenario/"+fixture.manifest.ID+"/") {
		http.NotFound(writer, request)
		return
	}
	page := strings.TrimPrefix(request.URL.Path, "/scenario/"+fixture.manifest.ID+"/")
	switch page {
	case "login":
		fixture.login(writer)
	case "challenge":
		fixture.challenge(writer)
	case "dashboard":
		fixture.dashboard(writer)
	case "embedded":
		fixture.embedded(writer)
	case "popup":
		fixture.popup(writer)
	default:
		http.NotFound(writer, request)
	}
}

func (fixture *LoopbackFixture) login(writer http.ResponseWriter) {
	action := fixture.path("dashboard")
	if fixture.manifest.Authentication.ChallengeKind != "" {
		action = fixture.path("challenge")
	}
	fixture.mu.RLock()
	runtime, originEscape := fixture.runtime, fixture.manifest.Fault == "origin_escape"
	fixture.mu.RUnlock()
	if runtime && originEscape {
		action = "http://127.0.0.1:9/blocked-origin"
	}
	button := `<button type="submit">Sign in</button>`
	formAttributes := `method="post" action="` + html.EscapeString(action) + `"`
	writeScenarioHTML(writer, "Sign in", `<main>
<h1>Scenario sign in</h1>
<form `+formAttributes+`>
<label>Email address <input type="email" autocomplete="username" aria-label="Email address"></label>
<label>Password <input type="password" autocomplete="current-password" aria-label="Password"></label>
`+button+`
</form>
</main>`)
}

func (fixture *LoopbackFixture) challenge(writer http.ResponseWriter) {
	kind := fixture.manifest.Authentication.ChallengeKind
	if kind == "" {
		fixture.dashboard(writer)
		return
	}
	if allowedOTPChallenge(kind) {
		writeScenarioHTML(writer, "Verification", `<main>
<h1>Verification required</h1>
<form method="post" action="`+html.EscapeString(fixture.path("dashboard"))+`">
<label>Verification code <input type="text" inputmode="numeric" autocomplete="one-time-code" aria-label="Verification code"></label>
<button type="submit">Verify</button>
</form>
</main>`)
		return
	}
	number := ""
	if kind == "push_number_match" {
		number = `<p>Number <strong>42</strong></p>`
	}
	writeScenarioHTML(writer, "Approve sign in", `<main>
<h1>Additional verification</h1>
<div role="status" aria-label="Approve verification request">Approve verification request</div>
`+number+`
<form method="post" action="`+html.EscapeString(fixture.path("dashboard"))+`"><button type="submit">Continue</button></form>
</main>`)
}

func (fixture *LoopbackFixture) dashboard(writer http.ResponseWriter) {
	frame := ""
	if fixture.manifest.Authentication.ContextMode == "frame" {
		frame = `<iframe name="scenario-frame" title="Embedded dashboard frame" src="` + html.EscapeString(fixture.path("embedded")) + `"></iframe>`
	}
	if fixture.manifest.Authentication.ContextMode == "popup" {
		frame = `<a href="` + html.EscapeString(fixture.path("popup")) + `" target="scenario-popup">Open member report</a>`
	}
	writeScenarioHTML(writer, "Member dashboard", `<main>
<h1>Member dashboard</h1>
`+fixture.outputMarkup()+frame+`
</main>`)
}

func (fixture *LoopbackFixture) popup(writer http.ResponseWriter) {
	writeScenarioHTML(writer, "Member report", `<main>
<h1>Member dashboard</h1>
`+fixture.outputMarkup()+`
</main>`)
}

func (fixture *LoopbackFixture) embedded(writer http.ResponseWriter) {
	writeScenarioHTML(writer, "Embedded dashboard", `<main>
<h1>Embedded dashboard</h1>
<div role="status" aria-label="Embedded status">Embedded ready</div>
</main>`)
}

func (fixture *LoopbackFixture) outputMarkup() string {
	fixture.mu.RLock()
	variant, runtime := fixture.variant, fixture.runtime
	fixture.mu.RUnlock()
	account := "Ada Lovelace"
	itemCount, usageRatio, enabled := "42", "-12.5e2", "true"
	if runtime && fixture.manifest.Fault == "secret_output" {
		account = "sk-proj-abcdefghijklmnopqrstuvwxyz1234567890"
	}
	if runtime {
		switch variant {
		case "integer_leading_zero":
			itemCount = "01"
		case "integer_plus":
			itemCount = "+1"
		case "integer_comma":
			itemCount = "1,000"
		case "integer_unsafe":
			itemCount = "9007199254740992"
		case "number_nan":
			usageRatio = "NaN"
		case "number_infinity":
			usageRatio = "Infinity"
		case "number_comma":
			usageRatio = "1,000.5"
		case "boolean_uppercase":
			enabled = "True"
		case "boolean_numeric":
			enabled = "1"
		case "empty":
			itemCount = "   "
		}
	}
	var builder strings.Builder
	fmt.Fprintf(&builder, `<div role="status" aria-label="Account name">%s</div>`, html.EscapeString(account))
	fmt.Fprintf(&builder, `<div role="status" aria-label="Item count">%s</div>`, html.EscapeString(itemCount))
	fmt.Fprintf(&builder, `<div role="status" aria-label="Usage ratio">%s</div>`, html.EscapeString(usageRatio))
	fmt.Fprintf(&builder, `<div role="status" aria-label="Feature enabled">%s</div>`, html.EscapeString(enabled))
	builder.WriteString(`<section role="region" aria-label="Plan summary">Professional plan</section>`)
	builder.WriteString(`<div role="status" aria-label="Primary status">Ready</div><div role="status" aria-label="Secondary status">Healthy</div>`)
	if fixture.manifest.Fault == "ambiguous_unique_role" {
		builder.WriteString(`<article role="article" aria-label="Primary article">Primary summary</article><article role="article" aria-label="Secondary article">Secondary summary</article>`)
	} else {
		builder.WriteString(`<article role="article" aria-label="Summary">Stable summary</article>`)
	}
	for index := 1; index <= 17; index++ {
		fmt.Fprintf(&builder, `<h2>Metric %02d</h2>`, index)
	}
	return builder.String()
}

func writeScenarioHTML(writer http.ResponseWriter, title, body string) {
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprintf(writer, "<!doctype html><html lang=\"en\"><head><meta charset=\"utf-8\"><title>%s</title></head><body>%s</body></html>", html.EscapeString(title), body)
}

func allowedOTPChallenge(value string) bool {
	return value == "totp" || value == "sms_otp" || value == "email_otp" || value == "voice_otp"
}
