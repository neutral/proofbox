package commands

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	"github.com/neutral/proofbox/pkg/proof"
	"github.com/neutral/proofbox/pkg/types"
	"github.com/spf13/cobra"
)

var (
	outputFile string
)

// proofTypeToString converts proof type to string
func proofTypeToString(pt proof.ProofType) string {
	switch pt {
	case proof.ProofTypeInclusion:
		return "inclusion"
	case proof.ProofTypeExclusionEmpty:
		return "exclusion_empty"
	case proof.ProofTypeExclusionNeighbor:
		return "exclusion_neighbor"
	default:
		return "unknown"
	}
}

// proveCmd represents the prove command
var proveCmd = &cobra.Command{
	Use:   "prove <key>",
	Short: "Generate a Merkle proof for a key",
	Long: `Generate a Merkle proof that can be used to verify the existence
or non-existence of a key in the tree at a specific version.`,
	Args: cobra.ExactArgs(1),
	RunE: runProve,
}

func init() {
	rootCmd.AddCommand(proveCmd)
	proveCmd.Flags().BoolVar(&hexKey, "hex-key", false, "interpret key as hex-encoded bytes")
	proveCmd.Flags().Uint64Var(&version, "version", 0, "version to generate proof for (default: latest)")
	proveCmd.Flags().StringVar(&outputFile, "output", "", "write proof to file (default: stdout)")
}

func runProve(cmd *cobra.Command, args []string) error {
	// Open database
	store, keyEncoder, err := openDatabase(dbPath)
	if err != nil {
		return err
	}
	defer store.Close()

	// Parse key
	keyStr := args[0]
	var key types.Key
	if hexKey {
		keyBytes, err := hex.DecodeString(keyStr)
		if err != nil {
			return fmt.Errorf("invalid hex key: %w", err)
		}
		if len(keyBytes) != 32 {
			return fmt.Errorf("key must be 32 bytes, got %d", len(keyBytes))
		}
		copy(key[:], keyBytes)
	} else {
		key = types.KeyHash([]byte(keyStr))
	}

	// Create tree
	tree, err := openTree(store, keyEncoder)
	if err != nil {
		return err
	}

	// Use latest version if not specified
	if !cmd.Flags().Changed("version") {
		version = uint64(tree.GetLatestVersion())
	}

	// Create reader for the specified version
	reader, err := tree.Reader(types.Version(version))
	if err != nil {
		return fmt.Errorf("failed to create reader: %w", err)
	}
	defer reader.Close()

	// Generate proof
	generator := proof.NewGenerator(reader)
	merkleProof, err := generator.Generate(key)
	if err != nil {
		return fmt.Errorf("failed to generate proof: %w", err)
	}

	// Prepare proof data
	proofData := map[string]interface{}{
		"key":       keyStr,
		"key_hex":   hex.EncodeToString(key[:]),
		"version":   merkleProof.Version,
		"root_hash": hex.EncodeToString(merkleProof.RootHash[:]),
		"type":      proofTypeToString(merkleProof.Type),
	}

	// Add type-specific data
	switch merkleProof.Type {
	case proof.ProofTypeInclusion:
		proofData["value"] = string(merkleProof.Value)
		proofData["value_hex"] = hex.EncodeToString(merkleProof.Value)
	case proof.ProofTypeExclusionNeighbor:
		if merkleProof.NeighborLeaf != nil {
			proofData["neighbor_key"] = hex.EncodeToString(merkleProof.NeighborLeaf.Key[:])
			proofData["neighbor_value_hash"] = hex.EncodeToString(merkleProof.NeighborLeaf.ValueHash[:])
		}
	}

	// Add siblings
	siblings := make([]map[string]interface{}, len(merkleProof.Siblings))
	for i, sibling := range merkleProof.Siblings {
		siblingData := map[string]interface{}{
			"depth": sibling.Depth,
		}
		children := make(map[string]string)
		for nibble, hash := range sibling.Children {
			children[fmt.Sprintf("%x", nibble)] = hex.EncodeToString(hash[:])
		}
		siblingData["children"] = children
		siblings[i] = siblingData
	}
	proofData["siblings"] = siblings

	// Encode proof using the codec
	codec := proof.NewProofCodec()
	encodedProof, err := codec.Encode(merkleProof)
	if err != nil {
		return fmt.Errorf("failed to encode proof: %w", err)
	}
	proofData["encoded"] = hex.EncodeToString(encodedProof)

	// Output proof
	if outputFile != "" {
		// Validate output path
		if err := validatePath(outputFile); err != nil {
			return fmt.Errorf("invalid proof output path: %w", err)
		}

		file, err := os.Create(outputFile)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer file.Close()

		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(proofData); err != nil {
			return fmt.Errorf("failed to write proof: %w", err)
		}

		result := map[string]string{
			"status":  "success",
			"message": fmt.Sprintf("Proof written to %s", outputFile),
		}
		return outputResult(result)
	}

	return outputResult(proofData)
}
