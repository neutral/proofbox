# Tutorials

Welcome to the ProofBox tutorials! These guides are designed to help you learn ProofBox by taking you through your first steps with the system.

## What You'll Learn

Our tutorials are carefully crafted learning experiences that:
- Start from zero knowledge
- Build understanding step by step
- Provide working examples you can run
- Explain not just "how" but "why"

## Available Tutorials

### 1. [Getting Started with ProofBox](getting-started.md)
**Time: 15 minutes** | **Level: Beginner**

Your first introduction to ProofBox concepts and installation. You'll learn:
- What ProofBox is and when to use it
- How to install ProofBox
- Basic concepts: keys, values, versions, and proofs
- Running your first commands

### 2. [CLI Quick Start](cli-quickstart.md)
**Time: 10 minutes** | **Level: Beginner**

A hands-on introduction to the ProofBox command-line interface. You'll learn:
- Essential CLI commands
- Storing and retrieving data
- Working with versions
- Generating and verifying proofs

### 3. [Using ProofBox as a Library](library-quickstart.md)
**Time: 20 minutes** | **Level: Intermediate**

Learn how to integrate ProofBox into your Go applications. You'll learn:
- Setting up ProofBox in your project
- Basic CRUD operations
- Batch operations for efficiency
- Proof generation and verification

### 4. Building Your First Application *(coming soon)*
**Time: 30 minutes** | **Level: Intermediate**

Build a complete application using ProofBox. You'll create:
- A simple audit log system
- Version history tracking
- Cryptographic proof generation
- A basic verification service

## Before You Begin

### Prerequisites
- Basic command-line familiarity
- For library tutorials: Go 1.21+ installed
- For application tutorial: Basic Go programming knowledge

### System Requirements
- macOS, Linux, or Windows
- 8GB RAM recommended
- 1GB free disk space

## Tutorial Conventions

Throughout our tutorials, we use these conventions:

**Command to run:**
```bash
$ pb put mykey myvalue
```

**Expected output:**
```
Key: mykey
Value: myvalue
Version: 1
Root Hash: 0x3f4a5b...
```

**Important notes:**
> 💡 **Tip**: Important information appears in boxes like this

**Code examples:**
```go
// Go code examples are syntax-highlighted
tree, err := tree.New(db)
```

## Getting Help

If you get stuck:
1. Check the error message carefully
2. Review the previous steps
3. See our [troubleshooting guide](../how-to/troubleshooting.md)
4. Ask in [GitHub Discussions](https://github.com/neutral/proofbox/discussions)

## Next Steps

After completing these tutorials, explore:
- [How-to Guides](../how-to/index.md) for specific tasks
- [Reference Documentation](../reference/index.md) for detailed information
- [Explanation](../explanation/index.md) for deeper understanding

---

Ready to start? Begin with [Getting Started with ProofBox](getting-started.md) →