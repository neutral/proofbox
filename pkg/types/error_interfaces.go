package types

import (
	"context"
	"time"
)

// Note: NodeKey and Version types will be defined in future steps.
// For now, we'll define placeholder types to make the interfaces compile.
// These will be replaced when the actual types are implemented.

// NodeKey represents a key for a node in the tree (placeholder)
type NodeKey struct {
	Version    Version
	NibblePath NibblePath
}

// Version represents a version number (placeholder)
type Version uint64

// NibblePath represents a path of nibbles in the tree (placeholder)
type NibblePath struct {
	Nibbles []byte
	Length  uint16
}

// ReadErrorHandler defines recovery strategy for read errors
// Concrete implementations will be provided in step 8 with tree implementation
type ReadErrorHandler interface {
	HandleCorruptedNode(key NodeKey, err error) error
	HandleMissingNode(key NodeKey) error
}

// WriteErrorHandler defines recovery strategy for write errors
// Concrete implementations will be provided in step 8 with tree implementation
type WriteErrorHandler interface {
	HandleWriteFailure(err error) error
	ShouldRetry(err error) bool
	RetryDelay(attempt int) time.Duration
}

// ErrorReporter allows pluggable error metrics collection
type ErrorReporter interface {
	RecordError(code ErrorCode, err error)
	GetStats() map[ErrorCode]uint64
}

// HealthChecker defines interface for tree integrity verification
// Concrete implementation will be provided in step 8 with tree structure
type HealthChecker interface {
	VerifyIntegrity(ctx context.Context, version Version) error
}
