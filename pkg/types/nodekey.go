package types

import (
	"encoding/binary"
	"fmt"
)

// NodeKey uniquely identifies a node in the tree across all versions
type NodeKey struct {
	Version    Version
	NibblePath NibblePath
}

// Version represents a state version in the JMT
type Version uint64

const (
	// MaxVersion is the maximum supported version number
	MaxVersion Version = ^Version(0) - 1

	// InitialVersion is the first version in a new tree
	InitialVersion Version = 0

	// NodeKeyPrefix is the prefix for node storage keys
	NodeKeyPrefix = "n"

	// RootKeyPrefix is the prefix for root hash storage
	RootKeyPrefix = "r"
)

// RootNodeKey returns the key for the root node at a version
func RootNodeKey(version Version) NodeKey {
	return NodeKey{
		Version: version,
		NibblePath: NibblePath{
			Nibbles: []Nibble{},
			Length:  0,
		},
	}
}

// Child returns the NodeKey for a child at the given nibble
func (nk NodeKey) Child(nibble Nibble, childVersion Version) NodeKey {
	newNibbles := make([]Nibble, nk.NibblePath.Length+1)
	copy(newNibbles, nk.NibblePath.Nibbles)
	newNibbles[nk.NibblePath.Length] = nibble

	return NodeKey{
		Version: childVersion,
		NibblePath: NibblePath{
			Nibbles: newNibbles,
			Length:  nk.NibblePath.Length + 1,
		},
	}
}

// IsRoot checks if this is the root node
func (nk NodeKey) IsRoot() bool {
	return nk.NibblePath.Length == 0
}

// String returns a human-readable representation
func (nk NodeKey) String() string {
	if nk.NibblePath.Length == 0 {
		return fmt.Sprintf("v%d:", nk.Version)
	}
	nibbleStr := ""
	for i := uint16(0); i < nk.NibblePath.Length; i++ {
		if i > 0 {
			nibbleStr += fmt.Sprintf("%x", nk.NibblePath.Nibbles[i])
		} else {
			nibbleStr = fmt.Sprintf("%x", nk.NibblePath.Nibbles[i])
		}
	}
	return fmt.Sprintf("v%d:%s", nk.Version, nibbleStr)
}

// EncodeNodeKey encodes a NodeKey to bytes for storage
// Format: Version(8) || Length(2) || Nibbles(packed)
func EncodeNodeKey(key NodeKey) []byte {
	// Calculate buffer size
	nibbleBytes := (key.NibblePath.Length + 1) / 2
	bufSize := 8 + 2 + int(nibbleBytes)
	buf := make([]byte, bufSize)

	// Encode version (8 bytes, big-endian)
	binary.BigEndian.PutUint64(buf[0:8], uint64(key.Version))

	// Encode nibble count (2 bytes, big-endian)
	binary.BigEndian.PutUint16(buf[8:10], key.NibblePath.Length)

	// Encode nibbles (packed, 2 per byte)
	offset := 10
	for i := uint16(0); i < key.NibblePath.Length; i += 2 {
		high := key.NibblePath.Nibbles[i]
		low := Nibble(0)
		if i+1 < key.NibblePath.Length {
			low = key.NibblePath.Nibbles[i+1]
		}
		buf[offset] = byte(high)<<4 | byte(low)
		offset++
	}

	return buf[:offset]
}

// DecodeNodeKey decodes a NodeKey from storage bytes
func DecodeNodeKey(buf []byte) (NodeKey, error) {
	if len(buf) < 10 {
		return NodeKey{}, ErrInvalidNodeKey
	}

	// Decode version
	version := Version(binary.BigEndian.Uint64(buf[0:8]))

	// Decode nibble count
	nibbleCount := binary.BigEndian.Uint16(buf[8:10])
	if nibbleCount > MaxTreeDepth {
		return NodeKey{}, WrapError(ErrInvalidNibbleCount, "count %d exceeds max %d", nibbleCount, MaxTreeDepth)
	}

	// Calculate expected size
	expectedSize := 10 + (nibbleCount+1)/2
	if len(buf) < int(expectedSize) {
		return NodeKey{}, WrapError(ErrInvalidNodeKey, "buffer too small: got %d, need %d", len(buf), expectedSize)
	}

	// Decode nibbles
	nibbles := make([]Nibble, nibbleCount)
	offset := 10
	for i := uint16(0); i < nibbleCount; i += 2 {
		b := buf[offset]
		nibbles[i] = Nibble(b >> 4)
		if i+1 < nibbleCount {
			nibbles[i+1] = Nibble(b & 0x0F)
		}
		offset++
	}

	return NodeKey{
		Version: version,
		NibblePath: NibblePath{
			Nibbles: nibbles,
			Length:  nibbleCount,
		},
	}, nil
}

// StorageKey creates the full storage key for PebbleDB
func (nk NodeKey) StorageKey() []byte {
	encoded := EncodeNodeKey(nk)
	storageKey := make([]byte, len(NodeKeyPrefix)+len(encoded))
	copy(storageKey, []byte(NodeKeyPrefix))
	copy(storageKey[len(NodeKeyPrefix):], encoded)
	return storageKey
}

// Compare returns -1, 0, or 1 for less than, equal, or greater than
func (nk NodeKey) Compare(other NodeKey) int {
	// Compare versions first
	if nk.Version < other.Version {
		return -1
	}
	if nk.Version > other.Version {
		return 1
	}

	// Same version, compare nibble paths
	return nk.NibblePath.Compare(other.NibblePath)
}

// ValidateVersion ensures a version is valid
func ValidateVersion(v Version) error {
	if v > MaxVersion {
		return ErrInvalidVersion
	}
	return nil
}

// EncodeVersion encodes a version to 8 bytes big-endian
func EncodeVersion(v Version) [8]byte {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], uint64(v))
	return buf
}

// DecodeVersion decodes a version from 8 bytes big-endian
func DecodeVersion(buf [8]byte) Version {
	return Version(binary.BigEndian.Uint64(buf[:]))
}
