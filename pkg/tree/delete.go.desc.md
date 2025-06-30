# Delete Operations

The `delete.go` file implements key deletion functionality for the Jellyfish Merkle Tree.

## Purpose

This component provides a high-level API for removing keys from the tree while maintaining version consistency and structural integrity.

## Key Design Decisions

1. **Version-Based Deletion**: All deletions create a new version, preserving historical states.

2. **Automatic Version Management**: Handles version lifecycle (begin, delete, commit/abort) transparently.

3. **Error Recovery**: Automatically aborts the version on any error to maintain consistency.

4. **Delegated Implementation**: Actual deletion logic is handled by TreeUpdater for proper node management.

## Why This Design

The separate delete file provides a clean API separation:
- **Simplicity**: Users can call Delete() without managing versions manually
- **Safety**: Automatic cleanup on errors prevents partial states
- **Consistency**: Integrates seamlessly with the versioning system
- **Maintainability**: Keeps the main tree.go file focused on core functionality