# Leaf Node Codec

## Purpose

Implements binary encoding and decoding for leaf nodes with a fixed 64-byte format optimized for speed and zero allocations.

## Design

The leaf encoding uses a simple concatenation of fixed-size fields:
- Key: 32 bytes
- ValueHash: 32 bytes
- Total: 64 bytes (always)

## Performance

- Encoding: ~6.5ns with zero allocations
- Decoding: ~10ns with single allocation for node creation
- Fixed size enables memory mapping and cache-friendly access

## Why This Approach

Fixed-size encoding was chosen because:
1. Leaf nodes always have exactly these two fields
2. Both fields are always 32 bytes (types.Key and types.Hash)
3. No metadata or variable-length data needed
4. Enables extremely fast encode/decode operations
5. Predictable memory layout for cache optimization

The version is not included as it's provided externally during tree operations (see version-not-serialized decision).