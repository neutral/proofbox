# Error Interfaces Description

## Overview
This file defines interfaces for error handling strategies, allowing the JMT implementation to support different error recovery approaches and monitoring systems. The interfaces are defined here in the types package, with concrete implementations provided later when the tree structure is available.

## Interface Design Philosophy

### Separation of Interface and Implementation
We define interfaces in the `types` package but defer implementations to the `tree` package because:
1. **Dependency Direction**: Types package has no dependencies, tree depends on types
2. **Flexibility**: Users can provide custom implementations
3. **Testing**: Easy to mock interfaces for testing
4. **Evolution**: Can add new methods to interfaces without breaking existing code

### Placeholder Types
The file includes placeholder types (`NodeKey`, `Version`, `NibblePath`) that will be properly defined in future steps. This allows the interfaces to compile now while maintaining type safety.

## Core Interfaces

### ReadErrorHandler
```go
type ReadErrorHandler interface {
    HandleCorruptedNode(key NodeKey, err error) error
    HandleMissingNode(key NodeKey) error
}
```

**Purpose**: Defines how to handle errors during read operations.

**Key Methods**:
- `HandleCorruptedNode`: Called when node hash verification fails
- `HandleMissingNode`: Called when a referenced node doesn't exist

**Design Choices**:
- Returns error to allow handler to transform or escalate
- Receives full context (NodeKey) for informed decisions
- Separate methods for different error types enable specific handling

**Example Strategies**:
1. **Fail-Fast**: Return error immediately (default)
2. **Logging**: Log and return sanitized error
3. **Recovery**: Attempt to recover from backup
4. **Alerting**: Trigger operational alerts

### WriteErrorHandler
```go
type WriteErrorHandler interface {
    HandleWriteFailure(err error) error
    ShouldRetry(err error) bool
    RetryDelay(attempt int) time.Duration
}
```

**Purpose**: Manages write failures and retry logic.

**Key Methods**:
- `HandleWriteFailure`: Process/transform write errors
- `ShouldRetry`: Decide if error warrants retry
- `RetryDelay`: Calculate backoff delay for retries

**Design Rationale**:
- Separates retry decision from retry timing
- Allows sophisticated backoff strategies
- Enables circuit breaker patterns

**Common Implementations**:
1. **Exponential Backoff**: Increasing delays with jitter
2. **Linear Backoff**: Fixed delay increments
3. **Circuit Breaker**: Stop retrying after threshold
4. **Adaptive**: Adjust based on error patterns

### ErrorReporter
```go
type ErrorReporter interface {
    RecordError(code ErrorCode, err error)
    GetStats() map[ErrorCode]uint64
}
```

**Purpose**: Pluggable error metrics collection.

**Design Benefits**:
- Decouples error tracking from business logic
- Allows integration with various metrics systems
- Enables error rate monitoring and alerting

**Integration Examples**:
- Prometheus metrics
- StatsD counters
- Application insights
- Custom telemetry

### HealthChecker
```go
type HealthChecker interface {
    VerifyIntegrity(ctx context.Context, version Version) error
}
```

**Purpose**: Tree integrity verification interface.

**Key Features**:
- Context support for cancellation
- Version-specific checks
- Comprehensive integrity validation

**Use Cases**:
1. **Periodic Health Checks**: Scheduled integrity scans
2. **Post-Recovery Validation**: After crash recovery
3. **Debugging**: When corruption suspected
4. **Compliance**: Audit trail verification

## Implementation Strategy

### Step 3 (Current)
- Define interfaces only
- No concrete implementations
- Focus on API design

### Step 8 (Future)
- Implement `DefaultReadErrorHandler`
- Implement `DefaultWriteErrorHandler`
- Implement `TreeHealthChecker`
- Add retry logic with backoff

## Usage Patterns

### Dependency Injection
```go
type Tree struct {
    readHandler  ReadErrorHandler
    writeHandler WriteErrorHandler
    reporter     ErrorReporter
}

func NewTree(opts ...Option) *Tree {
    t := &Tree{
        readHandler:  &DefaultReadErrorHandler{},
        writeHandler: &DefaultWriteErrorHandler{},
    }
    // Apply options
    return t
}
```

### Custom Handlers
```go
type LoggingReadHandler struct {
    logger *log.Logger
}

func (h *LoggingReadHandler) HandleCorruptedNode(key NodeKey, err error) error {
    h.logger.Error("corruption detected", 
        "key", key, 
        "error", err)
    return err
}
```

## Future Enhancements

### Possible Extensions
1. **Async Error Reporting**: Non-blocking error recording
2. **Error Sampling**: For high-frequency errors
3. **Error Context**: Richer error metadata
4. **Recovery Strategies**: Automated recovery actions

### Compatibility
Interfaces designed to be extended without breaking changes:
- New methods can be added to new interfaces
- Existing interfaces remain stable
- Optional interfaces pattern for advanced features