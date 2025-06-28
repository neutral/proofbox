package types

import (
	"bytes"
	"encoding/hex"
)

const HashSize = 32 // SHA-256 output size

// Hash represents a 32-byte cryptographic hash
type Hash [HashSize]byte

// EmptyHash returns the zero hash
func EmptyHash() Hash {
	return Hash{}
}

// Bytes returns the hash as a byte slice
func (h Hash) Bytes() []byte {
	return h[:]
}

// String returns hex representation
func (h Hash) String() string {
	return hex.EncodeToString(h[:])
}

// Equal checks hash equality
func (h Hash) Equal(other Hash) bool {
	return bytes.Equal(h[:], other[:])
}

// IsEmpty checks if this is the zero hash
func (h Hash) IsEmpty() bool {
	return h == Hash{}
}

// FromBytes creates a Hash from a byte slice
// Returns error if slice is not exactly 32 bytes
func HashFromBytes(b []byte) (Hash, error) {
	if len(b) != HashSize {
		return Hash{}, ErrInvalidHashSize
	}
	var h Hash
	copy(h[:], b)
	return h, nil
}

// FromHex creates a Hash from a hex string
func HashFromHex(s string) (Hash, error) {
	b, err := hex.DecodeString(s)
	if err != nil {
		return Hash{}, err
	}
	return HashFromBytes(b)
}
