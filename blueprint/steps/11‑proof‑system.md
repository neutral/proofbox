---
id: step.11.proof-system
depends_on:
  - step.10.update-existing
tags: [proof, verification, step]
---

## Objective

Implement comprehensive Merkle proof generation and verification system supporting inclusion proofs, exclusion proofs (empty and neighbor variants), with optimizations for proof size and verification performance.

## Implements

- **Proof Generation Specification** (`/blueprint/features/proof/generate/specs/proof-generation.md`)
- **Proof Verification Specification** (`/blueprint/features/proof/verify/specs/proof-verification.md`)
- **§5 Merkle Proofs** - Inclusion and exclusion proof algorithms

## Technical Details

### Proof Types

Create `pkg/proof/types.go`:
```go
package proof

import (
    "github.com/acme/jmt/pkg/types"
)

// ProofType identifies the type of proof
type ProofType uint8

const (
    ProofTypeInclusion ProofType = iota
    ProofTypeExclusionEmpty
    ProofTypeExclusionNeighbor
)

// Proof represents a Merkle proof for a key
type Proof struct {
    Type         ProofType
    Key          types.Key
    Value        []byte              // For inclusion proofs
    NeighborLeaf *NeighborLeafData  // For neighbor exclusion proofs
    Siblings     []SiblingData      // Sibling hashes along the path
    RootHash     types.Hash         // Expected root hash
    Version      types.Version      // Tree version
}

// SiblingData contains sibling information at each level
type SiblingData struct {
    Nibble types.Nibble // Which nibble this is for
    Hash   types.Hash   // Sibling hash
}

// NeighborLeafData contains neighbor leaf information
type NeighborLeafData struct {
    Key       types.Key
    ValueHash types.Hash
    Path      []types.Nibble // Common path to neighbor
}

// IsValid performs basic validation
func (p *Proof) IsValid() error {
    if err := types.ValidateKey(p.Key); err != nil {
        return types.WrapError(err, "invalid proof key")
    }
    
    switch p.Type {
    case ProofTypeInclusion:
        if p.Value == nil {
            return types.ErrInvalidProof
        }
    case ProofTypeExclusionNeighbor:
        if p.NeighborLeaf == nil {
            return types.ErrInvalidProof
        }
    }
    
    return nil
}
```

### Proof Generation

Create `pkg/proof/generator.go`:
```go
package proof

import (
    "bytes"
    
    "github.com/acme/jmt/pkg/crypto"
    "github.com/acme/jmt/pkg/tree"
    "github.com/acme/jmt/pkg/types"
)

// Generator creates Merkle proofs
type Generator struct {
    reader tree.TreeReader
}

// NewGenerator creates a new proof generator
func NewGenerator(reader tree.TreeReader) *Generator {
    return &Generator{reader: reader}
}

// Generate creates a proof for the given key
func (g *Generator) Generate(key types.Key) (*Proof, error) {
    rootHash := g.reader.RootHash()
    version := g.reader.Version()
    
    proof := &Proof{
        Key:      key,
        RootHash: rootHash,
        Version:  version,
        Siblings: make([]SiblingData, 0),
    }
    
    // Handle empty tree
    if rootHash == crypto.EmptyTreeHash {
        proof.Type = ProofTypeExclusionEmpty
        return proof, nil
    }
    
    // Traverse tree collecting siblings
    nibblePath := key.ToNibblePath()
    nodeKey := types.RootNodeKey(version)
    
    for depth := 0; depth < types.MaxTreeDepth; depth++ {
        node, err := g.reader.GetNode(nodeKey)
        if err != nil {
            return nil, types.WrapError(err, "failed to load node at depth %d", depth)
        }
        
        switch n := node.(type) {
        case *tree.LeafNode:
            if n.Key() == key {
                // Inclusion proof
                proof.Type = ProofTypeInclusion
                proof.Value = n.Value()
                return proof, nil
            } else {
                // Neighbor exclusion proof
                proof.Type = ProofTypeExclusionNeighbor
                proof.NeighborLeaf = &NeighborLeafData{
                    Key:       n.Key(),
                    ValueHash: n.ValueHash(),
                    Path:      nibblePath.Nibbles[:depth],
                }
                return proof, nil
            }
            
        case *tree.InternalNode:
            nibble := nibblePath.Nibbles[depth]
            
            // Collect siblings
            siblings := g.collectSiblings(n, nibble)
            proof.Siblings = append(proof.Siblings, siblings...)
            
            // Check if child exists
            child, exists := n.Child(nibble)
            if !exists {
                // Empty exclusion proof
                proof.Type = ProofTypeExclusionEmpty
                return proof, nil
            }
            
            // Continue to child
            nodeKey = nodeKey.Child(nibble, child.Version)
            
        default:
            return nil, types.ErrInvalidNodeType
        }
    }
    
    return nil, types.WrapError(types.ErrMaxDepthExceeded, "proof generation exceeded max depth")
}

// collectSiblings gathers sibling hashes for proof
func (g *Generator) collectSiblings(node *tree.InternalNode, targetNibble types.Nibble) []SiblingData {
    siblings := make([]SiblingData, 0, 15)
    
    for nibble := types.Nibble(0); nibble <= types.MaxNibbleValue; nibble++ {
        if nibble == targetNibble {
            continue // Skip the path we're following
        }
        
        if child, exists := node.Child(nibble); exists {
            siblings = append(siblings, SiblingData{
                Nibble: nibble,
                Hash:   child.Hash,
            })
        }
    }
    
    return siblings
}

// GenerateBatch generates proofs for multiple keys efficiently
func (g *Generator) GenerateBatch(keys []types.Key) ([]*Proof, error) {
    proofs := make([]*Proof, len(keys))
    
    // TODO: Optimize by sharing common path traversals
    for i, key := range keys {
        proof, err := g.Generate(key)
        if err != nil {
            return nil, types.WrapError(err, "failed to generate proof for key %d", i)
        }
        proofs[i] = proof
    }
    
    return proofs, nil
}
```

### Proof Verification

Create `pkg/proof/verifier.go`:
```go
package proof

import (
    "bytes"
    
    "github.com/acme/jmt/pkg/crypto"
    "github.com/acme/jmt/pkg/tree"
    "github.com/acme/jmt/pkg/types"
)

// Verifier verifies Merkle proofs
type Verifier struct {
    hasher crypto.Hasher
}

// NewVerifier creates a new proof verifier
func NewVerifier() *Verifier {
    return &Verifier{
        hasher: crypto.DefaultHasher,
    }
}

// Verify checks if a proof is valid
func (v *Verifier) Verify(proof *Proof) error {
    if err := proof.IsValid(); err != nil {
        return err
    }
    
    // Compute root hash based on proof type
    var computedRoot types.Hash
    var err error
    
    switch proof.Type {
    case ProofTypeInclusion:
        computedRoot, err = v.verifyInclusion(proof)
    case ProofTypeExclusionEmpty:
        computedRoot, err = v.verifyExclusionEmpty(proof)
    case ProofTypeExclusionNeighbor:
        computedRoot, err = v.verifyExclusionNeighbor(proof)
    default:
        return types.ErrInvalidProof
    }
    
    if err != nil {
        return err
    }
    
    // Compare with expected root
    if !bytes.Equal(computedRoot[:], proof.RootHash[:]) {
        return types.ErrHashMismatch
    }
    
    return nil
}

// verifyInclusion verifies an inclusion proof
func (v *Verifier) verifyInclusion(proof *Proof) (types.Hash, error) {
    // Start with leaf hash
    valueHash := v.hasher.Hash(proof.Value)
    leafHash := v.hasher.HashConcat(
        []byte{byte(tree.NodeTypeLeaf)},
        proof.Key[:],
        valueHash[:],
    )
    
    // Traverse up the tree
    return v.computeRootHash(proof.Key, leafHash, proof.Siblings, true)
}

// verifyExclusionEmpty verifies an empty exclusion proof
func (v *Verifier) verifyExclusionEmpty(proof *Proof) (types.Hash, error) {
    // Start with empty hash
    emptyHash := crypto.EmptyTreeHash
    
    // Traverse up the tree
    return v.computeRootHash(proof.Key, emptyHash, proof.Siblings, false)
}

// verifyExclusionNeighbor verifies a neighbor exclusion proof
func (v *Verifier) verifyExclusionNeighbor(proof *Proof) (types.Hash, error) {
    neighbor := proof.NeighborLeaf
    
    // Verify the neighbor leaf is at the claimed position
    nibblePath := proof.Key.ToNibblePath()
    neighborPath := neighbor.Key.ToNibblePath()
    
    // Check common prefix
    for i, nibble := range neighbor.Path {
        if nibblePath.Nibbles[i] != neighborPath.Nibbles[i] {
            return types.Hash{}, types.ErrInvalidProof
        }
    }
    
    // The next nibble must differ
    divergeDepth := len(neighbor.Path)
    if divergeDepth >= types.MaxTreeDepth ||
       nibblePath.Nibbles[divergeDepth] == neighborPath.Nibbles[divergeDepth] {
        return types.Hash{}, types.ErrInvalidProof
    }
    
    // Compute neighbor leaf hash
    leafHash := v.hasher.HashConcat(
        []byte{byte(tree.NodeTypeLeaf)},
        neighbor.Key[:],
        neighbor.ValueHash[:],
    )
    
    // Traverse up from the divergence point
    return v.computeRootHashFromNeighbor(
        proof.Key,
        neighbor.Key,
        leafHash,
        proof.Siblings,
        divergeDepth,
    )
}

// computeRootHash reconstructs the root hash from a leaf
func (v *Verifier) computeRootHash(
    key types.Key,
    currentHash types.Hash,
    siblings []SiblingData,
    isLeaf bool,
) (types.Hash, error) {
    nibblePath := key.ToNibblePath()
    siblingIndex := 0
    
    // Process each level up to root
    for depth := types.MaxTreeDepth - 1; depth >= 0; depth-- {
        if isLeaf && depth == types.MaxTreeDepth-1 {
            // Skip the leaf level
            isLeaf = false
            continue
        }
        
        // Build internal node hash
        parts := [][]byte{{byte(tree.NodeTypeInternal)}}
        targetNibble := nibblePath.Nibbles[depth]
        
        // Process all 16 possible children
        for nibble := types.Nibble(0); nibble <= types.MaxNibbleValue; nibble++ {
            if nibble == targetNibble {
                // This is our path
                parts = append(parts, []byte{byte(nibble)})
                parts = append(parts, currentHash[:])
            } else {
                // Check if we have a sibling for this nibble
                found := false
                for siblingIndex < len(siblings) {
                    sib := siblings[siblingIndex]
                    if sib.Nibble == nibble {
                        parts = append(parts, []byte{byte(nibble)})
                        parts = append(parts, sib.Hash[:])
                        siblingIndex++
                        found = true
                        break
                    } else if sib.Nibble > nibble {
                        break
                    }
                    siblingIndex++
                }
                
                if !found {
                    // Empty child
                    parts = append(parts, crypto.EmptyTreeHash[:])
                }
            }
        }
        
        currentHash = v.hasher.HashConcat(parts...)
    }
    
    return currentHash, nil
}

// computeRootHashFromNeighbor handles neighbor exclusion proof verification
func (v *Verifier) computeRootHashFromNeighbor(
    key types.Key,
    neighborKey types.Key,
    neighborHash types.Hash,
    siblings []SiblingData,
    divergeDepth int,
) (types.Hash, error) {
    // Similar to computeRootHash but handles the neighbor case
    // Implementation details omitted for brevity
    // This would properly place the neighbor leaf and compute up
    
    return types.Hash{}, nil // TODO: Implement
}

// VerifyBatch verifies multiple proofs efficiently
func (v *Verifier) VerifyBatch(proofs []*Proof) error {
    // TODO: Optimize by sharing computation for common paths
    for i, proof := range proofs {
        if err := v.Verify(proof); err != nil {
            return types.WrapError(err, "proof %d verification failed", i)
        }
    }
    return nil
}
```

### Proof Optimization

Create `pkg/proof/optimization.go`:
```go
package proof

import (
    "bytes"
    "compress/gzip"
    "encoding/binary"
    
    "github.com/acme/jmt/pkg/types"
)

// CompressedProof represents a space-optimized proof
type CompressedProof struct {
    Type         ProofType
    KeyHash      types.Hash // Store hash instead of full key
    CompressedData []byte   // Gzip compressed proof data
}

// CompressProof reduces proof size
func CompressProof(proof *Proof) (*CompressedProof, error) {
    // Serialize proof data
    var buf bytes.Buffer
    
    // Write proof data based on type
    switch proof.Type {
    case ProofTypeInclusion:
        binary.Write(&buf, binary.BigEndian, uint32(len(proof.Value)))
        buf.Write(proof.Value)
    case ProofTypeExclusionNeighbor:
        buf.Write(proof.NeighborLeaf.Key[:])
        buf.Write(proof.NeighborLeaf.ValueHash[:])
        binary.Write(&buf, binary.BigEndian, uint16(len(proof.NeighborLeaf.Path)))
        for _, nibble := range proof.NeighborLeaf.Path {
            buf.WriteByte(byte(nibble))
        }
    }
    
    // Write siblings
    binary.Write(&buf, binary.BigEndian, uint16(len(proof.Siblings)))
    for _, sib := range proof.Siblings {
        buf.WriteByte(byte(sib.Nibble))
        buf.Write(sib.Hash[:])
    }
    
    // Compress
    var compressed bytes.Buffer
    gz := gzip.NewWriter(&compressed)
    if _, err := gz.Write(buf.Bytes()); err != nil {
        return nil, err
    }
    gz.Close()
    
    return &CompressedProof{
        Type:           proof.Type,
        KeyHash:        crypto.DefaultHasher.Hash(proof.Key[:]),
        CompressedData: compressed.Bytes(),
    }, nil
}

// BatchProofOptimization optimizes proofs that share common paths
type BatchProofOptimization struct {
    proofs []*Proof
}

// OptimizeBatch reduces redundancy in batch proofs
func OptimizeBatch(proofs []*Proof) *BatchProofOptimization {
    // TODO: Implement path sharing optimization
    // - Identify common prefixes
    // - Share sibling data
    // - Compress redundant information
    
    return &BatchProofOptimization{proofs: proofs}
}
```

## Testing Requirements

### Inclusion Proof Tests
```go
func TestInclusionProof(t *testing.T) {
    tree := setupTestTree(t)
    key := types.KeyHash([]byte("test-key"))
    value := []byte("test-value")
    
    // Insert value
    tree.Put(key, value)
    
    // Generate proof
    gen := NewGenerator(tree.Reader())
    proof, err := gen.Generate(key)
    assert.NoError(t, err)
    assert.Equal(t, ProofTypeInclusion, proof.Type)
    assert.Equal(t, value, proof.Value)
    
    // Verify proof
    verifier := NewVerifier()
    err = verifier.Verify(proof)
    assert.NoError(t, err)
}

func TestExclusionEmptyProof(t *testing.T) {
    tree := setupTestTree(t)
    key := types.KeyHash([]byte("non-existent"))
    
    // Generate proof for non-existent key
    gen := NewGenerator(tree.Reader())
    proof, err := gen.Generate(key)
    assert.NoError(t, err)
    assert.Equal(t, ProofTypeExclusionEmpty, proof.Type)
    
    // Verify proof
    verifier := NewVerifier()
    err = verifier.Verify(proof)
    assert.NoError(t, err)
}

func TestExclusionNeighborProof(t *testing.T) {
    tree := setupTestTree(t)
    
    // Insert some keys to create internal structure
    tree.Put(types.KeyHash([]byte("key1")), []byte("value1"))
    tree.Put(types.KeyHash([]byte("key3")), []byte("value3"))
    
    // Generate proof for key2 (should find key1 or key3 as neighbor)
    key := types.KeyHash([]byte("key2"))
    gen := NewGenerator(tree.Reader())
    proof, err := gen.Generate(key)
    assert.NoError(t, err)
    assert.Equal(t, ProofTypeExclusionNeighbor, proof.Type)
    assert.NotNil(t, proof.NeighborLeaf)
    
    // Verify proof
    verifier := NewVerifier()
    err = verifier.Verify(proof)
    assert.NoError(t, err)
}

func TestProofTampering(t *testing.T) {
    tree := setupTestTree(t)
    key := types.KeyHash([]byte("test"))
    tree.Put(key, []byte("value"))
    
    gen := NewGenerator(tree.Reader())
    proof, err := gen.Generate(key)
    assert.NoError(t, err)
    
    // Tamper with proof
    proof.Value = []byte("tampered")
    
    // Verification should fail
    verifier := NewVerifier()
    err = verifier.Verify(proof)
    assert.ErrorIs(t, err, types.ErrHashMismatch)
}
```

### Performance Tests
```go
func BenchmarkProofGeneration(b *testing.B) {
    tree := setupLargeTree(b, 10000)
    gen := NewGenerator(tree.Reader())
    keys := generateRandomKeys(100)
    
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
    tree := setupLargeTree(b, 10000)
    gen := NewGenerator(tree.Reader())
    verifier := NewVerifier()
    
    // Pre-generate proofs
    proofs := make([]*Proof, 100)
    for i := range proofs {
        key := types.KeyHash([]byte(fmt.Sprintf("key-%d", i)))
        proof, _ := gen.Generate(key)
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
```

## Implementation Steps

1. Create proof types in `pkg/proof/types.go`
2. Implement proof generator in `pkg/proof/generator.go`
3. Implement proof verifier in `pkg/proof/verifier.go`
4. Add proof optimization utilities
5. Write comprehensive tests for all proof types
6. Add benchmarks for performance validation
7. Implement batch proof optimizations
8. Add proof serialization for network transmission

## Performance Considerations

- Cache frequently accessed nodes during proof generation
- Optimize sibling collection to minimize allocations
- Use parallel verification for batch proofs
- Compress proofs for network transmission
- Target < 300μs for proof verification

## Security Notes

- Validate all proof inputs to prevent panics
- Ensure proof verification is deterministic
- Guard against maliciously crafted proofs
- Use constant-time operations where applicable
- Implement proof size limits to prevent DoS

## Done When ✓

- [ ] All three proof types implemented (inclusion, empty, neighbor)
- [ ] Proof generator with sibling collection
- [ ] Proof verifier with hash reconstruction
- [ ] Batch proof generation and verification
- [ ] Proof compression utilities
- [ ] Comprehensive tests for all proof types
- [ ] Tamper resistance tests
- [ ] Performance benchmarks meeting targets
- [ ] Security considerations addressed
- [ ] Documentation for proof format