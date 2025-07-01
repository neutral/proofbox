# Local Usage Guide

⚠️ **Note**: ProofBox is a local CLI tool and Go library. It does not have server, Docker, or Kubernetes deployment capabilities.

## What ProofBox Is

- **CLI Tool**: Command-line interface for local merkle tree operations
- **Go Library**: Can be embedded in your Go applications
- **Local Database**: Uses PebbleDB for local storage

## Available Guide

### 📍 [Local Usage](local-usage.md)
Learn how to:
- Install and use the CLI tool
- Embed ProofBox as a Go library
- Perform local database operations
- Generate and verify proofs
- Import and export data

## What ProofBox Does NOT Support

ProofBox currently does **not** support:
- ❌ Running as a server/daemon
- ❌ HTTP API endpoints
- ❌ Docker containers
- ❌ Kubernetes deployments
- ❌ Remote access
- ❌ Configuration files
- ❌ Metrics endpoints

These features may be added in future versions according to the project roadmap (steps 21-25 of the blueprint).

## For Production Use

To use ProofBox in production applications:
1. Embed it as a Go library
2. Build your own server/API layer around it
3. Implement your own deployment strategy

See the [Local Usage Guide](local-usage.md) for examples of embedding ProofBox in applications.