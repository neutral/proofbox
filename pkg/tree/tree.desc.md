# Tree Package

## Purpose

Implements the core Jellyfish Merkle Tree data structure with versioning, concurrency support, and snapshot-based reads.

## Architecture

The tree package follows a layered architecture:

1. **Tree Layer**: Main tree structure with version management
2. **Reader Layer**: Snapshot-isolated reads for consistency
3. **Storage Layer**: Key-value storage abstraction using PebbleDB
4. **Cache Layer**: Thread-safe LRU cache for performance

## Concurrency Model

- **Multiple Readers**: Concurrent reads using snapshot isolation
- **Single Writer**: Write operations serialized with `writeMu`
- **Lock Ordering**: writeMu → mu → cache.mu (prevents deadlocks)

## Key Components

### Tree Structure
- Manages versions and root hashes
- Coordinates reads and writes
- Integrates with storage backend

### TreeReader
- Provides consistent reads via snapshots
- Traverses tree to find values
- Handles node loading and caching

### NodeCache
- Thread-safe LRU cache
- Reduces storage I/O
- Tracks hit/miss statistics

### Error Handlers
- DefaultReadErrorHandler: Handles corrupted/missing nodes
- DefaultWriteErrorHandler: Implements retry with exponential backoff

### Health Checker
- Verifies tree integrity
- Recursive node validation
- Context-aware cancellation

## Storage Keys

- Root hashes: `r<version>`
- Nodes: `n<version><nibble_count><nibbles>`
- Values: `v<version><hash>` or `k<key>`

## Import Cycle Resolution

The tree package now uses a factory pattern to register with the codec package:
- Implements interfaces defined in the types package
- Registers node creation factory at initialization
- Enables full codec integration without circular imports

## Current Status

- Read operations fully implemented with empty tree handling
- Write operations not yet implemented (step 09+)
- Full codec integration completed via factory pattern