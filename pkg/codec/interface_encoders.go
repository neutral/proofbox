package codec

import (
	"fmt"

	"github.com/neutral/proofbox/pkg/types"
)

// EncodeLeafNodeInterface encodes a leaf node using only the interface
func EncodeLeafNodeInterface(leaf types.LeafNodeInterface) []byte {
	buf := make([]byte, 64)

	// Copy key (32 bytes)
	key := leaf.Key()
	copy(buf[0:32], key[:])

	// Copy value hash (32 bytes)
	valueHash := leaf.ValueHash()
	copy(buf[32:64], valueHash[:])

	return buf
}

// EncodeInternalNodeInterface encodes an internal node using only the interface
func EncodeInternalNodeInterface(node types.InternalNodeInterface) ([]byte, error) {
	numChildren := node.NumChildren()
	if numChildren > 16 {
		return nil, fmt.Errorf("invalid internal node: too many children (%d > 16)", numChildren)
	}

	// Calculate buffer size
	bufSize := 1 + numChildren*42
	buf := make([]byte, bufSize)

	// Write number of children
	buf[0] = byte(numChildren)

	// Get sorted nibbles for deterministic encoding
	children := node.Children()
	nibbles := make([]types.Nibble, 0, len(children))
	for nibble := range children {
		nibbles = append(nibbles, nibble)
	}
	// Sort nibbles
	for i := 0; i < len(nibbles)-1; i++ {
		for j := i + 1; j < len(nibbles); j++ {
			if nibbles[i] > nibbles[j] {
				nibbles[i], nibbles[j] = nibbles[j], nibbles[i]
			}
		}
	}

	// Write each child
	offset := 1
	for _, nibble := range nibbles {
		child := children[nibble]

		// Write nibble (1 byte)
		buf[offset] = byte(nibble)
		offset++

		// Write hash (32 bytes)
		copy(buf[offset:offset+32], child.Hash[:])
		offset += 32

		// Write version (8 bytes, big-endian)
		for i := 7; i >= 0; i-- {
			buf[offset+i] = byte(child.Version)
			child.Version >>= 8
		}
		offset += 8

		// Write is_leaf flag (1 byte)
		if child.IsLeaf {
			buf[offset] = 1
		} else {
			buf[offset] = 0
		}
		offset++
	}

	return buf[:offset], nil
}

// InternalNodeInterfaceSize calculates size for an internal node interface
func InternalNodeInterfaceSize(node types.InternalNodeInterface) int {
	return 1 + node.NumChildren()*42
}
