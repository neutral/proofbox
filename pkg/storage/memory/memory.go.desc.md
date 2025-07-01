# Memory Storage Implementation

## Purpose

This implementation provides an in-memory storage backend for testing and development. It implements the same resource management patterns as the PebbleDB storage to ensure consistent behavior across backends.

Ideal for:
- Unit testing without disk I/O
- Integration testing with predictable behavior
- Development with quick iteration
- Benchmarking tree algorithms without storage overhead

## Design Principles

### Thread-Safe In-Memory Map
- Uses `sync.RWMutex` for protecting the data map
- All operations create copies to prevent external modifications
- Snapshots create full copies for point-in-time consistency

### Resource Management
- Identical reference counting pattern as PebbleDB storage
- Same resource limits and early failure behavior
- Ensures tests using memory storage catch resource issues

### Key Differences from PebbleDB

1. **Data Storage**: Map[string][]byte instead of LSM tree
2. **Persistence**: None - data lost on close
3. **Performance**: Faster for small datasets, slower for large ones
4. **Memory Usage**: Keeps everything in memory

## Implementation Patterns

### Snapshot Consistency
```go
// Create a snapshot of data for consistent iteration
dataCopy := make(map[string][]byte, len(s.data))
for k, v := range s.data {
    valueCopy := make([]byte, len(v))
    copy(valueCopy, v)
    dataCopy[k] = valueCopy
}
```

### Iterator Implementation
- Sorts keys for deterministic iteration order
- Applies bounds and prefix filters after sorting
- Maintains snapshot of data to prevent concurrent modification issues

## Testing Benefits

Memory storage helps catch:
- Resource leaks (same limits as production)
- Race conditions (same synchronization patterns)
- API misuse (same error returns)
- Integration issues (same behavior contract)