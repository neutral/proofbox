---
id: adr.import-cycle-resolution
status: active
date: 2025-06-29
tags: [architecture, codec, tree, refactoring]
---

# ADR: Import Cycle Resolution Between Codec and Tree Packages

## Status

Accepted

## Context

During the implementation of Step 08 (Tree Structure), we encountered a circular dependency issue between the `codec` and `tree` packages:

- The `codec` package needed to import `tree` to access concrete node types (`LeafNode`, `InternalNode`) for encoding/decoding
- The `tree` package would need to import `codec` to decode nodes from storage
- This created an import cycle that prevented compilation in Go

This is a common architectural challenge when implementing serialization for domain objects, where the serialization layer needs to know about the domain types, but the domain also needs deserialization capabilities.

## Decision

We resolved the import cycle by implementing a three-layer architecture with interface-based encoding and the factory pattern:

### 1. Types Package (Foundation Layer)
- Created `/pkg/types/node.go` containing all node interfaces
- Moved shared types (`NodeType`, `Child`) to this package
- This package has no dependencies on other packages

### 2. Codec Package (Serialization Layer)
- Only imports the `types` package
- Works exclusively with interfaces, not concrete types
- Implements a factory pattern for node creation during decoding
- Provides interface-based encoding functions

### 3. Tree Package (Implementation Layer)
- Implements concrete types that satisfy the interfaces
- Registers a factory function with the codec package at initialization
- Provides helper functions for codec to construct nodes

## Implementation Details

### Interface Definitions

The `types` package defines:
```go
type Node interface {
    Type() NodeType
    Hash() Hash
    IsCached() bool
    Version() Version
}

type NodeWithChildren interface {
    Node
    Child(nibble Nibble) (Child, bool)
    NumChildren() int
    Children() map[Nibble]Child
}

type NodeWithKey interface {
    Node
    Key() Key
}

type NodeCloneable interface {
    Clone(newVersion Version) Node
}
```

### Factory Pattern

The codec package provides:
```go
type NodeFactory func(nodeType NodeType, data []byte, version Version) (Node, error)

func RegisterNodeFactory(f NodeFactory)
func DecodeNode(data []byte, version Version) (Node, error)
```

The tree package registers its factory:
```go
func init() {
    codec.RegisterNodeFactory(nodeFactory)
}

func nodeFactory(nodeType types.NodeType, data []byte, version types.Version) (types.Node, error) {
    switch nodeType {
    case types.NodeTypeLeaf:
        return decodeLeafNode(data, version)
    case types.NodeTypeInternal:
        return decodeInternalNode(data, version)
    default:
        return nil, fmt.Errorf("unknown node type: %v", nodeType)
    }
}
```

### Interface-Based Encoding

The codec uses only interface methods:
```go
func EncodeLeafNodeInterface(leaf types.LeafNodeInterface) []byte {
    key := leaf.Key()
    valueHash := leaf.ValueHash()
    // Encode using only interface methods
}
```

## Consequences

### Positive

- **No Import Cycles**: Dependencies flow in one direction: types → codec → tree
- **Clean Architecture**: Each package has a single, well-defined responsibility
- **Type Safety**: Strong typing maintained throughout with compile-time checks
- **Testability**: Each layer can be tested independently
- **Extensibility**: New node types can be added by implementing interfaces
- **Performance**: No runtime overhead in hot paths (encoding/decoding)

### Negative

- **Complexity**: Three packages instead of two
- **Indirection**: Factory pattern adds a layer of indirection
- **Registration**: Requires init-time registration of factory function

### Neutral

- **Interface Definitions**: All node contracts now live in the types package
- **Dependency Direction**: Clear dependency hierarchy must be maintained

## Alternatives Considered

1. **Merge Packages**: Combine codec and tree into a single package
   - Rejected: Would violate separation of concerns

2. **Reflection-Based Codec**: Use reflection to avoid compile-time dependencies
   - Rejected: Performance overhead and loss of type safety

3. **Generated Code**: Use code generation for serialization
   - Rejected: Added build complexity for minimal benefit

## References

- [Go FAQ: Import Cycles](https://go.dev/doc/faq#import_cycles)
- [Dependency Inversion Principle](https://en.wikipedia.org/wiki/Dependency_inversion_principle)
- Factory Pattern in Go