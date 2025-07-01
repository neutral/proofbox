# Security Model

This document explains ProofBox's security model, including cryptographic guarantees, threat model, trust assumptions, and best practices for secure usage.

## Overview

ProofBox provides cryptographic integrity for key-value data through:
- **Merkle Tree Structure**: Tamper-evident data organization
- **Cryptographic Proofs**: Verifiable evidence of data state
- **SHA-256 Hashing**: Industry-standard cryptographic hash function
- **Version Integrity**: Immutable historical states

## Cryptographic Foundation

### Hash Function

ProofBox uses SHA-256 for all cryptographic operations:
- **Security Level**: 128-bit collision resistance
- **Output Size**: 256 bits (32 bytes)
- **Properties**: Pre-image resistance, second pre-image resistance, collision resistance

### Merkle Tree Properties

The Jellyfish Merkle Tree provides:
1. **Tamper Detection**: Any modification changes the root hash
2. **Efficient Verification**: O(log n) proof size and verification time
3. **Sparse Optimization**: Secure handling of empty space
4. **Version Isolation**: Each version has independent cryptographic integrity

### Hash Computation

#### Leaf Node Hash
```
H(leaf) = SHA256(NodeTypeLeaf || key || H(value))
```
- Includes the key to prevent value swapping attacks
- Hashes the value to support large data

#### Internal Node Hash
```
H(internal) = SHA256(NodeTypeInternal || child₀ || child₁ || ... || child₁₅)
```
- All 16 positions included (empty children use `EmptyTreeHash`)
- Deterministic ordering prevents ambiguity

## Security Guarantees

### 1. Data Integrity

**Guarantee**: Any unauthorized modification is cryptographically detectable.

**Mechanism**:
- Every change propagates to root hash
- Root hash serves as cryptographic commitment
- Verification requires matching expected root hash

**Example**:
```go
// Original state
root1 := tree.RootHash()  // 0xabc123...

// After tampering
root2 := tree.RootHash()  // 0xdef456...

// Detection
if root1 != root2 {
    // Tampering detected
}
```

### 2. Proof Soundness

**Guarantee**: Valid proofs cannot be forged without breaking SHA-256.

**Properties**:
- **Inclusion Proofs**: Can only be generated for actual key-value pairs
- **Exclusion Proofs**: Correctly prove non-existence
- **Non-forgeability**: Cannot create valid proof for false claims

### 3. Version Integrity

**Guarantee**: Historical versions remain immutable.

**Mechanism**:
- Append-only storage model
- Each version has unique root hash
- Past states cannot be altered

### 4. Collision Resistance

**Guarantee**: Different trees produce different root hashes (with overwhelming probability).

**Critical Design**:
```go
// Empty value hash (SHA-256 of empty string)
DefaultDigest = SHA256("")

// Non-existent node marker (special constant)
EmptyTreeHash = [32]byte{0x5b, 0xa9, 0x3c, ...}
```

This distinction prevents:
- Ambiguity between empty values and missing keys
- Collision attacks using crafted empty entries

## Threat Model

### In Scope Threats

ProofBox protects against:

1. **Data Tampering**
   - Unauthorized modification of stored values
   - Reordering or deletion of entries
   - Rollback to previous states

2. **Proof Forgery**
   - Creating false inclusion proofs
   - Hiding existence of keys
   - Claiming false tree states

3. **Timing Attacks**
   - Side-channel attacks via comparison timing
   - Mitigation: Constant-time hash comparisons

### Out of Scope Threats

ProofBox does NOT protect against:

1. **Storage Layer Compromise**
   - Physical access to database
   - Storage encryption is user responsibility

2. **Key Disclosure**
   - ProofBox doesn't hide key existence
   - Keys visible in proofs

3. **Value Privacy**
   - Values are not encrypted
   - Hashes may leak information

4. **Denial of Service**
   - Resource exhaustion attacks
   - Requires application-level protection

## Trust Model

### What You Must Trust

1. **Root Hash Source**
   - The root hash must come from a trusted source
   - This is the anchor for all security guarantees

2. **SHA-256 Security**
   - Assumes SHA-256 remains cryptographically secure
   - No practical collision attacks

3. **Implementation Correctness**
   - ProofBox correctly implements the algorithms
   - Verified through extensive testing and fuzzing

### What You Don't Need to Trust

1. **Storage Provider**
   - Can verify data integrity even with untrusted storage
   - Tampering is detectable

2. **Network Transport**
   - Proofs can be verified regardless of transport
   - Man-in-the-middle cannot forge valid proofs

3. **Historical Data**
   - Can verify any version with just its root hash
   - No need to trust intermediate states

## Attack Scenarios and Mitigations

### 1. Value Swapping Attack

**Attack**: Try to swap values between keys.

**Mitigation**: Leaf hash includes the key:
```go
hash = SHA256(leafType || key || valueHash)
```

### 2. Empty Value Ambiguity

**Attack**: Confuse empty values with non-existent keys.

**Mitigation**: Different hash values:
- Empty value: `SHA256("")`
- Non-existent: `EmptyTreeHash` constant

### 3. Proof Replay Attack

**Attack**: Use old proof with new root hash.

**Mitigation**: Proof validity tied to specific root hash.

### 4. Timing Side-Channel

**Attack**: Measure comparison time to guess hashes.

**Mitigation**: Use `bytes.Equal` for constant-time comparison.

## Security Best Practices

### For Application Developers

1. **Secure Root Hash Distribution**
   ```go
   // Good: Get root hash from trusted source
   trustedRoot := getFromConsensus()
   
   // Bad: Trust unverified root
   untrustedRoot := getFromNetwork()
   ```

2. **Verify Proofs Properly**
   ```go
   // Always verify against known root
   err := proof.Verify(trustedRoot)
   if err != nil {
       // Reject the proof
   }
   ```

3. **Handle Keys Carefully**
   - Use cryptographic hashes as keys
   - Avoid sequential or predictable keys
   - Consider key privacy requirements

### For System Operators

1. **Secure Storage**
   - Use encrypted filesystems
   - Implement access controls
   - Regular backups

2. **Monitor Integrity**
   - Track root hash changes
   - Verify random samples
   - Alert on unexpected modifications

3. **Version Management**
   - Implement retention policies
   - Secure pruning procedures
   - Maintain audit trails

### For Security Auditors

1. **Verify Cryptographic Implementation**
   - Check hash function usage
   - Verify proof algorithms
   - Test edge cases

2. **Fuzz Testing**
   - Test with malformed proofs
   - Verify failure modes
   - Check resource limits

3. **Review Trust Boundaries**
   - Identify trust assumptions
   - Verify security claims
   - Document limitations

## Security Testing

ProofBox includes comprehensive security testing:

1. **Fuzzing Infrastructure**
   - Proof tampering detection
   - Malformed input handling
   - Property-based testing

2. **Invariant Testing**
   - Proof consistency across versions
   - Structural integrity validation
   - Cross-implementation verification

3. **Static Analysis**
   - `gosec` for security issues
   - Constant-time verification
   - Resource leak detection

## Limitations and Considerations

### 1. Quantum Computing

Current security relies on SHA-256, which is vulnerable to quantum attacks:
- Grover's algorithm reduces security to 128 bits
- Plan for post-quantum hash migration

### 2. Key Privacy

ProofBox reveals:
- Which keys exist (in proofs)
- Tree structure information
- Access patterns

### 3. Value Privacy

Values are not encrypted:
- Hashes may leak information
- Consider application-level encryption

## Summary

ProofBox provides strong cryptographic guarantees for data integrity through:

- **Proven Algorithms**: Based on well-understood Merkle tree principles
- **Conservative Choices**: SHA-256, standard constructions
- **Comprehensive Testing**: Fuzzing, property testing, static analysis
- **Clear Boundaries**: Well-defined trust model and limitations

When used correctly, ProofBox ensures that any tampering with data is cryptographically detectable, making it suitable for high-security applications including blockchain state management, audit logs, and verifiable databases.