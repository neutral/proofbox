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
- Type casting: `AsLeaf()`, `AsInternal()`
- Child validation: `IsEmpty()` checks for zero hash