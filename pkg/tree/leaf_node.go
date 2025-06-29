package tree

import (
	"sync"

	"github.com/neutral/proofbox/pkg/crypto"
	"github.com/neutral/proofbox/pkg/types"
)

// Compile-time interface compliance checks
var (
	_ types.Node            = (*LeafNode)(nil)
	_ types.NodeWithKey     = (*LeafNode)(nil)
	_ types.NodeCloneable   = (*LeafNode)(nil)
)

// LeafNode represents a leaf in the tree containing actual data
type LeafNode struct {
	key       types.Key  // The key this leaf represents
	valueHash types.Hash // Hash of the value
	value     []byte     // The actual value (may be nil if not loaded)
	version   types.Version

	// Cache
	mu         sync.RWMutex
	cachedHash *types.Hash
}

// NewLeafNode creates a new leaf node
func NewLeafNode(key types.Key, value []byte, version types.Version) (*LeafNode, error) {
	if err := types.ValidateKey(key); err != nil {
		return nil, err
	}

	if len(value) > types.MaxValueSize {
		return nil, types.ErrValueTooLarge
	}

	valueHash := crypto.DefaultHasher.Hash(value)

	return &LeafNode{
		key:       key,
		valueHash: valueHash,
		value:     value,
		version:   version,
	}, nil
}

// Type returns NodeTypeLeaf
func (n *LeafNode) Type() types.NodeType {
	return types.NodeTypeLeaf
}

// Hash computes the leaf node hash
// Format: NodeTypeLeaf || key || valueHash
func (n *LeafNode) Hash() types.Hash {
	n.mu.RLock()
	if n.cachedHash != nil {
		defer n.mu.RUnlock()
		return *n.cachedHash
	}
	n.mu.RUnlock()

	// Compute hash
	hash := crypto.DefaultHasher.HashConcat(
		[]byte{byte(types.NodeTypeLeaf)},
		n.key[:],
		n.valueHash[:],
	)

	// Cache it
	n.mu.Lock()
	n.cachedHash = &hash
	n.mu.Unlock()

	return hash
}

// IsCached returns true if hash is already computed
func (n *LeafNode) IsCached() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.cachedHash != nil
}

// Version returns the version when this leaf was created
func (n *LeafNode) Version() types.Version {
	return n.version
}

// Key returns the leaf's key
func (n *LeafNode) Key() types.Key {
	return n.key
}

// ValueHash returns the hash of the value
func (n *LeafNode) ValueHash() types.Hash {
	return n.valueHash
}

// Value returns the actual value (may be nil if not loaded)
func (n *LeafNode) Value() []byte {
	return n.value
}

// SetValue updates the value (used when loading from storage)
func (n *LeafNode) SetValue(value []byte) error {
	if crypto.DefaultHasher.Hash(value) != n.valueHash {
		return types.ErrHashMismatch
	}
	n.value = value
	return nil
}

// Clone creates a new leaf node with the same data but different version
func (n *LeafNode) Clone(newVersion types.Version) types.Node {
	n.mu.RLock()
	defer n.mu.RUnlock()

	return &LeafNode{
		key:       n.key,
		valueHash: n.valueHash,
		value:     n.value,
		version:   newVersion,
	}
}
