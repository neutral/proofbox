# Proof Package Public API

## Exported Types/Structs

```go
type ProofType uint8

type Proof struct {
	Type         ProofType
	Key          types.Key
	Value        []byte
	NeighborLeaf *NeighborLeafData
	Siblings     []SiblingData
	RootHash     types.Hash
	Version      types.Version
}

type SiblingData struct {
	Depth    int
	Nibble   types.Nibble
	Hash     types.Hash
	Children map[types.Nibble]types.Hash
}

type NeighborLeafData struct {
	Key          types.Key
	ValueHash    types.Hash
	Path         []types.Nibble
	DivergeDepth int
}

type ProofBatch struct {
	Proofs []*Proof
}

type Generator struct {
	reader tree.TreeReaderInterface
}

type ValueLoader interface {
	LoadValue(hash types.Hash) ([]byte, error)
}

type Verifier struct {
	hasher crypto.Hasher
}

type ProofCodec struct{}

type CompressedProof struct {
	Type           ProofType
	KeyHash        types.Hash
	CompressedData []byte
}

type BatchProofOptimization struct {
	proofs []*Proof
}
```

## Exported Functions

```go
func NewGenerator(reader tree.TreeReaderInterface) *Generator
func Generate(reader tree.TreeReaderInterface, key types.Key) (*Proof, error)
func NewVerifier() *Verifier
func Verify(proof *Proof) error
func NewProofCodec() *ProofCodec
func CompressProof(proof *Proof) (*CompressedProof, error)
func DecompressProof(compressed *CompressedProof) (*Proof, error)
func OptimizeBatch(proofs []*Proof) *BatchProofOptimization
```

## Exported Methods

### Proof methods
```go
func (p *Proof) IsValid() error
func (p *Proof) EstimateSize() int
```

### Generator methods
```go
func (g *Generator) Generate(key types.Key) (*Proof, error)
func (g *Generator) GenerateBatch(keys []types.Key) ([]*Proof, error)
```

### Verifier methods
```go
func (v *Verifier) Verify(proof *Proof) error
func (v *Verifier) VerifyBatch(proofs []*Proof) error
```

### ProofCodec methods
```go
func (c *ProofCodec) Encode(proof *Proof) ([]byte, error)
func (c *ProofCodec) Decode(data []byte) (*Proof, error)
```

## Exported Constants

```go
const ProofTypeInclusion ProofType = iota
const ProofTypeExclusionEmpty
const ProofTypeExclusionNeighbor
const MaxProofSize = 10 * 1024
```