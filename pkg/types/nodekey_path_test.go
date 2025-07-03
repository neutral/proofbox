package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNodeKeyWithPath(t *testing.T) {
	key := NodeKey{
		Version: 10,
		NibblePath: NibblePath{
			Nibbles: []Nibble{1, 2, 3},
			Length:  3,
		},
	}

	newPath := NibblePath{
		Nibbles: []Nibble{4, 5, 6, 7},
		Length:  4,
	}

	newKey := key.WithPath(newPath)

	// Check that version is preserved
	assert.Equal(t, key.Version, newKey.Version)

	// Check that path is replaced
	assert.Equal(t, newPath, newKey.NibblePath)

	// Original key should be unchanged
	assert.Equal(t, uint16(3), key.NibblePath.Length)
}

func TestNodeKeyExtendPath(t *testing.T) {
	key := NodeKey{
		Version: 10,
		NibblePath: NibblePath{
			Nibbles: []Nibble{1, 2, 3},
			Length:  3,
		},
	}

	// Test ExtendPath
	extended := key.ExtendPath(4)
	assert.Equal(t, Version(10), extended.Version)
	assert.Equal(t, uint16(4), extended.NibblePath.Length)
	assert.Equal(t, []Nibble{1, 2, 3, 4}, extended.NibblePath.Nibbles)

	// Original key should be unchanged
	assert.Equal(t, uint16(3), key.NibblePath.Length)

	// Test extending empty path
	rootKey := RootNodeKey(5)
	extended2 := rootKey.ExtendPath(7)
	assert.Equal(t, Version(5), extended2.Version)
	assert.Equal(t, uint16(1), extended2.NibblePath.Length)
	assert.Equal(t, []Nibble{7}, extended2.NibblePath.Nibbles)
}

func TestNodeKeyParentKey(t *testing.T) {
	// Test normal parent
	key := NodeKey{
		Version: 10,
		NibblePath: NibblePath{
			Nibbles: []Nibble{1, 2, 3, 4},
			Length:  4,
		},
	}

	parent, err := key.ParentKey()
	require.NoError(t, err)
	assert.Equal(t, Version(10), parent.Version)
	assert.Equal(t, uint16(3), parent.NibblePath.Length)
	assert.Equal(t, []Nibble{1, 2, 3}, parent.NibblePath.Nibbles)

	// Test parent of depth-1 node
	key2 := NodeKey{
		Version: 5,
		NibblePath: NibblePath{
			Nibbles: []Nibble{9},
			Length:  1,
		},
	}

	parent2, err := key2.ParentKey()
	require.NoError(t, err)
	assert.Equal(t, Version(5), parent2.Version)
	assert.Equal(t, uint16(0), parent2.NibblePath.Length)
	assert.True(t, parent2.IsRoot())

	// Test root parent (should error)
	root := RootNodeKey(10)
	_, err = root.ParentKey()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "root node has no parent")
}

func TestNodeKeyDepth(t *testing.T) {
	testCases := []struct {
		name     string
		key      NodeKey
		expected int
	}{
		{
			name:     "root node",
			key:      RootNodeKey(1),
			expected: 0,
		},
		{
			name: "depth 1",
			key: NodeKey{
				Version:    1,
				NibblePath: NibblePath{Nibbles: []Nibble{5}, Length: 1},
			},
			expected: 1,
		},
		{
			name: "depth 5",
			key: NodeKey{
				Version:    1,
				NibblePath: NibblePath{Nibbles: []Nibble{1, 2, 3, 4, 5}, Length: 5},
			},
			expected: 5,
		},
		{
			name: "max depth",
			key: NodeKey{
				Version:    1,
				NibblePath: Key{}.ToNibblePath(),
			},
			expected: MaxTreeDepth,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.key.Depth())
		})
	}
}

func TestNodeKeyConsistencyWithChild(t *testing.T) {
	// Test that ExtendPath is consistent with existing Child method
	key := NodeKey{
		Version: 10,
		NibblePath: NibblePath{
			Nibbles: []Nibble{1, 2, 3},
			Length:  3,
		},
	}

	nibble := Nibble(4)
	childVersion := Version(15)

	// Using Child method
	child1 := key.Child(nibble, childVersion)

	// Using ExtendPath and WithPath
	extended := key.ExtendPath(nibble)
	child2 := extended.WithPath(extended.NibblePath)
	child2.Version = childVersion

	// They should produce the same nibble path
	assert.Equal(t, child1.NibblePath, child2.NibblePath)
	assert.Equal(t, child1.Version, child2.Version)
}

func TestNodeKeyNavigationRoundTrip(t *testing.T) {
	// Test that we can navigate down and up the tree
	root := RootNodeKey(100)

	// Go down 4 levels
	key := root
	nibbles := []Nibble{5, 8, 2, 9}
	for _, n := range nibbles {
		key = key.ExtendPath(n)
	}

	assert.Equal(t, 4, key.Depth())
	assert.Equal(t, nibbles, key.NibblePath.Nibbles)

	// Go back up to root
	current := key
	for current.Depth() > 0 {
		parent, err := current.ParentKey()
		require.NoError(t, err)
		assert.Equal(t, current.Depth()-1, parent.Depth())
		current = parent
	}

	assert.True(t, current.IsRoot())
	assert.Equal(t, root.Version, current.Version)
}

func TestNodeKeyPathOperationsWithEncoding(t *testing.T) {
	// Test that path operations preserve encoding/decoding
	key := NodeKey{
		Version: 42,
		NibblePath: NibblePath{
			Nibbles: []Nibble{0xA, 0xB, 0xC, 0xD},
			Length:  4,
		},
	}

	// Extend path
	extended := key.ExtendPath(0xE)

	// Encode and decode
	encoded := EncodeNodeKey(extended)
	decoded, err := DecodeNodeKey(encoded)
	require.NoError(t, err)

	assert.Equal(t, extended.Version, decoded.Version)
	assert.Equal(t, extended.NibblePath, decoded.NibblePath)
	assert.Equal(t, extended.Depth(), decoded.Depth())
}

func BenchmarkNodeKeyExtendPath(b *testing.B) {
	key := NodeKey{
		Version:    10,
		NibblePath: NibblePath{Nibbles: make([]Nibble, 32), Length: 32},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = key.ExtendPath(5)
	}
}

func BenchmarkNodeKeyParentKey(b *testing.B) {
	key := NodeKey{
		Version:    10,
		NibblePath: Key{}.ToNibblePath(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Ignore error in benchmark - key has non-zero depth
		parent, _ := key.ParentKey()
		_ = parent
	}
}

func BenchmarkNodeKeyDepth(b *testing.B) {
	key := NodeKey{
		Version:    10,
		NibblePath: Key{}.ToNibblePath(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = key.Depth()
	}
}
