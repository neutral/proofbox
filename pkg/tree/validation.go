package tree

import (
	"fmt"

	"github.com/neutral/proofbox/pkg/types"
)

// ValidateTreeStructure ensures the tree maintains proper invariants
func ValidateTreeStructure(tree *Tree, version types.Version) error {
	rootHash, err := tree.GetRootHash(version)
	if err != nil {
		return fmt.Errorf("failed to get root hash: %w", err)
	}

	if rootHash == types.EmptyHash() {
		return nil // Empty tree is valid
	}

	// Create snapshot for consistent read
	snapshot := tree.db.NewSnapshot()
	defer snapshot.Close()

	// Create reader with snapshot
	reader := &TreeReader{
		tree:     tree,
		snapshot: snapshot,
		version:  version,
		rootHash: rootHash,
	}

	root, err := reader.loadNode(types.RootNodeKey(version))
	if err != nil {
		return fmt.Errorf("failed to load root: %w", err)
	}

	visited := make(map[string]bool)
	return validateNode(reader, root, version, types.NibblePath{}, visited)
}

func validateNode(reader *TreeReader, node types.Node, version types.Version,
	path types.NibblePath, visited map[string]bool) error {

	nodeKey := types.NodeKey{Version: version, NibblePath: path}
	nodeKeyStr := nodeKey.String()
	if visited[nodeKeyStr] {
		return fmt.Errorf("cycle detected at %v", nodeKey)
	}
	visited[nodeKeyStr] = true

	switch n := node.(type) {
	case *LeafNode:
		// Verify leaf's key starts with the tree position path
		key := n.Key()
		keyPath := types.NewNibblePath(key[:])
		// The leaf should be stored at a position that is a prefix of its full key
		if !path.IsPrefix(keyPath) {
			return fmt.Errorf("leaf path mismatch: position %v is not a prefix of key %v", path, keyPath)
		}

	case *InternalNode:
		// Check depth constraint - internal nodes can only exist at depths 0-63
		if path.Length >= types.MaxTreeDepth {
			return fmt.Errorf("internal node at depth %d exceeds maximum allowed depth %d", path.Length, types.MaxTreeDepth-1)
		}

		// Check minimum children
		numChildren := n.NumChildren()
		// In a radix tree, internal nodes can have a single child during intermediate states
		// The important invariant is that they have at least one child
		if numChildren < 1 {
			return fmt.Errorf("internal node at %v has no children", path)
		}

		// Validate all children
		for nibble := types.Nibble(0); nibble <= 15; nibble++ {
			if child, exists := n.Child(nibble); exists {
				childPath := path.Append(nibble)
				childKey := types.NodeKey{
					Version:    child.Version,
					NibblePath: childPath,
				}

				childNode, err := reader.loadNode(childKey)
				if err != nil {
					return fmt.Errorf("failed to load child at %v: %w", childPath, err)
				}

				if err := validateNode(reader, childNode, child.Version, childPath, visited); err != nil {
					return err
				}
			}
		}
	}

	return nil
}
