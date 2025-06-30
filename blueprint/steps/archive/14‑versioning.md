---
id: step.14.versioning
depends_on:
  - step.12.update‑existing
tags: [versioning, step]
---

## Objective

Implement `Begin(version)` and path‑cloning for new versions.

## Implements

- **§4.3 NodeKey**, **§6.2 Persistent Structure** ("path cloning"), and **§5 S2 sequential writes**.
  _What happens_:

  - `Begin(version)` forks the root; ancestor duplication realises the _persistent functional tree_ described under _Persistent Versioning_.

## Technical Details

### Multi-Version Architecture

The JMT implements persistent versioning through structural sharing. Each version maintains its own root, but unchanged subtrees are shared between versions. This enables efficient snapshots and historical queries.

**Key Concepts:**

- **Structural Sharing**: Unchanged nodes are shared between versions, minimizing storage overhead
- **Copy-on-Write**: Only modified paths are cloned when creating new versions
- **Version Isolation**: Each version provides a consistent view of the tree at a specific point in time
- **Parent-Child Relationships**: Each version (except the initial) has a parent version it was derived from

### Version Management

**Version States:**

- **Pending**: Version is being constructed, accepting updates
- **Committed**: Version is finalized and immutable
- **Aborted**: Version was cancelled and should not be accessible

**Version Lifecycle:**

1. **Begin**: Create a new version based on a parent version
2. **Update**: Apply changes (Put/Delete operations) to the pending version
3. **Commit/Abort**: Finalize or cancel the version

**Concurrency Model:**

- Writes are serialized (only one pending version at a time)
- Reads can happen concurrently across different versions
- Version metadata is protected by appropriate locking

### Path Cloning Strategy

When updating a key in a new version, only the nodes along the path from root to leaf need to be cloned:

1. **Identify Path**: Trace the path from root to the target key
2. **Clone Nodes**: Starting from the leaf, clone each node with the new version
3. **Update References**: Parent nodes reference the newly cloned children
4. **Preserve Sharing**: Siblings and unmodified subtrees remain shared

This approach ensures:

- Minimal node duplication
- Efficient memory usage
- Fast version creation
- Preserved tree integrity

### Storage Organization

**Version-Prefixed Keys:**

- Node keys include version number as prefix
- Enables efficient version-specific queries
- Optimizes LSM-tree performance through sequential writes

**Atomic Commits:**

- All nodes for a version are written in a single batch
- Ensures consistency even on crash/failure
- Root hash written last to finalize version

**Node Sharing:**

- Nodes can be referenced by multiple versions
- Child references include version information
- Enables efficient structural sharing

## Implementation Steps

- **Version Manager Component**

  - Track version metadata, states, and relationships
  - Maintain latest and committed version pointers
  - Handle version allocation and overflow protection

- **Version Lifecycle Operations**

  - Begin: Create new pending version from parent
  - Commit: Atomically persist all changes
  - Abort: Cancel pending version and cleanup

- **Path Cloning Mechanism**

  - Trace path from root to modified keys
  - Clone only necessary nodes with new version
  - Update parent-child references appropriately
  - Cache cloned nodes to avoid duplication

- **Version-Aware Tree Operations**

  - Extend Get/Put/Delete to accept version parameter
  - Route operations to appropriate version
  - Maintain backward compatibility for non-versioned API

- **Storage Layer Integration**

  - Implement version-prefixed key encoding
  - Batch writes for atomic commits
  - Efficient iteration within version boundaries

- **Version Pruning System**
  - Define retention policies (count, duration, checkpoints)
  - Identify shared vs version-specific nodes
  - Safe deletion without breaking references
  - Background pruning to minimize impact

## Testing Requirements

### Basic Versioning Tests

- **Multi-Version Coexistence**

  - Create multiple versions with different values for same keys
  - Verify each version maintains its own consistent view
  - Ensure values don't leak between versions

- **Old Root Verification**

  - Generate proofs for historical versions
  - Verify proofs against version-specific root hashes
  - Ensure proofs from one version fail against other versions

- **Structural Sharing Validation**
  - Create large tree with many nodes
  - Make small modification in new version
  - Verify only modified path creates new nodes
  - Measure sharing ratio (should be >90% for small changes)

### Concurrent Version Tests

- **Parallel Version Creation**

  - Multiple goroutines creating versions simultaneously
  - Each makes unique changes to avoid conflicts
  - Verify all versions commit successfully
  - Check version numbers are allocated correctly

- **Read-Write Concurrency**

  - Concurrent reads from committed versions
  - Simultaneous write to pending version
  - No interference between operations

- **Version Abort Behavior**
  - Abort pending version with changes
  - Verify changes don't persist
  - Ensure version number can be reused
  - Check cleanup of temporary data

### Version Pruning Tests

- **Retention Policy Enforcement**

  - Create many versions exceeding retention limit
  - Apply count-based pruning policy
  - Verify old versions are removed
  - Ensure recent versions remain accessible

- **Shared Node Protection**

  - Create versions with structural sharing
  - Prune intermediate versions
  - Verify shared nodes aren't deleted
  - Ensure tree integrity maintained

- **Time-Based Pruning**
  - Create versions over time period
  - Apply duration-based retention
  - Verify age-based deletion works correctly

### Path Cloning Tests

- **Minimal Cloning Verification**

  - Track nodes cloned during update
  - Verify only path nodes are cloned
  - Check sibling subtrees remain shared

- **Clone Integrity**

  - Cloned nodes have correct version
  - Parent-child relationships preserved
  - Hash values remain consistent

- **Deep Path Cloning**
  - Test cloning at maximum tree depth
  - Verify performance doesn't degrade
  - Check memory usage is reasonable

## Performance Considerations

- **Structural sharing**: Minimize node duplication
- **Sequential writes**: Version-prefixed keys optimize LSM
- **Batch operations**: Amortize version creation overhead
- **Lazy cloning**: Only clone modified paths
- **Version caching**: Keep recent versions in memory

## Security Considerations

- **Version isolation**: Changes don't affect committed versions
- **Atomic commits**: All-or-nothing version updates
- **Concurrent safety**: Serialized writes, parallel reads
- **Pruning safety**: Don't delete shared nodes

## REPL Testing Instructions

To test the versioning system interactively using gore REPL:

```bash
# Start gore REPL from project root
gore -autoimport

# Import required packages
:import github.com/neutral/proofbox/pkg/tree
:import github.com/neutral/proofbox/pkg/types
:import github.com/cockroachdb/pebble
:import fmt

# Create a test database
db, _ := pebble.Open("/tmp/jmt-test", &pebble.Options{})
defer db.Close()

# Create a new tree
jmt, _ := tree.NewTree(db, tree.DefaultTreeConfig())

# Test basic versioning
v1, _ := jmt.Put(types.KeyHash([]byte("key1")), []byte("value1"))
fmt.Printf("Created version: %d\n", v1)

v2, _ := jmt.Put(types.KeyHash([]byte("key2")), []byte("value2"))
fmt.Printf("Created version: %d\n", v2)

# Read from different versions
val1, _ := jmt.GetAtVersion(v1, types.KeyHash([]byte("key1")))
fmt.Printf("Value at v1: %s\n", val1)

# key1 should still exist in v2 with same value (structural sharing)
val2, _ := jmt.GetAtVersion(v2, types.KeyHash([]byte("key1")))
fmt.Printf("Value of key1 at v2: %s\n", val2)

# key2 only exists in v2
val3, _ := jmt.GetAtVersion(v2, types.KeyHash([]byte("key2")))
fmt.Printf("Value of key2 at v2: %s\n", val3)

# key2 should not exist in v1
val4, _ := jmt.GetAtVersion(v1, types.KeyHash([]byte("key2")))
fmt.Printf("Value of key2 at v1: %v (should be nil)\n", val4)

# Test explicit version management
v3, _ := jmt.BeginVersion()
jmt.PutVersioned(v3, types.KeyHash([]byte("key3")), []byte("value3"))
jmt.PutVersioned(v3, types.KeyHash([]byte("key4")), []byte("value4"))
jmt.CommitVersion(v3)
fmt.Printf("Committed version: %d\n", v3)

# Test version abort
v4, _ := jmt.BeginVersion()
jmt.PutVersioned(v4, types.KeyHash([]byte("key5")), []byte("value5"))
jmt.AbortVersion(v4)
_, err := jmt.GetAtVersion(v4, types.KeyHash([]byte("key5")))
fmt.Printf("Error after abort: %v\n", err)

# Test structural sharing
root1, _ := jmt.GetRootHash(v1)
root2, _ := jmt.GetRootHash(v2)
fmt.Printf("Root v1: %x\n", root1)
fmt.Printf("Root v2: %x\n", root2)

# Test deletion
v5, _ := jmt.Delete(types.KeyHash([]byte("key1")))
val5, _ := jmt.GetAtVersion(v5, types.KeyHash([]byte("key1")))
fmt.Printf("Value after delete: %v\n", val5)

# Test garbage collection
jmt.SetVersionRetentionPolicy(tree.RetentionPolicyCount, 3, 0)
removed, _ := jmt.CollectVersionGarbage()
fmt.Printf("Removed versions: %v\n", removed)

# Verify old versions are gone
_, err = jmt.GetRootHash(removed[0])
fmt.Printf("Error accessing removed version: %v\n", err)

# Test empty tree
isEmpty, _ := jmt.IsEmpty(0)
fmt.Printf("Version 0 is empty: %v\n", isEmpty)

# Test latest version
latest := jmt.GetLatestVersion()
fmt.Printf("Latest version: %d\n", latest)
```

## Done When ✓

- [x] Old root still verifiable after a new commit
- [x] Begin(version) creates isolated update context
- [x] Path cloning creates minimal new nodes
- [x] Structural sharing reduces storage overhead
- [x] Multiple versions can be read concurrently
- [x] Version pruning safely removes old data
- [x] Aborted versions don't persist
- [x] 100% test coverage for versioning scenarios
- [x] Add gore repl testing instructions to this step file
