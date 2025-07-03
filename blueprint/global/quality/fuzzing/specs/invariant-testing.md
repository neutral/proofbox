---
id: invariant-testing-specs
title: Invariant Testing Specifications
status: active
created: 2024-01-03
updated: 2024-01-03
tags: [specs, testing, invariants, fuzzing]
---

# Invariant Testing Specifications

## Overview

This document specifies the detailed implementation of invariant testing in ProofBox's fuzzing infrastructure. Each invariant represents a critical property that must hold true regardless of the sequence of operations performed on the tree.

## Core Invariants

### 1. Version Isolation Invariant

**Definition**: A key-value pair is only visible in versions equal to or greater than the version where it was first inserted.

**Implementation**:
```go
type VersionTracker struct {
    keyVersionMap map[string]types.Version  // hex(key) -> first version
}
```

**Testing Strategy**:
- Track when each key is first added to the tree
- For GET operations at any version V:
  - If V < firstVersion(key): value MUST be nil
  - If V >= firstVersion(key): value MUST match expected
- Handle deletions: deleted keys remain visible in historical versions

**Edge Cases**:
- Keys that are added, deleted, and re-added
- Operations on non-existent versions
- Version 0 (empty tree) queries

### 2. Proof Consistency Invariant

**Definition**: Cryptographic proofs must be internally consistent and verifiable across all versions.

**Implementation**:
```go
type ProofConsistencyTracker struct {
    proofs     map[types.Version]map[string]*proof.Proof
    rootHashes map[types.Version]types.Hash
}
```

**Testing Strategy**:
- Generate proof immediately after each modification
- Verify proof matches expected type:
  - Inclusion proof: contains correct value
  - Non-existence proof: value is nil
- Cross-version validation:
  - Unchanged keys have consistent proofs
  - Root hash changes reflect modifications

**Verification Requirements**:
- Proof with correct root hash MUST verify
- Proof with tampered root hash MUST fail
- Leaf hash computation MUST match expected format

## Testing Methodology

### Property-Based Testing with Rapid

**Configuration**:
```go
rapid.Check(t, func(t *rapid.T) {
    ops := generators.OperationSliceGen(1, 30).Draw(t, "operations")
    tree := fuzz.NewTestTree()
    err := invariant.Check(tree, ops)
    require.NoError(t, err)
})
```

**Key Features**:
- Automatic test case generation
- Intelligent shrinking on failure
- Deterministic reproduction with seeds
- Configurable iteration counts

### Operation Generation

**Structured Key Generation** (50% probability):
- Common prefixes to test internal node sharing
- Variable suffix lengths for tree depth testing
- Ensures realistic key distribution

**Random Key Generation** (50% probability):
- Full 32-byte random keys
- Tests sparse tree regions
- Stresses hash distribution

**Value Generation**:
- PUT: 1-1024 bytes (never empty)
- GET/DELETE: may have empty values
- Realistic payload sizes

### Version Management

**Version Tracking**:
- Each modification creates new version
- Failed operations don't create versions
- Version numbers are sequential
- Version 0 represents empty tree

**Historical Queries**:
- GET operations specify target version
- Version bounds checking prevents invalid queries
- Non-existent version handling is graceful

## Concurrent Testing Specifications

### Thread Safety Requirements

**Concurrent Operations**:
- Multiple readers at same version
- Single writer with multiple readers
- Version creation atomicity

**Synchronization**:
- No data races detected by race detector
- Consistent state under concurrent access
- Proper cleanup of resources

### Performance Invariants

**Latency Requirements**:
- P50 < 1ms for in-memory operations
- P99 < 10ms under normal load
- Logarithmic scaling with tree size

**Memory Requirements**:
- Linear growth with unique data
- Structural sharing effectiveness > 80%
- No memory leaks under extended operation

## Failure Handling

### Test Failure Analysis

**Minimal Reproduction**:
- Rapid automatically shrinks to minimal case
- Typically <10 operations for reproduction
- Saved to testdata/rapid/ directory

**Debugging Support**:
- Operation sequence logging
- State snapshots at each version
- Detailed error messages with context

### Known Failure Patterns

1. **Version Not Found**: Operations on non-sequential versions
2. **Key Visibility**: Version isolation violations
3. **Proof Mismatches**: Structural changes affecting proofs
4. **Memory Leaks**: Improper version cleanup

## Integration Points

### CI/CD Pipeline

**PR Validation**:
- Quick fuzz: 20 iterations
- All invariants tested
- ~30 second runtime

**Nightly Testing**:
- Comprehensive: 1000 iterations
- Performance baselines updated
- Coverage reports generated

### Corpus Management

**Corpus Builder**:
- Generates edge cases
- Security-focused inputs
- Performance stress tests
- Regression test cases

**Corpus Organization**:
- JSON format for portability
- Categorized by test type
- Gitignored to avoid bloat

## Future Enhancements

1. **Differential Testing**: Compare against reference implementation
2. **Coverage Metrics**: Track fuzzing effectiveness
3. **OSS-Fuzz Integration**: Continuous fuzzing infrastructure
4. **Security Fuzzing**: Adversarial input generation
5. **Mutation Testing**: Verify test effectiveness