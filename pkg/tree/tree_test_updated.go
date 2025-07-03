package tree

import (
	"testing"

	"github.com/neutral/proofbox/pkg/storage"
	"github.com/neutral/proofbox/pkg/storage/memory"
	"github.com/neutral/proofbox/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBasicPutWithNewStorage(t *testing.T) {
	// Create memory storage
	store := memory.NewStorage()
	defer store.Close()

	// Create key encoder
	keyEncoder := storage.NewDefaultKeyEncoder()

	// Create tree
	tree, err := NewTree(store, keyEncoder, DefaultTreeConfig())
	require.NoError(t, err, "Failed to create tree")

	// Test Put operation
	key := types.KeyHash([]byte("test-key"))
	value := []byte("test-value")

	version, err := tree.Put(key, value)
	require.NoError(t, err, "Failed to put key-value")
	assert.Equal(t, types.Version(1), version, "Expected version 1")

	// Test Get operation
	retrieved, err := tree.Get(version, key)
	require.NoError(t, err, "Failed to get value")
	assert.Equal(t, value, retrieved, "Retrieved value should match")

	// Test non-existent key
	nonExistentKey := types.KeyHash([]byte("non-existent"))
	val, err := tree.Get(version, nonExistentKey)
	require.NoError(t, err, "Get should not error for non-existent key")
	assert.Nil(t, val, "Value should be nil for non-existent key")
}

func TestVersioningWithNewStorage(t *testing.T) {
	// Create memory storage
	store := memory.NewStorage()
	defer store.Close()

	// Create tree with storage abstraction
	tree, err := NewTree(store, storage.NewDefaultKeyEncoder(), DefaultTreeConfig())
	require.NoError(t, err)

	key := types.KeyHash([]byte("versioned-key"))

	// Version 1: Initial value
	v1, err := tree.Put(key, []byte("value-v1"))
	require.NoError(t, err)
	assert.Equal(t, types.Version(1), v1)

	// Version 2: Updated value
	v2, err := tree.Put(key, []byte("value-v2"))
	require.NoError(t, err)
	assert.Equal(t, types.Version(2), v2)

	// Version 3: Another update
	v3, err := tree.Put(key, []byte("value-v3"))
	require.NoError(t, err)
	assert.Equal(t, types.Version(3), v3)

	// Verify all versions
	val1, err := tree.Get(v1, key)
	require.NoError(t, err)
	assert.Equal(t, []byte("value-v1"), val1)

	val2, err := tree.Get(v2, key)
	require.NoError(t, err)
	assert.Equal(t, []byte("value-v2"), val2)

	val3, err := tree.Get(v3, key)
	require.NoError(t, err)
	assert.Equal(t, []byte("value-v3"), val3)
}
