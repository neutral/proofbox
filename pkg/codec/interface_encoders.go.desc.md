# Interface-Based Encoders

This file implements encoding functions that work with node interfaces rather than concrete types, enabling the codec package to encode nodes without importing the tree package.

## Design Rationale

These encoders use only the methods available on the node interfaces defined in the types package. This approach breaks the import cycle while maintaining the same encoding performance and output format as direct struct access.

## Key Functions

- **EncodeLeafNodeInterface**: Encodes leaf nodes using the LeafNodeInterface methods
- **EncodeInternalNodeInterface**: Encodes internal nodes using the InternalNodeInterface methods

The encoding format remains identical to the original implementation, ensuring backward compatibility and maintaining the optimized binary layout that achieves ~6.5ns encoding performance.