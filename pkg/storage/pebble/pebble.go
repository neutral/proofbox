package pebble

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cockroachdb/pebble"
	storage "github.com/neutral/proofbox/pkg/storage"
)

// Storage implements the storage.Storage interface using PebbleDB.
type Storage struct {
	db *pebble.DB

	// Metrics tracking
	metrics *storage.MetricsCollector
	stopMetrics chan struct{}
	metricsWG   sync.WaitGroup

	// Resource management
	refCount      int32  // Active resources (iterators, snapshots, batches)
	closing       int32  // Atomic closing flag
	resourceWG    sync.WaitGroup // Wait for all resources
	openIterators int32  // Current open iterators
	openSnapshots int32  // Current open snapshots

	// Configuration
	keyEncoder    storage.KeyEncoder
	maxIterators  int32
	maxSnapshots  int32
}

// Options configures the PebbleDB storage.
type Options struct {
	// PebbleDB options
	PebbleOptions *pebble.Options

	// Key encoder (defaults to DefaultKeyEncoder)
	KeyEncoder storage.KeyEncoder

	// Enable metrics collection
	EnableMetrics bool

	// Resource limits (0 means unlimited)
	MaxIterators int32
	MaxSnapshots int32
}

// DefaultOptions returns default storage options.
func DefaultOptions() *Options {
	return &Options{
		PebbleOptions: &pebble.Options{
			// Default cache size: 64MB
			Cache: pebble.NewCache(64 << 20),
			// Default write buffer: 32MB
			MemTableSize: 32 << 20,
		},
		EnableMetrics: true,
		MaxIterators:  1000, // Default limit
		MaxSnapshots:  100,  // Default limit
	}
}

// NewStorage creates a new PebbleDB storage instance.
func NewStorage(path string, opts *Options) (*Storage, error) {
	if opts == nil {
		opts = DefaultOptions()
	}

	if opts.KeyEncoder == nil {
		opts.KeyEncoder = storage.NewDefaultKeyEncoder()
	}

	// Open PebbleDB
	pebbleOpts := opts.PebbleOptions
	if pebbleOpts == nil {
		pebbleOpts = &pebble.Options{}
	}
	
	db, err := pebble.Open(path, pebbleOpts)
	if err != nil {
		return nil, err
	}

	s := &Storage{
		db:           db,
		keyEncoder:   opts.KeyEncoder,
		maxIterators: opts.MaxIterators,
		maxSnapshots: opts.MaxSnapshots,
	}

	if opts.EnableMetrics {
		s.metrics = storage.NewMetricsCollector()
		s.stopMetrics = make(chan struct{})
	}

	// Start metrics goroutine after struct is fully initialized
	if opts.EnableMetrics {
		s.metricsWG.Add(1)
		go s.updateDatabaseMetrics()
	}

	return s, nil
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
	// Check if closing
	if atomic.LoadInt32(&s.closing) != 0 {
		return nil, storage.ErrStorageClosed
	}

	if s.metrics != nil {
		start := time.Now()
		defer func() {
			s.metrics.RecordGet(time.Since(start))
		}()
	}

	value, closer, err := s.db.Get(key)
	if err == pebble.ErrNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	// Copy value before returning
	result := make([]byte, len(value))
	copy(result, value)

	return result, nil
}

// Put stores a key-value pair.
func (s *Storage) Put(key, value []byte) error {
	// Check if closing
	if atomic.LoadInt32(&s.closing) != 0 {
		return storage.ErrStorageClosed
	}

	if s.metrics != nil {
		start := time.Now()
		defer func() {
			s.metrics.RecordPut(time.Since(start))
		}()
	}

	return s.db.Set(key, value, pebble.Sync)
}

// Delete removes a key-value pair.
func (s *Storage) Delete(key []byte) error {
	// Check if closing
	if atomic.LoadInt32(&s.closing) != 0 {
		return storage.ErrStorageClosed
	}

	if s.metrics != nil {
		s.metrics.RecordDelete()
	}

	return s.db.Delete(key, pebble.Sync)
}

// NewBatch creates a new batch for atomic updates.
func (s *Storage) NewBatch() storage.Batch {
	// Check if closing and add reference
	if !s.addRef() {
		return &errorBatch{err: storage.ErrStorageClosed}
	}

	return &batch{
		batch:   s.db.NewBatch(),
		storage: s,
	}
}

// NewIterator creates an iterator with the given options.
func (s *Storage) NewIterator(opts *storage.IteratorOptions) storage.Iterator {
	// Check if closing and add reference
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

	pebbleOpts := &pebble.IterOptions{}

	if opts != nil {
		if opts.Prefix != nil {
			// Set bounds for prefix iteration
			pebbleOpts.LowerBound = opts.Prefix
			pebbleOpts.UpperBound = prefixUpperBound(opts.Prefix)
		} else {
			pebbleOpts.LowerBound = opts.LowerBound
			pebbleOpts.UpperBound = opts.UpperBound
		}
	}

	iter, err := s.db.NewIter(pebbleOpts)
	if err != nil {
		// Clean up on error
		atomic.AddInt32(&s.openIterators, -1)
		s.releaseRef()
		return &errorIterator{err: err}
	}

	return &iterator{
		iter:    iter,
		storage: s,
	}
}

// NewSnapshot creates a point-in-time snapshot.
func (s *Storage) NewSnapshot() storage.Snapshot {
	// Check if closing and add reference
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

	return &snapshot{
		snap:    s.db.NewSnapshot(),
		storage: s,
	}
}

// Metrics returns storage metrics.
func (s *Storage) Metrics() storage.Metrics {
	if s.metrics == nil {
		return storage.NullMetrics{}
	}
	return s.metrics
}

// Close releases all resources.
func (s *Storage) Close() error {
	// Mark as closing atomically
	if !atomic.CompareAndSwapInt32(&s.closing, 0, 1) {
		return nil // Already closing
	}

	// Stop metrics collection
	if s.stopMetrics != nil {
		close(s.stopMetrics)
	}

	// Wait for metrics goroutine
	s.metricsWG.Wait()

	// Wait for all active resources to be released
	s.resourceWG.Wait()

	// Check for leaks (should be zero now)
	openIters := atomic.LoadInt32(&s.openIterators)
	openSnaps := atomic.LoadInt32(&s.openSnapshots)
	openRefs := atomic.LoadInt32(&s.refCount)
	if openIters > 0 || openSnaps > 0 || openRefs > 0 {
		// This indicates a bug in our reference counting
		fmt.Printf("ERROR: Storage closed with %d iterators, %d snapshots, %d refs\n", 
			openIters, openSnaps, openRefs)
	}

	// Now safe to close database
	var err error
	if s.db != nil {
		err = s.db.Close()
		s.db = nil
	}
	return err
}

// updateDatabaseMetrics periodically updates database size metrics.
func (s *Storage) updateDatabaseMetrics() {
	defer s.metricsWG.Done()
	
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopMetrics:
			return
		case <-ticker.C:
			// Check if we're closing
			if atomic.LoadInt32(&s.closing) != 0 {
				return
			}
			
			// Get metrics from database
			// This is safe because Close() waits for us to finish
			if s.db != nil {
				metrics := s.db.Metrics()
				diskSize := metrics.DiskSpaceUsage()
				if diskSize > 0 && s.metrics != nil {
					s.metrics.UpdateDatabaseSize(diskSize)
				}
			}
		}
	}
}

// batch implements storage.Batch using PebbleDB batch.
type batch struct {
	batch   *pebble.Batch
	storage *Storage
	closed  int32 // Atomic flag
}

func (b *batch) Put(key, value []byte) error {
	return b.batch.Set(key, value, nil)
}

func (b *batch) Delete(key []byte) error {
	return b.batch.Delete(key, nil)
}

func (b *batch) Commit(opts storage.CommitOptions) error {
	if atomic.LoadInt32(&b.closed) != 0 {
		return storage.ErrStorageClosed
	}

	if b.storage.metrics != nil {
		b.storage.metrics.RecordBatchCommit()
	}

	writeOpts := pebble.NoSync
	if opts.Sync {
		writeOpts = pebble.Sync
	}

	err := b.batch.Commit(writeOpts)
	// Release reference after commit
	b.close()
	return err
}

func (b *batch) close() {
	if atomic.CompareAndSwapInt32(&b.closed, 0, 1) {
		b.storage.releaseRef()
	}
}

func (b *batch) Close() error {
	b.close()
	return b.batch.Close()
}

// iterator implements storage.Iterator using PebbleDB iterator.
type iterator struct {
	iter    *pebble.Iterator
	storage *Storage
	closed  int32 // Atomic flag
}

func (i *iterator) SeekGE(key []byte) bool {
	return i.iter.SeekGE(key)
}

func (i *iterator) SeekLT(key []byte) bool {
	return i.iter.SeekLT(key)
}

func (i *iterator) First() bool {
	return i.iter.First()
}

func (i *iterator) Last() bool {
	return i.iter.Last()
}

func (i *iterator) Next() bool {
	return i.iter.Next()
}

func (i *iterator) Prev() bool {
	return i.iter.Prev()
}

func (i *iterator) Valid() bool {
	return i.iter.Valid()
}

func (i *iterator) Key() []byte {
	return i.iter.Key()
}

func (i *iterator) Value() []byte {
	return i.iter.Value()
}

func (i *iterator) Error() error {
	return i.iter.Error()
}

func (i *iterator) Close() error {
	if atomic.CompareAndSwapInt32(&i.closed, 0, 1) {
		err := i.iter.Close()
		atomic.AddInt32(&i.storage.openIterators, -1)
		i.storage.releaseRef()
		return err
	}
	return nil
}

// errorIterator is returned when iterator creation fails.
type errorIterator struct {
	err error
}

func (e *errorIterator) SeekGE(key []byte) bool { return false }
func (e *errorIterator) SeekLT(key []byte) bool { return false }
func (e *errorIterator) First() bool             { return false }
func (e *errorIterator) Last() bool              { return false }
func (e *errorIterator) Next() bool              { return false }
func (e *errorIterator) Prev() bool              { return false }
func (e *errorIterator) Valid() bool             { return false }
func (e *errorIterator) Key() []byte             { return nil }
func (e *errorIterator) Value() []byte           { return nil }
func (e *errorIterator) Error() error            { return e.err }
func (e *errorIterator) Close() error            { return nil }

// snapshot implements storage.Snapshot using PebbleDB snapshot.
type snapshot struct {
	snap    *pebble.Snapshot
	storage *Storage
	closed  int32 // Atomic flag
}

func (s *snapshot) Get(key []byte) ([]byte, error) {
	value, closer, err := s.snap.Get(key)
	if err == pebble.ErrNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	// Copy value before returning
	result := make([]byte, len(value))
	copy(result, value)

	return result, nil
}

func (s *snapshot) NewIterator(opts *storage.IteratorOptions) storage.Iterator {
	// Check if snapshot is closed
	if atomic.LoadInt32(&s.closed) != 0 {
		return &errorIterator{err: storage.ErrStorageClosed}
	}

	// Note: We don't add a new reference here because the snapshot
	// already holds a reference. We do track iterator count though.
	if s.storage.maxIterators > 0 {
		current := atomic.AddInt32(&s.storage.openIterators, 1)
		if current > s.storage.maxIterators {
			atomic.AddInt32(&s.storage.openIterators, -1)
			return &errorIterator{err: storage.ErrTooManyIterators}
		}
	} else {
		atomic.AddInt32(&s.storage.openIterators, 1)
	}

	pebbleOpts := &pebble.IterOptions{}

	if opts != nil {
		if opts.Prefix != nil {
			pebbleOpts.LowerBound = opts.Prefix
			pebbleOpts.UpperBound = prefixUpperBound(opts.Prefix)
		} else {
			pebbleOpts.LowerBound = opts.LowerBound
			pebbleOpts.UpperBound = opts.UpperBound
		}
	}

	iter, err := s.snap.NewIter(pebbleOpts)
	if err != nil {
		atomic.AddInt32(&s.storage.openIterators, -1)
		return &errorIterator{err: err}
	}

	// Create iterator that doesn't hold its own reference
	// (snapshot already holds one)
	return &snapshotIterator{
		iter:     iter,
		storage:  s.storage,
		snapshot: s,
	}
}

func (s *snapshot) Close() error {
	if atomic.CompareAndSwapInt32(&s.closed, 0, 1) {
		err := s.snap.Close()
		atomic.AddInt32(&s.storage.openSnapshots, -1)
		s.storage.releaseRef()
		return err
	}
	return nil
}

// snapshotIterator is an iterator created from a snapshot
type snapshotIterator struct {
	iter     *pebble.Iterator
	storage  *Storage
	snapshot *snapshot
	closed   int32
}

func (i *snapshotIterator) SeekGE(key []byte) bool { return i.iter.SeekGE(key) }
func (i *snapshotIterator) SeekLT(key []byte) bool { return i.iter.SeekLT(key) }
func (i *snapshotIterator) First() bool             { return i.iter.First() }
func (i *snapshotIterator) Last() bool              { return i.iter.Last() }
func (i *snapshotIterator) Next() bool              { return i.iter.Next() }
func (i *snapshotIterator) Prev() bool              { return i.iter.Prev() }
func (i *snapshotIterator) Valid() bool             { return i.iter.Valid() }
func (i *snapshotIterator) Key() []byte             { return i.iter.Key() }
func (i *snapshotIterator) Value() []byte           { return i.iter.Value() }
func (i *snapshotIterator) Error() error            { return i.iter.Error() }

func (i *snapshotIterator) Close() error {
	if atomic.CompareAndSwapInt32(&i.closed, 0, 1) {
		err := i.iter.Close()
		atomic.AddInt32(&i.storage.openIterators, -1)
		// Note: We don't release ref here - snapshot holds the ref
		return err
	}
	return nil
}

// errorBatch is returned when batch creation fails
type errorBatch struct {
	err error
}

func (e *errorBatch) Put(key, value []byte) error { return e.err }
func (e *errorBatch) Delete(key []byte) error     { return e.err }
func (e *errorBatch) Commit(opts storage.CommitOptions) error { return e.err }
func (e *errorBatch) Close() error                { return nil }

// errorSnapshot is returned when snapshot creation fails
type errorSnapshot struct {
	err error
}

func (e *errorSnapshot) Get(key []byte) ([]byte, error) { return nil, e.err }
func (e *errorSnapshot) NewIterator(opts *storage.IteratorOptions) storage.Iterator {
	return &errorIterator{err: e.err}
}
func (e *errorSnapshot) Close() error { return nil }

// prefixUpperBound returns the upper bound for prefix iteration.
func prefixUpperBound(prefix []byte) []byte {
	// Handle empty prefix
	if len(prefix) == 0 {
		return nil
	}

	// Create a copy to avoid modifying the original
	upper := make([]byte, len(prefix))
	copy(upper, prefix)

	// Increment the last byte that isn't 0xFF
	for i := len(upper) - 1; i >= 0; i-- {
		if upper[i] < 0xFF {
			upper[i]++
			return upper[:len(prefix)] // Maintain original length
		}
		// This byte is 0xFF, continue to the previous byte
	}

	// All bytes are 0xFF, return nil to indicate no upper bound
	return nil
}

