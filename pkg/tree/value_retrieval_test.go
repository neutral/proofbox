package tree

import (
	"testing"

	"github.com/neutral/proofbox/pkg/crypto"
	"github.com/neutral/proofbox/pkg/storage"
	"github.com/neutral/proofbox/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValueRetrieval reproduces the value retrieval bug where Get returns empty values
func TestValueRetrieval(t *testing.T) {
	store := createTestStorage(t)
	keyEncoder := storage.NewDefaultKeyEncoder()
	tr, err := NewTree(store, keyEncoder, testTreeConfig())
	require.NoError(t, err)

	// Test data
	key := types.KeyHash([]byte("test-key"))
	value := []byte("test value")

	// Put the value
	version, err := tr.Put(key, value)
	require.NoError(t, err)
	assert.Equal(t, types.Version(1), version)

	// Get the value directly from tree
	retrieved, err := tr.Get(version, key)
	require.NoError(t, err)

	// This should pass but currently fails - values come back empty
	assert.NotNil(t, retrieved, "retrieved value should not be nil")
	assert.NotEmpty(t, retrieved, "retrieved value should not be empty")
	assert.Equal(t, value, retrieved, "retrieved value should match original")
}

// TestBatchValueRetrieval tests value retrieval after batch operations
func TestBatchValueRetrieval(t *testing.T) {
	store := createTestStorage(t)
	keyEncoder := storage.NewDefaultKeyEncoder()
	tr, err := NewTree(store, keyEncoder, testTreeConfig())
	require.NoError(t, err)

	// Create batch
	batch := tr.NewBatchTransaction()

	// Add multiple values
	testData := map[string][]byte{
		"key1": []byte("value1"),
		"key2": []byte("value2"),
		"key3": []byte("value3"),
	}

	for k, v := range testData {
		key := types.KeyHash([]byte(k))
		err := batch.BatchPut(key, v)
		require.NoError(t, err)
	}

	// Execute batch
	version, err := batch.Execute()
	require.NoError(t, err)

	// Verify all values can be retrieved
	for k, expectedValue := range testData {
		key := types.KeyHash([]byte(k))
		retrieved, err := tr.Get(version, key)
		require.NoError(t, err)

		assert.NotNil(t, retrieved, "retrieved value for %s should not be nil", k)
		assert.NotEmpty(t, retrieved, "retrieved value for %s should not be empty", k)
		assert.Equal(t, expectedValue, retrieved, "retrieved value for %s should match original", k)
	}
}

// TestDirectStorageAccess verifies values are actually in storage and can be loaded
func TestDirectStorageAccess(t *testing.T) {
	store := createTestStorage(t)
	keyEncoder := storage.NewDefaultKeyEncoder()
	tr, err := NewTree(store, keyEncoder, testTreeConfig())
	require.NoError(t, err)

	// Test multiple keys to ensure we're testing real storage
	testData := map[string][]byte{
		"key1": []byte("value1"),
		"key2": []byte("value2"),
		"key3": []byte("value3"),
	}

	// Put all values
	var version types.Version
	for k, v := range testData {
		key := types.KeyHash([]byte(k))
		version, err = tr.Put(key, v)
		require.NoError(t, err)
	}

	// Create reader
	reader, err := tr.Reader(version)
	require.NoError(t, err)
	defer reader.Close()

	// Verify all values through the reader interface
	for k, expectedValue := range testData {
		key := types.KeyHash([]byte(k))

		// First verify through tree.Get
		retrieved, err := tr.Get(version, key)
		require.NoError(t, err)
		assert.Equal(t, expectedValue, retrieved)

		// Also verify we can compute the value hash and load it
		valueHash := crypto.DefaultHasher.Hash(expectedValue)
		storedValue, err := reader.LoadValue(valueHash)
		require.NoError(t, err)
		assert.Equal(t, expectedValue, storedValue)
	}
}
