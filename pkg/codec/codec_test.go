package codec

import (
	"bytes"
	"testing"

	"github.com/neutral/proofbox/pkg/tree"
	"github.com/neutral/proofbox/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLeafNodeCodec(t *testing.T) {
	// Create original leaf node
	original, err := tree.NewLeafNode(
		types.KeyHash([]byte("test-key")),
		[]byte("test-value"),
		100,
	)
	require.NoError(t, err)

	// Encode
	data := EncodeLeafNode(original)
	assert.Equal(t, 64, len(data), "Wrong encoded size")

	// Decode
	decoded, err := DecodeLeafNode(data, 100)
	require.NoError(t, err)

	// Compare
	assert.Equal(t, original.Key(), decoded.Key())
	assert.Equal(t, original.ValueHash(), decoded.ValueHash())
	assert.Equal(t, original.Version(), decoded.Version())
}

func TestLeafNodeCodecNil(t *testing.T) {
	// Encode nil should return nil
	data := EncodeLeafNode(nil)
	assert.Nil(t, data)
}

func TestLeafNodeCodecInvalidSize(t *testing.T) {
	// Decode with wrong size should error
	_, err := DecodeLeafNode([]byte{1, 2, 3}, 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid leaf data size")
}

func TestInternalNodeCodec(t *testing.T) {
	// Create original internal node
	original := tree.NewInternalNode(200)

	// Add some children
	err := original.SetChild(0x3, tree.Child{
		Hash:    types.Hash{0x11, 0x22, 0x33, 0x44}, // partial hash for test
		Version: 100,
		IsLeaf:  true,
	})
	require.NoError(t, err)

	err = original.SetChild(0xA, tree.Child{
		Hash:    types.Hash{0x55, 0x66, 0x77, 0x88}, // partial hash for test
		Version: 150,
		IsLeaf:  false,
	})
	require.NoError(t, err)

	// Encode
	data, err := EncodeInternalNode(original)
	require.NoError(t, err)
	expectedSize := 1 + 2*42 // 1 byte count + 2 children
	assert.Equal(t, expectedSize, len(data))

	// Decode
	decoded, err := DecodeInternalNode(data, 200)
	require.NoError(t, err)

	// Compare
	assert.Equal(t, original.NumChildren(), decoded.NumChildren())
	assert.Equal(t, original.Version(), decoded.Version())

	// Check children
	child3, ok := decoded.Child(0x3)
	assert.True(t, ok)
	assert.Equal(t, types.Version(100), child3.Version)
	assert.True(t, child3.IsLeaf)

	childA, ok := decoded.Child(0xA)
	assert.True(t, ok)
	assert.Equal(t, types.Version(150), childA.Version)
	assert.False(t, childA.IsLeaf)
}

func TestInternalNodeCodecEmpty(t *testing.T) {
	// Empty internal node
	original := tree.NewInternalNode(1)

	// Encode
	data, err := EncodeInternalNode(original)
	require.NoError(t, err)
	assert.Equal(t, 1, len(data)) // Just the count byte

	// Decode
	decoded, err := DecodeInternalNode(data, 1)
	require.NoError(t, err)
	assert.Equal(t, 0, decoded.NumChildren())
}

func TestInternalNodeCodecErrors(t *testing.T) {
	// Empty data
	_, err := DecodeInternalNode([]byte{}, 1)
	assert.Error(t, err)

	// Too many children
	_, err = DecodeInternalNode([]byte{17}, 1) // 17 > 16
	assert.Error(t, err)

	// Invalid data size
	_, err = DecodeInternalNode([]byte{1, 2, 3}, 1) // Says 1 child but not enough data
	assert.Error(t, err)

	// Invalid nibble
	data := make([]byte, 1+42)
	data[0] = 1  // 1 child
	data[1] = 16 // Invalid nibble (>15)
	_, err = DecodeInternalNode(data, 1)
	assert.Error(t, err)
}

func TestDeterministicEncoding(t *testing.T) {
	// Create node with children in random order
	node := tree.NewInternalNode(1)
	nibbles := []types.Nibble{0xF, 0x0, 0x7, 0x3, 0xA}

	for _, n := range nibbles {
		err := node.SetChild(n, tree.Child{
			Hash:    types.Hash{byte(n)}, // Simple hash for test
			Version: types.Version(n),
			IsLeaf:  n%2 == 0,
		})
		require.NoError(t, err)
	}

	// Encode multiple times
	encoding1, err := EncodeInternalNode(node)
	require.NoError(t, err)
	encoding2, err := EncodeInternalNode(node)
	require.NoError(t, err)

	// Should be identical
	assert.True(t, bytes.Equal(encoding1, encoding2), "Encoding not deterministic")

	// Decode and check nibbles are in order
	decoded, err := DecodeInternalNode(encoding1, 1)
	require.NoError(t, err)

	// Verify children are preserved
	assert.Equal(t, node.NumChildren(), decoded.NumChildren())
	for _, n := range nibbles {
		child, ok := decoded.Child(n)
		assert.True(t, ok)
		assert.Equal(t, types.Version(n), child.Version)
	}
}

func TestNodeCodec(t *testing.T) {
	codec := &NodeCodec{}

	t.Run("LeafNode", func(t *testing.T) {
		leaf, err := tree.NewLeafNode(
			types.KeyHash([]byte("test")),
			[]byte("value"),
			42,
		)
		require.NoError(t, err)

		// Encode
		data, err := codec.EncodeNode(leaf)
		require.NoError(t, err)
		assert.Equal(t, byte(tree.NodeTypeLeaf), data[0])

		// Decode
		decoded, err := codec.DecodeNode(data, 42)
		require.NoError(t, err)
		decodedLeaf, ok := decoded.(*tree.LeafNode)
		assert.True(t, ok)
		assert.Equal(t, leaf.Key(), decodedLeaf.Key())
	})

	t.Run("InternalNode", func(t *testing.T) {
		internal := tree.NewInternalNode(43)
		err := internal.SetChild(5, tree.Child{
			Hash:    types.Hash{1, 2, 3},
			Version: 10,
			IsLeaf:  true,
		})
		require.NoError(t, err)

		// Encode
		data, err := codec.EncodeNode(internal)
		require.NoError(t, err)
		assert.Equal(t, byte(tree.NodeTypeInternal), data[0])

		// Decode
		decoded, err := codec.DecodeNode(data, 43)
		require.NoError(t, err)
		decodedInternal, ok := decoded.(*tree.InternalNode)
		assert.True(t, ok)
		assert.Equal(t, internal.NumChildren(), decodedInternal.NumChildren())
	})

	t.Run("Errors", func(t *testing.T) {
		// Nil node
		_, err := codec.EncodeNode(nil)
		assert.Error(t, err)

		// Empty data
		_, err = codec.DecodeNode([]byte{}, 1)
		assert.Error(t, err)

		// Invalid node type
		_, err = codec.DecodeNode([]byte{99}, 1)
		assert.Error(t, err)
	})
}

func TestNodeCodecEstimateSize(t *testing.T) {
	codec := &NodeCodec{}

	// Nil node
	assert.Equal(t, 0, codec.EstimateSize(nil))

	// Leaf node
	leaf, _ := tree.NewLeafNode(types.KeyHash([]byte("test")), []byte("value"), 1)
	assert.Equal(t, 1+64, codec.EstimateSize(leaf)) // type byte + fixed leaf size

	// Internal node with 3 children
	internal := tree.NewInternalNode(1)
	for i := 0; i < 3; i++ {
		_ = internal.SetChild(types.Nibble(i), tree.Child{})
	}
	assert.Equal(t, 1+1+3*42, codec.EstimateSize(internal)) // type + count + children
}

func TestBatchEncoder(t *testing.T) {
	encoder := NewBatchEncoder()

	// Add some nodes
	key1 := types.RootNodeKey(100)
	leaf1, _ := tree.NewLeafNode(types.KeyHash([]byte("key1")), []byte("val1"), 100)
	err := encoder.Add(key1, leaf1)
	require.NoError(t, err)

	key2 := key1.Child(5, 101)
	internal := tree.NewInternalNode(101)
	err = encoder.Add(key2, internal)
	require.NoError(t, err)

	// Get encoded data
	data := encoder.Bytes()
	assert.Greater(t, len(data), 0)

	// Decode batch
	decoder := NewBatchDecoder(data)

	// First node
	decodedKey1, decodedNode1, err := decoder.Next()
	require.NoError(t, err)
	assert.Equal(t, key1.Version, decodedKey1.Version)
	assert.IsType(t, &tree.LeafNode{}, decodedNode1)

	// Second node
	decodedKey2, decodedNode2, err := decoder.Next()
	require.NoError(t, err)
	assert.Equal(t, key2.Version, decodedKey2.Version)
	assert.IsType(t, &tree.InternalNode{}, decodedNode2)

	// No more nodes
	assert.False(t, decoder.HasMore())
	_, _, err = decoder.Next()
	assert.NoError(t, err) // Returns nil, nil, nil for EOF
}

func TestBatchEncoderReset(t *testing.T) {
	encoder := NewBatchEncoder()

	// Add a node
	key := types.RootNodeKey(1)
	node := tree.NewInternalNode(1)
	err := encoder.Add(key, node)
	require.NoError(t, err)
	assert.Greater(t, encoder.Len(), 0)

	// Reset
	encoder.Reset()
	assert.Equal(t, 0, encoder.Len())
	assert.Empty(t, encoder.Bytes())
}

// Benchmarks

func BenchmarkLeafNodeEncode(b *testing.B) {
	leaf, _ := tree.NewLeafNode(types.KeyHash([]byte("benchmark-key")), []byte("benchmark-value"), 1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = EncodeLeafNode(leaf)
	}
}

func BenchmarkLeafNodeDecode(b *testing.B) {
	leaf, _ := tree.NewLeafNode(types.KeyHash([]byte("benchmark-key")), []byte("benchmark-value"), 1)
	data := EncodeLeafNode(leaf)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = DecodeLeafNode(data, 1)
	}
}

func BenchmarkInternalNodeEncode(b *testing.B) {
	node := tree.NewInternalNode(1)
	// Add 10 children
	for i := 0; i < 10; i++ {
		_ = node.SetChild(types.Nibble(i), tree.Child{
			Hash:    types.Hash{byte(i)},
			Version: types.Version(i),
			IsLeaf:  i%2 == 0,
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = EncodeInternalNode(node)
	}
}

func BenchmarkInternalNodeDecode(b *testing.B) {
	node := tree.NewInternalNode(1)
	// Add 10 children
	for i := 0; i < 10; i++ {
		_ = node.SetChild(types.Nibble(i), tree.Child{
			Hash:    types.Hash{byte(i)},
			Version: types.Version(i),
			IsLeaf:  i%2 == 0,
		})
	}
	data, _ := EncodeInternalNode(node)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = DecodeInternalNode(data, 1)
	}
}

