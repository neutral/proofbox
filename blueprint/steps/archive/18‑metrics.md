---
id: step.18.metrics
depends_on:
tags: [metrics, step]
---

## Objective

Expose Prometheus counter `jmt_commits_total`.

## Implements

- **Non‑Functional Goals – Flexibility & Observability** (wasn't named but implied).
  _What happens_:

  - Real‑time counter supports ops dashboards and mirrors spec's idea of "prometheus metrics appear at /metrics" (original 10‑step plan, step 09).

## Technical Details

### Comprehensive Metrics Collection

Per `nfr.observability.metrics` requirement, implement Prometheus-compatible metrics:

1. **Core JMT Metrics**

   ```
   # Counter metrics
   jmt_commits_total                 # Total number of commits
   jmt_operations_total{type="..."}  # Operations by type (insert/update/delete)
   jmt_proof_generations_total{type="..."} # Proofs generated (inclusion/exclusion)

   # Histogram metrics
   jmt_commit_latency_seconds        # Commit operation duration
   jmt_lookup_latency_seconds        # Lookup operation duration
   jmt_proof_generation_latency_seconds # Proof generation time
   jmt_batch_size                   # Number of operations per batch

   # Gauge metrics
   jmt_tree_height                   # Current tree height
   jmt_tree_nodes_total             # Total nodes in tree
   jmt_versions_stored              # Number of versions retained
   jmt_db_live_bytes               # Storage size in bytes
   ```

2. **Performance Metrics**

   ```
   jmt_hash_operations_total        # Hash computations
   jmt_db_reads_total              # Database read operations
   jmt_db_writes_total             # Database write operations
   jmt_proof_size_bytes            # Size of generated proofs
   ```

3. **Error Tracking**
   ```
   jmt_errors_total{type="..."}    # Errors by type
   jmt_validation_failures_total   # Proof validation failures
   ```

## Implementation Status

✅ **Completed** - All core metrics have been implemented with comprehensive test coverage.

See [Metrics Specification](../features/metrics/metrics-specification.md) for complete details including:
- Full metric definitions with current implementation status
- Performance requirements and testing guidelines
- Integration points and usage examples
- Production deployment recommendations
