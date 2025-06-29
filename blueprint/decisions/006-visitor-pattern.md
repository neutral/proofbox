---
id: adr.006.visitor-pattern
status: accepted
date: 2024-01-15
---

# ADR-006: Visitor Pattern for Node Operations

## Status

Accepted

## Context

We need a way to perform various operations on nodes (printing, validation, serialization, metrics collection) without:
1. Adding methods to node types for each operation
2. Using type switches scattered throughout the codebase
3. Violating the open/closed principle

## Decision

Implement the Visitor pattern with:
- `NodeVisitor` interface with methods for each node type
- `Accept(node Node, visitor NodeVisitor)` function for dispatch
- Concrete visitors for specific operations

## Consequences

### Positive
- **Open/Closed Principle**: Add new operations without modifying nodes
- **Type Safety**: Compiler ensures all node types handled
- **Centralized Logic**: Each operation in its own visitor
- **Extensibility**: Easy to add new visitors
- **Testability**: Visitors can be tested independently

### Negative
- **Boilerplate**: Must implement all visitor methods
- **Indirection**: Extra function call overhead
- **Learning Curve**: Pattern may be unfamiliar

### Neutral
- Well-established GoF pattern
- Common in compiler/tree implementations

## Implementation Details

```go
// Visitor interface
type NodeVisitor interface {
    VisitLeaf(*LeafNode) error
    VisitInternal(*InternalNode) error
}

// Accept function (not a method to keep Node interface minimal)
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

// Example visitor
type NodePrinter struct {
    Writer io.Writer
    Depth  int
}

func (p *NodePrinter) VisitLeaf(leaf *LeafNode) error {
    // Print leaf details
}

func (p *NodePrinter) VisitInternal(internal *InternalNode) error {
    // Print internal details
}
```

## Alternatives Considered

1. **Methods on Node Interface**
   ```go
   type Node interface {
       Print(io.Writer)
       Validate() error
       CollectMetrics(*Metrics)
   }
   ```
   - Rejected: Violates open/closed, bloats interface

2. **Type Switches Everywhere**
   ```go
   switch n := node.(type) {
   case *LeafNode:
       // handle leaf
   case *InternalNode:
       // handle internal
   }
   ```
   - Rejected: Scattered logic, hard to maintain

3. **Reflection-Based Operations**
   - Rejected: Loss of type safety, poor performance

4. **Function Tables**
   ```go
   var operations = map[reflect.Type]func(Node)error{...}
   ```
   - Rejected: No compile-time checking

## Future Extensions

- Async visitors for parallel operations
- Generic visitors with type parameters (Go 1.18+)
- Visitor combinators for complex operations

## References

- Design Patterns: Elements of Reusable Object-Oriented Software
- Similar usage in Go AST package (ast.Visitor)
- Implemented in pkg/tree/node.go