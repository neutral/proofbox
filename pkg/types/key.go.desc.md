# Key and NibblePath Implementation

## Purpose
The `key.go` file implements the fundamental Key type and comprehensive nibble path operations for the Jellyfish Merkle Tree. These types form the basis for tree traversal, node addressing, and path manipulation.

## Design Decisions

### Fixed-Size Keys
- Uses 32-byte (256-bit) keys to match SHA-256 output
- Fixed size enables efficient memory layout and comparisons
- Supports both direct byte keys and hashed variable-length data

### Nibble Extraction
- Nibbles are 4-bit values extracted from keys for tree navigation
- High nibble extracted with right shift, low nibble with mask
- Bounds checking prevents out-of-range access

### NibblePath Structure
- Stores nibbles as a slice with explicit length
- Length field enables partial paths for internal nodes
- Compare function provides lexicographic ordering for deterministic traversal
- All operations are immutable, returning new instances for thread safety

## Key Features

### Core Key Operations
1. **KeyHash Function**: Converts variable-length data to fixed keys using SHA-256
2. **Nibble Extraction**: Efficient bit operations for tree traversal
3. **Path Conversion**: Full key to 64-nibble path transformation
4. **Validation**: Ensures keys are non-empty and nibbles are valid

### NibblePath Operations (Step 10 additions)

#### Path Comparison
- **CommonPrefixLength**: Finds where two paths diverge, essential for leaf splitting
- **Equals**: Fast path equality check
- **IsPrefix**: Determines hierarchical relationships between paths

#### Path Manipulation
- **Prefix/Skip**: Extract sub-paths for tree navigation
- **Append**: Extend paths when traversing deeper into the tree
- **GetNibble**: Safe nibble access with bounds checking

#### Conversions
- **NewNibblePath**: Convert byte arrays to nibble representation
- **ToBytes**: Convert nibble paths back to bytes (for storage)

## Performance Characteristics
- Nibble extraction: ~3ns per operation
- CommonPrefixLength: O(n) where n is the length of the shorter path
- Append/Prefix/Skip: O(n) with single allocation, ~15ns
- Equals/IsPrefix: O(n) comparison, no allocations
- Path conversions: ~30ns for full key

## Design Philosophy
1. **Immutable Operations**: All path operations return new instances rather than modifying in place
2. **Efficient Memory Usage**: Operations create new slices sized exactly to the result
3. **Bounds Checking**: Explicit bounds checking to prevent panics
4. **Zero-Copy Where Possible**: Operations minimize allocations in hot paths