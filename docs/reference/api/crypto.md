# Crypto Package Public API

## Exported Variables

```go
var EmptyTreeHash = types.Hash{...}
var DefaultHasher Hasher = DefaultSHA256{}
var DefaultDigest = DefaultHasher.EmptyHash()
```

## Exported Interfaces

```go
type Hasher interface {
    Hash(data []byte) types.Hash
    HashConcat(parts ...[]byte) types.Hash
    EmptyHash() types.Hash
}
```

## Exported Types

```go
type DefaultSHA256 struct{}
```

## Exported Methods

### DefaultSHA256 methods
```go
func (h DefaultSHA256) Hash(data []byte) types.Hash
func (h DefaultSHA256) HashConcat(parts ...[]byte) types.Hash
func (h DefaultSHA256) EmptyHash() types.Hash
```