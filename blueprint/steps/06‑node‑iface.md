---
id: step.06.node‑iface
depends_on:
tags: [structs, step]
---

## Objective

Define `Node` interface satisfied by both node types.

## Implements

- Architecture's "in‑memory logic layer" abstraction → polymorphic `Node` interface.
  _What happens_:

  - Allows generic traversal code to call `Digest()` without type‑switching, keeping the implementation simple (Non‑functional goal: **Simplicity and Maintainability**).

## Technical Details

### Node Interface Definition

```go
// Node represents a node in the Jellyfish Merkle Tree
type Node interface {
    // Core methods
    Type() NodeType       // Returns NodeTypeLeaf or NodeTypeInternal
    IsLeaf() bool         // True for leaf nodes, false for internal
    Hash() Hash           // Returns the digest/hash of this node

    // Version tracking
    Version() Version     // Version when this node was created
    // Note: No SetVersion - nodes are immutable, use Clone() for new versions

    // Cache status
    IsCached() bool       // Returns true if hash is already computed
}

// NodeType identifies the type of node
type NodeType byte

const (
    NodeTypeInternal NodeType = 0x00
    NodeTypeLeaf     NodeType = 0x01
)

// Note: IsLeaf() is implemented as a helper function rather than an interface method
// This keeps the Node interface minimal and allows checking node types without casting
```

### Extended Node Operations

```go
// NodeWithChildren interface for nodes that can have children (read-only)
type NodeWithChildren interface {
    Node
    Child(nibble Nibble) (Child, bool)  // Get child at nibble
    Children() map[Nibble]Child         // Get all children (defensive copy)
    NumChildren() int                   // Count of non-empty children
    GetOnlyChild() (Nibble, Child, bool) // For single-child optimization
}

// NodeWithKey interface for nodes that have a key
type NodeWithKey interface {
    Node
    Key() Key
}

// NodeCloneable interface for creating new versions of nodes
type NodeCloneable interface {
    Node
    Clone(newVersion Version) Node  // Creates new node with different version
}
```

### Type Assertions and Conversions

```go
// AsLeaf attempts to convert a Node to LeafNode
func AsLeaf(n Node) (*LeafNode, bool) {
    leaf, ok := n.(*LeafNode)
    return leaf, ok
}

// AsInternal attempts to convert a Node to InternalNode
func AsInternal(n Node) (*InternalNode, bool) {
    internal, ok := n.(*InternalNode)
    return internal, ok
}

// AsLeafErr returns an error if node is not a leaf
func AsLeafErr(n Node) (*LeafNode, error) {
    leaf, ok := n.(*LeafNode)
    if !ok {
        return nil, fmt.Errorf("expected leaf node, got %T", n)
    }
    return leaf, nil
}

// AsInternalErr returns an error if node is not internal
func AsInternalErr(n Node) (*InternalNode, error) {
    internal, ok := n.(*InternalNode)
    if !ok {
        return nil, fmt.Errorf("expected internal node, got %T", n)
    }
    return internal, nil
}
```

### Node Factory Functions

```go
// NewNode creates a node from its type and data
func NewNode(nodeType NodeType, data []byte) (Node, error) {
    switch nodeType {
    case NodeTypeLeaf:
        var leaf LeafNode
        if err := DecodeLeafNode(data, &leaf); err != nil {
            return nil, fmt.Errorf("failed to decode leaf: %w", err)
        }
        return &leaf, nil

    case NodeTypeInternal:
        var internal InternalNode
        if err := DecodeInternalNode(data, &internal); err != nil {
            return nil, fmt.Errorf("failed to decode internal: %w", err)
        }
        return &internal, nil

    default:
        return nil, fmt.Errorf("unknown node type: %d", nodeType)
    }
}

// CloneNode creates a deep copy of any node with a new version
func CloneNode(n Node, newVersion Version) (Node, error) {
    if n == nil {
        return nil, nil
    }

    switch node := n.(type) {
    case *LeafNode:
        return node.Clone(newVersion), nil
    case *InternalNode:
        return node.Clone(newVersion), nil
    default:
        return nil, fmt.Errorf("unknown node type: %T", n)
    }
}
```

### Common Node Operations

```go
// NodePath represents the path to a node in the tree
type NodePath struct {
    Nibbles []Nibble
    Depth   int
}

// TraverseToLeaf follows a key path down to a leaf
func TraverseToLeaf(root Node, key Key) (*LeafNode, NodePath, error) {
    if root == nil {
        return nil, NodePath{}, ErrEmptyTree
    }

    nibblePath := KeyToNibblePath(key)
    path := NodePath{
        Nibbles: make([]Nibble, 0, MaxTreeDepth),
    }

    current := root
    for depth := 0; depth < MaxTreeDepth; depth++ {
        if leaf, ok := current.(*LeafNode); ok {
            path.Depth = depth
            return leaf, path, nil
        }

        internal, ok := current.(*InternalNode)
        if !ok {
            return nil, path, fmt.Errorf("invalid node type at depth %d", depth)
        }

        nibble := nibblePath[depth]
        path.Nibbles = append(path.Nibbles, nibble)

        child, exists := internal.Child(nibble)
        if !exists {
            path.Depth = depth
            return nil, path, ErrKeyNotFound
        }

        // Load child node (would involve storage in real implementation)
        // current = loadNode(child.Hash)
    }

    return nil, path, ErrMaxDepthExceeded
}

// ComputeRootHash recursively computes the hash of a subtree
func ComputeRootHash(node Node) Hash {
    if node == nil {
        return EmptyHash
    }
    return node.Hash()
}

// CountNodes recursively counts nodes in a subtree
func CountNodes(node Node, loader NodeLoader) (int, error) {
    if node == nil {
        return 0, nil
    }

    count := 1 // Count this node

    if internal, ok := node.(*InternalNode); ok {
        children := internal.Children()
        for _, child := range children {
            childNode, err := loader.LoadNode(child.Hash)
            if err != nil {
                return 0, err
            }

            childCount, err := CountNodes(childNode, loader)
            if err != nil {
                return 0, err
            }

            count += childCount
        }
    }

    return count, nil
}
```

### Node Visitor Pattern

```go
// NodeVisitor defines methods for visiting nodes
type NodeVisitor interface {
    VisitLeaf(*LeafNode) error
    VisitInternal(*InternalNode) error
}

// Accept implements the visitor pattern for nodes
func Accept(node Node, visitor NodeVisitor) error {
    switch n := node.(type) {
    case *LeafNode:
        return visitor.VisitLeaf(n)
    case *InternalNode:
        return visitor.VisitInternal(n)
    default:
        return fmt.Errorf("unknown node type: %T", node)
    }
}

// Example visitor: NodePrinter
type NodePrinter struct {
    Writer io.Writer
    Depth  int
}

func (p *NodePrinter) VisitLeaf(leaf *LeafNode) error {
    indent := strings.Repeat("  ", p.Depth)
    fmt.Fprintf(p.Writer, "%sLeaf: key=%x, value_hash=%x\n",
        indent, leaf.Key(), leaf.ValueHash())
    return nil
}

func (p *NodePrinter) VisitInternal(internal *InternalNode) error {
    indent := strings.Repeat("  ", p.Depth)
    fmt.Fprintf(p.Writer, "%sInternal: %d children\n",
        indent, internal.NumChildren())
    return nil
}
```

## Implementation Steps

1. **Define Node interface**: Core methods all nodes must implement
2. **Add interface compliance**: Ensure LeafNode and InternalNode implement it
3. **Create helper functions**: Type assertions and conversions
4. **Add factory functions**: Generic node creation from data
5. **Implement common operations**: Traversal, counting, visiting
6. **Add visitor pattern**: For extensible node processing

## Testing Requirements

### Compile-Time Checks

```go
// Ensure nodes implement the interface
var (
    _ Node = (*LeafNode)(nil)
    _ Node = (*InternalNode)(nil)
)

// Ensure optional interfaces are implemented correctly
var (
    _ NodeWithChildren = (*InternalNode)(nil)
    _ NodeWithKey      = (*LeafNode)(nil)
)
```

### Unit Tests

```go
func TestNodeInterface(t *testing.T) {
    // Test leaf node
    leaf := &LeafNode{
        Key:       KeyHash([]byte("test")),
        ValueHash: Hash{0x01, 0x02, /* ... */},
    }
    // Version is set during creation, not via setter
    leaf = NewLeafNode(key, value, 100)

    if leaf.Type() != NodeTypeLeaf {
        t.Error("Wrong type for leaf")
    }
    if !IsLeaf(leaf) {
        t.Error("IsLeaf should return true")
    }
    if leaf.Version() != 100 {
        t.Error("Version not set correctly")
    }

    // Test internal node
    internal := NewInternalNode(200)

    if internal.Type() != NodeTypeInternal {
        t.Error("Wrong type for internal")
    }
    if IsLeaf(internal) {
        t.Error("IsLeaf should return false for internal nodes")
    }
    if internal.Version() != 200 {
        t.Error("Version not set correctly")
    }

    // Test polymorphic usage
    nodes := []Node{leaf, internal}
    for _, node := range nodes {
        _ = node.Hash()  // Should work for both
    }
}

func TestTypeConversions(t *testing.T) {
    leaf := &LeafNode{}
    internal := &InternalNode{}

    // Test AsLeaf
    if l, ok := AsLeaf(leaf); !ok || l != leaf {
        t.Error("AsLeaf failed for leaf node")
    }
    if _, ok := AsLeaf(internal); ok {
        t.Error("AsLeaf should fail for internal node")
    }

    // Test AsInternal
    if _, ok := AsInternal(leaf); ok {
        t.Error("AsInternal should fail for leaf node")
    }
    if i, ok := AsInternal(internal); !ok || i != internal {
        t.Error("AsInternal failed for internal node")
    }
}
```

## Performance Considerations

- **Interface dispatch**: Small overhead for virtual method calls
- **Type assertions**: Cached type info makes assertions fast
- **No boxing**: Node types are pointers, no allocation overhead
- **Inlining**: Simple methods like Type() likely inlined

## Security Notes

- **Type safety**: Interface prevents mixing node types incorrectly
- **Immutability**: No mutation methods (no SetVersion, SetChild, etc.)
- **Version tracking**: All nodes track their creation version
- **No nil receiver**: Methods should handle nil gracefully

## Done When ✓

- [ ] `var _ Node = (*LeafNode)(nil)` compiles for both nodes
- [ ] Node interface with Type, IsLeaf, Hash, Version methods
- [ ] Both LeafNode and InternalNode implement the interface
- [ ] Type conversion helpers (AsLeaf, AsInternal)
- [ ] Factory functions for generic node creation
- [ ] Common operations like traversal implemented
- [ ] Visitor pattern for extensible processing
- [ ] 100% test coverage for interface methods
- [ ] Documentation explains polymorphic usage
