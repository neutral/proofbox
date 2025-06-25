# 📚 Blueprint

The `blueprint/` directory is the **authoritative source** for every requirement, decision, and prompt that an LLM—or a human—needs to generate, review, and maintain code. It captures product intent at a high level and refines it into machine‑executable instructions.

---

## Folder map

- **`global/goals.md`**
  Captures business and product goals—zero technical detail.

- **`global/<category>/<nfr>/`**
  Groups **non‑functional requirements (NFRs)** by category (e.g., `security`, `performance`, `compliance`).
  Each NFR folder contains:

  - `requirement.md` — the NFR statement.
  - `spec/` — supporting technical artifacts (API contracts, data schemas, sequence diagrams, proofs‑of‑concept, etc.).

- **`global/glossary.md`**
  A living dictionary of every term used across the project.

- **`features/<category>/<feature>/`**
  Groups **functional requirements** by product domain.
  Each feature folder contains:

  - `requirement.md` — the functional requirement.
  - `spec/` — supporting technical artifacts.

- **`steps/`**
  The implementation plan, decomposed into granular, compilable tasks so the LLM can focus within context limits.

  - **File naming**: prefix each file with a zero‑padded sequence number (`01-`, `02-`, …) followed by a short slug.
  - **Active step**: exactly one file carries the suffix `.active.md`. Agent tooling surfaces only this file.
  - **History**: completed steps are renamed to remove `.active.md` and may be moved to `steps/archive/` to reduce noise.

- **`prompts/`**
  Prompt templates fed into code‑generation tools.

- **`decisions/`**
  Architecture Decision Records (ADRs) explaining why a particular option was chosen when alternatives existed.

- **`archive/`**
  Optional parking lot for all deprecated or superseded Blueprint artifacts. Moving files here is encouraged—but not required—once their `status` is set to `deprecated`.

---

## Document flow

1. **Goals** set the strategic direction.
2. **Requirements**—functional or non‑functional—translate those goals into capabilities.
3. **Specifications** detail the behaviors and data required to satisfy each requirement.
4. **Steps** turn specs into bite‑sized implementation tasks for the agent.
5. **Decisions** capture the rationale behind architectural choices.

---

## Mandatory YAML front‑matter

All Blueprint files are Markdown and begin with a concise YAML block so agents and tools can resolve dependencies without parsing the entire document.

```yaml
---
id: spec.jmt-node_key # unique slug (required)
status: draft | active | deprecated # lifecycle state (required for Steps; optional elsewhere)
depends_on:
  - goal.auditability # upstream goal
  - req.export-inclusion-proofs # linked requirement
tags: [jmt, merkle, storage]
---
```

**Fields**

| Field        | Required | Purpose                                           |
| ------------ | -------- | ------------------------------------------------- |
| `id`         | ✅       | Globally unique slug used for cross‑references.   |
| `status`     | ➖       | Track lifecycle: `draft`, `active`, `deprecated`. |
| `depends_on` | ➖       | Explicit upstream links for traceability.         |
| `tags`       | ➖       | Arbitrary labels for search/filter tooling.       |

---

## Step lifecycle & archiving

1. Create a new step file with the next number and suffix `.active.md` (e.g., `05-integrate-auth.active.md`).
2. When the step is complete, change `status:` to `active`, remove the `.active.md` suffix, and optionally move the file to `steps/archive/`.
3. Start the next step by repeating step 1.

This workflow keeps the agent laser‑focused on a single task while preserving full history for humans.

---

## Quick‑Start Checklist

1. Clone the repo and open `blueprint/global/goals.md` to understand top‑level intent.
2. Explore `features/` and `global/` folders to see how requirements are organized.
3. Locate the current active step in `steps/`.
4. Follow any instructions in that step to contribute code or docs.
5. When adding a new doc, copy the relevant template and fill in the YAML front‑matter.
