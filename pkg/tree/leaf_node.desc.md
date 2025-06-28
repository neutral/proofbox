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

## Performance
- Hash computation cached after first calculation
- Benchmarks show ~1μs for uncached hash computation
- Cached hash access is essentially free (RLock only)