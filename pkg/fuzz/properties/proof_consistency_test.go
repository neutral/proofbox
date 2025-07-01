package properties

import (
	"testing"

	"github.com/neutral/proofbox/pkg/fuzz"
	"github.com/neutral/proofbox/pkg/fuzz/generators"
	"github.com/neutral/proofbox/pkg/fuzz/invariants"
	"github.com/neutral/proofbox/pkg/proof"
	"github.com/neutral/proofbox/pkg/types"
	"pgregory.net/rapid"
)

func TestProofConsistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate operations
		ops := generators.OperationSliceGen(1, 30).Draw(t, "operations")
		
		// Create tree
		tr, err := fuzz.NewTestTree()
		if err != nil {
			t.Fatalf("failed to create tree: %v", err)
		}
		
		// Check proof consistency
		if err := invariants.CheckProofConsistency(tr, ops); err != nil {
			t.Fatalf("proof consistency check failed: %v", err)
		}
	})
}

func TestProofInvalidation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Create tree
		tr, err := fuzz.NewTestTree()
		if err != nil {
			t.Fatalf("failed to create tree: %v", err)
		}
		
		// Generate two different keys
		key1 := generators.OperationGen().Draw(t, "key1").Key
		key2 := generators.OperationGen().Draw(t, "key2").Key
		
		// Ensure keys are different
		if string(key1.Bytes()) == string(key2.Bytes()) {
			key2[0] ^= 0xFF
		}
		
		value1 := rapid.SliceOfN(rapid.Byte(), 1, 100).Draw(t, "value1")
		value2 := rapid.SliceOfN(rapid.Byte(), 1, 100).Draw(t, "value2")
		
		// Put key1
		v1, err := tr.Put(key1, value1)
		if err != nil {
			t.Fatalf("failed to put key1: %v", err)
		}
		
		// Generate proof for key1 at v1
		reader1, err := tr.Reader(v1)
		if err != nil {
			t.Fatalf("failed to create reader: %v", err)
		}
		defer reader1.Close()
		
		proof1, err := proof.Generate(reader1, key1)
		if err != nil {
			t.Fatalf("failed to generate proof: %v", err)
		}
		
		// Put key2 (changes root)
		v2, err := tr.Put(key2, value2)
		if err != nil {
			t.Fatalf("failed to put key2: %v", err)
		}
		
		// Verify proof1 is still valid at v1 but not at v2
		rootHashV1, err := tr.GetRootHash(v1)
		if err != nil {
			t.Fatalf("failed to get root hash v1: %v", err)
		}
		
		rootHashV2, err := tr.GetRootHash(v2)
		if err != nil {
			t.Fatalf("failed to get root hash v2: %v", err)
		}
		
		// Proof should verify with correct root hash
		if proof1.RootHash != rootHashV1 {
			t.Fatalf("proof has wrong root hash")
		}
		
		if err := proof.Verify(proof1); err != nil {
			t.Fatalf("proof should verify at v1: %v", err)
		}
		
		// Proof should fail with v2's root hash
		proof1.RootHash = rootHashV2
		if err := proof.Verify(proof1); err == nil {
			t.Fatalf("proof should not verify with v2's root hash")
		}
	})
}

func TestConcurrentProofGeneration(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Create tree with some data
		tr, err := fuzz.NewTestTree()
		if err != nil {
			t.Fatalf("failed to create tree: %v", err)
		}
		
		// Add some keys
		numKeys := rapid.IntRange(5, 20).Draw(t, "num_keys")
		keys := make([]types.Key, numKeys)
		
		for i := 0; i < numKeys; i++ {
			key := generators.OperationGen().Draw(t, "key").Key
			value := rapid.SliceOfN(rapid.Byte(), 1, 100).Draw(t, "value")
			
			_, err := tr.Put(key, value)
			if err != nil {
				t.Fatalf("failed to put key: %v", err)
			}
			keys[i] = key
		}
		
		latestVersion := tr.GetLatestVersion()
		
		// Generate proofs concurrently
		type proofResult struct {
			proof *proof.Proof
			err   error
		}
		
		results := make(chan proofResult, numKeys)
		
		for _, key := range keys {
			go func(k types.Key) {
				reader, err := tr.Reader(latestVersion)
				if err != nil {
					results <- proofResult{nil, err}
					return
				}
				defer reader.Close()
				
				p, err := proof.Generate(reader, k)
				results <- proofResult{p, err}
			}(key)
		}
		
		// Collect and verify all proofs
		for i := 0; i < numKeys; i++ {
			result := <-results
			if result.err != nil {
				t.Fatalf("failed to generate proof: %v", result.err)
			}
			
			if err := proof.Verify(result.proof); err != nil {
				t.Fatalf("proof verification failed: %v", err)
			}
		}
	})
}