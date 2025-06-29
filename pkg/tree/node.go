package tree

import (
	"fmt"
	"io"
	"strings"

	"github.com/neutral/proofbox/pkg/types"
)

// NodeType identifies the type of node in the tree
type NodeType byte

const (
	// NodeTypeInternal represents an internal node with children
	NodeTypeInternal NodeType = 0x00
	// NodeTypeLeaf represents a leaf node with key-value data
	NodeTypeLeaf NodeType = 0x01
)

// Child represents a reference to a child node
type Child struct {
	Hash    types.Hash    // Hash of the child node
	Version types.Version // Version when child was created
	IsLeaf  bool          // True if child is a leaf node
}

// IsEmpty returns true if this is an empty child reference
func (c Child) IsEmpty() bool {
	return c.Hash.IsEmpty()
}

// Node represents a node in the Jellyfish Merkle Tree
type Node interface {
	// Type returns the node type (leaf or internal)
	Type() NodeType
	// Hash returns the cryptographic hash of this node
	Hash() types.Hash
	// IsCached returns true if the hash has been computed and cached
	IsCached() bool
	// Version returns the version when this node was created
	Version() types.Version
}

// NodeWithChildren interface for nodes that can have children (read-only)
type NodeWithChildren interface {
	Node
	Child(nibble types.Nibble) (Child, bool)   // Get child at nibble
	Children() map[types.Nibble]Child          // Get all children (defensive copy)
	NumChildren() int                          // Count of non-empty children
	GetOnlyChild() (types.Nibble, Child, bool) // For single-child optimization
}

// NodeWithKey interface for nodes that have a key
type NodeWithKey interface {
	Node
	Key() types.Key
}

// NodeCloneable interface for creating new versions of nodes
type NodeCloneable interface {
	Node
	Clone(newVersion types.Version) Node // Creates new node with different version
}

// IsLeaf returns true if the node is a leaf node
func IsLeaf(n Node) bool {
	return n.Type() == NodeTypeLeaf
}

// IsInternal returns true if the node is an internal node
func IsInternal(n Node) bool {
	return n.Type() == NodeTypeInternal
}

// AsLeaf attempts to convert a Node to LeafNode
func AsLeaf(n Node) (*LeafNode, bool) {
	leaf, ok := n.(*LeafNode)
	return leaf, ok
}

// AsInternal attempts to convert a Node to InternalNode
func AsInternal(n Node) (*InternalNode, bool) {
	internal, ok := n.(*InternalNode)
	return internal, ok
}

// AsLeafErr returns an error if node is not a leaf
func AsLeafErr(n Node) (*LeafNode, error) {
	leaf, ok := n.(*LeafNode)
	if !ok {
		return nil, fmt.Errorf("expected leaf node, got %T", n)
	}
	return leaf, nil
}

// AsInternalErr returns an error if node is not internal
func AsInternalErr(n Node) (*InternalNode, error) {
	internal, ok := n.(*InternalNode)
	if !ok {
		return nil, fmt.Errorf("expected internal node, got %T", n)
	}
	return internal, nil
}

// NodePath represents the path to a node in the tree
type NodePath struct {
	Nibbles []types.Nibble
	Depth   int
}

// NodeLoader defines the interface for loading nodes from storage
type NodeLoader interface {
	LoadNode(hash types.Hash) (Node, error)
}

// KeyToNibblePath converts a key to a nibble path
func KeyToNibblePath(key types.Key) []types.Nibble {
	nibbles := make([]types.Nibble, 0, types.MaxTreeDepth)
	for _, b := range key {
		nibbles = append(nibbles, types.Nibble(b>>4))
		nibbles = append(nibbles, types.Nibble(b&0x0f))
	}
	return nibbles
}

// TraverseToLeaf follows a key path down to a leaf
// NOTE: This is a partial implementation that only handles in-memory nodes.
// Full traversal requires storage integration to load child nodes.
func TraverseToLeaf(root Node, key types.Key) (*LeafNode, NodePath, error) {
	if root == nil {
		return nil, NodePath{}, types.ErrEmptyTree
	}

	nibblePath := KeyToNibblePath(key)
	path := NodePath{
		Nibbles: make([]types.Nibble, 0, types.MaxTreeDepth),
	}

	// If root is already a leaf, check if it matches
	if leaf, ok := root.(*LeafNode); ok {
		path.Depth = 0
		return leaf, path, nil
	}

	// For internal nodes, we can only check immediate children without storage
	internal, ok := root.(*InternalNode)
	if !ok {
		return nil, path, fmt.Errorf("invalid node type")
	}

	if len(nibblePath) == 0 {
		return nil, path, types.ErrInvalidKey
	}

	// Check first nibble only (shallow traversal)
	nibble := nibblePath[0]
	path.Nibbles = append(path.Nibbles, nibble)
	path.Depth = 0

	child, exists := internal.Child(nibble)
	if !exists {
		return nil, path, types.ErrKeyNotFound
	}

	// Without storage, we cannot load the child node to continue traversal
	_ = child
	return nil, path, fmt.Errorf("deeper traversal requires storage integration (child at nibble %d)", nibble)
}

// ComputeRootHash recursively computes the hash of a subtree
func ComputeRootHash(node Node) types.Hash {
	if node == nil {
		return types.EmptyHash()
	}
	return node.Hash()
}

// CountNodes recursively counts nodes in a subtree
func CountNodes(node Node, loader NodeLoader) (int, error) {
	if node == nil {
		return 0, nil
	}

	count := 1 // Count this node

	if internal, ok := node.(*InternalNode); ok {
		children := internal.Children()
		for _, child := range children {
			childNode, err := loader.LoadNode(child.Hash)
			if err != nil {
				return 0, err
			}

			childCount, err := CountNodes(childNode, loader)
			if err != nil {
				return 0, err
			}

			count += childCount
		}
	}

	return count, nil
}

// CloneNode creates a deep copy of any node with a new version
func CloneNode(n Node, newVersion types.Version) (Node, error) {
	if n == nil {
		return nil, nil
	}

	// Check if node implements NodeCloneable
	if cloneable, ok := n.(NodeCloneable); ok {
		return cloneable.Clone(newVersion), nil
	}

	// For now, return an error since Clone methods haven't been implemented
	return nil, fmt.Errorf("node type %T does not implement Clone", n)
}

// NewNode creates a node from its type and data
// This is a placeholder that will be implemented with codec support
func NewNode(nodeType NodeType, data []byte) (Node, error) {
	switch nodeType {
	case NodeTypeLeaf:
		// Placeholder - will use DecodeLeafNode when codec is implemented
		return nil, fmt.Errorf("leaf decoding not yet implemented")

	case NodeTypeInternal:
		// Placeholder - will use DecodeInternalNode when codec is implemented
		return nil, fmt.Errorf("internal decoding not yet implemented")

	default:
		return nil, fmt.Errorf("unknown node type: %d", nodeType)
	}
}

// NodeVisitor defines methods for visiting nodes
type NodeVisitor interface {
	VisitLeaf(*LeafNode) error
	VisitInternal(*InternalNode) error
}

// Accept implements the visitor pattern for nodes
func Accept(node Node, visitor NodeVisitor) error {
	switch n := node.(type) {
	case *LeafNode:
		return visitor.VisitLeaf(n)
	case *InternalNode:
		return visitor.VisitInternal(n)
	default:
		return fmt.Errorf("unknown node type: %T", node)
	}
}

// NodePrinter is an example visitor that prints node information
type NodePrinter struct {
	Writer io.Writer
	Depth  int
}

// VisitLeaf prints information about a leaf node
func (p *NodePrinter) VisitLeaf(leaf *LeafNode) error {
	indent := strings.Repeat("  ", p.Depth)
	_, err := fmt.Fprintf(p.Writer, "%sLeaf: key=%x, value_hash=%x\n",
		indent, leaf.Key(), leaf.ValueHash())
	return err
}

// VisitInternal prints information about an internal node
func (p *NodePrinter) VisitInternal(internal *InternalNode) error {
	indent := strings.Repeat("  ", p.Depth)
	_, err := fmt.Fprintf(p.Writer, "%sInternal: %d children\n",
		indent, internal.NumChildren())
	return err
}

