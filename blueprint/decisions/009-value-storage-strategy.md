---
id: adr.009.value-storage
status: accepted
date: 2024-01-15
---

# ADR-009: Leaf Node Value Storage Strategy

## Status

Accepted

## Context

Leaf nodes need to store both the value hash (for tree hashing) and optionally the actual value data. We need to decide:
1. Whether to always store values in nodes
2. How to handle lazy loading from storage
3. Maximum value size limits

## Decision

- Always store value hash (32 bytes) in the node
- Actual value is optional (supports lazy loading)
- Provide `SetValue()` method that validates hash before accepting value
- Maximum value size: 1MB (prevent DoS attacks)
- Values can be loaded separately from node structure

## Consequences

### Positive
- **Flexibility**: Can load node structure without values
- **Memory Efficiency**: Large values not always in memory
- **Integrity**: SetValue validates hash match
- **DoS Prevention**: 1MB limit prevents memory exhaustion
- **Performance**: Can compute tree hashes without loading values

### Negative
- **Complexity**: Two-phase loading (node then value)
- **Extra State**: Must track whether value is loaded
- **API Surface**: Extra SetValue method

### Neutral
- Common pattern in database systems
- Trade-off between memory and I/O

## Implementation Details

```go
type LeafNode struct {
    key       types.Key
    valueHash types.Hash
    value     []byte      // May be nil if not loaded
    version   types.Version
    // ...
}

// SetValue loads the actual value (after validation)
func (n *LeafNode) SetValue(value []byte) error {
    if crypto.DefaultHasher.Hash(value) != n.valueHash {
        return types.ErrHashMismatch
    }
    n.value = value
    return nil
}

// Value returns the actual value (may be nil if not loaded)
func (n *LeafNode) Value() []byte {
    return n.value
}

// ValueHash always returns the hash (always available)
func (n *LeafNode) ValueHash() types.Hash {
    return n.valueHash
}
```

## Alternatives Considered

1. **Always Load Values**
   - Simpler API but wastes memory
   - Rejected: Inefficient for large values

2. **Separate Value Storage**
   - Store values in different system entirely
   - Rejected: Complicates consistency

3. **Streaming Values**
   - Return io.Reader for values
   - Rejected: Over-engineered for current needs

4. **No Size Limit**
   - Allow arbitrary value sizes
   - Rejected: DoS vulnerability

5. **Smaller Limit (e.g., 64KB)**
   - More restrictive size
   - Rejected: 1MB reasonable for most use cases

## Future Considerations

- Could add value compression
- Might support streaming for very large values
- Consider separate column family for values > 256 bytes

## References

- Similar pattern in LevelDB/RocksDB
- types.MaxValueSize = 1MB constant
- Implemented in pkg/tree/leaf_node.go