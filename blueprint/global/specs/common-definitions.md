# Common Definitions Specification

## Overview
This specification defines common types, constants, and conventions used throughout the Jellyfish Merkle Tree implementation with PebbleDB in Go.

## Core Types

### Version
```go
// Version represents a state version in the JMT
type Version uint64

const (
    // MaxVersion is the maximum supported version number
    MaxVersion Version = math.MaxUint64 - 1
    
    // InitialVersion is the first version in a new tree
    InitialVersion Version = 0
)
```

### Hash
```go
// Hash represents a 32-byte cryptographic hash
type Hash [32]byte

// HashValue is an alias for Hash for compatibility
type HashValue = Hash

// EmptyHash is the hash of an empty/null node in sparse Merkle trees
// This is a special sentinel value, NOT the hash of empty data
var EmptyHash = Hash{
    0x5b, 0xa9, 0x3c, 0x9d, 0xb0, 0xcf, 0xf9, 0x3f,
    0x52, 0xb5, 0x21, 0xd7, 0x81, 0x4c, 0x7f, 0xa0,
    0x8a, 0xbe, 0x86, 0x13, 0x5f, 0x74, 0x6b, 0x46,
    0x68, 0xb0, 0x13, 0xbc, 0xd1, 0xc4, 0x8e, 0x9b,
}

// Implementation Notes:
// - May be named EmptyTreeHash in crypto package for clarity
// - Different from zero hash (all 0x00 bytes) used for null checks
// - Represents infinite empty subtree in sparse Merkle tree theory
```

### Key
```go
// Key represents a 256-bit key in the tree
type Key [32]byte

// KeyHash converts a variable-length key to fixed Key type
func KeyHash(data []byte) Key {
    return Key(SHA256(data))
}
```

### Nibble
```go
// Nibble represents a 4-bit value (0-15)
type Nibble uint8

const (
    // NibblePerByte is the number of nibbles in a byte
    NibblePerByte = 2
    
    // MaxNibbleValue is the maximum value of a nibble
    MaxNibbleValue Nibble = 15
)

// ValidateNibble ensures nibble is in valid range
func ValidateNibble(n Nibble) error {
    if n > MaxNibbleValue {
        return ErrInvalidNibble
    }
    return nil
}
```

## Cryptographic Functions

### Hash Function
```go
// HashFunction defines the cryptographic hash used throughout JMT
// We use SHA-256 for compatibility and security
func HashFunction(data []byte) Hash {
    return sha256.Sum256(data)
}

// HashConcat hashes the concatenation of multiple byte slices
func HashConcat(parts ...[]byte) Hash {
    h := sha256.New()
    for _, part := range parts {
        h.Write(part)
    }
    var result Hash
    h.Sum(result[:0])
    return result
}
```

### Node Hashing
```go
// NodeType identifies the type of node for hashing
type NodeType byte

const (
    NodeTypeInternal NodeType = 0x00
    NodeTypeLeaf     NodeType = 0x01
)

// HashInternalNode computes hash of an internal node
func HashInternalNode(children map[Nibble]Hash) Hash {
    // Sort nibbles for deterministic ordering
    nibbles := make([]Nibble, 0, len(children))
    for n := range children {
        nibbles = append(nibbles, n)
    }
    sort.Slice(nibbles, func(i, j int) bool {
        return nibbles[i] < nibbles[j]
    })
    
    // Hash: NodeTypeInternal || (nibble || child_hash)*
    parts := [][]byte{{byte(NodeTypeInternal)}}
    for _, n := range nibbles {
        parts = append(parts, []byte{byte(n)})
        parts = append(parts, children[n][:])
    }
    return HashConcat(parts...)
}

// HashLeafNode computes hash of a leaf node
func HashLeafNode(key Key, valueHash Hash) Hash {
    // Hash: NodeTypeLeaf || key || value_hash
    return HashConcat(
        []byte{byte(NodeTypeLeaf)},
        key[:],
        valueHash[:],
    )
}
```

## Size Limits

```go
const (
    // MaxKeySize is the fixed size of keys (256 bits)
    MaxKeySize = 32
    
    // MaxValueSize is the maximum size of a value blob (1 MB)
    MaxValueSize = 1 << 20
    
    // MaxProofSize is the maximum size of a Merkle proof (16 KB)
    MaxProofSize = 16 << 10
    
    // MaxBatchSize is the maximum number of operations in a batch
    MaxBatchSize = 10000
    
    // MaxTreeDepth is the maximum depth of the tree (64 nibbles)
    MaxTreeDepth = 64
)
```

## Serialization Formats

### Endianness
All multi-byte values are encoded in **big-endian** format for consistency and readability.

### Version Encoding
```go
// EncodeVersion encodes a version to 8 bytes big-endian
func EncodeVersion(v Version) [8]byte {
    var buf [8]byte
    binary.BigEndian.PutUint64(buf[:], uint64(v))
    return buf
}

// DecodeVersion decodes a version from 8 bytes big-endian
func DecodeVersion(buf [8]byte) Version {
    return Version(binary.BigEndian.Uint64(buf[:]))
}
```

### Value Serialization
```go
// ValueBlob represents a serialized value with its hash
type ValueBlob struct {
    Data []byte
    Hash Hash
}

// SerializeValue creates a value blob from raw data
func SerializeValue(data []byte) (ValueBlob, error) {
    if len(data) > MaxValueSize {
        return ValueBlob{}, ErrValueTooLarge
    }
    return ValueBlob{
        Data: data,
        Hash: HashFunction(data),
    }, nil
}
```

## Constants

```go
const (
    // DatabaseNamespace separates JMT data from other data in PebbleDB
    DatabaseNamespace = "jmt"
    
    // NodeKeyPrefix is the prefix for node storage keys
    NodeKeyPrefix = "n"
    
    // RootKeyPrefix is the prefix for root hash storage
    RootKeyPrefix = "r"
    
    // MetadataKeyPrefix is the prefix for metadata storage
    MetadataKeyPrefix = "m"
)
```

## Validation Functions

```go
// ValidateKey ensures a key is valid
func ValidateKey(key Key) error {
    // Keys are fixed size, so just check for zero key
    if key == (Key{}) {
        return ErrEmptyKey
    }
    return nil
}

// ValidateVersion ensures a version is valid
func ValidateVersion(v Version) error {
    if v > MaxVersion {
        return ErrInvalidVersion
    }
    return nil
}

// ValidateTreeDepth ensures we don't exceed maximum depth
func ValidateTreeDepth(depth int) error {
    if depth > MaxTreeDepth {
        return ErrMaxDepthExceeded
    }
    return nil
}
```

## Type Conversions

```go
// KeyToNibblePath converts a key to a nibble path
func KeyToNibblePath(key Key) []Nibble {
    nibbles := make([]Nibble, MaxTreeDepth)
    for i := 0; i < len(key); i++ {
        nibbles[i*2] = Nibble(key[i] >> 4)
        nibbles[i*2+1] = Nibble(key[i] & 0x0F)
    }
    return nibbles
}

// NibblePathToPartialKey converts nibbles back to partial key bytes
func NibblePathToPartialKey(nibbles []Nibble) []byte {
    if len(nibbles)%2 != 0 {
        panic("odd number of nibbles")
    }
    
    bytes := make([]byte, len(nibbles)/2)
    for i := 0; i < len(bytes); i++ {
        bytes[i] = byte(nibbles[i*2])<<4 | byte(nibbles[i*2+1])
    }
    return bytes
}
```