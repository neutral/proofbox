package fuzz

import (
	"github.com/neutral/proofbox/pkg/storage"
	memorystorage "github.com/neutral/proofbox/pkg/storage/memory"
	"github.com/neutral/proofbox/pkg/tree"
)

// NewTestTree creates a new tree with memory storage for testing
func NewTestTree() (*tree.Tree, error) {
	store := memorystorage.NewStorage()
	config := tree.DefaultTreeConfig()
	// Disable metrics for testing to avoid duplicate registration
	config.MetricsEnabled = false
	keyEncoder := storage.NewDefaultKeyEncoder()
	return tree.NewTree(store, keyEncoder, config)
}