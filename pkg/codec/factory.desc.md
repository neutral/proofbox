# Node Factory Pattern

This file implements the factory pattern that allows the tree package to register node creation functions with the codec package, enabling decoding without circular imports.

## Why Factory Pattern is Necessary

Simply moving the Node interface to the types package isn't sufficient for decoding because:

1. **Interfaces can't be instantiated**: The codec needs to create concrete instances (LeafNode or InternalNode), not just return an interface
2. **Concrete types live in tree package**: The codec can't import tree package to access LeafNode/InternalNode without creating a circular dependency
3. **Decoding requires object construction**: When decoding bytes, we must create actual struct instances with fields populated

### The Problem Without Factory

```go
// In codec/node.go - This won't work!
func DecodeNode(data []byte) (types.Node, error) {
    nodeType := data[0]
    switch nodeType {
    case types.NodeTypeLeaf:
        // ERROR: Can't create tree.LeafNode here
        // because we can't import tree package!
        return &tree.LeafNode{...} // Would create import cycle
    }
}
```

### The Solution With Factory

The factory pattern solves this by having the tree package provide a function that creates nodes:

```go
// codec defines what it needs
type NodeFactory func(nodeType, data, version) (types.Node, error)

// tree provides the implementation
func init() {
    codec.RegisterNodeFactory(func(...) (types.Node, error) {
        // tree package CAN create its own concrete types
        return &LeafNode{...}, nil
    })
}

// codec uses the factory without knowing concrete types
func DecodeNode(data []byte) (types.Node, error) {
    return nodeFactory(nodeType, data, version)
}
```

## Architecture

The complete flow:
1. **Compile Time**: codec imports only types (no tree import)
2. **Init Time**: tree registers its factory with codec
3. **Runtime**: codec calls factory to create concrete nodes
4. **Return**: Concrete nodes returned as types.Node interface

## Alternatives Considered

1. **Put concrete types in types package**: Would violate single responsibility and make types package too large
2. **Use reflection**: Performance penalty and loss of type safety
3. **Return raw data from codec**: Would duplicate decoding logic in tree package

## Benefits

The factory pattern elegantly lets the tree package "teach" codec how to create nodes without codec knowing the concrete types at compile time. This maintains clean separation while enabling full functionality.