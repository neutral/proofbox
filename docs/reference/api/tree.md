# Tree Package Public API

## Exported Types/Structs

```go
type VersionStatus int
type VersionInfo struct
type PendingVersion struct
type VersionRetentionPolicy int
type VersionManager struct
type PathCloner struct
type InternalNode struct
type UpdateBatch struct
type NodeWrite struct
type TreeUpdater struct
type LeafNode struct
type DefaultReadErrorHandler struct
type DefaultWriteErrorHandler struct
type BatchOptimizer struct
type BatchOptimizerConfig struct
type BatchStats struct
type NodeCache struct
type TreeHealthChecker struct
type TreeReaderInterface interface
type TreeReader struct
type TreeStats struct
type Tree struct
type TreeConfig struct
type BatchOperation struct
type BatchOpType int
type BatchTransaction struct
```

## Exported Functions

```go
func NewVersionManager() *VersionManager
func NewVersionManagerWithRetention(policy VersionRetentionPolicy, minVersions int, maxAge time.Duration) *VersionManager
func NewPathCloner(tree *Tree, sourceVersion, targetVersion types.Version) *PathCloner
func NewInternalNode(version types.Version) *InternalNode
func NewTreeUpdater(tree *Tree, oldVersion, newVersion types.Version) *TreeUpdater
func NewLeafNode(key types.Key, value []byte, version types.Version) (*LeafNode, error)
func AsLeaf(n types.Node) (*LeafNode, bool)
func AsInternal(n types.Node) (*InternalNode, bool)
func AsLeafErr(n types.Node) (*LeafNode, error)
func AsInternalErr(n types.Node) (*InternalNode, error)
func PrintNode(n types.Node, indent string)
func NewLeafNodeFromCodec(key types.Key, valueHash types.Hash, version types.Version) *LeafNode
func NewInternalNodeFromCodec(children map[types.Nibble]types.Child, version types.Version) *InternalNode
func NewDefaultReadErrorHandler(logger *log.Logger) *DefaultReadErrorHandler
func NewDefaultWriteErrorHandler(maxRetries int, baseDelay time.Duration, logger *log.Logger) *DefaultWriteErrorHandler
func DefaultBatchOptimizerConfig() BatchOptimizerConfig
func NewBatchOptimizer(config BatchOptimizerConfig) *BatchOptimizer
func WrapError(err error, format string, args ...interface{}) error
func ValidateTreeStructure(tree *Tree, version types.Version) error
func NewNodeCache(maxSize int) (*NodeCache, error)
func NewTreeHealthChecker(tree *Tree) *TreeHealthChecker
func NewTreeStats() *TreeStats
func DefaultTreeConfig() TreeConfig
func NewTree(db storage.Storage, keyEncoder storage.KeyEncoder, config TreeConfig) (*Tree, error)
func NewBatchTransaction(tree *Tree) *BatchTransaction
```

## Exported Methods

### PendingVersion methods
```go
func (pv *PendingVersion) GetUpdater() *TreeUpdater
```

### VersionManager methods
```go
func (vm *VersionManager) Begin(parentVersion types.Version) (types.Version, error)
func (vm *VersionManager) Commit(version types.Version, rootHash types.Hash, nodeCount int64) error
func (vm *VersionManager) Abort(version types.Version) error
func (vm *VersionManager) GetVersion(version types.Version) (*VersionInfo, error)
func (vm *VersionManager) GetLatestCommitted() types.Version
func (vm *VersionManager) GetPending(version types.Version) *PendingVersion
func (vm *VersionManager) SetPendingUpdater(version types.Version, updater *TreeUpdater) error
func (vm *VersionManager) GetAllVersions() []*VersionInfo
func (vm *VersionManager) CollectGarbage() ([]types.Version, error)
func (vm *VersionManager) SetRetentionPolicy(policy VersionRetentionPolicy, minVersions int, maxAge time.Duration)
func (vm *VersionManager) RemoveVersion(version types.Version) error
func (vm *VersionManager) IncrementOperations(version types.Version) error
```

### PathCloner methods
```go
func (pc *PathCloner) RegisterClone(oldKey, newKey types.NodeKey)
func (pc *PathCloner) IsCloned(oldKey types.NodeKey) (types.NodeKey, bool)
func (pc *PathCloner) ClonePath(path []types.NodeKey) error
```

### InternalNode methods
```go
func (n *InternalNode) Type() types.NodeType
func (n *InternalNode) Hash() types.Hash
func (n *InternalNode) IsCached() bool
func (n *InternalNode) Version() types.Version
func (n *InternalNode) Child(nibble types.Nibble) (types.Child, bool)
func (n *InternalNode) SetChild(nibble types.Nibble, child types.Child) error
func (n *InternalNode) RemoveChild(nibble types.Nibble) error
func (n *InternalNode) NumChildren() int
func (n *InternalNode) Children() map[types.Nibble]types.Child
func (n *InternalNode) GetOnlyChild() (types.Nibble, types.Child, bool)
func (n *InternalNode) Clone(newVersion types.Version) types.Node
```

### TreeUpdater methods
```go
func (tu *TreeUpdater) Put(key types.Key, value []byte) (types.Hash, error)
func (tu *TreeUpdater) Delete(key types.Key) error
func (tu *TreeUpdater) BuildUpdateBatch() (*UpdateBatch, error)
func (tu *TreeUpdater) BuildUpdateBatchParallel() (*UpdateBatch, error)
func (tu *TreeUpdater) ValidateBatch(batch *UpdateBatch) error
```

### LeafNode methods
```go
func (n *LeafNode) Type() types.NodeType
func (n *LeafNode) Hash() types.Hash
func (n *LeafNode) IsCached() bool
func (n *LeafNode) Version() types.Version
func (n *LeafNode) Key() types.Key
func (n *LeafNode) ValueHash() types.Hash
func (n *LeafNode) Value() []byte
func (n *LeafNode) SetValue(value []byte) error
func (n *LeafNode) Clone(newVersion types.Version) types.Node
```

### DefaultReadErrorHandler methods
```go
func (h *DefaultReadErrorHandler) HandleCorruptedNode(key types.NodeKey, err error) error
func (h *DefaultReadErrorHandler) HandleMissingNode(key types.NodeKey) error
```

### DefaultWriteErrorHandler methods
```go
func (h *DefaultWriteErrorHandler) HandleWriteFailure(err error) error
func (h *DefaultWriteErrorHandler) ShouldRetry(err error) bool
func (h *DefaultWriteErrorHandler) RetryDelay(attempt int) time.Duration
```

### BatchOptimizer methods
```go
func (bo *BatchOptimizer) OptimizeBatch(batch *UpdateBatch) (*UpdateBatch, error)
func (bo *BatchOptimizer) CompressBatch(batch *UpdateBatch) ([]byte, error)
func (bo *BatchOptimizer) DecompressBatch(data []byte) (*UpdateBatch, error)
func (bo *BatchOptimizer) GetBatchStats(batch *UpdateBatch) (*BatchStats, error)
```

### NodeCache methods
```go
func (c *NodeCache) Get(key types.NodeKey) (types.Node, bool)
func (c *NodeCache) Put(key types.NodeKey, node types.Node)
func (c *NodeCache) Remove(key types.NodeKey)
func (c *NodeCache) Clear()
func (c *NodeCache) Stats() (hits, misses uint64, size int)
func (c *NodeCache) HitRate() float64
```

### TreeHealthChecker methods
```go
func (hc *TreeHealthChecker) VerifyIntegrity(ctx context.Context, version types.Version) error
func (hc *TreeHealthChecker) CheckVersion(version types.Version) error
```

### TreeReader methods
```go
func (r *TreeReader) Get(key types.Key) ([]byte, error)
func (r *TreeReader) GetNode(key types.NodeKey) (types.Node, error)
func (r *TreeReader) LoadValue(hash types.Hash) ([]byte, error)
func (r *TreeReader) RootHash() types.Hash
func (r *TreeReader) Version() types.Version
func (r *TreeReader) Metrics() metrics.JMTMetrics
func (r *TreeReader) Close() error
```

### TreeStats methods
```go
func (ts *TreeStats) UpdateStats(height, nodeCount, version int64)
func (ts *TreeStats) GetHeight() int64
func (ts *TreeStats) GetNodeCount() int64
func (ts *TreeStats) GetVersion() int64
func (ts *TreeStats) UpdateHeight(height int64)
func (ts *TreeStats) UpdateNodeCount(count int64)
func (ts *TreeStats) UpdateVersion(version int64)
```

### Tree methods
```go
func (t *Tree) Get(version types.Version, key types.Key) ([]byte, error)
func (t *Tree) Delete(key types.Key) (types.Version, error)
func (t *Tree) GetLatestVersion() types.Version
func (t *Tree) HasVersion(version types.Version) bool
func (t *Tree) GetRootHash(version types.Version) (types.Hash, error)
func (t *Tree) IsEmpty(version types.Version) (bool, error)
func (t *Tree) Reader(version types.Version) (TreeReaderInterface, error)
func (t *Tree) BeginVersion() (types.Version, error)
func (t *Tree) PutVersioned(version types.Version, key types.Key, value []byte) error
func (t *Tree) DeleteVersioned(version types.Version, key types.Key) error
func (t *Tree) GetAtVersion(version types.Version, key types.Key) ([]byte, error)
func (t *Tree) CommitVersion(version types.Version) error
func (t *Tree) AbortVersion(version types.Version) error
func (t *Tree) GetStats() (height, nodeCount, version int64)
func (t *Tree) CollectVersionGarbage() ([]types.Version, error)
func (t *Tree) SetVersionRetentionPolicy(policy VersionRetentionPolicy, minVersions int, maxAge time.Duration)
func (t *Tree) NewBatchTransaction() *BatchTransaction
func (t *Tree) GetVersionManager() *VersionManager
func (t *Tree) Put(key types.Key, value []byte) (types.Version, error)
```

### BatchTransaction methods
```go
func (bt *BatchTransaction) BatchPut(key types.Key, value []byte) error
func (bt *BatchTransaction) BatchDelete(key types.Key) error
func (bt *BatchTransaction) Size() int
func (bt *BatchTransaction) GetOperations() []BatchOperation
func (bt *BatchTransaction) OptimizeOperations() []BatchOperation
func (bt *BatchTransaction) Execute() (types.Version, error)
func (bt *BatchTransaction) Clear()
```

## Exported Constants

```go
const (
    VersionStatusPending VersionStatus = iota
    VersionStatusCommitted
    VersionStatusAborted
)

const (
    RetentionPolicyCount VersionRetentionPolicy = iota
    RetentionPolicyTime
    RetentionPolicyNone
)

const (
    BatchOpPut BatchOpType = iota
    BatchOpDelete
)
```