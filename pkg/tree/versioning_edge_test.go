package tree

import (
	"fmt"
	"testing"

	"github.com/neutral/proofbox/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestVersionValidation tests error cases for version operations
func TestVersionValidation(t *testing.T) {
	db := createTestDB(t)
	tree, err := NewTree(db, DefaultTreeConfig())
	require.NoError(t, err)

	// Test 1: Operations on non-existent version
	t.Run("NonExistentVersion", func(t *testing.T) {
		err := tree.PutVersioned(999, types.KeyHash([]byte("key")), []byte("value"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not pending")

		err = tree.DeleteVersioned(999, types.KeyHash([]byte("key")))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not pending")

		err = tree.CommitVersion(999)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not pending")
	})

	// Test 2: Operations on committed version
	t.Run("CommittedVersion", func(t *testing.T) {
		v, err := tree.BeginVersion()
		require.NoError(t, err)
		
		err = tree.PutVersioned(v, types.KeyHash([]byte("key")), []byte("value"))
		require.NoError(t, err)
		
		err = tree.CommitVersion(v)
		require.NoError(t, err)

		// Try to modify committed version
		err = tree.PutVersioned(v, types.KeyHash([]byte("key2")), []byte("value2"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not pending")

		// Try to commit again
		err = tree.CommitVersion(v)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not pending")
	})

	// Test 3: Operations on aborted version
	t.Run("AbortedVersion", func(t *testing.T) {
		v, err := tree.BeginVersion()
		require.NoError(t, err)
		
		err = tree.AbortVersion(v)
		require.NoError(t, err)

		// Try to use aborted version
		err = tree.PutVersioned(v, types.KeyHash([]byte("key")), []byte("value"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not pending")

		// Try to commit aborted version
		err = tree.CommitVersion(v)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not pending")

		// Try to abort again
		err = tree.AbortVersion(v)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no pending version")
	})

	// Test 4: Invalid key/value validation
	t.Run("InvalidInputs", func(t *testing.T) {
		v, err := tree.BeginVersion()
		require.NoError(t, err)
		defer tree.AbortVersion(v)

		// Empty key
		err = tree.PutVersioned(v, types.Key{}, []byte("value"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid key")

		// Empty value
		err = tree.PutVersioned(v, types.KeyHash([]byte("key")), []byte{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid value")

		// Oversized value
		largeValue := make([]byte, types.MaxValueSize+1)
		err = tree.PutVersioned(v, types.KeyHash([]byte("key")), largeValue)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid value")
	})

	// Test 5: Version overflow protection
	t.Run("VersionOverflow", func(t *testing.T) {
		// This is hard to test without mocking, but we can test the validation
		err := tree.validateVersion(types.MaxVersion + 1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "version exceeds maximum")
	})
}

// TestBatchOperationsInVersion tests multiple operations within a single version
func TestBatchOperationsInVersion(t *testing.T) {
	db := createTestDB(t)
	tree, err := NewTree(db, DefaultTreeConfig())
	require.NoError(t, err)

	t.Run("MultiplePutsInVersion", func(t *testing.T) {
		v, err := tree.BeginVersion()
		require.NoError(t, err)

		// Insert multiple keys
		numKeys := 100
		for i := 0; i < numKeys; i++ {
			key := types.KeyHash([]byte(fmt.Sprintf("batch-key-%d", i)))
			value := []byte(fmt.Sprintf("batch-value-%d", i))
			err = tree.PutVersioned(v, key, value)
			require.NoError(t, err)
		}

		// Commit the batch
		err = tree.CommitVersion(v)
		require.NoError(t, err)

		// Verify all keys are accessible
		for i := 0; i < numKeys; i++ {
			key := types.KeyHash([]byte(fmt.Sprintf("batch-key-%d", i)))
			got, err := tree.GetAtVersion(v, key)
			require.NoError(t, err)
			expected := []byte(fmt.Sprintf("batch-value-%d", i))
			assert.Equal(t, expected, got)
		}
	})

	t.Run("MixedOperationsInVersion", func(t *testing.T) {
		// Setup: Create initial data
		setupVersion, err := tree.BeginVersion()
		require.NoError(t, err)

		for i := 0; i < 10; i++ {
			key := types.KeyHash([]byte(fmt.Sprintf("mixed-key-%d", i)))
			value := []byte(fmt.Sprintf("initial-value-%d", i))
			err = tree.PutVersioned(setupVersion, key, value)
			require.NoError(t, err)
		}
		err = tree.CommitVersion(setupVersion)
		require.NoError(t, err)

		// Test: Mixed puts, updates, and deletes
		v, err := tree.BeginVersion()
		require.NoError(t, err)

		// Update some keys
		for i := 0; i < 5; i++ {
			key := types.KeyHash([]byte(fmt.Sprintf("mixed-key-%d", i)))
			value := []byte(fmt.Sprintf("updated-value-%d", i))
			err = tree.PutVersioned(v, key, value)
			require.NoError(t, err)
		}

		// Delete some keys
		for i := 5; i < 8; i++ {
			key := types.KeyHash([]byte(fmt.Sprintf("mixed-key-%d", i)))
			err = tree.DeleteVersioned(v, key)
			require.NoError(t, err)
		}

		// Add new keys
		for i := 10; i < 15; i++ {
			key := types.KeyHash([]byte(fmt.Sprintf("mixed-key-%d", i)))
			value := []byte(fmt.Sprintf("new-value-%d", i))
			err = tree.PutVersioned(v, key, value)
			require.NoError(t, err)
		}

		// Commit all changes
		err = tree.CommitVersion(v)
		require.NoError(t, err)

		// Verify the state
		// Updated keys
		for i := 0; i < 5; i++ {
			key := types.KeyHash([]byte(fmt.Sprintf("mixed-key-%d", i)))
			got, err := tree.GetAtVersion(v, key)
			require.NoError(t, err)
			expected := []byte(fmt.Sprintf("updated-value-%d", i))
			assert.Equal(t, expected, got)
		}

		// Deleted keys
		for i := 5; i < 8; i++ {
			key := types.KeyHash([]byte(fmt.Sprintf("mixed-key-%d", i)))
			got, err := tree.GetAtVersion(v, key)
			require.NoError(t, err)
			assert.Nil(t, got, "Deleted key should return nil")
		}

		// Unchanged keys
		for i := 8; i < 10; i++ {
			key := types.KeyHash([]byte(fmt.Sprintf("mixed-key-%d", i)))
			got, err := tree.GetAtVersion(v, key)
			require.NoError(t, err)
			expected := []byte(fmt.Sprintf("initial-value-%d", i))
			assert.Equal(t, expected, got)
		}

		// New keys
		for i := 10; i < 15; i++ {
			key := types.KeyHash([]byte(fmt.Sprintf("mixed-key-%d", i)))
			got, err := tree.GetAtVersion(v, key)
			require.NoError(t, err)
			expected := []byte(fmt.Sprintf("new-value-%d", i))
			assert.Equal(t, expected, got)
		}
	})

	t.Run("AbortRollback", func(t *testing.T) {
		// Create initial state
		initialKey := types.KeyHash([]byte("rollback-key"))
		initialValue := []byte("initial-value")
		v1, err := tree.Put(initialKey, initialValue)
		require.NoError(t, err)

		// Start a version with changes
		v2, err := tree.BeginVersion()
		require.NoError(t, err)

		// Make changes
		err = tree.PutVersioned(v2, initialKey, []byte("modified-value"))
		require.NoError(t, err)
		
		newKey := types.KeyHash([]byte("new-key"))
		err = tree.PutVersioned(v2, newKey, []byte("new-value"))
		require.NoError(t, err)

		// Abort the version
		err = tree.AbortVersion(v2)
		require.NoError(t, err)

		// Verify original state is preserved
		got, err := tree.GetAtVersion(v1, initialKey)
		require.NoError(t, err)
		assert.Equal(t, initialValue, got)

		// New key should not exist in v1
		got, err = tree.GetAtVersion(v1, newKey)
		require.NoError(t, err)
		assert.Nil(t, got)

		// Aborted version should not be readable
		_, err = tree.GetAtVersion(v2, initialKey)
		assert.Error(t, err)
	})
}

// TestEmptyTreeVersioning tests versioning with empty trees
func TestEmptyTreeVersioning(t *testing.T) {
	db := createTestDB(t)
	tree, err := NewTree(db, DefaultTreeConfig())
	require.NoError(t, err)

	t.Run("VersioningEmptyTree", func(t *testing.T) {
		// Version 0 should exist and be empty
		isEmpty, err := tree.IsEmpty(0)
		require.NoError(t, err)
		assert.True(t, isEmpty)

		// Create a new empty version
		v1, err := tree.BeginVersion()
		require.NoError(t, err)
		err = tree.CommitVersion(v1)
		require.NoError(t, err)

		// Should still be empty
		isEmpty, err = tree.IsEmpty(v1)
		require.NoError(t, err)
		assert.True(t, isEmpty)

		// Root hashes should be equal
		root0, err := tree.GetRootHash(0)
		require.NoError(t, err)
		root1, err := tree.GetRootHash(v1)
		require.NoError(t, err)
		assert.Equal(t, root0, root1)
		assert.Equal(t, types.EmptyHash(), root1)
	})

	t.Run("SingleNodeTree", func(t *testing.T) {
		// Add single key
		key := types.KeyHash([]byte("single-key"))
		value := []byte("single-value")
		v1, err := tree.Put(key, value)
		require.NoError(t, err)

		// Tree should not be empty
		isEmpty, err := tree.IsEmpty(v1)
		require.NoError(t, err)
		assert.False(t, isEmpty)

		// Delete the only key
		v2, err := tree.Delete(key)
		require.NoError(t, err)

		// Tree should be empty again
		isEmpty, err = tree.IsEmpty(v2)
		require.NoError(t, err)
		assert.True(t, isEmpty)
	})
}

// TestDeepTreeVersioning tests versioning with very deep trees
func TestDeepTreeVersioning(t *testing.T) {
	db := createTestDB(t)
	tree, err := NewTree(db, DefaultTreeConfig())
	require.NoError(t, err)

	t.Run("MaxDepthTree", func(t *testing.T) {
		// Create keys that will create a deep path
		// Use keys with common prefix but different at the end
		key1 := types.Key{}
		key2 := types.Key{}
		
		// Set up keys that differ only in the last byte
		// Fill with a pattern to ensure non-zero key
		for i := 0; i < 31; i++ {
			key1[i] = 0x11
			key2[i] = 0x11
		}
		key1[31] = 0x00
		key2[31] = 0x01

		// Insert keys
		v1, err := tree.BeginVersion()
		require.NoError(t, err)
		
		err = tree.PutVersioned(v1, key1, []byte("deep-value-1"))
		require.NoError(t, err)
		
		err = tree.PutVersioned(v1, key2, []byte("deep-value-2"))
		require.NoError(t, err)
		
		err = tree.CommitVersion(v1)
		require.NoError(t, err)

		// Verify both keys are accessible
		got1, err := tree.GetAtVersion(v1, key1)
		require.NoError(t, err)
		assert.Equal(t, []byte("deep-value-1"), got1)

		got2, err := tree.GetAtVersion(v1, key2)
		require.NoError(t, err)
		assert.Equal(t, []byte("deep-value-2"), got2)

		// Update one key in new version
		v2, err := tree.BeginVersion()
		require.NoError(t, err)
		
		err = tree.PutVersioned(v2, key1, []byte("deep-value-1-updated"))
		require.NoError(t, err)
		
		err = tree.CommitVersion(v2)
		require.NoError(t, err)

		// Verify structural sharing works at max depth
		got1v2, err := tree.GetAtVersion(v2, key1)
		require.NoError(t, err)
		assert.Equal(t, []byte("deep-value-1-updated"), got1v2)

		got2v2, err := tree.GetAtVersion(v2, key2)
		require.NoError(t, err)
		assert.Equal(t, []byte("deep-value-2"), got2v2) // Unchanged
	})
}

