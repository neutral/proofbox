package storage_test

import (
	"sync"
	"testing"
	"time"

	"github.com/neutral/proofbox/pkg/storage"
	"github.com/neutral/proofbox/pkg/storage/memory"
	"github.com/neutral/proofbox/pkg/storage/pebble"
	"github.com/stretchr/testify/require"
)

// TestResourceLimits verifies resource limits are enforced
func TestResourceLimits(t *testing.T) {
	tests := []struct {
		name    string
		factory func() (storage.Storage, error)
	}{
		{
			name: "memory",
			factory: func() (storage.Storage, error) {
				return memory.NewStorageWithOptions(&memory.Options{
					MaxIterators: 5,
					MaxSnapshots: 3,
				}), nil
			},
		},
		{
			name: "pebble",
			factory: func() (storage.Storage, error) {
				dir := t.TempDir()
				return pebble.NewStorage(dir, &pebble.Options{
					MaxIterators: 5,
					MaxSnapshots: 3,
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, err := tt.factory()
			require.NoError(t, err)
			defer store.Close()

			// Test iterator limits
			t.Run("iterator_limits", func(t *testing.T) {
				iterators := make([]storage.Iterator, 0, 6)

				// Create up to limit
				for i := 0; i < 5; i++ {
					iter := store.NewIterator(nil)
					require.NotNil(t, iter)
					require.NoError(t, iter.Error())
					iterators = append(iterators, iter)
				}

				// Should fail when exceeding limit
				iter := store.NewIterator(nil)
				require.NotNil(t, iter)
				require.ErrorIs(t, iter.Error(), storage.ErrTooManyIterators)

				// Close one iterator
				require.NoError(t, iterators[0].Close())

				// Should be able to create new one
				iter = store.NewIterator(nil)
				require.NoError(t, iter.Error())
				require.NoError(t, iter.Close())

				// Clean up
				for i := 1; i < len(iterators); i++ {
					require.NoError(t, iterators[i].Close())
				}
			})

			// Test snapshot limits
			t.Run("snapshot_limits", func(t *testing.T) {
				snapshots := make([]storage.Snapshot, 0, 4)

				// Create up to limit
				for i := 0; i < 3; i++ {
					snap := store.NewSnapshot()
					require.NotNil(t, snap)
					snapshots = append(snapshots, snap)
				}

				// Should fail when exceeding limit
				snap := store.NewSnapshot()
				require.NotNil(t, snap)
				// Try to use it - should fail
				_, err := snap.Get([]byte("test"))
				require.ErrorIs(t, err, storage.ErrTooManySnapshots)

				// Close one snapshot
				require.NoError(t, snapshots[0].Close())

				// Should be able to create new one
				snap = store.NewSnapshot()
				_, err = snap.Get([]byte("test"))
				require.NoError(t, err) // No error (just not found)
				require.NoError(t, snap.Close())

				// Clean up
				for i := 1; i < len(snapshots); i++ {
					require.NoError(t, snapshots[i].Close())
				}
			})
		})
	}
}

// TestStorageCloseWaitsForResources verifies storage waits for resources before closing
func TestStorageCloseWaitsForResources(t *testing.T) {
	tests := []struct {
		name    string
		factory func() (storage.Storage, error)
	}{
		{
			name: "memory",
			factory: func() (storage.Storage, error) {
				return memory.NewStorage(), nil
			},
		},
		{
			name: "pebble",
			factory: func() (storage.Storage, error) {
				dir := t.TempDir()
				return pebble.NewStorage(dir, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, err := tt.factory()
			require.NoError(t, err)

			// Create some resources
			iter := store.NewIterator(nil)
			require.NoError(t, iter.Error())

			snap := store.NewSnapshot()
			require.NotNil(t, snap)

			batch := store.NewBatch()
			require.NotNil(t, batch)

			// Start goroutine that closes resources after delay
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()
				time.Sleep(100 * time.Millisecond)
				iter.Close()
				snap.Close()
				batch.Close()
			}()

			// Close should wait for resources
			start := time.Now()
			require.NoError(t, store.Close())
			elapsed := time.Since(start)

			// Should have waited for resources
			require.Greater(t, elapsed, 90*time.Millisecond)

			// Wait for goroutine
			wg.Wait()
		})
	}
}

// TestOperationsAfterClose verifies operations fail after close
func TestOperationsAfterClose(t *testing.T) {
	tests := []struct {
		name    string
		factory func() (storage.Storage, error)
	}{
		{
			name: "memory",
			factory: func() (storage.Storage, error) {
				return memory.NewStorage(), nil
			},
		},
		{
			name: "pebble",
			factory: func() (storage.Storage, error) {
				dir := t.TempDir()
				return pebble.NewStorage(dir, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, err := tt.factory()
			require.NoError(t, err)

			// Close the storage
			require.NoError(t, store.Close())

			// All operations should fail
			_, err = store.Get([]byte("key"))
			require.ErrorIs(t, err, storage.ErrStorageClosed)

			err = store.Put([]byte("key"), []byte("value"))
			require.ErrorIs(t, err, storage.ErrStorageClosed)

			err = store.Delete([]byte("key"))
			require.ErrorIs(t, err, storage.ErrStorageClosed)

			// Resource creation should return error types
			iter := store.NewIterator(nil)
			require.ErrorIs(t, iter.Error(), storage.ErrStorageClosed)

			snap := store.NewSnapshot()
			_, err = snap.Get([]byte("test"))
			require.ErrorIs(t, err, storage.ErrStorageClosed)

			batch := store.NewBatch()
			err = batch.Put([]byte("key"), []byte("value"))
			require.ErrorIs(t, err, storage.ErrStorageClosed)
		})
	}
}

// TestConcurrentResourceCreation tests thread safety of resource limits
func TestConcurrentResourceCreation(t *testing.T) {
	store := memory.NewStorageWithOptions(&memory.Options{
		MaxIterators: 10,
		MaxSnapshots: 5,
	})
	defer store.Close()

	// Test concurrent iterator creation
	t.Run("concurrent_iterators", func(t *testing.T) {
		var wg sync.WaitGroup
		successCount := 0
		var mu sync.Mutex

		// Try to create 20 iterators concurrently (limit is 10)
		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				iter := store.NewIterator(nil)
				if iter.Error() == nil {
					mu.Lock()
					successCount++
					mu.Unlock()
					// Hold for a bit
					time.Sleep(10 * time.Millisecond)
					iter.Close()
				}
			}()
		}

		wg.Wait()

		// Should have created exactly up to the limit
		require.LessOrEqual(t, successCount, 10)
		require.Greater(t, successCount, 0) // At least some should succeed
	})
}

// TestResourceDoubleClose verifies double close is safe
func TestResourceDoubleClose(t *testing.T) {
	tests := []struct {
		name    string
		factory func() (storage.Storage, error)
	}{
		{
			name: "memory",
			factory: func() (storage.Storage, error) {
				return memory.NewStorage(), nil
			},
		},
		{
			name: "pebble",
			factory: func() (storage.Storage, error) {
				dir := t.TempDir()
				return pebble.NewStorage(dir, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, err := tt.factory()
			require.NoError(t, err)

			// Create resources
			iter := store.NewIterator(nil)
			snap := store.NewSnapshot()
			batch := store.NewBatch()

			// Double close resources - should be safe
			require.NoError(t, iter.Close())
			require.NoError(t, iter.Close())

			require.NoError(t, snap.Close())
			require.NoError(t, snap.Close())

			require.NoError(t, batch.Close())
			require.NoError(t, batch.Close())

			// Double close storage - should be safe
			require.NoError(t, store.Close())
			require.NoError(t, store.Close())
		})
	}
}
