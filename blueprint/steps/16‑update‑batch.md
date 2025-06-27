---
id: step.16.update‑batch
depends_on:
  - step.15.versioning
tags: [batch, step]
---

## Objective

Return `UpdateBatch{NewNodes, StaleNodeKeys}` from each commit.

## Implements

- **§6.2** end paragraph ("TreeUpdateBatch containing node\*batch + stale*node_index_batch").
  \_What happens*:

  - Counts newly minted vs stale NodeKeys, setting up later Pebble persistence.

## Done When ✓

- [ ] Batch counts equal nodes touched in commit test.
