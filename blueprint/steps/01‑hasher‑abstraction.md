---
id: step.01.hasher‑abstraction
depends_on:
  - req.storage.put‑commit
  - nfr.security.integrity
tags: [crypto, step]
---

## Objective

Introduce a `Hasher` interface and a default SHA‑256 implementation so that all future code depends on an abstraction, not the concrete algorithm.

## Done When ✓

- [ ] `pkg/hasher/hasher.go` defines the interface and `DefaultSHA256`.
- [ ] Unit test (`hasher_test.go`) passes and ensures `DefaultHash == SHA256([]byte{})`.
- [ ] `go vet` and `go test ./...` green.
- [ ] Step document archived (`git mv` to `steps/archive/`) once merged.

## Implementation Hints

1. Interface with methods `Digest(data ...[]byte) [32]byte` and `Concat(hashes ...[32]byte) [32]byte`.
2. Provide a zero‑alloc fast path for one or two inputs.
3. Add a tiny benchmark to check < 50 ns per 32‑byte digest.
4. No Pebble interaction yet.
