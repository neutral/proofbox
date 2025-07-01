# Storage Configuration

This module provides a unified configuration system for different storage backends, supporting both in-memory and persistent storage with various tuning options.

## Purpose

The configuration system enables:
- Easy switching between storage backends (memory, PebbleDB, etc.)
- Performance tuning through cache and buffer size configuration
- Compression settings for space optimization
- Pruning policy configuration for automatic cleanup

## Configuration Structure

### Backend Selection
The `Backend` field determines which storage implementation to use:
- `"memory"`: Fast in-memory storage for testing and development
- `"pebble"`: Production-ready persistent storage using PebbleDB

### Performance Tuning
- **CacheSize**: Controls the in-memory cache for frequently accessed data
- **WriteBufferSize**: Sets the size of write buffers before flushing to disk
- **MaxOpenFiles**: Limits file descriptor usage for large databases

### Compression
Configurable compression reduces storage space at the cost of CPU usage:
- Multiple algorithms supported (Snappy, ZSTD)
- Can be disabled for maximum performance
- Particularly effective for repetitive data patterns

### Pruning Configuration
Integrates with the pruning policy system to automatically remove old versions:
- Policy type selection (keep last N, time-based, etc.)
- Configurable retention parameters
- Helps prevent unbounded storage growth

## Design Decisions

- **Validation**: All configuration values are validated to prevent runtime errors from invalid settings
- **Defaults**: Sensible defaults are provided for all settings, allowing minimal configuration for basic usage
- **Backend Abstraction**: The configuration structure is backend-agnostic, with backend-specific options handled internally
- **JSON/YAML Support**: Configuration can be easily loaded from files or environment variables

## Usage Patterns

The configuration system supports multiple usage patterns:
1. **Minimal**: Use `DefaultConfig()` for development and testing
2. **Production**: Tune cache sizes and enable compression for optimal performance
3. **Archival**: Configure longer retention periods with aggressive compression
4. **High Performance**: Maximize cache and buffer sizes, disable compression