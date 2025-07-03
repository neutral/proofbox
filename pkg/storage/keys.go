package storage

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/neutral/proofbox/pkg/types"
)

// KeyEncoder defines the interface for encoding and decoding storage keys.
// Different implementations can optimize for different storage backends.
type KeyEncoder interface {
	// Node storage keys
	NodeKey(key types.NodeKey) []byte
	ParseNodeKey(data []byte) (types.NodeKey, error)

	// Value storage keys (content-addressed)
	ValueKey(hash types.Hash) []byte

	// Value storage keys (key-addressed, for future use)
	ValueKeyByKey(key types.Key) []byte

	// Root storage keys
	RootKey(version types.Version) []byte
	ParseRootKey(data []byte) (types.Version, error)

	// Key prefix for iteration
	RootKeyPrefix() []byte
}

// DefaultKeyEncoder implements the standard key encoding format.
// This maintains backward compatibility with existing storage.
type DefaultKeyEncoder struct{}

// NewDefaultKeyEncoder creates a new default key encoder.
func NewDefaultKeyEncoder() KeyEncoder {
	return &DefaultKeyEncoder{}
}

// NodeKey creates storage key for a node.
// Format: "n" + version(8) + nibble_count(2) + nibbles(up to 64)
func (e *DefaultKeyEncoder) NodeKey(k types.NodeKey) []byte {
	nibbleCount := k.NibblePath.Length
	nibbleBytes := (nibbleCount + 1) / 2 // Round up for odd counts

	key := make([]byte, 1+8+2+nibbleBytes)
	key[0] = 'n' // Node prefix
	binary.BigEndian.PutUint64(key[1:9], uint64(k.Version))
	binary.BigEndian.PutUint16(key[9:11], nibbleCount)

	// Pack nibbles into bytes
	for i := uint16(0); i < nibbleCount; i++ {
		byteIdx := 11 + i/2
		if i%2 == 0 {
			key[byteIdx] = byte(k.NibblePath.Nibbles[i]) << 4
		} else {
			key[byteIdx] |= byte(k.NibblePath.Nibbles[i])
		}
	}

	return key
}

// ParseNodeKey extracts NodeKey from storage key.
func (e *DefaultKeyEncoder) ParseNodeKey(key []byte) (types.NodeKey, error) {
	if len(key) < 11 || key[0] != 'n' {
		return types.NodeKey{}, fmt.Errorf("invalid node key format")
	}

	version := types.Version(binary.BigEndian.Uint64(key[1:9]))
	nibbleCount := binary.BigEndian.Uint16(key[9:11])

	if nibbleCount > types.MaxTreeDepth {
		return types.NodeKey{}, fmt.Errorf("nibble count exceeds maximum: %d", nibbleCount)
	}

	expectedLen := 11 + (nibbleCount+1)/2
	if len(key) != int(expectedLen) {
		return types.NodeKey{}, fmt.Errorf("invalid node key length")
	}

	// Unpack nibbles
	nibbles := make([]types.Nibble, nibbleCount)
	for i := uint16(0); i < nibbleCount; i++ {
		byteIdx := 11 + i/2
		if i%2 == 0 {
			nibbles[i] = types.Nibble(key[byteIdx] >> 4)
		} else {
			nibbles[i] = types.Nibble(key[byteIdx] & 0x0F)
		}
	}

	return types.NodeKey{
		Version: version,
		NibblePath: types.NibblePath{
			Nibbles: nibbles,
			Length:  nibbleCount,
		},
	}, nil
}

// ValueKey creates a storage key for value.
// Values are content-addressed without version to enable deduplication.
// Format: "v" + hash(32)
func (e *DefaultKeyEncoder) ValueKey(hash types.Hash) []byte {
	key := make([]byte, 1+32)
	key[0] = 'v' // Value prefix
	copy(key[1:], hash[:])
	return key
}

// ValueKeyByKey creates a storage key for value by original key.
// Format: "k" + key(32)
func (e *DefaultKeyEncoder) ValueKeyByKey(key types.Key) []byte {
	result := make([]byte, 1+32)
	result[0] = 'k' // Key-based value prefix
	copy(result[1:], key[:])
	return result
}

// RootKey creates a storage key for root hash.
// Format: "r" + version(8)
func (e *DefaultKeyEncoder) RootKey(version types.Version) []byte {
	key := make([]byte, len(types.RootKeyPrefix)+8)
	copy(key, types.RootKeyPrefix)
	binary.BigEndian.PutUint64(key[len(types.RootKeyPrefix):], uint64(version))
	return key
}

// ParseRootKey extracts version from root key.
func (e *DefaultKeyEncoder) ParseRootKey(key []byte) (types.Version, error) {
	if !bytes.HasPrefix(key, []byte(types.RootKeyPrefix)) {
		return 0, errors.New("invalid root key prefix")
	}

	versionBytes := key[len(types.RootKeyPrefix):]
	if len(versionBytes) != 8 {
		return 0, errors.New("invalid root key length")
	}

	return types.Version(binary.BigEndian.Uint64(versionBytes)), nil
}

// RootKeyPrefix returns the prefix for root keys.
func (e *DefaultKeyEncoder) RootKeyPrefix() []byte {
	return []byte(types.RootKeyPrefix)
}
