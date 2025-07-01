# Import and Export

This guide covers data migration between ProofBox databases using import and export commands.

## Goal

Learn how to export data from ProofBox and import it into another database, useful for backups, migrations, and data sharing.

## Prerequisites

- ProofBox CLI installed
- Source database with data to export
- Target database initialized (for import)

## Export Operations

### Export Specific Keys

Export selected keys to a JSON file:

```bash
# Create a file with keys to export (one per line)
$ cat > keys.txt << EOF
user:1001
user:1002
user:1003
EOF

# Export keys from file
$ pb export-keys --keys keys.txt --output users.json
Exported 3 entries (of 3 requested) to users.json in 400µs

# Export via stdin
$ echo -e "config:app\nconfig:database\nconfig:cache" | pb export-keys --output config_backup.json
```

### Export Format

The exported JSON file contains:

```json
{
  "version": 25,
  "export_time": "2025-07-01T10:00:00Z",
  "root_hash": "8883a78df8fa9cb9ff97b1c972343e25d979c03d3dfd6b7029cd575e927a878c",
  "entry_count": 2,
  "entries": [
    {
      "key": "user:1001",
      "value": "Alice Johnson"
    },
    {
      "key": "user:1002", 
      "value": "Bob Smith"
    }
  ]
}
```

### Export All Keys

To export all keys, you need to maintain a list:

```bash
# Create a comprehensive key list
$ cat all_keys.txt
user:1001
user:1002
config:app
metadata:version

# Export all listed keys
$ pb export-keys --keys all_keys.txt --output full_backup.json
```

## Import Operations

### Basic Import

Import data atomically from a JSON file:

```bash
# Import from JSON file (all entries in one atomic transaction)
$ pb import users.json
&{TotalEntries:3 SuccessfulImports:3 FailedImports:0 StartVersion:30 EndVersion:31 Duration:15ms Errors:[]}
```

Note: Import is atomic - all entries are imported in a single transaction. If any entry fails, no changes are made.

### Import Formats

ProofBox supports multiple import formats:

```bash
# JSON format (default)
$ pb import data.json

# CSV format
$ pb import data.csv --format csv

# Specify format explicitly
$ pb import data.json --format json
```

### CSV Format

For CSV exports/imports:

```bash
# Export to CSV
$ pb export-keys --keys keys.txt --format csv --output data.csv

# CSV structure:
key,value
user:1001,Alice Johnson
user:1002,Bob Smith
config:theme,dark
```

### Validation Before Import

Always validate import files before executing:

```bash
# Validate import file
$ pb import users.json --validate
Validation successful: 3 entries ready to import

# Validation checks:
# - File format validity
# - JSON structure correctness
# - Value size limits
```

### Skip Errors During Import

Continue importing even if some entries fail:

```bash
# Import with error skipping
$ pb import mixed_data.json --skip-errors
&{TotalEntries:100 SuccessfulImports:95 FailedImports:5 StartVersion:40 EndVersion:135 Duration:200ms Errors:[entry 15: value too large, entry 67: invalid key]}
```

## Migration Workflows

### Database Backup

Create a complete backup:

```bash
#!/bin/bash
# backup.sh

# Maintain a key list file
cat > backup_keys.txt << EOF
user:1001
user:1002
user:1003
config:app
config:db
config:cache
metadata:version
metadata:schema
EOF

# Export to timestamped file
BACKUP_FILE="backup_$(date +%Y%m%d_%H%M%S).json"
pb export-keys --keys backup_keys.txt --output "$BACKUP_FILE"

echo "Backup created: $BACKUP_FILE"
```

### Database Migration

Migrate between databases:

```bash
# 1. Export from source database
$ echo -e "key1\nkey2\nkey3" | pb export-keys --db source.db --output migration.json

# 2. Import to target database  
$ pb import migration.json --db target.db

# 3. Verify migration
$ pb get key1 --db target.db
```

### Incremental Updates

Export and import only changed data:

```bash
# Export recent changes (manual tracking required)
$ echo -e "user:1004\nuser:1005\nconfig:newfeature" | pb export-keys --output updates.json

# Import updates to another instance
$ pb import updates.json --db replica.db
```

## Data Transformation

### Transform During Migration

Use standard tools to transform data:

```bash
# Export data
$ echo -e "user:old:1\nuser:old:2" | pb export-keys --output raw_data.json

# Transform with jq
$ jq '.entries |= map(
  .key |= sub("user:old:"; "user:new:")
)' raw_data.json > transformed_data.json

# Import transformed data
$ pb import transformed_data.json
```

### Format Conversion

Convert between formats:

```bash
# Export directly to CSV
$ echo -e "key1\nkey2\nkey3" | pb export-keys --format csv --output data.csv

# Or convert JSON to CSV using jq
$ pb export-keys --keys keys.txt --output data.json
$ jq -r '.entries[] | [.key, .value] | @csv' data.json > data.csv

# Import CSV
$ pb import data.csv --format csv
```

### Filtering Exports

Export and filter data:

```bash
# Export all user keys
$ cat > user_keys.txt << EOF
user:1001
user:1002 
user:1003
user:admin
EOF
$ pb export-keys --keys user_keys.txt --output all_users.json

# Filter out admin users
$ jq '.entries |= map(select(.key != "user:admin"))' all_users.json > regular_users.json

# Import filtered data
$ pb import regular_users.json
```

## Best Practices

### Backup Strategy

1. **Regular Backups**: Schedule periodic exports
2. **Versioned Backups**: Include version info in filenames
3. **Test Restores**: Regularly verify backup integrity
4. **Secure Storage**: Protect backup files appropriately

### Migration Safety

1. **Validate First**: Always use --validate before import
2. **Test Migration**: Try with subset of data first
3. **Version Tracking**: Note source and target versions
4. **Rollback Plan**: Keep source data until verified

### Performance Tips

1. **Batch Exports**: Export multiple keys in one command
2. **Compression**: Compress large export files
3. **Incremental**: Export only changed data when possible
4. **Atomic Imports**: All entries imported in one transaction (efficient)

## Common Scenarios

### Development to Production

```bash
# Export development data
$ echo -e "config:app\nconfig:features\nuser:test" | \
  pb export-keys --db dev.db --output dev_data.json

# Review and clean data
$ jq '.entries |= map(select(.key | startswith("config:")))' \
  dev_data.json > prod_config.json

# Import to production
$ pb import prod_config.json --db prod.db --validate
$ pb import prod_config.json --db prod.db
```

### Cross-Region Replication

```bash
# Export from region A
$ echo -e "user:1001\nuser:1002\nmetadata:region" | \
  pb export-keys --db regionA.db --output region_a_data.json

# Transfer file (scp, rsync, etc.)
$ scp region_a_data.json remote:/tmp/

# Import to region B
$ ssh remote "pb import /tmp/region_a_data.json --db regionB.db"
```

### Disaster Recovery

```bash
#!/bin/bash
# disaster_recovery.sh

# Check if backup exists
if [ ! -f "latest_backup.json" ]; then
  echo "Error: No backup file found"
  exit 1
fi

# Validate backup
pb import latest_backup.json --validate || exit 1

# Create new database
pb init --db recovered.db

# Import data
pb import latest_backup.json --db recovered.db

# Verify critical keys
pb get config:app --db recovered.db
pb get metadata:version --db recovered.db
```

## Error Handling

### Export Errors

```bash
# No keys to export
$ echo "" | pb export-keys --output output.json
Exported 0 entries (of 0 requested) to output.json

# Invalid output path
$ echo "key1" | pb export-keys --output /invalid/path/output.json
Error: failed to create output file: no such file or directory
```

### Import Errors

```bash
# Invalid JSON format
$ pb import corrupt.json
Error: failed to decode import file: invalid character

# Validation failure - nothing imported (atomic)
$ pb import large_values.json
Error: import failed at entry 3: value exceeds maximum size (1MB)

# Skip validation errors (still atomic for valid entries)
$ pb import large_values.json --skip-errors
&{TotalEntries:60 SuccessfulImports:58 FailedImports:2 StartVersion:100 EndVersion:101 Duration:120ms Errors:[entry 3: value too large, entry 47: invalid key]}
```

### Recovery from Failed Import

```bash
# Import failed - no changes made (atomic)
$ pb import data.json
Error: import batch execution failed: operation 45 (put) failed: value too large

# Fix the problematic entries and retry
$ pb import fixed_data.json
&{TotalEntries:60 SuccessfulImports:60 FailedImports:0 StartVersion:100 EndVersion:101 Duration:120ms Errors:[]}
```

## Advanced Usage

### Streaming Large Exports

For very large datasets:

```bash
#!/bin/bash
# export_large.sh

# Split key list into chunks
split -l 1000 all_keys.txt keys_chunk_

# Export each chunk
COUNTER=0
for chunk in keys_chunk_*; do
  pb export-keys --keys "$chunk" --output "export_${COUNTER}.json"
  COUNTER=$((COUNTER + 1))
done

# Combine chunks (careful with memory)
jq -s '.[0] + {entries: map(.entries) | add}' export_*.json > full_export.json
```

### Automated Sync

```bash
#!/bin/bash
# sync_databases.sh

# Export from primary
pb export-keys --keys key_list.txt --db primary.db --output sync_data.json

# Import to replicas
for replica in replica1.db replica2.db replica3.db; do
  pb import sync_data.json --db "$replica"
done
```

## Troubleshooting

### Import Validation Fails

```bash
# Check file format
$ file import.json
import.json: ASCII text

# Validate JSON syntax
$ jq . import.json > /dev/null || echo "Invalid JSON"

# Check structure
$ jq 'keys' import.json
["entries", "entry_count", "export_time", "root_hash", "version"]
```

### Performance Issues

```bash
# For very large imports (millions of entries):
# Consider splitting into manageable chunks
# Each chunk is still imported atomically

# 1. Split large JSON file (careful with JSON structure)
$ jq -c '.entries[0:10000]' large_import.json > chunk1.json
$ jq -c '.entries[10000:20000]' large_import.json > chunk2.json

# 2. Import each chunk atomically
$ pb import chunk1.json
$ pb import chunk2.json
```

### Version Conflicts

```bash
# Check database version
$ pb stats | grep version

# Export includes version info
$ jq '.version' export.json

# Handle version differences appropriately
```

## Next Steps

- Try [REPL Mode](repl-mode.md) for interactive import/export
- Learn about [Batch Operations](batch-operations.md) for bulk updates
- See the [CLI Reference](../../reference/cli/commands.md) for all options

## Quick Reference

```bash
# Export operations
pb export-keys --keys <file> --output <file>     # Export from key list
echo "key1" | pb export-keys --output <file>     # Export from stdin
pb export-keys --format csv --output <file>       # Export as CSV
pb export-keys --version <n> --output <file>      # Export at version

# Import operations  
pb import <file>                                  # Import from file
pb import <file> --format csv                     # Import CSV format
pb import <file> --validate                       # Validate without importing
pb import <file> --skip-errors                    # Continue on errors

# File formats
JSON: {"version": N, "entries": [{"key": "k", "value": "v"}]}
CSV: key,value
```