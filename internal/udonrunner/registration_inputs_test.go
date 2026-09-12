package udonrunner

import (
	"strings"
	"testing"
)

func TestV5PreparedFormRequiresPrivateExpectationAndRejectsLegacyMixing(t *testing.T) {
	valid := func() *BrowserConfig {
		return &BrowserConfig{Protocol: "v5", DriverPath: "/trusted/browserdriver", RegistrationInputService: "http://127.0.0.1:12345/input/synthetic/", RegistrationInputTokenEnv: "UDON_REGISTRATION_INPUT_TOKEN", RegistrationInputExpectedEnv: "UDON_REGISTRATION_INPUT_EXPECTED", ApprovedRegistration: []string{"register_member"}, AttestedRegistration: []string{"register_member"}, RegistrationAttestationSHA256: "sha256:" + strings.Repeat("a", 64)}
	}
	values := map[string]string{"UDON_REGISTRATION_INPUT_TOKEN": "synthetic-runtime-capability", "UDON_REGISTRATION_INPUT_EXPECTED": `{"inputRevision":1,"inputSha256":"` + strings.Repeat("b", 64) + `"}`}
	for _, change := range []string{"valid", "legacy", "credential", "missing_expected", "wrong_expected", "mixed_ui", "external_url", "token_in_url", "query"} {
		t.Run(change, func(t *testing.T) {
			config := valid()
			env := map[string]string{}
			for k, v := range values {
				env[k] = v
			}
			switch change {
			case "legacy":
				config.Protocol = "v4"
			case "credential":
				config.CredentialEnvironment = []EnvironmentBinding{{Name: "member_id", Environment: "UDON_CREDENTIAL_MEMBER_ID"}}
			case "missing_expected":
				delete(env, "UDON_REGISTRATION_INPUT_EXPECTED")
			case "wrong_expected":
				config.RegistrationInputExpectedEnv = "OTHER"
			case "mixed_ui":
				config.RegistrationInputUI = true
			case "external_url":
				config.RegistrationInputService = "https://example.test/input/private/"
			case "token_in_url":
				config.RegistrationInputService = "http://private@127.0.0.1:12345/input/private/"
			case "query":
				config.RegistrationInputService += "?secret=private"
			}
			_, err := validateBrowserConfig(config, []string{"member_id"}, env, true, true)
			if (err == nil) != (change == "valid") {
				t.Fatal("incorrect private handoff acceptance")
			}
			if err != nil && (strings.Contains(err.Error(), values["UDON_REGISTRATION_INPUT_TOKEN"]) || strings.Contains(err.Error(), values["UDON_REGISTRATION_INPUT_EXPECTED"])) {
				t.Fatal("private runtime value in error")
			}
		})
	}
	config := valid()
	if err := ValidateBrowserEvidenceConfig(config, []string{"member_id"}); err != nil {
		t.Fatal("offline evidence required private values")
	}
}
