# Batch Encoder

## Purpose

Provides efficient batch encoding and decoding of multiple nodes in a single operation, optimizing memory allocation and reducing overhead for bulk operations.

## Design

The batch format consists of a sequence of key-node pairs, each with length prefixes:
- For each entry:
  - Key length: 4 bytes (uint32, big-endian)
  - Key data: Variable-length encoded node key
  - Node length: 4 bytes (uint32, big-endian)
  - Node data: Variable-length encoded node data

Note: Unlike some batch formats, there is no header count field. The decoder reads entries until EOF or error.

## Benefits

1. **Single Allocation**: Pre-allocates buffer for entire batch
2. **Streaming Support**: Can process nodes without loading entire batch
3. **Error Recovery**: Length prefixes enable skipping corrupted entries
4. **Flexibility**: Each node can use different encoding (leaf vs internal)
5. **Key Association**: Each node is stored with its key for direct mapping

## Use Cases

- Database bulk operations
- Network protocol for node synchronization  
- Checkpoint/snapshot creation
- Import/export functionality

## Performance

Batch encoding amortizes overhead across multiple nodes, providing ~3-5x throughput improvement over individual node encoding for typical batch sizes (100-1000 nodes).