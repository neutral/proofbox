package tree

import (
	"bytes"
	"testing"

	"github.com/neutral/proofbox/pkg/storage"
	"github.com/neutral/proofbox/pkg/storage/memory"
	"github.com/neutral/proofbox/pkg/types"
)

// TestStorageIntegration verifies that the tree works with the storage abstraction
func TestStorageIntegration(t *testing.T) {
	// Create memory storage
	store := memory.NewStorage()
	defer store.Close()

	// Create key encoder
	keyEncoder := storage.NewDefaultKeyEncoder()

	// Create tree
	tree, err := NewTree(store, keyEncoder, testTreeConfig())
	if err != nil {
		t.Fatalf("Failed to create tree: %v", err)
	}

	// Test basic operations
	key := types.KeyHash([]byte("test-key"))
	value := []byte("test-value")

	// Put
	version, err := tree.Put(key, value)
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}
	if version != 1 {
		t.Fatalf("Expected version 1, got %d", version)
	}

	// Get
	retrieved, err := tree.Get(version, key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !bytes.Equal(retrieved, value) {
		t.Fatalf("Get returned wrong value: got %s, want %s", retrieved, value)
	}

	// Update
	newValue := []byte("updated-value")
	version2, err := tree.Put(key, newValue)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if version2 != 2 {
		t.Fatalf("Expected version 2, got %d", version2)
	}

	// Verify update
	retrieved2, err := tree.Get(version2, key)
	if err != nil {
		t.Fatalf("Get after update failed: %v", err)
	}
	if !bytes.Equal(retrieved2, newValue) {
		t.Fatalf("Get returned wrong value after update: got %s, want %s", retrieved2, newValue)
	}

	// Verify old version still accessible
	retrievedOld, err := tree.Get(version, key)
	if err != nil {
		t.Fatalf("Get old version failed: %v", err)
	}
	if !bytes.Equal(retrievedOld, value) {
		t.Fatalf("Old version returned wrong value: got %s, want %s", retrievedOld, value)
	}
}

// TestBatchWithStorage tests batch operations with storage abstraction
func TestBatchWithStorage(t *testing.T) {
	// Create memory storage
	store := memory.NewStorage()
	defer store.Close()

	// Create key encoder
	keyEncoder := storage.NewDefaultKeyEncoder()

	// Create tree
	tree, err := NewTree(store, keyEncoder, testTreeConfig())
	if err != nil {
		t.Fatalf("Failed to create tree: %v", err)
	}

	// Create batch
	batch := tree.NewBatchTransaction()

	// Add operations
	batch.BatchPut(types.KeyHash([]byte("key1")), []byte("value1"))
	batch.BatchPut(types.KeyHash([]byte("key2")), []byte("value2"))
	batch.BatchPut(types.KeyHash([]byte("key3")), []byte("value3"))

	// Execute batch
	version, err := batch.Execute()
	if err != nil {
		t.Fatalf("Batch execute failed: %v", err)
	}

	// Verify all keys
	for i := 1; i <= 3; i++ {
		val, err := tree.Get(version, types.KeyHash([]byte("key"+string(rune('0'+i)))))
		if err != nil {
			t.Fatalf("Get key%d failed: %v", i, err)
		}
		if string(val) != "value"+string(rune('0'+i)) {
			t.Fatalf("Wrong value for key%d: got %s, want value%d", i, val, i)
		}
	}
}
