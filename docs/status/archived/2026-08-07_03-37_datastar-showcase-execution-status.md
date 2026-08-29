# Status Report: DataStar Showcase Improvement Plan Execution

**Date:** 2026-08-07 03:37\
**Session:** Executing the 40-task Pareto plan from `docs/planning/archived/2026-08-07_00-50_SUPERB-datastar-showcase-improvement-plan.html`\
**Branch:** master\
**Working tree:** Clean (auto-commit daemon committed everything)

---

## a) FULLY DONE (Verified — build + vet + race tests pass)

### Tier 0 — Blockers & Correctness (7/7 tasks)

| # | Task                                                                                | State                                                     |
| - | ----------------------------------------------------------------------------------- | --------------------------------------------------------- |
| 1 | `nix flake check` — full hermetic build+test                                        | ✅ Passed                                                 |
| 2 | `nix fmt` — treefmt (gofumpt, goimports, golines)                                   | ✅ Passed (0 changes at start; 5 files reformatted later) |
| 3 | Refactored `feedItemEvent` — added `category {cat}` structured data line            | ✅ Removes fragile HTML-string dependency                 |
| 4 | Updated filter predicate — `strings.Contains(evt.Data, "category "+categoryAlert)`  | ✅ No longer greps rendered CSS classes                   |
| 5 | Build + vet after filter refactor                                                   | ✅                                                        |
| 6 | Concurrent multi-tab test harness — 2 goroutines collecting SSE via `bufio.Scanner` | ✅                                                        |
| 7 | Fan-out assertions — verifies shared event IDs between client A and B               | ✅ PASSING with `-race`                                   |

### Tier 1 — High-Impact (8/8 tasks)

| #  | Task                                                                                          | State |
| -- | --------------------------------------------------------------------------------------------- | ----- |
| 8  | Reset `$replayed` signal to 0 before count — `BroadcastMany(replayEvent(0), countEvent(...))` | ✅    |
| 9  | memStore test: empty store EventsAfter                                                        | ✅    |
| 10 | memStore test: non-integer / unknown ID (4 subtests)                                          | ✅    |
| 11 | memStore test: valid sequential replay                                                        | ✅    |
| 12 | memStore test: ring buffer cap & eviction                                                     | ✅    |
| 13 | Graceful shutdown test — broadcaster.Shutdown + Health().Closed check                         | ✅    |
| 14 | Browser verification checklist — `example/datastar/VERIFY.md`                                 | ✅    |
| 15 | Document subscriber count TOCTOU race in OnSubscribe comment                                  | ✅    |

### Tier 2 — Code Quality (8/8 tasks)

| #  | Task                                                                                | State                                                      |
| -- | ----------------------------------------------------------------------------------- | ---------------------------------------------------------- |
| 16 | Document memStore O(n) scan as demo-only                                            | ✅ Inline comment in `EventsAfter`                         |
| 17 | Extract `store.go` — memStore struct + methods                                      | ✅                                                         |
| 18 | Extract `producer.go` — activityItem, msgTemplates, generators, event builders      | ✅                                                         |
| 19 | Extract `handlers.go` — activityServer, handlers                                    | ✅ main.go is now ~80 lines (constants + main + embed)     |
| 20 | Verify build + lint after file split                                                | ✅                                                         |
| 21 | Add `IdleTimeout` (5min) to http.Server                                             | ✅ With comment explaining SSE vs ReadTimeout/WriteTimeout |
| 22 | Extract magic numbers — `nodeCount`, `serviceCount`, `endpointCount`, etc.          | ✅ All template generators use named constants             |
| 23 | Replay banner fade-out animation — CSS `bannerFade` keyframes (4s ease-in forwards) | ✅                                                         |

### Tier 3 — UI Polish (11/12 tasks)

| #  | Task                                                                                          | State                            |
| -- | --------------------------------------------------------------------------------------------- | -------------------------------- |
| 24 | Empty-state message ("Waiting for events…") — `data-show="$totalEvents === 0 && !$paused"`    | ✅                               |
| 25 | Total event count stat — `$totalEvents` signal + `totalEventSignal()` + "events sent" display | ✅                               |
| 26 | Connection status indicator — pulsing green `.live-dot` with CSS `pulse` animation            | ✅                               |
| 27 | Favicon — inline SVG lightning bolt via `data:image/svg+xml`                                  | ✅ No 404                        |
| 28 | Open Graph meta tags — `og:title`, `og:description`, `og:type`                                | ✅                               |
| 29 | Replay banner fade-out CSS (deduped with #23)                                                 | ✅                               |
| 30 | **Scroll-to-top on new event**                                                                | ❌ NOT STARTED                   |
| 31 | Pause button — `$paused` signal + Pause/Resume toggle + "Paused" banner                       | ✅                               |
| 32 | Timestamp filter (`?since=5m`)                                                                | ⚠️ PARTIALLY DONE (see section b) |
| 33 | CORS headers — `Access-Control-Allow-Origin: *` on /events                                    | ✅                               |
| 34 | Dark/light theme toggle — `data-theme` attribute + localStorage persistence                   | ✅                               |
| 35 | Keyboard shortcuts — `a` for alerts, `e` for all events                                       | ✅                               |

### Tier 4 — Documentation (5/5 tasks)

| #  | Task                                               | State                                                              |
| -- | -------------------------------------------------- | ------------------------------------------------------------------ |
| 36 | "Try It" 60-second checklist in README             | ✅ 5 numbered steps                                                |
| 37 | "How It Works" data flow (ASCII diagram) in README | ✅ producer → store → broadcaster → subscriber → stream → browser  |
| 38 | Feature mapping table in README                    | ✅ 8 rows mapping behavior → go-sse API                            |
| 39 | Shutdown behavior documentation in README          | ✅ 4-step sequence                                                 |
| 40 | Inline architecture comments in handlers.go        | ✅ Comments on NewStream, Replay, SubscribeFilter, Heartbeat, Send |

### Bonus (not in original plan)

- **Responsive CSS** — `@media (max-width: 640px)` breakpoint for mobile
- **Filter hint** — "press 'a' for alerts" text in filter bar

---

## b) PARTIALLY DONE

### T3-32: Timestamp filter (`?since=5m`) — DEAD CODE

**What was done:**

- Added `timestamps []time.Time` field to `memStore` struct
- Updated `Append()` to record `time.Now()` alongside each event
- Updated ring buffer eviction to trim timestamps in parallel with events
- Added `EventsAfterSince(lastID, since time.Duration)` method to store.go

**What was NOT done:**

- The `eventsHandler` in `handlers.go` **does not parse `?since=`**
- The `eventsHandler` **does not call `EventsAfterSince`**
- The live-event subscription path also doesn't filter by timestamp
- The `index.templ` has no UI for the timestamp filter

**Impact:** `EventsAfterSince` is **dead code** — it compiles, it's tested by nothing, and no code path calls it. The `timestamps` slice is maintained but never read outside of `EventsAfterSince`. This is a lie in the codebase: the code implies the feature exists, but it doesn't work.

**Fix:** Either wire it up (parse `?since=5m`, call `EventsAfterSince` in handler, add UI) or remove the dead code entirely.

---

## c) NOT STARTED

| Task                              | Why                                                                                               |
| --------------------------------- | ------------------------------------------------------------------------------------------------- |
| T3-30: Scroll-to-top on new event | Never attempted. Would need JS `scrollTop = 0` or DataStar `data-on` expression after feed patch. |

---

## d) TOTALLY FUCKED UP

### 1. Dead code: `EventsAfterSince` + timestamps tracking

**Severity: Medium.** The code compiles and all tests pass, but the feature is a phantom. A reader seeing `EventsAfterSince` and the `?since=` comment would assume timestamp filtering works. It doesn't. This violates the "names tell the whole truth" principle.

**The auto-commit daemon even committed this with message:** `feat(datastar): add time-based event filtering to in-memory store` — which is a lie. The store has the method, but the feature doesn't work end-to-end.

### 2. `bufio.Scanner` error check missing in test

**Severity: Low.** The `collectSSEEvents` helper in `main_test.go` uses `scanner.Scan()` in a loop but never checks `scanner.Err()`. If the scanner hits a buffer limit or I/O error, the test silently returns partial results instead of failing. gopls flags this as a warning.

### 3. LSP diagnostics completely stale for the entire session

**Severity: Development friction (not a code issue).** After splitting `main.go` into 4 files, gopls never recovered. It showed "redeclared in this block" errors for every symbol that was moved, despite the build passing cleanly. This made every edit show false errors and required mentally filtering out noise. The `lsp_restart` call didn't fix it. The code is correct; the LSP cache was poisoned.

### 4. No test for the timestamp filter

Even if the dead code were wired up, there's no test for `EventsAfterSince`. The memStore tests cover `EventsAfter` but not the time-based variant.

---

## e) WHAT WE SHOULD IMPROVE

### Code Quality

1. **Remove or wire up `EventsAfterSince`** — Dead code is worse than no code. Either complete the feature or delete it.
2. **Check `scanner.Err()`** in `collectSSEEvents` test helper — silent data loss in tests is dangerous.
3. **Test takes 8 seconds** — `TestConcurrentFanOut` uses a 6-second collection window. Could reduce to 3s with faster emit interval in test setup.
4. **memStore tests don't use `Append`** — `makeTestEvent` bypasses the store, so `timestamps` are never populated in tests. If `EventsAfterSince` were tested, it would fail.
5. **AGENTS.md not updated** — The file structure section still describes the old single-file `main.go`. Should document the 4-file split (store.go, producer.go, handlers.go, main.go).
6. **No `golangci-lint` run** — Only `go vet` was run. The project has golangci-lint configured in flake.nix but it was never invoked this session.
7. **DataStar `<script>` tags in templ** — The inline `<script>` for theme toggle and keyboard shortcuts uses raw JS, not DataStar expressions. This works but is inconsistent with the DataStar-first approach of the rest of the template.

### Architecture

8. **The `$paused` signal is client-side only** — Events are still received from the server and patched into the DOM (just hidden via `display:none`). On resume, all buffered items appear at once. This is correct behavior but could surprise users expecting a true pause.
9. **The theme toggle uses `data-theme` attribute** — A more idiomatic approach would use DataStar signals + CSS custom properties, but the current approach is simpler and works.
10. **README diagram is ASCII art** — Could use a Mermaid diagram for better rendering on GitHub/pkg.go.dev.

### Testing

11. **No test for the filter predicate** — The `category alert` filter is the #1 correctness fix (Tier 0), but there's no unit test for it. The concurrent test connects unfiltered.
12. **No test for CORS headers** — The `Access-Control-Allow-Origin` header was added but never verified.
13. **No test for `$replayed` reset** — The `BroadcastMany(replayEvent(0), countEvent(...))` fix is not tested.
14. **Shutdown test doesn't test the drain** — It calls `Shutdown` but doesn't verify that buffered events are drained before close.

---

## f) Up to 50 Things to Get Done Next

### Fix the fuckups (do these first)

1. Wire up `?since=` timestamp filter in `eventsHandler` OR remove `EventsAfterSince` + `timestamps` from store.go
2. Add `scanner.Err()` check in `collectSSEEvents` test helper
3. Run `golangci-lint run ./...` and fix any issues
4. Add unit test for the `category alert` filter predicate
5. Add test for CORS header on `/events`

### Complete the remaining plan task

6. T3-30: Scroll-to-top on new event (JS `feed.scrollTop = 0` after patch, or DataStar animation)

### Improve tests

7. Reduce `TestConcurrentFanOut` from 6s to 3s collection window
8. Add filter predicate test — connect with `?filter=alerts`, assert only alert events arrive
9. Add test for `$replayed` signal reset (BroadcastMany sends replayEvent(0))
10. Add test for `BroadcastMany` ordering guarantee
11. Add test for `EventsAfterSince` (if feature is kept)
12. Add test for `totalEventSignal` — verify count matches feed items
13. Add test for graceful shutdown drain (buffer empty before close)
14. Add test for heartbeat keeping connection alive
15. Add test for `IdleTimeout` not killing SSE connections

### Improve code quality

16. Update AGENTS.md with 4-file structure (store.go, producer.go, handlers.go, main.go)
17. Update VERIFY.md with new UI features (pause, theme toggle, event count, keyboard shortcuts)
18. Replace ASCII art diagram in README with Mermaid diagram
19. Move inline `<script>` JS to a separate file in static/
20. Add `nolint` comments where appropriate (e.g., `gochecknoglobals` for msgTemplates is already there)
21. Consider extracting event name constants (`datastarPatchElements`, `datastarPatchSignals`)
22. Add doc comments to all exported functions in the example package

### Tier 5 features (from the plan — ideas, not committed)

23. POST → Broadcast form (let visitors send events)
24. Simple chat room example (user input → broadcast → all clients)
25. WebSocket comparison example (same feed over WS vs SSE)
26. SQLite EventStore (persistent replay store)
27. `/health` endpoint returning `BroadcasterHealth` as JSON
28. ReplayFiltered demonstration
29. WithBufferSize demonstration (toggle buffer=1 vs buffer=64)
30. Shutdown drain indicator in UI
31. Last-Event-ID inspector in UI
32. Animated connection diagram (live SVG)
33. BroadcastMany batch demo
34. Multi-room (second broadcaster)
35. Rate limiting on /events
36. Prometheus metrics (subscriber count, broadcast rate, drop rate)
37. Mobile responsive CSS (partially done — could add touch-friendly controls)
38. Dockerfile for example
39. TLS support
40. "Copy curl command" button
41. Sound notification on alerts
42. Gzip compression middleware

### Process improvements

43. Restart gopls before each session or after file operations
44. Run `golangci-lint` as part of every change verification, not just `go vet`
45. Add a `make verify` or `nix run .#verify` that runs build + vet + lint + test + fmt --check
46. Consider adding the example tests to CI (they currently run but take 9s with race detector)
47. Add pre-commit hook for `nix fmt --check`
48. Consider splitting example tests into fast (unit) and slow (integration) buckets
49. Document the auto-commit daemon behavior in AGENTS.md
50. Consider adding a `CHANGELOG.md` entry for the example improvements

---

## g) Questions (that I CANNOT figure out myself)

### 1. Should I wire up the `?since=` timestamp filter or remove the dead code?

The `EventsAfterSince` method + `timestamps` tracking is already in store.go and committed. Wiring it up would require:

- Parsing `?since=5m` in `eventsHandler`
- Calling `EventsAfterSince` instead of `EventsAfter` when the param is present
- Adding a UI element (dropdown or button)
- Adding tests

Removing it would mean reverting the `timestamps` field, the `Append` change, and the method itself. **Which path do you want?**

### 2. Should the inline `<script>` JS (theme toggle, keyboard shortcuts) move to a separate file?

The current approach puts raw JavaScript inside the templ template. This works but:

- It's not CSP-safe (inline scripts require `unsafe-inline` in Content-Security-Policy)
- It mixes concerns (templ is for HTML structure, not behavior)
- DataStar expressions can't handle localStorage or keyboard events natively

Moving to `static/app.js` would be cleaner but adds a network request. **Do you want CSP safety or minimal requests for this demo?**

### 3. Should I update AGENTS.md now, or is the auto-commit daemon going to fight me?

The auto-commit daemon committed my work multiple times this session with generated messages. If I update AGENTS.md now, the daemon may commit it before I finish. **Is there a way to pause the daemon, or should I just work around it?**

---

## Build & Test Summary

```
go build ./...     ✅ PASS
go vet ./...       ✅ PASS
go test -race ./... ✅ PASS (9.0s for example, 1.1s for library)
nix flake check    ✅ PASS
nix fmt            ✅ PASS (5 files reformatted on final run)
golangci-lint      ⚠️ NOT RUN
```

## Commits This Session (auto-committed by daemon)

```
a190e36 feat(datastar): add time-based event filtering to in-memory store  ← DEAD CODE
3faa092 (empty message)
14177a9 feat(example/datastar): add theme toggle, pause control, CORS, and live event counter
f60ac1a docs(example): document demo limitations and design decisions in datastar example
ba19100 test(datastar): improve test reliability and concurrent execution
c665db0 refactor(example/datastar): use category metadata for server-side replay filtering
c583972 docs(planning): add datastar showcase improvement plan  ← (previous session)
```

## Scorecard

| Category              | Planned | Done   | Partial | Missed | % Complete |
| --------------------- | ------- | ------ | ------- | ------ | ---------- |
| Tier 0 (Blockers)     | 7       | 7      | 0       | 0      | 100%       |
| Tier 1 (High-Impact)  | 8       | 8      | 0       | 0      | 100%       |
| Tier 2 (Code Quality) | 8       | 8      | 0       | 0      | 100%       |
| Tier 3 (Polish)       | 12      | 10     | 1       | 1      | 83%        |
| Tier 4 (Docs)         | 5       | 5      | 0       | 0      | 100%       |
| **Total**             | **40**  | **38** | **1**   | **1**  | **95%**    |

---

## Resolution (2026-08-29, docs-health pass)

Every open item in this report was closed by the same-day cleanup session (`2026-08-07_05-46`): §b's dead `EventsAfterSince` removed, §c's scroll-to-top shipped, §d's scanner.Err fixed, and §f 1–5 (wire-or-remove, scanner.Err, golangci-lint, filter test, CORS test) plus 6 (scroll-to-top) all done there. §f 7–22 done or closed in `05-46` §a (test speedup, filter/CORS/replayed tests, AGENTS/VERIFY updates, event constants, inline-script extraction); Mermaid → Won't (see 05-46 resolution). §f 23–50 future items → YAGNI/example-scope.
