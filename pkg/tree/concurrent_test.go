package tree

import (
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/neutral/proofbox/pkg/storage"
	pebblestorage "github.com/neutral/proofbox/pkg/storage/pebble"
	"github.com/neutral/proofbox/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConcurrentVersionCreation(t *testing.T) {
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

	// Create multiple versions concurrently
	numGoroutines := 10
	versionsPerGoroutine := 5

	var wg sync.WaitGroup
	versions := make(chan types.Version, numGoroutines*versionsPerGoroutine)
	errors := make(chan error, numGoroutines*versionsPerGoroutine)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < versionsPerGoroutine; j++ {
				// Begin version
				v, err := tree.BeginVersion()
				if err != nil {
					errors <- err
					continue
				}

				// Insert some data
				key := types.KeyHash([]byte(fmt.Sprintf("g%d-k%d", goroutineID, j)))
				value := []byte(fmt.Sprintf("value-%d-%d", goroutineID, j))

				err = tree.PutVersioned(v, key, value)
				if err != nil {
					tree.AbortVersion(v)
					errors <- err
					continue
				}

				// Commit
				err = tree.CommitVersion(v)
				if err != nil {
					errors <- err
					continue
				}

				versions <- v
			}
		}(i)
	}

	wg.Wait()
	close(versions)
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("Concurrent operation failed: %v", err)
	}

	// Verify all versions were created
	allVersions := make([]types.Version, 0)
	for v := range versions {
		allVersions = append(allVersions, v)
	}

	assert.Equal(t, numGoroutines*versionsPerGoroutine, len(allVersions))

	// Verify all versions are unique
	versionSet := make(map[types.Version]bool)
	for _, v := range allVersions {
		assert.False(t, versionSet[v], "Duplicate version: %d", v)
		versionSet[v] = true
	}
}

func TestConcurrentReadsWhileWriting(t *testing.T) {
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

	// Insert initial data
	initialVersion, err := tree.Put(types.KeyHash([]byte("key1")), []byte("initial"))
	require.NoError(t, err)

	// Start concurrent readers
	numReaders := 5
	readDuration := 2 * time.Second
	stopReaders := make(chan bool)
	readErrors := make(chan error, 100)

	var readWg sync.WaitGroup
	for i := 0; i < numReaders; i++ {
		readWg.Add(1)
		go func(readerID int) {
			defer readWg.Done()

			for {
				select {
				case <-stopReaders:
					return
				default:
					// Read from initial version
					val, err := tree.GetAtVersion(initialVersion, types.KeyHash([]byte("key1")))
					if err != nil {
						readErrors <- err
						return
					}
					if string(val) != "initial" {
						readErrors <- fmt.Errorf("reader %d: unexpected value: %s", readerID, val)
						return
					}

					// Small delay
					time.Sleep(10 * time.Millisecond)
				}
			}
		}(i)
	}

	// Concurrent writer
	go func() {
		for i := 0; i < 20; i++ {
			key := types.KeyHash([]byte(fmt.Sprintf("key%d", i+2)))
			value := []byte(fmt.Sprintf("value%d", i))

			_, err := tree.Put(key, value)
			if err != nil {
				readErrors <- fmt.Errorf("write error: %w", err)
				return
			}

			time.Sleep(50 * time.Millisecond)
		}
	}()

	// Let it run
	time.Sleep(readDuration)
	close(stopReaders)
	readWg.Wait()
	close(readErrors)

	// Check for errors
	for err := range readErrors {
		t.Errorf("Concurrent read/write error: %v", err)
	}
}

func TestVersionGarbageCollectionConcurrency(t *testing.T) {
	// Create test tree with aggressive GC
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

	// Set retention policy to keep only 10 versions
	tree.SetVersionRetentionPolicy(RetentionPolicyCount, 10, 0)

	// Create versions and run GC concurrently
	var wg sync.WaitGroup
	errors := make(chan error, 100)
	versionCreationDone := make(chan bool)

	// Version creator
	wg.Add(1)
	go func() {
		defer wg.Done()

		for i := 0; i < 50; i++ {
			key := types.KeyHash([]byte(fmt.Sprintf("key%d", i)))
			value := []byte(fmt.Sprintf("value%d", i))

			_, err := tree.Put(key, value)
			if err != nil {
				errors <- fmt.Errorf("put error: %w", err)
				return
			}

			time.Sleep(10 * time.Millisecond)
		}
		close(versionCreationDone)
	}()

	// Garbage collector
	wg.Add(1)
	go func() {
		defer wg.Done()

		for i := 0; i < 10; i++ {
			time.Sleep(50 * time.Millisecond)

			removed, err := tree.CollectVersionGarbage()
			if err != nil {
				errors <- fmt.Errorf("GC error: %w", err)
				return
			}

			if len(removed) > 0 {
				t.Logf("GC removed %d versions", len(removed))
			}
		}

		// After version creation is done, run final GC passes to ensure cleanup
		<-versionCreationDone
		for i := 0; i < 5; i++ {
			removed, err := tree.CollectVersionGarbage()
			if err != nil {
				errors <- fmt.Errorf("final GC error: %w", err)
				return
			}
			if len(removed) > 0 {
				t.Logf("Final GC removed %d versions", len(removed))
			}
			time.Sleep(10 * time.Millisecond)
		}
	}()

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("Concurrent GC error: %v", err)
	}

	// Verify we don't have too many versions
	// The retention policy is 10, so we should have at most 10 versions
	// plus a small buffer for any in-flight operations
	allVersions := tree.versionManager.GetAllVersions()
	assert.LessOrEqual(t, len(allVersions), 15, "Too many versions retained")
}

func TestConcurrentAborts(t *testing.T) {
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

	// Create multiple versions and randomly abort some
	numGoroutines := 10
	var wg sync.WaitGroup
	committed := make(chan types.Version, numGoroutines)
	aborted := make(chan types.Version, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			v, err := tree.BeginVersion()
			require.NoError(t, err)

			// Insert data
			key := types.KeyHash([]byte(fmt.Sprintf("key%d", id)))
			value := []byte(fmt.Sprintf("value%d", id))

			err = tree.PutVersioned(v, key, value)
			require.NoError(t, err)

			// Randomly commit or abort
			if id%2 == 0 {
				err = tree.CommitVersion(v)
				if err == nil {
					committed <- v
				}
			} else {
				err = tree.AbortVersion(v)
				if err == nil {
					aborted <- v
				}
			}
		}(i)
	}

	wg.Wait()
	close(committed)
	close(aborted)

	// Verify committed versions are accessible
	for v := range committed {
		info, err := tree.versionManager.GetVersion(v)
		require.NoError(t, err)
		assert.Equal(t, VersionStatusCommitted, info.Status)
	}

	// Verify aborted versions are marked correctly
	for v := range aborted {
		info, err := tree.versionManager.GetVersion(v)
		require.NoError(t, err)
		assert.Equal(t, VersionStatusAborted, info.Status)
	}
}
