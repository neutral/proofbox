package codec

import (
	"encoding/binary"
	"errors"
	"fmt"
	"sort"

	"github.com/neutral/proofbox/pkg/types"
)

// InternalNode binary format:
// [NumChildren (1 byte)] [Children data...]
//
// Each child:
// [Nibble (1 byte)] [Hash (32 bytes)] [Version (8 bytes)] [IsLeaf (1 byte)]
// Total per child: 42 bytes

// EncodeInternalNode serializes an internal node to bytes
func EncodeInternalNode(node types.InternalNodeInterface) ([]byte, error) {
	if node == nil {
		return nil, nil
	}

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
	sort.Slice(nibbles, func(i, j int) bool {
		return nibbles[i] < nibbles[j]
	})

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
		binary.BigEndian.PutUint64(buf[offset:offset+8], uint64(child.Version))
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

// DecodeInternalNode deserializes an internal node from bytes
func DecodeInternalNode(data []byte, version types.Version) (types.InternalNodeInterface, error) {
	if len(data) < 1 {
		return nil, errors.New("empty internal node data")
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

		// Read version
		childVersion := types.Version(binary.BigEndian.Uint64(data[offset : offset+8]))
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

	// Return decoded data - actual node creation handled by factory
	return nil, fmt.Errorf("use codec.DecodeNode for node creation")
}

// InternalNodeSize calculates the serialized size of an internal node
func InternalNodeSize(node types.InternalNodeInterface) int {
	return 1 + node.NumChildren()*42
}
