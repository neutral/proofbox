---
id: step.16.storage-layer
depends_on:
  - step.15.update-batch
tags: [storage, persistence, step]
---

## Objective

Refactor the existing direct PebbleDB integration into a proper storage abstraction layer that supports pluggable backends, metrics collection, and storage optimizations. Currently, the tree package directly uses PebbleDB, which limits flexibility, testability, and monitoring capabilities.

## Current State

The tree package already has working storage functionality:

- Direct PebbleDB usage in `pkg/tree/tree.go`
- Key encoding functions in `pkg/tree/storage_keys.go`
- Atomic batch commits in `CommitVersion()`
- Snapshot support via `TreeReader`
- Basic version persistence and recovery

## High-Level Plan

1. **Extract Storage Interface** - Create abstraction layer without breaking existing functionality
2. **Implement PebbleDB Driver** - Wrap current PebbleDB usage behind the interface
3. **Migrate Key Encoding** - Move storage key logic to appropriate location
4. **Add Storage Metrics** - Implement comprehensive monitoring capabilities
5. **Enhance Pruning** - Build configurable version retention policies
6. **Optimize Performance** - Add key compression and storage tuning

## Implementation Details

### 1. Storage Interface (`pkg/storage/interface.go`)

```go
type Storage interface {
    // Basic operations
    Get(key []byte) ([]byte, error)
    Put(key, value []byte) error
    Delete(key []byte) error

    // Batch operations
    NewBatch() StorageBatch

    // Iteration
    NewIterator(opts *IteratorOptions) Iterator

    // Snapshots
    NewSnapshot() Snapshot

    // Metrics
    Metrics() StorageMetrics

    // Lifecycle
    Close() error
}

type StorageBatch interface {
    Put(key, value []byte) error
    Delete(key []byte) error
    Commit(opts CommitOptions) error
    Close() error
}
```

### 2. Key Encoder Interface (`pkg/storage/keys.go`)

Move existing key encoding from `pkg/tree/storage_keys.go`:

```go
type KeyEncoder interface {
    // Node storage keys
    NodeKey(key types.NodeKey) []byte
    ParseNodeKey(data []byte) (types.NodeKey, error)

    // Value storage keys
    ValueKey(hash types.Hash) []byte

    // Root storage keys
    RootKey(version types.Version) []byte
    ParseRootKey(data []byte) (types.Version, error)
}
```

### 3. PebbleDB Implementation (`pkg/storage/pebble/pebble.go`)

- Implement Storage interface wrapping existing PebbleDB usage
- Move PebbleDB initialization from tree package
- Add configuration options (cache size, compression, etc.)
- Implement metrics collection

### 4. Storage Configuration (`pkg/storage/config.go`)

```go
type StorageConfig struct {
    // Backend selection
    Backend string // "pebble", "memory" (for testing)

    // PebbleDB specific
    CacheSize        int64
    WriteBufferSize  int
    MaxOpenFiles     int
    Compression      CompressionType

    // Metrics
    EnableMetrics    bool
    MetricsInterval  time.Duration

    // Pruning
    PruningPolicy    PruningPolicy
    PruningInterval  time.Duration
}
```

### 5. Migration Path

- Update `Tree` struct to use Storage interface instead of direct PebbleDB
- Replace direct PebbleDB calls with Storage interface methods
- Move key encoding calls to use KeyEncoder interface
- Ensure backward compatibility with existing databases

### 6. Storage Metrics (`pkg/storage/metrics.go`)

```go
type StorageMetrics interface {
    // Operation metrics
    GetOperations() uint64
    PutOperations() uint64
    DeleteOperations() uint64

    // Performance metrics
    GetLatencyP50() time.Duration
    GetLatencyP99() time.Duration

    // Size metrics
    DatabaseSize() uint64
    LiveDataSize() uint64

    // Cache metrics
    CacheHitRate() float64
}
```

### 7. Version Pruning (`pkg/storage/pruning.go`)

```go
type PruningPolicy interface {
    ShouldPrune(version types.Version, info VersionInfo) bool
}

// Implementations:
// - KeepLastNVersions(n int)
// - KeepVersionsNewerThan(age time.Duration)
// - KeepMilestoneVersions(interval int)
```

### 8. Testing Strategy

- Create `MemoryStorage` for unit tests
- Integration tests with real PebbleDB
- Benchmark comparisons (before/after refactoring)
- Migration tests from old to new structure

## Key Decisions

### Where Key Encoding Lives

After analysis, key encoding will be:

1. Moved to storage package as `KeyEncoder` interface
2. Default implementation matches current format
3. Allows backend-specific optimizations

### Backward Compatibility

- Maintain exact same key format
- Support reading existing databases
- No data migration required

### Performance Requirements

- Zero regression in operation latency
- Metrics collection overhead < 1%
- Batch operations remain atomic

## Done When ✓

- [ ] Storage interface defined and documented
- [ ] PebbleDB driver implements full interface
- [ ] Tree package uses Storage interface exclusively
- [ ] Key encoding moved to storage layer
- [ ] Metrics collection operational
- [ ] Pruning policies implemented and tested
- [ ] All existing tests pass
- [ ] New storage-specific tests added
- [ ] Performance benchmarks show no regression
- [ ] Migration guide documented
- [ ] Example programs updated to show storage configuration
