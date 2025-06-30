# verifier.go

Implements cryptographic verification of Jellyfish Merkle Tree proofs, ensuring data integrity and non-repudiation.

## Key Components

- **Verifier**: Stateless proof verifier using the default hasher
- **Verify**: Main verification entry point that routes to specific proof type handlers
- **reconstructRoot**: Core algorithm that rebuilds root hash from leaf/empty position

## Verification Process

1. **Inclusion Proofs**: Computes leaf hash from value, reconstructs root through siblings
2. **Empty Exclusion**: Reconstructs from empty position where path leads to missing child
3. **Neighbor Exclusion**: Verifies neighbor position and reconstructs using neighbor's hash

## JMT Hash Format

Internal nodes follow JMT specification:
- Format: `NodeType || (nibble || hash)* || empty_hash*`
- Existing children include nibble prefix
- Empty children use EmptyTreeHash without nibble

## Critical Implementation Details

### Empty Exclusion Special Case
The most subtle part of verification - when proving a key doesn't exist:
- At the deepest sibling level, the target child position uses EmptyTreeHash
- This represents the empty child that would contain the key
- All other empty positions in internal nodes also use EmptyTreeHash
- This asymmetry is crucial for correct hash reconstruction

### Hash Compatibility
The verifier's hash computation must exactly match InternalNode.Hash():
- Same byte ordering for node type prefix
- Same nibble encoding (single byte)
- Same EmptyTreeHash constant (not EmptyHash)
- Any deviation causes verification failure

### Stateless Design
Verifier is stateless and thread-safe:
1. No tree access required during verification
2. Only needs proof data and expected root hash
3. Enables verification on different machines/processes
4. Supports concurrent verification of multiple proofs

### Security Considerations
- Validates neighbor key divergence to prevent forged neighbor proofs
- Checks depth bounds to prevent stack overflow
- Validates all inputs before processing
- Uses constant-time hash comparisons where applicable