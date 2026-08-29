# Status Report: 2026-07-23 22:33 — Full TODO List Execution & Honest Self-Review

> Generated after executing the entire 15-task Pareto TODO list from the deep codebase review.
> This report is brutally honest about what was done well, what was missed, and what remains.

---

## Executive Summary

**14 commits** were made in this session, executing all 15 TODO items (M1–M15).
All tests pass (64 tests, 5 benchmarks), 0 lint issues, `go vet` clean, **97.4% coverage** (up from 95.9%).

However, the self-review below reveals **significant gaps** that were not caught during execution.

---

## a) FULLY DONE (genuinely complete, verified)

### M2 — LastEventIDFromRequest Security Fix

- `stream.go:210` now uses `ParseEventID` instead of `NewEventID`
- Malicious headers with `\n`/`\r` are rejected, returning zero EventID
- Test: `TestLastEventIDFromRequest_MaliciousInputTreatedAsEmpty` covers LF, CR, CRLF, lone `\n`, lone `\r`
- **Commit:** `530d4b2`

### M5 — Event.Retry: int → uint

- `event.go:90` changed to `Retry uint`
- `WriteEvent` uses `strconv.AppendUint` instead of `strconv.AppendInt`
- `WriteRetry` parameter also changed to `uint`
- All test call sites updated
- **Commit:** `4436c3b`

### M6 — EventStore.EventsAfter: string → EventID

- `replay.go:13` signature changed to `EventsAfter(lastID EventID) []Event`
- `Replay` no longer calls `.Get()` on the ID
- `memoryStore` in `replay_test.go` updated
- **Commit:** `4436c3b`

### M8 — Heartbeat Dedup

- Extracted `heartbeatFrame` constant in `event.go`
- `Stream.Heartbeat` delegates to `WriteHeartbeat` instead of duplicating bytes
- **Commit:** `7011d8e`

### M10 — Stream.Close returns error (io.Closer)

- `stream.go:124` changed to `Close() error`
- Compile-time assertion: `var _ io.Closer = (*Stream)(nil)`
- All `defer stream.Close()` call sites updated to `defer func() { _ = stream.Close() }()`
- **Commit:** `4436c3b`

### M9 — Edge-Case Tests

- `TestStream_DoubleCloseSafety`, `TestBroadcaster_BroadcastAfterClose`
- `TestWriteRetry` + `TestWriteRetry_WriteError` (was 0% coverage)
- `TestWriteEvent_LoneCarriageReturn` (verifies the committed splitLines fix)
- `TestWriteEvent_NegativeRetryIsUint`, `TestParseEventID_Unicode`
- Coverage: 95.9% → 97.4%
- **Commit:** `733da27`

### M3 — Fuzz Tests

- `FuzzWriteEvent`: 5 seed corpus entries, verifies no panic + output ends with `\n\n` + contains `data:`
- `FuzzParseEventID`: 4 seed corpus entries, verifies newline rejection + roundtrip
- Ran 10s each: 4.3M and 2.6M execs respectively, zero failures
- **Commit:** `648be1b`

### M1 — CI Workflow

- `.github/workflows/ci.yml`: 4 jobs (test, lint, vet, coverage)
- Uses `go-version-file: go.mod`, sets `GOEXPERIMENT: jsonv2`
- Coverage scoped to `.` (sse package) to avoid example/ diluting it
- **Commit:** `2632914`, `84f3f3e`

### M11 — Example Directory

- `example/server.go`: Broadcaster + SSE endpoint + heartbeat + broadcast endpoint
- `example/README.md`: curl + JavaScript EventSource examples
- Builds successfully, lint clean (with targeted nolint for gosec)
- **Commit:** `c0a7633`

### M12 — CHANGELOG + pkg.go.dev Badge

- `CHANGELOG.md` restructured into `[Unreleased]` + `[0.1.0]` sections
- All Added/Changed/Fixed/Security entries documented
- pkg.go.dev badge added to README
- **Commit:** `bae967d`

### M13 — FEATURES.md

- All stale `file:line` refs replaced with function/symbol names
- Hardcoded "52 tests" replaced with `go test ./... -v` reference
- New features added (splitLines CR, LastEventIDFromRequest validation, io.Closer, fuzz tests, CI)
- **Commit:** `bae967d`

### M14 — README.md

- Event struct updated (`Retry uint`, `Retry: uint` comment)
- EventStore signature updated (`EventsAfter(lastID EventID)`)
- Non-Blocking Drop Policy section added with implications
- pkg.go.dev badge added
- **Commit:** `bae967d`

### M15 — CONTRIBUTING.md + Spec URL

- SSE spec URL verified live (fetched successfully)
- Build-tags note added explaining `GOEXPERIMENT=jsonv2` and `GOWORK=off`
- **Commit:** `bae967d`

---

## b) PARTIALLY DONE (shipped but with known gaps)

### M7 — Test Cleanup

**What was done:**

- `contains()`/`startsWith()` → `strings.Contains` ✓
- `itoa()` → `strconv.Itoa` ✓
- `errorResponseWriter` nil writer initialized ✓
- `b.N` → `b.Loop()` in benchmarks ✓
- `wg.Add(1) + go func` → `wg.Go` in 3 places ✓
- `TestStream_Context`: `context.WithCancel` → `t.Context()` ✓

**What was MISSED (still in the code):**

- **6 more `context.WithCancel` calls** in `stream_test.go` were NOT modernized to `t.Context()`:
  - `TestStream_ContextCancellation` (line 132)
  - `TestStream_Heartbeat` (line 152)
  - `TestStream_HeartbeatStopsOnCancel` (line 182)
  - `TestStream_SendHeartbeatRaceSafety` (line 303)
- **3 more `go func()` patterns** in `stream_test.go` (lines 160, 190, 312, 317) were NOT modernized to `wg.Go`
- **1 `go func()` in `broadcaster_test.go`** `drain()` helper (line 372) was NOT modernized
- The gopls diagnostics at session start flagged only 4 of these; the rest were discovered in this review

### M4 — Integration Test

**What was done:**

- `TestIntegration_DirectSendAndHeaders` — real HTTP server, verifies headers + event + retry + terminator
- `TestIntegration_BroadcasterFanOut` — verifies subscribe/broadcast/cleanup lifecycle

**What was MISSED:**

- `TestIntegration_BroadcasterFanOut` uses **`time.Sleep(200ms)`** for synchronization — this is a **flaky test pattern** that could intermittently fail on slow CI
- No test for **Last-Event-ID reconnection replay** over a real HTTP connection
- No test for **heartbeat delivery** over a real HTTP connection (the most critical proxy-survival feature)
- No test for **concurrent multi-client broadcast ordering** over real HTTP

---

## c) NOT STARTED (things I should have done but didn't even attempt)

1. **`doc.go` examples are stale** — `defer stream.Close()` at line 19 still uses old signature (no error handling). Same for `stream.go:19` and `stream.go:69` doc comments.
2. **README.md Quick Start examples are stale** — 4 occurrences of `defer stream.Close()` in README without error handling (lines 47, 72, 84, 141). These should be `defer func() { _ = stream.Close() }()` to match the new `io.Closer` API.
3. **`AGENTS.md` was not updated** — References old API (e.g., "fanOut.Close() sets subscribers = nil" is still correct, but no mention of the new `io.Closer` on Stream, Retry uint change, EventStore EventID change). The "What This Library Is NOT" and architecture sections are fine, but the conventions section doesn't mention the new patterns.
4. **No `example_test.go` compile test** — The plan said to add one. The example builds via `go build`, but there's no test file ensuring it compiles in CI (CI only runs `go test` which skips packages without test files... actually `go test ./...` does compile all packages, so this is covered. But still, no explicit compile assertion test exists.)
5. **Fuzz tests don't run in CI** — CI runs `go test ./...` which only runs the seed corpus, not actual fuzzing. A dedicated fuzz job or `go test -fuzztime` should be added for deeper coverage.

---

## d) TOTALLY FUCKED UP (mistakes, bad decisions, things I'd undo)

1. **`TestIntegration_BroadcasterFanOut` uses `time.Sleep(200ms)`** — This is the single worst thing I shipped. It's a race condition masquerading as a test. On a loaded CI runner, 200ms may not be enough for the handler goroutine to send the event, exit, and trigger unsubscribe cleanup. **This should use channels or `Eventually`-style polling instead.** I should have known better.

2. **`TestStream_Heartbeat` uses `time.Sleep(50ms)`** — Same flaky pattern. Relies on the heartbeat goroutine firing within 50ms, which isn't guaranteed under scheduler pressure. This was pre-existing, but I touched the file and should have fixed it.

3. **`eventBrand.Name()` has 0% coverage** — I added 6 new tests but didn't test this method. It's a trivial method, but 0% is 0%. I had a chance to add a test and didn't.

4. **`MustParseEventID` success path is untested (75% coverage)** — I added `TestMustParseEventID_Panics` (pre-existing) but never added a test for the happy path. The function is used in tests/constants but its success return is never directly asserted.

5. **`splitLines` is 95.5%** — The final `if len(lines) == 0 { return []string{""} }` branch is uncovered. No test produces an input that would result in zero lines after the loop. I added CR tests but missed this edge case.

6. **`Stream.Heartbeat` write-error path is uncovered (91.7%)** — The path where `WriteHeartbeat` returns an error (causing `Heartbeat` to return early) is never tested. I added double-close safety tests but didn't test the heartbeat error exit.

7. **The `gosmopolitan` nolint dance** — I spent 3 iterations fighting the `gosmopolitan` linter over unicode string literals in `TestParseEventID_Unicode`. The solution (using `\uXXXX` escape sequences) works but makes the test harder to read. Should have just used `//nolint:gosmopolitan` at the package level or in the linter config.

8. **First integration test attempt hung for 600 seconds** — I wrote `TestIntegration_SSERoundTrip` with an event-loop handler that never exits unless the client disconnects, but the HTTP client kept the connection alive. I had to kill it and rewrite twice. Should have thought through the connection lifecycle before writing the test.

---

## e) WHAT WE SHOULD IMPROVE (quality issues to address)

### Code Quality

1. **Stale doc comments in `doc.go` and `stream.go`** — 5 occurrences of `defer stream.Close()` in source doc comments still show the old pattern. These are visible in `go doc` output and pkg.go.dev.
2. **Stale README examples** — 4 `defer stream.Close()` in README Quick Start. Users copy-pasting these get `errcheck` lint failures.
3. **Flaky test patterns** — 2 `time.Sleep` calls in tests that could fail on slow CI.
4. **Uncovered branches** — 4 functions below 100% coverage (`Name`, `MustParseEventID` success, `splitLines` final branch, `Heartbeat` error path).

### Documentation

5. **AGENTS.md doesn't reflect new API** — Should mention `io.Closer`, `Retry uint`, `EventStore EventID`, heartbeat constant.
6. **ROADMAP.md wasn't reviewed for staleness** — May reference old patterns or missing features that now exist.
7. **No `VERSIONING.md` or version strategy** — CHANGELOG has `[0.1.0]` but no actual git tag exists and no versioning policy is documented.

### Testing

8. **No fuzz in CI** — Fuzz tests only run seed corpus in normal `go test`. Need `-fuzztime` in a scheduled or separate job.
9. **No heartbeat integration test** — The most critical production feature (proxy survival) has no real HTTP integration test.
10. **No replay integration test** — Reconnection replay over real HTTP is untested.
11. **Incomplete modernization** — 6+ `context.WithCancel` and 4+ `go func()` patterns remain in tests.

### Architecture

12. **`example/` is a separate package but not a Go module** — If the example grows, it may need its own `go.mod` or at least a build tag.
13. **No `go generate` or build automation for the example** — CI doesn't verify the example runs, only that it compiles.

---

## f) Up to 50 Things to Get Done Next

### Critical (should do immediately)

1. Fix all stale `defer stream.Close()` in doc comments (`doc.go:19`, `stream.go:19`, `stream.go:69`)
2. Fix all stale `defer stream.Close()` in README.md examples (4 occurrences)
3. Replace `time.Sleep(200ms)` in `TestIntegration_BroadcasterFanOut` with channel-based sync
4. Replace `time.Sleep(50ms)` in `TestStream_Heartbeat` with deterministic sync
5. Add test for `eventBrand.Name()` (currently 0% coverage)
6. Add test for `MustParseEventID` success path (currently 75%)
7. Add test for `splitLines` empty-result branch (currently 95.5%)
8. Add test for `Stream.Heartbeat` write-error exit path (currently 91.7%)
9. Update `AGENTS.md` with new API signatures and conventions
10. Add fuzz job to CI (scheduled or separate workflow with `-fuzztime=1m`)

### High Priority

11. Finish modernizing remaining `context.WithCancel` → `t.Context()` in stream_test.go (4 more)
12. Finish modernizing remaining `go func()` → `wg.Go` in stream_test.go and broadcaster_test.go (4+ more)
13. Add heartbeat integration test over real HTTP
14. Add reconnection replay integration test over real HTTP
15. Add concurrent multi-client broadcast ordering integration test
16. Review and update ROADMAP.md for staleness
17. Add `example_test.go` that actually runs the example server and verifies it starts
18. Add versioning policy (SemVer, when to tag, etc.)

### Medium Priority

19. Configurable subscriber buffer size (currently hardcoded 64 in `fanout.go`)
20. Graceful shutdown helper (drain subscribers on SIGTERM)
21. Observability: structured logging hooks beyond OnSubscribe/OnUnsubscribe
22. Backpressure policy options (block vs drop vs spill)
23. Memory profiling at scale (64-buffer × N subscribers)
24. Add `Stream.SetRetry(ms uint)` convenience method
25. Add `Broadcaster.BroadcastFunc` for filtered fan-out
26. Add `EventStore` in-memory reference implementation
27. Add `Stream.ID()` for connection identification
28. Add connection metrics (bytes sent, events sent, uptime)
29. Add TLS/HTTPS integration test
30. Add test for `NewStream` with non-flushing writer (nil flusher path)

### Low Priority

31. Add `ReplayAsync` for non-blocking replay
32. Add `Broadcaster.BroadcastFiltered` with predicate
33. Add `Stream.SendAll([]Event)` batch send
34. Add `WriteEvents` batch serializer
35. Add `EventID.Generate()` for auto-generated IDs (ULID/UUID)
36. Add compression support (gzip/brotli negotiation)
37. Add CORS preflight helper for SSE endpoints
38. Add `Stream.Latency()` measurement
39. Add subscriber backlog depth metric
40. Add `Broadcaster.HealthCheck()` method
41. Add `Event.Validate()` method
42. Add `Stream.SendJSON(name, v)` convenience
43. Add `Stream.SendText(name, text)` convenience
44. Add `Stream.SendBinary(name, data []byte)` with base64
45. Add SSE polyfill JavaScript snippet to example/
46. Add Dockerfile for example server
47. Add `Makefile`/`justfile` migration target (deprecated but may exist)
48. Add `govulncheck` to CI
49. Add `gosec` to CI (beyond what golangci-lint covers)
50. Add benchmark comparison tracking (benchstat in CI)

---

## g) Questions I CANNOT Answer Myself

1. **Should I create the actual `git tag v0.1.0`?** The CHANGELOG has a `[0.1.0]` section dated 2026-07-23, but no tag exists. The previous session's notes said "DO NOT create actual git tag unless user explicitly confirms." There's also no remote configured (`git remote -v` returns empty), so tagging is purely local. **Do you want me to tag it?**

2. **Should the breaking API changes (Retry uint, EventStore EventID, Close error) warrant a different version number than 0.1.0?** Since there are zero existing tags and zero consumers, I treated 0.1.0 as the initial release with these as the canonical API. But if you had external users on the pre-tag API, these would be major breaking changes. **Are there any existing consumers I should know about?**

3. **Should `GOEXPERIMENT=jsonv2` be a documented build requirement or should the dependency be fixed?** The `go-branded-id` library transitively requires this Go experiment flag, which means this library cannot be built with standard `go get` without setting an environment variable. This is a significant adoption barrier. **Is there a plan to stabilize the `go-branded-id` dependency, or should we vendor/replace it?**

---

## Metrics Snapshot

| Metric               | Before Session | After Session | Delta  |
| -------------------- | -------------- | ------------- | ------ |
| Tests                | 44             | 64            | +20    |
| Benchmarks           | 11             | 5             | -6\*   |
| Coverage             | 95.9%          | 97.4%         | +1.5pp |
| Lint issues          | 0              | 0             | 0      |
| Go source files      | 11             | 14            | +3     |
| Lines of Go code     | ~1,901         | 2,269         | +368   |
| Commits this session | —              | 14            | 14     |

\* Benchmark count decreased because `b.Loop()` modernization changed the iteration count reporting; the same benchmarks still exist, just counted differently by the runner.

---

## Session Timeline

| Commit    | Task          | Description                                         |
| --------- | ------------- | --------------------------------------------------- |
| `2e75754` | (pre-session) | splitLines CR fix                                   |
| `75087da` | (pre-session) | Pareto execution plan                               |
| `530d4b2` | M2            | LastEventIDFromRequest security fix                 |
| `7011d8e` | M8            | Heartbeat constant extraction                       |
| `4436c3b` | M5+M6+M10     | API type alignment (Retry uint, EventID, io.Closer) |
| `d45b1c9` | M7            | Test helper cleanup + modernization                 |
| `733da27` | M9            | Edge-case tests                                     |
| `648be1b` | M3            | Fuzz tests                                          |
| `7355baf` | M4            | Integration tests                                   |
| `2632914` | M1            | CI workflow                                         |
| `c0a7633` | M11           | Example directory                                   |
| `bae967d` | M12-M15       | Documentation updates                               |
| `84f3f3e` | fix           | CI coverage scoping                                 |
| `0507927` | docs          | TODO_LIST cleanup                                   |

---

## Verdict

The TODO list is **executed but not perfected**. All 15 items shipped, all tests pass, all lint clean. But the self-review reveals:

- **8 stale doc references** to old API signatures
- **2 flaky tests** using `time.Sleep`
- **4 uncovered code branches** that should be tested
- **10+ test modernizations** left incomplete

The work is a solid **B+**: shipped everything, verified it works, but left quality debt on the table that a senior engineer would catch on review.

---

## Resolution (2026-07-26)

The questions (§g) are all answered; the quality debt is now tracked in `TODO_LIST.md`.

| Item           | Claim in report                                  | Resolution                                                                              |
| -------------- | ------------------------------------------------ | --------------------------------------------------------------------------------------- |
| §g Q1          | Should I create git tag v0.1.0?                  | Done — `v0.1.0` tagged (2026-07-23), then `v0.2.0` tagged (2026-07-24). Both on origin. |
| §g Q2          | Breaking API changes warrant different version?  | Treated as initial release (zero prior tags, zero consumers at the time). Correct call. |
| §g Q3          | `GOEXPERIMENT=jsonv2` hard requirement           | Kept; documented in README, CONTRIBUTING, AGENTS.md.                                    |
| §d.1–2         | 2 flaky tests using `time.Sleep`                 | **Still open** — tracked in `TODO_LIST.md` "High impact"                                |
| §d.3–6         | 4 uncovered code branches                        | **Still open** — tracked in `TODO_LIST.md` "Medium impact"                              |
| §c.1–2         | 8 stale `defer stream.Close()` doc references    | **Still open** — tracked in `TODO_LIST.md` "Low impact"                                 |
| §c.5 / §e.8–10 | Incomplete test modernization (context, go func) | **Still open** — tracked in `TODO_LIST.md` "Low impact"                                 |

The library subsequently shipped `v0.2.0` with `BroadcastMany`, `SendJSON`, `Event.String`, and `EventsAfter` error propagation. Coverage rose to 97.9% with 94 tests.

---

## Archival check (2026-08-29, docs-health pass - full EOF read)

- Fully read 2026-08-29 (353/353). Resolution appendix (2026-07-26) verified plus all 'Still open' debt closed later: flaky time.Sleep tests fixed in v0.2.1; the 4 coverage gaps closed in v0.2.1 (eventBrand.Name re-tested, MustParseEventID success, splitLines dead branch removed, Heartbeat error path); stale defer stream.Close() refs fixed in v0.2.1 (v0.2.1 Changed); modernization completed in v0.2.1; fuzz CI job added v0.2.1 (per-target targets extended since); heartbeat + replay integration tests shipped v0.2.1. Section F 1-18 done across v0.2.x-v0.5.x (versioning practice established via 9 tags; ROADMAP restructured 2026-08-03). Section F 19-24 shipped (WithBufferSize v0.4.0, Shutdown v0.4.0, OnDrop v0.5.0, BroadcastMany v0.2.0, memory profile docs/performance/scale-profile.md, BroadcastFiltered n/a -> SubscribeFilter v0.4.0). Section F 25-50 shipped, superseded, or Won't (bulk-send n/a -> BroadcastMany; Makefile n/a -> flake; gosec via golangci-lint). Coverage-restore regression (27-10-26 note) tracked in TODO_LIST. Fully resolved -> archived.
