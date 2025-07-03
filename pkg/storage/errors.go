package storage

import "errors"

// Configuration errors
var (
	ErrInvalidBackend         = errors.New("invalid storage backend")
	ErrPathRequired           = errors.New("storage path required for persistent backend")
	ErrInvalidCacheSize       = errors.New("cache size must be non-negative")
	ErrInvalidWriteBufferSize = errors.New("write buffer size must be non-negative")
	ErrInvalidCompression     = errors.New("invalid compression type")
	ErrInvalidPruningPolicy   = errors.New("invalid pruning policy type")
)

// Storage operation errors
var (
	ErrStorageClosed    = errors.New("storage is closed")
	ErrKeyNotFound      = errors.New("key not found")
	ErrInvalidKey       = errors.New("invalid key")
	ErrInvalidValue     = errors.New("invalid value")
	ErrTooManyIterators = errors.New("too many open iterators")
	ErrTooManySnapshots = errors.New("too many open snapshots")
)
