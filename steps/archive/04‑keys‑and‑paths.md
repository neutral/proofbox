---
id: step.04.keys-and-paths
depends_on:
  - step.03.error-handling
tags: [types, storage, step]
---

## Objective

Implement Key types, nibble operations, and NodeKey structure for path-based node addressing in storage.

## Implements

- **Glossary – Key, Nibble, NodeKey**
- **NodeKey Encoding Specification** (`../../pkg/tree/_blueprint/metadata-versions/_specs/nodekey-encoding.md`)
- **Versioned Keys Specification** (`../../pkg/tree/_blueprint/metadata-versions/_specs/versioned-keys.md`)
- **Common Definitions** (`../../pkg/_blueprint/_specs/common-definitions.md`)

## Technical Details

### Core Type Definitions

Create `pkg/types/key.go`:
```go
package types

import (
    "crypto/sha256"
    "encoding/hex"
    "fmt"
)

// Key represents a 256-bit key in the tree
type Key [32]byte

// Nibble represents a 4-bit value (0-15)
type Nibble uint8

// NibblePath represents a path through the tree
type NibblePath struct {
    Nibbles []Nibble
    Length  uint16
}

const (
    KeySize          = 32  // 256 bits
    NibblePerByte    = 2
    MaxNibbleValue   = 15
    MaxTreeDepth     = 64  // 32 bytes * 2 nibbles per byte
)

// KeyFromBytes creates a Key from byte slice
func KeyFromBytes(b []byte) (Key, error) {
    if len(b) != KeySize {
        return Key{}, WrapError(ErrInvalidKey, "key must be 32 bytes, got %d", len(b))
    }
    var key Key
    copy(key[:], b)
    return key, nil
}

// KeyHash converts variable-length data to a fixed Key using SHA-256
func KeyHash(data []byte) Key {
    return Key(sha256.Sum256(data))
}

// String returns hex representation of the key
func (k Key) String() string {
    return hex.EncodeToString(k[:])
}

// Bytes returns the key as a byte slice
func (k Key) Bytes() []byte {
    return k[:]
}

// IsEmpty checks if the key is all zeros
func (k Key) IsEmpty() bool {
    return k == Key{}
}

// ValidateKey ensures a key is valid
func ValidateKey(k Key) error {
    if k.IsEmpty() {
        return ErrEmptyKey
    }
    return nil
}
```

### Nibble Operations

Add to `pkg/types/key.go`:
```go
// ExtractNibble extracts the nibble at the given depth
func (k Key) ExtractNibble(depth int) (Nibble, error) {
    if depth < 0 || depth >= MaxTreeDepth {
        return 0, WrapError(ErrInvalidNibble, "depth %d out of range [0, %d)", depth, MaxTreeDepth)
    }
    
    byteIndex := depth / NibblePerByte
    nibbleIndex := depth % NibblePerByte
    
    b := k[byteIndex]
    if nibbleIndex == 0 {
        return Nibble(b >> 4), nil  // High nibble
    }
    return Nibble(b & 0x0F), nil    // Low nibble
}

// ToNibblePath converts a key to a full nibble path
func (k Key) ToNibblePath() NibblePath {
    nibbles := make([]Nibble, MaxTreeDepth)
    for i := 0; i < MaxTreeDepth; i++ {
        nibbles[i], _ = k.ExtractNibble(i)
    }
    return NibblePath{
        Nibbles: nibbles,
        Length:  MaxTreeDepth,
    }
}

// ValidateNibble ensures nibble is in valid range
func ValidateNibble(n Nibble) error {
    if n > MaxNibbleValue {
        return WrapError(ErrInvalidNibble, "nibble %d exceeds max value %d", n, MaxNibbleValue)
    }
    return nil
}
```

### NodeKey Implementation

Create `pkg/types/nodekey.go`:
```go
package types

import (
    "bytes"
    "encoding/binary"
    "fmt"
)

// NodeKey uniquely identifies a node in the tree across all versions
type NodeKey struct {
    Version    Version
    NibblePath NibblePath
}

// Version represents a state version in the JMT
type Version uint64

const (
    // MaxVersion is the maximum supported version number
    MaxVersion Version = ^Version(0) - 1
    
    // InitialVersion is the first version in a new tree
    InitialVersion Version = 0
)

// RootNodeKey returns the key for the root node at a version
func RootNodeKey(version Version) NodeKey {
    return NodeKey{
        Version: version,
        NibblePath: NibblePath{
            Nibbles: []Nibble{},
            Length:  0,
        },
    }
}

// Child returns the NodeKey for a child at the given nibble
func (nk NodeKey) Child(nibble Nibble, childVersion Version) NodeKey {
    newNibbles := make([]Nibble, nk.NibblePath.Length+1)
    copy(newNibbles, nk.NibblePath.Nibbles)
    newNibbles[nk.NibblePath.Length] = nibble
    
    return NodeKey{
        Version: childVersion,
        NibblePath: NibblePath{
            Nibbles: newNibbles,
            Length:  nk.NibblePath.Length + 1,
        },
    }
}

// IsRoot checks if this is the root node
func (nk NodeKey) IsRoot() bool {
    return nk.NibblePath.Length == 0
}

// String returns a human-readable representation
func (nk NodeKey) String() string {
    return fmt.Sprintf("v%d:%x", nk.Version, nk.NibblePath.Nibbles[:nk.NibblePath.Length])
}
```

### NodeKey Encoding

Add to `pkg/types/nodekey.go`:
```go
// EncodeNodeKey encodes a NodeKey to bytes for storage
// Format: Version(8) || Length(2) || Nibbles(packed)
func EncodeNodeKey(key NodeKey) []byte {
    // Calculate buffer size
    nibbleBytes := (key.NibblePath.Length + 1) / 2
    bufSize := 8 + 2 + int(nibbleBytes)
    buf := make([]byte, bufSize)
    
    // Encode version (8 bytes, big-endian)
    binary.BigEndian.PutUint64(buf[0:8], uint64(key.Version))
    
    // Encode nibble count (2 bytes, big-endian)
    binary.BigEndian.PutUint16(buf[8:10], key.NibblePath.Length)
    
    // Encode nibbles (packed, 2 per byte)
    offset := 10
    for i := uint16(0); i < key.NibblePath.Length; i += 2 {
        high := key.NibblePath.Nibbles[i]
        low := Nibble(0)
        if i+1 < key.NibblePath.Length {
            low = key.NibblePath.Nibbles[i+1]
        }
        buf[offset] = byte(high)<<4 | byte(low)
        offset++
    }
    
    return buf[:offset]
}

// DecodeNodeKey decodes a NodeKey from storage bytes
func DecodeNodeKey(buf []byte) (NodeKey, error) {
    if len(buf) < 10 {
        return NodeKey{}, ErrInvalidNodeKey
    }
    
    // Decode version
    version := Version(binary.BigEndian.Uint64(buf[0:8]))
    
    // Decode nibble count
    nibbleCount := binary.BigEndian.Uint16(buf[8:10])
    if nibbleCount > MaxTreeDepth {
        return NodeKey{}, WrapError(ErrInvalidNibbleCount, "count %d exceeds max %d", nibbleCount, MaxTreeDepth)
    }
    
    // Calculate expected size
    expectedSize := 10 + (nibbleCount+1)/2
    if len(buf) < int(expectedSize) {
        return NodeKey{}, WrapError(ErrInvalidNodeKey, "buffer too small: got %d, need %d", len(buf), expectedSize)
    }
    
    // Decode nibbles
    nibbles := make([]Nibble, nibbleCount)
    offset := 10
    for i := uint16(0); i < nibbleCount; i += 2 {
        b := buf[offset]
        nibbles[i] = Nibble(b >> 4)
        if i+1 < nibbleCount {
            nibbles[i+1] = Nibble(b & 0x0F)
        }
        offset++
    }
    
    return NodeKey{
        Version: version,
        NibblePath: NibblePath{
            Nibbles: nibbles,
            Length:  nibbleCount,
        },
    }, nil
}

// StorageKey creates the full storage key for PebbleDB
func (nk NodeKey) StorageKey() []byte {
    const nodeKeyPrefix = "n"
    encoded := EncodeNodeKey(nk)
    storageKey := make([]byte, len(nodeKeyPrefix)+len(encoded))
    copy(storageKey, []byte(nodeKeyPrefix))
    copy(storageKey[len(nodeKeyPrefix):], encoded)
    return storageKey
}

// Compare returns -1, 0, or 1 for less than, equal, or greater than
func (nk NodeKey) Compare(other NodeKey) int {
    // Compare versions first
    if nk.Version < other.Version {
        return -1
    }
    if nk.Version > other.Version {
        return 1
    }
    
    // Same version, compare nibble paths
    return nk.NibblePath.Compare(other.NibblePath)
}

// Compare compares two nibble paths lexicographically
func (np NibblePath) Compare(other NibblePath) int {
    minLen := np.Length
    if other.Length < minLen {
        minLen = other.Length
    }
    
    // Compare common prefix
    for i := uint16(0); i < minLen; i++ {
        if np.Nibbles[i] < other.Nibbles[i] {
            return -1
        }
        if np.Nibbles[i] > other.Nibbles[i] {
            return 1
        }
    }
    
    // Shorter path comes first
    if np.Length < other.Length {
        return -1
    }
    if np.Length > other.Length {
        return 1
    }
    
    return 0
}
```

## Testing Requirements

### Key Operations Tests
```go
func TestKeyExtractNibble(t *testing.T) {
    key := Key{0x12, 0x34, 0x56, 0x78} // First 4 bytes
    
    tests := []struct {
        depth    int
        expected Nibble
    }{
        {0, 0x1},  // High nibble of first byte
        {1, 0x2},  // Low nibble of first byte
        {2, 0x3},  // High nibble of second byte
        {3, 0x4},  // Low nibble of second byte
    }
    
    for _, tc := range tests {
        nibble, err := key.ExtractNibble(tc.depth)
        assert.NoError(t, err)
        assert.Equal(t, tc.expected, nibble)
    }
}

func TestNodeKeyEncoding(t *testing.T) {
    key := NodeKey{
        Version: 1000,
        NibblePath: NibblePath{
            Nibbles: []Nibble{0x1, 0x2, 0xA, 0xB, 0xC},
            Length:  5,
        },
    }
    
    // Encode
    encoded := EncodeNodeKey(key)
    
    // Decode
    decoded, err := DecodeNodeKey(encoded)
    assert.NoError(t, err)
    assert.Equal(t, key.Version, decoded.Version)
    assert.Equal(t, key.NibblePath.Length, decoded.NibblePath.Length)
    assert.Equal(t, key.NibblePath.Nibbles[:5], decoded.NibblePath.Nibbles[:5])
}

func TestNodeKeyOrdering(t *testing.T) {
    keys := []NodeKey{
        {Version: 1, NibblePath: NibblePath{[]Nibble{0x1}, 1}},
        {Version: 1, NibblePath: NibblePath{[]Nibble{0x1, 0x2}, 2}},
        {Version: 2, NibblePath: NibblePath{[]Nibble{0x1}, 1}},
    }
    
    // Keys should be ordered by version first, then path
    assert.Equal(t, -1, keys[0].Compare(keys[1])) // Same version, shorter path first
    assert.Equal(t, -1, keys[0].Compare(keys[2])) // Lower version first
    assert.Equal(t, -1, keys[1].Compare(keys[2])) // Lower version first
}
```

### Fuzz Testing
```go
func FuzzNodeKeyEncoding(f *testing.F) {
    f.Fuzz(func(t *testing.T, version uint64, nibbleData []byte) {
        if len(nibbleData) > MaxTreeDepth {
            nibbleData = nibbleData[:MaxTreeDepth]
        }
        
        nibbles := make([]Nibble, len(nibbleData))
        for i, b := range nibbleData {
            nibbles[i] = Nibble(b & 0x0F)
        }
        
        original := NodeKey{
            Version: Version(version),
            NibblePath: NibblePath{
                Nibbles: nibbles,
                Length:  uint16(len(nibbles)),
            },
        }
        
        encoded := EncodeNodeKey(original)
        decoded, err := DecodeNodeKey(encoded)
        
        assert.NoError(t, err)
        assert.Equal(t, original.Version, decoded.Version)
        assert.Equal(t, original.NibblePath.Length, decoded.NibblePath.Length)
    })
}
```

## Implementation Steps

1. Create `pkg/types/key.go` with Key and Nibble types
2. Implement nibble extraction and path conversion
3. Create `pkg/types/nodekey.go` with NodeKey structure
4. Implement NodeKey encoding/decoding
5. Add storage key generation
6. Implement comparison functions for ordering
7. Write comprehensive unit tests
8. Add fuzz tests for encoding/decoding
9. Add benchmarks for critical operations

## Performance Considerations

- Pre-allocate nibble arrays to avoid allocations
- Use bit operations for nibble extraction
- Cache encoded storage keys for frequently accessed nodes
- Consider packed nibble representation in memory

## Security Notes

- Validate all inputs to prevent panics
- Ensure nibble values are always 0-15
- Check buffer sizes in decode operations
- Use constant-time operations where applicable

## Done When ✓

- [x] Key type with validation and utilities
- [x] Nibble extraction with bounds checking
- [x] NibblePath with comparison operations
- [x] NodeKey structure with version support
- [x] Binary encoding/decoding with proper error handling
- [x] Storage key generation for PebbleDB
- [x] Comprehensive unit tests with edge cases
- [x] Fuzz tests for encoding robustness
- [x] Benchmarks showing acceptable performance
- [x] All types properly documented
