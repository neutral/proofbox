package types

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
	Hash    Hash    // Hash of the child node
	Version Version // Version when child was created
	IsLeaf  bool    // True if child is a leaf node
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
	Hash() Hash
	// IsCached returns true if the hash has been computed and cached
	IsCached() bool
	// Version returns the version when this node was created
	Version() Version
}

// NodeWithChildren interface for nodes that can have children (read-only)
type NodeWithChildren interface {
	Node
	Child(nibble Nibble) (Child, bool)   // Get child at nibble
	Children() map[Nibble]Child          // Get all children (defensive copy)
	NumChildren() int                    // Count of non-empty children
	GetOnlyChild() (Nibble, Child, bool) // For single-child optimization
}

// NodeWithKey interface for nodes that have a key
type NodeWithKey interface {
	Node
	Key() Key
}

// NodeCloneable interface for creating new versions of nodes
type NodeCloneable interface {
	Node
	Clone(newVersion Version) Node // Creates new node with different version
}

// LeafNodeInterface represents the interface for leaf nodes
type LeafNodeInterface interface {
	Node
	NodeWithKey
	NodeCloneable
	ValueHash() Hash
	Value() []byte
	SetValue(value []byte) error
}

// InternalNodeInterface represents the interface for internal nodes
type InternalNodeInterface interface {
	Node
	NodeWithChildren
	NodeCloneable
	SetChild(nibble Nibble, child Child) error
	RemoveChild(nibble Nibble) error
}

// NodeVisitor defines the interface for visiting nodes
type NodeVisitor interface {
	VisitLeaf(leaf LeafNodeInterface) error
	VisitInternal(internal InternalNodeInterface) error
}

// IsLeaf returns true if the node is a leaf node
func IsLeaf(n Node) bool {
	return n.Type() == NodeTypeLeaf
}

// IsInternal returns true if the node is an internal node
func IsInternal(n Node) bool {
	return n.Type() == NodeTypeInternal
}