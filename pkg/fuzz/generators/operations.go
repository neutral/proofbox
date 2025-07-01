package generators

import (
	"github.com/neutral/proofbox/pkg/types"
	"pgregory.net/rapid"
)

// OpType represents the type of operation
type OpType int

const (
	OpPut OpType = iota
	OpGet
	OpDelete
)

// Operation represents a single tree operation
type Operation struct {
	Type    OpType
	Key     types.Key
	Value   []byte
	Version types.Version
}

// OperationGen generates random tree operations
func OperationGen() *rapid.Generator[Operation] {
	return rapid.Custom(func(t *rapid.T) Operation {
		opType := rapid.SampledFrom([]OpType{OpPut, OpGet, OpDelete}).Draw(t, "type")
		
		// Generate key - use both random and structured patterns
		var key types.Key
		if rapid.Bool().Draw(t, "use_structured_key") {
			// Sometimes use keys with common prefixes to test internal node sharing
			prefix := rapid.SliceOfN(rapid.Byte(), 1, 16).Draw(t, "prefix")
			suffix := rapid.SliceOfN(rapid.Byte(), 0, 16).Draw(t, "suffix")
			keyBytes := append(prefix, suffix...)
			key = types.KeyHash(keyBytes)
		} else {
			// Random keys
			keyBytes := rapid.SliceOfN(rapid.Byte(), 1, 32).Draw(t, "key")
			key = types.KeyHash(keyBytes)
		}
		
		// Generate value (non-empty for Put operations)
		var value []byte
		if opType == OpPut {
			value = rapid.SliceOfN(rapid.Byte(), 1, 1024).Draw(t, "value") // At least 1 byte
		} else {
			value = rapid.SliceOfN(rapid.Byte(), 0, 1024).Draw(t, "value")
		}
		
		// Generate version (0 means current version in most contexts)
		version := rapid.Uint64().Draw(t, "version")
		
		return Operation{
			Type:    opType,
			Key:     key,
			Value:   value,
			Version: types.Version(version),
		}
	})
}

// OperationSliceGen generates a slice of operations
func OperationSliceGen(minOps, maxOps int) *rapid.Generator[[]Operation] {
	return rapid.SliceOfN(OperationGen(), minOps, maxOps)
}

// VersionedOperationGen generates operations with controlled version progression
func VersionedOperationGen(maxVersion types.Version) *rapid.Generator[Operation] {
	return rapid.Custom(func(t *rapid.T) Operation {
		op := OperationGen().Draw(t, "base_operation")
		// Constrain version to be within bounds
		if maxVersion > 0 {
			op.Version = types.Version(rapid.Uint64Range(0, uint64(maxVersion)).Draw(t, "constrained_version"))
		}
		return op
	})
}