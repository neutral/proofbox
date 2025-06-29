# Storage Architecture Specification

## Overview

The Jellyfish Merkle Tree implements a two-tier storage architecture that separates tree structure from value data. This design optimizes performance while maintaining flexibility for value serialization.

## Two-Tier Architecture

### Tier 1: Tree Structure Storage

**Purpose**: Store the Merkle tree nodes for fast traversal and proof generation

**Characteristics**:
- Fixed binary format (65 bytes for leaf, variable for internal)
- Contains only metadata and hashes
- Optimized for sequential access
- Immutable once written

**Key Space**:
```
node:<version>:<node_hash> -> [NodeType][NodeData]
```

**Contents**:
- Internal nodes: Children metadata (nibble, hash, version, isLeaf)
- Leaf nodes: Key (32 bytes) + ValueHash (32 bytes)

### Tier 2: Value Storage

**Purpose**: Store actual value data with flexible serialization

**Characteristics**:
- Variable size data
- Multiple serialization formats supported
- Lazy loading on demand
- Can be stored in separate column family

**Key Space**:
```
value:<key> -> [FormatByte][SerializedData]
// OR
value:<value_hash> -> [FormatByte][SerializedData]
```

## Data Flow

### Write Path
1. Application provides key and value
2. Value serialized using chosen format
3. Value hash computed
4. Value stored in value storage
5. Tree updated with key and value hash
6. Tree nodes serialized using binary codec

### Read Path
1. Tree traversed to find leaf node (fast)
2. Leaf returns value hash
3. If value needed, load from value storage
4. Deserialize based on format byte
5. Verify hash matches

### Proof Path
1. Tree traversed collecting node hashes
2. Proof constructed from hashes only
3. No value loading required

## Format Flexibility

### Tree Format (Fixed)
Always uses optimized binary codec:
- Zero allocations
- ~6.5ns encoding time
- Deterministic output

### Value Formats (Flexible)
Format indicated by prefix byte:

| Format | Byte | Use Case |
|--------|------|----------|
| Raw | 0x00 | Binary blobs, hashes |
| ProtoBuf | 0x01 | Structured, evolving data |
| MessagePack | 0x02 | JSON-like, schema-free |
| CBOR | 0x03 | Binary JSON alternative |
| Custom | 0x80+ | Application-specific |

## Performance Characteristics

### Tree Operations
- Insert: O(log n) tree nodes only
- Lookup: O(log n) tree traversal
- Proof: O(log n) hash collection
- No value I/O needed

### Value Operations
- Store: One write + hash computation
- Load: One read + deserialization
- Lazy: Loaded only when accessed
- Cacheable: LRU cache friendly

## Storage Optimization

### For Small Values (<256 bytes)
- May inline in tree storage
- Reduces I/O operations
- Still use value hash for consistency

### For Large Values (>1MB)
- Must use separate storage
- Consider chunking
- Stream processing support

### Compression
- Values can be compressed
- Format byte indicates compression
- Tree structure never compressed

## Implementation Guidelines

### Storage Interface
```go
type Storage interface {
    // Tree operations
    GetNode(key []byte) ([]byte, error)
    PutNode(key, value []byte) error
    
    // Value operations  
    GetValue(key []byte) ([]byte, error)
    PutValue(key, value []byte) error
    
    // Batch operations
    NewBatch() Batch
}
```

### Value Store Wrapper
```go
type ValueStore struct {
    storage Storage
    formats map[string]ValueFormat
}

func (vs *ValueStore) Store(key Key, value interface{}, format ValueFormat) (Hash, error)
func (vs *ValueStore) Load(key Key) (interface{}, ValueFormat, error)
```

## Benefits

1. **Performance**: Tree operations remain fast
2. **Flexibility**: Values can evolve independently
3. **Memory**: Large values not loaded unnecessarily
4. **Compatibility**: Different serialization per use case
5. **Migration**: Format changes without tree rebuild

## Trade-offs

1. **Complexity**: Two storage paths
2. **I/O**: Extra read for values
3. **Space**: Slight key duplication
4. **Consistency**: Hash verification overhead

## Future Considerations

- Bloom filters for value existence
- Tiered storage (SSD/HDD/S3)
- Value compression strategies
- Streaming value support
- Encrypted value storage