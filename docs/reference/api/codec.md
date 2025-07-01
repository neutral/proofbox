# Codec Package Public API

## Exported Types/Structs

```go
type BatchEncoder struct
type BatchDecoder struct
type NodeFactory func(nodeType types.NodeType, data []byte, version types.Version) (types.Node, error)
type NodeCodec struct{}
```

## Exported Functions

```go
func NewBatchEncoder() *BatchEncoder
func NewBatchDecoder(data []byte) *BatchDecoder
func RegisterNodeFactory(f NodeFactory)
func DecodeNode(data []byte, version types.Version) (types.Node, error)
func EncodeLeafNodeInterface(leaf types.LeafNodeInterface) []byte
func EncodeInternalNodeInterface(node types.InternalNodeInterface) ([]byte, error)
func InternalNodeInterfaceSize(node types.InternalNodeInterface) int
func EncodeInternalNode(node types.InternalNodeInterface) ([]byte, error)
func DecodeInternalNode(data []byte, version types.Version) (types.InternalNodeInterface, error)
func InternalNodeSize(node types.InternalNodeInterface) int
func EncodeLeafNode(leaf types.LeafNodeInterface) []byte
func DecodeLeafNode(data []byte, version types.Version) (types.LeafNodeInterface, error)
func LeafNodeSize() int
```

## Exported Methods

### BatchEncoder methods
```go
func (b *BatchEncoder) Add(key types.NodeKey, node types.Node) error
func (b *BatchEncoder) Bytes() []byte
func (b *BatchEncoder) Len() int
func (b *BatchEncoder) Reset()
```

### BatchDecoder methods
```go
func (d *BatchDecoder) Next() (types.NodeKey, types.Node, error)
func (d *BatchDecoder) HasMore() bool
```

### NodeCodec methods
```go
func (c *NodeCodec) EncodeNode(node types.Node) ([]byte, error)
func (c *NodeCodec) DecodeNode(data []byte, version types.Version) (types.Node, error)
func (c *NodeCodec) EstimateSize(node types.Node) int
func (c *NodeCodec) EncodeNodeWithKey(key types.NodeKey, node types.Node) ([]byte, []byte, error)
```