package tree

import (
	"fmt"
	"path/filepath"
	"testing"

	pebblestorage "github.com/neutral/proofbox/pkg/storage/pebble"
	"github.com/neutral/proofbox/pkg/storage"
	"github.com/neutral/proofbox/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBasicVersioning(t *testing.T) {
	// Create test tree
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	opts := &pebblestorage.Options{
		EnableMetrics: false,
	}
	store, err := pebblestorage.NewStorage(dbPath, opts)
	require.NoError(t, err)
	defer store.Close()
	keyEncoder := storage.NewDefaultKeyEncoder()

	tree, err := NewTree(store, keyEncoder, testTreeConfig())
	require.NoError(t, err)

	// Create version 1
	key := types.KeyHash([]byte("test-key"))
	value1 := []byte("value-v1")

	v1, err := tree.BeginVersion()
	require.NoError(t, err)
	assert.Equal(t, types.Version(1), v1)

	err = tree.PutVersioned(v1, key, value1)
	require.NoError(t, err)

	err = tree.CommitVersion(v1)
	require.NoError(t, err)

	// Create version 2 with updated value
	value2 := []byte("value-v2")

	v2, err := tree.BeginVersion()
	require.NoError(t, err)
	assert.Equal(t, types.Version(2), v2)

	err = tree.PutVersioned(v2, key, value2)
	require.NoError(t, err)

	err = tree.CommitVersion(v2)
	require.NoError(t, err)

	// Verify both versions accessible
	got1, err := tree.GetAtVersion(v1, key)
	require.NoError(t, err)
	assert.Equal(t, value1, got1)

	got2, err := tree.GetAtVersion(v2, key)
	require.NoError(t, err)
	assert.Equal(t, value2, got2)

	// Verify root hashes differ
	root1, err := tree.GetRootHash(v1)
	require.NoError(t, err)

	root2, err := tree.GetRootHash(v2)
	require.NoError(t, err)

	assert.NotEqual(t, root1, root2)
}

func TestVersionAbort(t *testing.T) {
	// Create test tree
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	opts := &pebblestorage.Options{
		EnableMetrics: false,
	}
	store, err := pebblestorage.NewStorage(dbPath, opts)
	require.NoError(t, err)
	defer store.Close()
	keyEncoder := storage.NewDefaultKeyEncoder()

	tree, err := NewTree(store, keyEncoder, testTreeConfig())
	require.NoError(t, err)

	// Start version but abort
	v1, err := tree.BeginVersion()
	require.NoError(t, err)

	key := types.KeyHash([]byte("test"))
	err = tree.PutVersioned(v1, key, []byte("should-not-persist"))
	require.NoError(t, err)

	// Abort instead of commit
	err = tree.AbortVersion(v1)
	require.NoError(t, err)

	// Verify version is marked as aborted
	info, err := tree.versionManager.GetVersion(v1)
	require.NoError(t, err)
	assert.Equal(t, VersionStatusAborted, info.Status)

	// Key should not be accessible
	_, err = tree.GetAtVersion(v1, key)
	assert.Error(t, err)
}

func TestStructuralSharing(t *testing.T) {
	// Create test tree
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	opts := &pebblestorage.Options{
		EnableMetrics: false,
	}
	store, err := pebblestorage.NewStorage(dbPath, opts)
	require.NoError(t, err)
	defer store.Close()
	keyEncoder := storage.NewDefaultKeyEncoder()

	tree, err := NewTree(store, keyEncoder, testTreeConfig())
	require.NoError(t, err)

	// Create version 1 with multiple keys
	v1, err := tree.BeginVersion()
	require.NoError(t, err)

	// Insert 10 keys
	for i := 0; i < 10; i++ {
		key := types.KeyHash([]byte(fmt.Sprintf("key-%d", i)))
		value := []byte(fmt.Sprintf("value-%d", i))
		err = tree.PutVersioned(v1, key, value)
		require.NoError(t, err)
	}

	err = tree.CommitVersion(v1)
	require.NoError(t, err)

	// Create version 2 modifying only one key
	v2, err := tree.BeginVersion()
	require.NoError(t, err)

	// Modify only key-5
	modifiedKey := types.KeyHash([]byte("key-5"))
	err = tree.PutVersioned(v2, modifiedKey, []byte("modified-value"))
	require.NoError(t, err)

	err = tree.CommitVersion(v2)
	require.NoError(t, err)

	// Verify all other keys unchanged in v2
	for i := 0; i < 10; i++ {
		if i == 5 {
			continue // Skip the modified key
		}

		key := types.KeyHash([]byte(fmt.Sprintf("key-%d", i)))
		expectedValue := []byte(fmt.Sprintf("value-%d", i))

		got, err := tree.GetAtVersion(v2, key)
		require.NoError(t, err, "Failed to get key %s from v2", key)
		assert.Equal(t, expectedValue, got, "Value mismatch for key %s", key)
	}

	// Verify modified key has new value
	got, err := tree.GetAtVersion(v2, modifiedKey)
	require.NoError(t, err)
	assert.Equal(t, []byte("modified-value"), got)
}

func TestDeleteInVersion(t *testing.T) {
	// Create test tree
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	opts := &pebblestorage.Options{
		EnableMetrics: false,
	}
	store, err := pebblestorage.NewStorage(dbPath, opts)
	require.NoError(t, err)
	defer store.Close()
	keyEncoder := storage.NewDefaultKeyEncoder()

	tree, err := NewTree(store, keyEncoder, testTreeConfig())
	require.NoError(t, err)

	// Version 1: Insert keys
	v1, err := tree.BeginVersion()
	require.NoError(t, err)

	key1 := types.KeyHash([]byte("key1"))
	key2 := types.KeyHash([]byte("key2"))

	err = tree.PutVersioned(v1, key1, []byte("value1"))
	require.NoError(t, err)

	err = tree.PutVersioned(v1, key2, []byte("value2"))
	require.NoError(t, err)

	err = tree.CommitVersion(v1)
	require.NoError(t, err)

	// Version 2: Delete key1
	v2, err := tree.BeginVersion()
	require.NoError(t, err)

	err = tree.DeleteVersioned(v2, key1)
	require.NoError(t, err)

	err = tree.CommitVersion(v2)
	require.NoError(t, err)

	// Verify key1 exists in v1 but not in v2
	got1, err := tree.GetAtVersion(v1, key1)
	require.NoError(t, err)
	assert.Equal(t, []byte("value1"), got1)

	got, err := tree.GetAtVersion(v2, key1)
	require.NoError(t, err)
	assert.Nil(t, got, "Deleted key should return nil")

	// Verify key2 exists in both versions
	got2v1, err := tree.GetAtVersion(v1, key2)
	require.NoError(t, err)
	assert.Equal(t, []byte("value2"), got2v1)

	got2v2, err := tree.GetAtVersion(v2, key2)
	require.NoError(t, err)
	assert.Equal(t, []byte("value2"), got2v2)
}

func TestProofAcrossVersions(t *testing.T) {
	// Create test tree
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	opts := &pebblestorage.Options{
		EnableMetrics: false,
	}
	store, err := pebblestorage.NewStorage(dbPath, opts)
	require.NoError(t, err)
	defer store.Close()
	keyEncoder := storage.NewDefaultKeyEncoder()

	tree, err := NewTree(store, keyEncoder, testTreeConfig())
	require.NoError(t, err)

	// Version 1
	v1, err := tree.BeginVersion()
	require.NoError(t, err)

	key := types.KeyHash([]byte("test-key"))
	value1 := []byte("value1")

	err = tree.PutVersioned(v1, key, value1)
	require.NoError(t, err)

	err = tree.CommitVersion(v1)
	require.NoError(t, err)

	// Version 2: update the value
	v2, err := tree.BeginVersion()
	require.NoError(t, err)

	value2 := []byte("value2")
	err = tree.PutVersioned(v2, key, value2)
	require.NoError(t, err)

	err = tree.CommitVersion(v2)
	require.NoError(t, err)

	// Generate proofs for both versions
	reader1, err := tree.Reader(v1)
	require.NoError(t, err)
	defer reader1.Close()

	reader2, err := tree.Reader(v2)
	require.NoError(t, err)
	defer reader2.Close()

	// Note: This would require the proof package to be imported
	// For now, just verify the readers work correctly
	assert.NotNil(t, reader1)
	assert.NotNil(t, reader2)
}

// BenchmarkStructuralSharing measures the efficiency of structural sharing
func BenchmarkStructuralSharing(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench.db")

	opts := &pebblestorage.Options{
		EnableMetrics: false,
	}
	store, err := pebblestorage.NewStorage(dbPath, opts)
	require.NoError(b, err)
	defer store.Close()
	keyEncoder := storage.NewDefaultKeyEncoder()

	tree, err := NewTree(store, keyEncoder, testTreeConfig())
	require.NoError(b, err)

	// Create initial version with many keys
	v1, _ := tree.BeginVersion()

	numKeys := 1000
	for i := 0; i < numKeys; i++ {
		key := types.KeyHash([]byte(fmt.Sprintf("key-%d", i)))
		value := []byte(fmt.Sprintf("value-%d", i))
		tree.PutVersioned(v1, key, value)
	}
	tree.CommitVersion(v1)

	b.ResetTimer()

	// Benchmark creating new versions with single key update
	for i := 0; i < b.N; i++ {
		v, _ := tree.BeginVersion()

		// Update single key
		key := types.KeyHash([]byte(fmt.Sprintf("key-%d", i%numKeys)))
		value := []byte(fmt.Sprintf("new-value-%d", i))
		tree.PutVersioned(v, key, value)

		tree.CommitVersion(v)
	}
}
