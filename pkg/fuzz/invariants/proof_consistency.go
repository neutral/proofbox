package invariants

import (
	"bytes"
	"fmt"

	"github.com/neutral/proofbox/pkg/fuzz/generators"
	"github.com/neutral/proofbox/pkg/proof"
	"github.com/neutral/proofbox/pkg/tree"
	"github.com/neutral/proofbox/pkg/types"
)

// ProofConsistencyTracker tracks proofs across versions
type ProofConsistencyTracker struct {
	// Maps version -> key -> proof
	proofs map[types.Version]map[string]*proof.Proof
	// Maps version -> root hash
	rootHashes map[types.Version]types.Hash
}

// NewProofConsistencyTracker creates a new tracker
func NewProofConsistencyTracker() *ProofConsistencyTracker {
	return &ProofConsistencyTracker{
		proofs:     make(map[types.Version]map[string]*proof.Proof),
		rootHashes: make(map[types.Version]types.Hash),
	}
}

// CheckProofConsistency verifies proof generation and verification consistency
func CheckProofConsistency(t *tree.Tree, ops []generators.Operation) error {
	
	// Track keys that exist at each version
	versionKeys := make(map[types.Version]map[string][]byte)
	versionKeys[0] = make(map[string][]byte)
	
	for i, op := range ops {
		switch op.Type {
		case generators.OpPut:
			newVersion, err := t.Put(op.Key, op.Value)
			if err != nil {
				return fmt.Errorf("put operation %d failed: %w", i, err)
			}
			
			// Copy previous version's keys
			versionKeys[newVersion] = make(map[string][]byte)
			if newVersion > 0 {
				for k, v := range versionKeys[newVersion-1] {
					versionKeys[newVersion][k] = v
				}
			}
			versionKeys[newVersion][string(op.Key.Bytes())] = op.Value
			
			// Generate and verify proof for this key
			if err := verifyProofAtVersion(t, newVersion, op.Key, op.Value); err != nil {
				return fmt.Errorf("proof verification failed at version %d: %w", newVersion, err)
			}
			
		case generators.OpDelete:
			newVersion, err := t.Delete(op.Key)
			if err != nil {
				if err.Error() == "failed to delete: delete failed: jmt: key not found" {
					continue
				}
				return fmt.Errorf("delete operation %d failed: %w", i, err)
			}
			
			// Copy previous version's keys and remove deleted key
			versionKeys[newVersion] = make(map[string][]byte)
			if newVersion > 0 {
				for k, v := range versionKeys[newVersion-1] {
					if k != string(op.Key.Bytes()) {
						versionKeys[newVersion][k] = v
					}
				}
			}
			
			// Verify non-existence proof
			if err := verifyNonExistenceProof(t, newVersion, op.Key); err != nil {
				return fmt.Errorf("non-existence proof verification failed at version %d: %w", newVersion, err)
			}
			
		case generators.OpGet:
			// Verify we can generate valid proofs for gets
			if op.Version > t.GetLatestVersion() {
				continue
			}
			
			// Try to generate a proof at this version
			reader, err := t.Reader(op.Version)
			if err != nil {
				// Version not found or not committed
				errStr := err.Error()
				if errStr == "version "+fmt.Sprint(op.Version)+" not found" ||
				   errStr == "version "+fmt.Sprint(op.Version)+" not committed" {
					continue
				}
				return fmt.Errorf("failed to create reader at version %d: %w", op.Version, err)
			}
			defer reader.Close()
			
			genProof, err := proof.Generate(reader, op.Key)
			if err != nil {
				return fmt.Errorf("failed to generate proof at version %d: %w", op.Version, err)
			}
			
			// Verify the proof
			if err := proof.Verify(genProof); err != nil {
				return fmt.Errorf("proof verification failed at version %d: %w", op.Version, err)
			}
		}
	}
	
	// Cross-version consistency check
	return checkCrossVersionConsistency(t, versionKeys)
}

// verifyProofAtVersion generates and verifies a proof for a key at a specific version
func verifyProofAtVersion(t *tree.Tree, version types.Version, key types.Key, expectedValue []byte) error {
	reader, err := t.Reader(version)
	if err != nil {
		return fmt.Errorf("failed to create reader: %w", err)
	}
	defer reader.Close()
	
	// Generate proof
	genProof, err := proof.Generate(reader, key)
	if err != nil {
		return fmt.Errorf("failed to generate proof: %w", err)
	}
	
	// Verify proof
	if err := proof.Verify(genProof); err != nil {
		return fmt.Errorf("proof verification failed: %w", err)
	}
	
	// Check proof type and value
	if genProof.Type != proof.ProofTypeInclusion {
		return fmt.Errorf("expected inclusion proof, got %v", genProof.Type)
	}
	
	if !bytes.Equal(genProof.Value, expectedValue) {
		return fmt.Errorf("proof value mismatch: expected %x, got %x", expectedValue, genProof.Value)
	}
	
	// Verify proof fails with wrong root hash
	wrongProof := *genProof
	wrongProof.RootHash[0] ^= 0xFF
	if err := proof.Verify(&wrongProof); err == nil {
		return fmt.Errorf("proof should fail with wrong root hash")
	}
	
	return nil
}

// verifyNonExistenceProof verifies a non-existence proof
func verifyNonExistenceProof(t *tree.Tree, version types.Version, key types.Key) error {
	reader, err := t.Reader(version)
	if err != nil {
		return fmt.Errorf("failed to create reader: %w", err)
	}
	defer reader.Close()
	
	// Generate proof
	genProof, err := proof.Generate(reader, key)
	if err != nil {
		return fmt.Errorf("failed to generate non-existence proof: %w", err)
	}
	
	// Verify proof
	if err := proof.Verify(genProof); err != nil {
		return fmt.Errorf("non-existence proof verification failed: %w", err)
	}
	
	// Check proof type
	if genProof.Type == proof.ProofTypeInclusion {
		return fmt.Errorf("expected non-existence proof, got inclusion proof")
	}
	
	// Value should be nil for non-existence
	if genProof.Value != nil {
		return fmt.Errorf("non-existence proof should have nil value, got %x", genProof.Value)
	}
	
	return nil
}

// checkCrossVersionConsistency verifies proofs remain valid across versions for unchanged keys
func checkCrossVersionConsistency(t *tree.Tree, versionKeys map[types.Version]map[string][]byte) error {
	// For each version, check that proofs for unchanged keys remain consistent
	versions := make([]types.Version, 0, len(versionKeys))
	for v := range versionKeys {
		versions = append(versions, v)
	}
	
	// Sort versions
	for i := 0; i < len(versions)-1; i++ {
		for j := i + 1; j < len(versions); j++ {
			if versions[i] > versions[j] {
				versions[i], versions[j] = versions[j], versions[i]
			}
		}
	}
	
	// Check consistency between adjacent versions
	for i := 1; i < len(versions); i++ {
		prevVersion := versions[i-1]
		currVersion := versions[i]
		
		// For keys that exist in both versions with same value
		for keyStr, prevValue := range versionKeys[prevVersion] {
			if currValue, exists := versionKeys[currVersion][keyStr]; exists && bytes.Equal(prevValue, currValue) {
				// The key wasn't modified between versions
				// Proofs should be structurally similar (same siblings at unchanged depths)
				// keyStr contains the actual key bytes, not a string representation
				var key types.Key
				copy(key[:], []byte(keyStr))
				
				// Generate proofs at both versions
				reader1, err := t.Reader(prevVersion)
				if err != nil {
					continue // Skip if version not available
				}
				defer reader1.Close()
				
				reader2, err := t.Reader(currVersion)
				if err != nil {
					continue
				}
				defer reader2.Close()
				
				proof1, err := proof.Generate(reader1, key)
				if err != nil {
					return fmt.Errorf("failed to generate proof at version %d: %w", prevVersion, err)
				}
				
				proof2, err := proof.Generate(reader2, key)
				if err != nil {
					return fmt.Errorf("failed to generate proof at version %d: %w", currVersion, err)
				}
				
				// Both should be inclusion proofs with same value
				if proof1.Type != proof.ProofTypeInclusion || proof2.Type != proof.ProofTypeInclusion {
					return fmt.Errorf("proof type changed for unchanged key")
				}
				
				if !bytes.Equal(proof1.Value, proof2.Value) {
					return fmt.Errorf("proof value changed for unchanged key")
				}
			}
		}
	}
	
	return nil
}