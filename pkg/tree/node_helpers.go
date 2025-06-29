package tree

import (
	"fmt"

	"github.com/neutral/proofbox/pkg/types"
)

// AsLeaf attempts to convert a Node to LeafNode
func AsLeaf(n types.Node) (*LeafNode, bool) {
	leaf, ok := n.(*LeafNode)
	return leaf, ok
}

// AsInternal attempts to convert a Node to InternalNode
func AsInternal(n types.Node) (*InternalNode, bool) {
	internal, ok := n.(*InternalNode)
	return internal, ok
}

// AsLeafErr returns an error if node is not a leaf
func AsLeafErr(n types.Node) (*LeafNode, error) {
	leaf, ok := n.(*LeafNode)
	if !ok {
		return nil, fmt.Errorf("expected leaf node, got %T", n)
	}
	return leaf, nil
}

// AsInternalErr returns an error if node is not internal
func AsInternalErr(n types.Node) (*InternalNode, error) {
	internal, ok := n.(*InternalNode)
	if !ok {
		return nil, fmt.Errorf("expected internal node, got %T", n)
	}
	return internal, nil
}

// PrintNode prints a node's structure for debugging
func PrintNode(n types.Node, indent string) {
	if n == nil {
		fmt.Printf("%s<nil>\n", indent)
		return
	}

	switch node := n.(type) {
	case *LeafNode:
		fmt.Printf("%sLeaf[v%d]: key=%x, hash=%x\n",
			indent, node.Version(), node.Key(), node.Hash())
	case *InternalNode:
		fmt.Printf("%sInternal[v%d]: %d children, hash=%x\n",
			indent, node.Version(), node.NumChildren(), node.Hash())
		children := node.Children()
		for nibble, child := range children {
			fmt.Printf("%s  [%x] -> v%d %x (leaf=%v)\n",
				indent, nibble, child.Version, child.Hash, child.IsLeaf)
		}
	default:
		fmt.Printf("%sUnknown node type: %T\n", indent, n)
	}
}