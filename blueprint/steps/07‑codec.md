---
id: step.08.codec
depends_on:
  - step.07.node‑iface
tags: [codec, step]
---

## Objective

Provide binary encode/decode for `LeafNode` and `InternalNode`.

## Implements

- **§5 Persistent Storage Requirements S1** (node is persisted under its NodeKey exactly once).
  _What happens_:

  - Binary coders ensure deterministic serialisation so identical nodes in different runs hash to the same byte stream—critical for network proofs and storage hashing.

## Technical Details

### Encoding Format Overview

The codec uses a simple, deterministic binary format:
- Fixed-size fields use big-endian encoding
- Variable-size fields are prefixed with their length
- No versioning or magic bytes (format is implicit)

### LeafNode Encoding

```go
// LeafNode binary format:
// [Key (32 bytes)] [ValueHash (32 bytes)]
// Total: 64 bytes (fixed size)

// EncodeLeafNode serializes a leaf node to bytes
func EncodeLeafNode(leaf *LeafNode) []byte {
    if leaf == nil {
        return nil
    }
    
    buf := make([]byte, 64)
    
    // Copy key (32 bytes)
    copy(buf[0:32], leaf.Key[:])
    
    // Copy value hash (32 bytes)
    copy(buf[32:64], leaf.ValueHash[:])
    
    return buf
}

// DecodeLeafNode deserializes a leaf node from bytes
func DecodeLeafNode(data []byte, leaf *LeafNode) error {
    if len(data) != 64 {
        return fmt.Errorf("invalid leaf data size: %d, expected 64", len(data))
    }
    
    // Decode key
    copy(leaf.Key[:], data[0:32])
    
    // Decode value hash
    copy(leaf.ValueHash[:], data[32:64])
    
    // Note: Nodes are immutable - cached fields should be handled
    // by the constructor when creating the node instance
    
    return nil
}

// LeafNodeSize returns the serialized size of a leaf node
func LeafNodeSize() int {
    return 64 // Always fixed size
}
```

### InternalNode Encoding

```go
// InternalNode binary format:
// [NumChildren (1 byte)] [Children data...]
// 
// Each child:
// [Nibble (1 byte)] [Hash (32 bytes)] [Version (8 bytes)] [IsLeaf (1 byte)]
// Total per child: 42 bytes

// EncodeInternalNode serializes an internal node to bytes
func EncodeInternalNode(node *InternalNode) ([]byte, error) {
    if node == nil {
        return nil, nil
    }
    
    numChildren := len(node.Children)
    if numChildren > 16 {
        return nil, fmt.Errorf("invalid internal node: too many children (%d > 16)", numChildren)
    }
    
    // Calculate buffer size
    bufSize := 1 + numChildren*42
    buf := make([]byte, bufSize)
    
    // Write number of children
    buf[0] = byte(numChildren)
    
    // Get sorted nibbles for deterministic encoding
    nibbles := node.ChildNibbles()
    
    // Write each child
    offset := 1
    for _, nibble := range nibbles {
        child := node.Children[nibble]
        
        // Write nibble (1 byte)
        buf[offset] = byte(nibble)
        offset++
        
        // Write hash (32 bytes)
        copy(buf[offset:offset+32], child.Hash[:])
        offset += 32
        
        // Write version (8 bytes, big-endian)
        binary.BigEndian.PutUint64(buf[offset:offset+8], uint64(child.Version))
        offset += 8
        
        // Write is_leaf flag (1 byte)
        if child.IsLeaf {
            buf[offset] = 1
        } else {
            buf[offset] = 0
        }
        offset++
    }
    
    return buf[:offset], nil
}

// DecodeInternalNode deserializes an internal node from bytes
func DecodeInternalNode(data []byte, node *InternalNode) error {
    if len(data) < 1 {
        return errors.New("empty internal node data")
    }
    
    // Read number of children
    numChildren := int(data[0])
    if numChildren > 16 {
        return fmt.Errorf("too many children: %d", numChildren)
    }
    
    // Verify data size
    expectedSize := 1 + numChildren*42
    if len(data) != expectedSize {
        return fmt.Errorf("invalid data size: %d, expected %d", len(data), expectedSize)
    }
    
    // Initialize children map
    node.Children = make(map[Nibble]ChildMeta, numChildren)
    
    // Read each child
    offset := 1
    for i := 0; i < numChildren; i++ {
        // Read nibble
        nibble := Nibble(data[offset])
        if nibble > 15 {
            return fmt.Errorf("invalid nibble: %d", nibble)
        }
        offset++
        
        // Read hash
        var hash Hash
        copy(hash[:], data[offset:offset+32])
        offset += 32
        
        // Read version
        version := Version(binary.BigEndian.Uint64(data[offset : offset+8]))
        offset += 8
        
        // Read is_leaf flag
        isLeaf := data[offset] == 1
        offset++
        
        // Store child
        node.Children[nibble] = ChildMeta{
            Hash:    hash,
            Version: version,
            IsLeaf:  isLeaf,
        }
    }
    
    // Note: Nodes are immutable - cached fields should be handled
    // by the constructor when creating the node instance
    
    return nil
}

// InternalNodeSize calculates the serialized size of an internal node
func InternalNodeSize(node *InternalNode) int {
    return 1 + len(node.Children)*42
}
```

### Generic Node Codec

```go
// NodeCodec provides encoding/decoding for any node type
type NodeCodec struct{}

// EncodeNode serializes any node with type prefix
func (c *NodeCodec) EncodeNode(node Node) ([]byte, error) {
    if node == nil {
        return nil, errors.New("cannot encode nil node")
    }
    
    switch n := node.(type) {
    case *LeafNode:
        data := EncodeLeafNode(n)
        // Prepend type byte
        result := make([]byte, 1+len(data))
        result[0] = byte(NodeTypeLeaf)
        copy(result[1:], data)
        return result, nil
        
    case *InternalNode:
        data, err := EncodeInternalNode(n)
        if err != nil {
            return nil, err
        }
        // Prepend type byte
        result := make([]byte, 1+len(data))
        result[0] = byte(NodeTypeInternal)
        copy(result[1:], data)
        return result, nil
        
    default:
        return nil, fmt.Errorf("unknown node type: %T", node)
    }
}

// DecodeNode deserializes a node from bytes
func (c *NodeCodec) DecodeNode(data []byte) (Node, error) {
    if len(data) < 1 {
        return nil, errors.New("empty node data")
    }
    
    nodeType := NodeType(data[0])
    nodeData := data[1:]
    
    switch nodeType {
    case NodeTypeLeaf:
        var leaf LeafNode
        if err := DecodeLeafNode(nodeData, &leaf); err != nil {
            return nil, err
        }
        return &leaf, nil
        
    case NodeTypeInternal:
        var internal InternalNode
        if err := DecodeInternalNode(nodeData, &internal); err != nil {
            return nil, err
        }
        return &internal, nil
        
    default:
        return nil, fmt.Errorf("unknown node type: %d", nodeType)
    }
}

// EstimateSize returns the estimated serialized size of a node
func (c *NodeCodec) EstimateSize(node Node) int {
    if node == nil {
        return 0
    }
    
    // 1 byte for type prefix
    switch n := node.(type) {
    case *LeafNode:
        return 1 + LeafNodeSize()
    case *InternalNode:
        return 1 + InternalNodeSize(n)
    default:
        return 0
    }
}
```

### Batch Encoding

```go
// BatchEncoder efficiently encodes multiple nodes
type BatchEncoder struct {
    buffer bytes.Buffer
    codec  NodeCodec
}

// Add encodes and adds a node to the batch
func (b *BatchEncoder) Add(key NodeKey, node Node) error {
    // Encode node key
    keyData := EncodeNodeKey(key)
    if err := binary.Write(&b.buffer, binary.BigEndian, uint32(len(keyData))); err != nil {
        return err
    }
    b.buffer.Write(keyData)
    
    // Encode node
    nodeData, err := b.codec.EncodeNode(node)
    if err != nil {
        return err
    }
    if err := binary.Write(&b.buffer, binary.BigEndian, uint32(len(nodeData))); err != nil {
        return err
    }
    b.buffer.Write(nodeData)
    
    return nil
}

// Bytes returns the encoded batch data
func (b *BatchEncoder) Bytes() []byte {
    return b.buffer.Bytes()
}

// Reset clears the buffer for reuse
func (b *BatchEncoder) Reset() {
    b.buffer.Reset()
}
```

## Implementation Steps

1. **Define encoding formats**: Fixed layouts for each node type
2. **Implement leaf codec**: Simple 64-byte fixed format
3. **Implement internal codec**: Variable size based on children
4. **Add generic codec**: Type prefix for polymorphic handling
5. **Create batch encoder**: Efficient multi-node encoding
6. **Add comprehensive tests**: Round-trip and edge cases

## Testing Requirements

### Unit Tests

```go
func TestLeafNodeCodec(t *testing.T) {
    original := &LeafNode{
        Key:       Key{0x01, 0x02, 0x03, /* ... */},
        ValueHash: Hash{0xAA, 0xBB, 0xCC, /* ... */},
    }
    
    // Encode
    data := EncodeLeafNode(original)
    if len(data) != 64 {
        t.Errorf("Wrong encoded size: %d", len(data))
    }
    
    // Decode
    var decoded LeafNode
    if err := DecodeLeafNode(data, &decoded); err != nil {
        t.Fatalf("Decode failed: %v", err)
    }
    
    // Compare
    if decoded.Key != original.Key {
        t.Error("Key mismatch after round-trip")
    }
    if decoded.ValueHash != original.ValueHash {
        t.Error("ValueHash mismatch after round-trip")
    }
}

func TestInternalNodeCodec(t *testing.T) {
    original := NewInternalNode()
    original.SetChild(0x3, ChildMeta{
        Hash:    Hash{0x11, 0x22, /* ... */},
        Version: 100,
        IsLeaf:  true,
    })
    original.SetChild(0xA, ChildMeta{
        Hash:    Hash{0x33, 0x44, /* ... */},
        Version: 200,
        IsLeaf:  false,
    })
    
    // Encode
    data, err := EncodeInternalNode(original)
    if err != nil {
        t.Fatalf("Failed to encode: %v", err)
    }
    expectedSize := 1 + 2*42 // 1 byte count + 2 children
    if len(data) != expectedSize {
        t.Errorf("Wrong encoded size: %d, expected %d", len(data), expectedSize)
    }
    
    // Decode
    var decoded InternalNode
    if err := DecodeInternalNode(data, &decoded); err != nil {
        t.Fatalf("Decode failed: %v", err)
    }
    
    // Compare children
    if len(decoded.Children) != 2 {
        t.Errorf("Wrong number of children: %d", len(decoded.Children))
    }
    
    child3, ok := decoded.GetChild(0x3)
    if !ok || child3.Version != 100 || !child3.IsLeaf {
        t.Error("Child 0x3 mismatch")
    }
    
    childA, ok := decoded.GetChild(0xA)
    if !ok || childA.Version != 200 || childA.IsLeaf {
        t.Error("Child 0xA mismatch")
    }
}
```

### Determinism Tests

```go
func TestDeterministicEncoding(t *testing.T) {
    // Create node with children in random order
    node := NewInternalNode()
    nibbles := []Nibble{0xF, 0x0, 0x7, 0x3, 0xA}
    
    for _, n := range nibbles {
        node.SetChild(n, ChildMeta{
            Hash:    Hash{byte(n), /* ... */},
            Version: Version(n),
            IsLeaf:  n%2 == 0,
        })
    }
    
    // Encode multiple times
    encoding1, err1 := EncodeInternalNode(node)
    if err1 != nil {
        t.Fatalf("First encoding failed: %v", err1)
    }
    encoding2, err2 := EncodeInternalNode(node)
    if err2 != nil {
        t.Fatalf("Second encoding failed: %v", err2)
    }
    
    // Should be identical
    if !bytes.Equal(encoding1, encoding2) {
        t.Error("Encoding not deterministic")
    }
}
```

## Performance Considerations

- **Fixed-size leaves**: 64 bytes allows exact allocation
- **Sorted children**: Deterministic ordering for internal nodes
- **No compression**: Simple format optimized for speed
- **Batch encoding**: Amortize allocation costs

## Security Notes

- **Deterministic encoding**: Same node always encodes identically
- **Size validation**: Reject malformed data before allocation
- **Nibble validation**: Ensure nibbles are in valid range
- **No type confusion**: Type byte prevents misinterpretation

## Done When ✓

- [ ] Decode(Encode(x)) deep‑equals `x` in tests
- [ ] LeafNode codec with fixed 64-byte format
- [ ] InternalNode codec with variable child encoding
- [ ] Generic NodeCodec with type prefix handling
- [ ] Deterministic encoding (sorted children)
- [ ] Size validation prevents malicious inputs
- [ ] Batch encoder for efficient multi-node operations
- [ ] 100% test coverage including edge cases
- [ ] Benchmarks show minimal allocation overhead
