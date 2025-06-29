# Leaf Node Implementation

## Purpose
Terminal nodes in the JMT that store actual key-value mappings. Each leaf represents a single key-value pair at a specific version.

## Key Features
- **Value Storage**: Stores both value hash and optional actual value
- **Lazy Loading**: Values can be loaded separately from node structure
- **Hash Format**: NodeTypeLeaf || key || valueHash (deterministic)
- **Thread-Safe**: Uses RWMutex for concurrent access

## Storage Strategy
- Always stores value hash for integrity verification
- Actual value is optional (supports lazy loading from storage)
- SetValue() validates hash before accepting value
- Maximum value size: 1MB to prevent DoS

## Mutability Pattern
- **SetValue()**: Used only for loading values from storage, validates hash
- **Clone()**: Creates new node with different version, preserving all data
- **Immutable Updates**: Leaf nodes are replaced entirely during tree updates
- **Thread Safety**: SetValue is not thread-safe by design (called during loading)

## Interface Implementation
- Implements `types.Node` interface (Type, Hash, IsCached, Version)
- Implements `types.NodeWithKey` interface (Key accessor)
- Implements `types.NodeCloneable` interface (Clone method)
- Implements `types.LeafNodeInterface` for full functionality
- Compile-time interface checks ensure compliance
- Interfaces defined in types package to avoid import cycles

## Performance
- Hash computation cached after first calculation
- Benchmarks show ~278ns for uncached hash computation
- Cached hash access is essentially free (RLock only)
- Clone operation is O(1) with structural sharing