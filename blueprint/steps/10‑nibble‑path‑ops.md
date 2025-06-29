---
id: step.10.nibble‑path‑ops
depends_on:
  - step.09.insert‑basic
tags: [nibble, path, step]
---

## Objective

Implement comprehensive nibble path operations required for tree navigation, splitting, and manipulation.

## Implements

- Nibble path utilities for tree traversal
- Foundation for leaf splitting and internal node navigation
- Common prefix detection for key divergence

## Technical Details

### NibblePath Enhancement

The current `NibblePath` struct needs additional methods to support tree operations:

```go
// Enhanced NibblePath methods in pkg/types/key.go

// CommonPrefixLength returns the number of matching nibbles from the start
func (np NibblePath) CommonPrefixLength(other NibblePath) int {
    minLen := np.Length
    if other.Length < minLen {
        minLen = other.Length
    }
    
    for i := uint16(0); i < minLen; i++ {
        if np.Nibbles[i] != other.Nibbles[i] {
            return int(i)
        }
    }
    return int(minLen)
}

// GetNibble returns the nibble at the specified index with bounds checking
func (np NibblePath) GetNibble(index int) (Nibble, error) {
    if index < 0 || index >= int(np.Length) {
        return 0, fmt.Errorf("nibble index %d out of bounds [0, %d)", index, np.Length)
    }
    return np.Nibbles[index], nil
}

// Prefix returns a new NibblePath containing the first 'length' nibbles
func (np NibblePath) Prefix(length int) NibblePath {
    if length <= 0 {
        return NibblePath{Nibbles: []Nibble{}, Length: 0}
    }
    
    if length > int(np.Length) {
        length = int(np.Length)
    }
    
    newNibbles := make([]Nibble, length)
    copy(newNibbles, np.Nibbles[:length])
    
    return NibblePath{
        Nibbles: newNibbles,
        Length:  uint16(length),
    }
}

// Equals returns true if two nibble paths are identical
func (np NibblePath) Equals(other NibblePath) bool {
    if np.Length != other.Length {
        return false
    }
    
    for i := uint16(0); i < np.Length; i++ {
        if np.Nibbles[i] != other.Nibbles[i] {
            return false
        }
    }
    return true
}

// Append returns a new NibblePath with the given nibble appended
func (np NibblePath) Append(nibble Nibble) NibblePath {
    newNibbles := make([]Nibble, np.Length+1)
    copy(newNibbles, np.Nibbles)
    newNibbles[np.Length] = nibble
    
    return NibblePath{
        Nibbles: newNibbles,
        Length:  np.Length + 1,
    }
}

// IsPrefix returns true if this path is a prefix of the other path
func (np NibblePath) IsPrefix(other NibblePath) bool {
    if np.Length > other.Length {
        return false
    }
    
    for i := uint16(0); i < np.Length; i++ {
        if np.Nibbles[i] != other.Nibbles[i] {
            return false
        }
    }
    return true
}

// Skip returns a new NibblePath with the first 'count' nibbles removed
func (np NibblePath) Skip(count int) NibblePath {
    if count >= int(np.Length) {
        return NibblePath{Nibbles: []Nibble{}, Length: 0}
    }
    
    if count <= 0 {
        return np
    }
    
    newLength := int(np.Length) - count
    newNibbles := make([]Nibble, newLength)
    copy(newNibbles, np.Nibbles[count:])
    
    return NibblePath{
        Nibbles: newNibbles,
        Length:  uint16(newLength),
    }
}
```

### Key to NibblePath Conversion

```go
// NewNibblePath creates a NibblePath from a byte slice
func NewNibblePath(data []byte) NibblePath {
    nibbles := make([]Nibble, 0, len(data)*2)
    for _, b := range data {
        nibbles = append(nibbles, Nibble(b>>4), Nibble(b&0x0F))
    }
    return NibblePath{
        Nibbles: nibbles,
        Length:  uint16(len(nibbles)),
    }
}

// ToBytes converts a NibblePath back to bytes (for even-length paths)
func (np NibblePath) ToBytes() ([]byte, error) {
    if np.Length%2 != 0 {
        return nil, fmt.Errorf("cannot convert odd-length nibble path to bytes")
    }
    
    bytes := make([]byte, np.Length/2)
    for i := 0; i < len(bytes); i++ {
        high := np.Nibbles[i*2]
        low := np.Nibbles[i*2+1]
        bytes[i] = byte(high)<<4 | byte(low)
    }
    
    return bytes, nil
}
```

### NodeKey Path Operations

```go
// Enhanced NodeKey methods in pkg/types/nodekey.go

// WithPath returns a new NodeKey with the specified path
func (nk NodeKey) WithPath(path NibblePath) NodeKey {
    return NodeKey{
        Version:    nk.Version,
        NibblePath: path,
    }
}

// ExtendPath returns a new NodeKey with nibble appended to path
func (nk NodeKey) ExtendPath(nibble Nibble) NodeKey {
    return NodeKey{
        Version:    nk.Version,
        NibblePath: nk.NibblePath.Append(nibble),
    }
}

// ParentKey returns the NodeKey of this node's parent
func (nk NodeKey) ParentKey() (NodeKey, error) {
    if nk.NibblePath.Length == 0 {
        return NodeKey{}, fmt.Errorf("root node has no parent")
    }
    
    return NodeKey{
        Version:    nk.Version,
        NibblePath: nk.NibblePath.Prefix(int(nk.NibblePath.Length) - 1),
    }, nil
}

// Depth returns the depth of this node in the tree
func (nk NodeKey) Depth() int {
    return int(nk.NibblePath.Length)
}
```

## Testing Requirements

### NibblePath Operation Tests

```go
func TestCommonPrefixLength(t *testing.T) {
    testCases := []struct {
        name     string
        path1    NibblePath
        path2    NibblePath
        expected int
    }{
        {
            name:     "identical paths",
            path1:    NibblePath{Nibbles: []Nibble{1, 2, 3, 4}, Length: 4},
            path2:    NibblePath{Nibbles: []Nibble{1, 2, 3, 4}, Length: 4},
            expected: 4,
        },
        {
            name:     "different at first nibble",
            path1:    NibblePath{Nibbles: []Nibble{1, 2, 3, 4}, Length: 4},
            path2:    NibblePath{Nibbles: []Nibble{2, 2, 3, 4}, Length: 4},
            expected: 0,
        },
        {
            name:     "common prefix of 2",
            path1:    NibblePath{Nibbles: []Nibble{1, 2, 3, 4}, Length: 4},
            path2:    NibblePath{Nibbles: []Nibble{1, 2, 5, 6}, Length: 4},
            expected: 2,
        },
        {
            name:     "different lengths",
            path1:    NibblePath{Nibbles: []Nibble{1, 2, 3}, Length: 3},
            path2:    NibblePath{Nibbles: []Nibble{1, 2, 3, 4, 5}, Length: 5},
            expected: 3,
        },
        {
            name:     "empty paths",
            path1:    NibblePath{Nibbles: []Nibble{}, Length: 0},
            path2:    NibblePath{Nibbles: []Nibble{}, Length: 0},
            expected: 0,
        },
    }
    
    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            result := tc.path1.CommonPrefixLength(tc.path2)
            assert.Equal(t, tc.expected, result)
        })
    }
}

func TestGetNibble(t *testing.T) {
    path := NibblePath{Nibbles: []Nibble{1, 2, 3, 4}, Length: 4}
    
    // Valid indices
    for i := 0; i < 4; i++ {
        nibble, err := path.GetNibble(i)
        assert.NoError(t, err)
        assert.Equal(t, Nibble(i+1), nibble)
    }
    
    // Invalid indices
    _, err := path.GetNibble(-1)
    assert.Error(t, err)
    
    _, err = path.GetNibble(4)
    assert.Error(t, err)
}

func TestPrefix(t *testing.T) {
    path := NibblePath{Nibbles: []Nibble{1, 2, 3, 4, 5}, Length: 5}
    
    // Normal prefix
    prefix := path.Prefix(3)
    assert.Equal(t, uint16(3), prefix.Length)
    assert.Equal(t, []Nibble{1, 2, 3}, prefix.Nibbles)
    
    // Prefix longer than path
    prefix = path.Prefix(10)
    assert.Equal(t, path.Length, prefix.Length)
    
    // Empty prefix
    prefix = path.Prefix(0)
    assert.Equal(t, uint16(0), prefix.Length)
    
    // Negative prefix
    prefix = path.Prefix(-1)
    assert.Equal(t, uint16(0), prefix.Length)
}

func TestNibblePathConversion(t *testing.T) {
    // Test byte to nibble conversion
    data := []byte{0x12, 0x34, 0x56}
    path := NewNibblePath(data)
    
    assert.Equal(t, uint16(6), path.Length)
    expected := []Nibble{0x1, 0x2, 0x3, 0x4, 0x5, 0x6}
    assert.Equal(t, expected, path.Nibbles)
    
    // Test nibble to byte conversion
    bytes, err := path.ToBytes()
    assert.NoError(t, err)
    assert.Equal(t, data, bytes)
    
    // Test odd-length path
    oddPath := NibblePath{Nibbles: []Nibble{1, 2, 3}, Length: 3}
    _, err = oddPath.ToBytes()
    assert.Error(t, err)
}

func TestNodeKeyPathOperations(t *testing.T) {
    key := NodeKey{
        Version: 10,
        NibblePath: NibblePath{
            Nibbles: []Nibble{1, 2, 3},
            Length:  3,
        },
    }
    
    // Test ExtendPath
    extended := key.ExtendPath(4)
    assert.Equal(t, uint16(4), extended.NibblePath.Length)
    assert.Equal(t, Nibble(4), extended.NibblePath.Nibbles[3])
    
    // Test ParentKey
    parent, err := key.ParentKey()
    assert.NoError(t, err)
    assert.Equal(t, uint16(2), parent.NibblePath.Length)
    
    // Test root parent
    root := NodeKey{Version: 10, NibblePath: NibblePath{}}
    _, err = root.ParentKey()
    assert.Error(t, err)
    
    // Test Depth
    assert.Equal(t, 3, key.Depth())
    assert.Equal(t, 0, root.Depth())
}
```

### Benchmark Tests

```go
func BenchmarkCommonPrefixLength(b *testing.B) {
    path1 := Key{}.ToNibblePath() // Full 64-nibble path
    path2 := Key{0xFF}.ToNibblePath() // Differs at first byte
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = path1.CommonPrefixLength(path2)
    }
}

func BenchmarkNibblePathConversion(b *testing.B) {
    key := Key{}
    for i := range key {
        key[i] = byte(i)
    }
    
    b.Run("ToNibblePath", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            _ = NewNibblePath(key[:])
        }
    })
    
    b.Run("ToBytes", func(b *testing.B) {
        path := key.ToNibblePath()
        b.ResetTimer()
        for i := 0; i < b.N; i++ {
            _, _ = path.ToBytes()
        }
    })
}
```

## Implementation Steps

1. Add NibblePath methods to `pkg/types/key.go`
2. Add NodeKey path operations to `pkg/types/nodekey.go`
3. Create comprehensive tests in `pkg/types/key_nibblepath_test.go`
4. Create NodeKey tests in `pkg/types/nodekey_path_test.go`
5. Add benchmarks for performance validation

## Performance Considerations

- Use efficient slice operations for prefix/append
- Avoid unnecessary allocations in hot paths
- Consider caching common prefix lengths for repeated comparisons
- Optimize for the common case of full 64-nibble paths

## Done When ✓

- [ ] CommonPrefixLength correctly finds divergence point
- [ ] GetNibble performs bounds checking
- [ ] Prefix creates proper sub-paths
- [ ] Append efficiently extends paths
- [ ] NodeKey path operations maintain consistency
- [ ] Conversion between bytes and nibbles is bidirectional
- [ ] All edge cases tested (empty paths, boundary conditions)
- [ ] Performance benchmarks show efficient operations
- [ ] 100% test coverage for new methods