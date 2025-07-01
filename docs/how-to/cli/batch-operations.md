# Batch Operations

This guide shows how to perform efficient bulk operations using ProofBox's batch command.

## Goal

Execute multiple key-value operations efficiently in a single atomic transaction, ensuring all operations succeed or all fail together.

## Prerequisites

- ProofBox CLI installed and database initialized
- Basic understanding of JSON format
- Familiarity with basic CLI operations

## Understanding Batch Operations

Batch operations allow you to:
- Execute multiple operations atomically (all succeed or all fail)
- Achieve better performance than individual operations
- Maintain consistency across related data updates

## Creating Batch Files

### JSON Format

Create a JSON file with an array of operations:

```json
{
  "operations": [
    {
      "type": "put",
      "key": "user:1001",
      "value": "Alice Johnson"
    },
    {
      "type": "put", 
      "key": "user:1002",
      "value": "Bob Smith"
    },
    {
      "type": "delete",
      "key": "user:999"
    }
  ]
}
```

### Operation Types

Supported operation types:
- `put`: Insert or update a key-value pair
- `delete`: Remove a key

### Complex Values

Store structured data:

```json
{
  "operations": [
    {
      "type": "put",
      "key": "config:database",
      "value": "{\"host\":\"localhost\",\"port\":5432,\"ssl\":true}"
    },
    {
      "type": "put",
      "key": "metadata:version",
      "value": "2.1.0"
    }
  ]
}
```

## Executing Batch Operations

### Basic Execution

```bash
# Execute a batch file
$ pb batch operations.json
&{TotalOperations:3 SuccessfulOps:3 FailedOps:0 StartVersion:10 EndVersion:11 Duration:15.2ms Errors:[]}
```

### With Validation

```bash
# Validate without executing (dry run)
$ pb batch operations.json --dry-run
map[operation_count:3 operations:[...] status:dry_run valid:true]
```

### Output Format

```bash
# JSON output for parsing
$ pb batch operations.json --json | jq .
{
  "total_operations": 3,
  "successful_operations": 3,
  "failed_operations": 0,
  "start_version": 10,
  "end_version": 11,
  "duration": "15.2ms",
  "errors": []
}
```

## Practical Examples

### User Data Migration

Create `migrate_users.json`:
```json
{
  "operations": [
    {
      "type": "put",
      "key": "user:alice@example.com",
      "value": "{\"id\":1001,\"name\":\"Alice\",\"role\":\"admin\"}"
    },
    {
      "type": "put",
      "key": "user:bob@example.com", 
      "value": "{\"id\":1002,\"name\":\"Bob\",\"role\":\"user\"}"
    },
    {
      "type": "put",
      "key": "user:charlie@example.com",
      "value": "{\"id\":1003,\"name\":\"Charlie\",\"role\":\"user\"}"
    },
    {
      "type": "delete",
      "key": "legacy:user:1"
    },
    {
      "type": "delete",
      "key": "legacy:user:2"
    }
  ]
}
```

Execute:
```bash
$ pb batch migrate_users.json
&{TotalOperations:5 SuccessfulOps:5 FailedOps:0 StartVersion:15 EndVersion:16 Duration:25.5ms Errors:[]}
```

### Configuration Update

Create `update_config.json`:
```json
{
  "operations": [
    {
      "type": "put",
      "key": "app:settings:theme",
      "value": "dark"
    },
    {
      "type": "put",
      "key": "app:settings:language",
      "value": "en-US"
    },
    {
      "type": "put",
      "key": "app:settings:notifications",
      "value": "true"
    },
    {
      "type": "put",
      "key": "app:version",
      "value": "3.0.0"
    }
  ]
}
```

### Bulk Data Import

For large datasets, create `products.json`:
```json
{
  "operations": [
    {
      "type": "put",
      "key": "product:SKU001",
      "value": "{\"name\":\"Widget A\",\"price\":19.99,\"stock\":100}"
    },
    {
      "type": "put",
      "key": "product:SKU002",
      "value": "{\"name\":\"Widget B\",\"price\":29.99,\"stock\":50}"
    },
    {
      "type": "put",
      "key": "product:SKU003",
      "value": "{\"name\":\"Widget C\",\"price\":39.99,\"stock\":25}"
    }
  ]
}
```

## Generating Batch Files

### From Database Export

```bash
# Export keys to create a template
$ pb export-keys template.json product:*

# Edit template.json to modify values
# Then import back as batch
$ pb batch template.json
```

### Programmatically

Create batch files with scripts:

```bash
#!/bin/bash
# generate_batch.sh
cat > batch.json << EOF
{
  "operations": [
EOF

for i in {1..100}; do
  if [ $i -ne 1 ]; then echo "," >> batch.json; fi
  cat >> batch.json << EOF
    {
      "type": "put",
      "key": "test:key$i",
      "value": "value$i"
    }
EOF
done

cat >> batch.json << EOF
  ]
}
EOF
```

## Performance Considerations

### Batch Size

- Optimal batch size: 100-1000 operations
- Larger batches use more memory
- Very large batches may timeout

### Atomicity

Batch operations are atomic - all operations succeed or all fail:
```bash
# If any operation fails, no changes are committed
$ pb batch mixed_batch.json
&{TotalOperations:5 SuccessfulOps:5 FailedOps:0 StartVersion:20 EndVersion:21 Duration:30ms Errors:[]}

# Example of atomic failure - nothing is committed
$ pb batch invalid_batch.json
&{TotalOperations:5 SuccessfulOps:0 FailedOps:5 StartVersion:20 EndVersion:20 Duration:5ms Errors:[batch execution failed: operation 3 (delete) failed: jmt: key not found]}
```

### Memory Usage

For very large batches:
```bash
# Process in chunks
$ split -l 1000 large_batch.json chunk_
$ for chunk in chunk_*; do
  pb batch $chunk
done
```

## Error Handling

### Validation Errors

```bash
# Check batch file syntax
$ pb batch malformed.json --dry-run
Error: failed to decode batch file: unexpected end of JSON input
```

### Operation Errors

```bash
# Common errors
Error: operation 3: value too large (max 1MB)
Error: operation 7: invalid key format
```

### Handling Failures

```bash
# Batch execution is atomic - if any operation fails, nothing is committed
$ pb batch mixed_valid_invalid.json
&{TotalOperations:10 SuccessfulOps:0 FailedOps:10 StartVersion:20 EndVersion:20 Duration:10ms Errors:[batch execution failed: operation 3 (delete) failed: jmt: key not found]}
```

## Best Practices

1. **Validate First**: Always use --dry-run for large batches before execution
2. **Atomic Guarantee**: All operations succeed or all fail - no partial updates
3. **Monitor Size**: Keep batches reasonable (under 10MB)
4. **Check Results**: Verify the batch completed successfully
5. **Version Awareness**: A successful batch increments the version only once
6. **Error Recovery**: If a batch fails, no changes are made to the database

## Advanced Usage

### Conditional Updates

While ProofBox doesn't support conditional updates directly, you can:

1. Export current state
2. Generate batch based on current values
3. Apply batch atomically

The atomic nature of batch operations ensures consistency even if the state changes between export and import.

### Migration Scripts

```bash
#!/bin/bash
# migrate.sh
# Export old format
pb export-keys old_data.json "old:*"

# Transform data (using jq or other tools)
jq '.operations | map(
  if .key | startswith("old:") then
    .key = (.key | sub("old:"; "new:"))
  else . end
)' old_data.json > new_data.json

# Import new format
pb batch new_data.json
```

## Troubleshooting

### Batch Won't Execute

```bash
# Check JSON syntax
$ jq . batch.json
parse error: Invalid numeric literal at line 10

# Validate operations
$ pb batch batch.json --validate
```

### Performance Issues

```bash
# For slow batches, check:
# 1. Batch size (split if >1000 ops)
# 2. Value sizes (keep under 100KB each)
# 3. Database location (use SSD)
```

### Memory Errors

```bash
# For "out of memory" errors:
# Split large batches
$ jq -c '.operations' large.json | split -l 500 - batch_
```

## Next Steps

- Learn about [Import and Export](import-export.md) for data migration
- Explore [REPL Mode](repl-mode.md) for interactive batch creation
- Read about performance tuning in the reference documentation

## Quick Reference

```bash
# Batch operations
pb batch <file>              # Execute batch file atomically
pb batch <file> --dry-run    # Validate without executing

# Batch file format
{
  "operations": [
    {"type": "put", "key": "k1", "value": "v1"},
    {"type": "delete", "key": "k2"}
  ]
}
```