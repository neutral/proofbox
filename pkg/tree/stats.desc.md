# Tree Statistics Component

This component provides atomic, thread-safe statistics tracking for the Jellyfish Merkle Tree.

## Purpose

The TreeStats struct was introduced to solve race conditions when updating tree metrics. It provides atomic operations for tracking tree state that can be safely accessed from multiple goroutines.

## Implementation

### TreeStats Structure
```go
type TreeStats struct {
    height    atomic.Int64  // Tree height (not yet tracked)
    nodeCount atomic.Int64  // Total nodes in the tree
    version   atomic.Int64  // Latest version number
}
```

### Key Features

1. **Atomic Operations**: All fields use `atomic.Int64` for thread-safe updates
2. **Bulk Updates**: `UpdateStats()` method for updating multiple values atomically
3. **Individual Accessors**: Separate methods for reading/writing each statistic

### Usage

The TreeStats is integrated into the Tree struct and updated during operations:

- **CommitVersion**: Updates node count and version after successful commit
- **Metrics Export**: Values are read atomically and exported to Prometheus

### Thread Safety

All operations on TreeStats are atomic, preventing race conditions when:
- Multiple goroutines are committing versions
- Metrics are being scraped while tree operations occur
- Concurrent reads of tree statistics

## Future Enhancements

1. **Tree Height Tracking**: Currently not implemented due to performance considerations
2. **Additional Statistics**: Could track operation counts, cache hit rates, etc.
3. **Historical Tracking**: Could maintain statistics per version

## Design Rationale

This component was created to fix concurrency issues in the metrics implementation. By using atomic operations, we avoid the need for mutex locks on the hot path while still providing accurate statistics for monitoring.