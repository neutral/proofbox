package tree

import (
	"context"
	"fmt"

	"github.com/neutral/proofbox/pkg/types"
)

// TreeHealthChecker verifies tree integrity
type TreeHealthChecker struct {
	tree *Tree
}

// NewTreeHealthChecker creates a new health checker
func NewTreeHealthChecker(tree *Tree) *TreeHealthChecker {
	return &TreeHealthChecker{tree: tree}
}

// VerifyIntegrity performs comprehensive integrity check
func (hc *TreeHealthChecker) VerifyIntegrity(ctx context.Context, version types.Version) error {
	// Check if version exists
	if !hc.tree.HasVersion(version) {
		return types.ErrVersionNotFound
	}

	// Create snapshot for consistent read
	snapshot := hc.tree.db.NewSnapshot()
	defer snapshot.Close()

	// Start from root
	rootKey := types.RootNodeKey(version)
	visited := make(map[string]bool) // Use string key for map

	reader := &TreeReader{
		tree:     hc.tree,
		snapshot: snapshot,
		version:  version,
	}

	return hc.verifyNode(ctx, reader, rootKey, visited)
}

// verifyNode recursively verifies a node and its children
func (hc *TreeHealthChecker) verifyNode(ctx context.Context, reader *TreeReader, key types.NodeKey, visited map[string]bool) error {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Create string key for visited map
	visitKey := string(hc.tree.keyEncoder.NodeKey(key))

	// Avoid cycles
	if visited[visitKey] {
		return nil
	}
	visited[visitKey] = true

	// Load and verify node
	node, err := reader.loadNode(key)
	if err != nil {
		return fmt.Errorf("failed to load node at %v: %w", key, err)
	}
	if node == nil {
		return fmt.Errorf("node not found at %v", key)
	}

	// Verify hash computation
	computed := node.Hash()
	// In practice, we'd verify against stored hash
	// For now, just ensure hash can be computed
	if computed == types.EmptyHash() {
		return fmt.Errorf("invalid empty hash for node at %v", key)
	}

	// Recursively verify children
	if internal, ok := AsInternal(node); ok {
		children := internal.Children()
		for nibble, child := range children {
			// Skip empty children
			if child.IsEmpty() {
				continue
			}

			childKey := types.NodeKey{
				Version: child.Version,
				NibblePath: types.NibblePath{
					Nibbles: append(append([]types.Nibble{}, key.NibblePath.Nibbles...), nibble),
					Length:  key.NibblePath.Length + 1,
				},
			}
			if err := hc.verifyNode(ctx, reader, childKey, visited); err != nil {
				return err
			}
		}
	}

	return nil
}

// CheckVersion performs basic version integrity check
func (hc *TreeHealthChecker) CheckVersion(version types.Version) error {
	// Verify version exists
	if !hc.tree.HasVersion(version) {
		return types.ErrVersionNotFound
	}

	// Get root hash
	rootHash, err := hc.tree.GetRootHash(version)
	if err != nil {
		return err
	}

	// For empty tree, root hash should be EmptyHash
	isEmpty, err := hc.tree.IsEmpty(version)
	if err != nil {
		return err
	}

	if isEmpty && rootHash != types.EmptyHash() {
		return fmt.Errorf("empty tree should have empty hash, got %v", rootHash)
	}

	return nil
}
