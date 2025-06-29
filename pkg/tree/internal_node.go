package tree

import (
	"sync"

	"github.com/neutral/proofbox/pkg/crypto"
	"github.com/neutral/proofbox/pkg/types"
)

// Compile-time interface compliance checks
var (
	_ Node             = (*InternalNode)(nil)
	_ NodeWithChildren = (*InternalNode)(nil)
	_ NodeCloneable    = (*InternalNode)(nil)
)

// InternalNode represents an internal node with up to 16 children
type InternalNode struct {
	children map[types.Nibble]Child // Sparse array of children
	version  types.Version

	// Cache
	mu         sync.RWMutex
	cachedHash *types.Hash
}

// NewInternalNode creates a new internal node
func NewInternalNode(version types.Version) *InternalNode {
	return &InternalNode{
		children: make(map[types.Nibble]Child),
		version:  version,
	}
}

// Type returns NodeTypeInternal
func (n *InternalNode) Type() NodeType {
	return NodeTypeInternal
}

// Hash computes the internal node hash
// Format: NodeTypeInternal || child0 || child1 || ... || child15
// where each child is either (nibble || hash) or empty_hash
func (n *InternalNode) Hash() types.Hash {
	n.mu.RLock()
	if n.cachedHash != nil {
		defer n.mu.RUnlock()
		return *n.cachedHash
	}
	n.mu.RUnlock()

	// Build parts for hashing
	parts := [][]byte{{byte(NodeTypeInternal)}}

	// Process all 16 possible children in order
	for nibble := types.Nibble(0); nibble <= types.MaxNibbleValue; nibble++ {
		if child, exists := n.children[nibble]; exists {
			// Include nibble and child hash
			parts = append(parts, []byte{byte(nibble)})
			parts = append(parts, child.Hash[:])
		} else {
			// Use empty hash for missing children
			parts = append(parts, crypto.EmptyTreeHash[:])
		}
	}

	hash := crypto.DefaultHasher.HashConcat(parts...)

	// Cache it
	n.mu.Lock()
	n.cachedHash = &hash
	n.mu.Unlock()

	return hash
}

// IsCached returns true if hash is already computed
func (n *InternalNode) IsCached() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.cachedHash != nil
}

// Version returns the version when this node was created
func (n *InternalNode) Version() types.Version {
	return n.version
}

// Child returns the child at the given nibble
func (n *InternalNode) Child(nibble types.Nibble) (Child, bool) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	child, exists := n.children[nibble]
	return child, exists
}

// SetChild sets or updates a child
func (n *InternalNode) SetChild(nibble types.Nibble, child Child) error {
	if err := types.ValidateNibble(nibble); err != nil {
		return err
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	n.children[nibble] = child
	n.cachedHash = nil // Invalidate cache
	return nil
}

// RemoveChild removes a child
func (n *InternalNode) RemoveChild(nibble types.Nibble) {
	n.mu.Lock()
	defer n.mu.Unlock()

	delete(n.children, nibble)
	n.cachedHash = nil // Invalidate cache
}

// NumChildren returns the number of non-empty children
func (n *InternalNode) NumChildren() int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return len(n.children)
}

// Children returns a copy of all children
func (n *InternalNode) Children() map[types.Nibble]Child {
	n.mu.RLock()
	defer n.mu.RUnlock()

	result := make(map[types.Nibble]Child)
	for k, v := range n.children {
		result[k] = v
	}
	return result
}

// GetOnlyChild returns the single child if there's exactly one
func (n *InternalNode) GetOnlyChild() (types.Nibble, Child, bool) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if len(n.children) != 1 {
		return 0, Child{}, false
	}

	for nibble, child := range n.children {
		return nibble, child, true
	}
	return 0, Child{}, false
}

// Clone creates a copy of the internal node with a new version
func (n *InternalNode) Clone(newVersion types.Version) Node {
	n.mu.RLock()
	defer n.mu.RUnlock()

	clone := &InternalNode{
		children: make(map[types.Nibble]Child),
		version:  newVersion,
	}

	for k, v := range n.children {
		clone.children[k] = v
	}

	return clone
}
