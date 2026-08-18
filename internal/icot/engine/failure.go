package engine

import (
	"errors"
	"fmt"

	"github.com/OpenUdon/openudon/internal/icot/artifactwriter"
)

// FailureClass is the closed engine-to-driver error contract.
type FailureClass string

const (
	FailureRejected      FailureClass = "rejected"
	FailureConflict      FailureClass = "conflict"
	FailureOperational   FailureClass = "operational"
	FailureIndeterminate FailureClass = "indeterminate"
)

// Failure gives drivers a stable classification without requiring them to
// inspect filesystem or syscall implementation details.
type Failure struct {
	Class FailureClass
	Code  string
	Cause error
}

func (e *Failure) Error() string {
	if e == nil || e.Cause == nil {
		return "authoring engine failure"
	}
	return e.Cause.Error()
}

func (e *Failure) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// FailureDetails returns the stable class and code for an engine error.
// Untyped errors are treated as operational failures so transport adapters
// fail closed instead of guessing from concrete OS error types.
func FailureDetails(err error) (FailureClass, string) {
	var failure *Failure
	if errors.As(err, &failure) {
		return failure.Class, failure.Code
	}
	var transactionErr *artifactwriter.TransactionError
	if errors.As(err, &transactionErr) {
		return FailureIndeterminate, "transaction_indeterminate"
	}
	return FailureOperational, "engine_operation_failed"
}

func rejected(err error) error {
	return classified(FailureRejected, "engine_rejected", err)
}

func conflict(code string, err error) error {
	return classified(FailureConflict, code, err)
}

func operational(err error) error {
	return classified(FailureOperational, "engine_operation_failed", err)
}

func classifyCommit(err error) error {
	var transactionErr *artifactwriter.TransactionError
	if errors.As(err, &transactionErr) {
		return classified(FailureIndeterminate, "transaction_indeterminate", err)
	}
	return operational(err)
}

func classified(class FailureClass, code string, err error) error {
	if err == nil {
		err = fmt.Errorf("%s", code)
	}
	var existing *Failure
	if errors.As(err, &existing) {
		return err
	}
	return &Failure{Class: class, Code: code, Cause: err}
}
