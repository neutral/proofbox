package types

import "errors"

// Hash-related errors
var (
	// ErrInvalidHashSize indicates a hash byte slice is not the correct size
	ErrInvalidHashSize = errors.New("invalid hash size: must be 32 bytes")
)

// Data Integrity Errors
var (
	// ErrCorruptedNode indicates a node's hash doesn't match its content
	ErrCorruptedNode = errors.New("jmt: corrupted node detected")

	// ErrInvalidNodeType indicates an unknown node type in storage
	ErrInvalidNodeType = errors.New("jmt: invalid node type")

	// ErrHashMismatch indicates computed hash doesn't match expected
	ErrHashMismatch = errors.New("jmt: hash mismatch")

	// ErrInvalidProof indicates a proof is malformed or invalid
	ErrInvalidProof = errors.New("jmt: invalid proof")
)

// Operational Errors
var (
	// ErrKeyNotFound indicates the requested key doesn't exist
	ErrKeyNotFound = errors.New("jmt: key not found")

	// ErrVersionNotFound indicates the requested version doesn't exist
	ErrVersionNotFound = errors.New("jmt: version not found")

	// ErrNodeNotFound indicates a referenced node is missing
	ErrNodeNotFound = errors.New("jmt: node not found")

	// ErrEmptyTree indicates operation on empty tree
	ErrEmptyTree = errors.New("jmt: empty tree")
)

// Input Validation Errors
var (
	// ErrInvalidKey indicates an invalid key format
	ErrInvalidKey = errors.New("jmt: invalid key")

	// ErrEmptyKey indicates an empty key was provided
	ErrEmptyKey = errors.New("jmt: empty key not allowed")

	// ErrInvalidVersion indicates an invalid version number
	ErrInvalidVersion = errors.New("jmt: invalid version")

	// ErrValueTooLarge indicates value exceeds size limit
	ErrValueTooLarge = errors.New("jmt: value too large")

	// ErrInvalidNibble indicates nibble value > 15
	ErrInvalidNibble = errors.New("jmt: invalid nibble value")

	// ErrInvalidNodeKey indicates malformed node key
	ErrInvalidNodeKey = errors.New("jmt: invalid node key")

	// ErrInvalidNibbleCount indicates nibble path too long
	ErrInvalidNibbleCount = errors.New("jmt: nibble count exceeds maximum")
)

// Resource Errors
var (
	// ErrMaxDepthExceeded indicates tree depth limit reached
	ErrMaxDepthExceeded = errors.New("jmt: maximum tree depth exceeded")

	// ErrBatchTooLarge indicates batch exceeds size limit
	ErrBatchTooLarge = errors.New("jmt: batch size exceeds limit")

	// ErrOutOfMemory indicates memory allocation failure
	ErrOutOfMemory = errors.New("jmt: out of memory")
)

// Validation Helper Errors
var (
	// ErrInvalidVersionRange indicates version is out of valid range
	ErrInvalidVersionRange = errors.New("jmt: version exceeds maximum allowed value")
)

// Storage Errors
var (
	// ErrStorageFailure wraps underlying storage errors
	ErrStorageFailure = errors.New("jmt: storage operation failed")

	// ErrTransactionAborted indicates a transaction was aborted
	ErrTransactionAborted = errors.New("jmt: transaction aborted")

	// ErrDatabaseClosed indicates operation on closed database
	ErrDatabaseClosed = errors.New("jmt: database closed")
)

// Transaction Errors
var (
	// ErrTxConflict indicates a transaction conflict
	ErrTxConflict = errors.New("jmt: transaction conflict")

	// ErrTxStateInvalid indicates invalid transaction state
	ErrTxStateInvalid = errors.New("jmt: invalid transaction state")
)
