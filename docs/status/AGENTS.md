# AGENTS.md — docs/status/ conventions

Conventions for status reports in this directory. A report is a **point-in-time
historical snapshot**: write it once, then only annotate it non-destructively
(`~~item~~ done at \`hash\``) — never rewrite, reorder, or delete content after
the fact. Reports with every numbered item resolved move to`archived/`.

This file **specializes the global status-report format for this repo**: where
the global skill and this file disagree, this file wins (it adds the coverage
delta line, the a–g section letters, and the archive rules on top of the
general skeleton). A copy-pasteable skeleton lives in
[`_template.md`](_template.md).

## File naming

`YYYY-MM-DD_HH-MM_slug.md` — session start time, kebab-case slug naming the
session's work, e.g. `2026-08-29_16-36_todo-list-full-execution-and-self-review.md`.

## Required structure

1. **Title:** `# Status Report — YYYY-MM-DD HH:MM — <Session Title>`
2. **Preamble:** 3–6 lines — session scope, final gate state (`scripts/verify.sh`
   result, `nix flake check` result), and the **mandatory coverage-delta line** (below).
3. **Optional `## TL;DR`:** 3–5 bullets, plain sentences, before section a).
   Recommended for sessions that ship releases or land many items; skip for
   small single-topic sessions where the preamble already says it all.
4. **Sections a–g** — omit empty sections, but keep the letters of the ones you keep:

| Section                                         | Content                                                                                                                                              |
| ----------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| `## a) FULLY DONE`                              | Table `\| # \| Work \| Evidence \|`. Every row cites file paths, test names, or gate output — never "looks done".                                    |
| `## b) PARTIALLY DONE`                          | What shipped AND what exactly remains — both halves, explicitly.                                                                                     |
| `## c) NOT STARTED`                             | Items that were in scope but never began, and why.                                                                                                   |
| `## d) TOTALLY FUCKED UP`                       | Honest mistakes made this session, numbered. Blunt beats polished.                                                                                   |
| `## e) WHAT WE SHOULD IMPROVE`                  | Process (not product) improvements, `IMP<n>` table with a priority column.                                                                           |
| `## f) Up to 50 things we should get done next` | Numbered, grouped by priority. This is the harvest source for `TODO_LIST.md` — forward-looking intent that lives only in a timestamped file is lost. |
| `## g) Questions I CANNOT figure out myself`    | Questions route to the user or `ROADMAP.md` "Open questions" — never into `TODO_LIST.md` (questions are not tasks).                                  |

## The mandatory coverage-delta line

Every report's preamble MUST carry exactly one bullet of this form:

```
- cover: library 99.3% (=), ssetest 97.2% (+0.5)
```

- **Measured this session** via `nix run .#coverage-gate` (or
  `go tool cover -func=<profile>`) — never quoted from memory or from a
  previous report (reports are snapshots; the number may have moved since).
- Both modules, root `sse` library and `ssetest`, each with a delta vs the
  previous report's line. `=` when unchanged.
- **Cross-repo sessions:** when the session materially changed another repo
  (e.g. go-datastar), add that repo's measured coverage as extra bullets in the
  same format (e.g. `- cover (go-datastar): datastartest 9x.x% (…)`), measured
  in that repo, not quoted. The go-sse line stays mandatory in every report.
- **Rationale:** the 2026-07-27 session shipped a 100% → 99.5% coverage
  regression unnoticed because coverage was never a report field. This line
  makes coverage regressions impossible to ship silently.

## After writing a report

- **Harvest section f)** into `TODO_LIST.md` (bounded, short-term items) and
  `ROADMAP.md` (vague, long-term ones). Verify each item against code first —
  many "next tasks" are already done by a later session.
- Completed work goes to `CHANGELOG.md` `[Unreleased]` — done items never stay
  in `TODO_LIST.md`.
- When every numbered item in a report carries a resolution marker, `git mv`
  the file into `archived/`.
