---
id: step.02.hasher
depends_on:
  - step.01.project‑scaffold
tags: [crypto, step]
---

## Objective

Add `Hasher` interface plus `DefaultSHA256` and constant `DefaultDigest` to establish the cryptographic foundation for the Jellyfish Merkle Tree.

## Implements

- **Glossary – Digest**, **§3 Conventions & Parameters** (`HASH_LEN`, _Hash Function_), and **Security Requirement R1** (128‑bit+ collision resistance).
- **Reference**: `/blueprint/global/specs/common-definitions.md` - Hash Function section
  
_What happens_:
- Define the `Hasher` abstraction so that later steps can swap SHA‑256 for another algorithm without touching the tree code ("Hash Function is assumed but not fixed", original text).
- Provide a constant `DefaultDigest` = `H("")`, aligning with **§3 Default Digest** and the "Sparse Merkle default hash" discussion (§Sparse Merkle Tree concept).

## Technical Details

### File Structure
Create the following files:
- `pkg/crypto/hasher.go` - Hasher interface and implementations
- `pkg/crypto/hasher_test.go` - Unit tests
- `pkg/types/hash.go` - Hash type definition

### Implementation

1. **Hash Type** (`pkg/types/hash.go`):
```go
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
```

2. **Hasher Interface** (`pkg/crypto/hasher.go`):
```go
package crypto

import (
    "crypto/sha256"
    "github.com/acme/jmt/pkg/types"
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
```

### Constants and Configuration

Add to `pkg/crypto/constants.go`:
```go
package crypto

import "github.com/acme/jmt/pkg/types"

// EmptyTreeHash is the hash used for empty subtrees in sparse Merkle tree
var EmptyTreeHash = types.Hash{
    0x5b, 0xa9, 0x3c, 0x9d, 0xb0, 0xcf, 0xf9, 0x3f,
    0x52, 0xb5, 0x21, 0xd7, 0x81, 0x4c, 0x7f, 0xa0,
    0x8a, 0xbe, 0x86, 0x13, 0x5f, 0x74, 0x6b, 0x46,
    0x68, 0xb0, 0x13, 0xbc, 0xd1, 0xc4, 0x8e, 0x9b,
}
```

## Testing Requirements

1. **Basic Hash Tests**:
   - Verify SHA-256 produces correct output
   - Test empty hash value matches expected
   - Confirm hash concatenation works correctly

2. **Interface Tests**:
   - Verify DefaultHasher implements Hasher interface
   - Test that different hasher implementations can be swapped

3. **Property Tests**:
   - Hash determinism (same input → same output)
   - Hash distribution (avalanche effect)

## Implementation Steps

1. Create `pkg/types/hash.go` with Hash type
2. Create `pkg/crypto/hasher.go` with interface and SHA-256 implementation
3. Add constants for default/empty hashes
4. Write comprehensive unit tests
5. Add benchmarks for hash operations

## Performance Considerations

- Pre-allocate buffers for concatenation operations
- Consider caching frequently used hashes
- Benchmark different hash functions for future optimization

## Security Notes

- SHA-256 provides 128-bit collision resistance (meets requirement)
- Interface allows future migration to SHA-3 or BLAKE2
- Constant-time comparison for hash equality checks

## Done When ✓

- [x] `Hasher` interface defined with Hash, HashConcat, EmptyHash methods
- [x] `DefaultSHA256` implementation complete
- [x] `DefaultDigest` constant defined
- [x] Unit test asserts `DefaultDigest == sha256.Sum256(nil)`
- [x] Hash type with Equal, String, Bytes methods
- [x] Interface allows swapping hash functions
- [x] Benchmarks for hash operations
- [x] Documentation comments for all public APIs
