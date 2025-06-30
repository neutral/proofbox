# Tree Interfaces

This file defines interfaces that allow other packages to interact with the tree without depending on concrete implementations.

## TreeReaderInterface

Provides read-only access to tree nodes, primarily used by the proof generation system. By using an interface, we avoid circular dependencies between the tree and proof packages.

The interface exposes:
- `GetNode`: Direct node access for proof path traversal
- `RootHash`: The root hash at the current version
- `Version`: The version being read

This abstraction allows the proof generator to work with any tree reader implementation, facilitating testing and future optimizations.