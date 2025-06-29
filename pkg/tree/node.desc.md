# Node Interface

## Purpose

Defines the core interface for all node types in the Jellyfish Merkle Tree. The JMT uses only two node types (Leaf and Internal) to maintain simplicity.

## Design Decisions

- **Minimal Interface**: Only essential methods that all nodes must implement
- **Thread-Safe Hash Caching**: Nodes cache computed hashes for performance
- **Child References**: Use hash + version + type flag for efficient storage
- **No Extension Nodes**: Following JMT design principle of simplicity

## Key Types

- `NodeType`: Enum identifying node type (Internal=0x00, Leaf=0x01)
- `Node`: Core interface with Type(), Hash(), IsCached(), Version()
- `Child`: Reference to a child node containing hash, version, and leaf flag

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
