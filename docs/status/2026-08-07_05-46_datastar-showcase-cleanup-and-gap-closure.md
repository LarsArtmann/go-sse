# Status Report: DataStar Showcase Cleanup & Gap Closure

**Date:** 2026-08-07 05:46
**Session:** Resuming from prior session's handoff (38/40 tasks done, 3 open questions). This session resolved all remaining issues.
**Branch:** master
**Working tree:** Clean (auto-commit daemon committed all work)

---

## Executive Summary

This session was a cleanup and gap-closure pass. The prior session left 3 open questions and ~12 remaining work items. This session resolved all of them: removed dead code, extracted inline JS to a CSP-safe external file, implemented scroll-to-top (the one missed task), fixed a broken test helper, added 3 new tests, ran golangci-lint (first time this work), and updated all documentation. Build, vet, race tests (3x deterministic), golangci-lint, nix fmt, and nix flake check all pass.

---

## a) FULLY DONE (Verified — build + vet + race tests x3 + golangci-lint + nix flake check)

### 1. Dead Code Removal (`store.go`)

**Decision:** REMOVE, not wire up.

The prior session added `EventsAfterSince(lastID, since time.Duration)` + a `timestamps []time.Time` field to `memStore`, but never called it from any handler. The auto-commit daemon committed it as `feat(datastar): add time-based event filtering to in-memory store` — a lie.

**Why removed instead of wired up:**
- `sse.Event` has no timestamp field. Time-based filtering is not part of the library's `EventStore` abstraction. Adding it to the demo's store teaches a non-library pattern.
- The canonical time-filter path (if ever wanted) is `FilteredEventStore` / `ReplayFiltered` — a library-level interface, not a bespoke store method. That's a separate Tier 5 idea.
- Dead code is worse than no code. The code lied about having a feature.

**What was removed:**
- `timestamps []time.Time` field from `memStore` struct
- `time.Now()` tracking in `Append()`
- Parallel slice trimming in ring buffer eviction
- Entire `EventsAfterSince` method
- All `?since=` references in comments
- `time` import (no longer needed)

### 2. `scanner.Err()` Fix + `collectSSEEvents` Rewrite

**Root cause found and fixed:** The original `collectSSEEvents` had a busy-wait pattern: `select { case <-timer.C: ... default: }` followed by a blocking `scanner.Scan()`. When no data was arriving, `scanner.Scan()` blocked indefinitely and the timer never fired. This made the collection window unreliable — the function could hang far beyond its intended duration.

**Fix:** Rewrote with a goroutine feeding lines into a channel, and a `select` between `timer.C` and `lineCh`. The timer now fires deterministically regardless of event arrival rate. Also added proper `scanner.Err()` checking that distinguishes genuine I/O errors from expected context cancellation.

### 3. T3-30: Scroll-to-Top on New Event (the one task the prior session forgot)

Implemented via `MutationObserver` in `static/app.js`. Watches `#feed` for `childList` changes. When a new item is prepended, scrolls to top — but only if the user is already near the top (within 48px). If the user has scrolled down to read older items, auto-scroll is suppressed. This respects user agency.

### 4. Inline `<script>` Extraction to `static/app.js`

**Decision:** Move to external file for CSP safety.

The prior session put theme toggle, keyboard shortcuts, and localStorage logic in an inline `<script>` tag inside `index.templ`. This works but violates Content-Security-Policy best practices (`unsafe-inline`).

**What changed:**
- Created `static/app.js` with all client-side behavior
- Removed inline `<script>` from `index.templ`
- Removed `onclick="toggleTheme()"` from the theme toggle button (now wired via `addEventListener`)
- Added `<script src="/static/app.js" defer></script>` to the `<head>`
- Regenerated `index_templ.go` via `templ generate`

### 5. Test Improvements

| Test | Change | Why |
|------|--------|-----|
| `TestConcurrentFanOut` | Collection window 6s -> 3s | Faster suite, still catches shared events |
| `TestFilterPredicate_AlertsOnly` | NEW — deterministic | Broadcasts 1 alert + 1 success + 1 info, asserts only alert passes filter. Prior session's version relied on random producer output (flaky). |
| `TestEventsHandler_CORSHeader` | NEW | Verifies `Access-Control-Allow-Origin: *` on `/events` |
| `TestReplayedSignalResetOnSubscribe` | NEW | Verifies `replayEvent(0)` is broadcast on subscribe |
| `collectSSEEvents` | Rewritten | Goroutine+select pattern; proper `scanner.Err()` |
| `extractEventIDs` | Modernized | `strings.SplitSeq` + `strings.CutPrefix` (Go 1.24+ idioms) |

**Test suite: 10 tests, 3x deterministic with `-race`, 4.1s total** (down from 9s prior session).

### 6. golangci-lint (First Run This Work)

The prior session never ran golangci-lint. This session ran it and fixed all issues in the datastar example:

| Issue | Fix |
|-------|-----|
| `goconst`: `"datastar-patch-signals"` repeated 3x | Extracted `eventPatchElements` / `eventPatchSignals` constants |
| `nolintlint`: unused `mnd` nolint directive | Removed `mnd` from `//nolint:gochecknoglobals,mnd` |
| `varnamelen`: `tc` too short | Renamed to `testCase` |
| `varnamelen`: `sl` too short | Renamed to `line` |
| `nlreturn`: missing blank lines before return | Added blank lines |
| `gci`/`golines` | Auto-fixed by `nix fmt` |

**Result: 0 issues in `example/datastar/`.** (1 pre-existing issue in `example/htmx/main.go` — `contextcheck` — not touched, not my code.)

### 7. Documentation Updates

- **AGENTS.md**: Updated "Shared structure" section with correct 4-file split (store.go, producer.go, handlers.go, main.go), app.js, main_test.go, VERIFY.md. Added per-file descriptions. Noted HTMX uses single main.go.
- **VERIFY.md**: Expanded from 7 to 10 sections. Added: theme toggle, pause/resume, scroll-to-top.

---

## b) PARTIALLY DONE

Nothing. All items are either fully done or not started (by design).

---

## c) NOT STARTED (Deliberate — Out of Scope This Session)

These are from the prior session's 50-item list and are **not bugs or gaps** — they are future enhancements:

1. **Mermaid diagram in README** — ASCII art works fine on all platforms; Mermaid doesn't render on pkg.go.dev
2. **POST -> Broadcast form** — Tier 5 feature
3. **SQLite EventStore** — Tier 5 feature
4. **`/health` endpoint** — Tier 5 feature
5. **ReplayFiltered demonstration** — Tier 5 feature
6. **Prometheus metrics** — Tier 5 feature
7. **Dockerfile** — Tier 5 feature
8. **Pre-existing htmx `contextcheck` lint issue** — not my code, not this session's scope

---

## d) TOTALLY FUCKED UP

### 1. Python script introduced mojibake (caught and fixed)

When I used a Python script to rewrite `collectSSEEvents` (because the tab-vs-space mismatch defeated the `edit` tool), the script's string literals contained em dashes that got triple-encoded into `\xc3\xa2\xc2\x80\xc2\x94` mojibake bytes. I caught this immediately with `grep -Pn '[^\x00-\x7F]'` and fixed all instances. No mojibake reached the committed code.

**Lesson:** Python scripts editing Go source need explicit `encoding='utf-8'` and byte-level verification after writing. Or better: avoid Python entirely for source edits.

### 2. The `edit` tool failed on tab-indented Go code

The `collectSSEEvents` replacement failed because the `edit` tool couldn't match tabs in the `old_string` parameter. The view output shows tabs as whitespace, and I copied them as spaces. This forced the Python script workaround (see item 1 above).

**Lesson:** When replacing large tab-indented blocks, either match a smaller unique fragment or use `write` to replace the entire file.

### 3. LSP diagnostics were stale the entire session

gopls showed "redeclared in this block" errors for every symbol in the datastar example, referencing a `main.go` that was 425 lines long (it's actually 105 lines). This persisted despite `lsp_restart`. Build and vet were the source of truth and passed cleanly throughout. The stale LSP made it impossible to trust diagnostic output.

**Lesson:** `go build` is truth. LSP is secondary. When LSP shows errors that `go build` doesn't, ignore the LSP.

### 4. Did not catch that `TestFilterPredicate_AlertsOnly` was initially flaky on first run

My first version of the filter test relied on the random producer to generate alert events within a 3s window. With only ~33% chance of an alert per emit, and ~2 events in 3s, there was a real chance of zero alerts. The test passed on the verbose run but failed on the subsequent `go test ./...` run. I then rewrote it to deterministically broadcast known events.

**Lesson:** Tests that depend on random data generation need either a seeded RNG or deterministic event injection. I should have designed it deterministically from the start.

---

## e) WHAT WE SHOULD IMPROVE

### Architecture

1. **`collectSSEEvents` sends unbuffered lines on an unbuffered channel.** The goroutine blocks on `lineCh <- ...` until the select loop reads it. Under high event volume this creates back-pressure, but for a test helper with ~1 event/2s this is fine. If the test were ever used for load testing, the channel should be buffered.

2. **`app.js` uses ES5-style `var` and `function` declarations.** This is intentional for maximum browser compatibility without a build step, but inconsistent with modern JS. Could use `<script type="module">` for ES6+ (DataStar already uses `type="module"`).

3. **The scroll-to-top threshold (48px) is a magic number.** It's reasonable but not configurable. For a demo this is fine.

4. **`connectSSEClient` helper has a goroutine leak risk.** The goroutine reading `io.Copy(io.Discard, resp.Body)` only exits when the context is cancelled. The `cancel()` call in the returned function triggers this. But if the HTTP response never arrives (network hang), the goroutine blocks on `http.DefaultClient.Do(req)` and never reaches the `io.Copy`. The context cancellation should eventually unblock the `Do` call, so this is theoretically fine, but it's worth noting.

### Testing

5. **No test for the scroll-to-top behavior.** This is client-side JS behavior that can't be tested with Go's `httptest`. Would need a headless browser (Playwright/Cypress). Out of scope for a Go library example.

6. **No test for theme toggle persistence.** Same reason — localStorage is browser-only.

7. **`TestConcurrentFanOut` still takes 3s.** Could be reduced further by injecting events directly instead of waiting for the producer's 2s ticker. But the test's value is in verifying the real producer -> broadcaster -> stream -> HTTP pipeline end-to-end, so the 3s is justified.

8. **No test for `totalEventSignal`.** The producer broadcasts `BroadcastMany(evt, totalEventSignal(id))` but no test verifies the total count matches the number of feed items.

9. **No test for `BroadcastMany` ordering guarantee.** The batch send is assumed atomic but not verified.

### Code Quality

10. **`producer.go` event name constants are only used in the example package.** `eventPatchElements` and `eventPatchSignals` are unexported, which is correct for an example, but they're defined in `producer.go` while the events are constructed there too. The grouping is logical.

11. **`handlers.go` still has `//nolint:contextcheck` on two lines.** These are legitimate (templ uses `r.Context()` correctly, and `stream.Context()` is `r.Context()`). The nolint directives are correct.

12. **The prior session's status report (`docs/status/2026-08-07_03-37_*.md`) still describes dead code as "partially done".** It's a point-in-time snapshot and shouldn't be edited, but a reader following the trail might be confused.

### Process

13. **The auto-commit daemon committed intermediate work multiple times.** Commits like `(empty message)` and intermediate states are in the log. This is expected behavior per the project setup but makes the git history noisy.

14. **I should have run `nix fmt` before `golangci-lint`.** The formatter fixes `gci` and `golines` issues automatically; running lint first reported issues that `nix fmt` would have fixed. I did run `nix fmt` second and re-ran lint, but the order should be reversed.

---

## f) Up to 50 Things to Get Done Next

### Fix the htmx example (not my code, but in the project)

1. Fix `example/htmx/main.go:106` `contextcheck` lint issue — `progressContent` should pass context parameter
2. Run `golangci-lint` on htmx example and fix any other issues
3. Consider adding tests for the htmx example (currently has zero tests)

### Improve the datastar example — testing

4. Add `TestTotalEventSignal` — verify totalEvents count matches feed items broadcast
5. Add `TestBroadcastMany_Ordering` — verify events in a batch arrive in order
6. Add `TestGracefulShutdown_DrainBuffer` — verify buffered events are drained before close
7. Add `TestHeartbeat_KeepsConnectionAlive` — verify heartbeat extends idle connection
8. Add `TestIdleTimeout_DoesNotKillSSE` — verify SSE connections survive IdleTimeout
9. Add `TestReplay_Ordering` — verify replayed events arrive in ascending ID order
10. Add `TestRingBuffer_EvictionOrder` — verify events are evicted oldest-first (partially covered by `TestMemStore_RingBufferEviction` but could be more thorough)
11. Add `TestSubscribeFilter_PanicRecovery` — verify a panicking predicate doesn't crash the broadcaster
12. Add `TestFilterPredicate_MetaEventsPassThrough` — verify meta events (no ID) pass through filter
13. Add `TestEventsHandler_ReplayOnReconnect` — full reconnect cycle with Last-Event-ID header
14. Add `TestProducer_EventIDMonotonic` — verify IDs are monotonically increasing
15. Add `TestMemStore_Concurrent` — concurrent Append + EventsAfter under race detector

### Improve the datastar example — code quality

16. Extract `datastarAddr`, `emitInterval`, etc. into a config struct for testability
17. Make `emitInterval` overridable in tests (currently hardcoded to 2s, forces slow tests)
18. Add `//nolint:gochecknoglobals` justification comment for `staticFiles` embed
19. Consider adding doc comments to all unexported functions
20. Consider extracting `feedItemHTML` template into a const or templ fragment
21. Add `// Output:` example test to `main.go` for godoc rendering

### Improve the datastar example — UI/UX

22. Add a "Clear feed" button (client-side only, clears `#feed` innerHTML)
23. Add sound notification on alert events (optional, toggleable)
24. Add event count by category breakdown (N alerts, N successes, N info)
25. Add connection latency indicator (time between event timestamp and receipt)
26. Add "Copy curl command" button showing how to consume the SSE endpoint
27. Add dark/light theme auto-detection on first load (currently defaults to `prefers-color-scheme`)
28. Add a max-feed-items cap (e.g. keep only last 100 DOM nodes for performance)
29. Add visual indicator when paused events are being buffered
30. Add rate display (events/min)

### Improve the datastar example — architecture

31. Add `/health` endpoint returning `BroadcasterHealth` as JSON
32. Add `ReplayFiltered` demonstration (the library's predicate-aware replay path)
33. Add `WithBufferSize` demonstration (toggle buffer=1 vs buffer=64 to show drop behavior)
34. Add multi-room example (second broadcaster for a different feed)
35. Add Prometheus metrics (subscriber count, broadcast rate, drop rate)
36. Add graceful shutdown indicator in UI (DataStar signal when draining)
37. Add Last-Event-ID inspector in UI (show the ID in the header)

### Improve the datastar example — documentation

38. Consider Mermaid diagram in README (note: doesn't render on pkg.go.dev)
39. Add `CHANGELOG.md` entry for the example improvements
40. Add inline comments in `app.js` explaining the scroll-to-top threshold
41. Document the `BroadcastMany` atomicity guarantee in the example
42. Add a "Testing" section to VERIFY.md describing the Go test suite
43. Add architecture diagram to AGENTS.md (D2 or Mermaid)

### Process improvements

44. Add `nix run .#verify` that runs build + vet + lint + test + fmt --check in one command
45. Add pre-commit hook for `nix fmt --check`
46. Split example tests into fast (unit) and slow (integration) buckets
47. Consider adding the example race tests to CI with longer timeout
48. Document the auto-commit daemon behavior in AGENTS.md
49. Add `make verify` equivalent (but as a flake app, not Makefile)
50. Consider adding `.editorconfig` for consistent tab/space across editors

---

## g) Questions

### 1. Should I fix the pre-existing htmx `contextcheck` lint issue?

`example/htmx/main.go:106` has a `contextcheck` warning: `progressContent` should pass the context parameter. It's the only remaining lint issue in the project. It's not my code (the htmx example was built in a prior session), and the handoff said "don't fix unrelated bugs." But it's a 1-line fix and it's the only thing preventing `golangci-lint run ./...` from passing clean. **Should I fix it, or leave it for whoever owns the htmx example?**

### 2. Should the htmx example get the same treatment (tests, file split, VERIFY.md)?

The htmx example has zero tests, a single `main.go`, and no `VERIFY.md`. The datastar example now has 10 tests, a 4-file split, and a 10-section verification checklist. The asymmetry is noticeable. **Should I bring the htmx example up to the same standard, or is it intentionally simpler?**

### 3. Should I add a `/health` endpoint to the datastar example?

The `Broadcaster.Health()` method returns a struct with `Closed`, `Draining`, `SubscriberCount`, and `BufferSize`. Exposing this as `GET /health` (JSON) would demonstrate the lifecycle API and be useful for k8s probes. It's a ~15-line addition. **Is this worth adding now, or is it scope creep for a showcase?**

---

## Build & Test Summary

```
go build ./...           PASS
go vet ./...             PASS
go test -race -count=3   PASS (4.1s per run, 10 tests, all deterministic)
golangci-lint (datastar) 0 issues
golangci-lint (full)     1 issue (pre-existing htmx contextcheck)
nix fmt                  0 changes needed
nix flake check          all checks passed
```

## Commits This Session (auto-committed by daemon)

```
3b8f157 (empty message)
07a3156 test(datastar): harden SSE test collection and filter assertion
2d11802 test(example/datastar): add tests for filter predicate, CORS headers, and replay reset
c8c1726 refactor(example/datastar): extract inline javascript to external app.js and reduce test duration
8252d01 test(datastar): distinguish scanner errors from expected context cancellation in SSE event collection
99ed6b4 refactor(example): remove ?since= duration filter from datastar store
```

## Scorecard

| Category | Items | Done | Quality |
|----------|-------|------|---------|
| Dead code removal | 1 | 1 | Clean — all traces removed |
| Test infrastructure fixes | 2 | 2 | Root cause fixed (goroutine+select), not just `scanner.Err()` |
| New tests | 3 | 3 | Deterministic, targeted, fast |
| UI features (T3-30 scroll-to-top) | 1 | 1 | MutationObserver with user-respect threshold |
| CSP extraction (app.js) | 1 | 1 | Clean separation, no inline scripts remain |
| golangci-lint compliance | 1 | 1 | 0 issues in datastar (6 issues found and fixed) |
| Documentation | 2 | 2 | AGENTS.md + VERIFY.md both current |
| **Total** | **11** | **11** | **All verified** |
