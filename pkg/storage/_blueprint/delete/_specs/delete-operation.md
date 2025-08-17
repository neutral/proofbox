# Delete Operation Specification

## Overview
Deletion of a key is not explicitly described in the original JMT whitepaper, but can be handled in a similar persistent manner.

## Goal
Remove a key `k` from the tree at version `v`, creating a new version without that key.

## Approach
To delete a key:
1. Create a new version
2. Remove the leaf node for that key
3. Update all ancestors with new hashes

## Single-Child Optimization
If removal causes an internal node to have only one child remaining:
- **Option 1**: Remove internal node and connect child upwards (inverse of insertion collision)
- **Option 2**: Leave as single-child node (simpler, acceptable for sparse trees)

The Libra implementation marks nodes as stale but doesn't immediately merge nodes. The tree remains sparse.

## Algorithm (Basic)
1. Traverse tree to find leaf for key `k`
2. If not found, operation is no-op
3. If found:
   - Mark leaf as stale
   - Create new version of parent without this child
   - Propagate changes up to root
   - Some internal nodes may become single-child nodes

## Storage Considerations
- Deleted nodes marked as stale for potential pruning
- Historic versions still accessible (leaf exists in old versions)
- Pruning strategy needed for long-running deployments

## Open Questions
- Exact handling of single-child internal nodes
- When to trigger node merging vs leaving sparse
- Interaction with pruning old versions

Note: This specification is incomplete pending further design decisions on deletion handling.