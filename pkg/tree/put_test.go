package tree

import (
	"bytes"
	"sync"
	"testing"

	"github.com/neutral/proofbox/pkg/storage"
	"github.com/neutral/proofbox/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPut_InsertIntoEmptyTree(t *testing.T) {
	store := createTestStorage(t)
	keyEncoder := storage.NewDefaultKeyEncoder()

	tree, err := NewTree(store, keyEncoder, DefaultTreeConfig())
	require.NoError(t, err)

	// Insert into empty tree
	key := types.KeyHash([]byte("test-key"))
	value := []byte("test-value")

	version, err := tree.Put(key, value)
	require.NoError(t, err)

	// Verify version incremented
	assert.Equal(t, types.Version(1), version)
	assert.Equal(t, types.Version(1), tree.GetLatestVersion())

	// Verify we can retrieve the value
	retrieved, err := tree.Get(version, key)
	require.NoError(t, err)
	assert.Equal(t, value, retrieved)

	// Verify root hash changed
	rootHash, err := tree.GetRootHash(version)
	require.NoError(t, err)
	assert.NotEqual(t, types.EmptyHash(), rootHash)

	// Verify tree is no longer empty
	isEmpty, err := tree.IsEmpty(version)
	require.NoError(t, err)
	assert.False(t, isEmpty)
}

func TestPut_UpdateExistingKey(t *testing.T) {
	store := createTestStorage(t)
	keyEncoder := storage.NewDefaultKeyEncoder()

	tree, err := NewTree(store, keyEncoder, DefaultTreeConfig())
	require.NoError(t, err)

	key := types.KeyHash([]byte("update-key"))

	// Insert initial value
	value1 := []byte("value1")
	v1, err := tree.Put(key, value1)
	require.NoError(t, err)
	assert.Equal(t, types.Version(1), v1)

	// Update with new value
	value2 := []byte("value2")
	v2, err := tree.Put(key, value2)
	require.NoError(t, err)
	assert.Equal(t, types.Version(2), v2)

	// Check both versions
	retrieved1, err := tree.Get(v1, key)
	require.NoError(t, err)
	assert.Equal(t, value1, retrieved1)

	retrieved2, err := tree.Get(v2, key)
	require.NoError(t, err)
	assert.Equal(t, value2, retrieved2)

	// Verify different root hashes
	hash1, err := tree.GetRootHash(v1)
	require.NoError(t, err)
	hash2, err := tree.GetRootHash(v2)
	require.NoError(t, err)
	assert.NotEqual(t, hash1, hash2)
}

func TestPut_Persistence(t *testing.T) {
	store := createTestStorage(t)
	keyEncoder := storage.NewDefaultKeyEncoder()

	key := types.KeyHash([]byte("persistent-key"))
	value := []byte("persistent-value")
	var version types.Version

	// Insert with first tree instance
	{
		tree, err := NewTree(store, keyEncoder, DefaultTreeConfig())
		require.NoError(t, err)

		v, err := tree.Put(key, value)
		require.NoError(t, err)
		version = v
	}

	// Create new tree instance
	tree2, err := NewTree(store, keyEncoder, DefaultTreeConfig())
	require.NoError(t, err)

	// Should load latest version
	assert.Equal(t, version, tree2.GetLatestVersion())

	// Should be able to read the value
	retrieved, err := tree2.Get(version, key)
	require.NoError(t, err)
	assert.Equal(t, value, retrieved)

	// Should have correct root hash
	rootHash, err := tree2.GetRootHash(version)
	require.NoError(t, err)
	assert.NotEqual(t, types.EmptyHash(), rootHash)
}

func TestPut_ValueSizeLimit(t *testing.T) {
	store := createTestStorage(t)
	keyEncoder := storage.NewDefaultKeyEncoder()

	tree, err := NewTree(store, keyEncoder, DefaultTreeConfig())
	require.NoError(t, err)

	key := types.KeyHash([]byte("large-key"))

	// Try to insert value that's too large
	largeValue := make([]byte, types.MaxValueSize+1)
	_, err = tree.Put(key, largeValue)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "value too large")

	// Verify tree state unchanged
	assert.Equal(t, types.Version(0), tree.GetLatestVersion())

	// Should be able to insert max size value
	maxValue := make([]byte, types.MaxValueSize)
	version, err := tree.Put(key, maxValue)
	require.NoError(t, err)
	assert.Equal(t, types.Version(1), version)
}

func TestPut_InvalidInputs(t *testing.T) {
	store := createTestStorage(t)
	keyEncoder := storage.NewDefaultKeyEncoder()

	tree, err := NewTree(store, keyEncoder, DefaultTreeConfig())
	require.NoError(t, err)

	tests := []struct {
		name   string
		key    types.Key
		value  []byte
		errMsg string
	}{
		{
			name:   "empty key",
			key:    types.Key{},
			value:  []byte("value"),
			errMsg: "invalid key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tree.Put(tt.key, tt.value)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.errMsg)
		})
	}
}

func TestPut_ConcurrentWrites(t *testing.T) {
	store := createTestStorage(t)
	keyEncoder := storage.NewDefaultKeyEncoder()

	tree, err := NewTree(store, keyEncoder, DefaultTreeConfig())
	require.NoError(t, err)

	// Run concurrent puts
	numGoroutines := 10
	var wg sync.WaitGroup
	versions := make([]types.Version, numGoroutines)
	errors := make([]error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			key := types.KeyHash([]byte("concurrent-key"))
			value := []byte(string(rune('a' + idx)))
			versions[idx], errors[idx] = tree.Put(key, value)
		}(i)
	}

	wg.Wait()

	// All operations should succeed
	for i, err := range errors {
		require.NoError(t, err, "goroutine %d failed", i)
	}

	// Versions should be sequential (due to write serialization)
	versionSet := make(map[types.Version]bool)
	for _, v := range versions {
		versionSet[v] = true
	}
	assert.Len(t, versionSet, numGoroutines)

	// Latest version should be numGoroutines
	assert.Equal(t, types.Version(numGoroutines), tree.GetLatestVersion())
}

func TestPut_DifferentKeySuccess(t *testing.T) {
	store := createTestStorage(t)
	keyEncoder := storage.NewDefaultKeyEncoder()

	tree, err := NewTree(store, keyEncoder, DefaultTreeConfig())
	require.NoError(t, err)

	// Insert first key
	key1 := types.KeyHash([]byte("key1"))
	value1 := []byte("value1")
	v1, err := tree.Put(key1, value1)
	require.NoError(t, err)
	assert.Equal(t, types.Version(1), v1)

	// Try to insert different key (should now succeed with splitting)
	key2 := types.KeyHash([]byte("key2"))
	value2 := []byte("value2")
	v2, err := tree.Put(key2, value2)
	assert.NoError(t, err)
	assert.Equal(t, types.Version(2), v2)

	// Tree should now be at version 2
	assert.Equal(t, types.Version(2), tree.GetLatestVersion())

	// Original key should still work
	retrieved, err := tree.Get(v1, key1)
	require.NoError(t, err)
	assert.Equal(t, value1, retrieved)
}

func TestPut_MultipleUpdates(t *testing.T) {
	store := createTestStorage(t)
	keyEncoder := storage.NewDefaultKeyEncoder()

	tree, err := NewTree(store, keyEncoder, DefaultTreeConfig())
	require.NoError(t, err)

	key := types.KeyHash([]byte("multi-update-key"))

	// Perform multiple updates
	for i := 1; i <= 5; i++ {
		value := []byte(string(rune('0' + i)))
		version, err := tree.Put(key, value)
		require.NoError(t, err)
		assert.Equal(t, types.Version(i), version)

		// Verify immediate read
		retrieved, err := tree.Get(version, key)
		require.NoError(t, err)
		assert.Equal(t, value, retrieved)
	}

	// Verify all versions still accessible
	for i := 1; i <= 5; i++ {
		value := []byte(string(rune('0' + i)))
		retrieved, err := tree.Get(types.Version(i), key)
		require.NoError(t, err)
		assert.Equal(t, value, retrieved)
	}
}

func TestPut_LargeValues(t *testing.T) {
	store := createTestStorage(t)
	keyEncoder := storage.NewDefaultKeyEncoder()

	tree, err := NewTree(store, keyEncoder, DefaultTreeConfig())
	require.NoError(t, err)

	// Test with various sizes - using same key for all (no leaf splitting in step 09)
	sizes := []int{
		1,           // 1 byte
		1024,        // 1 KB
		10 * 1024,   // 10 KB
		100 * 1024,  // 100 KB
		1024 * 1024, // 1 MB (max size)
	}

	key := types.KeyHash([]byte("large-value-key"))

	for i, size := range sizes {
		value := bytes.Repeat([]byte{'x'}, size)

		version, err := tree.Put(key, value)
		require.NoError(t, err)
		assert.Equal(t, types.Version(i+1), version)

		// Verify retrieval
		retrieved, err := tree.Get(version, key)
		require.NoError(t, err)
		assert.Len(t, retrieved, size)
		assert.Equal(t, value, retrieved)
	}
}
