# generator.go

Implements proof generation for the Jellyfish Merkle Tree, creating cryptographic proofs for key existence or non-existence.

## Key Components

- **Generator**: Main proof generator that traverses the tree to collect sibling data
- **Generate**: Creates appropriate proof type based on tree traversal results
- **GenerateBatch**: Efficiently generates multiple proofs (with optimization potential)

## Proof Generation Logic

1. **Empty Tree**: Returns ExclusionEmpty proof immediately
2. **Tree Traversal**: Follows nibble path collecting siblings up to depth 64
3. **Leaf Found**: 
   - Matching key: Inclusion proof with value
   - Different key: Neighbor exclusion proof
4. **Empty Child**: Returns ExclusionEmpty proof at that depth

## Special Handling

### Depth 64 Edge Case
The main loop only processes 64 levels (0-63), but leaf nodes can exist at depth 64:
- After main traversal, explicitly check for leaf at depth 64
- This handles the case where two keys share the same 256-bit prefix
- Without this check, such keys would incorrectly produce ExclusionEmpty proofs

### TreeReaderInterface Design
Uses interface instead of concrete Tree type to:
1. Avoid circular dependency between tree and proof packages
2. Enable proof generation from different tree implementations
3. Support mock implementations for testing
4. Allow future read-only tree snapshots

### Sibling Collection Strategy
Collects full Children map at each level rather than just non-empty siblings:
1. Preserves nibble → hash mapping needed for verification
2. Avoids sorting complexity in verifier
3. Trades proof size for verification simplicity

### Value Loading
Values are loaded separately via LoadValue because:
1. Leaf nodes store only value hash for space efficiency
2. Enables content-addressed value storage
3. Supports future value compression/encryption
4. Allows lazy loading of large values