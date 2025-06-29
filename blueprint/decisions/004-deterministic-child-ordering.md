---
id: adr.004.deterministic-child-ordering
status: accepted
date: 2024-01-15
---

# ADR-004: Deterministic Child Ordering in Encoding

## Status

Accepted

## Context

Internal nodes store children in a map, which has non-deterministic iteration order in Go. For network proofs and consistent hashing, we need deterministic encoding.

## Decision

Always encode children in ascending nibble order (0x0 to 0xF) regardless of insertion order or map iteration order.

## Consequences

### Positive
- **Deterministic Hashing**: Same logical node always produces same hash
- **Network Compatibility**: Nodes can verify proofs from other nodes
- **Reproducible Storage**: Same tree state produces same storage bytes
- **Debugging**: Consistent output aids debugging and testing

### Negative  
- **Encoding Overhead**: Must sort nibbles before encoding (small cost)
- **Extra Allocation**: Need slice for sorted nibbles
- **Slight Complexity**: Encoding logic includes sorting step

### Neutral
- Decoding can insert children in any order since maps are unordered
- Performance impact minimal (max 16 nibbles to sort)

## Implementation Details

```go
func EncodeInternalNode(node *InternalNode) ([]byte, error) {
    children := node.Children()
    
    // Sort nibbles for deterministic ordering
    nibbles := make([]types.Nibble, 0, len(children))
    for nibble := range children {
        nibbles = append(nibbles, nibble)
    }
    sort.Slice(nibbles, func(i, j int) bool {
        return nibbles[i] < nibbles[j]
    })
    
    // Encode in sorted order
    for _, nibble := range nibbles {
        child := children[nibble]
        // encode child...
    }
}
```

## Alternatives Considered

1. **Natural Map Order**
   - Use whatever order Go's map iteration provides
   - Rejected: Non-deterministic, breaks network compatibility

2. **Insertion Order Tracking**
   - Maintain slice tracking insertion order
   - Rejected: Adds complexity, more memory, not needed

3. **Fixed Array Instead of Map**
   - Use [16]Child array instead of map
   - Rejected: Wastes memory for sparse nodes

4. **Canonical Ordering in Node**
   - Keep children always sorted in the node
   - Rejected: Complicates child mutations, performance impact

## References

- Go maps have intentionally randomized iteration
- Similar requirement in Ethereum RLP encoding
- Critical for proof verification across nodes