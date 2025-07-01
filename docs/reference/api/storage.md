# Storage Package Public API

## Exported Types/Structs

### From storage package
```go
type Config struct
type PebbleConfig struct
type MetricsConfig struct
type PruningConfig struct
type PruningPolicyConfig struct
type Storage interface
type Batch interface
type CommitOptions struct
type Iterator interface
type IteratorOptions struct
type Snapshot interface
type Metrics interface
type KeyEncoder interface
type DefaultKeyEncoder struct
type MetricsCollector struct
type NullMetrics struct
type PruningPolicy interface
type VersionInfo struct
type KeepLastNVersions struct
type KeepVersionsNewerThan struct
type KeepMilestoneVersions struct
type CompositePruningPolicy struct
```

### From memory subpackage
```go
type Storage struct
type Options struct
```

### From pebble subpackage
```go
type Storage struct
type Options struct
```

## Exported Functions

### Storage package functions
```go
func DefaultConfig(path string) Config
func MemoryConfig() Config
func NewDefaultKeyEncoder() KeyEncoder
func NewMetricsCollector() *MetricsCollector
func NewKeepLastNVersions(n int, currentVersion types.Version) PruningPolicy
func NewKeepVersionsNewerThan(age time.Duration) PruningPolicy
func NewKeepMilestoneVersions(interval int, currentVersion types.Version) PruningPolicy
func NewCompositePruningPolicy(all bool, policies ...PruningPolicy) PruningPolicy
```

### Memory subpackage functions
```go
func DefaultOptions() *Options
func NewStorage() *Storage
func NewStorageWithOptions(opts *Options) *Storage
```

### Pebble subpackage functions
```go
func DefaultOptions() *Options
func NewStorage(path string, opts *Options) (*Storage, error)
```

## Exported Methods

### Config methods
```go
func (c *Config) Validate() error
```

### DefaultKeyEncoder methods
```go
func (e *DefaultKeyEncoder) NodeKey(k types.NodeKey) []byte
func (e *DefaultKeyEncoder) ParseNodeKey(key []byte) (types.NodeKey, error)
func (e *DefaultKeyEncoder) ValueKey(hash types.Hash) []byte
func (e *DefaultKeyEncoder) ValueKeyByKey(key types.Key) []byte
func (e *DefaultKeyEncoder) RootKey(version types.Version) []byte
func (e *DefaultKeyEncoder) ParseRootKey(key []byte) (types.Version, error)
func (e *DefaultKeyEncoder) RootKeyPrefix() []byte
```

### MetricsCollector methods
```go
func (m *MetricsCollector) RecordGet(latency time.Duration)
func (m *MetricsCollector) RecordPut(latency time.Duration)
func (m *MetricsCollector) RecordDelete()
func (m *MetricsCollector) RecordBatchCommit()
func (m *MetricsCollector) RecordCacheHit()
func (m *MetricsCollector) RecordCacheMiss()
func (m *MetricsCollector) UpdateDatabaseSize(size uint64)
func (m *MetricsCollector) UpdateLiveDataSize(size uint64)
func (m *MetricsCollector) UpdateCacheSize(size uint64)
func (m *MetricsCollector) GetOperations() uint64
func (m *MetricsCollector) PutOperations() uint64
func (m *MetricsCollector) DeleteOperations() uint64
func (m *MetricsCollector) BatchCommits() uint64
func (m *MetricsCollector) GetLatencyP50() uint64
func (m *MetricsCollector) GetLatencyP99() uint64
func (m *MetricsCollector) PutLatencyP50() uint64
func (m *MetricsCollector) PutLatencyP99() uint64
func (m *MetricsCollector) DatabaseSize() uint64
func (m *MetricsCollector) LiveDataSize() uint64
func (m *MetricsCollector) CacheHitRate() float64
func (m *MetricsCollector) CacheSize() uint64
```

### NullMetrics methods
```go
func (n NullMetrics) GetOperations() uint64
func (n NullMetrics) PutOperations() uint64
func (n NullMetrics) DeleteOperations() uint64
func (n NullMetrics) BatchCommits() uint64
func (n NullMetrics) GetLatencyP50() uint64
func (n NullMetrics) GetLatencyP99() uint64
func (n NullMetrics) PutLatencyP50() uint64
func (n NullMetrics) PutLatencyP99() uint64
func (n NullMetrics) DatabaseSize() uint64
func (n NullMetrics) LiveDataSize() uint64
func (n NullMetrics) CacheHitRate() float64
func (n NullMetrics) CacheSize() uint64
```

### KeepLastNVersions methods
```go
func (p *KeepLastNVersions) ShouldPrune(version types.Version, info VersionInfo) bool
```

### KeepVersionsNewerThan methods
```go
func (p *KeepVersionsNewerThan) ShouldPrune(version types.Version, info VersionInfo) bool
```

### KeepMilestoneVersions methods
```go
func (p *KeepMilestoneVersions) ShouldPrune(version types.Version, info VersionInfo) bool
```

### CompositePruningPolicy methods
```go
func (p *CompositePruningPolicy) ShouldPrune(version types.Version, info VersionInfo) bool
```

### memory.Storage methods
```go
func (s *Storage) Get(key []byte) ([]byte, error)
func (s *Storage) Put(key, value []byte) error
func (s *Storage) Delete(key []byte) error
func (s *Storage) NewBatch() storage.Batch
func (s *Storage) NewIterator(opts *storage.IteratorOptions) storage.Iterator
func (s *Storage) NewSnapshot() storage.Snapshot
func (s *Storage) Metrics() storage.Metrics
func (s *Storage) Close() error
```

### pebble.Storage methods
```go
func (s *Storage) Get(key []byte) ([]byte, error)
func (s *Storage) Put(key, value []byte) error
func (s *Storage) Delete(key []byte) error
func (s *Storage) NewBatch() storage.Batch
func (s *Storage) NewIterator(opts *storage.IteratorOptions) storage.Iterator
func (s *Storage) NewSnapshot() storage.Snapshot
func (s *Storage) Metrics() storage.Metrics
func (s *Storage) Close() error
```

## Exported Variables (Errors)

```go
var ErrInvalidBackend = errors.New("invalid storage backend")
var ErrPathRequired = errors.New("storage path required for persistent backend")
var ErrInvalidCacheSize = errors.New("cache size must be non-negative")
var ErrInvalidWriteBufferSize = errors.New("write buffer size must be non-negative")
var ErrInvalidCompression = errors.New("invalid compression type")
var ErrInvalidPruningPolicy = errors.New("invalid pruning policy type")
var ErrStorageClosed = errors.New("storage is closed")
var ErrKeyNotFound = errors.New("key not found")
var ErrInvalidKey = errors.New("invalid key")
var ErrInvalidValue = errors.New("invalid value")
var ErrTooManyIterators = errors.New("too many open iterators")
var ErrTooManySnapshots = errors.New("too many open snapshots")
```