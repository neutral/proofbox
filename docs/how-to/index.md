# How-to Guides

How-to guides provide practical, step-by-step instructions for accomplishing specific tasks with ProofBox. Unlike tutorials, these guides assume you already understand ProofBox basics and need to get something done.

## Guide Categories

### 🖥️ CLI Operations
Practical guides for using the ProofBox command-line interface

- [Basic CLI Usage](cli/basic-usage.md) - Common CLI operations
- [Batch Operations](cli/batch-operations.md) - Processing multiple operations efficiently  
- [Import and Export](cli/import-export.md) - Moving data between databases
- [REPL Mode](cli/repl-mode.md) - Using the interactive shell

### 📚 Library Usage
Integrating ProofBox into your Go applications

- [Basic Operations](library/basic-operations.md) - CRUD operations with the Go API
- [Batch Processing](library/batch-processing.md) - Efficient bulk operations
- [Proof Generation](library/proof-generation.md) - Creating and verifying proofs programmatically
- [Version Management](library/version-management.md) - Working with multiple versions

### 🚀 [Local Usage](deployment/index.md)
Using ProofBox as a CLI tool and Go library

- [Local Usage Guide](deployment/local-usage.md) - Install and use ProofBox locally
- Embed ProofBox in Go applications
- Local database operations

⚠️ **Note**: ProofBox is a local tool only. It does not support server deployment, Docker, or Kubernetes.

### 🛠️ Development
Contributing to ProofBox

- [Development Workflow](development.md) - Setting up your development environment
- Testing Guide *(coming soon)* - Running and writing tests
- Benchmarking *(coming soon)* - Performance testing

## How to Use These Guides

Each guide:
1. **States the goal** clearly at the beginning
2. **Lists prerequisites** you need before starting
3. **Provides step-by-step instructions** to achieve the goal
4. **Shows expected outcomes** so you can verify success
5. **Includes troubleshooting tips** for common issues

## Guide Conventions

**Commands to run:**
```bash
$ pb command --flag value
```

**Configuration examples:**
```yaml
# config.yaml
storage:
  type: pebble
  path: /var/lib/proofbox
```

**Code snippets:**
```go
tree, err := tree.New(db)
if err != nil {
    return err
}
```

**Important warnings:**
> ⚠️ **Warning**: Critical information appears in boxes like this

**Helpful tips:**
> 💡 **Tip**: Useful suggestions appear like this

## Finding the Right Guide

Not sure which guide you need? Here are common scenarios:

**"I want to..."**
- Import data from another system → [Import and Export](cli/import-export.md)
- Process thousands of operations → [Batch Operations](cli/batch-operations.md)
- Use ProofBox interactively → [REPL Mode](cli/repl-mode.md)
- Learn basic operations → [Basic CLI Usage](cli/basic-usage.md)
- Monitor my ProofBox instance → Monitoring Setup *(coming soon)*
- Optimize performance → Performance Tuning *(coming soon)*
- Generate proofs in my app → [Proof Generation](library/proof-generation.md)
- Work with versions → [Version Management](library/version-management.md)
- Batch operations in code → [Batch Processing](library/batch-processing.md)

## Can't Find What You Need?

- Check the [Reference](../reference/index.md) for detailed technical information
- Read the [Explanation](../explanation/index.md) section for conceptual understanding
- Browse the [Examples](../examples/) for working code samples
- Ask in [GitHub Discussions](https://github.com/neutral/proofbox/discussions)
- [Open an issue](https://github.com/neutral/proofbox/issues) to request new guides

---

Choose a category above to browse the available guides, or use the search function to find specific tasks.