# Tree Validation

## Purpose
Provides validation functions to ensure the Jellyfish Merkle Tree maintains its structural invariants after modifications.

## Key Functions

### ValidateTreeStructure
Performs a comprehensive validation of the tree structure including:
- Verifying no cycles exist in the tree
- Ensuring internal nodes (except root) have at least 2 children
- Confirming leaf paths match their stored keys
- Checking all nodes are reachable and properly linked

## Design Decisions

### Snapshot-Based Validation
Uses a database snapshot for consistent reads during validation, preventing race conditions with concurrent updates.

### Path Verification
For leaf nodes, verifies that the path used to reach the leaf matches the nibble representation of its key, ensuring correct tree structure.

### Minimum Children Rule
Enforces that non-root internal nodes have at least 2 children, preventing degenerate tree structures.

## Usage
Primarily used in tests to verify tree operations maintain correctness. Can also be used for debugging and health checks in production.