# ProofBox Documentation

Welcome to the ProofBox documentation! ProofBox is a high-performance, versioned key-value store implementing the Jellyfish Merkle Tree (JMT) data structure in Go.

## Documentation Structure

Our documentation follows the [Diátaxis framework](https://diataxis.fr/), organizing content into four distinct types to best serve your needs:

### 📚 [Tutorials](tutorials/index.md)

**Learning-oriented** - Start here if you're new to ProofBox

- [Getting Started with ProofBox](tutorials/getting-started.md)
- [CLI Quick Start](tutorials/cli-quickstart.md)
- [Using ProofBox as a Library](tutorials/library-tutorial.md)
- See working code in the [examples directory](examples/)

### 🔧 [How-to Guides](how-to/index.md)

**Task-oriented** - Practical guides for specific tasks

- **CLI Operations**: [Basic usage](how-to/cli/basic-usage.md), [batch processing](how-to/cli/batch-operations.md), [import/export](how-to/cli/import-export.md), [REPL mode](how-to/cli/repl-mode.md)
- **Library Usage**: [CRUD operations](how-to/library/basic-operations.md), [batch processing](how-to/library/batch-processing.md), [proof generation](how-to/library/proof-generation.md), [versioning](how-to/library/version-management.md)
- **Deployment**: [Local usage](how-to/deployment/local-usage.md)
- **Development**: [Development workflow](how-to/development.md), testing, benchmarking

### 📖 [Reference](reference/index.md)

**Information-oriented** - Technical details and specifications

- [CLI Command Reference](reference/cli/commands.md)
- [API Documentation](reference/api/)
- Configuration Options _(coming soon)_
- Error Reference _(coming soon)_
- Metrics Reference _(coming soon)_

### 💡 [Explanation](explanation/index.md)

**Understanding-oriented** - Concepts and design decisions

- [Architecture Overview](explanation/architecture.md)
- [Jellyfish Merkle Tree Explained](explanation/jellyfish-merkle-tree.md)
- [Design Decisions](explanation/design-decisions.md)
- [Performance Characteristics](explanation/performance.md)
- [Security Model](explanation/security-model.md)

### 🚀 [Examples](examples/)

**Practice-oriented** - Working code examples

- **Batch Operations**: Basic transactions, deduplication, concurrent operations, validation
- **Performance**: Parallel processing, optimization techniques
- **Internals**: Understanding UpdateBatch structure
- Run examples directly from the [examples directory](examples/)

## Quick Links

- **New to ProofBox?** Start with [Getting Started](tutorials/getting-started.md)
- **Using the CLI?** See [CLI Quick Start](tutorials/cli-quickstart.md)
- **Building an application?** Check the [examples directory](examples/)
- **Need API details?** See the [API Reference](reference/api/)
- **Want to understand the internals?** Read [Architecture Overview](explanation/architecture.md)

## 📖 Reading Paths

Based on your role:

### Application Developer
1. [Getting Started](tutorials/getting-started.md)
2. [Using ProofBox as a Library](tutorials/library-tutorial.md)
3. [API Reference](reference/api/)
4. [Library How-to Guides](how-to/library/basic-operations.md)

### System Administrator
1. [CLI Quick Start](tutorials/cli-quickstart.md)
2. [CLI How-to Guides](how-to/cli/basic-usage.md)
3. [Local Usage Guide](how-to/deployment/local-usage.md)
4. [CLI Command Reference](reference/cli/commands.md)

### Contributor
1. [Architecture Overview](explanation/architecture.md)
2. [Development Guide](how-to/development.md)
3. [Design Decisions](explanation/design-decisions.md)
4. [API Reference](reference/api/)

## 🔍 Finding Information

- **Use the search box** (when viewing in documentation site)
- **Browse by category** using the navigation menu
- **Check the index** at the start of each section
- **Follow cross-references** between related topics

## 📦 Documentation Status

This documentation covers ProofBox v0.1.0. Some sections are still being expanded:

- ✅ CLI documentation (complete)
- ✅ Basic tutorials (complete)
- ✅ API reference (complete)
- ✅ Examples (complete)
- 🚧 Explanation sections (core concepts done)
- 📝 How-to guides (expanding based on user needs)

## Getting Help

- **GitHub Issues**: [Report bugs or request features](https://github.com/neutral/proofbox/issues)
- **Discussions**: [Ask questions and share ideas](https://github.com/neutral/proofbox/discussions)
- **Contributing**: See our [contribution guidelines](../CONTRIBUTING.md)

### Documentation Principles

When contributing to docs:
- **Clear and concise** - Avoid jargon, explain terms
- **Practical examples** - Show real-world usage
- **Accurate and tested** - All examples must work
- **Well-organized** - Follow Diátaxis categories

## Version

This documentation is for ProofBox v0.1.0. For other versions, see the [version selector](#) (coming soon).
