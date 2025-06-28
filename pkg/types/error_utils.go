package types

import (
	"errors"
	"fmt"
	"runtime"
	"time"
)

// WrapError adds context to an error
func WrapError(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	msg := fmt.Sprintf(format, args...)
	return fmt.Errorf("%s: %w", msg, err)
}

// ErrorInfo provides detailed error context
type ErrorInfo struct {
	Code      ErrorCode
	Message   string
	Details   map[string]interface{}
	Timestamp time.Time
	Stack     []byte
}

// ErrorCode identifies specific error conditions
type ErrorCode int

const (
	// CodeUnknown indicates an unknown error
	CodeUnknown ErrorCode = iota
	// CodeCorruption indicates data corruption
	CodeCorruption
	// CodeNotFound indicates something was not found
	CodeNotFound
	// CodeInvalidInput indicates invalid input
	CodeInvalidInput
	// CodeStorageFailure indicates storage operation failed
	CodeStorageFailure
	// CodeResourceExhausted indicates resource limits exceeded
	CodeResourceExhausted
	// CodeTransactionConflict indicates transaction conflict
	CodeTransactionConflict
)

// NewErrorInfo creates detailed error information
func NewErrorInfo(code ErrorCode, err error, details map[string]interface{}) *ErrorInfo {
	stack := make([]byte, 4096)
	n := runtime.Stack(stack, false)

	return &ErrorInfo{
		Code:      code,
		Message:   err.Error(),
		Details:   details,
		Timestamp: time.Now(),
		Stack:     stack[:n],
	}
}

// Error implements the error interface
func (e *ErrorInfo) Error() string {
	return e.Message
}

// IsRetryable determines if an error can be retried
func IsRetryable(err error) bool {
	return errors.Is(err, ErrStorageFailure) ||
		errors.Is(err, ErrTransactionAborted) ||
		errors.Is(err, ErrTxConflict)
}

// GetErrorCode returns the appropriate ErrorCode for a given error
func GetErrorCode(err error) ErrorCode {
	switch {
	case errors.Is(err, ErrCorruptedNode),
		errors.Is(err, ErrInvalidNodeType),
		errors.Is(err, ErrHashMismatch):
		return CodeCorruption

	case errors.Is(err, ErrKeyNotFound),
		errors.Is(err, ErrVersionNotFound),
		errors.Is(err, ErrNodeNotFound):
		return CodeNotFound

	case errors.Is(err, ErrInvalidKey),
		errors.Is(err, ErrEmptyKey),
		errors.Is(err, ErrInvalidVersion),
		errors.Is(err, ErrValueTooLarge),
		errors.Is(err, ErrInvalidNibble),
		errors.Is(err, ErrInvalidNodeKey),
		errors.Is(err, ErrInvalidNibbleCount):
		return CodeInvalidInput

	case errors.Is(err, ErrStorageFailure),
		errors.Is(err, ErrTransactionAborted),
		errors.Is(err, ErrDatabaseClosed):
		return CodeStorageFailure

	case errors.Is(err, ErrMaxDepthExceeded),
		errors.Is(err, ErrBatchTooLarge),
		errors.Is(err, ErrOutOfMemory):
		return CodeResourceExhausted

	case errors.Is(err, ErrTxConflict),
		errors.Is(err, ErrTxStateInvalid):
		return CodeTransactionConflict

	default:
		return CodeUnknown
	}
}
