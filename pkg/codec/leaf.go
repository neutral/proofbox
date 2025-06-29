package codec

import (
	"fmt"

	"github.com/neutral/proofbox/pkg/types"
)

// LeafNode binary format:
// [Key (32 bytes)] [ValueHash (32 bytes)]
// Total: 64 bytes (fixed size)

// EncodeLeafNode serializes a leaf node to bytes
func EncodeLeafNode(leaf types.LeafNodeInterface) []byte {
	if leaf == nil {
		return nil
	}

	buf := make([]byte, 64)

	// Copy key (32 bytes)
	key := leaf.Key()
	copy(buf[0:32], key[:])

	// Copy value hash (32 bytes)
	valueHash := leaf.ValueHash()
	copy(buf[32:64], valueHash[:])

	return buf
}

// DecodeLeafNode deserializes a leaf node from bytes
func DecodeLeafNode(data []byte, version types.Version) (types.LeafNodeInterface, error) {
	if len(data) != 64 {
		return nil, fmt.Errorf("invalid leaf data size: %d, expected 64", len(data))
	}

	// Decode key
	var key types.Key
	copy(key[:], data[0:32])

	// Decode value hash
	var valueHash types.Hash
	copy(valueHash[:], data[32:64])

	// Return decoded data - actual node creation handled by factory
	return nil, fmt.Errorf("use codec.DecodeNode for node creation")
}

// LeafNodeSize returns the serialized size of a leaf node
func LeafNodeSize() int {
	return 64 // Always fixed size
}

