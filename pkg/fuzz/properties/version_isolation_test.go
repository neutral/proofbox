package properties

import (
	"bytes"
	"testing"

	"github.com/neutral/proofbox/pkg/fuzz"
	"github.com/neutral/proofbox/pkg/fuzz/generators"
	"github.com/neutral/proofbox/pkg/fuzz/invariants"
	"github.com/neutral/proofbox/pkg/types"
	"pgregory.net/rapid"
)

func TestVersionIsolation(t *testing.T) {
	config := fuzz.GetTestConfig()
	config.ApplyToRapid(t)

	rapid.Check(t, func(t *rapid.T) {
		// Generate a sequence of operations
		ops := generators.OperationSliceGen(1, 100).Draw(t, "operations")

		// Create a new tree for testing
		tr, err := fuzz.NewTestTree()
		if err != nil {
			t.Fatalf("failed to create tree: %v", err)
		}

		// Execute operations and check version isolation
		if err := invariants.CheckVersionIsolation(tr, ops); err != nil {
			t.Fatalf("version isolation check failed: %v", err)
		}
	})
}

func TestVersionIsolationWithStructuredKeys(t *testing.T) {
	config := fuzz.GetTestConfig()
	config.ApplyToRapid(t)

	rapid.Check(t, func(t *rapid.T) {
		// Generate operations with a higher chance of structured keys
		// This tests internal node sharing scenarios
		ops := rapid.SliceOfN(generators.OperationGen(), 1, 50).Draw(t, "operations")

		// Create a new tree
		tr, err := fuzz.NewTestTree()
		if err != nil {
			t.Fatalf("failed to create tree: %v", err)
		}

		// Execute and verify
		if err := invariants.CheckVersionIsolationWithRootHashes(tr, ops); err != nil {
			t.Fatalf("version isolation check failed: %v", err)
		}
	})
}

func TestVersionIsolationEdgeCases(t *testing.T) {
	config := fuzz.GetTestConfig()
	config.ApplyToRapid(t)

	rapid.Check(t, func(t *rapid.T) {
		// Create a tree
		tr, err := fuzz.NewTestTree()
		if err != nil {
			t.Fatalf("failed to create tree: %v", err)
		}

		// Generate a key that will be used throughout the test
		key := generators.OperationGen().Draw(t, "key_gen").Key
		value1 := rapid.SliceOfN(rapid.Byte(), 1, 100).Draw(t, "value1")
		value2 := rapid.SliceOfN(rapid.Byte(), 1, 100).Draw(t, "value2")

		// Put key at version 1
		v1, err := tr.Put(key, value1)
		if err != nil {
			t.Fatalf("failed to put key: %v", err)
		}

		// Verify key is not visible at version 0
		val, err := tr.GetAtVersion(0, key)
		if err != nil {
			t.Fatalf("failed to get at version 0: %v", err)
		}
		if val != nil {
			t.Fatalf("key should not be visible at version 0")
		}

		// Verify key is visible at version 1
		val, err = tr.GetAtVersion(v1, key)
		if err != nil {
			t.Fatalf("failed to get at version 1: %v", err)
		}
		if val == nil {
			t.Fatalf("key should be visible at version %d", v1)
		}

		// Update key at version 2
		v2, err := tr.Put(key, value2)
		if err != nil {
			t.Fatalf("failed to update key: %v", err)
		}

		// Verify old value at version 1
		val, err = tr.GetAtVersion(v1, key)
		if err != nil {
			t.Fatalf("failed to get at version %d: %v", v1, err)
		}
		if !bytes.Equal(val, value1) {
			t.Fatalf("expected old value at version %d", v1)
		}

		// Verify new value at version 2
		val, err = tr.GetAtVersion(v2, key)
		if err != nil {
			t.Fatalf("failed to get at version %d: %v", v2, err)
		}
		if !bytes.Equal(val, value2) {
			t.Fatalf("expected new value at version %d", v2)
		}
	})
}

// TestVersionIsolationRegression tests the specific bug pattern from step 14
func TestVersionIsolationRegression(t *testing.T) {
	config := fuzz.GetTestConfig()
	config.ApplyToRapid(t)

	rapid.Check(t, func(t *rapid.T) {
		// Create a tree
		tr, err := fuzz.NewTestTree()
		if err != nil {
			t.Fatalf("failed to create tree: %v", err)
		}

		// Generate two keys that might share internal nodes
		key1Bytes := rapid.SliceOfN(rapid.Byte(), 8, 8).Draw(t, "key1")
		key2Bytes := make([]byte, len(key1Bytes))
		copy(key2Bytes, key1Bytes)
		// Modify last byte to create a different key that shares prefix
		key2Bytes[len(key2Bytes)-1] = key1Bytes[len(key1Bytes)-1] ^ 0x01

		key1 := types.KeyHash(key1Bytes)
		key2 := types.KeyHash(key2Bytes)

		value := rapid.SliceOfN(rapid.Byte(), 1, 100).Draw(t, "value")

		// Add key1 at version 1
		v1, err := tr.Put(key1, value)
		if err != nil {
			t.Fatalf("failed to put key1: %v", err)
		}

		// Verify key2 is not visible at any version <= v1
		for v := types.Version(0); v <= v1; v++ {
			val, err := tr.GetAtVersion(v, key2)
			if err != nil {
				t.Fatalf("failed to get key2 at version %d: %v", v, err)
			}
			if val != nil {
				t.Fatalf("key2 should not be visible at version %d (was added after)", v)
			}
		}

		// Add key2 at version 2
		v2, err := tr.Put(key2, value)
		if err != nil {
			t.Fatalf("failed to put key2: %v", err)
		}

		// Verify key2 is still not visible at version 1
		val, err := tr.GetAtVersion(v1, key2)
		if err != nil {
			t.Fatalf("failed to get key2 at version %d: %v", v1, err)
		}
		if val != nil {
			t.Fatalf("regression: key2 visible at version %d but was added at version %d", v1, v2)
		}
	})
}
