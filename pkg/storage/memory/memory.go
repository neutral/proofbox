package memory

import (
	"bytes"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"

	"github.com/neutral/proofbox/pkg/storage"
)

// Storage implements storage.Storage using an in-memory map.
// This is useful for testing and development.
type Storage struct {
	mu   sync.RWMutex
	data map[string][]byte

	// Resource management
	refCount      int32 // Active resources
	closing       int32 // Atomic closing flag
	resourceWG    sync.WaitGroup
	openIterators int32
	openSnapshots int32
	maxIterators  int32
	maxSnapshots  int32

	metrics storage.Metrics
}

// Options configures memory storage.
type Options struct {
	MaxIterators int32
	MaxSnapshots int32
}

// DefaultOptions returns default options.
func DefaultOptions() *Options {
	return &Options{
		MaxIterators: 1000,
		MaxSnapshots: 100,
	}
}

// NewStorage creates a new in-memory storage instance.
func NewStorage() *Storage {
	return NewStorageWithOptions(DefaultOptions())
}

// NewStorageWithOptions creates storage with custom options.
func NewStorageWithOptions(opts *Options) *Storage {
	if opts == nil {
		opts = DefaultOptions()
	}
	return &Storage{
		data:         make(map[string][]byte),
		maxIterators: opts.MaxIterators,
		maxSnapshots: opts.MaxSnapshots,
		metrics:      storage.NullMetrics{},
	}
}

// addRef increments reference count if not closing
func (s *Storage) addRef() bool {
	for {
		closing := atomic.LoadInt32(&s.closing)
		if closing != 0 {
			return false
		}
		refs := atomic.LoadInt32(&s.refCount)
		if atomic.CompareAndSwapInt32(&s.refCount, refs, refs+1) {
			s.resourceWG.Add(1)
			return true
		}
	}
}

// releaseRef decrements reference count
func (s *Storage) releaseRef() {
	atomic.AddInt32(&s.refCount, -1)
	s.resourceWG.Done()
}

// Get retrieves a value for the given key.
func (s *Storage) Get(key []byte) ([]byte, error) {
	if atomic.LoadInt32(&s.closing) != 0 {
		return nil, storage.ErrStorageClosed
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	value, exists := s.data[string(key)]
	if !exists {
		return nil, nil
	}

	// Return a copy to prevent modifications
	result := make([]byte, len(value))
	copy(result, value)
	return result, nil
}

// Put stores a key-value pair.
func (s *Storage) Put(key, value []byte) error {
	if atomic.LoadInt32(&s.closing) != 0 {
		return storage.ErrStorageClosed
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Store a copy to prevent external modifications
	valueCopy := make([]byte, len(value))
	copy(valueCopy, value)
	s.data[string(key)] = valueCopy

	return nil
}

// Delete removes a key-value pair.
func (s *Storage) Delete(key []byte) error {
	if atomic.LoadInt32(&s.closing) != 0 {
		return storage.ErrStorageClosed
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.data, string(key))
	return nil
}

// NewBatch creates a new batch for atomic updates.
func (s *Storage) NewBatch() storage.Batch {
	if !s.addRef() {
		return &errorBatch{err: storage.ErrStorageClosed}
	}

	return &batch{
		storage: s,
		ops:     make([]batchOp, 0),
	}
}

// NewIterator creates an iterator with the given options.
func (s *Storage) NewIterator(opts *storage.IteratorOptions) storage.Iterator {
	if !s.addRef() {
		return &errorIterator{err: storage.ErrStorageClosed}
	}

	// Check iterator limit
	if s.maxIterators > 0 {
		current := atomic.AddInt32(&s.openIterators, 1)
		if current > s.maxIterators {
			atomic.AddInt32(&s.openIterators, -1)
			s.releaseRef()
			return &errorIterator{err: storage.ErrTooManyIterators}
		}
	} else {
		atomic.AddInt32(&s.openIterators, 1)
	}

	s.mu.RLock()
	// Create a snapshot of data for consistent iteration
	dataCopy := make(map[string][]byte, len(s.data))
	for k, v := range s.data {
		valueCopy := make([]byte, len(v))
		copy(valueCopy, v)
		dataCopy[k] = valueCopy
	}
	s.mu.RUnlock()

	// Extract all keys and sort them
	keys := make([]string, 0, len(dataCopy))
	for k := range dataCopy {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Apply bounds if specified
	if opts != nil {
		if opts.Prefix != nil {
			// Filter by prefix
			filtered := make([]string, 0)
			for _, k := range keys {
				if bytes.HasPrefix([]byte(k), opts.Prefix) {
					filtered = append(filtered, k)
				}
			}
			keys = filtered
		} else {
			// Apply lower/upper bounds
			if opts.LowerBound != nil {
				lower := string(opts.LowerBound)
				idx := sort.SearchStrings(keys, lower)
				keys = keys[idx:]
			}
			if opts.UpperBound != nil {
				upper := string(opts.UpperBound)
				idx := sort.SearchStrings(keys, upper)
				keys = keys[:idx]
			}
		}
	}

	return &iterator{
		storage: s,
		data:    dataCopy,
		keys:    keys,
		index:   -1,
	}
}

// NewSnapshot creates a point-in-time snapshot.
func (s *Storage) NewSnapshot() storage.Snapshot {
	if !s.addRef() {
		return &errorSnapshot{err: storage.ErrStorageClosed}
	}

	// Check snapshot limit
	if s.maxSnapshots > 0 {
		current := atomic.AddInt32(&s.openSnapshots, 1)
		if current > s.maxSnapshots {
			atomic.AddInt32(&s.openSnapshots, -1)
			s.releaseRef()
			return &errorSnapshot{err: storage.ErrTooManySnapshots}
		}
	} else {
		atomic.AddInt32(&s.openSnapshots, 1)
	}

	s.mu.RLock()
	// Create a copy of the data
	dataCopy := make(map[string][]byte, len(s.data))
	for k, v := range s.data {
		valueCopy := make([]byte, len(v))
		copy(valueCopy, v)
		dataCopy[k] = valueCopy
	}
	s.mu.RUnlock()

	return &snapshot{
		data:    dataCopy,
		storage: s,
	}
}

// Metrics returns storage metrics.
func (s *Storage) Metrics() storage.Metrics {
	return s.metrics
}

// Close releases all resources.
func (s *Storage) Close() error {
	// Mark as closing atomically
	if !atomic.CompareAndSwapInt32(&s.closing, 0, 1) {
		return nil // Already closing
	}

	// Wait for all active resources
	s.resourceWG.Wait()

	// Check for leaks
	openIters := atomic.LoadInt32(&s.openIterators)
	openSnaps := atomic.LoadInt32(&s.openSnapshots)
	openRefs := atomic.LoadInt32(&s.refCount)
	if openIters > 0 || openSnaps > 0 || openRefs > 0 {
		// This indicates a bug
		fmt.Printf("ERROR: Memory storage closed with %d iterators, %d snapshots, %d refs\n",
			openIters, openSnaps, openRefs)
	}

	// Clear data
	s.mu.Lock()
	s.data = nil
	s.mu.Unlock()

	return nil
}

// batch implements storage.Batch for memory storage.
type batch struct {
	storage *Storage
	ops     []batchOp
	closed  int32
}

type batchOp struct {
	typ   opType
	key   string
	value []byte
}

type opType int

const (
	opPut opType = iota
	opDelete
)

func (b *batch) Put(key, value []byte) error {
	valueCopy := make([]byte, len(value))
	copy(valueCopy, value)

	b.ops = append(b.ops, batchOp{
		typ:   opPut,
		key:   string(key),
		value: valueCopy,
	})
	return nil
}

func (b *batch) Delete(key []byte) error {
	b.ops = append(b.ops, batchOp{
		typ: opDelete,
		key: string(key),
	})
	return nil
}

func (b *batch) Commit(opts storage.CommitOptions) error {
	if atomic.LoadInt32(&b.closed) != 0 {
		return storage.ErrStorageClosed
	}

	if atomic.LoadInt32(&b.storage.closing) != 0 {
		return storage.ErrStorageClosed
	}

	b.storage.mu.Lock()
	defer b.storage.mu.Unlock()

	// Apply all operations
	for _, op := range b.ops {
		switch op.typ {
		case opPut:
			b.storage.data[op.key] = op.value
		case opDelete:
			delete(b.storage.data, op.key)
		}
	}

	// Release reference after commit
	b.close()
	return nil
}

func (b *batch) close() {
	if atomic.CompareAndSwapInt32(&b.closed, 0, 1) {
		b.storage.releaseRef()
	}
}

func (b *batch) Close() error {
	b.close()
	b.ops = nil
	return nil
}

// iterator implements storage.Iterator for memory storage.
type iterator struct {
	storage *Storage
	data    map[string][]byte // snapshot of data
	keys    []string
	index   int
	closed  int32 // Atomic
}

func (i *iterator) SeekGE(key []byte) bool {
	target := string(key)
	idx := sort.SearchStrings(i.keys, target)
	if idx < len(i.keys) {
		i.index = idx
		return true
	}
	i.index = len(i.keys)
	return false
}

func (i *iterator) SeekLT(key []byte) bool {
	target := string(key)
	idx := sort.SearchStrings(i.keys, target)
	if idx > 0 {
		i.index = idx - 1
		return true
	}
	i.index = -1
	return false
}

func (i *iterator) First() bool {
	if len(i.keys) > 0 {
		i.index = 0
		return true
	}
	return false
}

func (i *iterator) Last() bool {
	if len(i.keys) > 0 {
		i.index = len(i.keys) - 1
		return true
	}
	return false
}

func (i *iterator) Next() bool {
	if i.index >= 0 && i.index < len(i.keys)-1 {
		i.index++
		return true
	}
	i.index = len(i.keys)
	return false
}

func (i *iterator) Prev() bool {
	if i.index > 0 && i.index <= len(i.keys) {
		i.index--
		return true
	}
	i.index = -1
	return false
}

func (i *iterator) Valid() bool {
	return i.index >= 0 && i.index < len(i.keys)
}

func (i *iterator) Key() []byte {
	if i.Valid() {
		return []byte(i.keys[i.index])
	}
	return nil
}

func (i *iterator) Value() []byte {
	if atomic.LoadInt32(&i.closed) != 0 || i.data == nil {
		return nil
	}
	if i.Valid() {
		// Use snapshot data instead of live storage
		if value, exists := i.data[i.keys[i.index]]; exists {
			// Return a copy to prevent external modification
			result := make([]byte, len(value))
			copy(result, value)
			return result
		}
	}
	return nil
}

func (i *iterator) Error() error {
	return nil
}

func (i *iterator) Close() error {
	if atomic.CompareAndSwapInt32(&i.closed, 0, 1) {
		i.keys = nil
		i.data = nil // Clear data reference to prevent memory leak
		i.index = -1
		atomic.AddInt32(&i.storage.openIterators, -1)
		i.storage.releaseRef()
	}
	return nil
}

// snapshot implements storage.Snapshot for memory storage.
type snapshot struct {
	data    map[string][]byte
	storage *Storage
	err     error
	closed  int32
}

func (s *snapshot) Get(key []byte) ([]byte, error) {
	if s.err != nil {
		return nil, s.err
	}

	value, exists := s.data[string(key)]
	if !exists {
		return nil, nil
	}

	// Return a copy
	result := make([]byte, len(value))
	copy(result, value)
	return result, nil
}

func (s *snapshot) NewIterator(opts *storage.IteratorOptions) storage.Iterator {
	if s.err != nil {
		return &errorIterator{err: s.err}
	}

	// Extract and sort keys
	keys := make([]string, 0, len(s.data))
	for k := range s.data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Apply bounds
	if opts != nil {
		if opts.Prefix != nil {
			filtered := make([]string, 0)
			for _, k := range keys {
				if bytes.HasPrefix([]byte(k), opts.Prefix) {
					filtered = append(filtered, k)
				}
			}
			keys = filtered
		} else {
			if opts.LowerBound != nil {
				lower := string(opts.LowerBound)
				idx := sort.SearchStrings(keys, lower)
				keys = keys[idx:]
			}
			if opts.UpperBound != nil {
				upper := string(opts.UpperBound)
				idx := sort.SearchStrings(keys, upper)
				keys = keys[:idx]
			}
		}
	}

	// Note: We don't add a reference here since snapshot already has one
	if s.storage != nil && s.storage.maxIterators > 0 {
		current := atomic.AddInt32(&s.storage.openIterators, 1)
		if current > s.storage.maxIterators {
			atomic.AddInt32(&s.storage.openIterators, -1)
			return &errorIterator{err: storage.ErrTooManyIterators}
		}
	} else if s.storage != nil {
		atomic.AddInt32(&s.storage.openIterators, 1)
	}

	return &snapshotIterator{
		snapshot: s,
		keys:     keys,
		index:    -1,
	}
}

func (s *snapshot) Close() error {
	if atomic.CompareAndSwapInt32(&s.closed, 0, 1) {
		s.data = nil
		if s.storage != nil {
			atomic.AddInt32(&s.storage.openSnapshots, -1)
			s.storage.releaseRef()
		}
	}
	return nil
}

// snapshotIterator is like iterator but reads from snapshot data
type snapshotIterator struct {
	snapshot *snapshot
	keys     []string
	index    int
	closed   int32
}

// Implement all iterator methods (same as iterator but reads from snapshot.data)
func (i *snapshotIterator) SeekGE(key []byte) bool {
	target := string(key)
	idx := sort.SearchStrings(i.keys, target)
	if idx < len(i.keys) {
		i.index = idx
		return true
	}
	i.index = len(i.keys)
	return false
}

func (i *snapshotIterator) SeekLT(key []byte) bool {
	target := string(key)
	idx := sort.SearchStrings(i.keys, target)
	if idx > 0 {
		i.index = idx - 1
		return true
	}
	i.index = -1
	return false
}

func (i *snapshotIterator) First() bool {
	if len(i.keys) > 0 {
		i.index = 0
		return true
	}
	return false
}

func (i *snapshotIterator) Last() bool {
	if len(i.keys) > 0 {
		i.index = len(i.keys) - 1
		return true
	}
	return false
}

func (i *snapshotIterator) Next() bool {
	if i.index >= 0 && i.index < len(i.keys)-1 {
		i.index++
		return true
	}
	i.index = len(i.keys)
	return false
}

func (i *snapshotIterator) Prev() bool {
	if i.index > 0 && i.index <= len(i.keys) {
		i.index--
		return true
	}
	i.index = -1
	return false
}

func (i *snapshotIterator) Valid() bool {
	return i.index >= 0 && i.index < len(i.keys)
}

func (i *snapshotIterator) Key() []byte {
	if i.Valid() {
		return []byte(i.keys[i.index])
	}
	return nil
}

func (i *snapshotIterator) Value() []byte {
	if i.Valid() {
		if value, exists := i.snapshot.data[i.keys[i.index]]; exists {
			result := make([]byte, len(value))
			copy(result, value)
			return result
		}
	}
	return nil
}

func (i *snapshotIterator) Error() error { return nil }
func (i *snapshotIterator) Close() error {
	if atomic.CompareAndSwapInt32(&i.closed, 0, 1) {
		i.keys = nil
		if i.snapshot.storage != nil {
			atomic.AddInt32(&i.snapshot.storage.openIterators, -1)
		}
	}
	return nil
}

// errorBatch is returned when batch creation fails
type errorBatch struct {
	err error
}

func (e *errorBatch) Put(key, value []byte) error             { return e.err }
func (e *errorBatch) Delete(key []byte) error                 { return e.err }
func (e *errorBatch) Commit(opts storage.CommitOptions) error { return e.err }
func (e *errorBatch) Close() error                            { return nil }

// errorSnapshot is returned when snapshot creation fails
type errorSnapshot struct {
	err error
}

func (e *errorSnapshot) Get(key []byte) ([]byte, error) { return nil, e.err }
func (e *errorSnapshot) NewIterator(opts *storage.IteratorOptions) storage.Iterator {
	return &errorIterator{err: e.err}
}
func (e *errorSnapshot) Close() error { return nil }

// errorIterator is returned when an error occurs
type errorIterator struct {
	err error
}

func (e *errorIterator) SeekGE(key []byte) bool { return false }
func (e *errorIterator) SeekLT(key []byte) bool { return false }
func (e *errorIterator) First() bool            { return false }
func (e *errorIterator) Last() bool             { return false }
func (e *errorIterator) Next() bool             { return false }
func (e *errorIterator) Prev() bool             { return false }
func (e *errorIterator) Valid() bool            { return false }
func (e *errorIterator) Key() []byte            { return nil }
func (e *errorIterator) Value() []byte          { return nil }
func (e *errorIterator) Error() error           { return e.err }
func (e *errorIterator) Close() error           { return nil }
