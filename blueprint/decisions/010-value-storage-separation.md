---
id: adr.010.value-storage-separation
status: accepted
date: 2024-01-15
---

# ADR-010: Separation of Tree Structure and Value Storage

## Status

Accepted

## Context

The Jellyfish Merkle Tree needs to store both tree structure (nodes) and actual values. We need to decide whether to:
1. Store values inline with tree nodes
2. Separate tree structure from value storage

This affects performance, flexibility, and memory usage patterns.

## Decision

Implement a **two-tier storage architecture**:
1. **Tree Storage**: Stores only nodes with value hashes (not values)
2. **Value Storage**: Stores actual values separately, indexed by key or hash

Tree nodes use our optimized binary codec, while values can use any serialization format.

## Architecture

```
┌─────────────────┐     ┌──────────────────┐
│   Tree Layer    │     │  Value Layer     │
├─────────────────┤     ├──────────────────┤
│ Fixed Format    │     │ Flexible Format  │
│ (Binary Codec)  │     │ (Proto/MsgPack)  │
├─────────────────┤     ├──────────────────┤
│ Only Hashes     │     │ Actual Data      │
│ Fast Access     │     │ Lazy Loading     │
└─────────────────┘     └──────────────────┘
         │                       │
         └───────────┬───────────┘
                     │
              ┌─────────────┐
              │  PebbleDB   │
              │  Storage    │
              └─────────────┘
```

## Storage Layout

```go
// Tree nodes (fast binary format):
"node:<version>:<node_hash>" -> [NodeType][Key][ValueHash]  // 65 bytes fixed

// Values (flexible format):
"value:<key>" -> [Format][SerializedData]  // Variable size
```

## Consequences

### Positive

- **Performance**: Tree operations only need hashes (fast)
- **Memory Efficiency**: Can traverse without loading values
- **Format Flexibility**: Values can use any serialization
- **Evolution**: Value format can change without tree changes
- **Cache Optimization**: Node cache doesn't hold large values
- **Proof Generation**: Merkle proofs only need hashes

### Negative

- **Complexity**: Two storage paths to manage
- **Extra I/O**: Additional read when value needed
- **Consistency**: Must ensure hash matches value
- **Storage Overhead**: Duplicate key storage

### Neutral

- Standard pattern in database systems
- Similar to B-tree leaf pointers
- Trade-off between speed and flexibility

## Implementation Example

```go
// 1. Writing a value
user := &UserProfile{Name: "Alice", Email: "alice@example.com"}
valueBytes := protobuf.Marshal(user)
valueHash := crypto.Hash(valueBytes)

// Tree stores only hash
tree.Insert(key, valueHash)

// Value stored separately
storage.Put("value:" + key, valueBytes)

// 2. Reading a value
node := tree.Get(key)  // Fast: only tree traversal
valueHash := node.ValueHash()

// Load value when needed (lazy)
if needValue {
    valueBytes := storage.Get("value:" + key)
    user := &UserProfile{}
    protobuf.Unmarshal(valueBytes, user)
}

// 3. Generating proof (no value needed)
proof := tree.GenerateProof(key)  // Uses only hashes
```

## Format Selection Guidelines

**Use Tree's Binary Format For**:
- All tree nodes (Internal, Leaf)
- Maximum performance needed
- Fixed-size structures

**Use Flexible Value Format For**:
- Application data
- Evolving schemas
- Cross-language compatibility

## Migration Path

1. Start with raw bytes in value storage
2. Add format prefix byte when structure needed
3. Support multiple formats simultaneously
4. Map key prefixes to formats if needed

## Security Considerations

- Always verify value hash matches stored hash
- Consider encryption at value layer
- Access control can be at value level

## Performance Impact

Benchmarks show:
- Tree operations: ~10ns (hash only)
- Value loading: +100-500ns (depends on size/format)
- Proof generation: No impact (hash only)
- Memory usage: 10-100x reduction for large values

## References

- Similar to RocksDB's BlobDB for large values
- LevelDB separates large values automatically
- Implements lazy loading pattern from ADR-009
- Complements binary codec decision (ADR-002)