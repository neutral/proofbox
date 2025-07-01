package tree

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/neutral/proofbox/pkg/codec"
	"github.com/neutral/proofbox/pkg/storage"
	pebblestorage "github.com/neutral/proofbox/pkg/storage/pebble"
	"github.com/neutral/proofbox/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestVersionPersistence tests version persistence across tree restarts
func TestVersionPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "persist.db")

	t.Run("BasicPersistence", func(t *testing.T) {
		// Create tree and add data
		opts := &pebblestorage.Options{
			EnableMetrics: false,
		}
		store1, err := pebblestorage.NewStorage(dbPath, opts)
		require.NoError(t, err)
		keyEncoder1 := storage.NewDefaultKeyEncoder()

		tree1, err := NewTree(store1, keyEncoder1, testTreeConfig())
		require.NoError(t, err)

		// Create multiple versions
		versions := make([]types.Version, 5)
		for i := 0; i < 5; i++ {
			key := types.KeyHash([]byte(fmt.Sprintf("persist-key-%d", i)))
			value := []byte(fmt.Sprintf("persist-value-%d", i))
			v, err := tree1.Put(key, value)
			require.NoError(t, err)
			versions[i] = v
		}

		// Close tree
		store1.Close()

		// Reopen tree
		store2, err := pebblestorage.NewStorage(dbPath, opts)
		require.NoError(t, err)
		defer store2.Close()
		keyEncoder2 := storage.NewDefaultKeyEncoder()

		tree2, err := NewTree(store2, keyEncoder2, testTreeConfig())
		require.NoError(t, err)

		// Verify latest version
		assert.Equal(t, versions[4], tree2.GetLatestVersion())

		// Verify all versions are accessible via Get (uses version parameter)
		for i, v := range versions {
			key := types.KeyHash([]byte(fmt.Sprintf("persist-key-%d", i)))
			got, err := tree2.Get(v, key)
			require.NoError(t, err)
			expected := []byte(fmt.Sprintf("persist-value-%d", i))
			assert.Equal(t, expected, got)

			// Verify root hash exists
			_, err = tree2.GetRootHash(v)
			assert.NoError(t, err)
		}
	})

	t.Run("PendingVersionsNotPersisted", func(t *testing.T) {
		// Create fresh DB
		tmpDir2 := t.TempDir()
		dbPath2 := filepath.Join(tmpDir2, "pending.db")

		opts := &pebblestorage.Options{
			EnableMetrics: false,
		}
		store1, err := pebblestorage.NewStorage(dbPath2, opts)
		require.NoError(t, err)
		keyEncoder1 := storage.NewDefaultKeyEncoder()

		tree1, err := NewTree(store1, keyEncoder1, testTreeConfig())
		require.NoError(t, err)

		// Create committed version
		committedVersion, err := tree1.Put(types.KeyHash([]byte("committed")), []byte("value"))
		require.NoError(t, err)

		// Create pending versions (not committed)
		pending1, err := tree1.BeginVersion()
		require.NoError(t, err)
		err = tree1.PutVersioned(pending1, types.KeyHash([]byte("pending1")), []byte("value1"))
		require.NoError(t, err)

		pending2, err := tree1.BeginVersion()
		require.NoError(t, err)

		// Close without committing
		store1.Close()

		// Reopen
		store2, err := pebblestorage.NewStorage(dbPath2, opts)
		require.NoError(t, err)
		defer store2.Close()
		keyEncoder2 := storage.NewDefaultKeyEncoder()

		tree2, err := NewTree(store2, keyEncoder2, testTreeConfig())
		require.NoError(t, err)

		// Committed version should be accessible
		got, err := tree2.Get(committedVersion, types.KeyHash([]byte("committed")))
		require.NoError(t, err)
		assert.Equal(t, []byte("value"), got)

		// Pending versions' data should not be accessible
		_, err = tree2.Get(pending1, types.KeyHash([]byte("pending1")))
		assert.Error(t, err)

		// Version manager should not have pending versions
		info1, err := tree2.versionManager.GetVersion(pending1)
		assert.Error(t, err)
		assert.Nil(t, info1)

		info2, err := tree2.versionManager.GetVersion(pending2)
		assert.Error(t, err)
		assert.Nil(t, info2)
	})

	t.Run("VersionManagerStatePersistence", func(t *testing.T) {
		// Create fresh DB
		tmpDir3 := t.TempDir()
		dbPath3 := filepath.Join(tmpDir3, "state.db")

		opts := &pebblestorage.Options{
			EnableMetrics: false,
		}
		store1, err := pebblestorage.NewStorage(dbPath3, opts)
		require.NoError(t, err)
		keyEncoder1 := storage.NewDefaultKeyEncoder()

		tree1, err := NewTree(store1, keyEncoder1, testTreeConfig())
		require.NoError(t, err)

		// Set custom retention policy
		tree1.SetVersionRetentionPolicy(RetentionPolicyCount, 25, 0)

		// Create versions
		var lastVersion types.Version
		for i := 0; i < 10; i++ {
			key := types.KeyHash([]byte(fmt.Sprintf("state-key-%d", i)))
			lastVersion, err = tree1.Put(key, []byte("value"))
			require.NoError(t, err)
		}

		// Run GC
		removed1, err := tree1.CollectVersionGarbage()
		require.NoError(t, err)

		store1.Close()

		// Reopen
		store2, err := pebblestorage.NewStorage(dbPath3, opts)
		require.NoError(t, err)
		defer store2.Close()
		keyEncoder2 := storage.NewDefaultKeyEncoder()

		tree2, err := NewTree(store2, keyEncoder2, testTreeConfig())
		require.NoError(t, err)

		// Latest version should match
		assert.Equal(t, lastVersion, tree2.GetLatestVersion())

		// Removed versions should still be inaccessible
		for _, v := range removed1 {
			_, err := tree2.GetRootHash(v)
			assert.Error(t, err, "Previously removed version %d should remain inaccessible", v)
		}

		// Note: Retention policy is not persisted, so it resets to default
		// This is a design decision - retention policy is runtime configuration
	})
}

// TestCrashRecovery tests recovery after crashes at various points
func TestCrashRecovery(t *testing.T) {
	t.Run("CrashDuringCommit", func(t *testing.T) {
		// This test simulates a crash during commit by not completing the operation
		// In real scenario, PebbleDB's WAL ensures atomicity

		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "crash.db")

		opts := &pebblestorage.Options{
			EnableMetrics: false,
		}
		store, err := pebblestorage.NewStorage(dbPath, opts)
		require.NoError(t, err)
		keyEncoder := storage.NewDefaultKeyEncoder()

		tree, err := NewTree(store, keyEncoder, testTreeConfig())
		require.NoError(t, err)

		// Create initial version
		v1, err := tree.Put(types.KeyHash([]byte("initial")), []byte("value1"))
		require.NoError(t, err)

		// Start new version
		v2, err := tree.BeginVersion()
		require.NoError(t, err)

		err = tree.PutVersioned(v2, types.KeyHash([]byte("crash-key")), []byte("crash-value"))
		require.NoError(t, err)

		// Simulate partial commit by manually building batch
		pending := tree.versionManager.GetPending(v2)
		require.NotNil(t, pending)

		batch, err := pending.updater.BuildUpdateBatch()
		require.NoError(t, err)

		// Write nodes but don't write root hash (simulating crash)
		writeBatch := store.NewBatch()
		nodeCodec := &codec.NodeCodec{}

		for _, nodeWrite := range batch.NewNodes {
			data, err := nodeCodec.EncodeNode(nodeWrite.Node)
			require.NoError(t, err)

			storageKey := storage.NewDefaultKeyEncoder().NodeKey(nodeWrite.Key)
			err = writeBatch.Put(storageKey, data)
			require.NoError(t, err)
		}

		// Commit partial write (no root hash)
		err = writeBatch.Commit(storage.CommitOptions{Sync: true})
		require.NoError(t, err)
		writeBatch.Close()

		// Close and reopen
		store.Close()

		store2, err := pebblestorage.NewStorage(dbPath, opts)
		require.NoError(t, err)
		defer store2.Close()
		keyEncoder2 := storage.NewDefaultKeyEncoder()

		tree2, err := NewTree(store2, keyEncoder2, testTreeConfig())
		require.NoError(t, err)

		// Version 2 should not exist (no root hash written)
		_, err = tree2.GetRootHash(v2)
		assert.Error(t, err)

		// Version 1 should still be intact
		got, err := tree2.GetAtVersion(v1, types.KeyHash([]byte("initial")))
		require.NoError(t, err)
		assert.Equal(t, []byte("value1"), got)

		// Latest version should still be v1
		assert.Equal(t, v1, tree2.GetLatestVersion())
	})
}
