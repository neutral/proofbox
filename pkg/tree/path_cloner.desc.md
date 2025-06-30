# Path Cloner

The `path_cloner.go` file implements efficient structural sharing for the versioned Jellyfish Merkle Tree.

## Purpose

This component enables copy-on-write semantics by cloning only the nodes along modified paths while preserving references to unchanged subtrees, achieving efficient multi-version storage.

## Key Design Decisions

1. **Path-Based Cloning**: Only nodes along the path from modified leaves to root are cloned, maximizing structural sharing.

2. **Clone Registry**: Maintains a mapping of cloned nodes to prevent duplicate cloning and ensure consistency.

3. **Bottom-Up Processing**: Clones nodes from leaf to root to properly update parent-child relationships.

4. **Version-Aware References**: Updates child references in cloned internal nodes to point to the appropriate version.

## Why This Design

The path cloner is critical for versioning efficiency:
- **Space Efficiency**: Unchanged subtrees are shared across versions, reducing storage requirements dramatically
- **Time Efficiency**: Only O(log n) nodes need cloning for each update
- **Consistency**: The clone registry ensures each node is cloned exactly once per version

Without this component, each version would require a full tree copy, making versioning impractical for large datasets.