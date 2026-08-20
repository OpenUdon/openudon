package authoring

import (
	"errors"
	"testing"
)

func TestQuestionErrorPreservesCauseAndFirstIdentity(t *testing.T) {
	cause := errors.New("invalid answer")
	err := WithQuestionID("question.one", cause)
	if !errors.Is(err, cause) || err.Error() != cause.Error() || QuestionID(err) != "question.one" {
		t.Fatalf("question error = %v id=%q", err, QuestionID(err))
	}
	wrapped := WithQuestionID("question.two", err)
	if QuestionID(wrapped) != "question.one" {
		t.Fatalf("nested question id = %q", QuestionID(wrapped))
	}
	if WithQuestionID("", cause) != cause || WithQuestionID("question", nil) != nil || QuestionID(cause) != "" {
		t.Fatal("empty question wrapping changed the error contract")
	}
}
