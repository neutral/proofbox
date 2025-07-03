package tree

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/neutral/proofbox/pkg/metrics"
	"github.com/neutral/proofbox/pkg/storage"
	"github.com/neutral/proofbox/pkg/storage/memory"
	pebblestorage "github.com/neutral/proofbox/pkg/storage/pebble"
)

// CreateTestTree creates a tree with memory storage for testing
func CreateTestTree(t *testing.T) *Tree {
	t.Helper()

	store := memory.NewStorage()
	keyEncoder := storage.NewDefaultKeyEncoder()

	// Use test configuration with metrics disabled
	config := DefaultTreeConfig()
	config.MetricsEnabled = false
	config.Metrics = metrics.NoOpMetrics{}

	tree, err := NewTree(store, keyEncoder, config)
	if err != nil {
		t.Fatalf("Failed to create test tree: %v", err)
	}

	t.Cleanup(func() {
		store.Close()
	})

	return tree
}

// CreateTestTreeWithConfig creates a tree with memory storage and custom config
func CreateTestTreeWithConfig(t *testing.T, config TreeConfig) *Tree {
	t.Helper()

	store := memory.NewStorage()
	keyEncoder := storage.NewDefaultKeyEncoder()
	tree, err := NewTree(store, keyEncoder, config)
	if err != nil {
		t.Fatalf("Failed to create test tree with config: %v", err)
	}

	t.Cleanup(func() {
		store.Close()
	})

	return tree
}

// CreatePersistentTestTree creates a tree with PebbleDB storage for persistence tests
func CreatePersistentTestTree(t *testing.T) (tree *Tree, tmpDir string, cleanup func()) {
	t.Helper()

	tmpDir = filepath.Join(os.TempDir(), "proofbox-test-"+t.Name())
	os.RemoveAll(tmpDir) // Clean up any existing directory

	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create PebbleDB storage using the proper driver
	opts := &pebblestorage.Options{
		EnableMetrics: false, // Disable metrics for tests
	}
	store, err := pebblestorage.NewStorage(tmpDir, opts)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create PebbleDB storage: %v", err)
	}

	keyEncoder := storage.NewDefaultKeyEncoder()

	tree, err = NewTree(store, keyEncoder, DefaultTreeConfig())
	if err != nil {
		store.Close()
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create persistent test tree: %v", err)
	}

	cleanup = func() {
		store.Close()
		os.RemoveAll(tmpDir)
	}

	return tree, tmpDir, cleanup
}
