package packagev3

import (
	"math"
	"strings"
	"testing"

	"github.com/OpenUdon/uws/uws1"
)

func TestBrowserAuthenticationTimeoutPredicate(t *testing.T) {
	for _, version := range []string{"1.0", "1.1"} {
		profile := "uws.browser-authentication-call." + version
		call := &BrowserCallV1{OperationID: "auth", Kind: "authentication", ProfileVersion: "uws.browser-authentication." + version}
		for _, c := range []struct {
			name    string
			timeout float64
			want    bool
		}{
			{"fractional", 0.5, true},
			{"ordinary", 30, true},
			{"inclusive maximum", 600, true},
			{"zero", 0, false},
			{"negative zero", math.Copysign(0, -1), false},
			{"negative", -1, false},
			{"above maximum", 600.0001, false},
			{"NaN", math.NaN(), false},
			{"positive infinity", math.Inf(1), false},
			{"negative infinity", math.Inf(-1), false},
		} {
			t.Run(version+"/"+c.name, func(t *testing.T) {
				op := &uws1.Operation{OperationID: "auth", Extensions: map[string]any{uws1.ExtensionOperationProfile: profile}}
				op.Timeout = &c.timeout
				if got := flatBrowserOperation(op, call); got != c.want {
					t.Fatalf("auth timeout %v admitted=%v, want %v", c.timeout, got, c.want)
				}
				if flatOperation(op) {
					t.Fatal("generic operation profile acquired timed authentication")
				}
				if op.Timeout != &c.timeout || math.Float64bits(*op.Timeout) != math.Float64bits(c.timeout) || op.ExtensionProfile() != profile {
					t.Fatal("predicate changed the authored operation")
				}
			})
		}
	}
	if flatBrowserOperation(nil, nil) {
		t.Fatal("nil operation admitted")
	}
	if !flatBrowserOperation(&uws1.Operation{}, nil) {
		t.Fatal("old untimed flat metadata refused")
	}
}

func TestBrowserAuthenticationTimeoutPredicateRequiresExactCall(t *testing.T) {
	timeout := 30.0
	for _, profile := range []string{
		"", "uws.browser-authentication-call.1.2", "UWS.browser-authentication-call.1.0",
		"uws.browser-authentication-call.1.0 ", "uws.browser-registration-call.1.0", "uws.runtime.1.0",
	} {
		t.Run(profile, func(t *testing.T) {
			op := &uws1.Operation{OperationID: "auth", Extensions: map[string]any{uws1.ExtensionOperationProfile: profile}}
			op.Timeout = &timeout
			call := &BrowserCallV1{OperationID: "auth", Kind: "authentication", ProfileVersion: strings.Replace(profile, "browser-authentication-call.", "browser-authentication.", 1)}
			if flatBrowserOperation(op, call) {
				t.Fatal("inexact authentication profile admitted")
			}
		})
	}
	op := &uws1.Operation{OperationID: "auth", Extensions: map[string]any{uws1.ExtensionOperationProfile: "uws.browser-authentication-call.1.0"}}
	op.Timeout = &timeout
	for _, call := range []*BrowserCallV1{
		nil,
		{OperationID: "auth", Kind: "action", ProfileVersion: "uws.browser-authentication.1.0"},
		{OperationID: "auth", Kind: "registration", ProfileVersion: "uws.browser-authentication.1.0"},
		{OperationID: "different", Kind: "authentication", ProfileVersion: "uws.browser-authentication.1.0"},
		{OperationID: "auth", Kind: "authentication", ProfileVersion: "uws.browser-authentication.1.1"},
	} {
		if flatBrowserOperation(op, call) {
			t.Fatal("timing admitted without matching authenticated browser provenance", call)
		}
	}
}

func TestBrowserAuthenticationTimeoutPredicateRetainsOtherFlatControls(t *testing.T) {
	for _, c := range []struct {
		name   string
		mutate func(*uws1.Operation)
	}{
		{"dependsOn", func(op *uws1.Operation) { op.DependsOn = []string{"prior"} }},
		{"when", func(op *uws1.Operation) { op.When = "$inputs.ready" }},
		{"forEach", func(op *uws1.Operation) { op.ForEach = "$inputs.items" }},
		{"wait", func(op *uws1.Operation) { op.Wait = "$inputs.ready" }},
		{"parallelGroup", func(op *uws1.Operation) { op.ParallelGroup = "parallel" }},
		{"onFailure", func(op *uws1.Operation) { op.OnFailure = []*uws1.FailureAction{{Type: "retry"}} }},
		{"onSuccess", func(op *uws1.Operation) { op.OnSuccess = []*uws1.SuccessAction{{Type: "goto"}} }},
	} {
		t.Run(c.name, func(t *testing.T) {
			timeout := 30.0
			op := &uws1.Operation{OperationID: "auth", Extensions: map[string]any{uws1.ExtensionOperationProfile: "uws.browser-authentication-call.1.0"}}
			op.Timeout = &timeout
			c.mutate(op)
			call := &BrowserCallV1{OperationID: "auth", Kind: "authentication", ProfileVersion: "uws.browser-authentication.1.0"}
			if flatBrowserOperation(op, call) {
				t.Fatal("authentication timeout bypassed another flat restriction")
			}
			op.Timeout = nil
			if flatBrowserOperation(op, call) || flatOperation(op) {
				t.Fatal("old untimed flat control restriction changed")
			}
		})
	}
}
