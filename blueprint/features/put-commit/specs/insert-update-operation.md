# Insert & Update Operation Specification

## Goal
Insert or update a key-value pair `(k, value)` at the next version `v`. If `k` already exists, its value is updated; if not, a new leaf is created. The result is a new root hash for version `v` and a batch of new nodes to persist.

## Algorithm

### 1. Traverse to Insertion Point
Start from previous version's root (`version = v-1`) or empty tree if genesis. Walk down following `k`'s nibbles:
- If internal node lacks child for needed nibble → insertion point (empty slot)
- If leaf node reached → insertion or update point

### 2. Modify at Insertion Point

#### Empty Slot (internal node child missing)
Create new leaf node for `k`:
- Create **Leaf Node** with key `k` and given `value`
- NodeKey: `(v, current_path + nibble)`
- Insert leaf's hash and version into parent internal node at index
- Create new Internal Node for version `v` with added child

#### Existing Leaf Encountered
Two sub-cases:

**A. Same Key (Update)**
- Operation is an **update** of existing key
- Create new Leaf Node for `k` at version `v` with new value
- Mark old leaf as stale
- Update parent internal node to point to new leaf

**B. Different Key (Collision)**
- Keys share prefix but diverge at some nibble
- Must introduce new internal nodes for diverging path:
  1. Create new Internal Node `X` at divergence point
  2. Add both leaves as children of `X` at their divergent nibbles
  3. May need chain of internal nodes if keys diverge by multiple nibbles
  4. All new nodes created at version `v`

### 3. Update Ancestors
After insertion/update, all ancestors back to root must be updated:
- Create new Internal Node for version `v` (copy of previous with updated child)
- Recompute hash based on children's hashes
- NodeKey: `(v, path_of_node)`
- Continue upward to new root at version `v`

## Output
- New root hash (state commitment for version `v`)
- Batch of new nodes to insert
- Set of stale nodes from previous version

## Complexity
- Touches O(log₁₆ N) nodes on path (worst-case ~64)
- Every touched node duplicated for new version
- Time and space: O(log N)
- Writes are sequential in key space (optimal for LSM)

## Batch Operations
Multiple key-value pairs can be inserted/updated in single version by:
1. Collecting all changes
2. Applying them to tree structure
3. Creating all new nodes with same version number
4. Writing batch atomically to storage

See [SN-012] and [SN-013] for implementation details.