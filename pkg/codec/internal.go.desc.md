# Internal Node Codec

## Purpose

Implements binary encoding and decoding for internal nodes with variable-size format that maintains deterministic output through sorted child ordering.

## Design

The encoding format consists of:
- NumChildren: 1 byte (0-16)
- Children: Array of child entries, each containing:
  - Nibble: 1 byte
  - Hash: 32 bytes
  - Version: 8 bytes (big-endian uint64)
  - IsLeaf: 1 byte (boolean flag)
  - Total per child: 42 bytes

Total size: 1 + (numChildren × 42) bytes

## Determinism

Children are always encoded in ascending nibble order (0-15) to ensure:
1. Consistent hash computation
2. Reproducible encoding across implementations
3. Canonical representation for verification

## Why Include Child Metadata

Unlike leaf nodes, internal nodes store additional child metadata:
- **Version**: Required for multi-version tree navigation
- **IsLeaf**: Enables type determination without loading child
- **Hash**: Allows tree traversal without child deserialization

This trades space for reduced I/O during tree operations.