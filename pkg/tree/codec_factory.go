package tree

import (
	"fmt"

	"github.com/neutral/proofbox/pkg/codec"
	"github.com/neutral/proofbox/pkg/types"
)

// init registers the node factory with the codec package
func init() {
	codec.RegisterNodeFactory(nodeFactory)
}

// nodeFactory creates concrete node types from decoded data
func nodeFactory(nodeType types.NodeType, data []byte, version types.Version) (types.Node, error) {
	switch nodeType {
	case types.NodeTypeLeaf:
		return decodeLeafNode(data, version)
	case types.NodeTypeInternal:
		return decodeInternalNode(data, version)
	default:
		return nil, fmt.Errorf("unknown node type: %d", nodeType)
	}
}

// decodeLeafNode creates a LeafNode from decoded data
func decodeLeafNode(data []byte, version types.Version) (*LeafNode, error) {
	if len(data) != 64 {
		return nil, fmt.Errorf("invalid leaf data size: %d, expected 64", len(data))
	}

	// Decode key
	var key types.Key
	copy(key[:], data[0:32])

	// Decode value hash
	var valueHash types.Hash
	copy(valueHash[:], data[32:64])

	// Use factory function to create node
	return NewLeafNodeFromCodec(key, valueHash, version), nil
}

// decodeInternalNode creates an InternalNode from decoded data
func decodeInternalNode(data []byte, version types.Version) (*InternalNode, error) {
	if len(data) < 1 {
		return nil, fmt.Errorf("empty internal node data")
	}

	// Read number of children
	numChildren := int(data[0])
	if numChildren > 16 {
		return nil, fmt.Errorf("too many children: %d", numChildren)
	}

	// Verify data size
	expectedSize := 1 + numChildren*42
	if len(data) != expectedSize {
		return nil, fmt.Errorf("invalid data size: %d, expected %d", len(data), expectedSize)
	}

	// Create children map
	children := make(map[types.Nibble]types.Child, numChildren)

	// Read each child
	offset := 1
	for i := 0; i < numChildren; i++ {
		// Read nibble
		nibble := types.Nibble(data[offset])
		if nibble > 15 {
			return nil, fmt.Errorf("invalid nibble: %d", nibble)
		}
		offset++

		// Read hash
		var hash types.Hash
		copy(hash[:], data[offset:offset+32])
		offset += 32

		// Read version (big-endian)
		childVersion := types.Version(0)
		for j := 0; j < 8; j++ {
			childVersion = (childVersion << 8) | types.Version(data[offset+j])
		}
		offset += 8

		// Read is_leaf flag
		isLeaf := data[offset] == 1
		offset++

		// Store child
		children[nibble] = types.Child{
			Hash:    hash,
			Version: childVersion,
			IsLeaf:  isLeaf,
		}
	}

	// Use factory function to create node
	return NewInternalNodeFromCodec(children, version), nil
}
