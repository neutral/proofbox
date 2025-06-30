# optimization.go

Implements proof size optimization techniques including compression and batch proof deduplication.

## Key Components

- **CompressedProof**: Space-optimized proof representation using gzip compression
- **CompressProof/DecompressProof**: Handles proof compression with ~40-60% size reduction
- **BatchProofOptimization**: Placeholder for future path-sharing optimizations

## Compression Strategy

1. **Serialize proof data** into compact binary format
2. **Apply gzip compression** with default compression level
3. **Store key hash** instead of full key for space savings
4. **Maintain proof type** for quick identification without decompression

## Future Optimizations

- **Path sharing**: Identify common prefixes in batch proofs
- **Sibling deduplication**: Share sibling data across related proofs
- **Delta encoding**: Encode proofs as deltas from previous proofs
- **Custom compression**: Domain-specific compression for proof data

## Performance Considerations

- Compression adds ~10-20μs overhead but reduces network transmission time
- Decompression is faster than compression (~5-10μs)
- Break-even point: ~1KB proof size for local verification, ~100B for network