package codec

import (
	"errors"
	"fmt"

	"github.com/neutral/proofbox/pkg/types"
)

// NodeCodec provides encoding/decoding for any node type
type NodeCodec struct{}

// EncodeNode serializes any node with type prefix
func (c *NodeCodec) EncodeNode(node types.Node) ([]byte, error) {
	if node == nil {
		return nil, errors.New("cannot encode nil node")
	}

	switch node.Type() {
	case types.NodeTypeLeaf:
		leaf, ok := node.(types.LeafNodeInterface)
		if !ok {
			return nil, fmt.Errorf("node reports leaf type but doesn't implement LeafNodeInterface")
		}
		data := EncodeLeafNodeInterface(leaf)
		// Prepend type byte
		result := make([]byte, 1+len(data))
		result[0] = byte(types.NodeTypeLeaf)
		copy(result[1:], data)
		return result, nil

	case types.NodeTypeInternal:
		internal, ok := node.(types.InternalNodeInterface)
		if !ok {
			return nil, fmt.Errorf("node reports internal type but doesn't implement InternalNodeInterface")
		}
		data, err := EncodeInternalNodeInterface(internal)
		if err != nil {
			return nil, err
		}
		// Prepend type byte
		result := make([]byte, 1+len(data))
		result[0] = byte(types.NodeTypeInternal)
		copy(result[1:], data)
		return result, nil

	default:
		return nil, fmt.Errorf("unknown node type: %T", node)
	}
}

// DecodeNode deserializes a node from bytes
func (c *NodeCodec) DecodeNode(data []byte, version types.Version) (types.Node, error) {
	if len(data) < 1 {
		return nil, errors.New("empty node data")
	}

	nodeType := types.NodeType(data[0])

	switch nodeType {
	case types.NodeTypeLeaf:
		// Need tree package for concrete types - will be handled by factory
		return nil, fmt.Errorf("leaf node decoding requires tree package factory")

	case types.NodeTypeInternal:
		// Need tree package for concrete types - will be handled by factory
		return nil, fmt.Errorf("internal node decoding requires tree package factory")

	default:
		return nil, fmt.Errorf("unknown node type: %d", nodeType)
	}
}

// EstimateSize returns the estimated serialized size of a node
func (c *NodeCodec) EstimateSize(node types.Node) int {
	if node == nil {
		return 0
	}

	// 1 byte for type prefix
	switch node.Type() {
	case types.NodeTypeLeaf:
		return 1 + LeafNodeSize()
	case types.NodeTypeInternal:
		if internal, ok := node.(types.InternalNodeInterface); ok {
			return 1 + InternalNodeInterfaceSize(internal)
		}
		return 0
	default:
		return 0
	}
}

// EncodeNodeWithKey encodes a node along with its key for storage
func (c *NodeCodec) EncodeNodeWithKey(key types.NodeKey, node types.Node) ([]byte, []byte, error) {
	// Encode the key
	keyData := types.EncodeNodeKey(key)

	// Encode the node
	nodeData, err := c.EncodeNode(node)
	if err != nil {
		return nil, nil, err
	}

	return keyData, nodeData, nil
}
