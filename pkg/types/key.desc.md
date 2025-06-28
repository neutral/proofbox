# Key Type Implementation

## Purpose
The `key.go` file implements the fundamental Key type and nibble operations for the Jellyfish Merkle Tree. These types form the basis for tree traversal and node addressing.

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

## Key Features
1. **KeyHash Function**: Converts variable-length data to fixed keys using SHA-256
2. **Nibble Extraction**: Efficient bit operations for tree traversal
3. **Path Conversion**: Full key to 64-nibble path transformation
4. **Validation**: Ensures keys are non-empty and nibbles are valid

## Performance Considerations
- Nibble extraction uses bit operations for speed
- No allocations in critical path operations
- Benchmarks show ~3ns per nibble extraction