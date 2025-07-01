package tree

import (
	"fmt"
	"testing"
	"time"

	"github.com/neutral/proofbox/pkg/storage"
	"github.com/neutral/proofbox/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGCWithPendingVersions tests GC behavior with pending versions
func TestGCWithPendingVersions(t *testing.T) {
	store := createTestStorage(t)
	keyEncoder := storage.NewDefaultKeyEncoder()
	tree, err := NewTree(store, keyEncoder, DefaultTreeConfig())
	require.NoError(t, err)

	// Set aggressive GC policy
	tree.SetVersionRetentionPolicy(RetentionPolicyCount, 3, 0)

	t.Run("GCDoesNotRemovePendingVersions", func(t *testing.T) {
		// Create some committed versions
		for i := 0; i < 5; i++ {
			key := types.KeyHash([]byte(fmt.Sprintf("key-%d", i)))
			_, err := tree.Put(key, []byte("value"))
			require.NoError(t, err)
		}

		// Create pending versions
		pending1, err := tree.BeginVersion()
		require.NoError(t, err)

		pending2, err := tree.BeginVersion()
		require.NoError(t, err)

		// Run GC
		removed, err := tree.CollectVersionGarbage()
		require.NoError(t, err)
		assert.NotEmpty(t, removed)

		// Verify pending versions still work
		err = tree.PutVersioned(pending1, types.KeyHash([]byte("pending1")), []byte("value1"))
		assert.NoError(t, err)

		err = tree.PutVersioned(pending2, types.KeyHash([]byte("pending2")), []byte("value2"))
		assert.NoError(t, err)

		// Commit pending versions
		err = tree.CommitVersion(pending1)
		assert.NoError(t, err)

		err = tree.CommitVersion(pending2)
		assert.NoError(t, err)
	})

	t.Run("GCDoesNotRemoveCurrentVersion", func(t *testing.T) {
		// Create fresh tree
		store2 := createTestStorage(t)
		keyEncoder2 := storage.NewDefaultKeyEncoder()
		tree2, err := NewTree(store2, keyEncoder2, DefaultTreeConfig())
		require.NoError(t, err)

		tree2.SetVersionRetentionPolicy(RetentionPolicyCount, 1, 0) // Keep only 1 version

		// Create multiple versions
		var lastVersion types.Version
		for i := 0; i < 10; i++ {
			key := types.KeyHash([]byte(fmt.Sprintf("key-%d", i)))
			lastVersion, err = tree2.Put(key, []byte("value"))
			require.NoError(t, err)
		}

		// Run GC
		_, err = tree2.CollectVersionGarbage()
		require.NoError(t, err)

		// Verify current version is still accessible
		key := types.KeyHash([]byte("key-9"))
		got, err := tree2.GetAtVersion(lastVersion, key)
		require.NoError(t, err)
		assert.Equal(t, []byte("value"), got)

		// Verify we have the expected number of versions
		versions := tree2.versionManager.GetAllVersions()
		// Should have version 0 (initial) + 1 retained version
		assert.LessOrEqual(t, len(versions), 2)

		// Current version should be retained
		found := false
		for _, v := range versions {
			if v.Version == lastVersion {
				found = true
				break
			}
		}
		assert.True(t, found, "Current version should be retained")
	})

	t.Run("GCWithAbortedVersions", func(t *testing.T) {
		store3 := createTestStorage(t)
		keyEncoder3 := storage.NewDefaultKeyEncoder()
		tree3, err := NewTree(store3, keyEncoder3, DefaultTreeConfig())
		require.NoError(t, err)

		tree3.SetVersionRetentionPolicy(RetentionPolicyCount, 2, 0)

		// Create and abort some versions
		for i := 0; i < 3; i++ {
			v, err := tree3.BeginVersion()
			require.NoError(t, err)
			err = tree3.AbortVersion(v)
			require.NoError(t, err)
		}

		// Create committed versions
		for i := 0; i < 3; i++ {
			_, err := tree3.Put(types.KeyHash([]byte(fmt.Sprintf("key-%d", i))), []byte("value"))
			require.NoError(t, err)
		}

		// Run GC
		_, err = tree3.CollectVersionGarbage()
		require.NoError(t, err)

		// Verify aborted versions are in the version list but marked as aborted
		versions := tree3.versionManager.GetAllVersions()
		abortedCount := 0
		for _, v := range versions {
			if v.Status == VersionStatusAborted {
				abortedCount++
			}
		}
		assert.Equal(t, 3, abortedCount, "Aborted versions should be retained")
	})
}

// TestTimeBasedGC tests time-based garbage collection
func TestTimeBasedGC(t *testing.T) {
	store := createTestStorage(t)
	keyEncoder := storage.NewDefaultKeyEncoder()
	tree, err := NewTree(store, keyEncoder, DefaultTreeConfig())
	require.NoError(t, err)

	t.Run("RemoveOldVersions", func(t *testing.T) {
		// Set time-based retention (keep versions from last 100ms)
		tree.SetVersionRetentionPolicy(RetentionPolicyTime, 2, 100*time.Millisecond)

		// Create some versions
		oldVersions := make([]types.Version, 3)
		for i := 0; i < 3; i++ {
			v, err := tree.Put(types.KeyHash([]byte(fmt.Sprintf("old-%d", i))), []byte("value"))
			require.NoError(t, err)
			oldVersions[i] = v
		}

		// Wait for versions to age
		time.Sleep(150 * time.Millisecond)

		// Create new versions
		newVersions := make([]types.Version, 2)
		for i := 0; i < 2; i++ {
			v, err := tree.Put(types.KeyHash([]byte(fmt.Sprintf("new-%d", i))), []byte("value"))
			require.NoError(t, err)
			newVersions[i] = v
		}

		// Run GC
		removed, err := tree.CollectVersionGarbage()
		require.NoError(t, err)

		// Old versions should be removed (except minimum retention)
		// We keep at least 2 versions regardless of age
		assert.LessOrEqual(t, len(removed), 3)

		// New versions should be accessible
		for _, v := range newVersions {
			_, err := tree.GetRootHash(v)
			assert.NoError(t, err, "New version %d should be accessible", v)
		}
	})

	t.Run("MinVersionsRespected", func(t *testing.T) {
		store2 := createTestStorage(t)
		keyEncoder2 := storage.NewDefaultKeyEncoder()
		tree2, err := NewTree(store2, keyEncoder2, DefaultTreeConfig())
		require.NoError(t, err)

		// Set time-based retention with min versions
		tree2.SetVersionRetentionPolicy(RetentionPolicyTime, 5, 1*time.Nanosecond) // Very short time

		// Create versions
		for i := 0; i < 10; i++ {
			_, err := tree2.Put(types.KeyHash([]byte(fmt.Sprintf("key-%d", i))), []byte("value"))
			require.NoError(t, err)
		}

		// Even though all versions are "old", we should keep minimum 5
		_, err = tree2.CollectVersionGarbage()
		require.NoError(t, err)

		versions := tree2.versionManager.GetAllVersions()
		committedCount := 0
		for _, v := range versions {
			if v.Status == VersionStatusCommitted {
				committedCount++
			}
		}
		assert.GreaterOrEqual(t, committedCount, 5, "Should keep minimum 5 versions")
	})
}

// TestGCMemoryReclamation tests that GC actually frees memory
func TestGCMemoryReclamation(t *testing.T) {
	store := createTestStorage(t)
	keyEncoder := storage.NewDefaultKeyEncoder()
	tree, err := NewTree(store, keyEncoder, DefaultTreeConfig())
	require.NoError(t, err)

	// Set aggressive GC
	tree.SetVersionRetentionPolicy(RetentionPolicyCount, 10, 0)

	t.Run("VersionMetadataRemoved", func(t *testing.T) {
		// Create many versions
		for i := 0; i < 100; i++ {
			_, err := tree.Put(types.KeyHash([]byte(fmt.Sprintf("key-%d", i))), []byte("value"))
			require.NoError(t, err)
		}

		// Check version count before GC
		versionsBefore := tree.versionManager.GetAllVersions()
		assert.GreaterOrEqual(t, len(versionsBefore), 100)

		// Run GC
		removed, err := tree.CollectVersionGarbage()
		require.NoError(t, err)
		assert.NotEmpty(t, removed)

		// Check version count after GC
		versionsAfter := tree.versionManager.GetAllVersions()
		assert.LessOrEqual(t, len(versionsAfter), 15) // Some buffer for safety

		// Verify removed versions are not accessible
		for _, v := range removed {
			_, err := tree.GetRootHash(v)
			assert.Error(t, err, "Removed version %d should not be accessible", v)
		}
	})

	t.Run("RootHashCacheCleared", func(t *testing.T) {
		// Create fresh tree
		store2 := createTestStorage(t)
		keyEncoder2 := storage.NewDefaultKeyEncoder()
		tree2, err := NewTree(store2, keyEncoder2, DefaultTreeConfig())
		require.NoError(t, err)

		tree2.SetVersionRetentionPolicy(RetentionPolicyCount, 5, 0)

		// Create versions
		versions := make([]types.Version, 20)
		for i := 0; i < 20; i++ {
			v, err := tree2.Put(types.KeyHash([]byte(fmt.Sprintf("key-%d", i))), []byte("value"))
			require.NoError(t, err)
			versions[i] = v
		}

		// Ensure all root hashes are cached
		for _, v := range versions {
			_, err := tree2.GetRootHash(v)
			require.NoError(t, err)
		}

		// Run GC
		removed, err := tree2.CollectVersionGarbage()
		require.NoError(t, err)

		// Verify removed versions' root hashes are gone
		for _, v := range removed {
			tree2.mu.RLock()
			_, exists := tree2.rootHashes[v]
			tree2.mu.RUnlock()
			assert.False(t, exists, "Root hash for version %d should be removed", v)
		}
	})
}

// TestGCConcurrentOperations tests GC during concurrent operations
func TestGCConcurrentOperations(t *testing.T) {
	store := createTestStorage(t)
	keyEncoder := storage.NewDefaultKeyEncoder()
	tree, err := NewTree(store, keyEncoder, DefaultTreeConfig())
	require.NoError(t, err)

	tree.SetVersionRetentionPolicy(RetentionPolicyCount, 20, 0)

	t.Run("GCDuringVersionCreation", func(t *testing.T) {
		done := make(chan bool)
		errors := make(chan error, 100)

		// Concurrent version creator
		go func() {
			for i := 0; i < 50; i++ {
				select {
				case <-done:
					return
				default:
					v, err := tree.BeginVersion()
					if err != nil {
						errors <- err
						return
					}

					key := types.KeyHash([]byte(fmt.Sprintf("concurrent-%d", i)))
					err = tree.PutVersioned(v, key, []byte("value"))
					if err != nil {
						errors <- err
						return
					}

					err = tree.CommitVersion(v)
					if err != nil {
						errors <- err
						return
					}

					time.Sleep(5 * time.Millisecond)
				}
			}
		}()

		// Concurrent GC runner
		go func() {
			for i := 0; i < 10; i++ {
				time.Sleep(25 * time.Millisecond)

				_, err := tree.CollectVersionGarbage()
				if err != nil {
					errors <- err
					return
				}
			}
			close(done)
		}()

		// Wait for completion
		<-done
		close(errors)

		// Check for errors
		for err := range errors {
			t.Errorf("Concurrent operation error: %v", err)
		}
	})
}
