package properties

import (
	"bytes"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/neutral/proofbox/pkg/fuzz"
	"github.com/neutral/proofbox/pkg/fuzz/generators"
	"github.com/neutral/proofbox/pkg/types"
	"pgregory.net/rapid"
)

func TestConcurrentOperations(t *testing.T) {
	config := fuzz.GetTestConfig()
	config.ApplyToRapid(t)
	rapid.Check(t, func(t *rapid.T) {
		// Create tree
		tr, err := fuzz.NewTestTree()
		if err != nil {
			t.Fatalf("failed to create tree: %v", err)
		}

		// Generate base keys that will be used by concurrent operations
		numKeys := rapid.IntRange(5, 20).Draw(t, "num_keys")
		keys := make([]types.Key, numKeys)

		for i := 0; i < numKeys; i++ {
			keys[i] = generators.OperationGen().Draw(t, "key").Key
		}

		// Number of concurrent workers
		numWorkers := rapid.IntRange(2, 8).Draw(t, "num_workers")
		opsPerWorker := rapid.IntRange(5, 20).Draw(t, "ops_per_worker")

		// Counters for verification
		var successfulPuts int64
		var successfulDeletes int64
		var errors int64

		// Run concurrent operations
		var wg sync.WaitGroup
		wg.Add(numWorkers)

		for w := 0; w < numWorkers; w++ {
			go func(workerID int) {
				defer wg.Done()

				// Use standard random for goroutines since rapid.T is not thread-safe
				r := rand.New(rand.NewSource(time.Now().UnixNano() + int64(workerID)))

				for op := 0; op < opsPerWorker; op++ {
					// Pick a random key
					keyIdx := r.Intn(len(keys))
					key := keys[keyIdx]

					// Pick a random operation
					opType := r.Intn(3)

					switch opType {
					case 0: // Put
						value := make([]byte, r.Intn(100)+1)
						r.Read(value)
						_, err := tr.Put(key, value)
						if err == nil {
							atomic.AddInt64(&successfulPuts, 1)
						} else {
							atomic.AddInt64(&errors, 1)
						}

					case 1: // Get
						version := tr.GetLatestVersion()
						_, err := tr.Get(version, key)
						if err != nil {
							// Some errors are expected
							errStr := err.Error()
							if errStr != "version "+fmt.Sprint(version)+" not found" &&
								errStr != "version "+fmt.Sprint(version)+" not committed" {
								atomic.AddInt64(&errors, 1)
							}
						}

					case 2: // Delete
						_, err := tr.Delete(key)
						if err == nil {
							atomic.AddInt64(&successfulDeletes, 1)
						} else if err.Error() != "failed to delete: delete failed: jmt: key not found" {
							atomic.AddInt64(&errors, 1)
						}
					}
				}
			}(w)
		}

		wg.Wait()

		// Verify tree is still consistent
		latestVersion := tr.GetLatestVersion()
		rootHash, err := tr.GetRootHash(latestVersion)
		if err != nil {
			t.Fatalf("failed to get root hash: %v", err)
		}

		// Root hash should be deterministic
		rootHash2, err := tr.GetRootHash(latestVersion)
		if err != nil {
			t.Fatalf("failed to get root hash again: %v", err)
		}

		if rootHash != rootHash2 {
			t.Fatalf("root hash not deterministic after concurrent ops")
		}

		// Log statistics
		t.Logf("Concurrent operations completed: puts=%d, deletes=%d, errors=%d",
			successfulPuts, successfulDeletes, errors)
	})
}

func TestConcurrentReaders(t *testing.T) {
	config := fuzz.GetTestConfig()
	config.ApplyToRapid(t)
	rapid.Check(t, func(t *rapid.T) {
		// Create tree with some data
		tr, err := fuzz.NewTestTree()
		if err != nil {
			t.Fatalf("failed to create tree: %v", err)
		}

		// Add initial data
		numKeys := rapid.IntRange(10, 50).Draw(t, "num_keys")
		keys := make([]types.Key, numKeys)
		expectedValues := make(map[string][]byte)

		for i := 0; i < numKeys; i++ {
			key := generators.OperationGen().Draw(t, "key").Key
			value := rapid.SliceOfN(rapid.Byte(), 1, 100).Draw(t, "value")

			_, err := tr.Put(key, value)
			if err != nil {
				t.Fatalf("failed to put key: %v", err)
			}

			keys[i] = key
			expectedValues[string(key.Bytes())] = value
		}

		version := tr.GetLatestVersion()

		// Concurrent readers
		numReaders := rapid.IntRange(5, 20).Draw(t, "num_readers")
		readsPerReader := rapid.IntRange(10, 50).Draw(t, "reads_per_reader")

		var readErrors int64
		var valueMismatches int64

		var wg sync.WaitGroup
		wg.Add(numReaders)

		for r := 0; r < numReaders; r++ {
			go func(readerID int) {
				defer wg.Done()

				// Create local random generator
				localRand := rand.New(rand.NewSource(time.Now().UnixNano() + int64(readerID)))

				// Create reader for this version
				reader, err := tr.Reader(version)
				if err != nil {
					atomic.AddInt64(&readErrors, 1)
					return
				}
				defer reader.Close()

				for read := 0; read < readsPerReader; read++ {
					// Pick a random key
					keyIdx := localRand.Intn(len(keys))
					key := keys[keyIdx]

					// Read via the tree
					value, err := tr.Get(version, key)
					if err != nil {
						atomic.AddInt64(&readErrors, 1)
						continue
					}

					// Verify value matches expected
					expected := expectedValues[string(key.Bytes())]
					if !bytes.Equal(value, expected) {
						atomic.AddInt64(&valueMismatches, 1)
					}
				}
			}(r)
		}

		wg.Wait()

		// Verify no errors occurred
		if readErrors > 0 {
			t.Fatalf("read errors occurred: %d", readErrors)
		}

		if valueMismatches > 0 {
			t.Fatalf("value mismatches occurred: %d", valueMismatches)
		}
	})
}

func TestConcurrentVersionCreation(t *testing.T) {
	config := fuzz.GetTestConfig()
	config.ApplyToRapid(t)
	rapid.Check(t, func(t *rapid.T) {
		// Create tree
		tr, err := fuzz.NewTestTree()
		if err != nil {
			t.Fatalf("failed to create tree: %v", err)
		}

		// Use a smaller key space to increase conflicts
		baseKeys := []string{"a", "b", "c", "d", "e"}

		// Number of concurrent writers
		numWriters := rapid.IntRange(2, 5).Draw(t, "num_writers")
		writesPerWriter := rapid.IntRange(5, 10).Draw(t, "writes_per_writer")

		var totalVersions int64
		versions := make([]types.Version, 0)
		var mu sync.Mutex

		var wg sync.WaitGroup
		wg.Add(numWriters)

		for w := 0; w < numWriters; w++ {
			go func(writerID int) {
				defer wg.Done()

				// Create local random generator
				localRand := rand.New(rand.NewSource(time.Now().UnixNano() + int64(writerID)))

				for write := 0; write < writesPerWriter; write++ {
					// Pick a random key
					keyStr := baseKeys[localRand.Intn(len(baseKeys))]
					key := types.KeyHash([]byte(keyStr))

					// Generate unique value
					value := []byte(fmt.Sprintf("writer-%d-write-%d", writerID, write))

					// Try to put
					newVersion, err := tr.Put(key, value)
					if err == nil {
						atomic.AddInt64(&totalVersions, 1)

						mu.Lock()
						versions = append(versions, newVersion)
						mu.Unlock()
					}
				}
			}(w)
		}

		wg.Wait()

		// Verify all versions are unique
		versionSet := make(map[types.Version]bool)
		for _, v := range versions {
			if versionSet[v] {
				t.Fatalf("duplicate version created: %d", v)
			}
			versionSet[v] = true
		}

		// Verify we can read from all versions
		for _, v := range versions {
			_, err := tr.GetRootHash(v)
			if err != nil {
				t.Fatalf("cannot get root hash for version %d: %v", v, err)
			}
		}

		t.Logf("Created %d versions concurrently", totalVersions)
	})
}
