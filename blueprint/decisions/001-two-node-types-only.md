---
id: adr.001.two-node-types
status: accepted
date: 2024-01-15
---

# ADR-001: Two Node Types Only (Leaf and Internal)

## Status

Accepted

## Context

The Jellyfish Merkle Tree specification allows for different node type configurations. Some Merkle tree implementations use three node types (internal, extension, and leaf), where extension nodes optimize for common key prefixes.

## Decision

We will implement the JMT with only two node types:
- **LeafNode**: Terminal nodes containing key-value data
- **InternalNode**: Branching nodes with up to 16 children (radix-16)

No extension nodes or other optimization node types will be used.

## Consequences

### Positive
- **Simplicity**: Fewer node types means simpler code and fewer edge cases
- **Easier Testing**: Only two node types to test and validate
- **Cleaner Interface**: The Node interface remains minimal
- **Reduced Complexity**: No special handling for extension node traversal or updates

### Negative
- **Potentially Deeper Trees**: Without extension nodes, trees may be deeper for keys with common prefixes
- **More Internal Nodes**: Each level requires a full internal node even for single-child cases
- **Higher Memory Usage**: Internal nodes with single children still allocate full structure

### Neutral
- Performance impact is minimal for typical blockchain use cases with random key distribution
- The radix-16 branching factor already provides good height/width balance

## Implementation Details

```go
type NodeType byte

const (
    NodeTypeInternal NodeType = 0x00
    NodeTypeLeaf     NodeType = 0x01
)
```

## Alternatives Considered

1. **Three Node Types (with Extension Nodes)**
   - Would optimize for common prefixes
   - Adds significant complexity to tree operations
   - Rejected for initial implementation

2. **Adaptive Node Types**
   - Dynamically converting between node types based on children count
   - Too complex for the benefits provided

## References

- Jellyfish Merkle Tree paper
- Ethereum MPT (uses extension nodes)
- Our implementation in pkg/tree/