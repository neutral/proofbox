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
	fmt.Println("Storage Abstraction Demo")
	fmt.Println("=======================")

	// Create memory storage
	store := memory.NewStorage()
	defer store.Close()

	// Create key encoder
	keyEncoder := storage.NewDefaultKeyEncoder()

	// Create tree with memory storage
	jmt, err := tree.NewTree(store, keyEncoder, tree.DefaultTreeConfig())
	if err != nil {
		log.Fatalf("Failed to create tree: %v", err)
	}

	fmt.Println("\nUsing memory storage backend")

	// Add some data
	fmt.Println("\nAdding data:")
	data := map[string]string{
		"alice":   "100",
		"bob":     "200",
		"charlie": "300",
	}

	for k, v := range data {
		key := types.KeyHash([]byte(k))
		value := []byte(v)
		version, err := jmt.Put(key, value)
		if err != nil {
			log.Fatalf("Failed to put %s: %v", k, err)
		}
		fmt.Printf("  %s = %s (version %d)\n", k, v, version)
	}

	// Read data back
	latestVersion := jmt.GetLatestVersion()
	fmt.Printf("\nReading data at version %d:\n", latestVersion)

	for k := range data {
		key := types.KeyHash([]byte(k))
		value, err := jmt.Get(latestVersion, key)
		if err != nil {
			log.Fatalf("Failed to get %s: %v", k, err)
		}
		fmt.Printf("  %s = %s\n", k, value)
	}

	// Demonstrate versioning
	fmt.Println("\nUpdating alice's balance:")
	key := types.KeyHash([]byte("alice"))
	newVersion, err := jmt.Put(key, []byte("150"))
	if err != nil {
		log.Fatalf("Failed to update alice: %v", err)
	}
	fmt.Printf("  alice = 150 (version %d)\n", newVersion)

	// Show both versions exist
	fmt.Println("\nVersion history:")
	oldValue, _ := jmt.Get(latestVersion, key)
	newValue, _ := jmt.Get(newVersion, key)
	fmt.Printf("  Version %d: alice = %s\n", latestVersion, oldValue)
	fmt.Printf("  Version %d: alice = %s\n", newVersion, newValue)

	// Get root hashes
	rootHash1, _ := jmt.GetRootHash(latestVersion)
	rootHash2, _ := jmt.GetRootHash(newVersion)
	fmt.Printf("\nRoot hashes:\n")
	fmt.Printf("  Version %d: %x\n", latestVersion, rootHash1[:8])
	fmt.Printf("  Version %d: %x\n", newVersion, rootHash2[:8])

	fmt.Println("\nStorage abstraction working successfully!")
}