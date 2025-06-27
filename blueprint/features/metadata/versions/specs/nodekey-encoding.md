# NodeKey Encoding Specification

## Overview
This specification defines the exact binary encoding format for NodeKeys used in PebbleDB storage. NodeKeys uniquely identify nodes across all versions of the tree.

## NodeKey Structure

```go
type NodeKey struct {
    Version    Version     // 8 bytes
    NibblePath NibblePath  // Variable length
}

type NibblePath struct {
    Nibbles []Nibble    // The actual nibbles
    Length  uint16      // Number of nibbles (0-64)
}
```

## Binary Encoding Format

### Layout
```
+----------------+----------------+----------------------+
| Version (8B)   | Length (2B)    | Nibbles (0-32B)      |
+----------------+----------------+----------------------+
| Big-endian u64 | Big-endian u16 | Packed, 2 per byte   |
+----------------+----------------+----------------------+
```

### Encoding Rules

1. **Version**: 8 bytes, big-endian encoded uint64
2. **Length**: 2 bytes, big-endian encoded uint16 (number of nibbles)
3. **Nibbles**: Variable length, packed 2 nibbles per byte
   - Even number of nibbles: `len(nibbles)/2` bytes
   - Odd number of nibbles: `(len(nibbles)+1)/2` bytes, last nibble in lower 4 bits

### Encoding Implementation

```go
// EncodeNodeKey encodes a NodeKey to bytes for storage
func EncodeNodeKey(key NodeKey) []byte {
    // Calculate buffer size
    nibbleBytes := (key.NibblePath.Length + 1) / 2
    bufSize := 8 + 2 + int(nibbleBytes)
    buf := make([]byte, bufSize)
    
    // Encode version (8 bytes)
    binary.BigEndian.PutUint64(buf[0:8], uint64(key.Version))
    
    // Encode nibble count (2 bytes)
    binary.BigEndian.PutUint16(buf[8:10], key.NibblePath.Length)
    
    // Encode nibbles (packed)
    offset := 10
    for i := 0; i < int(key.NibblePath.Length); i += 2 {
        high := key.NibblePath.Nibbles[i]
        low := Nibble(0)
        if i+1 < int(key.NibblePath.Length) {
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
        return NodeKey{}, ErrInvalidNibbleCount
    }
    
    // Calculate expected size
    expectedSize := 10 + (nibbleCount+1)/2
    if len(buf) < int(expectedSize) {
        return NodeKey{}, ErrInvalidNodeKey
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
```

## Storage Key Format

For PebbleDB storage, NodeKeys are prefixed with a namespace:

```go
// StorageKey creates the full storage key for PebbleDB
func (nk NodeKey) StorageKey() []byte {
    encoded := EncodeNodeKey(nk)
    storageKey := make([]byte, len(NodeKeyPrefix)+len(encoded))
    copy(storageKey, []byte(NodeKeyPrefix))
    copy(storageKey[len(NodeKeyPrefix):], encoded)
    return storageKey
}
```

## Ordering and Comparison

NodeKeys are ordered lexicographically by their encoded form, which provides:
1. All nodes of version V come before version V+1
2. Within a version, nodes are ordered by nibble path (depth-first)

```go
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

## Special Cases

### Root Node
The root node has an empty nibble path:
```go
var RootNodeKey = func(version Version) NodeKey {
    return NodeKey{
        Version: version,
        NibblePath: NibblePath{
            Nibbles: []Nibble{},
            Length:  0,
        },
    }
}
```

### Maximum Size
The maximum encoded size of a NodeKey is:
- Version: 8 bytes
- Length: 2 bytes  
- Nibbles: 32 bytes (64 nibbles / 2)
- **Total: 42 bytes**

## Examples

### Example 1: Root Node at Version 100
```
Input:
  Version: 100
  NibblePath: [] (empty)

Encoded (hex):
  00 00 00 00 00 00 00 64  // Version 100
  00 00                    // Length 0
```

### Example 2: Leaf Node
```
Input:
  Version: 1000
  NibblePath: [0x1, 0x2, 0xA, 0xB, 0xC]

Encoded (hex):
  00 00 00 00 00 00 03 E8  // Version 1000
  00 05                    // Length 5
  12 AB C0                 // Nibbles packed (last nibble in lower bits)
```

### Example 3: Deep Internal Node
```
Input:
  Version: 50000
  NibblePath: [0xF, 0xF, 0xE, 0xE, 0xD, 0xD]

Encoded (hex):
  00 00 00 00 00 00 C3 50  // Version 50000
  00 06                    // Length 6
  FF EE DD                 // Nibbles packed
```

## PebbleDB Integration

### Key Prefix Design
To optimize range scans in PebbleDB:
```go
// All node keys start with "n" prefix
// This allows efficient iteration over all nodes
const NodeKeyPrefix = "n"

// Range scan for all nodes at version V:
// Start: "n" + EncodeVersion(V) + MinNibblePath
// End:   "n" + EncodeVersion(V+1) + MinNibblePath
```

### Batch Operations
```go
// BatchEncodeNodeKeys efficiently encodes multiple keys
func BatchEncodeNodeKeys(keys []NodeKey) [][]byte {
    encoded := make([][]byte, len(keys))
    for i, key := range keys {
        encoded[i] = key.StorageKey()
    }
    return encoded
}
```