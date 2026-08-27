package workflowintent

import (
	"reflect"
	"strings"
	"testing"
)

const contentTrustIntentFixture = `
source = "browser/mail.yaml"

workflow {
  name = "Summarize mail"
}

content_trust {
  source_description "browser/mail.yaml" {
    level = "untrusted"
  }

  operation "read_message" {
    default = "untrusted"
    outputs = {
      body       = "untrusted"
      message_id = "trusted"
    }
  }

  trigger "incoming_mail" {
    level = "untrusted"
  }

  workflow "main" {
    default = "unknown"
    inputs = {
      locale = "trusted"
    }
  }
}

input "locale" {
  type = "string"
}

trigger "incoming_mail" {
  path = "/mail"
}

step "read_message" {
  type      = "browser"
  operation = "read_message"
}
`

func TestContentTrustIntentRoundTrip(t *testing.T) {
	intent, err := ParseIntent([]byte(contentTrustIntentFixture), IntentPath)
	if err != nil {
		t.Fatal(err)
	}
	if intent.ContentTrust == nil || len(intent.ContentTrust.SourceDescriptions) != 1 || len(intent.ContentTrust.Operations) != 1 || len(intent.ContentTrust.Triggers) != 1 || len(intent.ContentTrust.Workflows) != 1 {
		t.Fatalf("content trust = %#v", intent.ContentTrust)
	}
	rendered, err := RenderIntentHCL(intent)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(rendered, `source_description "browser/mail.yaml"`) || !strings.Contains(rendered, `message_id = "trusted"`) {
		t.Fatalf("rendered intent omitted declarations:\n%s", rendered)
	}
	parsed, err := ParseIntent([]byte(rendered), IntentPath)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(parsed.ContentTrust, intent.ContentTrust) {
		t.Fatalf("round trip content trust = %#v, want %#v", parsed.ContentTrust, intent.ContentTrust)
	}
	clone, err := intent.Clone()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(clone.ContentTrust, intent.ContentTrust) {
		t.Fatalf("clone content trust = %#v, want %#v", clone.ContentTrust, intent.ContentTrust)
	}
}

func TestContentTrustIntentRejectsMalformedDeclarations(t *testing.T) {
	tests := []struct {
		name    string
		replace string
		with    string
		want    string
	}{
		{name: "empty", replace: strings.TrimSpace(contentTrustIntentFixture[strings.Index(contentTrustIntentFixture, "content_trust {"):strings.Index(contentTrustIntentFixture, "\n\ninput")]), with: "content_trust {}", want: "at least one declaration"},
		{name: "level", replace: `level = "untrusted"`, with: `level = "reviewed"`, want: "must be unknown, trusted, or untrusted"},
		{name: "source", replace: `source_description "browser/mail.yaml"`, with: `source_description "browser/missing.yaml"`, want: "references an undeclared source"},
		{name: "operation", replace: `operation "read_message"`, with: `operation "missing"`, want: "references an undeclared leaf step"},
		{name: "trigger", replace: "trigger \"incoming_mail\" {\n    level", with: "trigger \"missing\" {\n    level", want: "references an undeclared trigger"},
		{name: "workflow", replace: `workflow "main"`, with: `workflow "nested"`, want: "only main is generated"},
		{name: "input", replace: `locale = "trusted"`, with: `missing = "trusted"`, want: "references undeclared input"},
		{name: "operation no-op", replace: "    default = \"untrusted\"\n    outputs = {\n      body       = \"untrusted\"\n      message_id = \"trusted\"\n    }", with: "", want: "must declare default or outputs"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := strings.Replace(contentTrustIntentFixture, tt.replace, tt.with, 1)
			_, err := ParseIntent([]byte(data), IntentPath)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("ParseIntent error = %v, want %q", err, tt.want)
			}
		})
	}
}
