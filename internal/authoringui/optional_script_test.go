package ui

import (
	"context"
	"io"
	"strings"
	"testing"
)

func TestOptionalScriptPolicyCannotEnterRegistrationOrLoopbackUI(t *testing.T) {
	config := RunConfig{CaptureBlockedScriptOrigin: "https://analytics.example.test"}
	if Run(context.Background(), config) == nil {
		t.Fatal("HTTP UI accepted private policy")
	}
	if RunRegistrationControl(context.Background(), config, io.NopCloser(strings.NewReader(""))) == nil {
		t.Fatal("registration accepted authentication policy")
	}
}

func TestOptionalScriptPolicyRejectsMalformedHandlerConfiguration(t *testing.T) {
	fake := &fakeEngine{}
	_, err := NewHandler(HandlerConfig{Engine: fake, Snapshot: fake.snapshot, ExampleDir: "/tmp/example", Token: testToken, AccessCode: testAccessCode, Authority: testAuthority, CaptureBlockedScriptOrigin: "https://user:secret-token-canary@example.test"})
	if err == nil || strings.Contains(err.Error(), "secret-token-canary") {
		t.Fatal("invalid policy or unsafe error")
	}
}
