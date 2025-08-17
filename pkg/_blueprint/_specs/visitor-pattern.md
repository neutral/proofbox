---
id: spec.visitor-pattern
tags: [design-pattern, node-traversal, extensibility]
---

# Visitor Pattern Specification

## Purpose

Defines the visitor pattern implementation for extensible node operations in the Jellyfish Merkle Tree. This pattern enables adding new operations without modifying node types.

## Overview

The visitor pattern implementation provides a flexible, extensible mechanism for performing operations on tree nodes without modifying the node structures themselves. This pattern separates the algorithm from the data structure, enabling new operations to be added without changing existing code.

### Key Benefits

- **Extensibility**: New operations added without changing nodes
- **Type Safety**: Compile-time checking for all node types  
- **Separation of Concerns**: Operations separate from data structure
- **Flexibility**: Multiple visitors can implement different algorithms

### Common Use Cases

- Tree traversal and printing
- Validation and integrity checking
- Serialization to different formats
- Metrics collection and analysis
- Tree comparison and diffing
- Migration and transformation

## Design Rationale

### Why Visitor Pattern?

1. **Open/Closed Principle**: Node types remain closed for modification but open for extension
2. **Type Safety**: Compile-time verification that all node types are handled
3. **Separation of Concerns**: Tree structure is independent of operations performed on it
4. **Extensibility**: New operations can be added by implementing new visitors

### Alternative Approaches Considered

- **Method Addition**: Adding methods directly to nodes (violates open/closed principle)
- **Type Switching**: External functions with type switches (scattered logic, harder to maintain)
- **Interface Pollution**: Adding many specialized interfaces (leads to fat interfaces)

## Core Components

### 1. NodeVisitor Interface

```go
type NodeVisitor interface {
    VisitLeaf(*LeafNode) error
    VisitInternal(*InternalNode) error
}
```

**Design Decisions:**
- Returns `error` to handle operation failures gracefully
- Takes concrete types (`*LeafNode`, `*InternalNode`) for type-safe access to node-specific methods
- Minimal interface with exactly one method per node type

### 2. Accept Function

```go
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
```

**Design Decisions:**
- Function instead of method to avoid modifying Node interface
- Type switch ensures exhaustive handling
- Returns error for unknown node types (defensive programming)

### 3. Visitor Implementations

Visitors implement specific operations by providing the `VisitLeaf` and `VisitInternal` methods.

## Example Implementation: NodePrinter

```go
type NodePrinter struct {
    Writer io.Writer
    Depth  int
}

func (p *NodePrinter) VisitLeaf(leaf *LeafNode) error {
    indent := strings.Repeat("  ", p.Depth)
    _, err := fmt.Fprintf(p.Writer, "%sLeaf: key=%x, value_hash=%x\n",
        indent, leaf.Key(), leaf.ValueHash())
    return err
}

func (p *NodePrinter) VisitInternal(internal *InternalNode) error {
    indent := strings.Repeat("  ", p.Depth)
    _, err := fmt.Fprintf(p.Writer, "%sInternal: %d children\n",
        indent, internal.NumChildren())
    return err
}
```

## Usage Patterns

### Basic Usage

```go
// Create visitor
printer := &NodePrinter{Writer: os.Stdout, Depth: 0}

// Visit a single node
err := Accept(node, printer)
```

### Tree Traversal

```go
type TreePrinter struct {
    *NodePrinter
}

func (t *TreePrinter) VisitInternal(internal *InternalNode) error {
    // Print current node
    if err := t.NodePrinter.VisitInternal(internal); err != nil {
        return err
    }
    
    // Visit children with increased depth
    t.Depth++
    defer func() { t.Depth-- }()
    
    for nibble := 0; nibble < 16; nibble++ {
        if child, exists := internal.Child(types.Nibble(nibble)); exists {
            // Load and visit child (requires storage integration)
            // childNode := loadNode(child.Hash)
            // Accept(childNode, t)
        }
    }
    return nil
}
```

## Potential Visitor Implementations

### 1. Validator Visitor
Validates tree structure integrity, hash correctness, and invariants.

### 2. Serializer Visitor
Converts nodes to various formats (JSON, binary, protobuf).

### 3. Metrics Visitor
Collects statistics about tree structure (depth, node count, memory usage).

### 4. Differ Visitor
Compares two trees and identifies differences.

### 5. Pruner Visitor
Marks nodes for deletion based on criteria.

### 6. Migrator Visitor
Transforms nodes during version upgrades.

## Implementation Guidelines

### Creating New Visitors

1. **Single Responsibility**: Each visitor should have one clear purpose
2. **Error Handling**: Always return meaningful errors
3. **State Management**: Use visitor struct fields for state between visits
4. **Resource Cleanup**: Use defer for cleanup operations

### Performance Considerations

1. **Allocation**: Reuse visitors when possible to reduce allocations
2. **Depth Tracking**: Maintain depth externally rather than in recursive calls
3. **Early Termination**: Return errors to stop traversal early when needed

## Testing Requirements

### Unit Tests
- Verify Accept function dispatches correctly
- Test error handling for unknown node types
- Ensure visitor methods are called with correct parameters

### Integration Tests
- Test complex visitors that maintain state
- Verify tree traversal patterns
- Test error propagation through visitor chain

## Security Considerations

1. **Resource Limits**: Visitors should implement depth/count limits for untrusted trees
2. **Error Information**: Don't leak sensitive information in error messages
3. **Panic Safety**: Visitors should not panic on malformed data

## Future Extensions

### Async Visitors
```go
type AsyncNodeVisitor interface {
    VisitLeafAsync(*LeafNode) <-chan error
    VisitInternalAsync(*InternalNode) <-chan error
}
```

### Generic Visitor
Using Go generics for type-safe visitor results:
```go
type NodeVisitor[T any] interface {
    VisitLeaf(*LeafNode) (T, error)
    VisitInternal(*InternalNode) (T, error)
}
```

### Visitor Composition
Combining multiple visitors:
```go
type CompositeVisitor struct {
    visitors []NodeVisitor
}
```

## Related Patterns

- **Strategy Pattern**: Visitors are strategies for node processing
- **Iterator Pattern**: Often combined with visitor for tree traversal
- **Command Pattern**: Visitors can encapsulate operations as commands