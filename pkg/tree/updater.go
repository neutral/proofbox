package tree

import (
	"crypto/sha256"
	"fmt"

	"github.com/cockroachdb/pebble"
	"github.com/neutral/proofbox/pkg/codec"
	"github.com/neutral/proofbox/pkg/types"
)

// TreeUpdater handles the logic for updating the tree
type TreeUpdater struct {
	tree       *Tree
	batch      *pebble.Batch
	oldVersion types.Version
	newVersion types.Version
	nodeWrites map[string]nodeWrite // Nodes to write in this update
	staleNodes []types.NodeKey      // Nodes replaced in this update
}

// nodeWrite contains a node and its key for writing
type nodeWrite struct {
	key  types.NodeKey
	node types.Node
}

// Put inserts or updates a key-value pair
func (u *TreeUpdater) Put(key types.Key, value []byte) (types.Hash, error) {
	// Compute value hash using SHA-256 for content-addressed storage
	valueHash := sha256.Sum256(value)

	// Store value using content-addressed key (no version for deduplication)
	valueKey := makeValueKey(types.Hash(valueHash))
	if err := u.batch.Set(valueKey, value, nil); err != nil {
		return types.Hash{}, fmt.Errorf("failed to store value: %w", err)
	}

	// Get current root
	var currentRoot types.Node
	if u.oldVersion > 0 {
		u.tree.mu.RLock()
		rootHash, exists := u.tree.rootHashes[u.oldVersion]
		u.tree.mu.RUnlock()

		if exists && rootHash != types.EmptyHash() {
			// Load existing root
			root, err := u.loadNode(types.RootNodeKey(u.oldVersion))
			if err != nil {
				return types.Hash{}, fmt.Errorf("failed to load root: %w", err)
			}
			currentRoot = root
		}
	}

	// Handle empty tree case
	if currentRoot == nil {
		return u.insertIntoEmptyTree(key, types.Hash(valueHash))
	}

	// Handle non-empty tree (will be implemented in later steps)
	return u.insertIntoTree(currentRoot, key, types.Hash(valueHash))
}

// insertIntoEmptyTree handles insertion into an empty tree
func (u *TreeUpdater) insertIntoEmptyTree(key types.Key, valueHash types.Hash) (types.Hash, error) {
	// Create new leaf node directly with the LeafNode type
	leaf := &LeafNode{
		key:       key,
		valueHash: valueHash,
		value:     nil,
		version:   u.newVersion,
	}

	// Store the leaf
	leafKey := types.RootNodeKey(u.newVersion)
	u.nodeWrites[leafKey.String()] = nodeWrite{key: leafKey, node: leaf}

	// Return leaf's hash as new root hash
	return leaf.Hash(), nil
}

// insertIntoTree handles insertion into existing tree with full logic
func (u *TreeUpdater) insertIntoTree(root types.Node, key types.Key, valueHash types.Hash) (types.Hash, error) {
	switch root.Type() {
	case types.NodeTypeLeaf:
		leaf := root.(*LeafNode)
		if leaf.Key() == key {
			// Same key - update value
			return u.updateLeaf(leaf, valueHash)
		}
		// Different keys - split required
		return u.splitLeafNode(leaf, key, valueHash)

	case types.NodeTypeInternal:
		// Navigate down the tree
		return u.insertIntoInternalNode(root.(*InternalNode), key, valueHash, 0)

	default:
		return types.Hash{}, fmt.Errorf("unexpected node type: %v", root.Type())
	}
}

// updateLeaf creates a new version of a leaf with updated value
func (u *TreeUpdater) updateLeaf(oldLeaf types.LeafNodeInterface, newValueHash types.Hash) (types.Hash, error) {
	// Create new leaf with same key but new value
	newLeaf := &LeafNode{
		key:       oldLeaf.Key(),
		valueHash: newValueHash,
		value:     nil,
		version:   u.newVersion,
	}

	// Store the new leaf
	leafKey := types.RootNodeKey(u.newVersion)
	u.nodeWrites[leafKey.String()] = nodeWrite{key: leafKey, node: newLeaf}

	return newLeaf.Hash(), nil
}

// loadNode loads a node from the previous version
func (u *TreeUpdater) loadNode(key types.NodeKey) (types.Node, error) {
	// First check if we're loading from the current update
	if nw, exists := u.nodeWrites[key.String()]; exists {
		return nw.node, nil
	}

	// Load from storage
	storageKey := key.StorageKey()
	data, closer, err := u.tree.db.Get(storageKey)
	if err == pebble.ErrNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	// Decode node
	return codec.DecodeNode(data, key.Version)
}

// writeToBatch writes all nodes to the batch
func (u *TreeUpdater) writeToBatch() error {
	nodeCodec := &codec.NodeCodec{}

	for _, nw := range u.nodeWrites {
		// Encode node
		data, err := nodeCodec.EncodeNode(nw.node)
		if err != nil {
			return fmt.Errorf("failed to encode node: %w", err)
		}

		// Write to batch
		storageKey := nw.key.StorageKey()
		if err := u.batch.Set(storageKey, data, nil); err != nil {
			return fmt.Errorf("failed to write node: %w", err)
		}

		// Update cache
		u.tree.nodeCache.Put(nw.key, nw.node)
	}

	return nil
}

// splitLeafNode handles the case where we need to split a leaf at the root
func (u *TreeUpdater) splitLeafNode(existingLeaf *LeafNode, newKey types.Key,
	newValueHash types.Hash) (types.Hash, error) {
	return u.splitLeafNodeAtDepth(existingLeaf, newKey, newValueHash, 0)
}

// splitLeafNodeAtDepth handles splitting a leaf at a specific depth in the tree
func (u *TreeUpdater) splitLeafNodeAtDepth(existingLeaf *LeafNode, newKey types.Key,
	newValueHash types.Hash, currentDepth int) (types.Hash, error) {

	// Get nibble paths for both keys
	existingKey := existingLeaf.Key()
	existingPath := types.NewNibblePath(existingKey[:])
	newPath := types.NewNibblePath(newKey[:])

	// Find where they diverge
	commonLen := existingPath.CommonPrefixLength(newPath)

	// Create new leaf for the new key
	newLeaf := &LeafNode{
		key:       newKey,
		valueHash: newValueHash,
		version:   u.newVersion,
	}

	// Create new version of existing leaf
	existingLeafNew := &LeafNode{
		key:       existingLeaf.Key(),
		valueHash: existingLeaf.valueHash,
		version:   u.newVersion,
	}

	// Mark existing leaf as stale
	u.staleNodes = append(u.staleNodes, types.NodeKey{
		Version:    existingLeaf.Version(),
		NibblePath: existingPath,
	})

	// Build the tree structure appropriately based on where we are
	if currentDepth == 0 {
		// We're at the root - build complete path
		return u.buildTreeFromSplit(existingLeafNew, newLeaf, existingPath, newPath, commonLen)
	} else {
		// We're inside the tree - only build from current depth
		return u.buildFromDepth(existingLeafNew, newLeaf, existingPath, newPath, commonLen, currentDepth)
	}
}

// buildTreeFromSplit creates the tree structure after a split
func (u *TreeUpdater) buildTreeFromSplit(existingLeaf *LeafNode, newLeaf *LeafNode,
	existingPath, newPath types.NibblePath, commonLen int) (types.Hash, error) {

	// Get the nibbles at divergence point
	existingNibble, err := existingPath.GetNibble(commonLen)
	if err != nil {
		return types.Hash{}, fmt.Errorf("failed to get existing nibble: %w", err)
	}

	newNibble, err := newPath.GetNibble(commonLen)
	if err != nil {
		return types.Hash{}, fmt.Errorf("failed to get new nibble: %w", err)
	}

	// Create internal node at divergence point
	divergeNode := NewInternalNode(u.newVersion)

	// Add both leaves as children
	divergeNode.SetChild(existingNibble, types.Child{
		Hash:    existingLeaf.Hash(),
		Version: u.newVersion,
		IsLeaf:  true,
	})

	divergeNode.SetChild(newNibble, types.Child{
		Hash:    newLeaf.Hash(),
		Version: u.newVersion,
		IsLeaf:  true,
	})

	// Store the leaves at their position in the tree (not their full key path)
	// Both leaves are at depth commonLen + 1 (after the divergence point)
	leafDepth := commonLen + 1
	existingLeafPath := existingPath.Prefix(leafDepth)
	newLeafPath := newPath.Prefix(leafDepth)

	existingKey := types.NodeKey{
		Version:    u.newVersion,
		NibblePath: existingLeafPath,
	}
	u.nodeWrites[existingKey.String()] = nodeWrite{key: existingKey, node: existingLeaf}

	newKey := types.NodeKey{
		Version:    u.newVersion,
		NibblePath: newLeafPath,
	}
	u.nodeWrites[newKey.String()] = nodeWrite{key: newKey, node: newLeaf}

	// Store the divergence node at its position
	divergePath := existingPath.Prefix(commonLen)
	divergeKey := types.NodeKey{
		Version:    u.newVersion,
		NibblePath: divergePath,
	}
	u.nodeWrites[divergeKey.String()] = nodeWrite{key: divergeKey, node: divergeNode}

	// Only build intermediate nodes if we're not at the root level
	if commonLen > 0 {
		// We need to build the path from divergence to root, but we should
		// avoid creating single-child chains. For now, we'll create them
		// but in a future optimization we could collapse them.
		currentNode := divergeNode
		currentDepth := commonLen

		// Build path from divergence to root
		for depth := currentDepth - 1; depth >= 0; depth-- {
			parentNode := NewInternalNode(u.newVersion)

			// Get the nibble for this level
			nibble, err := existingPath.GetNibble(depth)
			if err != nil {
				return types.Hash{}, fmt.Errorf("failed to get nibble at depth %d: %w", depth, err)
			}

			// Add current node as child
			parentNode.SetChild(nibble, types.Child{
				Hash:    currentNode.Hash(),
				Version: u.newVersion,
				IsLeaf:  false,
			})

			// Store current node
			nodeKey := types.NodeKey{
				Version:    u.newVersion,
				NibblePath: existingPath.Prefix(depth),
			}
			u.nodeWrites[nodeKey.String()] = nodeWrite{key: nodeKey, node: parentNode}

			currentNode = parentNode
		}

		return currentNode.Hash(), nil
	} else {
		// Divergence is at root level
		return divergeNode.Hash(), nil
	}
}

// buildFromDepth builds the split structure starting from a specific depth
func (u *TreeUpdater) buildFromDepth(existingLeaf *LeafNode, newLeaf *LeafNode,
	existingPath, newPath types.NibblePath, commonLen int, startDepth int) (types.Hash, error) {
	
	// If the split happens exactly at startDepth, we just return the divergence node
	if commonLen == startDepth {
		return u.buildDivergenceNodeOnly(existingLeaf, newLeaf, existingPath, newPath, commonLen)
	}
	
	// Otherwise, build from divergence back to startDepth
	return u.buildPathFromDivergence(existingLeaf, newLeaf, existingPath, newPath, commonLen, startDepth)
}

// buildDivergenceNodeOnly creates just the divergence node and its children
func (u *TreeUpdater) buildDivergenceNodeOnly(existingLeaf *LeafNode, newLeaf *LeafNode,
	existingPath, newPath types.NibblePath, divergeDepth int) (types.Hash, error) {
	
	// Get the nibbles at divergence point
	existingNibble, err := existingPath.GetNibble(divergeDepth)
	if err != nil {
		return types.Hash{}, fmt.Errorf("failed to get existing nibble: %w", err)
	}

	newNibble, err := newPath.GetNibble(divergeDepth)
	if err != nil {
		return types.Hash{}, fmt.Errorf("failed to get new nibble: %w", err)
	}

	// Create internal node at divergence point
	divergeNode := NewInternalNode(u.newVersion)

	// Add both leaves as children
	divergeNode.SetChild(existingNibble, types.Child{
		Hash:    existingLeaf.Hash(),
		Version: u.newVersion,
		IsLeaf:  true,
	})

	divergeNode.SetChild(newNibble, types.Child{
		Hash:    newLeaf.Hash(),
		Version: u.newVersion,
		IsLeaf:  true,
	})

	// Store the leaves at their position in the tree
	leafDepth := divergeDepth + 1
	existingLeafPath := existingPath.Prefix(leafDepth)
	newLeafPath := newPath.Prefix(leafDepth)

	existingKey := types.NodeKey{
		Version:    u.newVersion,
		NibblePath: existingLeafPath,
	}
	u.nodeWrites[existingKey.String()] = nodeWrite{key: existingKey, node: existingLeaf}

	newKey := types.NodeKey{
		Version:    u.newVersion,
		NibblePath: newLeafPath,
	}
	u.nodeWrites[newKey.String()] = nodeWrite{key: newKey, node: newLeaf}

	// Store the divergence node
	divergeKey := types.NodeKey{
		Version:    u.newVersion,
		NibblePath: existingPath.Prefix(divergeDepth),
	}
	u.nodeWrites[divergeKey.String()] = nodeWrite{key: divergeKey, node: divergeNode}

	return divergeNode.Hash(), nil
}

// buildPathFromDivergence builds internal nodes from divergence back to start depth
func (u *TreeUpdater) buildPathFromDivergence(existingLeaf *LeafNode, newLeaf *LeafNode,
	existingPath, newPath types.NibblePath, commonLen int, startDepth int) (types.Hash, error) {
	
	// First build the divergence node
	divergeHash, err := u.buildDivergenceNodeOnly(existingLeaf, newLeaf, existingPath, newPath, commonLen)
	if err != nil {
		return types.Hash{}, err
	}

	// Now build internal nodes from divergence point back to startDepth
	currentHash := divergeHash
	
	// Build path from divergence back to start depth
	for depth := commonLen - 1; depth >= startDepth; depth-- {
		parentNode := NewInternalNode(u.newVersion)

		// Get the nibble for this level
		nibble, err := existingPath.GetNibble(depth)
		if err != nil {
			return types.Hash{}, fmt.Errorf("failed to get nibble at depth %d: %w", depth, err)
		}

		// Add current node as child
		parentNode.SetChild(nibble, types.Child{
			Hash:    currentHash,
			Version: u.newVersion,
			IsLeaf:  false,
		})

		// Store the node
		nodeKey := types.NodeKey{
			Version:    u.newVersion,
			NibblePath: existingPath.Prefix(depth),
		}
		u.nodeWrites[nodeKey.String()] = nodeWrite{key: nodeKey, node: parentNode}

		currentHash = parentNode.Hash()
	}

	return currentHash, nil
}

// insertIntoInternalNode navigates internal nodes for insertion
func (u *TreeUpdater) insertIntoInternalNode(node *InternalNode, key types.Key,
	valueHash types.Hash, depth int) (types.Hash, error) {

	if depth >= types.MaxTreeDepth {
		return types.Hash{}, fmt.Errorf("maximum tree depth exceeded")
	}

	// Get nibble at current depth
	nibble, err := key.ExtractNibble(depth)
	if err != nil {
		return types.Hash{}, fmt.Errorf("failed to extract nibble at depth %d: %w", depth, err)
	}

	// Current node's path
	nodePath := types.NewNibblePath(key[:]).Prefix(depth)

	// Check if child exists at this nibble
	child, exists := node.Child(nibble)
	if !exists {
		// Empty slot - create new leaf here
		return u.insertLeafIntoEmptySlot(node, key, valueHash, nibble, depth, nodePath)
	}

	// Load child node - all nodes are stored by their position in the tree
	childPath := nodePath.Append(nibble)
	childKey := types.NodeKey{
		Version:    child.Version,
		NibblePath: childPath,
	}

	childNode, err := u.loadNode(childKey)
	if err != nil {
		return types.Hash{}, fmt.Errorf("failed to load child node: %w", err)
	}
	if childNode == nil {
		return types.Hash{}, fmt.Errorf("child node at %v not found", childKey)
	}

	// Recursively insert into child
	var newChildHash types.Hash
	var actualIsLeaf bool
	if child.IsLeaf {
		// Child is a leaf - might need to split
		childLeaf := childNode.(*LeafNode)
		actualIsLeaf = true
		if childLeaf.Key() == key {
			newChildHash, err = u.updateLeaf(childLeaf, valueHash)
		} else {
			// Pass the current depth so split knows where it is in the tree
			newChildHash, err = u.splitLeafNodeAtDepth(childLeaf, key, valueHash, depth+1)
			// After split, the child is no longer a leaf
			actualIsLeaf = false
		}
	} else {
		// Child is internal - continue navigation
		actualIsLeaf = false
		newChildHash, err = u.insertIntoInternalNode(childNode.(*InternalNode),
			key, valueHash, depth+1)
	}

	if err != nil {
		return types.Hash{}, err
	}

	// Create new version of current node with updated child
	return u.updateInternalNodeChild(node, nibble, newChildHash, actualIsLeaf, nodePath)
}

// insertLeafIntoEmptySlot adds a new leaf to an empty child slot
func (u *TreeUpdater) insertLeafIntoEmptySlot(parent *InternalNode, key types.Key,
	valueHash types.Hash, nibble types.Nibble, parentDepth int, parentPath types.NibblePath) (types.Hash, error) {

	// Create new leaf
	leaf := &LeafNode{
		key:       key,
		valueHash: valueHash,
		version:   u.newVersion,
	}

	// Store leaf at its position in the tree (not its full key path)
	leafPath := parentPath.Append(nibble)
	leafKey := types.NodeKey{
		Version:    u.newVersion,
		NibblePath: leafPath,
	}
	u.nodeWrites[leafKey.String()] = nodeWrite{key: leafKey, node: leaf}

	// Create new version of parent with added child
	newParent := parent.Clone(u.newVersion).(*InternalNode)
	newParent.SetChild(nibble, types.Child{
		Hash:    leaf.Hash(),
		Version: u.newVersion,
		IsLeaf:  true,
	})

	// Store updated parent
	parentKey := types.NodeKey{
		Version:    u.newVersion,
		NibblePath: parentPath,
	}
	u.nodeWrites[parentKey.String()] = nodeWrite{key: parentKey, node: newParent}

	// Mark old parent as stale
	u.staleNodes = append(u.staleNodes, types.NodeKey{
		Version:    parent.Version(),
		NibblePath: parentPath,
	})

	return newParent.Hash(), nil
}

// updateInternalNodeChild creates new version of internal node with updated child
func (u *TreeUpdater) updateInternalNodeChild(node *InternalNode, childNibble types.Nibble,
	childHash types.Hash, isLeaf bool, nodePath types.NibblePath) (types.Hash, error) {

	// Clone node with new version
	newNode := node.Clone(u.newVersion).(*InternalNode)

	// Update the specific child
	newNode.SetChild(childNibble, types.Child{
		Hash:    childHash,
		Version: u.newVersion,
		IsLeaf:  isLeaf,
	})

	// Store new node
	nodeKey := types.NodeKey{
		Version:    u.newVersion,
		NibblePath: nodePath,
	}
	u.nodeWrites[nodeKey.String()] = nodeWrite{key: nodeKey, node: newNode}

	// Mark old node as stale
	u.staleNodes = append(u.staleNodes, types.NodeKey{
		Version:    node.Version(),
		NibblePath: nodePath,
	})

	return newNode.Hash(), nil
}
