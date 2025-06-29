# Node Helpers and Utilities

## Purpose

Provides utility functions for working with nodes in the tree package. These helpers complement the node interfaces defined in the types package.

## Import Cycle Resolution

This file originally contained the Node interface definitions, which have been moved to the types package to resolve import cycles between codec and tree packages. Now contains only helper functions that work with the types.Node interface.

## Key Functions

- Type checking: `IsLeaf()`, `IsInternal()` - moved to types package
- Type casting: `AsLeaf()`, `AsInternal()` - safe type assertions for concrete types
- Helper utilities that work with types.Node interface

## Helper Functions

- Type checking: `IsLeaf()`, `IsInternal()`
- Type casting: `AsLeaf()`, `AsInternal()`, `AsLeafErr()`, `AsInternalErr()`
- Child validation: `IsEmpty()` checks for zero hash

## Extended Interfaces

- `NodeWithChildren`: For internal nodes with child operations
- `NodeWithKey`: For leaf nodes with key access
- `NodeCloneable`: For nodes supporting versioned cloning

## Common Operations

- `TraverseToLeaf()`: Follows key path down to leaf node (placeholder - needs storage)
- `ComputeRootHash()`: Gets hash from any node (nil-safe)
- `CountNodes()`: Recursively counts nodes in subtree (requires NodeLoader)
- `CloneNode()`: Generic node cloning with version update
- `KeyToNibblePath()`: Converts key to nibble sequence

## Visitor Pattern

- `NodeVisitor` interface for extensible node processing
- `Accept()` function dispatches visitor to correct node type
- Example `NodePrinter` visitor for debugging

## Implementation Notes

- **No IsLeaf() method**: Uses helper functions instead of interface methods
- **Immutability**: Nodes support Clone() for versioning, with limited mutable operations
- **No SetVersion()**: Version is immutable, set only during node creation
- **Factory Functions**: `NewNode()` placeholder for codec integration
- **Interface Compliance**: Both node types have compile-time interface checks
- **NodeLoader**: Interface defined for future storage integration
- **Storage Dependencies**: TraverseToLeaf and CountNodes need storage layer to fully function
