# Getting Started with ProofBox

Welcome to ProofBox! This tutorial will introduce you to the core concepts and help you understand what ProofBox is, why it exists, and how to get started using it.

## What is ProofBox?

ProofBox is a high-performance, versioned key-value store that provides cryptographic proofs of data integrity. It's built on the Jellyfish Merkle Tree (JMT) data structure, originally developed for blockchain systems but useful for any application requiring:

- **Versioned data storage** - Keep history of all changes
- **Cryptographic proofs** - Prove data exists (or doesn't) without revealing entire database
- **Data integrity** - Detect any tampering or corruption
- **Efficient storage** - Optimized for SSD storage patterns

## When to Use ProofBox

ProofBox is ideal for:

- **Audit systems** - Maintain tamper-proof logs with verification
- **Configuration management** - Track all configuration changes with proofs
- **Blockchain applications** - State storage with Merkle proofs
- **Compliance systems** - Prove data state at any point in time
- **Version control** - For structured data with cryptographic guarantees

## Core Concepts

Before we dive in, let's understand the key concepts:

### 1. Key-Value Store
At its core, ProofBox stores data as key-value pairs:
- **Key**: A unique identifier (up to 32 bytes)
- **Value**: The data associated with the key (up to 1MB)

### 2. Versions
Every change creates a new version:
- Each put/delete operation increments the version
- You can retrieve data from any past version
- Versions are never deleted (append-only)

### 3. Merkle Tree
ProofBox organizes data in a tree structure where:
- Each node contains a cryptographic hash
- The root hash represents the entire database state
- Any change produces a different root hash

### 4. Cryptographic Proofs
ProofBox can generate proofs that:
- Prove a key has a specific value (inclusion proof)
- Prove a key doesn't exist (exclusion proof)
- Can be verified without access to the database
- Are compact (typically < 2KB)

## Installation

### Option 1: Install with Go
If you have Go 1.21+ installed:

```bash
go install github.com/neutral/proofbox/cmd/pb@latest
```

### Option 2: Build from Source
```bash
git clone https://github.com/neutral/proofbox.git
cd proofbox
make build
# Binary will be at ./bin/pb
```

### Option 3: Download Pre-built Binary
Download the latest release from [GitHub Releases](https://github.com/neutral/proofbox/releases).

### Verify Installation
```bash
pb --help
```

This should display the help message with available commands.

## Your First ProofBox Database

Let's create a simple example to demonstrate ProofBox's key features.

### Step 1: Create a Database
```bash
pb init --db tutorial.db
```

This creates a new ProofBox database file.

### Step 2: Store Some Data
```bash
# Store configuration values
pb put "app:name" "My Application" --db tutorial.db
pb put "app:version" "1.0.0" --db tutorial.db
pb put "app:debug" "false" --db tutorial.db
```

Each operation creates a new version of the database.

### Step 3: Retrieve Data
```bash
pb get "app:name" --db tutorial.db
```

Output:
```
map[key:app:name status:found value:My Application version:3]
```

> 💡 **Tip**: For cleaner output, use the `--json` flag with `jq`:
> ```bash
> pb get "app:name" --db tutorial.db --json | jq .
> ```

### Step 4: View Database State
```bash
pb stats --db tutorial.db
```

Output shows tree statistics including height, node count, and current version.

### Step 5: Generate a Proof
Now let's create a cryptographic proof that "app:version" has the value "1.0.0":

```bash
pb prove "app:version" --db tutorial.db --output version_proof.json
```

### Step 6: Verify the Proof
The proof can be verified by anyone, even without the database:

```bash
pb verify version_proof.json
```

Output:
```
map[key:app:version message:Proof is valid root_hash:... status:valid type:inclusion value:1.0.0 version:3]
```

The proof is valid! The output shows the key, value, and root hash that was verified.

## Understanding What Just Happened

1. **Versioning**: Each put operation created a new version (1, 2, 3)
2. **Root Hash**: The database has a unique root hash representing its complete state
3. **Proof**: We generated a small file (~1KB) that proves "app:version" = "1.0.0"
4. **Verification**: Anyone can verify the proof without accessing your database

## Next Steps

Now that you understand the basics:

1. **For CLI users**: Continue with [CLI Quick Start](cli-quickstart.md)
2. **For developers**: Learn to [use ProofBox as a library](library-quickstart.md)
3. **For production**: Read about [deployment and operations](../how-to/deployment/index.md)

## Key Takeaways

- ✅ ProofBox is a versioned key-value store with cryptographic proofs
- ✅ Every change creates a new version with a unique root hash
- ✅ Proofs can be generated and verified independently
- ✅ Ideal for systems requiring audit trails and data integrity

## Getting Help

- **Documentation**: You're already here!
- **Examples**: Check the `examples/` directory in the repository
- **Issues**: [GitHub Issues](https://github.com/neutral/proofbox/issues)
- **Discussions**: [GitHub Discussions](https://github.com/neutral/proofbox/discussions)

---

Ready to dive deeper? Continue with the [CLI Quick Start](cli-quickstart.md) or explore [using ProofBox as a library](library-quickstart.md).