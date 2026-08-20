package browserscenario

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"html"
	"net/http"
	"net/http/httptest"
	"strconv"
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
	sessions map[string]loopbackSession
	replay   loopbackReplayEvidence
}

type loopbackSession struct {
	loginComplete     bool
	challengeComplete bool
}

type loopbackReplayEvidence struct {
	loginAccepted     int
	challengeAccepted int
	protectedReads    int
}

func NewLoopbackFixture(manifest Manifest) (*LoopbackFixture, error) {
	if err := ValidateManifest(manifest, time.Now().UTC()); err != nil || manifest.Suite != SuiteLoopback {
		return nil, fmt.Errorf("invalid loopback fixture manifest")
	}
	fixture := &LoopbackFixture{manifest: manifest, sessions: map[string]loopbackSession{}}
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
	fixture.sessions = map[string]loopbackSession{}
	fixture.replay = loopbackReplayEvidence{}
	fixture.mu.Unlock()
}

// AuthenticatedReplayObserved reports whether the runtime replay completed
// login, any configured challenge, and at least one protected read.
func (fixture *LoopbackFixture) AuthenticatedReplayObserved() bool {
	fixture.mu.RLock()
	defer fixture.mu.RUnlock()
	wantChallenge := fixture.manifest.Authentication.ChallengeKind != ""
	return fixture.replay.loginAccepted > 0 && fixture.replay.protectedReads > 0 && (!wantChallenge || fixture.replay.challengeAccepted > 0)
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
		fixture.login(writer, request)
	case "challenge":
		fixture.challenge(writer, request)
	case "dashboard":
		fixture.dashboard(writer, request)
	case "embedded":
		fixture.embedded(writer, request)
	case "popup":
		fixture.popup(writer, request)
	default:
		http.NotFound(writer, request)
	}
}

func (fixture *LoopbackFixture) login(writer http.ResponseWriter, request *http.Request) {
	next := fixture.path("dashboard")
	if fixture.manifest.Authentication.ChallengeKind != "" {
		next = fixture.path("challenge")
	}
	fixture.mu.RLock()
	runtime, originEscape := fixture.runtime, fixture.manifest.Fault == "origin_escape"
	fixture.mu.RUnlock()
	if runtime && originEscape {
		next = "http://127.0.0.1:9/blocked-origin"
	}
	if request.Method == http.MethodPost {
		if runtime {
			if request.ParseForm() != nil || request.PostForm.Get("identifier") != "member@example.test" || request.PostForm.Get("password") != "scenario-password-value" {
				http.Error(writer, "invalid credentials", http.StatusUnauthorized)
				return
			}
			token, err := newLoopbackSessionToken()
			if err != nil {
				http.Error(writer, "session unavailable", http.StatusInternalServerError)
				return
			}
			fixture.mu.Lock()
			fixture.sessions[token] = loopbackSession{loginComplete: true, challengeComplete: fixture.manifest.Authentication.ChallengeKind == ""}
			fixture.replay.loginAccepted++
			fixture.mu.Unlock()
			http.SetCookie(writer, &http.Cookie{Name: "openudon_scenario_session", Value: token, Path: fixture.path(""), HttpOnly: true, SameSite: http.SameSiteStrictMode})
		}
		http.Redirect(writer, request, next, http.StatusSeeOther)
		return
	}
	if request.Method != http.MethodGet {
		writer.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	button := `<button type="submit">Sign in</button>`
	formAttributes := `method="post" action="` + html.EscapeString(fixture.path("login")) + `"`
	writeScenarioHTML(writer, "Sign in", `<main>
<h1>Scenario sign in</h1>
<form `+formAttributes+`>
<label>Email address <input name="identifier" type="email" autocomplete="username" aria-label="Email address"></label>
<label>Password <input name="password" type="password" autocomplete="current-password" aria-label="Password"></label>
`+button+`
</form>
</main>`)
}

func (fixture *LoopbackFixture) challenge(writer http.ResponseWriter, request *http.Request) {
	kind := fixture.manifest.Authentication.ChallengeKind
	if kind == "" {
		fixture.dashboard(writer, request)
		return
	}
	if fixture.runtime {
		token, session, ok := fixture.runtimeSession(request)
		if !ok || !session.loginComplete {
			http.Error(writer, "authentication required", http.StatusUnauthorized)
			return
		}
		if request.Method == http.MethodPost {
			if request.ParseForm() != nil || !validLoopbackChallenge(kind, request.PostForm.Get("challenge"), time.Now()) {
				http.Error(writer, "invalid challenge", http.StatusUnauthorized)
				return
			}
			session.challengeComplete = true
			fixture.mu.Lock()
			fixture.sessions[token] = session
			fixture.replay.challengeAccepted++
			fixture.mu.Unlock()
			http.Redirect(writer, request, fixture.path("dashboard"), http.StatusSeeOther)
			return
		}
		if request.Method != http.MethodGet {
			writer.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
	}
	if allowedOTPChallenge(kind) {
		writeScenarioHTML(writer, "Verification", `<main>
<h1>Verification required</h1>
<form method="post" action="`+html.EscapeString(fixture.path("dashboard"))+`">
<label>Verification code <input name="challenge" type="text" inputmode="numeric" autocomplete="one-time-code" aria-label="Verification code"></label>
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
<form method="post" action="`+html.EscapeString(fixture.path("challenge"))+`"><button name="challenge" value="`+html.EscapeString(kind)+`" type="submit">Continue</button></form>
</main>`)
}

func (fixture *LoopbackFixture) dashboard(writer http.ResponseWriter, request *http.Request) {
	if fixture.runtime {
		if request.Method == http.MethodPost {
			// OTP profiles submit directly to the reviewed dashboard URL.
			fixture.challenge(writer, request)
			return
		}
		if !fixture.requireRuntimeAuthentication(writer, request) {
			return
		}
	}
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

func (fixture *LoopbackFixture) popup(writer http.ResponseWriter, request *http.Request) {
	if fixture.runtime && !fixture.requireRuntimeAuthentication(writer, request) {
		return
	}
	writeScenarioHTML(writer, "Member report", `<main>
<h1>Member dashboard</h1>
`+fixture.outputMarkup()+`
</main>`)
}

func (fixture *LoopbackFixture) embedded(writer http.ResponseWriter, request *http.Request) {
	if fixture.runtime && !fixture.requireRuntimeAuthentication(writer, request) {
		return
	}
	writeScenarioHTML(writer, "Embedded dashboard", `<main>
<h1>Embedded dashboard</h1>
<div role="status" aria-label="Embedded status">Embedded ready</div>
</main>`)
}

func (fixture *LoopbackFixture) outputMarkup() string {
	fixture.mu.RLock()
	variant, runtime := fixture.variant, fixture.runtime
	fixture.mu.RUnlock()
	values := loopbackRenderedValues(runtime, variant, fixture.manifest.Fault)
	var builder strings.Builder
	fmt.Fprintf(&builder, `<div role="status" aria-label="Account name">%s</div>`, html.EscapeString(values["Account name"]))
	fmt.Fprintf(&builder, `<div role="status" aria-label="Item count">%s</div>`, html.EscapeString(values["Item count"]))
	fmt.Fprintf(&builder, `<div role="status" aria-label="Usage ratio">%s</div>`, html.EscapeString(values["Usage ratio"]))
	fmt.Fprintf(&builder, `<div role="status" aria-label="Feature enabled">%s</div>`, html.EscapeString(values["Feature enabled"]))
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

// ExpectedOutputs derives replay expectations from the same fixture facts
// used to render the page, avoiding a second markup-knowledge table.
func (fixture *LoopbackFixture) ExpectedOutputs(outputs []Output) map[string]any {
	values := loopbackRenderedValues(false, "", "")
	result := map[string]any{"goal_present": true}
	for _, output := range outputs {
		raw := values[output.Name]
		if strings.HasPrefix(output.Name, "Metric ") {
			raw = output.Name
		}
		if output.LocatorMode == "unique_role" {
			raw = values["Summary"]
		}
		switch output.Type {
		case "integer":
			value, _ := strconv.ParseInt(raw, 10, 64)
			result[output.Key] = float64(value)
		case "number":
			value, _ := strconv.ParseFloat(raw, 64)
			result[output.Key] = value
		case "boolean":
			value, _ := strconv.ParseBool(raw)
			result[output.Key] = value
		case "presence":
			result[output.Key] = true
		default:
			result[output.Key] = raw
		}
	}
	return result
}

func loopbackRenderedValues(runtime bool, variant, fault string) map[string]string {
	values := map[string]string{
		"Account name": "Ada Lovelace", "Item count": "42", "Usage ratio": "-12.5e2", "Feature enabled": "true",
		"Plan summary": "Professional plan", "Primary status": "Ready", "Secondary status": "Healthy", "Summary": "Stable summary", "Embedded status": "Embedded ready",
	}
	if runtime && fault == "secret_output" {
		values["Account name"] = "sk-proj-abcdefghijklmnopqrstuvwxyz1234567890"
	}
	if runtime {
		switch variant {
		case "integer_leading_zero":
			values["Item count"] = "01"
		case "integer_plus":
			values["Item count"] = "+1"
		case "integer_comma":
			values["Item count"] = "1,000"
		case "integer_unsafe":
			values["Item count"] = "9007199254740992"
		case "number_nan":
			values["Usage ratio"] = "NaN"
		case "number_infinity":
			values["Usage ratio"] = "Infinity"
		case "number_comma":
			values["Usage ratio"] = "1,000.5"
		case "boolean_uppercase":
			values["Feature enabled"] = "True"
		case "boolean_numeric":
			values["Feature enabled"] = "1"
		case "empty":
			values["Item count"] = "   "
		}
	}
	return values
}

func writeScenarioHTML(writer http.ResponseWriter, title, body string) {
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprintf(writer, "<!doctype html><html lang=\"en\"><head><meta charset=\"utf-8\"><title>%s</title></head><body>%s</body></html>", html.EscapeString(title), body)
}

func allowedOTPChallenge(value string) bool {
	return value == "totp" || value == "sms_otp" || value == "email_otp" || value == "voice_otp"
}

func (fixture *LoopbackFixture) runtimeSession(request *http.Request) (string, loopbackSession, bool) {
	cookie, err := request.Cookie("openudon_scenario_session")
	if err != nil {
		return "", loopbackSession{}, false
	}
	fixture.mu.RLock()
	session, ok := fixture.sessions[cookie.Value]
	fixture.mu.RUnlock()
	return cookie.Value, session, ok
}

func (fixture *LoopbackFixture) requireRuntimeAuthentication(writer http.ResponseWriter, request *http.Request) bool {
	_, session, ok := fixture.runtimeSession(request)
	if !ok || !session.loginComplete || !session.challengeComplete {
		http.Error(writer, "authentication required", http.StatusUnauthorized)
		return false
	}
	fixture.mu.Lock()
	fixture.replay.protectedReads++
	fixture.mu.Unlock()
	return true
}

func newLoopbackSessionToken() (string, error) {
	var value [24]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(value[:]), nil
}

func validLoopbackChallenge(kind, value string, at time.Time) bool {
	value = strings.TrimSpace(value)
	if kind == "totp" {
		for offset := -1; offset <= 1; offset++ {
			if value == loopbackTOTP("JBSWY3DPEHPK3PXP", at.Add(time.Duration(offset)*30*time.Second)) {
				return true
			}
		}
		return false
	}
	if allowedOTPChallenge(kind) {
		return value == "123456"
	}
	return value == kind
}

func loopbackTOTP(seed string, at time.Time) string {
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(strings.TrimSpace(seed)))
	if err != nil {
		return ""
	}
	var counter [8]byte
	binary.BigEndian.PutUint64(counter[:], uint64(at.Unix()/30))
	digest := hmac.New(sha1.New, key)
	_, _ = digest.Write(counter[:])
	sum := digest.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	code := (binary.BigEndian.Uint32(sum[offset:offset+4]) & 0x7fffffff) % 1_000_000
	return fmt.Sprintf("%06d", code)
}
