# ProofBox

ProofBox is a Go implementation of the Jellyfish Merkle Tree (JMT), a space-efficient sparse Merkle tree with support for versioned key-value storage and cryptographic proofs.

## Features

- **Versioned Storage**: Track multiple versions of key-value pairs
- **Cryptographic Proofs**: Generate and verify inclusion/exclusion proofs
- **Space Efficient**: Optimized storage using sparse tree techniques
- **High Performance**: Designed for low-latency operations
- **Production Ready**: Comprehensive testing and benchmarking

## Quick Start

### Using the CLI

ProofBox includes a powerful command-line interface (`pb`) for interacting with the Merkle tree:

```bash
# Install the CLI
go install github.com/neutral/proofbox/cmd/pb@latest

# Create a database and store data
pb init --db mydata.db
pb put "hello" "world" --db mydata.db
pb get "hello" --db mydata.db

# Generate and verify proofs
pb prove "hello" --db mydata.db --output proof.json
pb verify proof.json
```

See the [CLI documentation](cmd/pb/README.md) for detailed usage instructions, or check out the [Quick Start Guide](cmd/pb/QUICKSTART.md) and [Cheat Sheet](cmd/pb/CHEATSHEET.md).

### Building from Source

```bash
# Build the project
make

# Run tests
make test

# Run linting
make lint
```

## Interactive Examples

Explore ProofBox features through interactive examples:

```bash
# Run all examples
./examples/run_all.sh

# Run a specific example
cd examples && go run example_batch_basic.go examples_utils.go
```

See the [examples directory](examples/) for:
- Batch operations and transactions
- Performance optimizations
- Concurrent operations
- And more...

## Project Structure

```
├── cmd/
│   └── pb/          # CLI application
├── pkg/             # Public packages
│   ├── crypto/      # Cryptographic functions
│   ├── proof/       # Proof generation/verification
│   ├── storage/     # Storage abstraction
│   ├── tree/        # Core tree implementation
│   └── types/       # Common types
├── internal/        # Internal packages
├── examples/        # Interactive examples
└── blueprint/       # Design specifications
```

## Quick tips for developers

- **Stay in sync:** every meaningful change to requirements or design should update the relevant file in `blueprint/` before updating the codebase.
- **Codebase documentation:** `blueprint/`is not a substitute for codebase documentation. Every source file needs to be accompanied by a description file clearly explaining its logic and relation to the information present in the `blueprint/` folder.

## License

The contents of this repository are released under a dual-license model, allowing you to choose **either** the **Creative Commons Zero v1.0 Universal** **or** the **Zero-Clause BSD (0BSD)** license. This applies to all materials in this project, including but not limited to source code, documentation, images, and data files.

For the full text of each license, please see the following files in this repository:

- [CC0-1.0](LICENSE.CC0-1.0)
- [0BSD](LICENSE.0BSD)

All applicable files in this repository should include an `SPDX-License-Identifier` header indicating the `(CC0-1.0 OR 0BSD)` dual-license choice.
