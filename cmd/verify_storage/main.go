package main

import (
	"fmt"
	"log"

	"github.com/neutral/proofbox/pkg/storage"
	"github.com/neutral/proofbox/pkg/storage/memory"
	"github.com/neutral/proofbox/pkg/tree"
	"github.com/neutral/proofbox/pkg/types"
)

func main() {
	fmt.Println("Verifying Storage Abstraction")
	fmt.Println("=============================")

	// Test 1: Memory Storage
	testMemoryStorage()

	// Test 2: Tree with Storage
	testTreeWithStorage()

	fmt.Println("\n✅ All storage abstraction tests passed!")
}

func testMemoryStorage() {
	fmt.Println("\n1. Testing Memory Storage Implementation:")
	
	store := memory.NewStorage()
	defer store.Close()

	// Test basic operations
	key := []byte("test-key")
	value := []byte("test-value")

	// Put
	err := store.Put(key, value)
	if err != nil {
		log.Fatalf("Put failed: %v", err)
	}
	fmt.Println("  ✓ Put operation successful")

	// Get
	retrieved, err := store.Get(key)
	if err != nil {
		log.Fatalf("Get failed: %v", err)
	}
	if string(retrieved) != string(value) {
		log.Fatalf("Value mismatch: got %s, want %s", retrieved, value)
	}
	fmt.Println("  ✓ Get operation successful")

	// Batch operations
	batch := store.NewBatch()
	batch.Put([]byte("batch-key1"), []byte("batch-value1"))
	batch.Put([]byte("batch-key2"), []byte("batch-value2"))
	err = batch.Commit(storage.CommitOptions{Sync: true})
	if err != nil {
		log.Fatalf("Batch commit failed: %v", err)
	}
	fmt.Println("  ✓ Batch operations successful")

	// Verify batch results
	val1, _ := store.Get([]byte("batch-key1"))
	val2, _ := store.Get([]byte("batch-key2"))
	if string(val1) != "batch-value1" || string(val2) != "batch-value2" {
		log.Fatalf("Batch values incorrect")
	}
	fmt.Println("  ✓ Batch values verified")

	// Iterator
	iter := store.NewIterator(nil)
	defer iter.Close()
	
	count := 0
	for iter.First(); iter.Valid(); iter.Next() {
		count++
	}
	if count != 3 { // test-key, batch-key1, batch-key2
		log.Fatalf("Expected 3 keys, got %d", count)
	}
	fmt.Println("  ✓ Iterator working correctly")

	// Snapshot
	snapshot := store.NewSnapshot()
	defer snapshot.Close()
	
	snapVal, err := snapshot.Get(key)
	if err != nil || string(snapVal) != string(value) {
		log.Fatalf("Snapshot get failed")
	}
	fmt.Println("  ✓ Snapshot operations successful")
}

func testTreeWithStorage() {
	fmt.Println("\n2. Testing Tree with Storage Abstraction:")

	// Create storage
	store := memory.NewStorage()
	defer store.Close()

	// Create key encoder
	keyEncoder := storage.NewDefaultKeyEncoder()

	// Create tree
	jmt, err := tree.NewTree(store, keyEncoder, tree.DefaultTreeConfig())
	if err != nil {
		log.Fatalf("Failed to create tree: %v", err)
	}
	fmt.Println("  ✓ Tree created with storage abstraction")

	// Test operations
	testData := map[string]string{
		"alice":   "100",
		"bob":     "200",
		"charlie": "300",
	}

	// Put operations
	for name, balance := range testData {
		key := types.KeyHash([]byte(name))
		value := []byte(balance)
		_, err := jmt.Put(key, value)
		if err != nil {
			log.Fatalf("Failed to put %s: %v", name, err)
		}
	}
	fmt.Println("  ✓ Put operations successful")

	// Get operations
	latestVersion := jmt.GetLatestVersion()
	for name, expectedBalance := range testData {
		key := types.KeyHash([]byte(name))
		value, err := jmt.Get(latestVersion, key)
		if err != nil {
			log.Fatalf("Failed to get %s: %v", name, err)
		}
		if string(value) != expectedBalance {
			log.Fatalf("Value mismatch for %s: got %s, want %s", name, value, expectedBalance)
		}
	}
	fmt.Println("  ✓ Get operations successful")

	// Batch transaction
	batch := jmt.NewBatchTransaction()
	batch.BatchPut(types.KeyHash([]byte("david")), []byte("400"))
	batch.BatchPut(types.KeyHash([]byte("eve")), []byte("500"))
	
	newVersion, err := batch.Execute()
	if err != nil {
		log.Fatalf("Batch execute failed: %v", err)
	}
	fmt.Println("  ✓ Batch transaction successful")

	// Verify batch results
	davidVal, _ := jmt.Get(newVersion, types.KeyHash([]byte("david")))
	eveVal, _ := jmt.Get(newVersion, types.KeyHash([]byte("eve")))
	if string(davidVal) != "400" || string(eveVal) != "500" {
		log.Fatalf("Batch values incorrect")
	}
	fmt.Println("  ✓ Batch values verified")

	// Versioning
	if newVersion != latestVersion+1 {
		log.Fatalf("Version increment incorrect: got %d, want %d", newVersion, latestVersion+1)
	}
	fmt.Println("  ✓ Versioning working correctly")

	// Root hash
	rootHash, err := jmt.GetRootHash(newVersion)
	if err != nil || rootHash == types.EmptyHash() {
		log.Fatalf("Failed to get root hash")
	}
	fmt.Printf("  ✓ Root hash retrieved: %x...\n", rootHash[:8])
}