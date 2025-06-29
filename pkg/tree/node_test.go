package tree

import (
	"testing"

	"github.com/neutral/proofbox/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestNodeTypeHelpers(t *testing.T) {
	// Create test nodes
	leafNode := &LeafNode{
		key:     types.KeyHash([]byte("test")),
		version: 1,
	}
	internalNode := &InternalNode{
		children: make(map[types.Nibble]types.Child),
		version:  1,
	}

	// Test IsLeaf
	assert.True(t, types.IsLeaf(leafNode))
	assert.False(t, types.IsLeaf(internalNode))

	// Test IsInternal
	assert.False(t, types.IsInternal(leafNode))
	assert.True(t, types.IsInternal(internalNode))

	// Test AsLeaf
	leaf, ok := AsLeaf(leafNode)
	assert.True(t, ok)
	assert.NotNil(t, leaf)
	assert.Equal(t, leafNode, leaf)

	_, ok = AsLeaf(internalNode)
	assert.False(t, ok)

	// Test AsInternal
	internal, ok := AsInternal(internalNode)
	assert.True(t, ok)
	assert.NotNil(t, internal)
	assert.Equal(t, internalNode, internal)

	_, ok = AsInternal(leafNode)
	assert.False(t, ok)
}

func TestChildIsEmpty(t *testing.T) {
	// Empty child
	emptyChild := types.Child{}
	assert.True(t, emptyChild.IsEmpty())

	// Non-empty child
	nonEmptyChild := types.Child{
		Hash:    types.Hash(types.KeyHash([]byte("test"))), // Convert Key to Hash
		Version: 1,
		IsLeaf:  true,
	}
	assert.False(t, nonEmptyChild.IsEmpty())
}

func TestNodeInterface(t *testing.T) {
	// Test that both node types implement the Node interface
	var _ types.Node = (*LeafNode)(nil)
	var _ types.Node = (*InternalNode)(nil)

	// Create test nodes
	leafNode, err := NewLeafNode(types.KeyHash([]byte("test")), []byte("value"), 1)
	assert.NoError(t, err)
	internalNode := NewInternalNode(2)

	// Test Type()
	assert.Equal(t, types.NodeTypeLeaf, leafNode.Type())
	assert.Equal(t, types.NodeTypeInternal, internalNode.Type())

	// Test Version()
	assert.Equal(t, types.Version(1), leafNode.Version())
	assert.Equal(t, types.Version(2), internalNode.Version())

	// Test IsCached() - should be false before hashing
	assert.False(t, leafNode.IsCached())
	assert.False(t, internalNode.IsCached())

	// Test Hash() - should compute hash
	leafHash := leafNode.Hash()
	assert.NotEqual(t, types.EmptyHash(), leafHash)
	internalHash := internalNode.Hash()
	assert.NotEqual(t, types.EmptyHash(), internalHash)

	// Test IsCached() - should be true after hashing
	assert.True(t, leafNode.IsCached())
	assert.True(t, internalNode.IsCached())

	// Hash should be deterministic
	assert.Equal(t, leafHash, leafNode.Hash())
	assert.Equal(t, internalHash, internalNode.Hash())
}
