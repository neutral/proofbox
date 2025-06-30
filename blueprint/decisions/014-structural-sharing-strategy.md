# ADR-014: Structural Sharing Strategy

## Status
Accepted

## Context
Multi-version trees can consume significant storage if each version stores a complete copy. We need an efficient way to share unchanged data between versions.

## Decision
Implement copy-on-write structural sharing where:
- Only nodes along modified paths are cloned
- Child references in cloned nodes point to appropriate versions
- Unchanged subtrees are shared between versions
- Each node is immutable once created

The TreeUpdater tracks all modifications for a version and only writes new/modified nodes.

## Consequences
### Positive
- O(log n) space per update instead of O(n)
- Natural integration with immutable node design
- Enables efficient version snapshots
- Crash-safe due to immutability

### Negative
- More complex update logic
- Must track which nodes to clone
- Storage may accumulate without garbage collection
- Requires careful management of node references

## Implementation Notes
- PathCloner tracks cloned nodes to avoid duplicates
- TreeUpdater uses current root, not old version root
- Node versions in child references enable cross-version sharing