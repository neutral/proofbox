# Error Handling Specification

## Overview
This specification defines the error handling strategy for the Jellyfish Merkle Tree implementation, including error types, codes, recovery strategies, and consistency guarantees.

## Error Categories

### 1. Data Integrity Errors
Errors that indicate corruption or inconsistency in stored data.

```go
var (
    // ErrCorruptedNode indicates a node's hash doesn't match its content
    ErrCorruptedNode = errors.New("jmt: corrupted node detected")
    
    // ErrInvalidNodeType indicates an unknown node type in storage
    ErrInvalidNodeType = errors.New("jmt: invalid node type")
    
    // ErrHashMismatch indicates computed hash doesn't match expected
    ErrHashMismatch = errors.New("jmt: hash mismatch")
    
    // ErrInvalidProof indicates a proof is malformed or invalid
    ErrInvalidProof = errors.New("jmt: invalid proof")
)
```

### 2. Operational Errors
Errors that occur during normal operations.

```go
var (
    // ErrKeyNotFound indicates the requested key doesn't exist
    ErrKeyNotFound = errors.New("jmt: key not found")
    
    // ErrVersionNotFound indicates the requested version doesn't exist
    ErrVersionNotFound = errors.New("jmt: version not found")
    
    // ErrNodeNotFound indicates a referenced node is missing
    ErrNodeNotFound = errors.New("jmt: node not found")
    
    // ErrEmptyTree indicates operation on empty tree
    ErrEmptyTree = errors.New("jmt: empty tree")
)
```

### 3. Input Validation Errors
Errors from invalid input parameters.

```go
var (
    // ErrInvalidKey indicates an invalid key format
    ErrInvalidKey = errors.New("jmt: invalid key")
    
    // ErrEmptyKey indicates an empty key was provided
    ErrEmptyKey = errors.New("jmt: empty key not allowed")
    
    // ErrInvalidVersion indicates an invalid version number
    ErrInvalidVersion = errors.New("jmt: invalid version")
    
    // ErrValueTooLarge indicates value exceeds size limit
    ErrValueTooLarge = errors.New("jmt: value too large")
    
    // ErrInvalidNibble indicates nibble value > 15
    ErrInvalidNibble = errors.New("jmt: invalid nibble value")
    
    // ErrInvalidNodeKey indicates malformed node key
    ErrInvalidNodeKey = errors.New("jmt: invalid node key")
    
    // ErrInvalidNibbleCount indicates nibble path too long
    ErrInvalidNibbleCount = errors.New("jmt: nibble count exceeds maximum")
)
```

### 4. Resource Errors
Errors related to system resources.

```go
var (
    // ErrMaxDepthExceeded indicates tree depth limit reached
    ErrMaxDepthExceeded = errors.New("jmt: maximum tree depth exceeded")
    
    // ErrBatchTooLarge indicates batch exceeds size limit
    ErrBatchTooLarge = errors.New("jmt: batch size exceeds limit")
    
    // ErrOutOfMemory indicates memory allocation failure
    ErrOutOfMemory = errors.New("jmt: out of memory")
)
```

### 5. Storage Errors
Errors from the underlying storage layer.

```go
var (
    // ErrStorageFailure wraps underlying storage errors
    ErrStorageFailure = errors.New("jmt: storage operation failed")
    
    // ErrTransactionAborted indicates a transaction was aborted
    ErrTransactionAborted = errors.New("jmt: transaction aborted")
    
    // ErrDatabaseClosed indicates operation on closed database
    ErrDatabaseClosed = errors.New("jmt: database closed")
)
```

## Error Wrapping

Use error wrapping to provide context while preserving error types:

```go
// WrapError adds context to an error
func WrapError(err error, format string, args ...interface{}) error {
    if err == nil {
        return nil
    }
    msg := fmt.Sprintf(format, args...)
    return fmt.Errorf("%s: %w", msg, err)
}

// Example usage:
func (t *Tree) Get(version Version, key Key) ([]byte, error) {
    node, err := t.loadNode(version, nibblePath)
    if err != nil {
        return nil, WrapError(err, "failed to load node at version %d", version)
    }
    // ...
}
```

## Error Recovery Strategies

### 1. Read Path Recovery

```go
// ReadErrorHandler defines recovery strategy for read errors
type ReadErrorHandler interface {
    HandleCorruptedNode(key NodeKey, err error) error
    HandleMissingNode(key NodeKey) error
}

// DefaultReadErrorHandler implements basic recovery
type DefaultReadErrorHandler struct {
    logger Logger
}

func (h *DefaultReadErrorHandler) HandleCorruptedNode(key NodeKey, err error) error {
    h.logger.Error("corrupted node detected", 
        "version", key.Version,
        "path", key.NibblePath,
        "error", err)
    
    // Return error to caller - don't try to recover
    return ErrCorruptedNode
}

func (h *DefaultReadErrorHandler) HandleMissingNode(key NodeKey) error {
    // Missing nodes might be due to pruning
    return ErrNodeNotFound
}
```

### 2. Write Path Recovery

```go
// WriteErrorHandler defines recovery strategy for write errors
type WriteErrorHandler interface {
    HandleWriteFailure(batch WriteBatch, err error) error
    ShouldRetry(err error) bool
    RetryDelay(attempt int) time.Duration
}

// DefaultWriteErrorHandler implements exponential backoff
type DefaultWriteErrorHandler struct {
    maxRetries int
    baseDelay  time.Duration
}

func (h *DefaultWriteErrorHandler) ShouldRetry(err error) bool {
    // Retry on temporary storage errors
    if errors.Is(err, ErrStorageFailure) {
        return true
    }
    return false
}

func (h *DefaultWriteErrorHandler) RetryDelay(attempt int) time.Duration {
    // Exponential backoff with jitter
    delay := h.baseDelay * time.Duration(1<<uint(attempt))
    jitter := time.Duration(rand.Int63n(int64(delay / 4)))
    return delay + jitter
}
```

## Consistency Guarantees

### Transaction Rollback

```go
// Transaction represents an atomic operation
type Transaction struct {
    batch    WriteBatch
    snapshot ReadSnapshot
    tree     *Tree
}

// Commit attempts to commit the transaction
func (tx *Transaction) Commit() error {
    // Validate all operations
    if err := tx.validate(); err != nil {
        return WrapError(err, "transaction validation failed")
    }
    
    // Attempt to write
    if err := tx.tree.db.Write(tx.batch); err != nil {
        // Automatic rollback - batch is discarded
        return WrapError(ErrStorageFailure, "commit failed")
    }
    
    return nil
}

// Rollback explicitly discards the transaction
func (tx *Transaction) Rollback() {
    tx.batch.Clear()
    tx.snapshot.Close()
}
```

### Corruption Detection

```go
// VerifyNode checks node integrity
func VerifyNode(key NodeKey, data []byte) error {
    node, err := DecodeNode(data)
    if err != nil {
        return WrapError(err, "failed to decode node")
    }
    
    // Verify hash matches content
    computed := node.Hash()
    stored := node.GetStoredHash()
    
    if !bytes.Equal(computed[:], stored[:]) {
        return ErrHashMismatch
    }
    
    return nil
}

// HealthCheck performs integrity check on tree
func (t *Tree) HealthCheck(version Version) error {
    // Start from root
    rootKey := RootNodeKey(version)
    visited := make(map[NodeKey]bool)
    
    var checkNode func(NodeKey) error
    checkNode = func(key NodeKey) error {
        if visited[key] {
            return nil
        }
        visited[key] = true
        
        // Load and verify node
        data, err := t.db.Get(key.StorageKey())
        if err != nil {
            return WrapError(err, "failed to load node")
        }
        
        if err := VerifyNode(key, data); err != nil {
            return err
        }
        
        // Recursively check children
        node, _ := DecodeNode(data)
        if internal, ok := node.(*InternalNode); ok {
            for nibble, child := range internal.Children {
                childKey := key.Child(nibble, child.Version)
                if err := checkNode(childKey); err != nil {
                    return err
                }
            }
        }
        
        return nil
    }
    
    return checkNode(rootKey)
}
```

## Error Reporting

### Structured Error Information

```go
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
)

// NewErrorInfo creates detailed error information
func NewErrorInfo(code ErrorCode, err error, details map[string]interface{}) *ErrorInfo {
    return &ErrorInfo{
        Code:      code,
        Message:   err.Error(),
        Details:   details,
        Timestamp: time.Now(),
        Stack:     debug.Stack(),
    }
}
```

### Metrics and Monitoring

```go
// ErrorMetrics tracks error occurrences
type ErrorMetrics struct {
    mu     sync.RWMutex
    counts map[ErrorCode]uint64
}

func (m *ErrorMetrics) RecordError(code ErrorCode) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.counts[code]++
}

func (m *ErrorMetrics) GetStats() map[ErrorCode]uint64 {
    m.mu.RLock()
    defer m.mu.RUnlock()
    
    stats := make(map[ErrorCode]uint64)
    for code, count := range m.counts {
        stats[code] = count
    }
    return stats
}
```

## Best Practices

1. **Always check errors**: Never ignore returned errors
2. **Fail fast**: Return errors immediately, don't continue with invalid state
3. **Preserve error types**: Use `errors.Is()` and `errors.As()` for type checking
4. **Add context**: Wrap errors with relevant context using `WrapError()`
5. **Log appropriately**: Log errors at appropriate levels (Error, Warn, Info)
6. **Monitor errors**: Track error rates and types for operational visibility

## Example Usage

```go
func ExampleErrorHandling() {
    tree := NewTree(db)
    
    // Handle not found error
    value, err := tree.Get(100, key)
    if err != nil {
        if errors.Is(err, ErrKeyNotFound) {
            // Key doesn't exist - this might be expected
            return nil
        }
        // Unexpected error
        return WrapError(err, "failed to get key %x", key)
    }
    
    // Handle batch operations with retry
    batch := tree.NewBatch()
    batch.Put(key1, value1)
    batch.Put(key2, value2)
    
    err = retryWithBackoff(func() error {
        return batch.Commit()
    }, 3, time.Second)
    
    if err != nil {
        // Log and return wrapped error
        logger.Error("batch commit failed after retries", "error", err)
        return WrapError(err, "failed to commit batch of %d operations", batch.Len())
    }
}

func retryWithBackoff(fn func() error, maxAttempts int, baseDelay time.Duration) error {
    var err error
    for i := 0; i < maxAttempts; i++ {
        err = fn()
        if err == nil {
            return nil
        }
        
        if !isRetryable(err) {
            return err
        }
        
        if i < maxAttempts-1 {
            delay := baseDelay * time.Duration(1<<uint(i))
            time.Sleep(delay)
        }
    }
    return err
}

func isRetryable(err error) bool {
    return errors.Is(err, ErrStorageFailure) || 
           errors.Is(err, ErrTransactionAborted)
}
```