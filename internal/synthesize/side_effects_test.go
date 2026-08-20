package synthesize

import "testing"

func TestSideEffectVerbClassifier(t *testing.T) {
	for _, input := range []string{
		"create ticket", "created a record", "creates users", "creating deployment",
		"send email", "sends alert", "sent notification", "write file", "writes audit log",
		"update account", "updates status", "updating profile", "delete row", "deletes object",
		"deleting message", "deploy service", "POST /messages", "put /objects", "patch /issues",
	} {
		if !containsSideEffectVerb(input) {
			t.Errorf("containsSideEffectVerb(%q) = false", input)
		}
	}
	for _, input := range []string{"get customer", "read status", "list messages", "creator name", "updateable field"} {
		if containsSideEffectVerb(input) {
			t.Errorf("containsSideEffectVerb(%q) = true", input)
		}
	}
}

func TestOpenAPIMethodSideEffectClassifier(t *testing.T) {
	for _, method := range []string{"POST", "put", " Patch ", "DELETE"} {
		if !openAPIMethodIsSideEffectful(method) {
			t.Errorf("openAPIMethodIsSideEffectful(%q) = false", method)
		}
	}
	for _, method := range []string{"GET", "HEAD", "OPTIONS", "TRACE", ""} {
		if openAPIMethodIsSideEffectful(method) {
			t.Errorf("openAPIMethodIsSideEffectful(%q) = true", method)
		}
	}
}
