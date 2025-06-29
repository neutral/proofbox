package codec

import (
	"errors"
	"fmt"

	"github.com/neutral/proofbox/pkg/tree"
	"github.com/neutral/proofbox/pkg/types"
)

// NodeCodec provides encoding/decoding for any node type
type NodeCodec struct{}

// EncodeNode serializes any node with type prefix
func (c *NodeCodec) EncodeNode(node tree.Node) ([]byte, error) {
	if node == nil {
		return nil, errors.New("cannot encode nil node")
	}

	switch n := node.(type) {
	case *tree.LeafNode:
		data := EncodeLeafNode(n)
		// Prepend type byte
		result := make([]byte, 1+len(data))
		result[0] = byte(tree.NodeTypeLeaf)
		copy(result[1:], data)
		return result, nil

	case *tree.InternalNode:
		data, err := EncodeInternalNode(n)
		if err != nil {
			return nil, err
		}
		// Prepend type byte
		result := make([]byte, 1+len(data))
		result[0] = byte(tree.NodeTypeInternal)
		copy(result[1:], data)
		return result, nil

	default:
		return nil, fmt.Errorf("unknown node type: %T", node)
	}
}

// DecodeNode deserializes a node from bytes
func (c *NodeCodec) DecodeNode(data []byte, version types.Version) (tree.Node, error) {
	if len(data) < 1 {
		return nil, errors.New("empty node data")
	}

	nodeType := tree.NodeType(data[0])
	nodeData := data[1:]

	switch nodeType {
	case tree.NodeTypeLeaf:
		leaf, err := DecodeLeafNode(nodeData, version)
		if err != nil {
			return nil, err
		}
		return leaf, nil

	case tree.NodeTypeInternal:
		internal, err := DecodeInternalNode(nodeData, version)
		if err != nil {
			return nil, err
		}
		return internal, nil

	default:
		return nil, fmt.Errorf("unknown node type: %d", nodeType)
	}
}

// EstimateSize returns the estimated serialized size of a node
func (c *NodeCodec) EstimateSize(node tree.Node) int {
	if node == nil {
		return 0
	}

	// 1 byte for type prefix
	switch n := node.(type) {
	case *tree.LeafNode:
		return 1 + LeafNodeSize()
	case *tree.InternalNode:
		return 1 + InternalNodeSize(n)
	default:
		return 0
	}
}

// EncodeNodeWithKey encodes a node along with its key for storage
func (c *NodeCodec) EncodeNodeWithKey(key types.NodeKey, node tree.Node) ([]byte, []byte, error) {
	// Encode the key
	keyData := types.EncodeNodeKey(key)

	// Encode the node
	nodeData, err := c.EncodeNode(node)
	if err != nil {
		return nil, nil, err
	}

	return keyData, nodeData, nil
}

