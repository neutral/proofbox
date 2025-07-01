# Storage Interface Description

## Purpose

The storage interface provides an abstraction layer over the underlying key-value storage implementation. This design allows for:

1. **Backend Flexibility**: Swap between PebbleDB, in-memory storage, or other backends
2. **Testability**: Use lightweight implementations for unit tests
3. **Monitoring**: Centralized metrics collection points
4. **Evolution**: Add new features without changing client code

## Design Decisions

### Why These Interfaces?

The interface hierarchy mirrors common database abstractions:
- `Storage`: Main entry point, similar to a database connection
- `Batch`: Atomic write operations, essential for consistency
- `Iterator`: Efficient range scans, critical for tree traversal
- `Snapshot`: Consistent reads, required for proof generation

### Iterator Design

The iterator API closely follows PebbleDB's design for two reasons:
1. Zero-cost abstraction when using PebbleDB backend
2. Familiar API for Go developers (similar to database/sql)

### Metrics as First-Class

Metrics are built into the interface rather than added later because:
- Storage is often the bottleneck in tree operations
- Production systems need observability from day one
- Metrics collection points are well-defined at the interface boundary

### VersionedStorage Extension

This optional interface enables version-aware optimizations:
- Direct version queries without tree traversal
- Efficient pruning of old versions
- Version listing for recovery and debugging

## Usage Patterns

### Typical Read Operation
```go
value, err := storage.Get(key)
if err != nil {
    return err
}
if value == nil {
    // Key not found
}
```

### Batch Write Pattern
```go
batch := storage.NewBatch()
defer batch.Close()

batch.Put(key1, value1)
batch.Put(key2, value2)
batch.Delete(key3)

err := batch.Commit(CommitOptions{Sync: true})
```

### Consistent Multi-Read
```go
snapshot := storage.NewSnapshot()
defer snapshot.Close()

// All reads see the same state
value1, _ := snapshot.Get(key1)
value2, _ := snapshot.Get(key2)
```