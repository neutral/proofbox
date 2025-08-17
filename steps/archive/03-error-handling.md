---
id: step.03.error-handling
depends_on:
  - step.02.hasher
tags: [foundation, error-handling, step]
---

## Objective

Implement core error handling framework with error types, utilities, and interfaces based on error-handling.md specification. This establishes the foundation for consistent error management, with concrete handler implementations deferred to step 8 when the tree structure is available.

## Implements

- **Error Handling Specification** (`../../pkg/_blueprint/_specs/error-handling.md`) - Core types and interfaces only
- Establishes error categories and utilities
- Defines interfaces for recovery strategies (implementations in step 8)
- Provides foundation for reliable operation in production environments

## Technical Details

### Error Categories Implementation

Create `pkg/types/errors.go`:
```go
package types

import (
    "errors"
    "fmt"
)

// Error categories
var (
    // Data Integrity Errors
    ErrCorruptedNode   = errors.New("jmt: corrupted node detected")
    ErrInvalidNodeType = errors.New("jmt: invalid node type")
    ErrHashMismatch    = errors.New("jmt: hash mismatch")
    ErrInvalidProof    = errors.New("jmt: invalid proof")
    
    // Operational Errors
    ErrKeyNotFound     = errors.New("jmt: key not found")
    ErrVersionNotFound = errors.New("jmt: version not found")
    ErrNodeNotFound    = errors.New("jmt: node not found")
    ErrEmptyTree       = errors.New("jmt: empty tree")
    
    // Input Validation Errors
    ErrInvalidKey         = errors.New("jmt: invalid key")
    ErrEmptyKey          = errors.New("jmt: empty key not allowed")
    ErrInvalidVersion    = errors.New("jmt: invalid version")
    ErrValueTooLarge     = errors.New("jmt: value too large")
    ErrInvalidNibble     = errors.New("jmt: invalid nibble value")
    ErrInvalidNodeKey    = errors.New("jmt: invalid node key")
    ErrInvalidNibbleCount = errors.New("jmt: nibble count exceeds maximum")
    
    // Resource Errors
    ErrMaxDepthExceeded   = errors.New("jmt: maximum tree depth exceeded")
    ErrBatchTooLarge      = errors.New("jmt: batch size exceeds limit")
    ErrOutOfMemory        = errors.New("jmt: out of memory")
    
    // Storage Errors
    ErrStorageFailure     = errors.New("jmt: storage operation failed")
    ErrTransactionAborted = errors.New("jmt: transaction aborted")
    ErrDatabaseClosed     = errors.New("jmt: database closed")
    
    // Transaction Errors
    ErrTxConflict      = errors.New("jmt: transaction conflict")
    ErrTxStateInvalid  = errors.New("jmt: invalid transaction state")
)
```

### Error Wrapping Utilities

Create `pkg/types/error_utils.go`:
```go
package types

import (
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
    CodeUnknown ErrorCode = iota
    CodeCorruption
    CodeNotFound
    CodeInvalidInput
    CodeStorageFailure
    CodeResourceExhausted
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

// IsRetryable determines if an error can be retried
func IsRetryable(err error) bool {
    return errors.Is(err, ErrStorageFailure) ||
           errors.Is(err, ErrTransactionAborted) ||
           errors.Is(err, ErrTxConflict)
}
```

### Error Handler Interfaces

Create `pkg/types/error_interfaces.go`:
```go
package types

import (
    "context"
    "time"
)

// ReadErrorHandler defines recovery strategy for read errors
// Concrete implementations will be provided in step 8 with tree implementation
type ReadErrorHandler interface {
    HandleCorruptedNode(key NodeKey, err error) error
    HandleMissingNode(key NodeKey) error
}

// WriteErrorHandler defines recovery strategy for write errors
// Concrete implementations will be provided in step 8 with tree implementation
type WriteErrorHandler interface {
    HandleWriteFailure(err error) error
    ShouldRetry(err error) bool
    RetryDelay(attempt int) time.Duration
}

// ErrorReporter allows pluggable error metrics collection
type ErrorReporter interface {
    RecordError(code ErrorCode, err error)
    GetStats() map[ErrorCode]uint64
}

// HealthChecker defines interface for tree integrity verification
// Concrete implementation will be provided in step 8 with tree structure
type HealthChecker interface {
    VerifyIntegrity(ctx context.Context, version Version) error
}
```

Note: Concrete implementations of these interfaces (`DefaultReadErrorHandler`, `DefaultWriteErrorHandler`, and tree-specific `HealthChecker`) will be implemented in step 8 when the tree structure is available.

## Testing Requirements

### Unit Tests
```go
func TestErrorWrapping(t *testing.T) {
    base := types.ErrKeyNotFound
    wrapped := types.WrapError(base, "failed to find key %x", []byte{0x01})
    
    assert.True(t, errors.Is(wrapped, types.ErrKeyNotFound))
    assert.Contains(t, wrapped.Error(), "failed to find key 01")
}

func TestRetryableErrors(t *testing.T) {
    tests := []struct {
        err        error
        retryable  bool
    }{
        {types.ErrStorageFailure, true},
        {types.ErrTransactionAborted, true},
        {types.ErrTxConflict, true},
        {types.ErrKeyNotFound, false},
        {types.ErrInvalidKey, false},
    }
    
    for _, tc := range tests {
        assert.Equal(t, tc.retryable, types.IsRetryable(tc.err))
    }
}
```

### Error Info Tests
```go
func TestErrorInfo(t *testing.T) {
    err := types.ErrKeyNotFound
    info := types.NewErrorInfo(types.CodeNotFound, err, map[string]interface{}{
        "key": "test-key",
        "version": 42,
    })
    
    assert.Equal(t, types.CodeNotFound, info.Code)
    assert.Equal(t, "jmt: key not found", info.Message)
    assert.Equal(t, "test-key", info.Details["key"])
    assert.NotNil(t, info.Stack)
    assert.NotZero(t, info.Timestamp)
}
```

## Implementation Steps

1. Create error type definitions in `pkg/types/errors.go`
2. Implement error wrapping utilities in `pkg/types/error_utils.go`
3. Define error handler interfaces in `pkg/types/error_interfaces.go`
4. Write comprehensive tests for error types and utilities
5. Create .desc.md documentation files
6. Update existing code to use new error types (if any)

## Performance Considerations

- Error creation should be lightweight
- Avoid stack trace collection in hot paths
- Cache error messages for common cases
- Use error variables instead of creating new errors

## Security Notes

- Don't expose internal details in error messages
- Sanitize error data before logging
- Rate limit error responses to prevent DoS
- Log security-relevant errors for audit

## Done When ✓

- [x] All error types defined according to specification
- [x] Error wrapping utilities implemented
- [x] Error handler interfaces defined
- [x] ErrorReporter interface for metrics collection
- [x] HealthChecker interface defined
- [x] Comprehensive test coverage for error types and utilities
- [x] .desc.md documentation files created
- [x] All imports use github.com/neutral/proofbox
- [x] Tests passing and linting clean
