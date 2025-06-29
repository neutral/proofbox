# Codec Tests

## Purpose

Comprehensive test suite validating the correctness, performance, and edge cases of all codec implementations.

## Test Coverage

### Correctness Tests
- Round-trip encoding/decoding for all node types
- Boundary conditions (empty nodes, maximum children)
- Invalid input handling (corrupted data, wrong sizes)
- Deterministic output verification

### Performance Benchmarks
- Leaf encoding: ~6.5ns baseline
- Internal encoding: Scales linearly with children
- Batch operations: Throughput measurements
- Memory allocation tracking

### Property-Based Tests
- Encoding determinism (same input → same output)
- Size calculations match actual output
- Child ordering preservation
- Version handling correctness

## Why Comprehensive Testing

The codec is critical infrastructure where bugs could cause:
1. Data corruption
2. Hash mismatches
3. Network protocol incompatibilities
4. Performance degradation

Tests ensure compatibility with reference implementations and catch regressions early.