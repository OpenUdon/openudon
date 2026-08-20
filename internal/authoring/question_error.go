package authoring

import (
	"errors"
	"strings"
)

// QuestionError attaches the authoritative frontier question to a rejected
// answer without changing the operator-safe error message.
type QuestionError struct {
	QuestionID string
	Cause      error
}

func (e *QuestionError) Error() string {
	if e == nil || e.Cause == nil {
		return "authoring question rejected"
	}
	return e.Cause.Error()
}

func (e *QuestionError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// WithQuestionID preserves the first authoritative question identity attached
// to err. Empty identities are ignored.
func WithQuestionID(questionID string, err error) error {
	questionID = strings.TrimSpace(questionID)
	if err == nil || questionID == "" {
		return err
	}
	var existing *QuestionError
	if errors.As(err, &existing) && strings.TrimSpace(existing.QuestionID) != "" {
		return err
	}
	return &QuestionError{QuestionID: questionID, Cause: err}
}

// QuestionID returns an attached authoritative frontier question identity.
func QuestionID(err error) string {
	var questionErr *QuestionError
	if errors.As(err, &questionErr) {
		return strings.TrimSpace(questionErr.QuestionID)
	}
	return ""
}
