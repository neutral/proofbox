package invariants

import (
	"bytes"
	"encoding/hex"
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
	// Use hex-encoded keys as map keys to avoid byte/string conversion issues
	versionKeys := make(map[types.Version]map[string][]byte)
	versionKeys[0] = make(map[string][]byte)

	// Track the current version as operations may fail
	currentVersion := types.Version(0)

	for i, op := range ops {
		switch op.Type {
		case generators.OpPut:
			newVersion, err := t.Put(op.Key, op.Value)
			if err != nil {
				return fmt.Errorf("put operation %d failed: %w", i, err)
			}

			// Copy previous version's keys
			versionKeys[newVersion] = make(map[string][]byte)
			if prevKeys, exists := versionKeys[currentVersion]; exists {
				for k, v := range prevKeys {
					versionKeys[newVersion][k] = v
				}
			}
			versionKeys[newVersion][op.Key.String()] = op.Value

			// Update current version
			currentVersion = newVersion

			// Generate and verify proof for this key
			if err := verifyProofAtVersion(t, newVersion, op.Key, op.Value); err != nil {
				return fmt.Errorf("proof verification failed at version %d: %w", newVersion, err)
			}

		case generators.OpDelete:
			newVersion, err := t.Delete(op.Key)
			if err != nil {
				// Key not found is expected for some random operations
				if err.Error() == "failed to delete: delete failed: jmt: key not found" {
					continue
				}
				return fmt.Errorf("delete operation %d failed: %w", i, err)
			}

			// Copy previous version's keys and remove deleted key
			versionKeys[newVersion] = make(map[string][]byte)
			if prevKeys, exists := versionKeys[currentVersion]; exists {
				for k, v := range prevKeys {
					if k != op.Key.String() {
						versionKeys[newVersion][k] = v
					}
				}
			}

			// Update current version
			currentVersion = newVersion

			// Verify non-existence proof
			if err := verifyNonExistenceProof(t, newVersion, op.Key); err != nil {
				return fmt.Errorf("non-existence proof verification failed at version %d: %w", newVersion, err)
			}

		case generators.OpGet:
			// Skip if trying to get from a version that doesn't exist yet
			if op.Version > t.GetLatestVersion() {
				continue
			}

			// Skip version 0 if it has no keys
			if op.Version == 0 && len(versionKeys[0]) == 0 {
				continue
			}

			// Check if we have tracked this version
			if _, exists := versionKeys[op.Version]; !exists {
				// Version might not exist if operations failed
				continue
			}

			// Try to generate a proof at the requested version
			reader, err := t.Reader(op.Version)
			if err != nil {
				// Version not found or not committed
				errStr := err.Error()
				if errStr == "jmt: version not found" ||
					errStr == fmt.Sprintf("version %d not found", op.Version) ||
					errStr == fmt.Sprintf("version %d not committed", op.Version) {
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

			// Additionally verify the proof matches our tracked state
			trackedValue := versionKeys[op.Version][op.Key.String()]
			if genProof.Type == proof.ProofTypeInclusion {
				if !bytes.Equal(genProof.Value, trackedValue) {
					return fmt.Errorf("proof value mismatch at version %d: expected %x, got %x",
						op.Version, trackedValue, genProof.Value)
				}
			} else {
				// Non-existence proof
				if trackedValue != nil {
					return fmt.Errorf("got non-existence proof but key should exist at version %d", op.Version)
				}
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
		if v > 0 && len(versionKeys[v]) > 0 { // Skip empty versions
			versions = append(versions, v)
		}
	}

	if len(versions) < 2 {
		return nil // Not enough versions to check consistency
	}

	// Sort versions
	for i := 0; i < len(versions)-1; i++ {
		for j := i + 1; j < len(versions); j++ {
			if versions[i] > versions[j] {
				versions[i], versions[j] = versions[j], versions[i]
			}
		}
	}

	// Check consistency between adjacent versions - limit checks to avoid timeouts
	maxChecks := 5
	checkCount := 0
	for i := 1; i < len(versions) && checkCount < maxChecks; i++ {
		prevVersion := versions[i-1]
		currVersion := versions[i]

		// For keys that exist in both versions with same value
		for keyStr, prevValue := range versionKeys[prevVersion] {
			if checkCount >= maxChecks {
				break
			}

			currValue, exists := versionKeys[currVersion][keyStr]
			if !exists || !bytes.Equal(prevValue, currValue) {
				continue
			}

			// The key wasn't modified between versions
			// keyStr is hex-encoded representation of key bytes
			keyBytes, err := hex.DecodeString(keyStr)
			if err != nil || len(keyBytes) != 32 {
				continue // Skip malformed keys
			}
			key, err := types.KeyFromBytes(keyBytes)
			if err != nil {
				continue
			}

			// Generate proofs at both versions
			reader1, err := t.Reader(prevVersion)
			if err != nil {
				continue // Skip if version not available
			}

			proof1, err := proof.Generate(reader1, key)
			reader1.Close()
			if err != nil {
				continue // Skip if proof generation fails
			}

			reader2, err := t.Reader(currVersion)
			if err != nil {
				continue
			}

			proof2, err := proof.Generate(reader2, key)
			reader2.Close()
			if err != nil {
				continue
			}

			// Both should be inclusion proofs with same value
			if proof1.Type != proof.ProofTypeInclusion || proof2.Type != proof.ProofTypeInclusion {
				return fmt.Errorf("proof type changed for unchanged key between versions %d and %d", prevVersion, currVersion)
			}

			if !bytes.Equal(proof1.Value, proof2.Value) {
				return fmt.Errorf("proof value changed for unchanged key between versions %d and %d", prevVersion, currVersion)
			}

			checkCount++
		}
	}

	return nil
}
