---
id: adr.version-not-serialized
status: accepted
date: 2024-01-15
---

# ADR: Version Not Part of Serialized Node Data

## Status

Accepted

## Context

When persisting nodes to storage, we need to decide whether to include the version number in the serialized node data. The version is already part of the NodeKey used to store the node.

## Decision

Do NOT include the version in the serialized node data. The version is provided externally when decoding nodes.

## Consequences

### Positive
- **Smaller Storage**: Saves 8 bytes per node (version is uint64)
- **No Redundancy**: Version isn't stored twice (in key and value)
- **Cleaner Separation**: Version is a storage concern, not a node property
- **Flexibility**: Storage layer controls versioning strategy

### Negative
- **Decoding Requires Version**: Must provide version parameter when decoding
- **API Complexity**: Decode functions need extra parameter
- **Potential Confusion**: Node has version field but it's not serialized

### Neutral
- Version is always available from NodeKey when loading from storage
- Consistent with how other storage systems handle metadata

## Implementation Details

Encoding (version not included):
```go
// LeafNode: [Key(32)] [ValueHash(32)] = 64 bytes
// InternalNode: [NumChildren(1)] [Children(N*42)]

func DecodeLeafNode(data []byte, version types.Version) (*LeafNode, error)
func DecodeInternalNode(data []byte, version types.Version) (*InternalNode, error)
```

Storage layer provides version:
```go
// When loading from storage
nodeKey := types.NodeKey{Version: 100, Path: ...}
data := storage.Get(nodeKey)
node := codec.DecodeNode(data, nodeKey.Version)
```

## Alternatives Considered

1. **Include Version in Serialized Data**
   - Pros: Self-contained nodes, simpler decode API
   - Cons: Redundant storage, version in two places
   - Rejected: Wasteful and creates consistency concerns

2. **Version as Node Metadata**
   - Store version in separate metadata structure
   - Rejected: Overcomplicates storage model

3. **Immutable Nodes Without Version**
   - Nodes don't track version at all
   - Rejected: Version needed for cache invalidation and debugging

## References

- NodeKey design includes version
- Similar pattern in other versioned stores (RocksDB, BadgerDB)
- Step 07 codec implementation