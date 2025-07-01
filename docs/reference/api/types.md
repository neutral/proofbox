# Types Package Public API

## Exported Constants

```go
const HashSize = 32
const (
    KeySize        = 32
    NibblePerByte  = 2
    MaxNibbleValue = 15
    MaxTreeDepth   = 64
    MaxValueSize   = 1 << 20
)
const (
    MaxVersion Version = ^Version(0) - 1
    InitialVersion Version = 0
    NodeKeyPrefix = "n"
    RootKeyPrefix = "r"
)
const (
    NodeTypeInternal NodeType = 0x00
    NodeTypeLeaf NodeType = 0x01
)
const (
    CodeUnknown ErrorCode = iota
    CodeCorruption
    CodeNotFound
    CodeInvalidInput
    CodeStorageFailure
    CodeResourceExhausted
    CodeTransactionConflict
)
```

## Exported Variables (Errors)

```go
var (
    ErrInvalidHashSize = errors.New("invalid hash size: must be 32 bytes")
)
var (
    ErrCorruptedNode = errors.New("jmt: corrupted node detected")
    ErrInvalidNodeType = errors.New("jmt: invalid node type")
    ErrHashMismatch = errors.New("jmt: hash mismatch")
    ErrInvalidProof = errors.New("jmt: invalid proof")
)
var (
    ErrKeyNotFound = errors.New("jmt: key not found")
    ErrEmptyValue = errors.New("jmt: empty value not allowed")
    ErrVersionNotFound = errors.New("jmt: version not found")
    ErrNodeNotFound = errors.New("jmt: node not found")
    ErrEmptyTree = errors.New("jmt: empty tree")
)
var (
    ErrInvalidKey = errors.New("jmt: invalid key")
    ErrEmptyKey = errors.New("jmt: empty key not allowed")
    ErrInvalidVersion = errors.New("jmt: invalid version")
    ErrValueTooLarge = errors.New("jmt: value too large")
    ErrInvalidNibble = errors.New("jmt: invalid nibble value")
    ErrInvalidNodeKey = errors.New("jmt: invalid node key")
    ErrInvalidNibbleCount = errors.New("jmt: nibble count exceeds maximum")
)
var (
    ErrMaxDepthExceeded = errors.New("jmt: maximum tree depth exceeded")
    ErrBatchTooLarge = errors.New("jmt: batch size exceeds limit")
    ErrOutOfMemory = errors.New("jmt: out of memory")
)
var (
    ErrInvalidVersionRange = errors.New("jmt: version exceeds maximum allowed value")
)
var (
    ErrStorageFailure = errors.New("jmt: storage operation failed")
    ErrTransactionAborted = errors.New("jmt: transaction aborted")
    ErrDatabaseClosed = errors.New("jmt: database closed")
)
var (
    ErrTxConflict = errors.New("jmt: transaction conflict")
    ErrTxStateInvalid = errors.New("jmt: invalid transaction state")
)
```

## Exported Types

```go
type Hash [HashSize]byte
type Key [32]byte
type Nibble uint8
type NibblePath struct {
    Nibbles []Nibble
    Length  uint16
}
type NodeKey struct {
    Version    Version
    NibblePath NibblePath
}
type Version uint64
type NodeType byte
type Child struct {
    Hash    Hash
    Version Version
    IsLeaf  bool
}
type ErrorInfo struct {
    Code      ErrorCode
    Message   string
    Details   map[string]interface{}
    Timestamp time.Time
    Stack     []byte
}
type ErrorCode int
```

## Exported Interfaces

```go
type ReadErrorHandler interface {
    HandleCorruptedNode(key NodeKey, err error) error
    HandleMissingNode(key NodeKey) error
}
type WriteErrorHandler interface {
    HandleWriteFailure(err error) error
    ShouldRetry(err error) bool
    RetryDelay(attempt int) time.Duration
}
type ErrorReporter interface {
    RecordError(code ErrorCode, err error)
    GetStats() map[ErrorCode]uint64
}
type HealthChecker interface {
    VerifyIntegrity(ctx context.Context, version Version) error
}
type Node interface {
    Type() NodeType
    Hash() Hash
    IsCached() bool
    Version() Version
}
type NodeWithChildren interface {
    Node
    Child(nibble Nibble) (Child, bool)
    Children() map[Nibble]Child
    NumChildren() int
    GetOnlyChild() (Nibble, Child, bool)
}
type NodeWithKey interface {
    Node
    Key() Key
}
type NodeCloneable interface {
    Node
    Clone(newVersion Version) Node
}
type LeafNodeInterface interface {
    Node
    NodeWithKey
    NodeCloneable
    ValueHash() Hash
    Value() []byte
    SetValue(value []byte) error
}
type InternalNodeInterface interface {
    Node
    NodeWithChildren
    NodeCloneable
    SetChild(nibble Nibble, child Child) error
    RemoveChild(nibble Nibble) error
}
type NodeVisitor interface {
    VisitLeaf(leaf LeafNodeInterface) error
    VisitInternal(internal InternalNodeInterface) error
}
```

## Exported Functions

```go
func EmptyHash() Hash
func HashFromBytes(b []byte) (Hash, error)
func HashFromHex(s string) (Hash, error)
func KeyFromBytes(b []byte) (Key, error)
func KeyHash(data []byte) Key
func ValidateKey(k Key) error
func ValidateNibble(n Nibble) error
func NewNibblePath(data []byte) NibblePath
func RootNodeKey(version Version) NodeKey
func EncodeNodeKey(key NodeKey) []byte
func DecodeNodeKey(buf []byte) (NodeKey, error)
func ValidateVersion(v Version) error
func EncodeVersion(v Version) [8]byte
func DecodeVersion(buf [8]byte) Version
func IsLeaf(n Node) bool
func IsInternal(n Node) bool
func WrapError(err error, format string, args ...interface{}) error
func NewErrorInfo(code ErrorCode, err error, details map[string]interface{}) *ErrorInfo
func IsRetryable(err error) bool
func GetErrorCode(err error) ErrorCode
```

## Exported Methods

### Hash Methods
```go
func (h Hash) Bytes() []byte
func (h Hash) String() string
func (h Hash) Equal(other Hash) bool
func (h Hash) IsEmpty() bool
```

### Key Methods
```go
func (k Key) String() string
func (k Key) Bytes() []byte
func (k Key) IsEmpty() bool
func (k Key) ExtractNibble(depth int) (Nibble, error)
func (k Key) ToNibblePath() NibblePath
```

### NibblePath Methods
```go
func (np NibblePath) String() string
func (np NibblePath) Compare(other NibblePath) int
func (np NibblePath) CommonPrefixLength(other NibblePath) int
func (np NibblePath) GetNibble(index int) (Nibble, error)
func (np NibblePath) Prefix(length int) NibblePath
func (np NibblePath) Equals(other NibblePath) bool
func (np NibblePath) Append(nibble Nibble) NibblePath
func (np NibblePath) IsPrefix(other NibblePath) bool
func (np NibblePath) Skip(count int) NibblePath
func (np NibblePath) ToBytes() ([]byte, error)
```

### NodeKey Methods
```go
func (nk NodeKey) Child(nibble Nibble, childVersion Version) NodeKey
func (nk NodeKey) IsRoot() bool
func (nk NodeKey) String() string
func (nk NodeKey) StorageKey() []byte
func (nk NodeKey) Compare(other NodeKey) int
func (nk NodeKey) WithPath(path NibblePath) NodeKey
func (nk NodeKey) ExtendPath(nibble Nibble) NodeKey
func (nk NodeKey) ParentKey() (NodeKey, error)
func (nk NodeKey) Depth() int
```

### Child Methods
```go
func (c Child) IsEmpty() bool
```

### ErrorInfo Methods
```go
func (e *ErrorInfo) Error() string
```