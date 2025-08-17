# Design Decisions

This document explains the key architectural and implementation decisions made in ProofBox, including the rationale behind each choice and the trade-offs involved.

## Design Philosophy

ProofBox follows these guiding principles:

1. **Correctness over Performance** - Get it right first, optimize later
2. **Simplicity over Features** - Fewer moving parts mean fewer bugs
3. **Explicit over Implicit** - Clear behavior, no magic
4. **Production Ready** - Built for real-world deployment from day one

## Language Choice: Why Go?

### Decision
ProofBox is implemented in Go rather than Rust, C++, or other systems languages.

### Rationale
- **Ecosystem Fit**: Strong ecosystem for distributed systems and infrastructure
- **Developer Velocity**: Faster development cycle, easier onboarding
- **Tooling**: Excellent built-in testing, benchmarking, and profiling
- **Concurrency**: First-class concurrency primitives (goroutines, channels)
- **Memory Safety**: Garbage collection eliminates entire classes of bugs
- **Cross-platform**: Easy compilation for multiple platforms

### Trade-offs
- **Performance**: ~20% slower than optimal C++ implementation
- **GC Pauses**: Potential latency spikes (mitigated by careful design)
- **Memory Usage**: Higher baseline memory consumption

## Data Structure: Two Node Types Only

### Decision
Use only Internal and Leaf nodes - no extension nodes, no optimizations for common prefixes.

### Rationale
From the two-node-types decision:
- **Simplicity**: Two node types are easier to reason about
- **Fewer Bugs**: Each node type adds complexity and edge cases
- **Cleaner Interface**: Uniform operations on all nodes
- **Sufficient Performance**: Radix-16 provides good height balance

### Trade-offs
- Trees may be deeper for sequential keys
- More internal nodes for sparse key distributions
- Slightly higher storage overhead

### Why This Works
The 16-way branching already provides significant height reduction compared to binary trees. For randomly distributed keys (common in cryptographic applications), the lack of extension nodes has minimal impact.

## Storage: Why PebbleDB?

### Decision
Use PebbleDB as the primary storage backend instead of RocksDB, LevelDB, or custom storage.

### Rationale
- **Modern Design**: Built on lessons learned from RocksDB/LevelDB
- **Go Native**: No CGO, pure Go implementation
- **LSM Optimized**: Designed specifically for LSM-tree workloads
- **Better Compaction**: More efficient compaction algorithms
- **Active Development**: Well-maintained by CockroachDB team

### Trade-offs
- **Newer**: Less battle-tested than RocksDB
- **Smaller Community**: Fewer resources and examples
- **Feature Set**: Fewer advanced features than RocksDB

## Architecture: Separate Tree from Values

### Decision
Two-tier architecture where tree nodes store only hashes, with values stored separately.

### Rationale
From the value storage separation decision:
- **Memory Efficiency**: Can traverse tree without loading values
- **Cache Optimization**: More nodes fit in memory
- **Flexibility**: Values can be compressed/encrypted independently
- **Lazy Loading**: Load values only when needed

### Implementation
```go
// Tree node stores hash reference
type LeafNode struct {
    key       Key
    valueHash Hash
    value     []byte // Optional, loaded on demand
}

// Actual value stored separately
storage.Put(valueKey(hash), actualValue)
```

### Trade-offs
- **Extra I/O**: Additional read when value is needed
- **Complexity**: Two-phase loading logic
- **Storage Overhead**: Duplicate key storage

## Concurrency: Thread-Safe Nodes

### Decision
Build thread safety directly into node types using `sync.RWMutex`.

### Rationale
From the thread-safe nodes decision:
- **Safety by Default**: Impossible to have data races
- **Simplified API**: No external synchronization needed
- **Read Optimization**: Multiple concurrent readers
- **Granular Locking**: Per-node rather than per-tree

### Trade-offs
- **Memory Overhead**: 24 bytes per node for mutex
- **Lock Contention**: Potential bottleneck on hot nodes
- **Complexity**: Must carefully manage lock ordering

## Error Handling: Minimal Panics

### Decision
Avoid panics in library code - return errors as values, with one critical exception.

### Rationale
From the no-panics decision:
- **Reliability**: Library should rarely crash host application
- **Explicit Handling**: Callers decide how to handle errors
- **Debugging**: Clear error chains and context
- **Go Best Practice**: Standard for production libraries

### Implementation
```go
// Standard practice:
if node == nil {
    return nil, errors.New("nil node")
}

// Critical exception (tree.go:534):
// Panic only when data corruption could occur
if err := t.versionManager.Commit(batch.version); err != nil {
    // Storage committed but version tracking failed
    // This is unrecoverable - data integrity at risk
    panic(fmt.Sprintf("critical: version manager commit failed: %v", err))
}
```

### The One Exception
ProofBox panics in exactly one scenario: when storage has been successfully committed but version tracking fails. This represents a critical data integrity issue where continuing could lead to corruption.

### Trade-offs
- **Verbose API**: More error checking code
- **Performance**: Small overhead for error handling
- **User Burden**: Callers must handle errors
- **Critical Safety**: One panic prevents data corruption

## Versioning: Explicit Lifecycle

### Decision
Versions have explicit states: Pending → Committed or Aborted.

### Rationale
From the version lifecycle decision:
- **Clear Semantics**: No ambiguity about version state
- **Resource Management**: Know when to clean up
- **Consistency**: Can't read from uncommitted versions
- **Debugging**: Clear audit trail of version changes

### State Machine
```
[New] → [Pending] → [Committed]
              ↓
          [Aborted]
```

### Trade-offs
- **API Complexity**: Must explicitly commit versions
- **Memory Usage**: Track state for all versions
- **Cleanup Burden**: Must handle aborted versions

## Value Limits: 1MB Maximum

### Decision
Enforce a 1MB maximum value size (`MaxValueSize = 1024 * 1024`).

### Rationale
- **DoS Prevention**: Prevent memory exhaustion attacks
- **Performance**: Keep operations predictable
- **Network Friendly**: Reasonable for transmission
- **Cache Friendly**: Values fit in reasonable cache sizes

### Trade-offs
- **Limited Use Cases**: Can't store large blobs directly
- **Application Complexity**: Must chunk large data
- **Storage Overhead**: May need external blob storage

## Performance: Write Optimization

### Decision
Optimize for write throughput over read latency.

### Rationale
- **System Workload**: More writes during state updates
- **LSM Alignment**: Writes are naturally fast in LSM trees
- **Batch Optimization**: Group writes for efficiency
- **Version Accumulation**: Many versions created quickly

### Implementation Choices
- Batch operations with optimization
- Append-only storage patterns
- Minimal write amplification
- Lazy deletion (tombstones)

### Trade-offs
- **Read Latency**: Reads may require multiple I/O operations
- **Space Usage**: Deleted data cleaned up asynchronously
- **Memory Pressure**: Write buffers consume memory

## API Design: Explicit Over Convenient

### Decision
Prefer explicit, verbose APIs over convenient shortcuts.

### Examples
```go
// Explicit version parameter
tree.Get(version, key)

// Not implicit "current version"
tree.Get(key)

// Explicit error handling
value, err := tree.Get(version, key)
if err != nil {
    return err
}

// Not panic on error
value := tree.MustGet(version, key)
```

### Rationale
- **No Surprises**: Behavior is always clear
- **Debugging**: Easy to trace what happened
- **Flexibility**: Callers have full control
- **Safety**: Can't accidentally use wrong version

### Trade-offs
- **Verbose Code**: More typing, more lines
- **Learning Curve**: Not immediately intuitive
- **Boilerplate**: Repetitive patterns

## Summary

These design decisions reflect a consistent philosophy:

1. **Production First**: Built for real deployments, not just demos
2. **Correctness Matters**: Better to be right than fast
3. **Maintenance Friendly**: Code should be maintainable for years
4. **No Magic**: Everything explicit and traceable

The result is a system that may not win micro-benchmarks but provides a solid foundation for building reliable, versioned storage systems. Every decision prioritizes long-term maintainability and operational reliability over short-term convenience or performance gains.