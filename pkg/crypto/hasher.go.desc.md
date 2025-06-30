# Hasher Interface Description

## Overview
The `Hasher` interface defines the cryptographic hash function abstraction for the Jellyfish Merkle Tree. It allows the tree implementation to be agnostic about the specific hash algorithm used.

## Interface Design

```go
type Hasher interface {
    Hash(data []byte) types.Hash
    HashConcat(parts ...[]byte) types.Hash
    EmptyHash() types.Hash
}
```

### Method Rationale

#### Hash
Basic hash computation. Takes arbitrary bytes, returns fixed-size hash.

#### HashConcat
Optimized concatenation + hashing in one operation. This is critical for tree operations where we frequently hash multiple node elements together:
```go
// Instead of:
combined := append(append([]byte{}, part1...), part2...)
hash := hasher.Hash(combined)

// We can do:
hash := hasher.HashConcat(part1, part2)
```
Benefits:
- Avoids intermediate allocations
- Can reuse internal hasher state
- Clearer intent

#### EmptyHash
Returns the hash of empty data. Having this as a method rather than computing `Hash(nil)` each time:
- Makes the empty hash a constant (computed once)
- Documents this special case
- Enables optimization in implementations

## DefaultSHA256 Implementation

Our default implementation uses SHA-256 because:
- **Security**: Provides 128-bit collision resistance (exceeds requirements)
- **Performance**: Hardware acceleration on modern CPUs
- **Ubiquity**: Available in all environments, well-tested
- **Size**: 256-bit output matches our Hash type perfectly

### Implementation Details

```go
func (h DefaultSHA256) HashConcat(parts ...[]byte) types.Hash {
    hasher := sha256.New()  // Reusable state
    for _, part := range parts {
        hasher.Write(part)  // Streaming API
    }
    var result types.Hash
    hasher.Sum(result[:0])  // Write directly into result
    return result
}
```

## Global Instances

### DefaultHasher
```go
var DefaultHasher Hasher = DefaultSHA256{}
```
- Package-level singleton for convenience
- Most code uses this rather than instantiating hashers
- Can be replaced for testing or alternative implementations

### DefaultDigest
```go
var DefaultDigest = DefaultHasher.EmptyHash()
```
- The hash of empty data: SHA-256("") 
- Value: `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`
- Used as the default value for empty leaves in the tree
- **NOT** the same as `EmptyTreeHash` (see constants.desc.md)

## Future Flexibility

The interface design allows for:

### Algorithm Migration
```go
type Blake2bHasher struct{}
func (h Blake2bHasher) Hash(data []byte) types.Hash { ... }

// Simply replace:
DefaultHasher = Blake2bHasher{}
```

### Performance Optimization
```go
type PooledHasher struct {
    pool sync.Pool  // Reuse hasher instances
}
```

### Testing
```go
type MockHasher struct {
    hashFunc func([]byte) types.Hash
}
// Inject deterministic hashes for testing
```

## Usage Guidelines

1. **Always use the interface type** in function signatures:
   ```go
   func ComputeRoot(hasher Hasher, nodes []Node) types.Hash
   ```

2. **Use DefaultHasher** unless you have specific requirements

3. **Prefer HashConcat** over manual concatenation for multiple parts

4. **Cache EmptyHash()** result if used frequently (though it's already cached in DefaultDigest)