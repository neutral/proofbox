# Hash Type Description

## Overview
The `Hash` type represents a 32-byte cryptographic hash value, designed specifically for SHA-256 output. This is a foundational type used throughout the Jellyfish Merkle Tree implementation.

## Design Decisions

### Fixed-Size Array
```go
type Hash [32]byte
```
We use a fixed-size array rather than a slice for several reasons:
- **Type Safety**: Compile-time guarantee of correct size
- **Performance**: No heap allocation, better cache locality
- **Clarity**: Makes the 256-bit requirement explicit in the type

### Methods

#### Core Operations
- `Bytes() []byte` - Returns a slice view of the hash for APIs that need `[]byte`
- `String() string` - Hex encoding for logging and debugging
- `Equal(other Hash) bool` - Constant-time comparison using `bytes.Equal`

#### Validation
- `IsEmpty() bool` - Checks if this is the zero hash (all bytes are 0)

### Constructor Functions

#### HashFromBytes
```go
func HashFromBytes(b []byte) (Hash, error)
```
- Validates input is exactly 32 bytes
- Returns `ErrInvalidHashSize` if not
- Copies the bytes to prevent aliasing issues

#### HashFromHex
```go
func HashFromHex(s string) (Hash, error)
```
- Parses hex-encoded strings
- Useful for configuration and testing
- Combines hex decoding with size validation

## Usage Patterns

### Creating Hashes
```go
// From computation
hash := sha256.Sum256(data)  // Returns [32]byte directly

// From bytes
h, err := HashFromBytes(someBytes)
if err != nil {
    // Handle invalid size
}

// From hex string
h, err := HashFromHex("e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855")
```

### Zero Value
The zero value of `Hash` is a valid hash (all zeros), which is why we provide:
- `EmptyHash()` - Explicitly returns the zero hash
- `IsEmpty()` - Checks if a hash is the zero value

## Performance Considerations

- **Stack Allocation**: As a 32-byte array, `Hash` can often be stack-allocated
- **Copy Semantics**: Passing `Hash` by value copies 32 bytes (consider pointers for hot paths)
- **String Conversion**: `String()` allocates; cache results if used frequently

## Security Notes

- The `Equal` method uses `bytes.Equal` which is constant-time on most platforms
- Always validate untrusted input with `HashFromBytes` or `HashFromHex`
- The type ensures hashes are always exactly 32 bytes