---
id: adr.separate-codec
status: accepted
date: 2024-01-15
---

# ADR: Separate Codec Package

## Status

Accepted

## Context

Node serialization/deserialization logic needs to be implemented for storage persistence. We need to decide whether to:
1. Include codec methods directly in the tree package
2. Create a separate codec package
3. Use a generic serialization library

## Decision

Create a separate `pkg/codec/` package for all encoding/decoding logic, with the tree package providing minimal factory functions for codec support.

## Consequences

### Positive
- **Clean Separation of Concerns**: Tree logic is separate from serialization logic
- **Easier Testing**: Can test tree operations independently from serialization
- **Flexibility**: Can swap or extend codec implementations without touching tree code
- **Encapsulation**: Private fields remain private; codec uses factory functions
- **Single Responsibility**: Each package has one clear purpose

### Negative
- **Additional Package**: More packages to maintain
- **Factory Functions**: Need to expose factory functions for codec use
- **Cross-Package Coupling**: Codec package depends on tree package types

### Neutral
- Performance impact is negligible (no interface indirection in hot path)
- Slightly more complex build but better long-term maintainability

## Implementation Details

Tree package provides factory functions:
```go
// pkg/tree/codec_support.go
func NewLeafNodeFromCodec(key types.Key, valueHash types.Hash, version types.Version) *LeafNode
func NewInternalNodeFromCodec(children map[types.Nibble]Child, version types.Version) *InternalNode
```

Codec package handles all serialization:
```go
// pkg/codec/
├── leaf.go      // LeafNode encoding/decoding
├── internal.go  // InternalNode encoding/decoding
├── node.go      // Generic node codec
└── batch.go     // Batch operations
```

## Alternatives Considered

1. **Methods on Node Types**
   ```go
   func (n *LeafNode) Encode() []byte
   func (n *LeafNode) Decode([]byte) error
   ```
   - Rejected: Mixes concerns, makes nodes aware of serialization

2. **Generic Serialization Library (protobuf, msgpack)**
   - Rejected: Overhead, external dependencies, less control
   - Need deterministic encoding for hashing

3. **Codec Interface in Tree Package**
   - Rejected: Still mixes concerns even with interface

## References

- Step 07 in blueprint
- Similar pattern in BadgerDB (separate `pb` package)
- Go standard library (encoding/json is separate from types)