package tree

import (
	"github.com/neutral/proofbox/pkg/types"
)

// NodeType identifies the type of node
type NodeType byte

const (
	NodeTypeInternal NodeType = 0x00
	NodeTypeLeaf     NodeType = 0x01
)

// Node represents a node in the Jellyfish Merkle Tree
type Node interface {
	// Type returns the node type
	Type() NodeType

	// Hash computes and returns the node's hash
	Hash() types.Hash

	// IsCached returns true if the hash is already computed
	IsCached() bool

	// Version returns the version when this node was created
	Version() types.Version
}

// Child represents a reference to a child node
type Child struct {
	Hash    types.Hash    // The hash of the child node
	Version types.Version // Version when child was created
	IsLeaf  bool          // Whether the child is a leaf
}

// IsEmpty checks if child reference is empty
func (c Child) IsEmpty() bool {
	return c.Hash == types.EmptyHash()
}

// IsLeaf checks if a node is a leaf
func IsLeaf(n Node) bool {
	return n.Type() == NodeTypeLeaf
}

// IsInternal checks if a node is internal
func IsInternal(n Node) bool {
	return n.Type() == NodeTypeInternal
}

// AsLeaf attempts to cast node to LeafNode
func AsLeaf(n Node) (*LeafNode, bool) {
	leaf, ok := n.(*LeafNode)
	return leaf, ok
}

// AsInternal attempts to cast node to InternalNode
func AsInternal(n Node) (*InternalNode, bool) {
	internal, ok := n.(*InternalNode)
	return internal, ok
}
