# Proof Verification Specification

## Overview
Proof verification allows clients to confirm that a key-value pair is included in (or excluded from) the committed state without accessing the full tree.

## Verification Process

### Inclusion Proof Verification
Given a proof claiming key `k` maps to value `v`:

1. **Verify Leaf**
   - Check that provided leaf contains key `k`
   - Verify value hash matches claimed value
   - Compute leaf hash: `H(k || H(v))`

2. **Reconstruct Root**
   - Starting from leaf hash
   - For each level up to root:
     - Insert sibling hashes at correct positions
     - Use default hash for empty siblings
     - Hash all 16 children to get parent hash
   - Continue until root level

3. **Verify Root**
   - Compare reconstructed root with known state commitment
   - If match: Proof is valid, key-value pair confirmed

### Exclusion Proof Verification

#### Neighbor Proof
1. Verify provided leaf `L` has different key than `k`
2. Confirm keys share common prefix up to divergence
3. Reconstruct root using leaf `L` and siblings
4. Verify reconstructed root matches commitment
5. Conclude: `k` cannot exist (would conflict with `L`)

#### Empty Subtree Proof
1. Identify point where path to `k` encounters empty child
2. Use default hash at that position
3. Reconstruct root with siblings and default
4. Verify reconstructed root matches commitment
5. Conclude: `k` path leads to empty subtree

## Critical Requirements

### Deterministic Hashing
- Same hash function as tree construction
- Consistent child ordering (0-15)
- Agreed default hash constant

### Proof Completeness
Verifier must be able to distinguish:
- Inclusion vs exclusion
- Neighbor vs empty subtree exclusion
- Correct positioning of siblings

## Performance
- Verification time: O(proof path length)
- Typically much faster than tree traversal
- Suitable for resource-constrained clients
- Target: < 300μs on modern hardware (see NFR)