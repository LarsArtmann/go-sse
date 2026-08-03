# DataStar Integration — Comprehensive Execution Plan

**Created:** August 3, 2026, 00:25
**Source:** Status report `2026-08-03_00-18_datastar-integration-keyed-lines-and-self-review.md`
**Scope:** ALL TODOs from the self-review, split into tasks ≤12 min each.

---

## Sorting Criteria

| Field      | Values  | Meaning                                                                                      |
| ---------- | ------- | -------------------------------------------------------------------------------------------- |
| **Pri**    | P0 → P3 | P0 = correctness/release blocker, P1 = high value now, P2 = medium, P3 = speculative/ROADMAP |
| **Impact** | H/M/L   | How much it improves DataStar usability                                                      |
| **Cust**   | H/M/L   | Customer-facing value (adoption, DX, trust)                                                  |
| **Effort** | min     | Estimated minutes (max 12 per task)                                                          |
| **Dep**    | —       | Task ID this depends on (blank = none)                                                       |

---

## Phase 0: Blocked (Waiting on User Answers)

These cannot start until the 3 questions from the status report are answered.

| ID  | Task                                                                | Pri | Impact | Cust | Effort | Dep | Blocked By |
| --- | ------------------------------------------------------------------- | --- | ------ | ---- | ------ | --- | ---------- |
| B1  | Decide: DataStar subpackage (`datastar/`) vs core-only              | —   | —      | —    | 0      | —   | **Q1**     |
| B2  | Decide: Release version (0.3.0 minor vs 0.2.2 patch)                | —   | —      | —    | 0      | —   | **Q2**     |
| B3  | Decide: Example uses `templ` vs raw HTML strings                    | —   | —      | —    | 0      | —   | **Q3**     |
| B4  | Create `datastar/` subpackage scaffold (`datastar.go`)              | P1  | H      | H    | 8      | B1  | Q1         |
| B5  | Add `PatchElements` typed builder in subpackage                     | P1  | H      | H    | 10     | B4  | Q1         |
| B6  | Add `PatchSignals` typed builder in subpackage                      | P1  | H      | H    | 8      | B4  | Q1         |
| B7  | Add DataStar event-name constants (`PatchElements`, `PatchSignals`) | P1  | M      | M    | 3      | B1  | Q1         |
| B8  | Add `IsDataStarRequest(r *http.Request) bool` helper                | P2  | M      | M    | 5      | B1  | Q1         |
| B9  | Add `ReadDataStarSignals(r, &v)` — parse `datastar` param/body      | P2  | M      | H    | 10     | B1  | Q1         |
| B10 | Add view-transition data-line support (`useViewTransition true`)    | P2  | L      | M    | 5      | B5  | Q1         |
| B11 | Add settle-duration data-line support                               | P2  | L      | M    | 5      | B5  | Q1         |
| B12 | Document all DataStar merge modes (morph/inner/outer/prepend/…)     | P2  | L      | M    | 8      | B1  | Q1         |
| B13 | Add `datastar` build tag for optional subpackage                    | P3  | L      | L    | 5      | B4  | Q1         |
| B14 | Add DataStar protocol version constant                              | P3  | L      | L    | 3      | B1  | Q1         |
| B15 | Add `example/datastar-templ/` — DataStar + templ rendering          | P3  | M      | M    | 10     | B3  | Q3         |
| B16 | Cut release (tag, release notes, GitHub release)                    | P0  | H      | H    | 10     | B2  | Q2         |

---

## Phase 1: Immediate Fixes (Correctness & Honesty)

Self-review items E1–E5. No dependencies, do first.

| ID  | Task                                                                       | Pri | Impact | Cust | Effort | Dep |
| --- | -------------------------------------------------------------------------- | --- | ------ | ---- | ------ | --- |
| 1   | Fix `KeyedLines` Grow calc — named constant for separator width            | P0  | L      | L    | 3      | —   |
| 2   | Add CRLF handling mention to `KeyedLines` doc comment                      | P1  | L      | M    | 3      | —   |
| 3   | Fix README DataStar example — use raw string literal with real `\n`        | P1  | M      | H    | 5      | —   |
| 4   | Investigate `gopls stdversion` warning at `stream.go:129` (`json.Marshal`) | P2  | L      | L    | 10     | —   |

---

## Phase 2: Release Blockers (Documentation Completeness)

| ID  | Task                                                                 | Pri | Impact | Cust | Effort | Dep |
| --- | -------------------------------------------------------------------- | --- | ------ | ---- | ------ | --- |
| 5   | Add CHANGELOG.md `[Unreleased]` — `KeyedLines` + `SendLines` entries | P0  | M      | H    | 5      | —   |
| 6   | Update TODO_LIST.md — add DataStar follow-up items from this plan    | P0  | L      | M    | 5      | —   |
| 7   | Update ROADMAP.md — add DataStar to "Realized in" callout            | P1  | L      | L    | 5      | —   |

---

## Phase 3: Testing Completeness

| ID  | Task                                                                     | Pri | Impact | Cust | Effort | Dep |
| --- | ------------------------------------------------------------------------ | --- | ------ | ---- | ------ | --- |
| 8   | Add `FuzzKeyedLines` — panic-safety with arbitrary key/value             | P0  | M      | M    | 8      | —   |
| 9   | Add `BenchmarkKeyedLines` — single-line + multi-line (10/100/1000 lines) | P1  | L      | L    | 8      | —   |
| 10  | Add DataStar integration test — HTTP round-trip, assert exact wire bytes | P0  | H      | H    | 10     | —   |
| 11  | Test `KeyedLines` with empty key — define behavior (error vs ` value`)   | P1  | M      | M    | 5      | —   |
| 12  | Add `FuzzKeyedLines` seed corpus — real HTML fragments (#31)             | P2  | L      | L    | 5      | 8   |
| 13  | Add `BenchmarkSendLines` — variadic join vs manual concat (#32)          | P2  | L      | L    | 8      | —   |
| 14  | Add KeyedLines to broadcaster benchmark — fan-out load (#49)             | P3  | L      | L    | 8      | 9   |

---

## Phase 4: API Completeness (Core)

| ID  | Task                                                                     | Pri | Impact | Cust | Effort | Dep |
| --- | ------------------------------------------------------------------------ | --- | ------ | ---- | ------ | --- |
| 15  | Add `WriteKeyedLines(w, eventType, key, value)` — wire-only helper impl  | P1  | H      | H    | 8      | —   |
| 16  | Add `WriteKeyedLines` tests — single/multi/empty/CRLF                    | P1  | H      | H    | 8      | 15  |
| 17  | Add `Stream.SendKeyed(eventName, key, value)` — shorthand for single-key | P2  | M      | M    | 5      | —   |
| 18  | Add `SendKeyed` tests                                                    | P2  | M      | M    | 5      | 17  |
| 19  | Add `KeyedLinesBuilder` — fluent `.Add(key, value).String()`             | P2  | M      | H    | 10     | —   |
| 20  | Add `KeyedLinesBuilder` tests — empty, single, multi, CRLF               | P2  | M      | H    | 8      | 19  |
| 21  | Add `KeyedLinesMulti(map[string]string)` (#17 from list)                 | P3  | L      | M    | 8      | —   |
| 22  | Profile `KeyedLines` with 10KB+ HTML — verify Grow sufficiency (#18)     | P2  | L      | L    | 10     | 1   |

---

## Phase 5: API Completeness (General SSE, not DataStar-specific)

| ID  | Task                                                                    | Pri | Impact | Cust | Effort | Dep |
| --- | ----------------------------------------------------------------------- | --- | ------ | ---- | ------ | --- |
| 23  | Add `Event.DataLines() []string` — accessor for split data (#21)        | P3  | L      | L    | 5      | —   |
| 24  | Add `Event.WithID(id)` / `Event.WithRetry(ms)` builders (#42)           | P3  | L      | M    | 8      | —   |
| 25  | Add `Event.Validate()` — wire-format safety check (#35)                 | P2  | M      | M    | 8      | —   |
| 26  | Add `WriteEventBytes(evt) []byte` — return bytes, no write (#36)        | P3  | L      | L    | 5      | —   |
| 27  | Add `Stream.SendRaw(bytes)` — zero-alloc pre-serialized send (#37)      | P3  | L      | L    | 8      | —   |
| 28  | Add `Stream.SendLinesf(eventName, format, args...)` (#19)               | P3  | L      | L    | 5      | —   |
| 29  | Add `Stream.SetRetry(ms)` — standalone retry frame through stream (#41) | P3  | L      | L    | 5      | —   |

---

## Phase 6: DataStar Examples

| ID  | Task                                                                  | Pri | Impact | Cust | Effort | Dep |
| --- | --------------------------------------------------------------------- | --- | ------ | ---- | ------ | --- |
| 30  | Create `example/datastar.go` — server structure + SSE handler         | P1  | H      | H    | 10     | —   |
| 31  | Create `example/datastar.go` — HTML page with DataStar JS client      | P1  | H      | H    | 10     | 30  |
| 32  | Manually test `example/datastar.go` — point browser, verify DOM patch | P1  | H      | H    | 5      | 31  |
| 33  | Create `example/datastar-signals/` — reactive state example (#46)     | P3  | M      | M    | 10     | 30  |
| 34  | Create `example/datastar-reconnect/` — Last-Event-ID replay (#47)     | P3  | M      | M    | 10     | 30  |

---

## Phase 7: Real-World Verification

| ID  | Task                                                                       | Pri | Impact | Cust | Effort | Dep |
| --- | -------------------------------------------------------------------------- | --- | ------ | ---- | ------ | --- |
| 35  | Point real DataStar JS client at go-sse server — verify events parse (E4)  | P1  | H      | H    | 12     | 30  |
| 36  | Add CI job: headless browser test — DataStar client + example server (#30) | P3  | M      | M    | 12     | 35  |

---

## Phase 8: DataStar Deep Integration (Non-Blocked Subset)

These don't require the subpackage decision — they're helpers that could live in core or a subpackage.

| ID  | Task                                                                         | Pri | Impact | Cust | Effort | Dep |
| --- | ---------------------------------------------------------------------------- | --- | ------ | ---- | ------ | --- |
| 37  | Add `JSONSignals(map[string]any)` — marshal + KeyedLines("signals", …) (#23) | P2  | M      | H    | 8      | —   |
| 38  | Add `Broadcaster.BroadcastKeyed` — broadcast + KeyedLines composition (#33)  | P3  | L      | M    | 8      | —   |
| 39  | Add per-subscriber DataStar event filtering — route by selector (#34)        | P3  | M      | M    | 12     | —   |
| 40  | Add connection metadata — track per-stream topics (#38)                      | P3  | M      | M    | 12     | —   |
| 41  | Add graceful shutdown for DataStar — drain patch events (#39)                | P3  | L      | M    | 10     | —   |
| 42  | Add observability hooks — per-event-send metrics (#40)                       | P3  | L      | M    | 10     | —   |

---

## Phase 9: Documentation & Ecosystem

| ID  | Task                                                                   | Pri | Impact | Cust | Effort | Dep |
| --- | ---------------------------------------------------------------------- | --- | ------ | ---- | ------ | --- |
| 43  | Document DataStar retry/reconnect semantics in doc.go (#22)            | P2  | L      | M    | 8      | —   |
| 44  | Add migration guide — official DataStar Go SDK → go-sse (#48)          | P2  | M      | H    | 10     | —   |
| 45  | Add CONTRIBUTING.md section — how to test DataStar compatibility (#50) | P3  | L      | L    | 5      | —   |

---

## Summary Statistics

| Metric                       | Value               |
| ---------------------------- | ------------------- |
| Total tasks (excl. blocked)  | 45                  |
| Total tasks (incl. blocked)  | 61                  |
| Total effort (excl. blocked) | ~370 min (~6.2 hrs) |
| Total effort (incl. blocked) | ~470 min (~7.8 hrs) |
| P0 tasks                     | 8                   |
| P1 tasks                     | 14                  |
| P2 tasks                     | 13                  |
| P3 tasks                     | 16                  |
| Blocked tasks                | 16                  |

---

## Recommended Execution Order

### Wave 1 — Immediate (P0, no deps, ~45 min)

`1 → 5 → 6 → 8 → 10`

### Wave 2 — High Value (P1, no deps, ~75 min)

`2 → 3 → 7 → 11 → 15 → 16 → 30 → 31 → 32`

### Wave 3 — Verification (P1, high impact, ~12 min)

`35`

### Wave 4 — Medium Value (P2, no deps, ~100 min)

`4 → 9 → 13 → 17 → 18 → 19 → 20 → 22 → 25 → 37 → 43 → 44`

### Wave 5 — Polish (P3, no deps, ~115 min)

`12 → 14 → 21 → 23 → 24 → 26 → 27 → 28 → 29 → 33 → 34 → 38 → 45`

### Wave 6 — Advanced (P3, high effort, ~58 min)

`36 → 39 → 40 → 41 → 42`

### Wave 7 — Blocked (after Q1/Q2/Q3 answered)

`B1–B16` (16 tasks, ~100 min)

---

## Legend

- **ID**: Stable task identifier for tracking
- **(#N)**: References item number from the original 50-item list in the status report
- **(E#)**: References self-review item E1–E5 from the status report

---

## Resolution notes (added 2026-08-03, post-ROADMAP restructure)

**Task 7 (moot):** "Update ROADMAP.md — add DataStar to 'Realized in' callout."
The "Realized in" callout pattern was removed entirely in the 2026-08-03 ROADMAP
restructure (completed-work history now lives exclusively in CHANGELOG.md). This
task will never execute. No replacement needed — CHANGELOG `[Unreleased]` already
documents the DataStar additions.

**Plan executed in Waves 1–4** (commits `7cbb01d` through `c8ae9a0`). Shipped:
`KeyedLines`, `SendLines`, `WriteKeyedLines`, `SendKeyed`, `FuzzKeyedLines`,
`BenchmarkKeyedLines`, `TestIntegration_DataStarWireFormat`, `example/datastar/`,
migration guide. Phase 0 blocked tasks resolved: Q1 = no subpackage (core-only),
Q2 = v0.3.0 tagged empty (real release is v0.4.0), Q3 = raw HTML. YAGNI-rejected:
`JSONSignals`, `KeyedLinesBuilder`, `KeyedLinesMulti`, `Event.DataLines`,
`WithID`/`WithRetry`, `Validate`, `WriteEventBytes`, `SendRaw`, `SendLinesf`,
`SetRetry` — no consumer has asked for these.

**Still open:** browser verification of example (TODO_LIST); CI headless browser
test (TODO_LIST, blocked).
