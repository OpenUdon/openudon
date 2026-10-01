package browserauthoring

import (
	"errors"
	"os"
	"strings"
)

func providerFromEnv() string {
	if os.Getenv("OPENUDON_LLM_PROVIDER") != "" {
		return strings.ToLower(strings.TrimSpace(os.Getenv("OPENUDON_LLM_PROVIDER")))
	}
	return "copilot-api"
}
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

type repeatedFlag []string

func (values *repeatedFlag) String() string { return strings.Join(*values, ",") }
func (values *repeatedFlag) Set(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return errors.New("flag value may not be empty")
	}
	*values = append(*values, value)
	return nil
}
