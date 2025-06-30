# Node Interfaces and Types

This file defines the core node interfaces that form the foundation of the Jellyfish Merkle Tree's type system. It was created to resolve an import cycle between the codec and tree packages.

## Architecture Decision

The interfaces in this file enable the codec package to work with nodes without importing the tree package, breaking what would otherwise be a circular dependency. The tree package implements these interfaces with concrete types.

## Key Interfaces

- **Node**: Base interface for all tree nodes, providing type identification, hashing, and version tracking
- **NodeWithChildren**: Extended interface for internal nodes that can have child nodes
- **NodeWithKey**: Extended interface for leaf nodes that store keys
- **NodeCloneable**: Interface for creating new versions of nodes

## Supporting Types

- **NodeType**: Enumeration distinguishing between internal and leaf nodes
- **Child**: Structure representing a reference to a child node with its hash and metadata
- **NodeVisitor**: Interface for implementing the visitor pattern on nodes

The factory pattern registration mechanism allows the tree package to provide concrete implementations at runtime while maintaining compile-time type safety.