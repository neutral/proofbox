# Proof Package

The proof package implements Merkle proof generation and verification for the Jellyfish Merkle Tree.

## Design Principles

1. **Proof Types**: Three distinct proof types handle all verification scenarios:
   - Inclusion: Proves a key-value pair exists
   - Exclusion Empty: Proves a key maps to an empty subtree
   - Exclusion Neighbor: Proves non-existence by showing a different leaf

2. **Security First**: All inputs are validated, proof sizes are limited, and verification is deterministic

3. **Optimization Ready**: Structure supports future batch proof optimizations and compression

## Key Design Decisions

### Sibling Data Structure
Unlike the blueprint, we include a `Children` map in `SiblingData`. This is necessary because JMT internal nodes hash all 16 children in order, not just non-empty ones. The verifier needs to know which positions are empty vs populated.

### Path Tracking
Neighbor proofs explicitly track the divergence depth and common path, making verification more straightforward and avoiding path reconstruction errors.

### Proof Size Limits
`MaxProofSize` prevents DoS attacks from maliciously large proofs while being generous enough for legitimate deep tree proofs.