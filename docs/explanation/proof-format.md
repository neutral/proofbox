# Proof Format Explanation

This document explains the structure of ProofBox proof files and how they relate to the Jellyfish Merkle Tree (JMT) proof standard.

## Overview

ProofBox generates cryptographic proofs that can prove whether a key-value pair exists in the tree (inclusion proof) or doesn't exist (exclusion proof). These proofs are self-contained and can be verified independently without access to the full tree.

## Proof JSON Structure

A proof JSON file (like `version_proof.json`) contains a JMT proof with the following structure:

### 1. Core Fields

- **`type`**: The type of proof

  - `"inclusion"`: Proves a key exists with a specific value
  - `"exclusion"`: Proves a key doesn't exist in the tree

- **`key`**: The original key string (e.g., "app:version")
- **`key_hex`**: The 256-bit key in hexadecimal format (64 hex characters)
- **`value`**: The value associated with the key (only for inclusion proofs)
- **`value_hex`**: The value in hexadecimal encoding
- **`version`**: The tree version number this proof is for
- **`root_hash`**: The expected root hash of the tree (256-bit, 64 hex characters)

### 2. Encoded Field

- **`encoded`**: The complete proof in compact binary format (hex-encoded)
  - Contains all proof data in a space-efficient format
  - Used for efficient transmission and storage
  - Can be decoded to reconstruct the full proof

**Important Note**: The `encoded` field is the only field required for proof verification. All other fields in the JSON (including `siblings`, `key`, `value`, etc.) are redundant and provided solely for human readability and convenience. The verification process uses only the `encoded` field, which contains all necessary information including the complete siblings data.

### 3. Siblings Array (For Reference Only)

The `siblings` array contains the sibling nodes needed to reconstruct the path from leaf to root:

```json
{
  "children": {
    "1": "2ae5c00481f3366f14ad936e9f8ab754770134d295b75048294883006567c6c7",
    "9": "d99254000fc1a01d888e502266b5e2476fc405e674f31c9b2ab2a966da5909ce"
  },
  "depth": 0
}
```

- **`depth`**: Level in the tree (0 = root, increasing as you go down)
- **`children`**: Map of nibble (hex digit) to child hash at that level
  - The JMT is a radix-16 tree, so each node can have up to 16 children
  - Each nibble represents one hex digit (0-9, a-f)
  - Only non-empty children are included (sparse representation)

## How the Proof Works

### Path Construction

The JMT proof works by providing a verifiable path from the leaf to the root:

1. **Leaf Node**: Contains the key-value pair being proven
2. **Internal Nodes**: Each has up to 16 children (one per hex nibble)
3. **Root Node**: The final hash that represents the entire tree state

### Verification Process

1. **Hash the leaf**: Compute hash of the key-value pair
2. **Traverse upward**: At each level:
   - Use the appropriate nibble from the key to determine position
   - Combine current hash with sibling hashes to compute parent hash
3. **Reach the root**: Continue until reaching depth 0
4. **Verify**: Compare computed root hash with expected `root_hash`

### Sparse Tree Optimization

The JMT only stores non-empty nodes, making it extremely efficient for sparse key spaces. This is why the siblings array only contains the necessary hashes, not all possible children.

## Example: version_proof.json

The included `version_proof.json` example demonstrates:

```json
{
  "type": "inclusion",
  "key": "app:version",
  "value": "1.0.0",
  "key_hex": "90113649945cf51c5f7219c397571a6787a74f49e9a199a5d7fc27f08901571c",
  "value_hex": "312e302e30",
  "version": 3,
  "root_hash": "d2bd15a98e07ee5411123849e7316fa789a9b69fcb34800c8c3ea4a45ab72100",
  "siblings": [...],
  "encoded": "..."
}
```

This proof:

- Proves the key "app:version" exists with value "1.0.0"
- Is for version 3 of the tree
- Can be verified against the given root hash
- Contains 64 sibling entries (for a 256-bit key space)

## Usage

### Generating a Proof

```bash
# Generate a proof for a specific key
pb prove "account:123" --output account_proof.json

# Generate a proof with specific database
pb prove "user:alice" --db mydb.db --output alice_proof.json
```

### Verifying a Proof

```bash
# Verify a proof file
pb verify account_proof.json

# Verify the example proof
pb verify version_proof.json
```

### Minimal Proof Verification

Since only the `encoded` field is required, you can create a minimal proof file:

```json
{
  "encoded": "0090113649945cf51c5f7219c397571a6787a74f49e9a199a5d7fc27f08901571c..."
}
```

This minimal file will verify successfully. The verify command extracts only the `encoded` field and ignores all other fields.

## Binary Encoding Format

The `encoded` field contains the proof in a compact binary format:

1. **Header**: Proof type (1 byte) + Key (32 bytes) + Version (8 bytes) + Root hash (32 bytes)
2. **Value data**: For inclusion proofs - length prefix + value bytes
3. **Siblings data**: Count + array of (depth + nibble + children count + children data)

This format is designed to minimize size while maintaining all necessary information for verification.

## Security Properties

The proof format provides several security guarantees:

1. **Tamper-proof**: Any modification to the proof invalidates it
2. **Complete**: Contains all data needed for independent verification
3. **Deterministic**: Same key/value always produces the same proof
4. **Compact**: Proof size is O(log n) where n is the number of keys
5. **Privacy-preserving**: Doesn't reveal information about other keys in the tree

## Size Considerations

- Maximum proof size: 10KB (enforced limit)
- Typical proof size: 1-3KB for normal trees
- Size grows logarithmically with tree size
- Binary encoding is ~50% more compact than JSON
