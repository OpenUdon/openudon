package browsercapture

import (
	"bytes"
	"errors"
	"sync"
	"time"

	"github.com/OpenUdon/browsertools/registrationauthorsession"
	"github.com/OpenUdon/openudon/docs/schemas"
	"github.com/OpenUdon/openudon/internal/browserauthor"
	"github.com/OpenUdon/openudon/internal/browserauthoring"
	"github.com/OpenUdon/openudon/internal/evidencefile"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

const StartVersion = "openudon.browser-capture-start.v1"

// StartRequest is a reviewed local file. Private transport paths stay CLI
// options; credential values, model configuration and unknown fields are absent.
type StartRequest struct {
	Version             string               `json:"version"`
	Mode                string               `json:"mode"`
	OperatorIdleSeconds int                  `json:"operator_idle_seconds,omitempty"`
	AbsoluteSeconds     int                  `json:"absolute_seconds,omitempty"`
	Authentication      *AuthenticationStart `json:"authentication,omitempty"`
	Registration        *RegistrationStart   `json:"registration,omitempty"`
}
type AuthenticationStart struct {
	ProfileID           string   `json:"profile_id"`
	URL                 string   `json:"url"`
	DashboardURL        string   `json:"dashboard_url"`
	GoalURL             string   `json:"goal_url"`
	Goal                string   `json:"goal"`
	Origins             []string `json:"origins"`
	GoalRole            string   `json:"goal_role"`
	GoalContext         string   `json:"goal_context"`
	GoalLabel           string   `json:"goal_label,omitempty"`
	AfterAuthentication string   `json:"after_authentication"`
	BlockedScriptOrigin string   `json:"blocked_script_origin,omitempty"`
	Diagnostic          bool     `json:"diagnostic,omitempty"`
}
type RegistrationStart struct {
	Protocol      string                            `json:"protocol,omitempty"`
	ProfileID     string                            `json:"profile_id"`
	URL           string                            `json:"url"`
	Origins       []string                          `json:"origins"`
	TransactionID string                            `json:"transaction_id"`
	Bounds        *registrationauthorsession.Bounds `json:"bounds,omitempty"`
}

var startSchema = sync.OnceValues(func() (*jsonschema.Schema, error) {
	compiler := jsonschema.NewCompiler()
	for _, version := range []string{Version, StartVersion} {
		data, err := schemas.BrowserCaptureResources.ReadFile(version + ".schema.json")
		if err != nil {
			return nil, err
		}
		document, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
		if err := compiler.AddResource("https://openudon.dev/schemas/"+version+".schema.json", document); err != nil {
			return nil, err
		}
	}
	return compiler.Compile("https://openudon.dev/schemas/" + StartVersion + ".schema.json")
})

func DecodeStart(data []byte) (StartRequest, error) {
	var request StartRequest
	invalid := errors.New("browser capture start invalid")
	if len(data) == 0 || len(data) > MaxMessageBytes || evidencefile.DecodeStrict(data, &request) != nil {
		return StartRequest{}, invalid
	}
	schema, err := startSchema()
	if err != nil {
		return StartRequest{}, errors.New("browser capture start schema unavailable")
	}
	instance, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err != nil || schema.Validate(instance) != nil {
		return StartRequest{}, invalid
	}
	return request, nil
}

// AuthenticationConfig reuses the retained config/goal policy. The live config
// is kept process-local for independent attested-result reconstruction later.
func (request StartRequest) AuthenticationConfig(example, private, driver string) (browserauthor.Config, browserauthoring.LiveConfig, error) {
	if request.Mode != Authenticated || request.Authentication == nil || request.Registration != nil {
		return browserauthor.Config{}, browserauthoring.LiveConfig{}, errors.New("authenticated start required")
	}
	auth := request.Authentication
	live := browserauthoring.LiveConfig{ExampleDir: example, PrivateRoot: private, DriverDir: driver, URL: auth.URL, DashboardURL: auth.DashboardURL, GoalURL: auth.GoalURL, Goal: auth.Goal,
		Origins: append([]string(nil), auth.Origins...), ProfileID: auth.ProfileID, AfterAuthentication: auth.AfterAuthentication, GoalRole: auth.GoalRole, GoalLabel: auth.GoalLabel, GoalContext: auth.GoalContext, NoLLM: true}
	if err := browserauthoring.NormalizeLiveConfig(&live); err != nil {
		return browserauthor.Config{}, browserauthoring.LiveConfig{}, errors.New("authenticated authority invalid")
	}
	config, err := browserauthoring.ControllerConfig(live)
	if err != nil {
		return browserauthor.Config{}, browserauthoring.LiveConfig{}, errors.New("authenticated goal invalid")
	}
	config.BlockedScriptOrigin, config.Diagnostic = auth.BlockedScriptOrigin, auth.Diagnostic
	config.Absolute, config.OperatorIdle = request.timeouts()
	config, err = browserauthor.NormalizeConfig(config)
	if err != nil {
		return browserauthor.Config{}, browserauthoring.LiveConfig{}, errors.New("authenticated authority invalid")
	}
	return config, live, nil
}

func (request StartRequest) RegistrationConfig(private, driver string) (browserauthor.RegistrationConfig, browserauthor.RegistrationCommand, error) {
	if request.Mode != Registration || request.Registration == nil || request.Authentication != nil {
		return browserauthor.RegistrationConfig{}, browserauthor.RegistrationCommand{}, errors.New("registration start required")
	}
	reg := request.Registration
	protocol := reg.Protocol
	if protocol == "" {
		protocol = registrationauthorsession.ProtocolV4
	}
	absolute, idle := request.timeouts()
	config, err := browserauthor.NormalizeRegistrationConfig(browserauthor.RegistrationConfig{PrivateRoot: private, DriverDir: driver, TransactionID: reg.TransactionID, Protocol: protocol, Absolute: absolute, OperatorIdle: idle})
	if err != nil {
		return browserauthor.RegistrationConfig{}, browserauthor.RegistrationCommand{}, errors.New("registration authority invalid")
	}
	start, err := browserauthor.NormalizeRegistrationStart(config.Protocol, browserauthor.RegistrationCommand{Type: "start", ProfileID: reg.ProfileID, URL: reg.URL, Origins: reg.Origins, Bounds: reg.Bounds})
	if err != nil {
		return browserauthor.RegistrationConfig{}, browserauthor.RegistrationCommand{}, errors.New("registration authority invalid")
	}
	return config, start, nil
}

func (request StartRequest) timeouts() (time.Duration, time.Duration) {
	absolute, idle := browserauthor.DefaultAbsolute, browserauthor.DefaultOperatorIdle
	if request.AbsoluteSeconds != 0 {
		absolute = time.Duration(request.AbsoluteSeconds) * time.Second
	}
	if request.OperatorIdleSeconds != 0 {
		idle = time.Duration(request.OperatorIdleSeconds) * time.Second
	}
	return absolute, idle
}
