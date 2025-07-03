package proof

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/neutral/proofbox/pkg/storage"
	"github.com/neutral/proofbox/pkg/storage/memory"
	"github.com/neutral/proofbox/pkg/tree"
	"github.com/neutral/proofbox/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestTree(t testing.TB) *tree.Tree {
	store := memory.NewStorage()
	t.Cleanup(func() { store.Close() })

	keyEncoder := storage.NewDefaultKeyEncoder()
	config := tree.DefaultTreeConfig()
	config.MetricsEnabled = false
	tree, err := tree.NewTree(store, keyEncoder, config)
	require.NoError(t, err)
	return tree
}

func TestInclusionProof(t *testing.T) {
	tr := createTestTree(t)
	key := types.KeyHash([]byte("test-key"))
	value := []byte("test-value")

	// Insert value
	version, err := tr.Put(key, value)
	require.NoError(t, err)

	// Create reader
	reader, err := tr.Reader(version)
	require.NoError(t, err)
	defer reader.Close()

	// Generate proof
	gen := NewGenerator(reader)
	proof, err := gen.Generate(key)
	require.NoError(t, err)
	assert.Equal(t, ProofTypeInclusion, proof.Type)
	assert.Equal(t, value, proof.Value)
	assert.Equal(t, key, proof.Key)
	assert.Equal(t, version, proof.Version)

	// Verify proof
	verifier := NewVerifier()
	err = verifier.Verify(proof)
	assert.NoError(t, err)
}

func TestExclusionEmptyProof(t *testing.T) {
	tr := createTestTree(t)

	// Empty tree case - version 0 is the initial empty state
	// But we need to ensure version 0 exists
	version := tr.GetLatestVersion() // Should be 0 for new tree
	reader, err := tr.Reader(version)
	require.NoError(t, err)
	defer reader.Close()

	key := types.KeyHash([]byte("non-existent"))
	gen := NewGenerator(reader)
	proof, err := gen.Generate(key)
	require.NoError(t, err)
	assert.Equal(t, ProofTypeExclusionEmpty, proof.Type)

	// Verify proof
	verifier := NewVerifier()
	err = verifier.Verify(proof)
	assert.NoError(t, err)
}

func TestExclusionEmptyProofInNonEmptyTree(t *testing.T) {
	tr := createTestTree(t)

	// Insert some keys to create internal structure
	version, err := tr.Put(types.KeyHash([]byte("key1")), []byte("value1"))
	require.NoError(t, err)
	_, err = tr.Put(types.KeyHash([]byte("key3")), []byte("value3"))
	require.NoError(t, err)
	version, err = tr.Put(types.KeyHash([]byte("key5")), []byte("value5"))
	require.NoError(t, err)

	// Try to get proof for key that leads to empty child
	key := types.KeyHash([]byte("key4"))
	reader, err := tr.Reader(version)
	require.NoError(t, err)
	defer reader.Close()

	gen := NewGenerator(reader)
	proof, err := gen.Generate(key)
	require.NoError(t, err)

	// Could be either empty or neighbor depending on tree structure
	assert.True(t, proof.Type == ProofTypeExclusionEmpty || proof.Type == ProofTypeExclusionNeighbor)

	// Debug output
	t.Logf("Proof type: %v", proof.Type)
	t.Logf("Number of siblings: %d", len(proof.Siblings))

	// Verify proof
	verifier := NewVerifier()
	err = verifier.Verify(proof)
	assert.NoError(t, err)
}

func TestExclusionNeighborProof(t *testing.T) {
	tr := createTestTree(t)

	// Insert keys that will create a specific structure
	key1 := types.Key{0x10} // Nibbles: 1,0,0,0,...
	key3 := types.Key{0x30} // Nibbles: 3,0,0,0,...

	_, err := tr.Put(key1, []byte("value1"))
	require.NoError(t, err)
	version, err := tr.Put(key3, []byte("value3"))
	require.NoError(t, err)

	// Generate proof for key2 (should find neighbor)
	key2 := types.Key{0x20} // Nibbles: 2,0,0,0,...
	reader, err := tr.Reader(version)
	require.NoError(t, err)
	defer reader.Close()

	gen := NewGenerator(reader)
	proof, err := gen.Generate(key2)
	require.NoError(t, err)

	// With this structure, should get neighbor proof
	if proof.Type == ProofTypeExclusionNeighbor {
		assert.NotNil(t, proof.NeighborLeaf)
		// Neighbor should be either key1 or key3
		assert.True(t,
			bytes.Equal(proof.NeighborLeaf.Key[:], key1[:]) ||
				bytes.Equal(proof.NeighborLeaf.Key[:], key3[:]))
	}

	// Verify proof
	verifier := NewVerifier()
	err = verifier.Verify(proof)
	assert.NoError(t, err)
}

func TestProofTampering(t *testing.T) {
	tr := createTestTree(t)
	key := types.KeyHash([]byte("test"))
	value := []byte("value")

	version, err := tr.Put(key, value)
	require.NoError(t, err)
	reader, err := tr.Reader(version)
	require.NoError(t, err)
	defer reader.Close()

	gen := NewGenerator(reader)
	proof, err := gen.Generate(key)
	require.NoError(t, err)
	require.Equal(t, ProofTypeInclusion, proof.Type)

	verifier := NewVerifier()

	// Test 1: Tamper with value
	originalValue := proof.Value
	proof.Value = []byte("tampered")
	err = verifier.Verify(proof)
	assert.ErrorIs(t, err, types.ErrHashMismatch)
	proof.Value = originalValue

	// Test 2: Tamper with root hash
	originalRoot := proof.RootHash
	proof.RootHash = types.Hash{0xFF}
	err = verifier.Verify(proof)
	assert.ErrorIs(t, err, types.ErrHashMismatch)
	proof.RootHash = originalRoot

	// Test 3: Tamper with sibling data if present
	if len(proof.Siblings) > 0 {
		t.Logf("Number of siblings: %d", len(proof.Siblings))
		t.Logf("First sibling depth: %d, children count: %d", proof.Siblings[0].Depth, len(proof.Siblings[0].Children))

		// Instead of clearing all children, let's tamper with an existing child hash
		originalChildren := proof.Siblings[0].Children
		if len(originalChildren) > 0 {
			// Find a child that's not on the proof path and modify its hash
			nibblePath := proof.Key.ToNibblePath()
			targetNibble := nibblePath.Nibbles[proof.Siblings[0].Depth]
			for nibble := range originalChildren {
				if nibble != targetNibble {
					// Tamper with this sibling's hash
					proof.Siblings[0].Children[nibble] = types.Hash{0xFF, 0xFF, 0xFF}
					err = verifier.Verify(proof)
					assert.Error(t, err, "tampering with sibling hash should fail verification")
					break
				}
			}
		}
		proof.Siblings[0].Children = originalChildren
	}

	// Original proof should still verify
	err = verifier.Verify(proof)
	assert.NoError(t, err)
}

func TestMaxDepthProof(t *testing.T) {
	tr := createTestTree(t)

	// Create keys that differ only in last nibble
	key1 := types.Key{}
	key2 := types.Key{}
	for i := 0; i < 31; i++ {
		key1[i] = 0xFF
		key2[i] = 0xFF
	}
	key1[31] = 0xF0 // Last nibbles: F,0
	key2[31] = 0xF1 // Last nibbles: F,1

	_, err := tr.Put(key1, []byte("v1"))
	require.NoError(t, err)
	version, err := tr.Put(key2, []byte("v2"))
	require.NoError(t, err)

	reader, err := tr.Reader(version)
	require.NoError(t, err)
	defer reader.Close()
	gen := NewGenerator(reader)

	// Generate and verify proof for key1
	proof1, err := gen.Generate(key1)
	require.NoError(t, err)
	assert.Equal(t, ProofTypeInclusion, proof1.Type)

	verifier := NewVerifier()
	err = verifier.Verify(proof1)
	assert.NoError(t, err)

	// Generate and verify proof for key2
	proof2, err := gen.Generate(key2)
	require.NoError(t, err)
	assert.Equal(t, ProofTypeInclusion, proof2.Type)

	err = verifier.Verify(proof2)
	assert.NoError(t, err)

	// Test non-existent key with same prefix
	key3 := types.Key{}
	for i := 0; i < 31; i++ {
		key3[i] = 0xFF
	}
	key3[31] = 0xF2 // Last nibbles: F,2

	proof3, err := gen.Generate(key3)
	require.NoError(t, err)
	// When keys differ only in last nibble, we get empty exclusion
	// because the internal node at depth 63 has children at F,0 and F,1
	// but not at F,2
	assert.Equal(t, ProofTypeExclusionEmpty, proof3.Type)

	err = verifier.Verify(proof3)
	assert.NoError(t, err)
}

func TestProofSerialization(t *testing.T) {
	tr := createTestTree(t)
	key := types.KeyHash([]byte("serialize-test"))
	value := []byte("serialize-value")

	version, err := tr.Put(key, value)
	require.NoError(t, err)
	reader, err := tr.Reader(version)
	require.NoError(t, err)
	defer reader.Close()

	gen := NewGenerator(reader)
	originalProof, err := gen.Generate(key)
	require.NoError(t, err)

	// Test codec serialization
	codec := NewProofCodec()

	// Encode
	encoded, err := codec.Encode(originalProof)
	require.NoError(t, err)
	assert.True(t, len(encoded) > 0)
	assert.True(t, len(encoded) <= MaxProofSize)

	// Decode
	decodedProof, err := codec.Decode(encoded)
	require.NoError(t, err)

	// Verify decoded proof matches original
	assert.Equal(t, originalProof.Type, decodedProof.Type)
	assert.Equal(t, originalProof.Key, decodedProof.Key)
	assert.Equal(t, originalProof.Value, decodedProof.Value)
	assert.Equal(t, originalProof.Version, decodedProof.Version)
	assert.Equal(t, originalProof.RootHash, decodedProof.RootHash)
	assert.Equal(t, len(originalProof.Siblings), len(decodedProof.Siblings))

	// Verify decoded proof still verifies
	verifier := NewVerifier()
	err = verifier.Verify(decodedProof)
	assert.NoError(t, err)
}

func TestProofCompression(t *testing.T) {
	tr := createTestTree(t)

	// Create a deeper tree to get more siblings
	for i := 0; i < 100; i++ {
		key := types.KeyHash([]byte(fmt.Sprintf("key-%d", i)))
		_, err := tr.Put(key, []byte(fmt.Sprintf("value-%d", i)))
		require.NoError(t, err)
	}

	key := types.KeyHash([]byte("compress-test"))
	value := []byte("compress-value")
	version, err := tr.Put(key, value)
	require.NoError(t, err)

	reader, err := tr.Reader(version)
	require.NoError(t, err)
	defer reader.Close()
	gen := NewGenerator(reader)
	proof, err := gen.Generate(key)
	require.NoError(t, err)

	// Compress
	compressed, err := CompressProof(proof)
	require.NoError(t, err)
	assert.True(t, len(compressed.CompressedData) > 0)

	// Decompress
	decompressed, err := DecompressProof(compressed)
	require.NoError(t, err)

	// Verify decompressed proof
	verifier := NewVerifier()
	err = verifier.Verify(decompressed)
	assert.NoError(t, err)
}

func TestBatchProofs(t *testing.T) {
	tr := createTestTree(t)

	// Insert multiple keys
	keys := make([]types.Key, 10)
	for i := 0; i < 10; i++ {
		keys[i] = types.KeyHash([]byte(fmt.Sprintf("batch-key-%d", i)))
		_, err := tr.Put(keys[i], []byte(fmt.Sprintf("value-%d", i)))
		require.NoError(t, err)
	}

	version := tr.GetLatestVersion()
	reader, err := tr.Reader(version)
	require.NoError(t, err)
	defer reader.Close()
	gen := NewGenerator(reader)

	// Generate batch proofs
	proofs, err := gen.GenerateBatch(keys)
	require.NoError(t, err)
	assert.Len(t, proofs, len(keys))

	// Verify all proofs
	verifier := NewVerifier()
	err = verifier.VerifyBatch(proofs)
	assert.NoError(t, err)
}

func TestInvalidProofValidation(t *testing.T) {
	// Test various invalid proof conditions

	// Invalid key
	proof := &Proof{
		Type: ProofTypeInclusion,
		Key:  types.Key{}, // Empty key
	}
	err := proof.IsValid()
	assert.Error(t, err)

	// Inclusion without value
	proof = &Proof{
		Type:  ProofTypeInclusion,
		Key:   types.KeyHash([]byte("test")),
		Value: nil,
	}
	err = proof.IsValid()
	assert.ErrorIs(t, err, types.ErrInvalidProof)

	// Neighbor exclusion without neighbor data
	proof = &Proof{
		Type:         ProofTypeExclusionNeighbor,
		Key:          types.KeyHash([]byte("test")),
		NeighborLeaf: nil,
	}
	err = proof.IsValid()
	assert.ErrorIs(t, err, types.ErrInvalidProof)

	// Invalid sibling depth
	proof = &Proof{
		Type: ProofTypeExclusionEmpty,
		Key:  types.KeyHash([]byte("test")),
		Siblings: []SiblingData{
			{Depth: -1}, // Invalid depth
		},
	}
	err = proof.IsValid()
	assert.ErrorIs(t, err, types.ErrInvalidProof)

	// Out of order siblings
	proof = &Proof{
		Type: ProofTypeExclusionEmpty,
		Key:  types.KeyHash([]byte("test")),
		Siblings: []SiblingData{
			{Depth: 5},
			{Depth: 3}, // Should be increasing
		},
	}
	err = proof.IsValid()
	assert.ErrorIs(t, err, types.ErrInvalidProof)
}

func BenchmarkProofGeneration(b *testing.B) {
	tr := createTestTree(b)

	// Build a small tree with 100 nodes
	for i := 0; i < 100; i++ {
		key := types.KeyHash([]byte(fmt.Sprintf("bench-key-%d", i)))
		_, err := tr.Put(key, []byte(fmt.Sprintf("value-%d", i)))
		require.NoError(b, err)
	}

	version := tr.GetLatestVersion()
	reader, err := tr.Reader(version)
	require.NoError(b, err)
	defer reader.Close()
	gen := NewGenerator(reader)

	keys := make([]types.Key, 100)
	for i := 0; i < 100; i++ {
		keys[i] = types.KeyHash([]byte(fmt.Sprintf("test-key-%d", i)))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := keys[i%len(keys)]
		_, err := gen.Generate(key)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkProofVerification(b *testing.B) {
	tr := createTestTree(b)

	// Build tree with 100 nodes
	for i := 0; i < 100; i++ {
		key := types.KeyHash([]byte(fmt.Sprintf("bench-key-%d", i)))
		_, err := tr.Put(key, []byte(fmt.Sprintf("value-%d", i)))
		require.NoError(b, err)
	}

	version := tr.GetLatestVersion()
	reader, err := tr.Reader(version)
	require.NoError(b, err)
	defer reader.Close()
	gen := NewGenerator(reader)
	verifier := NewVerifier()

	// Pre-generate proofs
	proofs := make([]*Proof, 100)
	for i := range proofs {
		key := types.KeyHash([]byte(fmt.Sprintf("bench-key-%d", i)))
		proof, err := gen.Generate(key)
		require.NoError(b, err)
		proofs[i] = proof
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		proof := proofs[i%len(proofs)]
		if err := verifier.Verify(proof); err != nil {
			b.Fatal(err)
		}
	}
}
