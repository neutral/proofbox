package types

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// Key represents a 256-bit key in the tree
type Key [32]byte

// Nibble represents a 4-bit value (0-15)
type Nibble uint8

// NibblePath represents a path through the tree
type NibblePath struct {
	Nibbles []Nibble
	Length  uint16
}

const (
	KeySize        = 32 // 256 bits
	NibblePerByte  = 2
	MaxNibbleValue = 15
	MaxTreeDepth   = 64      // 32 bytes * 2 nibbles per byte
	MaxValueSize   = 1 << 20 // 1 MB maximum value size
)

// KeyFromBytes creates a Key from byte slice
func KeyFromBytes(b []byte) (Key, error) {
	if len(b) != KeySize {
		return Key{}, WrapError(ErrInvalidKey, "key must be 32 bytes, got %d", len(b))
	}
	var key Key
	copy(key[:], b)
	return key, nil
}

// KeyHash converts variable-length data to a fixed Key using SHA-256
func KeyHash(data []byte) Key {
	return Key(sha256.Sum256(data))
}

// String returns hex representation of the key
func (k Key) String() string {
	return hex.EncodeToString(k[:])
}

// Bytes returns the key as a byte slice
func (k Key) Bytes() []byte {
	return k[:]
}

// IsEmpty checks if the key is all zeros
func (k Key) IsEmpty() bool {
	return k == Key{}
}

// ValidateKey ensures a key is valid
func ValidateKey(k Key) error {
	if k.IsEmpty() {
		return ErrEmptyKey
	}
	return nil
}

// ExtractNibble extracts the nibble at the given depth
func (k Key) ExtractNibble(depth int) (Nibble, error) {
	if depth < 0 || depth >= MaxTreeDepth {
		return 0, WrapError(ErrInvalidNibble, "depth %d out of range [0, %d)", depth, MaxTreeDepth)
	}

	byteIndex := depth / NibblePerByte
	nibbleIndex := depth % NibblePerByte

	b := k[byteIndex]
	if nibbleIndex == 0 {
		return Nibble(b >> 4), nil // High nibble
	}
	return Nibble(b & 0x0F), nil // Low nibble
}

// ToNibblePath converts a key to a full nibble path
func (k Key) ToNibblePath() NibblePath {
	nibbles := make([]Nibble, MaxTreeDepth)
	for i := 0; i < MaxTreeDepth; i++ {
		// ExtractNibble should never error here as we're iterating
		// within the valid range [0, MaxTreeDepth)
		nibble, err := k.ExtractNibble(i)
		if err != nil {
			// This should never happen, but handle it gracefully
			panic(fmt.Sprintf("ExtractNibble failed for valid depth %d: %v", i, err))
		}
		nibbles[i] = nibble
	}
	return NibblePath{
		Nibbles: nibbles,
		Length:  MaxTreeDepth,
	}
}

// ValidateNibble ensures nibble is in valid range
func ValidateNibble(n Nibble) error {
	if n > MaxNibbleValue {
		return WrapError(ErrInvalidNibble, "nibble %d exceeds max value %d", n, MaxNibbleValue)
	}
	return nil
}

// String returns a human-readable representation of the nibble path
func (np NibblePath) String() string {
	if np.Length == 0 {
		return "<empty>"
	}
	result := make([]byte, np.Length)
	for i := uint16(0); i < np.Length; i++ {
		result[i] = fmt.Sprintf("%X", np.Nibbles[i])[0]
	}
	return string(result)
}

// Compare compares two nibble paths lexicographically
func (np NibblePath) Compare(other NibblePath) int {
	minLen := np.Length
	if other.Length < minLen {
		minLen = other.Length
	}

	// Compare common prefix
	for i := uint16(0); i < minLen; i++ {
		if np.Nibbles[i] < other.Nibbles[i] {
			return -1
		}
		if np.Nibbles[i] > other.Nibbles[i] {
			return 1
		}
	}

	// Shorter path comes first
	if np.Length < other.Length {
		return -1
	}
	if np.Length > other.Length {
		return 1
	}

	return 0
}

// CommonPrefixLength returns the number of matching nibbles from the start
func (np NibblePath) CommonPrefixLength(other NibblePath) int {
	minLen := np.Length
	if other.Length < minLen {
		minLen = other.Length
	}

	for i := uint16(0); i < minLen; i++ {
		if np.Nibbles[i] != other.Nibbles[i] {
			return int(i)
		}
	}
	return int(minLen)
}

// GetNibble returns the nibble at the specified index with bounds checking
func (np NibblePath) GetNibble(index int) (Nibble, error) {
	if index < 0 || index >= int(np.Length) {
		return 0, fmt.Errorf("nibble index %d out of bounds [0, %d)", index, np.Length)
	}
	return np.Nibbles[index], nil
}

// Prefix returns a new NibblePath containing the first 'length' nibbles
func (np NibblePath) Prefix(length int) NibblePath {
	if length <= 0 {
		return NibblePath{Nibbles: []Nibble{}, Length: 0}
	}

	if length > int(np.Length) {
		length = int(np.Length)
	}

	newNibbles := make([]Nibble, length)
	copy(newNibbles, np.Nibbles[:length])

	return NibblePath{
		Nibbles: newNibbles,
		Length:  uint16(length),
	}
}

// Equals returns true if two nibble paths are identical
func (np NibblePath) Equals(other NibblePath) bool {
	if np.Length != other.Length {
		return false
	}

	for i := uint16(0); i < np.Length; i++ {
		if np.Nibbles[i] != other.Nibbles[i] {
			return false
		}
	}
	return true
}

// Append returns a new NibblePath with the given nibble appended
func (np NibblePath) Append(nibble Nibble) NibblePath {
	newNibbles := make([]Nibble, np.Length+1)
	copy(newNibbles, np.Nibbles)
	newNibbles[np.Length] = nibble

	return NibblePath{
		Nibbles: newNibbles,
		Length:  np.Length + 1,
	}
}

// IsPrefix returns true if this path is a prefix of the other path
func (np NibblePath) IsPrefix(other NibblePath) bool {
	if np.Length > other.Length {
		return false
	}

	for i := uint16(0); i < np.Length; i++ {
		if np.Nibbles[i] != other.Nibbles[i] {
			return false
		}
	}
	return true
}

// Skip returns a new NibblePath with the first 'count' nibbles removed
func (np NibblePath) Skip(count int) NibblePath {
	if count >= int(np.Length) {
		return NibblePath{Nibbles: []Nibble{}, Length: 0}
	}

	if count <= 0 {
		return np
	}

	newLength := int(np.Length) - count
	newNibbles := make([]Nibble, newLength)
	copy(newNibbles, np.Nibbles[count:])

	return NibblePath{
		Nibbles: newNibbles,
		Length:  uint16(newLength),
	}
}

// NewNibblePath creates a NibblePath from a byte slice
func NewNibblePath(data []byte) NibblePath {
	nibbles := make([]Nibble, 0, len(data)*2)
	for _, b := range data {
		nibbles = append(nibbles, Nibble(b>>4), Nibble(b&0x0F))
	}
	return NibblePath{
		Nibbles: nibbles,
		Length:  uint16(len(nibbles)),
	}
}

// ToBytes converts a NibblePath back to bytes (for even-length paths)
func (np NibblePath) ToBytes() ([]byte, error) {
	if np.Length%2 != 0 {
		return nil, fmt.Errorf("cannot convert odd-length nibble path to bytes")
	}

	bytes := make([]byte, np.Length/2)
	for i := 0; i < len(bytes); i++ {
		high := np.Nibbles[i*2]
		low := np.Nibbles[i*2+1]
		bytes[i] = byte(high)<<4 | byte(low)
	}

	return bytes, nil
}
