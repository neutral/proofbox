# Put Operation

The Put method implements key-value insertion and updates for the Jellyfish Merkle Tree.

## Purpose

Put is the primary write operation for the tree, now integrated with the versioning system to support multi-version concurrency control.

## Current Implementation

As of step 14, Put has been refactored to use the versioning API:
- Creates a new version automatically via `BeginVersion()`
- Delegates to `PutVersioned()` for the actual operation
- Commits the version atomically
- Returns the new version number to the caller

## Design Evolution

### Original Design (Pre-versioning)
- Direct tree modification with immediate commits
- Single-version tree with version numbers for tracking

### Current Design (With versioning)
- Version-aware operations through TreeUpdater
- Structural sharing via PathCloner
- Atomic batch commits per version
- Support for concurrent pending versions

## Operation Flow

1. Begin new version
2. Perform versioned put operation
3. Commit version (or abort on error)
4. Return new version number

## Integration Points

- **TreeUpdater**: Handles the actual tree modifications
- **VersionManager**: Manages version lifecycle
- **PathCloner**: Ensures structural sharing
- **Storage Layer**: Atomic batch commits

The simplified API maintains backward compatibility while leveraging the full versioning infrastructure internally.