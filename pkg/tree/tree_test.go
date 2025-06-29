package tree

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cockroachdb/pebble"
	"github.com/neutral/proofbox/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestDB creates an in-memory test database
func createTestDB(t *testing.T) *pebble.DB {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	opts := &pebble.Options{}

	db, err := pebble.Open(dbPath, opts)
	require.NoError(t, err, "Failed to create test database")

	t.Cleanup(func() {
		db.Close()
		os.RemoveAll(tmpDir)
	})

	return db
}

func TestEmptyTreeGet(t *testing.T) {
	// Create in-memory database
	db := createTestDB(t)

	// Add empty tree at version 0
	batch := db.NewBatch()
	require.NoError(t, batch.Set(makeRootKey(0), types.EmptyHash().Bytes(), nil))
	require.NoError(t, batch.Commit(pebble.Sync))
	batch.Close()

	// Create tree
	tree, err := NewTree(db, DefaultTreeConfig())
	require.NoError(t, err, "Failed to create tree")

	// Get from empty tree (version 0)
	key := types.KeyHash([]byte("test-key"))
	value, err := tree.Get(0, key)

	assert.NoError(t, err, "Get should not error on empty tree")
	assert.Nil(t, value, "Get should return nil for empty tree")
}

func TestVersionNotFound(t *testing.T) {
	db := createTestDB(t)

	tree, err := NewTree(db, DefaultTreeConfig())
	require.NoError(t, err)

	// Try to get from non-existent version
	_, err = tree.Get(999, types.KeyHash([]byte("key")))

	assert.ErrorIs(t, err, types.ErrVersionNotFound)
}

func TestTreeInitialization(t *testing.T) {
	db := createTestDB(t)

	// Add some root hashes manually
	batch := db.NewBatch()
	require.NoError(t, batch.Set(makeRootKey(1), types.Hash{0x01}.Bytes(), nil))
	require.NoError(t, batch.Set(makeRootKey(5), types.Hash{0x05}.Bytes(), nil))
	require.NoError(t, batch.Set(makeRootKey(3), types.Hash{0x03}.Bytes(), nil))
	require.NoError(t, batch.Commit(pebble.Sync))
	batch.Close()

	// Create tree - should load existing versions
	tree, err := NewTree(db, DefaultTreeConfig())
	require.NoError(t, err, "Failed to create tree")

	// Check latest version
	assert.Equal(t, types.Version(5), tree.GetLatestVersion())

	// Check all versions exist
	for _, v := range []types.Version{1, 3, 5} {
		assert.True(t, tree.HasVersion(v), "Version %d should exist", v)
	}

	// Check root hashes
	hash1, err := tree.GetRootHash(1)
	require.NoError(t, err)
	assert.Equal(t, types.Hash{0x01}, hash1)

	hash5, err := tree.GetRootHash(5)
	require.NoError(t, err)
	assert.Equal(t, types.Hash{0x05}, hash5)
}

func TestIsEmpty(t *testing.T) {
	db := createTestDB(t)

	// Add empty tree (version 0 with empty hash)
	batch := db.NewBatch()
	require.NoError(t, batch.Set(makeRootKey(0), types.EmptyHash().Bytes(), nil))
	// Add non-empty tree (version 1)
	require.NoError(t, batch.Set(makeRootKey(1), types.Hash{0x01}.Bytes(), nil))
	require.NoError(t, batch.Commit(pebble.Sync))
	batch.Close()

	tree, err := NewTree(db, DefaultTreeConfig())
	require.NoError(t, err)

	// Check empty tree
	isEmpty, err := tree.IsEmpty(0)
	require.NoError(t, err)
	assert.True(t, isEmpty, "Version 0 should be empty")

	// Check non-empty tree
	isEmpty, err = tree.IsEmpty(1)
	require.NoError(t, err)
	assert.False(t, isEmpty, "Version 1 should not be empty")

	// Check non-existent version
	_, err = tree.IsEmpty(999)
	assert.ErrorIs(t, err, types.ErrVersionNotFound)
}

func TestNodeCache(t *testing.T) {
	cache, err := NewNodeCache(100)
	require.NoError(t, err)

	// Test empty cache
	_, found := cache.Get(types.RootNodeKey(0))
	assert.False(t, found)

	// Test cache hit rate with no operations
	assert.Equal(t, float64(0), cache.HitRate())

	// Test stats
	hits, misses, size := cache.Stats()
	assert.Equal(t, uint64(0), hits)
	assert.Equal(t, uint64(1), misses) // from Get above
	assert.Equal(t, 0, size)

	// For testing cache operations, we need nodes
	// Since we can't create real nodes due to import cycle, we'll skip node-specific tests
	// This will be fixed when the codec cycle is resolved

	// Test cache clear
	cache.Clear()
	hits, misses, size = cache.Stats()
	assert.Equal(t, 0, size)
}

func TestErrorHandlers(t *testing.T) {
	t.Run("WriteRetryWithBackoff", func(t *testing.T) {
		handler := NewDefaultWriteErrorHandler(3, 100*time.Millisecond, nil)

		// Test exponential backoff
		for i := 0; i < 3; i++ {
			delay := handler.RetryDelay(i)
			expected := 100 * time.Millisecond * time.Duration(1<<uint(i))

			// Allow for jitter (up to 25%)
			assert.InDelta(t, expected, delay, float64(expected)*0.3,
				"Delay for attempt %d should be close to %v", i, expected)
		}

		// Test should not retry by default
		assert.False(t, handler.ShouldRetry(assert.AnError))
	})

	t.Run("DefaultReadErrorHandler", func(t *testing.T) {
		handler := NewDefaultReadErrorHandler(nil)

		// Test corrupted node handling
		key := types.NodeKey{
			Version: 1,
			NibblePath: types.NibblePath{
				Nibbles: []types.Nibble{0x1, 0x2},
				Length:  2,
			},
		}
		err := handler.HandleCorruptedNode(key, assert.AnError)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "corrupted node")

		// Test missing node handling
		err = handler.HandleMissingNode(key)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "node not found")
	})
}

func TestValidation(t *testing.T) {
	db := createTestDB(t)
	tree, err := NewTree(db, DefaultTreeConfig())
	require.NoError(t, err)

	t.Run("ValidateVersion", func(t *testing.T) {
		// Valid version
		err := tree.validateVersion(0)
		assert.NoError(t, err)

		err = tree.validateVersion(types.MaxVersion)
		assert.NoError(t, err)

		// Invalid version
		err = tree.validateVersion(types.MaxVersion + 1)
		assert.ErrorIs(t, err, types.ErrInvalidVersionRange)
	})

	t.Run("ValidateKey", func(t *testing.T) {
		// Valid key
		key := types.KeyHash([]byte("test"))
		err := tree.validateKey(key)
		assert.NoError(t, err)

		// Empty key
		err = tree.validateKey(types.Key{})
		assert.ErrorIs(t, err, types.ErrEmptyKey)
	})
}

func TestHealthChecker(t *testing.T) {
	db := createTestDB(t)

	// Add a version
	batch := db.NewBatch()
	require.NoError(t, batch.Set(makeRootKey(0), types.EmptyHash().Bytes(), nil))
	require.NoError(t, batch.Commit(pebble.Sync))
	batch.Close()

	tree, err := NewTree(db, DefaultTreeConfig())
	require.NoError(t, err)

	checker := NewTreeHealthChecker(tree)

	t.Run("CheckVersion", func(t *testing.T) {
		// Valid version
		err := checker.CheckVersion(0)
		assert.NoError(t, err)

		// Invalid version
		err = checker.CheckVersion(999)
		assert.ErrorIs(t, err, types.ErrVersionNotFound)
	})

	// Note: Full integrity check requires working node decoding
}

func TestStorageKeys(t *testing.T) {
	t.Run("RootKeys", func(t *testing.T) {
		// Test makeRootKey
		key := makeRootKey(42)
		assert.Equal(t, []byte("r"), key[:1])

		// Test parseRootKey
		version, err := parseRootKey(key)
		require.NoError(t, err)
		assert.Equal(t, types.Version(42), version)

		// Test invalid root key
		_, err = parseRootKey([]byte("invalid"))
		assert.Error(t, err)
	})

	t.Run("NodeKeys", func(t *testing.T) {
		nodeKey := types.NodeKey{
			Version: 10,
			NibblePath: types.NibblePath{
				Nibbles: []types.Nibble{0x1, 0x2, 0x3},
				Length:  3,
			},
		}

		// Test makeNodeKey
		storageKey := makeNodeKey(nodeKey)
		assert.Equal(t, byte('n'), storageKey[0])

		// Test parseNodeKey
		parsed, err := parseNodeKey(storageKey)
		require.NoError(t, err)
		assert.Equal(t, nodeKey.Version, parsed.Version)
		assert.Equal(t, nodeKey.NibblePath.Length, parsed.NibblePath.Length)
		assert.Equal(t, nodeKey.NibblePath.Nibbles, parsed.NibblePath.Nibbles)
	})

	t.Run("ValueKeys", func(t *testing.T) {
		// Test makeValueKey
		hash := types.Hash{0x01, 0x02, 0x03}
		key := makeValueKey(5, hash)
		assert.Equal(t, byte('v'), key[0])

		// Test makeValueKeyByKey
		testKey := types.KeyHash([]byte("test"))
		keyByKey := makeValueKeyByKey(testKey)
		assert.Equal(t, byte('k'), keyByKey[0])
	})
}