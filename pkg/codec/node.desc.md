# Generic Node Codec

## Purpose

Provides type-safe encoding and decoding for any node type by adding a type prefix to distinguish between leaf and internal nodes during deserialization.

## Design

Adds a single-byte type prefix:
- 0x01: Leaf node
- 0x00: Internal node

This minimal overhead (1 byte) enables:
1. Safe deserialization without prior type knowledge
2. Runtime type checking
3. Forward compatibility for future node types

## Usage

This codec is used when:
- Storing nodes in a generic container
- Type information is not available from context
- Implementing type-agnostic storage layers

## Trade-offs

The type byte adds minimal overhead but provides crucial type safety. For bulk operations where type is known, use the specific leaf/internal codecs directly to avoid the extra byte.