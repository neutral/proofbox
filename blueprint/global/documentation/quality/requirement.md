---
id: nfr.documentation.quality
tags: [documentation]
---

## Description

Documentation must be clear, complete, and co‑located with code.

## Default Target

Metric:

- README passes `markdownlint`
- Every exported Go symbol has a top‑level doc string
- Code file is commented well
- Each source file in the codebase is accompanied by a `.desc.md` file with the same filename capturing a high level description of what was acheived in the source file.

## Measurement

`golangci-lint` plus `markdownlint-cli` run in CI and fail on violations.

## Rationale

Good docs cut onboarding time and reduce support burden.
