package tree

import (
	"bytes"
	"testing"

	"github.com/neutral/proofbox/pkg/crypto"
	"github.com/neutral/proofbox/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewInternalNode(t *testing.T) {
	version := types.Version(42)
	node := NewInternalNode(version)

	assert.NotNil(t, node)
	assert.Equal(t, version, node.version)
	assert.NotNil(t, node.children)
	assert.Equal(t, 0, len(node.children))
	assert.Equal(t, NodeTypeInternal, node.Type())
}

func TestInternalNodeHash(t *testing.T) {
	node := NewInternalNode(1)

	// Empty internal node should use empty hashes for all children
	hash := node.Hash()

	parts := [][]byte{{0x00}} // NodeTypeInternal
	for i := 0; i < 16; i++ {
		parts = append(parts, crypto.EmptyTreeHash[:])
	}
	expected := crypto.DefaultHasher.HashConcat(parts...)
	assert.Equal(t, expected, hash)

	// Add a child
	childHash := crypto.DefaultHasher.Hash([]byte("child"))
	err := node.SetChild(5, Child{
		Hash:    childHash,
		Version: 1,
		IsLeaf:  true,
	})
	assert.NoError(t, err)

	// Hash should change
	newHash := node.Hash()
	assert.NotEqual(t, hash, newHash)

	// Hash should be cached
	assert.True(t, node.IsCached())

	// Hash should be deterministic
	assert.Equal(t, newHash, node.Hash())
}

func TestInternalNodeChildren(t *testing.T) {
	node := NewInternalNode(1)

	// Add multiple children
	for i := types.Nibble(0); i < 5; i++ {
		child := Child{
			Hash:    crypto.DefaultHasher.Hash([]byte{byte(i)}),
			Version: 1,
			IsLeaf:  true,
		}
		err := node.SetChild(i, child)
		assert.NoError(t, err)
	}

	assert.Equal(t, 5, node.NumChildren())

	// Test child retrieval
	child, exists := node.Child(3)
	assert.True(t, exists)
	assert.True(t, child.IsLeaf)

	// Test non-existent child
	_, exists = node.Child(10)
	assert.False(t, exists)

	// Test clone
	cloneNode := node.Clone(2)
	clone, ok := cloneNode.(*InternalNode)
	assert.True(t, ok)
	assert.Equal(t, types.Version(2), clone.Version())
	assert.Equal(t, node.NumChildren(), clone.NumChildren())

	// Verify clone is independent
	err := clone.SetChild(10, Child{
		Hash:    crypto.DefaultHasher.Hash([]byte("new")),
		Version: 2,
		IsLeaf:  false,
	})
	assert.NoError(t, err)
	assert.Equal(t, 6, clone.NumChildren())
	assert.Equal(t, 5, node.NumChildren()) // Original unchanged
}

func TestInternalNodeSetChild(t *testing.T) {
	node := NewInternalNode(1)

	// Valid nibble
	child := Child{
		Hash:    crypto.DefaultHasher.Hash([]byte("test")),
		Version: 1,
		IsLeaf:  true,
	}
	err := node.SetChild(10, child)
	assert.NoError(t, err)

	// Invalid nibble
	err = node.SetChild(16, child) // > MaxNibbleValue
	assert.Error(t, err)

	// Update existing child
	newChild := Child{
		Hash:    crypto.DefaultHasher.Hash([]byte("updated")),
		Version: 2,
		IsLeaf:  false,
	}
	err = node.SetChild(10, newChild)
	assert.NoError(t, err)

	retrieved, exists := node.Child(10)
	assert.True(t, exists)
	assert.Equal(t, newChild, retrieved)
}

func TestInternalNodeRemoveChild(t *testing.T) {
	node := NewInternalNode(1)

	// Add some children
	for i := types.Nibble(0); i < 3; i++ {
		child := Child{
			Hash:    crypto.DefaultHasher.Hash([]byte{byte(i)}),
			Version: 1,
			IsLeaf:  true,
		}
		err := node.SetChild(i, child)
		require.NoError(t, err)
	}

	assert.Equal(t, 3, node.NumChildren())

	// Remove a child
	node.RemoveChild(1)
	assert.Equal(t, 2, node.NumChildren())

	_, exists := node.Child(1)
	assert.False(t, exists)

	// Remove non-existent child (should not panic)
	node.RemoveChild(10)
	assert.Equal(t, 2, node.NumChildren())
}

func TestInternalNodeGetOnlyChild(t *testing.T) {
	node := NewInternalNode(1)

	// No children
	_, _, ok := node.GetOnlyChild()
	assert.False(t, ok)

	// One child
	child := Child{
		Hash:    crypto.DefaultHasher.Hash([]byte("only")),
		Version: 1,
		IsLeaf:  true,
	}
	err := node.SetChild(7, child)
	require.NoError(t, err)

	nibble, retrievedChild, ok := node.GetOnlyChild()
	assert.True(t, ok)
	assert.Equal(t, types.Nibble(7), nibble)
	assert.Equal(t, child, retrievedChild)

	// Multiple children
	err = node.SetChild(3, child)
	require.NoError(t, err)

	_, _, ok = node.GetOnlyChild()
	assert.False(t, ok)
}

func TestInternalNodeGetChildren(t *testing.T) {
	node := NewInternalNode(1)

	// Add some children
	expected := make(map[types.Nibble]Child)
	for i := types.Nibble(0); i < 5; i++ {
		child := Child{
			Hash:    crypto.DefaultHasher.Hash([]byte{byte(i)}),
			Version: types.Version(i),
			IsLeaf:  i%2 == 0,
		}
		err := node.SetChild(i, child)
		require.NoError(t, err)
		expected[i] = child
	}

	// Get all children
	children := node.Children()
	assert.Equal(t, len(expected), len(children))

	for nibble, child := range expected {
		assert.Equal(t, child, children[nibble])
	}

	// Verify it's a copy (modifying returned map doesn't affect node)
	children[10] = Child{Hash: types.EmptyHash()}
	assert.Equal(t, 5, node.NumChildren())
}

func TestInternalNodeHashInvalidatesCache(t *testing.T) {
	node := NewInternalNode(1)

	// Compute initial hash
	hash1 := node.Hash()
	assert.True(t, node.IsCached())

	// Add a child (should invalidate cache)
	child := Child{
		Hash:    crypto.DefaultHasher.Hash([]byte("test")),
		Version: 1,
		IsLeaf:  true,
	}
	err := node.SetChild(0, child)
	require.NoError(t, err)

	// Hash should be different
	hash2 := node.Hash()
	assert.False(t, bytes.Equal(hash1[:], hash2[:]))

	// Remove child (should invalidate cache again)
	node.RemoveChild(0)
	hash3 := node.Hash()
	assert.Equal(t, hash1, hash3) // Should be back to original
}

func TestInternalNodeHashOrdering(t *testing.T) {
	// Verify that children are hashed in nibble order
	node := NewInternalNode(1)

	// Add children in non-sequential order
	nibbles := []types.Nibble{15, 0, 7, 3, 11}
	for _, n := range nibbles {
		child := Child{
			Hash:    crypto.DefaultHasher.Hash([]byte{byte(n)}),
			Version: 1,
			IsLeaf:  true,
		}
		err := node.SetChild(n, child)
		require.NoError(t, err)
	}

	// Get hash
	hash := node.Hash()

	// Create another node with same children added in different order
	node2 := NewInternalNode(1)
	for i := len(nibbles) - 1; i >= 0; i-- {
		n := nibbles[i]
		child := Child{
			Hash:    crypto.DefaultHasher.Hash([]byte{byte(n)}),
			Version: 1,
			IsLeaf:  true,
		}
		err := node2.SetChild(n, child)
		require.NoError(t, err)
	}

	// Hashes should be identical
	hash2 := node2.Hash()
	assert.Equal(t, hash, hash2)
}

func BenchmarkInternalNodeHash(b *testing.B) {
	node := NewInternalNode(1)

	// Add some children
	for i := types.Nibble(0); i < 10; i++ {
		child := Child{
			Hash:    crypto.DefaultHasher.Hash([]byte{byte(i)}),
			Version: 1,
			IsLeaf:  true,
		}
		_ = node.SetChild(i, child)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = node.Hash()
	}
}

func BenchmarkInternalNodeHashUncached(b *testing.B) {
	// Create a template node
	template := NewInternalNode(1)
	for i := types.Nibble(0); i < 10; i++ {
		child := Child{
			Hash:    crypto.DefaultHasher.Hash([]byte{byte(i)}),
			Version: 1,
			IsLeaf:  true,
		}
		_ = template.SetChild(i, child)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		node := template.Clone(1).(*InternalNode)
		_ = node.Hash()
	}
}
