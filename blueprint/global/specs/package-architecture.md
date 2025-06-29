# Package Architecture Specification

## Overview

ProofBox follows a layered package architecture designed to maintain clean separation of concerns and avoid circular dependencies. This document describes the package structure and dependency rules.

## Package Layers

### Layer 1: Foundation (No Dependencies)

#### `pkg/types`
**Purpose**: Core type definitions and interfaces shared across all packages

**Contains**:
- Basic types: `Key`, `Hash`, `Version`, `Nibble`
- Node interfaces: `Node`, `NodeWithChildren`, `NodeWithKey`, `NodeCloneable`
- Common structures: `Child`, `NodeType`, `NibblePath`
- Validation functions and constants
- Error definitions

**Dependency Rule**: NO dependencies on other ProofBox packages

#### `pkg/crypto`
**Purpose**: Cryptographic primitives and hashing

**Contains**:
- `Hasher` interface and implementations
- Hash computation utilities
- Cryptographic constants

**Dependency Rule**: Only depends on `types` package

### Layer 2: Core Services

#### `pkg/codec`
**Purpose**: Serialization and deserialization of nodes

**Contains**:
- Binary encoding/decoding functions
- Interface-based encoders
- Factory pattern for node creation
- Batch encoding support

**Dependencies**: `types`, `crypto`

**Key Design**: Uses interfaces from `types` package to avoid importing concrete implementations

#### `pkg/storage`
**Purpose**: Storage abstraction and key management

**Contains**:
- Storage interfaces
- Key encoding/decoding
- PebbleDB wrapper (future)

**Dependencies**: `types`

### Layer 3: Domain Implementation

#### `pkg/tree`
**Purpose**: Jellyfish Merkle Tree implementation

**Contains**:
- Concrete node types: `LeafNode`, `InternalNode`
- Tree operations: `Get`, `Insert`, `Update`, `Delete`
- Proof generation and verification
- Node cache implementation
- Factory registration for codec

**Dependencies**: `types`, `crypto`, `codec`, `storage`

**Key Design**: Registers node factory with codec package at initialization

## Dependency Flow

```
┌─────────────┐
│   cmd/*     │  Layer 4: Applications
└──────┬──────┘
       │
┌──────▼──────┐
│  pkg/tree   │  Layer 3: Domain
└──────┬──────┘
       │
┌──────▼──────┐  ┌─────────────┐
│ pkg/codec   │  │ pkg/storage │  Layer 2: Services
└──────┬──────┘  └──────┬──────┘
       │                 │
┌──────▼─────────────────▼──────┐
│         pkg/types             │  Layer 1: Foundation
└───────────────┬───────────────┘
                │
        ┌───────▼────────┐
        │  pkg/crypto    │
        └────────────────┘
```

## Interface-Based Decoupling

### Problem
The codec package needs to encode/decode tree nodes, but importing the tree package would create a circular dependency.

### Solution
1. Define node interfaces in `types` package
2. Codec works with interfaces only
3. Tree package implements the interfaces
4. Tree registers a factory function with codec at initialization

### Implementation

```go
// In types/node.go
type Node interface {
    Type() NodeType
    Hash() Hash
    IsCached() bool
    Version() Version
}

// In codec/factory.go
type NodeFactory func(nodeType NodeType, data []byte, version Version) (Node, error)
var nodeFactory NodeFactory

func RegisterNodeFactory(f NodeFactory) {
    nodeFactory = f
}

// In tree/codec_factory.go
func init() {
    codec.RegisterNodeFactory(nodeFactory)
}
```

## Package Guidelines

### 1. Import Rules
- Lower layers MUST NOT import higher layers
- Packages at the same layer SHOULD minimize cross-imports
- Use interfaces to break circular dependencies

### 2. Interface Location
- Interfaces should be defined in the lowest possible layer
- Consumer defines the interface when possible
- Shared interfaces go in `types` package

### 3. Type Location
- Basic types and constants: `types` package
- Domain-specific types: Domain package
- Helper types: With their primary usage

### 4. Error Handling
- Package-specific errors defined in that package
- Common errors in `types` package
- Wrap errors with context when crossing boundaries

## Adding New Packages

When adding a new package:

1. **Determine Layer**: Which layer does it belong to?
2. **Check Dependencies**: What packages will it need?
3. **Define Interfaces**: What contracts does it need?
4. **Avoid Cycles**: Can dependencies flow one way?
5. **Document Purpose**: Clear package documentation

## Benefits

1. **No Circular Dependencies**: Clean, one-way dependency flow
2. **Testability**: Each package can be tested in isolation
3. **Maintainability**: Clear boundaries and responsibilities
4. **Flexibility**: Easy to swap implementations
5. **Compilation Speed**: Minimal recompilation on changes

## Anti-Patterns to Avoid

1. **God Package**: Putting too much in one package
2. **Circular Imports**: A imports B imports A
3. **Layer Violation**: Lower layer importing higher
4. **Interface Pollution**: Too many small interfaces
5. **Type Duplication**: Same type defined multiple places