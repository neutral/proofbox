# Explanation

The explanation section provides in-depth understanding of ProofBox's concepts, architecture, and design decisions. These guides explain the "why" behind ProofBox's implementation, helping you understand not just how to use it, but how it works and why it was built this way.

## Available Topics

### 🏗️ [Architecture Overview](architecture.md)
**Understanding ProofBox's system design**

Learn about:
- Overall system architecture
- Component interactions
- Data flow and lifecycle
- Storage layer design
- Concurrency model

### 🌳 [Jellyfish Merkle Tree Explained](jellyfish-merkle-tree.md)
**The data structure at ProofBox's core**

Understand:
- What is a Jellyfish Merkle Tree (JMT)?
- How JMT differs from other Merkle trees
- Why JMT is ideal for versioned storage
- Proof generation and verification mechanics
- Space and time complexity analysis

### 🎯 [Design Decisions](design-decisions.md)
**Key architectural choices and trade-offs**

Explore:
- Why Go instead of Rust/C++?
- Why PebbleDB for storage?
- Two-node type design rationale
- Version management approach
- API design philosophy

### ⚡ [Performance Characteristics](performance.md)
**Understanding ProofBox's performance profile**

Learn about:
- Read/write performance characteristics
- Memory usage patterns
- Disk I/O behavior
- Scalability limits
- Optimization strategies

### 🔒 [Security Model](security-model.md)
**ProofBox's approach to data integrity and security**

Understand:
- Cryptographic guarantees
- Threat model
- Trust assumptions
- Proof security properties
- Best practices for secure usage

### 🔄 [Comparison with Other Systems](comparison.md)
**How ProofBox relates to similar technologies**

Compare with:
- Traditional key-value stores (Redis, RocksDB)
- Other Merkle tree implementations
- Blockchain state stores
- Version control systems

## How to Read This Section

### For Different Audiences

**Application Developers:**
Start with [Architecture Overview](architecture.md) and [Performance Characteristics](performance.md) to understand how ProofBox will behave in your system.

**System Architects:**
Focus on [Design Decisions](design-decisions.md) and [Comparison](comparison.md) to evaluate if ProofBox fits your requirements.

**Researchers/Students:**
Begin with [Jellyfish Merkle Tree Explained](jellyfish-merkle-tree.md) for the theoretical foundation.

**Security Engineers:**
Review [Security Model](security-model.md) and relevant sections of [JMT Explained](jellyfish-merkle-tree.md).

### Reading Order

While each document stands alone, this sequence provides good conceptual flow:

1. [Jellyfish Merkle Tree Explained](jellyfish-merkle-tree.md) - Core concepts
2. [Architecture Overview](architecture.md) - System structure
3. [Design Decisions](design-decisions.md) - Implementation choices
4. [Performance Characteristics](performance.md) - Practical implications
5. [Security Model](security-model.md) - Trust and guarantees

## Key Concepts

Before diving in, familiarize yourself with these terms:

- **Merkle Tree**: A tree where each node contains a cryptographic hash of its children
- **Versioning**: Maintaining multiple states of the data over time
- **Sparse Tree**: A tree where most possible positions are empty
- **Proof**: Cryptographic evidence that data exists (or doesn't exist) in the tree
- **Root Hash**: The hash at the top of the tree that represents the entire state

## Visual Aids

Throughout this section, we use diagrams to illustrate complex concepts:

```
     Root Hash
      /     \
   Hash₁    Hash₂     <- Internal nodes contain child hashes
   /  \      /  \
 Leaf₁ Leaf₂ Leaf₃ □   <- Leaves contain actual data
```

## Deep Dives

Each explanation includes:
- **Conceptual Overview** - High-level understanding
- **Technical Details** - In-depth exploration
- **Practical Implications** - What this means for users
- **Examples** - Concrete illustrations
- **Further Reading** - Academic papers and references

## Contributing to Explanations

To improve explanation documentation:
1. Ensure technical accuracy
2. Use clear, accessible language
3. Include diagrams where helpful
4. Provide concrete examples
5. Link to relevant research

## Questions These Docs Answer

- Why does ProofBox exist?
- How does cryptographic verification work?
- What makes ProofBox fast?
- When should I use ProofBox vs alternatives?
- How does ProofBox ensure data integrity?
- What are the scalability limits?

---

Choose a topic above to deepen your understanding of ProofBox, or start with the [Architecture Overview](architecture.md) for a comprehensive introduction.