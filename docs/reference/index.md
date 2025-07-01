# Reference Documentation

The reference documentation provides comprehensive technical details about ProofBox's components, APIs, and configuration options. This section is designed for looking up specific information rather than learning or accomplishing tasks.

## Reference Categories

### 🖥️ CLI Reference
Complete documentation of all CLI commands and options

- [Command Reference](cli/commands.md) - All commands with full syntax and options
- Configuration *(coming soon)* - CLI configuration files and environment variables

### 📚 [API Reference](api/)
Detailed documentation of the Go API

- [Tree Package](api/tree.md) - Core tree operations and types
- [Proof Package](api/proof.md) - Proof generation and verification
- [Storage Package](api/storage.md) - Storage interfaces and implementations
- [Types Package](api/types.md) - Common types and utilities
- [Codec Package](api/codec.md) - Node encoding and decoding
- [Crypto Package](api/crypto.md) - Cryptographic utilities
- [Metrics Package](api/metrics.md) - Monitoring and metrics collection

### ⚙️ Configuration Reference *(coming soon)*
Complete guide to all configuration options

- Storage configuration
- Performance tuning parameters
- Security settings
- Logging and monitoring options

### ❌ Error Reference *(coming soon)*
Comprehensive list of error codes and their meanings

- Error types and categories
- Common causes and solutions
- Error handling best practices

### 📊 Metrics Reference *(coming soon)*
All available metrics for monitoring

- Metric names and types
- Metric labels and values
- Prometheus integration details
- Grafana dashboard examples

## Using the Reference Documentation

### Quick Lookup

Each reference page includes:
- **Synopsis** - Brief description of the component
- **Details** - Complete technical specifications
- **Examples** - Code snippets showing usage
- **Related** - Links to related documentation

### Navigation Aids

- **Search** - Use the search box to find specific items
- **Index** - Alphabetical index of all reference items
- **Cross-references** - Links between related topics

### Code Examples

Reference documentation includes minimal examples focused on syntax:

```go
// Creating a new tree
tree, err := tree.New(db, tree.WithMetrics(metrics))
```

For learning how to use these APIs, see the [Tutorials](../tutorials/index.md) and [How-to Guides](../how-to/index.md).

## API Stability

### Stable APIs
These APIs are stable and safe to use in production:
- Tree operations (Get, Put, Delete)
- Proof generation and verification
- Storage interfaces

### Experimental APIs
These APIs may change in future versions:
- Metrics collection details
- Advanced configuration options

### Deprecated APIs
Currently, no APIs are deprecated.

## Version Information

This reference documentation is for ProofBox v0.1.0. API compatibility:
- **Major version (0.x)**: May include breaking changes
- **Minor version (0.1.x)**: Backward compatible changes only
- **Patch version (0.1.x)**: Bug fixes only

## Contributing to Reference Docs

Reference documentation is generated from:
1. **Source code comments** - godoc-style comments
2. **Command definitions** - Cobra command structures
3. **Configuration schemas** - YAML/JSON schemas

To improve reference documentation:
1. Update source code comments
2. Ensure examples compile and run
3. Follow godoc conventions
4. Submit a pull request

## Quick Links

**Most Referenced Pages:**
- [CLI Commands](cli/commands.md) - Complete command reference
- [Tree API](api/tree.md) - Core tree operations
- [Types API](api/types.md) - Common types and constants
- [Storage API](api/storage.md) - Storage interfaces

---

Select a category above or use the search function to find specific reference information.