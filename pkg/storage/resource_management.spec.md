# Storage Resource Management Specification

## Overview

This specification defines the resource management implementation patterns used across all storage backends to prevent resource leaks and race conditions.

## Core Components

### 1. Reference Counting System

**Purpose**: Track active resources and prevent premature storage closure.

**Implementation**:
```go
type Storage struct {
    refCount   int32           // Atomic reference counter
    closing    int32           // Atomic closing flag
    resourceWG sync.WaitGroup  // Wait for all resources
}
```

**Operations**:
- `addRef()`: Atomically increment reference count if not closing
- `releaseRef()`: Atomically decrement reference count and signal completion

### 2. Resource Limit Enforcement

**Purpose**: Prevent resource exhaustion through early failure.

**Configuration**:
```go
type Options struct {
    MaxIterators int32  // Maximum concurrent iterators (0 = unlimited)
    MaxSnapshots int32  // Maximum concurrent snapshots (0 = unlimited)
}
```

**Enforcement Pattern**:
```go
// 1. Add reference first
if !s.addRef() {
    return &errorType{err: ErrStorageClosed}
}

// 2. Check and increment limit
if s.maxIterators > 0 {
    current := atomic.AddInt32(&s.openIterators, 1)
    if current > s.maxIterators {
        // Rollback on failure
        atomic.AddInt32(&s.openIterators, -1)
        s.releaseRef()
        return &errorIterator{err: ErrTooManyIterators}
    }
}

// 3. Create resource only after all checks pass
```

### 3. Graceful Shutdown Protocol

**Purpose**: Ensure clean shutdown without resource leaks or crashes.

**Sequence**:
1. Atomically set closing flag (CAS operation)
2. Signal background operations to stop
3. Wait for background operations (metricsWG)
4. Wait for all active resources (resourceWG)
5. Close underlying storage

**Implementation**:
```go
func (s *Storage) Close() error {
    // Step 1: Prevent new operations
    if !atomic.CompareAndSwapInt32(&s.closing, 0, 1) {
        return nil // Already closing
    }
    
    // Steps 2-5: Coordinated shutdown
    // ... (see full implementation)
}
```

### 4. Error Handling Strategy

**Error Types**:
- `ErrStorageClosed`: Operations on closed storage
- `ErrTooManyIterators`: Iterator limit exceeded
- `ErrTooManySnapshots`: Snapshot limit exceeded

**Error Resources**:
- `errorIterator`: Iterator that always returns error
- `errorSnapshot`: Snapshot that always returns error
- `errorBatch`: Batch that always returns error

## Testing Requirements

### Required Test Coverage

1. **Resource Limits**
   - Verify limits are enforced
   - Verify resources can be created after others are closed
   - Verify proper error types returned

2. **Concurrent Operations**
   - Race detector must pass
   - Concurrent resource creation/destruction
   - Proper synchronization under load

3. **Shutdown Behavior**
   - Storage waits for active resources
   - Operations fail after close
   - Double-close is safe

## Compliance Checklist

- [ ] Implements reference counting with addRef/releaseRef
- [ ] Enforces resource limits with early failure
- [ ] Uses atomic operations for state transitions
- [ ] Implements graceful shutdown protocol
- [ ] Returns specific error types
- [ ] Passes race detector tests
- [ ] Includes comprehensive test coverage