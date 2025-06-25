---
id: nfr.portability.build
tags: [portability]
---

## Description

The project must build and test on common developer workstations without extra toolchains.

## Default Target

Metric: `go test ./...` succeeds on Linux, macOS, and Windows using Go ≥ 1.22; no CGO.

## Measurement

GitHub Actions matrix job for each OS runs `go vet`, `go test -race`.

## Rationale

Lower barrier to entry encourages outside contributors and wider adoption.
