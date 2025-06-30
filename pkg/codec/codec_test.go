package codec

import (
	"testing"

	"github.com/neutral/proofbox/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockLeafNode implements types.LeafNodeInterface for testing
type MockLeafNode struct {
	key       types.Key
	valueHash types.Hash
	value     []byte
	version   types.Version
	hash      types.Hash
}

func (m *MockLeafNode) Type() types.NodeType        { return types.NodeTypeLeaf }
func (m *MockLeafNode) Hash() types.Hash            { return m.hash }
func (m *MockLeafNode) IsCached() bool              { return true }
func (m *MockLeafNode) Version() types.Version      { return m.version }
func (m *MockLeafNode) Key() types.Key              { return m.key }
func (m *MockLeafNode) ValueHash() types.Hash       { return m.valueHash }
func (m *MockLeafNode) Value() []byte               { return m.value }
func (m *MockLeafNode) SetValue(value []byte) error { m.value = value; return nil }
func (m *MockLeafNode) Clone(v types.Version) types.Node {
	return &MockLeafNode{
		key:       m.key,
		valueHash: m.valueHash,
		value:     m.value,
		version:   v,
		hash:      m.hash,
	}
}

// MockInternalNode implements types.InternalNodeInterface for testing
type MockInternalNode struct {
	children map[types.Nibble]types.Child
	version  types.Version
	hash     types.Hash
}

func (m *MockInternalNode) Type() types.NodeType   { return types.NodeTypeInternal }
func (m *MockInternalNode) Hash() types.Hash       { return m.hash }
func (m *MockInternalNode) IsCached() bool         { return true }
func (m *MockInternalNode) Version() types.Version { return m.version }
func (m *MockInternalNode) Child(nibble types.Nibble) (types.Child, bool) {
	c, ok := m.children[nibble]
	return c, ok
}
func (m *MockInternalNode) Children() map[types.Nibble]types.Child {
	result := make(map[types.Nibble]types.Child)
	for k, v := range m.children {
		result[k] = v
	}
	return result
}
func (m *MockInternalNode) NumChildren() int { return len(m.children) }
func (m *MockInternalNode) GetOnlyChild() (types.Nibble, types.Child, bool) {
	if len(m.children) != 1 {
		return 0, types.Child{}, false
	}
	for n, c := range m.children {
		return n, c, true
	}
	return 0, types.Child{}, false
}
func (m *MockInternalNode) SetChild(nibble types.Nibble, child types.Child) error {
	m.children[nibble] = child
	return nil
}
func (m *MockInternalNode) RemoveChild(nibble types.Nibble) error {
	delete(m.children, nibble)
	return nil
}
func (m *MockInternalNode) Clone(v types.Version) types.Node {
	newChildren := make(map[types.Nibble]types.Child)
	for k, v := range m.children {
		newChildren[k] = v
	}
	return &MockInternalNode{
		children: newChildren,
		version:  v,
		hash:     m.hash,
	}
}

func TestLeafNodeInterfaceCodec(t *testing.T) {
	// Create mock leaf node
	original := &MockLeafNode{
		key:       types.KeyHash([]byte("test-key")),
		valueHash: types.Hash{0x01, 0x02, 0x03},
		version:   100,
	}

	// Encode
	data := EncodeLeafNodeInterface(original)
	assert.Equal(t, 64, len(data), "Wrong encoded size")

	// Verify key in first 32 bytes
	assert.Equal(t, original.key[:], data[0:32])

	// Verify value hash in next 32 bytes
	assert.Equal(t, original.valueHash[:], data[32:64])
}

func TestInternalNodeInterfaceCodec(t *testing.T) {
	// Create mock internal node
	original := &MockInternalNode{
		children: map[types.Nibble]types.Child{
			0x5: {Hash: types.Hash{0x05}, Version: 10, IsLeaf: true},
			0xA: {Hash: types.Hash{0x0A}, Version: 20, IsLeaf: false},
		},
		version: 100,
	}

	// Encode
	data, err := EncodeInternalNodeInterface(original)
	require.NoError(t, err)

	// Expected size: 1 + 2*42 = 85 bytes
	assert.Equal(t, 85, len(data))

	// Check number of children
	assert.Equal(t, byte(2), data[0])

	// Children should be in sorted order (0x5, 0xA)
	// First child at offset 1
	assert.Equal(t, byte(0x5), data[1])
	// Second child at offset 43
	assert.Equal(t, byte(0xA), data[43])
}

func TestNodeCodec(t *testing.T) {
	codec := &NodeCodec{}

	t.Run("EncodeLeafNode", func(t *testing.T) {
		node := &MockLeafNode{
			key:       types.KeyHash([]byte("test")),
			valueHash: types.Hash{0x01},
			version:   1,
		}

		data, err := codec.EncodeNode(node)
		require.NoError(t, err)
		assert.Equal(t, byte(types.NodeTypeLeaf), data[0])
		assert.Equal(t, 65, len(data)) // 1 + 64
	})

	t.Run("EncodeInternalNode", func(t *testing.T) {
		node := &MockInternalNode{
			children: map[types.Nibble]types.Child{
				0x1: {Hash: types.Hash{0x01}, Version: 1, IsLeaf: true},
			},
			version: 1,
		}

		data, err := codec.EncodeNode(node)
		require.NoError(t, err)
		assert.Equal(t, byte(types.NodeTypeInternal), data[0])
		assert.Equal(t, 44, len(data)) // 1 + 1 + 42
	})

	t.Run("EstimateSize", func(t *testing.T) {
		leaf := &MockLeafNode{}
		assert.Equal(t, 65, codec.EstimateSize(leaf))

		internal := &MockInternalNode{
			children: map[types.Nibble]types.Child{
				0x1: {},
				0x2: {},
			},
		}
		assert.Equal(t, 86, codec.EstimateSize(internal)) // 1 + 1 + 2*42
	})
}

func TestBatchCodec(t *testing.T) {
	t.Run("EncodeDecode", func(t *testing.T) {
		encoder := NewBatchEncoder()

		// Add nodes
		key1 := types.NodeKey{Version: 1, NibblePath: types.NibblePath{Nibbles: []types.Nibble{0x1}, Length: 1}}
		node1 := &MockLeafNode{
			key:       types.KeyHash([]byte("key1")),
			valueHash: types.Hash{0x01},
			version:   1,
		}

		err := encoder.Add(key1, node1)
		require.NoError(t, err)

		key2 := types.NodeKey{Version: 2, NibblePath: types.NibblePath{Nibbles: []types.Nibble{0x2}, Length: 1}}
		node2 := &MockInternalNode{
			children: map[types.Nibble]types.Child{
				0x3: {Hash: types.Hash{0x03}, Version: 3, IsLeaf: false},
			},
			version: 2,
		}

		err = encoder.Add(key2, node2)
		require.NoError(t, err)

		// Get encoded data
		data := encoder.Bytes()
		assert.NotEmpty(t, data)

		// Note: Decoding would require the factory pattern to be set up
		// which happens at runtime via init()
	})

	t.Run("BatchEncoderReset", func(t *testing.T) {
		encoder := NewBatchEncoder()

		key := types.NodeKey{Version: 1}
		node := &MockLeafNode{version: 1}

		err := encoder.Add(key, node)
		require.NoError(t, err)
		assert.NotZero(t, encoder.Len())

		encoder.Reset()
		assert.Zero(t, encoder.Len())
	})
}

func TestInterfaceEncoders(t *testing.T) {
	t.Run("LeafNodeInterface", func(t *testing.T) {
		leaf := &MockLeafNode{
			key:       types.KeyHash([]byte("test-key")),
			valueHash: types.Hash{0xFF, 0xEE},
			version:   42,
		}

		data := EncodeLeafNodeInterface(leaf)
		assert.Equal(t, 64, len(data))

		// Check key
		assert.Equal(t, leaf.key[:], data[0:32])

		// Check value hash
		assert.Equal(t, leaf.valueHash[:], data[32:64])
	})

	t.Run("InternalNodeInterfaceTooManyChildren", func(t *testing.T) {
		// Create node with too many children
		internal := &MockInternalNode{
			children: make(map[types.Nibble]types.Child),
			version:  1,
		}

		// Add 17 children (max is 16)
		for i := 0; i < 17; i++ {
			internal.children[types.Nibble(i)] = types.Child{
				Hash:    types.Hash{byte(i)},
				Version: 1,
				IsLeaf:  false,
			}
		}

		_, err := EncodeInternalNodeInterface(internal)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "too many children")
	})

	t.Run("InternalNodeInterfaceOrdering", func(t *testing.T) {
		internal := &MockInternalNode{
			children: map[types.Nibble]types.Child{
				0xF: {Hash: types.Hash{0x0F}, Version: 1, IsLeaf: false},
				0x0: {Hash: types.Hash{0x00}, Version: 1, IsLeaf: false},
				0x8: {Hash: types.Hash{0x08}, Version: 1, IsLeaf: false},
			},
			version: 1,
		}

		data, err := EncodeInternalNodeInterface(internal)
		require.NoError(t, err)

		// Check that children are in order: 0x0, 0x8, 0xF
		assert.Equal(t, byte(3), data[0]) // 3 children
		assert.Equal(t, byte(0x0), data[1])
		assert.Equal(t, byte(0x8), data[43])
		assert.Equal(t, byte(0xF), data[85])
	})
}

// Benchmarks

func BenchmarkLeafNodeEncode(b *testing.B) {
	node := &MockLeafNode{
		key:       types.KeyHash([]byte("benchmark-key")),
		valueHash: types.Hash{0x01, 0x02, 0x03},
		version:   100,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = EncodeLeafNodeInterface(node)
	}
}

func BenchmarkInternalNodeEncode(b *testing.B) {
	node := &MockInternalNode{
		children: map[types.Nibble]types.Child{
			0x1: {Hash: types.Hash{0x01}, Version: 1, IsLeaf: true},
			0x2: {Hash: types.Hash{0x02}, Version: 2, IsLeaf: false},
			0x3: {Hash: types.Hash{0x03}, Version: 3, IsLeaf: true},
			0x4: {Hash: types.Hash{0x04}, Version: 4, IsLeaf: false},
		},
		version: 100,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = EncodeInternalNodeInterface(node)
	}
}

func BenchmarkBatchEncode(b *testing.B) {
	nodes := make([]types.Node, 100)
	keys := make([]types.NodeKey, 100)

	for i := 0; i < 100; i++ {
		if i%2 == 0 {
			nodes[i] = &MockLeafNode{
				key:       types.KeyHash([]byte{byte(i)}),
				valueHash: types.Hash{byte(i)},
				version:   types.Version(i),
			}
		} else {
			nodes[i] = &MockInternalNode{
				children: map[types.Nibble]types.Child{
					types.Nibble(i % 16): {Hash: types.Hash{byte(i)}, Version: types.Version(i), IsLeaf: false},
				},
				version: types.Version(i),
			}
		}
		keys[i] = types.NodeKey{
			Version: types.Version(i),
			NibblePath: types.NibblePath{
				Nibbles: []types.Nibble{types.Nibble(i % 16)},
				Length:  1,
			},
		}
	}

	encoder := NewBatchEncoder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		encoder.Reset()
		for j := 0; j < 100; j++ {
			_ = encoder.Add(keys[j], nodes[j])
		}
		_ = encoder.Bytes()
	}
}
