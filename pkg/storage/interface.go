package storage

// Storage is the main interface for persistent key-value storage.
// It provides basic CRUD operations, batch support, iteration, and snapshots.
type Storage interface {
	// Get retrieves the value for a given key.
	// Returns nil if the key does not exist.
	Get(key []byte) ([]byte, error)

	// Put stores a key-value pair.
	Put(key, value []byte) error

	// Delete removes a key-value pair.
	Delete(key []byte) error

	// NewBatch creates a new batch for atomic updates.
	NewBatch() Batch

	// NewIterator creates an iterator with the given options.
	NewIterator(opts *IteratorOptions) Iterator

	// NewSnapshot creates a point-in-time snapshot of the storage.
	NewSnapshot() Snapshot

	// Metrics returns storage metrics if available.
	Metrics() Metrics

	// Close releases all resources.
	Close() error
}

// Batch provides atomic batch operations.
type Batch interface {
	// Put adds a key-value pair to the batch.
	Put(key, value []byte) error

	// Delete adds a key deletion to the batch.
	Delete(key []byte) error

	// Commit atomically applies all operations in the batch.
	Commit(opts CommitOptions) error

	// Close releases batch resources without committing.
	Close() error
}

// CommitOptions controls batch commit behavior.
type CommitOptions struct {
	// Sync forces synchronous write to disk.
	Sync bool
}

// Iterator provides sequential access to key-value pairs.
type Iterator interface {
	// SeekGE seeks to the first key greater than or equal to the given key.
	SeekGE(key []byte) bool

	// SeekLT seeks to the last key less than the given key.
	SeekLT(key []byte) bool

	// First seeks to the first key.
	First() bool

	// Last seeks to the last key.
	Last() bool

	// Next advances to the next key.
	Next() bool

	// Prev moves to the previous key.
	Prev() bool

	// Valid returns true if the iterator is positioned at a valid key-value pair.
	Valid() bool

	// Key returns the current key.
	Key() []byte

	// Value returns the current value.
	Value() []byte

	// Error returns any error encountered during iteration.
	Error() error

	// Close releases iterator resources.
	Close() error
}

// IteratorOptions configures iterator behavior.
type IteratorOptions struct {
	// LowerBound specifies the smallest key (inclusive).
	LowerBound []byte

	// UpperBound specifies the largest key (exclusive).
	UpperBound []byte

	// Prefix is a convenience for setting LowerBound and UpperBound
	// to iterate over keys with a specific prefix.
	Prefix []byte
}

// Snapshot provides a consistent view of the storage at a point in time.
type Snapshot interface {
	// Get retrieves a value from the snapshot.
	Get(key []byte) ([]byte, error)

	// NewIterator creates an iterator over the snapshot.
	NewIterator(opts *IteratorOptions) Iterator

	// Close releases snapshot resources.
	Close() error
}

// Metrics provides storage performance and usage metrics.
type Metrics interface {
	// Operation counts
	GetOperations() uint64
	PutOperations() uint64
	DeleteOperations() uint64
	BatchCommits() uint64

	// Performance metrics (nanoseconds)
	GetLatencyP50() uint64
	GetLatencyP99() uint64
	PutLatencyP50() uint64
	PutLatencyP99() uint64

	// Size metrics (bytes)
	DatabaseSize() uint64
	LiveDataSize() uint64

	// Cache metrics
	CacheHitRate() float64
	CacheSize() uint64
}

// VersionedStorage is reserved for future use when storage-level
// versioning is implemented. Currently, versioning is handled at
// the tree layer.
