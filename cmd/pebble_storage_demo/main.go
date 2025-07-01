package main

import (
	"fmt"
	"log"
	"os"

	"github.com/neutral/proofbox/pkg/storage"
	pebblestorage "github.com/neutral/proofbox/pkg/storage/pebble"
	"github.com/neutral/proofbox/pkg/tree"
	"github.com/neutral/proofbox/pkg/types"
)

func main() {
	fmt.Println("PebbleDB Storage Driver Demo")
	fmt.Println("============================")

	// Create temporary directory
	tmpDir := "./test-pebble-storage"
	os.RemoveAll(tmpDir) // Clean up any existing
	defer os.RemoveAll(tmpDir) // Clean up after

	// Create PebbleDB storage using the official driver
	opts := &pebblestorage.Options{
		EnableMetrics: true,
	}
	
	store, err := pebblestorage.NewStorage(tmpDir, opts)
	if err != nil {
		log.Fatalf("Failed to create PebbleDB storage: %v", err)
	}
	defer store.Close()

	fmt.Println("\n✓ Created PebbleDB storage using official driver")

	// Create tree with PebbleDB storage
	keyEncoder := storage.NewDefaultKeyEncoder()
	jmt, err := tree.NewTree(store, keyEncoder, tree.DefaultTreeConfig())
	if err != nil {
		log.Fatalf("Failed to create tree: %v", err)
	}

	fmt.Println("✓ Created tree with PebbleDB storage")

	// Add some data
	fmt.Println("\nAdding data:")
	testData := map[string]string{
		"persistent1": "value1",
		"persistent2": "value2",
		"persistent3": "value3",
	}

	for k, v := range testData {
		key := types.KeyHash([]byte(k))
		value := []byte(v)
		version, err := jmt.Put(key, value)
		if err != nil {
			log.Fatalf("Failed to put %s: %v", k, err)
		}
		fmt.Printf("  %s = %s (version %d)\n", k, v, version)
	}

	// Get metrics
	metrics := store.Metrics()
	fmt.Printf("\nStorage Metrics:\n")
	fmt.Printf("  Get operations: %d\n", metrics.GetOperations())
	fmt.Printf("  Put operations: %d\n", metrics.PutOperations())
	fmt.Printf("  Batch commits: %d\n", metrics.BatchCommits())

	// Close and reopen to test persistence
	latestVersion := jmt.GetLatestVersion()
	store.Close()
	store = nil // Prevent double close in defer

	fmt.Println("\n✓ Closed storage")

	// Reopen
	store2, err := pebblestorage.NewStorage(tmpDir, opts)
	if err != nil {
		log.Fatalf("Failed to reopen storage: %v", err)
	}
	defer store2.Close()

	jmt2, err := tree.NewTree(store2, keyEncoder, tree.DefaultTreeConfig())
	if err != nil {
		log.Fatalf("Failed to recreate tree: %v", err)
	}

	fmt.Println("✓ Reopened storage and tree")

	// Verify data persisted
	fmt.Printf("\nVerifying persisted data (version %d):\n", latestVersion)
	for k, expectedValue := range testData {
		key := types.KeyHash([]byte(k))
		value, err := jmt2.Get(latestVersion, key)
		if err != nil {
			log.Fatalf("Failed to get %s: %v", k, err)
		}
		if string(value) != expectedValue {
			log.Fatalf("Value mismatch for %s: got %s, want %s", k, value, expectedValue)
		}
		fmt.Printf("  %s = %s ✓\n", k, value)
	}

	fmt.Println("\n✅ PebbleDB storage driver working correctly!")
}