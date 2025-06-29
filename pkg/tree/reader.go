package tree

import (
	"fmt"

	"github.com/cockroachdb/pebble"
	"github.com/neutral/proofbox/pkg/codec"
	"github.com/neutral/proofbox/pkg/types"
)

// TreeReader handles read operations with a snapshot
type TreeReader struct {
	tree     *Tree
	snapshot *pebble.Snapshot
	version  types.Version
	rootHash types.Hash
}

// Get retrieves a value by key at the specified version
func (t *Tree) Get(version types.Version, key types.Key) ([]byte, error) {
	// Validate inputs
	if err := t.validateVersion(version); err != nil {
		return nil, err
	}
	if err := t.validateKey(key); err != nil {
		return nil, err
	}

	// Check version exists
	t.mu.RLock()
	rootHash, exists := t.rootHashes[version]
	t.mu.RUnlock()

	if !exists {
		return nil, types.ErrVersionNotFound
	}

	// Empty tree case
	if rootHash == types.EmptyHash() {
		return nil, nil // Key not found in empty tree
	}

	// Create snapshot for consistent read
	snapshot := t.db.NewSnapshot()
	defer snapshot.Close()

	// Create reader with snapshot
	reader := &TreeReader{
		tree:     t,
		snapshot: snapshot,
		version:  version,
		rootHash: rootHash,
	}

	return reader.Get(key)
}

// Get retrieves a value by traversing the tree
func (r *TreeReader) Get(key types.Key) ([]byte, error) {
	// Load root node
	root, err := r.loadNode(types.RootNodeKey(r.version))
	if err != nil {
		return nil, err
	}

	// Empty tree
	if root == nil {
		return nil, nil
	}

	// Traverse to leaf
	nibblePath := key.ToNibblePath()
	current := root

	for depth := 0; depth < types.MaxTreeDepth; depth++ {
		switch node := current.(type) {
		case *LeafNode:
			// Found a leaf - check if it's our key
			if node.Key() == key {
				// Load actual value using hash
				return r.loadValue(node.ValueHash())
			}
			return nil, nil // Different key, not found

		case *InternalNode:
			// Follow the child for this nibble
			nibble := nibblePath.Nibbles[depth]
			child, exists := node.Child(nibble)
			if !exists {
				return nil, nil // Path doesn't exist
			}

			// Load child node
			childKey := types.NodeKey{
				Version: child.Version,
				NibblePath: types.NibblePath{
					Nibbles: nibblePath.Nibbles[:depth+1],
					Length:  uint16(depth + 1),
				},
			}

			next, err := r.loadNode(childKey)
			if err != nil {
				return nil, err
			}
			if next == nil {
				return nil, nil // Child doesn't exist
			}

			current = next

		default:
			return nil, fmt.Errorf("unknown node type: %T", node)
		}
	}

	return nil, types.ErrMaxDepthExceeded
}

// loadNode loads a node from storage or cache
func (r *TreeReader) loadNode(key types.NodeKey) (types.Node, error) {
	// Check cache first
	if node, found := r.tree.nodeCache.Get(key); found {
		return node, nil
	}

	// Load from storage
	storageKey := makeNodeKey(key)
	data, closer, err := r.snapshot.Get(storageKey)
	if err == pebble.ErrNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load node: %w", err)
	}
	defer closer.Close()

	// Copy data before closing
	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)

	// Decode node
	node, err := codec.DecodeNode(dataCopy, key.Version)
	if err != nil {
		return nil, fmt.Errorf("failed to decode node: %w", err)
	}

	// Update cache
	r.tree.nodeCache.Put(key, node)

	return node, nil
}

// loadValue loads a value by its hash
func (r *TreeReader) loadValue(hash types.Hash) ([]byte, error) {
	valueKey := makeValueKey(r.version, hash)
	data, closer, err := r.snapshot.Get(valueKey)
	if err != nil {
		return nil, fmt.Errorf("failed to load value: %w", err)
	}
	defer closer.Close()

	// Copy data before closing
	result := make([]byte, len(data))
	copy(result, data)

	return result, nil
}

// validateVersion checks if a version is valid
func (t *Tree) validateVersion(version types.Version) error {
	if version > types.MaxVersion {
		return types.ErrInvalidVersionRange
	}
	return nil
}

// validateKey checks if a key is valid
func (t *Tree) validateKey(key types.Key) error {
	// Keys are always 32 bytes, so just check for zero key
	if key == (types.Key{}) {
		return types.ErrEmptyKey
	}
	return nil
}