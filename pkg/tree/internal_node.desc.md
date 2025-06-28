# Internal Node Implementation

## Purpose
Branching nodes in the JMT that can have up to 16 children (one per nibble). Forms the tree structure by connecting to other internal nodes or leaf nodes.

## Key Features
- **Sparse Children**: Uses map[Nibble]Child for memory efficiency
- **Hash Format**: NodeTypeInternal || child0 || ... || child15
- **Empty Children**: Missing children use EmptyTreeHash in hash computation
- **Clone Support**: Creates independent copy for versioning

## Design Decisions
- **Sparse Map**: More efficient than 16-element array for typical usage
- **Ordered Hashing**: Children always hashed in nibble order (0-15)
- **Empty Children**: Use `crypto.EmptyTreeHash` (not zero hash) for missing children
- **Cache Invalidation**: Any child modification invalidates hash cache
- **GetOnlyChild**: Optimization helper for single-child nodes

## Thread Safety
- RWMutex protects both children map and cached hash
- Children() returns defensive copy
- All mutations properly synchronized

## Performance
- Child operations are O(1) with map lookup
- Hash computation scales with number of children
- Clone operation is O(n) where n is number of children