package crypto

import (
	"crypto/sha256"

	"github.com/neutral/proofbox/pkg/types"
)

// Hasher defines the hash function interface
type Hasher interface {
	// Hash computes hash of data
	Hash(data []byte) types.Hash

	// HashConcat hashes concatenated byte slices
	HashConcat(parts ...[]byte) types.Hash

	// EmptyHash returns hash of empty data
	EmptyHash() types.Hash
}

// DefaultSHA256 implements Hasher using SHA-256
type DefaultSHA256 struct{}

// Hash computes SHA-256 hash
func (h DefaultSHA256) Hash(data []byte) types.Hash {
	return sha256.Sum256(data)
}

// HashConcat hashes concatenated parts
func (h DefaultSHA256) HashConcat(parts ...[]byte) types.Hash {
	hasher := sha256.New()
	for _, part := range parts {
		hasher.Write(part)
	}
	var result types.Hash
	hasher.Sum(result[:0])
	return result
}

// EmptyHash returns SHA-256 of empty data
func (h DefaultSHA256) EmptyHash() types.Hash {
	return sha256.Sum256(nil)
}

// DefaultHasher is the default hasher instance
var DefaultHasher Hasher = DefaultSHA256{}

// DefaultDigest is the hash of empty data (sparse tree default)
var DefaultDigest = DefaultHasher.EmptyHash()
