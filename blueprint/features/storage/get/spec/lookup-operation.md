# Lookup Operation Specification

## Goal
Retrieve the value associated with a key `k` in a given version `v`, or determine that no such key exists, and provide a proof (if needed).

## Algorithm

### 1. Start at Root
Determine the `NodeKey` for the root of version `v`:
- Root NodeKey: `(v, path="")`
- The root has an empty nibble path by convention

### 2. Traverse Internal Nodes
Fetch the node with the current `NodeKey` from storage. If it's an **Internal Node**:

1. Take the next nibble of target key `k` (according to current depth)
2. Interpret nibble as index 0–15
3. Check if internal node has a child at that index:
   - **No child present**: Key `k` not found (empty subtree)
     - Produce exclusion proof indicating empty node
   - **Child present**: Retrieve child's version and digest
4. Compute next `NodeKey`:
   - `(child_version, current_path + nibble)`
   - Append nibble to path, use child's recorded version
5. Fetch child node and repeat

### 3. Handle Leaf Nodes
If fetched node is a **Leaf Node**, read the leaf's stored key and compare to target key `k`:

- **Keys match exactly**: Found the target
  - Return leaf's value or pointer
  - Generate inclusion proof with sibling hashes from traversed path
  
- **Keys don't match**: Different key's leaf encountered
  - Key `k` doesn't exist in tree
  - Produce exclusion proof citing this "neighbor" leaf
  - The existence of different leaf with same prefix prevents `k` from existing

## Performance Characteristics
- Reads at most one node per tree level (per nibble of key)
- Worst-case: 64 steps for 256-bit key (if tree is maximally deep)
- Average case: Shorter due to sparse tree structure
- Each node retrieval is a point DB lookup (efficient in LSM stores)

## Proof Generation
During traversal, collect information needed for proof:
- For inclusion: Sibling hashes along the path
- For exclusion: Either empty slot indicator or conflicting leaf