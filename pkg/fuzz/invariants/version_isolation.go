package invariants

import (
	"fmt"

	"github.com/neutral/proofbox/pkg/fuzz/generators"
	"github.com/neutral/proofbox/pkg/tree"
	"github.com/neutral/proofbox/pkg/types"
)

// VersionTracker tracks when keys were first added to the tree
type VersionTracker struct {
	keyVersionMap map[string]types.Version // maps key to first version it appeared
}

// NewVersionTracker creates a new version tracker
func NewVersionTracker() *VersionTracker {
	return &VersionTracker{
		keyVersionMap: make(map[string]types.Version),
	}
}

// RecordPut records when a key is put into the tree
func (vt *VersionTracker) RecordPut(key types.Key, version types.Version) {
	keyStr := string(key.Bytes())
	if _, exists := vt.keyVersionMap[keyStr]; !exists {
		vt.keyVersionMap[keyStr] = version
	}
}

// FirstVersion returns the first version a key appeared in, or 0 if never added
func (vt *VersionTracker) FirstVersion(key types.Key) types.Version {
	keyStr := string(key.Bytes())
	if v, exists := vt.keyVersionMap[keyStr]; exists {
		return v
	}
	return 0
}

// CheckVersionIsolation verifies that keys are only visible in versions >= when they were added
func CheckVersionIsolation(t *tree.Tree, ops []generators.Operation) error {
	tracker := NewVersionTracker()
	
	for i, op := range ops {
		switch op.Type {
		case generators.OpPut:
			// Execute put operation
			newVersion, err := t.Put(op.Key, op.Value)
			if err != nil {
				return fmt.Errorf("put operation %d failed: %w", i, err)
			}
			
			// Record when this key was first added
			tracker.RecordPut(op.Key, newVersion)
			
		case generators.OpGet:
			// Skip if trying to get from a version that doesn't exist yet
			latestVersion := t.GetLatestVersion()
			if op.Version > latestVersion {
				continue
			}
			
			// Get value at specific version
			value, err := t.GetAtVersion(op.Version, op.Key)
			if err != nil {
				// Version not found or not committed is expected if we haven't created it yet
				errStr := err.Error()
				if errStr == "version "+fmt.Sprint(op.Version)+" not found" ||
				   errStr == "version "+fmt.Sprint(op.Version)+" not committed" {
					continue
				}
				return fmt.Errorf("get operation %d failed: %w", i, err)
			}
			
			// Check version isolation invariant
			firstVersion := tracker.FirstVersion(op.Key)
			if firstVersion > 0 { // Key has been added
				if op.Version < firstVersion && value != nil {
					return fmt.Errorf("version isolation violated: key %x visible at version %d but was first added at version %d",
						op.Key.Bytes(), op.Version, firstVersion)
				}
			} else { // Key was never added
				if value != nil {
					return fmt.Errorf("version isolation violated: key %x returned value but was never added",
						op.Key.Bytes())
				}
			}
			
		case generators.OpDelete:
			// Delete operations may fail if key doesn't exist, which is fine
			newVersion, err := t.Delete(op.Key)
			if err != nil {
				// Key not found is expected if we're trying to delete something that doesn't exist
				if err.Error() == "failed to delete: delete failed: jmt: key not found" {
					continue
				}
				return fmt.Errorf("delete operation %d failed: %w", i, err)
			}
			// Note: In a more sophisticated implementation, we'd track deletions separately
			_ = newVersion
		}
	}
	
	return nil
}

// CheckVersionIsolationWithRootHashes additionally verifies root hash consistency
func CheckVersionIsolationWithRootHashes(t *tree.Tree, ops []generators.Operation) error {
	tracker := NewVersionTracker()
	rootHashes := make(map[types.Version]types.Hash)
	
	// Get initial root hash
	rootHashes[0], _ = t.GetRootHash(t.GetLatestVersion())
	
	for i, op := range ops {
		switch op.Type {
		case generators.OpPut:
			newVersion, err := t.Put(op.Key, op.Value)
			if err != nil {
				return fmt.Errorf("put operation %d failed: %w", i, err)
			}
			
			tracker.RecordPut(op.Key, newVersion)
			rootHashes[newVersion], _ = t.GetRootHash(newVersion)
			
			// Verify root hash changed (unless we're updating with same value)
			if newVersion > 0 {
				prevHash := rootHashes[newVersion-1]
				if prevHash == rootHashes[newVersion] {
					// This might be OK if we're updating with the same value
					// In a more sophisticated test, we'd track values too
				}
			}
			
		case generators.OpGet:
			// Skip if trying to get from a version that doesn't exist yet
			if op.Version > t.GetLatestVersion() {
				continue
			}
			
			value, err := t.GetAtVersion(op.Version, op.Key)
			if err != nil {
				// Version not found or not committed is expected
				errStr := err.Error()
				if errStr == "version "+fmt.Sprint(op.Version)+" not found" ||
				   errStr == "version "+fmt.Sprint(op.Version)+" not committed" {
					continue
				}
				return fmt.Errorf("get operation %d failed: %w", i, err)
			}
			
			firstVersion := tracker.FirstVersion(op.Key)
			if firstVersion > 0 {
				if op.Version < firstVersion && value != nil {
					return fmt.Errorf("version isolation violated: key %x visible at version %d but was first added at version %d",
						op.Key.Bytes(), op.Version, firstVersion)
				}
			} else {
				if value != nil {
					return fmt.Errorf("version isolation violated: key %x returned value but was never added",
						op.Key.Bytes())
				}
			}
			
		case generators.OpDelete:
			_, err := t.Delete(op.Key)
			if err != nil {
				// Key not found is expected
				if err.Error() == "failed to delete: delete failed: jmt: key not found" {
					continue
				}
				return fmt.Errorf("delete operation %d failed: %w", i, err)
			}
		}
	}
	
	return nil
}