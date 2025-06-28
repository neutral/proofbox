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
	MaxTreeDepth   = 64 // 32 bytes * 2 nibbles per byte
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
		nibbles[i], _ = k.ExtractNibble(i)
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

