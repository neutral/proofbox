package tree

import (
	"github.com/neutral/proofbox/pkg/types"
)

// NewLeafNodeFromCodec creates a leaf node from decoded data.
// This is used by the codec package to reconstruct nodes from storage.
func NewLeafNodeFromCodec(key types.Key, valueHash types.Hash, version types.Version) *LeafNode {
	return &LeafNode{
		key:       key,
		valueHash: valueHash,
		version:   version,
		// value is nil - will be loaded separately if needed
	}
}

// NewInternalNodeFromCodec creates an internal node from decoded data.
// This is used by the codec package to reconstruct nodes from storage.
func NewInternalNodeFromCodec(children map[types.Nibble]types.Child, version types.Version) *InternalNode {
	if children == nil {
		children = make(map[types.Nibble]types.Child)
	}
	return &InternalNode{
		children: children,
		version:  version,
		// cachedHash is nil - will be computed on demand
	}
}
