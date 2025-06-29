package tree

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/neutral/proofbox/pkg/types"
)

// makeRootKey creates a storage key for root hash
func makeRootKey(version types.Version) []byte {
	key := make([]byte, len(types.RootKeyPrefix)+8)
	copy(key, types.RootKeyPrefix)
	binary.BigEndian.PutUint64(key[len(types.RootKeyPrefix):], uint64(version))
	return key
}

// parseRootKey extracts version from root key
func parseRootKey(key []byte) (types.Version, error) {
	if !bytes.HasPrefix(key, []byte(types.RootKeyPrefix)) {
		return 0, errors.New("invalid root key prefix")
	}

	versionBytes := key[len(types.RootKeyPrefix):]
	if len(versionBytes) != 8 {
		return 0, errors.New("invalid root key length")
	}

	return types.Version(binary.BigEndian.Uint64(versionBytes)), nil
}

// makeValueKey creates a storage key for value
func makeValueKey(version types.Version, hash types.Hash) []byte {
	key := make([]byte, 1+8+32)
	key[0] = 'v' // Value prefix
	binary.BigEndian.PutUint64(key[1:9], uint64(version))
	copy(key[9:], hash[:])
	return key
}

// makeValueKeyByKey creates a storage key for value by original key
func makeValueKeyByKey(key types.Key) []byte {
	result := make([]byte, 1+32)
	result[0] = 'k' // Key-based value prefix
	copy(result[1:], key[:])
	return result
}

// makeNodeKey creates storage key for a node
func makeNodeKey(k types.NodeKey) []byte {
	// Format: "n" + version(8) + nibble_count(2) + nibbles(up to 64)
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

// parseNodeKey extracts NodeKey from storage key
func parseNodeKey(key []byte) (types.NodeKey, error) {
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