---
id: nfr.maintainability.test‑coverage
tags: [maintainability]
---

## Description

A strong safety net is required for refactors and external PRs.

## Default Target

Metric: global statement coverage ≥ 80 % (`go test -cover`).

## Measurement

CI gate fails the build if coverage drops below threshold.

## Rationale

High coverage reduces regressions and increases contributor confidence.
