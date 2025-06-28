package tree

import (
	"bytes"
	"testing"

	"github.com/neutral/proofbox/pkg/crypto"
	"github.com/neutral/proofbox/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLeafNode(t *testing.T) {
	key := types.KeyHash([]byte("test-key"))
	value := []byte("test-value")
	version := types.Version(1)

	// Valid leaf node
	leaf, err := NewLeafNode(key, value, version)
	require.NoError(t, err)
	assert.Equal(t, key, leaf.key)
	assert.Equal(t, value, leaf.value)
	assert.Equal(t, version, leaf.version)
	assert.Equal(t, crypto.DefaultHasher.Hash(value), leaf.valueHash)

	// Empty key
	_, err = NewLeafNode(types.Key{}, value, version)
	assert.ErrorIs(t, err, types.ErrEmptyKey)

	// Value too large
	largeValue := make([]byte, types.MaxValueSize+1)
	_, err = NewLeafNode(key, largeValue, version)
	assert.ErrorIs(t, err, types.ErrValueTooLarge)

	// Empty value is allowed
	emptyLeaf, err := NewLeafNode(key, []byte{}, version)
	require.NoError(t, err)
	assert.Equal(t, []byte{}, emptyLeaf.value)
}

func TestLeafNodeHash(t *testing.T) {
	key := types.KeyHash([]byte("test-key"))
	value := []byte("test-value")

	leaf, err := NewLeafNode(key, value, 1)
	require.NoError(t, err)

	// Hash should be deterministic
	hash1 := leaf.Hash()
	hash2 := leaf.Hash()
	assert.Equal(t, hash1, hash2)

	// Hash should be cached
	assert.True(t, leaf.IsCached())

	// Hash format: 0x01 || key || valueHash
	valueHash := crypto.DefaultHasher.Hash(value)
	expected := crypto.DefaultHasher.HashConcat(
		[]byte{0x01},
		key[:],
		valueHash[:],
	)
	assert.Equal(t, expected, hash1)
}

func TestLeafNodeValueValidation(t *testing.T) {
	key := types.KeyHash([]byte("test"))

	// Test value too large
	largeValue := make([]byte, types.MaxValueSize+1)
	_, err := NewLeafNode(key, largeValue, 1)
	assert.ErrorIs(t, err, types.ErrValueTooLarge)

	// Test empty key
	_, err = NewLeafNode(types.Key{}, []byte("value"), 1)
	assert.ErrorIs(t, err, types.ErrEmptyKey)

	// Test max size value (should succeed)
	maxValue := make([]byte, types.MaxValueSize)
	leaf, err := NewLeafNode(key, maxValue, 1)
	assert.NoError(t, err)
	assert.NotNil(t, leaf)
}

func TestLeafNodeSetValue(t *testing.T) {
	key := types.KeyHash([]byte("test"))
	originalValue := []byte("original")

	// Create leaf without value (simulating loading from storage)
	leaf := &LeafNode{
		key:       key,
		valueHash: crypto.DefaultHasher.Hash(originalValue),
		value:     nil,
		version:   1,
	}

	// Set correct value
	err := leaf.SetValue(originalValue)
	assert.NoError(t, err)
	assert.Equal(t, originalValue, leaf.value)

	// Try to set wrong value
	wrongValue := []byte("wrong")
	err = leaf.SetValue(wrongValue)
	assert.ErrorIs(t, err, types.ErrHashMismatch)
	// Value should not have changed
	assert.Equal(t, originalValue, leaf.value)
}

func TestLeafNodeAccessors(t *testing.T) {
	key := types.KeyHash([]byte("test-key"))
	value := []byte("test-value")
	version := types.Version(42)

	leaf, err := NewLeafNode(key, value, version)
	require.NoError(t, err)

	// Test all accessors
	assert.Equal(t, NodeTypeLeaf, leaf.Type())
	assert.Equal(t, key, leaf.Key())
	assert.Equal(t, value, leaf.Value())
	assert.Equal(t, version, leaf.Version())
	assert.Equal(t, crypto.DefaultHasher.Hash(value), leaf.ValueHash())
}

func TestLeafNodeHashDifferentValues(t *testing.T) {
	key := types.KeyHash([]byte("same-key"))

	// Create two leaves with same key but different values
	leaf1, err := NewLeafNode(key, []byte("value1"), 1)
	require.NoError(t, err)
	leaf2, err := NewLeafNode(key, []byte("value2"), 1)
	require.NoError(t, err)

	// Hashes should be different
	hash1 := leaf1.Hash()
	hash2 := leaf2.Hash()
	assert.False(t, bytes.Equal(hash1[:], hash2[:]))
}

func TestLeafNodeHashDifferentKeys(t *testing.T) {
	value := []byte("same-value")

	// Create two leaves with same value but different keys
	leaf1, err := NewLeafNode(types.KeyHash([]byte("key1")), value, 1)
	require.NoError(t, err)
	leaf2, err := NewLeafNode(types.KeyHash([]byte("key2")), value, 1)
	require.NoError(t, err)

	// Hashes should be different
	hash1 := leaf1.Hash()
	hash2 := leaf2.Hash()
	assert.False(t, bytes.Equal(hash1[:], hash2[:]))
}

func BenchmarkLeafNodeHash(b *testing.B) {
	key := types.KeyHash([]byte("benchmark-key"))
	value := []byte("benchmark-value")
	leaf, _ := NewLeafNode(key, value, 1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = leaf.Hash()
	}
}

func BenchmarkLeafNodeHashUncached(b *testing.B) {
	key := types.KeyHash([]byte("benchmark-key"))
	value := []byte("benchmark-value")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		leaf, _ := NewLeafNode(key, value, 1)
		_ = leaf.Hash()
	}
}
