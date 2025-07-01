# Generic Node Codec

## Purpose

Provides type-safe encoding and decoding for any node type by adding a type prefix to distinguish between leaf and internal nodes during deserialization.

## Design

Adds a single-byte type prefix:
- 0x00: Internal node (types.NodeTypeInternal)
- 0x01: Leaf node (types.NodeTypeLeaf)

This minimal overhead (1 byte) enables:
1. Safe deserialization without prior type knowledge
2. Runtime type checking
3. Forward compatibility for future node types

## Architecture Integration

After the import cycle resolution, this codec now:
- Works with `types.Node` interface instead of concrete types
- Uses the factory pattern for node creation during decoding
- Delegates to interface-based encoders for serialization

**Important**: The `NodeCodec.DecodeNode` method is non-functional and returns errors directing users to use the tree package factory. Use the standalone `DecodeNode` function from factory.go instead, which properly handles node creation through the factory pattern.

## Usage

This codec is used when:
- Storing nodes in a generic container
- Type information is not available from context
- Implementing type-agnostic storage layers

For decoding, use: `codec.DecodeNode(data, version, factory)` (from factory.go)

## Trade-offs

The type byte adds minimal overhead but provides crucial type safety. For bulk operations where type is known, use the specific leaf/internal codecs directly to avoid the extra byte.