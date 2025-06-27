---
id: step.03.error-handling
depends_on:
  - step.02.hasher
tags: [foundation, error-handling, step]
---

## Objective

Implement comprehensive error handling framework based on error-handling.md specification to ensure consistent error management across the entire JMT implementation.

## Implements

- **Error Handling Specification** (`/blueprint/global/specs/error-handling.md`)
- Establishes error categories, recovery strategies, and consistency guarantees
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

### Recovery Handlers

Create `pkg/tree/error_handlers.go`:
```go
package tree

import (
    "log"
    "time"
    
    "github.com/acme/jmt/pkg/types"
)

// ReadErrorHandler defines recovery strategy for read errors
type ReadErrorHandler interface {
    HandleCorruptedNode(key types.NodeKey, err error) error
    HandleMissingNode(key types.NodeKey) error
}

// DefaultReadErrorHandler implements basic recovery
type DefaultReadErrorHandler struct {
    logger *log.Logger
}

func NewDefaultReadErrorHandler(logger *log.Logger) *DefaultReadErrorHandler {
    return &DefaultReadErrorHandler{logger: logger}
}

func (h *DefaultReadErrorHandler) HandleCorruptedNode(key types.NodeKey, err error) error {
    h.logger.Printf("ERROR: Corrupted node detected at version=%d path=%x: %v",
        key.Version, key.NibblePath.Nibbles, err)
    
    // Don't try to recover - return error to caller
    return types.ErrCorruptedNode
}

func (h *DefaultReadErrorHandler) HandleMissingNode(key types.NodeKey) error {
    // Missing nodes might be due to pruning
    h.logger.Printf("WARN: Missing node at version=%d path=%x",
        key.Version, key.NibblePath.Nibbles)
    return types.ErrNodeNotFound
}

// WriteErrorHandler defines recovery strategy for write errors
type WriteErrorHandler interface {
    HandleWriteFailure(err error) error
    ShouldRetry(err error) bool
    RetryDelay(attempt int) time.Duration
}

// DefaultWriteErrorHandler implements exponential backoff
type DefaultWriteErrorHandler struct {
    maxRetries int
    baseDelay  time.Duration
    logger     *log.Logger
}

func NewDefaultWriteErrorHandler(maxRetries int, baseDelay time.Duration, logger *log.Logger) *DefaultWriteErrorHandler {
    return &DefaultWriteErrorHandler{
        maxRetries: maxRetries,
        baseDelay:  baseDelay,
        logger:     logger,
    }
}

func (h *DefaultWriteErrorHandler) HandleWriteFailure(err error) error {
    h.logger.Printf("ERROR: Write failure: %v", err)
    return types.WrapError(err, "write operation failed")
}

func (h *DefaultWriteErrorHandler) ShouldRetry(err error) bool {
    return types.IsRetryable(err)
}

func (h *DefaultWriteErrorHandler) RetryDelay(attempt int) time.Duration {
    // Exponential backoff with jitter
    delay := h.baseDelay * time.Duration(1<<uint(attempt))
    // Add up to 25% jitter
    jitter := time.Duration(float64(delay) * 0.25 * rand.Float64())
    return delay + jitter
}
```

### Health Check Implementation

Create `pkg/tree/health.go`:
```go
package tree

import (
    "bytes"
    "context"
    
    "github.com/acme/jmt/pkg/types"
)

// HealthChecker verifies tree integrity
type HealthChecker struct {
    tree *Tree
}

// VerifyIntegrity performs comprehensive integrity check
func (hc *HealthChecker) VerifyIntegrity(ctx context.Context, version types.Version) error {
    // Start from root
    rootKey := types.RootNodeKey(version)
    visited := make(map[types.NodeKey]bool)
    
    return hc.verifyNode(ctx, rootKey, visited)
}

func (hc *HealthChecker) verifyNode(ctx context.Context, key types.NodeKey, visited map[types.NodeKey]bool) error {
    // Check context cancellation
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }
    
    // Avoid cycles
    if visited[key] {
        return nil
    }
    visited[key] = true
    
    // Load node
    data, err := hc.tree.storage.Get(key.StorageKey())
    if err != nil {
        return types.WrapError(err, "failed to load node at %s", key)
    }
    
    // Verify node integrity
    node, err := DecodeNode(data)
    if err != nil {
        return types.WrapError(err, "failed to decode node at %s", key)
    }
    
    // Verify hash matches content
    computed := node.Hash()
    stored := node.GetStoredHash()
    
    if !bytes.Equal(computed[:], stored[:]) {
        return types.NewErrorInfo(types.CodeCorruption, types.ErrHashMismatch, map[string]interface{}{
            "node_key": key,
            "computed": computed.String(),
            "stored":   stored.String(),
        })
    }
    
    // Recursively verify children
    if internal, ok := node.(*InternalNode); ok {
        for nibble, child := range internal.Children {
            childKey := key.Child(nibble, child.Version)
            if err := hc.verifyNode(ctx, childKey, visited); err != nil {
                return err
            }
        }
    }
    
    return nil
}
```

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

### Recovery Tests
```go
func TestWriteRetryWithBackoff(t *testing.T) {
    handler := NewDefaultWriteErrorHandler(3, 100*time.Millisecond, log.New(io.Discard, "", 0))
    
    // Test exponential backoff
    for i := 0; i < 3; i++ {
        delay := handler.RetryDelay(i)
        expected := 100 * time.Millisecond * time.Duration(1<<uint(i))
        
        // Allow for jitter
        assert.InDelta(t, expected, delay, float64(expected)*0.3)
    }
}
```

## Implementation Steps

1. Create error type definitions in `pkg/types/errors.go`
2. Implement error wrapping utilities in `pkg/types/error_utils.go`
3. Create recovery handlers in `pkg/tree/error_handlers.go`
4. Implement health check functionality in `pkg/tree/health.go`
5. Add error metrics collection
6. Write comprehensive tests for all error scenarios
7. Update existing code to use new error types

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

- [ ] All error types defined according to specification
- [ ] Error wrapping utilities implemented
- [ ] Read error handler with corruption detection
- [ ] Write error handler with retry logic
- [ ] Health check implementation complete
- [ ] Error metrics collection integrated
- [ ] Comprehensive test coverage for error scenarios
- [ ] Existing code updated to use error framework
- [ ] Performance impact measured and acceptable
- [ ] Security considerations addressed