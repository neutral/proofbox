# Error Utilities Description

## Overview
This file provides utility functions and types for error handling, including error wrapping, detailed error information, and retry logic helpers.

## Key Components

### WrapError Function
```go
func WrapError(err error, format string, args ...interface{}) error
```

**Purpose**: Adds contextual information to errors while preserving the original error type.

**Why This Matters**:
- Maintains error type identity for `errors.Is()` checks
- Adds debugging context without losing the ability to handle specific error types
- Follows Go 1.13+ error wrapping conventions with `%w` verb

**Design Choice**: Returns `nil` for `nil` input to allow chaining without nil checks.

### ErrorInfo Type
Provides rich error context for logging and debugging:
- `Code`: Machine-readable error category
- `Message`: Human-readable error message  
- `Details`: Structured additional context (key-value pairs)
- `Timestamp`: When the error occurred
- `Stack`: Stack trace for debugging

**When to Use**:
- For errors that will be logged or sent to monitoring systems
- When debugging complex error scenarios
- For errors crossing service boundaries

**Performance Note**: Stack trace collection has overhead - use judiciously in hot paths.

### ErrorCode Enumeration
Categorizes errors for metrics and monitoring:
- `CodeCorruption`: Data integrity violations
- `CodeNotFound`: Expected "not found" conditions
- `CodeInvalidInput`: Client errors
- `CodeStorageFailure`: Infrastructure issues
- `CodeResourceExhausted`: Capacity limits
- `CodeTransactionConflict`: Concurrency issues

**Purpose**: Enables error categorization for:
- Metrics/monitoring dashboards
- SLA tracking (e.g., don't count client errors)
- Automated alerting rules

### IsRetryable Function
Determines if an error is transient and can be retried:
```go
func IsRetryable(err error) bool
```

**Retryable Errors**:
- `ErrStorageFailure`: Temporary storage issues
- `ErrTransactionAborted`: Transaction rolled back, can retry
- `ErrTxConflict`: Optimistic locking conflict

**Design Philosophy**: Only mark errors as retryable if:
1. The error is truly transient
2. Retrying has a reasonable chance of success
3. Retrying won't cause data inconsistency

### GetErrorCode Function
Maps errors to their appropriate ErrorCode category.

**Implementation Note**: Uses `errors.Is()` throughout to handle wrapped errors correctly.

## Usage Patterns

### Adding Context
```go
node, err := loadNode(key)
if err != nil {
    return WrapError(err, "failed to load node at version %d, path %x", 
        key.Version, key.NibblePath)
}
```

### Creating Detailed Errors
```go
if computedHash != storedHash {
    return NewErrorInfo(CodeCorruption, ErrHashMismatch, map[string]interface{}{
        "node_key": key,
        "computed": computedHash.Hex(),
        "stored": storedHash.Hex(),
    })
}
```

### Retry Logic
```go
if IsRetryable(err) {
    // Implement exponential backoff
    time.Sleep(backoffDuration)
    return retry()
}
```

## Design Decisions

### Why Sentinel Errors + Wrapping?
- Sentinel errors provide type safety and discoverability
- Wrapping adds context without losing type information
- Best of both worlds: type checking + debugging context

### Why ErrorInfo Includes Stack?
- Production debugging often requires stack traces
- Corruption errors especially benefit from full context
- Can be disabled in performance-critical paths

### Why Separate IsRetryable?
- Retry logic is cross-cutting concern
- Centralizes retry decision logic
- Makes retryable errors explicit and documented

## Future Considerations
- Could add error sampling for high-frequency errors
- Might integrate with OpenTelemetry for distributed tracing
- Could add error fingerprinting for deduplication