---
id: instructions-tech_spec_generation
---

# ========== TECH-SPEC PROMPT TEMPLATE ==========

**Context**

You are given one or more _authoritative_ texts (white-papers, RFCs, blog posts, code, internal memos, …)
about **{{PROJECT_NAME}}**.
Your goal is to produce a single, high-level technical specification that distills the _what_ and _why_—
explicitly deferring the _how_ to later iterations (mark these with <!-- TODO: ... -->).

**Input**

- Canonical sources (list them here with title + URL or path):
  1. …
  2. …
  3. …

**Required Output**

Create a Markdown file named **{{project_slug}}\_spec.md** containing _exactly_ these
top-level headings (keep the order):

1. Executive Summary
2. Scope & Assumptions
3. Core Concepts
4. High-Level Data Model & Operations
5. Architecture Overview _(include an ASCII or PlantUML stub)_
6. Non-Functional Goals
7. Open Questions & TODOs
8. Reference Mapping _(table linking each concept to snippet IDs)_
9. Appendix A – Source Snippets

**Source-Snippet Rules**

- Extract 1- to 3-sentence excerpts that best support each key idea.
- Assign IDs `[SN-001]`, `[SN-002]`, … and **reference them inline** like
  `(see [SN-007])`.
- Each snippet must include:
  - the verbatim quotation
  - a citation (URL + section/page or file + line range)
  - a 1-line note on which concept it supports.
- Group snippets by source in Appendix A for traceability.

**Style & Formatting**

- Clear, active voice; ≤ 120 characters per line.
- Prefer bullet lists; keep code to ≤ 5-line pseudocode when truly helpful.
- Defer deep implementation material with `<!-- TODO: ... -->`.
- Ensure all links and snippet IDs resolve (run markdownlint).
- No extension nodes? No problem—drop them if irrelevant!

**Acceptance Checklist**

✔ All nine headings present and populated.
✔ Every major concept in § 3–6 cites ≥ 1 snippet.
✔ Reference Mapping table cross-links concepts and snippets with no broken anchors.
✔ File passes markdownlint (no dead links, headings rule, etc.).
✔ Output filename is **{{project_slug}}\_spec.md**.

**Helpful Tips for the LLM (not part of final spec)**

- Summarise—don’t copy—outside Appendix A.
- If two sources say the same thing, prefer the clearer quote.
- Use PlantUML rather than mermaid; easier for downstream tooling.
- When in doubt, ask: “Does this help a future code-gen pass?” If not, defer.

# ===============================================
