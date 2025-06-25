---
id: nfr.documentation.quality
tags: [documentation]
---

## Description

Documentation must be clear, complete, and co‑located with code.

## Default Target

Metric: README passes `markdownlint`; every exported Go symbol has a top‑level doc string; docs/spec.md updated per release.

## Measurement

`golangci-lint` plus `markdownlint-cli` run in CI and fail on violations.

## Rationale

Good docs cut onboarding time and reduce support burden.
