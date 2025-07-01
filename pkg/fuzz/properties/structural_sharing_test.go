package properties

import (
	"testing"

	"github.com/neutral/proofbox/pkg/fuzz"
	"github.com/neutral/proofbox/pkg/fuzz/generators"
	"github.com/neutral/proofbox/pkg/fuzz/invariants"
	"github.com/neutral/proofbox/pkg/types"
	"pgregory.net/rapid"
)

func TestStructuralSharing(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a sequence of operations
		ops := generators.OperationSliceGen(5, 50).Draw(t, "operations")
		
		// Create a new tree
		tr, err := fuzz.NewTestTree()
		if err != nil {
			t.Fatalf("failed to create tree: %v", err)
		}
		
		// Execute operations and check structural sharing
		if err := invariants.CheckStructuralSharing(tr, ops); err != nil {
			t.Fatalf("structural sharing check failed: %v", err)
		}
	})
}

func TestStructuralSharingMetrics(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Create a tree
		tr, err := fuzz.NewTestTree()
		if err != nil {
			t.Fatalf("failed to create tree: %v", err)
		}
		
		// Generate operations that create multiple versions
		numVersions := rapid.IntRange(2, 10).Draw(t, "num_versions")
		keysPerVersion := rapid.IntRange(1, 5).Draw(t, "keys_per_version")
		
		// Create versions with some overlapping keys
		baseKeys := []string{"key1", "key2", "key3", "key4", "key5"}
		
		for v := 0; v < numVersions; v++ {
			// Put some keys
			for k := 0; k < keysPerVersion; k++ {
				keyIdx := rapid.IntRange(0, len(baseKeys)-1).Draw(t, "key_idx")
				key := types.KeyHash([]byte(baseKeys[keyIdx]))
				value := rapid.SliceOfN(rapid.Byte(), 1, 100).Draw(t, "value")
				
				_, err := tr.Put(key, value)
				if err != nil {
					t.Fatalf("failed to put key: %v", err)
				}
			}
		}
		
		// Calculate metrics
		metrics, err := invariants.CalculateStructuralSharingMetrics(tr)
		if err != nil {
			t.Fatalf("failed to calculate metrics: %v", err)
		}
		
		// Verify metrics are reasonable
		// Note: actual version count may vary due to overwrites creating new versions
		if metrics.VersionCount < numVersions {
			t.Fatalf("expected at least %d versions, got %d", numVersions, metrics.VersionCount)
		}
	})
}

func TestPathConsistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Create a tree
		tr, err := fuzz.NewTestTree()
		if err != nil {
			t.Fatalf("failed to create tree: %v", err)
		}
		
		// Generate a key that will remain unchanged
		unchangedKey := generators.OperationGen().Draw(t, "unchanged_key").Key
		unchangedValue := rapid.SliceOfN(rapid.Byte(), 1, 100).Draw(t, "unchanged_value")
		
		// Put the unchanged key
		v1, err := tr.Put(unchangedKey, unchangedValue)
		if err != nil {
			t.Fatalf("failed to put unchanged key: %v", err)
		}
		
		// Make changes to other keys
		numChanges := rapid.IntRange(1, 10).Draw(t, "num_changes")
		v2 := v1
		
		for i := 0; i < numChanges; i++ {
			// Generate a different key
			otherKey := generators.OperationGen().Draw(t, "other_key").Key
			if string(otherKey.Bytes()) == string(unchangedKey.Bytes()) {
				continue // Skip if we randomly got the same key
			}
			
			value := rapid.SliceOfN(rapid.Byte(), 1, 100).Draw(t, "value")
			v2, err = tr.Put(otherKey, value)
			if err != nil {
				t.Fatalf("failed to put other key: %v", err)
			}
		}
		
		// Verify path consistency for the unchanged key
		if err := invariants.VerifyPathConsistency(tr, unchangedKey, v1, v2); err != nil {
			t.Fatalf("path consistency check failed: %v", err)
		}
	})
}