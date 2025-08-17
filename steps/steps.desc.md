# 🧩 Steps Guide

This guide defines **how the `steps/` folder drives code‑generation work**. Each file represents a _single, compilable change_ that an LLM agent (or human) can complete end‑to‑end. Ongoing maintenance and refactoring live elsewhere; **Steps track only forward‑moving implementation tasks.**

---

## 1  Folder layout

```
steps/
├─ 01-bootstrap.md        <- current step (lowest number)
├─ 02-add-auth.md         <- draft (next in line)
├─ 03-ui-polish.md        <- draft (future)
├─ 999-roadmap.md         <- parking lot for high‑level roadmap items
└─ archive/
      ├─ 00-init.md        <- completed steps live here
      └─ 01-bootstrap.md   <- once 01 is finished it is moved here
```

| Path             | Purpose                                                                              |
| ---------------- | ------------------------------------------------------------------------------------ |
| `steps/`         | Home for **all** implementation steps, kept in strict numeric order.                 |
| `steps/archive/` | Mandatory resting place for every step after its code is merged.                     |
| `999-roadmap.md` | Broad, non‑granular roadmap. Break items out into numbered steps before work begins. |

---

## 2  File‑naming convention

1. **Zero‑padded sequence:** `01-`, `02-`, … keeps steps naturally ordered and defines execution order.
2. **Short slug:** a few words summarising the change (`add-auth`, `refactor-db-layer`).
3. **Implicit active step:** the **lowest‑numbered** file in `steps/` is considered _active_. No additional suffix or status is required.

Examples:

- `07-integrate-payment.md` ⬅️ current step if all lower numbers are archived
- `08-clean-up-error-handling.md`  ⬅️ draft (next in line)

> **Sequential guarantee**: A step can only begin once all lower‑numbered steps have moved to `archive/`.

---

## 3  Mandatory YAML front‑matter

```yaml
---
id: step.07-integrate-payment # globally unique (required)
depends_on:
  - spec.payment-api # upstream spec
  - step.06-db-migrations # previous step
tags: [payments]
---
```

| Field        | Required | Notes                                         |
| ------------ | -------- | --------------------------------------------- |
| `id`         | ✅       | Prefix with `step.` and keep globally unique. |
| `depends_on` | ➖       | Enforce ordering and traceability if needed.  |
| `tags`       | ➖       | Free‑form labels for search/filter tooling.   |

---

## 4  Lifecycle workflow

| Stage        | Human action                                                            | Filename            | Location         |
| ------------ | ----------------------------------------------------------------------- | ------------------- | ---------------- |
| **Draft**    | Write the next step description.                                        | `NN-new-feature.md` | `steps/`         |
| **Active**   | When all lower numbers are archived, this becomes active automatically. | _unchanged_         | `steps/`         |
| **Complete** | Merge code & tests; move file to `steps/archive/`.                      | _unchanged_         | `steps/archive/` |

> **Unbroken chain rule**: Only the **lowest‑numbered** file in `steps/` is considered _active_. All higher‑numbered files must remain drafts until their turn.

---

## 5  Writing a good step

- **Scope:** Something an agent can finish and compile in isolation (≈ 50–150 LOC).
- **Context:** Summarise only the relevant requirements/specs—link, don’t paste.
- **Acceptance criteria:** Define clear tests or checks. Example:

```md
## Objective

Implement <concise goal> …

## Tasks

- [ ] Code
- [ ] Unit tests
- [ ] Update docs

## Acceptance Criteria

- <clear, measurable checks>
```

---

## 6  Parking lot & roadmap (Step 999)

Create a high sequence number like `999-roadmap.md` for a broad, non‑granular roadmap. Use checklist bullets or sub‑headings for future work. Break each item out into a numbered step before it becomes actionable.

---

## 7  Automated enforcement (TODO)

A forthcoming linter will ensure:

1. Numbers are contiguous up to the highest draft step, with no gaps.
2. The **lowest‑numbered** file in `steps/` exists and is the only active task.
3. All completed steps live in `steps/archive/`.
4. `depends_on` references resolve.

---
