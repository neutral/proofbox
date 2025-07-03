package fuzz

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/neutral/proofbox/pkg/proof"
	"github.com/neutral/proofbox/pkg/types"
)

// FuzzTreeOperations tests tree operations with fuzzing
func FuzzTreeOperations(f *testing.F) {
	// Add seed corpus
	f.Add([]byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x02, 0x03, 0x04, 0x05})
	f.Add([]byte{0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff, 0xfe, 0xfd, 0xfc, 0xfb})
	f.Add([]byte{0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x11, 0x22, 0x33, 0x44})

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) < 9 {
			return // Need at least op type + 8 bytes for simple parsing
		}

		// Create tree
		tr, err := NewTestTree()
		if err != nil {
			t.Fatalf("failed to create tree: %v", err)
		}

		// Parse operations from fuzz input
		ops := parseOperations(data)

		// Track versions for validation
		versions := []types.Version{0}

		// Execute operations
		for _, op := range ops {
			switch op.Type {
			case 0: // Put
				newVersion, err := tr.Put(op.Key, op.Value)
				if err != nil {
					// Some errors are expected (e.g., invalid inputs)
					continue
				}
				versions = append(versions, newVersion)

				// Verify we can read back what we just wrote
				val, err := tr.Get(newVersion, op.Key)
				if err != nil {
					t.Fatalf("failed to read back key: %v", err)
				}
				if !bytes.Equal(val, op.Value) {
					t.Fatalf("value mismatch: expected %x, got %x", op.Value, val)
				}

			case 1: // Get
				// Try to get at various versions
				for _, v := range versions {
					_, err := tr.GetAtVersion(v, op.Key)
					if err != nil {
						// Some errors expected for non-existent keys
						continue
					}
				}

			case 2: // Delete
				_, err := tr.Delete(op.Key)
				if err != nil {
					continue
				}
			}
		}

		// Verify tree invariants
		if err := verifyTreeInvariants(tr); err != nil {
			t.Fatalf("tree invariant violated: %v", err)
		}
	})
}

// FuzzProofGeneration tests proof generation and verification
func FuzzProofGeneration(f *testing.F) {
	// Add seed corpus
	f.Add([]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08})
	f.Add([]byte{0xff, 0xfe, 0xfd, 0xfc, 0xfb, 0xfa, 0xf9, 0xf8})

	f.Fuzz(func(t *testing.T, keyData []byte) {
		if len(keyData) == 0 {
			return
		}

		// Create tree and add some data
		tr, err := NewTestTree()
		if err != nil {
			t.Fatalf("failed to create tree: %v", err)
		}

		// Add the fuzzed key
		key := types.KeyHash(keyData)
		value := []byte("test_value")

		newVersion, err := tr.Put(key, value)
		if err != nil {
			return // Some keys might be invalid
		}

		// Generate proof
		reader, err := tr.Reader(newVersion)
		if err != nil {
			t.Fatalf("failed to create reader: %v", err)
		}
		defer reader.Close()

		genProof, err := proof.Generate(reader, key)
		if err != nil {
			t.Fatalf("failed to generate proof: %v", err)
		}

		// Verify proof
		rootHash, err := tr.GetRootHash(newVersion)
		if err != nil {
			t.Fatalf("failed to get root hash: %v", err)
		}
		// Verify the proof
		err = proof.Verify(genProof)
		if err != nil {
			t.Fatalf("proof verification failed: %v", err)
		}

		// Check the value matches
		if !bytes.Equal(genProof.Value, value) {
			t.Fatalf("proof verified incorrect value: expected %x, got %x", value, genProof.Value)
		}
		if err != nil {
			t.Fatalf("proof verification failed: %v", err)
		}

		// Test non-existence proof
		nonExistentKey := make([]byte, len(keyData))
		copy(nonExistentKey, keyData)
		nonExistentKey[0] ^= 0xFF // Flip bits to create different key

		nonExistenceProof, err := proof.Generate(reader, types.KeyHash(nonExistentKey))
		if err != nil {
			t.Fatalf("failed to generate non-existence proof: %v", err)
		}

		// Verify the non-existence proof
		err = proof.Verify(nonExistenceProof)
		if err != nil {
			t.Fatalf("non-existence proof verification failed: %v", err)
		}

		// For non-existence proof, Value should be nil
		if nonExistenceProof.Value != nil {
			t.Fatalf("non-existence proof should have nil value, got %x", nonExistenceProof.Value)
		}

		// Verify proof fails with wrong root hash
		wrongRootHash := rootHash
		wrongRootHash[0] ^= 0xFF

		// Modify the proof to have wrong root hash
		genProof.RootHash = wrongRootHash
		err = proof.Verify(genProof)
		if err == nil {
			t.Fatalf("proof should fail with wrong root hash")
		}
	})
}

// parseOperations converts fuzz input into operations
func parseOperations(data []byte) []operation {
	var ops []operation

	for len(data) >= 9 {
		opType := data[0] % 3
		keyLen := int(data[1])%32 + 1

		if len(data) < 2+keyLen {
			break
		}

		key := data[2 : 2+keyLen]
		data = data[2+keyLen:]

		var value []byte
		if opType == 0 && len(data) > 0 { // Put operation needs value
			valueLen := int(data[0])%64 + 1
			if len(data) >= 1+valueLen {
				value = data[1 : 1+valueLen]
				data = data[1+valueLen:]
			}
		}

		ops = append(ops, operation{
			Type:  opType,
			Key:   types.KeyHash(key),
			Value: value,
		})

		if len(ops) > 100 {
			break // Limit operations to prevent timeouts
		}
	}

	return ops
}

type operation struct {
	Type  uint8 // 0: Put, 1: Get, 2: Delete
	Key   types.Key
	Value []byte
}

// verifyTreeInvariants checks basic tree invariants
func verifyTreeInvariants(tr interface {
	GetRootHash(types.Version) (types.Hash, error)
	GetLatestVersion() types.Version
}) error {
	// Basic invariant: root hash should be deterministic
	version := tr.GetLatestVersion()
	rootHash1, err := tr.GetRootHash(version)
	if err != nil {
		return fmt.Errorf("failed to get first root hash: %w", err)
	}
	rootHash2, err := tr.GetRootHash(version)
	if err != nil {
		return fmt.Errorf("failed to get second root hash: %w", err)
	}

	if !bytes.Equal(rootHash1[:], rootHash2[:]) {
		return fmt.Errorf("root hash not deterministic: %x != %x", rootHash1, rootHash2)
	}

	// More invariants can be added here
	return nil
}
