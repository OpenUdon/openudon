package credentialpolicy

import "testing"

func TestDeclaredBindingsAreRecognizedOnlyInBindingMaps(t *testing.T) {
	name := "w8m_dedicated_password"
	if !IsLikelyLiteral(name) {
		t.Fatal("fixture no longer reaches entropy heuristic")
	}
	for _, data := range []string{
		`{"credential_bindings":{"password":"w8m_dedicated_password"}}`,
		`{"x-uws-browser-registration":{"credentialBindings":{"password":"w8m_dedicated_password"}}}`,
		"credentialBindings:\n  password: w8m_dedicated_password\n",
		"credential_bindings = { password = \"w8m_dedicated_password\" }\n",
		"credentialBindings {\n password = \"w8m_dedicated_password\"\n}\n",
		"step \"register\" {\n call = { credentialBindings = { password = \"w8m_dedicated_password\" } }\n}\n",
	} {
		wasFlagged := ContainsLikelyValue([]byte(data))
		if ContainsArtifactValue([]byte(data), []string{name}) {
			t.Errorf("declared symbolic binding rejected: %s", data)
		}
		if wasFlagged && !ContainsArtifactValue([]byte(data), nil) {
			t.Fatal("undeclared binding escaped scan")
		}
	}
	for _, data := range []string{
		`{"password":"w8m_dedicated_password"}`,
		`{"credential_bindings":{"password":"m8Z-pQ4_R2x7N1cV9bK3sD6fH0jL5wT2"}}`,
		"credential_bindings = { password = \"w8m_dedicated_password\" }\n# password = \"m8Z-pQ4_R2x7N1cV9bK3sD6fH0jL5wT2\"\n",
		"credentialBindings:\n  password: w8m_dedicated_password\nrequest:\n  password: w8m_dedicated_password\n",
		`{"credentialBindings":{"password":"ghp_abcdefghijklmnopqrstuvwxyz1234567890AB"}}`,
		"credential_bindings = { password = true ? \"w8m_dedicated_password\" : \"ghp_abcdefghijklmnopqrstuvwxyz1234567890AB\" }\n",
		"credentialBindings { password = true ? \"w8m_dedicated_password\" : \"ghp_abcdefghijklmnopqrstuvwxyz1234567890AB\" }\n",
	} {
		if !ContainsArtifactValue([]byte(data), []string{name, "ghp_abcdefghijklmnopqrstuvwxyz1234567890AB"}) {
			t.Errorf("literal credential escaped scan: %s", data)
		}
	}
}
