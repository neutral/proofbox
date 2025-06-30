# Codec Factory Implementation

This file implements the tree package's side of the factory pattern, registering node creation functions with the codec package to enable decoding without circular imports.

## Registration Mechanism

The `init()` function automatically registers the tree package's node factory when the package is imported. This ensures the codec package can create concrete node instances during decoding operations.

## Factory Implementation

The factory function:
1. Switches on the node type to determine which decoder to use
2. Calls the appropriate decode function for leaf or internal nodes
3. Returns a types.Node interface containing the concrete implementation

This design maintains the separation between packages while allowing full functionality. The registration happens once at startup with negligible performance impact.