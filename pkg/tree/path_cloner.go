package tree

import (
	"fmt"

	"github.com/neutral/proofbox/pkg/types"
)

// PathCloner handles efficient node copying for new versions
type PathCloner struct {
	tree          *Tree
	sourceVersion types.Version
	targetVersion types.Version
	clonedNodes   map[string]types.NodeKey // Maps old storage key to new key
}

// NewPathCloner creates a new path cloner
func NewPathCloner(tree *Tree, sourceVersion, targetVersion types.Version) *PathCloner {
	return &PathCloner{
		tree:          tree,
		sourceVersion: sourceVersion,
		targetVersion: targetVersion,
		clonedNodes:   make(map[string]types.NodeKey),
	}
}

// RegisterClone records that a node was cloned
func (pc *PathCloner) RegisterClone(oldKey, newKey types.NodeKey) {
	pc.clonedNodes[string(oldKey.StorageKey())] = newKey
}

// IsCloned checks if a node has already been cloned
func (pc *PathCloner) IsCloned(oldKey types.NodeKey) (types.NodeKey, bool) {
	newKey, exists := pc.clonedNodes[string(oldKey.StorageKey())]
	return newKey, exists
}

// ClonePath copies nodes along a path for the new version
func (pc *PathCloner) ClonePath(path []types.NodeKey) error {
	// Clone from leaf to root
	for i := len(path) - 1; i >= 0; i-- {
		oldKey := path[i]

		// Check if already cloned
		if _, cloned := pc.IsCloned(oldKey); cloned {
			continue
		}

		// Load node
		node, err := pc.tree.loadNodeFromStorage(oldKey)
		if err != nil {
			return err
		}

		// Clone node with new version
		cloneable, ok := node.(types.NodeCloneable)
		if !ok {
			return fmt.Errorf("node does not support cloning")
		}
		clonedNode := cloneable.Clone(pc.targetVersion)

		// Create new key
		newKey := types.NodeKey{
			Version:    pc.targetVersion,
			NibblePath: oldKey.NibblePath,
		}

		// Register the clone
		pc.RegisterClone(oldKey, newKey)

		// Handle internal nodes - update child references
		if internal, ok := clonedNode.(*InternalNode); ok {
			children := internal.Children()
			for nibble, child := range children {
				// Check if child was cloned
				childKey := oldKey.Child(nibble, child.Version)
				if newChildKey, wasCloned := pc.IsCloned(childKey); wasCloned {
					// Update reference to cloned child
					updatedChild := types.Child{
						Hash:    child.Hash, // Hash remains same
						Version: newChildKey.Version,
						IsLeaf:  child.IsLeaf,
					}
					internal.SetChild(nibble, updatedChild)
				}
				// Otherwise child reference remains unchanged (structural sharing)
			}
		}
	}

	return nil
}
