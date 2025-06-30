# Storage Keys

## Purpose

Provides consistent key generation and parsing for all storage operations in the JMT.

## Key Formats

### Root Keys
- Format: `r<version(8 bytes)>`
- Used to store version → root hash mappings
- Example: `r\x00\x00\x00\x00\x00\x00\x00\x01` for version 1

### Node Keys
- Format: `n<version(8)><nibble_count(2)><packed_nibbles>`
- Nibbles packed 2 per byte
- Example: `n<version><count><0x12,0x34>` for path [1,2,3,4]

### Value Keys
Two formats supported:

1. **By Hash**: `v<version(8)><hash(32)>`
   - Used when values are content-addressed
   - Enables deduplication

2. **By Key**: `k<key(32)>`
   - Direct key → value mapping
   - Faster for known keys

## Design Rationale

- Fixed prefixes enable efficient range scans
- Big-endian encoding for proper ordering
- Packed nibbles save 50% space
- Version in key enables multi-version storage

## Usage

These functions are internal to the tree package and handle the low-level storage key encoding/decoding. They ensure consistent key formatting across all storage operations.