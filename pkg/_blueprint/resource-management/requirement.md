---
id: resource-management
title: Resource Management and Lifecycle
status: active
priority: critical
category: non-functional
tags: [reliability, performance, concurrency, memory-safety]
created: 2024-01-30
---

# Resource Management and Lifecycle

## Overview

This requirement defines mandatory patterns and practices for managing resources, preventing leaks, and ensuring thread-safe operations throughout the codebase. These requirements are derived from critical issues discovered during storage layer implementation and are essential for production reliability.

## Background

During the implementation of the storage abstraction layer, several fundamental design flaws were discovered that led to:
- Resource leaks (iterators, snapshots, file handles)
- Race conditions in concurrent operations
- Undefined behavior during shutdown
- Memory leaks and uncontrolled resource growth

These issues arose from:
1. Lack of clear resource ownership models
2. Missing lifecycle management
3. Improper synchronization patterns
4. No enforcement of resource limits

## Requirements

### 1. Resource Lifecycle Management

**REQ-RLM-001**: All resources MUST implement reference counting or equivalent ownership tracking.

**Rationale**: Without tracking, resources can be leaked or accessed after disposal.

**Implementation Pattern**:
```go
type Resource struct {
    refCount   int32
    closing    int32
    resourceWG sync.WaitGroup
}

func (r *Resource) addRef() bool {
    for {
        closing := atomic.LoadInt32(&r.closing)
        if closing != 0 {
            return false
        }
        refs := atomic.LoadInt32(&r.refCount)
        if atomic.CompareAndSwapInt32(&r.refCount, refs, refs+1) {
            r.resourceWG.Add(1)
            return true
        }
    }
}

func (r *Resource) releaseRef() {
    atomic.AddInt32(&r.refCount, -1)
    r.resourceWG.Done()
}
```

**Anti-pattern** (what NOT to do):
```go
// BAD: Just counting without preventing leaks
type Storage struct {
    openIterators int64  // Just tracks count
}

func (s *Storage) Close() error {
    if atomic.LoadInt64(&s.openIterators) > 0 {
        fmt.Printf("WARNING: %d iterators leaked\n", openIterators)
    }
    return s.db.Close() // Closes anyway!
}
```

### 2. Resource Limits and Early Failure

**REQ-RLM-002**: Resource creation MUST enforce limits and fail fast when exceeded.

**Rationale**: Unlimited resource creation leads to memory exhaustion and system instability.

**Implementation Pattern**:
```go
func (s *Storage) NewIterator(opts *IteratorOptions) Iterator {
    if !s.addRef() {
        return &errorIterator{err: ErrStorageClosed}
    }

    // Enforce limits atomically
    if s.maxIterators > 0 {
        current := atomic.AddInt32(&s.openIterators, 1)
        if current > s.maxIterators {
            atomic.AddInt32(&s.openIterators, -1)
            s.releaseRef()
            return &errorIterator{err: ErrTooManyIterators}
        }
    }
    
    // Create resource only after checks pass
    // ...
}
```

**Anti-pattern**:
```go
// BAD: No limits, just warnings
func (s *Storage) NewIterator() Iterator {
    atomic.AddInt64(&s.openIterators, 1)
    // No limit check - can create unlimited iterators!
    return &iterator{...}
}
```

### 3. Graceful Shutdown and Resource Cleanup

**REQ-RLM-003**: Components MUST wait for all resources to be released before shutdown.

**Rationale**: Premature shutdown leads to crashes, data corruption, and undefined behavior.

**Implementation Pattern**:
```go
func (s *Storage) Close() error {
    // 1. Atomically prevent new operations
    if !atomic.CompareAndSwapInt32(&s.closing, 0, 1) {
        return nil
    }

    // 2. Signal background operations to stop
    if s.stopChan != nil {
        close(s.stopChan)
    }

    // 3. Wait for background operations
    s.backgroundWG.Wait()

    // 4. Wait for all active resources
    s.resourceWG.Wait()

    // 5. Only now safe to close
    return s.db.Close()
}
```

**Anti-pattern**:
```go
// BAD: Race-prone shutdown
func (s *Storage) Close() error {
    s.mu.Lock()
    s.closed = true
    s.mu.Unlock()  // Race window!
    
    // Background goroutine might still access s.db
    return s.db.Close()
}
```

### 4. Thread-Safe State Transitions

**REQ-RLM-004**: State transitions MUST be atomic with no race windows.

**Rationale**: Non-atomic transitions create race conditions leading to crashes and data corruption.

**Implementation Pattern**:
```go
// Atomic state check and operation
func (s *Storage) Get(key []byte) ([]byte, error) {
    if atomic.LoadInt32(&s.closing) != 0 {
        return nil, ErrStorageClosed
    }
    // Proceed with operation
}

// Background goroutine with proper synchronization
func (s *Storage) backgroundWorker() {
    defer s.backgroundWG.Done()
    
    for {
        select {
        case <-s.stopChan:
            return
        case <-ticker.C:
            if atomic.LoadInt32(&s.closing) != 0 {
                return
            }
            // Safe to proceed
        }
    }
}
```

**Anti-pattern**:
```go
// BAD: Lock/unlock/wait/lock pattern creates races
func (s *Storage) Close() error {
    s.mu.Lock()
    s.closed = true
    close(s.stopChan)
    s.mu.Unlock()      // Race window starts!
    
    s.workersWG.Wait() // Other goroutine might see !closed
    
    s.mu.Lock()        // Race window ends
    // ...
}
```

### 5. Resource Ownership Model

**REQ-RLM-005**: Every resource MUST have clear ownership and prevent parent disposal while active.

**Rationale**: Unclear ownership leads to use-after-free and resource leaks.

**Implementation Pattern**:
```go
type Iterator struct {
    storage *Storage  // Parent reference
    closed  int32
}

func (i *Iterator) Close() error {
    if atomic.CompareAndSwapInt32(&i.closed, 0, 1) {
        err := i.cleanup()
        
        // Release parent reference last
        atomic.AddInt32(&i.storage.openIterators, -1)
        i.storage.releaseRef()
        
        return err
    }
    return nil
}
```

### 6. Error Propagation for Resource Constraints

**REQ-RLM-006**: Resource constraint violations MUST return specific errors, not generic failures.

**Rationale**: Specific errors enable proper handling and debugging.

**Required Error Types**:
```go
var (
    ErrStorageClosed     = errors.New("storage is closed")
    ErrTooManyIterators  = errors.New("too many open iterators")
    ErrTooManySnapshots  = errors.New("too many open snapshots")
    ErrResourceExhausted = errors.New("resource limit exceeded")
)
```

## Testing Requirements

### 1. Resource Leak Detection

All resource-managing components MUST include tests that verify:
- Resources are properly cleaned up
- No leaks occur under normal operation
- No leaks occur under error conditions

### 2. Concurrency Testing

All components MUST pass tests with race detector enabled:
```bash
go test -race -count=10 ./...
```

### 3. Stress Testing

Components MUST include stress tests that verify:
- Resource limits are enforced under load
- No resources leak under concurrent access
- Graceful degradation when limits are reached

## Compliance Checklist

When implementing any component that manages resources:

- [ ] Implements reference counting or equivalent ownership tracking
- [ ] Enforces resource limits with early failure
- [ ] Waits for all resources before shutdown
- [ ] Uses atomic operations for state transitions
- [ ] Has clear parent-child ownership model
- [ ] Returns specific errors for resource constraints
- [ ] Includes comprehensive tests for resource management
- [ ] Passes race detector tests
- [ ] Documents resource limits and lifecycle

## Industry Standards Alignment

This requirement aligns with:

1. **RAII (Resource Acquisition Is Initialization)** - Resources tied to object lifetime
2. **CAP Theorem** - Consistency over availability for resource limits
3. **Fail-Fast Principle** - Early failure on resource exhaustion
4. **Actor Model** - Clear ownership and message passing
5. **CSP (Communicating Sequential Processes)** - Proper channel-based synchronization

## References

- [Go Memory Model](https://go.dev/ref/mem)
- [Effective Go - Concurrency](https://go.dev/doc/effective_go#concurrency)
- [Linux Kernel Resource Management](https://www.kernel.org/doc/html/latest/admin-guide/resource-control.html)
- [POSIX Thread Safety](https://pubs.opengroup.org/onlinepubs/9699919799/)

## Examples from Codebase

See the following implementations for reference:
- `/pkg/storage/pebble/pebble.go` - Full reference counting implementation
- `/pkg/storage/memory/memory.go` - Memory-safe resource management
- `/pkg/storage/resource_test.go` - Comprehensive resource management tests
- `/pkg/storage/RESOURCE_MANAGEMENT.md` - Detailed implementation notes