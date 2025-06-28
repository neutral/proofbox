# Hash Values Distinction in Jellyfish Merkle Tree

## Overview
This document clarifies the cryptographic distinction between different hash values used in the JMT implementation, specifically `DefaultDigest` and `EmptyHash`. Understanding this distinction is critical for maintaining cryptographic integrity and proper tree semantics.

## Hash Value Types

### DefaultDigest (SHA-256 of empty string)
**Purpose**: The actual cryptographic hash of empty data
- **Usage**: When hashing empty values or empty byte arrays
- **Computation**: `SHA-256("")`
- **Value**: `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`
- **Semantic meaning**: "This leaf contains empty data"

### EmptyHash/EmptyTreeHash (Special constant)
**Purpose**: Represents empty/null nodes in the sparse Merkle tree structure
- **Usage**: Placeholder for missing branches in the tree
- **Computation**: Predefined constant (not derived from hashing)
- **Value**: `5ba93c9db0cff93f52b521d7814c7fa08abe86135f746b4668b013bcd1c48e9b`
- **Semantic meaning**: "No node exists at this position"

## Security Implications

The distinction between these values is crucial for security:

1. **Collision Prevention**: Using different values prevents semantic collisions between:
   - A leaf node storing empty data (uses DefaultDigest)
   - An absent node in the tree (uses EmptyHash)

2. **Proof Integrity**: Merkle proofs must correctly distinguish between:
   - Proving a key exists with empty value
   - Proving a key does not exist

3. **Tree Consistency**: The sparse tree optimization relies on EmptyHash being:
   - Deterministic across all implementations
   - Distinct from any possible data hash
   - Constant regardless of tree state

## Implementation Requirements

1. **Never mix these values** - they serve different cryptographic purposes
2. **EmptyHash must be hardcoded** - not computed dynamically
3. **DefaultDigest must match standard SHA-256** - for interoperability

This separation ensures the tree maintains both cryptographic integrity and semantic correctness.