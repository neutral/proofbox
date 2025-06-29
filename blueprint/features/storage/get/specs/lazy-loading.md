# Lazy Value Loading Specification

## Overview
Leaf nodes in the JMT support lazy loading of values to optimize memory usage and I/O operations. This allows the tree structure to be traversed without loading all value data into memory.

## Design Rationale

The separation of value storage from tree structure enables:
1. **Memory Efficiency**: Tree operations often only need value hashes, not full values
2. **I/O Optimization**: Values loaded only when actually needed
3. **Proof Generation**: Merkle proofs only require hashes, not values
4. **Cache Flexibility**: Node cache can evict values while keeping structure

## Implementation Details

### LeafNode Structure
```go
type LeafNode struct {
    key       types.Key     // Always present
    valueHash types.Hash    // Always present for integrity
    value     []byte        // May be nil (lazy loaded)
    version   types.Version // Always present
}
```

### Loading Strategy

1. **Initial Load from Storage**
   - Storage layer returns nodes with `valueHash` populated
   - `value` field may be nil to defer loading
   - Tree operations proceed using only the hash

2. **On-Demand Loading**
   - `Value()` method returns current value (may be nil)
   - `SetValue()` validates hash before accepting value
   - Hash mismatch returns `ErrHashMismatch`

3. **Memory Management**
   - Cache eviction can nil out values
   - Retains valueHash for integrity verification
   - Values reloaded on next access if needed

### Usage Patterns

```go
// Storage returns node without value
leafNode := storage.LoadNode(key) // value is nil

// Check if value is loaded
if leafNode.Value() == nil {
    // Load value from separate value storage
    value := storage.LoadValue(leafNode.ValueHash())
    err := leafNode.SetValue(value) // Validates hash
}

// Use value
data := leafNode.Value()
```

## Security Considerations

1. **Hash Verification**: SetValue() MUST verify the provided value matches valueHash
2. **Immutability**: Once set, values should not change (new version required)
3. **Consistency**: Tree operations must work correctly with nil values

## Performance Impact

- **Positive**: Reduced memory usage, faster tree traversal
- **Negative**: Additional I/O when values are needed
- **Mitigation**: Strategic caching based on access patterns

## Integration Points

1. **Storage Layer**: Must support separate value storage/retrieval
2. **Cache Manager**: Must handle nodes with/without values
3. **Proof System**: Must work with value hashes only
4. **Get Operations**: Must trigger value loading when needed

## Storage Architecture

Values are stored separately from tree nodes to enable:
- Different serialization formats for values
- Lazy loading without affecting tree performance
- Evolution of value schemas without tree changes

See `/blueprint/decisions/010-value-storage-separation.md` for detailed rationale and `/blueprint/global/specs/storage-architecture.md` for implementation details.