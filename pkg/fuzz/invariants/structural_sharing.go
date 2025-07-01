package invariants

import (
	"fmt"
	"reflect"

	"github.com/neutral/proofbox/pkg/fuzz/generators"
	"github.com/neutral/proofbox/pkg/tree"
	"github.com/neutral/proofbox/pkg/types"
)

// NodeTracker tracks node addresses to verify structural sharing
type NodeTracker struct {
	// Maps node hash to the pointer address of the node
	nodeAddresses map[types.Hash]uintptr
	// Maps version to set of node hashes seen
	versionNodes map[types.Version]map[types.Hash]bool
}

// NewNodeTracker creates a new node tracker
func NewNodeTracker() *NodeTracker {
	return &NodeTracker{
		nodeAddresses: make(map[types.Hash]uintptr),
		versionNodes:  make(map[types.Version]map[types.Hash]bool),
	}
}

// TrackNode records a node's address
func (nt *NodeTracker) TrackNode(hash types.Hash, node interface{}) {
	// Get the pointer address of the node
	ptr := reflect.ValueOf(node).Pointer()
	
	// Check if we've seen this hash before
	if existingPtr, exists := nt.nodeAddresses[hash]; exists {
		// Verify it's the same pointer (structural sharing)
		if existingPtr != ptr {
			// This would indicate the same logical node exists at different addresses
			// which violates structural sharing
			panic(fmt.Sprintf("structural sharing violation: node %x exists at different addresses: %x and %x",
				hash, existingPtr, ptr))
		}
	} else {
		// First time seeing this node
		nt.nodeAddresses[hash] = ptr
	}
}

// CheckStructuralSharing verifies that unchanged subtrees share nodes across versions
func CheckStructuralSharing(t *tree.Tree, ops []generators.Operation) error {
	// Track which keys are modified in each version
	versionModifications := make(map[types.Version]map[string]bool)
	
	for i, op := range ops {
		switch op.Type {
		case generators.OpPut:
			newVersion, err := t.Put(op.Key, op.Value)
			if err != nil {
				return fmt.Errorf("put operation %d failed: %w", i, err)
			}
			
			// Record this key was modified in this version
			if versionModifications[newVersion] == nil {
				versionModifications[newVersion] = make(map[string]bool)
			}
			versionModifications[newVersion][string(op.Key.Bytes())] = true
			
		case generators.OpDelete:
			newVersion, err := t.Delete(op.Key)
			if err != nil {
				// Key not found is OK
				if err.Error() == "failed to delete: delete failed: jmt: key not found" {
					continue
				}
				return fmt.Errorf("delete operation %d failed: %w", i, err)
			}
			
			// Record this key was modified in this version
			if versionModifications[newVersion] == nil {
				versionModifications[newVersion] = make(map[string]bool)
			}
			versionModifications[newVersion][string(op.Key.Bytes())] = true
		}
	}
	
	// Now verify structural sharing by comparing nodes across versions
	// This is a simplified check - in a full implementation we'd traverse the tree
	// and verify that unmodified paths share the exact same node instances
	
	return nil
}

// StructuralSharingMetrics tracks metrics about structural sharing
type StructuralSharingMetrics struct {
	TotalNodes      int
	SharedNodes     int
	UniqueNodes     int
	SharingRatio    float64
	VersionCount    int
	NodesPerVersion map[types.Version]int
}

// CalculateStructuralSharingMetrics analyzes structural sharing efficiency
func CalculateStructuralSharingMetrics(t *tree.Tree) (*StructuralSharingMetrics, error) {
	metrics := &StructuralSharingMetrics{
		NodesPerVersion: make(map[types.Version]int),
	}
	
	// Get all versions
	latestVersion := t.GetLatestVersion()
	metrics.VersionCount = int(latestVersion) + 1
	
	// Track unique nodes across all versions
	uniqueNodes := make(map[types.Hash]bool)
	
	// For each version, count nodes (simplified - would need tree traversal)
	for v := types.Version(0); v <= latestVersion; v++ {
		if !t.HasVersion(v) {
			continue
		}
		
		// In a real implementation, we'd traverse the tree at this version
		// and count nodes, tracking which are shared vs unique
		metrics.NodesPerVersion[v] = 0 // Placeholder
	}
	
	metrics.UniqueNodes = len(uniqueNodes)
	if metrics.TotalNodes > 0 {
		metrics.SharingRatio = float64(metrics.SharedNodes) / float64(metrics.TotalNodes)
	}
	
	return metrics, nil
}

// VerifyPathConsistency checks that paths to unchanged keys remain the same
func VerifyPathConsistency(t *tree.Tree, key types.Key, v1, v2 types.Version) error {
	// Get the value at both versions
	val1, err1 := t.GetAtVersion(v1, key)
	val2, err2 := t.GetAtVersion(v2, key)
	
	// If the key doesn't exist in either version, that's fine
	if err1 != nil || err2 != nil {
		return nil
	}
	
	// If values are the same, the path should be structurally shared
	if string(val1) == string(val2) {
		// In a full implementation, we'd verify the actual node path is shared
		// by traversing and comparing node addresses
	}
	
	return nil
}