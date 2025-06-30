package tree

import (
	"fmt"
	"runtime"
	"sync"

	"github.com/neutral/proofbox/pkg/types"
)

// UpdateBatch represents a batch of updates to be committed atomically
type UpdateBatch struct {
	NewRootHash types.Hash
	NewNodes    map[string]NodeWrite // Map of storage key to node write
	StaleNodes  []types.NodeKey
}

// NodeWrite contains a node and its key for storage
type NodeWrite struct {
	Key  types.NodeKey
	Node types.Node
}

// TreeUpdater handles updates for a specific version
type TreeUpdater struct {
	tree        *Tree
	oldVersion  types.Version
	newVersion  types.Version
	nodeWrites  map[string]NodeWrite // Map of storage key to node write
	staleNodes  []types.NodeKey
	pathCloner  *PathCloner
	currentRoot types.NodeKey
}

// nodeKeyToString converts a NodeKey to a string for map usage
func nodeKeyToString(key types.NodeKey) string {
	return string(key.StorageKey())
}

// NewTreeUpdater creates a new updater for version transitions
func NewTreeUpdater(tree *Tree, oldVersion, newVersion types.Version) *TreeUpdater {
	return &TreeUpdater{
		tree:        tree,
		oldVersion:  oldVersion,
		newVersion:  newVersion,
		nodeWrites:  make(map[string]NodeWrite),
		staleNodes:  make([]types.NodeKey, 0),
		pathCloner:  NewPathCloner(tree, oldVersion, newVersion),
		currentRoot: types.RootNodeKey(oldVersion),
	}
}

// Put inserts or updates a key-value pair
func (tu *TreeUpdater) Put(key types.Key, value []byte) (types.Hash, error) {
	// Validate inputs
	if err := types.ValidateKey(key); err != nil {
		return types.Hash{}, err
	}
	if len(value) == 0 {
		return types.Hash{}, types.ErrEmptyValue
	}

	// Create new leaf node
	leafNode, err := NewLeafNode(key, value, tu.newVersion)
	if err != nil {
		return types.Hash{}, err
	}

	// Get the nibble path
	nibblePath := key.ToNibblePath()

	// Start from current root (may have been updated by previous operations)
	rootKey := tu.currentRoot

	// Perform the update
	newRootKey, err := tu.insertAt(rootKey, nibblePath, 0, leafNode)
	if err != nil {
		return types.Hash{}, err
	}

	// Update current root
	tu.currentRoot = newRootKey

	// Return the hash of the value
	return leafNode.ValueHash(), nil
}

// Delete removes a key from the tree
func (tu *TreeUpdater) Delete(key types.Key) error {
	// Validate key
	if err := types.ValidateKey(key); err != nil {
		return err
	}

	// Get the nibble path
	nibblePath := key.ToNibblePath()

	// Start from current root (may have been updated by previous operations)
	rootKey := tu.currentRoot

	// Perform the deletion
	newRootKey, deleted, err := tu.deleteAt(rootKey, nibblePath, 0)
	if err != nil {
		return err
	}

	if !deleted {
		return types.ErrKeyNotFound
	}

	// Update current root
	tu.currentRoot = newRootKey

	return nil
}

// insertAt inserts a leaf node at the specified position
func (tu *TreeUpdater) insertAt(nodeKey types.NodeKey, targetPath types.NibblePath, depth int, newLeaf *LeafNode) (types.NodeKey, error) {
	// Check max depth
	if depth > types.MaxTreeDepth {
		return types.NodeKey{}, types.ErrMaxDepthExceeded
	}

	// Load the node
	node, err := tu.getNode(nodeKey)
	if err != nil {
		// Node doesn't exist, create path to leaf
		if depth == types.MaxTreeDepth {
			// At max depth, just place the leaf
			newKey := types.NodeKey{
				Version:    tu.newVersion,
				NibblePath: targetPath.Prefix(depth),
			}
			tu.nodeWrites[nodeKeyToString(newKey)] = NodeWrite{Key: newKey, Node: newLeaf}
			return newKey, nil
		}

		// Create internal node with single child
		internal := NewInternalNode(tu.newVersion)
		nibble := targetPath.Nibbles[depth]

		// Recursively create the rest of the path
		childKey, err := tu.insertAt(types.NodeKey{}, targetPath, depth+1, newLeaf)
		if err != nil {
			return types.NodeKey{}, err
		}

		// Get child node to compute hash
		childWrite := tu.nodeWrites[nodeKeyToString(childKey)]
		childInfo := types.Child{
			Hash:    childWrite.Node.Hash(),
			Version: childKey.Version,
			IsLeaf:  childWrite.Node.Type() == types.NodeTypeLeaf,
		}

		if err := internal.SetChild(nibble, childInfo); err != nil {
			return types.NodeKey{}, err
		}

		// Save the new internal node
		newKey := types.NodeKey{
			Version:    tu.newVersion,
			NibblePath: targetPath.Prefix(depth),
		}
		tu.nodeWrites[nodeKeyToString(newKey)] = NodeWrite{Key: newKey, Node: internal}
		return newKey, nil
	}

	switch n := node.(type) {
	case *LeafNode:
		// Check if it's the same key
		if n.Key() == newLeaf.Key() {
			// Update in place - just save the new leaf
			newKey := types.NodeKey{
				Version:    tu.newVersion,
				NibblePath: nodeKey.NibblePath,
			}
			tu.nodeWrites[nodeKeyToString(newKey)] = NodeWrite{Key: newKey, Node: newLeaf}
			tu.staleNodes = append(tu.staleNodes, nodeKey)
			return newKey, nil
		}

		// Different keys - need to create internal nodes
		return tu.splitLeafNodes(nodeKey, n, newLeaf, depth)

	case *InternalNode:
		// Clone the internal node
		cloned, newKey := tu.cloneInternalNode(nodeKey, n)

		// Get the target nibble
		targetNibble := targetPath.Nibbles[depth]

		// Recursively update the child
		child, exists := n.Child(targetNibble)
		var childKey types.NodeKey
		if exists {
			childKey = nodeKey.Child(targetNibble, child.Version)
		} else {
			childKey = types.NodeKey{} // Empty key for non-existent child
		}

		newChildKey, err := tu.insertAt(childKey, targetPath, depth+1, newLeaf)
		if err != nil {
			return types.NodeKey{}, err
		}

		// Update the child reference
		childWrite := tu.nodeWrites[nodeKeyToString(newChildKey)]
		childInfo := types.Child{
			Hash:    childWrite.Node.Hash(),
			Version: newChildKey.Version,
			IsLeaf:  childWrite.Node.Type() == types.NodeTypeLeaf,
		}

		if err := cloned.SetChild(targetNibble, childInfo); err != nil {
			return types.NodeKey{}, err
		}

		// Save the updated node
		tu.nodeWrites[nodeKeyToString(newKey)] = NodeWrite{Key: newKey, Node: cloned}
		return newKey, nil

	default:
		return types.NodeKey{}, types.ErrInvalidNodeType
	}
}

// deleteAt removes a key at the specified position
func (tu *TreeUpdater) deleteAt(nodeKey types.NodeKey, targetPath types.NibblePath, depth int) (types.NodeKey, bool, error) {
	// Check max depth
	if depth > types.MaxTreeDepth {
		return types.NodeKey{}, false, types.ErrMaxDepthExceeded
	}

	// Load the node
	node, err := tu.getNode(nodeKey)
	if err != nil {
		// Node doesn't exist, nothing to delete
		return types.NodeKey{}, false, nil
	}

	switch n := node.(type) {
	case *LeafNode:
		// Check if this is the target key
		leafPath := n.Key().ToNibblePath()
		if leafPath.Equals(targetPath) {
			// Found the key to delete
			tu.staleNodes = append(tu.staleNodes, nodeKey)
			return types.NodeKey{}, true, nil
		}
		// Different key, not found
		return nodeKey, false, nil

	case *InternalNode:
		// Get the target nibble
		targetNibble := targetPath.Nibbles[depth]

		// Check if child exists
		child, exists := n.Child(targetNibble)
		if !exists {
			// No child at this nibble, key not found
			return nodeKey, false, nil
		}

		// Recursively delete from child
		childKey := nodeKey.Child(targetNibble, child.Version)
		newChildKey, deleted, err := tu.deleteAt(childKey, targetPath, depth+1)
		if err != nil {
			return types.NodeKey{}, false, err
		}

		if !deleted {
			// Key not found in subtree
			return nodeKey, false, nil
		}

		// Clone the internal node
		cloned, newKey := tu.cloneInternalNode(nodeKey, n)

		// Remove the child if it's now empty
		if newChildKey.Version == 0 {
			if err := cloned.RemoveChild(targetNibble); err != nil {
				return types.NodeKey{}, false, err
			}
		} else {
			// Update child reference
			childWrite := tu.nodeWrites[nodeKeyToString(newChildKey)]
			childInfo := types.Child{
				Hash:    childWrite.Node.Hash(),
				Version: newChildKey.Version,
				IsLeaf:  childWrite.Node.Type() == types.NodeTypeLeaf,
			}
			if err := cloned.SetChild(targetNibble, childInfo); err != nil {
				return types.NodeKey{}, false, err
			}
		}

		// Check if node should be collapsed or removed
		if cloned.NumChildren() == 0 {
			// Internal node has no children - tree is empty
			tu.staleNodes = append(tu.staleNodes, nodeKey)
			return types.NodeKey{}, true, nil
		} else if cloned.NumChildren() == 1 {
			// Get the only child
			nibble, onlyChild, ok := cloned.GetOnlyChild()
			if ok && onlyChild.IsLeaf {
				// Load the leaf child
				leafKey := newKey.Child(nibble, onlyChild.Version)
				leafNode, err := tu.getNode(leafKey)
				if err != nil {
					return types.NodeKey{}, false, err
				}

				if leaf, ok := leafNode.(*LeafNode); ok {
					// Replace this internal node with the leaf
					tu.nodeWrites[nodeKeyToString(newKey)] = NodeWrite{Key: newKey, Node: leaf}
					tu.staleNodes = append(tu.staleNodes, nodeKey)
					return newKey, true, nil
				}
			}
		}

		// Save the updated internal node
		tu.nodeWrites[nodeKeyToString(newKey)] = NodeWrite{Key: newKey, Node: cloned}
		tu.staleNodes = append(tu.staleNodes, nodeKey)

		return newKey, true, nil

	default:
		return types.NodeKey{}, false, types.ErrInvalidNodeType
	}
}

// splitLeafNodes creates internal nodes to accommodate two leaf nodes
func (tu *TreeUpdater) splitLeafNodes(oldLeafKey types.NodeKey, oldLeaf, newLeaf *LeafNode, depth int) (types.NodeKey, error) {
	oldPath := oldLeaf.Key().ToNibblePath()
	newPath := newLeaf.Key().ToNibblePath()

	// Find common prefix length
	commonDepth := depth
	for commonDepth < types.MaxTreeDepth &&
		commonDepth < len(oldPath.Nibbles) &&
		commonDepth < len(newPath.Nibbles) &&
		oldPath.Nibbles[commonDepth] == newPath.Nibbles[commonDepth] {
		commonDepth++
	}

	// Create internal nodes up to divergence point
	if commonDepth == types.MaxTreeDepth {
		// Keys are identical - shouldn't happen in split
		return types.NodeKey{}, fmt.Errorf("identical keys in split")
	}

	// Create internal node at divergence point
	internal := NewInternalNode(tu.newVersion)

	// Add old leaf
	oldNibble := oldPath.Nibbles[commonDepth]
	oldLeafKey = types.NodeKey{
		Version:    tu.newVersion,
		NibblePath: oldPath.Prefix(commonDepth + 1),
	}
	tu.nodeWrites[nodeKeyToString(oldLeafKey)] = NodeWrite{Key: oldLeafKey, Node: oldLeaf}

	oldChild := types.Child{
		Hash:    oldLeaf.Hash(),
		Version: oldLeafKey.Version,
		IsLeaf:  true,
	}
	if err := internal.SetChild(oldNibble, oldChild); err != nil {
		return types.NodeKey{}, err
	}

	// Add new leaf
	newNibble := newPath.Nibbles[commonDepth]
	newLeafKey := types.NodeKey{
		Version:    tu.newVersion,
		NibblePath: newPath.Prefix(commonDepth + 1),
	}
	tu.nodeWrites[nodeKeyToString(newLeafKey)] = NodeWrite{Key: newLeafKey, Node: newLeaf}

	newChild := types.Child{
		Hash:    newLeaf.Hash(),
		Version: newLeafKey.Version,
		IsLeaf:  true,
	}
	if err := internal.SetChild(newNibble, newChild); err != nil {
		return types.NodeKey{}, err
	}

	// Save internal node
	internalKey := types.NodeKey{
		Version:    tu.newVersion,
		NibblePath: oldPath.Prefix(commonDepth),
	}
	tu.nodeWrites[nodeKeyToString(internalKey)] = NodeWrite{Key: internalKey, Node: internal}

	return internalKey, nil
}

// cloneInternalNode creates a new version of an internal node
func (tu *TreeUpdater) cloneInternalNode(oldKey types.NodeKey, node *InternalNode) (*InternalNode, types.NodeKey) {
	// Clone the node with new version
	cloned := node.Clone(tu.newVersion).(*InternalNode)

	// Create new key
	newKey := types.NodeKey{
		Version:    tu.newVersion,
		NibblePath: oldKey.NibblePath,
	}

	// Mark old node as stale
	tu.staleNodes = append(tu.staleNodes, oldKey)

	// Register in path cloner
	tu.pathCloner.RegisterClone(oldKey, newKey)

	return cloned, newKey
}

// getNode retrieves a node from writes or storage
func (tu *TreeUpdater) getNode(key types.NodeKey) (types.Node, error) {
	// Check pending writes first
	if nodeWrite, exists := tu.nodeWrites[nodeKeyToString(key)]; exists {
		return nodeWrite.Node, nil
	}

	// Check if it's an empty key (non-existent node)
	if key.Version == 0 {
		return nil, fmt.Errorf("node not found")
	}

	// Load from tree
	return tu.tree.loadNodeFromStorage(key)
}

// BuildUpdateBatch creates a batch of all changes
func (tu *TreeUpdater) BuildUpdateBatch() (*UpdateBatch, error) {
	// Special case: if tree is empty after deletions
	if tu.currentRoot.Version == 0 && tu.currentRoot.NibblePath.Length == 0 {
		// Empty tree - no nodes to write
		return &UpdateBatch{
			NewRootHash: types.EmptyHash(),
			NewNodes:    make(map[string]NodeWrite), // Empty map, not tu.nodeWrites
			StaleNodes:  tu.staleNodes,
		}, nil
	}

	// Get the root node
	var rootNode types.Node

	// Check if current root is in pending writes
	rootKeyStr := nodeKeyToString(tu.currentRoot)
	if rootWrite, exists := tu.nodeWrites[rootKeyStr]; exists {
		rootNode = rootWrite.Node
	} else if tu.currentRoot.Version != 0 {
		// No changes were made, load existing root
		var err error
		rootNode, err = tu.tree.loadNodeFromStorage(tu.currentRoot)
		if err != nil {
			return nil, fmt.Errorf("failed to load root: %w", err)
		}
	}

	var rootHash types.Hash
	if rootNode != nil {
		rootHash = rootNode.Hash()
	} else {
		rootHash = types.EmptyHash()
	}

	// Validate batch before returning
	batch := &UpdateBatch{
		NewRootHash: rootHash,
		NewNodes:    tu.nodeWrites,
		StaleNodes:  tu.staleNodes,
	}

	if err := tu.ValidateBatch(batch); err != nil {
		return nil, fmt.Errorf("batch validation failed: %w", err)
	}

	return batch, nil
}

// BuildUpdateBatchParallel creates a batch with parallel hash computation for large batches
func (tu *TreeUpdater) BuildUpdateBatchParallel() (*UpdateBatch, error) {
	// For small batches, use sequential version
	if len(tu.nodeWrites) < 100 {
		return tu.BuildUpdateBatch()
	}

	// Special case: if tree is empty after deletions
	if tu.currentRoot.Version == 0 && tu.currentRoot.NibblePath.Length == 0 {
		return &UpdateBatch{
			NewRootHash: types.EmptyHash(),
			NewNodes:    make(map[string]NodeWrite),
			StaleNodes:  tu.staleNodes,
		}, nil
	}

	// Parallelize hash computation for nodes
	numWorkers := runtime.NumCPU()
	if numWorkers > len(tu.nodeWrites)/10 {
		numWorkers = len(tu.nodeWrites)/10 + 1
	}

	type hashJob struct {
		key   string
		write NodeWrite
	}

	jobs := make(chan hashJob, len(tu.nodeWrites))
	results := make(chan struct{}, len(tu.nodeWrites))
	errors := make(chan error, numWorkers)

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				// Force hash computation
				_ = job.write.Node.Hash()
				results <- struct{}{}
			}
		}()
	}

	// Send jobs
	for key, write := range tu.nodeWrites {
		jobs <- hashJob{key: key, write: write}
	}
	close(jobs)

	// Wait for completion
	wg.Wait()
	close(results)
	close(errors)

	// Check for errors
	for err := range errors {
		if err != nil {
			return nil, err
		}
	}

	// Get root hash
	var rootNode types.Node
	rootKeyStr := nodeKeyToString(tu.currentRoot)
	if rootWrite, exists := tu.nodeWrites[rootKeyStr]; exists {
		rootNode = rootWrite.Node
	} else if tu.currentRoot.Version != 0 {
		var err error
		rootNode, err = tu.tree.loadNodeFromStorage(tu.currentRoot)
		if err != nil {
			return nil, fmt.Errorf("failed to load root: %w", err)
		}
	}

	var rootHash types.Hash
	if rootNode != nil {
		rootHash = rootNode.Hash()
	} else {
		rootHash = types.EmptyHash()
	}

	batch := &UpdateBatch{
		NewRootHash: rootHash,
		NewNodes:    tu.nodeWrites,
		StaleNodes:  tu.staleNodes,
	}

	if err := tu.ValidateBatch(batch); err != nil {
		return nil, fmt.Errorf("batch validation failed: %w", err)
	}

	return batch, nil
}

// ValidateBatch ensures batch integrity
func (tu *TreeUpdater) ValidateBatch(batch *UpdateBatch) error {
	if batch == nil {
		return fmt.Errorf("batch is nil")
	}

	// Check for circular references
	visited := make(map[string]bool)
	for key, write := range batch.NewNodes {
		if err := tu.validateNode(write.Node, batch, visited); err != nil {
			return fmt.Errorf("node %s validation failed: %w", key, err)
		}
	}

	// Verify all nodes are for the correct version
	for _, write := range batch.NewNodes {
		if write.Node.Version() != tu.newVersion {
			return fmt.Errorf("node has incorrect version: expected %d, got %d",
				tu.newVersion, write.Node.Version())
		}
	}

	// Verify stale nodes are not from future versions
	// Note: stale nodes can be from the current version if they were created
	// and then replaced during the same batch (e.g., during delete operations)
	for _, staleKey := range batch.StaleNodes {
		if staleKey.Version > tu.newVersion {
			return fmt.Errorf("stale node has future version: %d > %d",
				staleKey.Version, tu.newVersion)
		}
	}

	return nil
}

// validateNode recursively validates a node and its children
func (tu *TreeUpdater) validateNode(node types.Node, batch *UpdateBatch, visited map[string]bool) error {
	if node == nil {
		return fmt.Errorf("node is nil")
	}

	// For cycle detection, we need a unique identifier for each node
	// Using just the node's hash should be sufficient
	nodeID := fmt.Sprintf("%x", node.Hash())
	if visited[nodeID] {
		return fmt.Errorf("circular reference detected")
	}
	visited[nodeID] = true

	// Validate based on node type
	switch n := node.(type) {
	case *InternalNode:
		// Check children
		for nibble, child := range n.Children() {
			if child.IsEmpty() {
				continue
			}

			// Verify child version is not future
			if child.Version > tu.newVersion {
				return fmt.Errorf("child at nibble %d has future version: %d > %d",
					nibble, child.Version, tu.newVersion)
			}
		}
	case *LeafNode:
		// Validate key
		if err := types.ValidateKey(n.Key()); err != nil {
			return fmt.Errorf("invalid leaf key: %w", err)
		}
	default:
		return fmt.Errorf("unknown node type: %T", node)
	}

	return nil
}
