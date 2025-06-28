package types

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestErrorWrapping(t *testing.T) {
	base := ErrKeyNotFound
	wrapped := WrapError(base, "failed to find key %x", []byte{0x01})

	// Check error message contains context
	if !strings.Contains(wrapped.Error(), "failed to find key 01") {
		t.Errorf("Wrapped error message incorrect: %v", wrapped)
	}

	// Check original error is preserved
	if !errors.Is(wrapped, ErrKeyNotFound) {
		t.Error("Wrapped error should preserve original error type")
	}

	// Test nil error handling
	if WrapError(nil, "test") != nil {
		t.Error("WrapError should return nil for nil error")
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		retryable bool
	}{
		{"StorageFailure", ErrStorageFailure, true},
		{"TransactionAborted", ErrTransactionAborted, true},
		{"TxConflict", ErrTxConflict, true},
		{"KeyNotFound", ErrKeyNotFound, false},
		{"InvalidKey", ErrInvalidKey, false},
		{"CorruptedNode", ErrCorruptedNode, false},
		{"HashMismatch", ErrHashMismatch, false},
		{"Wrapped retryable", WrapError(ErrStorageFailure, "context"), true},
		{"Wrapped non-retryable", WrapError(ErrKeyNotFound, "context"), false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if IsRetryable(tc.err) != tc.retryable {
				t.Errorf("IsRetryable(%v) = %v, want %v", tc.err, !tc.retryable, tc.retryable)
			}
		})
	}
}

func TestErrorInfo(t *testing.T) {
	err := ErrKeyNotFound
	details := map[string]interface{}{
		"key":     "test-key",
		"version": 42,
	}

	info := NewErrorInfo(CodeNotFound, err, details)

	// Check basic fields
	if info.Code != CodeNotFound {
		t.Errorf("ErrorInfo.Code = %v, want %v", info.Code, CodeNotFound)
	}

	if info.Message != "jmt: key not found" {
		t.Errorf("ErrorInfo.Message = %v, want %v", info.Message, "jmt: key not found")
	}

	// Check details
	if info.Details["key"] != "test-key" {
		t.Errorf("ErrorInfo.Details[key] = %v, want %v", info.Details["key"], "test-key")
	}

	// Check stack trace exists
	if len(info.Stack) == 0 {
		t.Error("ErrorInfo should include stack trace")
	}

	// Check timestamp is recent
	if time.Since(info.Timestamp) > time.Second {
		t.Error("ErrorInfo timestamp should be recent")
	}

	// Check Error() method
	if info.Error() != info.Message {
		t.Errorf("ErrorInfo.Error() = %v, want %v", info.Error(), info.Message)
	}
}

func TestGetErrorCode(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode ErrorCode
	}{
		// Corruption errors
		{"CorruptedNode", ErrCorruptedNode, CodeCorruption},
		{"InvalidNodeType", ErrInvalidNodeType, CodeCorruption},
		{"HashMismatch", ErrHashMismatch, CodeCorruption},

		// Not found errors
		{"KeyNotFound", ErrKeyNotFound, CodeNotFound},
		{"VersionNotFound", ErrVersionNotFound, CodeNotFound},
		{"NodeNotFound", ErrNodeNotFound, CodeNotFound},

		// Invalid input errors
		{"InvalidKey", ErrInvalidKey, CodeInvalidInput},
		{"EmptyKey", ErrEmptyKey, CodeInvalidInput},
		{"InvalidVersion", ErrInvalidVersion, CodeInvalidInput},
		{"ValueTooLarge", ErrValueTooLarge, CodeInvalidInput},
		{"InvalidNibble", ErrInvalidNibble, CodeInvalidInput},
		{"InvalidNodeKey", ErrInvalidNodeKey, CodeInvalidInput},
		{"InvalidNibbleCount", ErrInvalidNibbleCount, CodeInvalidInput},

		// Storage errors
		{"StorageFailure", ErrStorageFailure, CodeStorageFailure},
		{"TransactionAborted", ErrTransactionAborted, CodeStorageFailure},
		{"DatabaseClosed", ErrDatabaseClosed, CodeStorageFailure},

		// Resource errors
		{"MaxDepthExceeded", ErrMaxDepthExceeded, CodeResourceExhausted},
		{"BatchTooLarge", ErrBatchTooLarge, CodeResourceExhausted},
		{"OutOfMemory", ErrOutOfMemory, CodeResourceExhausted},

		// Transaction errors
		{"TxConflict", ErrTxConflict, CodeTransactionConflict},
		{"TxStateInvalid", ErrTxStateInvalid, CodeTransactionConflict},

		// Unknown error
		{"UnknownError", errors.New("unknown"), CodeUnknown},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			code := GetErrorCode(tc.err)
			if code != tc.wantCode {
				t.Errorf("GetErrorCode(%v) = %v, want %v", tc.err, code, tc.wantCode)
			}
		})
	}
}

func TestErrorCategories(t *testing.T) {
	// Test that all errors have unique messages
	errorMessages := make(map[string]error)
	allErrors := []error{
		// Hash errors
		ErrInvalidHashSize,

		// Data Integrity
		ErrCorruptedNode, ErrInvalidNodeType, ErrHashMismatch, ErrInvalidProof,

		// Operational
		ErrKeyNotFound, ErrVersionNotFound, ErrNodeNotFound, ErrEmptyTree,

		// Input Validation
		ErrInvalidKey, ErrEmptyKey, ErrInvalidVersion, ErrValueTooLarge,
		ErrInvalidNibble, ErrInvalidNodeKey, ErrInvalidNibbleCount,

		// Resource
		ErrMaxDepthExceeded, ErrBatchTooLarge, ErrOutOfMemory,

		// Storage
		ErrStorageFailure, ErrTransactionAborted, ErrDatabaseClosed,

		// Transaction
		ErrTxConflict, ErrTxStateInvalid,
	}

	for _, err := range allErrors {
		msg := err.Error()
		if existing, found := errorMessages[msg]; found {
			t.Errorf("Duplicate error message %q for errors %v and %v", msg, existing, err)
		}
		errorMessages[msg] = err
	}

	// Test that JMT errors have consistent prefix
	for _, err := range allErrors {
		if err == ErrInvalidHashSize {
			continue // This is a hash-specific error, not JMT
		}
		if !strings.HasPrefix(err.Error(), "jmt:") {
			t.Errorf("Error %v should have 'jmt:' prefix, got: %v", err, err.Error())
		}
	}
}
