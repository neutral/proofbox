# PebbleDB Storage Implementation

## Purpose

This implementation provides a production-ready storage backend using PebbleDB (CockroachDB's fork of RocksDB). It implements the storage interface with proper resource management, preventing leaks and race conditions.

The driver serves as a thin wrapper over PebbleDB, providing:
- Production-ready LSM-tree based storage
- High performance for write-heavy workloads
- ACID guarantees with crash recovery
- Built-in compression and caching

## Resource Management Design

### Reference Counting
- Every storage instance maintains an atomic reference count (`refCount`)
- Resources (iterators, snapshots, batches) increment the count on creation via `addRef()`
- Resources decrement the count on close via `releaseRef()`
- Storage waits for all references to be released before closing using `resourceWG.Wait()`

### Resource Limits
- Configurable limits for iterators (`MaxIterators`) and snapshots (`MaxSnapshots`)
- Early failure with specific errors (`ErrTooManyIterators`, `ErrTooManySnapshots`)
- Atomic counters track current usage without race conditions

### Atomic State Transitions
- `closing` flag (atomic int32) prevents new operations during shutdown
- No race windows - operations check closing state atomically
- Metrics goroutine properly synchronized without lock/unlock/lock pattern

### Key Patterns

1. **Resource Creation**: Check closing → Add reference → Check limit → Create
2. **Resource Cleanup**: Close resource → Decrement counter → Release reference
3. **Storage Close**: Mark closing → Stop background → Wait for resources → Close DB

## Configuration

```go
opts := &pebble.Options{
    PebbleOptions: &pebble.Options{
        Cache:        pebble.NewCache(64 << 20), // 64MB cache
        MemTableSize: 32 << 20,                   // 32MB write buffer
    },
    EnableMetrics: true,
    MaxIterators:  1000,  // Limit concurrent iterators
    MaxSnapshots:  100,   // Limit concurrent snapshots
}
```

## Thread Safety

All operations are thread-safe through:
- Atomic operations for state checks and counters
- WaitGroups for coordinating shutdown
- No shared mutable state without protection
- Proper cleanup order prevents use-after-free