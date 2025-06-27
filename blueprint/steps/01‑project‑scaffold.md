---
id: step.01.project‑scaffold
depends_on: []
tags: [setup, step]
---

## Objective

Create Go module `github.com/acme/jmt` with an empty `go test ./...` CI run.

## Implements

Establishing the Go module and a "green" `go test ./...` run satisfies the **Scope & Assumptions** clause that "this spec can guide an automated code‑generation phase for implementations in Rust **or other languages**." Before any JMT‑specific logic exists, a deterministic build pipeline is needed to guarantee repeatability and CI hooks (§Architecture Overview, final paragraph).

## Done When ✓

- [ ] `go test ./...` passes with no packages.
