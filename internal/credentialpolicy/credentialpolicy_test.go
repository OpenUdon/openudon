package credentialpolicy

import "testing"

func TestSecretShapesAndSymbolicGrammar(t *testing.T) {
	secrets := []string{
		"GOCSPX-aBcDeFgHiJkLmNoPqRsTuV", "1//0gAbCdEfGhIjKlMnOpQrStUvWxYz",
		"xoxb-" + "123456789012-abcdefghijklmnop", "xapp-" + "1-A1b2C3d4E5f6G7h8I9j0", "ghp_abcdefghijklmnopqrstuvwxyz1234567890AB",
		"AKIAIOSFODNN7EXAMPLE", "Bearer a8K2mP9qR4sT7vW1xY6z",
		"eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjMifQ.abcdefghijklmno",
		"m8Z-pQ4_R2x7N1cV9bK3sD6fH0jL5wT2",
		"Tr0ub4dor&3-Correct-Horse!", "nqazwsxedcrfvtgbyhnujmikolp",
	}
	for _, value := range secrets {
		if SafeMappingValue(value) {
			t.Errorf("secret was accepted: %s", value)
		}
	}
	for _, value := range []string{"inputs.channel", "credentials.slack", "get_message.received_body.text", "received_body.items[0]"} {
		if !IsSymbolicReference(value) || !SafeMappingValue(value) {
			t.Errorf("symbolic reference was rejected: %s", value)
		}
	}
	if IsSymbolicReference("dash-separated-secret-value") {
		t.Fatal("dash-separated literal was treated as symbolic")
	}
	for _, value := range []string{"variables.inputs.channel", "variables.credentials.slack"} {
		if IsSymbolicReference(value) {
			t.Errorf("undocumented generated-expression form was treated as an authoring reference: %s", value)
		}
	}
}

func TestArtifactAssignmentsDoNotExemptLiteralSourceOrURLValues(t *testing.T) {
	for _, artifact := range []string{
		`token_from = "m8Z-pQ4_R2x7N1cV9bK3sD6fH0jL5wT2"`,
		`password_url = "a8K2mP9qR4sT7vW1xY6z"`,
		`client_secret = "Tr0ub4dor&3-Correct-Horse!"`,
		`refresh_token = "nqazwsxedcrfvtgbyhnujmikolp"`,
		`{"client_secret":"m8Z-pQ4_R2x7N1cV9bK3sD6fH0jL5wT2"}`,
		"refresh-token: nqazwsxedcrfvtgbyhnujmikolp # unsafe YAML literal",
	} {
		if !ContainsLikelyValue([]byte(artifact)) {
			t.Errorf("literal credential assignment was accepted: %s", artifact)
		}
	}
	for _, artifact := range []string{
		`token_from = "get_message.received_body.token"`,
		`authorizationUrl: "https://authserver.example/auth"`,
	} {
		if ContainsLikelyValue([]byte(artifact)) {
			t.Errorf("safe symbolic or URL assignment was rejected: %s", artifact)
		}
	}
}

func TestBenignArtifactValues(t *testing.T) {
	for _, value := range []string{
		"inputs {\n  recipient_email = \"me@example.com\"\n}\n",
		"authorizationUrl: 'https://authserver.example/auth'",
		"appid = \"credentials.openWeatherAPIKey\"",
		`{"flow_credential_slots":{"member_login_push":["password","username"]}}`,
		`{"credential_bindings":{"declared":["audit_events_bearer_token"]}}`,
		`{"security":{"token_url":"https://oauth2.googleapis.com/token"}}`,
		"credentialSlots:\n  password: {kind: password}\n",
	} {
		if ContainsLikelyValue([]byte(value)) {
			t.Errorf("benign artifact was flagged: %s", value)
		}
	}
}
