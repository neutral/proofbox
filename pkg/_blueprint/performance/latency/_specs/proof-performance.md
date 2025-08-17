# Proof Performance Specification

## Overview
JMT's design optimizes for fast proof generation and verification, critical for distributed system clients and validators.

## Proof Size Optimization
The number of sibling digests in a JMT proof is less on average than equivalent trees without optimization (see [SN-007]):
- JMT: Θ(log(number of existent leaves))
- Full tree: log(2^h) where h = key bits

### Concrete Benefits
For 256-bit keys:
- Worst case: 64 siblings (if every nibble has siblings)
- Average case: ~8-10 siblings (for billion-key tree)
- Binary tree equivalent: 30+ siblings

## Verification Performance

### Target Metrics
- Proof verification: p95 ≤ 300 μs on 2024-era laptop CPU
- Critical for light client UX
- Enables fast validator operations

### Performance Factors
1. **Fewer Hash Operations**
   - 16-ary tree = fewer levels
   - Each level: one 16-way hash vs multiple 2-way hashes
   
2. **Cache Efficiency**
   - Compact proof size fits in CPU cache
   - Sequential hash operations
   - Predictable memory access pattern

3. **SIMD Optimization Potential**
   - 16 children align with SIMD registers
   - Parallel hash computation possible
   - Further performance gains available

## Network Benefits
Smaller proofs mean:
- Less bandwidth for light clients
- Faster proof transmission
- More proofs per network packet
- Better mobile/constrained device support

## Benchmark Considerations
When measuring performance:
- Use representative key distributions
- Test with various tree sizes
- Include proof serialization/deserialization
- Measure on target hardware platforms