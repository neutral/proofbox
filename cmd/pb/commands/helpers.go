package commands

import (
	"fmt"
	"os"

	"github.com/neutral/proofbox/pkg/storage"
	"github.com/neutral/proofbox/pkg/storage/memory"
	"github.com/neutral/proofbox/pkg/storage/pebble"
	"github.com/neutral/proofbox/pkg/tree"
)

// openDatabase opens the storage backend based on the database path
func openDatabase(path string) (storage.Storage, storage.KeyEncoder, error) {
	// Check if database exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, nil, fmt.Errorf("database not found at %s (run 'pb init' first)", path)
	}

	// Check if it's a memory backend marker
	data, err := os.ReadFile(path)
	if err == nil && string(data) == "memory" {
		store := memory.NewStorage()
		keyEncoder := storage.NewDefaultKeyEncoder()
		return store, keyEncoder, nil
	}

	// Otherwise, assume PebbleDB
	store, err := pebble.NewStorage(path, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open database: %w", err)
	}

	keyEncoder := storage.NewDefaultKeyEncoder()
	return store, keyEncoder, nil
}

// openTree creates a tree instance with the given storage
func openTree(store storage.Storage, keyEncoder storage.KeyEncoder) (*tree.Tree, error) {
	config := tree.DefaultTreeConfig()
	config.MetricsEnabled = false
	
	tree, err := tree.NewTree(store, keyEncoder, config)
	if err != nil {
		return nil, fmt.Errorf("failed to open tree: %w", err)
	}

	return tree, nil
}