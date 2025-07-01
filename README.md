# ProofBox

[![Go Version](https://img.shields.io/badge/go-%3E%3D1.21-blue)](https://go.dev/)
[![License: CC0-1.0](https://img.shields.io/badge/License-CC0_1.0-lightgrey.svg)](LICENSE.CC0-1.0)
[![License: 0BSD](https://img.shields.io/badge/License-0BSD-brightgreen.svg)](LICENSE.0BSD)
[![Go Reference](https://pkg.go.dev/badge/github.com/neutral/proofbox.svg)](https://pkg.go.dev/github.com/neutral/proofbox)

ProofBox is a high-performance Go implementation of the [Jellyfish Merkle Tree (JMT)](docs/explanation/jellyfish-merkle-tree.md), a space-efficient sparse Merkle tree optimized for verifiable storage systems.

## 🚀 Features

- **🔄 Versioned Storage**: Maintain full history with efficient version management
- **🔐 Cryptographic Proofs**: Generate and verify compact inclusion/exclusion proofs
- **💾 Space Efficient**: Sparse tree design minimizes storage overhead
- **⚡ High Performance**: Optimized for low-latency operations with batch support
- **🛠️ Developer Friendly**: Clean API, comprehensive docs, and interactive REPL
- **📊 Production Ready**: Extensive testing, benchmarks, and metrics support

## 📦 Installation

### Install the CLI

```bash
go install github.com/neutral/proofbox/cmd/pb@latest
```

### Use as a Library

```bash
go get github.com/neutral/proofbox
```

## 🚀 Quick Start

### CLI Usage

```bash
# Create a new database
pb init --db mydata.db

# Store and retrieve data
pb put "hello" "world" --db mydata.db
pb get "hello" --db mydata.db

# Generate cryptographic proofs
pb prove "hello" --db mydata.db --output proof.json
pb verify proof.json

# Interactive REPL mode
pb repl --db mydata.db
```

📖 See [CLI Quick Start](cmd/pb/QUICKSTART.md) • [CLI Cheat Sheet](cmd/pb/CHEATSHEET.md) • [Full Documentation](docs/index.md)

### Library Usage

```go
import (
    "github.com/neutral/proofbox/pkg/tree"
    "github.com/neutral/proofbox/pkg/storage"
)

// Create storage and tree
store, _ := storage.NewPebbleDB("/tmp/proofbox")
jmt, _ := tree.New(store)

// Store data
batch := tree.NewUpdateBatch()
batch.Put([]byte("key"), []byte("value"))
rootHash, version, _ := jmt.CommitBatch(batch)

// Generate proof
proof, _ := jmt.GetProof([]byte("key"), version)
```

📖 See [Library Tutorial](docs/tutorials/library-tutorial.md) • [API Reference](docs/reference/api/)

## 🏗️ Building from Source

```bash
# Clone the repository
git clone https://github.com/neutral/proofbox.git
cd proofbox

# Build everything
make

# Run tests
make test

# Run benchmarks
make bench-quick
```

## 🎯 Examples

Explore ProofBox features through interactive examples:

```bash
# Run all examples
cd docs/examples && ./run_all.sh

# Run a specific example
cd docs/examples/batch_basic && go run .
```

Available examples:
- **Batch Operations**: Transactions, deduplication, validation
- **Performance**: Parallel processing, optimization techniques
- **Concurrency**: Thread-safe operations
- **Internals**: Understanding the UpdateBatch structure

📖 See [Examples Directory](docs/examples/) for full list

## 📚 Documentation

ProofBox has comprehensive documentation organized by use case:

- **[Tutorials](docs/tutorials/)** - Step-by-step guides to get started
- **[How-to Guides](docs/how-to/)** - Practical guides for specific tasks
- **[API Reference](docs/reference/api/)** - Complete API documentation
- **[Explanation](docs/explanation/)** - Understanding the concepts and design

## 🏛️ Architecture

```
proofbox/
├── cmd/pb/          # CLI application
├── pkg/             # Public API packages
│   ├── tree/        # Core JMT implementation
│   ├── proof/       # Proof generation/verification
│   ├── storage/     # Storage abstraction layer
│   ├── crypto/      # Cryptographic utilities
│   └── types/       # Common types and interfaces
├── internal/        # Internal implementation
├── docs/            # Documentation and examples
└── blueprint/       # Design specifications
```

Key design features:
- **Radix-16 tree structure** for optimal proof size
- **PebbleDB storage backend** for efficient disk usage
- **Append-only versioning** for historical queries
- **Batch operations** for improved throughput

📖 See [Architecture Overview](docs/explanation/architecture.md) for details

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

For developers:
- Follow the blueprint-driven development process
- Check `blueprint/` for requirements before implementing
- Add `.desc.md` files for non-trivial code
- Ensure all tests pass before submitting PRs

## 📄 License

ProofBox is released under a dual-license model. Choose either:
- **[CC0-1.0](LICENSE.CC0-1.0)** - Creative Commons Zero v1.0 Universal
- **[0BSD](LICENSE.0BSD)** - Zero-Clause BSD

This applies to all project materials. Files include `SPDX-License-Identifier: (CC0-1.0 OR 0BSD)`.

---

<div align="center">
  
**[Get Started](docs/tutorials/getting-started.md)** • **[Documentation](docs/)** • **[Examples](docs/examples/)** • **[API Reference](docs/reference/api/)**

</div>
