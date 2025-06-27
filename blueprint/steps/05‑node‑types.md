---
id: step.05.node‑types
depends_on:
  - step.04.keys‑and‑paths
tags: [tree, node, step]
---

## Objective

Implement LeafNode and InternalNode types as the only two node types in the Jellyfish Merkle Tree, establishing the core data structure.

## Implements

- **Node Types Specification** (`/blueprint/features/storage/get/specs/node-types.md`)
- **§4.1 Node Types** - Internal nodes with up to 16 children, Leaf nodes with key-value pairs
- **Design principle**: "Less Complexity – only two node types" (no extension nodes)

## Technical Details

### Node Interface

Create `pkg/tree/node.go`:
```go
package tree

import (
    "github.com/acme/jmt/pkg/crypto"
    "github.com/acme/jmt/pkg/types"
)

// NodeType identifies the type of node
type NodeType byte

const (
    NodeTypeInternal NodeType = 0x00
    NodeTypeLeaf     NodeType = 0x01
)

// Node represents a node in the Jellyfish Merkle Tree
type Node interface {
    // Type returns the node type
    Type() NodeType
    
    // Hash computes and returns the node's hash
    Hash() types.Hash
    
    // IsCached returns true if the hash is already computed
    IsCached() bool
    
    // Version returns the version when this node was created
    Version() types.Version
}

// Child represents a reference to a child node
type Child struct {
    Hash    types.Hash    // The hash of the child node
    Version types.Version // Version when child was created
    IsLeaf  bool         // Whether the child is a leaf
}

// IsEmpty checks if child reference is empty
func (c Child) IsEmpty() bool {
    return c.Hash == types.EmptyHash()
}
```

### Leaf Node Implementation

Create `pkg/tree/leaf_node.go`:
```go
package tree

import (
    "sync"
    
    "github.com/acme/jmt/pkg/crypto"
    "github.com/acme/jmt/pkg/types"
)

// LeafNode represents a leaf in the tree containing actual data
type LeafNode struct {
    key       types.Key    // The key this leaf represents
    valueHash types.Hash   // Hash of the value
    value     []byte       // The actual value (may be nil if not loaded)
    version   types.Version
    
    // Cache
    mu         sync.RWMutex
    cachedHash *types.Hash
}

// NewLeafNode creates a new leaf node
func NewLeafNode(key types.Key, value []byte, version types.Version) (*LeafNode, error) {
    if err := types.ValidateKey(key); err != nil {
        return nil, err
    }
    
    if len(value) > types.MaxValueSize {
        return nil, types.ErrValueTooLarge
    }
    
    valueHash := crypto.DefaultHasher.Hash(value)
    
    return &LeafNode{
        key:       key,
        valueHash: valueHash,
        value:     value,
        version:   version,
    }, nil
}

// Type returns NodeTypeLeaf
func (n *LeafNode) Type() NodeType {
    return NodeTypeLeaf
}

// Hash computes the leaf node hash
// Format: NodeTypeLeaf || key || valueHash
func (n *LeafNode) Hash() types.Hash {
    n.mu.RLock()
    if n.cachedHash != nil {
        defer n.mu.RUnlock()
        return *n.cachedHash
    }
    n.mu.RUnlock()
    
    // Compute hash
    hash := crypto.DefaultHasher.HashConcat(
        []byte{byte(NodeTypeLeaf)},
        n.key[:],
        n.valueHash[:],
    )
    
    // Cache it
    n.mu.Lock()
    n.cachedHash = &hash
    n.mu.Unlock()
    
    return hash
}

// IsCached returns true if hash is already computed
func (n *LeafNode) IsCached() bool {
    n.mu.RLock()
    defer n.mu.RUnlock()
    return n.cachedHash != nil
}

// Version returns the version when this leaf was created
func (n *LeafNode) Version() types.Version {
    return n.version
}

// Key returns the leaf's key
func (n *LeafNode) Key() types.Key {
    return n.key
}

// ValueHash returns the hash of the value
func (n *LeafNode) ValueHash() types.Hash {
    return n.valueHash
}

// Value returns the actual value (may be nil if not loaded)
func (n *LeafNode) Value() []byte {
    return n.value
}

// SetValue updates the value (used when loading from storage)
func (n *LeafNode) SetValue(value []byte) error {
    if crypto.DefaultHasher.Hash(value) != n.valueHash {
        return types.ErrHashMismatch
    }
    n.value = value
    return nil
}
```

### Internal Node Implementation

Create `pkg/tree/internal_node.go`:
```go
package tree

import (
    "sort"
    "sync"
    
    "github.com/acme/jmt/pkg/crypto"
    "github.com/acme/jmt/pkg/types"
)

// InternalNode represents an internal node with up to 16 children
type InternalNode struct {
    children map[types.Nibble]Child // Sparse array of children
    version  types.Version
    
    // Cache
    mu         sync.RWMutex
    cachedHash *types.Hash
}

// NewInternalNode creates a new internal node
func NewInternalNode(version types.Version) *InternalNode {
    return &InternalNode{
        children: make(map[types.Nibble]Child),
        version:  version,
    }
}

// Type returns NodeTypeInternal
func (n *InternalNode) Type() NodeType {
    return NodeTypeInternal
}

// Hash computes the internal node hash
// Format: NodeTypeInternal || child0 || child1 || ... || child15
// where each child is either (nibble || hash) or empty_hash
func (n *InternalNode) Hash() types.Hash {
    n.mu.RLock()
    if n.cachedHash != nil {
        defer n.mu.RUnlock()
        return *n.cachedHash
    }
    n.mu.RUnlock()
    
    // Build parts for hashing
    parts := [][]byte{{byte(NodeTypeInternal)}}
    
    // Process all 16 possible children in order
    for nibble := types.Nibble(0); nibble <= types.MaxNibbleValue; nibble++ {
        if child, exists := n.children[nibble]; exists {
            // Include nibble and child hash
            parts = append(parts, []byte{byte(nibble)})
            parts = append(parts, child.Hash[:])
        } else {
            // Use empty hash for missing children
            parts = append(parts, crypto.EmptyTreeHash[:])
        }
    }
    
    hash := crypto.DefaultHasher.HashConcat(parts...)
    
    // Cache it
    n.mu.Lock()
    n.cachedHash = &hash
    n.mu.Unlock()
    
    return hash
}

// IsCached returns true if hash is already computed
func (n *InternalNode) IsCached() bool {
    n.mu.RLock()
    defer n.mu.RUnlock()
    return n.cachedHash != nil
}

// Version returns the version when this node was created
func (n *InternalNode) Version() types.Version {
    return n.version
}

// Child returns the child at the given nibble
func (n *InternalNode) Child(nibble types.Nibble) (Child, bool) {
    n.mu.RLock()
    defer n.mu.RUnlock()
    child, exists := n.children[nibble]
    return child, exists
}

// SetChild sets or updates a child
func (n *InternalNode) SetChild(nibble types.Nibble, child Child) error {
    if err := types.ValidateNibble(nibble); err != nil {
        return err
    }
    
    n.mu.Lock()
    defer n.mu.Unlock()
    
    n.children[nibble] = child
    n.cachedHash = nil // Invalidate cache
    return nil
}

// RemoveChild removes a child
func (n *InternalNode) RemoveChild(nibble types.Nibble) {
    n.mu.Lock()
    defer n.mu.Unlock()
    
    delete(n.children, nibble)
    n.cachedHash = nil // Invalidate cache
}

// NumChildren returns the number of non-empty children
func (n *InternalNode) NumChildren() int {
    n.mu.RLock()
    defer n.mu.RUnlock()
    return len(n.children)
}

// Children returns a copy of all children
func (n *InternalNode) Children() map[types.Nibble]Child {
    n.mu.RLock()
    defer n.mu.RUnlock()
    
    result := make(map[types.Nibble]Child)
    for k, v := range n.children {
        result[k] = v
    }
    return result
}

// GetOnlyChild returns the single child if there's exactly one
func (n *InternalNode) GetOnlyChild() (types.Nibble, Child, bool) {
    n.mu.RLock()
    defer n.mu.RUnlock()
    
    if len(n.children) != 1 {
        return 0, Child{}, false
    }
    
    for nibble, child := range n.children {
        return nibble, child, true
    }
    return 0, Child{}, false
}

// Clone creates a copy of the internal node with a new version
func (n *InternalNode) Clone(newVersion types.Version) *InternalNode {
    n.mu.RLock()
    defer n.mu.RUnlock()
    
    clone := &InternalNode{
        children: make(map[types.Nibble]Child),
        version:  newVersion,
    }
    
    for k, v := range n.children {
        clone.children[k] = v
    }
    
    return clone
}
```

### Node Creation Helpers

Add to `pkg/tree/node.go`:
```go
// IsLeaf checks if a node is a leaf
func IsLeaf(n Node) bool {
    return n.Type() == NodeTypeLeaf
}

// IsInternal checks if a node is internal
func IsInternal(n Node) bool {
    return n.Type() == NodeTypeInternal
}

// AsLeaf attempts to cast node to LeafNode
func AsLeaf(n Node) (*LeafNode, bool) {
    leaf, ok := n.(*LeafNode)
    return leaf, ok
}

// AsInternal attempts to cast node to InternalNode
func AsInternal(n Node) (*InternalNode, bool) {
    internal, ok := n.(*InternalNode)
    return internal, ok
}
```

## Testing Requirements

### Leaf Node Tests
```go
func TestLeafNodeHash(t *testing.T) {
    key := types.KeyHash([]byte("test-key"))
    value := []byte("test-value")
    
    leaf, err := NewLeafNode(key, value, 1)
    assert.NoError(t, err)
    
    // Hash should be deterministic
    hash1 := leaf.Hash()
    hash2 := leaf.Hash()
    assert.Equal(t, hash1, hash2)
    
    // Hash should be cached
    assert.True(t, leaf.IsCached())
    
    // Hash format: 0x01 || key || valueHash
    valueHash := crypto.DefaultHasher.Hash(value)
    expected := crypto.DefaultHasher.HashConcat(
        []byte{0x01},
        key[:],
        valueHash[:],
    )
    assert.Equal(t, expected, hash1)
}

func TestLeafNodeValueValidation(t *testing.T) {
    key := types.KeyHash([]byte("test"))
    
    // Test value too large
    largeValue := make([]byte, types.MaxValueSize+1)
    _, err := NewLeafNode(key, largeValue, 1)
    assert.ErrorIs(t, err, types.ErrValueTooLarge)
    
    // Test empty key
    _, err = NewLeafNode(types.Key{}, []byte("value"), 1)
    assert.ErrorIs(t, err, types.ErrEmptyKey)
}
```

### Internal Node Tests
```go
func TestInternalNodeHash(t *testing.T) {
    node := NewInternalNode(1)
    
    // Empty internal node should use empty hashes for all children
    hash := node.Hash()
    
    parts := [][]byte{{0x00}} // NodeTypeInternal
    for i := 0; i < 16; i++ {
        parts = append(parts, crypto.EmptyTreeHash[:])
    }
    expected := crypto.DefaultHasher.HashConcat(parts...)
    assert.Equal(t, expected, hash)
    
    // Add a child
    childHash := crypto.DefaultHasher.Hash([]byte("child"))
    err := node.SetChild(5, Child{
        Hash:    childHash,
        Version: 1,
        IsLeaf:  true,
    })
    assert.NoError(t, err)
    
    // Hash should change
    newHash := node.Hash()
    assert.NotEqual(t, hash, newHash)
}

func TestInternalNodeChildren(t *testing.T) {
    node := NewInternalNode(1)
    
    // Add multiple children
    for i := types.Nibble(0); i < 5; i++ {
        child := Child{
            Hash:    crypto.DefaultHasher.Hash([]byte{byte(i)}),
            Version: 1,
            IsLeaf:  true,
        }
        err := node.SetChild(i, child)
        assert.NoError(t, err)
    }
    
    assert.Equal(t, 5, node.NumChildren())
    
    // Test child retrieval
    child, exists := node.Child(3)
    assert.True(t, exists)
    assert.True(t, child.IsLeaf)
    
    // Test non-existent child
    _, exists = node.Child(10)
    assert.False(t, exists)
    
    // Test clone
    clone := node.Clone(2)
    assert.Equal(t, types.Version(2), clone.Version())
    assert.Equal(t, node.NumChildren(), clone.NumChildren())
}
```

## Implementation Steps

1. Create node interface in `pkg/tree/node.go`
2. Implement LeafNode in `pkg/tree/leaf_node.go`
3. Implement InternalNode in `pkg/tree/internal_node.go`
4. Add helper functions for type assertions
5. Write comprehensive unit tests
6. Add benchmarks for hash computation
7. Verify thread safety with race detector

## Performance Considerations

- Cache computed hashes to avoid recomputation
- Use sync.RWMutex for concurrent read access
- Pre-allocate children map with expected size
- Consider memory pooling for frequently created nodes

## Security Notes

- Validate all inputs (keys, nibbles, value sizes)
- Ensure hash computation is deterministic
- Thread-safe implementation for concurrent access
- No extension nodes to prevent complexity attacks

## Done When ✓

- [ ] Node interface with required methods
- [ ] LeafNode implementation with value management
- [ ] InternalNode implementation with sparse children
- [ ] Thread-safe hash caching
- [ ] Helper functions for type assertions
- [ ] Comprehensive unit tests for both node types
- [ ] Hash computation follows specification exactly
- [ ] No extension nodes (only Leaf and Internal)
- [ ] Benchmarks show acceptable performance
- [ ] Race detector passes