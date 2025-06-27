---
id: step.21.metrics
depends_on:
  - step.18.persist‑multi‑ver
tags: [metrics, step]
---

## Objective

Expose Prometheus counter `jmt_commits_total`.

## Implements

- **Non‑Functional Goals – Flexibility & Observability** (wasn't named but implied).
  _What happens_:

  - Real‑time counter supports ops dashboards and mirrors spec's idea of "prometheus metrics appear at /metrics" (original 10‑step plan, step 09).

## Done When ✓

- [ ] `/metrics` shows counter > 0 after tests.
