## Repo Structure

### Description Files

This repo uses description files which have `.desc.md` in the file name. You can understand information that concerns any file or folder and what its purpose is by reading the corresponding description file when present.

A source file needs to both have code comments as well as a corresponding description file.

### Blueprint files

Blueprint files is the authoritative source for goals, requirements, specs, and decisions — for both humans and agents. ProofBox colocates blueprint docs directly next to the code they describe, while keeping global non‑functional requirements in a central package blueprint.

Overview

- Colocated feature docs: `_blueprint` folders live beside implementation packages.
- Global NFRs: shared non‑functional concerns live in `pkg/_blueprint`.

Folder Map

- Functional requirements: `<pkg-or-cmd>/.../_blueprint/<feature>/...`
  Example: `pkg/storage/_blueprint/get/requirement.md`
  Typical structure:

  - `requirement.md` — feature requirement
  - `_specs/` — design notes, API contracts, data formats, diagrams
  - `_decisions/` (optional) — ADRs local to the feature/package

- Global NFRs: `pkg/_blueprint/<domain>/<topic>/...`
  Examples: `pkg/_blueprint/observability/metrics/`, `pkg/_blueprint/performance/latency/`, `pkg/_blueprint/reliability/crash‑safety/`

  - `requirement.md` — non-functional requirement
  - `_specs/` — design notes, API contracts, data formats, diagrams
  - `_decisions/` global ADRs

Document Flow

1. Goals and requirements define intent (feature or NFR).
2. Specs refine behavior, data, and interfaces to satisfy requirements.
3. Decisions explain trade‑offs and chosen designs.

YAML Front‑Matter (All Blueprint Docs)

```yaml
---
id: req.storage.put-commit # globally unique slug
status: draft | active | deprecated # optional except for steps
depends_on:
  - goal.proofbox
  - nfr.performance.throughput
tags: [storage, api]
---
```

Fields

- id: globally unique slug for cross‑reference and tooling.
- status: lifecycle tracking (commonly used by steps; optional elsewhere).
- depends_on: upstream links (goals, requirements, other specs).
- tags: free‑form labels for filtering and search.

Authoring Guidelines

- New feature: create `<package>/_blueprint/<feature>/` with `requirement.md` and optional `_specs/`.
- New NFR: add under `pkg/_blueprint/<domain>/<topic>/` with `requirement.md` and optional `_specs/`.
- Keep requirements concise; put detail in `_specs/` with examples and interfaces.
- Prefer colocated ADRs when a choice affects only that area; use global ADRs for cross‑cutting decisions.
