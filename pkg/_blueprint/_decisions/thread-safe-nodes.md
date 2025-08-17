---
id: adr.thread-safe-nodes
status: accepted
date: 2024-01-15
---

# ADR: Thread-Safe Node Implementation

## Status

Accepted

## Context

Nodes may be accessed concurrently by multiple goroutines:
- Readers accessing cached values
- Writers modifying children (InternalNode)
- Hash computation accessing fields

We need to decide on the concurrency model for node types.

## Decision

Implement nodes with built-in thread safety using sync.RWMutex:
- Each node has its own mutex
- Read operations use RLock()
- Write operations use Lock()
- Cached hash is protected by the same mutex

## Consequences

### Positive
- **Safe Concurrent Access**: Multiple readers, exclusive writers
- **No External Synchronization**: Callers don't need to manage locks
- **Cache Coherency**: Hash cache invalidation is atomic with mutations
- **Prevents Data Races**: Guaranteed by Go race detector

### Negative
- **Memory Overhead**: 24 bytes per mutex per node
- **Lock Contention**: Possible under high concurrency
- **Complexity**: Must carefully manage lock/unlock pairs

### Neutral
- Standard Go pattern for concurrent data structures
- Performance acceptable for tree operations

## Implementation Details

```go
type LeafNode struct {
    key       types.Key
    valueHash types.Hash
    value     []byte
    version   types.Version
    
    // Protects all fields including cachedHash
    mu         sync.RWMutex
    cachedHash *types.Hash
}

func (n *LeafNode) Hash() types.Hash {
    n.mu.RLock()
    if n.cachedHash != nil {
        defer n.mu.RUnlock()
        return *n.cachedHash
    }
    n.mu.RUnlock()
    
    // Compute hash...
    
    n.mu.Lock()
    n.cachedHash = &hash
    n.mu.Unlock()
    
    return hash
}
```

## Alternatives Considered

1. **External Synchronization**
   - Nodes have no locks, caller manages synchronization
   - Rejected: Error-prone, easy to miss locks

2. **Immutable Nodes**
   - Nodes never change after creation
   - Rejected: Need SetChild() for building trees, SetValue() for loading

3. **Copy-on-Write**
   - Return new nodes on every modification
   - Rejected: Excessive allocations, GC pressure

4. **Lock-Free Algorithms**
   - Use atomic operations only
   - Rejected: Too complex for tree structures

5. **Channel-Based Access**
   - All node access through channels
   - Rejected: Performance overhead, complexity

## References

- sync.RWMutex documentation
- Observed no race conditions with -race flag
- Common pattern in Go standard library (sync.Map)